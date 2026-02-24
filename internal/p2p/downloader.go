package p2p

import (
	"fmt"
	"os"
	"sync"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/messages"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/tracker"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/utils"
)

type Downloader struct {
	PeerID       [20]byte
	Peers        []peers.Peer
	File         torrentfile.TorrentFile
	PieceManager *PieceManager
	Clients      map[string]*client.Client
	mu           sync.Mutex
}

func NewDownloader(path string) (*Downloader, error) {
	tf, err := torrentfile.Open(path)
	if err != nil {
		return nil, err
	}

	peerID, err := utils.GeneratePeerID()
	if err != nil {
		return nil, err
	}

	peers, err := tracker.DiscoverPeers(tf.Announce, peerID, tf.InfoHash, tf.Length)
	if err != nil {
		return nil, err
	}

	downloader := &Downloader{
		PeerID:       peerID,
		Peers:        peers,
		PieceManager: NewPieceManager(tf),
		File:         tf,
		Clients:      make(map[string]*client.Client),
	}

	return downloader, nil
}

func (d *Downloader) CreateClient(peer peers.Peer) error {
	c, err := client.New(peer, d.PeerID, d.File.InfoHash)
	if err != nil {
		return err
	}
	d.Clients[peer.String()] = c
	return nil
}

func (d *Downloader) DownloadPiece(index int, peer peers.Peer) error {
	/* peer messages */
	// 1. receive bitfield
	// included in the client instantiation
	client, ok := d.Clients[peer.String()]
	if !ok {
		return fmt.Errorf("no client connected to peer %s", peer)
	}

	// 2. send interested
	if err := client.SendInterested(); err != nil {
		return fmt.Errorf("failed to send interested: %w", err)
	}

	// 3. send unchoke
	if err := client.SendUnchoke(); err != nil {
		return fmt.Errorf("failed to receive unchoke: %w", err)
	}

	m, err := client.Read()
	if err != nil {
		return fmt.Errorf("failed to receive unchoke: %w", err)
	}

	if m == nil {
		return fmt.Errorf("expected unchoke but got %s", m)
	}

	if m.ID != messages.MsgUnchoke {
		return fmt.Errorf("expected unchoke but got ID %d", m.ID)
	}

	client.Choked = false

	_, err = d.attemptToDownloadPiece(peer.String(), index)
	if err != nil {
		return fmt.Errorf("failed to download piece: %w", err)
	}

	d.PieceManager.Pieces[index].State = PieceStateComplete

	if !d.PieceManager.Pieces[index].Verify() {
		return fmt.Errorf("piece received has different value to piece hash in torrent file")
	}

	return nil
}

func (d *Downloader) DownloadFile() {
	fmt.Printf("# pieces to download: %d\n", d.File.NumPieces())
	fmt.Printf("# of available peers: %d\n", len(d.Peers))

	results := make(chan int)
	workQueue := make(chan int, len(d.PieceManager.Missing))

	for i := range d.PieceManager.Missing {
		workQueue <- i
	}

	// I'm assuming all pieces will be downloaded with no problem
	// whatsoever
	for _, peer := range d.Peers {
		go d.startDownloadWorker(peer, workQueue, results)
	}

	donePieces := 0

	for donePieces < d.File.NumPieces() {
		idx := <-results
		fmt.Println("Received piece", idx)
		donePieces++
	}

	fmt.Println("Download Complete!")
}

func (d *Downloader) startDownloadWorker(peer peers.Peer, workQueue chan int, results chan int) {
	client := d.Clients[peer.String()]
	client.DoHandshake()
	client.ReadBitfield()

	fmt.Printf("Started connection with peer %s\n", peer)

	// 2. send interested
	if err := client.SendInterested(); err != nil {
		fmt.Printf("failed to send interested: %s", err)
	}

	// 3. send unchoke
	if err := client.SendUnchoke(); err != nil {
		fmt.Printf("failed to receive unchoke: %s", err)
	}

	m, err := client.Read()
	if err != nil {
		fmt.Printf("failed to receive unchoke: %s", err)
	}

	if m == nil {
		fmt.Printf("expected unchoke but got %s", m)
	}

	if m.ID != messages.MsgUnchoke {
		fmt.Printf("expected unchoke but got ID %d", m.ID)
	}

	client.Choked = false

	for idx := range workQueue {
		d.PieceManager.Missing[idx] = false
		d.PieceManager.Pieces[idx].State = PieceStatePending
		d.PieceManager.InProgress[idx] = true

		fmt.Printf("Attempting to download piece %d\n", idx)
		_, err := d.attemptToDownloadPiece(peer.String(), idx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to download piece %d", idx)
			delete(d.PieceManager.InProgress, idx)
			d.PieceManager.Missing[idx] = true
			workQueue <- idx
		}
		fmt.Printf("Downloaded piece %d\n", idx)

		d.PieceManager.Pieces[idx].State = PieceStateComplete
		d.PieceManager.InProgress[idx] = false
		d.PieceManager.Downloaded[idx] = true

		results <- idx
	}
}

func (d *Downloader) attemptToDownloadPiece(peerAddress string, pieceIndex int) ([]byte, error) {
	client := d.Clients[peerAddress]
	piece := d.PieceManager.Pieces[pieceIndex]
	// 4.2 send a request message for each block
	// 5. Wait for piece message for each requested block
	data := make([]byte, piece.Length)

	for _, block := range piece.Blocks {
		fmt.Printf("sending request to %s\n", peerAddress)
		if err := client.SendRequest(pieceIndex, block.Begin, block.Length); err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		fmt.Printf("reading response from %s\n", peerAddress)
		m, err := client.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to receive request: %w", err)
		}

		fmt.Printf("parsing piece received from %s\n", peerAddress)
		_, err = messages.ParsePiece(pieceIndex, data, m)
		if err != nil {
			return nil, fmt.Errorf("error parsing piece: %w", err)
		}
		piece.Downloaded += block.Length
		block.Data = data[block.Begin : block.Begin+block.Length]
	}

	return data, nil
}

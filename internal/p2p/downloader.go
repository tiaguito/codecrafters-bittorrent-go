package p2p

import (
	"fmt"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/messages"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/utils"
)

type Downloader struct {
	PeerID       [20]byte
	Peers        []peers.Peer
	File         torrentfile.TorrentFile
	PieceManager *PieceManager
	Clients      map[string]*client.Client
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

	peers, err := tf.DiscoverPeers(peerID, torrentfile.PORT)
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

func (d *Downloader) AddClient(peer peers.Peer) error {
	if _, ok := d.Clients[peer.String()]; !ok {
		c, err := client.New(peer, d.PeerID, d.File.InfoHash)
		if err != nil {
			return err
		}
		d.Clients[peer.String()] = c
	}

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

	blocks, err := d.attemptToDownloadPiece(peer.String(), index)
	if err != nil {
		return fmt.Errorf("failed to download piece: %w", err)
	}

	d.PieceManager.Pieces[index].Blocks = blocks

	if !d.PieceManager.Pieces[index].Verify() {
		return fmt.Errorf("piece received has different value to piece hash in torrent file")
	}

	return nil
}

func (d *Downloader) DownloadFile() {
	results := make(chan int)

	// I'm assuming all pieces will be downloaded with no problem
	// whatsoever
	for _, peer := range d.Peers {
		go d.startDownloadWorker(peer, results)
	}

	donePieces := 0

	for donePieces < d.File.NumPieces() {
		<-results
		donePieces++
	}

	fmt.Println("Download Complete!")
}

func (d *Downloader) startDownloadWorker(peer peers.Peer, results chan int) {
	d.AddClient(peer)
	d.Clients[peer.String()].DoHandshake()
	d.Clients[peer.String()].ReadBitfield()

	for idx := range d.PieceManager.Pieces {
		d.PieceManager.Pieces[idx].mu.Lock()
		defer d.PieceManager.Pieces[idx].mu.Unlock()

		d.PieceManager.Missing[idx] = false
		d.PieceManager.Pieces[idx].State = PieceStatePending
		d.PieceManager.InProgress[idx] = true

		buf, err := d.attemptToDownloadPiece(peer.String(), idx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to download piece %d", idx)
			d.PieceManager.InProgress[idx] = false
			d.PieceManager.Missing[idx] = true
		}

		d.PieceManager.Pieces[idx].Blocks = buf
		d.PieceManager.Pieces[idx].State = PieceStateComplete
		d.PieceManager.InProgress[idx] = false
		d.PieceManager.Downloaded[idx] = true

		results <- idx
	}
}

func (d *Downloader) attemptToDownloadPiece(peerAddress string, pieceIndex int) ([]*Block, error) {
	client := d.Clients[peerAddress]
	piece := d.PieceManager.Pieces[pieceIndex]
	// 4.2 send a request message for each block
	// 5. Wait for piece message for each requested block
	blocks := make([]*Block, piece.NumBlocks())
	data := make([]byte, piece.Length)

	for i, block := range piece.Blocks {
		if err := client.SendRequest(pieceIndex, block.Begin, block.Length); err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		m, err := client.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to receive request: %w", err)
		}

		_, err = messages.ParsePiece(pieceIndex, data, m)
		if err != nil {
			return nil, fmt.Errorf("error parsing piece: %w", err)
		}
		blocks[i] = &Block{
			Index:  piece.Index,
			Begin:  block.Begin,
			Length: block.Length,
			Data:   data[block.Begin:],
		}
	}

	return blocks, nil
}

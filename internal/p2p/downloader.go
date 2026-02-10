package p2p

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/messages"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/utils"
)

type Downloader struct {
	PeerID  [20]byte
	Peers   []peers.Peer
	File    torrentfile.TorrentFile
	Clients map[string]*client.Client
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
		PeerID:  peerID,
		Peers:   peers,
		File:    tf,
		Clients: make(map[string]*client.Client),
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

func (d *Downloader) DownloadPiece(destinationPath string, index int, peer peers.Peer) error {
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

	data, err := d.attemptToDownloadPiece(client, index)
	if err != nil {
		return fmt.Errorf("failed to download piece: %w", err)
	}

	// check integrity of piece
	pieceHash := sha1.Sum(data)

	if !bytes.Equal(d.File.PieceHashes[index][:], pieceHash[:]) {
		return fmt.Errorf("piece received has different value to piece hash in torrent file")
	}

	// save piece to file
	file, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data to file: %w", err)
	}
	return nil
}

func (d *Downloader) attemptToDownloadPiece(client *client.Client, index int) ([]byte, error) {
	// 4
	// 4.1 break the piece into blocks of 16 kiB
	piece := NewPiece(index, d.File.Length, d.File.PieceLength, d.File.InfoHash)

	// 4.2 send a request message for each block
	// 5. Wait for piece message for each requested block
	data := make([]byte, piece.Length)
	for _, block := range piece.Blocks {
		if err := client.SendRequest(piece.Index, block.Begin, block.Length); err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		m, err := client.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to receive request: %w", err)
		}

		_, err = messages.ParsePiece(piece.Index, data, m)
		if err != nil {
			return nil, fmt.Errorf("error parsing piece: %w", err)
		}
	}
	return data, nil
}

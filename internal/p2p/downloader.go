package p2p

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/messages"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/utils"
)

type Downloader struct {
	PeerID [20]byte
	Peers  []peers.Peer
	File   torrentfile.TorrentFile
	Client *client.Client
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
		PeerID: peerID,
		Peers:  peers,
		File:   tf,
	}

	return downloader, nil
}

func (d *Downloader) CreateClient(peer peers.Peer) error {
	c, err := client.New(peer, d.PeerID, d.File.InfoHash)
	if err != nil {
		return err
	}
	d.Client = c
	return nil
}

func (d *Downloader) DownloadPiece(destinationPath string, index int) error {
	/* peer messages */
	// 1. receive bitfield
	// included in the client instantiation

	// 2. send interested
	if err := d.Client.SendInterested(); err != nil {
		return fmt.Errorf("failed to send interested: %w", err)
	}

	// 3. send unchoke
	if err := d.Client.SendUnchoke(); err != nil {
		return fmt.Errorf("failed to receive unchoke: %w", err)
	}

	m, err := d.Client.Read()
	if err != nil {
		return fmt.Errorf("failed to receive unchoke: %w", err)
	}

	if m == nil {
		return fmt.Errorf("expected unchoke but got %s", m)
	}

	if m.ID != messages.MsgUnchoke {
		return fmt.Errorf("expected unchoke but got ID %d", m.ID)
	}

	d.Client.Choked = false

	data, err := d.attemptToDownloadPiece(index)
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

func (d *Downloader) attemptToDownloadPiece(index int) ([]byte, error) {
	// 4
	// 4.1 break the piece into blocks of 16 kiB
	pieceLength := d.File.PieceLength
	const blockSize int = 16 * 1024

	pieceCount := int(math.Ceil(float64(d.File.Length) / float64(d.File.PieceLength)))
	if index == pieceCount-1 {
		pieceLength = d.File.Length % d.File.PieceLength
	}

	blocks := int(math.Ceil(float64(pieceLength) / float64(blockSize)))

	// 4.2 send a request message for each block
	// 5. Wait for piece message for each requested block
	data := make([]byte, pieceLength)
	for block := 0; block < blocks; block++ {
		blockLength := blockSize
		if block == blocks-1 {
			blockLength = pieceLength - ((blocks - 1) * blockSize)
		}

		if err := d.Client.SendRequest(index, block*blockSize, blockLength); err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		m, err := d.Client.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to receive request: %w", err)
		}

		_, err = messages.ParsePiece(index, data, m)
		if err != nil {
			return nil, fmt.Errorf("error parsing piece: %w", err)
		}
	}
	return data, nil
}

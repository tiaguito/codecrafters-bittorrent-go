package p2p

import (
	"sync"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
)

type PieceManager struct {
	Torrent    torrentfile.TorrentFile
	Pieces     []*Piece
	Downloaded map[int]bool
	InProgress map[int]bool
	Missing    map[int]bool
	Completed  int
	mu         sync.RWMutex
}

func NewPieceManager(torrentFile torrentfile.TorrentFile) *PieceManager {
	pieces := make([]*Piece, torrentFile.NumPieces())
	for i := 0; i < torrentFile.NumPieces(); i++ {
		pieceSize := torrentFile.PieceLength
		pieces[i] = NewPiece(i, torrentFile.Length, pieceSize, torrentFile.PieceHashes[i])
	}

	missing := make(map[int]bool)
	for i := 0; i < len(torrentFile.PieceHashes); i++ {
		missing[i] = true
	}

	return &PieceManager{
		Torrent:    torrentFile,
		Pieces:     pieces,
		Downloaded: make(map[int]bool),
		InProgress: make(map[int]bool),
		Missing:    missing,
		Completed:  0,
	}
}

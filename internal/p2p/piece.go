package p2p

import (
	"bytes"
	"crypto/sha1"
	"math"
	"sync"
)

const (
	BlockSize = 16 * 1024
)

type PieceState int

const (
	PieceStateNone PieceState = iota
	PieceStatePending
	PieceStateComplete
)

type Block struct {
	Index  int
	Begin  int
	Length int
	Data   []byte
}

type Piece struct {
	Index      int
	Hash       [20]byte
	Length     int
	Blocks     []*Block
	State      PieceState
	Downloaded int
	mu         sync.RWMutex
}

// 4
// 4.1 break the piece into blocks of 16 kiB
func NewPiece(index, fileLength, pieceLength int, hash [20]byte) *Piece {
	pl := pieceLength
	pieceCount := int(math.Ceil(float64(fileLength) / float64(pieceLength)))
	if index == pieceCount-1 {
		pl = fileLength % pieceLength
	}

	numBlocks := int(math.Ceil(float64(pl) / float64(BlockSize)))
	blocks := make([]*Block, numBlocks)

	for i := 0; i < numBlocks; i++ {
		begin := i * BlockSize
		blockLength := BlockSize
		if i == numBlocks-1 {
			blockLength = pl - ((numBlocks - 1) * BlockSize)
		}

		blocks[i] = &Block{
			Index:  i,
			Begin:  begin,
			Length: blockLength,
		}
	}

	return &Piece{
		Index:  index,
		Hash:   hash,
		Length: pl,
		Blocks: blocks,
		State:  PieceStateNone,
	}
}

func (p *Piece) NumBlocks() int {
	return len(p.Blocks)
}

func (p *Piece) IsComplete() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.Length == p.Downloaded
}

func (p *Piece) AssembleData() []byte {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.IsComplete() {
		return nil
	}

	data := make([]byte, p.Length)

	for _, block := range p.Blocks {
		if block.Data != nil {
			copy(data[block.Begin:], block.Data)
		}
	}

	return data
}

func (p *Piece) Verify() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.IsComplete() {
		return false
	}

	data := p.AssembleData()

	if data == nil {
		return false
	}

	hash := sha1.Sum(data)
	return bytes.Equal(p.Hash[:], hash[:])
}

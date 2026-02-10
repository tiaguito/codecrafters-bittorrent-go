package p2p

import "math"

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
	Index    int
	Hash     [20]byte
	Length   int
	Blocks   []*Block
	State    PieceState
	Download int
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

package handshake

import (
	"fmt"
	"io"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
)

type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

func New(infoHash, peerID [20]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (h *Handshake) Serialize() []byte {
	buf := make([]byte, len(h.Pstr)+49)
	buf[0] = byte(len(h.Pstr))
	currIdx := 1
	currIdx += copy(buf[currIdx:], h.Pstr)
	currIdx += copy(buf[currIdx:], make([]byte, 8))
	currIdx += copy(buf[currIdx:], h.InfoHash[:])
	currIdx += copy(buf[currIdx:], h.PeerID[:])

	return buf
}

func Read(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0])

	if pstrlen == 0 {
		return nil, fmt.Errorf("pstrlen cannot be 0")
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peerID [torrentfile.HASHLEN]byte

	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+torrentfile.HASHLEN])
	copy(peerID[:], handshakeBuf[pstrlen+8+torrentfile.HASHLEN:])

	h := &Handshake{
		Pstr:     string(handshakeBuf[0:pstrlen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	return h, nil
}

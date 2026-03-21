package handshake

import (
	"fmt"
	"io"
)

type Handshake struct {
	Pstr     string
	Reserved [8]byte
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
	currIdx += copy(buf[currIdx:], h.Reserved[:])
	currIdx += copy(buf[currIdx:], h.InfoHash[:])
	currIdx += copy(buf[currIdx:], h.PeerID[:])

	return buf
}

func Read(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, lengthBuf); err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0])

	if pstrlen == 0 {
		return nil, fmt.Errorf("pstrlen cannot be 0")
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	if _, err := io.ReadFull(r, handshakeBuf); err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte
	var reserved [8]byte

	offset := pstrlen
	offset += copy(reserved[:], handshakeBuf[offset:offset+8])
	offset += copy(infoHash[:], handshakeBuf[offset:offset+20])
	copy(peerID[:], handshakeBuf[offset:])

	h := &Handshake{
		Pstr:     string(handshakeBuf[0:pstrlen]),
		Reserved: reserved,
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	return h, nil
}

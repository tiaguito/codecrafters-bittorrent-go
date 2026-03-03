package tracker

import (
	"crypto/rand"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
)

func GeneratePeerID() ([20]byte, error) {
	var peerID [torrentfile.HASHLEN]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return [torrentfile.HASHLEN]byte{}, err
	}
	return peerID, nil
}

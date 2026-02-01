package utils

import (
	"crypto/rand"

	"github.com/codecrafters-io/bittorrent-starter-go/torrentfile"
)

func generatePeerID() ([torrentfile.HashLen]byte, error) {
	var peerID [torrentfile.HashLen]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return [torrentfile.HashLen]byte{}, err
	}
	return peerID, nil
}

package tracker

import (
	"crypto/rand"

	infohash "github.com/codecrafters-io/bittorrent-starter-go/internal/types"
)

func GeneratePeerID() ([20]byte, error) {
	var peerID [infohash.Size]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return [infohash.Size]byte{}, err
	}
	return peerID, nil
}

package torrentfile

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
	"github.com/stretchr/testify/assert"
)

func TestBuildTrackerURL(t *testing.T) {
	to := TorrentFile{
		Announce: "http://bittorrent-test-tracker.codecrafters.io/announce",
		InfoHash: [20]byte{214, 159, 145, 230, 178, 174, 76, 84, 36, 104, 209, 7, 58, 113, 212, 234, 19, 135, 154, 127},
		PieceHashes: [][20]byte{
			{232, 118, 246, 122, 42, 136, 134, 232, 243, 107, 19, 103, 38, 195, 15, 162, 151, 3, 2, 45},
			{110, 34, 117, 230, 4, 160, 118, 102, 86, 115, 110, 129, 255, 16, 181, 82, 4, 173, 141, 53},
			{240, 13, 147, 122, 2, 19, 223, 25, 130, 188, 141, 9, 114, 39, 173, 158, 144, 154, 204, 23},
		},
		PieceLength: 32768,
		Length:      92063,
		Name:        "../sample.torrent",
	}
	peerID := [20]byte{160, 137, 13, 47, 173, 183, 131, 17, 39, 118, 162, 195, 93, 41, 91, 81, 121, 252, 66, 154}
	const port uint16 = 6881
	url, err := to.buildTrackerURL(peerID, port)
	expected := "http://bittorrent-test-tracker.codecrafters.io/announce?compact=1&downloaded=0&info_hash=%D6%9F%91%E6%B2%AELT%24h%D1%07%3Aq%D4%EA%13%87%9A%7F&left=92063&peer_id=%A0%89%0D%2F%AD%B7%83%11%27v%A2%C3%5D%29%5BQy%FCB%9A&port=6881&uploaded=0"
	assert.Nil(t, err)
	assert.Equal(t, url, expected)
}

func TestDiscoverPeers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []byte(
			"d" +
				"8:interval" + "i900e" +
				"5:peers" + "12:" +
				string([]byte{
					192, 0, 2, 123, 0x1A, 0xE1, // 0x1AE1 = 6881
					127, 0, 0, 1, 0x1A, 0xE9, // 0x1AE9 = 6889
				}) + "e")
		w.Write(response)
	}))
	defer ts.Close()
	tf := TorrentFile{
		Announce: ts.URL,
		InfoHash: [20]byte{214, 159, 145, 230, 178, 174, 76, 84, 36, 104, 209, 7, 58, 113, 212, 234, 19, 135, 154, 127},
		PieceHashes: [][20]byte{
			{232, 118, 246, 122, 42, 136, 134, 232, 243, 107, 19, 103, 38, 195, 15, 162, 151, 3, 2, 45},
			{110, 34, 117, 230, 4, 160, 118, 102, 86, 115, 110, 129, 255, 16, 181, 82, 4, 173, 141, 53},
			{240, 13, 147, 122, 2, 19, 223, 25, 130, 188, 141, 9, 114, 39, 173, 158, 144, 154, 204, 23},
		},
		PieceLength: 32768,
		Length:      92063,
		Name:        "../sample.torrent",
	}
	peerID := [20]byte{160, 137, 13, 47, 173, 183, 131, 17, 39, 118, 162, 195, 93, 41, 91, 81, 121, 252, 66, 154}
	const port uint16 = 6881
	expected := []peers.Peer{
		{IP: net.IP{192, 0, 2, 123}, Port: 6881},
		{IP: net.IP{127, 0, 0, 1}, Port: 6889},
	}
	p, err := tf.DiscoverPeers(peerID, port)
	assert.Nil(t, err)
	assert.Equal(t, expected, p)
}

package tracker

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
	"github.com/jackpal/bencode-go"
)

const PORT uint16 = 6881

type bencodeTrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"string"`
}

func buildTrackerURL(trackerURL string, peerID, infoHash [20]byte, length int) (string, error) {
	base, err := url.Parse(trackerURL)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"info_hash":  []string{string(infoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(PORT))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(length)},
	}

	base.RawQuery = params.Encode()
	return base.String(), nil
}

func DiscoverPeers(trackerURL string, peerID, infoHash [20]byte, length int) ([]peers.Peer, error) {
	url, err := buildTrackerURL(trackerURL, peerID, infoHash, length)
	if err != nil {
		return nil, err
	}

	c := &http.Client{Timeout: 15 * time.Second}
	res, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	trackerRes := bencodeTrackerResponse{}
	err = bencode.Unmarshal(res.Body, &trackerRes)
	if err != nil {
		return nil, err
	}

	return peers.Unmarshal([]byte(trackerRes.Peers))
}

package client

import (
	"bytes"
	"fmt"
	"net"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/handshake"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
)

type Bitfield []byte

type Client struct {
	Conn      net.Conn
	Peer      peers.Peer
	InfoHash  [20]byte
	PeerID    [20]byte
	Handshake *handshake.Handshake
}

func doHandshake(conn net.Conn, infoHash, peerID [20]byte) (*handshake.Handshake, error) {
	req := handshake.New(infoHash, peerID)

	_, err := conn.Write(req.Serialize())
	if err != nil {
		return nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	res, err := handshake.Read(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to received handshake: %w", err)
	}

	if !bytes.Equal(res.InfoHash[:], infoHash[:]) {
		return nil, fmt.Errorf("expected infohash %x but got %x", res.InfoHash, infoHash)
	}

	return res, nil
}

func New(peer peers.Peer, peerID, infoHash [20]byte) (*Client, error) {
	conn, err := net.Dial("tcp", peer.String())
	if err != nil {
		return nil, err
	}

	_, err = doHandshake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Client{
		Conn:     conn,
		Peer:     peer,
		InfoHash: infoHash,
		PeerID:   peerID,
	}, nil
}

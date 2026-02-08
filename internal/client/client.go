package client

import (
	"bytes"
	"fmt"
	"net"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/handshake"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/messages"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
)

type Bitfield []byte

type Client struct {
	Conn      net.Conn
	Bitfield  Bitfield
	Choked    bool
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

func readBitfield(conn net.Conn) (Bitfield, error) {
	msg, err := messages.Read(conn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, err
	}
	if msg.ID != messages.MsgBitfield {
		return nil, fmt.Errorf("expected bitfield but got ID %d", msg.ID)
	}
	return msg.Payload, nil
}

func New(peer peers.Peer, peerID, infoHash [20]byte) (*Client, error) {
	conn, err := net.Dial("tcp", peer.String())
	if err != nil {
		return nil, err
	}

	handshakeResp, err := doHandshake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	bf, err := readBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Client{
		Conn:      conn,
		Bitfield:  bf,
		Choked:    true,
		Peer:      peer,
		InfoHash:  infoHash,
		PeerID:    peerID,
		Handshake: handshakeResp,
	}, nil
}

func (c *Client) Read() (*messages.Message, error) {
	msg, err := messages.Read(c.Conn)
	return msg, err
}

func (c *Client) SendUnchoke() error {
	msg := messages.Message{ID: messages.MsgUnchoke}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *Client) SendRequest(index, begin, length int) error {
	req := messages.FormatRequest(index, begin, length)
	_, err := c.Conn.Write(req.Serialize())
	return err
}

func (c *Client) SendInterested() error {
	msg := messages.Message{ID: messages.MsgInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

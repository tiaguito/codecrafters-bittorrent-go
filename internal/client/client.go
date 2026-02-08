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

func (c *Client) DoHandshake() error {
	req := handshake.New(c.InfoHash, c.PeerID)

	_, err := c.Conn.Write(req.Serialize())
	if err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	res, err := handshake.Read(c.Conn)
	if err != nil {
		return fmt.Errorf("failed to received handshake: %w", err)
	}

	if !bytes.Equal(res.InfoHash[:], c.InfoHash[:]) {
		return fmt.Errorf("expected infohash %x but got %x", res.InfoHash, c.InfoHash)
	}

	c.Handshake = res

	return nil
}

func (c *Client) ReadBitfield() error {
	msg, err := messages.Read(c.Conn)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("expected bitfield but got %s", msg)
	}
	if msg.ID != messages.MsgBitfield {
		return fmt.Errorf("expected bitfield but got ID %d", msg.ID)
	}
	c.Bitfield = msg.Payload

	return nil
}

func New(peer peers.Peer, peerID, infoHash [20]byte) (*Client, error) {
	conn, err := net.Dial("tcp", peer.String())
	if err != nil {
		return nil, err
	}

	return &Client{
		Conn:     conn,
		Choked:   true,
		Peer:     peer,
		InfoHash: infoHash,
		PeerID:   peerID,
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

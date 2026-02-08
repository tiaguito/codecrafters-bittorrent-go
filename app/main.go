package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/handshake"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/messages"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/utils"
	bencode "github.com/jackpal/bencode-go"
)

type Cmd struct {
	out io.Writer
}

func NewCmd(out io.Writer) *Cmd {
	if out == nil {
		out = os.Stdout
	}
	return &Cmd{out: out}
}

func (c *Cmd) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: <command> <argument>")
	}

	cmd := args[0]
	handler, ok := commandHandlers[cmd]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd)
	}

	return handler(c, args[1:])
}

var commandHandlers = map[string]func(*Cmd, []string) error{
	"decode":         decodeCommand,
	"info":           infoCommand,
	"peers":          peersCommand,
	"handshake":      handshakeCommand,
	"download_piece": downloadPieceCommand,
}

func decodeCommand(c *Cmd, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: decode <bencoded string>")
	}

	bencodedValue := bytes.NewReader([]byte(args[0]))
	decoded, err := bencode.Decode(bencodedValue)
	if err != nil {
		return fmt.Errorf("failed to decode: %w", err)
	}

	return json.NewEncoder(c.out).Encode(decoded)
}

func infoCommand(c *Cmd, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: info <torrent file>")
	}

	tf, err := torrentfile.Open(args[0])
	if err != nil {
		return fmt.Errorf("failed to create torrent: %w", err)
	}
	fmt.Fprintln(c.out, tf)

	return nil
}

func peersCommand(c *Cmd, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: peers <torrent file>")
	}

	tf, err := torrentfile.Open(args[0])
	if err != nil {
		return fmt.Errorf("failed to create torrent: %w", err)
	}

	peerID, err := utils.GeneratePeerID()
	if err != nil {
		return fmt.Errorf("failed to generate peer ID: %w", err)
	}

	peers, err := tf.DiscoverPeers(peerID, torrentfile.PORT)
	for _, peer := range peers {
		fmt.Fprintln(c.out, peer)
	}

	return nil
}

func handshakeCommand(c *Cmd, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: handshake <torrent file> <peer_ip>:<peer_port>")
	}
	tf, err := torrentfile.Open(args[0])
	if err != nil {
		return fmt.Errorf("failed to create torrent: %w", err)
	}

	peerID, err := utils.GeneratePeerID()
	if err != nil {
		return fmt.Errorf("failed to generate peer ID: %w", err)
	}

	peer, err := peers.StringToPeer(args[1])
	if err != nil {
		return fmt.Errorf("invalid IP address: %w", err)
	}

	clnt, err := client.New(peer, peerID, tf.InfoHash)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	fmt.Fprintf(c.out, "Peer ID: %x\n", clnt.Handshake.PeerID)
	return nil
}

func downloadPieceCommand(c *Cmd, args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("usage: download_piece -o <piece destination path> <torrent file> <piece index>")
	}

	tf, err := torrentfile.Open(args[2])
	if err != nil {
		return fmt.Errorf("failed to create torrent: %w", err)
	}

	peerID, err := utils.GeneratePeerID()
	if err != nil {
		return fmt.Errorf("failed to generate peer ID: %w", err)
	}

	peers, err := tf.DiscoverPeers(peerID, torrentfile.PORT)
	if err != nil {
		return fmt.Errorf("failed to get peers: %w", err)
	}

	conn, err := net.Dial("tcp", peers[0].String())
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}
	defer conn.Close()

	h := handshake.New(tf.InfoHash, peerID)

	_, err = conn.Write(h.Serialize())
	if err != nil {
		return fmt.Errorf("failed to send handshake request: %w", err)
	}

	_, err = handshake.Read(conn)
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %w", err)
	}

	/* peer messages */
	// 1. receive bitfield
	m, err := messages.Read(conn)
	if err != nil {
		return fmt.Errorf("failed to read bitfield message: %w", err)
	}

	if m == nil {
		return fmt.Errorf("expected bitfield but got %s", m)
	}

	if m.ID != messages.MsgBitfield {
		return fmt.Errorf("expected bitfield but got ID %d", m.ID)
	}

	fmt.Fprintln(c.out, m)

	// 2. send interested
	m = &messages.Message{ID: messages.MsgInterested}
	_, err = conn.Write(m.Serialize())
	if err != nil {
		return fmt.Errorf("failed to send interested message: %w", err)
	}

	// 3. wait until receive unchoke
	m, err = messages.Read(conn)
	if err != nil {
		return fmt.Errorf("failed to read unchoke message: %w", err)
	}

	if m == nil {
		return fmt.Errorf("expected unchoke but got %s", m)
	}

	if m.ID != messages.MsgUnchoke {
		return fmt.Errorf("expected unchoke but got ID %d", m.ID)
	}

	fmt.Fprintln(c.out, m)

	// 4
	// 4.1 break the piece into blocks of 16 kiB (blockSize)
	pieceLength := tf.PieceLength
	pieceCount := int(math.Ceil(float64(float64(tf.Length) / float64(tf.PieceLength))))
	index, err := strconv.Atoi(args[3])
	if err != nil {
		return fmt.Errorf("failed to convert index: %w", err)
	}
	if index == pieceCount-1 {
		pieceLength = tf.Length % tf.PieceLength
	}

	const blockSize int = 16 * 1024
	blocks := int(math.Ceil(float64(pieceLength) / float64(blockSize)))

	// 4.2 send a request message for each block
	// 5. Wait for piece message for each requested block
	data := make([]byte, pieceLength)
	for block := 0; block < blocks; block++ {
		blockLength := blockSize
		if block == blocks-1 {
			blockLength = pieceLength - ((blocks - 1) * (blockSize))
		}

		m = messages.FormatRequest(index, block*blockSize, blockLength)

		_, err = conn.Write(m.Serialize())
		if err != nil {
			return fmt.Errorf("failed to send request message: %w", err)
		}

		m, err := messages.Read(conn)
		_, err = messages.ParsePiece(index, data, m)
		if err != nil {
			return fmt.Errorf("error parsing piece: %w", err)
		}
	}

	// check integrity of piece
	pieceHash := sha1.Sum(data)

	if tf.PieceHashes[index] != pieceHash {
		return fmt.Errorf("piece received has different has value to piece hash in torrent file")
	}

	// save piece to file
	file, err := os.Create(args[1])
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("file to write data to file: %w", err)
	}

	return nil
}

func main() {
	cmd := NewCmd(nil)
	if err := cmd.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

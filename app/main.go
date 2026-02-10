package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/p2p"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
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
	"download":       downloadCommand,
}

func decodeCommand(c *Cmd, args []string) error {
	if len(args) != 1 {
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
	if len(args) != 1 {
		return fmt.Errorf("usage: info <torrent file>")
	}

	downloader, err := p2p.NewDownloader(args[0])
	if err != nil {
		return fmt.Errorf("failed to create dowloader: %w", err)
	}

	fmt.Fprintln(c.out, downloader.File.Info())

	return nil
}

func peersCommand(c *Cmd, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: peers <torrent file>")
	}

	downloader, err := p2p.NewDownloader(args[0])
	if err != nil {
		return fmt.Errorf("failed to create dowloader: %w", err)
	}

	for _, peer := range downloader.Peers {
		fmt.Fprintln(c.out, peer)
	}

	return nil
}

func handshakeCommand(c *Cmd, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: handshake <torrent file> <peer_ip>:<peer_port>")
	}
	downloader, err := p2p.NewDownloader(args[0])
	if err != nil {
		return fmt.Errorf("failed to create dowloader: %w", err)
	}

	peer, err := peers.StringToPeer(args[1])
	if err != nil {
		return fmt.Errorf("invalid IP address: %w", err)
	}

	if err = downloader.CreateClient(peer); err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	downloader.Clients[peer.String()].DoHandshake()

	fmt.Fprintf(c.out, "Peer ID: %x\n", downloader.Clients[peer.String()].Handshake.PeerID)
	return nil
}

func downloadPieceCommand(c *Cmd, args []string) error {
	if len(args) != 4 {
		return fmt.Errorf("usage: download_piece -o <piece destination path> <torrent file> <piece index>")
	}

	downloader, err := p2p.NewDownloader(args[2])
	if err != nil {
		return fmt.Errorf("failed to create dowloader: %w", err)
	}

	downloader.CreateClient(downloader.Peers[0])

	downloader.Clients[downloader.Peers[0].String()].DoHandshake()
	downloader.Clients[downloader.Peers[0].String()].ReadBitfield()

	index, err := strconv.Atoi(args[3])
	if err != nil {
		return fmt.Errorf("failed to convert index: %w", err)
	}

	if err = downloader.DownloadPiece(args[1], index, downloader.Peers[0]); err != nil {
		return err
	}

	return nil
}

func downloadCommand(c *Cmd, args []string) error {
	return nil
}

func main() {
	cmd := NewCmd(nil)
	if err := cmd.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

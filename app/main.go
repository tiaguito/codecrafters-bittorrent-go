package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

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
	"decode": decodeCommand,
	"info":   infoCommand,
	"peers":  peersCommand,
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

func main() {
	client := NewCmd(nil)
	if err := client.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

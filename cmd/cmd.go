package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/magnet"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/messages"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/p2p"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/tracker"
	"github.com/jackpal/bencode-go"
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
	"decode":           decodeCommand,
	"info":             infoCommand,
	"peers":            peersCommand,
	"handshake":        handshakeCommand,
	"download_piece":   downloadPieceCommand,
	"download":         downloadCommand,
	"magnet_parse":     magnetParseCommand,
	"magnet_handshake": magnetHandshakeCommand,
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

	if err = downloader.CreateClient(peer, true); err != nil {
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

	downloader.CreateClient(downloader.Peers[0], true)

	downloader.Clients[downloader.Peers[0].String()].DoHandshake()
	downloader.Clients[downloader.Peers[0].String()].ReadBitfield()

	index, err := strconv.Atoi(args[3])
	if err != nil {
		return fmt.Errorf("failed to convert index: %w", err)
	}

	if err = downloader.DownloadPiece(index, downloader.Peers[0]); err != nil {
		return err
	}

	// save piece to file
	file, err := os.Create(args[1])
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(downloader.PieceManager.Pieces[index].AssembleData())
	if err != nil {
		return fmt.Errorf("failed to write data to file: %w", err)
	}

	return nil
}

func downloadCommand(c *Cmd, args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: download -o <destination path> <torrent file>")
	}

	downloader, err := p2p.NewDownloader(args[2])
	if err != nil {
		return fmt.Errorf("failed to create dowloader: %w", err)
	}

	for _, peer := range downloader.Peers {
		downloader.CreateClient(peer, true)
	}

	downloader.DownloadFile()

	// save the piece to a file
	file, err := os.Create(args[1])
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	var fullData []byte
	for i := range downloader.PieceManager.Pieces {
		fullData = append(fullData, downloader.PieceManager.Pieces[i].AssembleData()...)
	}
	_, err = file.Write(fullData)
	if err != nil {
		return fmt.Errorf("failed to write data to file: %w", err)
	}

	return nil
}

func magnetParseCommand(c *Cmd, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: magnet_parse <magnet-link>")
	}

	magnet, err := magnet.New(args[0])
	if err != nil {
		return err
	}

	fmt.Fprint(c.out, magnet)

	return nil
}

func magnetHandshakeCommand(c *Cmd, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: magnet_handshake <magnet-link>")
	}

	magnt, err := magnet.New(args[0])
	if err != nil {
		return err
	}

	peerID, err := tracker.GeneratePeerID()
	if err != nil {
		return err
	}

	peers, err := tracker.DiscoverPeers(magnt.Trackers[0], peerID, magnt.InfoHash, 1)
	if err != nil {
		return err
	}

	clt, err := client.New(peers[0], peerID, magnt.InfoHash, true)
	if err != nil {
		return err
	}

	if err := clt.DoHandshake(); err != nil {
		return err
	}

	fmt.Fprintf(c.out, "Peer ID: %x\n", clt.Handshake.PeerID)

	if err := clt.ReadBitfield(); err != nil {
		return err
	}

	dict := make(map[string]map[string]uint8)
	dict["m"] = map[string]uint8{
		"ut_metadata": uint8(1),
	}

	var bencodedDict bytes.Buffer

	if err := bencode.Marshal(&bencodedDict, dict); err != nil {
		return err
	}

	msg := &messages.Message{
		ID:      messages.MsgExtension,
		Payload: []byte{0},
	}

	msg.Payload = append(msg.Payload, bencodedDict.Bytes()...)

	if _, err = clt.Conn.Write(msg.Serialize()); err != nil {
		return err
	}

	resp, err := clt.Read()
	if err != nil {
		return err
	}

	if resp == nil {
		return fmt.Errorf("expected extension handshake but got %s", resp)
	}

	if resp.ID != messages.MsgExtension {
		return fmt.Errorf("expected extension handshake but got ID %d", resp.ID)
	}

	bencodedValue := bytes.NewReader(resp.Payload[1:])
	respHandshake := &magnet.Handshake{}
	if err := bencode.Unmarshal(bencodedValue, respHandshake); err != nil {
		return err
	}

	fmt.Printf("Peer Metadata Extension ID: %v\n", respHandshake.M["ut_metadata"])

	return nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/utils"
	bencode "github.com/jackpal/bencode-go"
)

func main() {
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := bytes.NewReader([]byte(os.Args[2]))

		decoded, err := bencode.Decode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, err := json.Marshal(decoded)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(jsonOutput))
	} else if command == "info" {
		filePath := os.Args[2]
		tf, err := torrentfile.Open(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(tf)
	} else if command == "peers" {
		filepath := os.Args[2]
		tf, err := torrentfile.Open(filepath)
		if err != nil {
			fmt.Println(err)
			return
		}

		peerID, err := utils.GeneratePeerID()
		if err != nil {
			fmt.Printf("error generating peer ID: %w", err)
			return
		}

		peers, err := tf.DiscoverPeers(peerID, torrentfile.PORT)
		for _, peer := range peers {
			fmt.Println(peer)
		}
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

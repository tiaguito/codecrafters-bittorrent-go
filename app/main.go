package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/codecrafters-io/bittorrent-starter-go/torrentfile"
	bencode "github.com/jackpal/bencode-go"
)

func main() {
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := strings.NewReader(os.Args[2])

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
		bto, err := torrentfile.Open(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Tracker URL: %s\n", bto.Announce)
		fmt.Printf("Piece Length: %d\n", bto.Info.Length)
		hash, err := bto.Info.Hash()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Info Hash: %x\n", hash)
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}

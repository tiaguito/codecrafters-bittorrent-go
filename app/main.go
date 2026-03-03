package main

import (
	"fmt"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/cmd"
)

func main() {
	cmd := cmd.NewCmd(nil)
	if err := cmd.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

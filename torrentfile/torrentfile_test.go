package torrentfile

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	torrent, err := Open("../sample.torrent")
	require.Nil(t, err)

	goldenPath := "testdata/sample.torrent.json"

	expected := bencodeTorrent{}
	golden, err := os.ReadFile(goldenPath)
	require.Nil(t, err)
	err = json.Unmarshal(golden, &expected)
	fmt.Println(expected)
	require.Nil(t, err)

	assert.Equal(t, expected.Announce, torrent.Announce)
	assert.Equal(t, expected.Info.Length, torrent.Info.Length)
}

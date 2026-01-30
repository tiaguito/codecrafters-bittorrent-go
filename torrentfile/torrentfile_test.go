package torrentfile

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	torrenFile, err := Open("../sample.torrent")
	require.Nil(t, err)

	goldenPath := "testdata/sample.torrent.json"

	expected := TorrentFile{}
	golden, err := os.ReadFile(goldenPath)
	require.Nil(t, err)
	err = json.Unmarshal(golden, &expected)
	require.Nil(t, err)

	assert.Equal(t, expected, torrenFile)
}

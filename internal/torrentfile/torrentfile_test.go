package torrentfile

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	torrenFile, err := Open("../../sample.torrent")
	require.Nil(t, err)

	goldenPath := "testdata/sample.torrent.json"

	expected := TorrentFile{}
	golden, err := os.ReadFile(goldenPath)
	require.Nil(t, err)
	err = json.Unmarshal(golden, &expected)
	require.Nil(t, err)

	assert.Equal(t, expected, torrenFile)
}

func TestInfo(t *testing.T) {
	tests := []struct {
		input  TorrentFile
		output string
	}{
		{
			input: TorrentFile{
				Announce: "http://bittorrent-test-tracker.codecrafters.io/announce",
				InfoHash: [20]byte{214, 159, 145, 230, 178, 174, 76, 84, 36, 104, 209, 7, 58, 113, 212, 234, 19, 135, 154, 127},
				PieceHashes: [][20]byte{
					{232, 118, 246, 122, 42, 136, 134, 232, 243, 107, 19, 103, 38, 195, 15, 162, 151, 3, 2, 45},
					{110, 34, 117, 230, 4, 160, 118, 102, 86, 115, 110, 129, 255, 16, 181, 82, 4, 173, 141, 53},
					{240, 13, 147, 122, 2, 19, 223, 25, 130, 188, 141, 9, 114, 39, 173, 158, 144, 154, 204, 23},
				},
				PieceLength: 32768,
				Length:      92063,
				Name:        "../sample.torrent",
			},
			output: `Tracker URL: http://bittorrent-test-tracker.codecrafters.io/announce
Length: 92063
Info Hash: d69f91e6b2ae4c542468d1073a71d4ea13879a7f
Piece Length: 32768
Piece Hashes:
e876f67a2a8886e8f36b136726c30fa29703022d
6e2275e604a0766656736e81ff10b55204ad8d35
f00d937a0213df1982bc8d097227ad9e909acc17`,
		},
	}
	for _, test := range tests {
		s := test.input.Info()
		assert.Equal(t, test.output, s)
	}
}

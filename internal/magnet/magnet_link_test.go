package magnet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		input  string
		output *Magnet
		fails  bool
	}{
		"correct magnet link": {
			input: "magnet:?xt=urn:btih:ad42ce8109f54c99613ce38f9b4d87e70f24a165&dn=magnet1.gif&tr=http%3A%2F%2Fbittorrent-test-tracker.codecrafters.io%2Fannounce",
			output: &Magnet{
				InfoHash:    [20]byte{173, 66, 206, 129, 9, 245, 76, 153, 97, 60, 227, 143, 155, 77, 135, 231, 15, 36, 161, 101},
				DisplayName: "magnet1.gif",
				Trackers:    []string{"http://bittorrent-test-tracker.codecrafters.io/announce"},
			},
			fails: false,
		},
		"wrong scheme": {
			input:  "http:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567",
			output: nil,
			fails:  true,
		},
		"garbage after scheme": {
			input:  "magnet:xt=urn:btih:0123456789abcdef0123456789abcdef01234567",
			output: nil,
			fails:  true,
		},
		"missing xt": {
			input:  "magnet:?dn=Ubuntu+ISO&tr=udp://tracker.opentrackr.org:1337/announce",
			output: nil,
			fails:  true,
		},
		"invalid xt format": {
			input:  "magnet:?xt=0123456789abcdef0123456789abcdef01234567",
			output: nil,
			fails:  true,
		},
		"invalid infohash length": {
			input:  "magnet:?xt=urn:btih:12345",
			output: nil,
			fails:  true,
		},
		"invalid percent encoding": {
			input: "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&tr=udp%3A%2F%2Ftracker.example.com%3A80%2Fannounce%ZZ",
			output: &Magnet{
				InfoHash:    [20]byte{1, 35, 69, 103, 137, 171, 205, 239, 1, 35, 69, 103, 137, 171, 205, 239, 1, 35, 69, 103},
				DisplayName: "",
				Trackers:    []string(nil),
			},
			fails: false,
		},
		"duplicate but conflicting xt parameters": {
			input:  "magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa&xt=urn:btih:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			output: nil,
			fails:  true,
		},
	}

	for _, test := range tests {
		got, err := New(test.input)
		if test.fails {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		assert.Equal(t, test.output, got)
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		input  *Magnet
		output string
	}{
		{
			input: &Magnet{
				InfoHash:    [20]byte{173, 66, 206, 129, 9, 245, 76, 153, 97, 60, 227, 143, 155, 77, 135, 231, 15, 36, 161, 101},
				DisplayName: "",
				Trackers:    []string{"http://bittorrent-test-tracker.codecrafters.io/announce"},
			},
			output: `Tracker URL: http://bittorrent-test-tracker.codecrafters.io/announce
Info Hash: ad42ce8109f54c99613ce38f9b4d87e70f24a165
`,
		},
	}

	for _, test := range tests {
		s := test.input.String()
		assert.Equal(t, test.output, s)
	}
}

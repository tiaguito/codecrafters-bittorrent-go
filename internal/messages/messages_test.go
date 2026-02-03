package messages

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerialize(t *testing.T) {
	tests := map[string]struct {
		input  *Message
		output []byte
	}{
		"serialize message": {
			input:  &Message{ID: MsgHave, Payload: []byte{1, 2, 3, 4}},
			output: []byte{0, 0, 0, 5, 4, 1, 2, 3, 4},
		},
		"serialize keep-alive": {
			input:  nil,
			output: []byte{0, 0, 0, 0},
		},
	}

	for _, test := range tests {
		buf := test.input.Serialize()
		assert.Equal(t, test.output, buf)
	}
}

func TestRead(t *testing.T) {
	tests := map[string]struct {
		input  []byte
		output *Message
		fails  bool
	}{
		"parse normal message into struct": {
			input:  []byte{0, 0, 0, 5, 4, 1, 2, 3, 4},
			output: &Message{ID: MsgHave, Payload: []byte{1, 2, 3, 4}},
			fails:  false,
		},
		"parse keep-alive into nil": {
			input:  []byte{0, 0, 0, 0},
			output: nil,
			fails:  false,
		},
		"length too short": {
			input:  []byte{1, 2, 3},
			output: nil,
			fails:  true,
		},
		"buffer too short for length": {
			input:  []byte{0, 0, 0, 5, 4, 1, 2},
			output: nil,
			fails:  true,
		},
	}

	for _, test := range tests {
		reader := bytes.NewReader(test.input)
		m, err := Read(reader)
		if test.fails {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		assert.Equal(t, test.output, m)
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		input  *Message
		output string
	}{
		{nil, "KeepAlive"},
		{&Message{MsgChoke, []byte{1, 2, 3}}, "Choke [3]"},
		{&Message{MsgUnchoke, []byte{1, 2, 3}}, "Unchoke [3]"},
		{&Message{MsgInterested, []byte{1, 2, 3}}, "Interested [3]"},
		{&Message{MsgNotInterested, []byte{1, 2, 3}}, "NotInterested [3]"},
		{&Message{MsgHave, []byte{1, 2, 3}}, "Have [3]"},
		{&Message{MsgBitfield, []byte{1, 2, 3}}, "Bitfield [3]"},
		{&Message{MsgRequest, []byte{1, 2, 3}}, "Request [3]"},
		{&Message{MsgPiece, []byte{1, 2, 3}}, "Piece [3]"},
		{&Message{MsgCancel, []byte{1, 2, 3}}, "Cancel [3]"},
		{&Message{99, []byte{1, 2, 3}}, "Unknown#99 [3]"},
	}

	for _, test := range tests {
		s := test.input.String()
		assert.Equal(t, test.output, s)
	}
}

package magnet

import (
	"bytes"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrInvalidURI            = errors.New("invalid magnet URI")
	ErrInvalidScheme         = errors.New("invalid scheme")
	ErrMissingXT             = errors.New("missing xt parameter")
	ErrInvalidXT             = errors.New("invalid xt parameter")
	ErrInvalidDuplicateXT    = errors.New("duplicated but conflicting xt parameters")
	ErrInvalidInfoHash       = errors.New("invalid infohash")
	ErrInvalidInfoHashLength = errors.New("invalid infohash length")
)

type Magnet struct {
	InfoHash    []byte
	DisplayName string
	Trackers    []string
}

func New(link string) (*Magnet, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, ErrInvalidURI
	}

	if u.Scheme != "magnet" {
		return nil, ErrInvalidScheme
	}

	params := u.Query()

	infoHash, err := parseXT(params["xt"])
	if err != nil {
		return nil, err
	}

	trackers := parseTrackers(params["tr"])
	name := params.Get("dn")

	return &Magnet{
		InfoHash:    infoHash,
		DisplayName: name,
		Trackers:    trackers,
	}, nil
}

func parseXT(values []string) ([]byte, error) {
	if len(values) == 0 {
		return nil, ErrMissingXT
	}

	var found []byte

	for _, v := range values {
		if strings.HasPrefix(v, "urn:btih:") {
			hashPart := strings.TrimPrefix(v, "urn:btih:")
			decoded, err := decodeInfoHash(hashPart)
			if err != nil {
				return nil, err
			}
			if len(found) != 0 && !bytes.Equal(found[:], decoded[:]) {
				return nil, ErrInvalidDuplicateXT
			} else if len(found) == 0 {
				found = decoded
			}
		}
	}

	if found == nil {
		return nil, ErrInvalidXT
	}

	return found, nil
}

func decodeInfoHash(s string) ([]byte, error) {
	switch len(s) {
	case 40: // hex
		b, err := hex.DecodeString(s)
		if err != nil {
			return nil, ErrInvalidInfoHash
		}
		return b, nil

	case 32: // base32
		b, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(s))
		if err != nil {
			return nil, ErrInvalidInfoHash
		}
		return b, nil

	default:
		return nil, ErrInvalidInfoHashLength
	}
}

func parseTrackers(values []string) []string {
	var out []string

	for _, v := range values {
		u, err := url.Parse(v)
		if err != nil {
			continue
		}

		switch u.Scheme {
		case "udp", "http", "https":
			out = append(out, v)
		}
	}

	return out
}

func (m *Magnet) String() string {
	var str []string

	for _, trackerURL := range m.Trackers {
		str = append(str, fmt.Sprint("Tracker URL: ", trackerURL))
	}
	str = append(str, fmt.Sprintf("Info Hash: %x\n", m.InfoHash))

	return strings.Join(str, "\n")
}

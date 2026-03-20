package magnet

import (
	"bytes"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
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
	InfoHash    [20]byte
	DisplayName string
	Trackers    []string
}

type Handshake struct {
	M            map[string]int `bencode:"m"`
	MetadataSize int            `bencode:"metadata_size"`
	P            int            `bencode:"p"`
	V            string         `bencode:"v"`
	YourIP       string         `bencode:"yourip"`
	Ipv6         string         `bencode:"ipv6"`
	Ipv4         string         `bencode:"ipv4"`
	Req          int            `bencode:"reqq"`
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

func parseXT(values []string) ([20]byte, error) {
	var zero [20]byte

	if len(values) == 0 {
		return zero, ErrMissingXT
	}

	var found []byte

	for _, v := range values {
		if strings.HasPrefix(v, "urn:btih:") {
			hashPart := strings.TrimPrefix(v, "urn:btih:")
			decoded, err := decodeInfoHash(hashPart)
			if err != nil {
				return zero, err
			}
			if len(found) != 0 && !bytes.Equal(found[:], decoded[:]) {
				return zero, ErrInvalidDuplicateXT
			} else if len(found) == 0 {
				found = decoded
			}
		}
	}

	if found == nil {
		return zero, ErrInvalidXT
	}

	var out [20]byte
	copy(out[:], found)
	return out, nil
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

func (m *Handshake) String() string {
	var str []string

	str = append(str, fmt.Sprint("m:"))
	for key, value := range m.M {
		str = append(str, fmt.Sprintf("\t%s: %d", key, value))
	}
	str = append(str, fmt.Sprintf("metadata_size: %d", m.MetadataSize))
	str = append(str, fmt.Sprintf("reqq: %d", m.Req))
	str = append(str, fmt.Sprintf("v: %s", m.V))
	str = append(str, fmt.Sprintf("yourip: %s", net.IP(m.YourIP)))
	str = append(str, fmt.Sprintf("ipv4: %s", net.IP(m.Ipv4)))
	str = append(str, fmt.Sprintf("ipv6: %s", net.IP(m.Ipv6)))
	str = append(str, fmt.Sprintf("p: %d", m.P))

	return strings.Join(str, "\n")
}

package peers

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"strconv"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func Unmarshal(peersBin []byte) ([]Peer, error) {
	const peerSize = 6
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		return nil, fmt.Errorf("received malformed peers")
	}
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBin[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16([]byte(peersBin[offset+4 : offset+6]))
	}
	return peers, nil
}

func StringToPeer(address string) (Peer, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return Peer{}, fmt.Errorf("invalid address format: %w", err)
	}

	ip, err := netip.ParseAddr(host)
	if err != nil {
		return Peer{}, fmt.Errorf("invalid IP address: %w", err)
	}

	prt, err := strconv.Atoi(port)
	if err != nil {
		return Peer{}, fmt.Errorf("error parsing port: %w", err)
	}

	return Peer{
		IP:   net.IP(ip.AsSlice()),
		Port: uint16(prt),
	}, nil
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}

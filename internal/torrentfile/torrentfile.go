package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/magnet"
	infohash "github.com/codecrafters-io/bittorrent-starter-go/internal/types"
	"github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Announce    string
	InfoHash    infohash.T
	PieceHashes []infohash.T
	PieceLength int
	Length      int
	Name        string
}

type bencodeInfo struct {
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	bto := bencodeTorrent{}
	err = bencode.Unmarshal(file, &bto)
	if err != nil {
		return TorrentFile{}, err
	}
	return bto.toTorrentFile()
}

func ToTorrentFile(m magnet.Magnet, payload []byte) (TorrentFile, error) {
	bi := bencodeInfo{}

	if err := bencode.Unmarshal(bytes.NewReader(payload), &bi); err != nil {
		return TorrentFile{}, fmt.Errorf("error unmarshalling bencode stream")
	}

	t, err := bi.toTorrentFile()

	t.Announce = m.Trackers[0]

	return t, err
}

func (i *bencodeInfo) hash() (infohash.T, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return infohash.T{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (i *bencodeInfo) splitPieceHashes() ([]infohash.T, error) {
	buf := []byte(i.Pieces)

	if len(buf)%infohash.Size != 0 {
		return nil, fmt.Errorf("received malformed pieces of length %d", len(buf))
	}
	numHashes := len(buf) / infohash.Size
	hashes := make([]infohash.T, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*infohash.Size:(i+1)*infohash.Size])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}
	t := TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
	}
	return t, nil
}

func (bi *bencodeInfo) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bi.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bi.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}

	t := TorrentFile{
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bi.PieceLength,
		Length:      bi.Length,
		Name:        bi.Name,
	}

	return t, nil
}

func (t TorrentFile) Info() string {
	var str []string
	str = append(str, fmt.Sprintf("Tracker URL: %s", t.Announce))
	str = append(str, fmt.Sprintf("Length: %d", t.Length))
	str = append(str, fmt.Sprintf("Info Hash: %s", t.InfoHash))
	str = append(str, fmt.Sprintf("Piece Length: %d", t.PieceLength))
	str = append(str, fmt.Sprint("Piece Hashes:"))
	for _, hash := range t.PieceHashes {
		str = append(str, fmt.Sprintf("%x", hash))
	}

	return strings.Join(str, "\n")
}

func (t TorrentFile) NumPieces() int {
	return len(t.PieceHashes)
}

func (bi bencodeInfo) String() string {
	var str []string
	str = append(str, fmt.Sprintf("Piece Length: %d", bi.PieceLength))
	pieceHashes, _ := bi.splitPieceHashes()
	str = append(str, fmt.Sprintln("Piece Hashes: "))
	for pieceHash := range pieceHashes {
		str = append(str, fmt.Sprintln(pieceHash))
	}
	str = append(str, fmt.Sprintf("Name: %s", bi.Name))
	str = append(str, fmt.Sprintf("Length: %d", bi.Length))

	return strings.Join(str, "\n")
}

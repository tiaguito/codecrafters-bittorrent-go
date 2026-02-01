package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	"github.com/jackpal/bencode-go"
)

const HASHLEN int = 20

type TorrentFile struct {
	Announce    string
	InfoHash    [HASHLEN]byte
	PieceHashes [][HASHLEN]byte
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

func (i *bencodeInfo) hash() ([HASHLEN]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [HASHLEN]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][HASHLEN]byte, error) {
	hashLen := HASHLEN
	buf := []byte(i.Pieces)

	if len(buf)%hashLen != 0 {
		return nil, fmt.Errorf("received malformed pieces of length %d", len(buf))
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
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

func (t TorrentFile) String() string {
	var str []string
	str = append(str, fmt.Sprintf("Tracker URL: %s", t.Announce))
	str = append(str, fmt.Sprintf("Length: %d", t.Length))
	str = append(str, fmt.Sprintf("Info Hash: %x", t.InfoHash))
	str = append(str, fmt.Sprintf("Piece Length: %d", t.PieceLength))
	str = append(str, fmt.Sprint("Piece Hashes:"))
	for _, hash := range t.PieceHashes {
		str = append(str, fmt.Sprintf("%x", hash))
	}

	return strings.Join(str, "\n")
}

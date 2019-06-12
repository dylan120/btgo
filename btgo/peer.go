package btgo

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
)

type Peer struct {
	ID       []byte        `bencode:"ID,omitempty"`
	IP       net.IP        `bencode:"ip"`
	Port     int           `bencode:"port"`
	Info     Info          `bencode:"Info,omitempty"`
	InfoHash []byte        `bencode:"InfoHash,omitempty"`
	SavePath string        `bencode:"SavePath,omitempty"`
	sock     io.ReadWriter `bencode:",omitempty"`
}

func (peer *Peer) Connect(peerIP, peerPort string) (conn net.Conn, err error) {
	addr := net.JoinHostPort(peerIP, peerPort)
	conn, err = net.DialTimeout("tcp", addr, NewConnectionTimeout)
	if err != nil {
		//fmt.Println("dial error:", err)
		return
	}
	Handshake(conn, peer.ID, peer.InfoHash)
	return
}

func (peer *Peer) handleConn(conn net.Conn, lastKeepAlive int) (err error) {
	clientDesc, _, infoHash, _, err := ParseHandshake(conn)
	if err != nil {
		if err != io.EOF {
			return err
		}
	}

	if bytes.Compare(peer.InfoHash, infoHash) != 0 {
		conn.Close() //TODO if failed
		return errors.New("info hash does not match")
	}
	fmt.Println("\nclient:", clientDesc, "\ninfohash :", infoHash)
	return

}

func (peer *Peer) Server() {
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", peer.Port))
	if err != nil {
		fmt.Println("listen error: ", err)
		return
	}

	for {
		var lastKeepAlive int
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("accept error: ", err)
			break
		}
		go peer.handleConn(conn, lastKeepAlive)
	}
}

func (peer *Peer) VerifyFiles() (bitField []byte, err error) {
	info := peer.Info
	files := GetFiles(info)

	var (
		index, offset       = 0, 0
		field         uint8 = 0
	)
	bitField = append(bitField, field)
	bitFieldInfoMap := make(map[string]BitFieldInfo)
	for _, sf := range files {
		var tfi *os.File
		pieceNum := math.Ceil(float64(sf.Length) / float64(info.PieceLength))
		capacity := offset + int(pieceNum)
		pieces := info.Pieces[offset:capacity]
		target := filepath.Join(peer.SavePath, sf.Path[len(sf.Path)-1]+".bt.xltd")

		if _, err := os.Stat(target); os.IsNotExist(err) {
			tfi, err = os.Create(target)
			if err != nil {
				return nil, err
			}
			defer tfi.Close()
		} else {
			tfi, err = os.Open(target)
			if err != nil {
				return nil, err
			}
			defer tfi.Close()
		}

		stat, err := tfi.Stat()
		if err != nil {
			return nil, err
		}

		for i, buf, reader := 0, make([]byte, info.PieceLength), bufio.NewReader(tfi); ; i += 20 {
			h := sha1.New()
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			h.Write(buf)
			sha1 := h.Sum(nil)
			if bytes.Compare(pieces[i:i+20], sha1) == 0 {
				fieldLength := len(bitField)
				if index < fieldLength*8 {
					c := uint(8 - (fieldLength*8 - index))
					bitField[fieldLength-1] |= 1 << c

				} else {
					bitField = append(bitField, uint8(0))
				}
			}
			bitFieldInfoMap[target] = BitFieldInfo{
				Index:    index,
				Path:     target,
				Sha1Hash: sha1,
			}

			if int64(n) < info.PieceLength {
				break
			}
			index += 1
		}

		if sf.Md5sum != "" {
			md5, err := MD5sum(target)
			if err != nil {
				return nil, err
			}
			if md5 == sf.Md5sum {
				fmt.Println("file ", stat.Name(), " finished")
			}
		}
		offset = capacity
	}
	return
}

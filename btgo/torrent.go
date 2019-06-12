package btgo

import (
	"../btgo/bencode"
	"bufio"
	"crypto/sha1"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Info struct {
	Name        string `bencode:"name"`
	PieceLength int64  `bencode:"piece length"`
	Pieces      []byte `bencode:"pieces"`
	Length      int64  `bencode:"length,omitempty"`
	Files       []File `bencode:"files,omitempty"`
	File        string `bencode:"file,omitempty"`
}

type NodeAddr string

func (n *NodeAddr) UnmarshalBencode(d []byte) (err error) {
	var if_ interface{}
	err = bencode.Unmarshal(d, &if_)
	//fmt.Println(if_)
	switch v := if_.(type) {
	case []interface{}:
		func() {
			defer func() {
				r := recover()
				if r != nil {
					err = r.(error)
				}
			}()
			*n = NodeAddr(net.JoinHostPort(v[0].(string), strconv.Itoa(int(v[1].(int64)))))
		}()
	}
	return err
}

type MetaInfo struct {
	Announce     string     `bencode:"announce"`
	AnnounceList [][]string `bencode:"announce-list"`
	Nodes        []NodeAddr `bencode:"nodes,omitempty"`
	Info         Info       `bencode:"info"`
	CreationDate int64      `bencode:"creation date,omitempty"`
	Comment      string     `bencode:"comment,omitempty"`
	CreatedBy    string     `bencode:"created by,omitempty"`
}

type Torrent struct {
	MetaInfo MetaInfo
}

func (info *Info) GenPieces(files []File) (err error) {
	var (
		fi     *os.File
		pieces []byte
	)
	for _, f := range files {
		fi, err = os.Open(
			strings.Join(f.Path, string(filepath.Separator)))
		defer fi.Close()
		if err != nil {
			return
		}
		for buf, reader := make([]byte, info.PieceLength), bufio.NewReader(fi); ; {
			h := sha1.New()
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			h.Write(buf)
			pieces = h.Sum(pieces)
			if int64(n) < info.PieceLength {
				break
			}
		}
	}
	info.Pieces = pieces
	return
}

func NewTorrent(jid string, files []string, announceList [][]string) (t *Torrent, err error) {
	info := Info{Name: jid, PieceLength: PieceLength}
	metaInfo := MetaInfo{
		Info:         info,
		AnnounceList: announceList,
		Comment:      "",
		CreatedBy:    "go agent",
		CreationDate: time.Now().Unix()}
	for _, f := range files {
		fi, err := os.Stat(f)
		if err != nil {
			return nil, err
		}
		switch mode := fi.Mode(); {
		case mode.IsRegular():
			if err != nil {
				return nil, err
			}
			md5, err := MD5sum(f)
			if err != nil {
				return nil, err
			}
			metaInfo.Info.Files = append(
				metaInfo.Info.Files,
				File{
					Length: fi.Size(),
					Path:   strings.Split(f, string(filepath.Separator)),
					Md5sum: md5})
			err = metaInfo.Info.GenPieces(metaInfo.Info.Files)
			if err != nil {
				return nil, err
			}

		default:
			fmt.Println("directory")
		}
	}

	sort.Slice(info.Files, func(i, j int) bool {
		return strings.Join(info.Files[i].Path, "/") < strings.Join(info.Files[j].Path, "/")
	})

	x, _ := bencode.Marshal(metaInfo)
	tfile, err := os.Create(jid + ".torrent")
	if err != nil {
		return
	}
	defer tfile.Close()
	tfile.Write(x)
	log.Errorf("%s", string(x))
	return

}

func BuildTorrentFromFile(torrentFile string) (t *Torrent, err error) {
	var (
		c []byte
		f *os.File
	)
	t = &Torrent{}
	f, err = os.Open(torrentFile)
	if err != nil {
		return
	}
	defer f.Close()

	for buf, reader := make([]byte, 256*1024), bufio.NewReader(f); ; {
		_, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		c = append(c, buf...)
	}
	mi := MetaInfo{}
	err = bencode.Unmarshal(c, &mi)
	if err != nil {
		fmt.Println(err)
		return
	}

	t.MetaInfo = mi
	return
}

package btgo

import (
	"../btgo/bencode"
	"bytes"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	Peer         Peer
	AnnounceList [][]string
	MetaInfo     MetaInfo
}

func genPeerID() []byte {
	h := sha1.New()
	h.Write([]byte(time.Now().String()))
	return h.Sum(nil)
}

func NewClient(torrentFile string, savePath string, listenPort int) (*Client, error) {
	cli := Client{}
	t, err := BuildTorrentFromFile(torrentFile)
	if err != nil {
		return nil, err
	}
	cli.MetaInfo = t.MetaInfo

	buf, err := bencode.Marshal(cli.MetaInfo.Info)
	if err != nil {
		return nil, err
	}
	//m, _ := json.Marshal(cli.MetaInfo)
	//fmt.Println("cli.MetaInfo.Info",string(buf))
	h := sha1.New()
	h.Write(buf)
	cli.Peer = Peer{ID: genPeerID(), Port: listenPort,
		InfoHash: h.Sum(nil), Info: cli.MetaInfo.Info, SavePath: savePath}
	cli.AnnounceList = t.MetaInfo.AnnounceList
	//j, _ := json.Marshal(cli.MetaInfo)
	//fmt.Println(string(j), len(cli.MetaInfo.Info.Pieces))
	fmt.Printf("%08b", cli.Peer.InfoHash)
	return &cli, nil
}

func (cli *Client) RequestTracker() (body []byte, err error) {
	for {
		fmt.Println("Request trackers....")
		for _, announces := range cli.AnnounceList {
			for _, announce := range announces {
				var peers []Peer
				params := url.Values{}
				url, err := url.Parse(announce)
				if err != nil {
					//fmt.Println(err)
					continue
				}
				//fmt.Printf("cli.Peer.InfoHash %08b",cli.Peer.InfoHash)
				params.Set("info_hash", string(cli.Peer.InfoHash))
				params.Set("peer_id", string(cli.Peer.ID))
				params.Set("port", fmt.Sprintf("%d", cli.Peer.Port))
				params.Set("compact", "1")
				url.RawQuery = params.Encode()
				urlPath := url.String()
				resp, err := http.Get(urlPath)
				if err != nil {
					//fmt.Println(err)
					continue
				}
				defer resp.Body.Close()
				body, err = ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Println(err)
					//return nil, err
				} else {
					t := &TrackerResp1{}
					err = bencode.Unmarshal(body, &t)
					if err != nil {
						t := &TrackerResp2{}
						err = bencode.Unmarshal(body, &t)
						fmt.Println(t)
						if err != nil {
							fmt.Println(err)
						}
						peers = t.ParsePeers()

					} else {
						peers = t.Peers
					}
					for _, p := range peers {
						InfoHashPeers[string(cli.Peer.InfoHash)] <- p
					}
				}
			}
		}

		time.Sleep(time.Duration(60) * time.Second)
	}
	return
}

func CheckPiece(piece []byte, sha1Hash []byte) bool {
	h := sha1.New()
	_, err := h.Write(piece)
	if err != nil {
		return false
	}
	if bytes.Compare(sha1Hash, h.Sum(nil)) != 0 {
		return false
	}
	return true

}

type BitFieldInfo struct {
	Path     string
	Index    int
	Sha1Hash []byte
}

var (
	InfoHashPeers = make(map[string]chan Peer)
)

func (cli *Client) Run() (err error) {
	InfoHashPeers[string(cli.Peer.InfoHash)] = make(chan Peer, 3)
	go cli.RequestTracker()
	go func() {
		for peer := range InfoHashPeers[string(cli.Peer.InfoHash)] {
			go func() {
				fmt.Println(peer, cli.Peer.InfoHash)
				cli.Peer.Connect(peer.IP.String(), strconv.Itoa(peer.Port))
				//if err != nil {
				//	fmt.Print(err)
				//}
			}()
		}
	}()

	if EnableDHT {
		node := NewNode(cli.MetaInfo.Nodes)
		node.Run(cli.Peer.InfoHash)
		//peers = InfoHashPeers[string(cli.Peer.InfoHash)]
	}
	cli.Peer.Server()

	bitField, err := cli.Peer.VerifyFiles()
	if err != nil {
		fmt.Println(err)
		return err
	}
	msg, err := MarshalMessage(Message{ID: BitField, BitField: bitField})
	fmt.Println("||||>>>>", msg)
	if err != nil {
		return err
	}

	select {}
	return
}

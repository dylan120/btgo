package btgo

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type TrackerResp1 struct {
	Interval    int    `bencode:"interval"`
	FailReason  string `bencode:"failure reason,omitempty"`
	Peers       []Peer `bencode:"peers"`
	Complete    int    `bencode:"complete"`
	Incomplete  int    `bencode:"incomplete"`
	MinInterval int    `bencode:"min interval"`
	//Downloaded  int    `bencode:"downloaded"`
}

type TrackerResp2 struct {
	Interval    int    `bencode:"interval"`
	FailReason  string `bencode:"failure reason,omitempty"`
	Peers       []byte `bencode:"peers"`
	Complete    int    `bencode:"complete"`
	Incomplete  int    `bencode:"incomplete"`
	MinInterval int    `bencode:"min interval"`
	//Downloaded  int    `bencode:"downloaded"`
}

func (t2 *TrackerResp2) ParsePeers() (peers []Peer) {
	r := bytes.NewBuffer(t2.Peers)
	for _, reader := make([]byte, 6), bufio.NewReader(r); ; {
		buf := make([]byte, 6)
		_, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
		}
		port := int(binary.BigEndian.Uint16(buf[4:]))
		peers = append(peers, Peer{IP: net.IP(buf[0:4]), Port: port})
	}

	return
}

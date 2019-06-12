package main

import (
	"fmt"
	"github.com/anacrolix/dht/krpc"
	"github.com/anacrolix/torrent/bencode"
	"math"
	"testing"
)

func TestEncode(t *testing.T) {
	//s,_:=btgo.MarshalKRPCMessage(btgo.KRPCMessage{T: []byte("aa"), Y: "q", Q: "ping", A: btgo.KRPCParam{ID: "abcdefghij0123456789"}})

	//assert.EqualValues(t, "d1:ad2:id20:abcdefghij0123456789e1:q4:ping1:t2:aa1:y1:qe", string(s))
	data := []byte("d1:rd2:id20:abcdefghij01234567895:nodes9:def456...5:token8:aoeusnthe1:t2:aa1:y1:re")
	k := krpc.Return{}
	bencode.Unmarshal(data,k)
	fmt.Println(k)
	fmt.Println(math.Floor(math.Sqrt(15)))

	var b interface{}

	ss := []byte("l13:203.202.47.11i17821ee")
	fmt.Println(bencode.Unmarshal(ss,&b))
	fmt.Println("xxxxxx",b)
}

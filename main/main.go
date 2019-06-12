package main

import (
	"../btgo"
	"fmt"
)

func testRecursive(index int) int {
	s := []int{1, 2, 3}
	if index > 3 {
		return index
	}

	for i := range s {
		r := testRecursive(i)
		fmt.Println("testRecursive xxxxx", r)
	}
	return index
}

func main() {


	//cli, err := btgo.NewClient("F:\\Go\\test_go\\2.torrent", "F:\\Go\\downloads", 6881)
	//cli, err := btgo.NewClient("C:\\Users\\Administrator\\Desktop\\1.torrent", "F:\\Go\\downloads", 6881)
	cli, err := btgo.NewClient("C:\\Users\\Administrator\\Downloads\\517FCDD3B97A43F92EAC8696A432DB4A193D6B1B.torrent", "C:\\Users\\Administrator\\Downloads", 6881)
	//cli, err := btgo.NewClient("C:\\Users\\Administrator\\Downloads\\新编中文版Premiere Pro CC标准教程.pdf.torrent", "C:\\Users\\Administrator\\Downloads", 6881)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("start ")
		//fmt.Print(btgo.MarshalKRPCMessage(btgo.KRPCMessage{T: []byte("aa"), Y: "q", Q: "ping", A: btgo.KRPCQueryParam{ID:"abcdefghij0123456789"}}))
		fmt.Println(cli.Run())
	}
}

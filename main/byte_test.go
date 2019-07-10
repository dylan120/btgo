package main

import (
	"../btgo"
	"fmt"
	"testing"
)

func TestByte(t *testing.T) {
	var s = "-ga0001-"
	b := btgo.GenerateID(12)
	b = append([]byte(s), b...)
	//s := "ab"
	//fmt.Println([]byte(s))
	fmt.Printf("%08b", b)
}

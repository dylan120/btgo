package main

import (
	"../btgo"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestBep40Priority(t *testing.T) {
	prio, _ := btgo.GetPriority("123.213.32.10", 0, "98.76.54.32", 0)
	assert.EqualValues(t, 0xec2d7224, prio)
	assert.EqualValues(t, "\x00\x00\x00\x00", func() []byte {
		b, _ := btgo.GetPriorityBytes("123.213.32.10", 0, "123.213.32.10", 0)
		return b
	}())
	assert.EqualValues(t, 0xec2d7224, btgo.Bep40PriorityIgnoreError(
		btgo.IpPort{net.ParseIP("123.213.32.10"), 0},
		btgo.IpPort{net.ParseIP("98.76.54.32"), 0},
	))

	//var b []byte
	//v :=reflect.ValueOf(b)
	//fmt.Println(v.SetBytes([]byte{"sdfd"}))
}

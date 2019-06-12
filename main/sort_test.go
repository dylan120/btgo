package main

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

type File struct {
	Length int64    `bencode:"length"`
	Path   []string `bencode:"path"`
	Md5sum string   `bencode:"md5sum"`
}

type fileValues []File

func (f fileValues) Len() int           { return len(f) }
func (f fileValues) Less(i, j int) bool { return strings.Join(f[i].Path, "/") < strings.Join(f[j].Path, "/") }
func (f fileValues) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }

func TestSort(t *testing.T) {
	a:=File{Length:1,Path:[]string{"c","aestb"}}
	b:=File{Length:1,Path:[]string{"b","testb"}}

	fs1 := []File{a,b}

	fmt.Println(fs1)
	sort.Sort(fileValues(fs1))
	fmt.Println(fs1)


	c:=File{Length:1,Path:[]string{"c","aestb"}}
	d:=File{Length:1,Path:[]string{"b","testb"}}

	fs2 := []File{c,d}

	fmt.Println(fs2)
	sort.Slice(fs2, func(i, j int) bool {
		return strings.Join(fs2[i].Path, "/") < strings.Join(fs2[j].Path, "/")
	})
	fmt.Println(fs2)

	fmt.Println(3/float32(2))
}

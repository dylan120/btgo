package btgo

import (
	"sort"
	"strings"
)

type File struct {
	Length int64    `bencode:"length"`
	Path   []string `bencode:"path"`
	Md5sum string   `bencode:"md5sum,omitempty"`
}

func GetFiles(info *Info) (fs []File) {
	sort.Slice(info.Files, func(i, j int) bool {
		return strings.Join(info.Files[i].Path, "/") < strings.Join(info.Files[j].Path, "/")
	})

	if len(info.Files) == 0 {
		fs = append(fs, File{Path: []string{info.Name}, Length: info.Length})
	} else {
		fs = info.Files
	}
	return fs
}

func SplitFile() {

}

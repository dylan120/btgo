package btgo

type File struct {
	Length int64    `bencode:"length"`
	Path   []string `bencode:"path"`
	Md5sum string   `bencode:"md5sum,omitempty"`
}

func GetFiles(info Info) (fs []File) {
	if len(info.Files) == 0 {
		fs = append(fs, File{Path: []string{info.Name}, Length: info.Length})
	} else {
		fs = info.Files
	}
	return fs
}

func SplitFile() {

}

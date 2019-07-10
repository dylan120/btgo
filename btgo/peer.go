package btgo

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

var (
	piecesInfo     = make(map[string]map[int]*Piece)
	piecesInfoLock sync.Mutex
	peerConns      = make(map[string]net.Conn)
	blockQueue     = make(chan []interface{}, 100)
)

type Peer struct {
	ID       []byte        `bencode:"ID,omitempty"`
	IP       net.IP        `bencode:"ip"`
	Port     int           `bencode:"port"`
	Info     *Info         `bencode:"Info,omitempty"`
	InfoHash []byte        `bencode:"InfoHash,omitempty"`
	SavePath string        `bencode:"SavePath,omitempty"`
	sock     io.ReadWriter `bencode:",omitempty"`
	BitField []byte        `bencode:"bit_field,omitempty"`
}

func NewPeer(listenPort int, infoHash []byte, info *Info, savePath string) *Peer {
	bitField, _ := VerifyFiles(info, string(infoHash), savePath)
	peer := Peer{ID: genPeerID(), Port: listenPort,
		InfoHash: infoHash, Info: info, SavePath: savePath, BitField: bitField}
	return &peer
}

func (peer *Peer) keepalive() {
	for {
		for _, conn := range peerConns {
			msg := make([]byte, 4)
			//msg, err := MarshalMessage(Message{ID: Interested})
			//if err != nil {
			//	fmt.Println(err)
			//	continue
			//}
			//msg = append(make([]byte, 4), msg...)
			_, err := conn.Write(msg)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println("send Keepalive")
		}
		time.Sleep(120 * time.Second)
	}
}

func (peer *Peer) Connect(addr string) (conn net.Conn, err error) {
	//RemoteAddr, err := net.ResolveUDPAddr("udp", addr)
	//conn, err = net.DialUDP("udp", nil, RemoteAddr)
	//if err !=nil{
	//	fmt.Println(err)
	//	conn, err = net.DialTimeout("tcp", addr, NewConnectionTimeout)
	//}
	conn, err = net.DialTimeout("tcp", addr, NewConnectionTimeout)

	if err != nil {
		return nil, err
	}
	peerConns[addr] = conn
	err = Handshake(conn, peer.ID, peer.InfoHash)
	go peer.HandleConn(conn, string(peer.InfoHash))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return
}

func (peer *Peer) ReadBytes(conn net.Conn, buf *bytes.Buffer) (b []byte, err error) {
	readByteCount := 65536
	n := 0
	b = make([]byte, 0)
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	for {
		b = make([]byte, readByteCount)
		n, err = conn.Read(b)
		if err != nil {
			break
		}
		buf.Write(b[:n])
		if n < readByteCount {
			break
		}
		n = 0
	}
	return b, err
}

func (peer *Peer) blockLength(index, begin, fileLength int64) int64 {
	var length int64 = 0
	if fileLength < BlockLength {
		return fileLength
	}

	if BlockLength > peer.Info.PieceLength {
		length = int64(peer.Info.PieceLength)
	} else {
		length = BlockLength
	}
	left := fileLength - int64(index)*peer.Info.PieceLength
	if left < peer.Info.PieceLength {
		length = left
	}
	return length
}

var pieceQueue = make(chan *Piece, 50)

func (peer *Peer) PushbitField2PieceQueue(bitField []byte, infoHash string, addr string) {
	for i, b := range bitField {
		for j := 0; j < 8; j++ {
			c := byte(1 << uint(7-j))
			if b&c == c {
				index := uint32(i*8 + j)
				piecesInfoLock.Lock()
				if piece, ok := piecesInfo[infoHash][int(index)]; ok {
					piece.PeerAddrs = append(piece.PeerAddrs, addr)
					piecesInfo[infoHash][int(index)] = piece
					pieceQueue <- piece
				}
				piecesInfoLock.Unlock()
			}
		}
	}
}

func (peer *Peer) PushPiece2PieceQueue(index int, infoHash string, addr string) {
	piecesInfoLock.Lock()
	if piece, ok := piecesInfo[infoHash][index]; ok {
		piece.PeerAddrs = append(piece.PeerAddrs, addr)
		piecesInfo[infoHash][index] = piece
		pieceQueue <- piece
	}
	piecesInfoLock.Unlock()
}

func (peer *Peer) downloadBlock(infoHash string) {
	msg, _ := MarshalMessage(Message{ID: Interested})
	b := make([]byte, 0)
	b = append(b, msg...)
	rate := time.Second / 1
	throttle := time.Tick(rate)
	for {
		<-throttle
		piece := <-pieceQueue
		fmt.Println(time.Now().String(), "start download piece", piece.Index)
		for _, block := range piece.Blocks {
			msg, _ := MarshalMessage(
				Message{ID: Request, Index: piece.Index, Offset: block.Offset, Length: block.BlocksLength})
			b = append(b, msg...)
		}
		for _, addr := range piece.PeerAddrs {
			conn := peerConns[addr]
			_, err := conn.Write(b)
			if err != nil {
				fmt.Println(err)
				continue
			}
			break
		}
		b = b[:0]
	}
}

func PushBlockQueue(index int, block *Block) {
	blockQueue <- []interface{}{index, block}
}

func PopBlockQueue(infoHash string) {
	for {
		bq := <-blockQueue
		index := bq[0].(int)
		block := bq[1].(*Block)
		length := 0

		piecesInfoLock.Lock()
		piece := piecesInfo[infoHash][index]
		piece.Blocks = append(piece.Blocks, block)
		piecesInfo[infoHash][index] = piece
		for _, block := range piece.Blocks {
			length += len(block.Data)
		}
		if length == piece.Length {
			c := make([]byte, 0)
			sort.Slice(piece.Blocks, func(i, j int) bool {
				return piece.Blocks[i].Offset < piece.Blocks[j].Offset
			})
			for _, block := range piece.Blocks {
				c = append(c, block.Data...)
			}

			h := sha1.New()
			h.Write(c)
			sha1 := h.Sum(nil)
			if bytes.Compare(piece.PieceHash, sha1) == 0 {
				fmt.Println(time.Now().String(), "verify piece", index, " successfully")
				fi, err := os.OpenFile(piece.Path, os.O_WRONLY, 0666)
				if err == nil {
					_, err := fi.WriteAt(c, int64(piece.Index)*piece.PieceLength)
					fmt.Println(err)
					fi.Close()
					delete(piecesInfo[infoHash], index)
					HavePiece(uint32(index))
					fmt.Println("piecesInfo[infoHash] length left", len(piecesInfo[infoHash]))
				}
			} else {
				fmt.Println("verify piece failed")
			}
		}
		piecesInfoLock.Unlock()
	}
}

func HavePiece(index uint32) {
	for _, conn := range peerConns {
		msg, _ := MarshalMessage(
			Message{ID: Have, PieceIndex: index})
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("send Have piece", index)
	}
}

func HandleRequest() {

}

func (peer *Peer) HandleConn(conn net.Conn, infoHash string) (err error) {
	buf := bytes.NewBuffer(make([]byte, 0))
	reader := bufio.NewReaderSize(bufio.NewReader(buf), 2*BlockLength)
	fmt.Println("===========")
	for {
		peer.ReadBytes(conn, buf)
		b, err := reader.Peek(4)
		if err != nil {
			continue
		}
		length := binary.BigEndian.Uint32(b)

		if length == 0 {
			continue
		}
		b, err = reader.Peek(5)
		if err != nil {
			fmt.Println(err)
			continue
		}
		msgID := b[4]
		if length == 1 {
			switch msgID {
			case UnChoke:
				fmt.Println("receive UnChoke message")
				go peer.downloadBlock(infoHash)
				reader.Discard(5)
				continue
			case Interested:
				fmt.Println("receive Interested message")
				reader.Discard(5)
				continue
			case Choke:
				fmt.Println("receive Choke message")
				reader.Discard(5)
				continue
			case NotInterested:
				fmt.Println("receive NotInterested message")
				reader.Discard(5)
				continue
			}
		}
		switch msgID {
		case Request:
			fmt.Println("receive Request message", length)
			reader.Discard(int(length) + 4) //TODO
		case Cancel:
			fmt.Println("receive Cancel message", length)
			reader.Discard(int(length) + 4) //TODO
		case Have:
			reader.Discard(5)
			b := make([]byte, 4)
			reader.Read(b)
			index := binary.BigEndian.Uint32(b)
			go peer.PushPiece2PieceQueue(int(index), infoHash, conn.RemoteAddr().String())
		case Piece_:
			_, err := reader.Peek(int(length))
			if err != nil {
				//re := make([]byte, int(length)+4)
				//fmt.Println("piece continue", err, re, length)
				continue
			}
			reader.Discard(5)
			b := make([]byte, 4)
			reader.Read(b)
			index := binary.BigEndian.Uint32(b)
			reader.Read(b)
			offset := binary.BigEndian.Uint32(b)
			block := make([]byte, length-9)
			_, err = reader.Read(block)
			if err != nil {
				fmt.Println(err)
				continue
			}
			go PushBlockQueue(int(index), &Block{Offset: offset, Data: block})

		case BitField:
			_, err := reader.Peek(int(length) + 4)
			if err != nil {
				fmt.Println(err)
				continue
			}
			reader.Discard(5)
			bitField := make([]byte, length-1)
			reader.Read(bitField)
			go peer.PushbitField2PieceQueue(bitField, infoHash, conn.RemoteAddr().String())
		case Port:
			_, err := reader.Peek(int(length) + 4)
			if err != nil {
				fmt.Println(err)
				continue
			}
			reader.Discard(5)
			port := make([]byte, 2)
			reader.Read(port)
			//fmt.Println("receive Port message", port)
		}
		_, err = reader.Peek(1)
		if err != nil {
			buf.Reset()
		}
		//buf.Reset()
	}
	return
}

func (peer *Peer) Server() {
	runtime.GOMAXPROCS(4)
	go peer.keepalive()
	go PopBlockQueue(string(peer.InfoHash))
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", peer.Port))
	if err != nil {
		fmt.Println("listen error: ", err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
		}
		go peer.HandleConn(conn, string(peer.InfoHash))
	}

}

func VerifyFiles(info *Info, infoHash string, savePath string) (bitField []byte, err error) {
	pieceMap := make(map[int]*Piece)
	piecesInfo[infoHash] = pieceMap
	files := GetFiles(info)
	var (
		index, offset = 0, 0
	)

	for _, sf := range files {
		var tfi *os.File
		target := filepath.Join(savePath, sf.Path[len(sf.Path)-1])

		if _, err := os.Stat(target); os.IsNotExist(err) {
			tfi, err = os.Create(target)
			err = tfi.Truncate(sf.Length)
			if err != nil {
				return nil, err
			}
			defer tfi.Close()
		} else {
			tfi, err = os.Open(target)
			if err != nil {
				return nil, err
			}
			defer tfi.Close()
		}

		stat, err := tfi.Stat()
		if err != nil {
			return nil, err
		}
		for i, buf, reader := 0, make([]byte, info.PieceLength), bufio.NewReader(tfi); ; i += 20 {
			h := sha1.New()
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			buf = buf[:n]
			h.Write(buf)
			sha1 := h.Sum(nil)

			length := len(bitField)
			if index >= length*8 {
				bitField = append(bitField, uint8(0))
			}
			//fmt.Println("index", index, len(buf), bitField[len(bitField)-1])
			pieceHash := info.Pieces[offset : offset+20]
			if bytes.Compare(pieceHash, sha1) == 0 {
				c := uint(8 - (len(bitField)*8 - index))
				bitField[len(bitField)-1] |= 1 << c
				//bitMap = append(bitMap,true)

			} else {
				var blockLength uint32 = BlockLength
				bufLength := len(buf)
				reqTime := uint32(math.Ceil(float64(bufLength) / BlockLength))

				blocks := make([]*Block, 0)
				for i := uint32(0); i < reqTime; i++ {
					offset := i * BlockLength
					if (i*BlockLength + BlockLength) > uint32(bufLength) {
						blockLength = uint32(len(buf[i*BlockLength:]))
					}
					blocks = append(blocks, &Block{Offset: offset, BlocksLength: blockLength})
				}
				//bitMap = append(bitMap,false)
				piecesInfo[infoHash][index] = &Piece{
					Index: uint32(index), Path: target, PieceHash: pieceHash, PieceLength: info.PieceLength,
					Length: bufLength, Blocks: blocks}
			}
			index += 1
			offset += 20
			if int64(n) < info.PieceLength {
				break
			}
		}

		if sf.Md5sum != "" {
			md5, err := MD5sum(target)
			if err != nil {
				return nil, err
			}
			if md5 == sf.Md5sum {
				fmt.Println("file ", stat.Name(), " finished")
			}
		}
	}
	fmt.Println("bitField", bitField)
	fmt.Println(piecesInfo[infoHash])
	return bitField, err
}

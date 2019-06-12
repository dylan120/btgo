package btgo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

type Message struct {
	Index, Begin, Length uint32
	ID                   byte
	Payload              []byte
	PieceIndex           uint32
	BitField             []byte
	Block                []byte
}

func Handshake(conn io.ReadWriter, peerId []byte, infoHash []byte) (msg []byte) {
	doneChan := make(chan error)
	sendChan := make(chan []byte, 4)

	defer func() {
		close(sendChan)
	}()

	go SendByte(conn, sendChan, doneChan)
	fmt.Println("start Handshake")
	var m []byte
	m = append(m, []byte("\x13" + ProtocolName)...)
	m = append(m, make([]byte, 8)...)
	m = append(m, infoHash...)
	m = append(m, peerId...)
	sendChan <- m

	//sendChan <- []byte("\x13" + ProtocolName)
	//reservedByte := make([]byte, 8)
	//
	//if EnableDHT {
	//	reservedByte[7] |= 1
	//}
	//sendChan <- reservedByte
	//sendChan <- infoHash
	//sendChan <- peerId

	buf := make([]byte, 100)
	n, err := conn.Read(buf)
	fmt.Println(n, err)
	return
}

func ParseHandshake(conn net.Conn) (clientDesc string, reservedBit []byte,
	infoHash []byte, peerId []byte, err error) {
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err != nil {
		return
	}
	psLen := int(buf[0])
	totalByteNum := psLen + HandshakeReservedBitLen + InfoHashLen
	buf = make([]byte, totalByteNum)
	err = binary.Read(conn, binary.BigEndian, buf)
	if err != nil {
		return "", nil, nil, nil, err
	}
	clientDesc = string(buf[:psLen])
	infoHash = buf[psLen+HandshakeReservedBitLen : totalByteNum]
	return
}

func MarshalMessage(msg Message) (data []byte, err error) {
	buf := &bytes.Buffer{}

	fmt.Print(BitField, msg.ID)
	err = buf.WriteByte(msg.ID)
	if err != nil {
		return
	}
	switch msg.ID {
	//case Choke, UnChoke, Interested, NotInterested:
	case Request:
		for d := range []uint32{msg.Index, msg.Begin, msg.Length} {
			err = binary.Write(buf, binary.BigEndian, d)
			if err != nil {
				break
			}
		}
	case Have:
		err = binary.Write(buf, binary.BigEndian, msg.PieceIndex)
		if err != nil {
			break
		}
	case Piece:
		for d := range []uint32{msg.Index, msg.Begin} {
			err = binary.Write(buf, binary.BigEndian, d)
			if err != nil {
				break
			}

		}

		n, err := buf.Write(msg.Block)
		if err != nil {
			return nil, err
		}
		if n != len(msg.Block) {
			return nil, errors.New("write block length != piece message")
		}
	case BitField:
		err = binary.Write(buf, binary.BigEndian, msg.BitField)
		if err != nil {
			break
		}
	}

	data = make([]byte, 4+buf.Len())
	binary.BigEndian.PutUint32(data, uint32(4+buf.Len()))
	copy(data, buf.Bytes())
	return
}

func MarshalKeepAlive() (data []byte) {
	binary.BigEndian.PutUint32(data, uint32(0))
	return
}

package btgo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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

func Handshake(conn net.Conn, peerId []byte, infoHash []byte) (err error) {
	doneChan := make(chan error)
	sendChan := make(chan []byte, 4)

	defer func() {
		close(sendChan)
	}()

	go SendByte(conn, sendChan, doneChan)
	fmt.Println("start Handshake")
	sendChan <- []byte("\x13" + ProtocolName)
	reservedByte := make([]byte, 8)

	if EnableDHT {
		reservedByte[7] |= 1
	}
	sendChan <- reservedByte
	sendChan <- infoHash
	sendChan <- peerId

	p, reservedBit, _, peerId, err := ParseHandshake(conn)
	fmt.Println(p, reservedBit)

	return err
}

func ParseHandshake(conn net.Conn) (clientDesc string, reservedBit []byte,
	infoHash []byte, peerID []byte, err error) {
	//conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err != nil {
		return "", nil, nil, nil, nil
	}
	psLen := int(buf[0])

	buf = make([]byte, psLen)
	_, err = conn.Read(buf)
	if err != nil {
		return "", nil, nil, nil, nil
	}
	clientDesc = string(buf[:])

	reservedBit = make([]byte, HandshakeReservedBitLen)
	_, err = conn.Read(reservedBit)
	if err != nil {
		return "", nil, nil, nil, nil
	}
	infoHash = make([]byte, 20)
	_, err = conn.Read(infoHash)
	if err != nil {
		return "", nil, nil, nil, nil
	}

	peerID = make([]byte, 20)
	_, err = conn.Read(peerID)
	if err != nil {
		fmt.Println(err)
		return "", nil, nil, nil, nil
	}
	return clientDesc, reservedBit, infoHash, peerID, nil
}

func MarshalMessage(msg Message) (data []byte, err error) {
	buf := &bytes.Buffer{}
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
	binary.BigEndian.PutUint32(data, uint32(buf.Len()))
	copy(data[4:], buf.Bytes())
	return
}

func MarshalKeepAlive() (data []byte) {
	binary.BigEndian.PutUint32(data, uint32(0))
	return
}

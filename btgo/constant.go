package btgo

import "time"

const (
	//BEP0003
	Choke byte = iota
	UnChoke
	Interested
	NotInterested
	Have
	BitField
	Request
	Piece
	Cancel

	ProtocolName = "BitTorrent protocol"

	PieceLength             = 256 * 1024 //256KB
	InfoHashLen             = 20
	HandshakeReservedBitLen = 8
	KeepAliveInterval       = 60 * 60 //2 min

	EnableDHT = true

	NewConnectionTimeout = 5 * time.Second
	CacheDir             = "J:\\GO\\test_go\\cache"
)

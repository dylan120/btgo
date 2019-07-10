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
	Piece_
	Cancel
	Port

	ProtocolName            = "BitTorrent protocol"
	ClientName              = "-ga0001-"
	PieceLength             = 256 * 1024 //256KB
	BlockLength             = 32 * 1024  //32KB
	InfoHashLen             = 20
	HandshakeReservedBitLen = 8
	KeepAliveInterval       = 60 * 60 //2 min

	EnableDHT = true

	NewConnectionTimeout = 5 * time.Second

	CacheDir = "F:\\Go\\test_go\\btgo\\cache"
)

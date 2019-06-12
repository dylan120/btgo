package btgo

func BootstrapNodes() ([]NodeAddr) {
	bootstrapNodes := []NodeAddr{
		"router.utorrent.com:6881",
		"router.bittorrent.com:6881",
		"dht.transmissionbt.com:6881",
		"dht.aelitis.com:6881",
		"router.silotis.us:6881",
		"dht.libtorrent.org:25401",
	}
	return bootstrapNodes
}

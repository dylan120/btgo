package btgo

import (
	"../btgo/bencode"
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	K        = 8
	ping     = "ping"
	findNode = "find_node"
	getPeers = "get_peers"
)

type DistanceRange struct {
	Start float64
	End   float64
}

type Node struct {
	ID   []byte
	IP   net.IP
	Port int
}

type Nodes []Node

func (n Nodes) Len() int           { return len(n) }
func (n Nodes) Less(i, j int) bool { return bytes.Compare(n[i].ID, n[j].ID) == -1 }
func (n Nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

type KBucket struct {
	Neighbours []Node
}

type KRPCParam struct {
	ID       []byte `bencode:"id,omitempty"`
	NodeID   []byte `bencode:"node_id,omitempty"`
	InfoHash []byte `bencode:"info_hash,omitempty"`
	Target   []byte `bencode:"target,omitempty"`
	Nodes    string `bencode:"nodes,omitempty"`
}

type KRPCMessage struct {
	T []byte    `bencode:"t,omitempty"`
	Y string    `bencode:"y,omitempty"`
	Q string    `bencode:"q,omitempty"`
	A KRPCParam `bencode:"a,omitempty"`
	R KRPCParam `bencode:"r,omitempty"`
}

type KRPCRValue struct {
	ID     string   `bencode:"id,omitempty"`
	Token  []byte   `bencode:"token,omitempty"`
	Values []string `bencode:"values,omitempty"`
	Nodes  string   `bencode:"nodes,omitempty"`
}

type KRPCResponse struct {
	T     []byte     `bencode:"t,omitempty"`
	Y     string     `bencode:"y,omitempty"`
	E     string     `bencode:"e,omitempty"`
	R     KRPCRValue `bencode:"r,omitempty"`
	Token []byte     `bencode:"token,omitempty"`
}

var (
	routingTable = make([]KBucket, 160)
)


func (b *KBucket) insert() {

}

func updateKBuket(nodes []Node, targetID []byte) {
	for _, nd := range nodes {
		distance, _ := Distance(nd.ID, targetID)
		index := Index(distance)
		kBuket := routingTable[index]
		fmt.Println(" try to update kbucket")
		if !NodeExist(kBuket.Neighbours, nd) {
			kBuket.Neighbours = append(kBuket.Neighbours, nd)
			kBuket.Neighbours = QuickSortByTargetID(kBuket.Neighbours, targetID)
			fmt.Println("update kbucket")
			if len(kBuket.Neighbours) > K {
				kBuket.Neighbours = kBuket.Neighbours[:K]
			}
			routingTable[index] = kBuket
		}
	}
}

func NewNode(nodeAddrs []NodeAddr) *Node {
	node := &Node{
		ID: GenerateID(20),
	}
	node.Init(nodeAddrs)
	return node
}

func (node *Node) InitRoutingTable() {

}

func (node *Node) UpdateRoutingTable() {}

func DistanceToInt(distance []byte) (out uint64) {
	big := &big.Int{}
	big.SetBytes(distance)
	return big.Uint64()
}

func Distance(a, b []byte) ([]byte, error) {
	n := len(a)
	if len(b) != len(a) {
		return nil, errors.New("node a length != node b length")
	}
	dst := make([]byte, n)
	for i := 0; i < n; i++ {

		dst[i] = a[i] ^ b[i]
	}
	return dst, nil
}

func Index(distance []byte) (out int) {
	mid := &big.Int{}
	mid.SetBytes(distance)
	one := big.NewInt(1)
	two := big.NewInt(2)

	for out = 1; ; out++ {
		mid.Div(mid, two)
		n := mid.Cmp(one)
		if n == 0 || n == -1 {
			break
		}
	}
	return out
}

func handleConn(conn *net.UDPConn) (err error) {
	for {
		buf := make([]byte, 65536)
		err = binary.Read(conn, binary.BigEndian, buf)
		if err != nil {
			fmt.Println(err)
		}
		print("dhtServer receive", buf)
		time.Sleep(time.Duration(500) * time.Millisecond)
	}

}

func (node *Node) dhtServer() {
	listen, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0})
	if err != nil {
		fmt.Println("listen error: ", err)
		return
	}
	//a := strings.Split(listen.LocalAddr().String(), ":")
	fmt.Println(">>>>>>>>>>>>>>>>", listen.LocalAddr().String())
	go handleConn(listen)
}

func QuickSortByTargetID(nodes []Node, targetID []byte) []Node {
	if len(nodes) < 2 {
		return nodes
	}
	left, right := 0, len(nodes)-1

	pivotIndex := rand.Int() % len(nodes)

	nodes[pivotIndex], nodes[right] = nodes[right], nodes[pivotIndex]

	r, _ := Distance(nodes[right].ID, targetID)
	for i := range nodes {
		l, _ := Distance(nodes[i].ID, targetID)
		if DistanceToInt(l) < DistanceToInt(r) {
			nodes[i], nodes[left] = nodes[left], nodes[i]
			left++
		}
	}
	nodes[left], nodes[right] = nodes[right], nodes[left]
	QuickSortByTargetID(nodes[:left], targetID)
	QuickSortByTargetID(nodes[left+1:], targetID)
	return nodes
}

func ClosedNeighbours(requestID, targetID []byte) ([]Node, error) {
	distance, err := Distance(requestID, targetID)
	if err != nil {
		return nil, err
	}
	index := Index(distance)
	neighbours := routingTable[index].Neighbours
	length := len(neighbours)

	for i, j := index-1, index+1; length < K; i, j = i-1, j+1 {
		neighbours = append(neighbours, routingTable[i].Neighbours...)
		neighbours = append(neighbours, routingTable[j].Neighbours...)
		length = len(neighbours)
	}

	neighbours = QuickSortByTargetID(neighbours, targetID)
	if len(neighbours) > K {
		neighbours = neighbours[:K]
	}
	return neighbours, nil
}

func Connect(addr string) (conn *net.UDPConn, err error) {
	RemoteAddr, err := net.ResolveUDPAddr("udp", addr)
	conn, err = net.DialUDP("udp", nil, RemoteAddr)
	if err != nil {
		return nil, err
	}
	return
}

func MarshalKRPCMessage(msg KRPCMessage) (data []byte, err error) {
	data, err = bencode.Marshal(msg)
	return
}

func (node *Node) Ping() {

}

func Response(conn *net.UDPConn) (data []byte, err error) {
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	for {
		buf := make([]byte, 65536)
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return data, err
		}
		if n < 1024 {
			data = append(data, buf[0:n]...)
			break
		}
		data = append(data, buf...)
	}
	return data, nil
}

var wg sync.WaitGroup

func (node *Node) Init(boostNodes []NodeAddr) (err error) {
	requestedAddrs := make([]NodeAddr, 0)
	boostNodes = append(boostNodes, BootstrapNodes()...)
	getNodes := func(addrs []NodeAddr, targetID []byte, nodes []Node) []Node {
		fmt.Println("getNodes===================")
		for _, addr := range addrs {
			requestedAddrs = append(requestedAddrs, addr)
			wg.Add(1)
			add := make([]byte, len([]byte(addr)))
			copy(add, []byte(addr))
			go func() {
				ns, _ := FindNodes(string(add), node.ID, targetID, 0)
				nodes = append(nodes, ns...)
			}()
		}
		wg.Wait()
		fmt.Println("allNodes", len(nodes))
		nodes = QuickSortByTargetID(nodes, node.ID)
		return nodes
	}
	targetID := node.ID
	nodes := getNodes(boostNodes, targetID, make([]Node, 0))
	for {
		if len(nodes) == 0 {
			break
		}

		addrs := make([]NodeAddr, 0)
		for _, n := range nodes {
			addr := NodeAddr(net.JoinHostPort(n.IP.String(), strconv.Itoa(n.Port)))
			if SliceExists(requestedAddrs, addr) {
				continue
			}
			addrs = append(addrs, addr)
		}
		nodes = getNodes(addrs, targetID, nodes[:0])
	}
	return err
}

func getAllNodes(infoHash [] byte) []Node {
	neighbours := make([]Node, 0)
	//fmt.Println(routingTable)
	for _, bucket := range (routingTable) {
		if len(bucket.Neighbours) != 0 {
			neighbours = append(neighbours, bucket.Neighbours...)
		}
	}
	neighbours = QuickSortByTargetID(neighbours, infoHash)
	return neighbours
}

func (node *Node) Run(infoHash []byte) (err error) {
	go node.dhtServer()
	runtime.GOMAXPROCS(runtime.NumCPU())
	neighbours := getAllNodes(infoHash) //[:K]
	if len(neighbours) == 0 {
		return errors.New("zero neighbours")
	}
	if len(neighbours) > K {
		neighbours = neighbours[:K]
	}
	peers := make([]Peer, 0)

	tmpNodes := make([]Node, K)

	index := 0
	for {
		if index+K > len(neighbours) {
			tmpNodes = neighbours[index:]
			index = 0
			neighbours = getAllNodes(infoHash)
			existNodes = existNodes[:0]
		} else {
			tmpNodes = neighbours[index : index+K]
			index = index + K
		}
		peers = GetPeers(tmpNodes, node.ID, infoHash)
		if len(peers) != 0 {
			for _, p := range peers {
				//a := net.JoinHostPort(p.IP.String(), strconv.Itoa(p.Port))
				//_, err = net.DialTimeout("tcp", a, time.Millisecond*500)
				//fmt.Println(err)
				if err == nil {
					InfoHashPeers[string(infoHash)] <- p
					//fmt.Println(a, "ok")
				}
			}

		}
	}
	return nil
}

var requestedNodes = make(map[string]bool)
var requestedAddrs = make([]string, 0)

var existNodes = make([]Node, 0)

func NodeExist(nodes []Node, target Node) bool {
	a := net.JoinHostPort(target.IP.String(), strconv.Itoa(target.Port))
	for _, n := range nodes {
		b := net.JoinHostPort(n.IP.String(), strconv.Itoa(n.Port))
		if a == b {
			//fmt.Println(a, b)
			return true
		}
	}
	return false

}

func GetPeers(nodes []Node, nodeId, targetID []byte) ([]Peer) {
	getPeers := func(addr string, NodeID, infoHash []byte) ([]Node, []Peer) {
		nodes, peers := make([]Node, 0), make([]Peer, 0)
		conn, err := Connect(string(addr))
		if err != nil {
			return nodes, peers
		}

		data, _ := MarshalKRPCMessage(
			KRPCMessage{T: []byte("ww"), Y: "q", Q: getPeers, A: KRPCParam{ID: NodeID, InfoHash: infoHash}})
		if err != nil {
			fmt.Println("MarshalKRPCMessage", err)
		}
		_, err = conn.Write(data)
		if err != nil {
			return nodes, peers
		}
		response, err := Response(conn)
		if err != nil {
			return nodes, peers
		}

		resp := KRPCResponse{}
		bencode.Unmarshal(response, &resp)
		if len(resp.R.Values) != 0 {
			//fmt.Println(resp.R.Values)
			for _, val := range resp.R.Values {
				ip := make(net.IP, len(val)-2)
				port := make([]byte, 2)
				copy(ip, val[:len(val)-2])
				if err != nil {
					fmt.Println(err)
				}
				if string(val) != "" {
					copy(port, val[len(val)-2:])
					port := binary.BigEndian.Uint16(port)
					peers = append(peers, Peer{
						IP:   ip,
						Port: int(port),
					})
				}
			}

		} else {
			bufs := bytes.NewBuffer([]byte(resp.R.Nodes))
			for buf, reader := make([]byte, 26), bufio.NewReader(bufs); ; {
				n, err := reader.Read(buf)
				if err != nil {
					if err == io.EOF {
						break
					}
					fmt.Println(err)
				}
				if n < 26 {
					break
				}
				nodeID := make([]byte, 20)
				NodeIP := make(net.IP, 4)
				NodePort := make([]byte, 2)
				copy(nodeID, buf[0:20])
				copy(NodeIP, buf[20:24])
				copy(NodePort, buf[24:26])
				newNode := Node{ID: nodeID, IP: NodeIP, Port: int(binary.BigEndian.Uint16(NodePort))}
				nodes = append(nodes, newNode)
			}
			if len(nodes) != 0 {
				updateKBuket(nodes, targetID)
			}

		}
		nodes = QuickSortByTargetID(nodes, targetID)
		return nodes, peers
	}
	peers := make([]Peer, 0)
	tmpNodes := make([]Node, 0)
	for _, node := range nodes {
		addr := net.JoinHostPort(node.IP.String(), strconv.Itoa(node.Port))
		if NodeExist(existNodes, node) {
			fmt.Println("NodeExist", node)
			continue
		}
		existNodes = append(existNodes, node)
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
			}()
			ns, ps := getPeers(addr, nodeId, targetID)
			peers = append(peers, ps...)
			if len(ns) != 0 {
				tmpNodes = append(tmpNodes, ns...)
			}

		}()

	}
	wg.Wait()
	if len(peers) != 0 {
		return peers
	}
	fmt.Println("tmpNodes", len(tmpNodes))
	fmt.Println("tmpNodes====================")
	if len(tmpNodes) != 0 {
		nodes = QuickSortByTargetID(append(nodes, tmpNodes...), targetID)
		if len(nodes) > K {
			nodes = nodes[:K]
		}

		return GetPeers(nodes, nodeId, targetID)
	} else {
		fmt.Println("tmpNodes empty")
	}

	return peers
}

func FindNodes(addr string, nodeID, targetID [] byte, i int) (nodes []Node, err error) {
	defer func() {
		wg.Done()
	}()
	conn, err := Connect(addr)
	if err != nil {
		return nil, err
	}
	data, _ := MarshalKRPCMessage(
		KRPCMessage{T: []byte("ww"), Y: "q", Q: findNode, A: KRPCParam{ID: nodeID, Target: targetID}})
	_, err = conn.Write(data)
	if err != nil {
		return nil, err
	}

	response, err := Response(conn)
	if err != nil {
		return nil, err
	}
	resp := KRPCResponse{}
	bencode.Unmarshal(response, &resp)

	if resp.R.Nodes != "" {
		bufs := bytes.NewBuffer([]byte(resp.R.Nodes))

		for buf, reader := make([]byte, 26), bufio.NewReader(bufs); ; {
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Println(err)
				continue
			}
			if n < 26 {
				break
			}
			nID := make([]byte, 20)
			nIP := make(net.IP, 4)
			nPort := make([]byte, 2)
			copy(nID, buf[0:20])
			copy(nIP, buf[20:24])
			copy(nPort, buf[24:26])
			distance, _ := Distance(nID, nodeID)
			index := Index(distance)

			newNode := Node{ID: nID, IP: nIP, Port: int(binary.BigEndian.Uint16(nPort))}
			nodes = append(nodes, newNode)
			kBuket := routingTable[index]
			if len(kBuket.Neighbours) < K && !NodeExist(kBuket.Neighbours, newNode) {
				kBuket.Neighbours = append(kBuket.Neighbours, newNode)
				routingTable[index] = kBuket
			}
		}
	} else {
		return nil, errors.New("empty nodes")
	}
	return nodes, err
}

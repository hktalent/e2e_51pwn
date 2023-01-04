package util

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	util1 "github.com/hktalent/go-utils"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/muxer/mplex"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// "/ipfs/peerIdC/p2p-circuit/ipfs/peerIdB"
var (
	NCnt      = 0
	RelayPort = "44479"
	E2EServer = E2EServerUrl
	RelayId   = "12D3KooWFdYnUVrpPbtFVHZv3vzmEQUkJ2YUCwCh5dPscfKoa7KE"
	/*
		base64解码，并转换为hex
		QmWjz6xb8v9K4KnYEwP5Yk75k5mMBCehzWFLCvvQpYxF3d
		4d577a3678613876394b344b6e5945775035596b37356b356d4d424365687a57464c437676517946583364
	*/
	E2eReplayServer = "/ip4/" + E2EServer + "/tcp/" + RelayPort + "/p2p/" + RelayId
)

// https://github.com/libp2p/go-libp2p/blob/master/examples/ipfs-camp-2019/05-Discovery/protocol.go
const (
	E2EServerUrl = "e2e.51pwn.com"
	E2eProtocol  = protocol.ID("/" + E2EServerUrl + "/1.0.0")
	E2eUrl       = "https://" + E2EServerUrl + "/e2e"
	// 用于加密，固定 host.ID 不变
	E2eKey = "7B9ABA76-6C03-4517-856C-4648AAA5406A"
	// DiscoveryServiceTag is used in our mDNS advertisements to discover other chat peers.
	DiscoveryServiceTag = "e2e-51pwn-penetration"
	TcpPort             = 65530
)

func init() {
	util1.RegInitFunc(func() {
		a := util1.GetIps(E2EServer)
		if 0 < len(a) {
			E2EServer = a[0]
			fnInitVars()
			log.Println("E2EServer ip:", E2EServer)
		}
	})
}

// e2e 通讯封装
type E2e51pwn struct {
	NCnt              int
	Host              host.Host
	Ctx               context.Context
	KademliaDHT       *kaddht.IpfsDHT
	PeerCbk           func(*E2e51pwn, *bufio.ReadWriter, network.Stream)
	E2eProtocolCbk    func(*E2e51pwn, *bufio.ReadWriter, network.Stream)
	RelayPeerInfo     *peer.AddrInfo
	EnableRelayServer bool
	IsServer          bool
	Client            *sync.Map
	ClientLists       []string
}

// 基于 p2p 通讯
func NewE2e51pwn(ctx context.Context) *E2e51pwn {
	cbk := func(r *E2e51pwn, rw *bufio.ReadWriter, s network.Stream) {
		log.Println("PeerCbk、E2eProtocolCbk -> ", s.Conn().RemotePeer(), s.Conn().RemoteMultiaddr().String())
	}
	e1 := &E2e51pwn{Client: &sync.Map{}, NCnt: 0, Ctx: ctx, PeerCbk: cbk, E2eProtocolCbk: cbk, EnableRelayServer: true}
	//e1.initHost()
	return e1
}

func fnInitVars() {
	E2eReplayServer = "/ip4/" + E2EServer + "/tcp/" + RelayPort + "/p2p/" + RelayId
}

// 注册服务器信息
func (r *E2e51pwn) regReplayServerInfo(s string) {
	a := strings.Split(s, "/")
	log.Printf("%+v\n", a)
	szPost := `p=` + a[len(a)-1] + `&k=` + r.Host.ID().String() + ``
	log.Println("reg post data", szPost)
	util1.PipE.DoGetWithClient4SetHd(nil, E2eUrl, "POST", strings.NewReader(szPost), func(resp *http.Response, err error, szU string) {
		if nil == err {
			resp.Body.Close()
		}
		log.Println("regReplayServerInfo ok", resp.StatusCode)
	}, func() map[string]string {
		return map[string]string{"Content-Type": "application/x-www-form-urlencoded; charset=UTF-8"}
	}, true)
}

// 获取中继服务器信息
func (r *E2e51pwn) getReplayServerInfo() {
	util1.PipE.DoGet(E2eUrl, func(resp *http.Response, err error, szU string) {
		var m1 = map[string]string{}
		if nil == err {
			if data, err := ioutil.ReadAll(resp.Body); nil == err {
				if err := json.Unmarshal(data, &m1); nil == err {
					if s, ok := m1["k"]; ok {
						RelayId = s
					}
					if s, ok := m1["p"]; ok {
						RelayPort = s
					}
					fnInitVars()
				} else {
					log.Println("getReplayServerInfo -> json.Unmarshal", err)
				}
			}
		}
	})
}

func (r *E2e51pwn) fncheckIsServer() bool {
	s1 := "/" + E2EServer + "/tcp/"
	for _, x := range r.Host.Addrs() {
		if -1 < strings.Index(x.String(), s1) && !strings.HasSuffix(x.String(), "ws") {
			r.IsServer = true
			r.regReplayServerInfo(x.String())
			return true
		}
	}
	r.getReplayServerInfo()
	return false
}

// 检查当前是否为 中继server
func (r *E2e51pwn) CheckIsServer() bool {
	return r.IsServer
}

func (r *E2e51pwn) RegCbk(cbkOld func(*E2e51pwn, *bufio.ReadWriter), cbk ...func(*E2e51pwn, *bufio.ReadWriter)) {
	old := cbkOld
	cbkOld = func(r1 *E2e51pwn, rw *bufio.ReadWriter) {
		util1.DoSyncFunc(func() {
			for _, x1 := range cbk {
				x1(r1, rw)
			}
			old(r1, rw)
		})
	}
}
func (r *E2e51pwn) RegE2eProtocolCbk(cbk ...func(*E2e51pwn, *bufio.ReadWriter, network.Stream)) {
	old := r.E2eProtocolCbk
	r.E2eProtocolCbk = func(r1 *E2e51pwn, rw *bufio.ReadWriter, s network.Stream) {
		util1.DoSyncFunc(func() {
			for _, x1 := range cbk {
				x1(r1, rw, s)
			}
			old(r1, rw, s)
		})
	}
}
func (r *E2e51pwn) RegPeerCbk(cbk ...func(*E2e51pwn, *bufio.ReadWriter, network.Stream)) {
	old := r.PeerCbk
	r.PeerCbk = func(r1 *E2e51pwn, rw *bufio.ReadWriter, s network.Stream) {
		util1.DoSyncFunc(func() {
			for _, x1 := range cbk {
				x1(r1, rw, s)
			}
			old(r1, rw, s)
		})
	}
}

func (r *E2e51pwn) GetPort() int {
	//r.NCnt++
	//return TcpPort - r.NCnt
	return 0
}

func (r *E2e51pwn) initHost() {
	muxers := libp2p.ChainOptions(
		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
	)
	listenAddrs := libp2p.ListenAddrStrings(
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/ws", r.GetPort()),
		// default
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", r.GetPort()),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", r.GetPort()),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", r.GetPort()),
		fmt.Sprintf("/ip6/::/tcp/%d", r.GetPort()),
		fmt.Sprintf("/ip6/::/udp/%d/quic", r.GetPort()),
		fmt.Sprintf("/ip6/::/udp/%d/quic-v1", r.GetPort()),
	)
	//listenAddrs=libp2p.NoListenAddrs

	// 1. 使用RSA和随机数流创建密钥对
	//privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, randomness)
	/*
		Ed25519和RSA加密算法的主要区别如下：
		1. 加密长度：Ed25519的密钥长度为256位，而RSA的密钥长度可以达到4096位。
		2. 安全性：Ed25519更安全，因为它采用了Elliptic Curve Cryptography（ECC），这是一种强大的加密技术，而RSA是基于大整数乘法的算法。
		3. 速度：Ed25519比RSA加密速度快，因为它的密钥长度更短。
		4. 应用：Ed25519用于密钥交换，数字签名，而RSA则用于加密和数字签名。
		Crypto.Ed25519是基于椭圆曲线算法（ECDSA）的实现，具有更高的安全等级，可以抵御更多的攻击。它是一种更安全的算法
		Crypto.Secp256k1更适合用于传统的金融服务，如跨境支付、网上银行、支付宝等；Crypto.Ed25519更适合用于区块链项目，如比特币、以太坊等
		crypto.ECDSA是椭圆曲线数字签名算法，它利用椭圆曲线算法构建了一个可以用于数字签名的私钥和公钥对。它可以用于用户身份认证、文件签名和交易签名等场景。
		crypto.Ed25519是一个非对称加密算法，它可以用于签名和加密消息，它的安全性比椭圆曲线算法更高，可以用于重要的数字签名和交易签名场景。
	*/
	szIds := util1.GetActiveMac()
	if n1 := len(szIds); 32 > n1 {
		szIds = fmt.Sprintf("%s%s", szIds, strings.Repeat("0", 32-n1))
	}
	log.Println("Mac Hex ", szIds)
	priv, _, err := crypto.GenerateKeyPairWithReader(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		256,            // Select key length when possible (i.e. RSA).
		strings.NewReader(szIds),
	)
	if err != nil {
		log.Println("crypto.GenerateKeyPair ", err)
	}
	// Construct a datastore (needed by the DHT). This is just a simple, in-memory thread-safe datastore.
	routing := libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
		dht := kaddht.NewDHT(r.Ctx, h, dsync.MutexWrap(ds.NewMapDatastore()))
		return dht, nil
	})

	/*
		解释下面 "github.com/libp2p/go-libp2p/p2p/net/connmg“ 各参数的作用
		Lowwater：最低水位，用于定义连接的最小数量。
		HighWater：最高水位，用于定义连接的最大数量。
		WithGracePeriod：定义连接维护的宽限期，在宽限期内，连接的建立和释放不受LowWater和HighWater的限制。
	*/
	connmgr, err := connmgr.NewConnManager(
		100,  // Lowwater
		1000, // HighWater,
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		log.Println("connmgr.NewConnManager ", err)
	}

	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.DefaultTransports, // libp2p.Transport(quic.NewTransport); TCP Transport + quic Transport + ws Transport
		muxers,                   // libp2p.DefaultMuxers,  只有 yamux
		/*
			libp2p.Security(noise.ID, noise.New),是libp2p的安全模块，使用Noise协议来实现安全的通信。
			它可以在libp2p应用中实现安全的通信，以及在多个节点之间传输数据时的身份验证。
			它适用于任何需要安全的P2P网络的应用，如分布式存储，消息传递，点对点文件共享，数据同步等
		*/
		libp2p.Security(tls.ID, tls.New), // support TLS connections
		listenAddrs,                      // libp2p.DefaultListenAddrs,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(connmgr),
		libp2p.EnableRelay(),
		//libp2p.EnableAutoRelay(),
		libp2p.EnableRelayService(),
		routing, // Let this host use the DHT to find other hosts
		/*
			libp2p/go-libp2p 中libp2p.NATPortMap、libp2p.EnableNATService作用是什么，适合哪些场景？
			NATPortMap和EnableNATService是libp2p中用于解决NAT(Network Address Translation, 网络地址转换)穿越问题的函数。
			NATPortMap函数用来将内部网络中的端口映射到外部网络中的一个端口，供外部节点连接。
			EnableNATService函数用于开启NAT穿越服务，它会自动识别NAT类型，并尝试设置NAT映射。
			适合的场景：
			1. 当用户使用NAT网关时，可以使用NATPortMap和EnableNATService函数，实现自动映射，使用libp2p服务。
			2. 当用户使用NAT网关时，可以使用NATPortMap函数，将内部网络中的端口映射到外部网络中的一个端口，供外部节点连接，实现NAT穿越。
		*/
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
	}
	host, err := libp2p.New(opts...)
	if err != nil {
		log.Println("libp2p.New", err)
	}
	r.Host = host

	//log.Println(host.Network().LocalPeer().String())
	// 连接到 replay 穿透服务
	//host.Connect(ctx, *CreatePeerAddr(E2eReplayServer))
	if !r.fncheckIsServer() {
		if err := r.ConnectRelay(E2eReplayServer); nil != err {
			log.Println("r.ConnectRelay ", E2eReplayServer, err)
		} else {
			log.Println("r.ConnectRelay ok")
		}

		if err := r.SetupMdnsDiscovery(); nil != err {
			log.Println("r.SetupMdnsDiscovery ", err)
		} else {
			log.Println("r.SetupMdnsDiscovery ok")
		}
	}
	if r.CheckIsServer() {
		r.RegE2eProtocolCbk(r.RegClient)
	} else {
		r.RegE2eProtocolCbk(r.getClientLists)
	}
	host.SetStreamHandler(E2eProtocol, r.E2eProtocolHandler)
	log.Println("host.ID = ", host.ID(), host.Addrs())
	r.doDht()
	//r.GetP2pPort(multiaddr.P_TCP)
	//_, err = client.Reserve(ctx, unreachable2, relay1info)
	if r.EnableRelayServer {
		_, err = relay.New(host)
	}
	log.Printf("ID: %v Addr: %v\n", host.ID(), host.Addrs())
}

func (r *E2e51pwn) doDht() {
	if kademliaDHT, err := kaddht.New(r.Ctx, r.Host); nil == err {
		r.KademliaDHT = kademliaDHT
		// thread that will refresh the peer table every five minutes.每五分钟刷新一次对等表的线程。
		log.Println("Bootstrapping the DHT")
		if err = kademliaDHT.Bootstrap(r.Ctx); err != nil {
			log.Println("DHT.Bootstrap", err)
		}
		util1.DoSyncFunc(r.FindPeers)
	} else {
		log.Println("kaddht.New", err)
	}
}

func (r *E2e51pwn) CreateHost() host.Host {
	if nil == r.Host {
		r.initHost()
	}
	return r.Host
}

// r.GetP2pPort(multiaddr.P_TCP)
func (r *E2e51pwn) GetP2pPort(nType int) string {
	// Let's get the actual TCP port from our listen multiaddr, in case we're using 0 (default; random available port).
	var port string
	for _, la := range r.Host.Network().ListenAddresses() {
		//log.Println(la.String())
		// ws
		if p, err := la.ValueForProtocol(nType); err == nil {
			port = p
			//log.Println(la.String())
			break
		}
	}
	s1 := fmt.Sprintf("/ip4/0.0.0.0/tcp/%v/p2p/%s", port, r.Host.ID().Pretty())
	log.Println(s1)
	return port
}

// 相同的id就跳过 不连接自己
func (r *E2e51pwn) CheckPeerId(id peer.ID) bool {
	if nil != r.Host && id == r.Host.ID() {
		log.Println(r.Host.ID(), "==", id)
		return true
	}
	return false
}

// call back regPeerCbk
// ns
func (r *E2e51pwn) FindPeers() {
	log.Println("Announcing ourselves...")
	routingDiscovery := drouting.NewRoutingDiscovery(r.KademliaDHT)
	szNs := string(E2eProtocol)
	dutil.Advertise(r.Ctx, routingDiscovery, szNs)
	log.Println("Successfully announced!")
	peerChan, err := routingDiscovery.FindPeers(r.Ctx, szNs)
	if err != nil {
		log.Println("routingDiscovery.FindPeers", err)
		return
	}

	for peer := range peerChan {
		if r.CheckPeerId(peer.ID) {
			continue
		}
		log.Println("Connecting to:", peer)
		if stream, err := r.Host.NewStream(r.Ctx, peer.ID, E2eProtocol); nil == err {
			if rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream)); nil != rw {
				r.PeerCbk(r, rw, stream)
			}
		} else {
			log.Println("FindPeers -> r.Host.NewStream", err)
		}
	}
}

// "/p2p/%s/p2p-circuit/p2p/%s"
func (r *E2e51pwn) ConnectPeers4Relay(aPeer ...string) {
	for i, k := range aPeer {
		aPeer[i] = fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", RelayId, k)
	}
	r.ConnectPeers(aPeer...)
}

// 连接 若干 peer
func (r *E2e51pwn) ConnectPeers(aPeer ...string) {
	var wg sync.WaitGroup
	client.Reserve(r.Ctx, r.Host, *r.RelayPeerInfo)
	for _, x := range aPeer {
		wg.Add(1)
		go func(x1 string) {
			defer wg.Done()
			if maddr, err := multiaddr.NewMultiaddr(x1); nil != err {
				log.Println("multiaddr.NewMultiaddr", err)
				return
			} else {
				if peerinfo, err := peer.AddrInfoFromP2pAddr(maddr); nil == err {
					if r.CheckPeerId(peerinfo.ID) {
						return
					}
					r.Host.Peerstore().AddAddrs(peerinfo.ID, peerinfo.Addrs, peerstore.PermanentAddrTTL)
					if err := r.Host.Connect(r.Ctx, *peerinfo); err != nil {
						log.Println("r.Host.Connect ", err, peerinfo.Addrs, peerinfo.ID)
					} else {
						log.Println("Connection bootstrap node:", *peerinfo)
						if ss, err := r.Host.NewStream(network.WithUseTransient(r.Ctx, E2eKey), peerinfo.ID, E2eProtocol); nil == err {
							// Create a buffered stream so that read and writes are non blocking.
							rw := bufio.NewReadWriter(bufio.NewReader(ss), bufio.NewWriter(ss))
							r.E2eProtocolCbk(r, rw, ss)

							if n, err := rw.Write([]byte("ok")); nil == err {
								log.Println("rw.Write = ", n)
							}
							rw.Flush()
						}
					}
				} else {
					log.Println("peer.AddrInfoFromP2pAddr", err)
				}
			}
		}(x)
	}
	wg.Wait()
}

// 连接到中继服务器，利用中继服务器辅助、快速建立 e2e 连接
func (r *E2e51pwn) ConnectRelay(szRelay string) error {
	relayAddr, err := ma.NewMultiaddr(szRelay)
	if err != nil {
		return err
	}
	pid, err := relayAddr.ValueForProtocol(ma.P_P2P)
	if err != nil {
		return err
	}
	relayPeerID, err := peer.Decode(pid)
	if err != nil {
		return err
	}

	relayPeerAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", pid))
	relayAddress := relayAddr.Decapsulate(relayPeerAddr)

	relayPInfo := peer.AddrInfo{
		ID:    relayPeerID,
		Addrs: []ma.Multiaddr{relayAddress},
	}
	//log.Println(relayPInfo.ID)
	r.RelayPeerInfo = &relayPInfo // r.Ctx
	return r.Host.Connect(r.Ctx, relayPInfo)
}

// 等待 p2p/e2e 连接我
func (r *E2e51pwn) E2eProtocolHandler(s network.Stream) {
	if r.CheckPeerId(s.Conn().RemotePeer()) {
		return
	}
	log.Println("HostStreamHandler ", s.ID(), s.Protocol(), s.Conn().RemoteMultiaddr().String(), s.Conn().ID())
	r.Host.Peerstore().AddAddrs(s.Conn().RemotePeer(), []ma.Multiaddr{s.Conn().RemoteMultiaddr()}, peerstore.PermanentAddrTTL)
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	r.E2eProtocolCbk(r, rw, s)
}

// 客户端注册的一些信息
type ClientData struct {
	Rw     *bufio.ReadWriter `json:"rw"`
	Stream network.Stream    `json:"stream"`
}

// 获取 其他 客户端信息
func (r *E2e51pwn) getClientLists(e *E2e51pwn, rw *bufio.ReadWriter, s network.Stream) {
	var data = make([]byte, 1024*800)
	if n, err := rw.Read(data); 0 < n && nil == err {
		data = data[0:n]
		var a []string
		if err := json.Unmarshal(data, &a); nil == err {
			if 0 < len(a) {
				r.ClientLists = a
				log.Println("other client lists", a)
			}
		}
	}
}

// 注册客户端
func (r *E2e51pwn) RegClient(e *E2e51pwn, rw *bufio.ReadWriter, s network.Stream) {
	szId := s.Conn().RemotePeer()
	if r.CheckPeerId(szId) {
		return
	}
	r.Client.Store(szId, &ClientData{Rw: rw, Stream: s})
	log.Println("RegClient ok: ", s.Conn().RemotePeer(), s.Conn().RemoteMultiaddr())
	var a []string
	s1 := fmt.Sprintf("%v", szId)
	s2 := fmt.Sprintf("%s/p2p/%s", s.Conn().RemoteMultiaddr(), szId)
	var a1 = []string{s2}
	data1, _ := json.Marshal(a1)
	r.Client.Range(func(k, v any) bool {
		if fmt.Sprintf("%v", k) != s1 {
			// 将当前 节点 信息告知在这之前注册的节点
			if o, ok := v.(*ClientData); ok {
				a = append(a, fmt.Sprintf("%s/p2p/%s", o.Stream.Conn().RemoteMultiaddr(), k))
				o.Rw.Write(data1)
				o.Rw.Flush()
			}
		}
		return true
	})
	// 将之前注册的节点信息，告知当前注册的节点
	if 0 < len(a) {
		if data, err := json.Marshal(a); nil == err {
			rw.Write(data)
			rw.Flush()
		}
	}
}

func (r *E2e51pwn) CreatePeerAddr(s string) *peer.AddrInfo {
	targetAddr, err := multiaddr.NewMultiaddr(s)
	if err != nil {
		log.Println("multiaddr.NewMultiaddr", err)
	}

	targetInfo, err := peer.AddrInfoFromP2pAddr(targetAddr)
	if err != nil {
		log.Println("peer.AddrInfoFromP2pAddr", err)
	}
	return targetInfo
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func (r *E2e51pwn) SetupMdnsDiscovery() error {
	s := mdns.NewMdnsService(r.Host, DiscoveryServiceTag, r)
	return s.Start()
}

func (m *E2e51pwn) HandlePeerFound(pi peer.AddrInfo) {
	if m.Host.Network().Connectedness(pi.ID) != network.Connected {
		fmt.Printf("Found %s!\n", pi.ID.ShortString())
		if err := m.Host.Connect(m.Ctx, pi); nil != err {
			log.Println("HandlePeerFound -> m.Host.Connect", err)
		}
	}
}

package main

import (
	"bufio"
	"context"
	"github.com/hktalent/e2e_51pwn/pkg/util"
	util1 "github.com/hktalent/go-utils"
	"github.com/libp2p/go-libp2p/core/network"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// 12D3KooWCRDnTaehHkj3EiMZAUaSt29iDJKQQx2agXcLqmu5zNZN
func main() {
	util1.DoInitAll()
	app := kingpin.New("E2E/P2P network", "001")
	//oMod := app.Flag("isServer", "isServer").Default("false").Short('s').Bool()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e2e := util.NewE2e51pwn(ctx)
	e2e.RegE2eProtocolCbk(func(pwn *util.E2e51pwn, rw *bufio.ReadWriter, s network.Stream) {
		log.Println("e2e.RegE2eProtocolCbk ->", s.Conn().RemotePeer(), s.Protocol(), s.Conn().RemoteMultiaddr())
	})
	//e2e.EnableRelayServer = false
	host := e2e.CreateHost()
	//e2e.ConnectPeers4Relay("103.174.136.111/tcp/35711")
	if !e2e.CheckIsServer() {
		e2e.ConnectPeers("/ip4/103.174.136.111/tcp/" + util.RelayPort + "/p2p/" + util.RelayId) //util.E2eReplayServer
	}
	defer host.Close()

	// https://github.com/libp2p/go-libp2p/blob/master/examples/routed-echo/bootstrap.go#L21
	// https://github.com/libp2p/go-libp2p/blob/master/examples/ipfs-camp-2019/05-Discovery/protocol.go
	//
	//err = dht.Bootstrap(ctx)
	//if err != nil {
	//	panic(err)
	//}
	//
	//routingDiscovery := drouting.NewRoutingDiscovery(dht)
	//dutil.Advertise(ctx, routingDiscovery, string(chatProtocol))
	//peers, err := dutil.FindPeers(ctx, routingDiscovery, string(chatProtocol))
	//if err != nil {
	//	panic(err)
	//}
	//for _, peer := range peers {
	//	notifee.HandlePeerFound(peer)
	//}
	//
	//fmt.Println("Connected to", targetInfo.ID)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	host.Close()

	util1.Wg.Wait()
	util1.CloseAll()
	//select {}
}

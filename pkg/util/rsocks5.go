package util

import (
	"bufio"
	"github.com/armon/go-socks5"
	util "github.com/hktalent/go-utils"
	"io"
	"log"
	"net"
)

var NPoolNum = 1000

type RSocks5Imp struct {
	Socks5Ip string
}

// 127.0.0.1:0 实现动态端口，避免端口被占用的情况
// :0 也可以
func GetAddr(s, szType string) string {
	if "udp" == szType {
		udpAddr, err := net.ResolveUDPAddr("udp4", s)
		if err != nil {
			log.Fatal(err)
		}
		if l, err := net.ListenUDP("udp", udpAddr); nil == err {
			l.Close()
			return l.LocalAddr().String()
		}
		return ""
	}
	l, err := net.Listen(szType, s)
	if err != nil {
		log.Fatalf("err: %v", err)
		return ""
	}
	defer l.Close()
	x1 := l.Addr().(*net.TCPAddr)
	return x1.String()
}

func NewRSocks5(rw *bufio.ReadWriter) *RSocks5Imp {
	var r = &RSocks5Imp{}
	r.Socks5Ip = GetAddr("0.0.0.0:0", "tcp")
	if server, err := socks5.New(&socks5.Config{}); nil == err {
		go server.ListenAndServe("tcp", r.Socks5Ip)
		log.Println("socks5 listen is ok ", r.Socks5Ip)
		if nil != rw {
			r.PipData(rw)
		}
	}
	return r
}

func (r *RSocks5Imp) PipData(rw *bufio.ReadWriter) {
	if nil != rw {
		util.DoSyncFunc(func() {
			if conn, err := net.Dial("tcp", r.Socks5Ip); nil == err {
				go io.Copy(rw, conn)
				go io.Copy(conn, rw)
			}
		})
	}
}

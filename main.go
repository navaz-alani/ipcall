package main

import (
	"flag"
	"log"
	"net"
	"time"
)

var (
	port       = flag.Int("port", 3000, "listen port for client")
	svrPort    = flag.Int("svr-port", 10000, "server to communicate with")
	clientPort = flag.Int("client-port", 3001, "server to communicate with")

  laddrStr = flag.String("listen-addr", "", "listen-address of the client")
  svrAddrStr = flag.String("svr-addr", "", "address of the server")
  clientAddrStr = flag.String("client-addr", "", "address of the client")
)

func main() {
	flag.Parse()
  if *laddrStr == "" || *svrAddrStr == "" || *clientAddrStr == "" {
    log.Fatalln("usage: voip -listen-addr=<laddr> -svr-addr=<svrAddr> -client-addr=<clienrAddr>")
  }
  var laddr, svrAddr *net.UDPAddr
  var err error
  svrAddr, err = net.ResolveUDPAddr("udp", *svrAddrStr)
  chkErr("svrAddr resolve fail: ", err)
  laddr, err = net.ResolveUDPAddr("udp", *laddrStr)
  chkErr("laddr resolve fail: ", err)
	client, err := NewClient(svrAddr, laddr)
	chkErr("client init fail: ", err)
	for {
		time.Sleep(2 * time.Second)
		client.OpenAudioChan(nil, *clientAddrStr)
	}
}

func chkErr(prefix string, err error) {
	if err != nil {
		log.Fatalln(prefix, err.Error())
	}
}

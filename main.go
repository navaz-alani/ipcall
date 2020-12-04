package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
)

var (
	laddrStr      = flag.String("listen-addr", "", "listen-address of the client")
	svrAddrStr    = flag.String("svr-addr", "", "address of the server")
	clientAddrStr = flag.String("client-addr", "", "address of the client")
)

func main() {
	flag.Parse()
	if *laddrStr == "" || *svrAddrStr == "" || *clientAddrStr == "" {
		log.Fatalln("usage: voip -listen-addr=<laddr> -svr-addr=<svrAddr> -client-addr=<clienrAddr>")
	}
	var laddr, svrAddr *net.UDPAddr
	var err error
	{
		svrAddr, err = net.ResolveUDPAddr("udp", *svrAddrStr)
		chkErr("svrAddr resolve fail: ", err)
		laddr, err = net.ResolveUDPAddr("udp", *laddrStr)
		chkErr("laddr resolve fail: ", err)
	}
	client, err := NewClient(svrAddr, laddr)
	chkErr("client init fail: ", err)
	// prompt for call start
	fmt.Printf("Press any key to begin call with \"%s\"\n", *clientAddrStr)
	bufio.NewReader(os.Stdin).ReadRune()
	// execute call
	callDone := make(chan struct{})
	go client.OpenAudioChan(callDone, *clientAddrStr)
	fmt.Printf("Call started. Press Ctrl+C to end.\n")
	// handle closure
	killChan := make(chan os.Signal)
	signal.Notify(killChan, os.Interrupt, os.Kill)
	<-killChan
	callDone <- struct{}{}
}

func chkErr(prefix string, err error) {
	if err != nil {
		log.Fatalln(prefix, err.Error())
	}
}

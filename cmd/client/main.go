package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/navaz-alani/ipcall"
)

var (
	secure        = flag.Bool("secure", true, "whether to use transport layer & end-to-end encryption")
	sampleRate    = flag.Int("sample-rate", 8000, "audio sample rate (higher -> more throughput)")
	laddrStr      = flag.String("l-addr", "", "listen-address of the client")
	svrAddrStr    = flag.String("s-addr", "", "address of the server")
	clientAddrStr = flag.String("c-addr", "", "address of the client")
)

func handleFlags() {
	flag.Parse()
	if *laddrStr == "" || *svrAddrStr == "" || *clientAddrStr == "" {
		fmt.Printf(
			`usage: %s -l-addr=<listen-address> -s-addr=<server-address> -c-addr=<client-address>`,
			os.Args[0],
		)
		fmt.Printf("\n")
		os.Exit(1)
	}
}

func main() {
	handleFlags()
	var laddr, svrAddr *net.UDPAddr
	var err error
	{
		svrAddr, err = net.ResolveUDPAddr("udp", *svrAddrStr)
		chkErr("svrAddr resolve fail: ", err)
		laddr, err = net.ResolveUDPAddr("udp", *laddrStr)
		chkErr("laddr resolve fail: ", err)
	}
	client, err := ipcall.NewClient(svrAddr, laddr, *sampleRate, *secure)
	chkErr("client init fail: ", err)
	// prompt for call start
	fmt.Printf("Press any key to begin call with \"%s\"\n", *clientAddrStr)
	bufio.NewReader(os.Stdin).ReadRune()
	// execute call
	callDone := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		chkErr("audio channel error: ", client.OpenAudioChan(callDone, *clientAddrStr))
	}()
	fmt.Printf("Call started. Press Ctrl+C to end.\n")
	// handle closure
	killChan := make(chan os.Signal)
	signal.Notify(killChan, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-killChan
	fmt.Printf("Ending call...\n")
	callDone <- struct{}{}
	wg.Wait()
	fmt.Printf("Call ended.\n")
}

func chkErr(prefix string, err error) {
	if err != nil {
		log.Fatalln(prefix, err.Error())
	}
}

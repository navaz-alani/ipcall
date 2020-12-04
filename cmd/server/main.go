package main

import (
	"log"
	"net"

	"github.com/navaz-alani/concord/core/throttle"
	"github.com/navaz-alani/concord/packet"
	"github.com/navaz-alani/concord/server"
)

func main() {
	// instantiate server
	addr := &net.UDPAddr{
		IP:   []byte{0, 0, 0, 0},
		Port: 10000,
	}
	svr, err := server.NewUDPServer(addr, 10000, &packet.JSONPktCreator{}, throttle.Rate10k)
	if err != nil {
		log.Fatalln("Failed to initialize server")
	}
	// run server
	log.Println("Listening on port ", addr.Port)
	svr.Serve()
}

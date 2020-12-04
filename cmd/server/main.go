package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"log"
	"net"

	"github.com/navaz-alani/concord/core/crypto"
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
	// generate private key
	privKey, err := ecdsa.GenerateKey(crypto.Curve, rand.Reader)
	if err != nil {
		log.Fatalln("Failed to generate public key: " + err.Error())
	}
	// initialize Crypto extension
	cr, err := crypto.NewCrypto(privKey)
	if err != nil {
		log.Fatalln("Cryto extenstion error: " + err.Error())
	}
	// install extension on server pipelines
	cr.Extend("server", svr)
	// run server
	log.Println("Listening on port ", addr.Port)
	svr.Serve()
}

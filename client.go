package ipcall

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/navaz-alani/concord/client"
	"github.com/navaz-alani/concord/core/crypto"
	"github.com/navaz-alani/concord/core/throttle"
	"github.com/navaz-alani/concord/packet"
	"github.com/navaz-alani/concord/server"
)

var (
	// maximum size of packet data (each sample is int32 (4 bytes) and they are
	// recorded in buffSize chunks, so the recorded chunk (packet data to be sent)
	// is max size 4*buffSize.
	pktMaxSize = 4 * buffSize
	// maximum size of packet metadata
	pktMetaMaxSize = 2048
	// maximum size of packet == read buff size
	pktTotalMaxSize = pktMaxSize + pktMetaMaxSize
)

type Client struct {
	callMu     sync.Mutex
	svrAddr    string
	client     client.Client
	pc         packet.PacketCreator
	sampleRate int
	statsMu    sync.RWMutex
	secure     bool
	cr         *crypto.Crypto
}

func NewClient(svrAddr, listenAddr *net.UDPAddr, sampleRate int, secure bool) (*Client, error) {
	pc := &packet.JSONPktCreator{}
	if concordClient, err := client.NewUDPClient(svrAddr, listenAddr, pktTotalMaxSize,
		&packet.JSONPktCreator{}, throttle.Rate100K); err != nil {
		return nil, fmt.Errorf("concord client init err: %s", err.Error())
	} else {
		var cr *crypto.Crypto
		if secure {
			cr, err = crypto.ConfigureClient(concordClient, svrAddr.String(), pc.NewPkt("", svrAddr.String()))
			if err != nil {
				return nil, fmt.Errorf("server kex err: %s", err.Error())
			}
		}
		pc := &packet.JSONPktCreator{}
		client := &Client{
			callMu:     sync.Mutex{},
			svrAddr:    svrAddr.String(),
			client:     concordClient,
			pc:         pc,
			sampleRate: sampleRate,
			statsMu:    sync.RWMutex{},
			secure:     secure,
			cr:         cr,
		}
		return client, nil
	}
}

// Call opens a virtual audio channel with the given address. It begins
// recording audio from the local machine and relaying it to the given `addr`.
// Audio recorded remotely and relayed is played on the audio channel of the
// local machine. The call ends when the done channel is signalled.
func (c *Client) OpenAudioChan(done <-chan struct{}, addr string) error {
	c.callMu.Lock() // obtain lock for placing a call
	defer c.callMu.Unlock()

	if c.secure {
		// perform key exchange with client (retry if not successful)
		fmt.Printf("Beginning handshake with %s\n", addr)
		var err error
		retries := 5
		for i := 0; i < retries; i++ {
			if err = c.cr.ClientKEx(c.client, addr, c.pc.NewPkt("", c.svrAddr)); err != nil {
				// sleep for 1/4 second and retry
				time.Sleep(250 * time.Millisecond)
				continue
			}
			break
		}
		if err != nil {
			return fmt.Errorf("%s kex fail: %s\n", addr, err.Error())
		} else {
			fmt.Printf("Success. Line with %s is now end-to-end encrypted.\n", addr)
		}
	}

	a := NewAudioIO(c.sampleRate)
	recordingDone := make(chan struct{})
	// begin recording and relaying chunks
	recordStream, err := a.Record(recordingDone)
	if err != nil {
		return fmt.Errorf("record err: %s", err.Error())
	}
	audioDataStream := make(chan []int32)
	go c.relayChunks(addr, audioDataStream, a)

	// play incoming
	playBuffStream := make(chan []int32)
	go a.Play(playBuffStream)
	var relayedFrom string
	relayedChunk := a.BuffPool.Get().([]int32)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() { // manages audioDataStream, ends when recordStream closes
		defer wg.Done()
		var chunk []int32
		for chunk = range recordStream {
			audioDataStream <- chunk
		}
		close(audioDataStream)
	}()
	// keep track of the number of bytes read
	var read uint64
	progressDone := make(chan struct{})
	wg.Add(1)
	go showProgress(&wg, "read", progressDone, &read)
	// process incoming packets
	deserializeBuff := new(bytes.Buffer)
	for {
		select {
		case <-done:
			goto EXIT
		case pkt := <-c.client.Misc():
			{
				// ignore if not a relayed message from `addr`
				relayedFrom = pkt.Meta().Get(server.KeyRelayFrom)
				if relayedFrom != addr {
					continue
				} else {
					c.cr.DecryptE2E(relayedFrom, pkt)
				}
				// otherwise, process data into audio chunk and send for playing
				read += uint64(len(pkt.Data()))
				deserializeBuff.Reset()
				deserializeBuff.ReadFrom(bytes.NewReader(pkt.Data()))
				deserializeBuff = decompress(deserializeBuff)
				binary.Read(deserializeBuff, binary.BigEndian, relayedChunk)
				playBuffStream <- relayedChunk
				relayedChunk = a.BuffPool.Get().([]int32)
			}
		}
	}
EXIT:
	close(playBuffStream)
	recordingDone <- struct{}{}
	progressDone <- struct{}{}
	wg.Wait()
	//c.client.Cleanup()
	return nil
}

func (c *Client) configureRelayPkt(addr string, data []byte) packet.Packet {
	pkt := c.pc.NewPkt("", c.svrAddr)
	writer := pkt.Writer()
	writer.Meta().Add(packet.KeyTarget, server.TargetRelay)
	writer.Meta().Add(server.KeyRelayTo, addr) // set server target to "relay"
	writer.Clear()
	writer.Write(data)
	writer.Close()
	return pkt
}

func showProgress(wg *sync.WaitGroup, verb string, done chan struct{}, n *uint64) {
	defer wg.Done()
	start := time.Now()
	for {
		select {
		case <-done:
			fmt.Printf("%s rate: %f bytes/second\n", verb, float64(*n)/time.Since(start).Seconds())
			return
		case <-time.After(5 * time.Second):
			log.Printf("(%v) %s: %d bytes\n", n, verb, *n)
		}
	}
}

// relayChunks comsumes chunks from the audioDataStream and relays them to the
// given address through the underlying UDP connection. It winds down work when
// the audioDataStream is closed.
func (c *Client) relayChunks(addr string, audioDataStream <-chan []int32, a *AudioIO) {
	// keep track of written bytes
	wg := sync.WaitGroup{}
	wg.Add(1)
	var written uint64
	progressDone := make(chan struct{})
	go showProgress(&wg, "written", progressDone, &written)
	// write recorded chunks as packets and relay
	var chunk []int32
	serializeBuff := new(bytes.Buffer)
	for chunk = range audioDataStream {
		// write the binary representation of the recorded chunk into serializeBuff
		binary.Write(serializeBuff, binary.BigEndian, chunk)
		a.BuffPool.Put(chunk) // return chunk
		serializeBuff = compress(serializeBuff)
		written += uint64(serializeBuff.Len())
		pkt := c.configureRelayPkt(addr, serializeBuff.Bytes())
		c.cr.EncryptE2E(addr, pkt) // end-to-end encrypt packet
		c.client.Send(pkt, nil)
		serializeBuff.Reset()
	}
	progressDone <- struct{}{}
	wg.Wait()
}

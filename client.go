package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/navaz-alani/concord/client"
	"github.com/navaz-alani/concord/core/throttle"
	"github.com/navaz-alani/concord/packet"
	"github.com/navaz-alani/concord/server"
)

type Client struct {
	callMu  sync.Mutex
	svrAddr string
	client  client.Client
	pc      packet.PacketCreator
}

func NewClient(svrAddr, listenAddr *net.UDPAddr) (*Client, error) {
	if concordClient, err := client.NewUDPClient(svrAddr, listenAddr, 10000,
		&packet.JSONPktCreator{}, throttle.Rate100K); err != nil {
		return nil, fmt.Errorf("concord client init err: %s", err.Error())
	} else {
		pc := &packet.JSONPktCreator{}
		return &Client{
			callMu:  sync.Mutex{},
			svrAddr: svrAddr.String(),
			client:  concordClient,
			pc:      pc,
		}, nil
	}
}

// Call opens a virtual audio channel with the given address. It begins
// recording audio from the local machine and relaying it to the given `addr`.
// Audio recorded remotely and relayed is played on the audio channel of the
// local machine. The call ends when the done channel is signalled.
func (c *Client) OpenAudioChan(done <-chan struct{}, addr string) error {
	c.callMu.Lock() // obtain lock for placing a call
	defer c.callMu.Unlock()
	a := NewAudioIO()
	killStream := make(chan struct{})
	wg := &sync.WaitGroup{}
	// begin recording and relaying chunks
	recordStream, err := a.Record(killStream)
	chkErr("record err: ", err)
	audioDataStream := make(chan []int32)
	go c.relayChunks(wg, addr, audioDataStream, a)

	// play incoming
	playBuffStream := make(chan []int32)
	go a.Play(playBuffStream)
	var relayedFrom string
	relayedChunk := a.BuffPool.Get().([]int32)
	go func() {
		var chunk []int32
		for chunk = range recordStream {
			audioDataStream <- chunk
		}
	}()
	for {
		// update the playBuff
		select {
		case <-done:
			goto EXIT
		case pkt := <-c.client.Misc():
			{
        fmt.Println("got chunk")
				// ignore if not a relayed message from `addr`
				relayedFrom = pkt.Meta().Get(server.KeyRelayFrom)
				if relayedFrom != addr {
					fmt.Printf("ignoring pkt from: %s\n", relayedFrom)
					continue
				}
				// otherwise, process data into audio chunk and send for playing
				chunk, err := base64.StdEncoding.DecodeString(string(pkt.Data()))
				if err != nil {
					fmt.Printf("decode err: %s\n", err.Error())
					continue
				}
				if err := binary.Read(bytes.NewReader(chunk), binary.BigEndian, relayedChunk); err != nil {
					fmt.Printf("binary read err: %s\n", err.Error())
					continue
				}
				fmt.Printf("got chunk\n")
				playBuffStream <- relayedChunk
				relayedChunk = a.BuffPool.Get().([]int32)
			}
		}
	}
EXIT:
	close(audioDataStream)
  killStream <- struct{}{}
	wg.Wait()
	return nil
}

func (c *Client) configureRelayPkt(addr string, data string) packet.Packet {
	pkt := c.pc.NewPkt("", c.svrAddr)
	writer := pkt.Writer()
	writer.Meta().Add(packet.KeyTarget, server.TargetRelay)
	writer.Meta().Add(server.KeyRelayTo, addr) // set server target to "relay"
	writer.Clear()
	writer.Write([]byte(data))
	writer.Close()
	return pkt
}

func (c *Client) relayChunks(wg *sync.WaitGroup, addr string, audioDataStream <-chan []int32, a *AudioIO) {
	defer wg.Done()

	var chunk []int32
	var encoded string
	encodeBuff := new(bytes.Buffer)

	for chunk = range audioDataStream {
		// send chunk as message
		binary.Write(encodeBuff, binary.BigEndian, chunk)
		a.BuffPool.Put(chunk) // return chunk
		encoded = base64.StdEncoding.EncodeToString(encodeBuff.Bytes())
		if err := c.client.Send(c.configureRelayPkt(addr, encoded), nil); err != nil {
			fmt.Printf("send err: %s\n", err.Error())
		}
		fmt.Printf("%v relayed chunk\n", a)
		encodeBuff.Reset()
	}
}

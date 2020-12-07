package ipcall

import (
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
)

const (
	buffSize     = 480 // to get frame size of 60ms (at 8kHz sampling rate)
	buffPoolSize = 10_000
)

type AudioIO struct {
	mu         sync.RWMutex
	BuffPool   *sync.Pool
	sampleRate int
}

func NewAudioIO(sampleRate int) *AudioIO {
	buffPool := &sync.Pool{
		New: func() interface{} {
			return make([]int32, buffSize)
		},
	}
	// warn up buffer pool
	for i := 0; i < buffPoolSize; i++ {
		buffPool.Put(buffPool.New())
	}
	a := &AudioIO{
		mu:         sync.RWMutex{},
		BuffPool:   buffPool,
		sampleRate: sampleRate,
	}
	return a
}

// Play streams over the given `buffStream` and plays the audio chunks. It exits
// when `buffStream` is closed.
func (a *AudioIO) Play(buffStream <-chan []int32) error {
	if err := portaudio.Initialize(); err != nil {
		return fmt.Errorf("portaudio init err: %s", err.Error())
	}
	defer portaudio.Terminate()
	// instantiate audio stream
	paStreamBuff := a.BuffPool.Get().([]int32)
	stream, err := portaudio.OpenDefaultStream(0, 1, float64(a.sampleRate), buffSize, &paStreamBuff)
	if err != nil {
		return fmt.Errorf("stream init fail: %s", err.Error())
	}
	defer stream.Close()
	// start audio stream (commence audio processing)
	if err := stream.Start(); err != nil {
		return fmt.Errorf("stream start err: %s", err.Error())
	}
	defer stream.Stop()
	// stream over the buffStream and writes to the audio stream
	for next := range buffStream {
		copy(paStreamBuff, next)
		a.BuffPool.Put(next)
		stream.Write()
	}
	return nil
}

// Record records audio data in buffers which are sent on the returned channel.
// The `done` chan will complete the recording operation and close the returned
// record stream channel.
func (a *AudioIO) Record(done chan struct{}) (<-chan []int32, error) {
	recBuff := a.BuffPool.Get().([]int32)
	// initialize portaudio and create audio stream
	portaudio.Initialize()
	stream, err := portaudio.OpenDefaultStream(1, 0, float64(a.sampleRate), buffSize, &recBuff)
	if err != nil {
		return nil, fmt.Errorf("stream init fail: %s", err.Error())
	}
	if err := stream.Start(); err != nil {
		return nil, fmt.Errorf("stream start err: %s", err.Error())
	}
	// start routine to read audio data in buffers from the audio stream and serve
	// them over `recordStream`
	recordStream := make(chan []int32)
	go func() {
		defer portaudio.Terminate()
		defer stream.Close()
		defer stream.Stop()

		var next []int32 // the next recorded chunks
		for {
			select {
			case <-done:
				stream.Stop()
				close(recordStream)
				return
			default:
				stream.Read()
				next = a.BuffPool.Get().([]int32)
				copy(next, recBuff)
				recordStream <- next
			}
		}
	}()
	return recordStream, nil
}

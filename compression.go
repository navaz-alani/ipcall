package ipcall

import (
	"bytes"
	"compress/gzip"
	"fmt"

	"gopkg.in/hraban/opus.v2"
)

type OpusCompressor struct {
	sampleRate, channels int
	enc                  *opus.Encoder
	dec                  *opus.Decoder
}

func NewOpusCompressor(sampleRate, channels int) (*OpusCompressor, error) {
	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		return nil, err
	}
	dec, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		return nil, err
	}
	return &OpusCompressor{
		sampleRate: sampleRate,
		enc:        enc,
		dec:        dec,
	}, nil
}

func (oc *OpusCompressor) compress(audio []int32, dst []byte) {
	pcm := make([]float32, len(audio))
	for _, sample := range audio {
		pcm = append(pcm, float32(sample))
	}
	if _, err := oc.enc.EncodeFloat32(pcm, dst); err != nil {
		fmt.Printf("encode err: %s\n", err.Error())
	}
}

func (oc *OpusCompressor) decompress(data []byte, dst *[]int32) {
	audio := make([]float32, buffSize)
	if _, err := oc.dec.DecodeFloat32(data, audio); err != nil {
		fmt.Printf("decode err: %s\n", err.Error())
	}
	for _, sample := range audio {
		*dst = append(*dst, int32(sample))
	}
}

// gzip compression

func compress(buff *bytes.Buffer) *bytes.Buffer {
	compressed := new(bytes.Buffer)
	w := gzip.NewWriter(compressed)
	w.Write(buff.Bytes())
	w.Close()
	return compressed
}

func decompress(buff *bytes.Buffer) *bytes.Buffer {
	decompressed := make([]byte, pktMaxSize)
	r, _ := gzip.NewReader(buff)
	r.Read(decompressed)
	r.Close()
	return bytes.NewBuffer(decompressed)
}

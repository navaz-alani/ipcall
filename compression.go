package ipcall

import (
	"bytes"
	"compress/gzip"
)

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

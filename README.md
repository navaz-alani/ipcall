# `ipcall`

`ipcall` is a command-line application to perform voice calls between two IP
addresses. It is implemented in Golang and builds on a Go wrapper for
`portaudio`.

To use `ipcall`, install portaudio on your machine and grant the terminal
emulator access to the mircrophone (if/when prompted when running `ipcall`). To
build the application, simply run `go build` and the `ipcall` binary will be
produced. The following shows an invocation of the `ipcall` program.

```
./ipcall -s-addr=<server-addr> -l-addr=<listen-addr> -c-addr=<client-addr>
```

Here are the flags:

* `-s-addr` is the flag which specifies the address of the server used to relay
  packets to the client (`<client-addr>`).
* `-c-addr` is the flag which specifies the address of the client with which the
  call is to be performed.
* `-l-addr` is the flag which specifies the address on which the client will
  listen for packets relayed from `<client-addr>`.

## Network Stats

Currently, the throughput of this server is about 16.5 kbps, with a 5000Hz
sample rate (so about 17.5% compression on 20kbps).

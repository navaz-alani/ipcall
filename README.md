# `ipcall`

`ipcall` is a command-line application to perform voice calls between two IP
addresses. It is implemented in Golang and builds on a Go wrapper for
`portaudio`.

To use `ipcall`, install portaudio on your machine and grant the terminal
emulator access to the mircrophone (if/when prompted when running `ipcall`). To
build the application, simply run `go build` and the `ipcall` binary will be
produced. The following shows an invocation of the `ipcall` program.

```
./ipcall -svr-addr=<svrAddr> -listen-addr=<lAddr> -client-addr=<clientAddr>
```

Here are the flags:

* `-svr-addr` is the flag which specifies the address of the server used to
  relay packets to the client (`clientAddr`).
* `-client-addr` is the flag which specifies the address of the client with
  which the call is to be performed.
* `-listen-addr` is the flag which specifies the address on which the client
  will listen for packets relayed from `clientAddr`.

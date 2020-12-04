CLIENT_SRCS=$(wildcard *.go cmd/client/*.go)
SERVER_SRCS=$(wildcard *.go cmd/server/*.go)

.PHONY: all clean

ipcall: $(CLIENT_SRCS)
	go build -o $@ ./cmd/client
ipcall-svr: $(SERVER_SRCS)
	go build -o $@ ./cmd/server

all: ipcall ipcall-svr
clean:
	rm -rf ipcall{,-svr}

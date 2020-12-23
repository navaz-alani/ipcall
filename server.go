package ipcall

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"net"
	"sync"

	"github.com/navaz-alani/concord/core"
	"github.com/navaz-alani/concord/core/crypto"
	"github.com/navaz-alani/concord/core/throttle"
	"github.com/navaz-alani/concord/packet"
	"github.com/navaz-alani/concord/server"
)

const (
	KeyIPCallAlias          string = "ipcall_alias"
	KeyIPCallRelayToAlias          = "ipcall_relay_to"
	KeyIPCallRelayFromAlias        = "ipcall_relay_from"
)

type IPCall struct {
	svr   server.Server
	addr  *net.UDPAddr
	mu    *sync.RWMutex
	users *BidiMap /* key = alias, val = addr */
}

func NewIPCallServer(addr *net.UDPAddr, secure bool) (*IPCall, error) {
	svr, err := server.NewUDPServer(addr, 10000, &packet.JSONPktCreator{}, throttle.Rate10k)
	if err != nil {
		return nil, fmt.Errorf("server create fail: " + err.Error())
	}
	if secure {
		// generate private key
		privKey, err := ecdsa.GenerateKey(crypto.Curve, rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("public key gen fail: " + err.Error())
		}
		// initialize Crypto extension
		cr, err := crypto.NewCrypto(privKey)
		if err != nil {
			return nil, fmt.Errorf("cryto extenstion error: " + err.Error())
		}
		// install extension on server pipelines
		cr.Extend("server", svr)
	}
	return &IPCall{
		svr:   svr,
		addr:  addr,
		users: NewBidiMap(1000),
	}, nil
}

func (ip *IPCall) Serve() {
	// install the targets onto the server
	ip.extend()
	// begin the server RW loop
	ip.svr.Serve()
}

// `extend` installs `IPCall`'s targets on the underlying concord server.
func (ip *IPCall) extend() {
	ip.svr.PacketProcessor().AddCallback("register", ip.register)
	ip.svr.PacketProcessor().AddCallback("unregister", ip.unregister)
	ip.svr.PacketProcessor().AddCallback("proxy", ip.proxy)
}

// `__register__` registers the given `alias` under the given `addr` to the
// `IPCall` server.
func (ip *IPCall) __register__(alias, addr string) error {
	if _, ok := ip.users.GetByKey(alias); ok {
		return fmt.Errorf("alias registered, unregister and try again")
	}
	ip.users.Add(alias, addr)
	return nil
}

// `__unregister__` unregisters the given `alias` and `addr` pair from the
// `IPCall` server.
func (ip *IPCall) __unregister__(alias, addr string) error {
	if aliasAddr, ok := ip.users.GetByKey(alias); !ok {
		return fmt.Errorf("alias unregistered")
	} else if aliasAddr == addr {
		return fmt.Errorf("unauthorized unregistration")
	}
	ip.users.Delete(alias)
	return nil
}

// `register` registers a user onto the `IPCall` server.
func (ip *IPCall) register(ctx *core.TargetCtx, pw packet.Writer) {
	// ensure the user has provided an `alias` in the registration packet
	var alias string
	if alias = ctx.Pkt.Meta().Get(KeyIPCallAlias); alias == "" {
		ctx.Stat = core.CodeStopError
		ctx.Msg = "registration alias (username) not provided"
		return
	}
	// attempt to register the user's address under the `alias` provided
	if err := ip.__register__(alias, ctx.From); err != nil {
		ctx.Stat = core.CodeStopError
		ctx.Msg = err.Error()
		return
	}
}

func (ip *IPCall) unregister(ctx *core.TargetCtx, pw packet.Writer) {
	// ensure the user has provided an `alias` in the registration packet
	var alias string
	if alias = ctx.Pkt.Meta().Get(KeyIPCallAlias); alias == "" {
		ctx.Stat = core.CodeStopError
		ctx.Msg = "unregistration alias (username) not provided"
		return
	}
	// attempt to unregister the user's address under the `alias` provided
	if err := ip.__unregister__(alias, ctx.From); err != nil {
		ctx.Stat = core.CodeStopError
		ctx.Msg = err.Error()
	}
}

// `proxy` proxies a packet between users registered on the server.
func (ip *IPCall) proxy(ctx *core.TargetCtx, pw packet.Writer) {
	// ensure packet sender `ctx.From` is registered with the `IPCall` server
	var fromAlias string
	var ok bool
	if fromAlias, ok = ip.users.GetByVal(ctx.From); !ok {
		ctx.Stat = core.CodeStopError
		ctx.Msg = "proxy error: unauthorized request (sender not registered)"
		return
	}
	// ensure that the alias being relayed to is specified and registered with the
	// `IPCall` server
	var relayAlias, relayAddr string
	if relayAlias = ctx.Pkt.Meta().Get(KeyIPCallRelayToAlias); relayAlias == "" {
		ctx.Stat = core.CodeStopError
		ctx.Msg = "proxy error: relay alias not specified"
		return
	} else if relayAddr, ok = ip.users.GetByKey(relayAlias); !ok {
		ctx.Stat = core.CodeStopError
		ctx.Msg = "proxy error: relay alias not registered"
		return
	}
	// set context status so that server relays the response packet underlying
	// `pw` to `relayAddr`
	ctx.Stat = core.CodeRelay
	pw.Meta().Add(server.KeyRelayTo, relayAddr)
	pw.Meta().Add(KeyIPCallRelayToAlias, relayAlias)
	pw.Meta().Add(KeyIPCallRelayFromAlias, fromAlias)
	pw.Write(ctx.Pkt.Data())
	pw.Close()
}

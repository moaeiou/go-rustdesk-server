package transport

import (
	"context"
	"net"
	"sync"
)

const (
	tcp = "tcp"
	udp = "udp"
	TCP = tcp
	UDP = udp
)

var (
	PortHTTPAPI = Config.HTTPPort
	PortNAT     = Config.NATPort
	PortSignal  = Config.IDPort
	PortRelay   = Config.RelayPort
	PortWS      = Config.WebSocketPort
	PortWebAPI  = Config.WebAPIPort
)

type monitor struct {
	network     string
	addr        string
	listen      net.Listener
	conn        *net.UDPConn
	call        func([]byte, *Writer)
	relay       bool
	connections sync.Map
}

func NewMonitor(relay bool, network, addr string, call func([]byte, *Writer)) *monitor {
	return &monitor{network: network, addr: addr, call: call, relay: relay}
}

func (m *monitor) Run(ctx context.Context) error {
	if m.network == udp {
		return m.startUDP(ctx)
	}
	return m.startTCP(ctx)
}

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
	PortHTTPAPI = Config.BasePort - 2
	PortNAT     = Config.BasePort - 1
	PortSignal  = Config.BasePort
	PortRelay   = Config.BasePort + 1
	PortWS      = Config.BasePort + 2
	PortWebAPI  = Config.BasePort + 3
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

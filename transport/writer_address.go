package transport

import (
	"errors"
	"net"
)

type Addr struct {
	ip   string
	port uint32
}

func (a *Addr) GetIP() string        { return a.ip }
func (a *Addr) GetPort() uint32      { return a.port }
func (a *Addr) Parsing(value string) { ip, port := GetHostPort(value); a.ip, a.port = ip, uint32(port) }

func (w *Writer) WriteToAddr(payload []byte, address string) (int, error) {
	if w._type != udp || w.uConn == nil {
		return 0, errors.New("UDP connection is unavailable")
	}
	remote, err := net.ResolveUDPAddr(udp, address)
	if err != nil {
		return 0, err
	}
	return w.uConn.WriteToUDP(payload, remote)
}

func (w *Writer) GetAddrStr() string {
	if w._type == udp {
		return w.addr.String()
	}
	if w._type == tcp {
		return w.tConn.RemoteAddr().String()
	}
	return ""
}

func (w *Writer) GetAddr() *Addr { address := &Addr{}; address.Parsing(w.GetAddrStr()); return address }

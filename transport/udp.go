package transport

import (
	"context"
	"errors"
	"net"
)

func (m *monitor) startUDP(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr(udp, m.addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP(udp, addr)
	if err != nil {
		return err
	}
	if Config.UDPReceiveBuffer > 0 {
		if err := conn.SetReadBuffer(Config.UDPReceiveBuffer); err != nil {
			conn.Close()
			return err
		}
	}
	m.conn = conn
	defer conn.Close()
	go func() { <-ctx.Done(); _ = conn.Close() }()
	Info("UDP listening %s", m.addr)
	buffer := make([]byte, 65535)
	for {
		n, remote, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				Warn("UDP read: %v", err)
			}
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if n == 0 {
			continue
		}
		payload := append([]byte(nil), buffer[:n]...)
		writer, err := GetWriter(remote.String(), udp)
		if err != nil {
			writer = &Writer{_type: udp, uConn: conn, addr: remote}
			addWriter(remote.String(), udp, writer)
		}
		m.call(payload, writer)
	}
}

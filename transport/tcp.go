package transport

import (
	"context"
	"errors"
	"go-rustdesk-server/protocol"
	"go.uber.org/zap/buffer"
	"io"
	"net"
	"sync/atomic"
	"time"
)

const (
	frameReadTimeout = 15 * time.Second
)

var activeTCPConnections atomic.Int64

func (m *monitor) startTCP(ctx context.Context) error {
	listener, err := net.Listen(tcp, m.addr)
	if err != nil {
		return err
	}
	m.listen = listener
	defer listener.Close()
	go func() {
		<-ctx.Done()
		_ = listener.Close()
		m.connections.Range(func(key, _ any) bool { _ = key.(net.Conn).Close(); return true })
	}()
	Info("TCP listening %s", m.addr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				Warn("TCP accept %s: %v", m.addr, err)
			}
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if activeTCPConnections.Add(1) > Config.MaxConnections {
			activeTCPConnections.Add(-1)
			_ = conn.Close()
			continue
		}
		m.connections.Store(conn, struct{}{})
		go m.acceptTCP(conn)
	}
}

func (m *monitor) acceptTCP(conn net.Conn) {
	writer := &Writer{_type: tcp, tConn: conn, loop: true, onClose: func() {
		activeTCPConnections.Add(-1)
		m.connections.Delete(conn)
	}}
	addWriter(conn.RemoteAddr().String(), tcp, writer)
	defer func() {
		if !writer.Detached() {
			writer.Close()
		}
	}()
	buf := buffer.NewPool().Get()
	defer buf.Free()
	var frameLength uint
	temp := make([]byte, 4096)
	_ = conn.SetReadDeadline(time.Now().Add(frameReadTimeout))
	for writer.loop {
		n, err := conn.Read(temp)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				Info("TCP read %s: %v", conn.RemoteAddr(), err)
			}
			return
		}
		if n == 0 {
			continue
		}
		_, _ = buf.Write(temp[:n])
		_ = conn.SetReadDeadline(time.Now().Add(frameReadTimeout))
		for buf.Len() > 0 {
			if frameLength == 0 {
				headerLength := int((buf.Bytes()[0] & 3) + 1)
				if buf.Len() < headerLength {
					break
				}
				_, frameLength, err = protocol.DecodeHead(buf.Bytes())
				if err != nil || frameLength < uint(headerLength) || frameLength > Config.MaxFrameSize {
					return
				}
			}
			if buf.Len() < int(frameLength) {
				break
			}
			frame := append([]byte(nil), buf.Bytes()[:frameLength]...)
			remaining := append([]byte(nil), buf.Bytes()[frameLength:]...)
			buf.Reset()
			_, _ = buf.Write(remaining)
			frameLength = 0
			if m.relay {
				prepareRelayStream(conn, writer, remaining)
				m.processTCPFrame(frame, conn)
				return
			}
			go m.processTCPFrame(frame, conn)
		}
		if frameLength == 0 && buf.Len() == 0 {
			_ = conn.SetReadDeadline(time.Time{})
		}
	}
}

func prepareRelayStream(conn net.Conn, writer *Writer, pending []byte) {
	// The deadline in acceptTCP only protects the framed relay handshake. Once
	// handed to the raw relay copier, an idle session must not inherit it.
	_ = conn.SetReadDeadline(time.Time{})
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		// Keep small control/video packets moving promptly and give short bursts
		// room without imposing an application-level queue on the relay path.
		_ = tcpConn.SetNoDelay(true)
		_ = tcpConn.SetKeepAliveConfig(net.KeepAliveConfig{
			Enable: true, Idle: 30 * time.Second, Interval: 10 * time.Second, Count: 3,
		})
		if size := Config.RelayBufferSize; size > 0 {
			_ = tcpConn.SetReadBuffer(size)
			_ = tcpConn.SetWriteBuffer(size)
		}
	}
	// A TCP read can contain both the framed RequestRelay and the beginning of
	// the raw encrypted session. Preserve those coalesced bytes for Copy;
	// dropping them makes both peers pair successfully but their handshake time
	// out because its first packet is missing.
	writer.setPending(pending)
	writer.loop = false
}

func (m *monitor) processTCPFrame(frame []byte, conn net.Conn) {
	defer func() {
		if value := recover(); value != nil {
			Warn("message from %s panicked: %v", conn.RemoteAddr(), value)
		}
	}()
	payload, err := protocol.Decode(frame)
	if err != nil {
		Warn("Decode %s: %v", conn.RemoteAddr(), err)
		return
	}
	writer, err := GetWriter(conn.RemoteAddr().String(), tcp)
	if err != nil {
		writer = &Writer{_type: tcp, tConn: conn}
		addWriter(conn.RemoteAddr().String(), tcp, writer)
	}
	m.call(payload, writer)
}

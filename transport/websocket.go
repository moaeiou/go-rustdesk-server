package transport

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go-rustdesk-server/protocol"
)

// WebSocket messages contain protobuf/session payloads without the TCP length
// header. Never stream arbitrary TCP chunks into WebSocket messages.
func (m *monitor) startWebSocket(ctx context.Context) error {
	upgrader := websocket.Upgrader{
		HandshakeTimeout: frameReadTimeout,
		// Browser clients may be hosted on a different origin. Authorization is
		// performed by the same licence-key checks as the TCP protocol.
		CheckOrigin: func(*http.Request) bool { return true },
	}
	server := &http.Server{Addr: m.addr, ReadHeaderTimeout: frameReadTimeout}
	server.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if !websocket.IsWebSocketUpgrade(req) {
			if m.relay {
				rw.WriteHeader(http.StatusOK)
			} else {
				http.Error(rw, "WebSocket upgrade required", http.StatusUpgradeRequired)
			}
			return
		}
		if activeTCPConnections.Add(1) > Config.MaxConnections {
			activeTCPConnections.Add(-1)
			http.Error(rw, "connection limit", http.StatusServiceUnavailable)
			return
		}
		ws, err := upgrader.Upgrade(rw, req, nil)
		if err != nil {
			activeTCPConnections.Add(-1)
			return
		}
		conn := ws.UnderlyingConn()
		m.connections.Store(conn, struct{}{})
		writer := &Writer{_type: tcp, tConn: conn, wsConn: ws, onClose: func() {
			activeTCPConnections.Add(-1)
			m.connections.Delete(conn)
		}}
		addWriter(conn.RemoteAddr().String(), tcp, writer)
		defer func() {
			if !writer.Detached() {
				writer.Close()
			}
		}()
		if ctx.Err() != nil {
			writer.Close()
			return
		}
		ws.SetReadLimit(int64(Config.MaxFrameSize))
		if m.relay {
			_ = ws.SetReadDeadline(time.Now().Add(frameReadTimeout))
		}
		for {
			kind, payload, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if kind != websocket.BinaryMessage {
				continue
			}
			if m.relay {
				_ = ws.SetReadDeadline(time.Time{})
			}
			m.call(payload, writer)
			if m.relay {
				return
			}
		}
	})
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
		case <-done:
		}
		_ = server.Close()
		// net/http does not close hijacked WebSocket connections.
		m.connections.Range(func(key, _ any) bool { _ = key.(net.Conn).Close(); return true })
	}()
	Info("WebSocket listening %s", m.addr)
	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func readSessionFrame(reader *bufio.Reader) ([]byte, error) {
	first, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	header := make([]byte, int(first&3)+1)
	header[0] = first
	if _, err = io.ReadFull(reader, header[1:]); err != nil {
		return nil, err
	}
	size, _, err := protocol.DecodeHead(header)
	if err != nil {
		return nil, err
	}
	if size > Config.MaxFrameSize {
		return nil, fmt.Errorf("relay frame exceeds max_frame_size")
	}
	payload := make([]byte, size)
	_, err = io.ReadFull(reader, payload)
	return payload, err
}

func (w *Writer) copyMessages(peer *Writer) {
	started := time.Now()
	budget := newRelayBudget(w, peer)
	done := make(chan copyResult, 2)
	copyDirection := func(dst, src *Writer) {
		reader := bufio.NewReader(io.MultiReader(bytes.NewReader(src.takePending()), src.tConn))
		var total int64
		for {
			var data []byte
			var err error
			if src.IsWebSocket() {
				var kind int
				kind, data, err = src.wsConn.ReadMessage()
				if err == nil && kind != websocket.BinaryMessage {
					continue
				}
			} else {
				data, err = readSessionFrame(reader)
			}
			if err == nil {
				err = budget.wait(len(data))
				if err == nil {
					_, err = dst.Write(data)
				}
				total += int64(len(data))
			}
			if err != nil {
				done <- copyResult{bytes: total, err: err}
				return
			}
		}
	}
	go copyDirection(peer, w)
	go copyDirection(w, peer)
	first := <-done
	close(budget.stop)
	w.Close()
	peer.Close()
	second := <-done
	Info("RELAY WebSocket stream closed duration=%s bytes=%d/%d first_error=%v", time.Since(started).Round(time.Millisecond), first.bytes, second.bytes, first.err)
}

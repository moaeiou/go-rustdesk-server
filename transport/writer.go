package transport

import (
	"errors"
	"github.com/gorilla/websocket"
	"go-rustdesk-server/protocol"
	"net"
	"sync"
)

type Writer struct {
	mu             sync.Mutex
	closeOnce      sync.Once
	key, _type     string
	tConn          net.Conn
	wsConn         *websocket.Conn
	uConn          *net.UDPConn
	addr           *net.UDPAddr
	pending        []byte
	loop, detached bool
	onClose        func()
}

func (w *Writer) Type() string { return w._type }

func (w *Writer) Write(payload []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w._type == udp {
		if w.uConn == nil {
			return 0, errors.New("UDP connection is nil")
		}
		return w.uConn.WriteToUDP(payload, w.addr)
	}
	if w.wsConn != nil {
		err := w.wsConn.WriteMessage(websocket.BinaryMessage, payload)
		if err != nil {
			return 0, err
		}
		return len(payload), nil
	}
	if w._type == tcp {
		if w.tConn == nil {
			return 0, errors.New("TCP connection is nil")
		}
		frame, err := protocol.Encoder(payload)
		if err != nil {
			return 0, err
		}
		n, err := writeAll(w.tConn, frame)
		if err != nil {
			return n, err
		}
		return len(payload), nil
	}
	return 0, errors.New("unknown writer type")
}

func (w *Writer) SetKey(key string) {
	_ = cache.Set(ctx, w._type+key, w, cacheTimeOut)
	w.mu.Lock()
	w.key = key
	w.mu.Unlock()
}

func (w *Writer) Detach()        { w.mu.Lock(); w.detached = true; w.mu.Unlock() }
func (w *Writer) Detached() bool { w.mu.Lock(); defer w.mu.Unlock(); return w.detached }

func (w *Writer) setPending(data []byte) {
	w.mu.Lock()
	w.pending = data
	w.mu.Unlock()
}

func (w *Writer) takePending() []byte {
	w.mu.Lock()
	data := w.pending
	w.pending = nil
	w.mu.Unlock()
	return data
}

func (w *Writer) Close() {
	w.closeOnce.Do(func() {
		if w._type == tcp && w.tConn != nil {
			_ = w.tConn.Close()
		}
		w.remove()
		if w.onClose != nil {
			w.onClose()
		}
	})
}

func (w *Writer) IsWebSocket() bool { return w.wsConn != nil }

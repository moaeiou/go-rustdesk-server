package relay

import (
	"context"
	"go-rustdesk-server/transport"
	"sync"
	"time"
)

const relayWaitTimeout = time.Minute

type waitingPeer struct {
	writer  *transport.Writer
	expires time.Time
}

type waitingPool struct {
	mu      sync.Mutex
	entries map[string]waitingPeer
}

func newWaitingPool() *waitingPool {
	return &waitingPool{entries: make(map[string]waitingPeer)}
}

func (p *waitingPool) match(uuid string, writer *transport.Writer) *transport.Writer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer, found := p.entries[uuid]; found {
		delete(p.entries, uuid)
		return peer.writer
	}
	p.entries[uuid] = waitingPeer{writer: writer, expires: time.Now().Add(relayWaitTimeout)}
	return nil
}

func (p *waitingPool) run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			p.expire(now)
		case <-ctx.Done():
			return
		}
	}
}

func (p *waitingPool) expire(now time.Time) {
	var expired []waitingPeer
	p.mu.Lock()
	for uuid, peer := range p.entries {
		if !now.Before(peer.expires) {
			delete(p.entries, uuid)
			expired = append(expired, peer)
			transport.Info("RELAY wait timeout for %s", uuid)
		}
	}
	p.mu.Unlock()
	for _, peer := range expired {
		peer.writer.Close()
	}
}

func (p *waitingPool) close() {
	p.mu.Lock()
	writers := make([]*transport.Writer, 0, len(p.entries))
	for uuid, peer := range p.entries {
		delete(p.entries, uuid)
		writers = append(writers, peer.writer)
	}
	p.mu.Unlock()
	for _, writer := range writers {
		writer.Close()
	}
}

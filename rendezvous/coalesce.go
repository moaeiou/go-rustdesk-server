package rendezvous

import (
	"go-rustdesk-server/transport"
	"google.golang.org/protobuf/reflect/protoreflect"
	"sync"
)

// pendingCall represents one in-flight call shared by every caller that
// asked for the same key while it was running.
type pendingCall struct {
	wg          sync.WaitGroup
	writer      *transport.Writer
	message     protoreflect.ProtoMessage
	relayServer string
}

// callGroup deduplicates concurrent calls for the same key: only the
// first caller actually runs fn; everyone else who calls Do with the same
// key while it's still running just waits and gets the exact same result.
//
// This is what stops a client's retried/duplicate request from
// triggering a brand new notify-the-peer-and-wait round trip every single
// time — they all share the one already in flight instead of racing each
// other (and the original RustDesk client retries fairly aggressively
// when it doesn't get a fast answer, so this matters in practice).
type callGroup struct {
	mu sync.Mutex
	m  map[string]*pendingCall
}

func newCallGroup() *callGroup {
	return &callGroup{m: make(map[string]*pendingCall)}
}

func (g *callGroup) Do(key string, fn func() (*transport.Writer, protoreflect.ProtoMessage, string)) (*transport.Writer, protoreflect.ProtoMessage, string) {
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.writer, c.message, c.relayServer
	}
	c := &pendingCall{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.writer, c.message, c.relayServer = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.writer, c.message, c.relayServer
}

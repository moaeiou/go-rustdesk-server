package rendezvous

import (
	"sync"
	"time"
)

const requestsPerMinute = 600

type rateEntry struct {
	window time.Time
	count  int
}

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]rateEntry
}

var requestLimiter = rateLimiter{clients: make(map[string]rateEntry)}

func (l *rateLimiter) Allow(client string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	entry := l.clients[client]
	if entry.window.IsZero() || now.Sub(entry.window) >= time.Minute {
		entry = rateEntry{window: now}
	}
	entry.count++
	l.clients[client] = entry
	if len(l.clients) > 4096 {
		for key, value := range l.clients {
			if now.Sub(value.window) >= time.Minute {
				delete(l.clients, key)
			}
		}
	}
	return entry.count <= requestsPerMinute
}

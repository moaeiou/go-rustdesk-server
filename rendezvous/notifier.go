package rendezvous

import (
	"go-rustdesk-server/transport"
	"google.golang.org/protobuf/reflect/protoreflect"
	"sync"
	"time"
)

type notifyResult struct {
	Writer  *transport.Writer
	Message protoreflect.ProtoMessage
}

type notifier struct {
	mu      sync.Mutex
	waiters map[string][]chan notifyResult
}

var msgNotifier = &notifier{waiters: make(map[string][]chan notifyResult)}

func (n *notifier) subscribe(keys []string) (chan notifyResult, func()) {
	ch := make(chan notifyResult, 1)
	n.mu.Lock()
	for _, key := range keys {
		n.waiters[key] = append(n.waiters[key], ch)
	}
	n.mu.Unlock()

	cancel := func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		for _, key := range keys {
			waiters := n.waiters[key]
			for i, waiter := range waiters {
				if waiter == ch {
					n.waiters[key] = append(waiters[:i], waiters[i+1:]...)
					break
				}
			}
			if len(n.waiters[key]) == 0 {
				delete(n.waiters, key)
			}
		}
	}
	return ch, cancel
}

func (n *notifier) publish(key string, writer *transport.Writer, message protoreflect.ProtoMessage) {
	n.mu.Lock()
	waiters := n.waiters[key]
	delete(n.waiters, key)
	n.mu.Unlock()

	for _, waiter := range waiters {
		select {
		case waiter <- notifyResult{Writer: writer, Message: message}:
		default:
		}
	}
}

type messageWaiter func(timeoutSeconds uint) (*transport.Writer, protoreflect.ProtoMessage)

// subscribeForMsg installs the subscription immediately and returns a waiter.
// Callers must create the waiter before sending the request, otherwise a fast
// peer can reply between send and subscribe and force an unnecessary timeout.
func subscribeForMsg(id string, types []string) messageWaiter {
	keys := make([]string, len(types))
	for i, messageType := range types {
		keys[i] = id + messageType
	}
	return subscribeForKeys(keys)
}

func subscribeForKeys(keys []string) messageWaiter {
	results, cancel := msgNotifier.subscribe(keys)
	return func(timeoutSeconds uint) (*transport.Writer, protoreflect.ProtoMessage) {
		defer cancel()
		if timeoutSeconds == 0 {
			timeoutSeconds = 3
		}
		timer := time.NewTimer(time.Duration(timeoutSeconds) * time.Second)
		defer timer.Stop()
		select {
		case result := <-results:
			return result.Writer, result.Message
		case <-timer.C:
			return nil, nil
		}
	}
}

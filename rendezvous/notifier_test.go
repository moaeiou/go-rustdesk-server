package rendezvous

import (
	"go-rustdesk-server/protocol"
	"testing"
)

func TestSubscribeBeforeImmediateReply(t *testing.T) {
	previous := msgNotifier
	msgNotifier = &notifier{waiters: make(map[string][]chan notifyResult)}
	t.Cleanup(func() { msgNotifier = previous })

	id := "fast-public-peer"
	wait := subscribeForMsg(id, []string{protocol.TypeRendezvousMessageRelayResponse})
	reply := &protocol.RelayResponse{}
	msgNotifier.publish(id+protocol.TypeRendezvousMessageRelayResponse, nil, reply)

	_, received := wait(1)
	if received != reply {
		t.Fatal("immediate relay response was lost")
	}
}

package relay

import (
	"context"
	"go-rustdesk-server/protocol"
	"go-rustdesk-server/transport"
	"google.golang.org/protobuf/proto"
	"reflect"
)

func Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	transport.Info("Relay TCP listening on port %d", transport.PortRelay)
	waiting := newWaitingPool()
	go waiting.run(ctx)
	defer waiting.close()
	handler := func(message []byte, writer *transport.Writer) { handlerMsg(message, writer, waiting) }
	results := make(chan error, 2)
	go func() {
		results <- transport.NewMonitor(true, "tcp", transport.ListenAddress(transport.PortRelay), handler).Run(ctx)
	}()
	go func() {
		results <- transport.NewMonitor(true, "ws", transport.ListenAddress(transport.PortWebAPI), handler).Run(ctx)
	}()
	select {
	case <-ctx.Done():
		<-results
		<-results
		return nil
	case err := <-results:
		cancel()
		<-results
		return err
	}
}

func handlerMsg(msg []byte, writer *transport.Writer, waiting *waitingPool) {
	if transport.BlockedRelayAddress(writer.GetAddrStr()) {
		writer.Close()
		return
	}
	transport.Info("RELAY-RX %s len %d", writer.GetAddrStr(), len(msg))
	envelope := protocol.RendezvousMessage{}
	if err := proto.Unmarshal(msg, &envelope); err != nil || envelope.Union == nil {
		if err != nil {
			transport.Warn("RELAY unmarshal from %s: %v", writer.GetAddrStr(), err)
		}
		return
	}
	msgType := reflect.TypeOf(envelope.Union).String()
	transport.Info("RELAY-RX type %s from %s", msgType, writer.GetAddrStr())
	switch msgType {
	case protocol.TypeRendezvousMessageRequestRelay:
		rr := envelope.GetRequestRelay()
		if rr == nil {
			return
		}
		if !transport.KeyMatches(rr.GetLicenceKey()) {
			transport.Warn("Relay authentication failed from %s: invalid key", writer.GetAddrStr())
			writer.Close()
			return
		}
		uuid := rr.GetUuid()
		transport.Info("RELAY uuid %s from %s %s", uuid, writer.GetAddrStr(), protocol.DecodeAddr(rr.SocketAddr))
		if uuid == "" {
			return
		}
		writer.Detach()
		w := waiting.match(uuid, writer)
		if w == nil {
			transport.Info("RELAY waiting for peer %s", uuid)
		} else {
			w.Detach()
			transport.Info("RELAY pairing %s <-> %s", writer.GetAddrStr(), w.GetAddrStr())
			transport.RemoveWriter(writer)
			transport.RemoveWriter(w)
			go writer.Copy(w)
		}
	default:
		transport.Info("RELAY unknown type %s", msgType)
	}
}

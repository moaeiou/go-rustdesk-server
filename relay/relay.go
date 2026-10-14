package relay

import (
	"context"
	"go-rustdesk-server/protocol"
	"go-rustdesk-server/transport"
	"google.golang.org/protobuf/proto"
	"reflect"
)

func Run(ctx context.Context) error {
	transport.Info("Relay TCP listening on port %d", transport.PortRelay)
	waiting := newWaitingPool()
	go waiting.run(ctx)
	defer waiting.close()
	handler := func(message []byte, writer *transport.Writer) { handlerMsg(message, writer, waiting) }
	return transport.NewMonitor(true, "tcp", transport.ListenAddress(transport.PortRelay), handler).Run(ctx)
}

func handlerMsg(msg []byte, writer *transport.Writer, waiting *waitingPool) {
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

package rendezvous

import (
	"go-rustdesk-server/protocol"
	"go-rustdesk-server/transport"
	"google.golang.org/protobuf/proto"
	"reflect"
)

func handlerMsg(msg []byte, writer *transport.Writer) {
	transport.Info("RX %s %s len %d", writer.Type(), writer.GetAddrStr(), len(msg))
	envelope := protocol.RendezvousMessage{}
	if err := proto.Unmarshal(msg, &envelope); err != nil {
		transport.Warn("unmarshal from %s: %v", writer.GetAddrStr(), err)
		return
	}
	if envelope.Union == nil {
		transport.Info("empty Union from %s", writer.GetAddrStr())
		return
	}
	if !requestLimiter.Allow(writer.GetAddr().GetIP()) {
		transport.Warn("rate limit exceeded by %s", writer.GetAddr().GetIP())
		return
	}
	msgType := reflect.TypeOf(envelope.Union).String()
	transport.Info("RX type %s from %s %s", msgType, writer.Type(), writer.GetAddrStr())
	var response proto.Message
	switch msgType {
	case protocol.TypeRendezvousMessagePunchHoleRequest:
		if req := envelope.GetPunchHoleRequest(); req != nil {
			go func(r *protocol.PunchHoleRequest, w *transport.Writer) {
				resp := RendezvousMessagePunchHoleRequest(r, w)
				if resp != nil {
					transport.Info("TX response to %s %s", w.Type(), w.GetAddrStr())
					w.SendMsg(protocol.NewRendezvousMessage(resp))
				}
			}(req, writer)
		}
		return
	case protocol.TypeRendezvousMessageRegisterPk:
		if req := envelope.GetRegisterPk(); req != nil {
			response = protocol.NewRendezvousMessage(RendezvousMessageRegisterPk(req, writer))
		}
	case protocol.TypeRendezvousMessageRegisterPeer:
		if req := envelope.GetRegisterPeer(); req != nil {
			response = protocol.NewRendezvousMessage(RendezvousMessageRegisterPeer(req, writer))
		}
	case protocol.TypeRendezvousMessageSoftwareUpdate:
		if req := envelope.GetSoftwareUpdate(); req != nil {
			response = protocol.NewRendezvousMessage(RendezvousMessageSoftwareUpdate(req))
		}
	case protocol.TypeRendezvousMessageTestNatRequest:
		if req := envelope.GetTestNatRequest(); req != nil {
			response = protocol.NewRendezvousMessage(RendezvousMessageTestNatRequest(req, writer))
		}
	case protocol.TypeRendezvousMessageLocalAddr:
		if req := envelope.GetLocalAddr(); req != nil {
			RendezvousMessageLocalAddr(req, writer)
		}
	case protocol.TypeRendezvousMessageRequestRelay:
		if req := envelope.GetRequestRelay(); req != nil {
			go func(r *protocol.RequestRelay, w *transport.Writer) {
				resp := RendezvousMessageRequestRelay(r, w)
				if resp != nil {
					transport.Info("TX response to %s %s", w.Type(), w.GetAddrStr())
					w.SendMsg(protocol.NewRendezvousMessage(resp))
				}
			}(req, writer)
		}
		return
	case protocol.TypeRendezvousMessageRelayResponse:
		if req := envelope.GetRelayResponse(); req != nil {
			RendezvousMessageRelayResponse(writer, req)
		}
	case protocol.TypeRendezvousMessagePunchHoleSent:
		if req := envelope.GetPunchHoleSent(); req != nil {
			RendezvousMessagePunchHoleSent(req, writer)
		}
	case protocol.TypeRendezvousMessageConfigureUpdate:
		if req := envelope.GetConfigureUpdate(); req != nil {
			RendezvousMessageConfigureUpdate(req)
		}
	case protocol.TypeRendezvousMessageOnlineRequest:
		if req := envelope.GetOnlineRequest(); req != nil {
			response = protocol.NewRendezvousMessage(RendezvousMessageOnlineRequest(req))
		}
	default:
		transport.Info("RX unknown type %s", msgType)
	}
	if response != nil {
		transport.Info("TX response to %s %s", writer.Type(), writer.GetAddrStr())
		writer.SendMsg(response)
	}
}

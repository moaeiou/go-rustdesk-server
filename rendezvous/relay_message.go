package rendezvous

import (
	"go-rustdesk-server/protocol"
	"go-rustdesk-server/transport"
)

func RendezvousMessageLocalAddr(req *protocol.LocalAddr, writer *transport.Writer) {
	msgNotifier.publish(req.GetId()+protocol.TypeRendezvousMessageLocalAddr, writer, req)
}

func RendezvousMessageRequestRelay(req *protocol.RequestRelay) *protocol.RelayResponse {
	peerWriter, err := transport.GetWriter(req.GetId(), transport.UDP)
	if err != nil {
		return nil
	}
	value, exists := memoryPeerMap.Load(req.Id)
	if !exists {
		return &protocol.RelayResponse{}
	}
	peer := value.(*Peer)
	peerWriter.SendMsg(protocol.NewRendezvousMessage(req))
	_, reply := getMsgForm(req.GetId(), []string{protocol.TypeRendezvousMessageRelayResponse}, 3)
	response, ok := reply.(*protocol.RelayResponse)
	if !ok {
		return &protocol.RelayResponse{}
	}
	response.Union = &protocol.RelayResponse_Pk{Pk: transport.GetSignPK(response.GetVersion(), response.GetId(), peer.PK)}
	return response
}

func RendezvousMessageRelayResponse(writer *transport.Writer, req *protocol.RelayResponse) {
	msgNotifier.publish(req.GetId()+protocol.TypeRendezvousMessageRelayResponse, writer, req)
}

func RendezvousMessagePunchHoleSent(req *protocol.PunchHoleSent, writer *transport.Writer) {
	transport.Info("PunchHoleSent id=%s relay=%s natType=%v", req.GetId(), req.GetRelayServer(), req.GetNatType())
	msgNotifier.publish(req.GetId()+protocol.TypeRendezvousMessagePunchHoleSent, writer, req)
}

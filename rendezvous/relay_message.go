package rendezvous

import (
	"go-rustdesk-server/protocol"
	"go-rustdesk-server/transport"
	"google.golang.org/protobuf/proto"
)

func RendezvousMessageLocalAddr(req *protocol.LocalAddr, writer *transport.Writer) {
	msgNotifier.publish(req.GetId()+protocol.TypeRendezvousMessageLocalAddr, writer, req)
}

func RendezvousMessageRequestRelay(req *protocol.RequestRelay, writer *transport.Writer) *protocol.RelayResponse {
	peerWriter, err := transport.GetWriter(req.GetId(), transport.UDP)
	if err != nil {
		return nil
	}
	value, exists := memoryPeerMap.Load(req.Id)
	if !exists {
		return &protocol.RelayResponse{}
	}
	peer := value.(*Peer)
	relayAddress := transport.RelayServerAddress(writer, peer.IP)
	transport.Info("Relay request id=%s advertised=%s requester=%s peer=%s", req.GetId(), relayAddress, writer.GetAddrStr(), peer.IP)
	forwarded := proto.Clone(req).(*protocol.RequestRelay)
	// The peer echoes this address in RelayResponse. It is the correlation key
	// for the requesting TCP connection; RelayResponse.id is optional.
	forwarded.SocketAddr = protocol.EncodeAddr(writer.GetAddrStr())
	forwarded.RelayServer = relayAddress
	wait := subscribeForMsg(writer.GetAddrStr(), []string{protocol.TypeRendezvousMessageRelayResponse})
	peerWriter.SendMsg(protocol.NewRendezvousMessage(forwarded))
	_, reply := wait(3)
	response, ok := reply.(*protocol.RelayResponse)
	if !ok {
		return &protocol.RelayResponse{}
	}
	// socket_addr only routes the response inside the rendezvous server and
	// must not be exposed back to the requester as a peer endpoint.
	response.SocketAddr = nil
	response.RelayServer = relayAddress
	response.Union = &protocol.RelayResponse_Pk{Pk: transport.GetSignPK(response.GetVersion(), response.GetId(), peer.PK)}
	return response
}

func RendezvousMessageRelayResponse(writer *transport.Writer, req *protocol.RelayResponse) {
	correlation := req.GetId()
	if socketAddr := req.GetSocketAddr(); len(socketAddr) > 0 {
		if requester := protocol.DecodeAddr(socketAddr); requester != "0.0.0.0:0" && requester != "[::]:0" {
			correlation = requester
		}
	}
	msgNotifier.publish(correlation+protocol.TypeRendezvousMessageRelayResponse, writer, req)
}

func RendezvousMessagePunchHoleSent(req *protocol.PunchHoleSent, writer *transport.Writer) {
	transport.Info("PunchHoleSent id=%s relay=%s natType=%v", req.GetId(), req.GetRelayServer(), req.GetNatType())
	msgNotifier.publish(req.GetId()+protocol.TypeRendezvousMessagePunchHoleSent, writer, req)
}

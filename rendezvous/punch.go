package rendezvous

import (
	"go-rustdesk-server/protocol"
	"go-rustdesk-server/transport"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var punchHoleGroup = newCallGroup()

func RendezvousMessagePunchHoleRequest(req *protocol.PunchHoleRequest, writer *transport.Writer) protoreflect.ProtoMessage {
	res := &protocol.PunchHoleResponse{}
	value, exists := memoryPeerMap.Load(req.Id)
	if !exists {
		res.Failure = protocol.PunchHoleResponse_ID_NOT_EXIST
		return res
	}
	peer := value.(*Peer)
	relayAddress := transport.RelayServerAddress(writer, peer.IP)
	transport.Info("Punch request id=%s advertised_relay=%s requester=%s peer=%s", req.GetId(), relayAddress, writer.GetAddrStr(), peer.IP)
	peerWriter, err := transport.GetWriter(peer.ID, transport.UDP)
	if err != nil {
		res.Failure = protocol.PunchHoleResponse_OFFLINE
		return res
	}

	peerIsLAN := transport.InSubnet(peer.IP)
	requesterIsLAN := transport.InSubnet(writer.GetAddr().GetIP())
	natType := req.GetNatType()
	if peerIsLAN != requesterIsLAN {
		natType = protocol.NatType_SYMMETRIC
	}
	sameIntranet := writer.GetAddr().GetIP() == peer.IP
	dedupKey := peer.ID
	if sameIntranet {
		dedupKey += ":local"
	}

	replyWriter, reply := punchHoleGroup.Do(dedupKey, func() (*transport.Writer, protoreflect.ProtoMessage) {
		if sameIntranet {
			peerWriter.SendMsg(protocol.NewRendezvousMessage(&protocol.FetchLocalAddr{
				SocketAddr: protocol.EncodeAddr(writer.GetAddrStr()), RelayServer: relayAddress,
			}))
			return getMsgForm(req.GetId(), []string{protocol.TypeRendezvousMessageLocalAddr}, 3)
		}
		peerWriter.SendMsg(protocol.NewRendezvousMessage(&protocol.PunchHole{
			SocketAddr: protocol.EncodeAddr(writer.GetAddrStr()), RelayServer: relayAddress, NatType: natType,
		}))
		return getMsgForm(req.GetId(), []string{
			protocol.TypeRendezvousMessagePunchHoleSent,
			protocol.TypeRendezvousMessageRelayResponse,
		}, 3)
	})
	if reply == nil {
		res.OtherFailure = "NoReturnMessage"
		return res
	}
	if local, ok := reply.(*protocol.LocalAddr); ok {
		res.SocketAddr = local.GetLocalAddr()
		res.RelayServer = relayAddress
		res.Pk = transport.GetSignPK(local.GetVersion(), peer.ID, peer.PK)
		res.Union = &protocol.PunchHoleResponse_IsLocal{IsLocal: true}
		return res
	}
	if sent, ok := reply.(*protocol.PunchHoleSent); ok {
		res.SocketAddr = protocol.EncodeAddr(replyWriter.GetAddrStr())
		res.RelayServer = relayAddress
		res.Pk = transport.GetSignPK(sent.GetVersion(), peer.ID, peer.PK)
		res.Union = &protocol.PunchHoleResponse_NatType{NatType: sent.GetNatType()}
	}
	if relay, ok := reply.(*protocol.RelayResponse); ok {
		relay.SocketAddr = protocol.EncodeAddr(replyWriter.GetAddrStr())
		relay.RelayServer = relayAddress
		relay.Union = &protocol.RelayResponse_Pk{Pk: transport.GetSignPK(relay.GetVersion(), peer.ID, peer.PK)}
		return relay
	}
	return res
}

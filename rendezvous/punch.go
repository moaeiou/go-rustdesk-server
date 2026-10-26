package rendezvous

import (
	"go-rustdesk-server/protocol"
	"go-rustdesk-server/transport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var punchHoleGroup = newCallGroup()

func RendezvousMessagePunchHoleRequest(req *protocol.PunchHoleRequest, writer *transport.Writer) protoreflect.ProtoMessage {
	res := &protocol.PunchHoleResponse{}
	if !transport.KeyMatches(req.GetLicenceKey()) {
		transport.Warn("Punch authentication failed from %s for peer %s: invalid key", writer.GetAddrStr(), req.GetId())
		res.Failure = protocol.PunchHoleResponse_LICENSE_MISMATCH
		return res
	}
	value, exists := memoryPeerMap.Load(req.Id)
	if !exists {
		res.Failure = protocol.PunchHoleResponse_ID_NOT_EXIST
		return res
	}
	peer := value.(*Peer)
	peerWriter, err := transport.GetWriter(peer.ID, transport.UDP)
	if err != nil {
		res.Failure = protocol.PunchHoleResponse_OFFLINE
		return res
	}

	peerIsLAN := transport.InSubnet(peer.IP)
	requesterIsLAN := transport.InSubnet(writer.GetAddr().GetIP())
	natType := req.GetNatType()
	if transport.Config.AlwaysUseRelay || writer.IsWebSocket() || peerIsLAN != requesterIsLAN {
		natType = protocol.NatType_SYMMETRIC
	}
	sameIntranet := shouldFetchLocalAddr(
		transport.Config.AlwaysUseRelay, writer.IsWebSocket(), peerIsLAN, requesterIsLAN,
		writer.GetAddr().GetIP(), peer.IP,
	)
	dedupKey := peer.ID + ":" + writer.GetAddrStr()
	if sameIntranet {
		dedupKey += ":local"
	}

	replyWriter, reply, relayAddress := punchHoleGroup.Do(dedupKey, func() (*transport.Writer, protoreflect.ProtoMessage, string) {
		relayAddress := transport.RelayServerAddress(writer, peer.IP)
		transport.Info("Punch request id=%s advertised_relay=%s requester=%s peer=%s", req.GetId(), relayAddress, writer.GetAddrStr(), peer.IP)
		if sameIntranet {
			wait := subscribeForMsg(writer.GetAddrStr(), []string{protocol.TypeRendezvousMessageLocalAddr})
			peerWriter.SendMsg(protocol.NewRendezvousMessage(&protocol.FetchLocalAddr{
				SocketAddr: protocol.EncodeAddr(writer.GetAddrStr()), RelayServer: relayAddress,
			}))
			replyWriter, reply := wait(3)
			return replyWriter, reply, relayAddress
		}
		wait := subscribeForKeys([]string{
			writer.GetAddrStr() + protocol.TypeRendezvousMessagePunchHoleSent,
			writer.GetAddrStr() + protocol.TypeRendezvousMessageRelayResponse,
		})
		peerWriter.SendMsg(protocol.NewRendezvousMessage(&protocol.PunchHole{
			SocketAddr: protocol.EncodeAddr(writer.GetAddrStr()), RelayServer: relayAddress, NatType: natType,
		}))
		replyWriter, reply := wait(3)
		return replyWriter, reply, relayAddress
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
		relay = proto.Clone(relay).(*protocol.RelayResponse)
		relay.SocketAddr = nil
		relay.RelayServer = relayAddress
		relay.Union = &protocol.RelayResponse_Pk{Pk: transport.GetSignPK(relay.GetVersion(), peer.ID, peer.PK)}
		return relay
	}
	return res
}

func shouldFetchLocalAddr(alwaysUseRelay, websocket, peerIsLAN, requesterIsLAN bool, requesterIP, peerIP string) bool {
	return !alwaysUseRelay && !websocket && (peerIsLAN && requesterIsLAN || requesterIP == peerIP)
}

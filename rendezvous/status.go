package rendezvous

import (
	"go-rustdesk-server/protocol"
	"go-rustdesk-server/transport"
	"time"
)

func RendezvousMessageSoftwareUpdate(_ *protocol.SoftwareUpdate) *protocol.SoftwareUpdate {
	return &protocol.SoftwareUpdate{}
}

func RendezvousMessageTestNatRequest(req *protocol.TestNatRequest, writer *transport.Writer) *protocol.TestNatResponse {
	return &protocol.TestNatResponse{Port: int32(writer.GetAddr().GetPort()), Cu: configurationUpdate(req.GetSerial())}
}

func RendezvousMessageConfigureUpdate(req *protocol.ConfigUpdate) {
	transport.Info("ConfigureUpdate serial %v", req.Serial)
}

func RendezvousMessageOnlineRequest(req *protocol.OnlineRequest) *protocol.OnlineResponse {
	states := make([]byte, (len(req.GetPeers())+7)/8)
	for i, peerID := range req.GetPeers() {
		var peer *Peer
		if value, ok := memoryPeerMap.Load(peerID); ok {
			peer = value.(*Peer)
		}
		if peer != nil && peer.LastRegTime != nil && time.Since(*peer.LastRegTime) < 30*time.Second {
			states[i/8] |= 0x01 << (7 - i%8)
		}
	}
	return &protocol.OnlineResponse{States: states}
}

func configurationUpdate(serial int32) *protocol.ConfigUpdate {
	if transport.Config.ConfigSerial <= serial {
		return nil
	}
	return &protocol.ConfigUpdate{Serial: transport.Config.ConfigSerial, RendezvousServers: append([]string(nil), transport.Config.RendezvousServers...)}
}

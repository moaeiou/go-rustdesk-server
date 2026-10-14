package rendezvous

import (
	"go-rustdesk-server/protocol"
	"go-rustdesk-server/transport"
	"sync"
	"time"
)

var (
	memoryPeerMap sync.Map
	relayAddr     = transport.RelayServer()
)

func RendezvousMessageRegisterPeer(req *protocol.RegisterPeer, writer *transport.Writer) *protocol.RegisterPeerResponse {
	response := &protocol.RegisterPeerResponse{}
	value, found := memoryPeerMap.Load(req.GetId())
	if !found {
		response.RequestPk = true
		return response
	}
	peer := value.(*Peer)
	oldWriter, err := transport.GetWriter(req.GetId(), transport.UDP)
	changed := err != nil || oldWriter.GetAddrStr() != writer.GetAddrStr()
	now := time.Now()
	response.RequestPk = len(peer.PK) == 0 || changed
	updated := *peer
	updated.LastRegTime = &now
	if changed {
		updated.IP = writer.GetAddr().GetIP()
	}
	memoryPeerMap.Store(updated.ID, &updated)
	writer.SetKey(req.GetId())
	return response
}

func RendezvousMessageRegisterPk(req *protocol.RegisterPk, writer *transport.Writer) *protocol.RegisterPkResponse {
	response := &protocol.RegisterPkResponse{Result: protocol.RegisterPkResponse_SERVER_ERROR}
	if len(req.GetId()) < 6 {
		response.Result = protocol.RegisterPkResponse_UUID_MISMATCH
		return response
	}
	lookupID, changing := req.GetId(), req.GetOldId() != ""
	if changing {
		lookupID = req.GetOldId()
	}
	peer := findPeer(lookupID, string(req.GetUuid()))
	if peer != nil && !changing {
		if peer.UUID != string(req.GetUuid()) {
			response.Result = protocol.RegisterPkResponse_ID_EXISTS
			return response
		}
		if string(peer.PK) != string(req.GetPk()) {
			if peer.IP != writer.GetAddr().GetIP() {
				response.Result = protocol.RegisterPkResponse_UUID_MISMATCH
				return response
			}
			changing = true
		}
		response.Result = protocol.RegisterPkResponse_OK
	}
	oldWriter, err := transport.GetWriter(req.GetId(), transport.UDP)
	addressChanged := err == nil && oldWriter.GetAddrStr() != writer.GetAddrStr()
	if addressChanged && peer != nil {
		updated := *peer
		updated.IP = writer.GetAddr().GetIP()
		peer = &updated
		memoryPeerMap.Store(updated.ID, peer)
		writer.SetKey(peer.ID)
	}
	dirty := addressChanged || changing || peer == nil
	if dirty {
		uid := ""
		if peer != nil {
			uid = peer.Uid
		}
		now := time.Now()
		peer = &Peer{Uid: uid, ID: req.Id, UUID: string(req.Uuid), PK: req.Pk, LastRegTime: &now, IP: writer.GetAddr().GetIP()}
		memoryPeerMap.Store(peer.ID, peer)
		if lookupID != peer.ID {
			memoryPeerMap.Delete(lookupID)
		}
		response.Result = protocol.RegisterPkResponse_OK
		writer.SetKey(req.GetId())
	}
	if dirty {
		if err := savePeers(); err != nil {
			transport.Warn("save peers: %v", err)
		}
	}
	return response
}

func findPeer(id, uuid string) *Peer {
	if value, found := memoryPeerMap.Load(id); found {
		return value.(*Peer)
	}
	var result *Peer
	memoryPeerMap.Range(func(_, value interface{}) bool {
		peer := value.(*Peer)
		if peer.UUID == uuid {
			result = peer
			return false
		}
		return true
	})
	return result
}

package rendezvous

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"go-rustdesk-server/transport"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var peerStoreMu sync.Mutex

type peerFile struct {
	Peers []storedPeer `toml:"peer"`
}

type storedPeer struct {
	UID              string `toml:"uid,omitempty"`
	ID               string `toml:"id"`
	IP               string `toml:"ip,omitempty"`
	UUIDBase64       string `toml:"uuid_base64"`
	PublicKeyBase64  string `toml:"public_key_base64"`
	User             string `toml:"user,omitempty"`
	Disabled         bool   `toml:"disabled,omitempty"`
	LastRegisteredAt string `toml:"last_registered_at,omitempty"`
}

func peerStorePath() string { return filepath.Join(transport.DataDir(), "peers.toml") }

func loadPeers() error {
	peers, err := readPeersTOML(peerStorePath())
	if err == nil {
		storePeersInMemory(peers)
		return removeLegacyPeerFiles()
	}
	if !os.IsNotExist(err) {
		return err
	}
	peers, migrated, err := readLegacyPeers()
	if err != nil {
		return err
	}
	if !migrated {
		return removeLegacyPeerFiles()
	}
	storePeersInMemory(peers)
	if err := savePeers(); err != nil {
		return fmt.Errorf("migrate peers to TOML: %w", err)
	}
	transport.Info("Migrated peer store to %s", peerStorePath())
	return removeLegacyPeerFiles()
}

func storePeersInMemory(peers []*Peer) {
	for _, peer := range peers {
		if peer != nil && peer.ID != "" {
			memoryPeerMap.Store(peer.ID, peer)
		}
	}
}

func readPeersTOML(path string) ([]*Peer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file peerFile
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	peers := make([]*Peer, 0, len(file.Peers))
	for _, stored := range file.Peers {
		peer, err := stored.peer()
		if err != nil {
			return nil, fmt.Errorf("parse peer %q: %w", stored.ID, err)
		}
		peers = append(peers, peer)
	}
	return peers, nil
}

func readLegacyPeers() ([]*Peer, bool, error) {
	paths := []string{
		filepath.Join(transport.DataDir(), "peers.json"),
		filepath.Join(transport.DataDir(), "peers.json.bak"),
	}
	var lastError error
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			lastError = err
			continue
		}
		var peers []*Peer
		if err := json.Unmarshal(data, &peers); err != nil {
			lastError = fmt.Errorf("parse %s: %w", path, err)
			continue
		}
		return peers, true, nil
	}
	return nil, false, lastError
}

func savePeers() error {
	peerStoreMu.Lock()
	defer peerStoreMu.Unlock()
	peers := make([]*Peer, 0)
	memoryPeerMap.Range(func(_, value interface{}) bool { copy := *(value.(*Peer)); peers = append(peers, &copy); return true })
	sort.Slice(peers, func(i, j int) bool { return peers[i].ID < peers[j].ID })
	file := peerFile{Peers: make([]storedPeer, 0, len(peers))}
	for _, peer := range peers {
		file.Peers = append(file.Peers, storePeer(peer))
	}
	data, err := toml.Marshal(file)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(transport.DataDir(), "peers-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, peerStorePath()); err != nil {
		return err
	}
	directory, err := os.Open(transport.DataDir())
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func storePeer(peer *Peer) storedPeer {
	stored := storedPeer{
		UID:             peer.Uid,
		ID:              peer.ID,
		IP:              peer.IP,
		UUIDBase64:      base64.StdEncoding.EncodeToString([]byte(peer.UUID)),
		PublicKeyBase64: base64.StdEncoding.EncodeToString(peer.PK),
		User:            peer.User,
		Disabled:        peer.Disabled,
	}
	if peer.LastRegTime != nil {
		stored.LastRegisteredAt = peer.LastRegTime.UTC().Format(time.RFC3339Nano)
	}
	return stored
}

func (stored storedPeer) peer() (*Peer, error) {
	uuid, err := base64.StdEncoding.DecodeString(stored.UUIDBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid_base64: %w", err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(stored.PublicKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid public_key_base64: %w", err)
	}
	var lastRegisteredAt *time.Time
	if stored.LastRegisteredAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, stored.LastRegisteredAt)
		if err != nil {
			return nil, fmt.Errorf("invalid last_registered_at: %w", err)
		}
		lastRegisteredAt = &parsed
	}
	return &Peer{
		Uid: stored.UID, ID: stored.ID, IP: stored.IP, UUID: string(uuid), PK: publicKey,
		User: stored.User, Disabled: stored.Disabled, LastRegTime: lastRegisteredAt,
	}, nil
}

func removeLegacyPeerFiles() error {
	paths := []string{
		filepath.Join(transport.DataDir(), "peer.toml.bak"),
		filepath.Join(transport.DataDir(), "peers.toml.bak"),
		filepath.Join(transport.DataDir(), "peers.json"),
		filepath.Join(transport.DataDir(), "peers.json.bak"),
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove legacy peer store %s: %w", path, err)
		}
	}
	return nil
}

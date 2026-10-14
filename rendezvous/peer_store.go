package rendezvous

import (
	"encoding/json"
	"fmt"
	"go-rustdesk-server/transport"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var peerStoreMu sync.Mutex

func peerStorePath() string  { return filepath.Join(transport.DataDir(), "peers.json") }
func peerBackupPath() string { return peerStorePath() + ".bak" }

func loadPeers() error {
	peers, primaryErr := readPeers(peerStorePath())
	if primaryErr != nil {
		var backupErr error
		peers, backupErr = readPeers(peerBackupPath())
		if backupErr != nil {
			if os.IsNotExist(primaryErr) && os.IsNotExist(backupErr) {
				return nil
			}
			return fmt.Errorf("primary: %v; backup: %v", primaryErr, backupErr)
		}
		transport.Warn("load peers from backup because primary store is unavailable: %v", primaryErr)
	}
	for _, peer := range peers {
		if peer != nil && peer.ID != "" {
			memoryPeerMap.Store(peer.ID, peer)
		}
	}
	return nil
}

func readPeers(path string) ([]*Peer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var peers []*Peer
	if err := json.Unmarshal(data, &peers); err != nil {
		return nil, err
	}
	return peers, nil
}

func savePeers() error {
	peerStoreMu.Lock()
	defer peerStoreMu.Unlock()
	peers := make([]*Peer, 0)
	memoryPeerMap.Range(func(_, value interface{}) bool { copy := *(value.(*Peer)); peers = append(peers, &copy); return true })
	sort.Slice(peers, func(i, j int) bool { return peers[i].ID < peers[j].ID })
	data, err := json.MarshalIndent(peers, "", "  ")
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
	primaryPath := peerStorePath()
	if _, err := readPeers(primaryPath); err == nil {
		if err := os.Rename(primaryPath, peerBackupPath()); err != nil {
			return fmt.Errorf("backup peers: %w", err)
		}
	}
	if err := os.Rename(temporaryPath, primaryPath); err != nil {
		return err
	}
	directory, err := os.Open(transport.DataDir())
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

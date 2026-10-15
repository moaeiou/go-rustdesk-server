package transport

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"go-rustdesk-server/protocol"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

var pk []byte
var pkStr string
var sk []byte

func genKey(dir string) error {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}

	privText := base64.StdEncoding.EncodeToString(priv)
	pubText := base64.StdEncoding.EncodeToString(pub)

	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), []byte(privText), 0600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "id_ed25519.pub"), []byte(pubText), 0644)
}

func LoadKey() error {
	dir := DataDir()
	privPath := filepath.Join(dir, "id_ed25519")
	pubPath := filepath.Join(dir, "id_ed25519.pub")

	if !Exists(privPath) {
		if err := genKey(dir); err != nil {
			return fmt.Errorf("generate key: %w", err)
		}
	}

	privText, err := os.ReadFile(privPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	pubText, err := os.ReadFile(pubPath)
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}

	sk, err = base64.StdEncoding.DecodeString(string(privText))
	if err != nil {
		return fmt.Errorf("decode private key: %w", err)
	}

	pk, err = base64.StdEncoding.DecodeString(string(pubText))
	if err != nil {
		return fmt.Errorf("decode public key: %w", err)
	}
	if len(sk) != ed25519.PrivateKeySize || len(pk) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid Ed25519 key size")
	}
	if !bytes.Equal(sk[32:], pk) {
		return fmt.Errorf("public and private keys do not match")
	}

	pkStr = string(pubText)
	Info("key= %s", pkStr)
	return nil
}

func Sign(data []byte) []byte {
	return append(ed25519.Sign(sk, data), data...)
}

func GetSignPK(version, id string, peerPK []byte) []byte {
	if version == "" || id == "" {
		return []byte{}
	}
	marshal, _ := proto.Marshal(&protocol.IdPk{Id: id, Pk: peerPK})
	return Sign(marshal)
}

func GetPkStr() string { return pkStr }

// KeyMatches reports whether a client may use this server. RustDesk sends the
// configured server public key in licence_key when requesting a connection.
func KeyMatches(clientKey string) bool {
	return !Config.EnforceKey || clientKey == pkStr
}

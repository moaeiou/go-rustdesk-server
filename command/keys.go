package command

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
)

// GenerateKeyPair prints a fresh pair without modifying the running server's
// identity or overwriting id_ed25519 files.
func GenerateKeyPair(out io.Writer) error {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Public Key: %s\nSecret Key: %s\n", base64.StdEncoding.EncodeToString(public), base64.StdEncoding.EncodeToString(private))
	return err
}

func ValidateKeyPair(publicText, privateText string) error {
	public, err := base64.StdEncoding.DecodeString(publicText)
	if err != nil || len(public) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key")
	}
	private, err := base64.StdEncoding.DecodeString(privateText)
	if err != nil || len(private) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid secret key")
	}
	message := []byte("RustDesk key pair validation")
	if !ed25519.Verify(public, message, ed25519.Sign(private, message)) {
		return fmt.Errorf("key pair does not match")
	}
	return nil
}

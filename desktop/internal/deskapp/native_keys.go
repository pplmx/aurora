package deskapp

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
)

func (s *nativeServices) GenerateKeyPair() (string, string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(pub), base64.StdEncoding.EncodeToString(priv), nil
}

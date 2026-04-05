package lottery

import (
	"crypto/sha256"
	"encoding/hex"
)

func NameToAddress(name string) string {
	h := sha256.Sum256([]byte(name))
	return "0x" + hex.EncodeToString(h[:20])
}

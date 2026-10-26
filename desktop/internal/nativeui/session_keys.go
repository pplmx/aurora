package nativeui

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"

	"github.com/egoist/mygo/ui"
)

// rememberSuccessfulKey stores a private key only when the user enabled
// session key memory and the operation succeeded. Storing a key before a
// failed signature could preserve a wrong key and make the next attempt fail
// in a more confusing way.
func (a *App) rememberSuccessfulKey(publicKey, privateKey string) {
	publicKey, privateKey = strings.TrimSpace(publicKey), strings.TrimSpace(privateKey)
	if !a.rememberKeys || privateKey == "" {
		return
	}
	actual, ok := publicKeyForPrivate(privateKey)
	if !ok {
		return
	}
	// A form's optional public field can be wrong (Transfer From does not send
	// it to the service). Store under the key the private half really belongs
	// to, and refuse an explicitly different pair rather than create a mapping
	// that would fill a wrong key later.
	if publicKey != "" && publicKey != actual {
		return
	}
	if a.savedKeys == nil {
		a.savedKeys = make(map[string]string)
	}
	a.savedKeys[actual] = privateKey
}

// publicKeyForPrivate derives the base64 public half of an Ed25519 private key.
// The services accept the 64-byte form; deriving here keeps the key ring from
// trusting a public field that was never sent to the service.
func publicKeyForPrivate(privateKey string) (string, bool) {
	raw, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return "", false
	}
	public := ed25519.PrivateKey(raw).Public().(ed25519.PublicKey)
	return base64.StdEncoding.EncodeToString(public), true
}

// savedKey returns the private key remembered for a public key.
func (a *App) savedKey(publicKey string) (string, bool) {
	if a.savedKeys == nil || strings.TrimSpace(publicKey) == "" {
		return "", false
	}
	key, ok := a.savedKeys[strings.TrimSpace(publicKey)]
	return key, ok
}

func (a *App) rememberedKeyCount() int { return len(a.savedKeys) }

func (a *App) forgetSessionKeys() { a.savedKeys = nil }

// useSavedKeyButton offers a remembered private key when the form's signer
// public key matches one. It is a quiet action: the key itself stays in memory
// and only fills the field the user would otherwise paste into.
func (a *App) useSavedKeyButton(c *ui.Context, publicKey string, target *string) {
	key, ok := a.savedKey(publicKey)
	if !ok {
		return
	}
	if quietButton(c, "Use saved key").Clicked() {
		*target = key
		c.Toast(tr("Saved key filled."))
	}
}

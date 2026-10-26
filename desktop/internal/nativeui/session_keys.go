package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// rememberSuccessfulKey stores a private key only when the user enabled
// session key memory and the operation succeeded. Storing a key before a
// failed signature could preserve a wrong key and make the next attempt fail
// in a more confusing way.
func (a *App) rememberSuccessfulKey(publicKey, privateKey string) {
	publicKey, privateKey = strings.TrimSpace(publicKey), strings.TrimSpace(privateKey)
	if !a.rememberKeys || publicKey == "" || privateKey == "" {
		return
	}
	if a.savedKeys == nil {
		a.savedKeys = make(map[string]string)
	}
	a.savedKeys[publicKey] = privateKey
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

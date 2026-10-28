package deskapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pplmx/aurora/internal/utils"
)

// preferencesName is the desktop-only settings file. It sits beside the
// database, not in aurora.toml: these are window choices, not service
// configuration, and rewriting the user's TOML from a checkbox would be a
// surprising side effect. Session keys are deliberately absent.
const preferencesName = "desktop-settings.json"

// Preferences are the desktop choices that survive a launch. Only harmless
// presentation/behavior flags belong here: no key material, no database path.
type Preferences struct {
	LiveUpdates bool `json:"liveUpdates"`
}

// preferencesPath is empty when there is no database path yet; saving then
// reports an error instead of writing somewhere unexpected.
func preferencesPath(dbPath string) string {
	if dbPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(dbPath), preferencesName)
}

// loadPreferences ignores an absent or malformed file and returns the defaults:
// a corrupt cosmetic setting must not stop the window from opening, and the
// next save replaces it.
func loadPreferences(path string) Preferences {
	var p Preferences
	if path == "" {
		return p
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return p
	}
	_ = json.Unmarshal(data, &p)
	return p
}

func savePreferences(path string, p Preferences) error {
	if path == "" {
		return errors.New("desktop settings path is empty")
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := utils.WriteFileAtomic(path, data, 0o600); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict desktop settings: %w", err)
	}
	return nil
}

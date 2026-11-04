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
	LiveUpdates bool   `json:"liveUpdates"`
	Theme       string `json:"theme,omitempty"`
	// Language is the language the operator chose, empty when the window
	// follows the system. It is stored because MyGo reports only the language
	// in effect: without this field Settings cannot tell an explicit choice
	// from a system locale that happens to match it, and the chooser would
	// show the same state for both.
	Language string `json:"language,omitempty"`
	// LastPage is the page the window was showing when it last closed, so the
	// next launch opens there instead of always at the dashboard. Only the
	// stable page id is stored, never its title: a title is shown text and may
	// be translated, and a renamed page must open on its default rather than on
	// a page the record no longer names.
	LastPage string `json:"lastPage,omitempty"`
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

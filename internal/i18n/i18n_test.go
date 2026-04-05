package i18n

import (
	"testing"
)

func TestTranslator_Init(t *testing.T) {
	tr := Init("en")

	if tr == nil {
		t.Fatal("Init returned nil")
	}
	if tr.locale != "en" {
		t.Errorf("Locale = %v, want 'en'", tr.locale)
	}
}

func TestTranslator_T(t *testing.T) {
	tr := Init("en")

	tests := []struct {
		key      string
		expected string
	}{
		{"app.name", "Aurora - VRF Lottery System"},
		{"app.version", "Version"},
		{"cmd.create", "Create a new lottery"},
		{"cmd.history", "Show lottery history"},
		{"flag.participants", "Participant names (comma-separated)"},
		{"msg.success", "Lottery created successfully!"},
	}

	for _, tt := range tests {
		result := tr.T(tt.key)
		if result != tt.expected {
			t.Errorf("T(%q) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestTranslator_T_Chinese(t *testing.T) {
	tr := Init("zh")

	tests := []struct {
		key      string
		expected string
	}{
		{"app.name", "Aurora - VRF 抽奖系统"},
		{"app.version", "版本"},
		{"cmd.create", "创建新抽奖"},
	}

	for _, tt := range tests {
		result := tr.T(tt.key)
		if result != tt.expected {
			t.Errorf("T(%q) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestTranslator_T_MissingKey(t *testing.T) {
	tr := Init("en")

	result := tr.T("nonexistent.key")
	if result != "nonexistent.key" {
		t.Errorf("T(missing key) = %v, want key as fallback", result)
	}
}

func TestTranslator_TFormat(t *testing.T) {
	tr := Init("en")

	result := tr.TFormat("msg.exported", 5, "test.json")
	expected := "Exported 5 lottery records to test.json"

	if result != expected {
		t.Errorf("TFormat = %v, want %v", result, expected)
	}
}

func TestTranslator_SetLocale(t *testing.T) {
	tr := Init("en")
	tr.SetLocale("zh")

	if tr.GetLocale() != "zh" {
		t.Errorf("GetLocale() = %v, want 'zh'", tr.GetLocale())
	}
}

func TestTranslator_AvailableLocales(t *testing.T) {
	tr := Init("en")

	locales := tr.AvailableLocales()

	if len(locales) == 0 {
		t.Error("AvailableLocales should not be empty")
	}
	found := false
	for _, l := range locales {
		if l == "en" {
			found = true
			break
		}
	}
	if !found {
		t.Error("'en' should be in available locales")
	}
}

func TestGetTranslator(t *testing.T) {
	tr := GetTranslator()

	if tr == nil {
		t.Fatal("GetTranslator returned nil")
	}
}

func TestDetectLocale(t *testing.T) {
	locale := DetectLocale()

	if locale != "en" && locale != "zh" {
		t.Errorf("DetectLocale = %v, want 'en' or 'zh'", locale)
	}
}

func TestGetText(t *testing.T) {
	result := GetText("app.name")

	if result == "" {
		t.Error("GetText should not return empty string")
	}
}

func TestGetTextF(t *testing.T) {
	result := GetTextF("msg.exported", 10, "file.json")

	if result == "" {
		t.Error("GetTextF should not return empty string")
	}
}

func TestDetectAndInit(t *testing.T) {
	tr := DetectAndInit()

	if tr == nil {
		t.Fatal("DetectAndInit returned nil")
	}
}

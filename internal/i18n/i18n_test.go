package i18n

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTranslator_Init(t *testing.T) {
	tr := Init("en")

	require.NotNil(t, tr)
	require.Equal(t, "en", tr.locale)
}

func TestTranslator_T(t *testing.T) {
	tr := Init("en")

	tests := []struct {
		key      string
		expected string
	}{
		{"app.name", "Aurora - Blockchain System"},
		{"app.version", "Version"},
		{"lottery.create", "Create a new lottery"},
		{"lottery.history", "Show lottery history"},
		{"lottery.participants", "Participant names (comma-separated)"},
		{"lottery.success", "Lottery created successfully!"},
		{"voting.candidate.add", "Add a candidate"},
		{"nft.mint", "Mint a new NFT"},
		{"oracle.fetch", "Fetch data from source"},
		{"error.invalid_input", "Invalid input"},
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
		{"app.name", "Aurora - 区块链系统"},
		{"app.version", "版本"},
		{"lottery.create", "创建新抽奖"},
		{"voting.candidate.add", "添加候选人"},
		{"nft.mint", "铸造新 NFT"},
		{"oracle.fetch", "从数据源获取数据"},
		{"error.invalid_input", "输入无效"},
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

	result := tr.TFormat("lottery.exported", 5, "test.json")
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
	result := GetTextF("lottery.exported", 10, "file.json")

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

// TestTranslator_Concurrent_NoRace stresses the Translator's read/write
// surface from multiple goroutines. Without the mutex added in this
// round, this test trips -race within a few iterations.
func TestTranslator_Concurrent_NoRace(t *testing.T) {
	tr := Init("en")

	const readers = 8
	const writers = 4
	const iters = 200

	done := make(chan struct{})
	for i := 0; i < readers; i++ {
		go func() {
			for j := 0; j < iters; j++ {
				_ = tr.T("app.name")
				_ = tr.GetLocale()
				_ = tr.AvailableLocales()
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < writers; i++ {
		go func(id int) {
			for j := 0; j < iters; j++ {
				if j%2 == 0 {
					tr.SetLocale("zh")
				} else {
					tr.SetLocale("en")
				}
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < readers+writers; i++ {
		<-done
	}
}

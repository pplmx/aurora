package oracle

import (
	"testing"
)

func TestExtractByPath(t *testing.T) {
	tests := []struct {
		json string
		path string
		want string
	}{
		{`{"bitcoin":{"usd":50000}}`, "bitcoin.usd", "50000"},
		{`{"data":{"temp":25}}`, "data.temp", "25"},
		{`{"value":"test"}`, "value", "test"},
		{`{"price":123.45}`, "price", "123.45"},
		{`{"nested":{"deep":{"value":42}}}`, "nested.deep.value", "42"},
	}

	for _, tt := range tests {
		got := extractByPath(tt.json, tt.path)
		if got != tt.want {
			t.Errorf("extractByPath(%q, %q) = %v, want %v", tt.json, tt.path, got, tt.want)
		}
	}
}

func TestExtractByPathInvalid(t *testing.T) {
	result := extractByPath("not valid json", "any.path")
	if result != "not valid json" {
		t.Errorf("Should return original for invalid JSON")
	}

	result = extractByPath(`{"a":1}`, "b")
	if result != `{"a":1}` {
		t.Errorf("Should return original for non-existent path")
	}
}

func TestFetcherStructure(t *testing.T) {
	fetcher := NewFetcher()
	if fetcher.client == nil {
		t.Error("Client should not be nil")
	}
}

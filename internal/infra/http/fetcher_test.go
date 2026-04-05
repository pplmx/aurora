package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

func TestFetcher_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	data, err := fetcher.Get(server.URL)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(data) != "test response" {
		t.Errorf("Expected 'test response', got '%s'", string(data))
	}
}

func TestFetcher_Get_Error(t *testing.T) {
	fetcher := NewFetcher()
	_, err := fetcher.Get("http://invalid-domain-that-does-not-exist.local")
	if err == nil {
		t.Fatal("Expected error for invalid domain")
	}
}

func TestFetcher_FetchData_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"value": "test"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "source-1",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	data, err := fetcher.FetchData(source)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}

	if data.SourceID != "source-1" {
		t.Errorf("Expected SourceID 'source-1', got '%s'", data.SourceID)
	}

	if data.Value == "" {
		t.Error("Expected Value to not be empty")
	}
}

func TestFetcher_FetchData_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"result": "ok"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "source-2",
		URL:     server.URL,
		Method:  "POST",
		Enabled: true,
	}

	_, err := fetcher.FetchData(source)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}
}

func TestFetcher_FetchData_WithPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data": {"price": 123.45}}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "source-3",
		URL:     server.URL,
		Method:  "GET",
		Path:    "data.price",
		Enabled: true,
	}

	data, err := fetcher.FetchData(source)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}

	if data.Value != "123.45" {
		t.Errorf("Expected Value '123.45', got '%s'", data.Value)
	}
}

func TestFetcher_FetchData_InvalidURL(t *testing.T) {
	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:     "source-4",
		URL:    "http://invalid-url-that-does-not-exist.local",
		Method: "GET",
	}

	_, err := fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}
}

func TestFetcher_FetchData_NestedPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"outer": {"inner": {"deep": "nested-value"}}}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "source-5",
		URL:     server.URL,
		Method:  "GET",
		Path:    "outer.inner.deep",
		Enabled: true,
	}

	data, err := fetcher.FetchData(source)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}

	if data.Value != "nested-value" {
		t.Errorf("Expected Value 'nested-value', got '%s'", data.Value)
	}
}

func TestFetcher_FetchData_InvalidPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"value": "test"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "source-6",
		URL:     server.URL,
		Method:  "GET",
		Path:    "nonexistent.path",
		Enabled: true,
	}

	data, err := fetcher.FetchData(source)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}

	if !strings.Contains(data.Value, "value") {
		t.Errorf("Expected Value to contain 'value', got '%s'", data.Value)
	}
}

func TestFetcher_FetchData_NonJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain text response"))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "source-7",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	data, err := fetcher.FetchData(source)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}

	if data.Value != "plain text response" {
		t.Errorf("Expected Value 'plain text response', got '%s'", data.Value)
	}
}

func TestExtractByPath_Nested(t *testing.T) {
	result := extractByPath(`{"a": {"b": {"c": "value"}}}`, "a.b.c")
	if result != "value" {
		t.Errorf("Expected 'value', got '%s'", result)
	}
}

func TestExtractByPath_InvalidJSON(t *testing.T) {
	result := extractByPath("not valid json", "a.b")
	if result != "not valid json" {
		t.Errorf("Expected original string on invalid JSON, got '%s'", result)
	}
}

func TestExtractByPath_NonExistentPath(t *testing.T) {
	result := extractByPath(`{"a": "b"}`, "nonexistent")
	if result != `{"a": "b"}` {
		t.Errorf("Expected original string on non-existent path, got '%s'", result)
	}
}

func TestExtractByPath_NumberValue(t *testing.T) {
	result := extractByPath(`{"value": 123.45}`, "value")
	if result != "123.45" {
		t.Errorf("Expected '123.45', got '%s'", result)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetcher_FetchData_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.Header.Get("X-Custom-Header")
		w.Write([]byte(`{"result": "ok"}`))
	}))
	defer server.Close()

	fetcher := &Fetcher{
		client: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				req.Header.Set("X-Custom-Header", "custom-value")
				return http.DefaultTransport.RoundTrip(req)
			}),
		},
	}

	source := &oracle.DataSource{
		ID:      "source-8",
		URL:     server.URL,
		Method:  "GET",
		Headers: `{"X-Custom-Header": "custom-value"}`,
		Enabled: true,
	}

	_, err := fetcher.FetchData(source)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}
}

var _ http.RoundTripper = roundTripperFunc(nil)

type failingRoundTripper struct{}

func (f failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network error")
}

func TestFetcher_FetchData_NetworkError(t *testing.T) {
	fetcher := &Fetcher{
		client: &http.Client{
			Transport: failingRoundTripper{},
		},
	}

	source := &oracle.DataSource{
		ID:     "source-9",
		URL:    "http://example.com",
		Method: "GET",
	}

	_, err := fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected error for network failure")
	}
}

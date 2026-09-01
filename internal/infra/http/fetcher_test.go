package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/spf13/viper"
)

func TestFetcher_Get_Success(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test response"))
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

// TestNewFetcher_BareNumericDurationFromViper is the regression test for the
// v1.82 hand-off (TASK-118, ISS-110): NewFetcher must route the http.timeout /
// http.rateLimit.window keys through config.DurationSeconds, so a bare TOML
// number (`timeout = 60`) produces a 60s HTTP timeout and a 10s window — NOT
// the 60ns / 10ns that raw viper.GetDuration returns, which would fail every
// oracle fetch / reset the limiter budget on every request.
func TestNewFetcher_BareNumericDurationFromViper(t *testing.T) {
	viper.Set("http.timeout", 60)
	viper.Set("http.rateLimit.window", 10)
	t.Cleanup(func() { viper.Reset() })

	f := NewFetcher()
	if got := f.client.Timeout; got != 60*time.Second {
		t.Fatalf("expected 60s HTTP timeout from bare `timeout = 60`, got %v", got)
	}
	if got := f.rateLimiter.window; got != 10*time.Second {
		t.Fatalf("expected 10s window from bare `window = 10`, got %v", got)
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
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"value": "test"}`))
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
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"result": "ok"}`))
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
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data": {"price": 123.45}}`))
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
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"outer": {"inner": {"deep": "nested-value"}}}`))
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
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"value": "test"}`))
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
	// A mistyped/non-existent extraction path must fail closed (v1.63,
	// TASK-076) rather than returning the whole raw body as the value.
	if !errors.Is(err, oracle.ErrInvalidSource) {
		t.Fatalf("invalid path: got err=%v data=%v, want ErrInvalidSource", err, data)
	}
}

func TestFetcher_FetchData_NonJSON(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain text response"))
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
	result, ok := extractByPath(`{"a": {"b": {"c": "value"}}}`, "a.b.c")
	if !ok || result != "value" {
		t.Errorf("Expected ('value', true), got (%q, %v)", result, ok)
	}
}

func TestExtractByPath_InvalidJSON(t *testing.T) {
	result, ok := extractByPath("not valid json", "a.b")
	if ok {
		t.Errorf("Expected failed extraction on invalid JSON, got (%q, true)", result)
	}
}

func TestExtractByPath_NonExistentPath(t *testing.T) {
	result, ok := extractByPath(`{"a": "b"}`, "nonexistent")
	if ok {
		t.Errorf("Expected failed extraction on non-existent path, got (%q, true)", result)
	}
}

// TestExtractByPath_NonObject returns ("" ,false) when traversal reaches a
// scalar before the path is exhausted (e.g. path "a.b" against {"a":"x"}).
func TestExtractByPath_NonObject(t *testing.T) {
	result, ok := extractByPath(`{"a": "x"}`, "a.b")
	if ok {
		t.Errorf("Expected failure when traversing into a scalar, got (%q, true)", result)
	}
}

func TestExtractByPath_NumberValue(t *testing.T) {
	result, ok := extractByPath(`{"value": 123.45}`, "value")
	if !ok || result != "123.45" {
		t.Errorf("Expected ('123.45', true), got (%q, %v)", result, ok)
	}
}

// TestExtractByPath_EmptyStringIsAValue pins TASK-237/ISS-235: a leaf that
// legitimately resolves to "" is a real data point (ok=true), not an
// extraction failure — so FetchDataWithValidationContext records it instead of
// failing a healthy 200 fetch with ErrInvalidSource.
func TestExtractByPath_EmptyStringIsAValue(t *testing.T) {
	result, ok := extractByPath(`{"price": ""}`, "price")
	if !ok || result != "" {
		t.Errorf("Expected ('', true) for an empty-string leaf, got (%q, %v)", result, ok)
	}
}

// TestExtractByPath_NonScalarFailsClosed pins the second half of TASK-237/
// ISS-235: a path resolving to a nested object/array/null must be a failed
// extraction (ok=false), not fmt.Sprintf("%v") Go-representation garbage
// (map[usd:64000]) persisted as the oracle value.
func TestExtractByPath_NonScalarFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name, body, path string
	}{
		{"nested object", `{"price": {"usd": 64000}}`, "price"},
		{"array", `{"tags": ["a", "b"]}`, "tags"},
		{"null", `{"price": null}`, "price"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := extractByPath(tc.body, tc.path)
			if ok {
				t.Errorf("non-scalar leaf must fail extraction, got (%q, true)", result)
			}
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetcher_FetchData_WithHeaders(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.Header.Get("X-Custom-Header")
		_, _ = w.Write([]byte(`{"result": "ok"}`))
	}))
	defer server.Close()

	fetcher := &Fetcher{
		client: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				req.Header.Set("X-Custom-Header", "custom-value")
				return http.DefaultTransport.RoundTrip(req)
			}),
		},
		rateLimiter: NewRateLimiter(100, time.Minute),
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

// TestFetcher_FetchData_MistypedPathFailsClosed locks the v1.63 fix: a
// configured extraction path that does not match the JSON must cause the fetch
// to fail with ErrInvalidSource rather than silently storing the entire raw
// response body as the source's value (TASK-076, ISS-068). This uses a
// scripted round tripper so no loopback listener is required.
func TestFetcher_FetchData_MistypedPathFailsClosed(t *testing.T) {
	fetcher := &Fetcher{
		client: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"a": {"b": "value"}}`)),
				}, nil
			}),
		},
		rateLimiter: NewRateLimiter(100, time.Minute),
	}

	source := &oracle.DataSource{
		ID:      "source-mistyped",
		URL:     "https://example.com/data",
		Method:  "GET",
		Path:    "c.d", // not present in the JSON
		Enabled: true,
	}

	_, err := fetcher.FetchData(source)
	if !errors.Is(err, oracle.ErrInvalidSource) {
		t.Fatalf("mistyped path: got err %v, want ErrInvalidSource", err)
	}
}

// TestFetcher_FetchData_EmptyStringLeafIsRecorded pins TASK-237/ISS-235: an
// upstream that legitimately returns {"price": ""} with Path=price must yield a
// successful fetch whose Value is "" — not ErrInvalidSource (which used to
// abort a healthy 200 and make the scheduler back off + report "invalid
// source"). Scripted round tripper so it runs here without a loopback list
// ener.
func TestFetcher_FetchData_EmptyStringLeafIsRecorded(t *testing.T) {
	fetcher := &Fetcher{
		client: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"price": ""}`)),
				}, nil
			}),
		},
		rateLimiter: NewRateLimiter(100, time.Minute),
	}

	source := &oracle.DataSource{
		ID:      "source-empty-leaf",
		URL:     "https://example.com/data",
		Method:  "GET",
		Path:    "price",
		Enabled: true,
	}

	data, err := fetcher.FetchData(source)
	if err != nil {
		t.Fatalf("empty-string leaf must be a successful fetch, got err %v", err)
	}
	if data.Value != "" {
		t.Errorf("Expected Value '', got %q", data.Value)
	}
}

// TestFetcher_FetchData_NonScalarLeafFailsClosed pins the second half of
// TASK-237/ISS-235: a path resolving to a nested object must NOT persist
// fmt.Sprintf("%v", ...) garbage (map[usd:64000]) as the oracle value — the
// fetch fails closed with ErrInvalidSource just like a missing path.
func TestFetcher_FetchData_NonScalarLeafFailsClosed(t *testing.T) {
	fetcher := &Fetcher{
		client: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"price": {"usd": 64000}}`)),
				}, nil
			}),
		},
		rateLimiter: NewRateLimiter(100, time.Minute),
	}

	source := &oracle.DataSource{
		ID:      "source-object-leaf",
		URL:     "https://example.com/data",
		Method:  "GET",
		Path:    "price",
		Enabled: true,
	}

	_, err := fetcher.FetchData(source)
	if !errors.Is(err, oracle.ErrInvalidSource) {
		t.Fatalf("object leaf: got err %v, want ErrInvalidSource", err)
	}
}

type failingRoundTripper struct{}

func (f failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network error")
}

func TestFetcher_FetchData_NetworkError(t *testing.T) {
	fetcher := &Fetcher{
		client: &http.Client{
			Transport: failingRoundTripper{},
		},
		rateLimiter: NewRateLimiter(100, time.Minute),
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

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !rl.Allow("source-1") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	if rl.Allow("source-1") {
		t.Error("Request 4 should be rate limited")
	}
}

func TestRateLimiter_AllowDifferentSources(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	if !rl.Allow("source-a") {
		t.Error("source-a request 1 should be allowed")
	}
	if !rl.Allow("source-a") {
		t.Error("source-a request 2 should be allowed")
	}
	if rl.Allow("source-a") {
		t.Error("source-a request 3 should be rate limited")
	}

	if !rl.Allow("source-b") {
		t.Error("source-b request 1 should be allowed (different source)")
	}
	if !rl.Allow("source-b") {
		t.Error("source-b request 2 should be allowed")
	}
	if rl.Allow("source-b") {
		t.Error("source-b request 3 should be rate limited")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	rl.Allow("source-1")
	rl.Allow("source-1")

	if rl.Allow("source-1") {
		t.Error("Should be rate limited after 2 requests")
	}

	rl.Reset("source-1")

	if !rl.Allow("source-1") {
		t.Error("Should be allowed after reset")
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute)

	if rl.Remaining("source-1") != 5 {
		t.Errorf("Expected 5 remaining, got %d", rl.Remaining("source-1"))
	}

	rl.Allow("source-1")
	rl.Allow("source-1")

	if rl.Remaining("source-1") != 3 {
		t.Errorf("Expected 3 remaining, got %d", rl.Remaining("source-1"))
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond)

	rl.Allow("source-1")
	rl.Allow("source-1")

	if rl.Allow("source-1") {
		t.Error("Should be rate limited")
	}

	time.Sleep(60 * time.Millisecond)

	if !rl.Allow("source-1") {
		t.Error("Should be allowed after window expiry")
	}
}

func TestFetcher_RateLimited(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"value": "test"}`))
	}))
	defer server.Close()

	fetcher := NewFetcherWithConfig(2, time.Minute)

	source := &oracle.DataSource{
		ID:      "rate-test-1",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	_, _ = fetcher.FetchData(source)
	_, _ = fetcher.FetchData(source)

	_, err := fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected rate limit error on 3rd request")
	}

	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("Expected ErrRateLimited, got %v", err)
	}
}

func TestFetcher_RateLimitResetsAfterWindow(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"value": "test"}`))
	}))
	defer server.Close()

	fetcher := NewFetcherWithConfig(1, 50*time.Millisecond)

	source := &oracle.DataSource{
		ID:      "rate-test-2",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	_, _ = fetcher.FetchData(source)

	_, err := fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected rate limit error")
	}

	time.Sleep(60 * time.Millisecond)

	_, err = fetcher.FetchData(source)
	if err != nil {
		t.Errorf("Should succeed after window reset, got: %v", err)
	}
}

func TestFetcher_Get_HasSecurityHeaders(t *testing.T) {
	skipIfLoopbackBlocked(t)
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		_, _ = w.Write([]byte("test"))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	_, _ = fetcher.Get(server.URL)

	// Hardened: a failed fetch used to leave capturedReq nil and panic the
	// test; fail cleanly instead.
	if capturedReq == nil {
		t.Fatal("request never reached the test server")
	}
	if capturedReq.Header.Get("User-Agent") == "" {
		t.Error("Expected User-Agent header to be set")
	}
	if capturedReq.Header.Get("Accept") == "" {
		t.Error("Expected Accept header to be set")
	}
	if capturedReq.Header.Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}

func TestFetcher_FetchData_HasSecurityHeaders(t *testing.T) {
	skipIfLoopbackBlocked(t)
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		_, _ = w.Write([]byte(`{"value": "test"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "header-test",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}
	_, _ = fetcher.FetchData(source)

	if capturedReq == nil {
		t.Fatal("request never reached the test server")
	}
	if capturedReq.Header.Get("User-Agent") != "Aurora/1.0" {
		t.Errorf("Expected User-Agent 'Aurora/1.0', got '%s'", capturedReq.Header.Get("User-Agent"))
	}
	if capturedReq.Header.Get("Accept") == "" {
		t.Error("Expected Accept header to be set")
	}
	if capturedReq.Header.Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}

func TestFetcher_FetchData_POST_HasContentType(t *testing.T) {
	skipIfLoopbackBlocked(t)
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		_, _ = w.Write([]byte(`{"result": "ok"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "post-header-test",
		URL:     server.URL,
		Method:  "POST",
		Enabled: true,
	}
	_, _ = fetcher.FetchData(source)

	if capturedReq == nil {
		t.Fatal("request never reached the test server")
	}
	if capturedReq.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", capturedReq.Header.Get("Content-Type"))
	}
}

func TestFetcher_FetchData_PUT_HasContentType(t *testing.T) {
	skipIfLoopbackBlocked(t)
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		_, _ = w.Write([]byte(`{"result": "ok"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "put-header-test",
		URL:     server.URL,
		Method:  "PUT",
		Enabled: true,
	}
	_, _ = fetcher.FetchData(source)

	if capturedReq == nil {
		t.Fatal("request never reached the test server")
	}
	if capturedReq.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", capturedReq.Header.Get("Content-Type"))
	}
}

func TestFetcher_CustomUserAgent(t *testing.T) {
	skipIfLoopbackBlocked(t)
	var capturedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		_, _ = w.Write([]byte(`{"value": "test"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher(WithUserAgent("CustomAgent/2.0"))
	source := &oracle.DataSource{
		ID:      "ua-test",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}
	_, _ = fetcher.FetchData(source)

	if capturedReq == nil {
		t.Fatal("request never reached the test server")
	}
	if capturedReq.Header.Get("User-Agent") != "CustomAgent/2.0" {
		t.Errorf("Expected User-Agent 'CustomAgent/2.0', got '%s'", capturedReq.Header.Get("User-Agent"))
	}
}

func TestFetcherWithTimeout_Success(t *testing.T) {
	fetcher, err := NewFetcherWithTimeout(10, time.Minute, 5*time.Second)
	if err != nil {
		t.Fatalf("NewFetcherWithTimeout failed: %v", err)
	}
	if fetcher.client.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", fetcher.client.Timeout)
	}
}

func TestFetcherWithTimeout_InvalidZero(t *testing.T) {
	_, err := NewFetcherWithTimeout(10, time.Minute, 0)
	if err == nil {
		t.Fatal("Expected error for zero timeout")
	}
	if !errors.Is(err, ErrInvalidTimeout) {
		t.Errorf("Expected ErrInvalidTimeout, got %v", err)
	}
}

func TestFetcherWithTimeout_InvalidNegative(t *testing.T) {
	_, err := NewFetcherWithTimeout(10, time.Minute, -1*time.Second)
	if err == nil {
		t.Fatal("Expected error for negative timeout")
	}
	if !errors.Is(err, ErrInvalidTimeout) {
		t.Errorf("Expected ErrInvalidTimeout, got %v", err)
	}
}

func TestFetcherWithTimeout_UsesDefaults(t *testing.T) {
	fetcher, err := NewFetcherWithTimeout(0, 0, 30*time.Second)
	if err != nil {
		t.Fatalf("NewFetcherWithTimeout failed: %v", err)
	}
	if fetcher.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", fetcher.client.Timeout)
	}
}

func TestFetcher_TimeoutFromConfig(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test"))
	}))
	defer server.Close()

	fetcher, err := NewFetcherWithTimeout(10, time.Minute, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("NewFetcherWithTimeout failed: %v", err)
	}

	start := time.Now()
	_, _ = fetcher.Get(server.URL)
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Errorf("Request took too long: %v", elapsed)
	}
}

func TestNewFetcherWithTimeout_SetsCorrectClientTimeout(t *testing.T) {
	expectedTimeout := 15 * time.Second
	fetcher, err := NewFetcherWithTimeout(5, time.Minute, expectedTimeout)
	if err != nil {
		t.Fatalf("NewFetcherWithTimeout failed: %v", err)
	}

	if fetcher.client.Timeout != expectedTimeout {
		t.Errorf("Expected client timeout %v, got %v", expectedTimeout, fetcher.client.Timeout)
	}
}

func TestNewFetcherWithTimeout_SetsRateLimiter(t *testing.T) {
	fetcher, err := NewFetcherWithTimeout(20, 2*time.Minute, 10*time.Second)
	if err != nil {
		t.Fatalf("NewFetcherWithTimeout failed: %v", err)
	}

	if fetcher.rateLimiter == nil {
		t.Fatal("Expected rate limiter to be set")
	}

	if fetcher.rateLimiter.Remaining("test") != 20 {
		t.Errorf("Expected rate limit 20, got %d", fetcher.rateLimiter.Remaining("test"))
	}
}

func TestFetcher_Get_HTTPError_4xx(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	_, err := fetcher.Get(server.URL)
	if err == nil {
		t.Fatal("Expected error for 4xx status")
	}
	if !errors.Is(err, ErrHTTPError) {
		t.Errorf("Expected ErrHTTPError, got %v", err)
	}
}

func TestFetcher_Get_HTTPError_5xx(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	_, err := fetcher.Get(server.URL)
	if err == nil {
		t.Fatal("Expected error for 5xx status")
	}
	if !errors.Is(err, ErrHTTPError) {
		t.Errorf("Expected ErrHTTPError, got %v", err)
	}
}

func TestFetcher_FetchData_HTTPError_4xx(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "error-1",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	_, err := fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected error for 4xx status")
	}
	if !errors.Is(err, ErrHTTPError) {
		t.Errorf("Expected ErrHTTPError, got %v", err)
	}
}

func TestFetcher_FetchData_HTTPError_5xx(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error": "service unavailable"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "error-2",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	_, err := fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected error for 5xx status")
	}
	if !errors.Is(err, ErrHTTPError) {
		t.Errorf("Expected ErrHTTPError, got %v", err)
	}
}

func TestFetcher_FetchData_EmptyResponse(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{})
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "empty-1",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	_, err := fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected error for empty response")
	}
	if !errors.Is(err, ErrEmptyResponse) {
		t.Errorf("Expected ErrEmptyResponse, got %v", err)
	}
}

func TestFetcher_FetchData_Validation_Success(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"price": 50000}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "valid-1",
		URL:     server.URL,
		Method:  "GET",
		Path:    "price",
		Enabled: true,
	}

	data, err := fetcher.FetchDataWithValidation(source, true)
	if err != nil {
		t.Fatalf("FetchDataWithValidation failed: %v", err)
	}

	if data.Value != "50000" {
		t.Errorf("Expected Value '50000', got '%s'", data.Value)
	}
}

func TestFetcher_FetchData_Validation_InvalidJSON(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "invalid-1",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	_, err := fetcher.FetchDataWithValidation(source, true)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
	if !errors.Is(err, ErrInvalidJSON) {
		t.Errorf("Expected ErrInvalidJSON, got %v", err)
	}
}

// TestFetcher_FetchData_Validation_InvalidPath locks the v1.63 fail-closed
// contract (TASK-076, ISS-068) for the validation-enabled fetch path: a
// configured extraction path that does not match the JSON must make
// FetchDataWithValidation return oracle.ErrInvalidSource — whether or not
// JSON validation is requested — instead of falling back to the whole raw
// body as the value. Uses a scripted round tripper so no loopback listener
// is required. (This test previously asserted the pre-v1.63 raw-fallback
// behavior and made `go test ./...` red; updated in v1.64, TASK-078.)
func TestFetcher_FetchData_Validation_InvalidPath(t *testing.T) {
	fetcher := &Fetcher{
		client: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"data": "test"}`)),
				}, nil
			}),
		},
		rateLimiter: NewRateLimiter(100, time.Minute),
	}
	source := &oracle.DataSource{
		ID:      "path-err-1",
		URL:     "https://example.com/feed",
		Method:  "GET",
		Path:    "nonexistent.path",
		Enabled: true,
	}

	if _, err := fetcher.FetchDataWithValidation(source, false); !errors.Is(err, oracle.ErrInvalidSource) {
		t.Fatalf("invalid path without validation: got err=%v, want ErrInvalidSource", err)
	}
	if data, err := fetcher.FetchDataWithValidation(source, true); !errors.Is(err, oracle.ErrInvalidSource) {
		t.Fatalf("invalid path with validation: got err=%v data=%v, want ErrInvalidSource", err, data)
	}
}

func TestFetcher_FetchData_Timeout(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{"data": "test"}`))
	}))
	defer server.Close()

	fetcher, err := NewFetcherWithTimeout(10, time.Minute, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewFetcherWithTimeout failed: %v", err)
	}

	source := &oracle.DataSource{
		ID:      "timeout-1",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	_, err = fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
}

func TestFetcher_FetchData_DefaultMethod(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"value": "test"}`))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:      "default-method",
		URL:     server.URL,
		Method:  "",
		Enabled: true,
	}

	_, err := fetcher.FetchData(source)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}
}

func TestFetcher_FetchData_EmptyURL(t *testing.T) {
	fetcher := NewFetcher()
	source := &oracle.DataSource{
		ID:     "empty-url",
		URL:    "",
		Method: "GET",
	}

	_, err := fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected error for empty URL")
	}
}

func TestFetcher_FetchData_RateLimitedError(t *testing.T) {
	skipIfLoopbackBlocked(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"value": "test"}`))
	}))
	defer server.Close()

	fetcher := NewFetcherWithConfig(1, time.Minute)

	source := &oracle.DataSource{
		ID:      "rate-1",
		URL:     server.URL,
		Method:  "GET",
		Enabled: true,
	}

	_, _ = fetcher.FetchData(source)

	_, err := fetcher.FetchData(source)
	if err == nil {
		t.Fatal("Expected rate limit error")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("Expected ErrRateLimited, got %v", err)
	}
}

func TestFetcher_FetchData_SuccessStatus(t *testing.T) {
	skipIfLoopbackBlocked(t)
	testCases := []int{200, 201, 202, 204}

	for _, status := range testCases {
		t.Run(fmt.Sprintf("Status_%d", status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				if status != 204 {
					_, _ = w.Write([]byte(`{"result": "ok"}`))
				}
			}))
			defer server.Close()

			fetcher := NewFetcher()
			source := &oracle.DataSource{
				ID:      fmt.Sprintf("success-%d", status),
				URL:     server.URL,
				Method:  "GET",
				Enabled: true,
			}

			data, err := fetcher.FetchData(source)
			if status == 204 {
				if err == nil {
					t.Fatal("Expected error for empty 204 response")
				}
				return
			}

			if err != nil {
				t.Fatalf("FetchData failed: %v", err)
			}
			if data.Value == "" {
				t.Error("Expected Value to not be empty")
			}
		})
	}
}

// TestRateLimiter_EvictsIdleSources pins the TASK-136/ISS-129 fix: the
// requests map must not retain every sourceID ever fetched (Reset was never
// wired to the source lifecycle). Once the map crosses limiterSweepThreshold,
// Allow reclaims keys whose last request predates the window.
func TestRateLimiter_EvictsIdleSources(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	for i := 0; i < limiterSweepThreshold+10; i++ {
		rl.requests[fmt.Sprintf("source-%d", i)] = []time.Time{time.Now().Add(-2 * time.Minute)}
	}

	if !rl.Allow("fresh") {
		t.Fatal("fresh source must be admitted")
	}
	if len(rl.requests) != 1 {
		t.Fatalf("expected only the fresh source to remain after eviction, got %d", len(rl.requests))
	}
}

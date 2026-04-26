package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/spf13/viper"
)

const defaultHTTPTimeout = 10 * time.Second

var (
	ErrRateLimited       = errors.New("rate limit exceeded")
	ErrInvalidTimeout    = errors.New("timeout must be greater than 0")
	ErrHTTPError         = errors.New("http error response")
	ErrInvalidJSON       = errors.New("invalid JSON response")
	ErrEmptyResponse     = errors.New("empty response body")
	ErrPathExtraction    = errors.New("path extraction failed")
)

type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

type Fetcher struct {
	client      *http.Client
	rateLimiter *RateLimiter
	userAgent   string
}

const defaultUserAgent = "Aurora/1.0"

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (r *RateLimiter) Allow(sourceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	requests := r.requests[sourceID]
	valid := make([]time.Time, 0, len(requests))
	for _, t := range requests {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= r.limit {
		r.requests[sourceID] = valid
		return false
	}

	valid = append(valid, now)
	r.requests[sourceID] = valid
	return true
}

func (r *RateLimiter) Reset(sourceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.requests, sourceID)
}

func (r *RateLimiter) Remaining(sourceID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	count := 0
	for _, t := range r.requests[sourceID] {
		if t.After(windowStart) {
			count++
		}
	}

	remaining := r.limit - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

func NewFetcher(opts ...FetcherOption) *Fetcher {
	limit := viper.GetInt("http.rateLimit.requests")
	if limit <= 0 {
		limit = 10
	}

	window := viper.GetDuration("http.rateLimit.window")
	if window <= 0 {
		window = time.Minute
	}

	timeout := viper.GetDuration("http.timeout")
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}

	f := &Fetcher{
		client:      &http.Client{Timeout: timeout},
		rateLimiter: NewRateLimiter(limit, window),
		userAgent:   defaultUserAgent,
	}
	for _, opt := range opts {
		opt(f)
	}
	f.client.Transport = &securityTransport{
		transport: http.DefaultTransport,
		userAgent: f.userAgent,
	}
	return f
}

func NewFetcherWithConfig(limit int, window time.Duration) *Fetcher {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	f := &Fetcher{
		client:      &http.Client{Timeout: defaultHTTPTimeout},
		rateLimiter: NewRateLimiter(limit, window),
		userAgent:   defaultUserAgent,
	}
	f.client.Transport = &securityTransport{
		transport: http.DefaultTransport,
		userAgent: f.userAgent,
	}
	return f
}

func NewFetcherWithTimeout(limit int, window, timeout time.Duration) (*Fetcher, error) {
	if timeout <= 0 {
		return nil, ErrInvalidTimeout
	}
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	f := &Fetcher{
		client:      &http.Client{Timeout: timeout},
		rateLimiter: NewRateLimiter(limit, window),
		userAgent:   defaultUserAgent,
	}
	f.client.Transport = &securityTransport{
		transport: http.DefaultTransport,
		userAgent: f.userAgent,
	}
	return f, nil
}

type FetcherOption func(*Fetcher)

func WithUserAgent(ua string) FetcherOption {
	return func(f *Fetcher) {
		f.userAgent = ua
	}
}

type securityTransport struct {
	transport http.RoundTripper
	userAgent string
}

func (t *securityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	if req.Header.Get("Content-Type") == "" && (req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodPatch) {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Request-ID", uuid.New().String())
	return t.transport.RoundTrip(req)
}

func (f *Fetcher) Get(url string) ([]byte, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrHTTPError, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return body, nil
}

func (f *Fetcher) FetchData(source *oracle.DataSource) (*oracle.OracleData, error) {
	return f.FetchDataWithValidation(source, false)
}

func (f *Fetcher) FetchDataWithValidation(source *oracle.DataSource, validateJSON bool) (*oracle.OracleData, error) {
	if !f.rateLimiter.Allow(source.ID) {
		return nil, fmt.Errorf("%w: source %s has exceeded rate limit", ErrRateLimited, source.ID)
	}

	if source.URL == "" {
		return nil, errors.New("source URL is required")
	}

	method := source.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequest(method, source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d for URL %s", ErrHTTPError, resp.StatusCode, source.URL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if len(body) == 0 {
		return nil, ErrEmptyResponse
	}

	if validateJSON {
		if !json.Valid(body) {
			return nil, fmt.Errorf("%w: response is not valid JSON", ErrInvalidJSON)
		}
	}

	value := string(body)
	if source.Path != "" {
		value = extractByPath(string(body), source.Path)
		if value == "" {
			return nil, errors.New("path extraction resulted in empty value")
		}
	}

	return &oracle.OracleData{
		ID:          uuid.New().String(),
		SourceID:    source.ID,
		Value:       value,
		RawResponse: string(body),
		Timestamp:   time.Now().Unix(),
	}, nil
}

func extractByPath(jsonStr, path string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if v, exists := m[part]; exists {
				current = v
			} else {
				return jsonStr
			}
		} else {
			return jsonStr
		}
	}

	if result, ok := current.(string); ok {
		return result
	}
	if result, ok := current.(float64); ok {
		return fmt.Sprintf("%v", result)
	}
	return fmt.Sprintf("%v", current)
}

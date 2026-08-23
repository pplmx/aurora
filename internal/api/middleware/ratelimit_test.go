package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedWindowLimiter_LimitAndWindowReset(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewFixedWindowLimiter(3, time.Minute, func() time.Time { return now })

	for i := 0; i < 3; i++ {
		if !l.Allow("ip") {
			t.Fatalf("request %d within budget should be allowed", i+1)
		}
	}
	if l.Allow("ip") {
		t.Fatal("4th request should be rejected")
	}

	// A different key is unaffected.
	if !l.Allow("other") {
		t.Fatal("different key should have its own budget")
	}

	// Advancing past the window resets the original key.
	now = now.Add(61 * time.Second)
	for i := 0; i < 3; i++ {
		if !l.Allow("ip") {
			t.Fatalf("request %d in new window should be allowed", i+1)
		}
	}
}

func TestFixedWindowLimiter_Concurrent(t *testing.T) {
	l := NewFixedWindowLimiter(50, time.Minute, nil)
	const goroutines = 20
	const each = 5
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < each; i++ {
				if l.Allow(fmt.Sprintf("client-%d", id)) {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			}
		}(g)
	}
	wg.Wait()
	// 20 clients * 5 each = 100, all under the per-client limit of 50.
	assert.Equal(t, goroutines*each, allowed, "every allowed request should be within its own client budget")
}

func TestRateLimit_MiddlewareReturns429(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewFixedWindowLimiter(2, time.Minute, func() time.Time { return now })
	handler := RateLimit(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// httptest requests share a RemoteAddr, so keying is by that address.
	req := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, r)
		return rr
	}

	require.Equal(t, http.StatusOK, req().Code)
	require.Equal(t, http.StatusOK, req().Code)
	third := req()
	assert.Equal(t, http.StatusTooManyRequests, third.Code)
	assert.Equal(t, "60", third.Header().Get("Retry-After"))
}

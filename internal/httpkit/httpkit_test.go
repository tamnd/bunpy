package httpkit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestLimitedConcurrency verifies that Limited enforces the per-host cap.
// It fires (2 × limit) concurrent requests against a slow test server and
// checks that the peak number of simultaneously active requests never
// exceeds the configured limit.
func TestLimitedConcurrency(t *testing.T) {
	const limit = 4
	const total = limit * 2

	var active atomic.Int32
	var peakMu sync.Mutex
	var peak int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := active.Add(1)
		defer active.Add(-1)
		peakMu.Lock()
		if n > peak {
			peak = n
		}
		peakMu.Unlock()
		time.Sleep(30 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rt := Default(limit)
	var wg sync.WaitGroup
	for range total {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", srv.URL, nil)
			resp, err := rt.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	if peak > int32(limit) {
		t.Errorf("peak concurrency %d exceeded per-host limit %d", peak, limit)
	}
}

// TestLimitedConcurrency_Unlimited verifies that a zero perHost value
// disables the semaphore — all requests run concurrently.
func TestLimitedConcurrency_Unlimited(t *testing.T) {
	const total = 8

	var active atomic.Int32
	var peakMu sync.Mutex
	var peak int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := active.Add(1)
		defer active.Add(-1)
		peakMu.Lock()
		if n > peak {
			peak = n
		}
		peakMu.Unlock()
		time.Sleep(30 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rt := Default(0) // unlimited

	var wg sync.WaitGroup
	for range total {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", srv.URL, nil)
			resp, err := rt.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	if peak < int32(total) {
		t.Errorf("unlimited transport: peak %d < total %d; semaphore may still be active", peak, total)
	}
}

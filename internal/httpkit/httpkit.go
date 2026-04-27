// Package httpkit is a thin layer over net/http that the v0.1.x
// PyPI client and wheel installer share. Two reasons it exists:
//
//   - A RoundTripper interface lets unit tests swap in a fixture
//     transport so CI never reaches the live PyPI index.
//   - A per-host concurrency limiter keeps us a polite citizen on
//     PyPI and on private indexes once those land.
//
// The default transport is plain net/http with sane timeouts; the
// limiter is a buffered channel keyed by request URL host.
package httpkit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RoundTripper is the surface PM code calls. It is satisfied by
// *http.Client (via the Do method) and by Limited and FixturesFS
// in this package.
type RoundTripper interface {
	Do(req *http.Request) (*http.Response, error)
}

// Default returns a RoundTripper backed by net/http with sane
// timeouts and a per-host concurrency limit of perHost. perHost
// of 0 or less disables the limiter (still useful for tests).
func Default(perHost int) RoundTripper {
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          32,
		MaxIdleConnsPerHost:   8,
	}
	c := &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second,
	}
	if perHost <= 0 {
		return clientAdapter{c: c}
	}
	return &Limited{client: clientAdapter{c: c}, perHost: perHost, gates: map[string]chan struct{}{}}
}

type clientAdapter struct{ c *http.Client }

func (a clientAdapter) Do(req *http.Request) (*http.Response, error) { return a.c.Do(req) }

// Limited wraps a RoundTripper with a per-host semaphore.
type Limited struct {
	client  RoundTripper
	perHost int

	mu    sync.Mutex
	gates map[string]chan struct{}
}

func (l *Limited) Do(req *http.Request) (*http.Response, error) {
	host := ""
	if req.URL != nil {
		host = req.URL.Host
	}
	gate := l.gate(host)
	gate <- struct{}{}
	defer func() { <-gate }()
	return l.client.Do(req)
}

func (l *Limited) gate(host string) chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	g, ok := l.gates[host]
	if !ok {
		g = make(chan struct{}, l.perHost)
		l.gates[host] = g
	}
	return g
}

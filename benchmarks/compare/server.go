// Package compare provides a local HTTP server for cross-tool benchmark
// comparisons. It serves the committed fixture data (benchmarks/fixtures/index)
// with wheel download URLs rewritten to point at the local listener, so both
// bunpy (via BUNPY_PYPI_INDEX_URL) and uv (via --index-url) can use the same
// server without touching the internet.
package compare

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FixtureHandler is an HTTP handler that serves benchmark fixture data.
// Index pages are served at /simple/{pkg}/ with wheel URLs rewritten to
// /files/{pkg}/{filename}. Wheel files are served from the fixtures tree.
type FixtureHandler struct {
	// fixturesRoot is the absolute path to benchmarks/fixtures/index.
	fixturesRoot string
	// addr is the base address (scheme + host:port) used when rewriting URLs.
	addr string
}

// NewFixtureHandler creates a FixtureHandler rooted at fixturesRoot.
// addr must be the scheme+host+port the server is listening on
// (e.g. "http://127.0.0.1:12345").
func NewFixtureHandler(fixturesRoot, addr string) *FixtureHandler {
	return &FixtureHandler{fixturesRoot: fixturesRoot, addr: addr}
}

func (h *FixtureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/simple/" || path == "/simple":
		w.WriteHeader(http.StatusOK)
	case strings.HasPrefix(path, "/simple/"):
		h.serveIndex(w, r)
	case strings.HasPrefix(path, "/files/"):
		h.serveWheel(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveIndex serves a PEP 691 JSON index page for a package, rewriting
// the wheel download URLs to point at this server's /files/ path.
func (h *FixtureHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	// /simple/{pkg}/ → pkg name
	pkg := strings.Trim(strings.TrimPrefix(r.URL.Path, "/simple/"), "/")
	if pkg == "" {
		http.NotFound(w, r)
		return
	}

	indexPath := filepath.Join(h.fixturesRoot, "pypi.org", "simple", pkg, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Parse, rewrite URLs, re-encode.
	var page indexPage
	if err := json.Unmarshal(data, &page); err != nil {
		http.Error(w, "bad fixture JSON", http.StatusInternalServerError)
		return
	}
	for i, f := range page.Files {
		// Extract just the filename from the original URL.
		parts := strings.Split(f.URL, "/")
		filename := parts[len(parts)-1]
		page.Files[i].URL = fmt.Sprintf("%s/files/%s/%s", h.addr, pkg, filename)
	}

	out, err := json.Marshal(page)
	if err != nil {
		http.Error(w, "marshal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
	w.WriteHeader(http.StatusOK)
	w.Write(out) //nolint:errcheck
}

// serveWheel serves a wheel file from the fixtures tree.
func (h *FixtureHandler) serveWheel(w http.ResponseWriter, r *http.Request) {
	// /files/{pkg}/{filename} → fixtures/index/files.example/{pkg}/{filename}
	rest := strings.TrimPrefix(r.URL.Path, "/files/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	pkg, filename := parts[0], parts[1]
	wheelPath := filepath.Join(h.fixturesRoot, "files.example", pkg, filename)
	http.ServeFile(w, r, wheelPath)
}

// indexPage is a minimal PEP 691 project page JSON shape.
type indexPage struct {
	Name  string      `json:"name"`
	Files []indexFile `json:"files"`
	Meta  any `json:"meta"`
}

type indexFile struct {
	Filename string            `json:"filename"`
	URL      string            `json:"url"`
	Hashes   map[string]string `json:"hashes"`
}

// StartServer starts an HTTP server on a random local port and returns
// the base URL and a shutdown function. The server is backed by the
// fixture data found in fixturesRoot.
func StartServer(fixturesRoot string) (baseURL string, shutdown func(), err error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, fmt.Errorf("listen: %w", err)
	}
	addr := "http://" + ln.Addr().String()
	handler := NewFixtureHandler(fixturesRoot, addr)
	srv := &http.Server{Handler: handler}
	go srv.Serve(ln) //nolint:errcheck
	return addr, func() { srv.Close() }, nil
}

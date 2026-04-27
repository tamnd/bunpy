package httpkit

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FixturesFS returns a RoundTripper that serves canned responses
// from a directory tree rooted at root. The on-disk layout mirrors
// the URL: a request to https://pypi.org/simple/foo/ resolves to
//
//	root/pypi.org/simple/foo/index.json
//	root/pypi.org/simple/foo/index.headers
//
// index.headers is optional. When present, it is a small text file
// with one header per line (Name: Value) plus an optional first
// line of just digits, which becomes the response status code.
//
// A missing fixture returns a synthetic 404 response.
//
// The fixture transport is the offline-first hook: tests pin every
// byte of every PyPI exchange so CI never hits the live index.
func FixturesFS(root string) RoundTripper {
	return fixtureTransport{root: root}
}

type fixtureTransport struct{ root string }

func (f fixtureTransport) Do(req *http.Request) (*http.Response, error) {
	if req.URL == nil {
		return nil, errors.New("httpkit fixtures: request has no URL")
	}
	dir, ok := f.dirFor(req.URL)
	if !ok {
		return nil, errors.New("httpkit fixtures: cannot map URL to fixtures root: " + req.URL.String())
	}
	body, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return notFound(req), nil
		}
		return nil, err
	}
	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    req,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	if hdr, err := os.Open(filepath.Join(dir, "index.headers")); err == nil {
		defer hdr.Close()
		applyHeaders(resp, hdr)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Type", firstNonEmpty(resp.Header.Get("Content-Type"), "application/vnd.pypi.simple.v1+json"))
	if etag := req.Header.Get("If-None-Match"); etag != "" && etag == resp.Header.Get("ETag") {
		resp.StatusCode = http.StatusNotModified
		resp.Status = "304 Not Modified"
		resp.Body = io.NopCloser(bytes.NewReader(nil))
		resp.ContentLength = 0
	}
	return resp, nil
}

func (f fixtureTransport) dirFor(u *url.URL) (string, bool) {
	if u.Host == "" {
		return "", false
	}
	parts := []string{f.root, u.Host}
	clean := strings.Trim(u.Path, "/")
	if clean != "" {
		parts = append(parts, strings.Split(clean, "/")...)
	}
	return filepath.Join(parts...), true
}

func notFound(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode:    404,
		Status:        "404 Not Found",
		Header:        http.Header{},
		Body:          io.NopCloser(bytes.NewReader(nil)),
		Request:       req,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: 0,
	}
}

func applyHeaders(resp *http.Response, r io.Reader) {
	scanner := bufio.NewScanner(r)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			if code, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && code > 0 {
				resp.StatusCode = code
				resp.Status = strconv.Itoa(code) + " " + http.StatusText(code)
				continue
			}
		}
		if i := strings.IndexByte(line, ':'); i > 0 {
			name := strings.TrimSpace(line[:i])
			value := strings.TrimSpace(line[i+1:])
			resp.Header.Add(name, value)
		}
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

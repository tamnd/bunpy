package httpkit

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestFixturesFSServesBody(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pypi.org", "simple", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"name":"demo"}`)
	if err := os.WriteFile(filepath.Join(dir, "index.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.headers"),
		[]byte("ETag: \"abc\"\nContent-Type: application/json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := FixturesFS(root)
	req, _ := http.NewRequestWithContext(context.Background(), "GET",
		"https://pypi.org/simple/demo/", nil)
	resp, err := tr.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != string(body) {
		t.Errorf("body: got %q, want %q", got, body)
	}
	if resp.Header.Get("ETag") != "\"abc\"" {
		t.Errorf("etag: got %q", resp.Header.Get("ETag"))
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content-type: got %q", resp.Header.Get("Content-Type"))
	}
}

func TestFixturesFSMissing404(t *testing.T) {
	tr := FixturesFS(t.TempDir())
	req, _ := http.NewRequestWithContext(context.Background(), "GET",
		"https://pypi.org/simple/missing/", nil)
	resp, err := tr.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestFixturesFSIfNoneMatch(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pypi.org", "simple", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.headers"),
		[]byte("ETag: \"v1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := FixturesFS(root)
	req, _ := http.NewRequestWithContext(context.Background(), "GET",
		"https://pypi.org/simple/demo/", nil)
	req.Header.Set("If-None-Match", "\"v1\"")
	resp, err := tr.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 304 {
		t.Errorf("status: got %d, want 304", resp.StatusCode)
	}
}

func TestFixturesFSStatusOverride(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "host", "x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte("oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.headers"),
		[]byte("500\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := FixturesFS(root)
	req, _ := http.NewRequestWithContext(context.Background(), "GET",
		"https://host/x/", nil)
	resp, err := tr.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 500 {
		t.Errorf("status: got %d, want 500", resp.StatusCode)
	}
}

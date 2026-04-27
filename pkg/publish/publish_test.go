package publish_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/publish"
)

// makeWheel creates a minimal wheel file at path/name with one METADATA entry.
func makeWheel(t *testing.T, dir, name, version string) string {
	t.Helper()
	whlName := fmt.Sprintf("%s-%s-py3-none-any.whl", name, version)
	path := filepath.Join(dir, whlName)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	meta := fmt.Sprintf("Metadata-Version: 2.1\nName: %s\nVersion: %s\n", name, version)
	fw, _ := zw.Create(fmt.Sprintf("%s-%s.dist-info/METADATA", name, version))
	fw.Write([]byte(meta))
	zw.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUploadSendsMultipart(t *testing.T) {
	var gotContentType string
	var gotBodyHas bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		mediaType, params, _ := mime.ParseMediaType(gotContentType)
		if mediaType == "multipart/form-data" {
			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err != nil {
					break
				}
				if p.FormName() == "content" {
					gotBodyHas = true
				}
				io.Copy(io.Discard, p)
			}
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	dir := t.TempDir()
	whl := makeWheel(t, dir, "mypkg", "0.1.0")
	req := publish.UploadRequest{
		Files:    []string{whl},
		Registry: ts.URL,
		Token:    "pypi-test-token",
	}
	_, err := publish.Upload(req)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if !strings.Contains(gotContentType, "multipart/form-data") {
		t.Errorf("want multipart/form-data, got %s", gotContentType)
	}
	if !gotBodyHas {
		t.Error("request body did not contain 'content' part")
	}
}

func TestUploadAuthHeader(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer ts.Close()

	dir := t.TempDir()
	whl := makeWheel(t, dir, "mypkg", "0.1.0")
	req := publish.UploadRequest{
		Files:    []string{whl},
		Registry: ts.URL,
		Token:    "pypi-secret-token",
	}
	publish.Upload(req)

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("__token__:pypi-secret-token"))
	if gotAuth != want {
		t.Errorf("want %s, got %s", want, gotAuth)
	}
}

func TestDryRunSkipsHTTP(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer ts.Close()

	dir := t.TempDir()
	whl := makeWheel(t, dir, "mypkg", "0.1.0")
	req := publish.UploadRequest{
		Files:    []string{whl},
		Registry: ts.URL,
		Token:    "pypi-test",
		DryRun:   true,
	}
	results, err := publish.Upload(req)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if called {
		t.Error("HTTP server was called during dry-run")
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
}

func TestUpload403ReturnsErrUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte("Forbidden"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	whl := makeWheel(t, dir, "mypkg", "0.1.0")
	req := publish.UploadRequest{
		Files:    []string{whl},
		Registry: ts.URL,
		Token:    "bad-token",
	}
	_, err := publish.Upload(req)
	if err == nil {
		t.Fatal("want ErrUnauthorized, got nil")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("want unauthorized error, got %v", err)
	}
}

func TestUpload400AlreadyExists(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte("File already exists."))
	}))
	defer ts.Close()

	dir := t.TempDir()
	whl := makeWheel(t, dir, "mypkg", "0.1.0")
	req := publish.UploadRequest{
		Files:    []string{whl},
		Registry: ts.URL,
		Token:    "pypi-test",
	}
	_, err := publish.Upload(req)
	if err == nil {
		t.Fatal("want ErrAlreadyExists, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("want already exists error, got %v", err)
	}
}

func TestUploadMetadataFields(t *testing.T) {
	var gotFields map[string]string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		gotFields = map[string]string{}
		if mediaType == "multipart/form-data" {
			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err != nil {
					break
				}
				if p.FormName() != "content" {
					data, _ := io.ReadAll(p)
					gotFields[p.FormName()] = string(data)
				} else {
					io.Copy(io.Discard, p)
				}
			}
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	dir := t.TempDir()
	whl := makeWheel(t, dir, "mypkg", "0.1.0")

	// Compute expected SHA-256.
	data, _ := os.ReadFile(whl)
	h := sha256.Sum256(data)
	wantSHA := fmt.Sprintf("%x", h)

	req := publish.UploadRequest{
		Files:    []string{whl},
		Registry: ts.URL,
		Token:    "pypi-test",
	}
	publish.Upload(req)

	if got := gotFields["sha256_digest"]; got != wantSHA {
		t.Errorf("sha256_digest: want %s, got %s", wantSHA, got)
	}
	if got := gotFields["name"]; got != "mypkg" {
		t.Errorf("name: want mypkg, got %s", got)
	}
	if got := gotFields["version"]; got != "0.1.0" {
		t.Errorf("version: want 0.1.0, got %s", got)
	}
}

package pypi

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/pkg/cache"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Foo":           "foo",
		"foo_bar":       "foo-bar",
		"foo.bar":       "foo-bar",
		"foo--bar":      "foo-bar",
		"FOO__bar..baz": "foo-bar-baz",
		"already-norm":  "already-norm",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestKindOfFilename(t *testing.T) {
	cases := []struct {
		name string
		want FileKind
	}{
		{"requests-2.31.0-py3-none-any.whl", FileWheel},
		{"requests-2.31.0.tar.gz", FileSdist},
		{"requests-2.31.0.zip", FileSdist},
		{"README.md", FileUnknown},
	}
	for _, c := range cases {
		if got := kindOfFilename(c.name); got != c.want {
			t.Errorf("kindOfFilename(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestVersionFromFilename(t *testing.T) {
	cases := []struct {
		project, name, want string
	}{
		{"requests", "requests-2.31.0-py3-none-any.whl", "2.31.0"},
		{"requests", "requests-2.31.0.tar.gz", "2.31.0"},
		{"my-pkg", "my_pkg-1.0.0-py3-none-any.whl", "1.0.0"},
		{"my-pkg", "my_pkg-1.0.0.tar.gz", "1.0.0"},
		{"x", "junk", ""},
	}
	for _, c := range cases {
		if got := versionFromFilename(c.project, c.name); got != c.want {
			t.Errorf("versionFromFilename(%q, %q) = %q, want %q",
				c.project, c.name, got, c.want)
		}
	}
}

// fixtureClient builds a Client that serves widget.json for any
// request to <base>/widget/.
func fixtureClient(t *testing.T) *Client {
	t.Helper()
	page, err := os.ReadFile(filepath.Join("testdata", "widget.json"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	dir := filepath.Join(root, "pypi.org", "simple", "widget")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), page, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.headers"),
		[]byte("ETag: \"v1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New()
	c.HTTP = httpkit.FixturesFS(root)
	return c
}

func TestGetParsesPage(t *testing.T) {
	c := fixtureClient(t)
	p, err := c.Get(context.Background(), "Widget")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "widget" {
		t.Errorf("name: got %q", p.Name)
	}
	if len(p.Files) != 4 {
		t.Errorf("files: got %d, want 4", len(p.Files))
	}
	if got, want := p.Versions, []string{"1.0.0", "1.1.0", "1.2.0"}; !equal(got, want) {
		t.Errorf("versions: got %v, want %v", got, want)
	}
	if p.Meta.APIVersion != "1.1" {
		t.Errorf("api version: got %q", p.Meta.APIVersion)
	}
	if p.Meta.LastSerial != 12345 {
		t.Errorf("last serial: got %d", p.Meta.LastSerial)
	}
	if p.Meta.ETag != "\"v1\"" {
		t.Errorf("etag: got %q", p.Meta.ETag)
	}
	var yanked, sdist int
	for _, f := range p.Files {
		if f.Yanked {
			yanked++
			if f.YankedReason == "" {
				t.Error("yanked file should carry reason")
			}
		}
		if f.Kind == "sdist" {
			sdist++
		}
		if f.Kind == "wheel" && f.RequiresPython == "" && f.Filename == "widget-1.0.0-py3-none-any.whl" {
			t.Error("wheel: requires_python should flow through")
		}
	}
	if yanked != 1 {
		t.Errorf("yanked count: got %d, want 1", yanked)
	}
	if sdist != 1 {
		t.Errorf("sdist count: got %d, want 1", sdist)
	}
}

func TestGetWithCacheETag(t *testing.T) {
	c := fixtureClient(t)
	idx, err := cache.NewIndex(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c.Cache = idx

	if _, err := c.Get(context.Background(), "widget"); err != nil {
		t.Fatal(err)
	}
	body, etag, ok := idx.Get("widget")
	if !ok || len(body) == 0 {
		t.Fatal("cache empty after first Get")
	}
	if etag != "\"v1\"" {
		t.Errorf("cached etag: got %q", etag)
	}

	c.HTTP = recordingTransport{inner: c.HTTP}
	p, err := c.Get(context.Background(), "widget")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Files) != 4 {
		t.Errorf("after revalidate: files = %d", len(p.Files))
	}
}

type recordingTransport struct {
	inner httpkit.RoundTripper
}

func (r recordingTransport) Do(req *http.Request) (*http.Response, error) {
	if req.Header.Get("If-None-Match") == "" {
		return nil, errors.New("recordingTransport: revalidation request missing If-None-Match")
	}
	return r.inner.Do(req)
}

func TestGetMissingProjectIs404(t *testing.T) {
	c := New()
	c.HTTP = httpkit.FixturesFS(t.TempDir())
	_, err := c.Get(context.Background(), "nope")
	if err == nil {
		t.Fatal("Get: want error, got nil")
	}
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Errorf("Get: want *NotFoundError, got %T (%v)", err, err)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pypi.org", "simple", "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New()
	c.HTTP = httpkit.FixturesFS(root)
	_, err := c.Get(context.Background(), "broken")
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Errorf("want parse error, got %v", err)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

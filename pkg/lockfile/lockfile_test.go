package lockfile

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadWriteRoundtrip(t *testing.T) {
	when := time.Date(2026, 4, 27, 7, 5, 16, 0, time.UTC)
	l := &Lock{
		Version:     1,
		Generated:   when,
		ContentHash: "sha256:deadbeef",
		Packages: []Package{
			{Name: "widget", Version: "1.1.0", Filename: "widget-1.1.0-py3-none-any.whl", URL: "https://files.example/widget/widget-1.1.0-py3-none-any.whl", Hash: "sha256:abcd"},
		},
	}
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bunpy.lock")
	if err := l.WriteFile(path); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Version != 1 || got.ContentHash != "sha256:deadbeef" {
		t.Errorf("header: %+v", got)
	}
	if !got.Generated.Equal(when) {
		t.Errorf("generated: got %v want %v", got.Generated, when)
	}
	if len(got.Packages) != 1 || got.Packages[0].Name != "widget" || got.Packages[0].Version != "1.1.0" {
		t.Errorf("packages: %+v", got.Packages)
	}
}

func TestWriteSortsPackages(t *testing.T) {
	l := &Lock{
		Version: 1,
		Packages: []Package{
			{Name: "zeta", Version: "1.0"},
			{Name: "alpha", Version: "1.0"},
			{Name: "Mid_Pkg", Version: "1.0"},
		},
	}
	out := string(l.Bytes())
	a := strings.Index(out, "name = \"alpha\"")
	m := strings.Index(out, "name = \"Mid_Pkg\"")
	z := strings.Index(out, "name = \"zeta\"")
	if a < 0 || m < 0 || z < 0 || !(a < m && m < z) {
		t.Errorf("not sorted: alpha=%d mid=%d zeta=%d\n%s", a, m, z, out)
	}
}

func TestUpsertReplaces(t *testing.T) {
	l := &Lock{Packages: []Package{{Name: "Widget", Version: "1.0.0"}}}
	l.Upsert(Package{Name: "widget", Version: "1.1.0"})
	if len(l.Packages) != 1 {
		t.Fatalf("packages: %+v", l.Packages)
	}
	if l.Packages[0].Version != "1.1.0" {
		t.Errorf("version not replaced: %+v", l.Packages[0])
	}
}

func TestUpsertNew(t *testing.T) {
	l := &Lock{Packages: []Package{{Name: "alpha", Version: "1.0"}}}
	l.Upsert(Package{Name: "beta", Version: "2.0"})
	if len(l.Packages) != 2 {
		t.Fatalf("packages: %+v", l.Packages)
	}
}

func TestRemove(t *testing.T) {
	l := &Lock{Packages: []Package{{Name: "alpha"}, {Name: "beta"}}}
	if !l.Remove("Alpha") {
		t.Fatal("Remove returned false for existing entry")
	}
	if len(l.Packages) != 1 || l.Packages[0].Name != "beta" {
		t.Errorf("packages after remove: %+v", l.Packages)
	}
	if l.Remove("nope") {
		t.Error("Remove returned true for missing entry")
	}
}

func TestFind(t *testing.T) {
	l := &Lock{Packages: []Package{{Name: "Foo_Bar", Version: "1.0"}}}
	if p, ok := l.Find("foo-bar"); !ok || p.Version != "1.0" {
		t.Errorf("Find: %+v ok=%v", p, ok)
	}
	if _, ok := l.Find("missing"); ok {
		t.Error("Find: found missing package")
	}
}

func TestHashDependenciesStable(t *testing.T) {
	a := HashDependencies([]string{"alpha", "widget>=1.0"})
	b := HashDependencies([]string{"widget>=1.0", "alpha"})
	if a != b {
		t.Errorf("hash differs: %q vs %q", a, b)
	}
	if !strings.HasPrefix(a, "sha256:") {
		t.Errorf("missing prefix: %q", a)
	}
}

func TestHashDependenciesIgnoresWhitespace(t *testing.T) {
	a := HashDependencies([]string{"widget>=1.0"})
	b := HashDependencies([]string{"  widget>=1.0  ", ""})
	if a != b {
		t.Errorf("whitespace not normalised: %q vs %q", a, b)
	}
}

func TestReadMissing(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "absent.lock"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Foo_Bar":     "foo-bar",
		"foo.bar":     "foo-bar",
		"foo___bar":   "foo-bar",
		"FooBar":      "foobar",
		"foo--__.bar": "foo-bar",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

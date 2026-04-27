package links

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setLinkDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("BUNPY_LINK_DIR", dir)
	return dir
}

func TestWriteRead(t *testing.T) {
	setLinkDir(t)
	want := Entry{
		Name:       "widget",
		Version:    "1.0.0",
		Source:     "/abs/widget",
		Registered: time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC),
	}
	if err := Write(want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read("widget")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestReadMissingReturnsErrNotFound(t *testing.T) {
	setLinkDir(t)
	_, err := Read("notapkg")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestWriteSetsRegisteredWhenZero(t *testing.T) {
	setLinkDir(t)
	before := time.Now().UTC().Add(-time.Second)
	if err := Write(Entry{Name: "widget", Source: "/abs/widget"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read("widget")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Registered.Before(before) {
		t.Errorf("Registered = %v, want >= %v", got.Registered, before)
	}
}

func TestDeleteMissingIsNoop(t *testing.T) {
	setLinkDir(t)
	if err := Delete("notapkg"); err != nil {
		t.Errorf("Delete on missing: %v", err)
	}
}

func TestList(t *testing.T) {
	dir := setLinkDir(t)
	entries := []Entry{
		{Name: "alpha", Source: "/a", Registered: time.Now().UTC()},
		{Name: "widget", Source: "/w", Registered: time.Now().UTC()},
		{Name: "beta", Source: "/b", Registered: time.Now().UTC()},
	}
	for _, e := range entries {
		if err := Write(e); err != nil {
			t.Fatalf("Write %s: %v", e.Name, err)
		}
	}
	got, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	wantOrder := []string{"alpha", "beta", "widget"}
	for i, e := range got {
		if e.Name != wantOrder[i] {
			t.Errorf("[%d] = %q, want %q", i, e.Name, wantOrder[i])
		}
	}
	// Sanity: registry dir actually exists where we expect.
	if _, err := os.Stat(filepath.Join(dir, "widget.json")); err != nil {
		t.Errorf("widget.json missing: %v", err)
	}
}

func TestListMissingDirReturnsEmpty(t *testing.T) {
	t.Setenv("BUNPY_LINK_DIR", filepath.Join(t.TempDir(), "does-not-exist"))
	got, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestDirHonoursEnv(t *testing.T) {
	want := t.TempDir()
	t.Setenv("BUNPY_LINK_DIR", want)
	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Dir = %q, want %q", got, want)
	}
}

func TestReadCorruptJSONReturnsError(t *testing.T) {
	dir := setLinkDir(t)
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Read("broken")
	if err == nil {
		t.Fatal("expected parse error")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want a real parse error", err)
	}
}

package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bunpy_history")
	h := loadHistory(path, 10)
	for _, e := range []string{"a = 1", "b = 2", "c = 3"} {
		h.append(e)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	want := "a = 1\nb = 2\nc = 3\n"
	if string(got) != want {
		t.Fatalf("file = %q, want %q", got, want)
	}

	h2 := loadHistory(path, 10)
	if len(h2.entries) != 3 {
		t.Fatalf("reload entries = %d, want 3", len(h2.entries))
	}
}

func TestHistoryCap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".h")
	h := loadHistory(path, 2)
	for _, e := range []string{"one", "two", "three"} {
		h.append(e)
	}
	if len(h.entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(h.entries))
	}
	if h.entries[0] != "two" || h.entries[1] != "three" {
		t.Fatalf("entries = %v, want [two three]", h.entries)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "two\nthree\n" {
		t.Fatalf("file = %q", got)
	}
}

func TestHistorySizeZeroDisables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".h")
	h := loadHistory(path, 0)
	h.append("nope")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("history file should not exist when size=0, got err=%v", err)
	}
	if len(h.entries) != 0 {
		t.Fatalf("entries = %d, want 0", len(h.entries))
	}
}

func TestHistoryMultilineRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".h")
	h := loadHistory(path, 5)
	h.append("x = 1\ny = 2")

	got, _ := os.ReadFile(path)
	if !bytes.Contains(got, []byte(`x = 1\ny = 2`)) {
		t.Fatalf("multiline not escaped on disk: %q", got)
	}

	h2 := loadHistory(path, 5)
	if len(h2.entries) != 1 || !strings.Contains(h2.entries[0], "\n") {
		t.Fatalf("reload lost newline: %#v", h2.entries)
	}
}

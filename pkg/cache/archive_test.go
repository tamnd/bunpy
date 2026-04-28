package cache

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestArchiveKey(t *testing.T) {
	key := ArchiveKey("abc123")
	if len(key) != 20 {
		t.Fatalf("ArchiveKey len = %d, want 20", len(key))
	}
	// Deterministic.
	if ArchiveKey("abc123") != key {
		t.Fatal("ArchiveKey not deterministic")
	}
	if ArchiveKey("different") == key {
		t.Fatal("ArchiveKey collision on different input")
	}
}

func TestPointerRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sniffio-1.3.1-py3-none-any.http")

	const (
		archiveKey = "abcdefghijklmnopqrst"
		sha256hex  = "deadbeef1234"
		filename   = "sniffio-1.3.1-py3-none-any.whl"
		url        = "https://files.example/sniffio.whl"
	)

	if err := WritePointer(path, archiveKey, sha256hex, filename, url); err != nil {
		t.Fatalf("WritePointer: %v", err)
	}
	gotKey, gotSHA, ok := ReadPointer(path)
	if !ok {
		t.Fatal("ReadPointer returned ok=false")
	}
	if gotKey != archiveKey {
		t.Errorf("archive key: got %q, want %q", gotKey, archiveKey)
	}
	if gotSHA != sha256hex {
		t.Errorf("sha256hex: got %q, want %q", gotSHA, sha256hex)
	}
}

func TestExtractAndInstall(t *testing.T) {
	// Build a minimal wheel zip in memory.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	addFile := func(name, content string) {
		w, _ := zw.Create(name)
		w.Write([]byte(content)) //nolint:errcheck
	}
	addFile("mypkg/__init__.py", "# init\n")
	addFile("mypkg/mod.py", "x = 1\n")
	addFile("mypkg-1.0.dist-info/METADATA", "Name: mypkg\nVersion: 1.0\n")
	addFile("mypkg-1.0.dist-info/WHEEL", "Wheel-Version: 1.0\n")
	addFile("mypkg-1.0.dist-info/INSTALLER", "uv\n")
	zw.Close()

	cacheRoot := t.TempDir()
	key := ArchiveKey("fakehash")

	if HasArchive(cacheRoot, key) {
		t.Fatal("HasArchive should be false before extraction")
	}
	if err := ExtractToArchive(cacheRoot, key, buf.Bytes()); err != nil {
		t.Fatalf("ExtractToArchive: %v", err)
	}
	if !HasArchive(cacheRoot, key) {
		t.Fatal("HasArchive should be true after extraction")
	}
	// Idempotent second call.
	if err := ExtractToArchive(cacheRoot, key, buf.Bytes()); err != nil {
		t.Fatalf("ExtractToArchive idempotent: %v", err)
	}

	targetDir := t.TempDir()
	if err := InstallFromArchive(cacheRoot, key, targetDir, "bunpy"); err != nil {
		t.Fatalf("InstallFromArchive: %v", err)
	}

	// Verify files were installed.
	for _, rel := range []string{
		"mypkg/__init__.py",
		"mypkg/mod.py",
		"mypkg-1.0.dist-info/METADATA",
	} {
		if _, err := os.Stat(filepath.Join(targetDir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing installed file %s: %v", rel, err)
		}
	}

	// INSTALLER must be overwritten with "bunpy".
	installerPath := filepath.Join(targetDir, "mypkg-1.0.dist-info", "INSTALLER")
	got, err := os.ReadFile(installerPath)
	if err != nil {
		t.Fatalf("read INSTALLER: %v", err)
	}
	if string(got) != "bunpy\n" {
		t.Errorf("INSTALLER = %q, want %q", got, "bunpy\n")
	}
}

func TestPointerPath(t *testing.T) {
	p := PointerPath("/cache", "requests", "requests-2.31.0-py3-none-any.whl")
	want := "/cache/wheels-v6/pypi/requests/2.31.0-py3-none-any.http"
	if p != want {
		t.Errorf("PointerPath = %q, want %q", p, want)
	}
}

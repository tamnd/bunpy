package wheel

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildWheel creates an in-memory wheel zip with the given body
// files plus dist-info/{WHEEL,METADATA,RECORD,...extras}. body is a
// map of zip-entry path to bytes; only paths outside dist-info count
// as body files.
func buildWheel(t *testing.T, name, version, wheelMeta string, body map[string]string, extras map[string]string) []byte {
	t.Helper()
	di := name + "-" + version + ".dist-info/"
	files := map[string][]byte{}
	for p, c := range body {
		files[p] = []byte(c)
	}
	files[di+"WHEEL"] = []byte(wheelMeta)
	files[di+"METADATA"] = []byte("Metadata-Version: 2.1\nName: " + name + "\nVersion: " + version + "\n")
	for p, c := range extras {
		files[di+p] = []byte(c)
	}
	// Build RECORD now that we know every body file. RECORD lists
	// itself with empty hash/size.
	rec := emitRECORD(files, di+"RECORD")
	files[di+"RECORD"] = rec

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for p, c := range files {
		w, err := zw.Create(p)
		if err != nil {
			t.Fatalf("zip create %s: %v", p, err)
		}
		if _, err := w.Write(c); err != nil {
			t.Fatalf("zip write %s: %v", p, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

const purelibTrue = "Wheel-Version: 1.0\nGenerator: bunpy/test\nRoot-Is-Purelib: true\nTag: py3-none-any\n"
const purelibFalse = "Wheel-Version: 1.0\nGenerator: bunpy/test\nRoot-Is-Purelib: false\nTag: py3-none-any\n"

func TestOpenParsesDistInfo(t *testing.T) {
	body := buildWheel(t, "tinypkg", "0.1.0", purelibTrue, map[string]string{
		"tinypkg/__init__.py": "x = 1\n",
	}, nil)
	w, err := OpenReader("tinypkg-0.1.0-py3-none-any.whl", body)
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	if w.Name != "tinypkg" {
		t.Errorf("Name = %q, want tinypkg", w.Name)
	}
	if w.Version != "0.1.0" {
		t.Errorf("Version = %q, want 0.1.0", w.Version)
	}
	if w.DistInfo != "tinypkg-0.1.0.dist-info/" {
		t.Errorf("DistInfo = %q", w.DistInfo)
	}
	if !w.WHEEL.RootIsPurelib {
		t.Errorf("RootIsPurelib = false, want true")
	}
	if len(w.Tags) != 1 || w.Tags[0] != (Tag{Python: "py3", ABI: "none", Platform: "any"}) {
		t.Errorf("Tags = %+v", w.Tags)
	}
	if !strings.Contains(string(w.Metadata), "Name: tinypkg") {
		t.Errorf("Metadata missing Name: tinypkg: %q", w.Metadata)
	}
}

func TestRecordParse(t *testing.T) {
	rec := []byte("foo/bar.py,sha256=abc,123\nfoo/baz.py,sha256=def,456\n")
	entries, err := parseRECORD(rec)
	if err != nil {
		t.Fatalf("parseRECORD: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Path != "foo/bar.py" || entries[0].Hash != "sha256=abc" || entries[0].Size != 123 {
		t.Errorf("entry 0 = %+v", entries[0])
	}
	if entries[1].Size != 456 {
		t.Errorf("entry 1 size = %d, want 456", entries[1].Size)
	}
}

func TestInstallPureLib(t *testing.T) {
	src := "x = 1\nprint(x)\n"
	body := buildWheel(t, "tinypkg", "0.1.0", purelibTrue, map[string]string{
		"tinypkg/__init__.py": src,
	}, nil)
	w, err := OpenReader("tinypkg-0.1.0-py3-none-any.whl", body)
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	target := t.TempDir()
	created, err := w.Install(target, InstallOptions{})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(created) == 0 {
		t.Fatal("Install returned no paths")
	}
	got, err := os.ReadFile(filepath.Join(target, "tinypkg", "__init__.py"))
	if err != nil {
		t.Fatalf("read installed file: %v", err)
	}
	if string(got) != src {
		t.Errorf("installed body = %q, want %q", got, src)
	}
	installer, err := os.ReadFile(filepath.Join(target, "tinypkg-0.1.0.dist-info", "INSTALLER"))
	if err != nil {
		t.Fatalf("read INSTALLER: %v", err)
	}
	if string(installer) != "bunpy\n" {
		t.Errorf("INSTALLER = %q, want %q", installer, "bunpy\n")
	}
}

func TestInstallWritesINSTALLER(t *testing.T) {
	body := buildWheel(t, "tinypkg", "0.1.0", purelibTrue, map[string]string{
		"tinypkg/__init__.py": "",
	}, nil)
	w, _ := OpenReader("tinypkg-0.1.0-py3-none-any.whl", body)
	target := t.TempDir()
	if _, err := w.Install(target, InstallOptions{Installer: "bunpy-test"}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(target, "tinypkg-0.1.0.dist-info", "INSTALLER"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "bunpy-test\n" {
		t.Errorf("INSTALLER = %q", got)
	}
}

func TestInstallRejectsRootIsPurelibFalse(t *testing.T) {
	body := buildWheel(t, "tinypkg", "0.1.0", purelibFalse, map[string]string{
		"tinypkg/__init__.py": "",
	}, nil)
	w, _ := OpenReader("tinypkg-0.1.0-py3-none-any.whl", body)
	_, err := w.Install(t.TempDir(), InstallOptions{})
	if err == nil {
		t.Fatal("expected error for Root-Is-Purelib: false")
	}
	if !strings.Contains(err.Error(), "Root-Is-Purelib") {
		t.Errorf("error = %v, want one mentioning Root-Is-Purelib", err)
	}
}

func TestInstallRejectsDataDir(t *testing.T) {
	body := buildWheel(t, "tinypkg", "0.1.0", purelibTrue, map[string]string{
		"tinypkg/__init__.py":         "",
		"tinypkg-0.1.0.data/scripts/x": "x",
	}, nil)
	w, _ := OpenReader("tinypkg-0.1.0-py3-none-any.whl", body)
	_, err := w.Install(t.TempDir(), InstallOptions{})
	if err == nil {
		t.Fatal("expected error for .data subdir")
	}
	if !strings.Contains(err.Error(), ".data") {
		t.Errorf("error = %v, want one mentioning .data", err)
	}
}

func TestInstallRejectsZipSlip(t *testing.T) {
	body := buildWheel(t, "tinypkg", "0.1.0", purelibTrue, map[string]string{
		"../etc/passwd": "x",
	}, nil)
	w, err := OpenReader("tinypkg-0.1.0-py3-none-any.whl", body)
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	_, err = w.Install(t.TempDir(), InstallOptions{})
	if err == nil {
		t.Fatal("expected zip-slip rejection")
	}
}

func TestInstallRejectsAbsolute(t *testing.T) {
	body := buildWheel(t, "tinypkg", "0.1.0", purelibTrue, map[string]string{
		"/etc/passwd": "x",
	}, nil)
	w, err := OpenReader("tinypkg-0.1.0-py3-none-any.whl", body)
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	_, err = w.Install(t.TempDir(), InstallOptions{})
	if err == nil {
		t.Fatal("expected absolute-path rejection")
	}
}

func TestInstallVerifyHashFails(t *testing.T) {
	// Build a wheel by hand whose RECORD claims a hash that does not
	// match the actual body. We cannot use buildWheel because that
	// computes RECORD honestly.
	di := "tinypkg-0.1.0.dist-info/"
	files := map[string][]byte{
		"tinypkg/__init__.py": []byte("real\n"),
		di + "WHEEL":          []byte(purelibTrue),
		di + "METADATA":       []byte("Metadata-Version: 2.1\nName: tinypkg\nVersion: 0.1.0\n"),
		di + "RECORD":         []byte(`tinypkg/__init__.py,sha256=DEADBEEF,5` + "\n" + di + "WHEEL,," + "\n" + di + "METADATA,," + "\n" + di + "RECORD,," + "\n"),
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for p, c := range files {
		w, _ := zw.Create(p)
		_, _ = w.Write(c)
	}
	zw.Close()
	w, err := OpenReader("tinypkg-0.1.0-py3-none-any.whl", buf.Bytes())
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	_, err = w.Install(t.TempDir(), InstallOptions{})
	if err == nil {
		t.Fatal("expected hash-mismatch error")
	}
	if !strings.Contains(err.Error(), "hash") {
		t.Errorf("error = %v, want one mentioning hash", err)
	}
}

func TestInstallSkipHashVerify(t *testing.T) {
	di := "tinypkg-0.1.0.dist-info/"
	files := map[string][]byte{
		"tinypkg/__init__.py": []byte("real\n"),
		di + "WHEEL":          []byte(purelibTrue),
		di + "METADATA":       []byte("Metadata-Version: 2.1\nName: tinypkg\nVersion: 0.1.0\n"),
		di + "RECORD":         []byte(`tinypkg/__init__.py,sha256=DEADBEEF,5` + "\n"),
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for p, c := range files {
		w, _ := zw.Create(p)
		_, _ = w.Write(c)
	}
	zw.Close()
	w, err := OpenReader("tinypkg-0.1.0-py3-none-any.whl", buf.Bytes())
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	off := false
	if _, err := w.Install(t.TempDir(), InstallOptions{VerifyHashes: &off}); err != nil {
		t.Fatalf("Install with VerifyHashes off: %v", err)
	}
}

func TestSha256Record(t *testing.T) {
	want := "sha256=" + base64.RawURLEncoding.EncodeToString(sha256Sum([]byte("hello")))
	if got := sha256Record([]byte("hello")); got != want {
		t.Errorf("sha256Record = %q, want %q", got, want)
	}
}

func sha256Sum(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}

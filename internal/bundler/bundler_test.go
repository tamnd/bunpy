package bundler_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/bunpy/v1/internal/bundler"
)

// --- helpers ---

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readZip(t *testing.T, pyzPath string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(pyzPath)
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("readZip: %v", err)
	}
	files := map[string]string{}
	for _, f := range r.File {
		rc, _ := f.Open()
		var buf bytes.Buffer
		buf.ReadFrom(rc)
		rc.Close()
		files[f.Name] = buf.String()
	}
	return files
}

// --- v0.6.0 tests ---

func TestScanImports(t *testing.T) {
	src := `import os
import utils
from helpers import foo
from . import rel
x = 1
`
	dir := t.TempDir()
	entry := writeFile(t, dir, "entry.py", src)
	writeFile(t, dir, "utils.py", "x = 1")
	writeFile(t, dir, "helpers.py", "foo = 2")

	b, err := bundler.Build(entry, bundler.Options{Outdir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Files["utils.py"]; !ok {
		t.Error("expected utils.py in bundle")
	}
	if _, ok := b.Files["helpers.py"]; !ok {
		t.Error("expected helpers.py in bundle")
	}
}

func TestBuildSingleFile(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	outdir := t.TempDir()

	b, err := bundler.Build(entry, bundler.Options{Outdir: outdir})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.WritePYZ(b.OutPath()); err != nil {
		t.Fatal(err)
	}

	files := readZip(t, b.OutPath())
	if _, ok := files["__main__.py"]; !ok {
		t.Error("missing __main__.py in pyz")
	}
}

func TestBuildMultiFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "helper.py", "VALUE = 42\n")
	entry := writeFile(t, dir, "app.py", "import helper\nx = helper.VALUE\n")
	outdir := t.TempDir()

	b, err := bundler.Build(entry, bundler.Options{Outdir: outdir})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.WritePYZ(b.OutPath()); err != nil {
		t.Fatal(err)
	}

	files := readZip(t, b.OutPath())
	if _, ok := files["__main__.py"]; !ok {
		t.Error("missing __main__.py")
	}
	if _, ok := files["helper.py"]; !ok {
		t.Error("missing helper.py")
	}
}

func TestPYZHasShebang(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	b, _ := bundler.Build(entry, bundler.Options{Outdir: t.TempDir()})
	b.WritePYZ(b.OutPath())

	data, _ := os.ReadFile(b.OutPath())
	if !bytes.HasPrefix(data, []byte("#!/usr/bin/env bunpy\n")) {
		n := len(data)
		if n > 40 {
			n = 40
		}
		t.Errorf("pyz missing shebang prefix, got: %q", string(data[:n]))
	}
}


// --- v0.6.1 minify tests ---

func TestMinifyComments(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "# comment\nx = 1\n")
	b, err := bundler.Build(entry, bundler.Options{Outdir: t.TempDir(), Minify: true})
	if err != nil {
		t.Fatal(err)
	}
	src := b.Files["__main__.py"]
	if strings.Contains(src, "# comment") {
		t.Error("comment not stripped")
	}
}

func TestMinifyBlankLines(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n\n\ny = 2\n")
	b, _ := bundler.Build(entry, bundler.Options{Outdir: t.TempDir(), Minify: true})
	src := b.Files["__main__.py"]
	if strings.Count(src, "\n\n") > 0 {
		t.Error("blank lines not stripped")
	}
}

func TestMinifyInlineComment(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1  # inline\n")
	b, _ := bundler.Build(entry, bundler.Options{Outdir: t.TempDir(), Minify: true})
	src := b.Files["__main__.py"]
	if strings.Contains(src, "# inline") {
		t.Error("inline comment not stripped")
	}
	if !strings.Contains(src, "x = 1") {
		t.Error("code was stripped along with comment")
	}
}

func TestMinifyStringPreserved(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", `x = "# not a comment"`+"\n")
	b, _ := bundler.Build(entry, bundler.Options{Outdir: t.TempDir(), Minify: true})
	src := b.Files["__main__.py"]
	if !strings.Contains(src, `"# not a comment"`) {
		t.Errorf("string literal modified: %q", src)
	}
}

// --- v0.6.2 target tests ---

func TestTargetMetadata(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	b, err := bundler.Build(entry, bundler.Options{Outdir: t.TempDir(), Target: "linux-x64"})
	if err != nil {
		t.Fatal(err)
	}
	b.WritePYZ(b.OutPath())
	files := readZip(t, b.OutPath())
	meta, ok := files["METADATA"]
	if !ok {
		t.Fatal("METADATA missing from zip")
	}
	if !strings.Contains(meta, "bunpy-target: linux-x64") {
		t.Errorf("unexpected METADATA: %q", meta)
	}
}

func TestTargetBrowser(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	b, _ := bundler.Build(entry, bundler.Options{Outdir: t.TempDir(), Target: "browser"})
	b.WritePYZ(b.OutPath())
	files := readZip(t, b.OutPath())
	if _, ok := files["WASM_NOTE"]; !ok {
		t.Error("WASM_NOTE missing for browser target")
	}
}

func TestTargetDefault(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	b, _ := bundler.Build(entry, bundler.Options{Outdir: t.TempDir()})
	b.WritePYZ(b.OutPath())
	files := readZip(t, b.OutPath())
	if _, ok := files["METADATA"]; ok {
		t.Error("METADATA should not be present with no target")
	}
}

func TestTargetUnknown(t *testing.T) {
	if err := bundler.ValidateTarget("dos-x16"); err == nil {
		t.Error("expected error for unknown target")
	}
}

// --- v0.6.4 sourcemap tests ---

func TestSourceMapCreated(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	outdir := t.TempDir()
	b, err := bundler.Build(entry, bundler.Options{Outdir: outdir, SourceMap: true})
	if err != nil {
		t.Fatal(err)
	}
	outpath := b.OutPath()
	b.WritePYZ(outpath)
	if err := bundler.WriteSourceMap(b.Sources, outpath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outpath + ".map"); err != nil {
		t.Errorf("sourcemap file not created: %v", err)
	}
}

func TestSourceMapContent(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	outdir := t.TempDir()
	b, _ := bundler.Build(entry, bundler.Options{Outdir: outdir})
	outpath := b.OutPath()
	b.WritePYZ(outpath)
	bundler.WriteSourceMap(b.Sources, outpath)

	data, err := os.ReadFile(outpath + ".map")
	if err != nil {
		t.Fatal(err)
	}
	var sm struct {
		Version int `json:"version"`
		Sources []struct {
			Bundled string `json:"bundled"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(data, &sm); err != nil {
		t.Fatalf("invalid sourcemap JSON: %v", err)
	}
	if sm.Version != 1 {
		t.Errorf("expected version 1, got %d", sm.Version)
	}
	if len(sm.Sources) == 0 {
		t.Error("sources array is empty")
	}
}

// --- v0.6.5 watch tests ---

func TestWatchRebuildsOnChange(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	outdir := t.TempDir()
	opts := bundler.Options{Outdir: outdir}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rebuilt := make(chan struct{}, 2)
	out := watchWriter{ch: rebuilt}

	go func() {
		bundler.Watch(ctx, entry, opts, &out)
	}()

	// Wait for initial build.
	select {
	case <-rebuilt:
	case <-ctx.Done():
		t.Fatal("timed out waiting for initial build")
	}

	// Modify the file.
	time.Sleep(250 * time.Millisecond)
	os.WriteFile(entry, []byte("x = 2\n"), 0o644)

	select {
	case <-rebuilt:
		// rebuild detected
	case <-ctx.Done():
		t.Fatal("timed out waiting for rebuild after file change")
	}
}

func TestWatchExitsOnCancel(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	opts := bundler.Options{Outdir: t.TempDir()}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- bundler.Watch(ctx, entry, opts, &watchWriter{})
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Watch returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Watch did not exit after context cancel")
	}
}

type watchWriter struct {
	ch chan struct{}
}

func (w *watchWriter) Write(p []byte) (int, error) {
	if w.ch != nil && strings.Contains(string(p), "built") {
		select {
		case w.ch <- struct{}{}:
		default:
		}
	}
	return len(p), nil
}

// --- v0.6.6 define tests ---

func TestDefineSimple(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = DEBUG\n")
	b, _ := bundler.Build(entry, bundler.Options{
		Outdir:  t.TempDir(),
		Defines: map[string]string{"DEBUG": "False"},
	})
	if !strings.Contains(b.Files["__main__.py"], "x = False") {
		t.Errorf("define not applied: %q", b.Files["__main__.py"])
	}
}

func TestDefineNoPartialMatch(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "DEBUGGER = 1\n")
	b, _ := bundler.Build(entry, bundler.Options{
		Outdir:  t.TempDir(),
		Defines: map[string]string{"DEBUG": "False"},
	})
	if strings.Contains(b.Files["__main__.py"], "False") {
		t.Errorf("partial match should not replace: %q", b.Files["__main__.py"])
	}
}

func TestDefineMultiple(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = A + B\n")
	b, _ := bundler.Build(entry, bundler.Options{
		Outdir:  t.TempDir(),
		Defines: map[string]string{"A": "1", "B": "2"},
	})
	src := b.Files["__main__.py"]
	if !strings.Contains(src, "x = 1 + 2") {
		t.Errorf("both defines not applied: %q", src)
	}
}

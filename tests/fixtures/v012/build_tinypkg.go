//go:build ignore

// build_tinypkg builds tests/fixtures/v012/tinypkg-0.1.0-py3-none-any.whl
// and the index/ copy used by the URL-fetch path. Run once and commit
// the result; do not run on every CI invocation because RECORD hashes
// must stay byte-stable.
//
// usage: go run tests/fixtures/v012/build_tinypkg.go
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

func main() {
	files := map[string][]byte{
		"tinypkg/__init__.py":                 []byte("def hello():\n    return \"hello from tinypkg\"\n"),
		"tinypkg-0.1.0.dist-info/WHEEL":       []byte("Wheel-Version: 1.0\nGenerator: bunpy/0.1.2\nRoot-Is-Purelib: true\nTag: py3-none-any\n"),
		"tinypkg-0.1.0.dist-info/METADATA":    []byte("Metadata-Version: 2.1\nName: tinypkg\nVersion: 0.1.0\nSummary: A tiny package for the bunpy v0.1.2 fixture.\nLicense: MIT\n"),
	}

	files["tinypkg-0.1.0.dist-info/RECORD"] = emitRECORD(files, "tinypkg-0.1.0.dist-info/RECORD")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	var paths []string
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		w, err := zw.Create(p)
		if err != nil {
			die(err)
		}
		if _, err := w.Write(files[p]); err != nil {
			die(err)
		}
	}
	if err := zw.Close(); err != nil {
		die(err)
	}

	root, err := os.Getwd()
	if err != nil {
		die(err)
	}
	target := filepath.Join(root, "tests", "fixtures", "v012", "tinypkg-0.1.0-py3-none-any.whl")
	if err := os.WriteFile(target, buf.Bytes(), 0o644); err != nil {
		die(err)
	}
	urlCopy := filepath.Join(root, "tests", "fixtures", "v012", "index", "files.example", "tinypkg", "tinypkg-0.1.0-py3-none-any.whl")
	if err := os.MkdirAll(filepath.Dir(urlCopy), 0o755); err != nil {
		die(err)
	}
	if err := os.WriteFile(urlCopy, buf.Bytes(), 0o644); err != nil {
		die(err)
	}
	// Stash the SHA-256 so the changelog and tests/run.sh can pin
	// against drift.
	sum := sha256.Sum256(buf.Bytes())
	fmt.Printf("wrote %s (%d bytes, sha256=%x)\n", target, len(buf.Bytes()), sum)
	fmt.Printf("wrote %s\n", urlCopy)
}

func sha256Record(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:])
}

func emitRECORD(files map[string][]byte, recordPath string) []byte {
	paths := make([]string, 0, len(files)+1)
	for p := range files {
		paths = append(paths, p)
	}
	paths = append(paths, recordPath)
	sort.Strings(paths)
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for _, p := range paths {
		if p == recordPath {
			_ = w.Write([]string{p, "", ""})
			continue
		}
		_ = w.Write([]string{p, sha256Record(files[p]), strconv.Itoa(len(files[p]))})
	}
	w.Flush()
	return buf.Bytes()
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

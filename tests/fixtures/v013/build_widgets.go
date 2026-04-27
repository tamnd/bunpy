//go:build ignore

// build_widgets builds the two widget wheels and the matching PEP 691
// simple index entry consumed by `bunpy add`. Run once and commit the
// result; do not regenerate on every CI run because RECORD hashes
// must stay byte-stable.
//
// usage: go run tests/fixtures/v013/build_widgets.go
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
	root, err := os.Getwd()
	if err != nil {
		die(err)
	}

	type build struct {
		version string
		init    string
	}
	builds := []build{
		{version: "1.0.0", init: "def hello():\n    return \"widget 1.0.0\"\n"},
		{version: "1.1.0", init: "def hello():\n    return \"widget 1.1.0\"\n"},
	}

	type fileEntry struct {
		Filename string `json:"filename"`
		URL      string `json:"url"`
		Hashes   map[string]string
		Sha256   string
	}
	var entries []fileEntry
	for _, b := range builds {
		body := buildWheel("widget", b.version, b.init)
		filename := fmt.Sprintf("widget-%s-py3-none-any.whl", b.version)

		commit := filepath.Join(root, "tests", "fixtures", "v013", filename)
		if err := os.WriteFile(commit, body, 0o644); err != nil {
			die(err)
		}
		mirror := filepath.Join(root, "tests", "fixtures", "v013", "index", "files.example", "widget", filename)
		if err := os.MkdirAll(filepath.Dir(mirror), 0o755); err != nil {
			die(err)
		}
		if err := os.WriteFile(mirror, body, 0o644); err != nil {
			die(err)
		}

		sum := sha256.Sum256(body)
		hex := fmt.Sprintf("%x", sum)
		entries = append(entries, fileEntry{
			Filename: filename,
			URL:      "https://files.example/widget/" + filename,
			Sha256:   hex,
		})
		fmt.Printf("wrote %s (%d bytes, sha256=%s)\n", commit, len(body), hex)
	}

	indexPath := filepath.Join(root, "tests", "fixtures", "v013", "index", "pypi.org", "simple", "widget", "index.json")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		die(err)
	}
	var idx bytes.Buffer
	idx.WriteString("{\n  \"name\": \"widget\",\n  \"files\": [\n")
	for i, e := range entries {
		fmt.Fprintf(&idx, "    {\n      \"filename\": \"%s\",\n      \"url\": \"%s\",\n      \"hashes\": {\"sha256\": \"%s\"}\n    }", e.Filename, e.URL, e.Sha256)
		if i+1 < len(entries) {
			idx.WriteString(",")
		}
		idx.WriteString("\n")
	}
	idx.WriteString("  ],\n  \"meta\": {\"api-version\": \"1.1\", \"_last-serial\": 1}\n}\n")
	if err := os.WriteFile(indexPath, idx.Bytes(), 0o644); err != nil {
		die(err)
	}
	fmt.Println("wrote", indexPath)
}

func buildWheel(name, version, initBody string) []byte {
	dist := fmt.Sprintf("%s-%s.dist-info", name, version)
	files := map[string][]byte{
		fmt.Sprintf("%s/__init__.py", name): []byte(initBody),
		dist + "/WHEEL":                     []byte("Wheel-Version: 1.0\nGenerator: bunpy/0.1.3\nRoot-Is-Purelib: true\nTag: py3-none-any\n"),
		dist + "/METADATA":                  []byte(fmt.Sprintf("Metadata-Version: 2.1\nName: %s\nVersion: %s\nSummary: A widget for the bunpy v0.1.3 fixture.\nLicense: MIT\n", name, version)),
	}
	files[dist+"/RECORD"] = emitRECORD(files, dist+"/RECORD")

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
	return buf.Bytes()
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

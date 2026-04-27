//go:build ignore

// build_widgets builds the v0.1.5 transitive fixture: widget-1.0.0
// depends on gizmo>=2.0; gizmo ships 2.0.0. Run once and commit;
// RECORD hashes must stay byte-stable across CI runs.
//
// usage: go run tests/fixtures/v015/build_widgets.go
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

type spec struct {
	name    string
	version string
	init    string
	requires []string
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		die(err)
	}

	specs := []spec{
		{
			name: "widget", version: "1.0.0",
			init:     "from gizmo import gear\n\ndef hello():\n    return f\"widget 1.0.0 + {gear()}\"\n",
			requires: []string{"gizmo>=2.0"},
		},
		{
			name: "gizmo", version: "2.0.0",
			init: "def gear():\n    return \"gizmo 2.0.0\"\n",
		},
	}

	type entry struct {
		Filename string
		URL      string
		Sha256   string
	}
	byProject := map[string][]entry{}

	for _, s := range specs {
		body := buildWheel(s)
		filename := fmt.Sprintf("%s-%s-py3-none-any.whl", s.name, s.version)
		commit := filepath.Join(root, "tests", "fixtures", "v015", filename)
		if err := os.WriteFile(commit, body, 0o644); err != nil {
			die(err)
		}
		mirror := filepath.Join(root, "tests", "fixtures", "v015", "index", "files.example", s.name, filename)
		if err := os.MkdirAll(filepath.Dir(mirror), 0o755); err != nil {
			die(err)
		}
		if err := os.WriteFile(mirror, body, 0o644); err != nil {
			die(err)
		}
		sum := sha256.Sum256(body)
		hex := fmt.Sprintf("%x", sum)
		byProject[s.name] = append(byProject[s.name], entry{
			Filename: filename,
			URL:      fmt.Sprintf("https://files.example/%s/%s", s.name, filename),
			Sha256:   hex,
		})
		fmt.Printf("wrote %s (%d bytes, sha256=%s)\n", commit, len(body), hex)
	}

	for name, files := range byProject {
		var idx bytes.Buffer
		fmt.Fprintf(&idx, "{\n  \"name\": \"%s\",\n  \"files\": [\n", name)
		for i, e := range files {
			fmt.Fprintf(&idx, "    {\n      \"filename\": \"%s\",\n      \"url\": \"%s\",\n      \"hashes\": {\"sha256\": \"%s\"}\n    }", e.Filename, e.URL, e.Sha256)
			if i+1 < len(files) {
				idx.WriteString(",")
			}
			idx.WriteString("\n")
		}
		idx.WriteString("  ],\n  \"meta\": {\"api-version\": \"1.1\", \"_last-serial\": 1}\n}\n")
		out := filepath.Join(root, "tests", "fixtures", "v015", "index", "pypi.org", "simple", name, "index.json")
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			die(err)
		}
		if err := os.WriteFile(out, idx.Bytes(), 0o644); err != nil {
			die(err)
		}
		fmt.Println("wrote", out)
	}
}

func buildWheel(s spec) []byte {
	dist := fmt.Sprintf("%s-%s.dist-info", s.name, s.version)
	metadata := fmt.Sprintf("Metadata-Version: 2.1\nName: %s\nVersion: %s\nSummary: bunpy v0.1.5 fixture.\nLicense: MIT\n",
		s.name, s.version)
	for _, r := range s.requires {
		metadata += "Requires-Dist: " + r + "\n"
	}
	files := map[string][]byte{
		fmt.Sprintf("%s/__init__.py", s.name): []byte(s.init),
		dist + "/WHEEL":                       []byte("Wheel-Version: 1.0\nGenerator: bunpy/0.1.5\nRoot-Is-Purelib: true\nTag: py3-none-any\n"),
		dist + "/METADATA":                    []byte(metadata),
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

//go:build ignore

// build_fixtures generates the benchmark fixture set:
//
//   - 47 minimal wheel files at fixtures/index/files.example/pkgNN/
//   - 47 PEP 691 simple index pages at fixtures/index/pypi.org/simple/pkgNN/
//   - fixtures/47pkg/pyproject.toml listing all 47 packages as direct deps
//   - 100 Python test files at fixtures/100tests/test_NNN.py
//
// Run once and commit the result. Do not regenerate on every CI run because
// RECORD hashes must stay byte-stable.
//
// usage: go run benchmarks/fixtures/build_fixtures.go
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
	"strconv"
	"strings"
)

const numPkgs = 47
const numTests = 100
const testFnsPerFile = 5

func main() {
	root, err := os.Getwd()
	if err != nil {
		die(err)
	}
	fixturesRoot := filepath.Join(root, "benchmarks", "fixtures")

	var indexEntries []indexEntry
	for n := 1; n <= numPkgs; n++ {
		name := fmt.Sprintf("pkg%02d", n)
		body := buildWheel(name, "1.0.0")
		filename := fmt.Sprintf("%s-1.0.0-py3-none-any.whl", name)

		mirrorDir := filepath.Join(fixturesRoot, "index", "files.example", name)
		must(os.MkdirAll(mirrorDir, 0o755))
		must(os.WriteFile(filepath.Join(mirrorDir, filename), body, 0o644))

		sum := sha256.Sum256(body)
		hex := fmt.Sprintf("%x", sum)
		indexEntries = append(indexEntries, indexEntry{
			name:     name,
			filename: filename,
			url:      fmt.Sprintf("https://files.example/%s/%s", name, filename),
			sha256:   hex,
		})
		fmt.Printf("wheel  %s (%d bytes, sha256=%s)\n", filename, len(body), hex)
	}

	for _, e := range indexEntries {
		writeIndex(fixturesRoot, e)
	}

	writeManifest(fixturesRoot, indexEntries)
	writeTestFiles(fixturesRoot)

	fmt.Println("done.")
}

type indexEntry struct {
	name, filename, url, sha256 string
}

func writeIndex(root string, e indexEntry) {
	dir := filepath.Join(root, "index", "pypi.org", "simple", e.name)
	must(os.MkdirAll(dir, 0o755))

	var b bytes.Buffer
	fmt.Fprintf(&b, "{\n  \"name\": %q,\n  \"files\": [\n", e.name)
	fmt.Fprintf(&b, "    {\n      \"filename\": %q,\n      \"url\": %q,\n      \"hashes\": {\"sha256\": %q}\n    }\n", e.filename, e.url, e.sha256)
	b.WriteString("  ],\n  \"meta\": {\"api-version\": \"1.1\", \"_last-serial\": 1}\n}\n")

	must(os.WriteFile(filepath.Join(dir, "index.json"), b.Bytes(), 0o644))
	fmt.Printf("index  %s/index.json\n", filepath.Join("pypi.org/simple", e.name))
}

func writeManifest(root string, entries []indexEntry) {
	var sb strings.Builder
	sb.WriteString("[project]\nname = \"bench-47pkg\"\nversion = \"0.0.1\"\ndependencies = [\n")
	for _, e := range entries {
		fmt.Fprintf(&sb, "    %q,\n", e.name+">=1.0")
	}
	sb.WriteString("]\n")

	dir := filepath.Join(root, "47pkg")
	must(os.MkdirAll(dir, 0o755))
	must(os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(sb.String()), 0o644))
	fmt.Printf("manifest  47pkg/pyproject.toml (%d packages)\n", len(entries))
}

func writeTestFiles(root string) {
	dir := filepath.Join(root, "100tests")
	must(os.MkdirAll(dir, 0o755))
	for i := 0; i < numTests; i++ {
		var sb strings.Builder
		for fn := 0; fn < testFnsPerFile; fn++ {
			base := i*testFnsPerFile + fn
			a := base + 1
			b := base + 2
			fmt.Fprintf(&sb, "x%d = %d\n", fn, a+b)
		}
		name := fmt.Sprintf("test_%03d.py", i)
		must(os.WriteFile(filepath.Join(dir, name), []byte(sb.String()), 0o644))
	}
	fmt.Printf("tests  100tests/ (%d files, %d statements each)\n", numTests, testFnsPerFile)
}

// buildWheel constructs a minimal .whl (zip) for a package named name at version ver.
func buildWheel(name, ver string) []byte {
	distInfo := fmt.Sprintf("%s-%s.dist-info", name, ver)

	initContent := fmt.Sprintf("# %s %s\nVERSION = %q\n", name, ver, ver)
	wheelContent := "Wheel-Version: 1.0\nGenerator: build_fixtures\nRoot-Is-Purelib: true\nTag: py3-none-any\n"
	metaContent := fmt.Sprintf("Metadata-Version: 2.1\nName: %s\nVersion: %s\nSummary: Benchmark fixture package.\n", name, ver)

	type entry struct {
		path    string
		content []byte
	}
	files := []entry{
		{name + "/__init__.py", []byte(initContent)},
		{distInfo + "/WHEEL", []byte(wheelContent)},
		{distInfo + "/METADATA", []byte(metaContent)},
	}

	// Build RECORD.
	var rec bytes.Buffer
	w := csv.NewWriter(&rec)
	for _, f := range files {
		sum := sha256.Sum256(f.content)
		h := base64.RawURLEncoding.EncodeToString(sum[:])
		if err := w.Write([]string{f.path, "sha256=" + h, strconv.Itoa(len(f.content))}); err != nil {
			die(err)
		}
	}
	if err := w.Write([]string{distInfo + "/RECORD", "", ""}); err != nil {
		die(err)
	}
	w.Flush()
	must(w.Error())

	files = append(files, entry{distInfo + "/RECORD", rec.Bytes()})

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		fw, err := zw.Create(f.path)
		if err != nil {
			die(err)
		}
		if _, err := fw.Write(f.content); err != nil {
			die(err)
		}
	}
	must(zw.Close())
	return buf.Bytes()
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func must(err error) {
	if err != nil {
		die(err)
	}
}


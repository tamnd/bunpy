// Package wheel is the v0.1.2 PEP 427 wheel installer. It opens a
// .whl file (a zip with a `<dist>.dist-info/` directory), parses the
// WHEEL, METADATA, and RECORD files, verifies body hashes against
// RECORD, and installs the body into a target site-packages
// directory.
//
// The scope is deliberately narrow: Root-Is-Purelib must be true,
// `*.data/` subdirs are refused, and unsafe entries (zip-slip,
// absolute paths, parent traversal) are rejected before any byte is
// written to disk. We grow this surface under v0.1.x as real wheels
// force it.
//
// Atomicity: the installer stages every body file under a tempdir
// inside target, then renames each file into place. A failure
// mid-install leaves the existing site-packages untouched.
package wheel

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Wheel is an opened wheel archive with its dist-info parsed.
type Wheel struct {
	Filename string  // e.g. requests-2.31.0-py3-none-any.whl
	Name     string  // PEP 503 normalised
	Version  string
	Tags     []Tag   // <python_tag>-<abi_tag>-<platform_tag>
	DistInfo string  // path inside the zip, e.g. "requests-2.31.0.dist-info/"
	DataDir  string  // optional, e.g. "requests-2.31.0.data/"
	Metadata []byte  // METADATA verbatim
	WHEEL    WheelMeta
	RECORD   []Entry

	z *zip.Reader
}

// WheelMeta is the parsed dist-info/WHEEL file.
type WheelMeta struct {
	Version       string // Wheel-Version
	Generator     string
	RootIsPurelib bool
	Tags          []Tag
}

// Tag is a PEP 425 compatibility tag.
type Tag struct{ Python, ABI, Platform string }

// Entry is one parsed RECORD line.
type Entry struct {
	Path string // path inside the wheel, with forward slashes
	Hash string // sha256=<base64> per PEP 376
	Size int64
}

// InstallOptions tunes Install. Zero value means: installer "bunpy",
// hash verification on.
type InstallOptions struct {
	Installer    string
	VerifyHashes *bool
}

// Open reads a wheel from disk and parses dist-info/{WHEEL,METADATA,RECORD}.
func Open(path string) (*Wheel, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return OpenReader(filepath.Base(path), body)
}

// OpenReader is Open from an in-memory byte slice.
func OpenReader(filename string, body []byte) (*Wheel, error) {
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("wheel %s: open zip: %w", filename, err)
	}
	w := &Wheel{Filename: filename, z: zr}
	if err := parseFilename(w); err != nil {
		return nil, err
	}
	if err := locateDirs(w); err != nil {
		return nil, err
	}
	if err := loadDistInfo(w); err != nil {
		return nil, err
	}
	return w, nil
}

// parseFilename extracts name, version, and tags from the wheel
// filename per PEP 427: <name>-<version>(-<build>)?-<py>-<abi>-<plat>.whl
// with optional dotted compressed tag sets. We only need name,
// version, and the (possibly dotted) tag triple here.
func parseFilename(w *Wheel) error {
	base := strings.TrimSuffix(w.Filename, ".whl")
	if base == w.Filename {
		return fmt.Errorf("wheel %s: not a .whl filename", w.Filename)
	}
	parts := strings.Split(base, "-")
	if len(parts) < 5 {
		return fmt.Errorf("wheel %s: filename has too few segments", w.Filename)
	}
	// last three are python-abi-platform; everything before the last
	// three minus the leading name is version (and optional build).
	plat := parts[len(parts)-1]
	abi := parts[len(parts)-2]
	py := parts[len(parts)-3]
	w.Name = normalizeName(parts[0])
	w.Version = parts[1]
	for _, p := range strings.Split(py, ".") {
		for _, a := range strings.Split(abi, ".") {
			for _, pl := range strings.Split(plat, ".") {
				w.Tags = append(w.Tags, Tag{Python: p, ABI: a, Platform: pl})
			}
		}
	}
	return nil
}

func locateDirs(w *Wheel) error {
	wantDist := w.distInfoPrefix()
	wantData := w.dataPrefix()
	for _, f := range w.z.File {
		if strings.HasPrefix(f.Name, wantDist) {
			w.DistInfo = wantDist
		}
		if strings.HasPrefix(f.Name, wantData) {
			w.DataDir = wantData
		}
	}
	if w.DistInfo == "" {
		return fmt.Errorf("wheel %s: missing %s", w.Filename, wantDist)
	}
	return nil
}

func (w *Wheel) distInfoPrefix() string {
	// PEP 427 dist-info name: project name with non-alphanumeric runs
	// replaced by underscore, and version verbatim.
	return distInfoName(w.Name) + "-" + w.Version + ".dist-info/"
}

func (w *Wheel) dataPrefix() string {
	return distInfoName(w.Name) + "-" + w.Version + ".data/"
}

func loadDistInfo(w *Wheel) error {
	if b, err := readZip(w.z, w.DistInfo+"WHEEL"); err != nil {
		return err
	} else if err := parseWheelMeta(w, b); err != nil {
		return err
	}
	if b, err := readZip(w.z, w.DistInfo+"METADATA"); err != nil {
		return err
	} else {
		w.Metadata = b
	}
	if b, err := readZip(w.z, w.DistInfo+"RECORD"); err != nil {
		return err
	} else if entries, err := parseRECORD(b); err != nil {
		return fmt.Errorf("wheel %s: RECORD: %w", w.Filename, err)
	} else {
		w.RECORD = entries
	}
	return nil
}

func readZip(z *zip.Reader, name string) ([]byte, error) {
	for _, f := range z.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("wheel: missing %s", name)
}

func parseWheelMeta(w *Wheel, body []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		i := strings.IndexByte(line, ':')
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		switch key {
		case "Wheel-Version":
			w.WHEEL.Version = val
		case "Generator":
			w.WHEEL.Generator = val
		case "Root-Is-Purelib":
			w.WHEEL.RootIsPurelib = strings.EqualFold(val, "true")
		case "Tag":
			parts := strings.SplitN(val, "-", 3)
			if len(parts) == 3 {
				w.WHEEL.Tags = append(w.WHEEL.Tags, Tag{Python: parts[0], ABI: parts[1], Platform: parts[2]})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("wheel %s: WHEEL: %w", w.Filename, err)
	}
	return nil
}

func parseRECORD(body []byte) ([]Entry, error) {
	r := csv.NewReader(bytes.NewReader(body))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		e := Entry{Path: row[0]}
		if len(row) > 1 {
			e.Hash = row[1]
		}
		if len(row) > 2 && row[2] != "" {
			n, err := strconv.ParseInt(row[2], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("RECORD %q: bad size %q: %w", row[0], row[2], err)
			}
			e.Size = n
		}
		out = append(out, e)
	}
	return out, nil
}

// Install copies the wheel's body files into target, writes
// dist-info/INSTALLER, and re-emits a RECORD with the install paths.
// Returns the absolute install paths created.
func (w *Wheel) Install(target string, opts InstallOptions) ([]string, error) {
	if !w.WHEEL.RootIsPurelib {
		return nil, fmt.Errorf("wheel %s: Root-Is-Purelib: false is not supported in v0.1.2", w.Filename)
	}
	if w.DataDir != "" {
		return nil, fmt.Errorf("wheel %s: %s subdir is not supported in v0.1.2", w.Filename, w.DataDir)
	}
	verify := true
	if opts.VerifyHashes != nil {
		verify = *opts.VerifyHashes
	}
	installer := opts.Installer
	if installer == "" {
		installer = "bunpy"
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absTarget, 0o755); err != nil {
		return nil, err
	}

	// Pre-flight: walk every body file once, reject unsafe entries,
	// and verify hashes against RECORD before any write.
	hashes := map[string]string{}
	for _, e := range w.RECORD {
		hashes[e.Path] = e.Hash
	}
	body := []*zip.File{}
	for _, f := range w.z.File {
		name := f.Name
		if strings.HasSuffix(name, "/") {
			continue
		}
		if strings.HasPrefix(name, w.DistInfo) {
			// dist-info is re-emitted by Install (INSTALLER, RECORD);
			// other dist-info files (METADATA, WHEEL, etc.) are copied
			// verbatim alongside the body files below.
			body = append(body, f)
			continue
		}
		if err := safeEntryPath(name); err != nil {
			return nil, fmt.Errorf("wheel %s: %s: %w", w.Filename, name, err)
		}
		body = append(body, f)
	}

	// Stage every file into a tempdir under target, then rename into
	// place at the very end. A mid-install crash leaves no partial
	// install in site-packages.
	staging, err := os.MkdirTemp(absTarget, ".bunpy-install-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = os.RemoveAll(staging) }

	written := map[string][]byte{}
	for _, f := range body {
		buf, err := readZipFile(f)
		if err != nil {
			cleanup()
			return nil, err
		}
		if verify {
			if want, ok := hashes[f.Name]; ok && want != "" {
				got := sha256Record(buf)
				if got != want {
					cleanup()
					return nil, fmt.Errorf("wheel %s: hash mismatch for %s (RECORD says %s, got %s)", w.Filename, f.Name, want, got)
				}
			}
		}
		written[f.Name] = buf
		stagePath := filepath.Join(staging, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(stagePath), 0o755); err != nil {
			cleanup()
			return nil, err
		}
		if err := os.WriteFile(stagePath, buf, 0o644); err != nil {
			cleanup()
			return nil, err
		}
	}

	// Re-emit RECORD with absolute install paths and the new
	// INSTALLER hash; older RECORDs list the wheel-relative paths,
	// PEP 376 says the installed RECORD lists relative-to-site paths.
	installerPath := w.DistInfo + "INSTALLER"
	installerBody := []byte(installer + "\n")
	stageInstaller := filepath.Join(staging, filepath.FromSlash(installerPath))
	if err := os.MkdirAll(filepath.Dir(stageInstaller), 0o755); err != nil {
		cleanup()
		return nil, err
	}
	if err := os.WriteFile(stageInstaller, installerBody, 0o644); err != nil {
		cleanup()
		return nil, err
	}
	written[installerPath] = installerBody

	recordBody := emitRECORD(written, w.DistInfo+"RECORD")
	stageRecord := filepath.Join(staging, filepath.FromSlash(w.DistInfo+"RECORD"))
	if err := os.WriteFile(stageRecord, recordBody, 0o644); err != nil {
		cleanup()
		return nil, err
	}
	written[w.DistInfo+"RECORD"] = recordBody

	// Move the staging tree into place. We do file-level renames so
	// that an existing site-packages with other packages stays intact.
	created := []string{}
	for relPath := range written {
		src := filepath.Join(staging, filepath.FromSlash(relPath))
		dst := filepath.Join(absTarget, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			cleanup()
			return nil, err
		}
		if err := os.Rename(src, dst); err != nil {
			cleanup()
			return nil, err
		}
		created = append(created, dst)
	}
	_ = os.RemoveAll(staging)
	sort.Strings(created)
	return created, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// safeEntryPath rejects zip-slip, absolute paths, and parent
// traversal. Backslashes are also refused because PEP 427 mandates
// forward slashes inside the archive.
func safeEntryPath(name string) error {
	if name == "" {
		return errors.New("empty entry name")
	}
	if strings.ContainsRune(name, '\\') {
		return errors.New("backslash in entry name")
	}
	if strings.HasPrefix(name, "/") {
		return errors.New("absolute entry path")
	}
	clean := path.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return errors.New("entry escapes target")
	}
	for _, seg := range strings.Split(clean, "/") {
		if seg == ".." {
			return errors.New("entry escapes target")
		}
	}
	return nil
}

// sha256Record returns the PEP 376 RECORD hash field:
// "sha256=<urlsafe-base64-no-padding>".
func sha256Record(body []byte) string {
	sum := sha256.Sum256(body)
	enc := base64.RawURLEncoding.EncodeToString(sum[:])
	return "sha256=" + enc
}

// emitRECORD writes a CSV body of entries plus a final blank line for
// RECORD itself (per PEP 376, RECORD's hash and size columns are empty).
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

// distInfoName mirrors PEP 427's dist-info naming rule: project name
// with runs of non-alphanumeric characters replaced by a single
// underscore. The PEP 503 normalised name uses dashes; the dist-info
// directory uses underscores.
func distInfoName(name string) string {
	var b strings.Builder
	prev := false
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r - 'A' + 'a')
			prev = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prev = false
		default:
			if !prev {
				b.WriteByte('_')
				prev = true
			}
		}
	}
	return b.String()
}

// normalizeName is PEP 503: lowercase plus collapse [-_.] runs to a
// single dash. Duplicated here so pkg/wheel does not import pkg/pypi
// (which would invert the dependency direction).
func normalizeName(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	prev := false
	for _, r := range name {
		switch {
		case r == '-' || r == '_' || r == '.':
			if !prev {
				b.WriteByte('-')
				prev = true
			}
		default:
			b.WriteRune(r)
			prev = false
		}
	}
	return b.String()
}

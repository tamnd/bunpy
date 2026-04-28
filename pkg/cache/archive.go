package cache

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// UVCacheDir returns uv's cache root. It runs `uv cache dir` when uv is in
// PATH; otherwise falls back to the platform default.
func UVCacheDir() string {
	if out, err := exec.Command("uv", "cache", "dir").Output(); err == nil {
		if dir := strings.TrimSpace(string(out)); dir != "" {
			return dir
		}
	}
	switch runtime.GOOS {
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "uv", "cache")
		}
	default:
		if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
			return filepath.Join(v, "uv")
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".cache", "uv")
		}
	}
	return ""
}

// ArchiveDir returns {cacheRoot}/archive-v0.
func ArchiveDir(cacheRoot string) string {
	return filepath.Join(cacheRoot, "archive-v0")
}

// PointerPath returns the path to the uv-format .http pointer file for a wheel.
// name is the normalised package name; filename is the .whl filename.
//
//	{cacheRoot}/wheels-v6/pypi/{name}/{ver}-{py}-{abi}-{plat}.http
func PointerPath(cacheRoot, name, filename string) string {
	base := strings.TrimSuffix(filename, ".whl")
	// strip the dist-name prefix: "requests-2.31.0-py3-none-any" → "2.31.0-py3-none-any"
	parts := strings.SplitN(base, "-", 2)
	var key string
	if len(parts) == 2 {
		key = parts[1]
	} else {
		key = base
	}
	return filepath.Join(cacheRoot, "wheels-v6", "pypi", normalize(name), key+".http")
}

// ReadPointer parses a uv .http pointer file and returns the archive key and
// SHA-256 hex. Returns ok=false if the file does not exist or cannot be parsed.
//
// The file is msgpack-encoded: fixarray([archive_key, [[algo,sha256hex]], ...]).
// We read only the first two elements.
func ReadPointer(path string) (archiveKey, sha256hex string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}
	return parseMsgpackPointer(data)
}

// WritePointer writes a uv-compatible .http pointer file.
func WritePointer(path, archiveKey, sha256hex, filename, url string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body := encodeMsgpackPointer(archiveKey, sha256hex, filename, url)
	return atomicWrite(path, body)
}

// ArchiveKey returns a deterministic archive key for a wheel given its SHA-256.
// Uses the first 15 bytes of SHA-256(sha256hex) encoded as URL-safe base64 (no
// padding), producing a 20-character key matching uv's key length.
func ArchiveKey(sha256hex string) string {
	h := sha256.Sum256([]byte(sha256hex))
	return base64.RawURLEncoding.EncodeToString(h[:15])
}

// HasArchive reports whether archive-v0/{key}/ exists and contains files.
func HasArchive(cacheRoot, archiveKey string) bool {
	dir := filepath.Join(ArchiveDir(cacheRoot), archiveKey)
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) > 0
}

// ExtractToArchive extracts the wheel zip body into archive-v0/{key}/.
// Idempotent: returns nil if the directory already has content.
func ExtractToArchive(cacheRoot, archiveKey string, whlBody []byte) error {
	archDir := filepath.Join(ArchiveDir(cacheRoot), archiveKey)
	if HasArchive(cacheRoot, archiveKey) {
		return nil
	}
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(whlBody), int64(len(whlBody)))
	if err != nil {
		return fmt.Errorf("archive: open wheel zip: %w", err)
	}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if err := safeArchivePath(f.Name); err != nil {
			return fmt.Errorf("archive: %s: %w", f.Name, err)
		}
		dst := filepath.Join(archDir, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// InstallFromArchive copies (using clone or hardlink) all files from
// archive-v0/{key}/ into targetDir, writing INSTALLER into the dist-info dir.
func InstallFromArchive(cacheRoot, archiveKey, targetDir, installer string) error {
	archDir := filepath.Join(ArchiveDir(cacheRoot), archiveKey)
	if installer == "" {
		installer = "bunpy"
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	// Walk the archive tree and clone/copy every file.
	// Also collect .dist-info dir paths so we can write INSTALLER after.
	var distInfoDirs []string
	err := filepath.WalkDir(archDir, func(srcPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(archDir, srcPath)
		dst := filepath.Join(targetDir, rel)
		if d.IsDir() {
			if strings.HasSuffix(filepath.ToSlash(rel), ".dist-info") {
				distInfoDirs = append(distInfoDirs, dst)
			}
			return os.MkdirAll(dst, 0o755)
		}
		// Skip INSTALLER from archive — we always write it fresh below.
		if strings.HasSuffix(filepath.ToSlash(rel), ".dist-info/INSTALLER") {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return cloneOrCopy(srcPath, dst)
	})
	if err != nil {
		return fmt.Errorf("install from archive: %w", err)
	}
	// Write INSTALLER into every dist-info directory.
	for _, dir := range distInfoDirs {
		_ = os.WriteFile(filepath.Join(dir, "INSTALLER"), []byte(installer+"\n"), 0o644)
	}
	return nil
}

// --- Minimal msgpack parser ---

func parseMsgpackPointer(data []byte) (archiveKey, sha256hex string, ok bool) {
	r := &msgpackReader{b: data}
	n, err := r.readArrayLen()
	if err != nil || n < 2 {
		return "", "", false
	}
	archiveKey, err = r.readString()
	if err != nil {
		return "", "", false
	}
	// Element 1: [[algo, sha256hex]]
	outer, err := r.readArrayLen()
	if err != nil || outer < 1 {
		return archiveKey, "", true // key found, no sha256
	}
	inner, err := r.readArrayLen()
	if err != nil || inner < 2 {
		return archiveKey, "", true
	}
	_, err = r.readString() // algo name, e.g. "Sha256"
	if err != nil {
		return archiveKey, "", true
	}
	sha256hex, err = r.readString()
	if err != nil {
		return archiveKey, "", true
	}
	return archiveKey, sha256hex, true
}

type msgpackReader struct {
	b   []byte
	pos int
}

func (r *msgpackReader) peek() (byte, error) {
	if r.pos >= len(r.b) {
		return 0, errors.New("msgpack: unexpected EOF")
	}
	return r.b[r.pos], nil
}

func (r *msgpackReader) readByte() (byte, error) {
	b, err := r.peek()
	if err != nil {
		return 0, err
	}
	r.pos++
	return b, nil
}

func (r *msgpackReader) readArrayLen() (int, error) {
	b, err := r.readByte()
	if err != nil {
		return 0, err
	}
	switch {
	case b >= 0x90 && b <= 0x9f: // fixarray
		return int(b & 0x0f), nil
	case b == 0xdc: // array16
		if r.pos+2 > len(r.b) {
			return 0, errors.New("msgpack: truncated array16")
		}
		n := int(binary.BigEndian.Uint16(r.b[r.pos:]))
		r.pos += 2
		return n, nil
	case b == 0xdd: // array32
		if r.pos+4 > len(r.b) {
			return 0, errors.New("msgpack: truncated array32")
		}
		n := int(binary.BigEndian.Uint32(r.b[r.pos:]))
		r.pos += 4
		return n, nil
	}
	return 0, fmt.Errorf("msgpack: expected array, got 0x%02x", b)
}

func (r *msgpackReader) readString() (string, error) {
	b, err := r.readByte()
	if err != nil {
		return "", err
	}
	var n int
	switch {
	case b >= 0xa0 && b <= 0xbf: // fixstr
		n = int(b & 0x1f)
	case b == 0xd9: // str8
		lb, err := r.readByte()
		if err != nil {
			return "", err
		}
		n = int(lb)
	case b == 0xda: // str16
		if r.pos+2 > len(r.b) {
			return "", errors.New("msgpack: truncated str16")
		}
		n = int(binary.BigEndian.Uint16(r.b[r.pos:]))
		r.pos += 2
	case b == 0xdb: // str32
		if r.pos+4 > len(r.b) {
			return "", errors.New("msgpack: truncated str32")
		}
		n = int(binary.BigEndian.Uint32(r.b[r.pos:]))
		r.pos += 4
	default:
		return "", fmt.Errorf("msgpack: expected string, got 0x%02x", b)
	}
	if r.pos+n > len(r.b) {
		return "", fmt.Errorf("msgpack: string length %d exceeds buffer", n)
	}
	s := string(r.b[r.pos : r.pos+n])
	r.pos += n
	return s, nil
}

// --- Minimal msgpack encoder ---

// encodeMsgpackPointer encodes the 4-element pointer array in uv's format.
func encodeMsgpackPointer(archiveKey, sha256hex, filename, url string) []byte {
	var b bytes.Buffer
	b.WriteByte(0x94) // fixarray(4)
	writeStr(&b, archiveKey)
	b.WriteByte(0x91) // fixarray(1)
	b.WriteByte(0x92) // fixarray(2)
	writeStr(&b, "Sha256")
	writeStr(&b, sha256hex)
	writeStr(&b, filename)
	writeStr(&b, url)
	return b.Bytes()
}

func writeStr(b *bytes.Buffer, s string) {
	n := len(s)
	switch {
	case n <= 31:
		b.WriteByte(byte(0xa0 | n))
	case n <= 255:
		b.WriteByte(0xd9)
		b.WriteByte(byte(n))
	case n <= 65535:
		b.WriteByte(0xda)
		b.WriteByte(byte(n >> 8))
		b.WriteByte(byte(n))
	default:
		b.WriteByte(0xdb)
		b.WriteByte(byte(n >> 24))
		b.WriteByte(byte(n >> 16))
		b.WriteByte(byte(n >> 8))
		b.WriteByte(byte(n))
	}
	b.WriteString(s)
}

func safeArchivePath(name string) error {
	if strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return errors.New("unsafe path")
	}
	for _, seg := range strings.Split(name, "/") {
		if seg == ".." {
			return errors.New("path traversal")
		}
	}
	return nil
}

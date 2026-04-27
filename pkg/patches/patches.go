// Package patches drives `bunpy patch <pkg>` and the patch-apply
// step inside `bunpy install`. v0.1.10 emits whole-file hunks (one
// hunk per changed file) into a unified-diff body. Apply is strict:
// pristine context must match the target byte-for-byte. The shape
// is the user-visible patch artefact under `./patches/`.
package patches

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/manifest"
)

// InstallerTag marks the dist-info INSTALLER for a patched install.
// `bunpy install` rewrites the wheel's INSTALLER once a registered
// patch lands successfully so a future inspection can tell the two
// apart.
const InstallerTag = "bunpy-patch"

// Entry is one row in [tool.bunpy.patches].
type Entry struct {
	Name    string // PEP 503 normalised
	Version string
	Path    string // patch file path, relative to project root
}

// Key returns the table key (e.g., "flask@2.3.0").
func (e Entry) Key() string { return e.Name + "@" + e.Version }

// Read returns every patch entry registered in the manifest. Keys
// are split on "@"; entries without a version are skipped (the
// table key contract is `<name>@<version>`).
func Read(m *manifest.Manifest) ([]Entry, error) {
	if m == nil || m.Tool.Raw == nil {
		return nil, nil
	}
	raw, ok := m.Tool.Raw["patches"].(map[string]any)
	if !ok {
		return nil, nil
	}
	var out []Entry
	for k, v := range raw {
		path, ok := v.(string)
		if !ok || path == "" {
			continue
		}
		name, version, found := strings.Cut(k, "@")
		if !found || name == "" || version == "" {
			continue
		}
		out = append(out, Entry{
			Name:    Normalize(name),
			Version: version,
			Path:    path,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out, nil
}

// Lookup finds the patch entry for (name, version). Name is matched
// PEP 503-normalised so callers can pass `Foo_Bar` and still get
// the same row.
func Lookup(entries []Entry, name, version string) (Entry, bool) {
	want := Normalize(name)
	for _, e := range entries {
		if e.Name == want && e.Version == version {
			return e, true
		}
	}
	return Entry{}, false
}

// Normalize is a PEP 503 normaliser local to the patches package so
// it does not pull pkg/pypi at link time.
func Normalize(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	prev := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == '_' || c == '.' {
			if prev == '-' {
				continue
			}
			sb.WriteByte('-')
			prev = '-'
			continue
		}
		sb.WriteByte(c)
		prev = c
	}
	return sb.String()
}

// Diff walks pristine and scratch and returns a unified-diff body
// using whole-file hunks for files that differ. Files identical
// in both trees are skipped. Binary files (any 0x00 byte in the
// first 4 KiB) are reported as an error: patching binaries is out
// of scope. Paths are sorted so output is byte-stable.
func Diff(pristine, scratch string) ([]byte, error) {
	paths := map[string]bool{}
	if err := walkInto(paths, pristine); err != nil {
		return nil, err
	}
	if err := walkInto(paths, scratch); err != nil {
		return nil, err
	}
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)

	var out bytes.Buffer
	for _, rel := range sorted {
		leftPath := filepath.Join(pristine, filepath.FromSlash(rel))
		rightPath := filepath.Join(scratch, filepath.FromSlash(rel))
		leftData, leftErr := os.ReadFile(leftPath)
		rightData, rightErr := os.ReadFile(rightPath)
		if leftErr != nil && !errors.Is(leftErr, fs.ErrNotExist) {
			return nil, leftErr
		}
		if rightErr != nil && !errors.Is(rightErr, fs.ErrNotExist) {
			return nil, rightErr
		}
		leftExists := leftErr == nil
		rightExists := rightErr == nil
		if leftExists && rightExists && bytes.Equal(leftData, rightData) {
			continue
		}
		if isBinary(leftData) || isBinary(rightData) {
			return nil, fmt.Errorf("patches: binary file %s differs; not supported", rel)
		}
		if leftExists && len(leftData) > 0 && !bytes.HasSuffix(leftData, []byte{'\n'}) {
			return nil, fmt.Errorf("patches: %s lacks a trailing newline; not supported", rel)
		}
		if rightExists && len(rightData) > 0 && !bytes.HasSuffix(rightData, []byte{'\n'}) {
			return nil, fmt.Errorf("patches: %s lacks a trailing newline; not supported", rel)
		}
		writeHunk(&out, rel, leftData, rightData, leftExists, rightExists)
	}
	return out.Bytes(), nil
}

// Apply mutates target in place using a unified-diff body. The
// diff must be the shape Diff emits: whole-file hunks, one hunk
// per file. Apply rejects any hunk whose '-' context does not
// match the target byte-for-byte. No fuzz, no offset slack.
func Apply(target string, body []byte) error {
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 256*1024), 16*1024*1024)
	var leftHeader, rightHeader string
	var leftLines, rightLines []string
	hadHunk := false
	flush := func() error {
		if leftHeader == "" && rightHeader == "" {
			return nil
		}
		rel := patchRel(leftHeader, rightHeader)
		if rel == "" {
			return errors.New("patches: malformed patch (no file path)")
		}
		full := filepath.Clean(filepath.Join(abs, filepath.FromSlash(rel)))
		if !strings.HasPrefix(full, abs+string(filepath.Separator)) && full != abs {
			return fmt.Errorf("patches: refused path escape: %s", rel)
		}
		switch {
		case leftHeader == "/dev/null":
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return err
			}
			return os.WriteFile(full, joinLines(rightLines), 0o644)
		case rightHeader == "/dev/null":
			actual, err := os.ReadFile(full)
			if err != nil {
				return fmt.Errorf("patches: cannot apply patch: %s: %w", rel, err)
			}
			if !bytes.Equal(actual, joinLines(leftLines)) {
				return fmt.Errorf("patches: cannot apply patch: %s does not match pristine", rel)
			}
			return os.Remove(full)
		default:
			actual, err := os.ReadFile(full)
			if err != nil {
				return fmt.Errorf("patches: cannot apply patch: %s: %w", rel, err)
			}
			if !bytes.Equal(actual, joinLines(leftLines)) {
				return fmt.Errorf("patches: cannot apply patch: %s does not match pristine", rel)
			}
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return err
			}
			return os.WriteFile(full, joinLines(rightLines), 0o644)
		}
	}
	inHunk := false
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "--- "):
			if hadHunk {
				if err := flush(); err != nil {
					return err
				}
				leftLines, rightLines = nil, nil
				hadHunk = false
			}
			leftHeader = stripHeader(strings.TrimPrefix(line, "--- "))
			inHunk = false
		case strings.HasPrefix(line, "+++ "):
			rightHeader = stripHeader(strings.TrimPrefix(line, "+++ "))
		case strings.HasPrefix(line, "@@"):
			inHunk = true
			hadHunk = true
		case !inHunk:
			// header noise
		case strings.HasPrefix(line, "-"):
			leftLines = append(leftLines, line[1:])
		case strings.HasPrefix(line, "+"):
			rightLines = append(rightLines, line[1:])
		case strings.HasPrefix(line, " "):
			// context (whole-file hunks have none, but accept)
			leftLines = append(leftLines, line[1:])
			rightLines = append(rightLines, line[1:])
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if hadHunk {
		return flush()
	}
	return nil
}

// patchRel picks the relative path from one of the headers,
// preferring the right-side path when both exist.
func patchRel(left, right string) string {
	if right != "" && right != "/dev/null" {
		return strings.TrimPrefix(right, "b/")
	}
	if left != "" && left != "/dev/null" {
		return strings.TrimPrefix(left, "a/")
	}
	return ""
}

// stripHeader trims a trailing tab + timestamp that some diff
// implementations emit (we never emit one, but be defensive).
func stripHeader(h string) string {
	before, _, _ := strings.Cut(h, "\t")
	return before
}

func joinLines(lines []string) []byte {
	if len(lines) == 0 {
		return nil
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func walkInto(set map[string]bool, root string) error {
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		set[filepath.ToSlash(rel)] = true
		return nil
	})
}

func isBinary(data []byte) bool {
	n := min(len(data), 4096)
	return bytes.IndexByte(data[:n], 0) >= 0
}

func writeHunk(out *bytes.Buffer, rel string, left, right []byte, leftExists, rightExists bool) {
	leftLines := splitDiffLines(left)
	rightLines := splitDiffLines(right)
	leftPath := "/dev/null"
	rightPath := "/dev/null"
	if leftExists {
		leftPath = "a/" + rel
	}
	if rightExists {
		rightPath = "b/" + rel
	}
	leftStart := 1
	rightStart := 1
	if !leftExists || len(leftLines) == 0 {
		leftStart = 0
	}
	if !rightExists || len(rightLines) == 0 {
		rightStart = 0
	}
	fmt.Fprintf(out, "--- %s\n", leftPath)
	fmt.Fprintf(out, "+++ %s\n", rightPath)
	fmt.Fprintf(out, "@@ -%d,%d +%d,%d @@\n", leftStart, len(leftLines), rightStart, len(rightLines))
	for _, l := range leftLines {
		fmt.Fprintf(out, "-%s\n", l)
	}
	for _, l := range rightLines {
		fmt.Fprintf(out, "+%s\n", l)
	}
}

// splitDiffLines splits raw bytes into diff lines. v0.1.10 requires
// every file to end with a trailing newline; Diff's caller errors
// out before reaching here if it does not.
func splitDiffLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	s := strings.TrimSuffix(string(data), "\n")
	return strings.Split(s, "\n")
}

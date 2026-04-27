package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// AddPatchEntry inserts or replaces a row in the
// `[tool.bunpy.patches]` table. The table is created (and the
// `[tool.bunpy]` parent created on demand) if absent. Returns the
// rewritten bytes plus the number of edits that landed (0 means
// the table already had the same key/value pair).
func (m *Manifest) AddPatchEntry(key, value string) ([]byte, int, error) {
	if m.Source == nil {
		return nil, 0, errors.New("manifest: AddPatchEntry requires source bytes; use Parse or Load")
	}
	if key == "" {
		return nil, 0, errors.New("manifest: AddPatchEntry requires a key")
	}
	src := m.Source
	tableStart, bodyStart, bodyEnd, ok := findPatchesTable(src)
	if !ok {
		return insertPatchesTable(src, key, value)
	}
	_ = tableStart
	return rewritePatchesTable(src, bodyStart, bodyEnd, key, value)
}

// RemovePatchEntry drops the row whose key matches. Missing keys
// are not an error: returns (src, 0, nil) so callers stay
// idempotent.
func (m *Manifest) RemovePatchEntry(key string) ([]byte, int, error) {
	if m.Source == nil {
		return nil, 0, errors.New("manifest: RemovePatchEntry requires source bytes; use Parse or Load")
	}
	if key == "" {
		return nil, 0, errors.New("manifest: RemovePatchEntry requires a key")
	}
	src := m.Source
	_, bodyStart, bodyEnd, ok := findPatchesTable(src)
	if !ok {
		return src, 0, nil
	}
	return removePatchEntry(src, bodyStart, bodyEnd, key)
}

// findPatchesTable returns (header offset, body start, body end,
// found). Body starts after the header line; body ends at the next
// section header or EOF.
func findPatchesTable(src []byte) (hdr, start, end int, ok bool) {
	text := string(src)
	hdr = -1
	i := 0
	for i < len(text) {
		nl := strings.IndexByte(text[i:], '\n')
		var lineEnd int
		var line string
		if nl < 0 {
			line = text[i:]
			lineEnd = len(text)
		} else {
			line = text[i : i+nl]
			lineEnd = i + nl + 1
		}
		trimmed := strings.TrimSpace(stripLineComment(line))
		if trimmed == "[tool.bunpy.patches]" {
			hdr = i
			start = lineEnd
			break
		}
		i = lineEnd
	}
	if hdr < 0 {
		return -1, -1, -1, false
	}
	j := start
	for j < len(text) {
		nl := strings.IndexByte(text[j:], '\n')
		var lineEnd int
		var line string
		if nl < 0 {
			line = text[j:]
			lineEnd = len(text)
		} else {
			line = text[j : j+nl]
			lineEnd = j + nl + 1
		}
		if isSectionHeader(line) {
			return hdr, start, j, true
		}
		if nl < 0 {
			break
		}
		j = lineEnd
	}
	return hdr, start, len(text), true
}

func rewritePatchesTable(src []byte, bodyStart, bodyEnd int, key, value string) ([]byte, int, error) {
	rows, err := parsePatchRows(src[bodyStart:bodyEnd])
	if err != nil {
		return nil, 0, err
	}
	prev, hadKey := rows[key]
	if hadKey && prev == value {
		return src, 0, nil
	}
	rows[key] = value
	rendered := renderPatchRows(rows)
	var out bytes.Buffer
	out.Write(src[:bodyStart])
	out.WriteString(rendered)
	out.Write(src[bodyEnd:])
	return out.Bytes(), 1, nil
}

func removePatchEntry(src []byte, bodyStart, bodyEnd int, key string) ([]byte, int, error) {
	rows, err := parsePatchRows(src[bodyStart:bodyEnd])
	if err != nil {
		return nil, 0, err
	}
	if _, ok := rows[key]; !ok {
		return src, 0, nil
	}
	delete(rows, key)
	rendered := renderPatchRows(rows)
	var out bytes.Buffer
	out.Write(src[:bodyStart])
	out.WriteString(rendered)
	out.Write(src[bodyEnd:])
	return out.Bytes(), 1, nil
}

// insertPatchesTable appends a `[tool.bunpy.patches]` table at the
// end of the file. Two newlines are inserted before the header to
// keep the diff readable when the source had no trailing newline.
func insertPatchesTable(src []byte, key, value string) ([]byte, int, error) {
	var out bytes.Buffer
	out.Write(src)
	if len(src) > 0 && src[len(src)-1] != '\n' {
		out.WriteByte('\n')
	}
	if len(src) > 0 {
		out.WriteByte('\n')
	}
	out.WriteString("[tool.bunpy.patches]\n")
	fmt.Fprintf(&out, "%s = %s\n", quoteBasic(key), quoteBasic(value))
	return out.Bytes(), 1, nil
}

// parsePatchRows reads the body of a `[tool.bunpy.patches]` table
// into a map. Comments and blank lines are dropped on rewrite.
func parsePatchRows(body []byte) (map[string]string, error) {
	rows := map[string]string{}
	for raw := range strings.SplitSeq(string(body), "\n") {
		line := strings.TrimSpace(stripLineComment(raw))
		if line == "" {
			continue
		}
		kPart, vPart, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("manifest: malformed patches row: %q", raw)
		}
		k := strings.TrimSpace(kPart)
		v := strings.TrimSpace(vPart)
		key, err := unquoteAny(k)
		if err != nil {
			return nil, err
		}
		val, err := unquoteAny(v)
		if err != nil {
			return nil, err
		}
		rows[key] = val
	}
	return rows, nil
}

func renderPatchRows(rows map[string]string) string {
	keys := make([]string, 0, len(rows))
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&sb, "%s = %s\n", quoteBasic(k), quoteBasic(rows[k]))
	}
	return sb.String()
}

func unquoteAny(s string) (string, error) {
	if len(s) < 2 {
		return "", fmt.Errorf("manifest: bare value not supported: %q", s)
	}
	switch s[0] {
	case '"':
		return unquote([]byte(s), '"')
	case '\'':
		return unquote([]byte(s), '\'')
	}
	return "", fmt.Errorf("manifest: bare value not supported: %q", s)
}

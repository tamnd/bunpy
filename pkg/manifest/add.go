package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// AddDependency inserts spec into [project].dependencies and
// returns the rewritten TOML text. If the same package is already
// listed, its line is replaced (upgrade). The array is created
// when absent. The strategy is dumb-and-correct: we keep the file
// as text, only touching the dependencies array region.
func (m *Manifest) AddDependency(spec string) ([]byte, error) {
	if m.Source == nil {
		return nil, errors.New("manifest: AddDependency requires source bytes; use Parse or Load")
	}
	spec = strings.TrimSpace(spec)
	name := splitDepName(spec)
	if name == "" {
		return nil, fmt.Errorf("manifest: invalid dependency spec %q", spec)
	}
	src := m.Source

	hdrStart, bodyStart, bodyEnd, err := findProjectSection(src)
	if err != nil {
		return nil, err
	}
	_ = hdrStart

	keyOff, arrStart, arrEnd, ok, err := findDependenciesArray(src, bodyStart, bodyEnd)
	if err != nil {
		return nil, err
	}
	if !ok {
		return insertDependencies(src, bodyStart, bodyEnd, spec)
	}
	_ = keyOff
	return rewriteDependencies(src, arrStart, arrEnd, name, spec)
}

// splitDepName returns the package name prefix of a PEP 508 spec.
// "widget>=1.0" -> "widget"; "" if the spec is empty or starts with
// a non-name rune.
func splitDepName(spec string) string {
	spec = strings.TrimSpace(spec)
	for i, r := range spec {
		if !isDepNameRune(r) {
			return strings.TrimSpace(spec[:i])
		}
	}
	return spec
}

func isDepNameRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == '-' || r == '_' || r == '.':
		return true
	}
	return false
}

// normalizeDepName is PEP 503: lower-case and collapse runs of -_.
// to a single -.
func normalizeDepName(s string) string {
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

var sectionHeaderRE = regexp.MustCompile(`^\s*\[\[?[^\]]+\]\]?\s*(#.*)?$`)

// findProjectSection returns (header line offset, body start, body end).
// Body starts after the header line; body ends at the next section
// header or EOF.
func findProjectSection(src []byte) (hdr, start, end int, err error) {
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
		if trimmed == "[project]" {
			hdr = i
			start = lineEnd
			break
		}
		i = lineEnd
	}
	if hdr < 0 {
		return -1, -1, -1, errors.New("manifest: [project] table missing")
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
			return hdr, start, j, nil
		}
		if nl < 0 {
			break
		}
		j = lineEnd
	}
	return hdr, start, len(text), nil
}

func isSectionHeader(line string) bool {
	return sectionHeaderRE.MatchString(strings.TrimRight(line, "\r\n"))
}

func stripLineComment(line string) string {
	inStr := false
	var strCh byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		if inStr {
			if c == '\\' && strCh == '"' {
				i++
				continue
			}
			if c == strCh {
				inStr = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			inStr = true
			strCh = c
			continue
		}
		if c == '#' {
			return line[:i]
		}
	}
	return line
}

// findDependenciesArray locates `dependencies = [ ... ]` within the
// [project] body. Returns (key offset, '[' offset, ']' offset, true)
// on success. Returns (..., false, nil) when the key is absent;
// returns an error when present but not in array form we can rewrite.
func findDependenciesArray(src []byte, start, end int) (keyOff, arrStart, arrEnd int, ok bool, err error) {
	i := start
	for i < end {
		nl := bytes.IndexByte(src[i:end], '\n')
		var lineEnd int
		var line []byte
		if nl < 0 {
			line = src[i:end]
			lineEnd = end
		} else {
			line = src[i : i+nl]
			lineEnd = i + nl + 1
		}
		trimmed := strings.TrimLeft(string(line), " \t")
		if strings.HasPrefix(trimmed, "dependencies") {
			rest := trimmed[len("dependencies"):]
			rest = strings.TrimLeft(rest, " \t")
			if strings.HasPrefix(rest, "=") {
				bracket := bytes.IndexByte(line, '[')
				if bracket < 0 {
					return 0, 0, 0, false, errors.New("manifest: dependencies value is not an array")
				}
				as := i + bracket
				ae, found := matchClose(src, as, len(src))
				if !found {
					return 0, 0, 0, false, errors.New("manifest: unterminated dependencies array")
				}
				return i, as, ae, true, nil
			}
		}
		if nl < 0 {
			break
		}
		i = lineEnd
	}
	return 0, 0, 0, false, nil
}

// matchClose finds the byte offset of the ']' matching the '[' at
// start. It tracks string state and skips '#' comments.
func matchClose(src []byte, start, end int) (int, bool) {
	if start >= end || src[start] != '[' {
		return -1, false
	}
	depth := 0
	i := start
	inStr := false
	var strCh byte
	for i < end {
		c := src[i]
		if inStr {
			if c == '\\' && strCh == '"' && i+1 < end {
				i += 2
				continue
			}
			if c == strCh {
				inStr = false
			}
			i++
			continue
		}
		switch c {
		case '"', '\'':
			inStr = true
			strCh = c
			i++
		case '#':
			nl := bytes.IndexByte(src[i:end], '\n')
			if nl < 0 {
				return -1, false
			}
			i += nl + 1
		case '[':
			depth++
			i++
		case ']':
			depth--
			if depth == 0 {
				return i, true
			}
			i++
		default:
			i++
		}
	}
	return -1, false
}

type depItem struct {
	name string
	text string
}

// rewriteDependencies parses the array body, inserts or replaces
// the entry for spec, and emits the array back. Comments inside the
// array body are dropped on rewrite (best-effort).
func rewriteDependencies(src []byte, arrStart, arrEnd int, name, spec string) ([]byte, error) {
	body := src[arrStart+1 : arrEnd]
	items, err := parseArrayItems(body)
	if err != nil {
		return nil, err
	}
	wantNorm := normalizeDepName(name)
	replaced := false
	for i := range items {
		if normalizeDepName(items[i].name) == wantNorm {
			items[i] = depItem{name: name, text: spec}
			replaced = true
			break
		}
	}
	if !replaced {
		items = append(items, depItem{name: name, text: spec})
	}
	sort.SliceStable(items, func(a, b int) bool {
		return normalizeDepName(items[a].name) < normalizeDepName(items[b].name)
	})

	multiline := bytes.Contains(body, []byte{'\n'})
	indent := "    "
	closeIndent := ""
	if multiline {
		indent, closeIndent = detectIndents(src, arrStart, arrEnd)
	}

	var sb strings.Builder
	if multiline {
		sb.WriteString("\n")
		for _, it := range items {
			sb.WriteString(indent)
			sb.WriteString(quoteBasic(it.text))
			sb.WriteString(",\n")
		}
		sb.WriteString(closeIndent)
	} else {
		for i, it := range items {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(quoteBasic(it.text))
		}
	}

	var out bytes.Buffer
	out.Write(src[:arrStart+1])
	out.WriteString(sb.String())
	out.Write(src[arrEnd:])
	return out.Bytes(), nil
}

// detectIndents returns (item indent, closing-bracket indent) by
// reading the first item line and the line containing the closing
// bracket. Falls back to "    " and "" when detection fails.
func detectIndents(src []byte, arrStart, arrEnd int) (string, string) {
	itemIndent := "    "
	closeIndent := ""
	body := src[arrStart+1 : arrEnd]
	if nl := bytes.IndexByte(body, '\n'); nl >= 0 {
		rest := body[nl+1:]
		w := 0
		for w < len(rest) && (rest[w] == ' ' || rest[w] == '\t') {
			w++
		}
		if w > 0 && w < len(rest) && rest[w] != '\n' {
			itemIndent = string(rest[:w])
		}
	}
	ls := bytes.LastIndexByte(src[:arrEnd], '\n')
	if ls >= 0 {
		k := ls + 1
		for k < arrEnd && (src[k] == ' ' || src[k] == '\t') {
			k++
		}
		closeIndent = string(src[ls+1 : k])
	}
	return itemIndent, closeIndent
}

func parseArrayItems(body []byte) ([]depItem, error) {
	var items []depItem
	i := 0
	for i < len(body) {
		c := body[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == ',':
			i++
		case c == '#':
			nl := bytes.IndexByte(body[i:], '\n')
			if nl < 0 {
				return items, nil
			}
			i += nl + 1
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(body) {
				if body[j] == '\\' && c == '"' && j+1 < len(body) {
					j += 2
					continue
				}
				if body[j] == c {
					break
				}
				j++
			}
			if j >= len(body) {
				return nil, errors.New("manifest: unterminated string in dependencies")
			}
			raw, err := unquote(body[i:j+1], c)
			if err != nil {
				return nil, err
			}
			items = append(items, depItem{name: splitDepName(raw), text: raw})
			i = j + 1
		default:
			return nil, fmt.Errorf("manifest: unexpected %q in dependencies", c)
		}
	}
	return items, nil
}

func unquote(b []byte, ch byte) (string, error) {
	if len(b) < 2 || b[0] != ch || b[len(b)-1] != ch {
		return "", errors.New("manifest: malformed string")
	}
	inner := b[1 : len(b)-1]
	if ch == '\'' {
		return string(inner), nil
	}
	var sb strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) {
			switch inner[i+1] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			default:
				sb.WriteByte(inner[i+1])
			}
			i++
			continue
		}
		sb.WriteByte(inner[i])
	}
	return sb.String(), nil
}

func quoteBasic(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteByte(s[i])
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// insertDependencies adds a new `dependencies = [...]` block to a
// [project] section that lacks one. The block lands after the last
// non-blank line of the section so existing keys come first.
func insertDependencies(src []byte, start, end int, spec string) ([]byte, error) {
	insertAt := start
	i := start
	for i < end {
		nl := bytes.IndexByte(src[i:end], '\n')
		var lineEnd int
		if nl < 0 {
			lineEnd = end
		} else {
			lineEnd = i + nl + 1
		}
		if strings.TrimSpace(string(src[i:lineEnd])) != "" {
			insertAt = lineEnd
		}
		if nl < 0 {
			break
		}
		i = lineEnd
	}

	var prefix string
	if insertAt > 0 && src[insertAt-1] != '\n' {
		prefix = "\n"
	}
	insertion := prefix + fmt.Sprintf("dependencies = [\n    %s,\n]\n", quoteBasic(spec))

	var out bytes.Buffer
	out.Write(src[:insertAt])
	out.WriteString(insertion)
	out.Write(src[insertAt:])
	return out.Bytes(), nil
}

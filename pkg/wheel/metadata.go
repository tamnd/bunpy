package wheel

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// Metadata captures the subset of dist-info/METADATA bunpy reads:
// the package identity and the Requires-Dist edges. Other RFC 822
// fields are surfaced as Raw so future code can pick them up
// without re-parsing.
type Metadata struct {
	Name         string
	Version      string
	RequiresDist []RequiresDist
	Raw          map[string][]string
}

// RequiresDist is one parsed Requires-Dist line: name (PEP 503
// pre-normalisation), the optional bracketed extras, the version
// specifier (raw text — caller passes to pkg/version), and the
// marker source (raw text — caller passes to pkg/marker).
type RequiresDist struct {
	Name   string
	Extras []string
	Spec   string
	Marker string
	Raw    string
}

// ParseMetadata reads METADATA bytes (RFC 822-style) and returns the
// fields bunpy cares about. Multi-line continuations (lines starting
// with whitespace) are folded into the prior field.
func ParseMetadata(body []byte) (*Metadata, error) {
	m := &Metadata{Raw: map[string][]string{}}
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lastKey string
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			break // body separator: rest is the long description
		}
		if line[0] == ' ' || line[0] == '\t' {
			if lastKey != "" {
				vals := m.Raw[lastKey]
				if len(vals) > 0 {
					vals[len(vals)-1] += "\n" + strings.TrimSpace(line)
					m.Raw[lastKey] = vals
				}
			}
			continue
		}
		i := strings.IndexByte(line, ':')
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		lastKey = key
		m.Raw[key] = append(m.Raw[key], val)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("wheel: scan METADATA: %w", err)
	}
	if v := first(m.Raw["Name"]); v != "" {
		m.Name = v
	}
	if v := first(m.Raw["Version"]); v != "" {
		m.Version = v
	}
	for _, raw := range m.Raw["Requires-Dist"] {
		rd, err := ParseRequiresDist(raw)
		if err != nil {
			return nil, fmt.Errorf("wheel: Requires-Dist %q: %w", raw, err)
		}
		m.RequiresDist = append(m.RequiresDist, rd)
	}
	return m, nil
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

// ParseRequiresDist splits one Requires-Dist value into its parts.
// Grammar (PEP 508-ish, narrowed): name [extras] [spec] [; marker].
// Whitespace inside the spec is preserved so callers can pass it
// straight to version.ParseSpec.
func ParseRequiresDist(s string) (RequiresDist, error) {
	out := RequiresDist{Raw: s}
	body, marker := splitMarker(s)
	out.Marker = strings.TrimSpace(marker)
	body = strings.TrimSpace(body)

	end := 0
	for end < len(body) && isReqNameRune(body[end]) {
		end++
	}
	if end == 0 {
		return out, fmt.Errorf("missing name")
	}
	out.Name = body[:end]
	rest := strings.TrimSpace(body[end:])

	if strings.HasPrefix(rest, "[") {
		closeIdx := strings.IndexByte(rest, ']')
		if closeIdx < 0 {
			return out, fmt.Errorf("unterminated extras")
		}
		extras := rest[1:closeIdx]
		for _, e := range strings.Split(extras, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				out.Extras = append(out.Extras, e)
			}
		}
		rest = strings.TrimSpace(rest[closeIdx+1:])
	}
	if strings.HasPrefix(rest, "(") && strings.HasSuffix(rest, ")") {
		rest = strings.TrimSpace(rest[1 : len(rest)-1])
	}
	out.Spec = rest
	return out, nil
}

func splitMarker(s string) (string, string) {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ';':
			if depth == 0 {
				return s[:i], s[i+1:]
			}
		}
	}
	return s, ""
}

func isReqNameRune(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '-' || b == '_' || b == '.':
		return true
	}
	return false
}

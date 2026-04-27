package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// AddOptionalDependency writes spec into
// [project.optional-dependencies].<group>. Creates the table and
// the group array when absent. Behaves like AddDependency on the
// rewrite-or-insert axis.
func (m *Manifest) AddOptionalDependency(group, spec string) ([]byte, error) {
	if err := validGroupName(group); err != nil {
		return nil, err
	}
	return m.addToTableArray("project.optional-dependencies", group, spec)
}

// AddGroupDependency writes spec into [dependency-groups].<group>.
// PEP 735's modern home for dev/test/etc lanes.
func (m *Manifest) AddGroupDependency(group, spec string) ([]byte, error) {
	if err := validGroupName(group); err != nil {
		return nil, err
	}
	return m.addToTableArray("dependency-groups", group, spec)
}

// AddPeerDependency writes spec into [tool.bunpy] peer-dependencies.
func (m *Manifest) AddPeerDependency(spec string) ([]byte, error) {
	return m.addToTableArray("tool.bunpy", "peer-dependencies", spec)
}

func validGroupName(g string) error {
	if g == "" {
		return errors.New("manifest: group name is empty")
	}
	if !groupNameRE.MatchString(g) {
		return fmt.Errorf("manifest: group name %q is not a valid PEP 685 name", g)
	}
	return nil
}

// addToTableArray rewrites src so that key = ["...", spec] appears
// under [header]. Three cases:
//
//  1. [header] section absent: append a new section at EOF.
//  2. [header] present, key absent: insert key = ["spec"].
//  3. Both present: parse the array, replace-or-add by PEP 503
//     normalised name, re-emit.
func (m *Manifest) addToTableArray(header, key, spec string) ([]byte, error) {
	if m.Source == nil {
		return nil, errors.New("manifest: AddDependency requires source bytes; use Parse or Load")
	}
	spec = strings.TrimSpace(spec)
	name := splitDepName(spec)
	if name == "" {
		return nil, fmt.Errorf("manifest: invalid dependency spec %q", spec)
	}

	src := m.Source
	hdr, start, end, ok := findNamedSection(src, header)
	if !ok {
		return appendNewSection(src, header, key, spec), nil
	}
	_ = hdr

	keyOff, arrStart, arrEnd, found, err := findArrayKey(src, start, end, key)
	if err != nil {
		return nil, err
	}
	if !found {
		return insertArrayKey(src, start, end, key, spec)
	}
	_ = keyOff
	return rewriteDependencies(src, arrStart, arrEnd, name, spec)
}

// findNamedSection returns the byte offsets for the section
// [header] within src: (header line offset, body start, body end,
// found). Body ends at the next section header or EOF.
func findNamedSection(src []byte, header string) (hdr, start, end int, ok bool) {
	target := "[" + header + "]"
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
		if trimmed == target {
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

// findArrayKey locates `<key> = [...]` within the section body.
func findArrayKey(src []byte, start, end int, key string) (keyOff, arrStart, arrEnd int, ok bool, err error) {
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
		if strings.HasPrefix(trimmed, key) {
			rest := trimmed[len(key):]
			rest = strings.TrimLeft(rest, " \t")
			if strings.HasPrefix(rest, "=") {
				bracket := bytes.IndexByte(line, '[')
				if bracket < 0 {
					return 0, 0, 0, false, fmt.Errorf("manifest: %s value is not an array", key)
				}
				as := i + bracket
				ae, found := matchClose(src, as, len(src))
				if !found {
					return 0, 0, 0, false, fmt.Errorf("manifest: unterminated %s array", key)
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

// insertArrayKey adds `key = ["spec"]` after the last non-blank
// line of the section body.
func insertArrayKey(src []byte, start, end int, key, spec string) ([]byte, error) {
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
	insertion := prefix + fmt.Sprintf("%s = [\n    %s,\n]\n", key, quoteBasic(spec))

	var out bytes.Buffer
	out.Write(src[:insertAt])
	out.WriteString(insertion)
	out.Write(src[insertAt:])
	return out.Bytes(), nil
}

// appendNewSection appends a `[header]\nkey = ["spec"]` block to
// EOF. Inserts a leading newline when the source does not already
// end with one.
func appendNewSection(src []byte, header, key, spec string) []byte {
	var sb strings.Builder
	sb.Write(src)
	if len(src) > 0 && src[len(src)-1] != '\n' {
		sb.WriteByte('\n')
	}
	sb.WriteByte('\n')
	fmt.Fprintf(&sb, "[%s]\n", header)
	fmt.Fprintf(&sb, "%s = [\n    %s,\n]\n", key, quoteBasic(spec))
	return []byte(sb.String())
}

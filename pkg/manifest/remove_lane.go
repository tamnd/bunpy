package manifest

import (
	"bytes"
	"errors"
	"sort"
	"strings"
)

// RemoveDependency drops the entry for name from
// [project].dependencies. Matches by PEP 503 normalised name; the
// existing spec text is irrelevant. Returns the rewritten source
// and the count of items removed (0 when nothing matches, which is
// not an error: remove is idempotent).
func (m *Manifest) RemoveDependency(name string) ([]byte, int, error) {
	if m.Source == nil {
		return nil, 0, errors.New("manifest: RemoveDependency requires source bytes; use Parse or Load")
	}
	src := m.Source
	_, bodyStart, bodyEnd, err := findProjectSection(src)
	if err != nil {
		return nil, 0, err
	}
	_, arrStart, arrEnd, ok, err := findDependenciesArray(src, bodyStart, bodyEnd)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return src, 0, nil
	}
	return removeFromArray(src, arrStart, arrEnd, name)
}

// RemoveOptionalDependency drops name from
// [project.optional-dependencies].<group>.
func (m *Manifest) RemoveOptionalDependency(group, name string) ([]byte, int, error) {
	if err := validGroupName(group); err != nil {
		return nil, 0, err
	}
	return m.removeFromTableArray("project.optional-dependencies", group, name)
}

// RemoveGroupDependency drops name from [dependency-groups].<group>.
func (m *Manifest) RemoveGroupDependency(group, name string) ([]byte, int, error) {
	if err := validGroupName(group); err != nil {
		return nil, 0, err
	}
	return m.removeFromTableArray("dependency-groups", group, name)
}

// RemovePeerDependency drops name from
// [tool.bunpy].peer-dependencies.
func (m *Manifest) RemovePeerDependency(name string) ([]byte, int, error) {
	return m.removeFromTableArray("tool.bunpy", "peer-dependencies", name)
}

// RemoveDependencyAllLanes deletes name from every lane it appears
// in: main, every PEP 735 group, every PEP 621 optional group, and
// peer. The Manifest must have been re-parsed after each successful
// edit so subsequent passes see the new source. Returns the total
// removal count across all lanes.
func (m *Manifest) RemoveDependencyAllLanes(name string) ([]byte, int, error) {
	src := m.Source
	if src == nil {
		return nil, 0, errors.New("manifest: RemoveDependencyAllLanes requires source bytes; use Parse or Load")
	}
	total := 0

	// Snapshot the lane map up front: deletes can move byte offsets,
	// but the set of group names is stable across the pass.
	groupNames := make([]string, 0, len(m.DependencyGroups))
	for g := range m.DependencyGroups {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)
	optionalNames := make([]string, 0, len(m.Project.OptionalDeps))
	for g := range m.Project.OptionalDeps {
		optionalNames = append(optionalNames, g)
	}
	sort.Strings(optionalNames)

	steps := []func(*Manifest) ([]byte, int, error){
		func(mm *Manifest) ([]byte, int, error) { return mm.RemoveDependency(name) },
		func(mm *Manifest) ([]byte, int, error) { return mm.RemovePeerDependency(name) },
	}
	for _, g := range groupNames {
		steps = append(steps, func(mm *Manifest) ([]byte, int, error) {
			return mm.RemoveGroupDependency(g, name)
		})
	}
	for _, g := range optionalNames {
		steps = append(steps, func(mm *Manifest) ([]byte, int, error) {
			return mm.RemoveOptionalDependency(g, name)
		})
	}

	cur := m
	for _, step := range steps {
		out, n, err := step(cur)
		if err != nil {
			return nil, total, err
		}
		total += n
		if n == 0 || bytes.Equal(out, cur.Source) {
			continue
		}
		next, err := Parse(out)
		if err != nil {
			return nil, total, err
		}
		cur = next
	}
	return cur.Source, total, nil
}

// removeFromTableArray drops one item by PEP 503 normalised name
// from `key = [...]` under [header]. Missing section, missing key,
// or no matching item is a no-op (returns the original source and
// count 0).
func (m *Manifest) removeFromTableArray(header, key, name string) ([]byte, int, error) {
	if m.Source == nil {
		return nil, 0, errors.New("manifest: remove requires source bytes; use Parse or Load")
	}
	src := m.Source
	_, start, end, ok := findNamedSection(src, header)
	if !ok {
		return src, 0, nil
	}
	_, arrStart, arrEnd, found, err := findArrayKey(src, start, end, key)
	if err != nil {
		return nil, 0, err
	}
	if !found {
		return src, 0, nil
	}
	return removeFromArray(src, arrStart, arrEnd, name)
}

// removeFromArray parses the items between arrStart and arrEnd,
// drops every entry whose PEP 503 normalised name matches name, and
// re-emits the array preserving multiline/single-line shape.
func removeFromArray(src []byte, arrStart, arrEnd int, name string) ([]byte, int, error) {
	body := src[arrStart+1 : arrEnd]
	items, err := parseArrayItems(body)
	if err != nil {
		return nil, 0, err
	}
	wantNorm := normalizeDepName(name)
	kept := items[:0]
	removed := 0
	for _, it := range items {
		if normalizeDepName(it.name) == wantNorm {
			removed++
			continue
		}
		kept = append(kept, it)
	}
	if removed == 0 {
		return src, 0, nil
	}

	multiline := bytes.Contains(body, []byte{'\n'})
	indent := "    "
	closeIndent := ""
	if multiline {
		indent, closeIndent = detectIndents(src, arrStart, arrEnd)
	}

	var sb strings.Builder
	switch {
	case multiline && len(kept) == 0:
		// Keep the array empty but multiline: `dependencies = [\n]`
		// is still valid TOML and avoids a noisy diff that collapses
		// the formatting on the last delete.
		sb.WriteString("\n")
		sb.WriteString(closeIndent)
	case multiline:
		sb.WriteString("\n")
		for _, it := range kept {
			sb.WriteString(indent)
			sb.WriteString(quoteBasic(it.text))
			sb.WriteString(",\n")
		}
		sb.WriteString(closeIndent)
	default:
		for i, it := range kept {
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
	return out.Bytes(), removed, nil
}


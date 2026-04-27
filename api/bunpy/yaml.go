package bunpy

import (
	"fmt"
	"strconv"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildYAML(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.yaml", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("parse", &goipyObject.BuiltinFunc{
		Name: "parse",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("yaml.parse() requires a YAML string")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("yaml.parse(): argument must be str")
			}
			val, err := parseYAML(s.V)
			if err != nil {
				return nil, err
			}
			return val, nil
		},
	})

	mod.Dict.SetStr("stringify", &goipyObject.BuiltinFunc{
		Name: "stringify",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("yaml.stringify() requires a value")
			}
			var sb strings.Builder
			stringifyYAML(&sb, args[0], 0)
			return &goipyObject.Str{V: sb.String()}, nil
		},
	})

	return mod
}

// parseYAML is a minimal YAML parser supporting scalars, mappings,
// and sequences. It handles indented block style only.
func parseYAML(src string) (goipyObject.Object, error) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	val, _, err := parseYAMLLines(lines, 0, 0)
	return val, err
}

func parseYAMLLines(lines []string, pos, indent int) (goipyObject.Object, int, error) {
	if pos >= len(lines) {
		return goipyObject.None, pos, nil
	}

	// skip blank and comment lines at current level
	for pos < len(lines) {
		l := lines[pos]
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			pos++
			continue
		}
		break
	}
	if pos >= len(lines) {
		return goipyObject.None, pos, nil
	}

	l := lines[pos]
	lineIndent := countIndent(l)
	trimmed := strings.TrimSpace(l)

	if lineIndent < indent {
		return goipyObject.None, pos, nil
	}

	// sequence item
	if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
		return parseYAMLSequence(lines, pos, lineIndent)
	}

	// mapping
	if isMappingLine(trimmed) {
		return parseYAMLMapping(lines, pos, lineIndent)
	}

	// scalar
	pos++
	return yamlScalar(trimmed), pos, nil
}

func parseYAMLSequence(lines []string, pos, indent int) (goipyObject.Object, int, error) {
	var items []goipyObject.Object
	for pos < len(lines) {
		l := lines[pos]
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			pos++
			continue
		}
		lineIndent := countIndent(l)
		if lineIndent < indent {
			break
		}
		if !strings.HasPrefix(trimmed, "- ") && trimmed != "-" {
			break
		}
		rest := strings.TrimPrefix(trimmed, "-")
		rest = strings.TrimPrefix(rest, " ")
		pos++

		if rest == "" {
			// value on next lines
			if pos < len(lines) {
				child, newPos, err := parseYAMLLines(lines, pos, indent+2)
				if err != nil {
					return nil, newPos, err
				}
				items = append(items, child)
				pos = newPos
			} else {
				items = append(items, goipyObject.None)
			}
		} else if isMappingLine(rest) {
			// inline mapping after dash
			subLines := []string{strings.Repeat(" ", indent+2) + rest}
			// collect following lines at deeper indent
			for pos < len(lines) {
				sl := lines[pos]
				si := countIndent(sl)
				st := strings.TrimSpace(sl)
				if st == "" || strings.HasPrefix(st, "#") {
					pos++
					continue
				}
				if si <= indent {
					break
				}
				subLines = append(subLines, sl)
				pos++
			}
			child, _, err := parseYAMLMapping(subLines, 0, indent+2)
			if err != nil {
				return nil, pos, err
			}
			items = append(items, child)
		} else {
			items = append(items, yamlScalar(rest))
		}
	}
	objs := make([]goipyObject.Object, len(items))
	copy(objs, items)
	return &goipyObject.List{V: objs}, pos, nil
}

func parseYAMLMapping(lines []string, pos, indent int) (goipyObject.Object, int, error) {
	d := goipyObject.NewDict()
	for pos < len(lines) {
		l := lines[pos]
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			pos++
			continue
		}
		lineIndent := countIndent(l)
		if lineIndent < indent {
			break
		}
		if !isMappingLine(trimmed) {
			break
		}
		colonIdx := strings.Index(trimmed, ": ")
		var key, rest string
		if colonIdx >= 0 {
			key = trimmed[:colonIdx]
			rest = strings.TrimSpace(trimmed[colonIdx+2:])
		} else if strings.HasSuffix(trimmed, ":") {
			key = trimmed[:len(trimmed)-1]
			rest = ""
		} else {
			break
		}
		key = strings.Trim(key, `"'`)
		pos++

		if rest == "" {
			// value on next lines
			if pos < len(lines) {
				nextL := lines[pos]
				nextTrimmed := strings.TrimSpace(nextL)
				nextIndent := countIndent(nextL)
				if nextTrimmed != "" && !strings.HasPrefix(nextTrimmed, "#") && nextIndent > indent {
					var child goipyObject.Object
					var err error
					if strings.HasPrefix(nextTrimmed, "- ") || nextTrimmed == "-" {
						child, pos, err = parseYAMLSequence(lines, pos, nextIndent)
					} else {
						child, pos, err = parseYAMLMapping(lines, pos, nextIndent)
					}
					if err != nil {
						return nil, pos, err
					}
					d.SetStr(key, child)
					continue
				}
			}
			d.SetStr(key, goipyObject.None)
		} else if strings.HasPrefix(rest, "[") {
			// flow sequence
			inner := strings.Trim(rest, "[]")
			var items []goipyObject.Object
			for _, item := range strings.Split(inner, ",") {
				item = strings.TrimSpace(item)
				if item != "" {
					items = append(items, yamlScalar(item))
				}
			}
			objs := make([]goipyObject.Object, len(items))
			copy(objs, items)
			d.SetStr(key, &goipyObject.List{V: objs})
		} else if strings.HasPrefix(rest, "{") {
			// flow mapping
			inner := strings.Trim(rest, "{}")
			sub := goipyObject.NewDict()
			for _, pair := range strings.Split(inner, ",") {
				pair = strings.TrimSpace(pair)
				k2, v2, ok := strings.Cut(pair, ":")
				if ok {
					sub.SetStr(strings.TrimSpace(k2), yamlScalar(strings.TrimSpace(v2)))
				}
			}
			d.SetStr(key, sub)
		} else {
			d.SetStr(key, yamlScalar(rest))
		}
	}
	return d, pos, nil
}

func isMappingLine(s string) bool {
	if strings.HasPrefix(s, "- ") || s == "-" {
		return false
	}
	return strings.Contains(s, ": ") || strings.HasSuffix(s, ":")
}

func countIndent(s string) int {
	for i, c := range s {
		if c != ' ' && c != '\t' {
			return i
		}
	}
	return len(s)
}

func yamlScalar(s string) goipyObject.Object {
	s = strings.Trim(s, `"'`)
	switch strings.ToLower(s) {
	case "true", "yes", "on":
		return goipyObject.BoolOf(true)
	case "false", "no", "off":
		return goipyObject.BoolOf(false)
	case "null", "~", "":
		return goipyObject.None
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return goipyObject.NewInt(i)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return &goipyObject.Float{V: f}
	}
	return &goipyObject.Str{V: s}
}

func stringifyYAML(sb *strings.Builder, obj goipyObject.Object, indent int) {
	prefix := strings.Repeat("  ", indent)
	switch v := obj.(type) {
	case *goipyObject.Dict:
		keys, vals := v.Items()
		for i, k := range keys {
			ks, ok := k.(*goipyObject.Str)
			if !ok {
				continue
			}
			val := vals[i]
			switch inner := val.(type) {
			case *goipyObject.Dict:
				sb.WriteString(prefix + ks.V + ":\n")
				stringifyYAML(sb, inner, indent+1)
			case *goipyObject.List:
				sb.WriteString(prefix + ks.V + ":\n")
				stringifyYAML(sb, inner, indent+1)
			default:
				sb.WriteString(prefix + ks.V + ": " + yamlScalarStr(val) + "\n")
			}
		}
	case *goipyObject.List:
		for _, item := range v.V {
			switch inner := item.(type) {
			case *goipyObject.Dict:
				sb.WriteString(prefix + "-\n")
				stringifyYAML(sb, inner, indent+1)
			default:
				sb.WriteString(prefix + "- " + yamlScalarStr(inner) + "\n")
			}
		}
	default:
		sb.WriteString(prefix + yamlScalarStr(obj) + "\n")
	}
}

func yamlScalarStr(obj goipyObject.Object) string {
	switch v := obj.(type) {
	case *goipyObject.Str:
		s := v.V
		if strings.ContainsAny(s, ":#{}[]|>&*!,") || strings.Contains(s, "\n") ||
			strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
			return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
		}
		return s
	case *goipyObject.Int:
		return strconv.FormatInt(v.Int64(), 10)
	case *goipyObject.Float:
		return strconv.FormatFloat(v.V, 'f', -1, 64)
	case *goipyObject.Bool:
		if v.V {
			return "true"
		}
		return "false"
	case *goipyObject.NoneType:
		return "null"
	}
	return "null"
}

package bunpy

import (
	"fmt"
	"regexp"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildURLPattern builds the bunpy.URLPattern module.
// URLPattern(pattern) compiles a URL pattern like "/users/:id" or
// "/files/*" into a matcher object with test() and exec() methods.
func BuildURLPattern(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.URLPattern", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("URLPattern", &goipyObject.BuiltinFunc{
		Name: "URLPattern",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			pattern := ""
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					pattern = s.V
				}
			} else if kwargs != nil {
				if v, ok := kwargs.GetStr("pathname"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						pattern = s.V
					}
				}
			}
			if pattern == "" {
				return nil, fmt.Errorf("URLPattern() requires a pattern string")
			}
			re, names, err := compileURLPattern(pattern)
			if err != nil {
				return nil, err
			}
			return newURLPatternInstance(pattern, re, names), nil
		},
	})

	return mod
}

func newURLPatternInstance(pattern string, re *regexp.Regexp, names []string) *goipyObject.Instance {
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "URLPattern"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("pattern", &goipyObject.Str{V: pattern})

	inst.Dict.SetStr("test", &goipyObject.BuiltinFunc{
		Name: "test",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("URLPattern.test() requires a URL or pathname string")
			}
			path := urlPatternExtractPath(args[0])
			return goipyObject.BoolOf(re.MatchString(path)), nil
		},
	})

	inst.Dict.SetStr("exec", &goipyObject.BuiltinFunc{
		Name: "exec",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("URLPattern.exec() requires a URL or pathname string")
			}
			path := urlPatternExtractPath(args[0])
			m := re.FindStringSubmatch(path)
			if m == nil {
				return goipyObject.None, nil
			}
			groups := goipyObject.NewDict()
			for i, name := range names {
				if i+1 < len(m) {
					groups.SetStr(name, &goipyObject.Str{V: m[i+1]})
				}
			}
			result := goipyObject.NewDict()
			result.SetStr("pathname", goipyObject.NewDict())
			pathnameDict := goipyObject.NewDict()
			pathnameDict.SetStr("input", &goipyObject.Str{V: path})
			pathnameDict.SetStr("groups", groups)
			result.SetStr("pathname", pathnameDict)
			return result, nil
		},
	})

	return inst
}

// compileURLPattern converts a URL pattern string like "/users/:id/posts/:postId"
// or "/files/*" into a regexp and a list of capture group names.
func compileURLPattern(pattern string) (*regexp.Regexp, []string, error) {
	var sb strings.Builder
	var names []string
	sb.WriteByte('^')

	i := 0
	for i < len(pattern) {
		// wildcard
		if pattern[i] == '*' {
			sb.WriteString("(.*)")
			names = append(names, "*")
			i++
			continue
		}
		// named param
		if pattern[i] == ':' {
			j := i + 1
			for j < len(pattern) && (isAlphaNum(pattern[j]) || pattern[j] == '_') {
				j++
			}
			name := pattern[i+1 : j]
			if name == "" {
				return nil, nil, fmt.Errorf("URLPattern: empty param name at position %d", i)
			}
			names = append(names, name)
			sb.WriteString(`([^/]+)`)
			i = j
			continue
		}
		// literal character — escape for regexp
		sb.WriteString(regexp.QuoteMeta(string(pattern[i])))
		i++
	}

	sb.WriteByte('$')
	re, err := regexp.Compile(sb.String())
	if err != nil {
		return nil, nil, fmt.Errorf("URLPattern: compile error: %w", err)
	}
	return re, names, nil
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func urlPatternExtractPath(obj goipyObject.Object) string {
	s, ok := obj.(*goipyObject.Str)
	if !ok {
		return ""
	}
	// if it looks like a full URL, extract just the path
	url := s.V
	if idx := strings.Index(url, "://"); idx >= 0 {
		rest := url[idx+3:]
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			url = rest[slash:]
		} else {
			url = "/"
		}
	}
	// strip query string and fragment
	if idx := strings.IndexByte(url, '?'); idx >= 0 {
		url = url[:idx]
	}
	if idx := strings.IndexByte(url, '#'); idx >= 0 {
		url = url[:idx]
	}
	return url
}

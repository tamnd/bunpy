package bunpy

import (
	"fmt"
	"strconv"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildSemver(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.semver", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("parse", &goipyObject.BuiltinFunc{
		Name: "parse",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("semver.parse() requires a version string")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("semver.parse(): version must be str")
			}
			v, err := parseSemver(s.V)
			if err != nil {
				return nil, err
			}
			return semverToDict(v), nil
		},
	})

	mod.Dict.SetStr("compare", &goipyObject.BuiltinFunc{
		Name: "compare",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("semver.compare() requires two version strings")
			}
			a, ok1 := args[0].(*goipyObject.Str)
			b, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("semver.compare(): versions must be str")
			}
			va, err := parseSemver(a.V)
			if err != nil {
				return nil, err
			}
			vb, err := parseSemver(b.V)
			if err != nil {
				return nil, err
			}
			return goipyObject.NewInt(int64(compareSemver(va, vb))), nil
		},
	})

	mod.Dict.SetStr("satisfies", &goipyObject.BuiltinFunc{
		Name: "satisfies",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("semver.satisfies() requires version and range")
			}
			vs, ok1 := args[0].(*goipyObject.Str)
			rs, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("semver.satisfies(): version and range must be str")
			}
			ok, err := semverSatisfies(vs.V, rs.V)
			if err != nil {
				return nil, err
			}
			return goipyObject.BoolOf(ok), nil
		},
	})

	mod.Dict.SetStr("gt", buildSemverCmp("gt", func(c int) bool { return c > 0 }))
	mod.Dict.SetStr("gte", buildSemverCmp("gte", func(c int) bool { return c >= 0 }))
	mod.Dict.SetStr("lt", buildSemverCmp("lt", func(c int) bool { return c < 0 }))
	mod.Dict.SetStr("lte", buildSemverCmp("lte", func(c int) bool { return c <= 0 }))
	mod.Dict.SetStr("eq", buildSemverCmp("eq", func(c int) bool { return c == 0 }))

	mod.Dict.SetStr("valid", &goipyObject.BuiltinFunc{
		Name: "valid",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.BoolOf(false), nil
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return goipyObject.BoolOf(false), nil
			}
			_, err := parseSemver(s.V)
			return goipyObject.BoolOf(err == nil), nil
		},
	})

	return mod
}

func buildSemverCmp(name string, pred func(int) bool) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: name,
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("semver.%s() requires two version strings", name)
			}
			a, ok1 := args[0].(*goipyObject.Str)
			b, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("semver.%s(): versions must be str", name)
			}
			va, err := parseSemver(a.V)
			if err != nil {
				return nil, err
			}
			vb, err := parseSemver(b.V)
			if err != nil {
				return nil, err
			}
			return goipyObject.BoolOf(pred(compareSemver(va, vb))), nil
		},
	}
}

type semver struct {
	major, minor, patch int
	pre                 string
	build               string
}

func parseSemver(s string) (semver, error) {
	s = strings.TrimPrefix(s, "v")
	var v semver
	// strip build metadata
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		v.build = s[idx+1:]
		s = s[:idx]
	}
	// strip pre-release
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		v.pre = s[idx+1:]
		s = s[:idx]
	}
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("semver.parse(%q): invalid semver", s)
	}
	var err error
	v.major, err = strconv.Atoi(parts[0])
	if err != nil {
		return semver{}, fmt.Errorf("semver.parse: invalid major %q", parts[0])
	}
	v.minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return semver{}, fmt.Errorf("semver.parse: invalid minor %q", parts[1])
	}
	v.patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return semver{}, fmt.Errorf("semver.parse: invalid patch %q", parts[2])
	}
	return v, nil
}

func compareSemver(a, b semver) int {
	if a.major != b.major {
		return cmpInt(a.major, b.major)
	}
	if a.minor != b.minor {
		return cmpInt(a.minor, b.minor)
	}
	if a.patch != b.patch {
		return cmpInt(a.patch, b.patch)
	}
	// pre-release: version without pre > version with pre
	if a.pre == "" && b.pre != "" {
		return 1
	}
	if a.pre != "" && b.pre == "" {
		return -1
	}
	if a.pre != b.pre {
		if a.pre < b.pre {
			return -1
		}
		return 1
	}
	return 0
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func semverToDict(v semver) *goipyObject.Dict {
	d := goipyObject.NewDict()
	d.SetStr("major", goipyObject.NewInt(int64(v.major)))
	d.SetStr("minor", goipyObject.NewInt(int64(v.minor)))
	d.SetStr("patch", goipyObject.NewInt(int64(v.patch)))
	d.SetStr("pre", &goipyObject.Str{V: v.pre})
	d.SetStr("build", &goipyObject.Str{V: v.build})
	return d
}

// semverSatisfies checks if version satisfies a simple range like ">=1.0.0",
// "^1.2.3", "~1.2.3", "1.x", or an exact version.
func semverSatisfies(version, rangeStr string) (bool, error) {
	rangeStr = strings.TrimSpace(rangeStr)
	v, err := parseSemver(version)
	if err != nil {
		return false, err
	}

	// caret range: ^1.2.3 means >=1.2.3 <2.0.0
	if strings.HasPrefix(rangeStr, "^") {
		base, err := parseSemver(rangeStr[1:])
		if err != nil {
			return false, err
		}
		if compareSemver(v, base) < 0 {
			return false, nil
		}
		upper := semver{major: base.major + 1}
		return compareSemver(v, upper) < 0, nil
	}

	// tilde range: ~1.2.3 means >=1.2.3 <1.3.0
	if strings.HasPrefix(rangeStr, "~") {
		base, err := parseSemver(rangeStr[1:])
		if err != nil {
			return false, err
		}
		if compareSemver(v, base) < 0 {
			return false, nil
		}
		upper := semver{major: base.major, minor: base.minor + 1}
		return compareSemver(v, upper) < 0, nil
	}

	// comparison operators
	for _, op := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(rangeStr, op) {
			base, err := parseSemver(strings.TrimPrefix(rangeStr, op))
			if err != nil {
				return false, err
			}
			c := compareSemver(v, base)
			switch op {
			case ">=":
				return c >= 0, nil
			case "<=":
				return c <= 0, nil
			case ">":
				return c > 0, nil
			case "<":
				return c < 0, nil
			case "=":
				return c == 0, nil
			}
		}
	}

	// wildcard: 1.x or 1.2.x
	if strings.Contains(rangeStr, "x") || strings.Contains(rangeStr, "*") {
		normalized := strings.ReplaceAll(rangeStr, "*", "x")
		parts := strings.SplitN(normalized, ".", 3)
		if parts[0] != "x" {
			maj, err := strconv.Atoi(parts[0])
			if err != nil {
				return false, fmt.Errorf("semver range: invalid %q", rangeStr)
			}
			if v.major != maj {
				return false, nil
			}
		}
		if len(parts) >= 2 && parts[1] != "x" {
			min, err := strconv.Atoi(parts[1])
			if err != nil {
				return false, fmt.Errorf("semver range: invalid %q", rangeStr)
			}
			if v.minor != min {
				return false, nil
			}
		}
		return true, nil
	}

	// exact match
	base, err := parseSemver(rangeStr)
	if err != nil {
		return false, err
	}
	return compareSemver(v, base) == 0, nil
}

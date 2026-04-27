package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func semverMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildSemver(nil)
}

func TestSemverModuleMethods(t *testing.T) {
	mod := semverMod(t)
	for _, name := range []string{"parse", "compare", "satisfies", "gt", "gte", "lt", "lte", "eq", "valid"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("semver module missing %q", name)
		}
	}
}

func TestSemverParse(t *testing.T) {
	mod := semverMod(t)
	fn, _ := mod.Dict.GetStr("parse")
	result, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "1.2.3"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := result.(*goipyObject.Dict)
	if !ok {
		t.Fatalf("expected dict, got %T", result)
	}
	for key, want := range map[string]int64{"major": 1, "minor": 2, "patch": 3} {
		v, _ := d.GetStr(key)
		if iv, ok := v.(*goipyObject.Int); !ok || iv.Int64() != want {
			t.Errorf("%s: expected %d, got %v", key, want, v)
		}
	}
}

func TestSemverParseWithVPrefix(t *testing.T) {
	mod := semverMod(t)
	fn, _ := mod.Dict.GetStr("parse")
	_, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "v2.0.0"},
	}, nil)
	if err != nil {
		t.Fatalf("v-prefix parse failed: %v", err)
	}
}

func TestSemverCompare(t *testing.T) {
	mod := semverMod(t)
	fn, _ := mod.Dict.GetStr("compare")
	call := func(a, b string) int64 {
		t.Helper()
		r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
			&goipyObject.Str{V: a}, &goipyObject.Str{V: b},
		}, nil)
		if err != nil {
			t.Fatalf("compare(%s,%s): %v", a, b, err)
		}
		return r.(*goipyObject.Int).Int64()
	}
	if c := call("1.0.0", "2.0.0"); c >= 0 {
		t.Errorf("1.0.0 < 2.0.0 expected -1, got %d", c)
	}
	if c := call("2.0.0", "1.0.0"); c <= 0 {
		t.Errorf("2.0.0 > 1.0.0 expected 1, got %d", c)
	}
	if c := call("1.0.0", "1.0.0"); c != 0 {
		t.Errorf("equal expected 0, got %d", c)
	}
}

func TestSemverGtLt(t *testing.T) {
	mod := semverMod(t)
	callBool := func(fn *goipyObject.BuiltinFunc, a, b string) bool {
		r, err := fn.Call(nil, []goipyObject.Object{
			&goipyObject.Str{V: a}, &goipyObject.Str{V: b},
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return r.(*goipyObject.Bool).V
	}
	gt, _ := mod.Dict.GetStr("gt")
	lt, _ := mod.Dict.GetStr("lt")
	if !callBool(gt.(*goipyObject.BuiltinFunc), "2.0.0", "1.0.0") {
		t.Error("gt(2.0.0, 1.0.0) should be true")
	}
	if !callBool(lt.(*goipyObject.BuiltinFunc), "1.0.0", "2.0.0") {
		t.Error("lt(1.0.0, 2.0.0) should be true")
	}
}

func TestSemverValid(t *testing.T) {
	mod := semverMod(t)
	fn, _ := mod.Dict.GetStr("valid")
	check := func(s string, want bool) {
		r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
			&goipyObject.Str{V: s},
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if r.(*goipyObject.Bool).V != want {
			t.Errorf("valid(%q) expected %v", s, want)
		}
	}
	check("1.2.3", true)
	check("v1.2.3", true)
	check("notaversion", false)
}

func TestSemverSatisfiesCaret(t *testing.T) {
	mod := semverMod(t)
	fn, _ := mod.Dict.GetStr("satisfies")
	check := func(v, r string, want bool) {
		res, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
			&goipyObject.Str{V: v}, &goipyObject.Str{V: r},
		}, nil)
		if err != nil {
			t.Fatalf("satisfies(%s, %s): %v", v, r, err)
		}
		if res.(*goipyObject.Bool).V != want {
			t.Errorf("satisfies(%s, %s) expected %v", v, r, want)
		}
	}
	check("1.5.0", "^1.0.0", true)
	check("2.0.0", "^1.0.0", false)
	check("1.2.3", "~1.2.0", true)
	check("1.3.0", "~1.2.0", false)
	check("1.2.3", ">=1.2.0", true)
	check("1.1.0", ">=1.2.0", false)
	check("1.2.3", "1.x", true)
	check("2.0.0", "1.x", false)
}

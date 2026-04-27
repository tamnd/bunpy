package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func cookieMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildCookie(nil)
}

func TestCookieModuleMethods(t *testing.T) {
	mod := cookieMod(t)
	for _, name := range []string{"parse", "serialize"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("cookie module missing %q", name)
		}
	}
}

func TestCookieParse(t *testing.T) {
	mod := cookieMod(t)
	fn, _ := mod.Dict.GetStr("parse")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "session=abc123; user=alice"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := r.(*goipyObject.Dict)
	if !ok {
		t.Fatalf("expected dict, got %T", r)
	}
	if v, ok2 := d.GetStr("session"); !ok2 || v.(*goipyObject.Str).V != "abc123" {
		t.Errorf("session cookie not parsed correctly: %v", v)
	}
	if v, ok2 := d.GetStr("user"); !ok2 || v.(*goipyObject.Str).V != "alice" {
		t.Errorf("user cookie not parsed correctly: %v", v)
	}
}

func TestCookieParseEmpty(t *testing.T) {
	mod := cookieMod(t)
	fn, _ := mod.Dict.GetStr("parse")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: ""},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := r.(*goipyObject.Dict)
	keys, _ := d.Items()
	if len(keys) != 0 {
		t.Errorf("expected empty dict for empty cookie string, got %d keys", len(keys))
	}
}

func TestCookieSerializeBasic(t *testing.T) {
	mod := cookieMod(t)
	fn, _ := mod.Dict.GetStr("serialize")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "session"},
		&goipyObject.Str{V: "abc123"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.(*goipyObject.Str).V
	if !strings.Contains(got, "session=abc123") {
		t.Errorf("expected session=abc123 in %q", got)
	}
}

func TestCookieSerializeWithOptions(t *testing.T) {
	mod := cookieMod(t)
	fn, _ := mod.Dict.GetStr("serialize")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("http_only", goipyObject.BoolOf(true))
	kwargs.SetStr("secure", goipyObject.BoolOf(true))
	kwargs.SetStr("path", &goipyObject.Str{V: "/"})
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "token"},
		&goipyObject.Str{V: "xyz"},
	}, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	got := r.(*goipyObject.Str).V
	if !strings.Contains(got, "HttpOnly") {
		t.Errorf("expected HttpOnly in %q", got)
	}
	if !strings.Contains(got, "Secure") {
		t.Errorf("expected Secure in %q", got)
	}
}

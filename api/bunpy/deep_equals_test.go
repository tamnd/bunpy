package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func deepEqMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildDeepEquals(nil)
}

func callDeepEq(t *testing.T, a, b goipyObject.Object) bool {
	t.Helper()
	mod := deepEqMod(t)
	fn, _ := mod.Dict.GetStr("deep_equals")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{a, b}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*goipyObject.Bool).V
}

func TestDeepEqualsModule(t *testing.T) {
	mod := deepEqMod(t)
	if _, ok := mod.Dict.GetStr("deep_equals"); !ok {
		t.Fatal("deep_equals not found in module")
	}
}

func TestDeepEqualsInts(t *testing.T) {
	if !callDeepEq(t, goipyObject.NewInt(42), goipyObject.NewInt(42)) {
		t.Error("42 == 42")
	}
	if callDeepEq(t, goipyObject.NewInt(1), goipyObject.NewInt(2)) {
		t.Error("1 != 2")
	}
}

func TestDeepEqualsStrings(t *testing.T) {
	if !callDeepEq(t, &goipyObject.Str{V: "hello"}, &goipyObject.Str{V: "hello"}) {
		t.Error("hello == hello")
	}
	if callDeepEq(t, &goipyObject.Str{V: "a"}, &goipyObject.Str{V: "b"}) {
		t.Error("a != b")
	}
}

func TestDeepEqualsLists(t *testing.T) {
	a := &goipyObject.List{V: []goipyObject.Object{goipyObject.NewInt(1), goipyObject.NewInt(2)}}
	b := &goipyObject.List{V: []goipyObject.Object{goipyObject.NewInt(1), goipyObject.NewInt(2)}}
	c := &goipyObject.List{V: []goipyObject.Object{goipyObject.NewInt(1), goipyObject.NewInt(3)}}
	if !callDeepEq(t, a, b) {
		t.Error("[1,2] == [1,2]")
	}
	if callDeepEq(t, a, c) {
		t.Error("[1,2] != [1,3]")
	}
}

func TestDeepEqualsDicts(t *testing.T) {
	a := goipyObject.NewDict()
	a.SetStr("x", goipyObject.NewInt(1))
	b := goipyObject.NewDict()
	b.SetStr("x", goipyObject.NewInt(1))
	c := goipyObject.NewDict()
	c.SetStr("x", goipyObject.NewInt(2))
	if !callDeepEq(t, a, b) {
		t.Error("{x:1} == {x:1}")
	}
	if callDeepEq(t, a, c) {
		t.Error("{x:1} != {x:2}")
	}
}

func TestDeepEqualsNested(t *testing.T) {
	inner1 := &goipyObject.List{V: []goipyObject.Object{goipyObject.NewInt(1)}}
	inner2 := &goipyObject.List{V: []goipyObject.Object{goipyObject.NewInt(1)}}
	a := goipyObject.NewDict()
	a.SetStr("items", inner1)
	b := goipyObject.NewDict()
	b.SetStr("items", inner2)
	if !callDeepEq(t, a, b) {
		t.Error("nested equal dicts should match")
	}
}

func TestDeepEqualsNone(t *testing.T) {
	if !callDeepEq(t, goipyObject.None, goipyObject.None) {
		t.Error("None == None")
	}
	if callDeepEq(t, goipyObject.None, goipyObject.NewInt(0)) {
		t.Error("None != 0")
	}
}

func TestDeepEqualsDifferentTypes(t *testing.T) {
	if callDeepEq(t, &goipyObject.Str{V: "1"}, goipyObject.NewInt(1)) {
		t.Error("str '1' != int 1")
	}
}

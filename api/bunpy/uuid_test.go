package bunpy_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

var uuidV4RE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestUUIDV4Format(t *testing.T) {
	mod := bunpyAPI.BuildUUID(nil)
	v4Fn, _ := mod.Dict.GetStr("v4")
	result, err := v4Fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*goipyObject.Str)
	if !ok {
		t.Fatalf("expected Str, got %T", result)
	}
	if !uuidV4RE.MatchString(s.V) {
		t.Fatalf("UUID v4 format mismatch: %q", s.V)
	}
}

func TestUUIDV4Unique(t *testing.T) {
	mod := bunpyAPI.BuildUUID(nil)
	v4Fn, _ := mod.Dict.GetStr("v4")
	r1, _ := v4Fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	r2, _ := v4Fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if r1.(*goipyObject.Str).V == r2.(*goipyObject.Str).V {
		t.Fatal("two consecutive v4() calls returned the same UUID")
	}
}

func TestUUIDV7Version(t *testing.T) {
	mod := bunpyAPI.BuildUUID(nil)
	v7Fn, _ := mod.Dict.GetStr("v7")
	result, err := v7Fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := result.(*goipyObject.Str).V
	// 15th character (index 14) is the version nibble
	parts := strings.Split(s, "-")
	if len(parts) != 5 || parts[2][0] != '7' {
		t.Fatalf("UUID v7 version nibble mismatch: %q", s)
	}
}

func TestUUIDV7Ordered(t *testing.T) {
	mod := bunpyAPI.BuildUUID(nil)
	v7Fn, _ := mod.Dict.GetStr("v7")
	r1, _ := v7Fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	time.Sleep(2 * time.Millisecond)
	r2, _ := v7Fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	s1 := r1.(*goipyObject.Str).V
	s2 := r2.(*goipyObject.Str).V
	// Lexicographic order should be non-decreasing (same millisecond is fine)
	if s1 > s2 {
		t.Fatalf("UUID v7 ordering violated: %q > %q", s1, s2)
	}
}

func TestUUIDIsValid(t *testing.T) {
	mod := bunpyAPI.BuildUUID(nil)
	isValidFn, _ := mod.Dict.GetStr("is_valid")
	v4Fn, _ := mod.Dict.GetStr("v4")

	result, _ := v4Fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	ok, err := isValidFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{result}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok.(*goipyObject.Bool).V {
		t.Fatal("is_valid returned false for a freshly generated UUID")
	}
}

func TestUUIDIsValidFalse(t *testing.T) {
	mod := bunpyAPI.BuildUUID(nil)
	isValidFn, _ := mod.Dict.GetStr("is_valid")
	ok, _ := isValidFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "not-a-uuid"},
	}, nil)
	if ok.(*goipyObject.Bool).V {
		t.Fatal("is_valid should return false for 'not-a-uuid'")
	}
}

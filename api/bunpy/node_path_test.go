package bunpy

import (
	"path/filepath"
	"runtime"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
)

func TestNodePathJoin(t *testing.T) {
	mod := BuildNodePath(nil)
	fn := mustGetBuiltin(t, mod.Dict, "join")
	res, err := fn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "a"},
		&goipyObject.Str{V: "b"},
		&goipyObject.Str{V: "c.txt"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("a", "b", "c.txt")
	if s, ok := res.(*goipyObject.Str); !ok || s.V != want {
		t.Errorf("got %v, want %q", res, want)
	}
}

func TestNodePathDirname(t *testing.T) {
	mod := BuildNodePath(nil)
	fn := mustGetBuiltin(t, mod.Dict, "dirname")
	res, _ := fn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "/a/b/c.txt"}}, nil)
	s, ok := res.(*goipyObject.Str)
	if !ok {
		t.Fatalf("expected Str, got %T", res)
	}
	if s.V != filepath.Dir("/a/b/c.txt") {
		t.Errorf("got %q", s.V)
	}
}

func TestNodePathBasename(t *testing.T) {
	mod := BuildNodePath(nil)
	fn := mustGetBuiltin(t, mod.Dict, "basename")
	res, _ := fn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "/a/b/c.txt"},
		&goipyObject.Str{V: ".txt"},
	}, nil)
	s, ok := res.(*goipyObject.Str)
	if !ok || s.V != "c" {
		t.Errorf("got %v", res)
	}
}

func TestNodePathExtname(t *testing.T) {
	mod := BuildNodePath(nil)
	fn := mustGetBuiltin(t, mod.Dict, "extname")
	res, _ := fn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "file.go"}}, nil)
	if s, ok := res.(*goipyObject.Str); !ok || s.V != ".go" {
		t.Errorf("got %v", res)
	}
}

func TestNodePathIsAbsolute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style absolute paths (no drive letter) not absolute on Windows")
	}
	mod := BuildNodePath(nil)
	fn := mustGetBuiltin(t, mod.Dict, "isAbsolute")
	res, _ := fn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "/absolute"}}, nil)
	if res != goipyObject.BoolOf(true) {
		t.Error("expected true for absolute path")
	}
	res2, _ := fn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "relative"}}, nil)
	if res2 != goipyObject.BoolOf(false) {
		t.Error("expected false for relative path")
	}
}

func TestNodePathNormalize(t *testing.T) {
	mod := BuildNodePath(nil)
	fn := mustGetBuiltin(t, mod.Dict, "normalize")
	res, _ := fn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "a/b/../c"}}, nil)
	want := filepath.Clean("a/b/../c")
	if s, ok := res.(*goipyObject.Str); !ok || s.V != want {
		t.Errorf("got %v, want %q", res, want)
	}
}

func TestNodePathSepAndDelimiter(t *testing.T) {
	mod := BuildNodePath(nil)
	sep, _ := mod.Dict.GetStr("sep")
	if s, ok := sep.(*goipyObject.Str); !ok || s.V != string(filepath.Separator) {
		t.Errorf("sep: got %v", sep)
	}
	delim, _ := mod.Dict.GetStr("delimiter")
	want := ":"
	if runtime.GOOS == "windows" {
		want = ";"
	}
	if s, ok := delim.(*goipyObject.Str); !ok || s.V != want {
		t.Errorf("delimiter: got %v", delim)
	}
}

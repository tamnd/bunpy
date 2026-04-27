package bunpy_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func buildGlobTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.py"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte(""), 0o644)
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "c.py"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(sub, ".hidden.py"), []byte(""), 0o644)
	deep := filepath.Join(sub, "deep")
	os.MkdirAll(deep, 0o755)
	os.WriteFile(filepath.Join(deep, "d.py"), []byte(""), 0o644)
	return dir
}

func globResult(t *testing.T, fn *goipyObject.BuiltinFunc, args []goipyObject.Object, kwargs *goipyObject.Dict) []string {
	t.Helper()
	result, err := fn.Call(nil, args, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	lst := result.(*goipyObject.List)
	out := make([]string, len(lst.V))
	for i, item := range lst.V {
		out[i] = item.(*goipyObject.Str).V
	}
	sort.Strings(out)
	return out
}

func TestGlobNonRecursive(t *testing.T) {
	dir := buildGlobTree(t)
	i := serveInterp(t)
	fn := bunpyAPI.BuildGlob(i)
	kw := goipyObject.NewDict()
	kw.SetStr("cwd", &goipyObject.Str{V: dir})
	got := globResult(t, fn, []goipyObject.Object{&goipyObject.Str{V: "*.py"}}, kw)
	if len(got) != 1 || got[0] != "a.py" {
		t.Fatalf("got %v, want [a.py]", got)
	}
}

func TestGlobRecursive(t *testing.T) {
	dir := buildGlobTree(t)
	i := serveInterp(t)
	fn := bunpyAPI.BuildGlob(i)
	kw := goipyObject.NewDict()
	kw.SetStr("cwd", &goipyObject.Str{V: dir})
	got := globResult(t, fn, []goipyObject.Object{&goipyObject.Str{V: "**/*.py"}}, kw)
	want := []string{"a.py", "sub/c.py", "sub/deep/d.py"}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i2 := range want {
		if got[i2] != want[i2] {
			t.Fatalf("got[%d] = %q, want %q", i2, got[i2], want[i2])
		}
	}
}

func TestGlobDotFiles(t *testing.T) {
	dir := buildGlobTree(t)
	i := serveInterp(t)
	fn := bunpyAPI.BuildGlob(i)
	kw := goipyObject.NewDict()
	kw.SetStr("cwd", &goipyObject.Str{V: dir})
	kw.SetStr("dot", goipyObject.BoolOf(true))
	got := globResult(t, fn, []goipyObject.Object{&goipyObject.Str{V: "**/*.py"}}, kw)
	found := false
	for _, p := range got {
		if p == "sub/.hidden.py" {
			found = true
		}
	}
	if !found {
		t.Fatalf("dot=True should include hidden files, got %v", got)
	}
}

func TestGlobNoDotFilesByDefault(t *testing.T) {
	dir := buildGlobTree(t)
	i := serveInterp(t)
	fn := bunpyAPI.BuildGlob(i)
	kw := goipyObject.NewDict()
	kw.SetStr("cwd", &goipyObject.Str{V: dir})
	got := globResult(t, fn, []goipyObject.Object{&goipyObject.Str{V: "**/*.py"}}, kw)
	for _, p := range got {
		if p == "sub/.hidden.py" {
			t.Fatalf("dot=False should exclude hidden files, got %v", got)
		}
	}
}

func TestGlobAbsolutePaths(t *testing.T) {
	dir := buildGlobTree(t)
	i := serveInterp(t)
	fn := bunpyAPI.BuildGlob(i)
	kw := goipyObject.NewDict()
	kw.SetStr("cwd", &goipyObject.Str{V: dir})
	kw.SetStr("absolute", goipyObject.BoolOf(true))
	got := globResult(t, fn, []goipyObject.Object{&goipyObject.Str{V: "**/*.py"}}, kw)
	for _, p := range got {
		if !filepath.IsAbs(p) {
			t.Fatalf("absolute=True should return absolute paths, got %q", p)
		}
	}
}

func TestGlobMatch(t *testing.T) {
	i := serveInterp(t)
	fn := bunpyAPI.BuildGlobMatch(i)

	res, err := fn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "*.py"},
		&goipyObject.Str{V: "script.py"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.(*goipyObject.Bool).V {
		t.Fatal("*.py should match script.py")
	}

	res2, _ := fn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "*.py"},
		&goipyObject.Str{V: "script.go"},
	}, nil)
	if res2.(*goipyObject.Bool).V {
		t.Fatal("*.py should not match script.go")
	}
}

func TestBunpyModuleHasGlob(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	for _, name := range []string{"glob", "glob_match"} {
		if _, ok := m.Dict.GetStr(name); !ok {
			t.Fatalf("bunpy.%s missing from top-level module", name)
		}
	}
}

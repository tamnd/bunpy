package bunpy

import (
	"os"
	"path/filepath"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
)

func TestNodeFSReadWriteFile(t *testing.T) {
	mod := BuildNodeFS(nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")

	writeFn := mustGetBuiltin(t, mod.Dict, "writeFile")
	_, err := writeFn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: path},
		&goipyObject.Str{V: "hello world"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	readFn := mustGetBuiltin(t, mod.Dict, "readFile")
	result, err := readFn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: path},
		&goipyObject.Str{V: "utf8"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*goipyObject.Str)
	if !ok {
		t.Fatalf("expected Str, got %T", result)
	}
	if s.V != "hello world" {
		t.Errorf("got %q", s.V)
	}
}

func TestNodeFSReadFileBytes(t *testing.T) {
	mod := BuildNodeFS(nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "bin.dat")
	os.WriteFile(path, []byte{1, 2, 3}, 0o644)

	readFn := mustGetBuiltin(t, mod.Dict, "readFile")
	result, err := readFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := result.(*goipyObject.Bytes)
	if !ok {
		t.Fatalf("expected Bytes, got %T", result)
	}
	if len(b.V) != 3 {
		t.Errorf("expected 3 bytes, got %d", len(b.V))
	}
}

func TestNodeFSExists(t *testing.T) {
	mod := BuildNodeFS(nil)
	dir := t.TempDir()
	path := filepath.Join(dir, "x.txt")
	os.WriteFile(path, []byte("x"), 0o644)

	existsFn := mustGetBuiltin(t, mod.Dict, "exists")
	res, _ := existsFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}}, nil)
	if res != goipyObject.BoolOf(true) {
		t.Error("expected true")
	}

	res2, _ := existsFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path + ".nope"}}, nil)
	if res2 != goipyObject.BoolOf(false) {
		t.Error("expected false")
	}
}

func TestNodeFSMkdirAndReaddir(t *testing.T) {
	mod := BuildNodeFS(nil)
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")

	mkdirFn := mustGetBuiltin(t, mod.Dict, "mkdir")
	_, err := mkdirFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: sub}}, nil)
	if err != nil {
		t.Fatal(err)
	}

	os.WriteFile(filepath.Join(sub, "a.txt"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(sub, "b.txt"), []byte("b"), 0o644)

	readdirFn := mustGetBuiltin(t, mod.Dict, "readdir")
	res, err := readdirFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: sub}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	lst, ok := res.(*goipyObject.List)
	if !ok {
		t.Fatalf("expected List, got %T", res)
	}
	if len(lst.V) != 2 {
		t.Errorf("expected 2 entries, got %d", len(lst.V))
	}
}

func TestNodeFSStat(t *testing.T) {
	mod := BuildNodeFS(nil)
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	os.WriteFile(path, []byte("abc"), 0o644)

	statFn := mustGetBuiltin(t, mod.Dict, "stat")
	res, err := statFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := res.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", res)
	}
	sizeObj, _ := inst.Dict.GetStr("size")
	if n, ok := sizeObj.(*goipyObject.Int); !ok || n.Int64() != 3 {
		t.Errorf("expected size=3, got %v", sizeObj)
	}
}

func TestNodeFSAppendFile(t *testing.T) {
	mod := BuildNodeFS(nil)
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")

	appendFn := mustGetBuiltin(t, mod.Dict, "appendFile")
	appendFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}, &goipyObject.Str{V: "line1\n"}}, nil)
	appendFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}, &goipyObject.Str{V: "line2\n"}}, nil)

	data, _ := os.ReadFile(path)
	if string(data) != "line1\nline2\n" {
		t.Errorf("got %q", data)
	}
}

func TestNodeFSCopyFile(t *testing.T) {
	mod := BuildNodeFS(nil)
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	os.WriteFile(src, []byte("copy me"), 0o644)

	copyFn := mustGetBuiltin(t, mod.Dict, "copyFile")
	_, err := copyFn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: src}, &goipyObject.Str{V: dst},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "copy me" {
		t.Errorf("got %q", data)
	}
}

func TestNodeFSUnlinkAndRmdir(t *testing.T) {
	mod := BuildNodeFS(nil)
	dir := t.TempDir()
	path := filepath.Join(dir, "del.txt")
	os.WriteFile(path, []byte("x"), 0o644)

	unlinkFn := mustGetBuiltin(t, mod.Dict, "unlink")
	_, err := unlinkFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}

	sub := filepath.Join(dir, "subdir")
	os.Mkdir(sub, 0o755)
	rmdirFn := mustGetBuiltin(t, mod.Dict, "rmdir")
	_, err = rmdirFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: sub}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sub); !os.IsNotExist(err) {
		t.Error("dir should be deleted")
	}
}

func TestNodeFSRename(t *testing.T) {
	mod := BuildNodeFS(nil)
	dir := t.TempDir()
	src := filepath.Join(dir, "old.txt")
	dst := filepath.Join(dir, "new.txt")
	os.WriteFile(src, []byte("rename"), 0o644)

	renameFn := mustGetBuiltin(t, mod.Dict, "rename")
	_, err := renameFn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: src}, &goipyObject.Str{V: dst},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Error("new file should exist")
	}
}

func mustGetBuiltin(t *testing.T, d *goipyObject.Dict, name string) *goipyObject.BuiltinFunc {
	t.Helper()
	v, ok := d.GetStr(name)
	if !ok {
		t.Fatalf("missing %q in dict", name)
	}
	fn, ok := v.(*goipyObject.BuiltinFunc)
	if !ok {
		t.Fatalf("%q is not a BuiltinFunc", name)
	}
	return fn
}

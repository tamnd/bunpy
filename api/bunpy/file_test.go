package bunpy_test

import (
	"os"
	"path/filepath"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestBunFileText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	os.WriteFile(path, []byte("hello world"), 0o644)

	i := serveInterp(t)
	fileFn := bunpyAPI.BuildFile(i)
	result, err := fileFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)
	textFn, _ := inst.Dict.GetStr("text")
	got, err := textFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.(*goipyObject.Str).V != "hello world" {
		t.Fatalf("text = %q, want %q", got.(*goipyObject.Str).V, "hello world")
	}
}

func TestBunFileBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.bin")
	os.WriteFile(path, []byte{0x01, 0x02, 0x03}, 0o644)

	i := serveInterp(t)
	fileFn := bunpyAPI.BuildFile(i)
	result, _ := fileFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}}, nil)
	inst := result.(*goipyObject.Instance)
	bytesFn, _ := inst.Dict.GetStr("bytes")
	got, err := bytesFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := got.(*goipyObject.Bytes).V
	if len(b) != 3 || b[0] != 0x01 {
		t.Fatalf("bytes = %v, want [1 2 3]", b)
	}
}

func TestBunFileSizeAndExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("abcde"), 0o644)

	i := serveInterp(t)
	fileFn := bunpyAPI.BuildFile(i)
	result, _ := fileFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}}, nil)
	inst := result.(*goipyObject.Instance)

	sizeFn, _ := inst.Dict.GetStr("size")
	sz, err := sizeFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sz.(*goipyObject.Int).Int64() != 5 {
		t.Fatalf("size = %d, want 5", sz.(*goipyObject.Int).Int64())
	}

	existsFn, _ := inst.Dict.GetStr("exists")
	ex, _ := existsFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if !ex.(*goipyObject.Bool).V {
		t.Fatal("exists should be true")
	}

	// missing path
	result2, _ := fileFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "/no/such/file"}}, nil)
	inst2 := result2.(*goipyObject.Instance)
	existsFn2, _ := inst2.Dict.GetStr("exists")
	ex2, _ := existsFn2.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if ex2.(*goipyObject.Bool).V {
		t.Fatal("exists should be false for missing file")
	}
}

func TestBunFileTextMissing(t *testing.T) {
	i := serveInterp(t)
	fileFn := bunpyAPI.BuildFile(i)
	result, _ := fileFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "/no/such/file.txt"}}, nil)
	inst := result.(*goipyObject.Instance)
	textFn, _ := inst.Dict.GetStr("text")
	_, err := textFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBunpyWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	i := serveInterp(t)
	writeFn := bunpyAPI.BuildWrite(i)
	_, err := writeFn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: path},
		&goipyObject.Str{V: "hello"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "hello" {
		t.Fatalf("file content = %q, want %q", b, "hello")
	}
}

func TestBunpyWriteAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	os.WriteFile(path, []byte("line1\n"), 0o644)

	i := serveInterp(t)
	writeFn := bunpyAPI.BuildWrite(i)
	kw := goipyObject.NewDict()
	kw.SetStr("append", goipyObject.BoolOf(true))
	_, err := writeFn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: path},
		&goipyObject.Str{V: "line2\n"},
	}, kw)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "line1\nline2\n" {
		t.Fatalf("content = %q", b)
	}
}

func TestBunpyWriteBunFileCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	os.WriteFile(src, []byte{0xAA, 0xBB}, 0o644)

	i := serveInterp(t)
	fileFn := bunpyAPI.BuildFile(i)
	writeFn := bunpyAPI.BuildWrite(i)

	srcInst, _ := fileFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: src}}, nil)
	_, err := writeFn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: dst},
		srcInst,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dst)
	if len(b) != 2 || b[0] != 0xAA {
		t.Fatalf("copy failed: %v", b)
	}
}

func TestBunpyRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.bin")
	os.WriteFile(path, []byte{1, 2, 3}, 0o644)

	i := serveInterp(t)
	readFn := bunpyAPI.BuildRead(i)
	got, err := readFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: path}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := got.(*goipyObject.Bytes).V
	if len(b) != 3 || b[2] != 3 {
		t.Fatalf("read = %v", b)
	}
}

func TestBunpyModuleHasFileWriteRead(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	for _, name := range []string{"file", "write", "read"} {
		if _, ok := m.Dict.GetStr(name); !ok {
			t.Fatalf("bunpy.%s missing from top-level module", name)
		}
	}
}

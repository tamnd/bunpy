package bunpy

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
)

func TestNodeZlibGzipRoundtrip(t *testing.T) {
	mod := BuildNodeZlib(nil)

	gzipFn := mustGetBuiltin(t, mod.Dict, "gzip")
	gunzipFn := mustGetBuiltin(t, mod.Dict, "gunzip")

	original := "hello zlib world"
	compressed, err := gzipFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: original}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cb, ok := compressed.(*goipyObject.Bytes)
	if !ok {
		t.Fatalf("expected Bytes, got %T", compressed)
	}

	decompressed, err := gunzipFn.Call(nil, []goipyObject.Object{&goipyObject.Bytes{V: cb.V}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	db, ok := decompressed.(*goipyObject.Bytes)
	if !ok {
		t.Fatalf("expected Bytes, got %T", decompressed)
	}
	if string(db.V) != original {
		t.Errorf("roundtrip failed: got %q", db.V)
	}
}

func TestNodeZlibDeflateInflate(t *testing.T) {
	mod := BuildNodeZlib(nil)

	deflateFn := mustGetBuiltin(t, mod.Dict, "deflate")
	inflateFn := mustGetBuiltin(t, mod.Dict, "inflate")

	original := "deflate me please"
	comp, err := deflateFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: original}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cb := comp.(*goipyObject.Bytes)

	decomp, err := inflateFn.Call(nil, []goipyObject.Object{&goipyObject.Bytes{V: cb.V}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	db := decomp.(*goipyObject.Bytes)
	if string(db.V) != original {
		t.Errorf("got %q", db.V)
	}
}

func TestNodeZlibSyncAliases(t *testing.T) {
	mod := BuildNodeZlib(nil)

	gzipFn := mustGetBuiltin(t, mod.Dict, "gzipSync")
	gunzipFn := mustGetBuiltin(t, mod.Dict, "gunzipSync")

	original := "sync test"
	comp, err := gzipFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: original}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cb := comp.(*goipyObject.Bytes)

	decomp, err := gunzipFn.Call(nil, []goipyObject.Object{&goipyObject.Bytes{V: cb.V}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	db := decomp.(*goipyObject.Bytes)
	if string(db.V) != original {
		t.Errorf("got %q", db.V)
	}
}

func TestNodeZlibCreateGzip(t *testing.T) {
	mod := BuildNodeZlib(nil)

	createFn := mustGetBuiltin(t, mod.Dict, "createGzip")
	transform, err := createFn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := transform.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", transform)
	}

	mustGetBuiltin(t, inst.Dict, "write").Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "compress this"},
	}, nil)

	flushFn := mustGetBuiltin(t, inst.Dict, "flush")
	res, err := flushFn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := res.(*goipyObject.Bytes)
	if !ok || len(b.V) == 0 {
		t.Error("expected compressed bytes from flush")
	}
}

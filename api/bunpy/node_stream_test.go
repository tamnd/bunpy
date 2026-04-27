package bunpy

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
)

func TestNodeStreamReadable(t *testing.T) {
	mod := BuildNodeStream(nil)
	fn := mustGetBuiltin(t, mod.Dict, "Readable")
	res, err := fn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := res.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", res)
	}
	if inst.Class.Name != "Readable" {
		t.Errorf("expected Readable, got %q", inst.Class.Name)
	}
}

func TestNodeStreamWritable(t *testing.T) {
	mod := BuildNodeStream(nil)
	fn := mustGetBuiltin(t, mod.Dict, "Writable")
	res, err := fn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := res.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", res)
	}

	writeFn := mustGetBuiltin(t, inst.Dict, "write")
	writeFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "hello"}}, nil)

	getContents := mustGetBuiltin(t, inst.Dict, "getContents")
	result, err := getContents.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := result.(*goipyObject.Bytes)
	if !ok || string(b.V) != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestNodeStreamPassThrough(t *testing.T) {
	mod := BuildNodeStream(nil)
	fn := mustGetBuiltin(t, mod.Dict, "PassThrough")
	res, err := fn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := res.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", res)
	}

	mustGetBuiltin(t, inst.Dict, "write").Call(nil, []goipyObject.Object{
		&goipyObject.Bytes{V: []byte("data")},
	}, nil)

	readResult, _ := mustGetBuiltin(t, inst.Dict, "read").Call(nil, nil, nil)
	b, ok := readResult.(*goipyObject.Bytes)
	if !ok || string(b.V) != "data" {
		t.Errorf("expected 'data', got %v", readResult)
	}
}

func TestNodeStreamReadablePush(t *testing.T) {
	r := newReadableInstance([]byte("initial"))
	mustGetBuiltin(t, r.Dict, "push").Call(nil, []goipyObject.Object{
		&goipyObject.Bytes{V: []byte(" more")},
	}, nil)
	readFn := mustGetBuiltin(t, r.Dict, "read")
	res, _ := readFn.Call(nil, nil, nil)
	b, ok := res.(*goipyObject.Bytes)
	if !ok {
		t.Fatalf("expected Bytes, got %T", res)
	}
	if string(b.V) != "initial more" {
		t.Errorf("got %q", b.V)
	}
}

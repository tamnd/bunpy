package bunpy

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
)

func TestNodeCryptoRandomBytes(t *testing.T) {
	mod := BuildNodeCrypto(nil)
	fn := mustGetBuiltin(t, mod.Dict, "randomBytes")
	res, err := fn.Call(nil, []goipyObject.Object{goipyObject.NewInt(32)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := res.(*goipyObject.Bytes)
	if !ok {
		t.Fatalf("expected Bytes, got %T", res)
	}
	if len(b.V) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(b.V))
	}
}

func TestNodeCryptoRandomUUID(t *testing.T) {
	mod := BuildNodeCrypto(nil)
	fn := mustGetBuiltin(t, mod.Dict, "randomUUID")
	res, err := fn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := res.(*goipyObject.Str)
	if !ok {
		t.Fatalf("expected Str, got %T", res)
	}
	parts := strings.Split(s.V, "-")
	if len(parts) != 5 {
		t.Errorf("UUID should have 5 parts, got: %q", s.V)
	}
}

func TestNodeCryptoCreateHashSHA256(t *testing.T) {
	mod := BuildNodeCrypto(nil)
	fn := mustGetBuiltin(t, mod.Dict, "createHash")
	hashObj, err := fn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "sha256"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := hashObj.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", hashObj)
	}
	updateFn := mustGetBuiltin(t, inst.Dict, "update")
	updateFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "hello"}}, nil)

	digestFn := mustGetBuiltin(t, inst.Dict, "digest")
	res, err := digestFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "hex"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := res.(*goipyObject.Str)
	if !ok {
		t.Fatalf("expected Str, got %T", res)
	}
	// SHA-256 of "hello"
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if s.V != want {
		t.Errorf("got %q, want %q", s.V, want)
	}
}

func TestNodeCryptoCreateHmac(t *testing.T) {
	mod := BuildNodeCrypto(nil)
	fn := mustGetBuiltin(t, mod.Dict, "createHmac")
	hmacObj, err := fn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "sha256"},
		&goipyObject.Str{V: "secret"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := hmacObj.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", hmacObj)
	}
	updateFn := mustGetBuiltin(t, inst.Dict, "update")
	updateFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "message"}}, nil)

	digestFn := mustGetBuiltin(t, inst.Dict, "digest")
	res, _ := digestFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "hex"}}, nil)
	s, ok := res.(*goipyObject.Str)
	if !ok || len(s.V) != 64 {
		t.Errorf("expected 64-char hex HMAC, got %v", res)
	}
}

func TestNodeCryptoHashSHA1(t *testing.T) {
	mod := BuildNodeCrypto(nil)
	fn := mustGetBuiltin(t, mod.Dict, "createHash")
	hashObj, _ := fn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "sha1"}}, nil)
	inst := hashObj.(*goipyObject.Instance)
	mustGetBuiltin(t, inst.Dict, "update").Call(nil, []goipyObject.Object{&goipyObject.Str{V: "abc"}}, nil)
	res, _ := mustGetBuiltin(t, inst.Dict, "digest").Call(nil, []goipyObject.Object{&goipyObject.Str{V: "hex"}}, nil)
	s := res.(*goipyObject.Str)
	if len(s.V) != 40 {
		t.Errorf("sha1 hex should be 40 chars, got %d: %q", len(s.V), s.V)
	}
}

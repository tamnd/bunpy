package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestCryptoRandom(t *testing.T) {
	mod := bunpyAPI.BuildCrypto(nil)
	fn, _ := mod.Dict.GetStr("random")
	result, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		goipyObject.NewInt(16),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := result.(*goipyObject.Bytes).V
	if len(b) != 16 {
		t.Fatalf("expected 16 bytes, got %d", len(b))
	}
	allZero := true
	for _, v := range b {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatal("random bytes are all zero (extremely unlikely, likely broken)")
	}
}

func TestCryptoEncryptDecryptRoundtrip(t *testing.T) {
	mod := bunpyAPI.BuildCrypto(nil)
	encFn, _ := mod.Dict.GetStr("encrypt")
	decFn, _ := mod.Dict.GetStr("decrypt")

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	keyObj := &goipyObject.Bytes{V: key}
	pt := &goipyObject.Bytes{V: []byte("hello world")}

	ct, err := encFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{pt, keyObj}, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := decFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{ct, keyObj}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(result.(*goipyObject.Bytes).V) != "hello world" {
		t.Fatalf("roundtrip failed: got %q", result.(*goipyObject.Bytes).V)
	}
}

func TestCryptoDecryptWrongKey(t *testing.T) {
	mod := bunpyAPI.BuildCrypto(nil)
	encFn, _ := mod.Dict.GetStr("encrypt")
	decFn, _ := mod.Dict.GetStr("decrypt")

	key := make([]byte, 32)
	wrongKey := make([]byte, 32)
	wrongKey[0] = 0xff

	ct, _ := encFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Bytes{V: []byte("secret")},
		&goipyObject.Bytes{V: key},
	}, nil)
	_, err := decFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		ct,
		&goipyObject.Bytes{V: wrongKey},
	}, nil)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestCryptoHMAC(t *testing.T) {
	mod := bunpyAPI.BuildCrypto(nil)
	fn, _ := mod.Dict.GetStr("hmac")
	result, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "message"},
		&goipyObject.Str{V: "key"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := result.(*goipyObject.Bytes).V
	if len(b) != 32 {
		t.Fatalf("expected 32 byte HMAC, got %d", len(b))
	}
}

func TestCryptoHMACVerify(t *testing.T) {
	mod := bunpyAPI.BuildCrypto(nil)
	hmacFn, _ := mod.Dict.GetStr("hmac")
	verifyFn, _ := mod.Dict.GetStr("hmac_verify")

	sig, _ := hmacFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "msg"},
		&goipyObject.Str{V: "key"},
	}, nil)

	ok, err := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "msg"},
		&goipyObject.Str{V: "key"},
		sig,
	}, nil)
	if err != nil || !ok.(*goipyObject.Bool).V {
		t.Fatal("hmac_verify should return True for correct signature")
	}

	// tampered sig
	tampered := make([]byte, len(sig.(*goipyObject.Bytes).V))
	copy(tampered, sig.(*goipyObject.Bytes).V)
	tampered[0] ^= 0xff
	bad, _ := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "msg"},
		&goipyObject.Str{V: "key"},
		&goipyObject.Bytes{V: tampered},
	}, nil)
	if bad.(*goipyObject.Bool).V {
		t.Fatal("hmac_verify should return False for tampered signature")
	}
}

func TestCryptoSHA256Hex(t *testing.T) {
	mod := bunpyAPI.BuildCrypto(nil)
	fn, _ := mod.Dict.GetStr("sha256_hex")
	result, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Bytes{V: []byte{}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if !strings.HasPrefix(result.(*goipyObject.Str).V, expected[:16]) {
		t.Fatalf("SHA256 of empty bytes mismatch: got %q", result.(*goipyObject.Str).V)
	}
}

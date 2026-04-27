package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func jwtMod() *goipyObject.Module { return bunpyAPI.BuildJWT(nil) }

func TestJWTSignHasTwoDots(t *testing.T) {
	mod := jwtMod()
	signFn, _ := mod.Dict.GetStr("sign")
	claims := goipyObject.NewDict()
	claims.SetStr("sub", &goipyObject.Str{V: "user:1"})
	result, err := signFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		claims,
		&goipyObject.Str{V: "secret"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	token := result.(*goipyObject.Str).V
	if strings.Count(token, ".") != 2 {
		t.Fatalf("expected two dots in JWT, got %q", token)
	}
}

func TestJWTRoundtrip(t *testing.T) {
	mod := jwtMod()
	signFn, _ := mod.Dict.GetStr("sign")
	verifyFn, _ := mod.Dict.GetStr("verify")

	claims := goipyObject.NewDict()
	claims.SetStr("sub", &goipyObject.Str{V: "user:42"})
	token, _ := signFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		claims,
		&goipyObject.Str{V: "my-secret"},
	}, nil)

	payload, err := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		token,
		&goipyObject.Str{V: "my-secret"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := payload.(*goipyObject.Dict)
	sub, ok := d.GetStr("sub")
	if !ok {
		t.Fatal("sub missing from verified payload")
	}
	if sub.(*goipyObject.Str).V != "user:42" {
		t.Fatalf("expected sub=user:42, got %v", sub)
	}
}

func TestJWTWrongSecretFails(t *testing.T) {
	mod := jwtMod()
	signFn, _ := mod.Dict.GetStr("sign")
	verifyFn, _ := mod.Dict.GetStr("verify")

	claims := goipyObject.NewDict()
	claims.SetStr("x", &goipyObject.Str{V: "y"})
	token, _ := signFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		claims,
		&goipyObject.Str{V: "secret"},
	}, nil)

	_, err := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		token,
		&goipyObject.Str{V: "wrong-secret"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestJWTExpired(t *testing.T) {
	mod := jwtMod()
	signFn, _ := mod.Dict.GetStr("sign")
	verifyFn, _ := mod.Dict.GetStr("verify")

	claims := goipyObject.NewDict()
	// exp=-1 means the token expired 1 second ago
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("exp", goipyObject.NewInt(-1))
	token, _ := signFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		claims,
		&goipyObject.Str{V: "s"},
	}, kwargs)

	_, err := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		token,
		&goipyObject.Str{V: "s"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestJWTDecodeNoVerify(t *testing.T) {
	mod := jwtMod()
	signFn, _ := mod.Dict.GetStr("sign")
	decodeFn, _ := mod.Dict.GetStr("decode")

	claims := goipyObject.NewDict()
	claims.SetStr("role", &goipyObject.Str{V: "admin"})
	token, _ := signFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		claims,
		&goipyObject.Str{V: "secret"},
	}, nil)

	payload, err := decodeFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		token,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := payload.(*goipyObject.Dict)
	role, ok := d.GetStr("role")
	if !ok || role.(*goipyObject.Str).V != "admin" {
		t.Fatal("expected role=admin from decoded token")
	}
}

func TestJWTTamperedPayloadFails(t *testing.T) {
	mod := jwtMod()
	signFn, _ := mod.Dict.GetStr("sign")
	verifyFn, _ := mod.Dict.GetStr("verify")

	claims := goipyObject.NewDict()
	claims.SetStr("sub", &goipyObject.Str{V: "user"})
	result, _ := signFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		claims, &goipyObject.Str{V: "s"},
	}, nil)
	token := result.(*goipyObject.Str).V

	// Flip a byte in the payload part (index 1)
	parts := strings.SplitN(token, ".", 3)
	payload := []byte(parts[1])
	payload[0] ^= 0x01
	tampered := parts[0] + "." + string(payload) + "." + parts[2]

	_, err := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: tampered},
		&goipyObject.Str{V: "s"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

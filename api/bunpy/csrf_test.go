package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func csrfMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildCSRF(nil)
}

func TestCSRFModuleMethods(t *testing.T) {
	mod := csrfMod(t)
	for _, name := range []string{"token", "verify"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("csrf module missing %q", name)
		}
	}
}

func TestCSRFTokenNoSecret(t *testing.T) {
	mod := csrfMod(t)
	fn, _ := mod.Dict.GetStr("token")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	tok := r.(*goipyObject.Str).V
	if len(tok) < 10 {
		t.Errorf("expected non-trivial token, got %q", tok)
	}
}

func TestCSRFTokenWithSecret(t *testing.T) {
	mod := csrfMod(t)
	fn, _ := mod.Dict.GetStr("token")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "my-secret"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tok := r.(*goipyObject.Str).V
	if !strings.Contains(tok, ".") {
		t.Errorf("signed token should contain a dot, got %q", tok)
	}
}

func TestCSRFVerifyValid(t *testing.T) {
	mod := csrfMod(t)
	tokenFn, _ := mod.Dict.GetStr("token")
	verifyFn, _ := mod.Dict.GetStr("verify")
	secret := &goipyObject.Str{V: "supersecret"}

	tokObj, _ := tokenFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{secret}, nil)
	r, err := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{tokObj, secret}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !r.(*goipyObject.Bool).V {
		t.Error("verify should return true for valid token")
	}
}

func TestCSRFVerifyInvalid(t *testing.T) {
	mod := csrfMod(t)
	verifyFn, _ := mod.Dict.GetStr("verify")
	r, err := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "fakeanonce.fakesig"},
		&goipyObject.Str{V: "supersecret"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.(*goipyObject.Bool).V {
		t.Error("verify should return false for tampered token")
	}
}

func TestCSRFVerifyUnsigned(t *testing.T) {
	mod := csrfMod(t)
	verifyFn, _ := mod.Dict.GetStr("verify")
	// token without dot, no secret
	r, err := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "sometoken"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !r.(*goipyObject.Bool).V {
		t.Error("non-empty token with no secret should be considered valid")
	}
}

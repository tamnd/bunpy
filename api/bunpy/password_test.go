package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestBcryptHashAndVerify(t *testing.T) {
	i := serveInterp(t)
	m := bunpyAPI.BuildPassword(i)

	hashFn, _ := m.Dict.GetStr("hash")
	result, err := hashFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "hunter2"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	hash := result.(*goipyObject.Str).V
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("bcrypt hash should start with $2, got %q", hash)
	}

	verifyFn, _ := m.Dict.GetStr("verify")
	ok, err2 := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "hunter2"},
		&goipyObject.Str{V: hash},
	}, nil)
	if err2 != nil {
		t.Fatal(err2)
	}
	if !ok.(*goipyObject.Bool).V {
		t.Fatal("verify should return True for correct password")
	}
}

func TestBcryptVerifyWrongPassword(t *testing.T) {
	i := serveInterp(t)
	m := bunpyAPI.BuildPassword(i)

	hashFn, _ := m.Dict.GetStr("hash")
	result, _ := hashFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "secret"},
	}, nil)
	hash := result.(*goipyObject.Str).V

	verifyFn, _ := m.Dict.GetStr("verify")
	ok, _ := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "wrong"},
		&goipyObject.Str{V: hash},
	}, nil)
	if ok.(*goipyObject.Bool).V {
		t.Fatal("verify should return False for wrong password")
	}
}

func TestArgon2idHashAndVerify(t *testing.T) {
	i := serveInterp(t)
	m := bunpyAPI.BuildPassword(i)

	kw := goipyObject.NewDict()
	kw.SetStr("algo", &goipyObject.Str{V: "argon2id"})
	kw.SetStr("memory", goipyObject.NewInt(16*1024)) // 16 MiB for fast test
	kw.SetStr("time", goipyObject.NewInt(1))
	kw.SetStr("threads", goipyObject.NewInt(1))

	hashFn, _ := m.Dict.GetStr("hash")
	result, err := hashFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "passw0rd"},
	}, kw)
	if err != nil {
		t.Fatal(err)
	}
	hash := result.(*goipyObject.Str).V
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("argon2id hash should start with $argon2id$, got %q", hash)
	}

	verifyFn, _ := m.Dict.GetStr("verify")
	ok, err2 := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "passw0rd"},
		&goipyObject.Str{V: hash},
	}, nil)
	if err2 != nil {
		t.Fatal(err2)
	}
	if !ok.(*goipyObject.Bool).V {
		t.Fatal("verify should return True for correct argon2id password")
	}
}

func TestArgon2idVerifyWrong(t *testing.T) {
	i := serveInterp(t)
	m := bunpyAPI.BuildPassword(i)

	kw := goipyObject.NewDict()
	kw.SetStr("algo", &goipyObject.Str{V: "argon2id"})
	kw.SetStr("memory", goipyObject.NewInt(16*1024))
	kw.SetStr("time", goipyObject.NewInt(1))
	kw.SetStr("threads", goipyObject.NewInt(1))

	hashFn, _ := m.Dict.GetStr("hash")
	result, _ := hashFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "correct"},
	}, kw)
	hash := result.(*goipyObject.Str).V

	verifyFn, _ := m.Dict.GetStr("verify")
	ok, _ := verifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "wrong"},
		&goipyObject.Str{V: hash},
	}, nil)
	if ok.(*goipyObject.Bool).V {
		t.Fatal("verify should return False for wrong argon2id password")
	}
}

func TestPasswordModuleHasHashVerify(t *testing.T) {
	i := serveInterp(t)
	m := bunpyAPI.BuildPassword(i)
	for _, name := range []string{"hash", "verify"} {
		if _, ok := m.Dict.GetStr(name); !ok {
			t.Fatalf("bunpy.password.%s missing", name)
		}
	}
}

func TestBunpyModuleHasPassword(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	if _, ok := m.Dict.GetStr("password"); !ok {
		t.Fatal("bunpy.password missing from top-level module")
	}
}

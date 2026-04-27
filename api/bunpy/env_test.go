package bunpy_test

import (
	"os"
	"path/filepath"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestEnvLoad(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildEnv(i)

	tmp := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(tmp, []byte("TEST_ENV_LOAD=hello\n# comment\n\nTEST_ENV_QUOTED=\"world\"\n"), 0o644)

	loadFn, _ := mod.Dict.GetStr("load")
	_, err := loadFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: tmp},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("TEST_ENV_LOAD") != "hello" {
		t.Fatalf("expected TEST_ENV_LOAD=hello, got %q", os.Getenv("TEST_ENV_LOAD"))
	}
	if os.Getenv("TEST_ENV_QUOTED") != "world" {
		t.Fatalf("expected TEST_ENV_QUOTED=world, got %q", os.Getenv("TEST_ENV_QUOTED"))
	}
}

func TestEnvGetMissing(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildEnv(i)
	os.Unsetenv("_BUNPY_NO_SUCH_VAR")

	getFn, _ := mod.Dict.GetStr("get")
	result, err := getFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "_BUNPY_NO_SUCH_VAR"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*goipyObject.NoneType); !ok {
		t.Fatalf("expected None, got %T", result)
	}
}

func TestEnvGetDefault(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildEnv(i)
	os.Unsetenv("_BUNPY_NO_SUCH_VAR")

	getFn, _ := mod.Dict.GetStr("get")
	result, err := getFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "_BUNPY_NO_SUCH_VAR"},
		&goipyObject.Str{V: "default"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := result.(*goipyObject.Str)
	if !ok || s.V != "default" {
		t.Fatalf("expected 'default', got %v", result)
	}
}

func TestEnvIntDefault(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildEnv(i)
	os.Unsetenv("_BUNPY_INT_VAR")

	intFn, _ := mod.Dict.GetStr("int")
	result, err := intFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "_BUNPY_INT_VAR"},
		goipyObject.NewInt(42),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != goipyObject.NewInt(42) {
		t.Fatalf("expected 42, got %v", result)
	}
}

func TestEnvBoolParsing(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildEnv(i)

	boolFn, _ := mod.Dict.GetStr("bool")

	for _, v := range []string{"true", "1", "yes", "on"} {
		os.Setenv("_BUNPY_BOOL_VAR", v)
		result, err := boolFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
			&goipyObject.Str{V: "_BUNPY_BOOL_VAR"},
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		b, ok := result.(*goipyObject.Bool)
		if !ok || !b.V {
			t.Fatalf("expected true for %q, got %v", v, result)
		}
	}

	os.Setenv("_BUNPY_BOOL_VAR", "false")
	result, _ := boolFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "_BUNPY_BOOL_VAR"},
	}, nil)
	b, ok := result.(*goipyObject.Bool)
	if !ok || b.V {
		t.Fatal("expected false")
	}
}

func TestEnvAll(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildEnv(i)

	allFn, _ := mod.Dict.GetStr("all")
	result, err := allFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := result.(*goipyObject.Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", result)
	}
	if _, ok := d.GetStr("PATH"); !ok {
		t.Fatal("expected PATH in env.all()")
	}
}

func TestEnvCommentIgnored(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildEnv(i)

	tmp := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(tmp, []byte("# this is a comment\nREAL_VAR=yes\n"), 0o644)

	loadFn, _ := mod.Dict.GetStr("load")
	loadFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: tmp},
	}, nil)

	if os.Getenv("REAL_VAR") != "yes" {
		t.Fatal("expected REAL_VAR=yes after loading dotenv")
	}
}

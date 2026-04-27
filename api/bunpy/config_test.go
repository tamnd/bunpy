package bunpy_test

import (
	"os"
	"path/filepath"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func configLoad(t *testing.T, content, ext string) *goipyObject.Instance {
	t.Helper()
	i := serveInterp(t)
	mod := bunpyAPI.BuildConfig(i)
	loadFn, _ := mod.Dict.GetStr("load")
	tmp := filepath.Join(t.TempDir(), "config"+ext)
	os.WriteFile(tmp, []byte(content), 0o644)
	result, err := loadFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: tmp},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result.(*goipyObject.Instance)
}

func cfgGet(t *testing.T, inst *goipyObject.Instance, method string, key string, def goipyObject.Object) goipyObject.Object {
	t.Helper()
	fn, _ := inst.Dict.GetStr(method)
	args := []goipyObject.Object{&goipyObject.Str{V: key}}
	if def != nil {
		args = append(args, def)
	}
	result, err := fn.(*goipyObject.BuiltinFunc).Call(nil, args, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestConfigLoadTOML(t *testing.T) {
	cfg := configLoad(t, "[server]\nport = 9090\nhost = \"localhost\"\n", ".toml")
	v := cfgGet(t, cfg, "int", "server.port", nil)
	if v.(*goipyObject.Int).Int64() != 9090 {
		t.Fatalf("expected 9090, got %v", v)
	}
}

func TestConfigGetMissingReturnsNone(t *testing.T) {
	cfg := configLoad(t, "[app]\nname = \"test\"\n", ".toml")
	v := cfgGet(t, cfg, "get", "no.such.key", nil)
	if _, ok := v.(*goipyObject.NoneType); !ok {
		t.Fatalf("expected None for missing key, got %T", v)
	}
}

func TestConfigGetDefault(t *testing.T) {
	cfg := configLoad(t, "{}", ".json")
	v := cfgGet(t, cfg, "int", "port", goipyObject.NewInt(8080))
	if v.(*goipyObject.Int).Int64() != 8080 {
		t.Fatalf("expected default 8080, got %v", v)
	}
}

func TestConfigBool(t *testing.T) {
	cfg := configLoad(t, "[app]\ndebug = true\n", ".toml")
	v := cfgGet(t, cfg, "bool", "app.debug", nil)
	if !v.(*goipyObject.Bool).V {
		t.Fatal("expected true for app.debug")
	}
}

func TestConfigMergeSources(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildConfig(i)
	loadFn, _ := mod.Dict.GetStr("load")

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.toml"), []byte("[server]\nport=8080\nhost=\"old\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.toml"), []byte("[server]\nhost=\"new\"\n"), 0o644)

	result, err := loadFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: filepath.Join(dir, "a.toml")},
		&goipyObject.Str{V: filepath.Join(dir, "b.toml")},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg := result.(*goipyObject.Instance)
	host := cfgGet(t, cfg, "get", "server.host", nil)
	if host.(*goipyObject.Str).V != "new" {
		t.Fatalf("expected second source to win, got %v", host)
	}
	port := cfgGet(t, cfg, "int", "server.port", nil)
	if port.(*goipyObject.Int).Int64() != 8080 {
		t.Fatalf("expected port from first source, got %v", port)
	}
}

func TestConfigEnvPrefix(t *testing.T) {
	os.Setenv("MYAPP_SERVER_PORT", "9999")
	defer os.Unsetenv("MYAPP_SERVER_PORT")

	i := serveInterp(t)
	mod := bunpyAPI.BuildConfig(i)
	loadFn, _ := mod.Dict.GetStr("load")

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "c.toml"), []byte("[server]\nport=8080\n"), 0o644)

	kwargs := goipyObject.NewDict()
	kwargs.SetStr("env_prefix", &goipyObject.Str{V: "MYAPP"})
	result, err := loadFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: filepath.Join(dir, "c.toml")},
	}, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	cfg := result.(*goipyObject.Instance)
	port := cfgGet(t, cfg, "get", "server.port", nil)
	if port.(*goipyObject.Str).V != "9999" {
		t.Fatalf("expected env override 9999, got %v", port)
	}
}

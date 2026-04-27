package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestShellEcho(t *testing.T) {
	i := serveInterp(t)
	shellFn := bunpyAPI.BuildShell(i)
	result, err := shellFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "echo hello"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)
	stdout, _ := inst.Dict.GetStr("stdout")
	got := stdout.(*goipyObject.Str).V
	if strings.TrimSpace(got) != "hello" {
		t.Fatalf("stdout = %q, want %q", got, "hello")
	}
}

func TestShellExitCode(t *testing.T) {
	i := serveInterp(t)
	shellFn := bunpyAPI.BuildShell(i)
	result, err := shellFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "exit 42"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)
	ec, _ := inst.Dict.GetStr("exitcode")
	if ec.(*goipyObject.Int).Int64() != 42 {
		t.Fatalf("exitcode = %d, want 42", ec.(*goipyObject.Int).Int64())
	}
}

func TestShellCwd(t *testing.T) {
	i := serveInterp(t)
	shellFn := bunpyAPI.BuildShell(i)
	kw := goipyObject.NewDict()
	kw.SetStr("cwd", &goipyObject.Str{V: "/tmp"})
	result, err := shellFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "pwd"}}, kw)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)
	stdout, _ := inst.Dict.GetStr("stdout")
	got := strings.TrimSpace(stdout.(*goipyObject.Str).V)
	// /tmp may resolve to /private/tmp on macOS
	if got != "/tmp" && got != "/private/tmp" {
		t.Fatalf("cwd stdout = %q", got)
	}
}

func TestSpawnEcho(t *testing.T) {
	i := serveInterp(t)
	spawnFn := bunpyAPI.BuildSpawn(i)
	argv := &goipyObject.List{V: []goipyObject.Object{
		&goipyObject.Str{V: "echo"},
		&goipyObject.Str{V: "hi"},
	}}
	result, err := spawnFn.Call(nil, []goipyObject.Object{argv}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)
	waitFn, _ := inst.Dict.GetStr("wait")
	waitFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	stdout, _ := inst.Dict.GetStr("stdout")
	got := strings.TrimSpace(stdout.(*goipyObject.Str).V)
	if got != "hi" {
		t.Fatalf("spawn stdout = %q, want %q", got, "hi")
	}
}

func TestDollarTemplate(t *testing.T) {
	i := serveInterp(t)
	dollarFn := bunpyAPI.BuildDollar(i)
	kw := goipyObject.NewDict()
	kw.SetStr("name", &goipyObject.Str{V: "world"})
	result, err := dollarFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "echo {name}"}}, kw)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)
	stdout, _ := inst.Dict.GetStr("stdout")
	got := strings.TrimSpace(stdout.(*goipyObject.Str).V)
	if got != "world" {
		t.Fatalf("dollar stdout = %q, want %q", got, "world")
	}
}

func TestDollarQuotesSpecialChars(t *testing.T) {
	i := serveInterp(t)
	dollarFn := bunpyAPI.BuildDollar(i)
	kw := goipyObject.NewDict()
	kw.SetStr("msg", &goipyObject.Str{V: "hello world"})
	result, err := dollarFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: "echo {msg}"}}, kw)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)
	stdout, _ := inst.Dict.GetStr("stdout")
	got := strings.TrimSpace(stdout.(*goipyObject.Str).V)
	if got != "hello world" {
		t.Fatalf("dollar stdout = %q, want %q", got, "hello world")
	}
}

func TestBunpyModuleHasShellSpawnDollar(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	for _, name := range []string{"shell", "spawn", "dollar"} {
		if _, ok := m.Dict.GetStr(name); !ok {
			t.Fatalf("bunpy.%s missing from top-level module", name)
		}
	}
}

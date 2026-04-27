package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func terminalMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildTerminal(nil)
}

func TestTerminalModuleMethods(t *testing.T) {
	mod := terminalMod(t)
	for _, name := range []string{"style", "strip", "columns", "rows", "is_tty", "red", "green", "bold"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("terminal module missing %q", name)
		}
	}
}

func TestTerminalStyle(t *testing.T) {
	mod := terminalMod(t)
	fn, _ := mod.Dict.GetStr("style")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "hello"},
		&goipyObject.Str{V: "red"},
		&goipyObject.Str{V: "bold"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.(*goipyObject.Str).V
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("expected ANSI codes in output %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("expected 'hello' in output %q", got)
	}
}

func TestTerminalStripRemovesANSI(t *testing.T) {
	mod := terminalMod(t)
	styleFn, _ := mod.Dict.GetStr("red")
	stripFn, _ := mod.Dict.GetStr("strip")

	styled, _ := styleFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "hello"},
	}, nil)
	r, err := stripFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{styled}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.(*goipyObject.Str).V
	if got != "hello" {
		t.Errorf("strip: expected 'hello', got %q", got)
	}
}

func TestTerminalColumnsAndRows(t *testing.T) {
	mod := terminalMod(t)
	colFn, _ := mod.Dict.GetStr("columns")
	rowFn, _ := mod.Dict.GetStr("rows")
	colR, _ := colFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	rowR, _ := rowFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if colR.(*goipyObject.Int).Int64() <= 0 {
		t.Error("columns should be positive")
	}
	if rowR.(*goipyObject.Int).Int64() <= 0 {
		t.Error("rows should be positive")
	}
}

func TestTerminalIsTTY(t *testing.T) {
	mod := terminalMod(t)
	fn, _ := mod.Dict.GetStr("is_tty")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.(*goipyObject.Bool); !ok {
		t.Fatalf("is_tty should return bool, got %T", r)
	}
}

func TestTerminalConvenienceFunctions(t *testing.T) {
	mod := terminalMod(t)
	for _, name := range []string{"red", "green", "blue", "bold", "underline"} {
		fn, ok := mod.Dict.GetStr(name)
		if !ok {
			t.Fatalf("terminal.%s not found", name)
		}
		r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
			&goipyObject.Str{V: "text"},
		}, nil)
		if err != nil {
			t.Fatalf("terminal.%s error: %v", name, err)
		}
		if !strings.Contains(r.(*goipyObject.Str).V, "text") {
			t.Errorf("terminal.%s output should contain 'text'", name)
		}
	}
}

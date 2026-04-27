package bunpy

import (
	"runtime"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
)

func TestNodeOSPlatform(t *testing.T) {
	mod := BuildNodeOS(nil)
	fn := mustGetBuiltin(t, mod.Dict, "platform")
	res, err := fn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := res.(*goipyObject.Str)
	if !ok {
		t.Fatalf("expected Str, got %T", res)
	}
	if s.V != runtime.GOOS {
		t.Errorf("expected %q, got %q", runtime.GOOS, s.V)
	}
}

func TestNodeOSArch(t *testing.T) {
	mod := BuildNodeOS(nil)
	fn := mustGetBuiltin(t, mod.Dict, "arch")
	res, _ := fn.Call(nil, nil, nil)
	s, ok := res.(*goipyObject.Str)
	if !ok {
		t.Fatalf("expected Str, got %T", res)
	}
	if s.V == "" {
		t.Error("arch should not be empty")
	}
}

func TestNodeOSHomedir(t *testing.T) {
	mod := BuildNodeOS(nil)
	fn := mustGetBuiltin(t, mod.Dict, "homedir")
	res, _ := fn.Call(nil, nil, nil)
	if _, ok := res.(*goipyObject.Str); !ok {
		t.Fatalf("expected Str, got %T", res)
	}
}

func TestNodeOSTmpdir(t *testing.T) {
	mod := BuildNodeOS(nil)
	fn := mustGetBuiltin(t, mod.Dict, "tmpdir")
	res, _ := fn.Call(nil, nil, nil)
	s, ok := res.(*goipyObject.Str)
	if !ok || s.V == "" {
		t.Error("tmpdir should return non-empty string")
	}
}

func TestNodeOSCPUs(t *testing.T) {
	mod := BuildNodeOS(nil)
	fn := mustGetBuiltin(t, mod.Dict, "cpus")
	res, err := fn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	lst, ok := res.(*goipyObject.List)
	if !ok {
		t.Fatalf("expected List, got %T", res)
	}
	if len(lst.V) < 1 {
		t.Error("expected at least 1 CPU")
	}
}

func TestNodeOSUptime(t *testing.T) {
	mod := BuildNodeOS(nil)
	fn := mustGetBuiltin(t, mod.Dict, "uptime")
	res, _ := fn.Call(nil, nil, nil)
	if _, ok := res.(*goipyObject.Float); !ok {
		t.Fatalf("expected Float, got %T", res)
	}
}

func TestNodeOSNetworkInterfaces(t *testing.T) {
	mod := BuildNodeOS(nil)
	fn := mustGetBuiltin(t, mod.Dict, "networkInterfaces")
	res, err := fn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.(*goipyObject.Dict); !ok {
		t.Fatalf("expected Dict, got %T", res)
	}
}

func TestNodeOSEOL(t *testing.T) {
	mod := BuildNodeOS(nil)
	eol, ok := mod.Dict.GetStr("EOL")
	if !ok {
		t.Fatal("EOL not found")
	}
	s, ok := eol.(*goipyObject.Str)
	if !ok {
		t.Fatalf("expected Str, got %T", eol)
	}
	if runtime.GOOS == "windows" {
		if s.V != "\r\n" {
			t.Errorf("expected CRLF on windows, got %q", s.V)
		}
	} else {
		if s.V != "\n" {
			t.Errorf("expected LF, got %q", s.V)
		}
	}
}

package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func dnsInterp(t *testing.T) *goipyObject.Module {
	t.Helper()
	i := serveInterp(t)
	return bunpyAPI.BuildDNS(i)
}

func TestDNSModuleMethods(t *testing.T) {
	mod := dnsInterp(t)
	for _, name := range []string{"resolve", "lookup", "reverse"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("dns module missing %q", name)
		}
	}
}

func TestDNSLookupLocalhost(t *testing.T) {
	mod := dnsInterp(t)
	fn, _ := mod.Dict.GetStr("lookup")
	result, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "localhost"},
	}, nil)
	if err != nil {
		t.Fatalf("dns.lookup(localhost): %v", err)
	}
	if result == goipyObject.None {
		t.Skip("localhost did not resolve in this environment")
	}
	if _, ok := result.(*goipyObject.Str); !ok {
		t.Fatalf("expected str, got %T", result)
	}
}

func TestDNSLookupRequiresArg(t *testing.T) {
	mod := dnsInterp(t)
	fn, _ := mod.Dict.GetStr("lookup")
	_, err := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestDNSResolveRequiresArg(t *testing.T) {
	mod := dnsInterp(t)
	fn, _ := mod.Dict.GetStr("resolve")
	_, err := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestDNSResolveUnsupportedType(t *testing.T) {
	mod := dnsInterp(t)
	fn, _ := mod.Dict.GetStr("resolve")
	_, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "localhost"},
		&goipyObject.Str{V: "BOGUS"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported record type")
	}
}

func TestDNSResolveALocalhost(t *testing.T) {
	mod := dnsInterp(t)
	fn, _ := mod.Dict.GetStr("resolve")
	result, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "localhost"},
	}, nil)
	if err != nil {
		t.Skipf("dns.resolve(localhost) error (may be env): %v", err)
	}
	lst, ok := result.(*goipyObject.List)
	if !ok {
		t.Fatalf("expected list, got %T", result)
	}
	if len(lst.V) == 0 {
		t.Fatal("expected at least one address")
	}
}

func TestDNSReverseRequiresArg(t *testing.T) {
	mod := dnsInterp(t)
	fn, _ := mod.Dict.GetStr("reverse")
	_, err := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing arg")
	}
}

func TestDNSResolveTypeKwarg(t *testing.T) {
	mod := dnsInterp(t)
	fn, _ := mod.Dict.GetStr("resolve")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("type", &goipyObject.Str{V: "BOGUS"})
	_, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "localhost"},
	}, kwargs)
	if err == nil {
		t.Fatal("expected error for unsupported type via kwarg")
	}
}

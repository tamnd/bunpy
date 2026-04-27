package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func yamlMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildYAML(nil)
}

func yamlParse(t *testing.T, src string) goipyObject.Object {
	t.Helper()
	mod := yamlMod(t)
	fn, _ := mod.Dict.GetStr("parse")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: src},
	}, nil)
	if err != nil {
		t.Fatalf("yaml.parse error: %v", err)
	}
	return r
}

func TestYAMLModuleMethods(t *testing.T) {
	mod := yamlMod(t)
	for _, name := range []string{"parse", "stringify"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("yaml module missing %q", name)
		}
	}
}

func TestYAMLParseScalar(t *testing.T) {
	r := yamlParse(t, "hello")
	if r.(*goipyObject.Str).V != "hello" {
		t.Errorf("expected 'hello', got %v", r)
	}
}

func TestYAMLParseMapping(t *testing.T) {
	src := "name: alice\nage: 30\n"
	r := yamlParse(t, src)
	d, ok := r.(*goipyObject.Dict)
	if !ok {
		t.Fatalf("expected dict, got %T", r)
	}
	name, _ := d.GetStr("name")
	if name.(*goipyObject.Str).V != "alice" {
		t.Errorf("name: expected alice, got %v", name)
	}
	age, _ := d.GetStr("age")
	if age.(*goipyObject.Int).Int64() != 30 {
		t.Errorf("age: expected 30, got %v", age)
	}
}

func TestYAMLParseSequence(t *testing.T) {
	src := "- apple\n- banana\n- cherry\n"
	r := yamlParse(t, src)
	lst, ok := r.(*goipyObject.List)
	if !ok {
		t.Fatalf("expected list, got %T", r)
	}
	if len(lst.V) != 3 {
		t.Fatalf("expected 3 items, got %d", len(lst.V))
	}
	if lst.V[0].(*goipyObject.Str).V != "apple" {
		t.Errorf("expected apple, got %v", lst.V[0])
	}
}

func TestYAMLParseBooleans(t *testing.T) {
	src := "a: true\nb: false\nc: yes\nd: no\n"
	r := yamlParse(t, src)
	d := r.(*goipyObject.Dict)
	a, _ := d.GetStr("a")
	if !a.(*goipyObject.Bool).V {
		t.Error("a should be true")
	}
	b, _ := d.GetStr("b")
	if b.(*goipyObject.Bool).V {
		t.Error("b should be false")
	}
}

func TestYAMLParseNested(t *testing.T) {
	src := "server:\n  host: localhost\n  port: 8080\n"
	r := yamlParse(t, src)
	d := r.(*goipyObject.Dict)
	server, _ := d.GetStr("server")
	sd := server.(*goipyObject.Dict)
	host, _ := sd.GetStr("host")
	if host.(*goipyObject.Str).V != "localhost" {
		t.Errorf("expected localhost, got %v", host)
	}
	port, _ := sd.GetStr("port")
	if port.(*goipyObject.Int).Int64() != 8080 {
		t.Errorf("expected 8080, got %v", port)
	}
}

func TestYAMLStringifyAndReparse(t *testing.T) {
	mod := yamlMod(t)
	stringifyFn, _ := mod.Dict.GetStr("stringify")
	parseFn, _ := mod.Dict.GetStr("parse")

	d := goipyObject.NewDict()
	d.SetStr("name", &goipyObject.Str{V: "bob"})
	d.SetStr("score", goipyObject.NewInt(42))

	r, err := stringifyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{d}, nil)
	if err != nil {
		t.Fatal(err)
	}
	yaml := r.(*goipyObject.Str).V
	if yaml == "" {
		t.Fatal("stringify returned empty string")
	}

	r2, err := parseFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: yaml},
	}, nil)
	if err != nil {
		t.Fatalf("re-parse failed: %v (yaml was: %q)", err, yaml)
	}
	d2, ok := r2.(*goipyObject.Dict)
	if !ok {
		t.Fatalf("expected dict after re-parse, got %T", r2)
	}
	name, _ := d2.GetStr("name")
	if name.(*goipyObject.Str).V != "bob" {
		t.Errorf("name: expected bob after round-trip, got %v", name)
	}
}

func TestYAMLParseNull(t *testing.T) {
	src := "key: null\n"
	r := yamlParse(t, src)
	d := r.(*goipyObject.Dict)
	v, _ := d.GetStr("key")
	if v != goipyObject.None {
		t.Errorf("expected None for null, got %T", v)
	}
}

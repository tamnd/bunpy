package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func tmplMod() *goipyObject.Module { return bunpyAPI.BuildTemplate(nil) }

func TestTemplateRenderSimple(t *testing.T) {
	mod := tmplMod()
	renderFn, _ := mod.Dict.GetStr("render")
	data := goipyObject.NewDict()
	data.SetStr("name", &goipyObject.Str{V: "World"})
	result, err := renderFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "Hello {{ .name }}!"},
		data,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.(*goipyObject.Str).V != "Hello World!" {
		t.Fatalf("unexpected output: %q", result.(*goipyObject.Str).V)
	}
}

func TestTemplateRenderRange(t *testing.T) {
	mod := tmplMod()
	renderFn, _ := mod.Dict.GetStr("render")
	data := goipyObject.NewDict()
	items := &goipyObject.List{V: []goipyObject.Object{
		&goipyObject.Str{V: "a"},
		&goipyObject.Str{V: "b"},
	}}
	data.SetStr("items", items)
	result, err := renderFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "{{ range .items }}{{ . }}{{ end }}"},
		data,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.(*goipyObject.Str).V != "ab" {
		t.Fatalf("unexpected output: %q", result.(*goipyObject.Str).V)
	}
}

func TestTemplateHTMLEscapes(t *testing.T) {
	mod := tmplMod()
	renderFn, _ := mod.Dict.GetStr("render")
	data := goipyObject.NewDict()
	data.SetStr("name", &goipyObject.Str{V: "<script>"})
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("html", goipyObject.BoolOf(true))
	result, err := renderFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "Hi {{ .name }}"},
		data,
	}, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*goipyObject.Str).V
	if strings.Contains(out, "<script>") {
		t.Fatalf("expected HTML-escaped output, got: %q", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected &lt;script&gt; in output, got: %q", out)
	}
}

func TestTemplateMissingVar(t *testing.T) {
	mod := tmplMod()
	renderFn, _ := mod.Dict.GetStr("render")
	result, err := renderFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "Hello {{ .missing }}!"},
		goipyObject.NewDict(),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Go default behavior: missing fields render as <no value>
	out := result.(*goipyObject.Str).V
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected output containing 'Hello', got: %q", out)
	}
}

func TestTemplateCompile(t *testing.T) {
	mod := tmplMod()
	compileFn, _ := mod.Dict.GetStr("compile")
	tmpl, err := compileFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "Hello {{ .name }}!"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := tmpl.(*goipyObject.Instance)
	renderFn, ok := inst.Dict.GetStr("render")
	if !ok {
		t.Fatal("compiled template missing render method")
	}
	data := goipyObject.NewDict()
	data.SetStr("name", &goipyObject.Str{V: "Alice"})
	result, err := renderFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{data}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.(*goipyObject.Str).V != "Hello Alice!" {
		t.Fatalf("unexpected output: %q", result.(*goipyObject.Str).V)
	}
}

func TestTemplateSyntaxError(t *testing.T) {
	mod := tmplMod()
	compileFn, _ := mod.Dict.GetStr("compile")
	_, err := compileFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "{{ .unclosed"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for syntax error in template")
	}
}

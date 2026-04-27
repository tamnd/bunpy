package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func escapeHTMLMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildEscapeHTML(nil)
}

func TestEscapeHTMLModuleMethods(t *testing.T) {
	mod := escapeHTMLMod(t)
	for _, name := range []string{"escape", "unescape", "strip_tags"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("escape_html module missing %q", name)
		}
	}
}

func TestEscapeHTMLEscape(t *testing.T) {
	mod := escapeHTMLMod(t)
	fn, _ := mod.Dict.GetStr("escape")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: `<script>alert("xss")</script>`},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.(*goipyObject.Str).V
	for _, want := range []string{"&lt;", "&gt;", "&#34;"} {
		if !containsStr(got, want) {
			t.Errorf("escaped output %q missing %q", got, want)
		}
	}
}

func TestEscapeHTMLUnescape(t *testing.T) {
	mod := escapeHTMLMod(t)
	fn, _ := mod.Dict.GetStr("unescape")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "&lt;b&gt;bold&lt;/b&gt;"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.(*goipyObject.Str).V
	if got != "<b>bold</b>" {
		t.Errorf("unescape: expected <b>bold</b>, got %q", got)
	}
}

func TestEscapeHTMLStripTags(t *testing.T) {
	mod := escapeHTMLMod(t)
	fn, _ := mod.Dict.GetStr("strip_tags")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "<b>hello</b> <em>world</em>"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.(*goipyObject.Str).V
	if got != "hello world" {
		t.Errorf("strip_tags: expected 'hello world', got %q", got)
	}
}

func TestEscapeHTMLRoundTrip(t *testing.T) {
	mod := escapeHTMLMod(t)
	escFn, _ := mod.Dict.GetStr("escape")
	unescFn, _ := mod.Dict.GetStr("unescape")
	original := `<p class="x">hello & "world"</p>`
	r1, _ := escFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: original},
	}, nil)
	r2, _ := unescFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{r1}, nil)
	if r2.(*goipyObject.Str).V != original {
		t.Errorf("round-trip failed: got %q", r2.(*goipyObject.Str).V)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}())
}

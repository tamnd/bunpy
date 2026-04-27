package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func htmlRWMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	i := serveInterp(t)
	return bunpyAPI.BuildHTMLRewriter(i)
}

func newHTMLRewriter(t *testing.T, src string) *goipyObject.Instance {
	t.Helper()
	mod := htmlRWMod(t)
	fn, _ := mod.Dict.GetStr("HTMLRewriter")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: src},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*goipyObject.Instance)
}

func rewriterTransform(t *testing.T, rw *goipyObject.Instance) string {
	t.Helper()
	fn, _ := rw.Dict.GetStr("transform")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*goipyObject.Str).V
}

func TestHTMLRewriterModuleHasConstructor(t *testing.T) {
	mod := htmlRWMod(t)
	if _, ok := mod.Dict.GetStr("HTMLRewriter"); !ok {
		t.Fatal("HTMLRewriter constructor not found in module")
	}
}

func TestHTMLRewriterNoHandlers(t *testing.T) {
	rw := newHTMLRewriter(t, "<p>hello</p>")
	out := rewriterTransform(t, rw)
	if out != "<p>hello</p>" {
		t.Errorf("no-op transform changed output: %q", out)
	}
}

func TestHTMLRewriterSetAttribute(t *testing.T) {
	rw := newHTMLRewriter(t, `<a href="old">link</a>`)
	onFn, _ := rw.Dict.GetStr("on")
	handler := &goipyObject.BuiltinFunc{
		Name: "handler",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			el := args[0].(*goipyObject.Instance)
			setAttr, _ := el.Dict.GetStr("set_attribute")
			setAttr.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
				&goipyObject.Str{V: "href"},
				&goipyObject.Str{V: "https://example.com"},
			}, nil)
			return goipyObject.None, nil
		},
	}
	onFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "a"}, handler,
	}, nil)
	out := rewriterTransform(t, rw)
	if !strings.Contains(out, "https://example.com") {
		t.Errorf("expected updated href in %q", out)
	}
}

func TestHTMLRewriterRemoveElement(t *testing.T) {
	rw := newHTMLRewriter(t, "<div><script>bad()</script></div>")
	onFn, _ := rw.Dict.GetStr("on")
	handler := &goipyObject.BuiltinFunc{
		Name: "handler",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			el := args[0].(*goipyObject.Instance)
			removeFn, _ := el.Dict.GetStr("remove")
			removeFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
			return goipyObject.None, nil
		},
	}
	onFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "script"}, handler,
	}, nil)
	out := rewriterTransform(t, rw)
	if strings.Contains(out, "<script>") {
		t.Errorf("script tag should have been removed from %q", out)
	}
}

func TestHTMLRewriterSetInnerContent(t *testing.T) {
	rw := newHTMLRewriter(t, "<title>old title</title>")
	onFn, _ := rw.Dict.GetStr("on")
	handler := &goipyObject.BuiltinFunc{
		Name: "handler",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			el := args[0].(*goipyObject.Instance)
			setInner, _ := el.Dict.GetStr("set_inner_content")
			setInner.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
				&goipyObject.Str{V: "new title"},
			}, nil)
			return goipyObject.None, nil
		},
	}
	onFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "title"}, handler,
	}, nil)
	out := rewriterTransform(t, rw)
	if !strings.Contains(out, "new title") {
		t.Errorf("expected 'new title' in %q", out)
	}
}

func TestHTMLRewriterAppendPrepend(t *testing.T) {
	rw := newHTMLRewriter(t, "<body></body>")
	onFn, _ := rw.Dict.GetStr("on")
	handler := &goipyObject.BuiltinFunc{
		Name: "handler",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			el := args[0].(*goipyObject.Instance)
			prepFn, _ := el.Dict.GetStr("prepend")
			prepFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
				&goipyObject.Str{V: "<!-- before -->"},
			}, nil)
			appFn, _ := el.Dict.GetStr("append")
			appFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
				&goipyObject.Str{V: "<!-- after -->"},
			}, nil)
			return goipyObject.None, nil
		},
	}
	onFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "body"}, handler,
	}, nil)
	out := rewriterTransform(t, rw)
	if !strings.Contains(out, "<!-- before -->") || !strings.Contains(out, "<!-- after -->") {
		t.Errorf("prepend/append not found in %q", out)
	}
}

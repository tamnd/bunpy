package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func urlPatternMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildURLPattern(nil)
}

func newURLPattern(t *testing.T, pattern string) *goipyObject.Instance {
	t.Helper()
	mod := urlPatternMod(t)
	fn, _ := mod.Dict.GetStr("URLPattern")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: pattern},
	}, nil)
	if err != nil {
		t.Fatalf("URLPattern(%q): %v", pattern, err)
	}
	return r.(*goipyObject.Instance)
}

func urlPatternTest(t *testing.T, p *goipyObject.Instance, path string) bool {
	t.Helper()
	fn, _ := p.Dict.GetStr("test")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: path},
	}, nil)
	if err != nil {
		t.Fatalf("test(%q): %v", path, err)
	}
	return r.(*goipyObject.Bool).V
}

func urlPatternExec(t *testing.T, p *goipyObject.Instance, path string) goipyObject.Object {
	t.Helper()
	fn, _ := p.Dict.GetStr("exec")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: path},
	}, nil)
	if err != nil {
		t.Fatalf("exec(%q): %v", path, err)
	}
	return r
}

func TestURLPatternModuleHasConstructor(t *testing.T) {
	mod := urlPatternMod(t)
	if _, ok := mod.Dict.GetStr("URLPattern"); !ok {
		t.Fatal("URLPattern constructor not found in module")
	}
}

func TestURLPatternTestMatch(t *testing.T) {
	p := newURLPattern(t, "/users/:id")
	if !urlPatternTest(t, p, "/users/123") {
		t.Error("/users/123 should match /users/:id")
	}
}

func TestURLPatternTestNoMatch(t *testing.T) {
	p := newURLPattern(t, "/users/:id")
	if urlPatternTest(t, p, "/posts/123") {
		t.Error("/posts/123 should not match /users/:id")
	}
}

func TestURLPatternExecCaptures(t *testing.T) {
	p := newURLPattern(t, "/users/:id/posts/:postId")
	r := urlPatternExec(t, p, "/users/42/posts/99")
	if r == goipyObject.None {
		t.Fatal("exec should return a match object, got None")
	}
	result := r.(*goipyObject.Dict)
	pathnameV, _ := result.GetStr("pathname")
	pathname := pathnameV.(*goipyObject.Dict)
	groups, _ := pathname.GetStr("groups")
	gd := groups.(*goipyObject.Dict)
	id, _ := gd.GetStr("id")
	if id.(*goipyObject.Str).V != "42" {
		t.Errorf("id: expected 42, got %v", id)
	}
	postId, _ := gd.GetStr("postId")
	if postId.(*goipyObject.Str).V != "99" {
		t.Errorf("postId: expected 99, got %v", postId)
	}
}

func TestURLPatternExecNoMatch(t *testing.T) {
	p := newURLPattern(t, "/users/:id")
	r := urlPatternExec(t, p, "/other/path")
	if r != goipyObject.None {
		t.Error("non-matching exec should return None")
	}
}

func TestURLPatternWildcard(t *testing.T) {
	p := newURLPattern(t, "/files/*")
	if !urlPatternTest(t, p, "/files/a/b/c.txt") {
		t.Error("/files/a/b/c.txt should match /files/*")
	}
}

func TestURLPatternFullURL(t *testing.T) {
	p := newURLPattern(t, "/users/:id")
	if !urlPatternTest(t, p, "https://example.com/users/99") {
		t.Error("full URL should match via path extraction")
	}
}

func TestURLPatternStaticPath(t *testing.T) {
	p := newURLPattern(t, "/about")
	if !urlPatternTest(t, p, "/about") {
		t.Error("/about should match /about")
	}
	if urlPatternTest(t, p, "/about/us") {
		t.Error("/about/us should not match /about")
	}
}

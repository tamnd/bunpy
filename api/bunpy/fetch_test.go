package bunpy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func fetchInterp(t *testing.T) *goipyVM.Interp {
	t.Helper()
	i := goipyVM.New()
	i.SetNativeModules(bunpyAPI.Modules())
	bunpyAPI.InjectGlobals(i)
	return i
}

func callFetch(t *testing.T, i *goipyVM.Interp, url string, kwargs map[string]goipyObject.Object) *goipyObject.Instance {
	t.Helper()
	fetchObj, ok := i.Builtins.GetStr("fetch")
	if !ok {
		t.Fatal("fetch not in builtins")
	}
	fn := fetchObj.(*goipyObject.BuiltinFunc)
	var kw *goipyObject.Dict
	if kwargs != nil {
		kw = goipyObject.NewDict()
		for k, v := range kwargs {
			kw.SetStr(k, v)
		}
	}
	result, err := fn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: url}}, kw)
	if err != nil {
		t.Fatalf("fetch(%q) error: %v", url, err)
	}
	inst, ok := result.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("fetch returned %T, want Instance", result)
	}
	return inst
}

func TestFetchGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello from test server"))
	}))
	defer srv.Close()

	i := fetchInterp(t)
	resp := callFetch(t, i, srv.URL, nil)

	statusObj, _ := resp.Dict.GetStr("status")
	if goipyObject.NewInt(200).Int64() != statusObj.(*goipyObject.Int).Int64() {
		t.Fatalf("status = %v, want 200", statusObj)
	}
	okObj, _ := resp.Dict.GetStr("ok")
	if !okObj.(*goipyObject.Bool).V {
		t.Fatal("ok should be true for 200")
	}
}

func TestFetchTextMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello text"))
	}))
	defer srv.Close()

	i := fetchInterp(t)
	resp := callFetch(t, i, srv.URL, nil)

	textFn, _ := resp.Dict.GetStr("text")
	out, err := textFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s := out.(*goipyObject.Str).V; s != "hello text" {
		t.Fatalf("text() = %q, want %q", s, "hello text")
	}
}

func TestFetchJSONMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"alice","age":30}`))
	}))
	defer srv.Close()

	i := fetchInterp(t)
	resp := callFetch(t, i, srv.URL, nil)

	jsonFn, _ := resp.Dict.GetStr("json")
	out, err := jsonFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := out.(*goipyObject.Dict)
	if !ok {
		t.Fatalf("json() returned %T, want Dict", out)
	}
	nameVal, _ := d.GetStr("name")
	if nameVal.(*goipyObject.Str).V != "alice" {
		t.Fatalf("json().name = %q, want %q", nameVal, "alice")
	}
}

func TestFetchPOST(t *testing.T) {
	var gotMethod string
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(201)
	}))
	defer srv.Close()

	i := fetchInterp(t)
	callFetch(t, i, srv.URL, map[string]goipyObject.Object{
		"method": &goipyObject.Str{V: "POST"},
		"body":   &goipyObject.Bytes{V: []byte(`{"hello":"world"}`)},
	})

	if gotMethod != "POST" {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotBody != `{"hello":"world"}` {
		t.Fatalf("body = %q", gotBody)
	}
}

func TestFetch404IsNotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	i := fetchInterp(t)
	resp := callFetch(t, i, srv.URL, nil)

	okObj, _ := resp.Dict.GetStr("ok")
	if okObj.(*goipyObject.Bool).V {
		t.Fatal("ok should be false for 404")
	}
}

// --- URL class ---

func TestURLParsesComponents(t *testing.T) {
	i := fetchInterp(t)
	urlClass, ok := i.Builtins.GetStr("URL")
	if !ok {
		t.Fatal("URL not in builtins")
	}
	cls := urlClass.(*goipyObject.Class)
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}
	initFn, _ := cls.Dict.GetStr("__init__")
	_, err := initFn.(*goipyObject.BuiltinFunc).Call(inst, []goipyObject.Object{
		&goipyObject.Str{V: "https://example.com:8080/path?q=1#frag"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	check := func(field, want string) {
		v, ok := inst.Dict.GetStr(field)
		if !ok {
			t.Errorf("URL.%s missing", field)
			return
		}
		if s := v.(*goipyObject.Str).V; s != want {
			t.Errorf("URL.%s = %q, want %q", field, s, want)
		}
	}
	check("protocol", "https:")
	check("hostname", "example.com")
	check("port", "8080")
	check("pathname", "/path")
	check("search", "?q=1")
	check("hash", "#frag")
}

// --- Response class ---

func TestResponseConstructor(t *testing.T) {
	i := fetchInterp(t)
	respClass, _ := i.Builtins.GetStr("Response")
	cls := respClass.(*goipyObject.Class)
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}
	initFn, _ := cls.Dict.GetStr("__init__")
	_, err := initFn.(*goipyObject.BuiltinFunc).Call(inst, []goipyObject.Object{
		&goipyObject.Str{V: "hello"},
		goipyObject.NewInt(201),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	statusObj, _ := inst.Dict.GetStr("status")
	if statusObj.(*goipyObject.Int).Int64() != 201 {
		t.Fatalf("status = %v, want 201", statusObj)
	}
}

// --- InjectGlobals ---

func TestInjectGlobalsPresent(t *testing.T) {
	i := fetchInterp(t)
	for _, name := range []string{"fetch", "URL", "Request", "Response", "Headers"} {
		if _, ok := i.Builtins.GetStr(name); !ok {
			t.Errorf("global %q not injected", name)
		}
	}
}

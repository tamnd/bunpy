package bunpy_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func httpMod() *goipyObject.Module { return bunpyAPI.BuildHTTP(nil) }

func TestHTTPModuleHasMethods(t *testing.T) {
	mod := httpMod()
	for _, m := range []string{"get", "post", "put", "delete", "session"} {
		if _, ok := mod.Dict.GetStr(m); !ok {
			t.Fatalf("http module missing %q", m)
		}
	}
}

func TestHTTPGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	mod := httpMod()
	getFn, _ := mod.Dict.GetStr("get")
	result, err := getFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: srv.URL},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)
	statusObj, _ := inst.Dict.GetStr("status")
	if statusObj.(*goipyObject.Int).Int64() != 200 {
		t.Fatalf("expected status 200, got %v", statusObj)
	}
}

func TestHTTPResponseText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	mod := httpMod()
	getFn, _ := mod.Dict.GetStr("get")
	result, _ := getFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: srv.URL},
	}, nil)
	textFn, _ := result.(*goipyObject.Instance).Dict.GetStr("text")
	text, err := textFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text.(*goipyObject.Str).V != "hello world" {
		t.Fatalf("expected 'hello world', got %q", text.(*goipyObject.Str).V)
	}
}

func TestHTTPResponseJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"key": "value"})
	}))
	defer srv.Close()

	mod := httpMod()
	getFn, _ := mod.Dict.GetStr("get")
	result, _ := getFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: srv.URL},
	}, nil)
	jsonFn, _ := result.(*goipyObject.Instance).Dict.GetStr("json")
	parsed, err := jsonFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := parsed.(*goipyObject.Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", parsed)
	}
	v, ok2 := d.GetStr("key")
	if !ok2 || v.(*goipyObject.Str).V != "value" {
		t.Fatalf("expected key=value, got %v", v)
	}
}

func TestHTTPSessionBaseURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	mod := httpMod()
	sessionFn, _ := mod.Dict.GetStr("session")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("base_url", &goipyObject.Str{V: srv.URL})
	sess, err := sessionFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	inst := sess.(*goipyObject.Instance)
	getFn, _ := inst.Dict.GetStr("get")
	result, err := getFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "/health"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	status, _ := result.(*goipyObject.Instance).Dict.GetStr("status")
	if status.(*goipyObject.Int).Int64() != 200 {
		t.Fatalf("expected 200, got %v", status)
	}
}

func TestHTTPSessionPersistentHeader(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	mod := httpMod()
	sessionFn, _ := mod.Dict.GetStr("session")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("base_url", &goipyObject.Str{V: srv.URL})
	hdr := goipyObject.NewDict()
	hdr.SetStr("Authorization", &goipyObject.Str{V: "Bearer token123"})
	kwargs.SetStr("headers", hdr)
	sess, _ := sessionFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)

	getFn, _ := sess.(*goipyObject.Instance).Dict.GetStr("get")
	getFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "/"},
	}, nil)

	if receivedAuth != "Bearer token123" {
		t.Fatalf("expected Authorization header, got %q", receivedAuth)
	}
}

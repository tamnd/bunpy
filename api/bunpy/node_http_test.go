package bunpy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
)

func TestNodeHTTPGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	mod := BuildNodeHTTP(nil)
	getFn := mustGetBuiltin(t, mod.Dict, "get")
	res, err := getFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: srv.URL}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := res.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", res)
	}
	bodyObj, _ := inst.Dict.GetStr("body")
	s, ok := bodyObj.(*goipyObject.Str)
	if !ok || s.V != "ok" {
		t.Errorf("expected body 'ok', got %v", bodyObj)
	}
}

func TestNodeHTTPRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", 400)
			return
		}
		fmt.Fprint(w, "posted")
	}))
	defer srv.Close()

	mod := BuildNodeHTTP(nil)
	reqFn := mustGetBuiltin(t, mod.Dict, "request")

	opts := goipyObject.NewDict()
	opts.SetStr("url", &goipyObject.Str{V: srv.URL})
	opts.SetStr("method", &goipyObject.Str{V: "POST"})

	res, err := reqFn.Call(nil, []goipyObject.Object{opts}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := res.(*goipyObject.Instance)
	bodyObj, _ := inst.Dict.GetStr("body")
	s := bodyObj.(*goipyObject.Str)
	if s.V != "posted" {
		t.Errorf("expected 'posted', got %q", s.V)
	}
}

func TestNodeHTTPResponseFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(201)
		fmt.Fprint(w, "created")
	}))
	defer srv.Close()

	mod := BuildNodeHTTP(nil)
	getFn := mustGetBuiltin(t, mod.Dict, "get")
	res, _ := getFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: srv.URL}}, nil)
	inst := res.(*goipyObject.Instance)

	statusObj, _ := inst.Dict.GetStr("statusCode")
	n, ok := statusObj.(*goipyObject.Int)
	if !ok || n.Int64() != 201 {
		t.Errorf("expected statusCode 201, got %v", statusObj)
	}

	hdrs, _ := inst.Dict.GetStr("headers")
	hd, ok := hdrs.(*goipyObject.Dict)
	if !ok {
		t.Fatal("headers should be a Dict")
	}
	xtest, ok := hd.GetStr("x-test")
	if !ok {
		t.Error("X-Test header not found (lowercased)")
	}
	if s, ok := xtest.(*goipyObject.Str); !ok || s.V != "yes" {
		t.Errorf("X-Test: got %v", xtest)
	}
}

package bunpy_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func serveInterp(t *testing.T) *goipyVM.Interp {
	t.Helper()
	i := goipyVM.New()
	i.NativeModules = bunpyAPI.Modules()
	bunpyAPI.InjectGlobals(i)
	return i
}

// makeBuiltinHandler returns a BuiltinFunc that returns a fixed Response body.
func makeBuiltinHandler(statusCode int, body string) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "handler",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			inst := &goipyObject.Instance{
				Class: &goipyObject.Class{Name: "Response", Dict: goipyObject.NewDict()},
				Dict:  goipyObject.NewDict(),
			}
			inst.Dict.SetStr("status", goipyObject.NewInt(int64(statusCode)))
			inst.Dict.SetStr("_body", &goipyObject.Bytes{V: []byte(body)})
			inst.Dict.SetStr("headers", &goipyObject.Instance{
				Class: &goipyObject.Class{Name: "Headers", Dict: goipyObject.NewDict()},
				Dict:  goipyObject.NewDict(),
			})
			return inst, nil
		},
	}
}

func startTestServer(t *testing.T, handler goipyObject.Object) (serverURL string, stopFn func()) {
	t.Helper()
	i := serveInterp(t)
	serveFn := bunpyAPI.BuildServe(i)
	kw := goipyObject.NewDict()
	kw.SetStr("port", goipyObject.NewInt(0)) // port 0 = OS-assigned
	kw.SetStr("handler", handler)
	result, err := serveFn.Call(nil, nil, kw)
	if err != nil {
		t.Fatalf("bunpy.serve(): %v", err)
	}
	inst := result.(*goipyObject.Instance)
	portObj, _ := inst.Dict.GetStr("port")
	port := portObj.(*goipyObject.Int).Int64()
	urlObj, _ := inst.Dict.GetStr("url")
	_ = urlObj
	stopObj, _ := inst.Dict.GetStr("stop")
	stopFn = func() {
		stopObj.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	}
	return fmt.Sprintf("http://localhost:%d", port), stopFn
}

func TestServeBasicGET(t *testing.T) {
	url, stop := startTestServer(t, makeBuiltinHandler(200, "hello serve"))
	defer stop()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if string(body) != "hello serve" {
		t.Fatalf("body = %q, want %q", body, "hello serve")
	}
}

func TestServe404Handler(t *testing.T) {
	url, stop := startTestServer(t, makeBuiltinHandler(404, "not found"))
	defer stop()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestServePortIsNonZero(t *testing.T) {
	i := serveInterp(t)
	serveFn := bunpyAPI.BuildServe(i)
	kw := goipyObject.NewDict()
	kw.SetStr("port", goipyObject.NewInt(0))
	kw.SetStr("handler", makeBuiltinHandler(200, "ok"))
	result, err := serveFn.Call(nil, nil, kw)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)
	portObj, _ := inst.Dict.GetStr("port")
	port := portObj.(*goipyObject.Int).Int64()
	if port == 0 {
		t.Fatal("port should be non-zero after bind")
	}
	// stop
	stopFn, _ := inst.Dict.GetStr("stop")
	stopFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
}

func TestServeStopPreventsNewConnections(t *testing.T) {
	url, stop := startTestServer(t, makeBuiltinHandler(200, "alive"))
	// first request should succeed
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	// stop the server
	stop()
	// subsequent request should fail
	_, err = http.Get(url)
	if err == nil {
		t.Fatal("expected connection error after server stop")
	}
}

func TestServeReloadHandler(t *testing.T) {
	url, stop := startTestServer(t, makeBuiltinHandler(200, "before"))
	defer stop()

	// check initial response
	resp, _ := http.Get(url)
	resp.Body.Close()

	// reload with new handler
	i := serveInterp(t)
	serveFn := bunpyAPI.BuildServe(i)
	kw := goipyObject.NewDict()
	kw.SetStr("port", goipyObject.NewInt(0))
	kw.SetStr("handler", makeBuiltinHandler(200, "before"))
	result, _ := serveFn.Call(nil, nil, kw)
	inst := result.(*goipyObject.Instance)

	reloadFn, _ := inst.Dict.GetStr("reload")
	reloadFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		makeBuiltinHandler(200, "after"),
	}, nil)

	portObj, _ := inst.Dict.GetStr("port")
	port := portObj.(*goipyObject.Int).Int64()
	resp2, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	body, _ := io.ReadAll(resp2.Body)
	if string(body) != "after" {
		t.Fatalf("after reload, body = %q, want %q", body, "after")
	}
	stopFn, _ := inst.Dict.GetStr("stop")
	stopFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
}

func TestServeHandlerReceivesRequestMethod(t *testing.T) {
	var gotMethod string
	handler := &goipyObject.BuiltinFunc{
		Name: "handler",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) == 1 {
				req := args[0].(*goipyObject.Instance)
				if m, ok := req.Dict.GetStr("method"); ok {
					gotMethod = m.(*goipyObject.Str).V
				}
			}
			return &goipyObject.Str{V: "ok"}, nil
		},
	}
	url, stop := startTestServer(t, handler)
	defer stop()

	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotMethod != "DELETE" {
		t.Fatalf("method = %q, want DELETE", gotMethod)
	}
}

func TestBunpyModuleHasServe(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	if _, ok := m.Dict.GetStr("serve"); !ok {
		t.Fatal("bunpy.serve missing from top-level module")
	}
}

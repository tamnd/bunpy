package bunpy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildServe adds `bunpy.serve` to the bunpy top-level module.
// It is called from BuildBunpy so `bunpy.serve(...)` works directly.
//
// Python surface:
//
//	server = bunpy.serve(port=3000, handler=my_fn)
//	server = bunpy.serve(port=3000, handler=lambda req: Response("hi"))
//	server.port          # int - the actual bound port
//	server.hostname      # str - e.g. "localhost"
//	server.url           # str - e.g. "http://localhost:3000"
//	server.stop()        # stop accepting new connections
//	server.reload(new_handler)  # swap handler without restart
//
// The handler receives a Request and must return a Response.
// Requests are processed concurrently (one goroutine per connection).
func BuildServe(i *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "serve",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			port := 3000
			hostname := "localhost"
			var handler goipyObject.Object

			if kwargs != nil {
				if v, ok := kwargs.GetStr("port"); ok {
					if n, ok := v.(*goipyObject.Int); ok {
						port = int(n.Int64())
					}
				}
				if v, ok := kwargs.GetStr("hostname"); ok {
					if s, ok := v.(*goipyObject.Str); ok {
						hostname = s.V
					}
				}
				if v, ok := kwargs.GetStr("handler"); ok {
					handler = v
				}
			}
			// positional: serve(handler) or serve(handler, port)
			if len(args) >= 1 && handler == nil {
				handler = args[0]
			}
			if handler == nil {
				return nil, fmt.Errorf("bunpy.serve(): handler argument required")
			}

			srv, err := startServer(i, hostname, port, handler)
			if err != nil {
				return nil, fmt.Errorf("bunpy.serve(): %w", err)
			}
			return srv, nil
		},
	}
}

// serverInstance is the Python-side Server object returned by bunpy.serve().
type serverState struct {
	httpSrv  *http.Server
	listener net.Listener
	hostname string
	port     int
	handler  atomic.Pointer[goipyObject.Object]
	interp   *goipyVM.Interp
	wg       sync.WaitGroup
}

func startServer(i *goipyVM.Interp, hostname string, port int, handler goipyObject.Object) (*goipyObject.Instance, error) {
	addr := hostname + ":" + strconv.Itoa(port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port

	state := &serverState{
		hostname: hostname,
		port:     actualPort,
		interp:   i,
	}
	state.handler.Store(&handler)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		state.wg.Add(1)
		defer state.wg.Done()
		handleHTTPRequest(i, w, r, state)
	})
	state.httpSrv = &http.Server{Handler: mux}

	go func() {
		_ = state.httpSrv.Serve(ln)
	}()

	return makeServerInstance(state), nil
}

func makeServerInstance(state *serverState) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Server", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	inst.Dict.SetStr("port", goipyObject.NewInt(int64(state.port)))
	inst.Dict.SetStr("hostname", &goipyObject.Str{V: state.hostname})
	inst.Dict.SetStr("url", &goipyObject.Str{V: "http://" + state.hostname + ":" + strconv.Itoa(state.port)})

	inst.Dict.SetStr("stop", &goipyObject.BuiltinFunc{
		Name: "stop",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			_ = state.httpSrv.Shutdown(context.Background())
			state.wg.Wait()
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("reload", &goipyObject.BuiltinFunc{
		Name: "reload",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("Server.reload() takes 1 argument")
			}
			h := args[0]
			state.handler.Store(&h)
			return goipyObject.None, nil
		},
	})

	return inst
}

// handleHTTPRequest converts an http.Request to a goipy Request, calls the
// Python handler, and writes the Response back to the http.ResponseWriter.
func handleHTTPRequest(i *goipyVM.Interp, w http.ResponseWriter, r *http.Request, state *serverState) {
	handlerPtr := state.handler.Load()
	if handlerPtr == nil {
		http.Error(w, "no handler", 500)
		return
	}
	handler := *handlerPtr

	body, _ := io.ReadAll(r.Body)

	// Build a Request instance.
	reqCls := &goipyObject.Class{Name: "Request", Dict: goipyObject.NewDict()}
	reqInst := &goipyObject.Instance{Class: reqCls, Dict: goipyObject.NewDict()}
	reqInst.Dict.SetStr("method", &goipyObject.Str{V: r.Method})
	reqInst.Dict.SetStr("url", &goipyObject.Str{V: r.URL.String()})
	reqInst.Dict.SetStr("_body", &goipyObject.Bytes{V: body})
	reqInst.Dict.SetStr("text", &goipyObject.BuiltinFunc{
		Name: "text",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			v, _ := reqInst.Dict.GetStr("_body")
			return &goipyObject.Str{V: string(v.(*goipyObject.Bytes).V)}, nil
		},
	})
	reqInst.Dict.SetStr("bytes", &goipyObject.BuiltinFunc{
		Name: "bytes",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			v, _ := reqInst.Dict.GetStr("_body")
			return v, nil
		},
	})

	hdrDict := goipyObject.NewDict()
	for k, vals := range r.Header {
		hdrDict.SetStr(strings.ToLower(k), &goipyObject.Str{V: strings.Join(vals, ", ")})
	}
	reqInst.Dict.SetStr("headers", &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Headers", Dict: goipyObject.NewDict()},
		Dict:  hdrDict,
	})

	// Call the Python handler.
	result, callErr := i.Call(handler, []goipyObject.Object{reqInst}, nil)
	if callErr != nil {
		http.Error(w, callErr.Error(), 500)
		return
	}

	writeResponse(w, result)
}

// writeResponse converts a Python Response (or str/bytes) to an HTTP response.
func writeResponse(w http.ResponseWriter, result goipyObject.Object) {
	switch r := result.(type) {
	case *goipyObject.Str:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(r.V))
	case *goipyObject.Bytes:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(200)
		_, _ = w.Write(r.V)
	case *goipyObject.Instance:
		status := 200
		if sv, ok := r.Dict.GetStr("status"); ok {
			if n, ok := sv.(*goipyObject.Int); ok {
				status = int(n.Int64())
			}
		}
		// headers
		if hv, ok := r.Dict.GetStr("headers"); ok {
			if hinst, ok := hv.(*goipyObject.Instance); ok {
				keys, vals := hinst.Dict.Items()
				for idx, k := range keys {
					ks, ok1 := k.(*goipyObject.Str)
					vs, ok2 := vals[idx].(*goipyObject.Str)
					if ok1 && ok2 {
						w.Header().Set(ks.V, vs.V)
					}
				}
			}
		}
		w.WriteHeader(status)
		if bv, ok := r.Dict.GetStr("_body"); ok {
			switch b := bv.(type) {
			case *goipyObject.Bytes:
				_, _ = w.Write(b.V)
			case *goipyObject.Str:
				_, _ = w.Write([]byte(b.V))
			}
		}
	default:
		w.WriteHeader(500)
		_, _ = fmt.Fprintf(w, "handler returned unexpected type: %T", result)
	}
}

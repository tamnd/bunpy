package bunpy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNodeHTTP builds the bunpy.node.http module.
func BuildNodeHTTP(i *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.http", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("get", nodeHTTPGet(false))
	mod.Dict.SetStr("request", nodeHTTPRequest(false))
	mod.Dict.SetStr("createServer", nodeHTTPCreateServer(i))

	return mod
}

// BuildNodeHTTPS builds the bunpy.node.https module.
func BuildNodeHTTPS(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.https", Dict: goipyObject.NewDict()}
	mod.Dict.SetStr("get", nodeHTTPGet(true))
	mod.Dict.SetStr("request", nodeHTTPRequest(true))
	return mod
}

func nodeHTTPGet(https bool) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "get",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("http.get() requires url")
			}
			url, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("http.get(): url must be str")
			}
			u := url.V
			if https && !strings.HasPrefix(u, "https://") {
				u = "https://" + strings.TrimPrefix(u, "http://")
			}
			resp, err := http.Get(u) //nolint:gosec
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			return nodeHTTPResponseObj(resp)
		},
	}
}

func nodeHTTPRequest(https bool) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "request",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("http.request() requires options dict or url")
			}
			var (
				method  = "GET"
				url     = ""
				headers = map[string]string{}
				body    []byte
			)
			switch v := args[0].(type) {
			case *goipyObject.Str:
				url = v.V
			case *goipyObject.Dict:
				if u, ok := v.GetStr("url"); ok {
					if s, ok := u.(*goipyObject.Str); ok {
						url = s.V
					}
				}
				if m, ok := v.GetStr("method"); ok {
					if s, ok := m.(*goipyObject.Str); ok {
						method = strings.ToUpper(s.V)
					}
				}
				if h, ok := v.GetStr("headers"); ok {
					if hd, ok := h.(*goipyObject.Dict); ok {
						keys, vals := hd.Items()
						for i, k := range keys {
							if ks, ok := k.(*goipyObject.Str); ok {
								if vs, ok := vals[i].(*goipyObject.Str); ok {
									headers[ks.V] = vs.V
								}
							}
						}
					}
				}
				if b, ok := v.GetStr("body"); ok {
					switch bv := b.(type) {
					case *goipyObject.Str:
						body = []byte(bv.V)
					case *goipyObject.Bytes:
						body = bv.V
					}
				}
			}
			if https && !strings.HasPrefix(url, "https://") {
				url = "https://" + strings.TrimPrefix(url, "http://")
			}
			req, err := http.NewRequest(method, url, bytes.NewReader(body))
			if err != nil {
				return nil, err
			}
			for k, v := range headers {
				req.Header.Set(k, v)
			}
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			return nodeHTTPResponseObj(resp)
		},
	}
}

func nodeHTTPResponseObj(resp *http.Response) (goipyObject.Object, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	d := goipyObject.NewDict()
	d.SetStr("status", goipyObject.NewInt(int64(resp.StatusCode)))
	d.SetStr("statusCode", goipyObject.NewInt(int64(resp.StatusCode)))
	d.SetStr("body", &goipyObject.Str{V: string(body)})
	hd := goipyObject.NewDict()
	for k, vs := range resp.Header {
		hd.SetStr(strings.ToLower(k), &goipyObject.Str{V: strings.Join(vs, ", ")})
	}
	d.SetStr("headers", hd)
	return &goipyObject.Instance{Class: &goipyObject.Class{Name: "IncomingMessage"}, Dict: d}, nil
}

func nodeHTTPCreateServer(interp *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "createServer",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			var handler goipyObject.Object
			if len(args) >= 1 {
				handler = args[0]
			}
			return newNodeHTTPServerInstance(interp, handler), nil
		},
	}
}

func newNodeHTTPServerInstance(interp *goipyVM.Interp, handler goipyObject.Object) *goipyObject.Instance {
	type serverState struct {
		mu     sync.Mutex
		server *http.Server
	}
	state := &serverState{}

	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Server"},
		Dict:  goipyObject.NewDict(),
	}

	inst.Dict.SetStr("listen", &goipyObject.BuiltinFunc{
		Name: "listen",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			port := 3000
			if len(args) >= 1 {
				if n, ok := args[0].(*goipyObject.Int); ok {
					port = int(n.Int64())
				}
			}
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if handler == nil {
					return
				}
				// Build req object.
				bodyBytes, _ := io.ReadAll(r.Body)
				reqD := goipyObject.NewDict()
				reqD.SetStr("method", &goipyObject.Str{V: r.Method})
				reqD.SetStr("url", &goipyObject.Str{V: r.URL.String()})
				reqD.SetStr("body", &goipyObject.Str{V: string(bodyBytes)})
				reqObj := &goipyObject.Instance{Class: &goipyObject.Class{Name: "IncomingMessage"}, Dict: reqD}

				// Build res object.
				var code int = 200
				resD := goipyObject.NewDict()
				resD.SetStr("statusCode", goipyObject.NewInt(200))
				var body strings.Builder
				resD.SetStr("write", &goipyObject.BuiltinFunc{
					Name: "write",
					Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
						if len(args) >= 1 {
							if s, ok := args[0].(*goipyObject.Str); ok {
								body.WriteString(s.V)
							}
						}
						return goipyObject.None, nil
					},
				})
				resD.SetStr("end", &goipyObject.BuiltinFunc{
					Name: "end",
					Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
						if len(args) >= 1 {
							if s, ok := args[0].(*goipyObject.Str); ok {
								body.WriteString(s.V)
							}
						}
						w.WriteHeader(code)
						w.Write([]byte(body.String()))
						return goipyObject.None, nil
					},
				})
				resD.SetStr("setHeader", &goipyObject.BuiltinFunc{
					Name: "setHeader",
					Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
						if len(args) >= 2 {
							k, ok1 := args[0].(*goipyObject.Str)
							v, ok2 := args[1].(*goipyObject.Str)
							if ok1 && ok2 {
								w.Header().Set(k.V, v.V)
							}
						}
						return goipyObject.None, nil
					},
				})
				resObj := &goipyObject.Instance{Class: &goipyObject.Class{Name: "ServerResponse"}, Dict: resD}

				interp.Call(handler, []goipyObject.Object{reqObj, resObj}, nil)
			})

			srv := &http.Server{
				Addr:    ":" + strconv.Itoa(port),
				Handler: mux,
			}
			state.mu.Lock()
			state.server = srv
			state.mu.Unlock()

			go srv.ListenAndServe()
			return inst, nil
		},
	})

	inst.Dict.SetStr("close", &goipyObject.BuiltinFunc{
		Name: "close",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			state.mu.Lock()
			srv := state.server
			state.mu.Unlock()
			if srv != nil {
				srv.Close()
			}
			return goipyObject.None, nil
		},
	})

	return inst
}

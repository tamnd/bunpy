package bunpy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildFetch builds the global fetch function and registers URL, Request,
// Response classes in the bunpy namespace.
//
// Python surface:
//
//	resp = fetch("https://example.com")
//	resp = fetch("https://example.com", method="POST", body=b"data",
//	              headers={"Content-Type": "application/json"})
//	text   = resp.text()
//	data   = resp.json()
//	body   = resp.bytes()
//	status = resp.status            # int
//	ok     = resp.ok                # bool (status 200-299)
//	hdrs   = resp.headers           # dict-like
func BuildFetch(i *goipyVM.Interp) *goipyObject.Module {
	m := &goipyObject.Module{Name: "bunpy._fetch", Dict: goipyObject.NewDict()}

	responseClass := buildResponseClass(i)
	requestClass := buildRequestClass(i)
	urlClass := buildURLClass(i)
	headersClass := buildHeadersClass(i)

	fetchFn := &goipyObject.BuiltinFunc{
		Name: "fetch",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("fetch() requires at least 1 argument")
			}
			urlStr, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fetch(): url must be str, not %T", args[0])
			}

			method := "GET"
			var bodyBytes []byte
			var headers http.Header = make(http.Header)

			if kwargs != nil {
				if v, ok := kwargs.GetStr("method"); ok {
					if s, ok := v.(*goipyObject.Str); ok {
						method = strings.ToUpper(s.V)
					}
				}
				if v, ok := kwargs.GetStr("body"); ok {
					switch b := v.(type) {
					case *goipyObject.Bytes:
						bodyBytes = b.V
					case *goipyObject.Str:
						bodyBytes = []byte(b.V)
					}
				}
				if v, ok := kwargs.GetStr("headers"); ok {
					if d, ok := v.(*goipyObject.Dict); ok {
						keys, vals := d.Items()
						for idx, k := range keys {
							ks, ok1 := k.(*goipyObject.Str)
							vs, ok2 := vals[idx].(*goipyObject.Str)
							if ok1 && ok2 {
								headers.Set(ks.V, vs.V)
							}
						}
					}
				}
			}

			var bodyReader io.Reader
			if len(bodyBytes) > 0 {
				bodyReader = bytes.NewReader(bodyBytes)
			}
			req, err := http.NewRequest(method, urlStr.V, bodyReader)
			if err != nil {
				return nil, fmt.Errorf("fetch(): %w", err)
			}
			for k, vals := range headers {
				for _, v := range vals {
					req.Header.Set(k, v)
				}
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("fetch(): %w", err)
			}
			defer resp.Body.Close()
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("fetch(): read body: %w", err)
			}

			return makeResponse(i, responseClass, headersClass, resp, respBody), nil
		},
	}

	m.Dict.SetStr("fetch", fetchFn)
	m.Dict.SetStr("Response", responseClass)
	m.Dict.SetStr("Request", requestClass)
	m.Dict.SetStr("URL", urlClass)
	m.Dict.SetStr("Headers", headersClass)
	return m
}

// makeResponse constructs a Response instance from an http.Response.
func makeResponse(i *goipyVM.Interp, cls *goipyObject.Class, hdrCls *goipyObject.Class, resp *http.Response, body []byte) *goipyObject.Instance {
	_ = i
	obj := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}
	obj.Dict.SetStr("status", goipyObject.NewInt(int64(resp.StatusCode)))
	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	obj.Dict.SetStr("ok", goipyObject.BoolOf(ok))
	obj.Dict.SetStr("url", &goipyObject.Str{V: resp.Request.URL.String()})
	obj.Dict.SetStr("_body", &goipyObject.Bytes{V: body})

	// headers as a dict
	hdrDict := goipyObject.NewDict()
	for k, vals := range resp.Header {
		hdrDict.SetStr(strings.ToLower(k), &goipyObject.Str{V: strings.Join(vals, ", ")})
	}
	hdrInst := &goipyObject.Instance{Class: hdrCls, Dict: hdrDict}
	obj.Dict.SetStr("headers", hdrInst)

	// text() method
	obj.Dict.SetStr("text", &goipyObject.BuiltinFunc{
		Name: "text",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			v, _ := obj.Dict.GetStr("_body")
			b := v.(*goipyObject.Bytes)
			return &goipyObject.Str{V: string(b.V)}, nil
		},
	})
	// bytes() method
	obj.Dict.SetStr("bytes", &goipyObject.BuiltinFunc{
		Name: "bytes",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			v, _ := obj.Dict.GetStr("_body")
			return v, nil
		},
	})
	// json() method
	obj.Dict.SetStr("json", &goipyObject.BuiltinFunc{
		Name: "json",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			v, _ := obj.Dict.GetStr("_body")
			b := v.(*goipyObject.Bytes)
			var raw any
			if err := json.Unmarshal(b.V, &raw); err != nil {
				return nil, fmt.Errorf("Response.json(): %w", err)
			}
			return jsonToObject(raw), nil
		},
	})
	return obj
}

func buildResponseClass(_ *goipyVM.Interp) *goipyObject.Class {
	cls := &goipyObject.Class{Name: "Response", Dict: goipyObject.NewDict()}
	cls.Dict.SetStr("__init__", &goipyObject.BuiltinFunc{
		Name: "__init__",
		Call: func(self any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			inst := self.(*goipyObject.Instance)
			var bodyBytes []byte
			status := 200
			if len(args) >= 1 {
				switch b := args[0].(type) {
				case *goipyObject.Bytes:
					bodyBytes = b.V
				case *goipyObject.Str:
					bodyBytes = []byte(b.V)
				}
			}
			if len(args) >= 2 {
				if n, ok := args[1].(*goipyObject.Int); ok {
					status = int(n.Int64())
				}
			}
			if kwargs != nil {
				if v, ok := kwargs.GetStr("status"); ok {
					if n, ok := v.(*goipyObject.Int); ok {
						status = int(n.Int64())
					}
				}
			}
			inst.Dict.SetStr("status", goipyObject.NewInt(int64(status)))
			inst.Dict.SetStr("ok", goipyObject.BoolOf(status >= 200 && status < 300))
			inst.Dict.SetStr("_body", &goipyObject.Bytes{V: bodyBytes})
			inst.Dict.SetStr("headers", &goipyObject.Instance{
				Class: &goipyObject.Class{Name: "Headers", Dict: goipyObject.NewDict()},
				Dict:  goipyObject.NewDict(),
			})
			return goipyObject.None, nil
		},
	})
	return cls
}

func buildRequestClass(_ *goipyVM.Interp) *goipyObject.Class {
	cls := &goipyObject.Class{Name: "Request", Dict: goipyObject.NewDict()}
	cls.Dict.SetStr("__init__", &goipyObject.BuiltinFunc{
		Name: "__init__",
		Call: func(self any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			inst := self.(*goipyObject.Instance)
			if len(args) < 1 {
				return nil, fmt.Errorf("Request() requires a url argument")
			}
			urlStr, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("Request(): url must be str")
			}
			inst.Dict.SetStr("url", urlStr)
			method := &goipyObject.Str{V: "GET"}
			if kwargs != nil {
				if v, ok := kwargs.GetStr("method"); ok {
					if s, ok := v.(*goipyObject.Str); ok {
						method = &goipyObject.Str{V: strings.ToUpper(s.V)}
					}
				}
			}
			inst.Dict.SetStr("method", method)
			return goipyObject.None, nil
		},
	})
	return cls
}

func buildURLClass(_ *goipyVM.Interp) *goipyObject.Class {
	cls := &goipyObject.Class{Name: "URL", Dict: goipyObject.NewDict()}
	cls.Dict.SetStr("__init__", &goipyObject.BuiltinFunc{
		Name: "__init__",
		Call: func(self any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			inst := self.(*goipyObject.Instance)
			if len(args) < 1 {
				return nil, fmt.Errorf("URL() requires a href argument")
			}
			hrefStr, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("URL(): href must be str")
			}
			u, err := url.Parse(hrefStr.V)
			if err != nil {
				return nil, fmt.Errorf("URL(): invalid URL: %w", err)
			}
			inst.Dict.SetStr("href", hrefStr)
			inst.Dict.SetStr("protocol", &goipyObject.Str{V: u.Scheme + ":"})
			inst.Dict.SetStr("host", &goipyObject.Str{V: u.Host})
			inst.Dict.SetStr("hostname", &goipyObject.Str{V: u.Hostname()})
			port := u.Port()
			inst.Dict.SetStr("port", &goipyObject.Str{V: port})
			inst.Dict.SetStr("pathname", &goipyObject.Str{V: u.Path})
			inst.Dict.SetStr("search", &goipyObject.Str{V: func() string {
				if u.RawQuery != "" {
					return "?" + u.RawQuery
				}
				return ""
			}()})
			inst.Dict.SetStr("hash", &goipyObject.Str{V: func() string {
				if u.Fragment != "" {
					return "#" + u.Fragment
				}
				return ""
			}()})
			inst.Dict.SetStr("origin", &goipyObject.Str{V: u.Scheme + "://" + u.Host})
			inst.Dict.SetStr("__str__", &goipyObject.BuiltinFunc{
				Name: "__str__",
				Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
					return hrefStr, nil
				},
			})
			return goipyObject.None, nil
		},
	})
	return cls
}

func buildHeadersClass(_ *goipyVM.Interp) *goipyObject.Class {
	cls := &goipyObject.Class{Name: "Headers", Dict: goipyObject.NewDict()}
	cls.Dict.SetStr("get", &goipyObject.BuiltinFunc{
		Name: "get",
		Call: func(self any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			inst := self.(*goipyObject.Instance)
			if len(args) != 1 {
				return nil, fmt.Errorf("Headers.get() takes 1 argument")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("Headers.get(): key must be str")
			}
			v, ok := inst.Dict.GetStr(strings.ToLower(key.V))
			if !ok {
				return goipyObject.None, nil
			}
			return v, nil
		},
	})
	return cls
}

// jsonToObject converts a Go JSON value to a goipy object.
func jsonToObject(v any) goipyObject.Object {
	if v == nil {
		return goipyObject.None
	}
	switch x := v.(type) {
	case bool:
		return goipyObject.BoolOf(x)
	case float64:
		if x == float64(int64(x)) {
			return goipyObject.NewInt(int64(x))
		}
		return &goipyObject.Float{V: x}
	case string:
		return &goipyObject.Str{V: x}
	case []any:
		items := make([]goipyObject.Object, len(x))
		for i, item := range x {
			items[i] = jsonToObject(item)
		}
		return &goipyObject.List{V: items}
	case map[string]any:
		d := goipyObject.NewDict()
		for k, val := range x {
			d.SetStr(k, jsonToObject(val))
		}
		return d
	}
	return goipyObject.None
}

package bunpy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildHTTP(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.http", Dict: goipyObject.NewDict()}

	doRequest := func(method, rawURL string, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
		return httpDoRequest(&http.Client{Timeout: 30 * time.Second}, method, rawURL, "", kwargs)
	}

	for _, m := range []string{"get", "post", "put", "delete", "patch", "head"} {
		m := m
		mod.Dict.SetStr(m, &goipyObject.BuiltinFunc{
			Name: m,
			Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
				if len(args) < 1 {
					return nil, fmt.Errorf("http.%s() requires a URL argument", m)
				}
				u, ok := args[0].(*goipyObject.Str)
				if !ok {
					return nil, fmt.Errorf("http.%s(): URL must be str", m)
				}
				return doRequest(strings.ToUpper(m), u.V, kwargs)
			},
		})
	}

	mod.Dict.SetStr("session", &goipyObject.BuiltinFunc{
		Name: "session",
		Call: func(_ any, _ []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			baseURL := ""
			timeout := 30 * time.Second
			retries := 0
			headers := map[string]string{}

			if kwargs != nil {
				if v, ok := kwargs.GetStr("base_url"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						baseURL = s.V
					}
				}
				if v, ok := kwargs.GetStr("timeout"); ok {
					switch tv := v.(type) {
					case *goipyObject.Int:
						timeout = time.Duration(tv.Int64()) * time.Second
					case *goipyObject.Float:
						timeout = time.Duration(tv.V * float64(time.Second))
					}
				}
				if v, ok := kwargs.GetStr("retries"); ok {
					if iv, ok2 := v.(*goipyObject.Int); ok2 {
						retries = int(iv.Int64())
					}
				}
				if v, ok := kwargs.GetStr("headers"); ok {
					if d, ok2 := v.(*goipyObject.Dict); ok2 {
						keys, vals := d.Items()
						for j, k := range keys {
							if ks, ok3 := k.(*goipyObject.Str); ok3 {
								if vs, ok4 := vals[j].(*goipyObject.Str); ok4 {
									headers[ks.V] = vs.V
								}
							}
						}
					}
				}
			}

			jar, _ := cookiejar.New(nil)
			client := &http.Client{
				Timeout:   timeout,
				Jar:       jar,
				Transport: http.DefaultTransport,
			}
			return buildSessionInstance(client, baseURL, headers, retries), nil
		},
	})

	return mod
}

func httpDoRequest(client *http.Client, method, rawURL, baseURL string, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
	// resolve relative URL against base
	if baseURL != "" && !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		base, err := url.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("http: invalid base URL: %w", err)
		}
		ref, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("http: invalid URL: %w", err)
		}
		rawURL = base.ResolveReference(ref).String()
	}

	var bodyReader io.Reader
	extraHeaders := map[string]string{}

	if kwargs != nil {
		if v, ok := kwargs.GetStr("json"); ok {
			if d, ok2 := v.(*goipyObject.Dict); ok2 {
				b, err := json.Marshal(pyDictToGoMap(d))
				if err != nil {
					return nil, fmt.Errorf("http: failed to marshal json body: %w", err)
				}
				bodyReader = bytes.NewReader(b)
				extraHeaders["Content-Type"] = "application/json"
			}
		}
		if v, ok := kwargs.GetStr("body"); ok {
			switch bv := v.(type) {
			case *goipyObject.Bytes:
				bodyReader = bytes.NewReader(bv.V)
			case *goipyObject.Str:
				bodyReader = strings.NewReader(bv.V)
			}
		}
		if v, ok := kwargs.GetStr("headers"); ok {
			if d, ok2 := v.(*goipyObject.Dict); ok2 {
				keys, vals := d.Items()
				for j, k := range keys {
					if ks, ok3 := k.(*goipyObject.Str); ok3 {
						if vs, ok4 := vals[j].(*goipyObject.Str); ok4 {
							extraHeaders[ks.V] = vs.V
						}
					}
				}
			}
		}
	}

	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("http.%s(): %w", strings.ToLower(method), err)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http.%s(): %w", strings.ToLower(method), err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return buildResponseInstance(resp.StatusCode, resp.Header, body), nil
}

func buildResponseInstance(status int, header http.Header, body []byte) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Response", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	inst.Dict.SetStr("status", goipyObject.NewInt(int64(status)))
	inst.Dict.SetStr("ok", goipyObject.BoolOf(status >= 200 && status < 300))

	hdr := goipyObject.NewDict()
	for k, vs := range header {
		if len(vs) > 0 {
			hdr.SetStr(k, &goipyObject.Str{V: vs[0]})
		}
	}
	inst.Dict.SetStr("headers", hdr)

	inst.Dict.SetStr("text", &goipyObject.BuiltinFunc{
		Name: "text",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return &goipyObject.Str{V: string(body)}, nil
		},
	})
	inst.Dict.SetStr("bytes", &goipyObject.BuiltinFunc{
		Name: "bytes",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return &goipyObject.Bytes{V: body}, nil
		},
	})
	inst.Dict.SetStr("json", &goipyObject.BuiltinFunc{
		Name: "json",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			var v any
			if err := json.Unmarshal(body, &v); err != nil {
				return nil, fmt.Errorf("response.json(): %w", err)
			}
			return goValueToPyObj(v), nil
		},
	})

	return inst
}

func buildSessionInstance(client *http.Client, baseURL string, defaultHeaders map[string]string, retries int) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Session", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	doWithRetry := func(method, rawURL string, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
		// merge default headers into kwargs
		merged := mergeHeaders(kwargs, defaultHeaders)
		var lastErr error
		for attempt := 0; attempt <= retries; attempt++ {
			resp, err := httpDoRequest(client, method, rawURL, baseURL, merged)
			if err != nil {
				lastErr = err
				if attempt < retries {
					time.Sleep(time.Duration(100<<uint(attempt)) * time.Millisecond)
				}
				continue
			}
			return resp, nil
		}
		return nil, lastErr
	}

	for _, m := range []string{"get", "post", "put", "delete", "patch", "head"} {
		m := m
		inst.Dict.SetStr(m, &goipyObject.BuiltinFunc{
			Name: m,
			Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
				if len(args) < 1 {
					return nil, fmt.Errorf("session.%s() requires a URL argument", m)
				}
				u, ok := args[0].(*goipyObject.Str)
				if !ok {
					return nil, fmt.Errorf("session.%s(): URL must be str", m)
				}
				return doWithRetry(strings.ToUpper(m), u.V, kwargs)
			},
		})
	}

	inst.Dict.SetStr("close", &goipyObject.BuiltinFunc{
		Name: "close",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("__enter__", &goipyObject.BuiltinFunc{
		Name: "__enter__",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return inst, nil
		},
	})
	inst.Dict.SetStr("__exit__", &goipyObject.BuiltinFunc{
		Name: "__exit__",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.BoolOf(false), nil
		},
	})

	return inst
}

func mergeHeaders(kwargs *goipyObject.Dict, defaults map[string]string) *goipyObject.Dict {
	merged := goipyObject.NewDict()
	// copy existing kwargs
	if kwargs != nil {
		keys, vals := kwargs.Items()
		for j, k := range keys {
			if ks, ok := k.(*goipyObject.Str); ok {
				merged.SetStr(ks.V, vals[j])
			}
		}
	}
	if len(defaults) == 0 {
		return merged
	}
	// build or extend headers dict
	existing, hasHeaders := merged.GetStr("headers")
	var hd *goipyObject.Dict
	if hasHeaders {
		if d, ok := existing.(*goipyObject.Dict); ok {
			hd = d
		} else {
			hd = goipyObject.NewDict()
		}
	} else {
		hd = goipyObject.NewDict()
	}
	for k, v := range defaults {
		if _, already := hd.GetStr(k); !already {
			hd.SetStr(k, &goipyObject.Str{V: v})
		}
	}
	merged.SetStr("headers", hd)
	return merged
}

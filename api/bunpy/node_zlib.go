package bunpy

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNodeZlib builds the bunpy.node.zlib module.
func BuildNodeZlib(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.zlib", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("gzip", &goipyObject.BuiltinFunc{
		Name: "gzip",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			data, err := zlibToBytes(args, "zlib.gzip")
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			w := gzip.NewWriter(&buf)
			if _, err := w.Write(data); err != nil {
				return nil, err
			}
			if err := w.Close(); err != nil {
				return nil, err
			}
			return &goipyObject.Bytes{V: buf.Bytes()}, nil
		},
	})

	mod.Dict.SetStr("gunzip", &goipyObject.BuiltinFunc{
		Name: "gunzip",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			data, err := zlibToBytes(args, "zlib.gunzip")
			if err != nil {
				return nil, err
			}
			r, err := gzip.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			defer r.Close()
			out, err := io.ReadAll(r)
			if err != nil {
				return nil, err
			}
			return &goipyObject.Bytes{V: out}, nil
		},
	})

	mod.Dict.SetStr("deflate", &goipyObject.BuiltinFunc{
		Name: "deflate",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			data, err := zlibToBytes(args, "zlib.deflate")
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			w, err := flate.NewWriter(&buf, flate.DefaultCompression)
			if err != nil {
				return nil, err
			}
			if _, err := w.Write(data); err != nil {
				return nil, err
			}
			if err := w.Close(); err != nil {
				return nil, err
			}
			return &goipyObject.Bytes{V: buf.Bytes()}, nil
		},
	})

	mod.Dict.SetStr("inflate", &goipyObject.BuiltinFunc{
		Name: "inflate",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			data, err := zlibToBytes(args, "zlib.inflate")
			if err != nil {
				return nil, err
			}
			r := flate.NewReader(bytes.NewReader(data))
			defer r.Close()
			out, err := io.ReadAll(r)
			if err != nil {
				return nil, err
			}
			return &goipyObject.Bytes{V: out}, nil
		},
	})

	mod.Dict.SetStr("deflateRaw", &goipyObject.BuiltinFunc{
		Name: "deflateRaw",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			data, err := zlibToBytes(args, "zlib.deflateRaw")
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			w, err := flate.NewWriter(&buf, flate.DefaultCompression)
			if err != nil {
				return nil, err
			}
			if _, err := w.Write(data); err != nil {
				return nil, err
			}
			if err := w.Close(); err != nil {
				return nil, err
			}
			return &goipyObject.Bytes{V: buf.Bytes()}, nil
		},
	})

	mod.Dict.SetStr("inflateRaw", &goipyObject.BuiltinFunc{
		Name: "inflateRaw",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			data, err := zlibToBytes(args, "zlib.inflateRaw")
			if err != nil {
				return nil, err
			}
			r := flate.NewReader(bytes.NewReader(data))
			defer r.Close()
			out, err := io.ReadAll(r)
			if err != nil {
				return nil, err
			}
			return &goipyObject.Bytes{V: out}, nil
		},
	})

	mod.Dict.SetStr("deflateSync", zlibSyncAlias("deflate"))
	mod.Dict.SetStr("inflateSync", zlibSyncAlias("inflate"))
	mod.Dict.SetStr("gzipSync", zlibSyncAlias("gzip"))
	mod.Dict.SetStr("gunzipSync", zlibSyncAlias("gunzip"))

	mod.Dict.SetStr("createGzip", &goipyObject.BuiltinFunc{
		Name: "createGzip",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newZlibTransformInstance("gzip"), nil
		},
	})

	mod.Dict.SetStr("createGunzip", &goipyObject.BuiltinFunc{
		Name: "createGunzip",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newZlibTransformInstance("gunzip"), nil
		},
	})

	mod.Dict.SetStr("createDeflate", &goipyObject.BuiltinFunc{
		Name: "createDeflate",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newZlibTransformInstance("deflate"), nil
		},
	})

	mod.Dict.SetStr("createInflate", &goipyObject.BuiltinFunc{
		Name: "createInflate",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newZlibTransformInstance("inflate"), nil
		},
	})

	return mod
}

func zlibToBytes(args []goipyObject.Object, fnName string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("%s() requires data", fnName)
	}
	switch v := args[0].(type) {
	case *goipyObject.Str:
		return []byte(v.V), nil
	case *goipyObject.Bytes:
		return v.V, nil
	default:
		return nil, fmt.Errorf("%s(): data must be str or bytes", fnName)
	}
}

func zlibSyncAlias(kind string) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: kind + "Sync",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			data, err := zlibToBytes(args, "zlib."+kind+"Sync")
			if err != nil {
				return nil, err
			}
			switch kind {
			case "gzip":
				var buf bytes.Buffer
				w := gzip.NewWriter(&buf)
				w.Write(data)
				w.Close()
				return &goipyObject.Bytes{V: buf.Bytes()}, nil
			case "gunzip":
				r, err := gzip.NewReader(bytes.NewReader(data))
				if err != nil {
					return nil, err
				}
				out, err := io.ReadAll(r)
				r.Close()
				return &goipyObject.Bytes{V: out}, err
			case "deflate":
				var buf bytes.Buffer
				w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
				w.Write(data)
				w.Close()
				return &goipyObject.Bytes{V: buf.Bytes()}, nil
			case "inflate":
				r := flate.NewReader(bytes.NewReader(data))
				out, err := io.ReadAll(r)
				r.Close()
				return &goipyObject.Bytes{V: out}, err
			}
			return nil, fmt.Errorf("unknown zlib kind: %s", kind)
		},
	}
}

func newZlibTransformInstance(kind string) *goipyObject.Instance {
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Transform"},
		Dict:  goipyObject.NewDict(),
	}
	var inputBuf bytes.Buffer
	inst.Dict.SetStr("write", &goipyObject.BuiltinFunc{
		Name: "write",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.BoolOf(true), nil
			}
			switch v := args[0].(type) {
			case *goipyObject.Str:
				inputBuf.WriteString(v.V)
			case *goipyObject.Bytes:
				inputBuf.Write(v.V)
			}
			return goipyObject.BoolOf(true), nil
		},
	})
	inst.Dict.SetStr("flush", &goipyObject.BuiltinFunc{
		Name: "flush",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			data := inputBuf.Bytes()
			switch kind {
			case "gzip":
				var buf bytes.Buffer
				w := gzip.NewWriter(&buf)
				w.Write(data)
				w.Close()
				return &goipyObject.Bytes{V: buf.Bytes()}, nil
			case "gunzip":
				r, err := gzip.NewReader(bytes.NewReader(data))
				if err != nil {
					return nil, err
				}
				out, err := io.ReadAll(r)
				r.Close()
				return &goipyObject.Bytes{V: out}, err
			case "deflate":
				var buf bytes.Buffer
				w2, _ := flate.NewWriter(&buf, flate.DefaultCompression)
				w2.Write(data)
				w2.Close()
				return &goipyObject.Bytes{V: buf.Bytes()}, nil
			case "inflate":
				r2 := flate.NewReader(bytes.NewReader(data))
				out, err := io.ReadAll(r2)
				r2.Close()
				return &goipyObject.Bytes{V: out}, err
			}
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("pipe", &goipyObject.BuiltinFunc{
		Name: "pipe",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("stream.pipe() requires destination")
			}
			dst, ok := args[0].(*goipyObject.Instance)
			if !ok {
				return nil, fmt.Errorf("stream.pipe(): destination must be a Writable")
			}
			data := inputBuf.Bytes()
			var out []byte
			switch kind {
			case "gzip":
				var buf bytes.Buffer
				w := gzip.NewWriter(&buf)
				w.Write(data)
				w.Close()
				out = buf.Bytes()
			case "gunzip":
				r, err := gzip.NewReader(bytes.NewReader(data))
				if err != nil {
					return nil, err
				}
				out, err = io.ReadAll(r)
				r.Close()
				if err != nil {
					return nil, err
				}
			case "deflate":
				var buf bytes.Buffer
				w2, _ := flate.NewWriter(&buf, flate.DefaultCompression)
				w2.Write(data)
				w2.Close()
				out = buf.Bytes()
			case "inflate":
				r2 := flate.NewReader(bytes.NewReader(data))
				var err error
				out, err = io.ReadAll(r2)
				r2.Close()
				if err != nil {
					return nil, err
				}
			}
			if write, ok := dst.Dict.GetStr("write"); ok {
				if fn, ok := write.(*goipyObject.BuiltinFunc); ok {
					fn.Call(nil, []goipyObject.Object{&goipyObject.Bytes{V: out}}, nil)
				}
			}
			return args[0], nil
		},
	})
	return inst
}

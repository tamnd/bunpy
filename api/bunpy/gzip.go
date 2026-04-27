package bunpy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildGzip builds the bunpy.gzip module.
//
// Python surface:
//
//	bunpy.gzip.compress(data: bytes, level: int = -1) -> bytes
//	bunpy.gzip.decompress(data: bytes) -> bytes
func BuildGzip(_ *goipyVM.Interp) *goipyObject.Module {
	m := &goipyObject.Module{Name: "bunpy.gzip", Dict: goipyObject.NewDict()}

	m.Dict.SetStr("compress", &goipyObject.BuiltinFunc{
		Name: "compress",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 || len(args) > 2 {
				return nil, fmt.Errorf("bunpy.gzip.compress() takes 1 or 2 arguments (%d given)", len(args))
			}
			data, ok := args[0].(*goipyObject.Bytes)
			if !ok {
				return nil, fmt.Errorf("bunpy.gzip.compress(): argument must be bytes, not %T", args[0])
			}
			level := gzip.DefaultCompression
			if len(args) == 2 {
				n, ok := args[1].(*goipyObject.Int)
				if !ok {
					return nil, fmt.Errorf("bunpy.gzip.compress(): level must be int")
				}
				level = int(n.Int64())
			}
			var buf bytes.Buffer
			w, err := gzip.NewWriterLevel(&buf, level)
			if err != nil {
				return nil, fmt.Errorf("bunpy.gzip.compress(): invalid level %d", level)
			}
			if _, err := w.Write(data.V); err != nil {
				return nil, fmt.Errorf("bunpy.gzip.compress(): %w", err)
			}
			if err := w.Close(); err != nil {
				return nil, fmt.Errorf("bunpy.gzip.compress(): %w", err)
			}
			return &goipyObject.Bytes{V: buf.Bytes()}, nil
		},
	})

	m.Dict.SetStr("decompress", &goipyObject.BuiltinFunc{
		Name: "decompress",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("bunpy.gzip.decompress() takes exactly 1 argument (%d given)", len(args))
			}
			data, ok := args[0].(*goipyObject.Bytes)
			if !ok {
				return nil, fmt.Errorf("bunpy.gzip.decompress(): argument must be bytes, not %T", args[0])
			}
			r, err := gzip.NewReader(bytes.NewReader(data.V))
			if err != nil {
				return nil, fmt.Errorf("bunpy.gzip.decompress(): %w", err)
			}
			defer r.Close()
			out, err := io.ReadAll(r)
			if err != nil {
				return nil, fmt.Errorf("bunpy.gzip.decompress(): %w", err)
			}
			return &goipyObject.Bytes{V: out}, nil
		},
	})

	return m
}

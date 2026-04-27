package bunpy

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNodeStream builds the bunpy.node.stream module.
func BuildNodeStream(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.stream", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("Readable", &goipyObject.BuiltinFunc{
		Name: "Readable",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newReadableInstance(nil), nil
		},
	})

	mod.Dict.SetStr("Writable", &goipyObject.BuiltinFunc{
		Name: "Writable",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newWritableInstance(), nil
		},
	})

	mod.Dict.SetStr("PassThrough", &goipyObject.BuiltinFunc{
		Name: "PassThrough",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newPassThroughInstance(), nil
		},
	})

	mod.Dict.SetStr("Transform", &goipyObject.BuiltinFunc{
		Name: "Transform",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newPassThroughInstance(), nil
		},
	})

	return mod
}

func newReadableInstance(data []byte) *goipyObject.Instance {
	type state struct {
		mu   sync.Mutex
		buf  *bytes.Reader
		done bool
	}
	st := &state{buf: bytes.NewReader(data)}
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Readable"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("read", &goipyObject.BuiltinFunc{
		Name: "read",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			n := -1
			if len(args) >= 1 {
				if v, ok := args[0].(*goipyObject.Int); ok {
					n = int(v.Int64())
				}
			}
			st.mu.Lock()
			defer st.mu.Unlock()
			if st.done {
				return goipyObject.None, nil
			}
			var chunk []byte
			var err error
			if n < 0 {
				chunk, err = io.ReadAll(st.buf)
			} else {
				chunk = make([]byte, n)
				nr, rerr := st.buf.Read(chunk)
				chunk = chunk[:nr]
				err = rerr
			}
			if err == io.EOF || len(chunk) == 0 {
				st.done = true
				return goipyObject.None, nil
			}
			if err != nil {
				return nil, err
			}
			return &goipyObject.Bytes{V: chunk}, nil
		},
	})
	inst.Dict.SetStr("push", &goipyObject.BuiltinFunc{
		Name: "push",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.None, nil
			}
			var data []byte
			switch v := args[0].(type) {
			case *goipyObject.Str:
				data = []byte(v.V)
			case *goipyObject.Bytes:
				data = v.V
			case *goipyObject.NoneType:
				st.mu.Lock()
				st.done = true
				st.mu.Unlock()
				return goipyObject.None, nil
			}
			st.mu.Lock()
			existing, _ := io.ReadAll(st.buf)
			combined := append(existing, data...)
			st.buf = bytes.NewReader(combined)
			st.mu.Unlock()
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
			st.mu.Lock()
			all, _ := io.ReadAll(st.buf)
			st.done = true
			st.mu.Unlock()
			if write, ok := dst.Dict.GetStr("write"); ok {
				if fn, ok := write.(*goipyObject.BuiltinFunc); ok {
					fn.Call(nil, []goipyObject.Object{&goipyObject.Bytes{V: all}}, nil)
				}
			}
			return args[0], nil
		},
	})
	return inst
}

func newWritableInstance() *goipyObject.Instance {
	type state struct {
		mu  sync.Mutex
		buf bytes.Buffer
	}
	st := &state{}
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Writable"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("write", &goipyObject.BuiltinFunc{
		Name: "write",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.BoolOf(true), nil
			}
			var data []byte
			switch v := args[0].(type) {
			case *goipyObject.Str:
				data = []byte(v.V)
			case *goipyObject.Bytes:
				data = v.V
			}
			st.mu.Lock()
			st.buf.Write(data)
			st.mu.Unlock()
			return goipyObject.BoolOf(true), nil
		},
	})
	inst.Dict.SetStr("end", &goipyObject.BuiltinFunc{
		Name: "end",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) >= 1 {
				var data []byte
				switch v := args[0].(type) {
				case *goipyObject.Str:
					data = []byte(v.V)
				case *goipyObject.Bytes:
					data = v.V
				}
				if len(data) > 0 {
					st.mu.Lock()
					st.buf.Write(data)
					st.mu.Unlock()
				}
			}
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("getContents", &goipyObject.BuiltinFunc{
		Name: "getContents",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			st.mu.Lock()
			defer st.mu.Unlock()
			return &goipyObject.Bytes{V: append([]byte(nil), st.buf.Bytes()...)}, nil
		},
	})
	return inst
}

func newPassThroughInstance() *goipyObject.Instance {
	type state struct {
		mu  sync.Mutex
		buf bytes.Buffer
	}
	st := &state{}
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "PassThrough"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("write", &goipyObject.BuiltinFunc{
		Name: "write",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.BoolOf(true), nil
			}
			var data []byte
			switch v := args[0].(type) {
			case *goipyObject.Str:
				data = []byte(v.V)
			case *goipyObject.Bytes:
				data = v.V
			}
			st.mu.Lock()
			st.buf.Write(data)
			st.mu.Unlock()
			return goipyObject.BoolOf(true), nil
		},
	})
	inst.Dict.SetStr("end", &goipyObject.BuiltinFunc{
		Name: "end",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("read", &goipyObject.BuiltinFunc{
		Name: "read",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			st.mu.Lock()
			defer st.mu.Unlock()
			if st.buf.Len() == 0 {
				return goipyObject.None, nil
			}
			data := append([]byte(nil), st.buf.Bytes()...)
			st.buf.Reset()
			return &goipyObject.Bytes{V: data}, nil
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
			st.mu.Lock()
			all := append([]byte(nil), st.buf.Bytes()...)
			st.buf.Reset()
			st.mu.Unlock()
			if write, ok := dst.Dict.GetStr("write"); ok {
				if fn, ok := write.(*goipyObject.BuiltinFunc); ok {
					fn.Call(nil, []goipyObject.Object{&goipyObject.Bytes{V: all}}, nil)
				}
			}
			return args[0], nil
		},
	})
	return inst
}

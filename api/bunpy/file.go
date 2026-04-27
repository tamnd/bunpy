package bunpy

import (
	"fmt"
	"io"
	"os"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildFile adds bunpy.file, bunpy.write, and bunpy.read to the bunpy module.
func BuildFile(i *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "file",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			path, err := filePathArg("file", args, kwargs)
			if err != nil {
				return nil, err
			}
			return makeBunFile(path), nil
		},
	}
}

// BuildWrite returns the bunpy.write built-in function.
func BuildWrite(_ *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "write",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("bunpy.write() requires path and data arguments")
			}
			pathStr, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("bunpy.write(): path must be a str")
			}
			path := pathStr.V

			appendMode := false
			if kwargs != nil {
				if v, ok2 := kwargs.GetStr("append"); ok2 {
					if b, ok3 := v.(*goipyObject.Bool); ok3 {
						appendMode = b.V
					}
				}
			}

			flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
			if appendMode {
				flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
			}
			f, err := os.OpenFile(path, flag, 0o644)
			if err != nil {
				return nil, fmt.Errorf("bunpy.write(): %w", err)
			}
			defer f.Close()

			switch data := args[1].(type) {
			case *goipyObject.Str:
				_, err = io.WriteString(f, data.V)
			case *goipyObject.Bytes:
				_, err = f.Write(data.V)
			case *goipyObject.Instance:
				// BunFile instance
				if srcPath, ok2 := getBunFilePath(data); ok2 {
					src, err2 := os.Open(srcPath)
					if err2 != nil {
						return nil, fmt.Errorf("bunpy.write(): %w", err2)
					}
					defer src.Close()
					_, err = io.Copy(f, src)
				} else {
					return nil, fmt.Errorf("bunpy.write(): data must be str, bytes, or BunFile")
				}
			default:
				return nil, fmt.Errorf("bunpy.write(): data must be str, bytes, or BunFile")
			}
			if err != nil {
				return nil, fmt.Errorf("bunpy.write(): %w", err)
			}
			return goipyObject.None, nil
		},
	}
}

// BuildRead returns the bunpy.read built-in function.
func BuildRead(_ *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "read",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			path, err := filePathArg("read", args, kwargs)
			if err != nil {
				return nil, err
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("bunpy.read(): %w", err)
			}
			return &goipyObject.Bytes{V: b}, nil
		},
	}
}

// makeBunFile builds a Python BunFile instance for the given path.
func makeBunFile(path string) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "BunFile", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}
	inst.Dict.SetStr("_path", &goipyObject.Str{V: path})
	inst.Dict.SetStr("name", &goipyObject.Str{V: path})

	inst.Dict.SetStr("exists", &goipyObject.BuiltinFunc{
		Name: "exists",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			_, err := os.Stat(path)
			return goipyObject.BoolOf(err == nil), nil
		},
	})
	inst.Dict.SetStr("size", &goipyObject.BuiltinFunc{
		Name: "size",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			info, err := os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("BunFile.size: %w", err)
			}
			return goipyObject.NewInt(info.Size()), nil
		},
	})
	inst.Dict.SetStr("text", &goipyObject.BuiltinFunc{
		Name: "text",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("BunFile.text: %w", err)
			}
			return &goipyObject.Str{V: string(b)}, nil
		},
	})
	inst.Dict.SetStr("bytes", &goipyObject.BuiltinFunc{
		Name: "bytes",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("BunFile.bytes: %w", err)
			}
			return &goipyObject.Bytes{V: b}, nil
		},
	})

	return inst
}

func getBunFilePath(inst *goipyObject.Instance) (string, bool) {
	if inst.Class == nil || inst.Class.Name != "BunFile" {
		return "", false
	}
	v, ok := inst.Dict.GetStr("_path")
	if !ok {
		return "", false
	}
	s, ok := v.(*goipyObject.Str)
	return s.V, ok
}

func filePathArg(fn string, args []goipyObject.Object, kwargs *goipyObject.Dict) (string, error) {
	var pathObj goipyObject.Object
	if len(args) >= 1 {
		pathObj = args[0]
	} else if kwargs != nil {
		pathObj, _ = kwargs.GetStr("path")
	}
	if pathObj == nil {
		return "", fmt.Errorf("bunpy.%s() requires a path argument", fn)
	}
	s, ok := pathObj.(*goipyObject.Str)
	if !ok {
		return "", fmt.Errorf("bunpy.%s(): path must be a str", fn)
	}
	return s.V, nil
}

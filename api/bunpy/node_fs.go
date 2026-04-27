package bunpy

import (
	"fmt"
	"io"
	"os"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNodeFS builds the bunpy.node.fs module (Node.js fs API shape).
func BuildNodeFS(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.fs", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("readFile", &goipyObject.BuiltinFunc{
		Name: "readFile",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("fs.readFile() requires path")
			}
			path, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fs.readFile(): path must be str")
			}
			data, err := os.ReadFile(path.V)
			if err != nil {
				return nil, err
			}
			// If encoding="utf8" kwarg or second arg is str, return str.
			enc := ""
			if len(args) >= 2 {
				if s, ok := args[1].(*goipyObject.Str); ok {
					enc = s.V
				}
			}
			if kwargs != nil {
				if v, ok := kwargs.GetStr("encoding"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						enc = s.V
					}
				}
			}
			if enc != "" {
				return &goipyObject.Str{V: string(data)}, nil
			}
			return &goipyObject.Bytes{V: data}, nil
		},
	})

	mod.Dict.SetStr("writeFile", &goipyObject.BuiltinFunc{
		Name: "writeFile",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("fs.writeFile() requires path and data")
			}
			path, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fs.writeFile(): path must be str")
			}
			var data []byte
			switch v := args[1].(type) {
			case *goipyObject.Str:
				data = []byte(v.V)
			case *goipyObject.Bytes:
				data = v.V
			default:
				data = []byte(fmt.Sprintf("%v", args[1]))
			}
			return goipyObject.None, os.WriteFile(path.V, data, 0o644)
		},
	})

	mod.Dict.SetStr("appendFile", &goipyObject.BuiltinFunc{
		Name: "appendFile",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("fs.appendFile() requires path and data")
			}
			path, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fs.appendFile(): path must be str")
			}
			var data []byte
			switch v := args[1].(type) {
			case *goipyObject.Str:
				data = []byte(v.V)
			case *goipyObject.Bytes:
				data = v.V
			}
			f, err := os.OpenFile(path.V, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			_, err = f.Write(data)
			return goipyObject.None, err
		},
	})

	mod.Dict.SetStr("exists", &goipyObject.BuiltinFunc{
		Name: "exists",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("fs.exists() requires path")
			}
			path, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fs.exists(): path must be str")
			}
			_, err := os.Stat(path.V)
			return goipyObject.BoolOf(err == nil), nil
		},
	})

	mod.Dict.SetStr("mkdir", &goipyObject.BuiltinFunc{
		Name: "mkdir",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("fs.mkdir() requires path")
			}
			path, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fs.mkdir(): path must be str")
			}
			recursive := false
			if kwargs != nil {
				if v, ok := kwargs.GetStr("recursive"); ok {
					if b, ok2 := v.(*goipyObject.Bool); ok2 {
						recursive = b.V
					}
				}
			}
			if recursive {
				return goipyObject.None, os.MkdirAll(path.V, 0o755)
			}
			return goipyObject.None, os.Mkdir(path.V, 0o755)
		},
	})

	mod.Dict.SetStr("mkdtemp", &goipyObject.BuiltinFunc{
		Name: "mkdtemp",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			prefix := "bunpy-"
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					prefix = s.V
				}
			}
			dir, err := os.MkdirTemp("", prefix)
			if err != nil {
				return nil, err
			}
			return &goipyObject.Str{V: dir}, nil
		},
	})

	mod.Dict.SetStr("unlink", &goipyObject.BuiltinFunc{
		Name: "unlink",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("fs.unlink() requires path")
			}
			path, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fs.unlink(): path must be str")
			}
			return goipyObject.None, os.Remove(path.V)
		},
	})

	mod.Dict.SetStr("rename", &goipyObject.BuiltinFunc{
		Name: "rename",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("fs.rename() requires src and dst")
			}
			src, ok1 := args[0].(*goipyObject.Str)
			dst, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("fs.rename(): paths must be str")
			}
			return goipyObject.None, os.Rename(src.V, dst.V)
		},
	})

	mod.Dict.SetStr("readdir", &goipyObject.BuiltinFunc{
		Name: "readdir",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("fs.readdir() requires path")
			}
			path, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fs.readdir(): path must be str")
			}
			entries, err := os.ReadDir(path.V)
			if err != nil {
				return nil, err
			}
			items := make([]goipyObject.Object, len(entries))
			for i, e := range entries {
				items[i] = &goipyObject.Str{V: e.Name()}
			}
			return &goipyObject.List{V: items}, nil
		},
	})

	mod.Dict.SetStr("stat", &goipyObject.BuiltinFunc{
		Name: "stat",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("fs.stat() requires path")
			}
			path, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fs.stat(): path must be str")
			}
			info, err := os.Stat(path.V)
			if err != nil {
				return nil, err
			}
			d := goipyObject.NewDict()
			d.SetStr("size", goipyObject.NewInt(info.Size()))
			d.SetStr("isDirectory", goipyObject.BoolOf(info.IsDir()))
			d.SetStr("isFile", goipyObject.BoolOf(!info.IsDir()))
			d.SetStr("mtime", goipyObject.NewInt(info.ModTime().Unix()))
			d.SetStr("name", &goipyObject.Str{V: info.Name()})
			return &goipyObject.Instance{Class: &goipyObject.Class{Name: "Stats"}, Dict: d}, nil
		},
	})

	mod.Dict.SetStr("copyFile", &goipyObject.BuiltinFunc{
		Name: "copyFile",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("fs.copyFile() requires src and dst")
			}
			src, ok1 := args[0].(*goipyObject.Str)
			dst, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("fs.copyFile(): paths must be str")
			}
			sf, err := os.Open(src.V)
			if err != nil {
				return nil, err
			}
			defer sf.Close()
			df, err := os.Create(dst.V)
			if err != nil {
				return nil, err
			}
			defer df.Close()
			_, err = io.Copy(df, sf)
			return goipyObject.None, err
		},
	})

	mod.Dict.SetStr("rmdir", &goipyObject.BuiltinFunc{
		Name: "rmdir",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("fs.rmdir() requires path")
			}
			path, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("fs.rmdir(): path must be str")
			}
			recursive := false
			if kwargs != nil {
				if v, ok := kwargs.GetStr("recursive"); ok {
					if b, ok2 := v.(*goipyObject.Bool); ok2 {
						recursive = b.V
					}
				}
			}
			if recursive {
				return goipyObject.None, os.RemoveAll(path.V)
			}
			return goipyObject.None, os.Remove(path.V)
		},
	})

	return mod
}

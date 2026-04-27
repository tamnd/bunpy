package bunpy

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNodePath builds the bunpy.node.path module (Node.js path API shape).
func BuildNodePath(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.path", Dict: goipyObject.NewDict()}

	sep := string(filepath.Separator)
	delim := ":"
	if runtime.GOOS == "windows" {
		delim = ";"
	}
	mod.Dict.SetStr("sep", &goipyObject.Str{V: sep})
	mod.Dict.SetStr("delimiter", &goipyObject.Str{V: delim})

	strArgs := func(args []goipyObject.Object) ([]string, error) {
		out := make([]string, len(args))
		for i, a := range args {
			s, ok := a.(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("path: argument %d must be str", i)
			}
			out[i] = s.V
		}
		return out, nil
	}

	mod.Dict.SetStr("join", &goipyObject.BuiltinFunc{
		Name: "join",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			parts, err := strArgs(args)
			if err != nil {
				return nil, err
			}
			return &goipyObject.Str{V: filepath.Join(parts...)}, nil
		},
	})

	mod.Dict.SetStr("resolve", &goipyObject.BuiltinFunc{
		Name: "resolve",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			parts, err := strArgs(args)
			if err != nil {
				return nil, err
			}
			p, err := filepath.Abs(filepath.Join(parts...))
			if err != nil {
				return nil, err
			}
			return &goipyObject.Str{V: p}, nil
		},
	})

	mod.Dict.SetStr("dirname", &goipyObject.BuiltinFunc{
		Name: "dirname",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("path.dirname() requires path")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("path.dirname(): path must be str")
			}
			return &goipyObject.Str{V: filepath.Dir(s.V)}, nil
		},
	})

	mod.Dict.SetStr("basename", &goipyObject.BuiltinFunc{
		Name: "basename",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("path.basename() requires path")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("path.basename(): path must be str")
			}
			base := filepath.Base(s.V)
			// Optional ext to strip.
			if len(args) >= 2 {
				if ext, ok := args[1].(*goipyObject.Str); ok && ext.V != "" {
					base = strings.TrimSuffix(base, ext.V)
				}
			}
			return &goipyObject.Str{V: base}, nil
		},
	})

	mod.Dict.SetStr("extname", &goipyObject.BuiltinFunc{
		Name: "extname",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("path.extname() requires path")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("path.extname(): path must be str")
			}
			return &goipyObject.Str{V: filepath.Ext(s.V)}, nil
		},
	})

	mod.Dict.SetStr("relative", &goipyObject.BuiltinFunc{
		Name: "relative",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("path.relative() requires from and to")
			}
			from, ok1 := args[0].(*goipyObject.Str)
			to, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("path.relative(): paths must be str")
			}
			rel, err := filepath.Rel(from.V, to.V)
			if err != nil {
				return nil, err
			}
			return &goipyObject.Str{V: rel}, nil
		},
	})

	mod.Dict.SetStr("isAbsolute", &goipyObject.BuiltinFunc{
		Name: "isAbsolute",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("path.isAbsolute() requires path")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("path.isAbsolute(): path must be str")
			}
			return goipyObject.BoolOf(filepath.IsAbs(s.V)), nil
		},
	})

	mod.Dict.SetStr("normalize", &goipyObject.BuiltinFunc{
		Name: "normalize",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("path.normalize() requires path")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("path.normalize(): path must be str")
			}
			return &goipyObject.Str{V: filepath.Clean(s.V)}, nil
		},
	})

	return mod
}

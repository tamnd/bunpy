package bunpy

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildEnv(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.env", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("load", &goipyObject.BuiltinFunc{
		Name: "load",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			path := ".env"
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					path = s.V
				}
			}
			return goipyObject.None, loadDotenv(path)
		},
	})

	mod.Dict.SetStr("get", &goipyObject.BuiltinFunc{
		Name: "get",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("env.get() requires a key argument")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("env.get(): key must be str")
			}
			val, exists := os.LookupEnv(key.V)
			if !exists {
				if len(args) >= 2 {
					return args[1], nil
				}
				return goipyObject.None, nil
			}
			return &goipyObject.Str{V: val}, nil
		},
	})

	mod.Dict.SetStr("int", &goipyObject.BuiltinFunc{
		Name: "int",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("env.int() requires a key argument")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("env.int(): key must be str")
			}
			val, exists := os.LookupEnv(key.V)
			if !exists {
				if len(args) >= 2 {
					return args[1], nil
				}
				return goipyObject.NewInt(0), nil
			}
			n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("env.int(): %q is not an integer", val)
			}
			return goipyObject.NewInt(n), nil
		},
	})

	mod.Dict.SetStr("float", &goipyObject.BuiltinFunc{
		Name: "float",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("env.float() requires a key argument")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("env.float(): key must be str")
			}
			val, exists := os.LookupEnv(key.V)
			if !exists {
				if len(args) >= 2 {
					return args[1], nil
				}
				return &goipyObject.Float{V: 0}, nil
			}
			f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
			if err != nil {
				return nil, fmt.Errorf("env.float(): %q is not a float", val)
			}
			return &goipyObject.Float{V: f}, nil
		},
	})

	mod.Dict.SetStr("bool", &goipyObject.BuiltinFunc{
		Name: "bool",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("env.bool() requires a key argument")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("env.bool(): key must be str")
			}
			val, exists := os.LookupEnv(key.V)
			if !exists {
				if len(args) >= 2 {
					return args[1], nil
				}
				return goipyObject.BoolOf(false), nil
			}
			v := strings.ToLower(strings.TrimSpace(val))
			return goipyObject.BoolOf(v == "1" || v == "true" || v == "yes" || v == "on"), nil
		},
	})

	mod.Dict.SetStr("set", &goipyObject.BuiltinFunc{
		Name: "set",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("env.set() requires key and value arguments")
			}
			k, ok1 := args[0].(*goipyObject.Str)
			v, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("env.set(): key and value must be str")
			}
			return goipyObject.None, os.Setenv(k.V, v.V)
		},
	})

	mod.Dict.SetStr("all", &goipyObject.BuiltinFunc{
		Name: "all",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			d := goipyObject.NewDict()
			for _, pair := range os.Environ() {
				idx := strings.IndexByte(pair, '=')
				if idx < 0 {
					continue
				}
				d.SetStr(pair[:idx], &goipyObject.Str{V: pair[idx+1:]})
			}
			return d, nil
		},
	})

	return mod
}

func loadDotenv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		os.Setenv(key, val)
	}
	return sc.Err()
}

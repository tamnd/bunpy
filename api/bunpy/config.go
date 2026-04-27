package bunpy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

type configObj struct {
	data map[string]any
}

func (c *configObj) get(key string) any {
	parts := strings.Split(key, ".")
	current := c.data
	for i, part := range parts {
		v, ok := current[part]
		if !ok {
			return nil
		}
		if i == len(parts)-1 {
			return v
		}
		sub, ok2 := v.(map[string]any)
		if !ok2 {
			return nil
		}
		current = sub
	}
	return nil
}

func BuildConfig(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.config", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("load", &goipyObject.BuiltinFunc{
		Name: "load",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			merged := map[string]any{}
			for _, arg := range args {
				s, ok := arg.(*goipyObject.Str)
				if !ok {
					continue
				}
				data, err := loadConfigFile(s.V)
				if err != nil {
					return nil, fmt.Errorf("config.load(%q): %w", s.V, err)
				}
				mergeMaps(merged, data)
			}
			envPrefix := ""
			if kwargs != nil {
				if v, ok := kwargs.GetStr("env_prefix"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						envPrefix = strings.ToUpper(s.V)
					}
				}
			}
			if envPrefix != "" {
				applyEnvOverrides(merged, envPrefix)
			}
			return buildConfigInstance(&configObj{data: merged}), nil
		},
	})

	return mod
}

func loadConfigFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".toml":
		if err2 := toml.Unmarshal(data, &result); err2 != nil {
			return nil, err2
		}
	case ".json":
		if err2 := json.Unmarshal(data, &result); err2 != nil {
			return nil, err2
		}
	case ".env":
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
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
			result[key] = val
		}
	default:
		// try TOML, then JSON
		if err2 := toml.Unmarshal(data, &result); err2 != nil {
			result = map[string]any{}
			if err3 := json.Unmarshal(data, &result); err3 != nil {
				return nil, fmt.Errorf("unknown config format for %s", path)
			}
		}
	}
	return result, nil
}

func mergeMaps(dst, src map[string]any) {
	for k, v := range src {
		if srcMap, ok := v.(map[string]any); ok {
			if dstMap, ok2 := dst[k].(map[string]any); ok2 {
				mergeMaps(dstMap, srcMap)
				continue
			}
		}
		dst[k] = v
	}
}

func applyEnvOverrides(m map[string]any, prefix string) {
	for _, pair := range os.Environ() {
		idx := strings.IndexByte(pair, '=')
		if idx < 0 {
			continue
		}
		k, val := pair[:idx], pair[idx+1:]
		if !strings.HasPrefix(k, prefix+"_") {
			continue
		}
		rest := k[len(prefix)+1:]
		parts := strings.Split(strings.ToLower(rest), "_")
		setNested(m, parts, val)
	}
}

func setNested(m map[string]any, parts []string, val string) {
	if len(parts) == 1 {
		m[parts[0]] = val
		return
	}
	sub, ok := m[parts[0]].(map[string]any)
	if !ok {
		sub = map[string]any{}
		m[parts[0]] = sub
	}
	setNested(sub, parts[1:], val)
}

func buildConfigInstance(c *configObj) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Config", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	inst.Dict.SetStr("get", &goipyObject.BuiltinFunc{
		Name: "get",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("config.get() requires a key")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("config.get(): key must be str")
			}
			v := c.get(key.V)
			if v == nil {
				if len(args) >= 2 {
					return args[1], nil
				}
				return goipyObject.None, nil
			}
			return goValueToPyObj(v), nil
		},
	})

	inst.Dict.SetStr("int", &goipyObject.BuiltinFunc{
		Name: "int",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("config.int() requires a key")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("config.int(): key must be str")
			}
			v := c.get(key.V)
			if v == nil {
				if len(args) >= 2 {
					return args[1], nil
				}
				return goipyObject.NewInt(0), nil
			}
			switch tv := v.(type) {
			case int64:
				return goipyObject.NewInt(tv), nil
			case float64:
				return goipyObject.NewInt(int64(tv)), nil
			case string:
				n, err := strconv.ParseInt(strings.TrimSpace(tv), 10, 64)
				if err != nil {
					return nil, fmt.Errorf("config.int(): %q is not an integer", tv)
				}
				return goipyObject.NewInt(n), nil
			}
			return goipyObject.NewInt(0), nil
		},
	})

	inst.Dict.SetStr("bool", &goipyObject.BuiltinFunc{
		Name: "bool",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("config.bool() requires a key")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("config.bool(): key must be str")
			}
			v := c.get(key.V)
			if v == nil {
				if len(args) >= 2 {
					return args[1], nil
				}
				return goipyObject.BoolOf(false), nil
			}
			switch tv := v.(type) {
			case bool:
				return goipyObject.BoolOf(tv), nil
			case string:
				s := strings.ToLower(strings.TrimSpace(tv))
				return goipyObject.BoolOf(s == "true" || s == "1" || s == "yes" || s == "on"), nil
			}
			return goipyObject.BoolOf(false), nil
		},
	})

	inst.Dict.SetStr("float", &goipyObject.BuiltinFunc{
		Name: "float",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("config.float() requires a key")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("config.float(): key must be str")
			}
			v := c.get(key.V)
			if v == nil {
				if len(args) >= 2 {
					return args[1], nil
				}
				return &goipyObject.Float{V: 0}, nil
			}
			switch tv := v.(type) {
			case float64:
				return &goipyObject.Float{V: tv}, nil
			case int64:
				return &goipyObject.Float{V: float64(tv)}, nil
			case string:
				f, err := strconv.ParseFloat(strings.TrimSpace(tv), 64)
				if err != nil {
					return nil, fmt.Errorf("config.float(): %q is not a float", tv)
				}
				return &goipyObject.Float{V: f}, nil
			}
			return &goipyObject.Float{V: 0}, nil
		},
	})

	return inst
}

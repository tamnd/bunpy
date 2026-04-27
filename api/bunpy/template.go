package bunpy

import (
	"bytes"
	"fmt"
	htemplate "html/template"
	"os"
	ttemplate "text/template"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildTemplate(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.template", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("render", &goipyObject.BuiltinFunc{
		Name: "render",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("template.render() requires a template string")
			}
			src, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("template.render(): first argument must be str")
			}
			data := map[string]any{}
			if len(args) >= 2 {
				if d, ok2 := args[1].(*goipyObject.Dict); ok2 {
					data = pyDictToGoMap(d)
				}
			}
			htmlMode := false
			if kwargs != nil {
				if hv, ok2 := kwargs.GetStr("html"); ok2 {
					if b, ok3 := hv.(*goipyObject.Bool); ok3 {
						htmlMode = b.V
					}
				}
			}
			out, err := renderTemplate(src.V, data, htmlMode)
			if err != nil {
				return nil, fmt.Errorf("template.render(): %w", err)
			}
			return &goipyObject.Str{V: out}, nil
		},
	})

	mod.Dict.SetStr("render_file", &goipyObject.BuiltinFunc{
		Name: "render_file",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("template.render_file() requires a path")
			}
			p, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("template.render_file(): path must be str")
			}
			data, err := os.ReadFile(p.V)
			if err != nil {
				return nil, fmt.Errorf("template.render_file(): %w", err)
			}
			vars := map[string]any{}
			if len(args) >= 2 {
				if d, ok2 := args[1].(*goipyObject.Dict); ok2 {
					vars = pyDictToGoMap(d)
				}
			}
			htmlMode := false
			if kwargs != nil {
				if hv, ok2 := kwargs.GetStr("html"); ok2 {
					if b, ok3 := hv.(*goipyObject.Bool); ok3 {
						htmlMode = b.V
					}
				}
			}
			out, err := renderTemplate(string(data), vars, htmlMode)
			if err != nil {
				return nil, fmt.Errorf("template.render_file(): %w", err)
			}
			return &goipyObject.Str{V: out}, nil
		},
	})

	mod.Dict.SetStr("compile", &goipyObject.BuiltinFunc{
		Name: "compile",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("template.compile() requires a template string")
			}
			src, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("template.compile(): argument must be str")
			}
			htmlMode := false
			if kwargs != nil {
				if hv, ok2 := kwargs.GetStr("html"); ok2 {
					if b, ok3 := hv.(*goipyObject.Bool); ok3 {
						htmlMode = b.V
					}
				}
			}
			return buildCompiledTemplate(src.V, htmlMode)
		},
	})

	return mod
}

func renderTemplate(src string, data map[string]any, html bool) (string, error) {
	var buf bytes.Buffer
	if html {
		t, err := htemplate.New("t").Parse(src)
		if err != nil {
			return "", err
		}
		if err2 := t.Execute(&buf, data); err2 != nil {
			return "", err2
		}
	} else {
		t, err := ttemplate.New("t").Parse(src)
		if err != nil {
			return "", err
		}
		if err2 := t.Execute(&buf, data); err2 != nil {
			return "", err2
		}
	}
	return buf.String(), nil
}

func buildCompiledTemplate(src string, html bool) (goipyObject.Object, error) {
	// pre-parse to catch syntax errors at compile time
	if html {
		if _, err := htemplate.New("t").Parse(src); err != nil {
			return nil, fmt.Errorf("template.compile(): %w", err)
		}
	} else {
		if _, err := ttemplate.New("t").Parse(src); err != nil {
			return nil, fmt.Errorf("template.compile(): %w", err)
		}
	}

	cls := &goipyObject.Class{Name: "Template", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}
	inst.Dict.SetStr("render", &goipyObject.BuiltinFunc{
		Name: "render",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			data := map[string]any{}
			if len(args) >= 1 {
				if d, ok := args[0].(*goipyObject.Dict); ok {
					data = pyDictToGoMap(d)
				}
			}
			out, err := renderTemplate(src, data, html)
			if err != nil {
				return nil, fmt.Errorf("template.render(): %w", err)
			}
			return &goipyObject.Str{V: out}, nil
		},
	})
	return inst, nil
}

func pyDictToGoMap(d *goipyObject.Dict) map[string]any {
	keys, vals := d.Items()
	m := make(map[string]any, len(keys))
	for i, k := range keys {
		if ks, ok := k.(*goipyObject.Str); ok {
			m[ks.V] = pyObjDeep(vals[i])
		}
	}
	return m
}

// pyObjDeep converts Python objects to Go-native types, recursing into lists and dicts.
func pyObjDeep(obj goipyObject.Object) any {
	switch v := obj.(type) {
	case *goipyObject.List:
		result := make([]any, len(v.V))
		for i, item := range v.V {
			result[i] = pyObjDeep(item)
		}
		return result
	case *goipyObject.Dict:
		return pyDictToGoMap(v)
	case *goipyObject.Instance:
		// expose instance dict as a map
		return pyDictToGoMap(v.Dict)
	default:
		return pyObjToGoValue(obj)
	}
}

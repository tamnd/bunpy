package bunpy

import (
	"fmt"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildHTMLRewriter builds the bunpy.HTMLRewriter module.
// HTMLRewriter(html) returns a rewriter instance with on(), transform() methods.
func BuildHTMLRewriter(i *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.HTMLRewriter", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("HTMLRewriter", &goipyObject.BuiltinFunc{
		Name: "HTMLRewriter",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			src := ""
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					src = s.V
				}
			}
			return newHTMLRewriterInstance(i, src), nil
		},
	})

	return mod
}

type htmlRewriterState struct {
	src      string
	handlers map[string]goipyObject.Object // selector -> handler callable
	interp   *goipyVM.Interp
}

func newHTMLRewriterInstance(i *goipyVM.Interp, src string) *goipyObject.Instance {
	state := &htmlRewriterState{
		src:      src,
		handlers: make(map[string]goipyObject.Object),
		interp:   i,
	}

	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "HTMLRewriter"},
		Dict:  goipyObject.NewDict(),
	}

	inst.Dict.SetStr("on", &goipyObject.BuiltinFunc{
		Name: "on",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("HTMLRewriter.on() requires selector and handler")
			}
			sel, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("HTMLRewriter.on(): selector must be str")
			}
			state.handlers[sel.V] = args[1]
			return inst, nil
		},
	})

	inst.Dict.SetStr("transform", &goipyObject.BuiltinFunc{
		Name: "transform",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			html := state.src
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					html = s.V
				}
			}
			out, err := rewriteHTML(i, html, state.handlers)
			if err != nil {
				return nil, err
			}
			return &goipyObject.Str{V: out}, nil
		},
	})

	return inst
}

// rewriteHTML does a simple element-level rewrite.
// For each registered selector (element name or "element" wildcard),
// it builds an element object and calls the handler.
// The handler can call element.set_attribute, element.remove_attribute,
// element.set_inner_content, element.prepend, element.append, element.remove.
func rewriteHTML(i *goipyVM.Interp, src string, handlers map[string]goipyObject.Object) (string, error) {
	if len(handlers) == 0 {
		return src, nil
	}

	var out strings.Builder
	pos := 0
	n := len(src)

	for pos < n {
		lt := strings.IndexByte(src[pos:], '<')
		if lt < 0 {
			out.WriteString(src[pos:])
			break
		}
		lt += pos
		out.WriteString(src[pos:lt])
		pos = lt

		gt := strings.IndexByte(src[pos:], '>')
		if gt < 0 {
			out.WriteString(src[pos:])
			break
		}
		gt += pos
		tag := src[pos : gt+1]
		pos = gt + 1

		// skip comments and doctype
		if strings.HasPrefix(tag, "<!--") || strings.HasPrefix(tag, "<!") || strings.HasPrefix(tag, "<?") {
			out.WriteString(tag)
			continue
		}

		// parse the tag
		inner := tag[1 : len(tag)-1] // strip < >
		selfClose := strings.HasSuffix(inner, "/")
		if selfClose {
			inner = inner[:len(inner)-1]
		}
		closing := strings.HasPrefix(inner, "/")
		if closing {
			inner = inner[1:]
		}

		parts := strings.Fields(inner)
		if len(parts) == 0 {
			out.WriteString(tag)
			continue
		}
		tagName := strings.ToLower(parts[0])

		// find handler
		var handler goipyObject.Object
		if h, ok := handlers[tagName]; ok && !closing {
			handler = h
		} else if h, ok := handlers["*"]; ok && !closing {
			handler = h
		}

		if handler == nil {
			out.WriteString(tag)
			continue
		}

		// build element object
		attrs := parseTagAttrs(parts[1:])
		el := buildElementObj(tagName, attrs, selfClose)
		_, err := i.Call(handler, []goipyObject.Object{el}, nil)
		if err != nil {
			return "", fmt.Errorf("HTMLRewriter handler error: %w", err)
		}

		// check if element was removed
		removedV, _ := el.Dict.GetStr("_removed")
		if removedV != nil {
			if b, ok := removedV.(*goipyObject.Bool); ok && b.V {
				continue
			}
		}

		// prepend content
		prependV, _ := el.Dict.GetStr("_prepend")

		// rebuild tag with potentially modified attributes
		rebuilt := rebuildTag(tagName, el, selfClose, closing)

		appendV, _ := el.Dict.GetStr("_append")
		innerContentV, _ := el.Dict.GetStr("_inner_content")

		if prependV != nil {
			if s, ok := prependV.(*goipyObject.Str); ok {
				out.WriteString(s.V)
			}
		}
		out.WriteString(rebuilt)

		// if inner content was replaced, consume until closing tag and write new content
		if innerContentV != nil {
			if s, ok := innerContentV.(*goipyObject.Str); ok {
				// consume until closing tag
				closeTag := "</" + tagName + ">"
				idx := strings.Index(src[pos:], closeTag)
				if idx >= 0 {
					out.WriteString(s.V)
					out.WriteString(closeTag)
					pos += idx + len(closeTag)
				} else {
					out.WriteString(s.V)
				}
			}
		}

		if appendV != nil {
			if s, ok := appendV.(*goipyObject.Str); ok {
				out.WriteString(s.V)
			}
		}
	}

	return out.String(), nil
}

func buildElementObj(tagName string, attrs map[string]string, selfClose bool) *goipyObject.Instance {
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Element"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("tag_name", &goipyObject.Str{V: tagName})

	// Convert attrs to a dict
	attrDict := goipyObject.NewDict()
	for k, v := range attrs {
		attrDict.SetStr(k, &goipyObject.Str{V: v})
	}
	inst.Dict.SetStr("attributes", attrDict)

	inst.Dict.SetStr("set_attribute", &goipyObject.BuiltinFunc{
		Name: "set_attribute",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("element.set_attribute() requires name and value")
			}
			k, ok1 := args[0].(*goipyObject.Str)
			v, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("element.set_attribute(): name and value must be str")
			}
			attrDict.SetStr(k.V, v)
			return inst, nil
		},
	})

	inst.Dict.SetStr("remove_attribute", &goipyObject.BuiltinFunc{
		Name: "remove_attribute",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("element.remove_attribute() requires name")
			}
			if k, ok := args[0].(*goipyObject.Str); ok {
				attrDict.SetStr(k.V, goipyObject.None)
			}
			return inst, nil
		},
	})

	inst.Dict.SetStr("get_attribute", &goipyObject.BuiltinFunc{
		Name: "get_attribute",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.None, nil
			}
			k, ok := args[0].(*goipyObject.Str)
			if !ok {
				return goipyObject.None, nil
			}
			v, exists := attrDict.GetStr(k.V)
			if !exists || v == goipyObject.None {
				return goipyObject.None, nil
			}
			return v, nil
		},
	})

	inst.Dict.SetStr("set_inner_content", &goipyObject.BuiltinFunc{
		Name: "set_inner_content",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("element.set_inner_content() requires content string")
			}
			if s, ok := args[0].(*goipyObject.Str); ok {
				inst.Dict.SetStr("_inner_content", s)
			}
			return inst, nil
		},
	})

	inst.Dict.SetStr("prepend", &goipyObject.BuiltinFunc{
		Name: "prepend",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					inst.Dict.SetStr("_prepend", s)
				}
			}
			return inst, nil
		},
	})

	inst.Dict.SetStr("append", &goipyObject.BuiltinFunc{
		Name: "append",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					inst.Dict.SetStr("_append", s)
				}
			}
			return inst, nil
		},
	})

	inst.Dict.SetStr("remove", &goipyObject.BuiltinFunc{
		Name: "remove",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			inst.Dict.SetStr("_removed", goipyObject.BoolOf(true))
			return inst, nil
		},
	})

	return inst
}

func parseTagAttrs(parts []string) map[string]string {
	attrs := make(map[string]string)
	for _, p := range parts {
		if idx := strings.IndexByte(p, '='); idx >= 0 {
			k := p[:idx]
			v := strings.Trim(p[idx+1:], `"'`)
			attrs[k] = v
		} else if p != "" {
			attrs[p] = ""
		}
	}
	return attrs
}

func rebuildTag(tagName string, el *goipyObject.Instance, selfClose, closing bool) string {
	if closing {
		return "</" + tagName + ">"
	}
	var sb strings.Builder
	sb.WriteByte('<')
	sb.WriteString(tagName)

	attrDictV, _ := el.Dict.GetStr("attributes")
	if ad, ok := attrDictV.(*goipyObject.Dict); ok {
		keys, vals := ad.Items()
		for i, k := range keys {
			ks, ok2 := k.(*goipyObject.Str)
			if !ok2 {
				continue
			}
			v := vals[i]
			if v == goipyObject.None {
				continue
			}
			sb.WriteByte(' ')
			sb.WriteString(ks.V)
			if vs, ok3 := v.(*goipyObject.Str); ok3 {
				if vs.V != "" {
					sb.WriteString(`="`)
					sb.WriteString(vs.V)
					sb.WriteByte('"')
				}
			}
		}
	}

	if selfClose {
		sb.WriteString(" />")
	} else {
		sb.WriteByte('>')
	}
	return sb.String()
}

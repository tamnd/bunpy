package bunpy

import (
	"fmt"
	"html"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildEscapeHTML(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.escape_html", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("escape", &goipyObject.BuiltinFunc{
		Name: "escape",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("escape_html.escape() requires a string")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("escape_html.escape(): argument must be str")
			}
			return &goipyObject.Str{V: html.EscapeString(s.V)}, nil
		},
	})

	mod.Dict.SetStr("unescape", &goipyObject.BuiltinFunc{
		Name: "unescape",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("escape_html.unescape() requires a string")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("escape_html.unescape(): argument must be str")
			}
			return &goipyObject.Str{V: html.UnescapeString(s.V)}, nil
		},
	})

	mod.Dict.SetStr("strip_tags", &goipyObject.BuiltinFunc{
		Name: "strip_tags",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("escape_html.strip_tags() requires a string")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("escape_html.strip_tags(): argument must be str")
			}
			return &goipyObject.Str{V: stripHTMLTags(s.V)}, nil
		},
	})

	return mod
}

func stripHTMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, ch := range s {
		switch {
		case ch == '<':
			inTag = true
		case ch == '>':
			inTag = false
		case !inTag:
			b.WriteRune(ch)
		}
	}
	return b.String()
}

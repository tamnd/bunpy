package bunpy

import (
	"fmt"
	"strings"

	"github.com/tamnd/bunpy/v1/internal/markdown"
	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// ANSI color/style codes
var termStyles = map[string]string{
	"reset":     "\x1b[0m",
	"bold":      "\x1b[1m",
	"dim":       "\x1b[2m",
	"italic":    "\x1b[3m",
	"underline": "\x1b[4m",
	"blink":     "\x1b[5m",
	"inverse":   "\x1b[7m",
	"strike":    "\x1b[9m",
	// foreground colors
	"black":   "\x1b[30m",
	"red":     "\x1b[31m",
	"green":   "\x1b[32m",
	"yellow":  "\x1b[33m",
	"blue":    "\x1b[34m",
	"magenta": "\x1b[35m",
	"cyan":    "\x1b[36m",
	"white":   "\x1b[37m",
	// bright variants
	"bright_black":   "\x1b[90m",
	"bright_red":     "\x1b[91m",
	"bright_green":   "\x1b[92m",
	"bright_yellow":  "\x1b[93m",
	"bright_blue":    "\x1b[94m",
	"bright_magenta": "\x1b[95m",
	"bright_cyan":    "\x1b[96m",
	"bright_white":   "\x1b[97m",
	// background colors
	"bg_black":   "\x1b[40m",
	"bg_red":     "\x1b[41m",
	"bg_green":   "\x1b[42m",
	"bg_yellow":  "\x1b[43m",
	"bg_blue":    "\x1b[44m",
	"bg_magenta": "\x1b[45m",
	"bg_cyan":    "\x1b[46m",
	"bg_white":   "\x1b[47m",
}

func BuildTerminal(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.terminal", Dict: goipyObject.NewDict()}

	// style(text, ...styles) applies one or more ANSI styles
	mod.Dict.SetStr("style", &goipyObject.BuiltinFunc{
		Name: "style",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("terminal.style() requires text and at least one style")
			}
			text, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("terminal.style(): text must be str")
			}
			var prefix strings.Builder
			for _, arg := range args[1:] {
				s, ok2 := arg.(*goipyObject.Str)
				if !ok2 {
					continue
				}
				if code, found := termStyles[s.V]; found {
					prefix.WriteString(code)
				}
			}
			if prefix.Len() == 0 {
				return text, nil
			}
			return &goipyObject.Str{V: prefix.String() + text.V + termStyles["reset"]}, nil
		},
	})

	// strip(text) removes ANSI escape codes
	mod.Dict.SetStr("strip", &goipyObject.BuiltinFunc{
		Name: "strip",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("terminal.strip() requires a string")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("terminal.strip(): argument must be str")
			}
			return &goipyObject.Str{V: stripANSI(s.V)}, nil
		},
	})

	// columns() returns the terminal width
	mod.Dict.SetStr("columns", &goipyObject.BuiltinFunc{
		Name: "columns",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.NewInt(int64(termColumns())), nil
		},
	})

	// rows() returns the terminal height
	mod.Dict.SetStr("rows", &goipyObject.BuiltinFunc{
		Name: "rows",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.NewInt(int64(termRows())), nil
		},
	})

	// is_tty() returns True if stdout is a terminal
	mod.Dict.SetStr("is_tty", &goipyObject.BuiltinFunc{
		Name: "is_tty",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.BoolOf(isTerminal()), nil
		},
	})

	// Register each style as a convenience function: terminal.red(text), etc.
	for name, code := range termStyles {
		styleName := name
		styleCode := code
		if strings.HasPrefix(styleName, "bg_") || styleName == "reset" {
			continue
		}
		mod.Dict.SetStr(styleName, &goipyObject.BuiltinFunc{
			Name: styleName,
			Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
				if len(args) < 1 {
					return nil, fmt.Errorf("terminal.%s() requires text", styleName)
				}
				s, ok := args[0].(*goipyObject.Str)
				if !ok {
					return nil, fmt.Errorf("terminal.%s(): text must be str", styleName)
				}
				return &goipyObject.Str{V: styleCode + s.V + termStyles["reset"]}, nil
			},
		})
	}

	// markdown(text) — render Markdown as ANSI terminal text.
	mod.Dict.SetStr("markdown", &goipyObject.BuiltinFunc{
		Name: "markdown",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("terminal.markdown() requires text")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("terminal.markdown(): text must be str")
			}
			return &goipyObject.Str{V: markdown.Render(s.V)}, nil
		},
	})

	return mod
}

func stripANSI(s string) string {
	var sb strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// skip until 'm'
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip 'm'
			continue
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}

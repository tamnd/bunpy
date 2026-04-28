package bunpy

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

var globalLogger atomic.Pointer[slog.Logger]
var globalLogFile atomic.Pointer[os.File]

func init() {
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	globalLogger.Store(l)
}

func BuildLog(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.log", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("configure", &goipyObject.BuiltinFunc{
		Name: "configure",
		Call: func(_ any, _ []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			level := slog.LevelInfo
			format := "text"
			var out io.Writer = os.Stderr

			if kwargs != nil {
				if lv, ok := kwargs.GetStr("level"); ok {
					if s, ok2 := lv.(*goipyObject.Str); ok2 {
						switch s.V {
						case "debug":
							level = slog.LevelDebug
						case "info":
							level = slog.LevelInfo
						case "warn", "warning":
							level = slog.LevelWarn
						case "error":
							level = slog.LevelError
						default:
							return nil, fmt.Errorf("bunpy.log.configure(): unknown level %q", s.V)
						}
					}
				}
				if fv, ok := kwargs.GetStr("format"); ok {
					if s, ok2 := fv.(*goipyObject.Str); ok2 {
						format = s.V
					}
				}
				if pv, ok := kwargs.GetStr("file"); ok {
					if s, ok2 := pv.(*goipyObject.Str); ok2 {
						f, err := os.OpenFile(s.V, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
						if err != nil {
							return nil, fmt.Errorf("bunpy.log.configure(): %w", err)
						}
						out = f
						// Close previous file-backed writer before replacing.
						if old := globalLogFile.Swap(f); old != nil {
							old.Close()
						}
					}
				} else {
					// No file specified: reset to stderr and close any open log file.
					if old := globalLogFile.Swap(nil); old != nil {
						old.Close()
					}
				}
			}

			opts := &slog.HandlerOptions{Level: level}
			var handler slog.Handler
			if format == "json" {
				handler = slog.NewJSONHandler(out, opts)
			} else {
				handler = slog.NewTextHandler(out, opts)
			}
			globalLogger.Store(slog.New(handler))
			return goipyObject.None, nil
		},
	})

	for _, lvl := range []struct {
		name  string
		level slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
	} {
		lvl := lvl
		mod.Dict.SetStr(lvl.name, &goipyObject.BuiltinFunc{
			Name: lvl.name,
			Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
				msg := ""
				if len(args) >= 1 {
					if s, ok := args[0].(*goipyObject.Str); ok {
						msg = s.V
					}
				}
				attrs := kwargsToSlogAttrs(kwargs)
				globalLogger.Load().Log(nil, lvl.level, msg, attrs...)
				return goipyObject.None, nil
			},
		})
	}

	mod.Dict.SetStr("with_fields", &goipyObject.BuiltinFunc{
		Name: "with_fields",
		Call: func(_ any, _ []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			attrs := kwargsToSlogAttrs(kwargs)
			child := globalLogger.Load().With(attrs...)
			return buildChildLogger(child), nil
		},
	})

	return mod
}

func kwargsToSlogAttrs(kwargs *goipyObject.Dict) []any {
	if kwargs == nil {
		return nil
	}
	keys, vals := kwargs.Items()
	result := make([]any, 0, len(keys)*2)
	for j, k := range keys {
		result = append(result, k, pyObjToGoValue(vals[j]))
	}
	return result
}

func buildChildLogger(logger *slog.Logger) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Logger", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	for _, lvl := range []struct {
		name  string
		level slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
	} {
		lvl := lvl
		inst.Dict.SetStr(lvl.name, &goipyObject.BuiltinFunc{
			Name: lvl.name,
			Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
				msg := ""
				if len(args) >= 1 {
					if s, ok := args[0].(*goipyObject.Str); ok {
						msg = s.V
					}
				}
				attrs := kwargsToSlogAttrs(kwargs)
				logger.Log(nil, lvl.level, msg, attrs...)
				return goipyObject.None, nil
			},
		})
	}
	return inst
}

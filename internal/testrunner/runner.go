package testrunner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
	gocopyCompiler "github.com/tamnd/gocopy/compiler"
	gocopyMarshal "github.com/tamnd/gocopy/marshal"
	goipyMarshal "github.com/tamnd/goipy/marshal"
	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// RunOptions configures a test run.
type RunOptions struct {
	// Verbose prints each test name as it runs.
	Verbose bool
	// Filter is an optional substring; only tests whose name contains it run.
	Filter string
}

// RunFile compiles and executes a single test file, collecting results.
func RunFile(path string, opts RunOptions) FileResult {
	result := FileResult{File: path}
	start := time.Now()

	src, err := os.ReadFile(path)
	if err != nil {
		result.CompileError = fmt.Sprintf("read error: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	co, err := gocopyCompiler.Compile(src, gocopyCompiler.Options{Filename: path})
	if err != nil {
		result.CompileError = fmt.Sprintf("compile error: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	stream, err := gocopyMarshal.Marshal(co)
	if err != nil {
		result.CompileError = fmt.Sprintf("marshal error: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	obj, err := goipyMarshal.Unmarshal(stream)
	if err != nil {
		result.CompileError = fmt.Sprintf("unmarshal error: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	code, ok := obj.(*goipyObject.Code)
	if !ok {
		result.CompileError = fmt.Sprintf("unexpected top-level object %T", obj)
		result.Duration = time.Since(start)
		return result
	}

	// Build a fresh interpreter with bunpy modules + expect + runner hook.
	interp := goipyVM.New()
	mods := bunpyAPI.Modules()
	bunpyAPI.InjectGlobals(interp)
	InjectExpect(interp)

	// Inject the __bunpy_runner__ NativeModule that collects test results.
	collector := &testCollector{opts: opts, file: path}
	mods["__bunpy_runner__"] = collector.buildModule
	interp.SetNativeModules(mods)

	if abs, aerr := filepath.Abs(filepath.Dir(path)); aerr == nil {
		interp.SearchPath = []string{abs}
	}

	// Run the module (which also runs the bootstrap at the end).
	if _, rerr := interp.Run(code); rerr != nil {
		if exc, ok2 := rerr.(*goipyObject.Exception); ok2 {
			msg := strings.TrimSpace(goipyVM.FormatException(exc))
			// A module-level error before any tests ran.
			if len(collector.results) == 0 {
				result.CompileError = "module-level error: " + msg
			}
		} else {
			if len(collector.results) == 0 {
				result.CompileError = fmt.Sprintf("module-level error: %v", rerr)
			}
		}
	}

	result.Results = collector.results
	result.Duration = time.Since(start)
	return result
}

// testCollector is the injected __bunpy_runner__ module state.
type testCollector struct {
	interp  *goipyVM.Interp
	opts    RunOptions
	file    string
	results []TestResult
}

func (c *testCollector) buildModule(i *goipyVM.Interp) *goipyObject.Module {
	c.interp = i
	mod := &goipyObject.Module{Name: "__bunpy_runner__", Dict: goipyObject.NewDict()}
	mod.Dict.SetStr("run", &goipyObject.BuiltinFunc{
		Name: "run",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return goipyObject.None, nil
			}
			nameStr, ok := args[0].(*goipyObject.Str)
			if !ok {
				return goipyObject.None, nil
			}
			name := nameStr.V
			fn := args[1]
			if c.opts.Filter != "" && !strings.Contains(name, c.opts.Filter) {
				return goipyObject.None, nil
			}
			tr := c.runOne(name, fn)
			c.results = append(c.results, tr)
			return goipyObject.None, nil
		},
	})
	return mod
}

func (c *testCollector) runOne(name string, fn goipyObject.Object) TestResult {
	tr := TestResult{File: c.file, Name: name}
	start := time.Now()

	defer func() {
		tr.Duration = time.Since(start)
		if r := recover(); r != nil {
			tr.Status = StatusError
			tr.Message = fmt.Sprintf("panic: %v", r)
		}
	}()

	_, err := c.interp.CallObject(fn, nil, nil)
	if err == nil {
		tr.Status = StatusPass
		return tr
	}

	if exc, ok := err.(*goipyObject.Exception); ok {
		msg := strings.TrimSpace(goipyVM.FormatException(exc))
		if isSkip(msg) {
			tr.Status = StatusSkip
			tr.Message = extractSkipMsg(msg)
		} else {
			tr.Status = StatusFail
			tr.Message = msg
		}
		return tr
	}

	tr.Status = StatusError
	tr.Message = err.Error()
	return tr
}

func isSkip(msg string) bool {
	return strings.Contains(msg, "SkipTest") || strings.Contains(msg, "skip:")
}

func extractSkipMsg(msg string) string {
	for _, line := range strings.Split(msg, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return msg
}

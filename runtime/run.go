// Package runtime is the bridge from a .py source file to the goipy
// bytecode VM. The pipeline is gocopy.Compile then a marshal hop into
// goipy's object.Code, then vm.Run.
//
// Subsequent rungs grow this package: hot reload (v0.7.0), env loader
// (v0.7.x), import path setup (v0.0.4 stdlib smoke), and so on.
package runtime

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	gocopyCompiler "github.com/tamnd/gocopy/v1/compiler"
	gocopyMarshal "github.com/tamnd/gocopy/v1/marshal"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
	goipyMarshal "github.com/tamnd/goipy/marshal"
	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// Run compiles source as a Python module, executes it on the goipy
// VM, and returns the process-level exit code. A SystemExit propagates
// its code; an uncaught Python exception is formatted to stderr and
// the function returns 1.
//
// stdout and stderr are wired into the VM. Pass os.Stdout / os.Stderr
// for the CLI; pass *bytes.Buffer for tests.
func Run(filename string, source []byte, args []string, stdout, stderr io.Writer) (int, error) {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	co, err := gocopyCompiler.Compile(source, gocopyCompiler.Options{Filename: filename})
	if err != nil {
		return 1, fmt.Errorf("compile %s: %w", filename, err)
	}

	stream, err := gocopyMarshal.Marshal(co)
	if err != nil {
		return 1, fmt.Errorf("marshal %s: %w", filename, err)
	}

	obj, err := goipyMarshal.Unmarshal(stream)
	if err != nil {
		return 1, fmt.Errorf("decode %s: %w", filename, err)
	}
	code, ok := obj.(*goipyObject.Code)
	if !ok {
		return 1, fmt.Errorf("decode %s: top-level object is %T, want code", filename, obj)
	}

	interp := goipyVM.New()
	interp.Stdout = stdout
	interp.Stderr = stderr
	interp.NativeModules = bunpyAPI.Modules()
	if abs, aerr := filepath.Abs(filepath.Dir(filename)); aerr == nil {
		interp.SearchPath = []string{abs}
	}
	interp.Argv = append([]string{filename}, args...)

	if _, rerr := interp.Run(code); rerr != nil {
		if code, ok := interp.SystemExitCode(rerr); ok {
			return code, nil
		}
		if exc, ok := rerr.(*goipyObject.Exception); ok {
			fmt.Fprint(stderr, goipyVM.FormatException(exc))
			return 1, nil
		}
		return 1, rerr
	}

	return 0, nil
}

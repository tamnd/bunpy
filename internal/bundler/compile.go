package bundler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

var compileMainTmpl = template.Must(template.New("main").Parse(`package main

import (
	_ "embed"
	"os"

	"github.com/tamnd/bunpy/v1/internal/bundler"
)

//go:embed app.pyz
var _bundle []byte

func main() {
	if err := bundler.RunPYZBytes(_bundle, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`))

// Compile builds a self-contained binary from the given .pyz file.
// It requires the Go toolchain to be on PATH.
func Compile(pyzPath, outBin string) error {
	goExe, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("bunpy build --compile requires Go to be installed (go not found on PATH)")
	}

	tmp, err := os.MkdirTemp("", "bunpy-compile-*")
	if err != nil {
		return fmt.Errorf("compile: mkdtemp: %w", err)
	}
	defer os.RemoveAll(tmp)

	// Copy .pyz as app.pyz.
	pyzData, err := os.ReadFile(pyzPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, "app.pyz"), pyzData, 0o644); err != nil {
		return err
	}

	// Write main.go.
	mainPath := filepath.Join(tmp, "main.go")
	mf, err := os.Create(mainPath)
	if err != nil {
		return err
	}
	if err := compileMainTmpl.Execute(mf, nil); err != nil {
		mf.Close()
		return err
	}
	mf.Close()

	// Write go.mod pointing at the current module.
	goMod := `module bunpy_compiled_app

go 1.21

require github.com/tamnd/bunpy/v1 v0.0.0

replace github.com/tamnd/bunpy/v1 => ` + moduleRoot() + `
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644); err != nil {
		return err
	}

	// Determine output binary name.
	if outBin == "" {
		stem := strings.TrimSuffix(filepath.Base(pyzPath), ".pyz")
		if runtime.GOOS == "windows" {
			stem += ".exe"
		}
		outBin = filepath.Join(filepath.Dir(pyzPath), stem)
	}

	cmd := exec.Command(goExe, "build", "-o", outBin, ".")
	cmd.Dir = tmp
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compile: go build: %w", err)
	}
	return nil
}

// moduleRoot returns the absolute path of the bunpy module root.
func moduleRoot() string {
	// Walk up from this file's package until go.mod is found.
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dir
}

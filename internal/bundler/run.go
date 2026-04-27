package bundler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tamnd/bunpy/v1/runtime"
)

// RunPYZ extracts a .pyz archive to a temp dir and runs __main__.py.
func RunPYZ(pyzPath string, args []string) error {
	data, err := os.ReadFile(pyzPath)
	if err != nil {
		return err
	}
	return RunPYZBytes(data, args)
}

// RunPYZBytes runs a .pyz bundle from in-memory bytes.
// Used by --compile binaries that embed the archive.
func RunPYZBytes(data []byte, args []string) error {
	// archive/zip handles arbitrary leading bytes (e.g. shebang).
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("pyz: open archive: %w", err)
	}

	tmp, err := os.MkdirTemp("", "bunpy-pyz-*")
	if err != nil {
		return fmt.Errorf("pyz: mkdtemp: %w", err)
	}
	defer os.RemoveAll(tmp)

	// Extract all files.
	for _, f := range r.File {
		dest := filepath.Join(tmp, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		buf.ReadFrom(rc)
		rc.Close()
		if err := os.WriteFile(dest, buf.Bytes(), 0o644); err != nil {
			return err
		}
	}

	mainPy := filepath.Join(tmp, "__main__.py")
	src, err := os.ReadFile(mainPy)
	if err != nil {
		return fmt.Errorf("pyz: missing __main__.py: %w", err)
	}

	code, err := runtime.Run(mainPy, src, args, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	if code != 0 {
		os.Exit(code)
	}
	return nil
}

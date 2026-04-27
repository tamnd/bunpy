package bundler

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
)

// WritePYZ writes the bundle as a PEP 441 .pyz archive.
// The archive has a shebang prefix so it can be executed directly on Unix.
func (b *Bundle) WritePYZ(outpath string) error {
	if err := os.MkdirAll(filepath.Dir(outpath), 0o755); err != nil {
		return fmt.Errorf("bundler: mkdir %s: %w", filepath.Dir(outpath), err)
	}

	f, err := os.OpenFile(outpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("bundler: create %s: %w", outpath, err)
	}
	defer f.Close()

	// Shebang prefix — archive/zip handles arbitrary leading bytes.
	if _, err := f.WriteString("#!/usr/bin/env bunpy\n"); err != nil {
		return err
	}

	zw := zip.NewWriter(f)

	// Write target metadata if set.
	if b.Opts.Target != "" {
		if err := writeTargetMetadata(zw, b.Opts.Target); err != nil {
			return err
		}
	}

	// Write all source files.
	for bundled, src := range b.Files {
		w, err := zw.Create(bundled)
		if err != nil {
			return fmt.Errorf("bundler: zip create %s: %w", bundled, err)
		}
		if _, err := w.Write([]byte(src)); err != nil {
			return err
		}
	}

	return zw.Close()
}

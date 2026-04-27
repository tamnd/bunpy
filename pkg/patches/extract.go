package patches

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extract writes the contents of a wheel body into dest, skipping
// RECORD and INSTALLER entries (those are install-time artefacts
// that vary across runs and would dirty the diff). dest is created
// if missing; existing files are overwritten. Path-escape entries
// are refused.
func Extract(wheelPath, dest string) error {
	r, err := zip.OpenReader(wheelPath)
	if err != nil {
		return err
	}
	defer r.Close()
	abs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/RECORD") || strings.HasSuffix(f.Name, "/INSTALLER") {
			continue
		}
		clean := filepath.Clean(filepath.Join(abs, f.Name))
		if !strings.HasPrefix(clean, abs+string(filepath.Separator)) && clean != abs {
			return fmt.Errorf("patches: refused path escape: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(clean, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(clean), 0o755); err != nil {
			return err
		}
		if err := writeZipEntry(f, clean); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, dst string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// CopyTree copies every regular file under src into dst. Directory
// structure is preserved; existing files in dst are overwritten.
func CopyTree(src, dst string) error {
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	if _, err := os.Stat(srcAbs); err != nil {
		return err
	}
	if err := os.MkdirAll(dstAbs, 0o755); err != nil {
		return err
	}
	return filepath.Walk(srcAbs, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcAbs, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dstAbs, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, body, info.Mode().Perm())
	})
}

// ScratchRoot returns the consumer-side scratch root. v0.1.10 lives
// under .bunpy/patches/.scratch/<name>-<version>/.
func ScratchRoot(target, name, version string) string {
	return filepath.Join(filepath.Dir(target), "patches", ".scratch", name+"-"+version)
}

// PristineRoot returns the consumer-side pristine root. Lives
// under .bunpy/patches/.pristine/<name>-<version>/.
func PristineRoot(target, name, version string) string {
	return filepath.Join(filepath.Dir(target), "patches", ".pristine", name+"-"+version)
}

// ErrNotInstalled is returned when the target has no install for
// the requested package; callers translate to a user message.
var ErrNotInstalled = errors.New("patches: package not installed")

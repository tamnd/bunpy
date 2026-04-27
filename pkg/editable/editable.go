// Package editable lays down PEP 660 editable installs for
// `bunpy link <pkg>`. The on-disk shape is the same as a regular
// wheel install: a `<name>-<version>.dist-info/` directory plus
// the package proxy. v0.1.9 emits a `.pth` file that prepends
// the source root to sys.path, which is the simplest PEP 660
// strategy.
package editable

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Spec describes one editable install.
type Spec struct {
	Name    string // PEP 503 normalised
	Version string
	Source  string // absolute source directory (must exist)
	Target  string // absolute site-packages root (created if missing)
}

// InstallerTag marks the dist-info INSTALLER file so
// `bunpy install` can skip overwriting linked packages.
const InstallerTag = "bunpy-link"

// Install lays down the editable proxy. Returns the absolute
// paths of every file written (the RECORD list).
func Install(s Spec) ([]string, error) {
	if s.Name == "" || s.Source == "" || s.Target == "" {
		return nil, errors.New("editable: Spec.Name, Source, and Target are required")
	}
	if !filepath.IsAbs(s.Source) || !filepath.IsAbs(s.Target) {
		return nil, errors.New("editable: Source and Target must be absolute")
	}
	if info, err := os.Stat(s.Source); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("editable: Source is not a directory: %s", s.Source)
	}
	if err := os.MkdirAll(s.Target, 0o755); err != nil {
		return nil, err
	}

	distInfoName := s.Name + "-" + s.Version + ".dist-info"
	distInfo := filepath.Join(s.Target, distInfoName)
	if err := os.MkdirAll(distInfo, 0o755); err != nil {
		return nil, err
	}

	written := []string{}
	writeFile := func(rel string, body []byte) error {
		path := filepath.Join(s.Target, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return err
		}
		written = append(written, rel)
		return nil
	}

	pthName := s.Name + ".pth"
	if err := writeFile(pthName, []byte(s.Source+"\n")); err != nil {
		return nil, err
	}

	metadata := fmt.Appendf(nil, "Metadata-Version: 2.1\nName: %s\nVersion: %s\n",
		s.Name, valueOrEmpty(s.Version))
	if err := writeFile(filepath.Join(distInfoName, "METADATA"), metadata); err != nil {
		return nil, err
	}
	if err := writeFile(filepath.Join(distInfoName, "INSTALLER"), []byte(InstallerTag+"\n")); err != nil {
		return nil, err
	}
	directURL, err := json.Marshal(map[string]any{
		"url": "file://" + s.Source,
		"dir_info": map[string]any{
			"editable": true,
		},
	})
	if err != nil {
		return nil, err
	}
	if err := writeFile(filepath.Join(distInfoName, "direct_url.json"), directURL); err != nil {
		return nil, err
	}

	recordRel := filepath.Join(distInfoName, "RECORD")
	record := buildRecord(s.Target, written, recordRel)
	if err := writeFile(recordRel, []byte(record)); err != nil {
		return nil, err
	}
	return written, nil
}

// Uninstall removes a previously-installed editable. Walks RECORD
// when present; missing dist-info is a no-op. Path entries that
// escape Target are skipped (defense-in-depth).
func Uninstall(target, name, version string) error {
	if target == "" || name == "" {
		return errors.New("editable: Uninstall requires target and name")
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	distInfoName := name + "-" + version + ".dist-info"
	distInfo := filepath.Join(abs, distInfoName)
	if _, err := os.Stat(distInfo); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	recordPath := filepath.Join(distInfo, "RECORD")
	if data, err := os.ReadFile(recordPath); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			rel := strings.SplitN(line, ",", 2)[0]
			if rel == "" {
				continue
			}
			path := filepath.Clean(filepath.Join(abs, rel))
			if !strings.HasPrefix(path, abs+string(filepath.Separator)) && path != abs {
				continue
			}
			_ = os.Remove(path)
		}
	}
	return os.RemoveAll(distInfo)
}

func valueOrEmpty(v string) string {
	if v == "" {
		return "0.0.0"
	}
	return v
}

// buildRecord emits a PEP 376 RECORD covering every file in paths
// (relative to target). The RECORD file itself appears as the last
// line with empty hash and size, matching the wheel installer.
func buildRecord(target string, paths []string, recordRel string) string {
	sort.Strings(paths)
	var sb strings.Builder
	for _, rel := range paths {
		abs := filepath.Join(target, rel)
		hash, size, err := hashFile(abs)
		if err != nil {
			fmt.Fprintf(&sb, "%s,,\n", rel)
			continue
		}
		fmt.Fprintf(&sb, "%s,sha256=%s,%d\n", rel, hash, size)
	}
	fmt.Fprintf(&sb, "%s,,\n", recordRel)
	return sb.String()
}

func hashFile(path string) (string, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(data)
	enc := base64.RawURLEncoding.EncodeToString(sum[:])
	return enc, int64(len(data)), nil
}

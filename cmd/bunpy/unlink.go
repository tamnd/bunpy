package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/editable"
	"github.com/tamnd/bunpy/v1/pkg/links"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
)

// unlinkSubcommand wires `bunpy unlink [pkg]...`. With no args it
// drops the current project from the global registry. With args
// it removes each named editable proxy from the consumer-side
// site-packages.
func unlinkSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		target = filepath.Join(".bunpy", "site-packages")
		pkgs   []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("unlink", stdout, stderr)
		case "--target":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy unlink: --target requires a value")
			}
			i++
			target = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--target="); ok {
				target = v
				continue
			}
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy unlink: unknown flag %q", a)
			}
			pkgs = append(pkgs, a)
		}
	}

	if len(pkgs) == 0 {
		return unregisterCurrent(stdout)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return 1, fmt.Errorf("bunpy unlink: %w", err)
	}
	for _, raw := range pkgs {
		name := pypi.Normalize(raw)
		// Find the version by scanning the dist-info filename pattern.
		version := findEditableVersion(absTarget, name)
		if version == "" {
			fmt.Fprintf(stdout, "no link for %s\n", raw)
			continue
		}
		if err := editable.Uninstall(absTarget, name, version); err != nil {
			return 1, fmt.Errorf("bunpy unlink: %s: %w", raw, err)
		}
		fmt.Fprintf(stdout, "unlinked %s %s\n", name, version)
	}
	return 0, nil
}

// unregisterCurrent reads ./pyproject.toml and drops the project
// from the global registry. Missing entry is a no-op.
func unregisterCurrent(stdout io.Writer) (int, error) {
	mf, err := manifest.Load("pyproject.toml")
	if err != nil {
		return 1, fmt.Errorf("bunpy unlink: %w", err)
	}
	if mf.Project.Name == "" {
		return 1, fmt.Errorf("bunpy unlink: pyproject.toml has no [project].name")
	}
	name := pypi.Normalize(mf.Project.Name)
	if err := links.Delete(name); err != nil {
		return 1, fmt.Errorf("bunpy unlink: %w", err)
	}
	fmt.Fprintf(stdout, "unregistered %s\n", name)
	return 0, nil
}

// findEditableVersion looks for `<name>-<version>.dist-info` under
// target and returns the first matching version. Empty string
// means no editable install for that name is present.
func findEditableVersion(target, name string) string {
	entries, err := os.ReadDir(target)
	if err != nil {
		return ""
	}
	prefix := name + "-"
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		n := ent.Name()
		if !strings.HasPrefix(n, prefix) || !strings.HasSuffix(n, ".dist-info") {
			continue
		}
		// Verify it is one of ours by reading INSTALLER.
		body, err := os.ReadFile(filepath.Join(target, n, "INSTALLER"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(body)) != editable.InstallerTag {
			continue
		}
		mid := strings.TrimSuffix(strings.TrimPrefix(n, prefix), ".dist-info")
		return mid
	}
	return ""
}

package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/tamnd/bunpy/v1/pkg/editable"
	"github.com/tamnd/bunpy/v1/pkg/links"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
)

// linkSubcommand wires `bunpy link [pkg]...`. With no args it
// registers the current project in the global link registry. With
// args it installs each named package from the registry as a
// PEP 660 editable proxy under .bunpy/site-packages.
func linkSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		target  = filepath.Join(".bunpy", "site-packages")
		list    bool
		pkgs    []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("link", stdout, stderr)
		case "--list":
			list = true
		case "--target":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy link: --target requires a value")
			}
			i++
			target = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--target="); ok {
				target = v
				continue
			}
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy link: unknown flag %q", a)
			}
			pkgs = append(pkgs, a)
		}
	}

	if list {
		entries, err := links.List()
		if err != nil {
			return 1, fmt.Errorf("bunpy link --list: %w", err)
		}
		for _, e := range entries {
			fmt.Fprintf(stdout, "%s %s -> %s\n", e.Name, e.Version, e.Source)
		}
		return 0, nil
	}

	if len(pkgs) == 0 {
		return registerCurrent(stdout)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return 1, fmt.Errorf("bunpy link: %w", err)
	}
	for _, raw := range pkgs {
		name := pypi.Normalize(raw)
		entry, err := links.Read(name)
		if err != nil {
			return 1, fmt.Errorf("bunpy link: %s: %w", raw, err)
		}
		spec := editable.Spec{
			Name:    name,
			Version: entry.Version,
			Source:  entry.Source,
			Target:  absTarget,
		}
		if _, err := editable.Install(spec); err != nil {
			return 1, fmt.Errorf("bunpy link: %s: %w", raw, err)
		}
		fmt.Fprintf(stdout, "linked %s %s -> %s\n", entry.Name, entry.Version, entry.Source)
	}
	return 0, nil
}

// registerCurrent reads ./pyproject.toml and writes a registry
// entry for the project so other consumers can `bunpy link <name>`.
func registerCurrent(stdout io.Writer) (int, error) {
	mf, err := manifest.Load("pyproject.toml")
	if err != nil {
		return 1, fmt.Errorf("bunpy link: %w", err)
	}
	if mf.Project.Name == "" {
		return 1, fmt.Errorf("bunpy link: pyproject.toml has no [project].name")
	}
	source, err := filepath.Abs(".")
	if err != nil {
		return 1, fmt.Errorf("bunpy link: %w", err)
	}
	entry := links.Entry{
		Name:       pypi.Normalize(mf.Project.Name),
		Version:    mf.Project.Version,
		Source:     source,
		Registered: time.Now().UTC(),
	}
	if err := links.Write(entry); err != nil {
		return 1, fmt.Errorf("bunpy link: %w", err)
	}
	fmt.Fprintf(stdout, "registered %s %s -> %s\n", entry.Name, entry.Version, entry.Source)
	return 0, nil
}

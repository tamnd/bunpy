package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/manifest"
)

// pmSubcommand routes the `bunpy pm <verb>` plumbing tree. v0.1.0
// only wires `config`; later rungs grow `info`, `install-wheel`, and
// the rest under the same umbrella.
func pmSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: bunpy pm <verb>")
		return 1, fmt.Errorf("bunpy pm requires a verb (known: config)")
	}
	switch args[0] {
	case "config":
		return pmConfig(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		return printHelp("pm", stdout, stderr)
	}
	return 1, fmt.Errorf("bunpy pm: unknown verb %q (known: config, --help)", args[0])
}

// pmConfig parses pyproject.toml and prints the structured manifest
// as indented JSON. Default path is ./pyproject.toml; an optional
// positional argument overrides.
func pmConfig(args []string, stdout, stderr io.Writer) (int, error) {
	path := "pyproject.toml"
	for _, a := range args {
		switch a {
		case "-h", "--help":
			return printHelp("pm-config", stdout, stderr)
		default:
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy pm config: unknown flag %q (known: --help)", a)
			}
			path = a
		}
	}
	m, err := manifest.Load(path)
	if err != nil {
		return 1, err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		return 1, fmt.Errorf("bunpy pm config: %w", err)
	}
	return 0, nil
}

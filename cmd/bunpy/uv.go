package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/tamnd/bunpy/v1/internal/uvdetect"
)

// uvSubcommand wires `bunpy uv <args>` — a transparent shim that
// delegates every argument to the real uv binary. This covers the full
// uv surface, including `bunpy uv pip install/list/show/freeze`. If uv
// is not installed the error message links to the install guide.
func uvSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stdout, "usage: bunpy uv [args...]")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Delegates to the real uv binary. All arguments are forwarded unchanged.")
		fmt.Fprintln(stdout, "Install uv: https://github.com/astral-sh/uv")
		return 0, nil
	}

	uvBin, err := uvdetect.Find()
	if err != nil {
		return 1, err
	}

	cmd := exec.Command(uvBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 1, fmt.Errorf("bunpy uv: %w", err)
	}
	return 0, nil
}

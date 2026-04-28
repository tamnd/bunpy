package main

import "io"

// syncSubcommand is an alias for `bunpy install`.
// It mirrors uv's `uv sync` verb so projects migrating from uv get
// muscle-memory compatibility.
func syncSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	return installSubcommand(args, stdout, stderr)
}

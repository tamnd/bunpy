package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tamnd/bunpy/v1/internal/manpages"
)

func installManpages(dir string) (int, error) {
	dest := filepath.Join(dir, "man1")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return 0, err
	}
	names, err := manpages.List()
	if err != nil {
		return 0, err
	}
	embedFS := manpages.FS()
	for _, n := range names {
		data, err := fs.ReadFile(embedFS, manpages.Root+"/"+n)
		if err != nil {
			return 0, err
		}
		if err := os.WriteFile(filepath.Join(dest, n), data, 0o644); err != nil {
			return 0, err
		}
	}
	return len(names), nil
}

func manSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: bunpy man <command> | bunpy man --install [dir]")
		return 1, fmt.Errorf("bunpy man requires a command or --install")
	}
	switch args[0] {
	case "-h", "--help", "help":
		return printHelp("man", stdout, stderr)
	case "--install":
		dir := ""
		if len(args) > 1 {
			dir = args[1]
		}
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return 1, fmt.Errorf("bunpy man --install: %w", err)
			}
			dir = filepath.Join(home, ".bunpy", "share", "man")
		}
		n, err := installManpages(dir)
		if err != nil {
			return 1, fmt.Errorf("bunpy man --install: %w", err)
		}
		fmt.Fprintf(stdout, "installed %d manpages into %s/man1\n", n, dir)
		return 0, nil
	}

	page, err := manpages.Page(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "bunpy: no manpage for", args[0])
		return 1, fmt.Errorf("no manpage for %q", args[0])
	}
	if _, err := stdout.Write(page); err != nil {
		return 1, err
	}
	return 0, nil
}

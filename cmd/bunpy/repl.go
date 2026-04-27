package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/tamnd/bunpy/v1/internal/repl"
)

func replSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	opts := repl.Options{}
	for _, a := range args {
		switch a {
		case "-h", "--help":
			return printHelp("repl", stdout, stderr)
		case "--quiet":
			opts.Quiet = true
		default:
			return 1, fmt.Errorf("bunpy repl: unknown flag %q (known: --quiet, --help)", a)
		}
	}

	opts.HistoryPath = historyPath()
	opts.HistorySize = historySize()
	rc := repl.Loop(os.Stdin, stdout, stderr, opts)
	return rc, nil
}

func historyPath() string {
	if v, ok := os.LookupEnv("BUNPY_HISTORY"); ok {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".bunpy_history")
}

func historySize() int {
	if v, ok := os.LookupEnv("BUNPY_HISTORY_SIZE"); ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return 0
		}
		return n
	}
	return 1000
}

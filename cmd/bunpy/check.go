package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tamnd/bunpy/v1/internal/checker"
)

func checkSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	noColor := fs.Bool("no-color", false, "disable ANSI color in output")
	if err := fs.Parse(args); err != nil {
		return 1, err
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: bunpy check <file.py|dir> [...]")
		return 1, fmt.Errorf("check requires at least one path")
	}

	files, err := collectPyFiles(fs.Args())
	if err != nil {
		return 1, err
	}

	_ = noColor
	total := 0
	fileCount := 0
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "check: %s: %v\n", path, err)
			continue
		}
		issues := checker.Check(path, string(src))
		if len(issues) > 0 {
			fileCount++
		}
		for _, iss := range issues {
			fmt.Fprintln(stdout, iss.String())
			total++
		}
	}

	if total > 0 {
		fmt.Fprintf(stdout, "%d issue(s) in %d file(s)\n", total, fileCount)
		return 1, nil
	}
	return 0, nil
}

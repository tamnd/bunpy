package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/bunpy/v1/internal/formatter"
)

func fmtSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	fs2 := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs2.SetOutput(stderr)
	checkMode := fs2.Bool("check", false, "exit 1 if any file would be changed (no writes)")
	diffMode := fs2.Bool("diff", false, "print unified diff, do not write")
	if err := fs2.Parse(args); err != nil {
		return 1, err
	}
	if fs2.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: bunpy fmt <file.py|dir> [...]")
		return 1, fmt.Errorf("fmt requires at least one path")
	}

	files, err := collectPyFiles(fs2.Args())
	if err != nil {
		return 1, err
	}

	changed := 0
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "fmt: %s: %v\n", path, err)
			continue
		}
		out := formatter.Format(src)
		if bytes.Equal(out, src) {
			continue
		}
		changed++
		if *checkMode {
			fmt.Fprintf(stdout, "would reformat: %s\n", path)
			continue
		}
		if *diffMode {
			fmt.Fprintf(stdout, "--- %s\n+++ %s (formatted)\n", path, path)
			printSimpleDiff(stdout, src, out)
			continue
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			fmt.Fprintf(stderr, "fmt: write %s: %v\n", path, err)
			continue
		}
		fmt.Fprintf(stdout, "formatted: %s\n", path)
	}

	if *checkMode && changed > 0 {
		return 1, nil
	}
	return 0, nil
}

func collectPyFiles(paths []string) ([]string, error) {
	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			if err := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if strings.HasSuffix(path, ".py") {
					files = append(files, path)
				}
				return nil
			}); err != nil {
				return nil, err
			}
		} else {
			files = append(files, p)
		}
	}
	return files, nil
}

func printSimpleDiff(w io.Writer, old, new []byte) {
	oldLines := strings.Split(string(old), "\n")
	newLines := strings.Split(string(new), "\n")
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}
	for i := 0; i < maxLen; i++ {
		o := ""
		n := ""
		if i < len(oldLines) {
			o = oldLines[i]
		}
		if i < len(newLines) {
			n = newLines[i]
		}
		if o != n {
			if i < len(oldLines) {
				fmt.Fprintf(w, "-%s\n", o)
			}
			if i < len(newLines) {
				fmt.Fprintf(w, "+%s\n", n)
			}
		}
	}
}

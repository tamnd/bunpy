package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/audit"
	"github.com/tamnd/bunpy/v1/pkg/lockfile"
)

func auditSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		jsonOut  bool
		quiet    bool
		ignore   []string
		lockPath string
		wsRoot   string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("audit", stdout, stderr)
		case "--json":
			jsonOut = true
		case "--quiet":
			quiet = true
		case "--ignore":
			if i+1 >= len(args) {
				return 2, fmt.Errorf("bunpy audit: --ignore requires a value")
			}
			i++
			ignore = append(ignore, args[i])
		case "--lockfile":
			if i+1 >= len(args) {
				return 2, fmt.Errorf("bunpy audit: --lockfile requires a value")
			}
			i++
			lockPath = args[i]
		case "--workspace":
			if i+1 >= len(args) {
				return 2, fmt.Errorf("bunpy audit: --workspace requires a value")
			}
			i++
			wsRoot = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--ignore="); ok {
				ignore = append(ignore, v)
				continue
			}
			if v, ok := strings.CutPrefix(a, "--lockfile="); ok {
				lockPath = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--workspace="); ok {
				wsRoot = v
				continue
			}
			return 2, fmt.Errorf("bunpy audit: unknown flag %q", a)
		}
	}

	if lockPath == "" {
		if wsRoot == "" {
			if cwd, err := os.Getwd(); err == nil {
				wsRoot, _ = findWorkspaceRoot(cwd)
			}
		}
		if wsRoot != "" {
			lockPath = filepath.Join(wsRoot, "bunpy.lock")
		} else {
			lockPath = "bunpy.lock"
		}
	}

	lock, err := lockfile.Read(lockPath)
	if err != nil {
		return 1, fmt.Errorf("bunpy audit: %w", err)
	}

	client := audit.NewOSVClient()
	findings, err := client.QueryBatch(lock.Packages)
	if err != nil {
		return 1, fmt.Errorf("bunpy audit: %w", err)
	}
	findings = audit.Filter(findings, ignore)
	audit.SortFindings(findings)

	if jsonOut {
		data, err := json.MarshalIndent(findings, "", "  ")
		if err != nil {
			return 1, fmt.Errorf("bunpy audit: %w", err)
		}
		if findings == nil {
			fmt.Fprintln(stdout, "[]")
		} else {
			fmt.Fprintln(stdout, string(data))
		}
		if len(findings) > 0 {
			return 1, nil
		}
		return 0, nil
	}

	if quiet {
		if len(findings) == 0 {
			fmt.Fprintln(stdout, "No vulnerabilities found.")
		} else {
			fmt.Fprintf(stdout, "%d %s found.\n", len(findings), pluralWord(len(findings), "vulnerability", "vulnerabilities"))
		}
		if len(findings) > 0 {
			return 1, nil
		}
		return 0, nil
	}

	if len(findings) == 0 {
		fmt.Fprintln(stdout, "No vulnerabilities found.")
		return 0, nil
	}

	// Table output.
	fmt.Fprintf(stdout, "%-25s %-12s %-28s %-10s %s\n", "Package", "Version", "ID", "Severity", "Summary")
	fmt.Fprintln(stdout, strings.Repeat("-", 100))
	for _, f := range findings {
		summary := f.Summary
		if len(summary) > 40 {
			summary = summary[:37] + "..."
		}
		fmt.Fprintf(stdout, "%-25s %-12s %-28s %-10s %s\n",
			f.Package, f.Version, f.ID, f.Severity, summary)
	}
	fmt.Fprintln(stdout, strings.Repeat("-", 100))
	fmt.Fprintf(stdout, "%d %s found. Run with --json for full details.\n",
		len(findings), pluralWord(len(findings), "vulnerability", "vulnerabilities"))
	return 1, nil
}

func pluralWord(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tamnd/bunpy/v1/internal/testrunner"
)

func testSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	opts := testrunner.RunOptions{}
	discoverOpts := testrunner.DiscoverOptions{}
	printer := testrunner.Printer{W: stdout}
	var isolate bool
	var parallel bool
	var shardIndex, shardTotal int
	var changed string
	var coverDir string

	positional := []string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			return printHelp("test", stdout, stderr)
		case a == "--verbose" || a == "-v":
			opts.Verbose = true
			printer.Verbose = true
		case a == "--no-color":
			printer.NoColor = true
		case a == "--isolate":
			isolate = true
		case a == "--parallel":
			parallel = true
		case strings.HasPrefix(a, "--filter="):
			opts.Filter = strings.TrimPrefix(a, "--filter=")
		case a == "--filter" && i+1 < len(args):
			i++
			opts.Filter = args[i]
		case strings.HasPrefix(a, "--shard="):
			fmt.Sscanf(strings.TrimPrefix(a, "--shard="), "%d/%d", &shardIndex, &shardTotal)
		case a == "--shard" && i+1 < len(args):
			i++
			fmt.Sscanf(args[i], "%d/%d", &shardIndex, &shardTotal)
		case strings.HasPrefix(a, "--changed="):
			changed = strings.TrimPrefix(a, "--changed=")
		case a == "--changed":
			changed = "HEAD"
		case a == "--changed" && i+1 < len(args):
			i++
			changed = args[i]
		case strings.HasPrefix(a, "--coverage-dir="):
			coverDir = strings.TrimPrefix(a, "--coverage-dir=")
		case a == "--coverage-dir" && i+1 < len(args):
			i++
			coverDir = args[i]
		case a == "--coverage":
			if coverDir == "" {
				coverDir = "coverage"
			}
		case a == "--timeout" && i+1 < len(args):
			i++ // future use
		case strings.HasPrefix(a, "-"):
			return 1, fmt.Errorf("bunpy test: unknown flag %q (see bunpy help test)", a)
		default:
			positional = append(positional, a)
		}
	}

	// Positional args are file/directory patterns.
	discoverOpts.Patterns = positional
	if len(positional) == 1 && !strings.ContainsAny(positional[0], "*?") {
		discoverOpts.Root = positional[0]
		discoverOpts.Patterns = nil
	}

	// Discover test files.
	var files []string
	var err error
	if changed != "" {
		files, err = testrunner.ChangedFiles(discoverOpts.Root, changed)
	} else {
		files, err = testrunner.DiscoverFiles(discoverOpts)
	}
	if err != nil {
		return 1, fmt.Errorf("bunpy test: discovery error: %w", err)
	}
	if len(files) == 0 {
		fmt.Fprintln(stdout, "No test files found.")
		return 0, nil
	}

	// Apply sharding.
	if shardTotal > 0 && shardIndex > 0 {
		files = testrunner.ShardFiles(files, shardIndex, shardTotal)
	}

	start := time.Now()
	var summary testrunner.Summary

	if isolate {
		// Run each file in a subprocess.
		for _, f := range files {
			fr, ierr := testrunner.RunIsolated(f, opts)
			if ierr != nil {
				fmt.Fprintf(stderr, "bunpy test: isolate error for %s: %v\n", f, ierr)
				continue
			}
			printer.PrintFileResult(fr)
			for _, r := range fr.Results {
				summary.Add(r)
			}
			if fr.CompileError != "" {
				summary.Errors++
				summary.Total++
			}
		}
	} else if parallel {
		results := testrunner.RunParallel(files, opts)
		for _, fr := range results {
			printer.PrintFileResult(fr)
			for _, r := range fr.Results {
				summary.Add(r)
			}
			if fr.CompileError != "" {
				summary.Errors++
				summary.Total++
			}
		}
	} else {
		for _, f := range files {
			fr := testrunner.RunFile(f, opts)
			printer.PrintFileResult(fr)
			for _, r := range fr.Results {
				summary.Add(r)
			}
			if fr.CompileError != "" {
				summary.Errors++
				summary.Total++
			}
		}
	}

	summary.Duration = time.Since(start)
	printer.PrintSummary(summary)

	if coverDir != "" {
		if cerr := testrunner.WriteCoverage(files, coverDir, stdout); cerr != nil {
			fmt.Fprintf(stderr, "bunpy test: coverage error: %v\n", cerr)
		}
	}

	if !summary.AllPassed() {
		return 1, nil
	}
	return 0, nil
}

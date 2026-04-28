package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tamnd/bunpy/v1/internal/bundler"
	"github.com/tamnd/bunpy/v1/runtime"
)

func buildSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)

	outfile := fs.String("outfile", "", "output file path")
	outdir := fs.String("outdir", "dist", "output directory")
	minify := fs.Bool("minify", false, "strip comments and blank lines")
	target := fs.String("target", "", "target platform (linux-x64, darwin-arm64, windows-x64, browser, ...)")
	sourcemap := fs.Bool("sourcemap", false, "write .pyz.map source map file")
	compile := fs.Bool("compile", false, "produce a self-contained binary (requires Go)")
	watch := fs.Bool("watch", false, "rebuild on file changes")
	var rawDefines multiFlag
	fs.Var(&rawDefines, "define", "KEY=VALUE build-time constant (repeatable)")
	var plugins multiFlag
	fs.Var(&plugins, "plugin", "path to transform plugin .py file (repeatable)")

	if err := fs.Parse(args); err != nil {
		return 1, err
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: bunpy build <entry.py> [options]")
		return 1, fmt.Errorf("build requires an entry file")
	}

	entry := fs.Arg(0)
	if !strings.HasSuffix(entry, ".py") {
		return 1, fmt.Errorf("entry must be a .py file, got %q", entry)
	}
	if _, err := os.Stat(entry); err != nil {
		return 1, fmt.Errorf("entry file not found: %s", entry)
	}

	if err := bundler.ValidateTarget(*target); err != nil {
		return 1, err
	}

	defines := parseDefines(rawDefines)

	opts := bundler.Options{
		Outfile:   *outfile,
		Outdir:    *outdir,
		Minify:    *minify,
		Target:    *target,
		Defines:   defines,
		Plugins:   []string(plugins),
		SourceMap: *sourcemap,
		Compile:   *compile,
	}

	if *watch {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		setupSignalCancel(cancel)
		return 0, bundler.Watch(ctx, entry, opts, stdout)
	}

	return doBuild(entry, opts, stdout, stderr)
}

func doBuild(entry string, opts bundler.Options, stdout, stderr io.Writer) (int, error) {
	entryAbs, err := filepath.Abs(entry)
	if err != nil {
		return 1, err
	}

	// Check the incremental build cache before doing any work.
	hit, cacheErr := bundler.CheckCache(entryAbs, opts, runtime.Version)
	if cacheErr == nil && hit {
		// Determine the output path from opts to print the right path.
		outpath := opts.Outfile
		if outpath == "" {
			stem := strings.TrimSuffix(filepath.Base(entry), ".py")
			dir := opts.Outdir
			if dir == "" {
				dir = "dist"
			}
			outpath = filepath.Join(dir, stem+".pyz")
		}
		fmt.Fprintf(stdout, "cache hit %s\n", outpath)
		return 0, nil
	}

	b, err := bundler.Build(entry, opts)
	if err != nil {
		return 1, err
	}

	// Apply plugins (no-op until gocopy supports function defs).
	if len(opts.Plugins) > 0 {
		bundler.ApplyPlugins(b, opts.Plugins, stderr)
	}

	outpath := b.OutPath()
	if err := b.WritePYZ(outpath); err != nil {
		return 1, err
	}

	if opts.SourceMap {
		if err := bundler.WriteSourceMap(b.Sources, outpath); err != nil {
			fmt.Fprintf(stderr, "warning: sourcemap: %v\n", err)
		}
	}

	fmt.Fprintf(stdout, "built %s (%d file(s))\n", outpath, len(b.Files))

	// Update cache (best-effort; don't fail the build on cache errors).
	if err := bundler.UpdateCache(entryAbs, opts, b, outpath, runtime.Version); err != nil {
		fmt.Fprintf(stderr, "warning: build cache: %v\n", err)
	}

	if opts.Compile {
		binOut := strings.TrimSuffix(outpath, ".pyz")
		if err := bundler.Compile(outpath, binOut); err != nil {
			return 1, err
		}
		fmt.Fprintf(stdout, "compiled %s\n", binOut)
	}

	return 0, nil
}

// multiFlag is a flag.Value that accumulates repeated string flags.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func setupSignalCancel(cancel context.CancelFunc) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()
}

func parseDefines(raw []string) map[string]string {
	out := map[string]string{}
	for _, r := range raw {
		k, v, ok := strings.Cut(r, "=")
		if ok {
			out[k] = v
		}
	}
	return out
}

package bundler

import (
	"context"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// Watch builds entry once, then polls for .py file changes and rebuilds.
// It runs until ctx is cancelled.
func Watch(ctx context.Context, entry string, opts Options, out io.Writer) error {
	build := func() (string, error) {
		b, err := Build(entry, opts)
		if err != nil {
			return "", err
		}
		outpath := b.OutPath()
		if err := b.WritePYZ(outpath); err != nil {
			return "", err
		}
		return outpath, nil
	}

	outpath, err := build()
	if err != nil {
		return err
	}
	printWatchLine(out, outpath, len(opts.Defines))

	mtimes := collectMtimes(filepath.Dir(entry))

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			current := collectMtimes(filepath.Dir(entry))
			if mapsChanged(mtimes, current) {
				mtimes = current
				if outpath, err = build(); err != nil {
					io.WriteString(out, "watch: build error: "+err.Error()+"\n")
					continue
				}
				printWatchLine(out, outpath, len(opts.Defines))
			}
		}
	}
}

func printWatchLine(out io.Writer, outpath string, _ int) {
	ts := time.Now().Format("15:04:05")
	io.WriteString(out, "["+ts+"] built "+outpath+"\n")
}

func collectMtimes(root string) map[string]time.Time {
	m := map[string]time.Time{}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".py") {
			if info, err := d.Info(); err == nil {
				m[path] = info.ModTime()
			}
		}
		return nil
	})
	return m
}

func mapsChanged(old, current map[string]time.Time) bool {
	if len(old) != len(current) {
		return true
	}
	for k, t := range current {
		if old[k] != t {
			return true
		}
	}
	return false
}

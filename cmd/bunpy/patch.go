package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/cache"
	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/patches"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/uvlock"
)

// patchSubcommand wires `bunpy patch <pkg>` (open scratch),
// `bunpy patch --commit <pkg>` (diff and persist), and
// `bunpy patch --list`.
func patchSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		target    = filepath.Join(".bunpy", "site-packages")
		cacheDir  string
		commit    bool
		list      bool
		printOnly bool
		noWrite   bool
		out       string
		pkgs      []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("patch", stdout, stderr)
		case "--commit":
			commit = true
		case "--list":
			list = true
		case "--print-only":
			printOnly = true
		case "--no-write":
			noWrite = true
		case "--target":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy patch: --target requires a value")
			}
			i++
			target = args[i]
		case "--cache-dir":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy patch: --cache-dir requires a value")
			}
			i++
			cacheDir = args[i]
		case "--out":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy patch: --out requires a value")
			}
			i++
			out = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--target="); ok {
				target = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--cache-dir="); ok {
				cacheDir = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--out="); ok {
				out = v
				continue
			}
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy patch: unknown flag %q", a)
			}
			pkgs = append(pkgs, a)
		}
	}

	if list {
		return patchList(stdout)
	}
	if commit {
		if len(pkgs) == 0 {
			return 1, fmt.Errorf("bunpy patch --commit: package name required")
		}
		return patchCommit(stdout, pkgs[0], target, out, noWrite)
	}
	if len(pkgs) == 0 {
		return 1, fmt.Errorf("bunpy patch: package name required")
	}
	return patchOpen(stdout, pkgs[0], target, cacheDir, printOnly)
}

// patchOpen lays out the pristine + scratch trees for pkg and
// prints the scratch path. Idempotent: re-running rewrites the
// scratch from the pristine baseline.
func patchOpen(stdout io.Writer, pkg, target, cacheDir string, printOnly bool) (int, error) {
	pin, err := lookupLockPin(pkg)
	if err != nil {
		return 1, fmt.Errorf("bunpy patch: %w", err)
	}
	if isLinkedPackage(target, pin.Name, pin.Version) {
		return 1, fmt.Errorf("bunpy patch: %s is a linked package; edit the source directly", pin.Name)
	}
	pristine := patches.PristineRoot(target, pin.Name, pin.Version)
	scratch := patches.ScratchRoot(target, pin.Name, pin.Version)
	if err := preparePristine(pristine, pin, cacheDir); err != nil {
		return 1, fmt.Errorf("bunpy patch: %w", err)
	}
	if err := os.RemoveAll(scratch); err != nil {
		return 1, fmt.Errorf("bunpy patch: %w", err)
	}
	if !printOnly {
		if err := patches.CopyTree(pristine, scratch); err != nil {
			return 1, fmt.Errorf("bunpy patch: %w", err)
		}
	}
	abs, err := filepath.Abs(scratch)
	if err != nil {
		return 1, fmt.Errorf("bunpy patch: %w", err)
	}
	fmt.Fprintf(stdout, "scratch %s %s -> %s\n", pin.Name, pin.Version, abs)
	return 0, nil
}

// patchCommit diffs scratch vs pristine, writes the patch file,
// and registers the entry in pyproject.toml. Removes the scratch
// on success.
func patchCommit(stdout io.Writer, pkg, target, out string, noWrite bool) (int, error) {
	pin, err := lookupLockPin(pkg)
	if err != nil {
		return 1, fmt.Errorf("bunpy patch --commit: %w", err)
	}
	pristine := patches.PristineRoot(target, pin.Name, pin.Version)
	scratch := patches.ScratchRoot(target, pin.Name, pin.Version)
	if _, err := os.Stat(scratch); err != nil {
		return 1, fmt.Errorf("bunpy patch --commit: scratch missing for %s; run `bunpy patch %s` first", pin.Name, pin.Name)
	}
	body, err := patches.Diff(pristine, scratch)
	if err != nil {
		return 1, fmt.Errorf("bunpy patch --commit: %w", err)
	}
	if len(body) == 0 {
		fmt.Fprintf(stdout, "no changes for %s %s\n", pin.Name, pin.Version)
		_ = os.RemoveAll(scratch)
		return 0, nil
	}
	patchPath := out
	if patchPath == "" {
		patchPath = filepath.Join("patches", pin.Name+"+"+pin.Version+".patch")
	}
	if err := os.MkdirAll(filepath.Dir(patchPath), 0o755); err != nil {
		return 1, fmt.Errorf("bunpy patch --commit: %w", err)
	}
	if err := os.WriteFile(patchPath, body, 0o644); err != nil {
		return 1, fmt.Errorf("bunpy patch --commit: %w", err)
	}
	if !noWrite {
		mf, err := manifest.LoadOpts("pyproject.toml", manifest.LoadOptions{})
		if err != nil {
			return 1, fmt.Errorf("bunpy patch --commit: %w", err)
		}
		key := pin.Name + "@" + pin.Version
		updated, _, err := mf.AddPatchEntry(key, patchPath)
		if err != nil {
			return 1, fmt.Errorf("bunpy patch --commit: %w", err)
		}
		if err := os.WriteFile("pyproject.toml", updated, 0o644); err != nil {
			return 1, fmt.Errorf("bunpy patch --commit: %w", err)
		}
	}
	if err := os.RemoveAll(scratch); err != nil {
		return 1, fmt.Errorf("bunpy patch --commit: %w", err)
	}
	fmt.Fprintf(stdout, "patched %s %s -> %s\n", pin.Name, pin.Version, patchPath)
	return 0, nil
}

// patchList prints every entry in [tool.bunpy.patches].
func patchList(stdout io.Writer) (int, error) {
	mf, err := manifest.LoadOpts("pyproject.toml", manifest.LoadOptions{})
	if err != nil {
		return 1, fmt.Errorf("bunpy patch --list: %w", err)
	}
	entries, err := patches.Read(mf)
	if err != nil {
		return 1, fmt.Errorf("bunpy patch --list: %w", err)
	}
	for _, e := range entries {
		exists := "missing"
		if _, err := os.Stat(e.Path); err == nil {
			exists = "ok"
		}
		fmt.Fprintf(stdout, "%s %s -> %s [%s]\n", e.Name, e.Version, e.Path, exists)
	}
	return 0, nil
}

// lookupLockPin reads uv.lock and returns the row for pkg
// (PEP 503 normalised). Errors out on missing lockfile or
// missing pin.
func lookupLockPin(pkg string) (lockfile.Package, error) {
	lock, err := uvlock.ReadLockfile("uv.lock")
	if err != nil {
		if errors.Is(err, uvlock.ErrNotFound) {
			return lockfile.Package{}, fmt.Errorf("uv.lock missing - run `bunpy pm lock` first")
		}
		return lockfile.Package{}, err
	}
	want := pypi.Normalize(pkg)
	for _, p := range lock.Packages {
		if pypi.Normalize(p.Name) == want {
			return p, nil
		}
	}
	return lockfile.Package{}, fmt.Errorf("no lockfile entry for %s", pkg)
}

// preparePristine extracts the wheel cache entry for pin into the
// pristine root. If the cache lacks the entry, the wheel is
// fetched first via the same path `bunpy install` uses.
func preparePristine(pristine string, pin lockfile.Package, cacheDir string) error {
	if _, err := os.Stat(filepath.Join(pristine, ".ready")); err == nil {
		return nil
	}
	if err := os.RemoveAll(pristine); err != nil {
		return err
	}
	wheelPath, err := wheelCachePath(pin, cacheDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(wheelPath); err != nil {
		body, err := fetchAddWheel(pypi.File{Filename: pin.Filename, URL: pin.URL}, pin.Name, cacheDir)
		if err != nil {
			return err
		}
		_ = body // fetchAddWheel writes to cache as a side effect
		if _, err := os.Stat(wheelPath); err != nil {
			return fmt.Errorf("wheel cache missing after fetch: %s", wheelPath)
		}
	}
	if err := patches.Extract(wheelPath, pristine); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pristine, ".ready"), []byte("v1\n"), 0o644)
}

// wheelCachePath returns the on-disk path for the cached wheel.
func wheelCachePath(pin lockfile.Package, cacheDir string) (string, error) {
	root := cacheDir
	if root == "" {
		root = cache.DefaultDir()
	}
	wc, err := cache.NewWheelCache(filepath.Join(root, "wheels"))
	if err != nil {
		return "", err
	}
	return wc.Path(pin.Name, pin.Filename), nil
}

// applyRegisteredPatch is the install-side hook. After a wheel
// install lands, look up (name, version) in the manifest's patch
// table and apply the patch in place. The dist-info INSTALLER is
// rewritten to bunpy-patch on success so later inspection can tell
// patched installs from pinned wheels.
func applyRegisteredPatch(target, name, version string, entries []patches.Entry) (bool, error) {
	entry, ok := patches.Lookup(entries, name, version)
	if !ok {
		return false, nil
	}
	body, err := os.ReadFile(entry.Path)
	if err != nil {
		return false, fmt.Errorf("read patch %s: %w", entry.Path, err)
	}
	if err := patches.Apply(target, body); err != nil {
		return false, err
	}
	if err := stampInstallerTag(target, name, version, patches.InstallerTag); err != nil {
		return false, err
	}
	return true, nil
}

// stampInstallerTag rewrites the dist-info INSTALLER file. Called
// after a successful patch apply.
func stampInstallerTag(target, name, version, tag string) error {
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	candidates := []string{
		filepath.Join(abs, pypi.Normalize(name)+"-"+version+".dist-info", "INSTALLER"),
		filepath.Join(abs, name+"-"+version+".dist-info", "INSTALLER"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return os.WriteFile(p, []byte(tag+"\n"), 0o644)
		}
	}
	return nil
}


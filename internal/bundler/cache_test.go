package bundler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/internal/bundler"
)

const testBunpyVersion = "0.12.7"

// buildAndCache runs Build + WritePYZ + UpdateCache for the given entry.
func buildAndCache(t testing.TB, entry string, opts bundler.Options) string {
	t.Helper()
	entryAbs, err := filepath.Abs(entry)
	if err != nil {
		t.Fatal(err)
	}
	b, err := bundler.Build(entry, opts)
	if err != nil {
		t.Fatal(err)
	}
	outpath := b.OutPath()
	if err := b.WritePYZ(outpath); err != nil {
		t.Fatal(err)
	}
	if err := bundler.UpdateCache(entryAbs, opts, b, outpath, testBunpyVersion); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}
	return outpath
}

// TestBuildCache_HitOnUnchanged builds once, updates the cache, then checks
// that the cache reports a hit with unchanged inputs.
func TestBuildCache_HitOnUnchanged(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	opts := bundler.Options{Outdir: t.TempDir()}

	buildAndCache(t, entry, opts)

	entryAbs, _ := filepath.Abs(entry)
	hit, err := bundler.CheckCache(entryAbs, opts, testBunpyVersion)
	if err != nil {
		t.Fatalf("CheckCache: %v", err)
	}
	if !hit {
		t.Error("expected cache hit on unchanged inputs, got miss")
	}
}

// TestBuildCache_MissOnSourceChange builds, then modifies the source file and
// verifies that the cache is invalidated.
func TestBuildCache_MissOnSourceChange(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	opts := bundler.Options{Outdir: t.TempDir()}

	buildAndCache(t, entry, opts)

	// Modify source.
	if err := os.WriteFile(entry, []byte("x = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entryAbs, _ := filepath.Abs(entry)
	hit, err := bundler.CheckCache(entryAbs, opts, testBunpyVersion)
	if err != nil {
		t.Fatalf("CheckCache: %v", err)
	}
	if hit {
		t.Error("expected cache miss after source change, got hit")
	}
}

// TestBuildCache_MissOnFlagChange verifies that changing a build flag (minify)
// invalidates the cache.
func TestBuildCache_MissOnFlagChange(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	opts := bundler.Options{Outdir: t.TempDir(), Minify: false}

	buildAndCache(t, entry, opts)

	entryAbs, _ := filepath.Abs(entry)
	optsChanged := opts
	optsChanged.Minify = true
	hit, err := bundler.CheckCache(entryAbs, optsChanged, testBunpyVersion)
	if err != nil {
		t.Fatalf("CheckCache: %v", err)
	}
	if hit {
		t.Error("expected cache miss after flag change, got hit")
	}
}

// TestBuildCache_MissOnVersionChange verifies that a different bunpy version
// invalidates the cache.
func TestBuildCache_MissOnVersionChange(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	opts := bundler.Options{Outdir: t.TempDir()}

	buildAndCache(t, entry, opts)

	entryAbs, _ := filepath.Abs(entry)
	hit, err := bundler.CheckCache(entryAbs, opts, "0.12.6") // different version
	if err != nil {
		t.Fatalf("CheckCache: %v", err)
	}
	if hit {
		t.Error("expected cache miss on version change, got hit")
	}
}

// TestBuildCache_MissOnDeletedOutput verifies that deleting the output archive
// causes a cache miss.
func TestBuildCache_MissOnDeletedOutput(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	opts := bundler.Options{Outdir: t.TempDir()}

	outpath := buildAndCache(t, entry, opts)

	if err := os.Remove(outpath); err != nil {
		t.Fatal(err)
	}

	entryAbs, _ := filepath.Abs(entry)
	hit, err := bundler.CheckCache(entryAbs, opts, testBunpyVersion)
	if err != nil {
		t.Fatalf("CheckCache: %v", err)
	}
	if hit {
		t.Error("expected cache miss after output deleted, got hit")
	}
}

// TestBuildCache_NoManifest verifies that CheckCache returns miss (no error)
// when no manifest exists.
func TestBuildCache_NoManifest(t *testing.T) {
	dir := t.TempDir()
	entry := writeFile(t, dir, "app.py", "x = 1\n")
	entryAbs, _ := filepath.Abs(entry)

	hit, err := bundler.CheckCache(entryAbs, bundler.Options{Outdir: t.TempDir()}, testBunpyVersion)
	if err != nil {
		t.Fatalf("CheckCache with no manifest should not error: %v", err)
	}
	if hit {
		t.Error("expected miss when no manifest exists")
	}
}

// BenchmarkCheckCache_Hit measures the cost of a warm cache check (read
// manifest + hash all source files + hash output archive).
func BenchmarkCheckCache_Hit(b *testing.B) {
	dir := b.TempDir()
	// Simulate a 10-file project.
	src := strings.Repeat("x = 1\ny = 2\n", 5)
	entry := writeFile(b, dir, "app.py", src)
	for i := range 9 {
		writeFile(b, dir, filepath.Join("pkg", filepath.Base(entry[:len(entry)-3])+
			"_"+string(rune('a'+i))+".py"), src)
	}
	opts := bundler.Options{Outdir: b.TempDir()}
	buildAndCache(b, entry, opts)

	entryAbs, _ := filepath.Abs(entry)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hit, _ := bundler.CheckCache(entryAbs, opts, testBunpyVersion)
		if !hit {
			b.Fatal("expected cache hit")
		}
	}
}

// TestBuildCache_MultiFile verifies that cache correctly tracks all bundled
// files, not just the entry.
func TestBuildCache_MultiFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "helper.py", "VALUE = 1\n")
	entry := writeFile(t, dir, "app.py", "import helper\n")
	opts := bundler.Options{Outdir: t.TempDir()}

	buildAndCache(t, entry, opts)

	// Modify the imported helper — should be a miss.
	helperPath := filepath.Join(dir, "helper.py")
	if err := os.WriteFile(helperPath, []byte("VALUE = 99\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entryAbs, _ := filepath.Abs(entry)
	hit, err := bundler.CheckCache(entryAbs, opts, testBunpyVersion)
	if err != nil {
		t.Fatalf("CheckCache: %v", err)
	}
	if hit {
		t.Error("expected cache miss after imported file changed, got hit")
	}
}

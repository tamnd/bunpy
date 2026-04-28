package main

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/tamnd/bunpy/v1/pkg/cache"
	"github.com/tamnd/bunpy/v1/pkg/resolver"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// installPins fetches and installs each pin in res.Pins into target
// using a bounded goroutine pool. Pool size is min(len(pins), GOMAXPROCS*2)
// so small lists stay sequential and large ones parallelise across cores.
//
// All goroutines write to distinct dist-info subdirectories, so
// concurrent wheel.Install calls are safe — the atomic tempfile+rename
// pattern inside Install prevents torn writes.
func installPins(pins []resolver.Pin, reg *pypiRegistry, target, cacheDir string) error {
	if len(pins) == 0 {
		return nil
	}
	workers := runtime.GOMAXPROCS(0) * 2
	if workers > len(pins) {
		workers = len(pins)
	}
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan resolver.Pin, len(pins))
	for _, pin := range pins {
		jobs <- pin
	}
	close(jobs)

	errc := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pin := range jobs {
				if err := installOnePin(pin, reg, target, cacheDir); err != nil {
					errc <- fmt.Errorf("%s: %w", pin.Name, err)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errc)
	return <-errc
}

func installOnePin(pin resolver.Pin, reg *pypiRegistry, target, cacheDir string) error {
	f, ok := reg.Pick(pin.Name, pin.Version)
	if !ok {
		return fmt.Errorf("no wheel for %s %s", pin.Name, pin.Version)
	}

	root := cacheDir
	if root == "" {
		root = cache.DefaultDir()
	}

	// Compute deterministic archive key from the wheel's SHA-256.
	sha256hex := f.Hashes["sha256"]
	archiveKey := cache.ArchiveKey(sha256hex)

	// 1. Bunpy's own archive cache.
	if cache.HasArchive(root, archiveKey) {
		return cache.InstallFromArchive(root, archiveKey, target, "bunpy")
	}

	// 2. uv's archive cache (read-only reuse).
	uvRoot := cache.UVCacheDir()
	if uvRoot != "" {
		uvPtr := cache.PointerPath(uvRoot, pin.Name, f.Filename)
		if uvKey, _, ok := cache.ReadPointer(uvPtr); ok && cache.HasArchive(uvRoot, uvKey) {
			return cache.InstallFromArchive(uvRoot, uvKey, target, "bunpy")
		}
	}

	// 3. Download, extract to archive, write pointer, then install.
	body, err := fetchAddWheel(f, pin.Name, cacheDir)
	if err != nil {
		return err
	}
	if sha256hex == "" {
		// Fall back to old path when no hash available (e.g. local fixture server).
		w, err := wheel.OpenReader(f.Filename, body)
		if err != nil {
			return err
		}
		verify := true
		_, err = w.Install(target, wheel.InstallOptions{
			Installer:    "bunpy",
			VerifyHashes: &verify,
		})
		return err
	}
	if err := cache.ExtractToArchive(root, archiveKey, body); err != nil {
		return err
	}
	ptrPath := cache.PointerPath(root, pin.Name, f.Filename)
	_ = cache.WritePointer(ptrPath, archiveKey, sha256hex, f.Filename, f.URL)
	return cache.InstallFromArchive(root, archiveKey, target, "bunpy")
}

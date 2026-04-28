package main

import (
	"fmt"
	"runtime"
	"sync"

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
	body, err := fetchAddWheel(f, pin.Name, cacheDir)
	if err != nil {
		return err
	}
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

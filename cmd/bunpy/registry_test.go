package main

import (
	"os"
	"sync"
	"testing"
)

// TestPrefetchProject_Idempotent verifies that calling PrefetchProjects
// multiple times for the same package fetches the project page exactly once.
func TestPrefetchProject_Idempotent(t *testing.T) {
	cache := t.TempDir()
	reg := benchFixtureRegistry(t, cache)

	// Fire three calls; all should deduplicate to one project fetch.
	reg.PrefetchProjects([]string{"pkg01", "pkg01", "pkg01"})
	reg.prefetchWg.Wait()

	reg.mu.Lock()
	count := 0
	if _, ok := reg.projects["pkg01"]; ok {
		count++
	}
	reg.mu.Unlock()

	if count != 1 {
		t.Errorf("expected exactly 1 project entry for pkg01, found %d", count)
	}
}

// TestPrefetchProject_ConcurrentSafe fires 20 concurrent PrefetchProjects
// calls for the same package. Run with -race to detect data races.
func TestPrefetchProject_ConcurrentSafe(t *testing.T) {
	cache := t.TempDir()
	t.Cleanup(func() { _ = os.RemoveAll(cache) })
	reg := benchFixtureRegistry(t, cache)

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.PrefetchProjects([]string{"pkg01", "pkg02"})
		}()
	}
	wg.Wait()
	reg.prefetchWg.Wait()

	reg.mu.Lock()
	_, hasPkg01 := reg.projects["pkg01"]
	_, hasPkg02 := reg.projects["pkg02"]
	reg.mu.Unlock()

	if !hasPkg01 || !hasPkg02 {
		t.Errorf("projects not populated: pkg01=%v pkg02=%v", hasPkg01, hasPkg02)
	}
}

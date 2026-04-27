package testrunner

import (
	"sync"
)

// RunParallel runs test files concurrently using one goroutine per file.
// Results are returned in the same order as files.
func RunParallel(files []string, opts RunOptions) []FileResult {
	results := make([]FileResult, len(files))
	var wg sync.WaitGroup
	for i, f := range files {
		wg.Add(1)
		go func(idx int, path string) {
			defer wg.Done()
			results[idx] = RunFile(path, opts)
		}(i, f)
	}
	wg.Wait()
	return results
}

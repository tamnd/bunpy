package testrunner

import (
	"os"
	"runtime"
	"strconv"
	"sync"
)

// RunParallel runs test files concurrently using a bounded worker pool.
// Pool size resolves as: RunOptions.Workers → BUNPY_TEST_PARALLELISM → GOMAXPROCS×2.
// Results are returned in the same order as files regardless of completion order.
func RunParallel(files []string, opts RunOptions) []FileResult {
	workers := opts.Workers
	if s := os.Getenv("BUNPY_TEST_PARALLELISM"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			workers = n
		}
	}
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0) * 2
	}
	if workers > len(files) {
		workers = len(files)
	}
	if workers < 1 {
		workers = 1
	}

	results := make([]FileResult, len(files))
	jobs := make(chan int, len(files))
	for i := range files {
		jobs <- i
	}
	close(jobs)

	runFile := RunFile
	if opts.RunFileFunc != nil {
		runFile = opts.RunFileFunc
	}

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				results[idx] = runFile(files[idx], opts)
			}
		}()
	}
	wg.Wait()
	return results
}

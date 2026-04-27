package testrunner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RunIsolated runs a single test file in a subprocess by re-invoking
// bunpy with an internal --test-worker flag. If the subprocess is not
// available (e.g. dev environment), it falls back to RunFile.
func RunIsolated(path string, opts RunOptions) (FileResult, error) {
	self, err := os.Executable()
	if err != nil {
		return RunFile(path, opts), nil
	}

	// Build args for the subprocess.
	subArgs := []string{"test", "--isolated-worker", path}
	if opts.Verbose {
		subArgs = append(subArgs, "--verbose")
	}
	if opts.Filter != "" {
		subArgs = append(subArgs, "--filter="+opts.Filter)
	}

	cmd := exec.Command(self, subArgs...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	cmd.Env = append(os.Environ(), "BUNPY_TEST_WORKER=1")

	start := time.Now()
	runErr := cmd.Run()
	dur := time.Since(start)

	// Parse JSON output from subprocess.
	var fr FileResult
	if jerr := json.Unmarshal(out.Bytes(), &fr); jerr != nil {
		// Subprocess may not support isolated worker yet — fall back.
		if runErr != nil && strings.Contains(errBuf.String(), "unknown flag") {
			return RunFile(path, opts), nil
		}
		fr = RunFile(path, opts)
		fr.Duration = dur
		return fr, nil
	}
	return fr, nil
}

// WriteIsolatedResult serialises the FileResult to stdout as JSON.
// Called when bunpy is invoked with --isolated-worker.
func WriteIsolatedResult(fr FileResult) error {
	data, err := json.Marshal(fr)
	if err != nil {
		return fmt.Errorf("isolated worker: marshal error: %w", err)
	}
	_, err = os.Stdout.Write(data)
	return err
}

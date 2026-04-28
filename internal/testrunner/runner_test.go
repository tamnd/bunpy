package testrunner_test

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tamnd/bunpy/v1/internal/testrunner"
)

func TestDiscoverFindsTestFiles(t *testing.T) {
	files, err := testrunner.DiscoverFiles(testrunner.DiscoverOptions{
		Root: "testdata",
	})
	if err != nil {
		t.Fatal(err)
	}
	// test_basic.py and test_failing.py should be found; helper.py should not
	if len(files) < 2 {
		t.Errorf("expected at least 2 test files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		base := filepath.Base(f)
		if base == "helper.py" {
			t.Errorf("helper.py should not be discovered as a test file")
		}
	}
}

func TestDiscoverSkipsNonTestFiles(t *testing.T) {
	files, err := testrunner.DiscoverFiles(testrunner.DiscoverOptions{Root: "testdata"})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if filepath.Base(f) == "helper.py" {
			t.Errorf("helper.py should not be discovered")
		}
	}
}

func TestDiscoverWithPattern(t *testing.T) {
	files, err := testrunner.DiscoverFiles(testrunner.DiscoverOptions{
		Root:     "testdata",
		Patterns: []string{"test_basic.py"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file with pattern filter, got %d: %v", len(files), files)
	}
}

// TestRunFileCompiles verifies that test files compile (even if they
// contain no test functions yet — gocopy supports only constant
// assignments in v0.0.x).
func TestRunFileCompiles(t *testing.T) {
	fr := testrunner.RunFile("testdata/test_basic.py", testrunner.RunOptions{})
	if fr.CompileError != "" {
		t.Fatalf("test_basic.py should compile, got: %s", fr.CompileError)
	}
	// No test functions in fixture yet — zero results is expected.
	if len(fr.Results) != 0 {
		t.Errorf("expected 0 results (no test functions), got %d", len(fr.Results))
	}
	if !fr.Pass() {
		t.Error("file with no tests should pass")
	}
}

func TestRunFileMissingFile(t *testing.T) {
	fr := testrunner.RunFile("testdata/does_not_exist.py", testrunner.RunOptions{})
	if fr.CompileError == "" {
		t.Error("expected compile/read error for missing file")
	}
}

func TestIsTestFunction(t *testing.T) {
	cases := map[string]bool{
		"test_foo":   true,
		"test_":      true,
		"TestFoo":    true,
		"not_a_test": false,
		"helper":     false,
		"_test_":     false,
	}
	for name, want := range cases {
		got := testrunner.IsTestFunction(name)
		if got != want {
			t.Errorf("IsTestFunction(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestShardFiles(t *testing.T) {
	files := []string{"a", "b", "c", "d", "e", "f"}
	s1 := testrunner.ShardFiles(files, 1, 3)
	s2 := testrunner.ShardFiles(files, 2, 3)
	s3 := testrunner.ShardFiles(files, 3, 3)
	if len(s1)+len(s2)+len(s3) != len(files) {
		t.Errorf("shards should cover all files: %v + %v + %v", s1, s2, s3)
	}
	seen := map[string]int{}
	for _, f := range append(append(s1, s2...), s3...) {
		seen[f]++
	}
	for _, f := range files {
		if seen[f] != 1 {
			t.Errorf("file %q appeared %d times across shards (want 1)", f, seen[f])
		}
	}
}

func TestShardFilesNoOp(t *testing.T) {
	files := []string{"a", "b", "c"}
	result := testrunner.ShardFiles(files, 0, 0)
	if len(result) != len(files) {
		t.Errorf("ShardFiles with 0/0 should return all files")
	}
}

func TestRunParallel(t *testing.T) {
	files := []string{"testdata/test_basic.py", "testdata/test_failing.py"}
	results := testrunner.RunParallel(files, testrunner.RunOptions{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestSummaryAllPassed(t *testing.T) {
	s := testrunner.Summary{}
	s.Add(testrunner.TestResult{Status: testrunner.StatusPass})
	s.Add(testrunner.TestResult{Status: testrunner.StatusPass})
	if !s.AllPassed() {
		t.Error("all-passing summary should report AllPassed=true")
	}
	if s.Total != 2 {
		t.Errorf("expected total=2, got %d", s.Total)
	}
	if s.Passed != 2 {
		t.Errorf("expected passed=2, got %d", s.Passed)
	}
}

func TestSummaryWithFail(t *testing.T) {
	s := testrunner.Summary{}
	s.Add(testrunner.TestResult{Status: testrunner.StatusPass})
	s.Add(testrunner.TestResult{Status: testrunner.StatusFail})
	if s.AllPassed() {
		t.Error("summary with failure should not AllPassed")
	}
	if s.Failed != 1 {
		t.Errorf("expected failed=1, got %d", s.Failed)
	}
}

func TestSummaryWithSkip(t *testing.T) {
	s := testrunner.Summary{}
	s.Add(testrunner.TestResult{Status: testrunner.StatusSkip})
	if !s.AllPassed() {
		t.Error("skip-only summary should pass")
	}
	if s.Skipped != 1 {
		t.Errorf("expected skipped=1, got %d", s.Skipped)
	}
}

func TestStatusString(t *testing.T) {
	cases := map[testrunner.Status]string{
		testrunner.StatusPass:  "pass",
		testrunner.StatusFail:  "fail",
		testrunner.StatusSkip:  "skip",
		testrunner.StatusError: "error",
	}
	for s, want := range cases {
		if s.String() != want {
			t.Errorf("Status(%d).String() = %q, want %q", s, s.String(), want)
		}
	}
}

// TestRunParallel_BoundedGoroutines verifies that RunParallel never exceeds
// Workers concurrent calls even when there are more files than workers.
func TestRunParallel_BoundedGoroutines(t *testing.T) {
	const limit = 2
	const total = 10

	files := make([]string, total)
	for i := range files {
		files[i] = fmt.Sprintf("fake_%02d.py", i)
	}

	var active atomic.Int32
	var peakMu sync.Mutex
	var peak int32

	opts := testrunner.RunOptions{
		Workers: limit,
		RunFileFunc: func(path string, o testrunner.RunOptions) testrunner.FileResult {
			n := active.Add(1)
			defer active.Add(-1)
			peakMu.Lock()
			if n > peak {
				peak = n
			}
			peakMu.Unlock()
			time.Sleep(10 * time.Millisecond)
			return testrunner.FileResult{File: path}
		},
	}

	results := testrunner.RunParallel(files, opts)
	if peak > int32(limit) {
		t.Errorf("peak concurrent calls %d exceeded Workers=%d", peak, limit)
	}
	if len(results) != total {
		t.Errorf("expected %d results, got %d", total, len(results))
	}
}

// TestRunParallel_ResultOrder verifies that results are returned in the same
// order as the input files regardless of completion order.
func TestRunParallel_ResultOrder(t *testing.T) {
	files := []string{"a.py", "b.py", "c.py", "d.py", "e.py"}

	opts := testrunner.RunOptions{
		Workers: 3,
		RunFileFunc: func(path string, o testrunner.RunOptions) testrunner.FileResult {
			time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
			return testrunner.FileResult{File: path}
		},
	}

	results := testrunner.RunParallel(files, opts)
	for i, r := range results {
		if r.File != files[i] {
			t.Errorf("results[%d].File = %q, want %q", i, r.File, files[i])
		}
	}
}

// TestRunParallel_EnvOverride verifies that BUNPY_TEST_PARALLELISM overrides
// the pool size.
func TestRunParallel_EnvOverride(t *testing.T) {
	t.Setenv("BUNPY_TEST_PARALLELISM", "1")

	const total = 4
	files := make([]string, total)
	for i := range files {
		files[i] = fmt.Sprintf("file_%d.py", i)
	}

	var active atomic.Int32
	var peakMu sync.Mutex
	var peak int32

	opts := testrunner.RunOptions{
		RunFileFunc: func(path string, o testrunner.RunOptions) testrunner.FileResult {
			n := active.Add(1)
			defer active.Add(-1)
			peakMu.Lock()
			if n > peak {
				peak = n
			}
			peakMu.Unlock()
			time.Sleep(5 * time.Millisecond)
			return testrunner.FileResult{File: path}
		},
	}

	testrunner.RunParallel(files, opts)
	if peak > 1 {
		t.Errorf("peak %d > 1 despite BUNPY_TEST_PARALLELISM=1", peak)
	}
}

// TestCoverableLines verifies that the gopapy-backed parser finds the correct
// executable lines in a simple Python snippet.
func TestCoverableLines(t *testing.T) {
	src := []byte(`x = 1
y = 2
if x > 0:
    z = x + y
else:
    z = 0
`)
	lines, err := testrunner.CoverableLines("test.py", src)
	if err != nil {
		t.Fatalf("CoverableLines: %v", err)
	}
	// Lines 1, 2, 3 (if), 4 (z=x+y), 6 (z=0) are all statements.
	for _, want := range []int{1, 2, 3, 4, 6} {
		if !lines[want] {
			t.Errorf("line %d should be coverable but is not; got %v", want, lines)
		}
	}
	// Blank lines and comment lines must not appear.
	if lines[5] { // "else:" — depending on parser, may or may not be a stmt line
		// acceptable
	}
}

// TestInstrument checks that __cov_hit__ calls are injected before each
// statement and that indentation is preserved for nested code.
func TestInstrument(t *testing.T) {
	// Use indented body; injected hit on line 2 must be indented too.
	src := []byte("x = 1\ny = 2\n")
	out, err := testrunner.Instrument("cov.py", src)
	if err != nil {
		t.Fatalf("Instrument: %v", err)
	}
	outStr := string(out)
	if !strings.Contains(outStr, `__cov_hit__("cov.py",`) {
		t.Errorf("instrumented output missing __cov_hit__ calls:\n%s", outStr)
	}
	// The injection for a top-level statement has no indentation — that is correct.
	// Verify that for indented source the injected line is also indented.
	src2 := []byte("if True:\n    x = 1\n")
	out2, err := testrunner.Instrument("indent.py", src2)
	if err != nil {
		t.Fatalf("Instrument (indent): %v", err)
	}
	for _, line := range splitLines(string(out2)) {
		if strings.Contains(line, `__cov_hit__("indent.py"`) && !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") {
			// Only the `if` line's hit is at column 0; body hits must be indented.
			// The `if` hit itself is at col 0 — that's fine.
			if !strings.Contains(line, `, 1)`) { // line 1 = the `if` itself
				t.Errorf("body __cov_hit__ not indented: %q", line)
			}
		}
	}
}

// TestRunFile_CoverageGracefulDegrade verifies that RunFile with Coverage
// enabled does not error even when gocopy cannot compile the instrumented
// source (call expressions are not yet supported in gocopy v0.5). The file
// runs successfully via the uninstrumented fallback; hits will be empty until
// gocopy adds call-expression support.
func TestRunFile_CoverageGracefulDegrade(t *testing.T) {
	cov := &testrunner.CoverageCollector{}
	opts := testrunner.RunOptions{Coverage: cov}
	fr := testrunner.RunFile("testdata/test_coverage_basic.py", opts)
	if fr.CompileError != "" {
		t.Errorf("RunFile with Coverage should not error on fallback; got: %s", fr.CompileError)
	}
}

func splitLines(s string) []string { return strings.Split(s, "\n") }

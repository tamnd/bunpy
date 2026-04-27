package testrunner_test

import (
	"path/filepath"
	"testing"

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

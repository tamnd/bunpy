package testrunner

import "time"

// Status of a single test.
type Status int

const (
	StatusPass Status = iota
	StatusFail
	StatusSkip
	StatusError // unexpected panic / VM error
)

func (s Status) String() string {
	switch s {
	case StatusPass:
		return "pass"
	case StatusFail:
		return "fail"
	case StatusSkip:
		return "skip"
	case StatusError:
		return "error"
	}
	return "unknown"
}

// TestResult is the outcome of one test function.
type TestResult struct {
	File     string
	Name     string
	Status   Status
	Duration time.Duration
	// Message holds the failure message or skip reason.
	Message string
}

// FileResult is the aggregated outcome for one test file.
type FileResult struct {
	File    string
	Results []TestResult
	// CompileError is set if the file could not be compiled.
	CompileError string
	Duration     time.Duration
}

// Pass reports whether all tests in the file passed.
func (f *FileResult) Pass() bool {
	if f.CompileError != "" {
		return false
	}
	for _, r := range f.Results {
		if r.Status == StatusFail || r.Status == StatusError {
			return false
		}
	}
	return true
}

// Summary holds aggregate counts across all files.
type Summary struct {
	Total    int
	Passed   int
	Failed   int
	Skipped  int
	Errors   int
	Duration time.Duration
}

func (s *Summary) Add(r TestResult) {
	s.Total++
	s.Duration += r.Duration
	switch r.Status {
	case StatusPass:
		s.Passed++
	case StatusFail:
		s.Failed++
	case StatusSkip:
		s.Skipped++
	case StatusError:
		s.Errors++
	}
}

// AllPassed reports whether the run had no failures or errors.
func (s *Summary) AllPassed() bool {
	return s.Failed == 0 && s.Errors == 0
}

package repl

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// stubRunner replaces runtime.Run for tests so we can assert what the
// REPL hands the eval path without dragging gocopy/goipy in.
type stubRunner struct {
	calls [][]byte
	err   error
}

func (s *stubRunner) run(filename string, source []byte, _ io.Writer, _ io.Writer) (int, error) {
	cp := make([]byte, len(source))
	copy(cp, source)
	s.calls = append(s.calls, cp)
	return 0, s.err
}

func loop(t *testing.T, input string, opts Options) (string, string, *stubRunner) {
	t.Helper()
	stub := &stubRunner{}
	opts.runner = stub.run
	opts.Quiet = true
	var stdout, stderr bytes.Buffer
	rc := Loop(strings.NewReader(input), &stdout, &stderr, opts)
	if rc != 0 {
		t.Fatalf("Loop rc = %d, want 0\nstderr: %s", rc, stderr.String())
	}
	return stdout.String(), stderr.String(), stub
}

func TestQuitEndsLoop(t *testing.T) {
	_, _, stub := loop(t, ":quit\n", Options{})
	if len(stub.calls) != 0 {
		t.Fatalf("expected no runner calls, got %d", len(stub.calls))
	}
}

func TestEofEndsLoop(t *testing.T) {
	_, _, stub := loop(t, "", Options{})
	if len(stub.calls) != 0 {
		t.Fatalf("expected no runner calls, got %d", len(stub.calls))
	}
}

func TestSingleAssignment(t *testing.T) {
	_, _, stub := loop(t, "x = 1\n\n:quit\n", Options{})
	if len(stub.calls) != 1 {
		t.Fatalf("got %d runner calls, want 1", len(stub.calls))
	}
	if got := string(stub.calls[0]); got != "x = 1\n" {
		t.Fatalf("call source = %q, want %q", got, "x = 1\n")
	}
}

func TestMultilineFlush(t *testing.T) {
	_, _, stub := loop(t, "x = 1\ny = 2\n\n:quit\n", Options{})
	if len(stub.calls) != 1 {
		t.Fatalf("got %d runner calls, want 1", len(stub.calls))
	}
	if got := string(stub.calls[0]); got != "x = 1\ny = 2\n" {
		t.Fatalf("call source = %q", got)
	}
}

func TestParseErrorRecoverable(t *testing.T) {
	stub := &stubRunner{err: errors.New("boom")}
	var stdout, stderr bytes.Buffer
	rc := Loop(strings.NewReader("garbage\n\n:quit\n"), &stdout, &stderr,
		Options{Quiet: true, runner: stub.run})
	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("stderr did not surface runner error: %q", stderr.String())
	}
}

func TestMetaHelpPrintsBody(t *testing.T) {
	out, _, _ := loop(t, ":help\n:quit\n", Options{})
	if !strings.Contains(out, "REPL commands") {
		t.Fatalf(":help body missing: %q", out)
	}
}

func TestMetaHistoryEmpty(t *testing.T) {
	out, _, _ := loop(t, ":history\n:quit\n", Options{})
	if out != "" {
		t.Fatalf(":history on empty session printed %q", out)
	}
}

func TestMetaHistoryAfterFlush(t *testing.T) {
	out, _, _ := loop(t, "x = 1\n\n:history\n:quit\n",
		Options{HistorySize: 10})
	if !strings.Contains(out, "x = 1") {
		t.Fatalf(":history did not include the flushed entry: %q", out)
	}
}

func TestMetaClearDiscardsBuffer(t *testing.T) {
	_, _, stub := loop(t, "x = 1\n:clear\n\n:quit\n", Options{})
	if len(stub.calls) != 0 {
		t.Fatalf(":clear failed to drop buffer; got %d runs", len(stub.calls))
	}
}

func TestUnknownMetaCommand(t *testing.T) {
	_, errOut, _ := loop(t, ":nope\n:quit\n", Options{})
	if !strings.Contains(errOut, ":nope") {
		t.Fatalf("unknown meta did not surface name: %q", errOut)
	}
}

func TestQuietSuppressesPrompts(t *testing.T) {
	out, _, _ := loop(t, "x = 1\n\n:quit\n", Options{})
	if strings.Contains(out, ">>>") || strings.Contains(out, "...") {
		t.Fatalf("--quiet still printed prompts: %q", out)
	}
}

func TestBannerInLoudMode(t *testing.T) {
	stub := &stubRunner{}
	var stdout, stderr bytes.Buffer
	Loop(strings.NewReader(":quit\n"), &stdout, &stderr,
		Options{runner: stub.run})
	if !strings.Contains(stdout.String(), "bunpy") {
		t.Fatalf("banner missing: %q", stdout.String())
	}
}

package formatter_test

import (
	"testing"

	"github.com/tamnd/bunpy/v1/internal/formatter"
)

func TestFmtTrailingWhitespace(t *testing.T) {
	out := formatter.Format([]byte("x = 1   \n"))
	if string(out) != "x = 1\n" {
		t.Errorf("got %q", out)
	}
}

func TestFmtCRLF(t *testing.T) {
	out := formatter.Format([]byte("x = 1\r\n"))
	if string(out) != "x = 1\n" {
		t.Errorf("got %q", out)
	}
}

func TestFmtCR(t *testing.T) {
	out := formatter.Format([]byte("x = 1\r"))
	if string(out) != "x = 1\n" {
		t.Errorf("got %q", out)
	}
}

func TestFmtFinalNewlineAdded(t *testing.T) {
	out := formatter.Format([]byte("x = 1"))
	if string(out) != "x = 1\n" {
		t.Errorf("got %q", out)
	}
}

func TestFmtExtraNewlineCollapsed(t *testing.T) {
	out := formatter.Format([]byte("x = 1\n\n\n"))
	if string(out) != "x = 1\n" {
		t.Errorf("got %q", out)
	}
}

func TestFmtTabIndent(t *testing.T) {
	out := formatter.Format([]byte("\tx = 1\n"))
	if string(out) != "    x = 1\n" {
		t.Errorf("got %q", out)
	}
}

func TestFmtDoubleTabIndent(t *testing.T) {
	out := formatter.Format([]byte("\t\tx = 1\n"))
	if string(out) != "        x = 1\n" {
		t.Errorf("got %q", out)
	}
}

func TestFmtNoop(t *testing.T) {
	src := []byte("x = 1\ny = 2\n")
	out := formatter.Format(src)
	if string(out) != string(src) {
		t.Errorf("got %q, want %q", out, src)
	}
	if formatter.Changed(src) {
		t.Error("Changed should be false for already-formatted source")
	}
}

func TestFmtChangedTrue(t *testing.T) {
	if !formatter.Changed([]byte("x = 1   \n")) {
		t.Error("Changed should be true for source with trailing whitespace")
	}
}

package dotenv_test

import (
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/internal/dotenv"
)

func parse(t *testing.T, s string) map[string]string {
	t.Helper()
	m, err := dotenv.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestParseBare(t *testing.T) {
	m := parse(t, "KEY=val\n")
	if m["KEY"] != "val" {
		t.Errorf("got %q", m["KEY"])
	}
}

func TestParseDoubleQuoted(t *testing.T) {
	m := parse(t, `KEY="hello world"`)
	if m["KEY"] != "hello world" {
		t.Errorf("got %q", m["KEY"])
	}
}

func TestParseSingleQuoted(t *testing.T) {
	m := parse(t, "KEY='hello'")
	if m["KEY"] != "hello" {
		t.Errorf("got %q", m["KEY"])
	}
}

func TestParseComment(t *testing.T) {
	m := parse(t, "# this is a comment\nKEY=val\n")
	if _, ok := m["#"]; ok {
		t.Error("comment should not be parsed as key")
	}
	if m["KEY"] != "val" {
		t.Errorf("got %q", m["KEY"])
	}
}

func TestParseBlankLines(t *testing.T) {
	m := parse(t, "\n\nKEY=val\n\n")
	if len(m) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m))
	}
}

func TestParseExport(t *testing.T) {
	m := parse(t, "export KEY=val")
	if m["KEY"] != "val" {
		t.Errorf("got %q", m["KEY"])
	}
}

func TestParseMultiple(t *testing.T) {
	src := "A=1\nB=2\nC=3\n"
	m := parse(t, src)
	if m["A"] != "1" || m["B"] != "2" || m["C"] != "3" {
		t.Errorf("unexpected map: %v", m)
	}
}

func TestParseEmpty(t *testing.T) {
	m := parse(t, "")
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

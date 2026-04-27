package manpages

import (
	"bytes"
	"strings"
	"testing"
)

func TestListNonEmpty(t *testing.T) {
	names, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("List() returned no manpages")
	}
}

func TestEveryPageStartsWithTH(t *testing.T) {
	names, err := List()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		data, err := Page(strings.TrimSuffix(strings.TrimPrefix(n, "bunpy-"), ".1"))
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if !bytes.HasPrefix(data, []byte(".TH")) {
			t.Errorf("%s does not start with .TH: %q", n, data[:min(len(data), 64)])
		}
	}
}

func TestPageMissing(t *testing.T) {
	_, err := Page("frobnicate")
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}

func TestBunpyTopLevelPage(t *testing.T) {
	for _, name := range []string{"", "bunpy"} {
		data, err := Page(name)
		if err != nil {
			t.Fatalf("Page(%q): %v", name, err)
		}
		if !bytes.HasPrefix(data, []byte(".TH BUNPY 1")) {
			t.Errorf("Page(%q) wrong header: %q", name, data[:min(len(data), 64)])
		}
	}
}

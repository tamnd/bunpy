package manifest

import (
	"strings"
	"testing"
)

func TestAddPatchEntryCreatesTable(t *testing.T) {
	src := []byte("[project]\nname = \"demo\"\n")
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, n, err := m.AddPatchEntry("flask@2.3.0", "patches/flask+2.3.0.patch")
	if err != nil || n != 1 {
		t.Fatalf("AddPatchEntry: n=%d err=%v", n, err)
	}
	got := string(out)
	if !strings.Contains(got, "[tool.bunpy.patches]") {
		t.Errorf("missing header: %q", got)
	}
	if !strings.Contains(got, `"flask@2.3.0" = "patches/flask+2.3.0.patch"`) {
		t.Errorf("missing row: %q", got)
	}
}

func TestAddPatchEntryAppendsToExistingTable(t *testing.T) {
	src := []byte(`[project]
name = "demo"

[tool.bunpy.patches]
"flask@2.3.0" = "patches/flask+2.3.0.patch"
`)
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, n, err := m.AddPatchEntry("requests@2.32.3", "patches/requests+2.32.3.patch")
	if err != nil || n != 1 {
		t.Fatalf("AddPatchEntry: n=%d err=%v", n, err)
	}
	got := string(out)
	if !strings.Contains(got, `"flask@2.3.0"`) || !strings.Contains(got, `"requests@2.32.3"`) {
		t.Errorf("expected both rows, got %q", got)
	}
}

func TestAddPatchEntryReplacesExisting(t *testing.T) {
	src := []byte(`[project]
name = "demo"

[tool.bunpy.patches]
"flask@2.3.0" = "old.patch"
`)
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, n, err := m.AddPatchEntry("flask@2.3.0", "new.patch")
	if err != nil || n != 1 {
		t.Fatalf("AddPatchEntry: n=%d err=%v", n, err)
	}
	got := string(out)
	if strings.Contains(got, "old.patch") {
		t.Errorf("old value still present: %q", got)
	}
	if !strings.Contains(got, "new.patch") {
		t.Errorf("new value missing: %q", got)
	}
}

func TestAddPatchEntryNoopWhenIdentical(t *testing.T) {
	src := []byte(`[project]
name = "demo"

[tool.bunpy.patches]
"flask@2.3.0" = "same.patch"
`)
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, n, err := m.AddPatchEntry("flask@2.3.0", "same.patch")
	if err != nil || n != 0 {
		t.Fatalf("expected noop: n=%d err=%v", n, err)
	}
}

func TestRemovePatchEntryDropsRow(t *testing.T) {
	src := []byte(`[project]
name = "demo"

[tool.bunpy.patches]
"flask@2.3.0" = "patches/flask.patch"
"requests@2.32.3" = "patches/requests.patch"
`)
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, n, err := m.RemovePatchEntry("flask@2.3.0")
	if err != nil || n != 1 {
		t.Fatalf("RemovePatchEntry: n=%d err=%v", n, err)
	}
	got := string(out)
	if strings.Contains(got, "flask@2.3.0") {
		t.Errorf("flask still present: %q", got)
	}
	if !strings.Contains(got, "requests@2.32.3") {
		t.Errorf("requests dropped: %q", got)
	}
}

func TestRemovePatchEntryMissingIsNoop(t *testing.T) {
	src := []byte(`[project]
name = "demo"
`)
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, n, err := m.RemovePatchEntry("flask@2.3.0")
	if err != nil || n != 0 {
		t.Fatalf("expected noop: n=%d err=%v", n, err)
	}
	if string(out) != string(src) {
		t.Errorf("source mutated: %q", out)
	}
}

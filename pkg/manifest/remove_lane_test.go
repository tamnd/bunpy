package manifest

import (
	"strings"
	"testing"
)

func TestRemoveDependencyOnlyEntry(t *testing.T) {
	src := `[project]
name = "demo"
version = "0.1.0"
dependencies = [
    "widget>=1.0",
]
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, n, err := m.RemoveDependency("widget")
	if err != nil {
		t.Fatalf("RemoveDependency: %v", err)
	}
	if n != 1 {
		t.Errorf("removed = %d, want 1", n)
	}
	if strings.Contains(string(out), "widget") {
		t.Errorf("widget still present:\n%s", string(out))
	}
	if !strings.Contains(string(out), "dependencies = [") {
		t.Errorf("dependencies array key dropped:\n%s", string(out))
	}
}

func TestRemoveDependencyMiddleEntry(t *testing.T) {
	src := `[project]
name = "demo"
dependencies = [
    "alpha",
    "widget>=1.0",
    "zebra",
]
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, n, err := m.RemoveDependency("widget")
	if err != nil {
		t.Fatalf("RemoveDependency: %v", err)
	}
	if n != 1 {
		t.Errorf("removed = %d, want 1", n)
	}
	got := string(out)
	if strings.Contains(got, "widget") {
		t.Errorf("widget still present:\n%s", got)
	}
	if !strings.Contains(got, `"alpha"`) || !strings.Contains(got, `"zebra"`) {
		t.Errorf("siblings missing:\n%s", got)
	}
}

func TestRemoveDependencyByNormalisedName(t *testing.T) {
	// PEP 503: Normalize collapses Foo_Bar / FOO-BAR / foo.bar.
	src := `[project]
name = "demo"
dependencies = [
    "Foo_Bar==1.0",
]
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, n, err := m.RemoveDependency("foo-bar")
	if err != nil {
		t.Fatalf("RemoveDependency: %v", err)
	}
	if n != 1 {
		t.Errorf("removed = %d, want 1 (PEP 503 match)", n)
	}
	if strings.Contains(string(out), "Foo_Bar") {
		t.Errorf("entry should be gone:\n%s", string(out))
	}
}

func TestRemoveDependencyMissingIsNoop(t *testing.T) {
	src := `[project]
name = "demo"
dependencies = ["widget"]
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, n, err := m.RemoveDependency("notapkg")
	if err != nil {
		t.Fatalf("RemoveDependency: %v", err)
	}
	if n != 0 {
		t.Errorf("removed = %d, want 0", n)
	}
	if string(out) != src {
		t.Errorf("source mutated on no-op:\n%s", string(out))
	}
}

func TestRemoveGroupDependency(t *testing.T) {
	src := `[project]
name = "demo"

[dependency-groups]
dev = [
    "pytest>=8",
    "widget==1.0",
]
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, n, err := m.RemoveGroupDependency("dev", "widget")
	if err != nil {
		t.Fatalf("RemoveGroupDependency: %v", err)
	}
	if n != 1 {
		t.Errorf("removed = %d, want 1", n)
	}
	if strings.Contains(string(out), "widget") {
		t.Errorf("widget still present:\n%s", string(out))
	}
	if !strings.Contains(string(out), "pytest>=8") {
		t.Errorf("pytest sibling lost:\n%s", string(out))
	}
}

func TestRemoveDependencyAllLanes(t *testing.T) {
	// Same package in main, dev group, optional, and peer. One call
	// must scrub every copy.
	src := `[project]
name = "demo"
dependencies = ["widget>=1.0"]

[project.optional-dependencies]
web = ["widget==1.1"]

[dependency-groups]
dev = ["widget==1.0"]

[tool.bunpy]
peer-dependencies = ["widget==1.2"]
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, n, err := m.RemoveDependencyAllLanes("widget")
	if err != nil {
		t.Fatalf("RemoveDependencyAllLanes: %v", err)
	}
	if n != 4 {
		t.Errorf("removed = %d, want 4", n)
	}
	if strings.Contains(string(out), "widget") {
		t.Errorf("widget still present in some lane:\n%s", string(out))
	}
	// The lane tables themselves should still be present (we do not
	// delete now-empty groups; that is an explicit user action).
	for _, hdr := range []string{
		"[project.optional-dependencies]",
		"[dependency-groups]",
		"[tool.bunpy]",
	} {
		if !strings.Contains(string(out), hdr) {
			t.Errorf("table %s lost on full-lane delete:\n%s", hdr, string(out))
		}
	}
}

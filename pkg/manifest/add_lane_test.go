package manifest

import (
	"strings"
	"testing"
)

func TestAddOptionalDependencyCreatesTable(t *testing.T) {
	src := `[project]
name = "demo"
version = "0.1.0"
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, err := m.AddOptionalDependency("web", "flask>=2")
	if err != nil {
		t.Fatalf("AddOptionalDependency: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "[project.optional-dependencies]") {
		t.Errorf("missing section header in:\n%s", got)
	}
	if !strings.Contains(got, `"flask>=2"`) {
		t.Errorf("missing spec in:\n%s", got)
	}

	m2, err := Parse(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if list := m2.Project.OptionalDeps["web"]; len(list) != 1 || list[0] != "flask>=2" {
		t.Errorf("optional-deps web: got %v", list)
	}
}

func TestAddOptionalDependencyReplacesExisting(t *testing.T) {
	src := `[project]
name = "demo"
version = "0.1.0"

[project.optional-dependencies]
web = ["flask>=1"]
`
	m, _ := Parse([]byte(src))
	out, err := m.AddOptionalDependency("web", "flask>=2")
	if err != nil {
		t.Fatalf("AddOptionalDependency: %v", err)
	}
	if strings.Contains(string(out), `"flask>=1"`) {
		t.Errorf("old spec still present:\n%s", out)
	}
	if !strings.Contains(string(out), `"flask>=2"`) {
		t.Errorf("new spec missing:\n%s", out)
	}
}

func TestAddGroupDependencyCreatesTable(t *testing.T) {
	src := `[project]
name = "demo"
version = "0.1.0"
`
	m, _ := Parse([]byte(src))
	out, err := m.AddGroupDependency("dev", "pytest>=8")
	if err != nil {
		t.Fatalf("AddGroupDependency: %v", err)
	}
	if !strings.Contains(string(out), "[dependency-groups]") {
		t.Errorf("missing dependency-groups header in:\n%s", out)
	}
	m2, _ := Parse(out)
	if got := m2.DependencyGroups["dev"]; len(got) != 1 || got[0] != "pytest>=8" {
		t.Errorf("dependency-groups dev: got %v", got)
	}
}

func TestAddPeerDependencyAppendsToToolBunpy(t *testing.T) {
	src := `[project]
name = "demo"
version = "0.1.0"

[tool.bunpy]
profile = "fast"
`
	m, _ := Parse([]byte(src))
	out, err := m.AddPeerDependency("typing-extensions>=4")
	if err != nil {
		t.Fatalf("AddPeerDependency: %v", err)
	}
	if !strings.Contains(string(out), `"typing-extensions>=4"`) {
		t.Errorf("missing peer dep in:\n%s", out)
	}
	m2, _ := Parse(out)
	if got := m2.Tool.PeerDependencies; len(got) != 1 || got[0] != "typing-extensions>=4" {
		t.Errorf("peer deps: got %v", got)
	}
}

func TestAddPeerDependencyCreatesToolBunpyTable(t *testing.T) {
	src := `[project]
name = "demo"
version = "0.1.0"
`
	m, _ := Parse([]byte(src))
	out, err := m.AddPeerDependency("typing-extensions>=4")
	if err != nil {
		t.Fatalf("AddPeerDependency: %v", err)
	}
	if !strings.Contains(string(out), "[tool.bunpy]") {
		t.Errorf("missing [tool.bunpy] section in:\n%s", out)
	}
}

func TestAddOptionalDependencyRejectsBadGroup(t *testing.T) {
	src := `[project]
name = "demo"
version = "0.1.0"
`
	m, _ := Parse([]byte(src))
	if _, err := m.AddOptionalDependency("bad name", "flask"); err == nil {
		t.Error("bad group name: want error")
	}
	if _, err := m.AddOptionalDependency("", "flask"); err == nil {
		t.Error("empty group name: want error")
	}
}

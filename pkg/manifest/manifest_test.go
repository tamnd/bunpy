package manifest

import (
	"strings"
	"testing"
)

func TestParseMinimal(t *testing.T) {
	src := `
[project]
name = "demo"
version = "0.1.0"
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.Project.Name != "demo" {
		t.Errorf("name: got %q want %q", m.Project.Name, "demo")
	}
	if m.Project.Version != "0.1.0" {
		t.Errorf("version: got %q want %q", m.Project.Version, "0.1.0")
	}
	if len(m.Warnings) != 0 {
		t.Errorf("warnings: got %v want none", m.Warnings)
	}
}

func TestParseFull(t *testing.T) {
	src := `
[project]
name = "full"
version = "1.2.3"
description = "a full example"
requires-python = ">=3.10"
dependencies = ["requests>=2", "click"]
keywords = ["alpha", "beta"]
classifiers = ["Programming Language :: Python"]

[project.optional-dependencies]
dev = ["pytest", "ruff"]
docs = ["sphinx"]

[project.urls]
Home = "https://example.com"
Source = "https://example.com/src"

[project.scripts]
full-cli = "full.cli:main"

[project.gui-scripts]
full-gui = "full.gui:main"

[project.entry-points."pkg.plugins"]
one = "full.plugins:one"

[[project.authors]]
name = "Ada"
email = "ada@example.com"

[[project.maintainers]]
name = "Bob"

[project.license]
text = "MIT"

[project.readme]
file = "README.md"
content-type = "text/markdown"
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	p := m.Project
	if p.Name != "full" || p.Version != "1.2.3" {
		t.Errorf("name/version: got %q/%q", p.Name, p.Version)
	}
	if p.Description != "a full example" {
		t.Errorf("description: got %q", p.Description)
	}
	if p.RequiresPython != ">=3.10" {
		t.Errorf("requires-python: got %q", p.RequiresPython)
	}
	if len(p.Dependencies) != 2 || p.Dependencies[0] != "requests>=2" {
		t.Errorf("deps: got %v", p.Dependencies)
	}
	if len(p.OptionalDeps["dev"]) != 2 || p.OptionalDeps["dev"][0] != "pytest" {
		t.Errorf("optional dev: got %v", p.OptionalDeps["dev"])
	}
	if p.URLs["Home"] != "https://example.com" {
		t.Errorf("urls home: got %q", p.URLs["Home"])
	}
	if p.Scripts["full-cli"] != "full.cli:main" {
		t.Errorf("scripts: got %v", p.Scripts)
	}
	if p.GUIScripts["full-gui"] != "full.gui:main" {
		t.Errorf("gui-scripts: got %v", p.GUIScripts)
	}
	if p.EntryPoints["pkg.plugins"]["one"] != "full.plugins:one" {
		t.Errorf("entry-points: got %v", p.EntryPoints)
	}
	if len(p.Authors) != 1 || p.Authors[0].Name != "Ada" || p.Authors[0].Email != "ada@example.com" {
		t.Errorf("authors: got %v", p.Authors)
	}
	if len(p.Maintainers) != 1 || p.Maintainers[0].Name != "Bob" {
		t.Errorf("maintainers: got %v", p.Maintainers)
	}
	if p.License.Text != "MIT" {
		t.Errorf("license text: got %q", p.License.Text)
	}
	if p.Readme.File != "README.md" || p.Readme.ContentType != "text/markdown" {
		t.Errorf("readme: got %+v", p.Readme)
	}
	if len(p.Keywords) != 2 || p.Keywords[0] != "alpha" {
		t.Errorf("keywords: got %v", p.Keywords)
	}
	if len(p.Classifiers) != 1 {
		t.Errorf("classifiers: got %v", p.Classifiers)
	}
}

func TestParseLicenseSPDX(t *testing.T) {
	src := `
[project]
name = "spdx"
version = "0.0.1"
license = "MIT OR Apache-2.0"
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.Project.License.SPDX != "MIT OR Apache-2.0" {
		t.Errorf("spdx: got %q", m.Project.License.SPDX)
	}
}

func TestParseReadmeShorthand(t *testing.T) {
	src := `
[project]
name = "rd"
version = "0.0.1"
readme = "README.rst"
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.Project.Readme.File != "README.rst" {
		t.Errorf("readme file: got %q", m.Project.Readme.File)
	}
}

func TestParseMissingProject(t *testing.T) {
	src := `[build-system]
requires = ["setuptools"]
`
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatal("Parse: want error, got nil")
	}
	m, err := ParseOpts([]byte(src), LoadOptions{Strict: false})
	if err != nil {
		t.Fatalf("soft Parse: %v", err)
	}
	if len(m.Warnings) == 0 {
		t.Error("soft mode: want warnings, got none")
	}
}

func TestParseMissingName(t *testing.T) {
	src := `[project]
version = "0.0.1"
`
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatal("Parse: want error, got nil")
	}
	m, err := ParseOpts([]byte(src), LoadOptions{Strict: false})
	if err != nil {
		t.Fatalf("soft Parse: %v", err)
	}
	found := false
	for _, w := range m.Warnings {
		if strings.Contains(w, "name") {
			found = true
		}
	}
	if !found {
		t.Errorf("soft warnings: want one mentioning name, got %v", m.Warnings)
	}
}

func TestParseInvalidName(t *testing.T) {
	for _, bad := range []string{" leading", "trailing ", "with/slash", "white space", "_under"} {
		src := "[project]\nname = \"" + bad + "\"\n"
		if _, err := Parse([]byte(src)); err == nil {
			t.Errorf("Parse %q: want error, got nil", bad)
		}
	}
}

func TestParseValidNames(t *testing.T) {
	for _, ok := range []string{"a", "ab", "demo", "my-pkg", "my_pkg", "my.pkg", "Pkg42"} {
		src := "[project]\nname = \"" + ok + "\"\n"
		if _, err := Parse([]byte(src)); err != nil {
			t.Errorf("Parse %q: want ok, got %v", ok, err)
		}
	}
}

func TestParseDynamicConflict(t *testing.T) {
	src := `
[project]
name = "dyn"
version = "0.0.1"
dynamic = ["version"]
`
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatal("Parse: want error, got nil")
	}
}

func TestParseDynamicNoConflict(t *testing.T) {
	src := `
[project]
name = "dyn"
dynamic = ["version"]
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(m.Project.Dynamic) != 1 || m.Project.Dynamic[0] != "version" {
		t.Errorf("dynamic: got %v", m.Project.Dynamic)
	}
}

func TestToolBunpyKept(t *testing.T) {
	src := `
[project]
name = "tb"
version = "0.0.1"

[tool.bunpy]
profile = "fast"
parallel = 4

[tool.bunpy.cache]
ttl = 600
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.Tool.Raw == nil {
		t.Fatal("tool.bunpy: not preserved")
	}
	if m.Tool.Raw["profile"] != "fast" {
		t.Errorf("tool.bunpy.profile: got %v", m.Tool.Raw["profile"])
	}
	if cache, ok := m.Tool.Raw["cache"].(map[string]any); !ok || cache["ttl"] == nil {
		t.Errorf("tool.bunpy.cache: got %v", m.Tool.Raw["cache"])
	}
}

func TestToolOtherKept(t *testing.T) {
	src := `
[project]
name = "to"
version = "0.0.1"

[tool.ruff]
line-length = 100

[tool.bunpy]
profile = "fast"
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	tool, ok := m.Other["tool"].(map[string]any)
	if !ok {
		t.Fatalf("Other.tool: not a map, got %T", m.Other["tool"])
	}
	if _, has := tool["bunpy"]; has {
		t.Error("Other.tool must not duplicate bunpy")
	}
	ruff, ok := tool["ruff"].(map[string]any)
	if !ok {
		t.Fatalf("Other.tool.ruff: missing or wrong type")
	}
	if ruff["line-length"] == nil {
		t.Errorf("Other.tool.ruff.line-length: got nil")
	}
}

func TestUnknownTopLevelKept(t *testing.T) {
	src := `
[project]
name = "ut"
version = "0.0.1"

[build-system]
requires = ["setuptools"]
build-backend = "setuptools.build_meta"
`
	m, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	bs, ok := m.Other["build-system"].(map[string]any)
	if !ok {
		t.Fatalf("Other.build-system: missing or wrong type")
	}
	if bs["build-backend"] != "setuptools.build_meta" {
		t.Errorf("build-system: got %v", bs)
	}
}

func TestParseInvalidTOML(t *testing.T) {
	if _, err := Parse([]byte("not = = toml")); err == nil {
		t.Fatal("Parse: want error on bad toml")
	}
}

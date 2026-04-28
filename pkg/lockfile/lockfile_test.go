package lockfile

import (
	"strings"
	"testing"
)

func TestUpsertReplaces(t *testing.T) {
	l := &Lock{Packages: []Package{{Name: "Widget", Version: "1.0.0"}}}
	l.Upsert(Package{Name: "widget", Version: "1.1.0"})
	if len(l.Packages) != 1 {
		t.Fatalf("packages: %+v", l.Packages)
	}
	if l.Packages[0].Version != "1.1.0" {
		t.Errorf("version not replaced: %+v", l.Packages[0])
	}
}

func TestUpsertNew(t *testing.T) {
	l := &Lock{Packages: []Package{{Name: "alpha", Version: "1.0"}}}
	l.Upsert(Package{Name: "beta", Version: "2.0"})
	if len(l.Packages) != 2 {
		t.Fatalf("packages: %+v", l.Packages)
	}
}

func TestRemove(t *testing.T) {
	l := &Lock{Packages: []Package{{Name: "alpha"}, {Name: "beta"}}}
	if !l.Remove("Alpha") {
		t.Fatal("Remove returned false for existing entry")
	}
	if len(l.Packages) != 1 || l.Packages[0].Name != "beta" {
		t.Errorf("packages after remove: %+v", l.Packages)
	}
	if l.Remove("nope") {
		t.Error("Remove returned true for missing entry")
	}
}

func TestFind(t *testing.T) {
	l := &Lock{Packages: []Package{{Name: "Foo_Bar", Version: "1.0"}}}
	if p, ok := l.Find("foo-bar"); !ok || p.Version != "1.0" {
		t.Errorf("Find: %+v ok=%v", p, ok)
	}
	if _, ok := l.Find("missing"); ok {
		t.Error("Find: found missing package")
	}
}

func TestHashDependenciesStable(t *testing.T) {
	a := HashDependencies([]string{"alpha", "widget>=1.0"})
	b := HashDependencies([]string{"widget>=1.0", "alpha"})
	if a != b {
		t.Errorf("hash differs: %q vs %q", a, b)
	}
	if !strings.HasPrefix(a, "sha256:") {
		t.Errorf("missing prefix: %q", a)
	}
}

func TestHashDependenciesIgnoresWhitespace(t *testing.T) {
	a := HashDependencies([]string{"widget>=1.0"})
	b := HashDependencies([]string{"  widget>=1.0  ", ""})
	if a != b {
		t.Errorf("whitespace not normalised: %q vs %q", a, b)
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Foo_Bar":     "foo-bar",
		"foo.bar":     "foo-bar",
		"foo___bar":   "foo-bar",
		"FooBar":      "foobar",
		"foo--__.bar": "foo-bar",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHashLanesEqualsHashDependenciesForMainOnly(t *testing.T) {
	deps := []string{"widget>=1.0", "requests>=2"}
	a := HashDependencies(deps)
	b := HashLanes(map[string][]string{"main": deps})
	if a != b {
		t.Errorf("hash mismatch: HashDependencies=%s HashLanes=%s", a, b)
	}
	c := HashLanes(map[string][]string{"main": deps, "dev": nil, "peer": {}})
	if c != a {
		t.Errorf("empty lanes should not change hash: got %s want %s", c, a)
	}
}

func TestHashLanesIncludesEveryLane(t *testing.T) {
	a := HashLanes(map[string][]string{"main": {"widget"}})
	b := HashLanes(map[string][]string{"main": {"widget"}, "dev": {"pytest"}})
	c := HashLanes(map[string][]string{"main": {"widget", "pytest"}})
	if a == b {
		t.Errorf("dev lane should change hash")
	}
	if b == c {
		t.Errorf("moving spec from dev to main should change hash")
	}
}

func TestHashLanesOptionalGroupsOrderStable(t *testing.T) {
	a := HashLanes(map[string][]string{"main": {"x"}, "optional:b": {"y"}, "optional:a": {"z"}})
	b := HashLanes(map[string][]string{"optional:a": {"z"}, "main": {"x"}, "optional:b": {"y"}})
	if a != b {
		t.Errorf("hash should not depend on map iteration order: a=%s b=%s", a, b)
	}
}

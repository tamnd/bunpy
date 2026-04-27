package why

import (
	"slices"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
)

func mustLock(t *testing.T, pins ...lockfile.Package) *lockfile.Lock {
	t.Helper()
	l := &lockfile.Lock{Version: lockfile.Version}
	l.Packages = append(l.Packages, pins...)
	return l
}

func mustManifest(t *testing.T, body string) *manifest.Manifest {
	t.Helper()
	m, err := manifest.Parse([]byte(body))
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	return m
}

func staticRequires(edges map[string][]Requires) RequiresFunc {
	return func(name, version string) ([]Requires, error) {
		key := name + "@" + version
		return edges[key], nil
	}
}

const baseManifest = `
[project]
name = "demo"
version = "0.0.1"
dependencies = ["requests>=2.32"]

[project.optional-dependencies]
tracing = ["httpx>=0.27"]

[tool.bunpy]
peer-dependencies = []
`

func TestWalkLeafToProject(t *testing.T) {
	lf := mustLock(t,
		lockfile.Package{Name: "requests", Version: "2.32.3", Lanes: []string{lockfile.LaneMain}},
		lockfile.Package{Name: "urllib3", Version: "2.2.3"},
	)
	mf := mustManifest(t, baseManifest)
	g, err := BuildGraph(lf, mf, staticRequires(map[string][]Requires{
		"requests@2.32.3": {{Name: "urllib3", Specifier: "<3,>=1.21.1"}},
	}))
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	res, err := Walk(g, "urllib3", 0)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if got, want := len(res.Chains), 1; got != want {
		t.Fatalf("chains: got %d, want %d", got, want)
	}
	c := res.Chains[0]
	if c.Lane != lockfile.LaneMain {
		t.Fatalf("lane: %q, want %q", c.Lane, lockfile.LaneMain)
	}
	if got, want := len(c.Edges), 3; got != want {
		t.Fatalf("edges: got %d, want %d", got, want)
	}
	if c.Edges[0].Name != "urllib3" || c.Edges[1].Name != "requests" || c.Edges[2].Name != ProjectName {
		t.Fatalf("chain shape: %+v", c.Edges)
	}
}

func TestWalkDiamondTwoChains(t *testing.T) {
	lf := mustLock(t,
		lockfile.Package{Name: "requests", Version: "2.32.3", Lanes: []string{lockfile.LaneMain}},
		lockfile.Package{Name: "httpx", Version: "0.27.0", Lanes: []string{lockfile.OptionalLane("tracing")}},
		lockfile.Package{Name: "urllib3", Version: "2.2.3"},
	)
	mf := mustManifest(t, baseManifest)
	g, err := BuildGraph(lf, mf, staticRequires(map[string][]Requires{
		"requests@2.32.3": {{Name: "urllib3", Specifier: "<3,>=1.21.1"}},
		"httpx@0.27.0":    {{Name: "urllib3", Specifier: ">=2.0"}},
	}))
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	res, err := Walk(g, "urllib3", 0)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if got, want := len(res.Chains), 2; got != want {
		t.Fatalf("chains: got %d, want %d (chains=%+v)", got, want, res.Chains)
	}
	lanes := []string{res.Chains[0].Lane, res.Chains[1].Lane}
	if !contains(lanes, lockfile.LaneMain) || !contains(lanes, lockfile.OptionalLane("tracing")) {
		t.Fatalf("lanes: %v", lanes)
	}
}

func TestWalkDepthCap(t *testing.T) {
	lf := mustLock(t,
		lockfile.Package{Name: "requests", Version: "2.32.3", Lanes: []string{lockfile.LaneMain}},
		lockfile.Package{Name: "urllib3", Version: "2.2.3"},
		lockfile.Package{Name: "charset-normalizer", Version: "3.3.0"},
	)
	mf := mustManifest(t, baseManifest)
	g, err := BuildGraph(lf, mf, staticRequires(map[string][]Requires{
		"requests@2.32.3": {{Name: "urllib3", Specifier: "<3,>=1.21.1"}},
		"urllib3@2.2.3":   {{Name: "charset-normalizer", Specifier: ""}},
	}))
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	res, err := Walk(g, "charset-normalizer", 2)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	for _, c := range res.Chains {
		if len(c.Edges) > 2 {
			t.Fatalf("depth cap not honoured: %d edges in %+v", len(c.Edges), c.Edges)
		}
	}
}

func TestWalkRefusesCycles(t *testing.T) {
	lf := mustLock(t,
		lockfile.Package{Name: "a", Version: "1", Lanes: []string{lockfile.LaneMain}},
		lockfile.Package{Name: "b", Version: "1"},
	)
	mf := mustManifest(t, `
[project]
name = "demo"
version = "0.0.1"
dependencies = ["a"]
`)
	g, err := BuildGraph(lf, mf, staticRequires(map[string][]Requires{
		"a@1": {{Name: "b"}},
		"b@1": {{Name: "a"}}, // cycle
	}))
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	res, err := Walk(g, "b", 0)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(res.Chains) == 0 {
		t.Fatalf("expected at least one chain, got none")
	}
	for _, c := range res.Chains {
		if len(c.Edges) > 16 {
			t.Fatalf("cycle: chain grew to %d edges", len(c.Edges))
		}
	}
}

func TestWalkOverlayState(t *testing.T) {
	lf := mustLock(t,
		lockfile.Package{Name: "requests", Version: "2.32.3", Lanes: []string{lockfile.LaneMain}},
	)
	mf := mustManifest(t, `
[project]
name = "demo"
version = "0.0.1"
dependencies = ["requests"]

[tool.bunpy]

[tool.bunpy.patches]
"requests@2.32.3" = "patches/requests+2.32.3.patch"

[tool.bunpy.links]
flask = "/abs/path/flask"
`)
	g, err := BuildGraph(lf, mf, staticRequires(nil))
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	res, err := Walk(g, "requests", 0)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if !res.Patched || res.Installer != "bunpy-patch" {
		t.Fatalf("expected patched/bunpy-patch, got patched=%v installer=%q", res.Patched, res.Installer)
	}
}

func TestWalkMissingPin(t *testing.T) {
	lf := mustLock(t)
	g, err := BuildGraph(lf, nil, nil)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	if _, err := Walk(g, "ghost", 0); err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("Walk: expected error mentioning ghost, got %v", err)
	}
}

func contains(haystack []string, needle string) bool {
	return slices.Contains(haystack, needle)
}

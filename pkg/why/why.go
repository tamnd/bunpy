// Package why answers "what depends on this package?". v0.1.11
// builds a forward dependency graph from uv.lock plus per-pin
// Requires-Dist (delivered via a RequiresFunc), inverts it, and
// walks upward from a queried pin to the project's direct
// requirements.
package why

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// ProjectName is the sentinel parent label for a top-of-chain edge:
// the project itself, declared in pyproject.toml.
const ProjectName = "@project"

// ErrNotFound is returned by Walk when name does not match any pin
// in the lockfile.
var ErrNotFound = errors.New("why: package not in lockfile")

// Pin is one row in the lockfile, with the normalised name as the
// graph key.
type Pin struct {
	Name    string
	Version string
	Lanes   []string
}

// Requires is one Requires-Dist edge after marker evaluation. The
// caller (cmd/bunpy/why.go) supplies a RequiresFunc bound to the
// wheel cache.
type Requires struct {
	Name      string // normalised
	Specifier string // raw spec as written by the parent
}

// RequiresFunc returns the runtime Requires-Dist edges for one pin.
// Markers are already evaluated against the host environment.
type RequiresFunc func(name, version string) ([]Requires, error)

// Edge is one parent->child link. Lane is non-empty only on the
// terminal @project edge.
type Edge struct {
	Name      string
	Version   string
	Specifier string
	Lane      string
}

// Chain is one path from the queried pin upward to @project. Edges
// are ordered child -> parent -> ... -> @project.
type Chain struct {
	Lane  string
	Edges []Edge
}

// Result is what `bunpy why <pkg>` prints. Linked / Patched are
// surfaced from the manifest's overlay tables.
type Result struct {
	Name      string
	Version   string
	Installer string
	Linked    bool
	Patched   bool
	Chains    []Chain
}

// Graph is the resolved forward + reverse view of uv.lock. The
// keys are PEP 503-normalised names.
type Graph struct {
	Pins    map[string]Pin
	Forward map[string][]Requires
	Reverse map[string][]Edge

	// directLanes records the lanes in which a name is declared as
	// a project-level direct dependency. Multiple lanes possible.
	directLanes map[string][]string

	// linked / patched are name sets harvested from the manifest's
	// overlay tables.
	linked  map[string]bool
	patched map[string]bool
}

// BuildGraph resolves the forward dependency graph by calling fetch
// for every pin in lf, then inverts it. mf supplies the direct-dep
// lane mapping plus overlay state (links, patches).
func BuildGraph(lf *lockfile.Lock, mf *manifest.Manifest, fetch RequiresFunc) (*Graph, error) {
	if lf == nil {
		return nil, errors.New("why: nil lockfile")
	}
	g := &Graph{
		Pins:        map[string]Pin{},
		Forward:     map[string][]Requires{},
		Reverse:     map[string][]Edge{},
		directLanes: map[string][]string{},
		linked:      map[string]bool{},
		patched:     map[string]bool{},
	}
	for _, p := range lf.Packages {
		key := lockfile.Normalize(p.Name)
		g.Pins[key] = Pin{Name: p.Name, Version: p.Version, Lanes: append([]string(nil), p.Lanes...)}
	}
	for _, p := range lf.Packages {
		key := lockfile.Normalize(p.Name)
		if fetch == nil {
			continue
		}
		reqs, err := fetch(p.Name, p.Version)
		if err != nil {
			return nil, fmt.Errorf("why: requires for %s@%s: %w", p.Name, p.Version, err)
		}
		g.Forward[key] = reqs
		for _, r := range reqs {
			child := lockfile.Normalize(r.Name)
			parent, ok := g.Pins[key]
			if !ok {
				continue
			}
			g.Reverse[child] = append(g.Reverse[child], Edge{
				Name:      parent.Name,
				Version:   parent.Version,
				Specifier: r.Specifier,
			})
		}
	}
	for k, parents := range g.Reverse {
		sort.Slice(parents, func(i, j int) bool {
			return lockfile.Normalize(parents[i].Name) < lockfile.Normalize(parents[j].Name)
		})
		g.Reverse[k] = parents
	}
	if mf != nil {
		applyManifest(g, mf)
	}
	return g, nil
}

// applyManifest populates directLanes plus the overlay sets.
func applyManifest(g *Graph, mf *manifest.Manifest) {
	add := func(spec, lane string) {
		name, ok := requirementName(spec)
		if !ok {
			return
		}
		key := lockfile.Normalize(name)
		g.directLanes[key] = appendUnique(g.directLanes[key], lane)
	}
	for _, d := range mf.Project.Dependencies {
		add(d, lockfile.LaneMain)
	}
	for grp, deps := range mf.Project.OptionalDeps {
		for _, d := range deps {
			add(d, lockfile.OptionalLane(grp))
		}
	}
	for _, d := range mf.Tool.PeerDependencies {
		add(d, lockfile.LanePeer)
	}
	for grp, deps := range mf.DependencyGroups {
		for _, d := range deps {
			add(d, lockfile.GroupLane(grp))
		}
	}
	if mf.Tool.Raw != nil {
		if t, ok := mf.Tool.Raw["links"].(map[string]any); ok {
			for k := range t {
				g.linked[lockfile.Normalize(k)] = true
			}
		}
		if t, ok := mf.Tool.Raw["patches"].(map[string]any); ok {
			for k := range t {
				name, _, _ := strings.Cut(k, "@")
				g.patched[lockfile.Normalize(name)] = true
			}
		}
	}
}

func appendUnique(out []string, v string) []string {
	if slices.Contains(out, v) {
		return out
	}
	return append(out, v)
}

// requirementName returns the PEP 503-style raw name from a
// pyproject dependency specifier, e.g. "requests>=2.32" -> "requests".
func requirementName(spec string) (string, bool) {
	rd, err := wheel.ParseRequiresDist(spec)
	if err != nil || rd.Name == "" {
		return "", false
	}
	return rd.Name, true
}

// Walk inverts the graph from name and returns the result tree.
// depth caps the number of steps from the leaf upward; depth <= 0
// disables the cap.
func Walk(g *Graph, name string, depth int) (*Result, error) {
	if g == nil {
		return nil, errors.New("why: nil graph")
	}
	key := lockfile.Normalize(name)
	pin, ok := g.Pins[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	res := &Result{
		Name:      pin.Name,
		Version:   pin.Version,
		Installer: "bunpy",
		Linked:    g.linked[key],
		Patched:   g.patched[key],
	}
	switch {
	case res.Linked:
		res.Installer = "bunpy-link"
	case res.Patched:
		res.Installer = "bunpy-patch"
	}
	res.Chains = walkUp(g, pin, depth)
	return res, nil
}

// walkUp does a depth-first enumeration of every path from pin
// upward, terminating at @project edges. Cycles are guarded via
// per-path visited sets.
func walkUp(g *Graph, leaf Pin, depth int) []Chain {
	type frame struct {
		path    []Edge
		current string
		visited map[string]bool
	}
	startKey := lockfile.Normalize(leaf.Name)
	leafEdge := Edge{Name: leaf.Name, Version: leaf.Version}
	stack := []frame{{
		path:    []Edge{leafEdge},
		current: startKey,
		visited: map[string]bool{startKey: true},
	}}
	var out []Chain
	for len(stack) > 0 {
		fr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if depth > 0 && len(fr.path) > depth {
			continue
		}
		topName := edgeNameKey(fr.path[len(fr.path)-1])
		lanes := g.directLanes[topName]
		appendedAtTop := false
		for _, lane := range lanes {
			projectEdge := Edge{Name: ProjectName, Lane: lane}
			chain := Chain{Lane: lane, Edges: append(append([]Edge(nil), fr.path...), projectEdge)}
			out = append(out, chain)
			appendedAtTop = true
		}
		parents := g.Reverse[fr.current]
		if len(parents) == 0 && !appendedAtTop {
			// Orphan: no project lane and no reverse edge. Emit a
			// chain anyway so the user sees the pin sits in the
			// lockfile without a known root.
			out = append(out, Chain{Edges: append([]Edge(nil), fr.path...)})
			continue
		}
		for _, p := range parents {
			pkey := lockfile.Normalize(p.Name)
			if fr.visited[pkey] {
				continue
			}
			next := frame{
				path:    append(append([]Edge(nil), fr.path...), p),
				current: pkey,
				visited: copyVisited(fr.visited, pkey),
			}
			stack = append(stack, next)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return chainKey(out[i]) < chainKey(out[j])
	})
	return out
}

func edgeNameKey(e Edge) string {
	if e.Name == ProjectName {
		return ""
	}
	return lockfile.Normalize(e.Name)
}

func copyVisited(in map[string]bool, add string) map[string]bool {
	out := make(map[string]bool, len(in)+1)
	maps.Copy(out, in)
	out[add] = true
	return out
}

func chainKey(c Chain) string {
	var sb strings.Builder
	sb.WriteString(c.Lane)
	sb.WriteByte('|')
	for _, e := range c.Edges {
		sb.WriteString(lockfile.Normalize(e.Name))
		sb.WriteByte('@')
		sb.WriteString(e.Version)
		sb.WriteByte('/')
	}
	return sb.String()
}

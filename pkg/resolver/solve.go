package resolver

import (
	"fmt"
	"slices"
	"sort"

	"github.com/tamnd/bunpy/v1/pkg/version"
)

// Solver runs the resolution loop. Construct via New and call Solve.
type Solver struct {
	Registry Registry
	// Incompats records every learned/derived incompatibility for
	// reporting. v0.1.5 does not learn during search; this is wired
	// for future CDCL.
	Incompats []Incompatibility
	// Locked maps PEP 503 normalised package names to a preferred
	// version. When a locked version satisfies every recorded
	// constraint it is picked over the highest matching candidate.
	// This is how `bunpy update <pkg>` holds unrelated pins steady
	// while letting one name drift. v0.1.7 adds this; an empty map
	// (or nil) preserves v0.1.5 behaviour exactly.
	Locked   map[string]string
	maxSteps int
}

// New returns a solver bound to reg.
func New(reg Registry) *Solver {
	return &Solver{Registry: reg, maxSteps: 10000}
}

// Resolution is the solver output: one pin per package, plus the
// requirements that produced it.
type Resolution struct {
	Pins []Pin
}

// Pin is a single resolved (package, version) edge with the parent
// that pulled it in.
type Pin struct {
	Name    string
	Version string
	From    string // empty for root requirements
}

// Solve runs propagate-then-decide over root, returning a
// Resolution or a *Conflict.
func (s *Solver) Solve(root []Requirement) (*Resolution, error) {
	state := &searchState{
		solver:  s,
		pending: append([]frame(nil), rootFrames(root)...),
	}
	if err := state.run(); err != nil {
		return nil, err
	}
	out := &Resolution{}
	for _, a := range state.partial.Decisions() {
		out.Pins = append(out.Pins, Pin{Name: a.Package, Version: a.Version, From: a.DecidedBy})
	}
	sort.Slice(out.Pins, func(i, j int) bool { return out.Pins[i].Name < out.Pins[j].Name })
	return out, nil
}

type frame struct {
	req  Requirement
	from string
}

func rootFrames(root []Requirement) []frame {
	out := make([]frame, len(root))
	for i, r := range root {
		out[i] = frame{req: r, from: ""}
	}
	return out
}

type searchState struct {
	solver  *Solver
	partial PartialSolution
	pending []frame // unprocessed constraints
	steps   int
}

func (s *searchState) run() error {
	for {
		s.steps++
		if s.steps > s.solver.maxSteps {
			return fmt.Errorf("resolver: exceeded %d steps (likely a cycle)", s.solver.maxSteps)
		}
		// Phase 1: record every pending constraint. Conflicts with
		// existing pins surface here.
		for _, fr := range s.pending {
			if err := s.absorb(fr); err != nil {
				return err
			}
		}
		s.pending = nil
		// Phase 2: pick the next undecided package and pin it. The
		// pin's deps become pending and feed the next iteration.
		next := s.nextUndecided()
		if next == "" {
			return nil
		}
		if err := s.decide(next); err != nil {
			return err
		}
	}
}

// absorb records fr's constraint in the partial solution. If pkg is
// already pinned, the new spec is verified against the pinned
// version.
func (s *searchState) absorb(fr frame) error {
	pkg := fr.req.Name
	if pinned := s.partial.Pinned(pkg); pinned != "" && !fr.req.Spec.Match(pinned) {
		return s.conflict(pkg, fr, pinned)
	}
	s.partial.Push(Assignment{
		Package: pkg, Spec: fr.req.Spec, DecidedBy: fr.from,
	})
	return nil
}

// nextUndecided returns the first package with constraints but no
// pinned version, in the order constraints were first recorded.
func (s *searchState) nextUndecided() string {
	seen := map[string]bool{}
	for _, a := range s.partial.entries {
		if seen[a.Package] {
			continue
		}
		seen[a.Package] = true
		if s.partial.Pinned(a.Package) == "" {
			return a.Package
		}
	}
	return ""
}

// decide picks the highest version for pkg that satisfies every
// recorded constraint, queues its deps, and records the decision.
func (s *searchState) decide(pkg string) error {
	combined := IntersectSpecs(s.partial.Constraints(pkg))
	versions, err := s.solver.Registry.Versions(pkg)
	if err != nil {
		return fmt.Errorf("resolver: %s: %w", pkg, err)
	}
	chosen := ""
	if locked, ok := s.solver.Locked[pkg]; ok && combined.Match(locked) && slices.Contains(versions, locked) {
		chosen = locked
	}
	if chosen == "" {
		chosen = version.Highest(combined, versions)
	}
	if chosen == "" {
		return s.noVersion(pkg, combined, versions)
	}
	s.partial.Push(Assignment{
		Package: pkg, Version: chosen, Spec: combined, Decision: true,
	})
	deps, err := s.solver.Registry.Dependencies(pkg, chosen)
	if err != nil {
		return fmt.Errorf("resolver: %s@%s: %w", pkg, chosen, err)
	}
	for _, d := range deps {
		s.pending = append(s.pending, frame{req: d, from: pkg})
	}
	return nil
}

func (s *searchState) conflict(pkg string, fr frame, pinned string) error {
	terms := []Term{
		{Package: pkg, Spec: fr.req.Spec, Positive: true},
	}
	inc := Incompatibility{Terms: terms, Cause: CauseDependency, From: fr.from}
	s.solver.Incompats = append(s.solver.Incompats, inc)
	return &Conflict{
		Package:   pkg,
		Pinned:    pinned,
		Requested: fr.req.Spec,
		From:      fr.from,
		Decisions: snapshotDecisions(&s.partial),
	}
}

func (s *searchState) noVersion(pkg string, combined version.Spec, candidates []string) error {
	terms := []Term{{Package: pkg, Spec: combined, Positive: true}}
	inc := Incompatibility{Terms: terms, Cause: CauseNoVersion, From: pkg}
	s.solver.Incompats = append(s.solver.Incompats, inc)
	return &Conflict{
		Package:    pkg,
		Requested:  combined,
		Candidates: append([]string(nil), candidates...),
		Decisions:  snapshotDecisions(&s.partial),
		NoVersion:  true,
	}
}

func snapshotDecisions(p *PartialSolution) []Pin {
	decs := p.Decisions()
	out := make([]Pin, len(decs))
	for i, a := range decs {
		out[i] = Pin{Name: a.Package, Version: a.Version, From: a.DecidedBy}
	}
	return out
}

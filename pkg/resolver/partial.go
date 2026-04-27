package resolver

import "github.com/tamnd/bunpy/v1/pkg/version"

// Assignment is one entry in the partial solution. A decision
// pins a concrete version; a derivation only narrows the spec.
type Assignment struct {
	Package   string
	Version   string       // empty for derivations
	Spec      version.Spec // narrowed spec at this point
	Decision  bool         // true if a version was pinned here
	DecidedBy string       // parent package that introduced the constraint
	Level     int
}

// PartialSolution is the ordered log of assignments. v0.1.5 uses it
// as a stack for backtracking; the explicit shape is here so future
// CDCL passes can rewind to a specific decision level.
type PartialSolution struct {
	entries []Assignment
}

func (p *PartialSolution) Len() int { return len(p.entries) }

func (p *PartialSolution) Push(a Assignment) { p.entries = append(p.entries, a) }

func (p *PartialSolution) Pop() (Assignment, bool) {
	if len(p.entries) == 0 {
		return Assignment{}, false
	}
	last := p.entries[len(p.entries)-1]
	p.entries = p.entries[:len(p.entries)-1]
	return last, true
}

// Decisions returns only the pinned (decision) assignments in
// insertion order.
func (p *PartialSolution) Decisions() []Assignment {
	var out []Assignment
	for _, a := range p.entries {
		if a.Decision {
			out = append(out, a)
		}
	}
	return out
}

// Pinned returns the version pinned for pkg, or "" if no decision
// has been made yet.
func (p *PartialSolution) Pinned(pkg string) string {
	for _, a := range p.entries {
		if a.Decision && a.Package == pkg {
			return a.Version
		}
	}
	return ""
}

// Constraints returns every spec accumulated so far for pkg in the
// order they were added. Callers intersect the slice via
// IntersectSpecs to get the effective constraint.
func (p *PartialSolution) Constraints(pkg string) []version.Spec {
	var out []version.Spec
	for _, a := range p.entries {
		if a.Package == pkg {
			out = append(out, a.Spec)
		}
	}
	return out
}

// Package resolver is bunpy's PubGrub-inspired dependency solver.
//
// The solver takes a Registry that knows how to enumerate versions
// and read their Requires-Dist edges, and produces a single-version
// pin per package such that every requirement is satisfied. When
// no satisfying assignment exists the solver returns a Conflict
// describing the chain of decisions that led there.
//
// The implementation is a deterministic, depth-first backtracker
// with constraint accumulation. PubGrub's core data shapes (Term,
// Incompatibility, partial solution) are present so the algorithm
// can grow CDCL behaviour in later rungs without a rewrite.
package resolver

import (
	"fmt"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/version"
)

// Term ties a package name to a version specifier. The empty Spec
// matches every version. Positive=true means "the chosen version
// must satisfy Spec"; Positive=false is the dual ("must not satisfy
// Spec"). v0.1.5 only emits positive terms; the field is here so
// later CDCL passes can flip a term during conflict resolution.
type Term struct {
	Package  string
	Spec     version.Spec
	Positive bool
}

// NewPositive builds a positive term from a raw specifier string.
func NewPositive(pkg, spec string) (Term, error) {
	s, err := version.ParseSpec(spec)
	if err != nil {
		return Term{}, fmt.Errorf("resolver: %s: %w", pkg, err)
	}
	return Term{Package: pkg, Spec: s, Positive: true}, nil
}

// Satisfies reports whether v passes the term.
func (t Term) Satisfies(v string) bool {
	ok := t.Spec.Match(v)
	if !t.Positive {
		return !ok
	}
	return ok
}

// String renders the term for diagnostics.
func (t Term) String() string {
	prefix := ""
	if !t.Positive {
		prefix = "not "
	}
	if len(t.Spec.Clauses) == 0 {
		return prefix + t.Package
	}
	parts := make([]string, len(t.Spec.Clauses))
	for i, c := range t.Spec.Clauses {
		parts[i] = string(c.Op) + c.Version
	}
	return fmt.Sprintf("%s%s %s", prefix, t.Package, strings.Join(parts, ","))
}

// IntersectSpecs ANDs every clause from each input spec. Empty
// inputs are dropped. The result accepts only versions that pass
// every clause.
func IntersectSpecs(specs []version.Spec) version.Spec {
	out := version.Spec{}
	for _, s := range specs {
		out.Clauses = append(out.Clauses, s.Clauses...)
	}
	return out
}

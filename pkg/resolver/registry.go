package resolver

import "github.com/tamnd/bunpy/v1/pkg/version"

// Registry is the data source the solver pulls from. Implementations
// proxy a PEP 691 index plus a wheel-metadata fetcher. Decoupling
// here keeps the algorithm offline-testable: the resolver test suite
// uses an in-memory map, while production wiring lands in
// cmd/bunpy/add.go.
type Registry interface {
	// Versions returns every release of pkg, newest-first ordering
	// is not required — the solver sorts via version.Highest.
	Versions(pkg string) ([]string, error)
	// Dependencies returns the Requires-Dist edges of pkg@ver that
	// pass the marker environment. Optional-deps are excluded by the
	// caller via marker.Env.Extra in v0.1.5.
	Dependencies(pkg, ver string) ([]Requirement, error)
}

// Requirement is one Requires-Dist edge resolved into a name+spec
// pair. Markers have already been evaluated by the registry.
type Requirement struct {
	Name string
	Spec version.Spec
}

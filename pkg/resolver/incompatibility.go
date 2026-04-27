package resolver

import (
	"fmt"
	"strings"
)

// Cause records why an incompatibility exists. v0.1.5 surfaces
// three: the root requirement set, a Requires-Dist edge from a
// concrete version, or "no version found" when intersection
// produces an empty candidate set.
type Cause int

const (
	CauseRoot Cause = iota
	CauseDependency
	CauseNoVersion
)

func (c Cause) String() string {
	switch c {
	case CauseRoot:
		return "root"
	case CauseDependency:
		return "dependency"
	case CauseNoVersion:
		return "no-version"
	}
	return "unknown"
}

// Incompatibility is a set of terms whose conjunction is impossible.
// In CDCL terms it is a clause; v0.1.5's solver only logs them for
// reporting, but storing them keeps the door open to learning.
type Incompatibility struct {
	Terms []Term
	Cause Cause
	From  string // package or version that introduced the rule
}

func (i Incompatibility) String() string {
	parts := make([]string, len(i.Terms))
	for k, t := range i.Terms {
		parts[k] = t.String()
	}
	return fmt.Sprintf("{%s} (%s from %s)",
		strings.Join(parts, " ∧ "), i.Cause, i.From)
}

package resolver

import (
	"fmt"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/version"
)

// Conflict is the structured failure the solver returns when no
// valid assignment exists. CLI callers format it via Error(); test
// callers reach into the fields directly.
type Conflict struct {
	Package    string
	Pinned     string       // already-decided version, when applicable
	Requested  version.Spec // failing constraint
	From       string       // package that introduced the failing edge
	Candidates []string     // versions tried (no-version case)
	Decisions  []Pin        // partial solution at the failure point
	NoVersion  bool         // true when the candidate set is empty
}

func (c *Conflict) Error() string {
	var b strings.Builder
	b.WriteString("resolver: ")
	switch {
	case c.NoVersion:
		fmt.Fprintf(&b, "no candidate satisfies %s%s", c.Package, formatSpec(c.Requested))
	case c.Pinned != "":
		fmt.Fprintf(&b, "%s is already pinned to %s but %s asks for %s%s",
			c.Package, c.Pinned, descFrom(c.From), c.Package, formatSpec(c.Requested))
	default:
		fmt.Fprintf(&b, "%s could not be resolved", c.Package)
	}
	if len(c.Decisions) > 0 {
		b.WriteString("\n  decided so far:")
		for _, d := range c.Decisions {
			fmt.Fprintf(&b, "\n    %s == %s", d.Name, d.Version)
		}
	}
	return b.String()
}

func formatSpec(s version.Spec) string {
	if len(s.Clauses) == 0 {
		return ""
	}
	parts := make([]string, len(s.Clauses))
	for i, c := range s.Clauses {
		parts[i] = string(c.Op) + c.Version
	}
	return " " + strings.Join(parts, ",")
}

func descFrom(from string) string {
	if from == "" {
		return "the root manifest"
	}
	return from
}

// Package version implements the slice of PEP 440 that bunpy's
// v0.1.x `bunpy pm lock` flow needs: parse a version, parse a spec,
// compare two versions, and pick the highest version in a candidate
// list that satisfies a spec.
//
// Supported operators: `==`, `!=`, `>=`, `>`, `<=`, `<`, `~=`, and
// wildcard forms `==X.*` / `!=X.*`. Arbitrary equality (`===`) is not
// supported. This surface is deliberately narrow; future rungs extend
// it as real PyPI projects force it (v0.10.16 adds wildcard support).
package version

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Op is one PEP 440 specifier operator.
type Op string

const (
	OpEQ Op = "=="
	OpNE Op = "!="
	OpGE Op = ">="
	OpGT Op = ">"
	OpLE Op = "<="
	OpLT Op = "<"
	OpCA Op = "~=" // compatible release
)

// Clause is one operator + version pair. When Wildcard is true the
// Version field holds the prefix (e.g. "1" for "==1.*", "1.2" for
// "==1.2.*") and Op must be OpEQ or OpNE.
type Clause struct {
	Op       Op
	Version  string
	Wildcard bool // true for ==X.* / !=X.*
}

// Spec is a comma-joined list of clauses. The empty Spec matches
// every version.
type Spec struct {
	Clauses []Clause
}

// ParseSpec parses a comma-joined PEP 440 specifier. An empty or
// whitespace-only string returns the empty Spec.
func ParseSpec(s string) (Spec, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Spec{}, nil
	}
	out := Spec{}
	for _, raw := range strings.Split(s, ",") {
		c, err := parseClause(strings.TrimSpace(raw))
		if err != nil {
			return Spec{}, err
		}
		out.Clauses = append(out.Clauses, c)
	}
	return out, nil
}

func parseClause(s string) (Clause, error) {
	if s == "" {
		return Clause{}, errors.New("version: empty clause")
	}
	if strings.HasPrefix(s, "===") {
		return Clause{}, fmt.Errorf("version: arbitrary equality (===) not supported")
	}
	for _, op := range []Op{OpEQ, OpNE, OpGE, OpLE, OpCA, OpGT, OpLT} {
		if strings.HasPrefix(s, string(op)) {
			v := strings.TrimSpace(strings.TrimPrefix(s, string(op)))
			if v == "" {
				return Clause{}, fmt.Errorf("version: missing version after %q", op)
			}
			// Wildcard: ==X.* or !=X.* (PEP 440 §6.3)
			if strings.HasSuffix(v, ".*") {
				if op != OpEQ && op != OpNE {
					return Clause{}, fmt.Errorf("version: wildcard only valid with == or !=: %q", s)
				}
				prefix := strings.TrimSuffix(v, ".*")
				if err := validateVersion(prefix); err != nil {
					return Clause{}, fmt.Errorf("version: bad wildcard prefix %q: %w", prefix, err)
				}
				return Clause{Op: op, Version: prefix, Wildcard: true}, nil
			}
			if err := validateVersion(v); err != nil {
				return Clause{}, err
			}
			return Clause{Op: op, Version: v}, nil
		}
	}
	if strings.Contains(s, "*") {
		return Clause{}, fmt.Errorf("version: wildcard only valid after == or !=: %q", s)
	}
	// Bare version is shorthand for ==.
	if err := validateVersion(s); err != nil {
		return Clause{}, err
	}
	return Clause{Op: OpEQ, Version: s}, nil
}

func validateVersion(v string) error {
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return errors.New("version: empty")
	}
	if _, err := parseVersion(v); err != nil {
		return err
	}
	return nil
}

// Match reports whether v satisfies every clause in s.
func (s Spec) Match(v string) bool {
	for _, c := range s.Clauses {
		if !c.match(v) {
			return false
		}
	}
	return true
}

func (c Clause) match(v string) bool {
	if c.Wildcard {
		return matchWildcard(v, c.Version, c.Op == OpEQ)
	}
	cmp := Compare(v, c.Version)
	switch c.Op {
	case OpEQ:
		return equalForSpec(v, c.Version)
	case OpNE:
		return !equalForSpec(v, c.Version)
	case OpGE:
		return cmp >= 0
	case OpGT:
		return cmp > 0
	case OpLE:
		return cmp <= 0
	case OpLT:
		return cmp < 0
	case OpCA:
		return matchCompatible(v, c.Version)
	}
	return false
}

// matchWildcard implements PEP 440 wildcard matching. prefix is the
// version before ".*" (e.g. "1" for "==1.*"). When eq is true this
// is an == wildcard; when false it is a != wildcard.
//
// ==X.Y.* is equivalent to >= X.Y.dev0, == X.Y.* (prefix match on
// the release tuple). Concretely, v matches if its release tuple
// starts with the prefix's release tuple.
func matchWildcard(v, prefix string, eq bool) bool {
	pv, errV := parseVersion(v)
	pp, errP := parseVersion(prefix)
	if errV != nil || errP != nil {
		return false
	}
	n := len(pp.Release)
	if len(pv.Release) < n {
		return !eq // "1.0" does NOT match "==1.0.1.*" semantics; negate for !=
	}
	for i := 0; i < n; i++ {
		if pv.Release[i] != pp.Release[i] {
			return !eq
		}
	}
	return eq
}

// equalForSpec mirrors PEP 440 == semantics: both versions are
// canonicalised (leading 'v' stripped, release segments compared as
// integers), local versions are stripped before compare so
// `1.0+foo == 1.0` holds.
func equalForSpec(a, b string) bool {
	pa, _ := parseVersion(a)
	pb, _ := parseVersion(b)
	pa.Local, pb.Local = "", ""
	return Compare(pa.String(), pb.String()) == 0
}

// matchCompatible implements ~= per PEP 440: equivalent to >=X.Y,
// <X+1.
func matchCompatible(v, spec string) bool {
	pv, errV := parseVersion(v)
	ps, errS := parseVersion(spec)
	if errV != nil || errS != nil {
		return false
	}
	if len(ps.Release) < 2 {
		return false
	}
	if Compare(pv.String(), ps.String()) < 0 {
		return false
	}
	upper := append([]int{}, ps.Release[:len(ps.Release)-1]...)
	upper[len(upper)-1]++
	upperV := joinRelease(upper)
	return Compare(pv.String(), upperV) < 0
}

func joinRelease(parts []int) string {
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strconv.Itoa(p)
	}
	return strings.Join(out, ".")
}

// Compare returns -1, 0, +1 by PEP 440 ordering on the release
// segment first, then dev/pre/post adjustments. Local-version
// segments are ignored in compare per PEP 440 §10.
func Compare(a, b string) int {
	pa, _ := parseVersion(a)
	pb, _ := parseVersion(b)
	return pa.compareTo(pb)
}

// Highest returns the highest candidate satisfying s, or "" when
// none matches. Pre-releases (a/b/rc/dev) are excluded unless every
// matching candidate is a pre-release or s explicitly pins a
// pre-release version.
func Highest(s Spec, candidates []string) string {
	allowPre := specRequiresPre(s)
	best := ""
	for _, c := range candidates {
		if !s.Match(c) {
			continue
		}
		if !allowPre && isPreRelease(c) {
			continue
		}
		if best == "" || Compare(c, best) > 0 {
			best = c
		}
	}
	if best != "" {
		return best
	}
	if allowPre {
		return ""
	}
	// Fall back to pre-releases when nothing else matches.
	for _, c := range candidates {
		if !s.Match(c) {
			continue
		}
		if best == "" || Compare(c, best) > 0 {
			best = c
		}
	}
	return best
}

func isPreRelease(v string) bool {
	pv, err := parseVersion(v)
	if err != nil {
		return false
	}
	return pv.Pre != nil || pv.Dev != nil
}

func specRequiresPre(s Spec) bool {
	for _, c := range s.Clauses {
		if isPreRelease(c.Version) {
			return true
		}
	}
	return false
}

// version is an internal parsed version used for Compare.
type version struct {
	Epoch    int
	Release  []int
	Pre      *segment // a/b/rc/...
	Post     *segment // post
	Dev      *segment // dev
	Local    string
	original string
}

type segment struct {
	Label string
	N     int
}

func parseVersion(s string) (version, error) {
	original := s
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return version{}, errors.New("version: empty")
	}
	v := version{original: original}
	// Local segment (after +).
	if i := strings.Index(s, "+"); i >= 0 {
		v.Local = s[i+1:]
		s = s[:i]
	}
	// Epoch.
	if i := strings.Index(s, "!"); i >= 0 {
		n, err := strconv.Atoi(s[:i])
		if err != nil {
			return version{}, fmt.Errorf("version: bad epoch in %q", original)
		}
		v.Epoch = n
		s = s[i+1:]
	}
	// Pre/Post/Dev segments. Find the first letter.
	idx := -1
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			idx = i
			break
		}
	}
	tail := ""
	if idx >= 0 {
		tail = s[idx:]
		s = s[:idx]
	}
	s = strings.TrimRight(s, ".-_")
	for _, part := range splitOnDot(s) {
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return version{}, fmt.Errorf("version: bad release segment %q in %q", part, original)
		}
		v.Release = append(v.Release, n)
	}
	if tail != "" {
		if err := parseTail(&v, tail); err != nil {
			return version{}, fmt.Errorf("version: %w (in %q)", err, original)
		}
	}
	return v, nil
}

func splitOnDot(s string) []string {
	return strings.Split(s, ".")
}

func parseTail(v *version, tail string) error {
	// Tail can carry combinations: e.g. "rc1.dev2" or "post1.dev3".
	// Split on dots, hyphens, and underscores, then walk segments.
	tail = strings.ReplaceAll(tail, "-", ".")
	tail = strings.ReplaceAll(tail, "_", ".")
	for _, raw := range strings.Split(tail, ".") {
		if raw == "" {
			continue
		}
		label, n := splitLabelN(raw)
		switch normalizeLabel(label) {
		case "a", "b", "rc":
			if v.Pre != nil {
				return fmt.Errorf("multiple pre-release segments")
			}
			v.Pre = &segment{Label: normalizeLabel(label), N: n}
		case "post":
			if v.Post != nil {
				return fmt.Errorf("multiple post-release segments")
			}
			v.Post = &segment{Label: "post", N: n}
		case "dev":
			if v.Dev != nil {
				return fmt.Errorf("multiple dev-release segments")
			}
			v.Dev = &segment{Label: "dev", N: n}
		default:
			return fmt.Errorf("unknown segment %q", raw)
		}
	}
	return nil
}

func splitLabelN(s string) (string, int) {
	i := 0
	for i < len(s) && !(s[i] >= '0' && s[i] <= '9') {
		i++
	}
	label := s[:i]
	if i == len(s) {
		return label, 0
	}
	n, err := strconv.Atoi(s[i:])
	if err != nil {
		return label, 0
	}
	return label, n
}

func normalizeLabel(s string) string {
	s = strings.ToLower(s)
	switch s {
	case "alpha":
		return "a"
	case "beta":
		return "b"
	case "c", "pre", "preview":
		return "rc"
	case "rev", "r":
		return "post"
	}
	return s
}

func (v version) String() string { return v.original }

// compareTo orders by epoch, release (right-padded with zeros),
// dev/pre/post per PEP 440 §13.
func (a version) compareTo(b version) int {
	if a.Epoch != b.Epoch {
		return cmpInt(a.Epoch, b.Epoch)
	}
	if c := cmpReleases(a.Release, b.Release); c != 0 {
		return c
	}
	// dev < pre < post == final
	ar := releaseRank(a)
	br := releaseRank(b)
	if ar.Tier != br.Tier {
		return cmpInt(ar.Tier, br.Tier)
	}
	if ar.LabelRank != br.LabelRank {
		return cmpInt(ar.LabelRank, br.LabelRank)
	}
	if ar.N != br.N {
		return cmpInt(ar.N, br.N)
	}
	return 0
}

type rankedRelease struct {
	Tier      int // 0=dev, 1=pre, 2=final, 3=post
	LabelRank int // a<b<rc, post=0
	N         int
}

func releaseRank(v version) rankedRelease {
	switch {
	case v.Dev != nil && v.Pre == nil && v.Post == nil:
		return rankedRelease{Tier: 0, N: v.Dev.N}
	case v.Pre != nil && v.Post == nil:
		// a < b < rc
		rank := 0
		switch v.Pre.Label {
		case "a":
			rank = 0
		case "b":
			rank = 1
		case "rc":
			rank = 2
		}
		// dev sub-tier: pre+dev < pre
		tier := 1
		if v.Dev != nil {
			return rankedRelease{Tier: 0, LabelRank: rank, N: v.Pre.N*10000 + v.Dev.N}
		}
		return rankedRelease{Tier: tier, LabelRank: rank, N: v.Pre.N}
	case v.Post != nil:
		return rankedRelease{Tier: 3, N: v.Post.N}
	default:
		return rankedRelease{Tier: 2}
	}
}

func cmpReleases(a, b []int) int {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(a) {
			ai = a[i]
		}
		if i < len(b) {
			bi = b[i]
		}
		if ai != bi {
			return cmpInt(ai, bi)
		}
	}
	return 0
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

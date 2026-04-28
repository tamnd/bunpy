// Package lockfile defines the in-memory lock representation shared
// across all pm sub-commands. Disk I/O is handled by pkg/uvlock
// (uv.lock format). This package holds no file readers or writers.
package lockfile

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Schema version. Carried on Lock.Version to distinguish resolver outputs.
const Version = 1

// Lock is the in-memory lock: a flat list of resolved packages.
type Lock struct {
	Version     int
	ContentHash string
	Packages    []Package
}

// Package is one resolved pin.
type Package struct {
	Name     string
	Version  string
	Filename string
	URL      string
	Hash     string
	Size     int64
	Lanes    []string
	// Sdist fields carry the source distribution for uv.lock compatibility.
	SdistURL  string
	SdistHash string
	SdistSize int64
}

// Lane labels carried on Package.Lanes.
const (
	LaneMain = "main"
	LaneDev  = "dev"
	LanePeer = "peer"
)

// OptionalLane returns the canonical lane label for an optional group.
func OptionalLane(group string) string { return "optional:" + group }

// GroupLane returns the canonical lane label for a [dependency-groups] entry.
func GroupLane(name string) string {
	if name == LaneDev {
		return LaneDev
	}
	return "group:" + name
}

// Upsert inserts p by PEP 503-normalised name, replacing any existing entry.
func (l *Lock) Upsert(p Package) {
	want := Normalize(p.Name)
	for i := range l.Packages {
		if Normalize(l.Packages[i].Name) == want {
			l.Packages[i] = p
			return
		}
	}
	l.Packages = append(l.Packages, p)
}

// Remove drops the package with the given name. Returns true when removed.
func (l *Lock) Remove(name string) bool {
	want := Normalize(name)
	for i := range l.Packages {
		if Normalize(l.Packages[i].Name) == want {
			l.Packages = append(l.Packages[:i], l.Packages[i+1:]...)
			return true
		}
	}
	return false
}

// Find returns the package row for name or (Package{}, false).
func (l *Lock) Find(name string) (Package, bool) {
	want := Normalize(name)
	for _, p := range l.Packages {
		if Normalize(p.Name) == want {
			return p, true
		}
	}
	return Package{}, false
}

// Normalize applies PEP 503: lower-case and collapse runs of -_. to -.
func Normalize(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	prev := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == '_' || c == '.' {
			if prev == '-' {
				continue
			}
			sb.WriteByte('-')
			prev = '-'
			continue
		}
		sb.WriteByte(c)
		prev = c
	}
	return sb.String()
}

// HashDependencies returns sha256:<hex> over sorted, trimmed dependency specs.
func HashDependencies(deps []string) string {
	cleaned := make([]string, 0, len(deps))
	for _, d := range deps {
		t := strings.TrimSpace(d)
		if t == "" {
			continue
		}
		cleaned = append(cleaned, t)
	}
	sort.Strings(cleaned)
	h := sha256.New()
	for _, c := range cleaned {
		h.Write([]byte(c))
		h.Write([]byte{'\n'})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// HashLanes returns sha256:<hex> over every populated dep lane in fixed order.
// Projects with only main deps produce the same hash as HashDependencies.
func HashLanes(lanes map[string][]string) string {
	if len(lanes) == 0 {
		return HashDependencies(nil)
	}
	if onlyMain(lanes) {
		return HashDependencies(lanes[LaneMain])
	}

	keys := orderedLaneKeys(lanes)
	h := sha256.New()
	for _, k := range keys {
		cleaned := trimSort(lanes[k])
		if len(cleaned) == 0 {
			continue
		}
		h.Write([]byte(k))
		h.Write([]byte{'\n'})
		for _, c := range cleaned {
			h.Write([]byte(c))
			h.Write([]byte{'\n'})
		}
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func onlyMain(lanes map[string][]string) bool {
	for k, v := range lanes {
		if k == LaneMain {
			continue
		}
		if len(trimSort(v)) > 0 {
			return false
		}
	}
	return true
}

func trimSort(deps []string) []string {
	out := make([]string, 0, len(deps))
	for _, d := range deps {
		t := strings.TrimSpace(d)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func orderedLaneKeys(lanes map[string][]string) []string {
	var (
		main      bool
		dev       bool
		peer      bool
		groupKeys []string
		optKeys   []string
		other     []string
	)
	for k := range lanes {
		switch {
		case k == LaneMain:
			main = true
		case k == LaneDev:
			dev = true
		case k == LanePeer:
			peer = true
		case strings.HasPrefix(k, "group:"):
			groupKeys = append(groupKeys, k)
		case strings.HasPrefix(k, "optional:"):
			optKeys = append(optKeys, k)
		default:
			other = append(other, k)
		}
	}
	sort.Strings(groupKeys)
	sort.Strings(optKeys)
	sort.Strings(other)
	out := make([]string, 0, len(lanes))
	if main {
		out = append(out, LaneMain)
	}
	if dev {
		out = append(out, LaneDev)
	}
	out = append(out, groupKeys...)
	out = append(out, optKeys...)
	out = append(out, other...)
	if peer {
		out = append(out, LanePeer)
	}
	return out
}

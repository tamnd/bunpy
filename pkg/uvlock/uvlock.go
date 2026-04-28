// Package uvlock parses and writes uv.lock — the lockfile format used by
// Astral's uv Python package manager (https://github.com/astral-sh/uv).
//
// The format is TOML. Each resolved package appears as a [[package]] block
// with optional sub-tables [[package.wheels]] and [package.metadata].
// bunpy reads uv.lock when a project ships it instead of (or alongside)
// bunpy.lock, and can write uv.lock via `bunpy pm lock --format uv`.
package uvlock

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// Version is the only uv.lock schema version bunpy understands.
const Version = 1

// UVLock is a parsed uv.lock.
type UVLock struct {
	Version        int
	Revision       int    // 0 means absent
	RequiresPython string // e.g. ">=3.12"
	Options        UVOptions
	// Root is the project's own [[package]] entry (source = { virtual = "." }).
	// When non-nil, Bytes() writes it sorted among Packages by name.
	Root     *UVPackage
	Packages []UVPackage
	// ContentHash is a bunpy extension (ignored by real uv) that stores
	// the manifest lane hash so `pm lock --check` can detect staleness.
	ContentHash string
}

// UVOptions mirrors the [options] table.
type UVOptions struct {
	ExcludeNewer string // RFC 3339 datetime or empty
}

// UVPackage is one [[package]] block.
type UVPackage struct {
	Name         string
	Version      string
	Source       UVSource
	Dependencies []UVDep
	Sdist        *UVFile
	Wheels       []UVFile
	Metadata     UVPackageMeta
	// Groups is a bunpy extension field (ignored by real uv) that encodes
	// lane membership, e.g. ["dev"] for dev-only packages.
	Groups []string
}

// UVSource describes where the package comes from.
type UVSource struct {
	// Kind is one of: registry, git, path, directory, url, editable.
	Kind     string
	URL      string // registry URL, git URL, or direct URL
	Rev      string // git revision (Kind=="git")
	Path     string // local path (Kind=="path", "directory", "editable")
	Editable bool
}

// UVDep is one entry in [[package]].dependencies.
type UVDep struct {
	Name      string
	Specifier string // optional PEP 440 spec, e.g. ">=1.0"
	Marker    string // optional PEP 508 marker expression
	Extra     []string
}

// UVFile is one wheel or sdist entry.
type UVFile struct {
	URL  string
	Hash string // "sha256:<hex>"
	Size int64
}

// Filename returns the basename of the URL.
func (f UVFile) Filename() string { return path.Base(f.URL) }

// UVPackageMeta mirrors [package.metadata].
type UVPackageMeta struct {
	RequiresDist   []UVMetaDep
	RequiresPython string
}

// UVMetaDep is one entry in [package.metadata].requires-dist.
type UVMetaDep struct {
	Name      string
	Specifier string
	Marker    string
	Extra     []string
}

// ─── raw TOML intermediates ───────────────────────────────────────────────────

type rawLock struct {
	Version        int          `toml:"version"`
	Revision       int          `toml:"revision"`
	RequiresPython string       `toml:"requires-python"`
	ContentHash    string       `toml:"content-hash"`
	Options        rawOptions   `toml:"options"`
	Packages       []rawPackage `toml:"package"`
}

type rawOptions struct {
	ExcludeNewer string `toml:"exclude-newer"`
}

type rawPackage struct {
	Name         string         `toml:"name"`
	Version      string         `toml:"version"`
	Source       map[string]any `toml:"source"`
	Dependencies []rawDep       `toml:"dependencies"`
	Sdist        *rawFile       `toml:"sdist"`
	Wheels       []rawFile      `toml:"wheels"`
	Metadata     rawMeta        `toml:"metadata"`
	Groups       []string       `toml:"groups"`
}

type rawDep struct {
	Name      string `toml:"name"`
	Specifier string `toml:"specifier"`
	Marker    string `toml:"marker"`
	Extra     []string `toml:"extra"`
}

type rawFile struct {
	URL  string `toml:"url"`
	Hash string `toml:"hash"`
	Size int64  `toml:"size"`
}

type rawMeta struct {
	RequiresDist   []rawMetaDep `toml:"requires-dist"`
	RequiresPython string       `toml:"requires-python"`
}

type rawMetaDep struct {
	Name      string   `toml:"name"`
	Specifier string   `toml:"specifier"`
	Marker    string   `toml:"marker"`
	Extra     []string `toml:"extra"`
}

// ─── Parse ────────────────────────────────────────────────────────────────────

// Parse reads a uv.lock file and returns the structured lock.
func Parse(data []byte) (*UVLock, error) {
	var raw rawLock
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, fmt.Errorf("uvlock: parse: %w", err)
	}
	if raw.Version != Version {
		return nil, fmt.Errorf("uvlock: unsupported version %d (want %d)", raw.Version, Version)
	}

	lock := &UVLock{
		Version:        raw.Version,
		Revision:       raw.Revision,
		RequiresPython: raw.RequiresPython,
		ContentHash:    raw.ContentHash,
		Options: UVOptions{
			ExcludeNewer: raw.Options.ExcludeNewer,
		},
	}

	for _, rp := range raw.Packages {
		pkg := UVPackage{
			Name:    rp.Name,
			Version: rp.Version,
			Source:  parseSource(rp.Source),
		}

		for _, rd := range rp.Dependencies {
			pkg.Dependencies = append(pkg.Dependencies, UVDep{
				Name:      rd.Name,
				Specifier: rd.Specifier,
				Marker:    rd.Marker,
				Extra:     rd.Extra,
			})
		}

		if rp.Sdist != nil {
			f := UVFile{URL: rp.Sdist.URL, Hash: rp.Sdist.Hash, Size: rp.Sdist.Size}
			pkg.Sdist = &f
		}

		for _, rw := range rp.Wheels {
			pkg.Wheels = append(pkg.Wheels, UVFile{
				URL:  rw.URL,
				Hash: rw.Hash,
				Size: rw.Size,
			})
		}
		pkg.Groups = rp.Groups

		for _, rd := range rp.Metadata.RequiresDist {
			pkg.Metadata.RequiresDist = append(pkg.Metadata.RequiresDist, UVMetaDep{
				Name:      rd.Name,
				Specifier: rd.Specifier,
				Marker:    rd.Marker,
				Extra:     rd.Extra,
			})
		}
		pkg.Metadata.RequiresPython = rp.Metadata.RequiresPython

		if pkg.Source.Kind == "virtual" {
			lock.Root = &pkg
		} else {
			lock.Packages = append(lock.Packages, pkg)
		}
	}

	return lock, nil
}

func parseSource(m map[string]any) UVSource {
	if m == nil {
		return UVSource{}
	}
	str := func(key string) string {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	switch {
	case str("registry") != "":
		return UVSource{Kind: "registry", URL: str("registry")}
	case str("git") != "":
		return UVSource{Kind: "git", URL: str("git"), Rev: str("rev")}
	case str("path") != "":
		return UVSource{Kind: "path", Path: str("path")}
	case str("directory") != "":
		return UVSource{Kind: "directory", Path: str("directory")}
	case str("url") != "":
		return UVSource{Kind: "url", URL: str("url")}
	case str("editable") != "":
		return UVSource{Kind: "editable", Path: str("editable"), Editable: true}
	case str("virtual") != "":
		return UVSource{Kind: "virtual", Path: str("virtual")}
	default:
		return UVSource{Kind: "registry", URL: "https://pypi.org/simple"}
	}
}

// ─── Bytes ────────────────────────────────────────────────────────────────────

// Bytes serialises the lock to canonical uv.lock TOML.
// Packages are sorted by PEP 503 normalised name.
func (l *UVLock) Bytes() []byte {
	var sb strings.Builder

	sb.WriteString("version = 1\n")
	if l.Revision > 0 {
		fmt.Fprintf(&sb, "revision = %d\n", l.Revision)
	}
	if l.RequiresPython != "" {
		fmt.Fprintf(&sb, "requires-python = %q\n", l.RequiresPython)
	}
	if l.ContentHash != "" {
		fmt.Fprintf(&sb, "content-hash = %q\n", l.ContentHash)
	}
	if l.Options.ExcludeNewer != "" {
		sb.WriteString("\n[options]\n")
		fmt.Fprintf(&sb, "exclude-newer = %q\n", l.Options.ExcludeNewer)
	}

	pkgs := append([]UVPackage(nil), l.Packages...)
	if l.Root != nil {
		pkgs = append(pkgs, *l.Root)
	}
	sort.SliceStable(pkgs, func(i, j int) bool {
		ni := normalize(pkgs[i].Name)
		nj := normalize(pkgs[j].Name)
		if ni != nj {
			return ni < nj
		}
		return pkgs[i].Version > pkgs[j].Version
	})

	for _, pkg := range pkgs {
		sb.WriteString("\n[[package]]\n")
		fmt.Fprintf(&sb, "name = %q\n", pkg.Name)
		fmt.Fprintf(&sb, "version = %q\n", pkg.Version)
		sb.WriteString(writeSource(pkg.Source))
		if len(pkg.Groups) > 0 {
			quoted := make([]string, len(pkg.Groups))
			for i, g := range pkg.Groups {
				quoted[i] = fmt.Sprintf("%q", g)
			}
			fmt.Fprintf(&sb, "groups = [%s]\n", strings.Join(quoted, ", "))
		}

		if len(pkg.Dependencies) > 0 {
			sb.WriteString("dependencies = [\n")
			for _, d := range pkg.Dependencies {
				sb.WriteString("  ")
				sb.WriteString(writeDep(d))
				sb.WriteString(",\n")
			}
			sb.WriteString("]\n")
		}

		if pkg.Sdist != nil {
			fmt.Fprintf(&sb, "sdist = { url = %q, hash = %q, size = %d }\n",
				pkg.Sdist.URL, pkg.Sdist.Hash, pkg.Sdist.Size)
		}

		for _, w := range pkg.Wheels {
			sb.WriteString("\n[[package.wheels]]\n")
			fmt.Fprintf(&sb, "url = %q\n", w.URL)
			fmt.Fprintf(&sb, "hash = %q\n", w.Hash)
			fmt.Fprintf(&sb, "size = %d\n", w.Size)
		}

		if len(pkg.Metadata.RequiresDist) > 0 || pkg.Metadata.RequiresPython != "" {
			sb.WriteString("\n[package.metadata]\n")
			if pkg.Metadata.RequiresPython != "" {
				fmt.Fprintf(&sb, "requires-python = %q\n", pkg.Metadata.RequiresPython)
			}
			if len(pkg.Metadata.RequiresDist) > 0 {
				sb.WriteString("requires-dist = [\n")
				for _, d := range pkg.Metadata.RequiresDist {
					sb.WriteString("  ")
					sb.WriteString(writeMetaDep(d))
					sb.WriteString(",\n")
				}
				sb.WriteString("]\n")
			}
		}
	}

	return []byte(sb.String())
}

func writeSource(s UVSource) string {
	switch s.Kind {
	case "registry":
		return fmt.Sprintf("source = { registry = %q }\n", s.URL)
	case "git":
		if s.Rev != "" {
			return fmt.Sprintf("source = { git = %q, rev = %q }\n", s.URL, s.Rev)
		}
		return fmt.Sprintf("source = { git = %q }\n", s.URL)
	case "path":
		return fmt.Sprintf("source = { path = %q }\n", s.Path)
	case "directory":
		return fmt.Sprintf("source = { directory = %q }\n", s.Path)
	case "url":
		return fmt.Sprintf("source = { url = %q }\n", s.URL)
	case "editable":
		return fmt.Sprintf("source = { editable = %q }\n", s.Path)
	case "virtual":
		return fmt.Sprintf("source = { virtual = %q }\n", s.Path)
	default:
		return fmt.Sprintf("source = { registry = %q }\n", "https://pypi.org/simple")
	}
}

func writeDep(d UVDep) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("name = %q", d.Name))
	if d.Specifier != "" {
		parts = append(parts, fmt.Sprintf("specifier = %q", d.Specifier))
	}
	if d.Marker != "" {
		parts = append(parts, fmt.Sprintf("marker = %q", d.Marker))
	}
	if len(d.Extra) > 0 {
		extras := make([]string, len(d.Extra))
		for i, e := range d.Extra {
			extras[i] = fmt.Sprintf("%q", e)
		}
		parts = append(parts, "extra = ["+strings.Join(extras, ", ")+"]")
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func writeMetaDep(d UVMetaDep) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("name = %q", d.Name))
	if d.Specifier != "" {
		parts = append(parts, fmt.Sprintf("specifier = %q", d.Specifier))
	}
	if d.Marker != "" {
		parts = append(parts, fmt.Sprintf("marker = %q", d.Marker))
	}
	if len(d.Extra) > 0 {
		extras := make([]string, len(d.Extra))
		for i, e := range d.Extra {
			extras[i] = fmt.Sprintf("%q", e)
		}
		parts = append(parts, "extra = ["+strings.Join(extras, ", ")+"]")
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// normalize returns the PEP 503 normalised package name.
func normalize(s string) string {
	s = strings.ToLower(s)
	var b bytes.Buffer
	prev := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == '_' || c == '.' {
			if prev == '-' {
				continue
			}
			b.WriteByte('-')
			prev = '-'
			continue
		}
		b.WriteByte(c)
		prev = c
	}
	return b.String()
}

// BestWheel returns the most compatible wheel for the current host platform
// using the same PEP 425 tag matching that bunpy uses during install.
// Returns nil if no wheels are available.
func (p *UVPackage) BestWheel() *UVFile {
	return p.BestWheelFor(wheel.HostTags())
}

// BestWheelFor returns the most compatible wheel for the given tag set.
// It uses PEP 425 tag matching, preferring universal wheels first, then
// the best platform-specific match, then the first wheel as a last resort.
func (p *UVPackage) BestWheelFor(tags []wheel.Tag) *UVFile {
	if len(p.Wheels) == 0 {
		return nil
	}
	// Use the same tag-rank matching that wheel.Pick uses.
	bestRank := -1
	bestIdx := -1
	for i, w := range p.Wheels {
		rank := wheelTagRank(path.Base(w.URL), tags)
		if rank >= 0 && (bestRank < 0 || rank < bestRank) {
			bestRank = rank
			bestIdx = i
		}
	}
	if bestIdx >= 0 {
		return &p.Wheels[bestIdx]
	}
	// Fallback: first wheel (e.g., sdist-only packages written by tests)
	return &p.Wheels[0]
}

// wheelTagRank returns the lowest tag rank (index in tags) that matches
// filename, or -1 if the filename is not a compatible wheel.
func wheelTagRank(filename string, tags []wheel.Tag) int {
	base := strings.TrimSuffix(filename, ".whl")
	if base == filename {
		return -1 // not a wheel
	}
	parts := strings.Split(base, "-")
	if len(parts) < 5 {
		return -1
	}
	py := parts[len(parts)-3]
	abi := parts[len(parts)-2]
	plat := parts[len(parts)-1]
	pys := strings.Split(py, ".")
	abis := strings.Split(abi, ".")
	plats := strings.Split(plat, ".")
	best := -1
	for _, p := range pys {
		for _, a := range abis {
			for _, pl := range plats {
				for i, t := range tags {
					if t.Python == p && t.ABI == a && t.Platform == pl {
						if best < 0 || i < best {
							best = i
						}
					}
				}
			}
		}
	}
	return best
}

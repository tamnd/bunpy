package uvlock

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/lockfile"
)

// ToBunpyLock converts a UVLock into a bunpy lockfile.Lock.
// Only registry-sourced packages with at least one wheel are included.
// Groups (bunpy extension) are mapped back to Lanes.
func ToBunpyLock(uv *UVLock) *lockfile.Lock {
	l := &lockfile.Lock{
		Version: lockfile.Version,
	}
	for i := range uv.Packages {
		pkg := &uv.Packages[i]
		if pkg.Source.Kind != "" && pkg.Source.Kind != "registry" {
			// skip git/path/editable — not wheel-installable via bunpy yet
			continue
		}
		w := pkg.BestWheel()
		if w == nil {
			continue
		}
		lanes := groupsToLanes(pkg.Groups)
		l.Packages = append(l.Packages, lockfile.Package{
			Name:     pkg.Name,
			Version:  pkg.Version,
			Filename: path.Base(w.URL),
			URL:      w.URL,
			Hash:     w.Hash,
			Lanes:    lanes,
		})
	}
	return l
}

// groupsToLanes converts uv.lock groups extension to lockfile lanes.
// Empty groups means main lane (implicit). Non-empty groups are the
// exact lane labels, possibly including "main" for mixed packages.
func groupsToLanes(groups []string) []string {
	if len(groups) == 0 {
		return nil // nil = main lane by convention
	}
	lanes := make([]string, len(groups))
	copy(lanes, groups)
	return lanes
}

// lanesToGroups converts lockfile lanes to uv.lock groups extension.
// Main-only packages return nil (no groups written). When a package
// belongs to main AND other lanes, "main" is written explicitly so
// the roundtrip is lossless.
func lanesToGroups(lanes []string) []string {
	if len(lanes) == 0 {
		return nil
	}
	if len(lanes) == 1 && lanes[0] == lockfile.LaneMain {
		return nil // absence of groups means main-only
	}
	// mixed or non-main: store all labels verbatim
	groups := make([]string, len(lanes))
	copy(groups, lanes)
	return groups
}

// ReadLockfile reads a uv.lock file from path and returns a lockfile.Lock
// with lane information decoded from the bunpy groups extension and the
// content-hash bunpy extension preserved.
func ReadLockfile(path string) (*lockfile.Lock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, lockfile.ErrNotFound
		}
		return nil, fmt.Errorf("uvlock: read %s: %w", path, err)
	}
	uv, err := Parse(data)
	if err != nil {
		return nil, err
	}
	l := ToBunpyLock(uv)
	l.ContentHash = uv.ContentHash
	return l, nil
}

// RootInfo carries the project's own pyproject.toml fields needed to
// write the root [[package]] entry (source = { virtual = "." }).
type RootInfo struct {
	Name    string
	Version string
	// Deps is the raw dependency specifier strings from pyproject.toml,
	// e.g. ["click>=8", "requests"]. They are written verbatim into
	// [package.metadata].requires-dist.
	Deps []string
}

// WriteLockfile converts l to uv.lock format and writes it to path,
// preserving lane information in the bunpy groups extension and the
// content-hash as a bunpy extension field. When root is non-nil a
// virtual root [[package]] entry is written for uv compatibility.
// ReadNonRegistryPackages returns the git/path/editable packages from a
// uv.lock that ToBunpyLock would otherwise drop. Pass these to WriteLockfile
// as extraPkgs so they survive a pm lock round-trip.
func ReadNonRegistryPackages(lockPath string) ([]UVPackage, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("uvlock: read %s: %w", lockPath, err)
	}
	uv, err := Parse(data)
	if err != nil {
		return nil, err
	}
	var out []UVPackage
	for i := range uv.Packages {
		pkg := &uv.Packages[i]
		if pkg.Source.Kind != "" && pkg.Source.Kind != "registry" {
			out = append(out, *pkg)
		}
	}
	return out, nil
}

// WriteOptions carries optional parameters for WriteLockfile.
type WriteOptions struct {
	// Graph maps normalised package name → dep names; nil omits dep edges.
	Graph map[string][]string
	// DepExtras maps pkg → dep → extras list for dep edge extras field.
	DepExtras map[string]map[string][]string
	// ExtraPackages carries non-registry (git/path) packages to preserve.
	ExtraPackages []UVPackage
	// Root, when non-nil, adds the virtual root [[package]] entry.
	Root *RootInfo
}

// WriteLockfile converts l to uv.lock format and writes it to path.
func WriteLockfile(path string, l *lockfile.Lock, requiresPython string, opts WriteOptions) error {
	if requiresPython == "" {
		requiresPython = ">=3.12"
	}
	uv := FromBunpyLock(l, requiresPython, opts.Graph, opts.DepExtras, opts.ExtraPackages)
	uv.ContentHash = l.ContentHash
	// Re-apply groups from lanes (FromBunpyLock doesn't know about lanes).
	pkgLanes := make(map[string][]string, len(l.Packages))
	for _, p := range l.Packages {
		pkgLanes[normalize(p.Name)] = p.Lanes
	}
	for i := range uv.Packages {
		uv.Packages[i].Groups = lanesToGroups(pkgLanes[normalize(uv.Packages[i].Name)])
	}
	if opts.Root != nil {
		uv.Root = buildRoot(opts.Root, uv.Packages)
	}
	return os.WriteFile(path, uv.Bytes(), 0o644)
}

// buildRoot constructs the virtual root UVPackage from RootInfo.
func buildRoot(r *RootInfo, _ []UVPackage) *UVPackage {
	root := &UVPackage{
		Name:    r.Name,
		Version: r.Version,
		Source:  UVSource{Kind: "virtual", Path: "."},
	}
	// Direct dep names: names mentioned in pyproject.toml dependencies.
	// We write them as { name = "..." } edges (no version pinning in root deps).
	for _, depSpec := range r.Deps {
		name := depSpecName(depSpec)
		if name != "" {
			root.Dependencies = append(root.Dependencies, UVDep{Name: normalize(name)})
		}
	}
	// [package.metadata].requires-dist: original specifier strings
	for _, depSpec := range r.Deps {
		name := depSpecName(depSpec)
		specifier := depSpecVersion(depSpec)
		if name != "" {
			root.Metadata.RequiresDist = append(root.Metadata.RequiresDist, UVMetaDep{
				Name:      name,
				Specifier: specifier,
			})
		}
	}
	return root
}

// depSpecName extracts the package name from a PEP 508 dependency string
// like "click>=8.0" → "click" or "requests[security]>=2.0" → "requests".
func depSpecName(spec string) string {
	spec = strings.TrimSpace(spec)
	for i, r := range spec {
		if r == '>' || r == '<' || r == '=' || r == '!' || r == '~' || r == '[' || r == ';' || r == ' ' {
			return strings.TrimSpace(spec[:i])
		}
	}
	return spec
}

// depSpecVersion extracts the version specifier part of a PEP 508 string
// like "click>=8.0" → ">=8.0". Returns "" if none.
func depSpecVersion(spec string) string {
	spec = strings.TrimSpace(spec)
	// Skip extras [...]
	if i := strings.IndexByte(spec, '['); i >= 0 {
		if j := strings.IndexByte(spec[i:], ']'); j >= 0 {
			spec = spec[:i] + spec[i+j+1:]
		}
	}
	for i, r := range spec {
		if r == '>' || r == '<' || r == '=' || r == '!' || r == '~' {
			return strings.TrimSpace(spec[i:])
		}
	}
	return ""
}

// FromBunpyLock converts a bunpy lockfile.Lock into a UVLock.
// graph maps PEP 503-normalised package name → list of dep names;
// it populates the [[package]].dependencies field. Pass nil for an
// empty graph (no dep edges recorded).
// Lane information is NOT transferred here; callers should set Groups
// on the resulting UVPackage entries via lanesToGroups.
func FromBunpyLock(l *lockfile.Lock, requiresPython string, graph map[string][]string, depExtras map[string]map[string][]string, extraPkgs []UVPackage) *UVLock {
	if requiresPython == "" {
		requiresPython = ">=3.12"
	}
	uv := &UVLock{
		Version:        Version,
		RequiresPython: requiresPython,
	}
	for _, p := range l.Packages {
		pkg := UVPackage{
			Name:    p.Name,
			Version: p.Version,
			Source: UVSource{
				Kind: "registry",
				URL:  "https://pypi.org/simple",
			},
		}

		// dependency edges
		if graph != nil {
			norm := normalize(p.Name)
			pkgExtras := depExtras[norm]
			for _, dep := range graph[norm] {
				d := UVDep{Name: dep}
				if pkgExtras != nil {
					d.Extra = pkgExtras[dep]
				}
				pkg.Dependencies = append(pkg.Dependencies, d)
			}
		}

		// wheel entry
		pkg.Wheels = append(pkg.Wheels, UVFile{
			URL:  p.URL,
			Hash: p.Hash,
			Size: p.Size,
		})

		// sdist entry
		if p.SdistURL != "" {
			pkg.Sdist = &UVFile{
				URL:  p.SdistURL,
				Hash: p.SdistHash,
				Size: p.SdistSize,
			}
		}

		uv.Packages = append(uv.Packages, pkg)
	}
	// Append non-registry packages (git/path/editable) preserved from prior lock.
	uv.Packages = append(uv.Packages, extraPkgs...)
	return uv
}

// DetectFormat returns "uv", "bunpy", or "none" depending on which lockfile
// exists in dir. uv.lock takes precedence if both exist.
func DetectFormat(dir string) string {
	if fileExists(filepath.Join(dir, "uv.lock")) {
		return "uv"
	}
	if fileExists(filepath.Join(dir, "bunpy.lock")) {
		return "bunpy"
	}
	return "none"
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

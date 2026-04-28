package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/tamnd/bunpy/v1/pkg/marker"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/resolver"
	"github.com/tamnd/bunpy/v1/pkg/version"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
)

// pypiRegistry adapts pkg/pypi + pkg/wheel + pkg/marker into the
// resolver.Registry interface. Versions are filtered to those with
// at least one wheel matching the host tag set; dependencies come
// from each chosen wheel's METADATA via PEP 658 (or wheel body
// fallback). Markers are evaluated against env so platform-only
// edges drop out before the resolver sees them.
type pypiRegistry struct {
	ctx       context.Context
	client    *pypi.Client
	tags      []wheel.Tag
	env       marker.Env
	fetchBody func(file pypi.File) ([]byte, error)

	tagsKey  string                          // fingerprint of tags set, for picks cache
	projects map[string]*pypi.Project       // name -> project page
	picks    map[string]map[string]pypi.File // name -> ver -> file
	// depsCache stores Dependencies() results so each (pkg, ver) pair is
	// fetched exactly once across both Solve() and laneClosure() (RC-2).
	// Protected by mu for concurrent prefetch goroutines (RC-4).
	mu        sync.Mutex
	depsCache map[string]map[string][]resolver.Requirement
	// depGraph stores the resolved dep edges for reuse by laneClosure (RC-3).
	depGraph map[string][]string // normalised name -> dep names
	// depExtras stores extras per (pkg, dep) edge for uv.lock dep edge writing.
	depExtras map[string]map[string][]string // normalised pkg -> dep -> extras

	// RC-4: prefetch semaphore — at most 4 concurrent metadata fetches.
	prefetchSem chan struct{}
	prefetchWg  sync.WaitGroup
}

func newPypiRegistry(ctx context.Context, client *pypi.Client, tags []wheel.Tag, env marker.Env, fetchBody func(pypi.File) ([]byte, error)) *pypiRegistry {
	r := &pypiRegistry{
		ctx: ctx, client: client, tags: tags, env: env, fetchBody: fetchBody,
		tagsKey:     tagsFingerprint(tags),
		projects:    map[string]*pypi.Project{},
		picks:       map[string]map[string]pypi.File{},
		depsCache:   map[string]map[string][]resolver.Requirement{},
		depGraph:    map[string][]string{},
		depExtras:   map[string]map[string][]string{},
		prefetchSem: make(chan struct{}, 4),
	}
	return r
}

// tagsFingerprint returns a short hex key that uniquely identifies the tag set.
// Used as part of the disk picks-cache filename so different platforms don't
// share stale picks.
func tagsFingerprint(tags []wheel.Tag) string {
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = t.Python + "-" + t.ABI + "-" + t.Platform
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, ",")))
	return fmt.Sprintf("%x", sum[:4]) // 8 hex chars is plenty
}

// prefetch fires a background goroutine to warm the depsCache for pkg@ver.
// The resolver calls this speculatively after Versions(); by the time it
// calls Dependencies() the result is often already in cache (RC-4).
func (r *pypiRegistry) prefetch(pkg, ver string) {
	// Skip if already cached.
	r.mu.Lock()
	cached := false
	if m, ok := r.depsCache[pkg]; ok {
		if _, ok := m[ver]; ok {
			cached = true
		}
	}
	r.mu.Unlock()
	if cached {
		return
	}
	r.prefetchWg.Add(1)
	go func() {
		defer r.prefetchWg.Done()
		r.prefetchSem <- struct{}{} // acquire slot
		defer func() { <-r.prefetchSem }()
		_, _ = r.Dependencies(pkg, ver) // result lands in depsCache
	}()
}

func (r *pypiRegistry) project(pkg string) (*pypi.Project, error) {
	if p, ok := r.projects[pkg]; ok {
		return p, nil
	}
	p, err := r.client.Get(r.ctx, pkg)
	if err != nil {
		return nil, err
	}
	r.projects[pkg] = p
	return p, nil
}

// Versions returns versions whose best matching wheel is compatible
// with the host tag set. Sorted ascending so version.Highest still
// works deterministically when multiple candidates tie.
func (r *pypiRegistry) Versions(pkg string) ([]string, error) {
	p, err := r.project(pkg)
	if err != nil {
		return nil, err
	}
	picks := map[string]pypi.File{}
	byVersion := map[string][]pypi.File{}
	for _, f := range p.Files {
		if f.Kind != "wheel" || f.Yanked || f.Version == "" {
			continue
		}
		byVersion[f.Version] = append(byVersion[f.Version], f)
	}
	var versions []string
	for ver, files := range byVersion {
		f, ok := wheel.Pick(files, r.tags)
		if !ok {
			continue
		}
		picks[ver] = f
		versions = append(versions, ver)
	}
	sort.Slice(versions, func(i, j int) bool {
		return version.Compare(versions[i], versions[j]) < 0
	})
	r.picks[pkg] = picks
	// RC-4: speculatively prefetch metadata for the highest version — the
	// resolver almost always picks it, so this overlaps the next Versions()
	// call with the current package's metadata fetch.
	if len(versions) > 0 {
		r.prefetch(pkg, versions[len(versions)-1])
	}
	return versions, nil
}

// Sdist returns the first sdist for pkg@ver, if any.
func (r *pypiRegistry) Sdist(pkg, ver string) (pypi.File, bool) {
	p, err := r.project(pkg)
	if err != nil {
		return pypi.File{}, false
	}
	for _, f := range p.Files {
		if f.Kind == "sdist" && f.Version == ver && !f.Yanked {
			return f, true
		}
	}
	return pypi.File{}, false
}

// Pick returns the wheel chosen for pkg@ver. The resolver only asks
// for Dependencies after picking, so this is filled by Versions.
func (r *pypiRegistry) Pick(pkg, ver string) (pypi.File, bool) {
	if m, ok := r.picks[pkg]; ok {
		f, ok := m[ver]
		return f, ok
	}
	return pypi.File{}, false
}

// Dependencies pulls METADATA for the picked wheel, parses
// Requires-Dist, and evaluates each marker against r.env. Edges
// gated on optional extras are dropped (extra is empty in v0.1.5).
// Results are cached in-process so each (pkg, ver) pair is fetched
// at most once across Solve() and laneClosure() calls (RC-2).
// The cache is mutex-protected for concurrent prefetch goroutines (RC-4).
func (r *pypiRegistry) Dependencies(pkg, ver string) ([]resolver.Requirement, error) {
	// RC-2: return cached result if already fetched.
	r.mu.Lock()
	if m, ok := r.depsCache[pkg]; ok {
		if deps, ok := m[ver]; ok {
			r.mu.Unlock()
			return deps, nil
		}
	}
	r.mu.Unlock()

	f, ok := r.Pick(pkg, ver)
	if !ok {
		return nil, fmt.Errorf("no wheel picked for %s@%s", pkg, ver)
	}
	body, err := r.client.FetchMetadata(r.ctx, f,
		func() ([]byte, error) { return r.fetchBody(f) },
		func(b []byte) ([]byte, error) { return wheel.LoadMetadata(b) },
	)
	if err != nil {
		return nil, err
	}
	md, err := wheel.ParseMetadata(body)
	if err != nil {
		return nil, err
	}
	var out []resolver.Requirement
	var depNames []string
	extras := map[string][]string{}
	for _, rd := range md.RequiresDist {
		if rd.Marker != "" {
			expr, err := marker.Parse(rd.Marker)
			if err != nil {
				return nil, fmt.Errorf("%s@%s marker %q: %w", pkg, ver, rd.Marker, err)
			}
			if !expr.Eval(r.env) {
				continue
			}
		}
		spec, err := version.ParseSpec(rd.Spec)
		if err != nil {
			return nil, fmt.Errorf("%s@%s requires %s: %w", pkg, ver, rd.Raw, err)
		}
		norm := pypi.Normalize(rd.Name)
		out = append(out, resolver.Requirement{Name: norm, Spec: spec})
		depNames = append(depNames, norm)
		if len(rd.Extras) > 0 {
			if extras[norm] == nil {
				extras[norm] = make([]string, 0, len(rd.Extras))
			}
			extras[norm] = append(extras[norm], rd.Extras...)
		}
	}

	// RC-2 + RC-3: store results under mutex.
	r.mu.Lock()
	if r.depsCache[pkg] == nil {
		r.depsCache[pkg] = map[string][]resolver.Requirement{}
	}
	r.depsCache[pkg][ver] = out
	r.depGraph[pkg] = depNames
	if len(extras) > 0 {
		r.depExtras[pkg] = extras
	}
	r.mu.Unlock()

	return out, nil
}

package main

import (
	"context"
	"fmt"
	"sort"

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

	projects map[string]*pypi.Project // name -> project page
	picks    map[string]map[string]pypi.File
}

func newPypiRegistry(ctx context.Context, client *pypi.Client, tags []wheel.Tag, env marker.Env, fetchBody func(pypi.File) ([]byte, error)) *pypiRegistry {
	return &pypiRegistry{
		ctx: ctx, client: client, tags: tags, env: env, fetchBody: fetchBody,
		projects: map[string]*pypi.Project{},
		picks:    map[string]map[string]pypi.File{},
	}
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
	return versions, nil
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
func (r *pypiRegistry) Dependencies(pkg, ver string) ([]resolver.Requirement, error) {
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
		out = append(out, resolver.Requirement{Name: pypi.Normalize(rd.Name), Spec: spec})
	}
	return out, nil
}

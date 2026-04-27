package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/cache"
	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/marker"
	"github.com/tamnd/bunpy/v1/pkg/wheel"
	"github.com/tamnd/bunpy/v1/pkg/why"
)

// whySubcommand wires `bunpy why <pkg>`. The graph comes from the
// lockfile plus per-pin Requires-Dist read out of the wheel cache;
// markers are evaluated against the host environment so the
// reverse-deps surface what would actually install on this box.
func whySubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		depth    int
		jsonOut  bool
		topOnly  bool
		laneArg  string
		cacheDir string
		manPath  = "pyproject.toml"
		lockPath = "bunpy.lock"
		pkg      string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("why", stdout, stderr)
		case "--json":
			jsonOut = true
		case "--top":
			topOnly = true
		case "--depth":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy why: --depth requires a value")
			}
			i++
			n, err := parsePositiveInt(args[i])
			if err != nil {
				return 1, fmt.Errorf("bunpy why: --depth: %w", err)
			}
			depth = n
		case "--lane":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy why: --lane requires a value")
			}
			i++
			laneArg = args[i]
		case "--cache-dir":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy why: --cache-dir requires a value")
			}
			i++
			cacheDir = args[i]
		case "--manifest":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy why: --manifest requires a value")
			}
			i++
			manPath = args[i]
		case "--lockfile":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy why: --lockfile requires a value")
			}
			i++
			lockPath = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--depth="); ok {
				n, err := parsePositiveInt(v)
				if err != nil {
					return 1, fmt.Errorf("bunpy why: --depth: %w", err)
				}
				depth = n
				continue
			}
			if v, ok := strings.CutPrefix(a, "--lane="); ok {
				laneArg = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--cache-dir="); ok {
				cacheDir = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--manifest="); ok {
				manPath = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--lockfile="); ok {
				lockPath = v
				continue
			}
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy why: unknown flag %q", a)
			}
			if pkg != "" {
				return 1, fmt.Errorf("bunpy why: only one package per invocation, got %q and %q", pkg, a)
			}
			pkg = a
		}
	}

	if pkg == "" {
		return 1, errors.New("bunpy why: package name required")
	}

	mf, err := manifest.LoadOpts(manPath, manifest.LoadOptions{})
	if err != nil {
		return 1, fmt.Errorf("bunpy why: read manifest: %w", err)
	}
	lf, err := lockfile.Read(lockPath)
	if err != nil {
		return 1, fmt.Errorf("bunpy why: read lockfile: %w", err)
	}

	fetch, err := newCacheRequiresFunc(cacheDir, lf)
	if err != nil {
		return 1, err
	}
	g, err := why.BuildGraph(lf, mf, fetch)
	if err != nil {
		return 1, fmt.Errorf("bunpy why: %w", err)
	}
	res, err := why.Walk(g, pkg, depth)
	if err != nil {
		if errors.Is(err, why.ErrNotFound) {
			return 1, fmt.Errorf("bunpy why: %s is not in %s (run `bunpy install` or check the spelling)", pkg, lockPath)
		}
		return 1, fmt.Errorf("bunpy why: %w", err)
	}
	res.Chains = filterChainsByLane(res.Chains, laneArg)

	switch {
	case jsonOut:
		return printWhyJSON(stdout, res)
	case topOnly:
		return printWhyTop(stdout, res)
	default:
		return printWhyTree(stdout, res)
	}
}

// newCacheRequiresFunc binds a why.RequiresFunc to the wheel cache.
// The lockfile is closed-over so the closure can resolve a pin name
// to a wheel filename. A cache miss returns no edges (the pin's
// reverse-deps will be invisible until the wheel is fetched).
func newCacheRequiresFunc(cacheDir string, lf *lockfile.Lock) (why.RequiresFunc, error) {
	root := cacheDir
	if root == "" {
		root = cache.DefaultDir()
	}
	wc, err := cache.NewWheelCache(filepath.Join(root, "wheels"))
	if err != nil {
		return nil, fmt.Errorf("bunpy why: open cache: %w", err)
	}
	env := marker.DefaultEnv()
	pins := map[string]lockfile.Package{}
	for _, p := range lf.Packages {
		pins[lockfile.Normalize(p.Name)] = p
	}
	return func(name, _ string) ([]why.Requires, error) {
		pin, ok := pins[lockfile.Normalize(name)]
		if !ok || pin.Filename == "" {
			return nil, nil
		}
		body, err := os.ReadFile(wc.Path(name, pin.Filename))
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		md, err := wheel.LoadMetadata(body)
		if err != nil {
			return nil, err
		}
		parsed, err := wheel.ParseMetadata(md)
		if err != nil {
			return nil, err
		}
		var out []why.Requires
		for _, rd := range parsed.RequiresDist {
			if rd.Marker != "" {
				expr, err := marker.Parse(rd.Marker)
				if err != nil {
					return nil, fmt.Errorf("%s marker %q: %w", name, rd.Marker, err)
				}
				if !expr.Eval(env) {
					continue
				}
			}
			if len(rd.Extras) > 0 {
				continue
			}
			out = append(out, why.Requires{Name: rd.Name, Specifier: rd.Spec})
		}
		return out, nil
	}, nil
}

// filterChainsByLane drops chains whose terminal lane does not match
// laneArg. Empty laneArg means "all lanes". Orphan chains (no lane
// at all) are kept only when laneArg is empty.
func filterChainsByLane(chains []why.Chain, laneArg string) []why.Chain {
	if laneArg == "" {
		return chains
	}
	out := make([]why.Chain, 0, len(chains))
	for _, c := range chains {
		if c.Lane == laneArg {
			out = append(out, c)
		}
	}
	return out
}

// printWhyTree prints the human-readable indent shape.
func printWhyTree(w io.Writer, r *why.Result) (int, error) {
	header := fmt.Sprintf("%s %s", r.Name, r.Version)
	if r.Linked {
		header += " (linked)"
	}
	if r.Patched {
		header += " (patched)"
	}
	fmt.Fprintln(w, header)
	if len(r.Chains) == 0 {
		fmt.Fprintln(w, "  no reverse dependencies (orphan in lockfile)")
		return 0, nil
	}
	for _, c := range r.Chains {
		// Edges[0] is the leaf; emit edges from the first parent up.
		for i := 1; i < len(c.Edges); i++ {
			indent := strings.Repeat("  ", i)
			e := c.Edges[i]
			label := edgeLabel(e, c.Lane)
			fmt.Fprintf(w, "%svia %s\n", indent, label)
		}
	}
	return 0, nil
}

// printWhyTop prints the deduped names of the direct project
// requirements that pull r.Name in.
func printWhyTop(w io.Writer, r *why.Result) (int, error) {
	seen := map[string]bool{}
	var names []string
	for _, c := range r.Chains {
		if len(c.Edges) < 2 {
			continue
		}
		// The direct-req is the second-to-last edge (the @project
		// edge is last). For a leaf that is itself a direct-req
		// the chain has [leaf, @project] and we want the leaf name.
		direct := c.Edges[len(c.Edges)-2]
		if direct.Name == why.ProjectName {
			continue
		}
		if !seen[direct.Name] {
			seen[direct.Name] = true
			names = append(names, direct.Name)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintln(w, n)
	}
	return 0, nil
}

// printWhyJSON marshals the result.
func printWhyJSON(w io.Writer, r *why.Result) (int, error) {
	type edgeJSON struct {
		Name      string `json:"name"`
		Version   string `json:"version,omitempty"`
		Specifier string `json:"specifier,omitempty"`
	}
	type chainJSON struct {
		Lane  string     `json:"lane"`
		Edges []edgeJSON `json:"edges"`
	}
	type resJSON struct {
		Package   string      `json:"package"`
		Version   string      `json:"version"`
		Installer string      `json:"installer"`
		Linked    bool        `json:"linked"`
		Patched   bool        `json:"patched"`
		Chains    []chainJSON `json:"chains"`
	}
	out := resJSON{
		Package:   r.Name,
		Version:   r.Version,
		Installer: r.Installer,
		Linked:    r.Linked,
		Patched:   r.Patched,
	}
	for _, c := range r.Chains {
		jc := chainJSON{Lane: c.Lane}
		for _, e := range c.Edges {
			jc.Edges = append(jc.Edges, edgeJSON{Name: e.Name, Version: e.Version, Specifier: e.Specifier})
		}
		out.Chains = append(out.Chains, jc)
	}
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return 1, err
	}
	fmt.Fprintln(w, string(body))
	return 0, nil
}

// edgeLabel formats one parent edge: `requests 2.32.3 (default)` or
// `project requirement` for the @project sentinel.
func edgeLabel(e why.Edge, lane string) string {
	if e.Name == why.ProjectName {
		l := lane
		if l == "" {
			l = lockfile.LaneMain
		}
		return fmt.Sprintf("project requirement (%s)", l)
	}
	if e.Specifier != "" {
		return fmt.Sprintf("%s %s [%s]", e.Name, e.Version, e.Specifier)
	}
	return fmt.Sprintf("%s %s", e.Name, e.Version)
}

// parsePositiveInt parses a non-negative integer from a flag value.
func parsePositiveInt(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("non-digit %q", c)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}


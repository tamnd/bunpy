package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/tamnd/bunpy/v1/internal/httpkit"
	"github.com/tamnd/bunpy/v1/pkg/lockfile"
	"github.com/tamnd/bunpy/v1/pkg/manifest"
	"github.com/tamnd/bunpy/v1/pkg/pypi"
	"github.com/tamnd/bunpy/v1/pkg/uvlock"
	"github.com/tamnd/bunpy/v1/pkg/version"
)

// outdatedRow is one line of the outdated table. Lanes is the
// pin's lockfile lane set (sorted), unfiltered by the caller's
// lane flags so JSON consumers can inspect the full membership.
type outdatedRow struct {
	Name    string   `json:"name"`
	Current string   `json:"current"`
	Wanted  string   `json:"wanted"`
	Latest  string   `json:"latest"`
	Lanes   []string `json:"lanes"`
}

// outdatedSubcommand wires `bunpy outdated [pkg]...`. v0.1.7 walks
// the lockfile, fetches each project's PEP 691 page through the
// same client `pm info` uses, and prints one row per pin that has
// a newer matching version. Lane flags mirror `bunpy install`.
func outdatedSubcommand(args []string, stdout, stderr io.Writer) (int, error) {
	var (
		baseURL    string
		cacheDir   string
		jsonOut    bool
		dev        bool
		peer       bool
		allExtras  bool
		production bool
		extras     []string
		pkgs       []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			return printHelp("outdated", stdout, stderr)
		case "--json":
			jsonOut = true
		case "-D", "--dev":
			dev = true
		case "-P", "--peer":
			peer = true
		case "--all-extras":
			allExtras = true
		case "--production":
			production = true
		case "-O", "--optional":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy outdated: %s requires a group name", a)
			}
			i++
			extras = append(extras, args[i])
		case "--index":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy outdated: --index requires a value")
			}
			i++
			baseURL = args[i]
		case "--cache-dir":
			if i+1 >= len(args) {
				return 1, fmt.Errorf("bunpy outdated: --cache-dir requires a value")
			}
			i++
			cacheDir = args[i]
		default:
			if v, ok := strings.CutPrefix(a, "--index="); ok {
				baseURL = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--cache-dir="); ok {
				cacheDir = v
				continue
			}
			if v, ok := strings.CutPrefix(a, "--optional="); ok {
				extras = append(extras, v)
				continue
			}
			if strings.HasPrefix(a, "-") {
				return 1, fmt.Errorf("bunpy outdated: unknown flag %q", a)
			}
			pkgs = append(pkgs, pypi.Normalize(a))
		}
	}
	_ = cacheDir
	if production && (dev || peer || allExtras || len(extras) > 0) {
		return 1, fmt.Errorf("bunpy outdated: --production cannot combine with --dev/--optional/--all-extras/--peer")
	}

	mf, err := manifest.Load("pyproject.toml")
	if err != nil {
		return 1, fmt.Errorf("bunpy outdated: %w", err)
	}
	lock, err := uvlock.ReadLockfile("uv.lock")
	if err != nil {
		if errors.Is(err, lockfile.ErrNotFound) {
			return 1, fmt.Errorf("bunpy outdated: uv.lock missing - run `bunpy pm lock` first")
		}
		return 1, fmt.Errorf("bunpy outdated: %w", err)
	}

	keep := installLaneFilter(dev, peer, allExtras, extras)
	wantPkg := func(name string) bool {
		if len(pkgs) == 0 {
			return true
		}
		return slices.Contains(pkgs, pypi.Normalize(name))
	}

	specByName := manifestSpecIndex(mf)

	client := pypi.New()
	if baseURL != "" {
		client.BaseURL = baseURL
	}
	if fix := os.Getenv("BUNPY_PYPI_FIXTURES"); fix != "" {
		client.HTTP = httpkit.FixturesFS(fix)
	}
	ctx := context.Background()

	var rows []outdatedRow
	for _, p := range lock.Packages {
		if !keep(p.Lanes) {
			continue
		}
		if !wantPkg(p.Name) {
			continue
		}
		spec := specByName[pypi.Normalize(p.Name)]
		parsed, err := version.ParseSpec(spec)
		if err != nil {
			return 1, fmt.Errorf("bunpy outdated: parse %q: %w", spec, err)
		}
		proj, err := client.Get(ctx, p.Name)
		if err != nil {
			return 1, fmt.Errorf("bunpy outdated: %s: %w", p.Name, err)
		}
		candidates := projectVersions(proj)
		wanted := version.Highest(parsed, candidates)
		latest := version.Highest(version.Spec{}, candidates)
		if wanted == "" {
			wanted = p.Version
		}
		if latest == "" {
			latest = p.Version
		}
		if version.Compare(wanted, p.Version) > 0 || version.Compare(latest, p.Version) > 0 {
			lanes := append([]string(nil), p.Lanes...)
			if len(lanes) == 0 {
				lanes = []string{lockfile.LaneMain}
			}
			rows = append(rows, outdatedRow{
				Name:    p.Name,
				Current: p.Version,
				Wanted:  wanted,
				Latest:  latest,
				Lanes:   lanes,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(map[string]any{"outdated": rows}); err != nil {
			return 1, fmt.Errorf("bunpy outdated: %w", err)
		}
		return 0, nil
	}

	if len(rows) == 0 {
		return 0, nil
	}
	fmt.Fprintf(stdout, "%-20s %-10s %-10s %-10s %s\n", "package", "current", "wanted", "latest", "lanes")
	for _, r := range rows {
		fmt.Fprintf(stdout, "%-20s %-10s %-10s %-10s %s\n", r.Name, r.Current, r.Wanted, r.Latest, strings.Join(r.Lanes, ","))
	}
	return 0, nil
}

// manifestSpecIndex maps PEP 503 normalised package names to the
// raw spec string from any lane in the manifest. When a name
// appears in multiple lanes with different specs (rare), the last
// one wins; the resolver runs the union elsewhere.
func manifestSpecIndex(mf *manifest.Manifest) map[string]string {
	out := map[string]string{}
	collect := func(specs []string) {
		for _, s := range specs {
			name, vSpec := splitNameSpec(s)
			if name == "" {
				continue
			}
			out[pypi.Normalize(name)] = vSpec
		}
	}
	collect(mf.Project.Dependencies)
	for _, deps := range mf.Project.OptionalDeps {
		collect(deps)
	}
	for _, deps := range mf.DependencyGroups {
		collect(deps)
	}
	collect(mf.Tool.PeerDependencies)
	return out
}

// projectVersions returns every non-yanked wheel-bearing version
// from a PEP 691 project page. Sorted ascending so version.Highest
// behaves deterministically when two versions tie.
func projectVersions(p *pypi.Project) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range p.Files {
		if f.Yanked || f.Kind != "wheel" || f.Version == "" {
			continue
		}
		if seen[f.Version] {
			continue
		}
		seen[f.Version] = true
		out = append(out, f.Version)
	}
	sort.Slice(out, func(i, j int) bool { return version.Compare(out[i], out[j]) < 0 })
	return out
}

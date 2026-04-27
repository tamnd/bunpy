// Package manifest parses pyproject.toml into a structured form
// every later v0.1.x rung consumes (resolver, installer, lockfile
// writer). PEP 621 ([project]) is the primary table; [tool.bunpy]
// is preserved verbatim for the rungs that grow it.
//
// The parser is deliberately tolerant on shape: PEP 621 has many
// optional fields and we keep the original TOML in Project.Raw so
// callers can pass through what we have not modelled yet. The only
// hard rejections (in strict mode) are:
//   - [project] table missing
//   - project.name missing or empty
//   - project.name failing the PEP 503 normalised-name regex
//   - dynamic listing a field that is also set literally
package manifest

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/BurntSushi/toml"
)

// LoadOptions tunes Parse / Load.
type LoadOptions struct {
	// Strict turns validation issues into errors. The default
	// (false) collects them on Manifest.Warnings and lets the
	// caller decide.
	Strict bool
}

// Manifest is the parsed pyproject.toml.
type Manifest struct {
	Project          Project             `json:"project"`
	Tool             Tool                `json:"tool"`
	DependencyGroups map[string][]string `json:"dependency_groups,omitempty"`
	Other            map[string]any      `json:"other,omitempty"`

	// Warnings is non-nil only in soft mode.
	Warnings []string `json:"warnings,omitempty"`

	// Source holds the original bytes parsed; AddDependency and
	// other text-based mutators consume it. Populated by Parse and
	// Load; ParseOpts copies the input slice.
	Source []byte `json:"-"`
}

// Project mirrors the PEP 621 [project] table.
type Project struct {
	Name           string                       `json:"name"`
	Version        string                       `json:"version,omitempty"`
	Description    string                       `json:"description,omitempty"`
	RequiresPython string                       `json:"requires_python,omitempty"`
	Dependencies   []string                     `json:"dependencies,omitempty"`
	OptionalDeps   map[string][]string          `json:"optional_dependencies,omitempty"`
	Authors        []Author                     `json:"authors,omitempty"`
	Maintainers    []Author                     `json:"maintainers,omitempty"`
	License        License                      `json:"license,omitzero"`
	Readme         Readme                       `json:"readme,omitzero"`
	Scripts        map[string]string            `json:"scripts,omitempty"`
	GUIScripts     map[string]string            `json:"gui_scripts,omitempty"`
	EntryPoints    map[string]map[string]string `json:"entry_points,omitempty"`
	Keywords       []string                     `json:"keywords,omitempty"`
	Classifiers    []string                     `json:"classifiers,omitempty"`
	URLs           map[string]string            `json:"urls,omitempty"`
	Dynamic        []string                     `json:"dynamic,omitempty"`

	// Raw holds the original [project] table verbatim. Useful for
	// fields we have not modelled yet (and for round-tripping).
	Raw map[string]any `json:"-"`
}

// Author is one entry in [project].authors / .maintainers.
type Author struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// License is the [project].license table or the SPDX shorthand.
type License struct {
	Text string `json:"text,omitempty"`
	File string `json:"file,omitempty"`
	SPDX string `json:"spdx,omitempty"`
}

// Readme is the [project].readme table or filename.
type Readme struct {
	File        string `json:"file,omitempty"`
	Text        string `json:"text,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// Tool holds the [tool.bunpy] table verbatim. Future rungs grow
// typed fields here as we use them; everything else stays in Raw.
type Tool struct {
	// PeerDependencies is the [tool.bunpy.peer-dependencies] list:
	// deps that downstream consumers must provide. Resolved for
	// compat checks but never installed by default.
	PeerDependencies []string `json:"peer_dependencies,omitempty"`

	// Workspace is the [tool.bunpy.workspace] table. Non-nil when
	// this pyproject.toml is a workspace root.
	Workspace *WorkspaceConfig `json:"workspace,omitempty"`

	Raw map[string]any `json:"raw,omitempty"`
}

// WorkspaceConfig is the [tool.bunpy.workspace] table.
type WorkspaceConfig struct {
	// Members holds the raw member path patterns (may include globs).
	Members []string `json:"members"`
}

// nameRE is PEP 503's normalised-name regex.
var nameRE = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)

// groupNameRE is the PEP 685 / PEP 621 group name regex. Group
// names share PEP 503's shape but are typically lowercased; we
// accept either case and let callers normalise.
var groupNameRE = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?$`)

// Load reads a pyproject.toml from disk and parses it.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// LoadOpts is Load with a custom LoadOptions value.
func LoadOpts(path string, opts LoadOptions) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseOpts(data, opts)
}

// Parse parses pyproject.toml bytes in strict mode.
func Parse(data []byte) (*Manifest, error) {
	return ParseOpts(data, LoadOptions{Strict: true})
}

// ParseOpts parses pyproject.toml bytes with the given options.
func ParseOpts(data []byte, opts LoadOptions) (*Manifest, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("manifest: parse toml: %w", err)
	}
	m := &Manifest{
		Other:  map[string]any{},
		Source: append([]byte(nil), data...),
	}
	for k, v := range raw {
		switch k {
		case "project":
			pt, _ := v.(map[string]any)
			if pt == nil {
				return nil, errors.New("manifest: [project] must be a table")
			}
			m.Project = parseProject(pt)
		case "tool":
			tt, _ := v.(map[string]any)
			if tt == nil {
				m.Other["tool"] = v
				continue
			}
			if bunpy, ok := tt["bunpy"].(map[string]any); ok {
				m.Tool.Raw = bunpy
				m.Tool.PeerDependencies = stringSlice(bunpy["peer-dependencies"])
				if ws, ok := bunpy["workspace"].(map[string]any); ok {
					m.Tool.Workspace = &WorkspaceConfig{
						Members: stringSlice(ws["members"]),
					}
				}
			}
			// Preserve the rest of [tool.*] verbatim under Other.
			rest := map[string]any{}
			for kk, vv := range tt {
				if kk == "bunpy" {
					continue
				}
				rest[kk] = vv
			}
			if len(rest) > 0 {
				m.Other["tool"] = rest
			}
		case "dependency-groups":
			gt, _ := v.(map[string]any)
			if gt == nil {
				return nil, errors.New("manifest: [dependency-groups] must be a table")
			}
			m.DependencyGroups = map[string][]string{}
			for name, entries := range gt {
				m.DependencyGroups[name] = stringSlice(entries)
			}
		default:
			m.Other[k] = v
		}
	}

	if err := m.validate(opts); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manifest) validate(opts LoadOptions) error {
	report := func(msg string) error {
		if opts.Strict {
			return errors.New("manifest: " + msg)
		}
		m.Warnings = append(m.Warnings, msg)
		return nil
	}

	if m.Project.Raw == nil {
		return report("[project] table missing")
	}
	if m.Project.Name == "" {
		return report("project.name missing or empty")
	}
	if !nameRE.MatchString(m.Project.Name) {
		if err := report(fmt.Sprintf("project.name %q is not a valid PEP 503 name", m.Project.Name)); err != nil {
			return err
		}
	}

	dyn := map[string]bool{}
	for _, d := range m.Project.Dynamic {
		dyn[d] = true
	}
	for _, k := range []string{"version", "description", "readme", "license", "dependencies"} {
		if !dyn[k] {
			continue
		}
		if _, ok := m.Project.Raw[k]; ok {
			if err := report(fmt.Sprintf("project.dynamic includes %q but %q is also set literally", k, k)); err != nil {
				return err
			}
		}
	}

	for name := range m.Project.OptionalDeps {
		if !groupNameRE.MatchString(name) {
			if err := report(fmt.Sprintf("project.optional-dependencies group %q is not a valid PEP 685 name", name)); err != nil {
				return err
			}
		}
	}
	for name := range m.DependencyGroups {
		if !groupNameRE.MatchString(name) {
			if err := report(fmt.Sprintf("dependency-groups name %q is not a valid PEP 685 name", name)); err != nil {
				return err
			}
		}
		if _, dup := m.Project.OptionalDeps[name]; dup {
			if err := report(fmt.Sprintf("dependency-groups name %q also appears in [project.optional-dependencies]", name)); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseProject(t map[string]any) Project {
	p := Project{Raw: t}
	if v, ok := t["name"].(string); ok {
		p.Name = v
	}
	if v, ok := t["version"].(string); ok {
		p.Version = v
	}
	if v, ok := t["description"].(string); ok {
		p.Description = v
	}
	if v, ok := t["requires-python"].(string); ok {
		p.RequiresPython = v
	}
	p.Dependencies = stringSlice(t["dependencies"])
	p.Keywords = stringSlice(t["keywords"])
	p.Classifiers = stringSlice(t["classifiers"])
	p.Dynamic = stringSlice(t["dynamic"])
	if od, ok := t["optional-dependencies"].(map[string]any); ok {
		p.OptionalDeps = map[string][]string{}
		for k, v := range od {
			p.OptionalDeps[k] = stringSlice(v)
		}
	}
	if u, ok := t["urls"].(map[string]any); ok {
		p.URLs = map[string]string{}
		for k, v := range u {
			if s, ok := v.(string); ok {
				p.URLs[k] = s
			}
		}
	}
	if s, ok := t["scripts"].(map[string]any); ok {
		p.Scripts = strMap(s)
	}
	if s, ok := t["gui-scripts"].(map[string]any); ok {
		p.GUIScripts = strMap(s)
	}
	if ep, ok := t["entry-points"].(map[string]any); ok {
		p.EntryPoints = map[string]map[string]string{}
		for k, v := range ep {
			if mm, ok := v.(map[string]any); ok {
				p.EntryPoints[k] = strMap(mm)
			}
		}
	}
	for _, key := range []string{"authors", "maintainers"} {
		var entries []map[string]any
		switch v := t[key].(type) {
		case []map[string]any:
			entries = v
		case []any:
			for _, e := range v {
				if mm, ok := e.(map[string]any); ok {
					entries = append(entries, mm)
				}
			}
		}
		var out []Author
		for _, mm := range entries {
			a := Author{}
			if s, ok := mm["name"].(string); ok {
				a.Name = s
			}
			if s, ok := mm["email"].(string); ok {
				a.Email = s
			}
			out = append(out, a)
		}
		switch key {
		case "authors":
			p.Authors = out
		case "maintainers":
			p.Maintainers = out
		}
	}
	if lic, ok := t["license"]; ok {
		switch v := lic.(type) {
		case string:
			p.License.SPDX = v
		case map[string]any:
			if s, ok := v["text"].(string); ok {
				p.License.Text = s
			}
			if s, ok := v["file"].(string); ok {
				p.License.File = s
			}
		}
	}
	if rd, ok := t["readme"]; ok {
		switch v := rd.(type) {
		case string:
			p.Readme.File = v
		case map[string]any:
			if s, ok := v["file"].(string); ok {
				p.Readme.File = s
			}
			if s, ok := v["text"].(string); ok {
				p.Readme.Text = s
			}
			if s, ok := v["content-type"].(string); ok {
				p.Readme.ContentType = s
			}
		}
	}
	return p
}

func stringSlice(v any) []string {
	if v == nil {
		return nil
	}
	s, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(s))
	for _, e := range s {
		if str, ok := e.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

func strMap(v any) map[string]string {
	if mm, ok := v.(map[string]any); ok {
		out := map[string]string{}
		for k, val := range mm {
			if s, ok := val.(string); ok {
				out[k] = s
			}
		}
		return out
	}
	return nil
}

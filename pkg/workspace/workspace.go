// Package workspace loads and navigates multi-member workspaces.
// A workspace root is a pyproject.toml that has a
// [tool.bunpy.workspace] table with a members list. Each member is
// a subdirectory that has its own pyproject.toml.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/manifest"
)

// ErrNoWorkspace is returned by FindRoot when no workspace root is
// found in the directory tree.
var ErrNoWorkspace = errors.New("workspace: no workspace root found")

// Workspace is a loaded workspace: the root directory and all members.
type Workspace struct {
	Root    string
	Members []Member
}

// Member is one member of a workspace.
type Member struct {
	Path     string             // absolute path to the member directory
	Name     string             // project.name from the member's pyproject.toml
	Manifest *manifest.Manifest
}

// Load reads the workspace root pyproject.toml at root and resolves
// all member paths. root must be an absolute path. Glob patterns in
// the members list are expanded via filepath.Glob.
func Load(root string) (*Workspace, error) {
	mf, err := manifest.LoadOpts(filepath.Join(root, "pyproject.toml"), manifest.LoadOptions{})
	if err != nil {
		return nil, fmt.Errorf("workspace: load root manifest: %w", err)
	}
	if mf.Tool.Workspace == nil {
		return nil, fmt.Errorf("workspace: %s has no [tool.bunpy.workspace] table", root)
	}
	patterns := mf.Tool.Workspace.Members
	if len(patterns) == 0 {
		return nil, fmt.Errorf("workspace: members list is empty in %s", root)
	}

	ws := &Workspace{Root: root}
	namesSeen := map[string]bool{}

	for _, pat := range patterns {
		var paths []string
		if isGlob(pat) {
			abs := filepath.Join(root, pat)
			matches, err := filepath.Glob(abs)
			if err != nil {
				return nil, fmt.Errorf("workspace: glob %q: %w", pat, err)
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("workspace: glob %q matched no directories", pat)
			}
			paths = matches
		} else {
			paths = []string{filepath.Join(root, pat)}
		}

		for _, p := range paths {
			abs, err := filepath.Abs(p)
			if err != nil {
				return nil, fmt.Errorf("workspace: %w", err)
			}
			if !strings.HasPrefix(abs+string(filepath.Separator), root+string(filepath.Separator)) {
				return nil, fmt.Errorf("workspace: member %q is outside the workspace root", p)
			}
			memberMF, err := manifest.LoadOpts(filepath.Join(abs, "pyproject.toml"), manifest.LoadOptions{})
			if err != nil {
				return nil, fmt.Errorf("workspace: load member %s: %w", abs, err)
			}
			name := memberMF.Project.Name
			if namesSeen[name] {
				return nil, fmt.Errorf("workspace: duplicate member name %q", name)
			}
			namesSeen[name] = true
			ws.Members = append(ws.Members, Member{
				Path:     abs,
				Name:     name,
				Manifest: memberMF,
			})
		}
	}
	return ws, nil
}

// FindRoot walks up from cwd looking for a pyproject.toml that
// contains a [tool.bunpy.workspace] table. Returns the directory
// path of the first such file found, or ErrNoWorkspace if none.
func FindRoot(cwd string) (string, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("workspace: %w", err)
	}
	dir := abs
	for {
		candidate := filepath.Join(dir, "pyproject.toml")
		if _, err := os.Stat(candidate); err == nil {
			mf, err := manifest.LoadOpts(candidate, manifest.LoadOptions{})
			if err == nil && mf.Tool.Workspace != nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", ErrNoWorkspace
}

// MemberByName returns the member with the given project.name,
// or (Member{}, false) if not found.
func MemberByName(ws *Workspace, name string) (Member, bool) {
	for _, m := range ws.Members {
		if m.Name == name {
			return m, true
		}
	}
	return Member{}, false
}

// MemberByCwd returns the member whose Path contains cwd, or
// (Member{}, false) if cwd is not inside any member directory.
func MemberByCwd(ws *Workspace, cwd string) (Member, bool) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return Member{}, false
	}
	for _, m := range ws.Members {
		if strings.HasPrefix(abs+string(filepath.Separator), m.Path+string(filepath.Separator)) || abs == m.Path {
			return m, true
		}
	}
	return Member{}, false
}

func isGlob(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

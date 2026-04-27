// Package scaffold renders new project skeletons from embedded templates.
package scaffold

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

//go:embed all:templates
var templateFS embed.FS

// Vars holds the substitution variables available in every template.
type Vars struct {
	Name        string // as given by the user, e.g. "my-cli"
	SnakeName   string // PEP 8 module name, e.g. "my_cli"
	Description string
	Author      string
	PythonMin   string // e.g. ">=3.11"
}

// Template describes one built-in template.
type Template struct {
	Name        string
	Description string
}

var builtins = []Template{
	{Name: "app", Description: "CLI application with src/ layout and __main__.py entry point"},
	{Name: "lib", Description: "Library with src/ layout and tests/"},
	{Name: "script", Description: "Single .py script with shebang"},
	{Name: "workspace", Description: "Root workspace with two member stubs"},
}

// List returns the built-in templates in sorted order.
func List() []Template {
	out := append([]Template(nil), builtins...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Lookup returns the template with the given name, or (Template{}, false).
func Lookup(name string) (Template, bool) {
	for _, t := range builtins {
		if t.Name == name {
			return t, true
		}
	}
	return Template{}, false
}

// Render expands all files of the named template into destDir.
// destDir must not already exist. Returns relative paths of created files.
func Render(templateName string, vars Vars, destDir string) ([]string, error) {
	if vars.Name == "" {
		return nil, errors.New("scaffold: Name is required")
	}
	if _, err := os.Stat(destDir); err == nil {
		return nil, fmt.Errorf("scaffold: destination %q already exists", destDir)
	}

	root := "templates/" + templateName
	var created []string

	err := fs.WalkDir(templateFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Relative path inside this template.
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Expand Vars in the path segments.
		expandedRel, err := expandPath(rel, vars)
		if err != nil {
			return fmt.Errorf("scaffold: expand path %s: %w", rel, err)
		}

		// Strip .tmpl suffix from output path.
		outRel := strings.TrimSuffix(expandedRel, ".tmpl")
		outPath := filepath.Join(destDir, outRel)

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return fmt.Errorf("scaffold: mkdir %s: %w", filepath.Dir(outPath), err)
		}

		rawBytes, err := fs.ReadFile(templateFS, path)
		if err != nil {
			return fmt.Errorf("scaffold: read %s: %w", path, err)
		}

		var content []byte
		if strings.HasSuffix(rel, ".tmpl") {
			tmpl, err := template.New(rel).Parse(string(rawBytes))
			if err != nil {
				return fmt.Errorf("scaffold: parse template %s: %w", rel, err)
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, vars); err != nil {
				return fmt.Errorf("scaffold: render %s: %w", rel, err)
			}
			content = buf.Bytes()
		} else {
			content = rawBytes
		}

		if err := os.WriteFile(outPath, content, 0o644); err != nil {
			return fmt.Errorf("scaffold: write %s: %w", outPath, err)
		}
		created = append(created, outRel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// expandPath substitutes {{.Field}} tokens in a file path.
func expandPath(path string, vars Vars) (string, error) {
	// Only expand {{.Name}} and {{.SnakeName}} in path segments.
	path = strings.ReplaceAll(path, "{{.Name}}", vars.Name)
	path = strings.ReplaceAll(path, "{{.SnakeName}}", vars.SnakeName)
	return path, nil
}

// SnakeName converts a project name (e.g. "my-cli") to a Python
// module name (e.g. "my_cli") by replacing hyphens and dots with
// underscores and lower-casing.
func SnakeName(name string) string {
	r := strings.NewReplacer("-", "_", ".", "_")
	return strings.ToLower(r.Replace(name))
}

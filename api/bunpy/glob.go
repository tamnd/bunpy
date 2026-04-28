package bunpy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildGlob returns the bunpy.glob built-in function.
func BuildGlob(_ *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "glob",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("bunpy.glob() requires a pattern argument")
			}
			patternStr, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("bunpy.glob(): pattern must be a str")
			}
			pattern := patternStr.V

			cwd := ""
			dot := false
			absolute := false
			if kwargs != nil {
				if v, ok2 := kwargs.GetStr("cwd"); ok2 {
					if s, ok3 := v.(*goipyObject.Str); ok3 {
						cwd = s.V
					}
				}
				if v, ok2 := kwargs.GetStr("dot"); ok2 {
					if b, ok3 := v.(*goipyObject.Bool); ok3 {
						dot = b.V
					}
				}
				if v, ok2 := kwargs.GetStr("absolute"); ok2 {
					if b, ok3 := v.(*goipyObject.Bool); ok3 {
						absolute = b.V
					}
				}
			}

			if cwd == "" {
				var err error
				cwd, err = os.Getwd()
				if err != nil {
					return nil, fmt.Errorf("bunpy.glob(): %w", err)
				}
			}

			matches, err := doubleStarGlob(cwd, pattern, dot)
			if err != nil {
				return nil, fmt.Errorf("bunpy.glob(): %w", err)
			}

			items := make([]goipyObject.Object, 0, len(matches))
			for _, m := range matches {
				if absolute {
					items = append(items, &goipyObject.Str{V: m})
				} else {
					rel, relErr := filepath.Rel(cwd, m)
					if relErr != nil {
						rel = m
					}
					items = append(items, &goipyObject.Str{V: filepath.ToSlash(rel)})
				}
			}
			return &goipyObject.List{V: items}, nil
		},
	}
}

// BuildGlobMatch returns the bunpy.glob_match built-in function.
func BuildGlobMatch(_ *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "glob_match",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("bunpy.glob_match() requires pattern and name arguments")
			}
			patStr, ok1 := args[0].(*goipyObject.Str)
			nameStr, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("bunpy.glob_match(): pattern and name must be str")
			}
			matched, err := filepath.Match(patStr.V, nameStr.V)
			if err != nil {
				return nil, fmt.Errorf("bunpy.glob_match(): invalid pattern: %w", err)
			}
			return goipyObject.BoolOf(matched), nil
		},
	}
}

// doubleStarGlob handles ** patterns by walking the tree.
// For patterns without **, it falls back to filepath.Glob.
func doubleStarGlob(root, pattern string, dot bool) ([]string, error) {
	// Normalize pattern to OS path separators so filepath functions work correctly.
	pattern = filepath.FromSlash(pattern)

	if !strings.Contains(pattern, "**") {
		return filepath.Glob(filepath.Join(root, pattern))
	}

	parts := strings.SplitN(pattern, "**", 2)
	prefix := filepath.Clean(filepath.Join(root, parts[0]))
	// Trim both possible separator styles after ** to handle cross-platform input.
	suffix := strings.TrimPrefix(parts[1], string(filepath.Separator))
	suffix = strings.TrimPrefix(suffix, "/")

	var matches []string
	err := filepath.WalkDir(prefix, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		name := d.Name()
		if !dot && strings.HasPrefix(name, ".") && name != "." {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if suffix == "" {
			matches = append(matches, path)
			return nil
		}
		matched, matchErr := filepath.Match(suffix, filepath.Base(path))
		if matchErr != nil {
			return matchErr
		}
		if matched {
			matches = append(matches, path)
		}
		return nil
	})
	return matches, err
}

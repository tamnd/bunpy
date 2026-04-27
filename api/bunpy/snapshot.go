package bunpy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildSnapshot builds the bunpy.snapshot module for snapshot testing.
func BuildSnapshot(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.snapshot", Dict: goipyObject.NewDict()}
	store := &snapshotStore{}

	mod.Dict.SetStr("match_snapshot", &goipyObject.BuiltinFunc{
		Name: "match_snapshot",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("match_snapshot() requires a value")
			}
			name := ""
			if len(args) >= 2 {
				if s, ok := args[1].(*goipyObject.Str); ok {
					name = s.V
				}
			}
			if kwargs != nil {
				if v, ok := kwargs.GetStr("name"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						name = s.V
					}
				}
			}
			if name == "" {
				name = fmt.Sprintf("snapshot_%d", store.nextID())
			}
			actual := snapshotRepr(args[0])
			return store.match(name, actual)
		},
	})

	mod.Dict.SetStr("update_snapshots", &goipyObject.BuiltinFunc{
		Name: "update_snapshots",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			dir := "__snapshots__"
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					dir = s.V
				}
			}
			return goipyObject.NewInt(int64(store.writeAll(dir))), nil
		},
	})

	mod.Dict.SetStr("set_snapshot_dir", &goipyObject.BuiltinFunc{
		Name: "set_snapshot_dir",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					store.mu.Lock()
					store.dir = s.V
					store.mu.Unlock()
				}
			}
			return goipyObject.None, nil
		},
	})

	return mod
}

type snapshotStore struct {
	mu       sync.Mutex
	counter  int
	dir      string
	recorded map[string]string // name -> repr
}

func (s *snapshotStore) nextID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return s.counter
}

func (s *snapshotStore) snapshotDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dir != "" {
		return s.dir
	}
	return "__snapshots__"
}

func (s *snapshotStore) match(name, actual string) (goipyObject.Object, error) {
	dir := s.snapshotDir()
	path := filepath.Join(dir, snapshotFileName(name))

	// Try to read existing snapshot.
	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run: write snapshot.
		if werr := writeSnapshot(path, actual); werr != nil {
			return nil, fmt.Errorf("snapshot: write %q: %w", name, werr)
		}
		return goipyObject.None, nil
	}
	if err != nil {
		return nil, fmt.Errorf("snapshot: read %q: %w", name, err)
	}

	// Compare.
	expected := strings.TrimRight(string(existing), "\n")
	if actual != expected {
		return nil, assertFail("Snapshot %q mismatch:\nexpected: %s\nactual:   %s", name, expected, actual)
	}
	return goipyObject.None, nil
}

func (s *snapshotStore) writeAll(dir string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for name, repr := range s.recorded {
		path := filepath.Join(dir, snapshotFileName(name))
		if err := writeSnapshot(path, repr); err == nil {
			n++
		}
	}
	return n
}

func writeSnapshot(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content+"\n"), 0o644)
}

func snapshotFileName(name string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, name)
	return safe + ".snap"
}

// snapshotRepr produces a deterministic string representation of a Python object.
func snapshotRepr(obj goipyObject.Object) string {
	return snapshotReprIndent(obj, 0)
}

func snapshotReprIndent(obj goipyObject.Object, depth int) string {
	indent := strings.Repeat("  ", depth)
	inner := strings.Repeat("  ", depth+1)

	if obj == nil || obj == goipyObject.None {
		return "None"
	}
	switch v := obj.(type) {
	case *goipyObject.Str:
		return fmt.Sprintf("%q", v.V)
	case *goipyObject.Int:
		return fmt.Sprintf("%d", v.Int64())
	case *goipyObject.Float:
		return fmt.Sprintf("%g", v.V)
	case *goipyObject.Bool:
		if v.V {
			return "True"
		}
		return "False"
	case *goipyObject.Bytes:
		return fmt.Sprintf("b%q", v.V)
	case *goipyObject.List:
		if len(v.V) == 0 {
			return "[]"
		}
		var sb strings.Builder
		sb.WriteString("[\n")
		for _, item := range v.V {
			sb.WriteString(inner)
			sb.WriteString(snapshotReprIndent(item, depth+1))
			sb.WriteString(",\n")
		}
		sb.WriteString(indent + "]")
		return sb.String()
	case *goipyObject.Tuple:
		if len(v.V) == 0 {
			return "()"
		}
		var sb strings.Builder
		sb.WriteString("(\n")
		for _, item := range v.V {
			sb.WriteString(inner)
			sb.WriteString(snapshotReprIndent(item, depth+1))
			sb.WriteString(",\n")
		}
		sb.WriteString(indent + ")")
		return sb.String()
	case *goipyObject.Dict:
		keys, vals := v.Items()
		if len(keys) == 0 {
			return "{}"
		}
		var sb strings.Builder
		sb.WriteString("{\n")
		for i, k := range keys {
			sb.WriteString(inner)
			sb.WriteString(snapshotReprIndent(k, depth+1))
			sb.WriteString(": ")
			sb.WriteString(snapshotReprIndent(vals[i], depth+1))
			sb.WriteString(",\n")
		}
		sb.WriteString(indent + "}")
		return sb.String()
	}
	return fmt.Sprintf("<%T>", obj)
}

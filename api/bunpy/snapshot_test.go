package bunpy_test

import (
	"os"
	"path/filepath"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func snapshotMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	i := serveInterp(t)
	return bunpyAPI.BuildSnapshot(i)
}

func callMatchSnapshot(t *testing.T, mod *goipyObject.Module, args []goipyObject.Object) (goipyObject.Object, error) {
	t.Helper()
	fn, ok := mod.Dict.GetStr("match_snapshot")
	if !ok {
		t.Fatal("missing match_snapshot")
	}
	return fn.(*goipyObject.BuiltinFunc).Call(nil, args, nil)
}

func TestSnapshotModuleMethods(t *testing.T) {
	mod := snapshotMod(t)
	for _, name := range []string{"match_snapshot", "update_snapshots", "set_snapshot_dir"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("snapshot module missing %q", name)
		}
	}
}

func TestSnapshotFirstRunWrites(t *testing.T) {
	dir := t.TempDir()
	mod := snapshotMod(t)

	// Set snapshot dir to temp dir.
	setDir, _ := mod.Dict.GetStr("set_snapshot_dir")
	setDir.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: dir}}, nil)

	_, err := callMatchSnapshot(t, mod, []goipyObject.Object{
		&goipyObject.Str{V: "hello"},
		&goipyObject.Str{V: "my_snap"},
	})
	if err != nil {
		t.Fatalf("first run should write and return None, got error: %v", err)
	}

	snapFile := filepath.Join(dir, "my_snap.snap")
	content, err := os.ReadFile(snapFile)
	if err != nil {
		t.Fatalf("snapshot file not written: %v", err)
	}
	if string(content) != "\"hello\"\n" {
		t.Errorf("unexpected snapshot content: %q", string(content))
	}
}

func TestSnapshotMatchPass(t *testing.T) {
	dir := t.TempDir()
	mod := snapshotMod(t)
	setDir, _ := mod.Dict.GetStr("set_snapshot_dir")
	setDir.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: dir}}, nil)

	// First run writes.
	callMatchSnapshot(t, mod, []goipyObject.Object{
		goipyObject.NewInt(42),
		&goipyObject.Str{V: "int_snap"},
	})

	// Second run should match.
	_, err := callMatchSnapshot(t, mod, []goipyObject.Object{
		goipyObject.NewInt(42),
		&goipyObject.Str{V: "int_snap"},
	})
	if err != nil {
		t.Errorf("matching snapshot should not error: %v", err)
	}
}

func TestSnapshotMatchFail(t *testing.T) {
	dir := t.TempDir()
	mod := snapshotMod(t)
	setDir, _ := mod.Dict.GetStr("set_snapshot_dir")
	setDir.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: dir}}, nil)

	// Write snapshot with value "a".
	callMatchSnapshot(t, mod, []goipyObject.Object{
		&goipyObject.Str{V: "a"},
		&goipyObject.Str{V: "str_snap"},
	})

	// Compare with different value.
	_, err := callMatchSnapshot(t, mod, []goipyObject.Object{
		&goipyObject.Str{V: "b"},
		&goipyObject.Str{V: "str_snap"},
	})
	if err == nil {
		t.Error("mismatched snapshot should return error")
	}
}

func TestSnapshotAutoName(t *testing.T) {
	dir := t.TempDir()
	mod := snapshotMod(t)
	setDir, _ := mod.Dict.GetStr("set_snapshot_dir")
	setDir.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: dir}}, nil)

	// No name provided — auto-generated.
	_, err := callMatchSnapshot(t, mod, []goipyObject.Object{&goipyObject.Str{V: "val"}})
	if err != nil {
		t.Fatalf("auto-named snapshot failed: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected 1 snapshot file, got %d", len(entries))
	}
}

func TestSnapshotReprTypes(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name  string
		obj   goipyObject.Object
		want  string
	}{
		{"none", goipyObject.None, "None\n"},
		{"bool_true", goipyObject.BoolOf(true), "True\n"},
		{"bool_false", goipyObject.BoolOf(false), "False\n"},
		{"int", goipyObject.NewInt(99), "99\n"},
		{"str", &goipyObject.Str{V: "hi"}, "\"hi\"\n"},
		{"list_empty", &goipyObject.List{V: nil}, "[]\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mod := snapshotMod(t)
			snapDir := filepath.Join(dir, tc.name)
			setDir, _ := mod.Dict.GetStr("set_snapshot_dir")
			setDir.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: snapDir}}, nil)

			callMatchSnapshot(t, mod, []goipyObject.Object{tc.obj, &goipyObject.Str{V: "snap"}})

			content, err := os.ReadFile(filepath.Join(snapDir, "snap.snap"))
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != tc.want {
				t.Errorf("repr: got %q, want %q", string(content), tc.want)
			}
		})
	}
}

func TestSnapshotUpdateAll(t *testing.T) {
	dir := t.TempDir()
	mod := snapshotMod(t)
	setDir, _ := mod.Dict.GetStr("set_snapshot_dir")
	setDir.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: dir}}, nil)

	// Write two snapshots.
	callMatchSnapshot(t, mod, []goipyObject.Object{&goipyObject.Str{V: "v1"}, &goipyObject.Str{V: "snap_a"}})
	callMatchSnapshot(t, mod, []goipyObject.Object{&goipyObject.Str{V: "v2"}, &goipyObject.Str{V: "snap_b"}})

	updateDir := t.TempDir()
	updateFn, _ := mod.Dict.GetStr("update_snapshots")
	r, err := updateFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: updateDir}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// update_snapshots only writes entries recorded in the store (match_snapshot records on first-write path
	// doesn't add to store.recorded since it writes directly; update_snapshots is for bulk re-write).
	// The function should return an int.
	if _, ok := r.(*goipyObject.Int); !ok {
		t.Errorf("update_snapshots should return int, got %T", r)
	}
}

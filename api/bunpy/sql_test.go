package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func openMemDB(t *testing.T) *goipyObject.Instance {
	t.Helper()
	i := serveInterp(t)
	sqlFn := bunpyAPI.BuildSQL(i)
	result, err := sqlFn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result.(*goipyObject.Instance)
}

func dbRun(t *testing.T, db *goipyObject.Instance, q string, args ...goipyObject.Object) {
	t.Helper()
	runFn, _ := db.Dict.GetStr("run")
	callArgs := []goipyObject.Object{&goipyObject.Str{V: q}}
	if len(args) > 0 {
		callArgs = append(callArgs, &goipyObject.List{V: args})
	}
	if _, err := runFn.(*goipyObject.BuiltinFunc).Call(nil, callArgs, nil); err != nil {
		t.Fatalf("db.run(%q): %v", q, err)
	}
}

func dbQuery(t *testing.T, db *goipyObject.Instance, q string, args ...goipyObject.Object) []goipyObject.Object {
	t.Helper()
	queryFn, _ := db.Dict.GetStr("query")
	callArgs := []goipyObject.Object{&goipyObject.Str{V: q}}
	if len(args) > 0 {
		callArgs = append(callArgs, &goipyObject.List{V: args})
	}
	result, err := queryFn.(*goipyObject.BuiltinFunc).Call(nil, callArgs, nil)
	if err != nil {
		t.Fatalf("db.query(%q): %v", q, err)
	}
	return result.(*goipyObject.List).V
}

func TestSQLBasicCRUD(t *testing.T) {
	db := openMemDB(t)
	dbRun(t, db, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	dbRun(t, db, "INSERT INTO users (name) VALUES (?)", &goipyObject.Str{V: "Alice"})
	dbRun(t, db, "INSERT INTO users (name) VALUES (?)", &goipyObject.Str{V: "Bob"})

	rows := dbQuery(t, db, "SELECT name FROM users ORDER BY name")
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	row0 := rows[0].(*goipyObject.Instance)
	nameObj, _ := row0.Dict.GetStr("name")
	if nameObj.(*goipyObject.Str).V != "Alice" {
		t.Fatalf("row[0].name = %q, want Alice", nameObj.(*goipyObject.Str).V)
	}
}

func TestSQLQueryOne(t *testing.T) {
	db := openMemDB(t)
	dbRun(t, db, "CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT)")
	dbRun(t, db, "INSERT INTO items (val) VALUES (?)", &goipyObject.Str{V: "hello"})

	queryOneFn, _ := db.Dict.GetStr("query_one")

	// found
	res, err := queryOneFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "SELECT val FROM items WHERE id = ?"},
		&goipyObject.List{V: []goipyObject.Object{goipyObject.NewInt(1)}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	row := res.(*goipyObject.Instance)
	v, _ := row.Dict.GetStr("val")
	if v.(*goipyObject.Str).V != "hello" {
		t.Fatalf("val = %q, want hello", v.(*goipyObject.Str).V)
	}

	// not found
	res2, err2 := queryOneFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "SELECT val FROM items WHERE id = ?"},
		&goipyObject.List{V: []goipyObject.Object{goipyObject.NewInt(999)}},
	}, nil)
	if err2 != nil {
		t.Fatal(err2)
	}
	if _, ok := res2.(*goipyObject.NoneType); !ok {
		t.Fatalf("query_one on missing row should return None, got %T", res2)
	}
}

func TestSQLTransaction(t *testing.T) {
	db := openMemDB(t)
	dbRun(t, db, "CREATE TABLE t (v TEXT)")

	i := serveInterp(t)
	txFn, _ := db.Dict.GetStr("transaction")

	handler := &goipyObject.BuiltinFunc{
		Name: "handler",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			tx := args[0].(*goipyObject.Instance)
			runFn, _ := tx.Dict.GetStr("run")
			runFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
				&goipyObject.Str{V: "INSERT INTO t VALUES (?)"},
				&goipyObject.List{V: []goipyObject.Object{&goipyObject.Str{V: "x"}}},
			}, nil)
			return goipyObject.None, nil
		},
	}
	_ = i
	if _, err := txFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{handler}, nil); err != nil {
		t.Fatal(err)
	}

	rows := dbQuery(t, db, "SELECT v FROM t")
	if len(rows) != 1 {
		t.Fatalf("expected 1 row after transaction, got %d", len(rows))
	}
}

func TestSQLRunMany(t *testing.T) {
	db := openMemDB(t)
	dbRun(t, db, "CREATE TABLE m (n TEXT)")

	runManyFn, _ := db.Dict.GetStr("run_many")
	rows := &goipyObject.List{V: []goipyObject.Object{
		&goipyObject.List{V: []goipyObject.Object{&goipyObject.Str{V: "A"}}},
		&goipyObject.List{V: []goipyObject.Object{&goipyObject.Str{V: "B"}}},
		&goipyObject.List{V: []goipyObject.Object{&goipyObject.Str{V: "C"}}},
	}}
	if _, err := runManyFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "INSERT INTO m VALUES (?)"},
		rows,
	}, nil); err != nil {
		t.Fatal(err)
	}

	got := dbQuery(t, db, "SELECT count(*) AS c FROM m")
	cObj, _ := got[0].(*goipyObject.Instance).Dict.GetStr("c")
	if cObj.(*goipyObject.Int).Int64() != 3 {
		t.Fatalf("expected 3 rows, got %v", cObj)
	}
}

func TestBunpyModuleHasSQL(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	if _, ok := m.Dict.GetStr("sql"); !ok {
		t.Fatal("bunpy.sql missing from top-level module")
	}
}

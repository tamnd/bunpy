package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestSQLInMemoryStillWorks(t *testing.T) {
	// Regression: adding new drivers must not break the default in-memory SQLite path.
	i := serveInterp(t)
	sqlFn := bunpyAPI.BuildSQL(i)
	result, err := sqlFn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	db := result.(*goipyObject.Instance)
	runFn, _ := db.Dict.GetStr("run")
	if _, err2 := runFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "CREATE TABLE x (v TEXT)"},
	}, nil); err2 != nil {
		t.Fatal(err2)
	}
}

func TestSQLUnsupportedScheme(t *testing.T) {
	i := serveInterp(t)
	sqlFn := bunpyAPI.BuildSQL(i)
	_, err := sqlFn.Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "redis://localhost"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

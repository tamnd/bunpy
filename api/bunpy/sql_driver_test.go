package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

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

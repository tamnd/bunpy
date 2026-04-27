package bunpy_test

import (
	"testing"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestBunpyModuleHasSQL(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	if _, ok := m.Dict.GetStr("sql"); !ok {
		t.Fatal("bunpy.sql missing from top-level module")
	}
}

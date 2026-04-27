package bunpy_test

import (
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func csvMod() *goipyObject.Module { return bunpyAPI.BuildCSV(nil) }

func TestCSVParseWithHeader(t *testing.T) {
	mod := csvMod()
	parseFn, _ := mod.Dict.GetStr("parse")
	result, err := parseFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "name,age\nAlice,30\nBob,25"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := result.(*goipyObject.List).V
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	first := rows[0].(*goipyObject.Dict)
	name, _ := first.GetStr("name")
	if name.(*goipyObject.Str).V != "Alice" {
		t.Fatalf("expected Alice, got %v", name)
	}
}

func TestCSVParseNoHeader(t *testing.T) {
	mod := csvMod()
	parseFn, _ := mod.Dict.GetStr("parse")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("header", goipyObject.BoolOf(false))
	result, err := parseFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "Alice,30\nBob,25"},
	}, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	rows := result.(*goipyObject.List).V
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	first := rows[0].(*goipyObject.List)
	if first.V[0].(*goipyObject.Str).V != "Alice" {
		t.Fatalf("expected Alice in first cell")
	}
}

func TestCSVWriteDicts(t *testing.T) {
	mod := csvMod()
	writeFn, _ := mod.Dict.GetStr("write")

	d := goipyObject.NewDict()
	d.SetStr("name", &goipyObject.Str{V: "Alice"})
	d.SetStr("age", &goipyObject.Str{V: "30"})
	rows := &goipyObject.List{V: []goipyObject.Object{d}}

	result, err := writeFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{rows}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*goipyObject.Str).V
	if !strings.Contains(out, "Alice") {
		t.Fatalf("expected Alice in CSV output, got: %q", out)
	}
}

func TestCSVWriteListsWithHeader(t *testing.T) {
	mod := csvMod()
	writeFn, _ := mod.Dict.GetStr("write")

	rows := &goipyObject.List{V: []goipyObject.Object{
		&goipyObject.List{V: []goipyObject.Object{
			&goipyObject.Str{V: "Alice"},
			&goipyObject.Str{V: "30"},
		}},
	}}
	header := &goipyObject.List{V: []goipyObject.Object{
		&goipyObject.Str{V: "name"},
		&goipyObject.Str{V: "age"},
	}}
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("header", header)
	result, err := writeFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{rows}, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*goipyObject.Str).V
	if !strings.HasPrefix(out, "name,age") {
		t.Fatalf("expected header row, got: %q", out)
	}
}

func TestCSVWriteQuotesCommas(t *testing.T) {
	mod := csvMod()
	writeFn, _ := mod.Dict.GetStr("write")

	rows := &goipyObject.List{V: []goipyObject.Object{
		&goipyObject.List{V: []goipyObject.Object{
			&goipyObject.Str{V: "hello, world"},
			&goipyObject.Str{V: "42"},
		}},
	}}
	result, _ := writeFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{rows}, nil)
	out := result.(*goipyObject.Str).V
	if !strings.Contains(out, `"hello, world"`) {
		t.Fatalf("expected quoted field, got: %q", out)
	}
}

func TestCSVParseQuotedFields(t *testing.T) {
	mod := csvMod()
	parseFn, _ := mod.Dict.GetStr("parse")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("header", goipyObject.BoolOf(false))
	result, err := parseFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: `"hello, world",42`},
	}, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	row := result.(*goipyObject.List).V[0].(*goipyObject.List)
	if row.V[0].(*goipyObject.Str).V != "hello, world" {
		t.Fatalf("expected 'hello, world', got %q", row.V[0].(*goipyObject.Str).V)
	}
}

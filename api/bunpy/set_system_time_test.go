package bunpy_test

import (
	"testing"
	"time"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func sstMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildSetSystemTime(nil)
}

func TestSetSystemTimeModuleMethods(t *testing.T) {
	mod := sstMod(t)
	for _, name := range []string{"set_system_time", "now", "reset"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("set_system_time module missing %q", name)
		}
	}
}

func TestSetSystemTimeFreezesTime(t *testing.T) {
	mod := sstMod(t)
	setFn, _ := mod.Dict.GetStr("set_system_time")
	nowFn, _ := mod.Dict.GetStr("now")
	resetFn, _ := mod.Dict.GetStr("reset")

	defer resetFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)

	frozen := "2020-01-15T10:00:00Z"
	setFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: frozen},
	}, nil)

	r, err := nowFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := r.(*goipyObject.Dict)
	year, _ := d.GetStr("year")
	if year.(*goipyObject.Int).Int64() != 2020 {
		t.Errorf("expected year 2020, got %v", year)
	}
	iso, _ := d.GetStr("iso")
	if iso.(*goipyObject.Str).V != frozen {
		t.Errorf("expected iso %q, got %q", frozen, iso.(*goipyObject.Str).V)
	}
}

func TestSetSystemTimeReset(t *testing.T) {
	mod := sstMod(t)
	setFn, _ := mod.Dict.GetStr("set_system_time")
	nowFn, _ := mod.Dict.GetStr("now")
	resetFn, _ := mod.Dict.GetStr("reset")

	setFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "2000-01-01T00:00:00Z"},
	}, nil)
	resetFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)

	r, _ := nowFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	d := r.(*goipyObject.Dict)
	year, _ := d.GetStr("year")
	if year.(*goipyObject.Int).Int64() == 2000 {
		t.Error("reset should restore real time, not 2000")
	}
}

func TestSetSystemTimeUnixMillis(t *testing.T) {
	mod := sstMod(t)
	setFn, _ := mod.Dict.GetStr("set_system_time")
	nowFn, _ := mod.Dict.GetStr("now")
	resetFn, _ := mod.Dict.GetStr("reset")
	defer resetFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)

	ms := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	setFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		goipyObject.NewInt(ms),
	}, nil)
	r, _ := nowFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	d := r.(*goipyObject.Dict)
	year, _ := d.GetStr("year")
	if year.(*goipyObject.Int).Int64() != 2023 {
		t.Errorf("expected year 2023, got %v", year)
	}
}

func TestSetSystemTimeNowFields(t *testing.T) {
	mod := sstMod(t)
	nowFn, _ := mod.Dict.GetStr("now")
	r, err := nowFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := r.(*goipyObject.Dict)
	for _, field := range []string{"year", "month", "day", "hour", "minute", "second", "unix", "unix_ms", "iso"} {
		if _, ok := d.GetStr(field); !ok {
			t.Errorf("now() result missing field %q", field)
		}
	}
}

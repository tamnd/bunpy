package bunpy_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestLogInfoNoPanic(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildLog(i)
	infoFn, _ := mod.Dict.GetStr("info")
	_, err := infoFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "test message"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLogJSONFormat(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildLog(i)

	tmp := filepath.Join(t.TempDir(), "test.log")
	t.Cleanup(func() {
		configureFn, _ := mod.Dict.GetStr("configure")
		configureFn.(*goipyObject.BuiltinFunc).Call(nil, nil, goipyObject.NewDict())
	})

	configureFn, _ := mod.Dict.GetStr("configure")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("format", &goipyObject.Str{V: "json"})
	kwargs.SetStr("file", &goipyObject.Str{V: tmp})
	kwargs.SetStr("level", &goipyObject.Str{V: "debug"})
	_, err := configureFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)
	if err != nil {
		t.Fatal(err)
	}

	infoFn, _ := mod.Dict.GetStr("info")
	infoFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "json test"},
	}, nil)

	data, _ := os.ReadFile(tmp)
	if len(data) == 0 {
		t.Fatal("expected log output in file")
	}
	var m map[string]any
	for _, line := range bytes.Split(data, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("not valid JSON: %s", line)
		}
	}
}

func TestLogLevelFiltering(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildLog(i)

	tmp := filepath.Join(t.TempDir(), "level.log")
	t.Cleanup(func() {
		configureFn, _ := mod.Dict.GetStr("configure")
		configureFn.(*goipyObject.BuiltinFunc).Call(nil, nil, goipyObject.NewDict())
	})

	configureFn, _ := mod.Dict.GetStr("configure")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("level", &goipyObject.Str{V: "info"})
	kwargs.SetStr("format", &goipyObject.Str{V: "text"})
	kwargs.SetStr("file", &goipyObject.Str{V: tmp})
	configureFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)

	debugFn, _ := mod.Dict.GetStr("debug")
	debugFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "should not appear"},
	}, nil)

	data, _ := os.ReadFile(tmp)
	if strings.Contains(string(data), "should not appear") {
		t.Fatal("debug message should be filtered when level=info")
	}
}

func TestLogWithFields(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildLog(i)

	tmp := filepath.Join(t.TempDir(), "fields.log")
	t.Cleanup(func() {
		configureFn, _ := mod.Dict.GetStr("configure")
		configureFn.(*goipyObject.BuiltinFunc).Call(nil, nil, goipyObject.NewDict())
	})

	configureFn, _ := mod.Dict.GetStr("configure")
	kw := goipyObject.NewDict()
	kw.SetStr("format", &goipyObject.Str{V: "text"})
	kw.SetStr("file", &goipyObject.Str{V: tmp})
	kw.SetStr("level", &goipyObject.Str{V: "debug"})
	configureFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kw)

	withFn, _ := mod.Dict.GetStr("with_fields")
	wfKw := goipyObject.NewDict()
	wfKw.SetStr("request_id", &goipyObject.Str{V: "abc123"})
	child, err := withFn.(*goipyObject.BuiltinFunc).Call(nil, nil, wfKw)
	if err != nil {
		t.Fatal(err)
	}

	inst, ok := child.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", child)
	}
	infoFn, ok2 := inst.Dict.GetStr("info")
	if !ok2 {
		t.Fatal("child logger missing info method")
	}
	infoFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "child message"},
	}, nil)

	data, _ := os.ReadFile(tmp)
	if !strings.Contains(string(data), "request_id") {
		t.Fatal("expected bound field request_id in child logger output")
	}
}

func TestLogConfigureFile(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildLog(i)

	tmp := filepath.Join(t.TempDir(), "out.log")
	t.Cleanup(func() {
		configureFn, _ := mod.Dict.GetStr("configure")
		configureFn.(*goipyObject.BuiltinFunc).Call(nil, nil, goipyObject.NewDict())
	})
	configureFn, _ := mod.Dict.GetStr("configure")
	kw := goipyObject.NewDict()
	kw.SetStr("file", &goipyObject.Str{V: tmp})
	kw.SetStr("level", &goipyObject.Str{V: "debug"})
	_, err := configureFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kw)
	if err != nil {
		t.Fatal(err)
	}

	infoFn, _ := mod.Dict.GetStr("info")
	infoFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "file test"},
	}, nil)

	data, _ := os.ReadFile(tmp)
	if !strings.Contains(string(data), "file test") {
		t.Fatalf("expected message in log file, got: %s", data)
	}
}

func TestLogUnknownLevelError(t *testing.T) {
	i := serveInterp(t)
	mod := bunpyAPI.BuildLog(i)

	configureFn, _ := mod.Dict.GetStr("configure")
	kw := goipyObject.NewDict()
	kw.SetStr("level", &goipyObject.Str{V: "verbose"})
	_, err := configureFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kw)
	if err == nil {
		t.Fatal("expected error for unknown level")
	}
}

// reset global logger to default stderr/info after tests that change it
func init() {
	_ = slog.LevelInfo // ensure slog is imported
}

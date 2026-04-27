package bunpy_test

import (
	"testing"
	"time"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func asyncioMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	i := serveInterp(t)
	return bunpyAPI.BuildAsyncio(i)
}

func TestAsyncioRun(t *testing.T) {
	mod := asyncioMod(t)
	runFn, _ := mod.Dict.GetStr("run")

	fn := &goipyObject.BuiltinFunc{
		Name: "fn",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.NewInt(42), nil
		},
	}
	r, err := runFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{fn}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.(*goipyObject.Int).Int64() != 42 {
		t.Errorf("expected 42, got %v", r)
	}
}

func TestAsyncioSleep(t *testing.T) {
	mod := asyncioMod(t)
	sleepFn, _ := mod.Dict.GetStr("sleep")
	_, err := sleepFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{goipyObject.NewInt(0)}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAsyncioGather(t *testing.T) {
	mod := asyncioMod(t)
	gatherFn, _ := mod.Dict.GetStr("gather")

	mk := func(n int64) *goipyObject.BuiltinFunc {
		return &goipyObject.BuiltinFunc{
			Name: "fn",
			Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
				return goipyObject.NewInt(n), nil
			},
		}
	}
	r, err := gatherFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{mk(1), mk(2)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	list := r.(*goipyObject.List)
	if len(list.V) != 2 {
		t.Errorf("expected 2 results, got %d", len(list.V))
	}
}

func TestAsyncioCreateTask(t *testing.T) {
	mod := asyncioMod(t)
	createFn, _ := mod.Dict.GetStr("create_task")

	fn := &goipyObject.BuiltinFunc{
		Name: "fn",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			time.Sleep(10 * time.Millisecond)
			return &goipyObject.Str{V: "done"}, nil
		},
	}

	task, err := createFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{fn}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := task.(*goipyObject.Instance)

	doneFn, _ := inst.Dict.GetStr("done")
	r, _ := doneFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	// May or may not be done yet — just confirm it returns a Bool.
	if _, ok := r.(*goipyObject.Bool); !ok {
		t.Errorf("done() should return Bool, got %T", r)
	}

	// Wait for completion via result().
	resultFn, _ := inst.Dict.GetStr("result")
	res, err := resultFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.(*goipyObject.Str).V != "done" {
		t.Errorf("expected 'done', got %v", res)
	}

	// After result(), done() must be true.
	r2, _ := doneFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if !r2.(*goipyObject.Bool).V {
		t.Error("done() should be true after result() returns")
	}
}

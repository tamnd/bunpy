package bunpy_test

import (
	"sync/atomic"
	"testing"
	"time"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func workerMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	i := serveInterp(t)
	return bunpyAPI.BuildWorker(i)
}

func TestWorkerModuleHasConstructor(t *testing.T) {
	mod := workerMod(t)
	if _, ok := mod.Dict.GetStr("Worker"); !ok {
		t.Fatal("Worker constructor not found in module")
	}
}

func TestWorkerReceivesInitCall(t *testing.T) {
	mod := workerMod(t)
	fn, _ := mod.Dict.GetStr("Worker")

	var called atomic.Bool
	workerFn := &goipyObject.BuiltinFunc{
		Name: "worker",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			called.Store(true)
			return goipyObject.None, nil
		},
	}
	fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{workerFn}, nil)
	time.Sleep(50 * time.Millisecond)
	if !called.Load() {
		t.Fatal("worker function should have been called on creation")
	}
}

func TestWorkerPostMessage(t *testing.T) {
	mod := workerMod(t)
	fn, _ := mod.Dict.GetStr("Worker")

	var received atomic.Value
	workerFn := &goipyObject.BuiltinFunc{
		Name: "worker",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, nil
		},
	}
	wObj, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{workerFn}, nil)
	inst := wObj.(*goipyObject.Instance)

	// register message listener
	onFn, _ := inst.Dict.GetStr("on")
	listener := &goipyObject.BuiltinFunc{
		Name: "listener",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					received.Store(s.V)
				}
			}
			return goipyObject.None, nil
		},
	}
	onFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "message"}, listener,
	}, nil)

	postFn, _ := inst.Dict.GetStr("post_message")
	postFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "hello from main"},
	}, nil)

	time.Sleep(60 * time.Millisecond)
	if v, ok := received.Load().(string); !ok || v != "hello from main" {
		t.Errorf("expected 'hello from main', got %v", received.Load())
	}
}

func TestWorkerTerminate(t *testing.T) {
	mod := workerMod(t)
	fn, _ := mod.Dict.GetStr("Worker")

	workerFn := &goipyObject.BuiltinFunc{
		Name: "worker",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, nil
		},
	}
	wObj, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{workerFn}, nil)
	inst := wObj.(*goipyObject.Instance)

	termFn, _ := inst.Dict.GetStr("terminate")
	_, err := termFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// double-terminate should be a no-op
	_, err = termFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatalf("double-terminate should not error: %v", err)
	}
}

func TestWorkerPostAfterTerminateErrors(t *testing.T) {
	mod := workerMod(t)
	fn, _ := mod.Dict.GetStr("Worker")

	workerFn := &goipyObject.BuiltinFunc{
		Name: "worker",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, nil
		},
	}
	wObj, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{workerFn}, nil)
	inst := wObj.(*goipyObject.Instance)

	termFn, _ := inst.Dict.GetStr("terminate")
	termFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)

	time.Sleep(20 * time.Millisecond)
	postFn, _ := inst.Dict.GetStr("post_message")
	_, err := postFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "msg"},
	}, nil)
	if err == nil {
		t.Fatal("posting to terminated worker should error")
	}
}

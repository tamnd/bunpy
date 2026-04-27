package bunpy_test

import (
	"sync/atomic"
	"testing"
	"time"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestSetTimeoutFires(t *testing.T) {
	i := serveInterp(t)
	bunpyAPI.InjectTimerGlobals(i)

	var fired atomic.Bool
	cb := &goipyObject.BuiltinFunc{
		Name: "cb",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			fired.Store(true)
			return goipyObject.None, nil
		},
	}
	setTimeoutFn, _ := i.Builtins.GetStr("setTimeout")
	setTimeoutFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		cb,
		goipyObject.NewInt(20),
	}, nil)

	time.Sleep(60 * time.Millisecond)
	if !fired.Load() {
		t.Fatal("setTimeout callback did not fire")
	}
}

func TestClearTimeoutCancels(t *testing.T) {
	i := serveInterp(t)
	bunpyAPI.InjectTimerGlobals(i)

	var fired atomic.Bool
	cb := &goipyObject.BuiltinFunc{
		Name: "cb",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			fired.Store(true)
			return goipyObject.None, nil
		},
	}
	setTimeoutFn, _ := i.Builtins.GetStr("setTimeout")
	idObj, _ := setTimeoutFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		cb,
		goipyObject.NewInt(100),
	}, nil)

	clearFn, _ := i.Builtins.GetStr("clearTimeout")
	clearFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{idObj}, nil)

	time.Sleep(150 * time.Millisecond)
	if fired.Load() {
		t.Fatal("cancelled timeout should not fire")
	}
}

func TestSetIntervalFires(t *testing.T) {
	i := serveInterp(t)
	bunpyAPI.InjectTimerGlobals(i)

	var count atomic.Int32
	cb := &goipyObject.BuiltinFunc{
		Name: "cb",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			count.Add(1)
			return goipyObject.None, nil
		},
	}
	setIntervalFn, _ := i.Builtins.GetStr("setInterval")
	idObj, _ := setIntervalFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		cb,
		goipyObject.NewInt(20),
	}, nil)

	time.Sleep(80 * time.Millisecond)
	clearFn, _ := i.Builtins.GetStr("clearInterval")
	clearFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{idObj}, nil)

	if count.Load() < 2 {
		t.Fatalf("expected at least 2 interval ticks, got %d", count.Load())
	}
}

func TestSetTimeoutReturnsID(t *testing.T) {
	i := serveInterp(t)
	bunpyAPI.InjectTimerGlobals(i)

	cb := &goipyObject.BuiltinFunc{
		Name: "noop",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, nil
		},
	}
	setTimeoutFn, _ := i.Builtins.GetStr("setTimeout")
	id, err := setTimeoutFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		cb, goipyObject.NewInt(10000),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := id.(*goipyObject.Int); !ok {
		t.Fatalf("expected int timer ID, got %T", id)
	}
	// cancel it
	clearFn, _ := i.Builtins.GetStr("clearTimeout")
	clearFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{id}, nil)
}

func TestTimerGlobalsInjected(t *testing.T) {
	i := serveInterp(t)
	bunpyAPI.InjectTimerGlobals(i)
	for _, name := range []string{"setTimeout", "clearTimeout", "setInterval", "clearInterval"} {
		if _, ok := i.Builtins.GetStr(name); !ok {
			t.Fatalf("timer global %q not injected", name)
		}
	}
}

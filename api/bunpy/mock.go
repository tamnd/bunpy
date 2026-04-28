package bunpy

import (
	"fmt"
	"sync"
	"sync/atomic"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildMock builds the bunpy.mock module providing mock() and spy_on().
func BuildMock(i *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.mock", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("mock", &goipyObject.BuiltinFunc{
		Name: "mock",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			// mock(return_value=None) builds a callable that tracks calls.
			var returnVal goipyObject.Object = goipyObject.None
			if len(args) >= 1 {
				returnVal = args[0]
			}
			if kwargs != nil {
				if v, ok := kwargs.GetStr("return_value"); ok {
					returnVal = v
				}
			}
			return newMockFn(i, returnVal, nil), nil
		},
	})

	mod.Dict.SetStr("spy_on", &goipyObject.BuiltinFunc{
		Name: "spy_on",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("spy_on() requires a callable to wrap")
			}
			original := args[0]
			return newMockFn(i, nil, original), nil
		},
	})

	return mod
}

type callRecord struct {
	args   []goipyObject.Object
	kwargs *goipyObject.Dict
}

type mockState struct {
	mu          sync.Mutex
	calls       []callRecord
	callCount   atomic.Int64
	returnValue goipyObject.Object
	original    goipyObject.Object // non-nil for spy_on
	interp      *goipyVM.Interp
	sideEffect  goipyObject.Object // callable to invoke instead (if set)
}

func newMockFn(i *goipyVM.Interp, returnValue goipyObject.Object, original goipyObject.Object) *goipyObject.Instance {
	state := &mockState{
		returnValue: returnValue,
		original:    original,
		interp:      i,
	}

	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Mock"},
		Dict:  goipyObject.NewDict(),
	}

	// The mock itself is callable via __call__.
	callFn := &goipyObject.BuiltinFunc{
		Name: "__call__",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			state.callCount.Add(1)
			state.mu.Lock()
			state.calls = append(state.calls, callRecord{args: args, kwargs: kwargs})
			state.mu.Unlock()

			if state.sideEffect != nil {
				return i.CallObject(state.sideEffect, args, kwargs)
			}
			if state.original != nil {
				return i.CallObject(state.original, args, kwargs)
			}
			if state.returnValue != nil {
				return state.returnValue, nil
			}
			return goipyObject.None, nil
		},
	}
	inst.Dict.SetStr("__call__", callFn)

	// call_count property.
	inst.Dict.SetStr("call_count", &goipyObject.BuiltinFunc{
		Name: "call_count",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.NewInt(state.callCount.Load()), nil
		},
	})

	// calls property — list of argument lists.
	inst.Dict.SetStr("calls", &goipyObject.BuiltinFunc{
		Name: "calls",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			state.mu.Lock()
			defer state.mu.Unlock()
			items := make([]goipyObject.Object, len(state.calls))
			for i, c := range state.calls {
				if len(c.args) == 0 {
					items[i] = &goipyObject.List{V: nil}
				} else {
					items[i] = &goipyObject.List{V: c.args}
				}
			}
			return &goipyObject.List{V: items}, nil
		},
	})

	// was_called() returns True if called at least once.
	inst.Dict.SetStr("was_called", &goipyObject.BuiltinFunc{
		Name: "was_called",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.BoolOf(state.callCount.Load() > 0), nil
		},
	})

	// called_with(args...) returns True if the last call matched.
	inst.Dict.SetStr("called_with", &goipyObject.BuiltinFunc{
		Name: "called_with",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			state.mu.Lock()
			defer state.mu.Unlock()
			if len(state.calls) == 0 {
				return goipyObject.BoolOf(false), nil
			}
			last := state.calls[len(state.calls)-1]
			if len(last.args) != len(args) {
				return goipyObject.BoolOf(false), nil
			}
			for i, a := range args {
				if !deepEquals(last.args[i], a) {
					return goipyObject.BoolOf(false), nil
				}
			}
			return goipyObject.BoolOf(true), nil
		},
	})

	// return_value setter.
	inst.Dict.SetStr("set_return_value", &goipyObject.BuiltinFunc{
		Name: "set_return_value",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) >= 1 {
				state.returnValue = args[0]
			}
			return inst, nil
		},
	})

	// side_effect setter — replaces the return value with a callable.
	inst.Dict.SetStr("set_side_effect", &goipyObject.BuiltinFunc{
		Name: "set_side_effect",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) >= 1 {
				state.sideEffect = args[0]
			}
			return inst, nil
		},
	})

	// reset() clears call records.
	inst.Dict.SetStr("reset", &goipyObject.BuiltinFunc{
		Name: "reset",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			state.mu.Lock()
			state.calls = nil
			state.mu.Unlock()
			state.callCount.Store(0)
			return inst, nil
		},
	})

	return inst
}

package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func mockMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	i := serveInterp(t)
	return bunpyAPI.BuildMock(i)
}

func TestMockModuleMethods(t *testing.T) {
	mod := mockMod(t)
	for _, name := range []string{"mock", "spy_on"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("mock module missing %q", name)
		}
	}
}

func TestMockCallsAndCount(t *testing.T) {
	mod := mockMod(t)
	fn, _ := mod.Dict.GetStr("mock")
	m, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	inst := m.(*goipyObject.Instance)

	callFn, _ := inst.Dict.GetStr("__call__")
	callFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{goipyObject.NewInt(1)}, nil)
	callFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{goipyObject.NewInt(2)}, nil)

	countFn, _ := inst.Dict.GetStr("call_count")
	r, _ := countFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if r.(*goipyObject.Int).Int64() != 2 {
		t.Errorf("expected call_count=2, got %v", r)
	}
}

func TestMockWasCalled(t *testing.T) {
	mod := mockMod(t)
	fn, _ := mod.Dict.GetStr("mock")
	m, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	inst := m.(*goipyObject.Instance)

	wasFn, _ := inst.Dict.GetStr("was_called")
	r, _ := wasFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if r.(*goipyObject.Bool).V {
		t.Error("was_called should be false before any call")
	}

	callFn, _ := inst.Dict.GetStr("__call__")
	callFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)

	r, _ = wasFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if !r.(*goipyObject.Bool).V {
		t.Error("was_called should be true after call")
	}
}

func TestMockReturnValue(t *testing.T) {
	mod := mockMod(t)
	fn, _ := mod.Dict.GetStr("mock")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("return_value", &goipyObject.Str{V: "hello"})
	m, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)
	inst := m.(*goipyObject.Instance)

	callFn, _ := inst.Dict.GetStr("__call__")
	r, err := callFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.(*goipyObject.Str).V != "hello" {
		t.Errorf("expected return_value 'hello', got %v", r)
	}
}

func TestMockCalledWith(t *testing.T) {
	mod := mockMod(t)
	fn, _ := mod.Dict.GetStr("mock")
	m, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	inst := m.(*goipyObject.Instance)

	callFn, _ := inst.Dict.GetStr("__call__")
	callFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "arg1"},
	}, nil)

	cwFn, _ := inst.Dict.GetStr("called_with")
	r, _ := cwFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "arg1"},
	}, nil)
	if !r.(*goipyObject.Bool).V {
		t.Error("called_with should match last call args")
	}
	r, _ = cwFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "other"},
	}, nil)
	if r.(*goipyObject.Bool).V {
		t.Error("called_with should not match different args")
	}
}

func TestMockReset(t *testing.T) {
	mod := mockMod(t)
	fn, _ := mod.Dict.GetStr("mock")
	m, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	inst := m.(*goipyObject.Instance)

	callFn, _ := inst.Dict.GetStr("__call__")
	callFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	callFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)

	resetFn, _ := inst.Dict.GetStr("reset")
	resetFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)

	countFn, _ := inst.Dict.GetStr("call_count")
	r, _ := countFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if r.(*goipyObject.Int).Int64() != 0 {
		t.Errorf("after reset call_count should be 0, got %v", r)
	}
}

func TestSpyOn(t *testing.T) {
	mod := mockMod(t)
	fn, _ := mod.Dict.GetStr("spy_on")

	var originalCalled bool
	original := &goipyObject.BuiltinFunc{
		Name: "original",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			originalCalled = true
			return &goipyObject.Str{V: "from original"}, nil
		},
	}
	m, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{original}, nil)
	inst := m.(*goipyObject.Instance)

	callFn, _ := inst.Dict.GetStr("__call__")
	r, err := callFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !originalCalled {
		t.Error("spy_on should call the original function")
	}
	if r.(*goipyObject.Str).V != "from original" {
		t.Errorf("spy_on should return original's return value, got %v", r)
	}

	countFn, _ := inst.Dict.GetStr("call_count")
	count, _ := countFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if count.(*goipyObject.Int).Int64() != 1 {
		t.Error("spy_on should track calls")
	}
}

func TestMockSideEffect(t *testing.T) {
	mod := mockMod(t)
	fn, _ := mod.Dict.GetStr("mock")
	m, _ := fn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	inst := m.(*goipyObject.Instance)

	setSE, _ := inst.Dict.GetStr("set_side_effect")
	sideEffect := &goipyObject.BuiltinFunc{
		Name: "side",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return &goipyObject.Str{V: "from side effect"}, nil
		},
	}
	setSE.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{sideEffect}, nil)

	callFn, _ := inst.Dict.GetStr("__call__")
	r, err := callFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.(*goipyObject.Str).V != "from side effect" {
		t.Errorf("expected side effect return, got %v", r)
	}
}

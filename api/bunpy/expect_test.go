package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func expectMod(t *testing.T) *goipyObject.Module {
	t.Helper()
	return bunpyAPI.BuildExpect(nil)
}

func callExpect(t *testing.T, val goipyObject.Object) *goipyObject.Instance {
	t.Helper()
	mod := expectMod(t)
	fn, _ := mod.Dict.GetStr("expect")
	r, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{val}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*goipyObject.Instance)
}

func callMatcher(t *testing.T, inst *goipyObject.Instance, name string, args ...goipyObject.Object) error {
	t.Helper()
	fn, ok := inst.Dict.GetStr(name)
	if !ok {
		t.Fatalf("matcher %q not found", name)
	}
	_, err := fn.(*goipyObject.BuiltinFunc).Call(nil, args, nil)
	return err
}

func TestExpectModuleMethods(t *testing.T) {
	mod := expectMod(t)
	for _, name := range []string{"expect", "describe", "it", "test", "skip"} {
		if _, ok := mod.Dict.GetStr(name); !ok {
			t.Fatalf("expect module missing %q", name)
		}
	}
}

func TestExpectToEqualPass(t *testing.T) {
	inst := callExpect(t, goipyObject.NewInt(42))
	if err := callMatcher(t, inst, "to_equal", goipyObject.NewInt(42)); err != nil {
		t.Errorf("to_equal should pass for 42==42: %v", err)
	}
}

func TestExpectToEqualFail(t *testing.T) {
	inst := callExpect(t, goipyObject.NewInt(42))
	if err := callMatcher(t, inst, "to_equal", goipyObject.NewInt(99)); err == nil {
		t.Error("to_equal should fail for 42!=99")
	}
}

func TestExpectToBeTruePass(t *testing.T) {
	inst := callExpect(t, goipyObject.BoolOf(true))
	if err := callMatcher(t, inst, "to_be_true"); err != nil {
		t.Errorf("to_be_true should pass: %v", err)
	}
}

func TestExpectToBeTrueFail(t *testing.T) {
	inst := callExpect(t, goipyObject.BoolOf(false))
	if err := callMatcher(t, inst, "to_be_true"); err == nil {
		t.Error("to_be_true should fail for false")
	}
}

func TestExpectToBeNonePass(t *testing.T) {
	inst := callExpect(t, goipyObject.None)
	if err := callMatcher(t, inst, "to_be_none"); err != nil {
		t.Errorf("to_be_none should pass: %v", err)
	}
}

func TestExpectToBeNoneFail(t *testing.T) {
	inst := callExpect(t, goipyObject.NewInt(0))
	if err := callMatcher(t, inst, "to_be_none"); err == nil {
		t.Error("to_be_none should fail for 0")
	}
}

func TestExpectToContainStr(t *testing.T) {
	inst := callExpect(t, &goipyObject.Str{V: "hello world"})
	if err := callMatcher(t, inst, "to_contain", &goipyObject.Str{V: "world"}); err != nil {
		t.Errorf("to_contain should pass: %v", err)
	}
}

func TestExpectToContainStrFail(t *testing.T) {
	inst := callExpect(t, &goipyObject.Str{V: "hello"})
	if err := callMatcher(t, inst, "to_contain", &goipyObject.Str{V: "world"}); err == nil {
		t.Error("to_contain should fail for missing substring")
	}
}

func TestExpectToContainList(t *testing.T) {
	lst := &goipyObject.List{V: []goipyObject.Object{
		goipyObject.NewInt(1), goipyObject.NewInt(2), goipyObject.NewInt(3),
	}}
	inst := callExpect(t, lst)
	if err := callMatcher(t, inst, "to_contain", goipyObject.NewInt(2)); err != nil {
		t.Errorf("to_contain should pass for list: %v", err)
	}
}

func TestExpectToHaveLength(t *testing.T) {
	inst := callExpect(t, &goipyObject.Str{V: "hello"})
	if err := callMatcher(t, inst, "to_have_length", goipyObject.NewInt(5)); err != nil {
		t.Errorf("to_have_length should pass: %v", err)
	}
}

func TestExpectToHaveLengthFail(t *testing.T) {
	inst := callExpect(t, &goipyObject.Str{V: "hello"})
	if err := callMatcher(t, inst, "to_have_length", goipyObject.NewInt(3)); err == nil {
		t.Error("to_have_length should fail for wrong length")
	}
}

func TestExpectGreaterLess(t *testing.T) {
	inst := callExpect(t, goipyObject.NewInt(10))
	if err := callMatcher(t, inst, "to_be_greater_than", goipyObject.NewInt(5)); err != nil {
		t.Errorf("to_be_greater_than should pass: %v", err)
	}
	if err := callMatcher(t, inst, "to_be_less_than", goipyObject.NewInt(20)); err != nil {
		t.Errorf("to_be_less_than should pass: %v", err)
	}
}

func TestExpectNotWrapper(t *testing.T) {
	inst := callExpect(t, goipyObject.NewInt(42))
	notInst, ok := inst.Dict.GetStr("not_")
	if !ok {
		t.Fatal("not_ wrapper missing")
	}
	neg := notInst.(*goipyObject.Instance)
	fn, _ := neg.Dict.GetStr("to_equal")
	if _, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{goipyObject.NewInt(99)}, nil); err != nil {
		t.Errorf("not_.to_equal should pass when values differ: %v", err)
	}
}

func TestExpectNotEqualShouldFail(t *testing.T) {
	inst := callExpect(t, goipyObject.NewInt(42))
	notInst, _ := inst.Dict.GetStr("not_")
	neg := notInst.(*goipyObject.Instance)
	fn, _ := neg.Dict.GetStr("to_equal")
	if _, err := fn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{goipyObject.NewInt(42)}, nil); err == nil {
		t.Error("not_.to_equal should fail when values are equal")
	}
}

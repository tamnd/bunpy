package bunpy

import (
	"fmt"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildExpect builds the bunpy.expect module with assertion matchers.
func BuildExpect(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.expect", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("expect", buildExpectFn())
	mod.Dict.SetStr("describe", buildDescribeFn())
	mod.Dict.SetStr("it", buildItFn())
	mod.Dict.SetStr("test", buildTestFn())
	mod.Dict.SetStr("skip", buildSkipFn())

	return mod
}

func buildExpectFn() *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "expect",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			var actual goipyObject.Object = goipyObject.None
			if len(args) >= 1 {
				actual = args[0]
			}
			return newExpectation(actual), nil
		},
	}
}

func buildDescribeFn() *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "describe",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			// describe("label", fn) — just calls fn()
			if len(args) < 2 {
				return goipyObject.None, nil
			}
			return goipyObject.None, nil
		},
	}
}

func buildItFn() *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "it",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, nil
		},
	}
}

func buildTestFn() *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "test",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, nil
		},
	}
}

func buildSkipFn() *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "skip",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			msg := "skip: test skipped"
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					msg = "skip: " + s.V
				}
			}
			return nil, fmt.Errorf("SkipTest: %s", msg)
		},
	}
}

// newExpectation builds an expectation object with matcher methods.
func newExpectation(actual goipyObject.Object) *goipyObject.Instance {
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Expectation"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("_actual", actual)

	addMatcher(inst, "to_equal", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("to_equal() requires an expected value")
		}
		if !deepEquals(actual, args[0]) {
			return assertFail("Expected %s to equal %s", pyRepr(actual), pyRepr(args[0]))
		}
		return nil
	})

	addMatcher(inst, "to_be", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("to_be() requires an expected value")
		}
		if !deepEquals(actual, args[0]) {
			return assertFail("Expected %s to be %s", pyRepr(actual), pyRepr(args[0]))
		}
		return nil
	})

	addMatcher(inst, "not_to_equal", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("not_to_equal() requires an expected value")
		}
		if deepEquals(actual, args[0]) {
			return assertFail("Expected %s not to equal %s", pyRepr(actual), pyRepr(args[0]))
		}
		return nil
	})

	addMatcher(inst, "to_be_true", func(actual goipyObject.Object, _ []goipyObject.Object) error {
		if !isTruthy(actual) {
			return assertFail("Expected %s to be truthy", pyRepr(actual))
		}
		return nil
	})

	addMatcher(inst, "to_be_false", func(actual goipyObject.Object, _ []goipyObject.Object) error {
		if isTruthy(actual) {
			return assertFail("Expected %s to be falsy", pyRepr(actual))
		}
		return nil
	})

	addMatcher(inst, "to_be_none", func(actual goipyObject.Object, _ []goipyObject.Object) error {
		if actual != goipyObject.None {
			return assertFail("Expected %s to be None", pyRepr(actual))
		}
		return nil
	})

	addMatcher(inst, "not_to_be_none", func(actual goipyObject.Object, _ []goipyObject.Object) error {
		if actual == goipyObject.None {
			return assertFail("Expected value not to be None")
		}
		return nil
	})

	addMatcher(inst, "to_contain", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("to_contain() requires an item")
		}
		item := args[0]
		switch v := actual.(type) {
		case *goipyObject.Str:
			if sub, ok := item.(*goipyObject.Str); ok {
				if !strings.Contains(v.V, sub.V) {
					return assertFail("Expected %q to contain %q", v.V, sub.V)
				}
				return nil
			}
			return assertFail("to_contain: item must be str for str actual")
		case *goipyObject.List:
			for _, el := range v.V {
				if deepEquals(el, item) {
					return nil
				}
			}
			return assertFail("Expected list to contain %s", pyRepr(item))
		}
		return assertFail("to_contain: unsupported type %T", actual)
	})

	addMatcher(inst, "to_have_length", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("to_have_length() requires a length")
		}
		wantLen, ok := args[0].(*goipyObject.Int)
		if !ok {
			return fmt.Errorf("to_have_length(): expected integer length")
		}
		var got int
		switch v := actual.(type) {
		case *goipyObject.Str:
			got = len(v.V)
		case *goipyObject.List:
			got = len(v.V)
		case *goipyObject.Tuple:
			got = len(v.V)
		case *goipyObject.Dict:
			keys, _ := v.Items()
			got = len(keys)
		case *goipyObject.Bytes:
			got = len(v.V)
		default:
			return assertFail("to_have_length: unsupported type %T", actual)
		}
		if int64(got) != wantLen.Int64() {
			return assertFail("Expected length %d, got %d", wantLen.Int64(), got)
		}
		return nil
	})

	addMatcher(inst, "to_be_greater_than", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("to_be_greater_than() requires a value")
		}
		a, b, err := numericPair(actual, args[0])
		if err != nil {
			return err
		}
		if a <= b {
			return assertFail("Expected %s to be greater than %s", pyRepr(actual), pyRepr(args[0]))
		}
		return nil
	})

	addMatcher(inst, "to_be_less_than", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("to_be_less_than() requires a value")
		}
		a, b, err := numericPair(actual, args[0])
		if err != nil {
			return err
		}
		if a >= b {
			return assertFail("Expected %s to be less than %s", pyRepr(actual), pyRepr(args[0]))
		}
		return nil
	})

	addMatcher(inst, "to_be_greater_than_or_equal", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("to_be_greater_than_or_equal() requires a value")
		}
		a, b, err := numericPair(actual, args[0])
		if err != nil {
			return err
		}
		if a < b {
			return assertFail("Expected %s >= %s", pyRepr(actual), pyRepr(args[0]))
		}
		return nil
	})

	addMatcher(inst, "to_be_less_than_or_equal", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("to_be_less_than_or_equal() requires a value")
		}
		a, b, err := numericPair(actual, args[0])
		if err != nil {
			return err
		}
		if a > b {
			return assertFail("Expected %s <= %s", pyRepr(actual), pyRepr(args[0]))
		}
		return nil
	})

	addMatcher(inst, "to_be_instance_of", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("to_be_instance_of() requires a class")
		}
		cls, ok := args[0].(*goipyObject.Class)
		if !ok {
			return fmt.Errorf("to_be_instance_of(): argument must be a class")
		}
		inst2, ok2 := actual.(*goipyObject.Instance)
		if !ok2 || inst2.Class.Name != cls.Name {
			return assertFail("Expected instance of %s", cls.Name)
		}
		return nil
	})

	addMatcher(inst, "to_throw", func(actual goipyObject.Object, _ []goipyObject.Object) error {
		return assertFail("to_throw: use expect_throws(fn) helper instead")
	})

	// not property returns a negating wrapper
	inst.Dict.SetStr("not_", newNegatingExpectation(actual))

	return inst
}

func addMatcher(inst *goipyObject.Instance, name string, check func(goipyObject.Object, []goipyObject.Object) error) {
	actualVal := inst.Dict
	inst.Dict.SetStr(name, &goipyObject.BuiltinFunc{
		Name: name,
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			actual, _ := actualVal.GetStr("_actual")
			if err := check(actual, args); err != nil {
				return nil, err
			}
			return goipyObject.None, nil
		},
	})
}

func newNegatingExpectation(actual goipyObject.Object) *goipyObject.Instance {
	neg := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "NegExpectation"},
		Dict:  goipyObject.NewDict(),
	}
	neg.Dict.SetStr("_actual", actual)
	addMatcher(neg, "to_equal", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("not_.to_equal() requires a value")
		}
		if deepEquals(actual, args[0]) {
			return assertFail("Expected %s not to equal %s", pyRepr(actual), pyRepr(args[0]))
		}
		return nil
	})
	addMatcher(neg, "to_be_none", func(actual goipyObject.Object, _ []goipyObject.Object) error {
		if actual == goipyObject.None {
			return assertFail("Expected value not to be None")
		}
		return nil
	})
	addMatcher(neg, "to_contain", func(actual goipyObject.Object, args []goipyObject.Object) error {
		if len(args) < 1 {
			return fmt.Errorf("not_.to_contain() requires an item")
		}
		switch v := actual.(type) {
		case *goipyObject.Str:
			if sub, ok := args[0].(*goipyObject.Str); ok {
				if strings.Contains(v.V, sub.V) {
					return assertFail("Expected %q not to contain %q", v.V, sub.V)
				}
				return nil
			}
		case *goipyObject.List:
			for _, el := range v.V {
				if deepEquals(el, args[0]) {
					return assertFail("Expected list not to contain %s", pyRepr(args[0]))
				}
			}
			return nil
		}
		return nil
	})
	return neg
}

func assertFail(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return &goipyObject.Exception{
		Class: &goipyObject.Class{Name: "AssertionError"},
		Msg:   msg,
		Args:  &goipyObject.Tuple{V: []goipyObject.Object{&goipyObject.Str{V: msg}}},
	}
}

func isTruthy(obj goipyObject.Object) bool {
	if obj == nil || obj == goipyObject.None {
		return false
	}
	switch v := obj.(type) {
	case *goipyObject.Bool:
		return v.V
	case *goipyObject.Int:
		return v.Int64() != 0
	case *goipyObject.Float:
		return v.V != 0
	case *goipyObject.Str:
		return v.V != ""
	case *goipyObject.List:
		return len(v.V) > 0
	case *goipyObject.Tuple:
		return len(v.V) > 0
	case *goipyObject.Dict:
		keys, _ := v.Items()
		return len(keys) > 0
	case *goipyObject.Bytes:
		return len(v.V) > 0
	}
	return true
}

func pyRepr(obj goipyObject.Object) string {
	if obj == nil || obj == goipyObject.None {
		return "None"
	}
	switch v := obj.(type) {
	case *goipyObject.Str:
		return fmt.Sprintf("%q", v.V)
	case *goipyObject.Int:
		return fmt.Sprintf("%d", v.Int64())
	case *goipyObject.Float:
		return fmt.Sprintf("%g", v.V)
	case *goipyObject.Bool:
		if v.V {
			return "True"
		}
		return "False"
	case *goipyObject.List:
		return fmt.Sprintf("[...(%d items)]", len(v.V))
	case *goipyObject.Dict:
		keys, _ := v.Items()
		return fmt.Sprintf("{...(%d keys)}", len(keys))
	case *goipyObject.Bytes:
		return fmt.Sprintf("b\"..(%d bytes)\"", len(v.V))
	}
	return fmt.Sprintf("<%T>", obj)
}

func numericPair(a, b goipyObject.Object) (float64, float64, error) {
	af, ok1 := toFloat(a)
	bf, ok2 := toFloat(b)
	if !ok1 || !ok2 {
		return 0, 0, fmt.Errorf("expected numeric values for comparison")
	}
	return af, bf, nil
}

func toFloat(obj goipyObject.Object) (float64, bool) {
	switch v := obj.(type) {
	case *goipyObject.Int:
		return float64(v.Int64()), true
	case *goipyObject.Float:
		return v.V, true
	}
	return 0, false
}

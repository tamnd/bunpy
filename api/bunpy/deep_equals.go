package bunpy

import (
	"fmt"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildDeepEquals(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.deep_equals", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("deep_equals", &goipyObject.BuiltinFunc{
		Name: "deep_equals",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("deep_equals() requires two arguments")
			}
			return goipyObject.BoolOf(deepEquals(args[0], args[1])), nil
		},
	})

	return mod
}

func deepEquals(a, b goipyObject.Object) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// None
	if a == goipyObject.None && b == goipyObject.None {
		return true
	}
	if a == goipyObject.None || b == goipyObject.None {
		return false
	}

	switch av := a.(type) {
	case *goipyObject.Int:
		if bv, ok := b.(*goipyObject.Int); ok {
			return av.Int64() == bv.Int64()
		}
		if bv, ok := b.(*goipyObject.Float); ok {
			return float64(av.Int64()) == bv.V
		}
		return false

	case *goipyObject.Float:
		if bv, ok := b.(*goipyObject.Float); ok {
			return av.V == bv.V
		}
		if bv, ok := b.(*goipyObject.Int); ok {
			return av.V == float64(bv.Int64())
		}
		return false

	case *goipyObject.Str:
		if bv, ok := b.(*goipyObject.Str); ok {
			return av.V == bv.V
		}
		return false

	case *goipyObject.Bytes:
		if bv, ok := b.(*goipyObject.Bytes); ok {
			if len(av.V) != len(bv.V) {
				return false
			}
			for i := range av.V {
				if av.V[i] != bv.V[i] {
					return false
				}
			}
			return true
		}
		return false

	case *goipyObject.Bool:
		if bv, ok := b.(*goipyObject.Bool); ok {
			return av.V == bv.V
		}
		return false

	case *goipyObject.List:
		bv, ok := b.(*goipyObject.List)
		if !ok {
			return false
		}
		if len(av.V) != len(bv.V) {
			return false
		}
		for i := range av.V {
			if !deepEquals(av.V[i], bv.V[i]) {
				return false
			}
		}
		return true

	case *goipyObject.Tuple:
		bv, ok := b.(*goipyObject.Tuple)
		if !ok {
			return false
		}
		if len(av.V) != len(bv.V) {
			return false
		}
		for i := range av.V {
			if !deepEquals(av.V[i], bv.V[i]) {
				return false
			}
		}
		return true

	case *goipyObject.Dict:
		bv, ok := b.(*goipyObject.Dict)
		if !ok {
			return false
		}
		akeys, avals := av.Items()
		bkeys, bvals := bv.Items()
		if len(akeys) != len(bkeys) {
			return false
		}
		// build map from bv for lookup
		bmap := make(map[string]goipyObject.Object, len(bkeys))
		for i, k := range bkeys {
			if ks, ok2 := k.(*goipyObject.Str); ok2 {
				bmap[ks.V] = bvals[i]
			}
		}
		for i, k := range akeys {
			ks, ok2 := k.(*goipyObject.Str)
			if !ok2 {
				return false
			}
			bval, exists := bmap[ks.V]
			if !exists {
				return false
			}
			if !deepEquals(avals[i], bval) {
				return false
			}
		}
		return true
	}

	// fallback: pointer equality
	return a == b
}

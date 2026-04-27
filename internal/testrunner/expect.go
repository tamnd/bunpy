package testrunner

import (
	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
	goipyVM "github.com/tamnd/goipy/vm"
)

// InjectExpect adds the expect() global and the bunpy.expect module
// to the interpreter so test files can use them without any import.
func InjectExpect(i *goipyVM.Interp) {
	mod := bunpyAPI.BuildExpect(i)
	if fn, ok := mod.Dict.GetStr("expect"); ok {
		i.Builtins.SetStr("expect", fn)
	}
	if fn, ok := mod.Dict.GetStr("describe"); ok {
		i.Builtins.SetStr("describe", fn)
	}
	if fn, ok := mod.Dict.GetStr("it"); ok {
		i.Builtins.SetStr("it", fn)
	}
	if fn, ok := mod.Dict.GetStr("test"); ok {
		i.Builtins.SetStr("test", fn)
	}
	if fn, ok := mod.Dict.GetStr("skip"); ok {
		i.Builtins.SetStr("skip", fn)
	}
}

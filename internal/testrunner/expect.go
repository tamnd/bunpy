package testrunner

import (
	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
	goipyVM "github.com/tamnd/goipy/vm"
)

// InjectExpect adds the expect() global and the bunpy.expect module
// to the interpreter so test files can use them without any import.
func InjectExpect(i *goipyVM.Interp) {
	mod := bunpyAPI.BuildExpect(i)
	for _, name := range []string{"expect", "describe", "it", "test", "skip"} {
		if fn, ok := mod.Dict.GetStr(name); ok {
			i.Builtins.SetStr(name, fn)
		}
	}
	// Inject mock() and spy_on() as globals too.
	mockMod := bunpyAPI.BuildMock(i)
	for _, name := range []string{"mock", "spy_on"} {
		if fn, ok := mockMod.Dict.GetStr(name); ok {
			i.Builtins.SetStr(name, fn)
		}
	}
}

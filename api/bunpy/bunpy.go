// Package bunpy exposes the bunpy.* built-in API modules to the goipy VM.
//
// Each sub-module is a function that builds an *object.Module from a
// *vm.Interp. The map returned by Modules() is assigned to
// interp.NativeModules before the interpreter runs user code.
//
// Sub-module layout:
//
//	bunpy            -- top-level namespace (base64, gzip, version)
//	bunpy.base64     -- bunpy.base64.encode / .decode
//	bunpy.gzip       -- bunpy.gzip.compress / .decompress
package bunpy

import (
	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// Version is baked in by the bunpy build pipeline.
const Version = "0.3.0"

// Modules returns the NativeModules map for the current v0.3.0 surface:
// bunpy, bunpy.base64, bunpy.gzip.
// Later rungs extend this map by adding more entries.
func Modules() map[string]func(*goipyVM.Interp) *goipyObject.Module {
	return map[string]func(*goipyVM.Interp) *goipyObject.Module{
		"bunpy":        BuildBunpy,
		"bunpy.base64": BuildBase64,
		"bunpy.gzip":   BuildGzip,
	}
}

// BuildBunpy builds the top-level `bunpy` module. It contains sub-module
// references and the version string.
func BuildBunpy(i *goipyVM.Interp) *goipyObject.Module {
	m := &goipyObject.Module{Name: "bunpy", Dict: goipyObject.NewDict()}
	m.Dict.SetStr("__version__", &goipyObject.Str{V: Version})
	m.Dict.SetStr("__name__", &goipyObject.Str{V: "bunpy"})

	// Attach sub-modules so `import bunpy; bunpy.base64.encode(...)` works.
	m.Dict.SetStr("base64", BuildBase64(i))
	m.Dict.SetStr("gzip", BuildGzip(i))

	return m
}

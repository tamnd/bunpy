package bunpy

import (
	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNode builds the top-level bunpy.node namespace module.
// It aggregates all node.* sub-modules as attributes so Python code can do:
//
//	import bunpy.node as node
//	node.fs.readFile(...)
func BuildNode(i *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("fs", BuildNodeFS(i))
	mod.Dict.SetStr("path", BuildNodePath(i))
	mod.Dict.SetStr("os", BuildNodeOS(i))
	mod.Dict.SetStr("http", BuildNodeHTTP(i))
	mod.Dict.SetStr("https", BuildNodeHTTPS(i))
	mod.Dict.SetStr("net", BuildNodeNet(i))
	mod.Dict.SetStr("tls", BuildNodeTLS(i))
	mod.Dict.SetStr("crypto", BuildNodeCrypto(i))
	mod.Dict.SetStr("stream", BuildNodeStream(i))
	mod.Dict.SetStr("zlib", BuildNodeZlib(i))
	mod.Dict.SetStr("worker_threads", BuildNodeWorkerThreads(i))

	return mod
}

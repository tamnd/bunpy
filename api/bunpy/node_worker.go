package bunpy

import (
	"sync"
	"sync/atomic"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNodeWorkerThreads builds the bunpy.node.worker_threads module.
func BuildNodeWorkerThreads(i *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.worker_threads", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("isMainThread", goipyObject.BoolOf(true))

	mod.Dict.SetStr("threadId", goipyObject.NewInt(0))

	mod.Dict.SetStr("Worker", &goipyObject.BuiltinFunc{
		Name: "Worker",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			var fn goipyObject.Object
			if len(args) >= 1 {
				fn = args[0]
			}
			return newNodeWorkerInstance(i, fn), nil
		},
	})

	mod.Dict.SetStr("MessageChannel", &goipyObject.BuiltinFunc{
		Name: "MessageChannel",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newMessageChannelInstance(), nil
		},
	})

	mod.Dict.SetStr("receiveMessageOnPort", &goipyObject.BuiltinFunc{
		Name: "receiveMessageOnPort",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.None, nil
			}
			port, ok := args[0].(*goipyObject.Instance)
			if !ok {
				return goipyObject.None, nil
			}
			if recv, ok := port.Dict.GetStr("receiveSync"); ok {
				if fn, ok := recv.(*goipyObject.BuiltinFunc); ok {
					return fn.Call(nil, nil, nil)
				}
			}
			return goipyObject.None, nil
		},
	})

	return mod
}

func newNodeWorkerInstance(interp *goipyVM.Interp, fn goipyObject.Object) *goipyObject.Instance {
	type state struct {
		mu       sync.Mutex
		handlers map[string][]goipyObject.Object
		done     atomic.Bool
		msgs     []goipyObject.Object
	}
	st := &state{handlers: map[string][]goipyObject.Object{}}
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Worker"},
		Dict:  goipyObject.NewDict(),
	}

	if fn != nil {
		go func() {
			interp.CallObject(fn, nil, nil)
			st.done.Store(true)
			st.mu.Lock()
			exitHandlers := append([]goipyObject.Object(nil), st.handlers["exit"]...)
			st.mu.Unlock()
			for _, h := range exitHandlers {
				interp.CallObject(h, []goipyObject.Object{goipyObject.NewInt(0)}, nil)
			}
		}()
	}

	inst.Dict.SetStr("on", &goipyObject.BuiltinFunc{
		Name: "on",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return inst, nil
			}
			evt, ok := args[0].(*goipyObject.Str)
			if !ok {
				return inst, nil
			}
			st.mu.Lock()
			st.handlers[evt.V] = append(st.handlers[evt.V], args[1])
			st.mu.Unlock()
			return inst, nil
		},
	})

	inst.Dict.SetStr("postMessage", &goipyObject.BuiltinFunc{
		Name: "postMessage",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.None, nil
			}
			msg := args[0]
			st.mu.Lock()
			msgHandlers := append([]goipyObject.Object(nil), st.handlers["message"]...)
			st.mu.Unlock()
			for _, h := range msgHandlers {
				interp.CallObject(h, []goipyObject.Object{msg}, nil)
			}
			return goipyObject.None, nil
		},
	})

	inst.Dict.SetStr("terminate", &goipyObject.BuiltinFunc{
		Name: "terminate",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			st.done.Store(true)
			return goipyObject.None, nil
		},
	})

	return inst
}

func newMessageChannelInstance() *goipyObject.Instance {
	ch := make(chan goipyObject.Object, 64)

	makePort := func(send, recv chan goipyObject.Object) *goipyObject.Instance {
		port := &goipyObject.Instance{
			Class: &goipyObject.Class{Name: "MessagePort"},
			Dict:  goipyObject.NewDict(),
		}
		port.Dict.SetStr("postMessage", &goipyObject.BuiltinFunc{
			Name: "postMessage",
			Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
				if len(args) >= 1 {
					send <- args[0]
				}
				return goipyObject.None, nil
			},
		})
		port.Dict.SetStr("receiveSync", &goipyObject.BuiltinFunc{
			Name: "receiveSync",
			Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
				select {
				case msg := <-recv:
					d := goipyObject.NewDict()
					d.SetStr("message", msg)
					return &goipyObject.Instance{Class: &goipyObject.Class{Name: "object"}, Dict: d}, nil
				default:
					return goipyObject.None, nil
				}
			},
		})
		return port
	}

	ch2 := make(chan goipyObject.Object, 64)

	port1 := makePort(ch, ch2)
	port2 := makePort(ch2, ch)

	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "MessageChannel"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("port1", port1)
	inst.Dict.SetStr("port2", port2)
	return inst
}

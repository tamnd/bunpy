package bunpy

import (
	"fmt"
	"sync"
	"sync/atomic"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildWorker(i *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.Worker", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("Worker", &goipyObject.BuiltinFunc{
		Name: "Worker",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("Worker() requires a callable (the worker function)")
			}
			fn := args[0]
			return newWorkerInstance(i, fn), nil
		},
	})

	return mod
}

type workerState struct {
	interp     *goipyVM.Interp
	fn         goipyObject.Object
	inbox      chan goipyObject.Object
	listeners  map[string][]goipyObject.Object
	listenerMu sync.Mutex
	terminated atomic.Bool
	wg         sync.WaitGroup
}

func newWorkerInstance(i *goipyVM.Interp, fn goipyObject.Object) *goipyObject.Instance {
	state := &workerState{
		interp:    i,
		fn:        fn,
		inbox:     make(chan goipyObject.Object, 64),
		listeners: make(map[string][]goipyObject.Object),
	}

	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Worker"},
		Dict:  goipyObject.NewDict(),
	}

	inst.Dict.SetStr("post_message", &goipyObject.BuiltinFunc{
		Name: "post_message",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if state.terminated.Load() {
				return nil, fmt.Errorf("Worker: cannot post to terminated worker")
			}
			var msg goipyObject.Object = goipyObject.None
			if len(args) >= 1 {
				msg = args[0]
			}
			state.inbox <- msg
			return goipyObject.None, nil
		},
	})

	inst.Dict.SetStr("on", &goipyObject.BuiltinFunc{
		Name: "on",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("Worker.on() requires event name and listener")
			}
			event, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("Worker.on(): event name must be str")
			}
			state.listenerMu.Lock()
			state.listeners[event.V] = append(state.listeners[event.V], args[1])
			state.listenerMu.Unlock()
			return inst, nil
		},
	})

	inst.Dict.SetStr("terminate", &goipyObject.BuiltinFunc{
		Name: "terminate",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if state.terminated.CompareAndSwap(false, true) {
				close(state.inbox)
			}
			return goipyObject.None, nil
		},
	})

	// Start the worker goroutine
	state.wg.Add(1)
	go func() {
		defer state.wg.Done()
		// Build a postMessage function for use inside the worker
		postToMain := &goipyObject.BuiltinFunc{
			Name: "postMessage",
			Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
				var data goipyObject.Object = goipyObject.None
				if len(args) >= 1 {
					data = args[0]
				}
				state.listenerMu.Lock()
				listeners := append([]goipyObject.Object{}, state.listeners["message"]...)
				state.listenerMu.Unlock()
				for _, l := range listeners {
					i.CallObject(l, []goipyObject.Object{data}, nil)
				}
				return goipyObject.None, nil
			},
		}
		// Call the worker function with postMessage
		i.CallObject(fn, []goipyObject.Object{postToMain}, nil)

		// Then drain inbox
		for msg := range state.inbox {
			state.listenerMu.Lock()
			listeners := append([]goipyObject.Object{}, state.listeners["message"]...)
			state.listenerMu.Unlock()
			for _, l := range listeners {
				i.CallObject(l, []goipyObject.Object{msg}, nil)
			}
		}

		// Fire "exit" listeners
		state.listenerMu.Lock()
		exitListeners := append([]goipyObject.Object{}, state.listeners["exit"]...)
		state.listenerMu.Unlock()
		for _, l := range exitListeners {
			i.CallObject(l, nil, nil)
		}
	}()

	return inst
}

package bunpy

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildAsyncio builds the bunpy.asyncio module.
// Functions map to goroutines since gocopy does not yet support async/await.
func BuildAsyncio(i *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.asyncio", Dict: goipyObject.NewDict()}

	// run(fn) — call fn() in a goroutine, block until result.
	mod.Dict.SetStr("run", &goipyObject.BuiltinFunc{
		Name: "run",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("asyncio.run() requires a callable")
			}
			type result struct {
				v   goipyObject.Object
				err error
			}
			ch := make(chan result, 1)
			go func() {
				v, err := i.Call(args[0], nil, nil)
				ch <- result{v, err}
			}()
			r := <-ch
			return r.v, r.err
		},
	})

	// gather(*fns) — run fns concurrently, return list of results in call order.
	mod.Dict.SetStr("gather", &goipyObject.BuiltinFunc{
		Name: "gather",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			results := make([]goipyObject.Object, len(args))
			errs := make([]error, len(args))
			var wg sync.WaitGroup
			for idx, fn := range args {
				wg.Add(1)
				go func(j int, f goipyObject.Object) {
					defer wg.Done()
					v, err := i.Call(f, nil, nil)
					results[j] = v
					errs[j] = err
				}(idx, fn)
			}
			wg.Wait()
			for _, err := range errs {
				if err != nil {
					return nil, err
				}
			}
			return &goipyObject.List{V: results}, nil
		},
	})

	// sleep(seconds) — sleep for n seconds.
	mod.Dict.SetStr("sleep", &goipyObject.BuiltinFunc{
		Name: "sleep",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			dur := 0.0
			if len(args) >= 1 {
				switch v := args[0].(type) {
				case *goipyObject.Int:
					dur = float64(v.Int64())
				case *goipyObject.Float:
					dur = v.V
				}
			}
			if dur > 0 {
				time.Sleep(time.Duration(dur * float64(time.Second)))
			}
			return goipyObject.None, nil
		},
	})

	// create_task(fn) — launch fn() in background goroutine; return task handle.
	mod.Dict.SetStr("create_task", &goipyObject.BuiltinFunc{
		Name: "create_task",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("asyncio.create_task() requires a callable")
			}
			return newTaskHandle(i, args[0]), nil
		},
	})

	return mod
}

type taskState struct {
	done   atomic.Bool
	result goipyObject.Object
	err    error
}

func newTaskHandle(i *goipyVM.Interp, fn goipyObject.Object) *goipyObject.Instance {
	state := &taskState{}
	var mu sync.Mutex
	var ready = make(chan struct{})

	go func() {
		v, err := i.Call(fn, nil, nil)
		mu.Lock()
		state.result = v
		state.err = err
		mu.Unlock()
		state.done.Store(true)
		close(ready)
	}()

	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Task"},
		Dict:  goipyObject.NewDict(),
	}

	inst.Dict.SetStr("done", &goipyObject.BuiltinFunc{
		Name: "done",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.BoolOf(state.done.Load()), nil
		},
	})

	inst.Dict.SetStr("result", &goipyObject.BuiltinFunc{
		Name: "result",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			<-ready
			mu.Lock()
			defer mu.Unlock()
			if state.err != nil {
				return nil, state.err
			}
			if state.result == nil {
				return goipyObject.None, nil
			}
			return state.result, nil
		},
	})

	return inst
}

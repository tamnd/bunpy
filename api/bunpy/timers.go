package bunpy

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

var timerIDCounter atomic.Int64

type timerHandle struct {
	id     int64
	cancel chan struct{}
}

var (
	timersMu sync.Mutex
	timers   = map[int64]*timerHandle{}
)

// InjectTimerGlobals adds setTimeout, clearTimeout, setInterval, clearInterval
// to interp.Builtins so scripts can use them without any import.
func InjectTimerGlobals(i *goipyVM.Interp) {
	i.Builtins.SetStr("setTimeout", buildSetTimeout(i))
	i.Builtins.SetStr("clearTimeout", buildClearTimer())
	i.Builtins.SetStr("setInterval", buildSetInterval(i))
	i.Builtins.SetStr("clearInterval", buildClearTimer())
}

func buildSetTimeout(i *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "setTimeout",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("setTimeout() requires a callback")
			}
			cb := args[0]
			delay := 0.0
			if len(args) >= 2 {
				delay = toMillis(args[1])
			}
			var cbArgs []goipyObject.Object
			if len(args) >= 3 {
				if lst, ok := args[2].(*goipyObject.List); ok {
					cbArgs = lst.V
				}
			}
			id := timerIDCounter.Add(1)
			cancel := make(chan struct{})
			h := &timerHandle{id: id, cancel: cancel}
			timersMu.Lock()
			timers[id] = h
			timersMu.Unlock()
			go func() {
				d := time.Duration(delay * float64(time.Millisecond))
				select {
				case <-time.After(d):
					timersMu.Lock()
					delete(timers, id)
					timersMu.Unlock()
					i.Call(cb, cbArgs, nil)
				case <-cancel:
				}
			}()
			return goipyObject.NewInt(id), nil
		},
	}
}

func buildSetInterval(i *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "setInterval",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("setInterval() requires a callback")
			}
			cb := args[0]
			delay := 0.0
			if len(args) >= 2 {
				delay = toMillis(args[1])
			}
			id := timerIDCounter.Add(1)
			cancel := make(chan struct{})
			h := &timerHandle{id: id, cancel: cancel}
			timersMu.Lock()
			timers[id] = h
			timersMu.Unlock()
			go func() {
				d := time.Duration(delay * float64(time.Millisecond))
				ticker := time.NewTicker(d)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						i.Call(cb, nil, nil)
					case <-cancel:
						timersMu.Lock()
						delete(timers, id)
						timersMu.Unlock()
						return
					}
				}
			}()
			return goipyObject.NewInt(id), nil
		},
	}
}

func buildClearTimer() *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "clearTimer",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.None, nil
			}
			id := int64(0)
			if iv, ok := args[0].(*goipyObject.Int); ok {
				id = iv.Int64()
			}
			timersMu.Lock()
			if h, ok := timers[id]; ok {
				close(h.cancel)
				delete(timers, id)
			}
			timersMu.Unlock()
			return goipyObject.None, nil
		},
	}
}

func toMillis(obj goipyObject.Object) float64 {
	switch v := obj.(type) {
	case *goipyObject.Int:
		return float64(v.Int64())
	case *goipyObject.Float:
		return v.V
	}
	return 0
}

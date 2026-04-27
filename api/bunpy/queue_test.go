package bunpy_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func newQueue(t *testing.T, workers int) *goipyObject.Instance {
	t.Helper()
	i := serveInterp(t)
	mod := bunpyAPI.BuildQueue(i)
	newFn, _ := mod.Dict.GetStr("new")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("workers", goipyObject.NewInt(int64(workers)))
	result, err := newFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	return result.(*goipyObject.Instance)
}

func queueCall(t *testing.T, inst *goipyObject.Instance, method string, args []goipyObject.Object, kwargs *goipyObject.Dict) goipyObject.Object {
	t.Helper()
	fn, ok := inst.Dict.GetStr(method)
	if !ok {
		t.Fatalf("queue missing method %q", method)
	}
	result, err := fn.(*goipyObject.BuiltinFunc).Call(nil, args, kwargs)
	if err != nil {
		t.Fatalf("queue.%s() error: %v", method, err)
	}
	return result
}

func TestQueueHasMethods(t *testing.T) {
	q := newQueue(t, 2)
	for _, m := range []string{"enqueue", "handler", "wait", "stop"} {
		if _, ok := q.Dict.GetStr(m); !ok {
			t.Fatalf("queue missing method %q", m)
		}
	}
	queueCall(t, q, "stop", nil, nil)
}

func TestQueueJobProcessed(t *testing.T) {
	var called atomic.Int32

	i := serveInterp(t)
	mod := bunpyAPI.BuildQueue(i)
	newFn, _ := mod.Dict.GetStr("new")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("workers", goipyObject.NewInt(2))
	result, _ := newFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)
	q := result.(*goipyObject.Instance)

	// register handler
	handlerFn, _ := q.Dict.GetStr("handler")
	decorator, _ := handlerFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "ping"},
	}, nil)
	handler := &goipyObject.BuiltinFunc{
		Name: "ping_handler",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			called.Add(1)
			return goipyObject.None, nil
		},
	}
	decorator.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{handler}, nil)

	// enqueue and wait
	enqueueFn, _ := q.Dict.GetStr("enqueue")
	enqueueFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "ping"},
		goipyObject.NewDict(),
	}, nil)

	queueCall(t, q, "wait", nil, nil)
	queueCall(t, q, "stop", nil, nil)

	if called.Load() != 1 {
		t.Fatalf("expected handler called once, got %d", called.Load())
	}
}

func TestQueueWaitBlocks(t *testing.T) {
	var done atomic.Bool

	i := serveInterp(t)
	mod := bunpyAPI.BuildQueue(i)
	newFn, _ := mod.Dict.GetStr("new")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("workers", goipyObject.NewInt(1))
	result, _ := newFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)
	q := result.(*goipyObject.Instance)

	handlerFn, _ := q.Dict.GetStr("handler")
	decorator, _ := handlerFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "slow"},
	}, nil)
	handler := &goipyObject.BuiltinFunc{
		Name: "slow_handler",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			time.Sleep(20 * time.Millisecond)
			done.Store(true)
			return goipyObject.None, nil
		},
	}
	decorator.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{handler}, nil)

	enqueueFn, _ := q.Dict.GetStr("enqueue")
	enqueueFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "slow"},
	}, nil)

	queueCall(t, q, "wait", nil, nil)
	if !done.Load() {
		t.Fatal("wait() returned before job finished")
	}
	queueCall(t, q, "stop", nil, nil)
}

func TestQueueStopPreventsEnqueue(t *testing.T) {
	q := newQueue(t, 1)
	queueCall(t, q, "stop", nil, nil)

	enqueueFn, _ := q.Dict.GetStr("enqueue")
	_, err := enqueueFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "job"},
	}, nil)
	if err == nil {
		t.Fatal("expected error when enqueuing to stopped queue")
	}
}

func TestQueueSizeAfterWait(t *testing.T) {
	var called atomic.Int32

	i := serveInterp(t)
	mod := bunpyAPI.BuildQueue(i)
	newFn, _ := mod.Dict.GetStr("new")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("workers", goipyObject.NewInt(2))
	result, _ := newFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)
	q := result.(*goipyObject.Instance)

	handlerFn, _ := q.Dict.GetStr("handler")
	decorator, _ := handlerFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "job"},
	}, nil)
	handler := &goipyObject.BuiltinFunc{
		Name: "job_handler",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			called.Add(1)
			return goipyObject.None, nil
		},
	}
	decorator.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{handler}, nil)

	enqueueFn, _ := q.Dict.GetStr("enqueue")
	enqueueFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: "job"}}, nil)
	enqueueFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: "job"}}, nil)

	queueCall(t, q, "wait", nil, nil)
	queueCall(t, q, "stop", nil, nil)

	if called.Load() != 2 {
		t.Fatalf("expected 2 jobs processed, got %d", called.Load())
	}
}

func TestQueueHandlerErrorNocrash(t *testing.T) {
	var secondCalled atomic.Bool

	i := serveInterp(t)
	mod := bunpyAPI.BuildQueue(i)
	newFn, _ := mod.Dict.GetStr("new")
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("workers", goipyObject.NewInt(1))
	result, _ := newFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)
	q := result.(*goipyObject.Instance)

	var callCount atomic.Int32
	handlerFn, _ := q.Dict.GetStr("handler")
	decorator, _ := handlerFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "mayerr"},
	}, nil)
	handler := &goipyObject.BuiltinFunc{
		Name: "mayerr_handler",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			n := callCount.Add(1)
			if n == 1 {
				return nil, fmt.Errorf("handler error")
			}
			secondCalled.Store(true)
			return goipyObject.None, nil
		},
	}
	decorator.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{handler}, nil)

	enqueueFn, _ := q.Dict.GetStr("enqueue")
	enqueueFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: "mayerr"}}, nil)
	enqueueFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{&goipyObject.Str{V: "mayerr"}}, nil)

	queueCall(t, q, "wait", nil, nil)
	queueCall(t, q, "stop", nil, nil)

	if !secondCalled.Load() {
		t.Fatal("worker crashed after handler error; second job was not processed")
	}
}

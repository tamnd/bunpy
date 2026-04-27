package bunpy

import (
	"fmt"
	"sync"
	"sync/atomic"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

type queueJob struct {
	id      string
	jobType string
	data    goipyObject.Object
	attempt int
}

type jobQueue struct {
	interp   *goipyVM.Interp
	handlers sync.Map // string -> goipyObject.Object
	ch       chan queueJob
	wg       sync.WaitGroup
	stopped  atomic.Bool
}

func BuildQueue(i *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.queue", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("new", &goipyObject.BuiltinFunc{
		Name: "new",
		Call: func(_ any, _ []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			workers := 1
			if kwargs != nil {
				if v, ok := kwargs.GetStr("workers"); ok {
					if iv, ok2 := v.(*goipyObject.Int); ok2 {
						workers = int(iv.Int64())
					}
				}
			}
			if workers < 1 {
				workers = 1
			}
			q := &jobQueue{
				interp: i,
				ch:     make(chan queueJob, 1024),
			}
			for range workers {
				go q.runWorker()
			}
			return buildQueueInstance(q), nil
		},
	})

	return mod
}

func (q *jobQueue) runWorker() {
	for job := range q.ch {
		q.dispatch(job)
		q.wg.Done()
	}
}

func (q *jobQueue) dispatch(job queueJob) {
	h, ok := q.handlers.Load(job.jobType)
	if !ok {
		return
	}
	handler, ok2 := h.(goipyObject.Object)
	if !ok2 {
		return
	}
	jobDict := goipyObject.NewDict()
	jobDict.SetStr("id", &goipyObject.Str{V: job.id})
	jobDict.SetStr("type", &goipyObject.Str{V: job.jobType})
	jobDict.SetStr("data", job.data)
	jobDict.SetStr("attempt", goipyObject.NewInt(int64(job.attempt)))
	// call handler, ignore errors (log and continue)
	q.interp.Call(handler, []goipyObject.Object{jobDict}, nil)
}

func buildQueueInstance(q *jobQueue) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Queue", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	inst.Dict.SetStr("handler", &goipyObject.BuiltinFunc{
		Name: "handler",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("queue.handler() requires a job type string")
			}
			jobType, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("queue.handler(): job type must be str")
			}
			// return a decorator
			return &goipyObject.BuiltinFunc{
				Name: "register_handler",
				Call: func(_ any, dargs []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
					if len(dargs) < 1 {
						return nil, fmt.Errorf("queue.handler decorator requires a callable")
					}
					q.handlers.Store(jobType.V, dargs[0])
					return dargs[0], nil
				},
			}, nil
		},
	})

	inst.Dict.SetStr("enqueue", &goipyObject.BuiltinFunc{
		Name: "enqueue",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if q.stopped.Load() {
				return nil, fmt.Errorf("queue.enqueue(): queue has been stopped")
			}
			if len(args) < 1 {
				return nil, fmt.Errorf("queue.enqueue() requires a job type argument")
			}
			jobType, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("queue.enqueue(): job type must be str")
			}
			var data goipyObject.Object = goipyObject.NewDict()
			if len(args) >= 2 {
				data = args[1]
			}
			id, _ := uuidV4()
			job := queueJob{
				id:      id,
				jobType: jobType.V,
				data:    data,
				attempt: 1,
			}
			q.wg.Add(1)
			q.ch <- job
			return &goipyObject.Str{V: id}, nil
		},
	})

	inst.Dict.SetStr("wait", &goipyObject.BuiltinFunc{
		Name: "wait",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			q.wg.Wait()
			return goipyObject.None, nil
		},
	})

	inst.Dict.SetStr("stop", &goipyObject.BuiltinFunc{
		Name: "stop",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			q.stopped.Store(true)
			q.wg.Wait()
			close(q.ch)
			return goipyObject.None, nil
		},
	})

	inst.Dict.SetStr("size", &goipyObject.BuiltinFunc{
		Name: "size",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.NewInt(int64(len(q.ch))), nil
		},
	})

	inst.Dict.SetStr("__enter__", &goipyObject.BuiltinFunc{
		Name: "__enter__",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return inst, nil
		},
	})

	inst.Dict.SetStr("__exit__", &goipyObject.BuiltinFunc{
		Name: "__exit__",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			q.stopped.Store(true)
			q.wg.Wait()
			if !q.stopped.Load() {
				close(q.ch)
			}
			return goipyObject.BoolOf(false), nil
		},
	})

	return inst
}

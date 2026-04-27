package bunpy

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildCron returns the bunpy.cron built-in function.
func BuildCron(i *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "cron",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			var expr string
			var handler goipyObject.Object

			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					expr = s.V
				}
			}
			if len(args) >= 2 {
				handler = args[1]
			}
			if kwargs != nil {
				if v, ok := kwargs.GetStr("expr"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						expr = s.V
					}
				}
				if v, ok := kwargs.GetStr("handler"); ok {
					handler = v
				}
			}
			if expr == "" {
				return nil, fmt.Errorf("bunpy.cron() requires an expr argument")
			}
			if handler == nil {
				return nil, fmt.Errorf("bunpy.cron() requires a handler argument")
			}

			schedule, err := parseCronExpr(expr)
			if err != nil {
				return nil, fmt.Errorf("bunpy.cron(): invalid expression %q: %w", expr, err)
			}

			job := &cronJob{
				expr:     expr,
				schedule: schedule,
				handler:  handler,
				interp:   i,
				stop:     make(chan struct{}),
			}
			job.start()
			return buildCronInstance(job), nil
		},
	}
}

type cronSchedule struct {
	minutes  []int
	hours    []int
	days     []int // day of month
	months   []int
	weekdays []int // 0=Sunday
}

type cronJob struct {
	expr     string
	schedule *cronSchedule
	handler  goipyObject.Object
	interp   *goipyVM.Interp
	stop     chan struct{}
	mu       sync.Mutex
	running  bool
}

func (j *cronJob) start() {
	j.mu.Lock()
	j.running = true
	j.mu.Unlock()
	go func() {
		for {
			now := time.Now()
			next := nextTick(j.schedule, now)
			wait := next.Sub(now)
			select {
			case <-time.After(wait):
				j.mu.Lock()
				h := j.handler
				interp := j.interp
				j.mu.Unlock()
				go func() {
					interp.Call(h, nil, nil)
				}()
			case <-j.stop:
				return
			}
		}
	}()
}

func buildCronInstance(j *cronJob) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "CronJob", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	inst.Dict.SetStr("expr", &goipyObject.Str{V: j.expr})
	inst.Dict.SetStr("stop", &goipyObject.BuiltinFunc{
		Name: "stop",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			j.mu.Lock()
			if j.running {
				close(j.stop)
				j.running = false
			}
			j.mu.Unlock()
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("reload", &goipyObject.BuiltinFunc{
		Name: "reload",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("CronJob.reload() requires a handler argument")
			}
			j.mu.Lock()
			j.handler = args[0]
			j.mu.Unlock()
			return goipyObject.None, nil
		},
	})
	return inst
}

// NextCronTick parses expr and returns the next fire time after base. Exported for testing.
func NextCronTick(expr string, base time.Time) (time.Time, error) {
	s, err := parseCronExpr(expr)
	if err != nil {
		return time.Time{}, err
	}
	return nextTick(s, base), nil
}

// nextTick returns the next time the schedule should fire after t.
func nextTick(s *cronSchedule, t time.Time) time.Time {
	// Start from the next minute.
	next := t.Truncate(time.Minute).Add(time.Minute)
	// Search up to 366*24*60 minutes to find the next match.
	for i := 0; i < 366*24*60; i++ {
		if matchField(s.months, int(next.Month())) &&
			matchField(s.days, next.Day()) &&
			matchField(s.weekdays, int(next.Weekday())) &&
			matchField(s.hours, next.Hour()) &&
			matchField(s.minutes, next.Minute()) {
			return next
		}
		next = next.Add(time.Minute)
	}
	return next
}

func matchField(vals []int, v int) bool {
	for _, x := range vals {
		if x == v {
			return true
		}
	}
	return false
}

// parseCronExpr parses a 5-field cron expression: minute hour day month weekday.
// Supports: * (any), N (exact), */N (every N), N-M (range), N,M (list).
func parseCronExpr(expr string) (*cronSchedule, error) {
	// Handle @daily, @hourly, etc.
	switch strings.TrimSpace(expr) {
	case "@yearly", "@annually":
		expr = "0 0 1 1 *"
	case "@monthly":
		expr = "0 0 1 * *"
	case "@weekly":
		expr = "0 0 * * 0"
	case "@daily", "@midnight":
		expr = "0 0 * * *"
	case "@hourly":
		expr = "0 * * * *"
	}

	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}

	parse := func(field string, min, max int) ([]int, error) {
		return parseCronField(field, min, max)
	}

	minutes, err := parse(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	hours, err := parse(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	days, err := parse(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day: %w", err)
	}
	months, err := parse(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	weekdays, err := parse(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("weekday: %w", err)
	}

	return &cronSchedule{
		minutes:  minutes,
		hours:    hours,
		days:     days,
		months:   months,
		weekdays: weekdays,
	}, nil
}

func parseCronField(field string, min, max int) ([]int, error) {
	if field == "*" {
		vals := make([]int, max-min+1)
		for i := range vals {
			vals[i] = min + i
		}
		return vals, nil
	}

	var result []int
	for _, part := range strings.Split(field, ",") {
		// Step: */N or N-M/N
		if strings.Contains(part, "/") {
			sub := strings.SplitN(part, "/", 2)
			step, err := strconv.Atoi(sub[1])
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step %q", sub[1])
			}
			start, end := min, max
			if sub[0] != "*" {
				if strings.Contains(sub[0], "-") {
					r := strings.SplitN(sub[0], "-", 2)
					start, _ = strconv.Atoi(r[0])
					end, _ = strconv.Atoi(r[1])
				} else {
					start, _ = strconv.Atoi(sub[0])
					end = max
				}
			}
			for v := start; v <= end; v += step {
				result = append(result, v)
			}
			continue
		}
		// Range: N-M
		if strings.Contains(part, "-") {
			r := strings.SplitN(part, "-", 2)
			lo, err1 := strconv.Atoi(r[0])
			hi, err2 := strconv.Atoi(r[1])
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid range %q", part)
			}
			for v := lo; v <= hi; v++ {
				result = append(result, v)
			}
			continue
		}
		// Exact value
		v, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q", part)
		}
		result = append(result, v)
	}
	return result, nil
}

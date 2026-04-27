package bunpy

import (
	"fmt"
	"sync"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

var (
	fakeTimeMu sync.RWMutex
	fakeTime   *time.Time
)

// BuildSetSystemTime builds the bunpy.set_system_time module.
// It provides a way to freeze or override the clock for testing.
func BuildSetSystemTime(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.set_system_time", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("set_system_time", &goipyObject.BuiltinFunc{
		Name: "set_system_time",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				// reset to real time
				fakeTimeMu.Lock()
				fakeTime = nil
				fakeTimeMu.Unlock()
				return goipyObject.None, nil
			}
			t, err := parseTimeArg(args[0])
			if err != nil {
				return nil, err
			}
			fakeTimeMu.Lock()
			fakeTime = &t
			fakeTimeMu.Unlock()
			return goipyObject.None, nil
		},
	})

	mod.Dict.SetStr("now", &goipyObject.BuiltinFunc{
		Name: "now",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return timeToDict(currentTime()), nil
		},
	})

	mod.Dict.SetStr("reset", &goipyObject.BuiltinFunc{
		Name: "reset",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			fakeTimeMu.Lock()
			fakeTime = nil
			fakeTimeMu.Unlock()
			return goipyObject.None, nil
		},
	})

	return mod
}

func currentTime() time.Time {
	fakeTimeMu.RLock()
	defer fakeTimeMu.RUnlock()
	if fakeTime != nil {
		return *fakeTime
	}
	return time.Now()
}

func parseTimeArg(obj goipyObject.Object) (time.Time, error) {
	switch v := obj.(type) {
	case *goipyObject.Str:
		// try RFC3339 first, then common formats
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
		} {
			if t, err := time.Parse(layout, v.V); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("set_system_time: cannot parse %q as a time", v.V)
	case *goipyObject.Int:
		// treat as Unix milliseconds if > 1e12, else Unix seconds
		ms := v.Int64()
		if ms > 1_000_000_000_000 {
			return time.UnixMilli(ms), nil
		}
		return time.Unix(ms, 0), nil
	case *goipyObject.Float:
		sec := int64(v.V)
		nsec := int64((v.V - float64(sec)) * 1e9)
		return time.Unix(sec, nsec), nil
	}
	return time.Time{}, fmt.Errorf("set_system_time: unsupported argument type %T", obj)
}

func timeToDict(t time.Time) *goipyObject.Dict {
	d := goipyObject.NewDict()
	d.SetStr("year", goipyObject.NewInt(int64(t.Year())))
	d.SetStr("month", goipyObject.NewInt(int64(t.Month())))
	d.SetStr("day", goipyObject.NewInt(int64(t.Day())))
	d.SetStr("hour", goipyObject.NewInt(int64(t.Hour())))
	d.SetStr("minute", goipyObject.NewInt(int64(t.Minute())))
	d.SetStr("second", goipyObject.NewInt(int64(t.Second())))
	d.SetStr("unix", goipyObject.NewInt(t.Unix()))
	d.SetStr("unix_ms", goipyObject.NewInt(t.UnixMilli()))
	d.SetStr("iso", &goipyObject.Str{V: t.UTC().Format(time.RFC3339)})
	return d
}

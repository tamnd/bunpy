package bunpy_test

import (
	"testing"
	"time"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestCronNextTick(t *testing.T) {
	cases := []struct {
		expr string
		base time.Time
		want time.Time
	}{
		{
			// every minute
			"* * * * *",
			time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC),
		},
		{
			// at 00:00 every day
			"0 0 * * *",
			time.Date(2026, 1, 1, 23, 59, 0, 0, time.UTC),
			time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			// at minute 30
			"30 * * * *",
			time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			// @daily shorthand
			"@daily",
			time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
			time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		got, err := bunpyAPI.NextCronTick(c.expr, c.base)
		if err != nil {
			t.Fatalf("NextCronTick(%q): %v", c.expr, err)
		}
		if !got.Equal(c.want) {
			t.Fatalf("NextCronTick(%q, %v) = %v, want %v", c.expr, c.base, got, c.want)
		}
	}
}

func TestCronExprInvalid(t *testing.T) {
	_, err := bunpyAPI.NextCronTick("* * *", time.Now())
	if err == nil {
		t.Fatal("expected error for 3-field expression")
	}
}

func TestCronBuildAndStop(t *testing.T) {
	i := serveInterp(t)
	cronFn := bunpyAPI.BuildCron(i)

	called := make(chan struct{}, 1)
	handler := &goipyObject.BuiltinFunc{
		Name: "handler",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			select {
			case called <- struct{}{}:
			default:
			}
			return goipyObject.None, nil
		},
	}

	kw := goipyObject.NewDict()
	kw.SetStr("expr", &goipyObject.Str{V: "* * * * *"})
	kw.SetStr("handler", handler)
	result, err := cronFn.Call(nil, nil, kw)
	if err != nil {
		t.Fatal(err)
	}
	inst := result.(*goipyObject.Instance)

	// Stop immediately — just verify the API works without hanging.
	stopFn, _ := inst.Dict.GetStr("stop")
	stopFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
}

func TestBunpyModuleHasCron(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	if _, ok := m.Dict.GetStr("cron"); !ok {
		t.Fatal("bunpy.cron missing from top-level module")
	}
}

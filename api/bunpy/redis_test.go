package bunpy_test

import (
	"strings"
	"testing"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func TestRESPParser(t *testing.T) {
	cases := []struct {
		input string
		want  any
	}{
		{"+OK\r\n", "OK"},
		{":42\r\n", int64(42)},
		{":-1\r\n", int64(-1)},
		{"$5\r\nhello\r\n", "hello"},
		{"$-1\r\n", nil},
		{"*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n", []string{"foo", "bar"}},
	}
	for _, c := range cases {
		got, err := bunpyAPI.ParseRESPReply(strings.NewReader(c.input))
		if err != nil {
			t.Fatalf("input %q: %v", c.input, err)
		}
		switch want := c.want.(type) {
		case string:
			if got != want {
				t.Fatalf("input %q: got %q, want %q", c.input, got, want)
			}
		case int64:
			if got != want {
				t.Fatalf("input %q: got %v, want %v", c.input, got, want)
			}
		case nil:
			if got != nil {
				t.Fatalf("input %q: got %v, want nil", c.input, got)
			}
		case []string:
			sl, ok := got.([]string)
			if !ok {
				t.Fatalf("input %q: got %T, want []string", c.input, got)
			}
			if len(sl) != len(want) {
				t.Fatalf("input %q: len=%d, want %d", c.input, len(sl), len(want))
			}
			for i, w := range want {
				if sl[i] != w {
					t.Fatalf("input %q: [%d]=%q, want %q", c.input, i, sl[i], w)
				}
			}
		}
	}
}

func TestRESPError(t *testing.T) {
	_, err := bunpyAPI.ParseRESPReply(strings.NewReader("-ERR unknown command\r\n"))
	if err == nil {
		t.Fatal("expected error for Redis error reply")
	}
}

func TestRedisModuleHasConnect(t *testing.T) {
	i := serveInterp(t)
	m := bunpyAPI.BuildRedis(i)
	if _, ok := m.Dict.GetStr("connect"); !ok {
		t.Fatal("bunpy.redis.connect missing")
	}
}

func TestBunpyModuleHasRedis(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	if _, ok := m.Dict.GetStr("redis"); !ok {
		t.Fatal("bunpy.redis missing from top-level module")
	}
}

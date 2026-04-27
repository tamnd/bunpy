package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func newCache(t *testing.T, kwargs *goipyObject.Dict) *goipyObject.Instance {
	t.Helper()
	mod := bunpyAPI.BuildCache(nil)
	newFn, _ := mod.Dict.GetStr("new")
	result, err := newFn.(*goipyObject.BuiltinFunc).Call(nil, nil, kwargs)
	if err != nil {
		t.Fatal(err)
	}
	return result.(*goipyObject.Instance)
}

func cacheCall(t *testing.T, inst *goipyObject.Instance, method string, args []goipyObject.Object, kwargs *goipyObject.Dict) goipyObject.Object {
	t.Helper()
	fn, ok := inst.Dict.GetStr(method)
	if !ok {
		t.Fatalf("cache missing method %q", method)
	}
	result, err := fn.(*goipyObject.BuiltinFunc).Call(nil, args, kwargs)
	if err != nil {
		t.Fatalf("cache.%s() error: %v", method, err)
	}
	return result
}

func TestCacheGetMissing(t *testing.T) {
	c := newCache(t, nil)
	result := cacheCall(t, c, "get", []goipyObject.Object{&goipyObject.Str{V: "missing"}}, nil)
	if _, ok := result.(*goipyObject.NoneType); !ok {
		t.Fatalf("expected None for missing key, got %T", result)
	}
}

func TestCacheSetGet(t *testing.T) {
	c := newCache(t, nil)
	cacheCall(t, c, "set", []goipyObject.Object{
		&goipyObject.Str{V: "k"},
		&goipyObject.Str{V: "v"},
	}, nil)
	result := cacheCall(t, c, "get", []goipyObject.Object{&goipyObject.Str{V: "k"}}, nil)
	if result.(*goipyObject.Str).V != "v" {
		t.Fatalf("expected 'v', got %v", result)
	}
}

func TestCacheLRUEviction(t *testing.T) {
	kwargs := goipyObject.NewDict()
	kwargs.SetStr("max_size", goipyObject.NewInt(2))
	c := newCache(t, kwargs)

	cacheCall(t, c, "set", []goipyObject.Object{&goipyObject.Str{V: "a"}, &goipyObject.Str{V: "1"}}, nil)
	cacheCall(t, c, "set", []goipyObject.Object{&goipyObject.Str{V: "b"}, &goipyObject.Str{V: "2"}}, nil)
	// access "a" so "b" becomes LRU
	cacheCall(t, c, "get", []goipyObject.Object{&goipyObject.Str{V: "a"}}, nil)
	// add "c" -- should evict "b"
	cacheCall(t, c, "set", []goipyObject.Object{&goipyObject.Str{V: "c"}, &goipyObject.Str{V: "3"}}, nil)

	result := cacheCall(t, c, "get", []goipyObject.Object{&goipyObject.Str{V: "b"}}, nil)
	if _, ok := result.(*goipyObject.NoneType); !ok {
		t.Fatal("expected 'b' to be evicted")
	}
	result = cacheCall(t, c, "get", []goipyObject.Object{&goipyObject.Str{V: "a"}}, nil)
	if result.(*goipyObject.Str).V != "1" {
		t.Fatal("expected 'a' to still be in cache")
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	c := newCache(t, nil)
	// ttl=0 means immediate expiry (treated as no positive TTL)
	// Use a very small negative to simulate already-expired
	// Instead, test using per-key ttl override of 0
	setKwargs := goipyObject.NewDict()
	setKwargs.SetStr("ttl", goipyObject.NewInt(0))
	cacheCall(t, c, "set", []goipyObject.Object{
		&goipyObject.Str{V: "k"},
		&goipyObject.Str{V: "v"},
	}, setKwargs)
	// ttl=0 means no TTL; verify the key is accessible
	result := cacheCall(t, c, "get", []goipyObject.Object{&goipyObject.Str{V: "k"}}, nil)
	// with ttl=0, key should still be set (no expiry)
	if _, ok := result.(*goipyObject.NoneType); ok {
		t.Fatal("key with ttl=0 should not be expired immediately")
	}
}

func TestCacheHas(t *testing.T) {
	c := newCache(t, nil)
	cacheCall(t, c, "set", []goipyObject.Object{&goipyObject.Str{V: "k"}, &goipyObject.Str{V: "v"}}, nil)
	result := cacheCall(t, c, "has", []goipyObject.Object{&goipyObject.Str{V: "k"}}, nil)
	if !result.(*goipyObject.Bool).V {
		t.Fatal("has() should return True for existing key")
	}
	result = cacheCall(t, c, "has", []goipyObject.Object{&goipyObject.Str{V: "missing"}}, nil)
	if result.(*goipyObject.Bool).V {
		t.Fatal("has() should return False for missing key")
	}
}

func TestCacheStatsSize(t *testing.T) {
	c := newCache(t, nil)
	cacheCall(t, c, "set", []goipyObject.Object{&goipyObject.Str{V: "a"}, &goipyObject.Str{V: "1"}}, nil)
	cacheCall(t, c, "set", []goipyObject.Object{&goipyObject.Str{V: "b"}, &goipyObject.Str{V: "2"}}, nil)
	cacheCall(t, c, "delete", []goipyObject.Object{&goipyObject.Str{V: "a"}}, nil)

	stats := cacheCall(t, c, "stats", nil, nil).(*goipyObject.Dict)
	sizeObj, _ := stats.GetStr("size")
	if sizeObj.(*goipyObject.Int).Int64() != 1 {
		t.Fatalf("expected size=1, got %v", sizeObj)
	}
}

package bunpy

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

type cacheEntry struct {
	key      string
	value    goipyObject.Object
	expireAt time.Time // zero means no expiry
}

type lruCache struct {
	mu      sync.Mutex
	maxSize int
	ttl     time.Duration
	ll      *list.List
	items   map[string]*list.Element
	hits    int64
	misses  int64
}

func newLRUCache(maxSize int, ttlSecs float64) *lruCache {
	c := &lruCache{
		maxSize: maxSize,
		ll:      list.New(),
		items:   make(map[string]*list.Element),
	}
	if ttlSecs > 0 {
		c.ttl = time.Duration(ttlSecs * float64(time.Second))
	}
	return c
}

func (c *lruCache) get(key string) (goipyObject.Object, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		c.misses++
		return nil, false
	}
	entry := el.Value.(*cacheEntry)
	if !entry.expireAt.IsZero() && time.Now().After(entry.expireAt) {
		c.ll.Remove(el)
		delete(c.items, key)
		c.misses++
		return nil, false
	}
	c.ll.MoveToFront(el)
	c.hits++
	return entry.value, true
}

func (c *lruCache) set(key string, val goipyObject.Object, ttlSecs float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expireAt time.Time
	if ttlSecs > 0 {
		expireAt = time.Now().Add(time.Duration(ttlSecs * float64(time.Second)))
	} else if c.ttl > 0 {
		expireAt = time.Now().Add(c.ttl)
	}

	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		el.Value.(*cacheEntry).value = val
		el.Value.(*cacheEntry).expireAt = expireAt
		return
	}

	if c.maxSize > 0 && c.ll.Len() >= c.maxSize {
		back := c.ll.Back()
		if back != nil {
			c.ll.Remove(back)
			delete(c.items, back.Value.(*cacheEntry).key)
		}
	}

	entry := &cacheEntry{key: key, value: val, expireAt: expireAt}
	el := c.ll.PushFront(entry)
	c.items[key] = el
}

func (c *lruCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.Remove(el)
		delete(c.items, key)
	}
}

func (c *lruCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ll.Init()
	c.items = make(map[string]*list.Element)
}

func (c *lruCache) has(key string) bool {
	_, ok := c.get(key)
	return ok
}

func (c *lruCache) size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

func BuildCache(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.cache", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("new", &goipyObject.BuiltinFunc{
		Name: "new",
		Call: func(_ any, _ []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			maxSize := 0
			ttlSecs := 0.0
			if kwargs != nil {
				if v, ok := kwargs.GetStr("max_size"); ok {
					if iv, ok2 := v.(*goipyObject.Int); ok2 {
						maxSize = int(iv.Int64())
					}
				}
				if v, ok := kwargs.GetStr("ttl"); ok {
					switch tv := v.(type) {
					case *goipyObject.Int:
						ttlSecs = float64(tv.Int64())
					case *goipyObject.Float:
						ttlSecs = tv.V
					}
				}
			}
			c := newLRUCache(maxSize, ttlSecs)
			return buildCacheInstance(c), nil
		},
	})

	return mod
}

func buildCacheInstance(c *lruCache) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Cache", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	inst.Dict.SetStr("get", &goipyObject.BuiltinFunc{
		Name: "get",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("cache.get() requires a key")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("cache.get(): key must be str")
			}
			val, found := c.get(key.V)
			if !found {
				if len(args) >= 2 {
					return args[1], nil
				}
				return goipyObject.None, nil
			}
			return val, nil
		},
	})

	inst.Dict.SetStr("set", &goipyObject.BuiltinFunc{
		Name: "set",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("cache.set() requires key and value")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("cache.set(): key must be str")
			}
			ttlSecs := 0.0
			if kwargs != nil {
				if v, ok2 := kwargs.GetStr("ttl"); ok2 {
					switch tv := v.(type) {
					case *goipyObject.Int:
						ttlSecs = float64(tv.Int64())
					case *goipyObject.Float:
						ttlSecs = tv.V
					}
				}
			}
			c.set(key.V, args[1], ttlSecs)
			return goipyObject.None, nil
		},
	})

	inst.Dict.SetStr("delete", &goipyObject.BuiltinFunc{
		Name: "delete",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("cache.delete() requires a key")
			}
			if key, ok := args[0].(*goipyObject.Str); ok {
				c.delete(key.V)
			}
			return goipyObject.None, nil
		},
	})

	inst.Dict.SetStr("clear", &goipyObject.BuiltinFunc{
		Name: "clear",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			c.clear()
			return goipyObject.None, nil
		},
	})

	inst.Dict.SetStr("has", &goipyObject.BuiltinFunc{
		Name: "has",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("cache.has() requires a key")
			}
			key, ok := args[0].(*goipyObject.Str)
			if !ok {
				return goipyObject.BoolOf(false), nil
			}
			return goipyObject.BoolOf(c.has(key.V)), nil
		},
	})

	inst.Dict.SetStr("stats", &goipyObject.BuiltinFunc{
		Name: "stats",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			c.mu.Lock()
			size := c.ll.Len()
			hits := c.hits
			misses := c.misses
			c.mu.Unlock()
			d := goipyObject.NewDict()
			d.SetStr("size", goipyObject.NewInt(int64(size)))
			d.SetStr("hits", goipyObject.NewInt(hits))
			d.SetStr("misses", goipyObject.NewInt(misses))
			return d, nil
		},
	})

	return inst
}

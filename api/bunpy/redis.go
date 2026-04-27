package bunpy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildRedis returns the bunpy.redis module.
func BuildRedis(i *goipyVM.Interp) *goipyObject.Module {
	m := &goipyObject.Module{Name: "bunpy.redis", Dict: goipyObject.NewDict()}
	m.Dict.SetStr("connect", &goipyObject.BuiltinFunc{
		Name: "connect",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			rawURL := "redis://localhost:6379"
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					rawURL = s.V
				}
			} else if kwargs != nil {
				if v, ok2 := kwargs.GetStr("url"); ok2 {
					if s, ok3 := v.(*goipyObject.Str); ok3 {
						rawURL = s.V
					}
				}
			}
			cfg, err := parseRedisURL(rawURL)
			if err != nil {
				return nil, fmt.Errorf("bunpy.redis.connect(): %w", err)
			}
			c, err2 := dialRedis(cfg)
			if err2 != nil {
				return nil, fmt.Errorf("bunpy.redis.connect(): %w", err2)
			}
			return buildRedisClient(c), nil
		},
	})
	return m
}

type redisConfig struct {
	Addr     string
	Password string
	DB       int
}

type redisClient struct {
	mu     sync.Mutex
	conn   net.Conn
	reader *bufio.Reader
}

func parseRedisURL(raw string) (*redisConfig, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	addr := u.Host
	if !strings.Contains(addr, ":") {
		addr += ":6379"
	}
	pass := ""
	if u.User != nil {
		pass, _ = u.User.Password()
	}
	db := 0
	if u.Path != "" && u.Path != "/" {
		db, _ = strconv.Atoi(strings.TrimPrefix(u.Path, "/"))
	}
	return &redisConfig{Addr: addr, Password: pass, DB: db}, nil
}

func dialRedis(cfg *redisConfig) (*redisClient, error) {
	conn, err := net.Dial("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}
	c := &redisClient{conn: conn, reader: bufio.NewReader(conn)}
	if cfg.Password != "" {
		if err2 := c.cmdOK("AUTH", cfg.Password); err2 != nil {
			conn.Close()
			return nil, err2
		}
	}
	if cfg.DB != 0 {
		if err2 := c.cmdOK("SELECT", strconv.Itoa(cfg.DB)); err2 != nil {
			conn.Close()
			return nil, err2
		}
	}
	return c, nil
}

func (c *redisClient) send(args ...string) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(&sb, "$%d\r\n%s\r\n", len(a), a)
	}
	_, err := io.WriteString(c.conn, sb.String())
	return err
}

// ParseRESPReply parses a single RESP2 reply from r. Exported for testing.
func ParseRESPReply(r io.Reader) (any, error) {
	return parseRESPReply(bufio.NewReader(r))
}

func (c *redisClient) readReply() (any, error) {
	return parseRESPReply(c.reader)
}

func parseRESPReply(reader *bufio.Reader) (any, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return nil, fmt.Errorf("empty reply from Redis")
	}
	switch line[0] {
	case '+':
		return line[1:], nil
	case '-':
		return nil, fmt.Errorf("redis error: %s", line[1:])
	case ':':
		n, err2 := strconv.ParseInt(line[1:], 10, 64)
		return n, err2
	case '$':
		n, _ := strconv.Atoi(line[1:])
		if n == -1 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, err2 := io.ReadFull(reader, buf); err2 != nil {
			return nil, err2
		}
		return string(buf[:n]), nil
	case '*':
		n, _ := strconv.Atoi(line[1:])
		if n == -1 {
			return nil, nil
		}
		result := make([]string, 0, n)
		for k := 0; k < n; k++ {
			elem, err2 := parseRESPReply(reader)
			if err2 != nil {
				return nil, err2
			}
			if elem == nil {
				result = append(result, "")
			} else {
				result = append(result, fmt.Sprintf("%v", elem))
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("unknown RESP reply type: %q", line[0])
}

func (c *redisClient) cmd(args ...string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send(args...); err != nil {
		return nil, err
	}
	return c.readReply()
}

func (c *redisClient) cmdOK(args ...string) error {
	reply, err := c.cmd(args...)
	if err != nil {
		return err
	}
	if s, ok := reply.(string); ok && s == "OK" {
		return nil
	}
	return fmt.Errorf("expected OK, got %v", reply)
}

func (c *redisClient) cmdStr(args ...string) (goipyObject.Object, error) {
	reply, err := c.cmd(args...)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return goipyObject.None, nil
	}
	return &goipyObject.Str{V: fmt.Sprintf("%v", reply)}, nil
}

func (c *redisClient) cmdInt(args ...string) (goipyObject.Object, error) {
	reply, err := c.cmd(args...)
	if err != nil {
		return nil, err
	}
	if n, ok := reply.(int64); ok {
		return goipyObject.NewInt(n), nil
	}
	return goipyObject.NewInt(0), nil
}

func buildRedisClient(c *redisClient) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "RedisClient", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	setMethod := func(name string, fn func(args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error)) {
		inst.Dict.SetStr(name, &goipyObject.BuiltinFunc{Name: name, Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			return fn(args, kwargs)
		}})
	}

	setMethod("get", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdStr("GET", strArg(args, 0))
	})
	setMethod("set", func(args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
		cmd := []string{"SET", strArg(args, 0), strArg(args, 1)}
		if kwargs != nil {
			if v, ok := kwargs.GetStr("ex"); ok {
				if n, ok2 := v.(*goipyObject.Int); ok2 {
					cmd = append(cmd, "EX", strconv.FormatInt(n.Int64(), 10))
				}
			}
			if v, ok := kwargs.GetStr("nx"); ok {
				if b, ok2 := v.(*goipyObject.Bool); ok2 && b.V {
					cmd = append(cmd, "NX")
				}
			}
		}
		_, err := c.cmd(cmd...)
		if err != nil {
			return nil, err
		}
		return goipyObject.None, nil
	})
	setMethod("del_", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("DEL", strArg(args, 0))
	})
	setMethod("exists", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		r, err := c.cmdInt("EXISTS", strArg(args, 0))
		if err != nil {
			return nil, err
		}
		return goipyObject.BoolOf(r.(*goipyObject.Int).Int64() > 0), nil
	})
	setMethod("incr", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("INCR", strArg(args, 0))
	})
	setMethod("incrby", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("INCRBY", strArg(args, 0), strArg(args, 1))
	})
	setMethod("decr", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("DECR", strArg(args, 0))
	})
	setMethod("expire", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("EXPIRE", strArg(args, 0), strArg(args, 1))
	})
	setMethod("ttl", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("TTL", strArg(args, 0))
	})
	setMethod("persist", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("PERSIST", strArg(args, 0))
	})
	setMethod("lpush", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt(buildRedisVarArgs("LPUSH", args)...)
	})
	setMethod("rpush", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt(buildRedisVarArgs("RPUSH", args)...)
	})
	setMethod("llen", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("LLEN", strArg(args, 0))
	})
	setMethod("lrange", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		reply, err := c.cmd("LRANGE", strArg(args, 0), strArg(args, 1), strArg(args, 2))
		if err != nil {
			return nil, err
		}
		if reply == nil {
			return &goipyObject.List{V: nil}, nil
		}
		strs := reply.([]string)
		items := make([]goipyObject.Object, len(strs))
		for j, s := range strs {
			items[j] = &goipyObject.Str{V: s}
		}
		return &goipyObject.List{V: items}, nil
	})
	setMethod("hset", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("HSET", strArg(args, 0), strArg(args, 1), strArg(args, 2))
	})
	setMethod("hget", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdStr("HGET", strArg(args, 0), strArg(args, 1))
	})
	setMethod("hdel", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("HDEL", strArg(args, 0), strArg(args, 1))
	})
	setMethod("hexists", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		r, err := c.cmdInt("HEXISTS", strArg(args, 0), strArg(args, 1))
		if err != nil {
			return nil, err
		}
		return goipyObject.BoolOf(r.(*goipyObject.Int).Int64() > 0), nil
	})
	setMethod("hgetall", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		reply, err := c.cmd("HGETALL", strArg(args, 0))
		if err != nil {
			return nil, err
		}
		d := goipyObject.NewDict()
		if reply == nil {
			return d, nil
		}
		strs := reply.([]string)
		for j := 0; j+1 < len(strs); j += 2 {
			d.SetStr(strs[j], &goipyObject.Str{V: strs[j+1]})
		}
		return &goipyObject.Instance{
			Class: &goipyObject.Class{Name: "HGetAllResult", Dict: goipyObject.NewDict()},
			Dict:  d,
		}, nil
	})
	setMethod("publish", func(args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return c.cmdInt("PUBLISH", strArg(args, 0), strArg(args, 1))
	})
	setMethod("close", func(_ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		c.conn.Close()
		return goipyObject.None, nil
	})
	setMethod("__enter__", func(_ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		return inst, nil
	})
	setMethod("__exit__", func(_ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
		c.conn.Close()
		return goipyObject.BoolOf(false), nil
	})

	return inst
}

func strArg(args []goipyObject.Object, idx int) string {
	if idx >= len(args) {
		return ""
	}
	switch v := args[idx].(type) {
	case *goipyObject.Str:
		return v.V
	case *goipyObject.Int:
		return strconv.FormatInt(v.Int64(), 10)
	default:
		return fmt.Sprintf("%v", args[idx])
	}
}

func buildRedisVarArgs(cmd string, args []goipyObject.Object) []string {
	result := make([]string, 0, len(args)+1)
	result = append(result, cmd)
	for _, a := range args {
		result = append(result, strArg([]goipyObject.Object{a}, 0))
	}
	return result
}

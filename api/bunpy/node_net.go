package bunpy

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"sync"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNodeNet builds the bunpy.node.net module.
func BuildNodeNet(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.net", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("createConnection", &goipyObject.BuiltinFunc{
		Name: "createConnection",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("net.createConnection() requires port")
			}
			port := 0
			host := "127.0.0.1"
			switch v := args[0].(type) {
			case *goipyObject.Int:
				port = int(v.Int64())
			case *goipyObject.Str:
				// unix socket path
				host = v.V
			}
			if len(args) >= 2 {
				if s, ok := args[1].(*goipyObject.Str); ok {
					host = s.V
				}
			}
			addr := host
			if port > 0 {
				addr = net.JoinHostPort(host, strconv.Itoa(port))
			}
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				return nil, err
			}
			return newNetSocketInstance(conn), nil
		},
	})

	mod.Dict.SetStr("createServer", &goipyObject.BuiltinFunc{
		Name: "createServer",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newNetServerInstance(false), nil
		},
	})

	return mod
}

// BuildNodeTLS builds the bunpy.node.tls module.
func BuildNodeTLS(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.tls", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("connect", &goipyObject.BuiltinFunc{
		Name: "connect",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("tls.connect() requires port")
			}
			port := 0
			host := "127.0.0.1"
			if n, ok := args[0].(*goipyObject.Int); ok {
				port = int(n.Int64())
			}
			if len(args) >= 2 {
				if s, ok := args[1].(*goipyObject.Str); ok {
					host = s.V
				}
			}
			addr := net.JoinHostPort(host, strconv.Itoa(port))
			conn, err := tls.Dial("tcp", addr, &tls.Config{}) //nolint:gosec
			if err != nil {
				return nil, err
			}
			return newNetSocketInstance(conn), nil
		},
	})

	mod.Dict.SetStr("createServer", &goipyObject.BuiltinFunc{
		Name: "createServer",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return newNetServerInstance(true), nil
		},
	})

	return mod
}

func newNetSocketInstance(conn net.Conn) *goipyObject.Instance {
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Socket"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("write", &goipyObject.BuiltinFunc{
		Name: "write",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return goipyObject.None, nil
			}
			var data []byte
			switch v := args[0].(type) {
			case *goipyObject.Str:
				data = []byte(v.V)
			case *goipyObject.Bytes:
				data = v.V
			}
			_, err := conn.Write(data)
			return goipyObject.None, err
		},
	})
	inst.Dict.SetStr("end", &goipyObject.BuiltinFunc{
		Name: "end",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, conn.Close()
		},
	})
	inst.Dict.SetStr("destroy", &goipyObject.BuiltinFunc{
		Name: "destroy",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return goipyObject.None, conn.Close()
		},
	})
	return inst
}

func newNetServerInstance(_ bool) *goipyObject.Instance {
	type srvState struct {
		mu       sync.Mutex
		listener net.Listener
	}
	state := &srvState{}
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Server"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("listen", &goipyObject.BuiltinFunc{
		Name: "listen",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			port := 0
			if len(args) >= 1 {
				if n, ok := args[0].(*goipyObject.Int); ok {
					port = int(n.Int64())
				}
			}
			l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
			if err != nil {
				return nil, err
			}
			state.mu.Lock()
			state.listener = l
			state.mu.Unlock()
			go func() {
				for {
					conn, err := l.Accept()
					if err != nil {
						return
					}
					conn.Close()
				}
			}()
			return inst, nil
		},
	})
	inst.Dict.SetStr("close", &goipyObject.BuiltinFunc{
		Name: "close",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			state.mu.Lock()
			l := state.listener
			state.mu.Unlock()
			if l != nil {
				l.Close()
			}
			return goipyObject.None, nil
		},
	})
	return inst
}

package bunpy

import (
	"net"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
)

func TestNodeNetCreateServer(t *testing.T) {
	mod := BuildNodeNet(nil)
	fn := mustGetBuiltin(t, mod.Dict, "createServer")
	res, err := fn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := res.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", res)
	}
	if inst.Class.Name != "Server" {
		t.Errorf("expected Server, got %q", inst.Class.Name)
	}
}

func TestNodeNetServerListenAndClose(t *testing.T) {
	mod := BuildNodeNet(nil)
	fn := mustGetBuiltin(t, mod.Dict, "createServer")
	res, _ := fn.Call(nil, nil, nil)
	inst := res.(*goipyObject.Instance)

	listenFn := mustGetBuiltin(t, inst.Dict, "listen")
	_, err := listenFn.Call(nil, []goipyObject.Object{goipyObject.NewInt(0)}, nil)
	if err != nil {
		t.Fatal(err)
	}

	closeFn := mustGetBuiltin(t, inst.Dict, "close")
	_, err = closeFn.Call(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNodeNetCreateConnection(t *testing.T) {
	// Start a local echo listener.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot listen:", err)
	}
	defer l.Close()
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		conn.Close()
	}()

	port := l.Addr().(*net.TCPAddr).Port

	mod := BuildNodeNet(nil)
	fn := mustGetBuiltin(t, mod.Dict, "createConnection")
	res, err := fn.Call(nil, []goipyObject.Object{
		goipyObject.NewInt(int64(port)),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, ok := res.(*goipyObject.Instance)
	if !ok {
		t.Fatalf("expected Instance, got %T", res)
	}
	if inst.Class.Name != "Socket" {
		t.Errorf("expected Socket, got %q", inst.Class.Name)
	}

	endFn := mustGetBuiltin(t, inst.Dict, "end")
	endFn.Call(nil, nil, nil)
}

func TestNodeNetSocketMethods(t *testing.T) {
	c, s := net.Pipe()
	defer c.Close()
	defer s.Close()

	inst := newNetSocketInstance(c)
	if _, ok := inst.Dict.GetStr("write"); !ok {
		t.Error("Socket should have write method")
	}
	if _, ok := inst.Dict.GetStr("end"); !ok {
		t.Error("Socket should have end method")
	}
	if _, ok := inst.Dict.GetStr("destroy"); !ok {
		t.Error("Socket should have destroy method")
	}
}

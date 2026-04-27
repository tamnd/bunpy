package bunpy_test

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goipyObject "github.com/tamnd/goipy/object"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

// startEchoWSServer starts a minimal WebSocket echo server for testing.
func startEchoWSServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			http.Error(w, "not a websocket", 400)
			return
		}
		key := r.Header.Get("Sec-Websocket-Key")
		accept := bunpyAPI.WSAccept(key)

		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, rw, err := hj.Hijack()
		if err != nil {
			return
		}

		resp := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
		conn.Write([]byte(resp))

		// Read one frame and echo it back.
		opcode, payload, err2 := bunpyAPI.WSReadFrame(rw.Reader)
		if err2 != nil || opcode == 8 {
			conn.Close()
			return
		}
		bunpyAPI.WSWriteServerFrame(conn, opcode, payload)
		conn.Close()
	}))
	t.Cleanup(srv.Close)
	return "ws://" + srv.Listener.Addr().String()
}

func TestWebSocketSendRecvText(t *testing.T) {
	url := startEchoWSServer(t)
	i := serveInterp(t)
	m := bunpyAPI.BuildWebSocket(i)
	connectFn, _ := m.Dict.GetStr("connect")
	ws, err := connectFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: url},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := ws.(*goipyObject.Instance)

	sendFn, _ := inst.Dict.GetStr("send")
	if _, err2 := sendFn.(*goipyObject.BuiltinFunc).Call(nil, []goipyObject.Object{
		&goipyObject.Str{V: "hello"},
	}, nil); err2 != nil {
		t.Fatal(err2)
	}

	recvFn, _ := inst.Dict.GetStr("recv")
	got, err3 := recvFn.(*goipyObject.BuiltinFunc).Call(nil, nil, nil)
	if err3 != nil {
		t.Fatal(err3)
	}
	if got.(*goipyObject.Str).V != "hello" {
		t.Fatalf("recv = %q, want %q", got.(*goipyObject.Str).V, "hello")
	}
}

func TestWebSocketModuleHasConnect(t *testing.T) {
	i := serveInterp(t)
	m := bunpyAPI.BuildWebSocket(i)
	if _, ok := m.Dict.GetStr("connect"); !ok {
		t.Fatal("bunpy.WebSocket.connect missing")
	}
}

func TestBunpyModuleHasWebSocket(t *testing.T) {
	m := bunpyAPI.BuildBunpy(serveInterp(t))
	if _, ok := m.Dict.GetStr("WebSocket"); !ok {
		t.Fatal("bunpy.WebSocket missing from top-level module")
	}
}

// Use the exported types to avoid unused import errors.
var _ net.Conn
var _ *bufio.Reader

package bunpy

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildWebSocket returns the bunpy.WebSocket module.
func BuildWebSocket(_ *goipyVM.Interp) *goipyObject.Module {
	m := &goipyObject.Module{Name: "bunpy.WebSocket", Dict: goipyObject.NewDict()}
	m.Dict.SetStr("connect", &goipyObject.BuiltinFunc{
		Name: "connect",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("WebSocket.connect() requires a URL argument")
			}
			rawURL, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("WebSocket.connect(): URL must be a str")
			}
			ws, err := wsConnect(rawURL.V)
			if err != nil {
				return nil, fmt.Errorf("WebSocket.connect(): %w", err)
			}
			return buildWSInstance(ws), nil
		},
	})
	return m
}

type wsConn struct {
	conn   net.Conn
	reader *bufio.Reader
}

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func wsConnect(rawURL string) (*wsConn, error) {
	// Convert ws:// to http:// for the upgrade request.
	httpURL := rawURL
	if strings.HasPrefix(rawURL, "ws://") {
		httpURL = "http://" + strings.TrimPrefix(rawURL, "ws://")
	} else if strings.HasPrefix(rawURL, "wss://") {
		return nil, fmt.Errorf("wss:// (TLS WebSocket) is not supported in v0.3.10; use a TLS-terminating proxy")
	}

	// Generate a random Sec-WebSocket-Key.
	keyBytes := make([]byte, 16)
	rand.Read(keyBytes)
	key := base64.StdEncoding.EncodeToString(keyBytes)

	req, err := http.NewRequest("GET", httpURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", key)
	req.Header.Set("Sec-WebSocket-Version", "13")

	host := req.URL.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	if err2 := req.Write(conn); err2 != nil {
		conn.Close()
		return nil, err2
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		conn.Close()
		return nil, err
	}
	resp.Body.Close()

	if resp.StatusCode != 101 {
		conn.Close()
		return nil, fmt.Errorf("WebSocket upgrade failed: %s", resp.Status)
	}

	// Verify Sec-WebSocket-Accept.
	accept := resp.Header.Get("Sec-Websocket-Accept")
	expected := wsAccept(key)
	if accept != expected {
		conn.Close()
		return nil, fmt.Errorf("WebSocket: invalid Sec-WebSocket-Accept header")
	}

	return &wsConn{conn: conn, reader: reader}, nil
}

// WSAccept computes Sec-WebSocket-Accept. Exported for testing.
func WSAccept(key string) string { return wsAccept(key) }

// WSReadFrame reads one WebSocket frame. Exported for testing.
func WSReadFrame(reader *bufio.Reader) (opcode byte, payload []byte, err error) {
	return (&wsConn{reader: reader}).wsRecv()
}

// WSWriteServerFrame writes an unmasked WebSocket frame (server→client). Exported for testing.
func WSWriteServerFrame(w io.Writer, opcode byte, payload []byte) error {
	var frame []byte
	frame = append(frame, 0x80|opcode)
	l := len(payload)
	switch {
	case l <= 125:
		frame = append(frame, byte(l))
	case l <= 65535:
		frame = append(frame, 0x7E, byte(l>>8), byte(l))
	default:
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(l))
		frame = append(frame, 0x7F)
		frame = append(frame, b...)
	}
	frame = append(frame, payload...)
	_, err := w.Write(frame)
	return err
}

func wsAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// wsSend sends a WebSocket frame. opcode: 1=text, 2=binary, 8=close.
func (ws *wsConn) wsSend(opcode byte, payload []byte) error {
	// Client frames must be masked (RFC 6455).
	maskKey := make([]byte, 4)
	rand.Read(maskKey)

	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ maskKey[i%4]
	}

	var frame []byte
	frame = append(frame, 0x80|opcode) // FIN=1 + opcode

	l := len(payload)
	switch {
	case l <= 125:
		frame = append(frame, byte(0x80|l)) // MASK=1 + length
	case l <= 65535:
		frame = append(frame, 0xFE)
		frame = append(frame, byte(l>>8), byte(l))
	default:
		frame = append(frame, 0xFF)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(l))
		frame = append(frame, b...)
	}
	frame = append(frame, maskKey...)
	frame = append(frame, masked...)
	_, err := ws.conn.Write(frame)
	return err
}

// wsRecv reads one WebSocket frame.
func (ws *wsConn) wsRecv() (opcode byte, payload []byte, err error) {
	header := make([]byte, 2)
	if _, err = io.ReadFull(ws.reader, header); err != nil {
		return
	}
	opcode = header[0] & 0x0F
	masked := (header[1] & 0x80) != 0
	payloadLen := int(header[1] & 0x7F)

	switch payloadLen {
	case 126:
		ext := make([]byte, 2)
		io.ReadFull(ws.reader, ext)
		payloadLen = int(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		io.ReadFull(ws.reader, ext)
		payloadLen = int(binary.BigEndian.Uint64(ext))
	}

	var maskKey []byte
	if masked {
		maskKey = make([]byte, 4)
		io.ReadFull(ws.reader, maskKey)
	}

	payload = make([]byte, payloadLen)
	if _, err = io.ReadFull(ws.reader, payload); err != nil {
		return
	}
	if masked {
		for i, b := range payload {
			payload[i] = b ^ maskKey[i%4]
		}
	}
	return
}

func buildWSInstance(ws *wsConn) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "WebSocket", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	inst.Dict.SetStr("send", &goipyObject.BuiltinFunc{
		Name: "send",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("WebSocket.send() requires a message argument")
			}
			switch v := args[0].(type) {
			case *goipyObject.Str:
				if err := ws.wsSend(1, []byte(v.V)); err != nil {
					return nil, err
				}
			case *goipyObject.Bytes:
				if err := ws.wsSend(2, v.V); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("WebSocket.send(): message must be str or bytes")
			}
			return goipyObject.None, nil
		},
	})

	inst.Dict.SetStr("recv", &goipyObject.BuiltinFunc{
		Name: "recv",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			opcode, payload, err := ws.wsRecv()
			if err != nil {
				return nil, err
			}
			switch opcode {
			case 1: // text
				return &goipyObject.Str{V: string(payload)}, nil
			case 2: // binary
				return &goipyObject.Bytes{V: payload}, nil
			case 8: // close
				ws.conn.Close()
				return goipyObject.None, nil
			default:
				return &goipyObject.Bytes{V: payload}, nil
			}
		},
	})

	inst.Dict.SetStr("close", &goipyObject.BuiltinFunc{
		Name: "close",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			ws.wsSend(8, []byte{})
			ws.conn.Close()
			return goipyObject.None, nil
		},
	})

	inst.Dict.SetStr("__enter__", &goipyObject.BuiltinFunc{
		Name: "__enter__",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return inst, nil
		},
	})
	inst.Dict.SetStr("__exit__", &goipyObject.BuiltinFunc{
		Name: "__exit__",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			ws.wsSend(8, []byte{})
			ws.conn.Close()
			return goipyObject.BoolOf(false), nil
		},
	})

	return inst
}

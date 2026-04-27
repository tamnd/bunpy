package bunpy

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
	"sync"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNodeCrypto builds the bunpy.node.crypto module.
func BuildNodeCrypto(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.crypto", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("randomBytes", &goipyObject.BuiltinFunc{
		Name: "randomBytes",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			n := 16
			if len(args) >= 1 {
				if v, ok := args[0].(*goipyObject.Int); ok {
					n = int(v.Int64())
				}
			}
			buf := make([]byte, n)
			if _, err := rand.Read(buf); err != nil {
				return nil, err
			}
			return &goipyObject.Bytes{V: buf}, nil
		},
	})

	mod.Dict.SetStr("randomUUID", &goipyObject.BuiltinFunc{
		Name: "randomUUID",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			var b [16]byte
			if _, err := rand.Read(b[:]); err != nil {
				return nil, err
			}
			b[6] = (b[6] & 0x0f) | 0x40
			b[8] = (b[8] & 0x3f) | 0x80
			uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
				b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
			return &goipyObject.Str{V: uuid}, nil
		},
	})

	mod.Dict.SetStr("createHash", &goipyObject.BuiltinFunc{
		Name: "createHash",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			alg := "sha256"
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					alg = strings.ToLower(s.V)
				}
			}
			return newHashInstance(alg)
		},
	})

	mod.Dict.SetStr("createHmac", &goipyObject.BuiltinFunc{
		Name: "createHmac",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("crypto.createHmac() requires algorithm and key")
			}
			alg := "sha256"
			if s, ok := args[0].(*goipyObject.Str); ok {
				alg = strings.ToLower(s.V)
			}
			var key []byte
			switch v := args[1].(type) {
			case *goipyObject.Str:
				key = []byte(v.V)
			case *goipyObject.Bytes:
				key = v.V
			default:
				return nil, fmt.Errorf("crypto.createHmac(): key must be str or bytes")
			}
			return newHmacInstance(alg, key)
		},
	})

	return mod
}

func newHasherForAlg(alg string) (hash.Hash, error) {
	switch alg {
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	case "sha1":
		return sha1.New(), nil //nolint:gosec
	default:
		return nil, fmt.Errorf("crypto: unsupported algorithm %q", alg)
	}
}

func newHashInstance(alg string) (*goipyObject.Instance, error) {
	h, err := newHasherForAlg(alg)
	if err != nil {
		return nil, err
	}
	type state struct{ mu sync.Mutex; h hash.Hash }
	st := &state{h: h}
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Hash"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("update", &goipyObject.BuiltinFunc{
		Name: "update",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return inst, nil
			}
			var data []byte
			switch v := args[0].(type) {
			case *goipyObject.Str:
				data = []byte(v.V)
			case *goipyObject.Bytes:
				data = v.V
			}
			st.mu.Lock()
			st.h.Write(data)
			st.mu.Unlock()
			return inst, nil
		},
	})
	inst.Dict.SetStr("digest", &goipyObject.BuiltinFunc{
		Name: "digest",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			enc := "hex"
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					enc = strings.ToLower(s.V)
				}
			}
			st.mu.Lock()
			sum := st.h.Sum(nil)
			st.mu.Unlock()
			if enc == "hex" {
				return &goipyObject.Str{V: hex.EncodeToString(sum)}, nil
			}
			return &goipyObject.Bytes{V: sum}, nil
		},
	})
	return inst, nil
}

func newHmacInstance(alg string, key []byte) (*goipyObject.Instance, error) {
	var inner func() hash.Hash
	switch alg {
	case "sha256":
		inner = sha256.New
	case "sha512":
		inner = sha512.New
	case "sha1":
		inner = sha1.New //nolint:gosec
	default:
		return nil, fmt.Errorf("crypto: unsupported algorithm %q", alg)
	}
	type state struct {
		mu sync.Mutex
		h  hash.Hash
	}
	st := &state{h: hmac.New(inner, key)}
	inst := &goipyObject.Instance{
		Class: &goipyObject.Class{Name: "Hmac"},
		Dict:  goipyObject.NewDict(),
	}
	inst.Dict.SetStr("update", &goipyObject.BuiltinFunc{
		Name: "update",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return inst, nil
			}
			var data []byte
			switch v := args[0].(type) {
			case *goipyObject.Str:
				data = []byte(v.V)
			case *goipyObject.Bytes:
				data = v.V
			}
			st.mu.Lock()
			st.h.Write(data)
			st.mu.Unlock()
			return inst, nil
		},
	})
	inst.Dict.SetStr("digest", &goipyObject.BuiltinFunc{
		Name: "digest",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			enc := "hex"
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					enc = strings.ToLower(s.V)
				}
			}
			st.mu.Lock()
			sum := st.h.Sum(nil)
			st.mu.Unlock()
			if enc == "hex" {
				return &goipyObject.Str{V: hex.EncodeToString(sum)}, nil
			}
			return &goipyObject.Bytes{V: sum}, nil
		},
	})
	return inst, nil
}

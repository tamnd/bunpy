package bunpy

import (
	"encoding/base64"
	"fmt"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildBase64 builds the bunpy.base64 module.
//
// Python surface:
//
//	bunpy.base64.encode(data: bytes | str) -> str
//	bunpy.base64.decode(s: str) -> bytes
//	bunpy.base64.encode_url(data: bytes | str) -> str   # URL-safe, no padding
//	bunpy.base64.decode_url(s: str) -> bytes
func BuildBase64(i *goipyVM.Interp) *goipyObject.Module {
	m := &goipyObject.Module{Name: "bunpy.base64", Dict: goipyObject.NewDict()}

	m.Dict.SetStr("encode", &goipyObject.BuiltinFunc{
		Name: "encode",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			data, err := toBytes(args, "encode")
			if err != nil {
				return nil, err
			}
			return &goipyObject.Str{V: base64.StdEncoding.EncodeToString(data)}, nil
		},
	})

	m.Dict.SetStr("decode", &goipyObject.BuiltinFunc{
		Name: "decode",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			s, err := toString(args, "decode")
			if err != nil {
				return nil, err
			}
			b, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				// try without padding
				b, err = base64.RawStdEncoding.DecodeString(s)
				if err != nil {
					return nil, fmt.Errorf("bunpy.base64.decode: invalid base64: %w", err)
				}
			}
			return &goipyObject.Bytes{V: b}, nil
		},
	})

	m.Dict.SetStr("encode_url", &goipyObject.BuiltinFunc{
		Name: "encode_url",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			data, err := toBytes(args, "encode_url")
			if err != nil {
				return nil, err
			}
			return &goipyObject.Str{V: base64.RawURLEncoding.EncodeToString(data)}, nil
		},
	})

	m.Dict.SetStr("decode_url", &goipyObject.BuiltinFunc{
		Name: "decode_url",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			s, err := toString(args, "decode_url")
			if err != nil {
				return nil, err
			}
			b, err := base64.RawURLEncoding.DecodeString(s)
			if err != nil {
				return nil, fmt.Errorf("bunpy.base64.decode_url: invalid base64url: %w", err)
			}
			return &goipyObject.Bytes{V: b}, nil
		},
	})

	// Convenience: expose the interp so callers can raise TypeError on bad input.
	_ = i

	return m
}

// toBytes extracts a []byte from args[0], accepting bytes or str.
func toBytes(args []goipyObject.Object, fn string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("bunpy.base64.%s() takes exactly 1 argument (%d given)", fn, len(args))
	}
	switch v := args[0].(type) {
	case *goipyObject.Bytes:
		return v.V, nil
	case *goipyObject.Str:
		return []byte(v.V), nil
	default:
		return nil, fmt.Errorf("bunpy.base64.%s(): argument must be bytes or str, not %T", fn, args[0])
	}
}

// toString extracts a string from args[0].
func toString(args []goipyObject.Object, fn string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("bunpy.base64.%s() takes exactly 1 argument (%d given)", fn, len(args))
	}
	v, ok := args[0].(*goipyObject.Str)
	if !ok {
		return "", fmt.Errorf("bunpy.base64.%s(): argument must be str, not %T", fn, args[0])
	}
	return v.V, nil
}

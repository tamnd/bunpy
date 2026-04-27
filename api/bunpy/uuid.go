package bunpy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func BuildUUID(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.uuid", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("v4", &goipyObject.BuiltinFunc{
		Name: "v4",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			u, err := uuidV4()
			if err != nil {
				return nil, fmt.Errorf("uuid.v4(): %w", err)
			}
			return &goipyObject.Str{V: u}, nil
		},
	})

	mod.Dict.SetStr("v7", &goipyObject.BuiltinFunc{
		Name: "v7",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			u, err := uuidV7()
			if err != nil {
				return nil, fmt.Errorf("uuid.v7(): %w", err)
			}
			return &goipyObject.Str{V: u}, nil
		},
	})

	mod.Dict.SetStr("is_valid", &goipyObject.BuiltinFunc{
		Name: "is_valid",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("uuid.is_valid() requires one argument")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return goipyObject.BoolOf(false), nil
			}
			return goipyObject.BoolOf(uuidRE.MatchString(s.V)), nil
		},
	})

	return mod
}

func uuidV4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// version 4
	b[6] = (b[6] & 0x0f) | 0x40
	// variant bits (RFC 4122)
	b[8] = (b[8] & 0x3f) | 0x80
	return formatUUID(b), nil
}

func uuidV7() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	ms := uint64(time.Now().UnixMilli())
	// top 48 bits: timestamp in milliseconds
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	// version 7
	b[6] = (b[6] & 0x0f) | 0x70
	// variant bits
	b[8] = (b[8] & 0x3f) | 0x80
	return formatUUID(b), nil
}

func formatUUID(b [16]byte) string {
	var buf [36]byte
	h := hex.EncodeToString(b[:])
	copy(buf[0:], h[0:8])
	buf[8] = '-'
	copy(buf[9:], h[8:12])
	buf[13] = '-'
	copy(buf[14:], h[12:16])
	buf[18] = '-'
	copy(buf[19:], h[16:20])
	buf[23] = '-'
	copy(buf[24:], h[20:32])
	return string(buf[:])
}

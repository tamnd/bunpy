package bunpy

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildCSRF(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.csrf", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("token", &goipyObject.BuiltinFunc{
		Name: "token",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			secret := ""
			if len(args) >= 1 {
				if s, ok := args[0].(*goipyObject.Str); ok {
					secret = s.V
				}
			}
			if kwargs != nil {
				if v, ok := kwargs.GetStr("secret"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						secret = s.V
					}
				}
			}
			raw := make([]byte, 16)
			if _, err := rand.Read(raw); err != nil {
				return nil, fmt.Errorf("csrf.token(): rand error: %w", err)
			}
			nonce := hex.EncodeToString(raw)
			if secret == "" {
				return &goipyObject.Str{V: nonce}, nil
			}
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(nonce))
			sig := hex.EncodeToString(mac.Sum(nil))
			return &goipyObject.Str{V: nonce + "." + sig}, nil
		},
	})

	mod.Dict.SetStr("verify", &goipyObject.BuiltinFunc{
		Name: "verify",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("csrf.verify() requires a token")
			}
			tokenStr, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("csrf.verify(): token must be str")
			}
			secret := ""
			if len(args) >= 2 {
				if s, ok2 := args[1].(*goipyObject.Str); ok2 {
					secret = s.V
				}
			}
			if kwargs != nil {
				if v, ok2 := kwargs.GetStr("secret"); ok2 {
					if s, ok3 := v.(*goipyObject.Str); ok3 {
						secret = s.V
					}
				}
			}
			if secret == "" {
				return goipyObject.BoolOf(tokenStr.V != ""), nil
			}
			dot := -1
			for i, c := range tokenStr.V {
				if c == '.' {
					dot = i
					break
				}
			}
			if dot < 0 {
				return goipyObject.BoolOf(false), nil
			}
			nonce := tokenStr.V[:dot]
			sig := tokenStr.V[dot+1:]
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(nonce))
			expected := hex.EncodeToString(mac.Sum(nil))
			return goipyObject.BoolOf(hmac.Equal([]byte(sig), []byte(expected))), nil
		},
	})

	return mod
}

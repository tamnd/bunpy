package bunpy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildJWT(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.jwt", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("sign", &goipyObject.BuiltinFunc{
		Name: "sign",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("jwt.sign() requires claims dict and secret")
			}
			claimsDict, ok := args[0].(*goipyObject.Dict)
			if !ok {
				return nil, fmt.Errorf("jwt.sign(): first argument must be a dict")
			}
			secret, err := toBytesArg(args[1], "sign")
			if err != nil {
				return nil, err
			}

			claims := dictToGoMap(claimsDict)
			now := time.Now().Unix()
			claims["iat"] = now

			if kwargs != nil {
				if expObj, ok := kwargs.GetStr("exp"); ok {
					switch v := expObj.(type) {
					case *goipyObject.Int:
						claims["exp"] = now + v.Int64()
					case *goipyObject.Float:
						claims["exp"] = now + int64(v.V)
					}
				}
			}

			token, err := jwtSign(claims, secret)
			if err != nil {
				return nil, fmt.Errorf("jwt.sign(): %w", err)
			}
			return &goipyObject.Str{V: token}, nil
		},
	})

	mod.Dict.SetStr("verify", &goipyObject.BuiltinFunc{
		Name: "verify",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("jwt.verify() requires token and secret")
			}
			token, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("jwt.verify(): token must be str")
			}
			secret, err := toBytesArg(args[1], "verify")
			if err != nil {
				return nil, err
			}
			claims, err := jwtVerify(token.V, secret)
			if err != nil {
				return nil, err
			}
			return goMapToDict(claims), nil
		},
	})

	mod.Dict.SetStr("decode", &goipyObject.BuiltinFunc{
		Name: "decode",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("jwt.decode() requires a token")
			}
			token, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("jwt.decode(): token must be str")
			}
			claims, err := jwtDecode(token.V)
			if err != nil {
				return nil, err
			}
			return goMapToDict(claims), nil
		},
	})

	return mod
}

func jwtSign(claims map[string]any, secret []byte) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	hdr, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pay, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	h := base64.RawURLEncoding.EncodeToString(hdr)
	p := base64.RawURLEncoding.EncodeToString(pay)
	msg := h + "." + p
	sig := jwtHMAC([]byte(msg), secret)
	return msg + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func jwtVerify(token string, secret []byte) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("jwt.verify(): malformed token")
	}
	msg := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("jwt.verify(): bad signature encoding")
	}
	expected := jwtHMAC([]byte(msg), secret)
	if !hmacEqual(sig, expected) {
		return nil, fmt.Errorf("jwt.verify(): signature mismatch")
	}
	claims, err := jwtDecode(token)
	if err != nil {
		return nil, err
	}
	if exp, ok := claims["exp"]; ok {
		expF, ok2 := exp.(float64)
		if ok2 && time.Now().Unix() > int64(expF) {
			return nil, fmt.Errorf("jwt.verify(): token expired")
		}
	}
	return claims, nil
}

func jwtDecode(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("jwt.decode(): malformed token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("jwt.decode(): bad payload encoding")
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("jwt.decode(): bad payload JSON")
	}
	return claims, nil
}

func jwtHMAC(msg, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	return mac.Sum(nil)
}

func hmacEqual(a, b []byte) bool {
	return hmac.Equal(a, b)
}

func dictToGoMap(d *goipyObject.Dict) map[string]any {
	keys, vals := d.Items()
	m := make(map[string]any, len(keys))
	for i, k := range keys {
		if ks, ok := k.(*goipyObject.Str); ok {
			m[ks.V] = pyObjToGoValue(vals[i])
		}
	}
	return m
}

func goMapToDict(m map[string]any) *goipyObject.Dict {
	d := goipyObject.NewDict()
	for k, v := range m {
		d.SetStr(k, goValueToPyObj(v))
	}
	return d
}

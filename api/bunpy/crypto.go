package bunpy

import (
	"crypto/aes"
	"crypto/cipher"
	gocrypto "crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildCrypto(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.crypto", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("random", &goipyObject.BuiltinFunc{
		Name: "random",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			n := int64(32)
			if len(args) >= 1 {
				switch v := args[0].(type) {
				case *goipyObject.Int:
					n = v.Int64()
				}
			}
			b := make([]byte, n)
			if _, err := io.ReadFull(rand.Reader, b); err != nil {
				return nil, fmt.Errorf("crypto.random(): %w", err)
			}
			return &goipyObject.Bytes{V: b}, nil
		},
	})

	mod.Dict.SetStr("encrypt", &goipyObject.BuiltinFunc{
		Name: "encrypt",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("crypto.encrypt() requires plaintext and key")
			}
			pt, err := toBytesArg(args[0], "encrypt")
			if err != nil {
				return nil, err
			}
			key, err := toBytesArg(args[1], "encrypt")
			if err != nil {
				return nil, err
			}
			ct, err := aesGCMEncrypt(pt, key)
			if err != nil {
				return nil, fmt.Errorf("crypto.encrypt(): %w", err)
			}
			return &goipyObject.Bytes{V: ct}, nil
		},
	})

	mod.Dict.SetStr("decrypt", &goipyObject.BuiltinFunc{
		Name: "decrypt",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("crypto.decrypt() requires ciphertext and key")
			}
			ct, err := toBytesArg(args[0], "decrypt")
			if err != nil {
				return nil, err
			}
			key, err := toBytesArg(args[1], "decrypt")
			if err != nil {
				return nil, err
			}
			pt, err := aesGCMDecrypt(ct, key)
			if err != nil {
				return nil, fmt.Errorf("crypto.decrypt(): %w", err)
			}
			return &goipyObject.Bytes{V: pt}, nil
		},
	})

	mod.Dict.SetStr("hmac", &goipyObject.BuiltinFunc{
		Name: "hmac",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("crypto.hmac() requires message and key")
			}
			msg, err := toBytesArg(args[0], "hmac")
			if err != nil {
				return nil, err
			}
			key, err := toBytesArg(args[1], "hmac")
			if err != nil {
				return nil, err
			}
			mac := gocrypto.New(sha256.New, key)
			mac.Write(msg)
			return &goipyObject.Bytes{V: mac.Sum(nil)}, nil
		},
	})

	mod.Dict.SetStr("hmac_verify", &goipyObject.BuiltinFunc{
		Name: "hmac_verify",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 3 {
				return nil, fmt.Errorf("crypto.hmac_verify() requires message, key, and signature")
			}
			msg, err := toBytesArg(args[0], "hmac_verify")
			if err != nil {
				return nil, err
			}
			key, err := toBytesArg(args[1], "hmac_verify")
			if err != nil {
				return nil, err
			}
			sig, err := toBytesArg(args[2], "hmac_verify")
			if err != nil {
				return nil, err
			}
			mac := gocrypto.New(sha256.New, key)
			mac.Write(msg)
			expected := mac.Sum(nil)
			return goipyObject.BoolOf(subtle.ConstantTimeCompare(sig, expected) == 1), nil
		},
	})

	mod.Dict.SetStr("sha256", &goipyObject.BuiltinFunc{
		Name: "sha256",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.sha256() requires data")
			}
			data, err := toBytesArg(args[0], "sha256")
			if err != nil {
				return nil, err
			}
			h := sha256.Sum256(data)
			return &goipyObject.Bytes{V: h[:]}, nil
		},
	})

	mod.Dict.SetStr("sha512", &goipyObject.BuiltinFunc{
		Name: "sha512",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.sha512() requires data")
			}
			data, err := toBytesArg(args[0], "sha512")
			if err != nil {
				return nil, err
			}
			h := sha512.Sum512(data)
			return &goipyObject.Bytes{V: h[:]}, nil
		},
	})

	mod.Dict.SetStr("sha256_hex", &goipyObject.BuiltinFunc{
		Name: "sha256_hex",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.sha256_hex() requires data")
			}
			data, err := toBytesArg(args[0], "sha256_hex")
			if err != nil {
				return nil, err
			}
			h := sha256.Sum256(data)
			return &goipyObject.Str{V: hex.EncodeToString(h[:])}, nil
		},
	})

	mod.Dict.SetStr("sha512_hex", &goipyObject.BuiltinFunc{
		Name: "sha512_hex",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("crypto.sha512_hex() requires data")
			}
			data, err := toBytesArg(args[0], "sha512_hex")
			if err != nil {
				return nil, err
			}
			h := sha512.Sum512(data)
			return &goipyObject.Str{V: hex.EncodeToString(h[:])}, nil
		},
	})

	return mod
}

func toBytesArg(obj goipyObject.Object, fn string) ([]byte, error) {
	switch v := obj.(type) {
	case *goipyObject.Bytes:
		return v.V, nil
	case *goipyObject.Str:
		return []byte(v.V), nil
	default:
		return nil, fmt.Errorf("crypto.%s(): argument must be bytes or str", fn)
	}
}

func aesGCMEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ct...), nil
}

func aesGCMDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

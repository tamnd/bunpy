package bunpy

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildPassword returns the bunpy.password module.
func BuildPassword(_ *goipyVM.Interp) *goipyObject.Module {
	m := &goipyObject.Module{Name: "bunpy.password", Dict: goipyObject.NewDict()}

	m.Dict.SetStr("hash", &goipyObject.BuiltinFunc{
		Name: "hash",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			pw, err := passwordArg("hash", args, kwargs)
			if err != nil {
				return nil, err
			}
			algo := passwordAlgo(kwargs)
			switch algo {
			case "argon2id":
				return argon2idHash(pw, kwargs)
			default: // bcrypt
				return bcryptHash(pw, kwargs)
			}
		},
	})

	m.Dict.SetStr("verify", &goipyObject.BuiltinFunc{
		Name: "verify",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("bunpy.password.verify() requires password and hash arguments")
			}
			pw, ok1 := args[0].(*goipyObject.Str)
			hash, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("bunpy.password.verify(): password and hash must be str")
			}
			ok, err := verifyPassword(pw.V, hash.V)
			if err != nil {
				return goipyObject.BoolOf(false), nil
			}
			return goipyObject.BoolOf(ok), nil
		},
	})

	return m
}

func passwordArg(fn string, args []goipyObject.Object, kwargs *goipyObject.Dict) (string, error) {
	if len(args) >= 1 {
		if s, ok := args[0].(*goipyObject.Str); ok {
			return s.V, nil
		}
	}
	if kwargs != nil {
		if v, ok := kwargs.GetStr("password"); ok {
			if s, ok2 := v.(*goipyObject.Str); ok2 {
				return s.V, nil
			}
		}
	}
	return "", fmt.Errorf("bunpy.password.%s() requires a password argument", fn)
}

func passwordAlgo(kwargs *goipyObject.Dict) string {
	if kwargs != nil {
		if v, ok := kwargs.GetStr("algo"); ok {
			if s, ok2 := v.(*goipyObject.Str); ok2 {
				return s.V
			}
		}
	}
	return "bcrypt"
}

func bcryptHash(password string, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
	cost := bcrypt.DefaultCost
	if kwargs != nil {
		if v, ok := kwargs.GetStr("cost"); ok {
			if n, ok2 := v.(*goipyObject.Int); ok2 {
				cost = int(n.Int64())
			}
		}
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return nil, fmt.Errorf("bcrypt.hash(): %w", err)
	}
	return &goipyObject.Str{V: string(hashed)}, nil
}

// argon2idHash hashes with Argon2id and encodes as a PHC string.
func argon2idHash(password string, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
	memory := uint32(64 * 1024) // 64 MiB
	time_ := uint32(1)
	threads := uint8(4)
	keyLen := uint32(32)

	if kwargs != nil {
		if v, ok := kwargs.GetStr("memory"); ok {
			if n, ok2 := v.(*goipyObject.Int); ok2 {
				memory = uint32(n.Int64())
			}
		}
		if v, ok := kwargs.GetStr("time"); ok {
			if n, ok2 := v.(*goipyObject.Int); ok2 {
				time_ = uint32(n.Int64())
			}
		}
		if v, ok := kwargs.GetStr("threads"); ok {
			if n, ok2 := v.(*goipyObject.Int); ok2 {
				threads = uint8(n.Int64())
			}
		}
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("argon2id: failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, time_, memory, threads, keyLen)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory, time_, threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return &goipyObject.Str{V: encoded}, nil
}

func verifyPassword(password, encoded string) (bool, error) {
	if strings.HasPrefix(encoded, "$argon2id$") {
		return verifyArgon2id(password, encoded)
	}
	// Assume bcrypt
	err := bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password))
	return err == nil, nil
}

func verifyArgon2id(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	// $argon2id$v=N$m=M,t=T,p=P$salt$hash
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid argon2id hash format")
	}

	var memory, time_, version uint32
	var threads uint8
	fmt.Sscanf(parts[2], "v=%d", &version)
	fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time_, &threads)

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	computed := argon2.IDKey([]byte(password), salt, time_, memory, threads, uint32(len(storedHash)))
	return subtle.ConstantTimeCompare(computed, storedHash) == 1, nil
}

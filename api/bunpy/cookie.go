package bunpy

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildCookie(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.cookie", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("parse", &goipyObject.BuiltinFunc{
		Name: "parse",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("cookie.parse() requires a Cookie header string")
			}
			s, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("cookie.parse(): argument must be str")
			}
			d := goipyObject.NewDict()
			for _, pair := range strings.Split(s.V, ";") {
				pair = strings.TrimSpace(pair)
				if pair == "" {
					continue
				}
				k, v, _ := strings.Cut(pair, "=")
				d.SetStr(strings.TrimSpace(k), &goipyObject.Str{V: strings.TrimSpace(v)})
			}
			return d, nil
		},
	})

	mod.Dict.SetStr("serialize", &goipyObject.BuiltinFunc{
		Name: "serialize",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("cookie.serialize() requires name and value")
			}
			name, ok1 := args[0].(*goipyObject.Str)
			val, ok2 := args[1].(*goipyObject.Str)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("cookie.serialize(): name and value must be str")
			}
			c := &http.Cookie{Name: name.V, Value: val.V}
			if kwargs != nil {
				if v, ok := kwargs.GetStr("path"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						c.Path = s.V
					}
				}
				if v, ok := kwargs.GetStr("domain"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						c.Domain = s.V
					}
				}
				if v, ok := kwargs.GetStr("max_age"); ok {
					if iv, ok2 := v.(*goipyObject.Int); ok2 {
						c.MaxAge = int(iv.Int64())
					}
				}
				if v, ok := kwargs.GetStr("http_only"); ok {
					if bv, ok2 := v.(*goipyObject.Bool); ok2 {
						c.HttpOnly = bv.V
					}
				}
				if v, ok := kwargs.GetStr("secure"); ok {
					if bv, ok2 := v.(*goipyObject.Bool); ok2 {
						c.Secure = bv.V
					}
				}
				if v, ok := kwargs.GetStr("same_site"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						switch strings.ToLower(s.V) {
						case "strict":
							c.SameSite = http.SameSiteStrictMode
						case "lax":
							c.SameSite = http.SameSiteLaxMode
						case "none":
							c.SameSite = http.SameSiteNoneMode
						}
					}
				}
				if v, ok := kwargs.GetStr("expires"); ok {
					if s, ok2 := v.(*goipyObject.Str); ok2 {
						t, err := time.Parse(time.RFC3339, s.V)
						if err == nil {
							c.Expires = t
						}
					}
				}
			}
			return &goipyObject.Str{V: c.String()}, nil
		},
	})

	return mod
}

package bunpy

import (
	"fmt"
	"net"
	"strings"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

func BuildDNS(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.dns", Dict: goipyObject.NewDict()}

	mod.Dict.SetStr("resolve", &goipyObject.BuiltinFunc{
		Name: "resolve",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			host, rtype, err := dnsArgs(args, kwargs)
			if err != nil {
				return nil, err
			}
			switch strings.ToUpper(rtype) {
			case "A", "AAAA", "":
				addrs, err2 := net.LookupHost(host)
				if err2 != nil {
					return nil, fmt.Errorf("dns.resolve(%q): %w", host, err2)
				}
				return strSliceToPyList(addrs), nil
			case "MX":
				records, err2 := net.LookupMX(host)
				if err2 != nil {
					return nil, fmt.Errorf("dns.resolve(%q, MX): %w", host, err2)
				}
				items := make([]goipyObject.Object, len(records))
				for i, r := range records {
					d := goipyObject.NewDict()
					d.SetStr("host", &goipyObject.Str{V: r.Host})
					d.SetStr("pref", goipyObject.NewInt(int64(r.Pref)))
					items[i] = d
				}
				return &goipyObject.List{V: items}, nil
			case "NS":
				records, err2 := net.LookupNS(host)
				if err2 != nil {
					return nil, fmt.Errorf("dns.resolve(%q, NS): %w", host, err2)
				}
				items := make([]goipyObject.Object, len(records))
				for i, r := range records {
					d := goipyObject.NewDict()
					d.SetStr("host", &goipyObject.Str{V: r.Host})
					items[i] = d
				}
				return &goipyObject.List{V: items}, nil
			case "TXT":
				records, err2 := net.LookupTXT(host)
				if err2 != nil {
					return nil, fmt.Errorf("dns.resolve(%q, TXT): %w", host, err2)
				}
				return strSliceToPyList(records), nil
			case "CNAME":
				cname, err2 := net.LookupCNAME(host)
				if err2 != nil {
					return nil, fmt.Errorf("dns.resolve(%q, CNAME): %w", host, err2)
				}
				return &goipyObject.List{V: []goipyObject.Object{&goipyObject.Str{V: cname}}}, nil
			case "PTR":
				names, err2 := net.LookupAddr(host)
				if err2 != nil {
					return nil, fmt.Errorf("dns.resolve(%q, PTR): %w", host, err2)
				}
				return strSliceToPyList(names), nil
			default:
				return nil, fmt.Errorf("dns.resolve(): unsupported record type %q", rtype)
			}
		},
	})

	mod.Dict.SetStr("lookup", &goipyObject.BuiltinFunc{
		Name: "lookup",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("dns.lookup() requires a hostname")
			}
			host, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("dns.lookup(): hostname must be str")
			}
			addrs, err := net.LookupHost(host.V)
			if err != nil {
				return nil, fmt.Errorf("dns.lookup(%q): %w", host.V, err)
			}
			if len(addrs) == 0 {
				return goipyObject.None, nil
			}
			return &goipyObject.Str{V: addrs[0]}, nil
		},
	})

	mod.Dict.SetStr("reverse", &goipyObject.BuiltinFunc{
		Name: "reverse",
		Call: func(_ any, args []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("dns.reverse() requires an IP address")
			}
			ip, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("dns.reverse(): IP must be str")
			}
			names, err := net.LookupAddr(ip.V)
			if err != nil {
				return nil, fmt.Errorf("dns.reverse(%q): %w", ip.V, err)
			}
			return strSliceToPyList(names), nil
		},
	})

	return mod
}

func dnsArgs(args []goipyObject.Object, kwargs *goipyObject.Dict) (string, string, error) {
	if len(args) < 1 {
		return "", "", fmt.Errorf("dns.resolve() requires a hostname")
	}
	host, ok := args[0].(*goipyObject.Str)
	if !ok {
		return "", "", fmt.Errorf("dns.resolve(): hostname must be str")
	}
	rtype := ""
	if len(args) >= 2 {
		if s, ok2 := args[1].(*goipyObject.Str); ok2 {
			rtype = s.V
		}
	}
	if kwargs != nil {
		if v, ok2 := kwargs.GetStr("type"); ok2 {
			if s, ok3 := v.(*goipyObject.Str); ok3 {
				rtype = s.V
			}
		}
	}
	return host.V, rtype, nil
}

func strSliceToPyList(ss []string) *goipyObject.List {
	items := make([]goipyObject.Object, len(ss))
	for i, s := range ss {
		items[i] = &goipyObject.Str{V: s}
	}
	return &goipyObject.List{V: items}
}

package bunpy

import (
	"net"
	"os"
	"runtime"
	"time"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildNodeOS builds the bunpy.node.os module (Node.js os API shape).
func BuildNodeOS(_ *goipyVM.Interp) *goipyObject.Module {
	mod := &goipyObject.Module{Name: "bunpy.node.os", Dict: goipyObject.NewDict()}

	startTime := time.Now()

	eol := "\n"
	if runtime.GOOS == "windows" {
		eol = "\r\n"
	}
	mod.Dict.SetStr("EOL", &goipyObject.Str{V: eol})

	mod.Dict.SetStr("platform", &goipyObject.BuiltinFunc{
		Name: "platform",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return &goipyObject.Str{V: runtime.GOOS}, nil
		},
	})

	mod.Dict.SetStr("arch", &goipyObject.BuiltinFunc{
		Name: "arch",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			arch := runtime.GOARCH
			if arch == "amd64" {
				arch = "x64"
			}
			return &goipyObject.Str{V: arch}, nil
		},
	})

	mod.Dict.SetStr("hostname", &goipyObject.BuiltinFunc{
		Name: "hostname",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			h, err := os.Hostname()
			if err != nil {
				return &goipyObject.Str{V: ""}, nil
			}
			return &goipyObject.Str{V: h}, nil
		},
	})

	mod.Dict.SetStr("homedir", &goipyObject.BuiltinFunc{
		Name: "homedir",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			h, err := os.UserHomeDir()
			if err != nil {
				return &goipyObject.Str{V: ""}, nil
			}
			return &goipyObject.Str{V: h}, nil
		},
	})

	mod.Dict.SetStr("tmpdir", &goipyObject.BuiltinFunc{
		Name: "tmpdir",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return &goipyObject.Str{V: os.TempDir()}, nil
		},
	})

	mod.Dict.SetStr("cpus", &goipyObject.BuiltinFunc{
		Name: "cpus",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			n := runtime.NumCPU()
			items := make([]goipyObject.Object, n)
			for i := range items {
				d := goipyObject.NewDict()
				d.SetStr("model", &goipyObject.Str{V: "unknown"})
				d.SetStr("speed", goipyObject.NewInt(0))
				items[i] = d
			}
			return &goipyObject.List{V: items}, nil
		},
	})

	mod.Dict.SetStr("freemem", &goipyObject.BuiltinFunc{
		Name: "freemem",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			return goipyObject.NewInt(int64(ms.Frees)), nil
		},
	})

	mod.Dict.SetStr("totalmem", &goipyObject.BuiltinFunc{
		Name: "totalmem",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			return goipyObject.NewInt(int64(ms.Sys)), nil
		},
	})

	mod.Dict.SetStr("uptime", &goipyObject.BuiltinFunc{
		Name: "uptime",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			return &goipyObject.Float{V: time.Since(startTime).Seconds()}, nil
		},
	})

	mod.Dict.SetStr("networkInterfaces", &goipyObject.BuiltinFunc{
		Name: "networkInterfaces",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			ifaces, err := net.Interfaces()
			result := goipyObject.NewDict()
			if err != nil {
				return result, nil
			}
			for _, iface := range ifaces {
				addrs, err := iface.Addrs()
				if err != nil {
					continue
				}
				items := make([]goipyObject.Object, 0, len(addrs))
				for _, addr := range addrs {
					d := goipyObject.NewDict()
					d.SetStr("address", &goipyObject.Str{V: addr.String()})
					d.SetStr("family", &goipyObject.Str{V: "IPv4"})
					d.SetStr("internal", goipyObject.BoolOf(iface.Flags&net.FlagLoopback != 0))
					items = append(items, d)
				}
				result.SetStr(iface.Name, &goipyObject.List{V: items})
			}
			return result, nil
		},
	})

	return mod
}

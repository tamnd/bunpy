// Package bunpy exposes the bunpy.* built-in API modules to the goipy VM.
//
// Each sub-module is a function that builds an *object.Module from a
// *vm.Interp. The map returned by Modules() is assigned to
// interp.NativeModules before the interpreter runs user code.
//
// InjectGlobals injects web-standard globals (fetch, URL, Request,
// Response) directly into interp.Builtins so they are available without
// any import statement.
//
// Sub-module layout:
//
//	bunpy              -- top-level namespace (base64, gzip, version)
//	bunpy.base64       -- bunpy.base64.encode / .decode
//	bunpy.gzip         -- bunpy.gzip.compress / .decompress
//	bunpy._fetch       -- internal module backing web globals
package bunpy

import (
	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// Version is baked in by the bunpy build pipeline.
const Version = "0.7.1"

// Modules returns the NativeModules map for the current v0.3.1 surface.
// Later rungs extend this map by adding more entries.
func Modules() map[string]func(*goipyVM.Interp) *goipyObject.Module {
	return map[string]func(*goipyVM.Interp) *goipyObject.Module{
		"bunpy":        BuildBunpy,
		"bunpy.base64": BuildBase64,
		"bunpy.gzip":   BuildGzip,
		"bunpy._fetch": BuildFetch,
		"bunpy.redis":    BuildRedis,
		"bunpy.s3":       BuildS3,
		"bunpy.WebSocket": BuildWebSocket,
		"bunpy.password": BuildPassword,
		"bunpy.env":      BuildEnv,
		"bunpy.log":      BuildLog,
		"bunpy.uuid":     BuildUUID,
		"bunpy.crypto":   BuildCrypto,
		"bunpy.jwt":      BuildJWT,
		"bunpy.csv":      BuildCSV,
		"bunpy.template": BuildTemplate,
		"bunpy.email":    BuildEmail,
		"bunpy.cache":    BuildCache,
		"bunpy.queue":    BuildQueue,
		"bunpy.http":     BuildHTTP,
		"bunpy.config":   BuildConfig,
		"bunpy.dns":         BuildDNS,
		"bunpy.semver":      BuildSemver,
		"bunpy.deep_equals":   BuildDeepEquals,
		"bunpy.escape_html":   BuildEscapeHTML,
		"bunpy.HTMLRewriter":  BuildHTMLRewriter,
		"bunpy.cookie":        BuildCookie,
		"bunpy.csrf":          BuildCSRF,
		"bunpy.yaml":          BuildYAML,
		"bunpy.URLPattern":    BuildURLPattern,
		"bunpy.Worker":           BuildWorker,
		"bunpy.terminal":         BuildTerminal,
		"bunpy.set_system_time":  BuildSetSystemTime,
		"bunpy.expect":           BuildExpect,
		"bunpy.mock":             BuildMock,
		"bunpy.snapshot":         BuildSnapshot,
		"bunpy.asyncio":          BuildAsyncio,
	}
}

// InjectGlobals adds web-standard globals to interp.Builtins so Python
// scripts can call fetch(), URL(), Request(), and Response() without an
// import statement.
func InjectGlobals(i *goipyVM.Interp) {
	fetchMod := BuildFetch(i)
	for _, name := range []string{"fetch", "URL", "Request", "Response", "Headers"} {
		if v, ok := fetchMod.Dict.GetStr(name); ok {
			i.Builtins.SetStr(name, v)
		}
	}
	InjectTimerGlobals(i)
}

// BuildBunpy builds the top-level `bunpy` module. It contains sub-module
// references and the version string.
func BuildBunpy(i *goipyVM.Interp) *goipyObject.Module {
	m := &goipyObject.Module{Name: "bunpy", Dict: goipyObject.NewDict()}
	m.Dict.SetStr("__version__", &goipyObject.Str{V: Version})
	m.Dict.SetStr("__name__", &goipyObject.Str{V: "bunpy"})

	// Attach sub-modules and functions.
	m.Dict.SetStr("base64", BuildBase64(i))
	m.Dict.SetStr("gzip", BuildGzip(i))
	m.Dict.SetStr("serve", BuildServe(i))
	m.Dict.SetStr("file", BuildFile(i))
	m.Dict.SetStr("write", BuildWrite(i))
	m.Dict.SetStr("read", BuildRead(i))
	m.Dict.SetStr("shell", BuildShell(i))
	m.Dict.SetStr("spawn", BuildSpawn(i))
	m.Dict.SetStr("dollar", BuildDollar(i))
	m.Dict.SetStr("glob", BuildGlob(i))
	m.Dict.SetStr("glob_match", BuildGlobMatch(i))
	m.Dict.SetStr("sql", BuildSQL(i))
	m.Dict.SetStr("redis", BuildRedis(i))
	m.Dict.SetStr("s3", BuildS3(i))
	m.Dict.SetStr("WebSocket", BuildWebSocket(i))
	m.Dict.SetStr("cron", BuildCron(i))
	m.Dict.SetStr("password", BuildPassword(i))
	m.Dict.SetStr("env", BuildEnv(i))
	m.Dict.SetStr("log", BuildLog(i))
	m.Dict.SetStr("uuid", BuildUUID(i))
	m.Dict.SetStr("crypto", BuildCrypto(i))
	m.Dict.SetStr("jwt", BuildJWT(i))
	m.Dict.SetStr("csv", BuildCSV(i))
	m.Dict.SetStr("template", BuildTemplate(i))
	m.Dict.SetStr("email", BuildEmail(i))
	m.Dict.SetStr("cache", BuildCache(i))
	m.Dict.SetStr("queue", BuildQueue(i))
	m.Dict.SetStr("http", BuildHTTP(i))
	m.Dict.SetStr("config", BuildConfig(i))
	m.Dict.SetStr("dns", BuildDNS(i))
	m.Dict.SetStr("semver", BuildSemver(i))
	m.Dict.SetStr("deep_equals", BuildDeepEquals(i))
	m.Dict.SetStr("escape_html", BuildEscapeHTML(i))
	m.Dict.SetStr("HTMLRewriter", BuildHTMLRewriter(i))
	m.Dict.SetStr("cookie", BuildCookie(i))
	m.Dict.SetStr("csrf", BuildCSRF(i))
	m.Dict.SetStr("yaml", BuildYAML(i))
	m.Dict.SetStr("URLPattern", BuildURLPattern(i))
	m.Dict.SetStr("Worker", BuildWorker(i))
	m.Dict.SetStr("terminal", BuildTerminal(i))
	m.Dict.SetStr("set_system_time", BuildSetSystemTime(i))
	m.Dict.SetStr("expect", BuildExpect(i))
	m.Dict.SetStr("mock", BuildMock(i))
	m.Dict.SetStr("snapshot", BuildSnapshot(i))
	m.Dict.SetStr("asyncio", BuildAsyncio(i))

	return m
}

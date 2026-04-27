package marker

import (
	"strings"
	"testing"
)

func TestParseEmpty(t *testing.T) {
	e, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if !e.Eval(Env{}) {
		t.Error("empty marker should be true")
	}
}

func TestEvalSysPlatform(t *testing.T) {
	cases := []struct {
		src  string
		env  Env
		want bool
	}{
		{`sys_platform == "linux"`, Env{SysPlatform: "linux"}, true},
		{`sys_platform == "linux"`, Env{SysPlatform: "win32"}, false},
		{`sys_platform != "win32"`, Env{SysPlatform: "linux"}, true},
		{`sys_platform != "win32"`, Env{SysPlatform: "win32"}, false},
	}
	for _, tc := range cases {
		e, err := Parse(tc.src)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.src, err)
		}
		if got := e.Eval(tc.env); got != tc.want {
			t.Errorf("eval %q on %+v = %v, want %v", tc.src, tc.env, got, tc.want)
		}
	}
}

func TestEvalPythonVersion(t *testing.T) {
	env := Env{PythonVersion: "3.14", PythonFullVersion: "3.14.0"}
	cases := []struct {
		src  string
		want bool
	}{
		{`python_version == "3.14"`, true},
		{`python_version >= "3.10"`, true},
		{`python_version < "3.10"`, false},
		{`python_version != "3.14"`, false},
		{`python_full_version >= "3.13.5"`, true},
	}
	for _, tc := range cases {
		e, err := Parse(tc.src)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.src, err)
		}
		if got := e.Eval(env); got != tc.want {
			t.Errorf("eval %q = %v, want %v", tc.src, got, tc.want)
		}
	}
}

func TestEvalAndOr(t *testing.T) {
	env := Env{SysPlatform: "linux", PythonVersion: "3.14"}
	cases := []struct {
		src  string
		want bool
	}{
		{`sys_platform == "linux" and python_version >= "3.10"`, true},
		{`sys_platform == "linux" and python_version < "3.10"`, false},
		{`sys_platform == "win32" or python_version >= "3.10"`, true},
		{`sys_platform == "win32" or python_version < "3.10"`, false},
	}
	for _, tc := range cases {
		e, err := Parse(tc.src)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.src, err)
		}
		if got := e.Eval(env); got != tc.want {
			t.Errorf("eval %q = %v, want %v", tc.src, got, tc.want)
		}
	}
}

func TestEvalNot(t *testing.T) {
	env := Env{SysPlatform: "linux"}
	e, err := Parse(`not sys_platform == "win32"`)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Eval(env) {
		t.Error("not eval false")
	}
	e2, err := Parse(`not (sys_platform == "linux")`)
	if err != nil {
		t.Fatal(err)
	}
	if e2.Eval(env) {
		t.Error("not(true) should be false")
	}
}

func TestEvalParens(t *testing.T) {
	env := Env{SysPlatform: "linux", PythonVersion: "3.14"}
	e, err := Parse(`(sys_platform == "linux" or sys_platform == "darwin") and python_version >= "3.10"`)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Eval(env) {
		t.Error("parens eval false")
	}
}

func TestEvalIn(t *testing.T) {
	env := Env{SysPlatform: "linux"}
	e, err := Parse(`sys_platform in "linux darwin"`)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Eval(env) {
		t.Error("in eval false")
	}
}

func TestEvalNotIn(t *testing.T) {
	env := Env{SysPlatform: "linux"}
	e, err := Parse(`sys_platform not in "win32 cygwin"`)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Eval(env) {
		t.Error("not in eval false")
	}
}

func TestEvalUnknownVariableEmpty(t *testing.T) {
	e, err := Parse(`unknown_var == ""`)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Eval(Env{}) {
		t.Error("unknown var should equal empty string")
	}
}

func TestEvalExtraEmpty(t *testing.T) {
	e, err := Parse(`extra == "dev"`)
	if err != nil {
		t.Fatal(err)
	}
	if e.Eval(Env{}) {
		t.Error("extra should be empty by default")
	}
}

func TestParseError(t *testing.T) {
	cases := []string{
		`sys_platform == `,
		`sys_platform == "linux" and`,
		`(sys_platform == "linux"`,
		`sys_platform "linux"`,
	}
	for _, src := range cases {
		_, err := Parse(src)
		if err == nil {
			t.Errorf("expected parse error for %q", src)
			continue
		}
		if !strings.Contains(err.Error(), "marker") {
			t.Errorf("expected marker prefix in error: %v", err)
		}
	}
}

func TestDefaultEnvShape(t *testing.T) {
	env := DefaultEnv()
	if env.PythonVersion == "" || env.SysPlatform == "" || env.PlatformMachine == "" {
		t.Errorf("DefaultEnv missing fields: %+v", env)
	}
}

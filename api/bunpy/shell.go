package bunpy

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"
)

// BuildShell returns the bunpy.shell built-in function.
func BuildShell(_ *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "shell",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("bunpy.shell() requires a command argument")
			}
			cmdStr, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("bunpy.shell(): command must be a str")
			}

			cwd, capture := "", true
			var envMap map[string]string
			if kwargs != nil {
				if v, ok2 := kwargs.GetStr("cwd"); ok2 {
					if s, ok3 := v.(*goipyObject.Str); ok3 {
						cwd = s.V
					}
				}
				if v, ok2 := kwargs.GetStr("capture"); ok2 {
					if b, ok3 := v.(*goipyObject.Bool); ok3 {
						capture = b.V
					}
				}
				if v, ok2 := kwargs.GetStr("env"); ok2 {
					envMap = dictToStringMap(v)
				}
			}

			c := shellCommand(cmdStr.V)
			if cwd != "" {
				c.Dir = cwd
			}
			if envMap != nil {
				c.Env = buildEnv(envMap)
			}

			var outBuf, errBuf bytes.Buffer
			if capture {
				c.Stdout = &outBuf
				c.Stderr = &errBuf
			}

			code := 0
			if err := c.Run(); err != nil {
				if ex, ok2 := err.(*exec.ExitError); ok2 {
					code = ex.ExitCode()
				}
			}

			return makeShellResult(outBuf.String(), errBuf.String(), code), nil
		},
	}
}

// BuildSpawn returns the bunpy.spawn built-in function.
func BuildSpawn(_ *goipyVM.Interp) *goipyObject.BuiltinFunc {
	return &goipyObject.BuiltinFunc{
		Name: "spawn",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("bunpy.spawn() requires an argv argument")
			}
			argv, err := toStringList(args[0])
			if err != nil {
				return nil, fmt.Errorf("bunpy.spawn(): %w", err)
			}
			if len(argv) == 0 {
				return nil, fmt.Errorf("bunpy.spawn(): argv must not be empty")
			}

			cwd, stdinStr := "", ""
			capture := true
			if kwargs != nil {
				if v, ok2 := kwargs.GetStr("cwd"); ok2 {
					if s, ok3 := v.(*goipyObject.Str); ok3 {
						cwd = s.V
					}
				}
				if v, ok2 := kwargs.GetStr("stdin"); ok2 {
					if s, ok3 := v.(*goipyObject.Str); ok3 {
						stdinStr = s.V
					}
				}
				if v, ok2 := kwargs.GetStr("capture"); ok2 {
					if b, ok3 := v.(*goipyObject.Bool); ok3 {
						capture = b.V
					}
				}
			}

			c := exec.Command(argv[0], argv[1:]...)
			if cwd != "" {
				c.Dir = cwd
			}

			var outBuf, errBuf bytes.Buffer
			if capture {
				c.Stdout = &outBuf
				c.Stderr = &errBuf
			}
			if stdinStr != "" {
				c.Stdin = strings.NewReader(stdinStr)
			}

			if err2 := c.Start(); err2 != nil {
				return nil, fmt.Errorf("bunpy.spawn(): %w", err2)
			}

			var mu sync.Mutex
			code := 0
			done := make(chan struct{})
			go func() {
				if err3 := c.Wait(); err3 != nil {
					if ex, ok2 := err3.(*exec.ExitError); ok2 {
						mu.Lock()
						code = ex.ExitCode()
						mu.Unlock()
					}
				}
				close(done)
			}()

			return makeProcInstance(c, &outBuf, &errBuf, done, &mu, &code), nil
		},
	}
}

// BuildDollar returns the bunpy.dollar built-in function.
func BuildDollar(i *goipyVM.Interp) *goipyObject.BuiltinFunc {
	shellFn := BuildShell(i)
	return &goipyObject.BuiltinFunc{
		Name: "dollar",
		Call: func(_ any, args []goipyObject.Object, kwargs *goipyObject.Dict) (goipyObject.Object, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("bunpy.dollar() requires a template argument")
			}
			tmpl, ok := args[0].(*goipyObject.Str)
			if !ok {
				return nil, fmt.Errorf("bunpy.dollar(): template must be a str")
			}

			cmd := tmpl.V
			if kwargs != nil {
				keys, vals := kwargs.Items()
				for idx, k := range keys {
					ks, ok2 := k.(*goipyObject.Str)
					if !ok2 {
						continue
					}
					var valStr string
					switch v := vals[idx].(type) {
					case *goipyObject.Str:
						valStr = v.V
					default:
						valStr = fmt.Sprintf("%v", vals[idx])
					}
					cmd = strings.ReplaceAll(cmd, "{"+ks.V+"}", shellQuote(valStr))
				}
			}

			return shellFn.Call(nil, []goipyObject.Object{&goipyObject.Str{V: cmd}}, nil)
		},
	}
}

func shellCommand(cmd string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd.exe", "/c", cmd)
	}
	return exec.Command("/bin/sh", "-c", cmd)
}

func makeShellResult(stdout, stderr string, exitcode int) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "ShellResult", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}
	inst.Dict.SetStr("stdout", &goipyObject.Str{V: stdout})
	inst.Dict.SetStr("stderr", &goipyObject.Str{V: stderr})
	inst.Dict.SetStr("exitcode", goipyObject.NewInt(int64(exitcode)))
	return inst
}

func makeProcInstance(c *exec.Cmd, outBuf, errBuf *bytes.Buffer, done chan struct{}, mu *sync.Mutex, code *int) *goipyObject.Instance {
	cls := &goipyObject.Class{Name: "Proc", Dict: goipyObject.NewDict()}
	inst := &goipyObject.Instance{Class: cls, Dict: goipyObject.NewDict()}

	inst.Dict.SetStr("pid", goipyObject.NewInt(int64(c.Process.Pid)))

	inst.Dict.SetStr("wait", &goipyObject.BuiltinFunc{
		Name: "wait",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			<-done
			mu.Lock()
			exitcode := *code
			mu.Unlock()
			inst.Dict.SetStr("exitcode", goipyObject.NewInt(int64(exitcode)))
			inst.Dict.SetStr("stdout", &goipyObject.Str{V: outBuf.String()})
			inst.Dict.SetStr("stderr", &goipyObject.Str{V: errBuf.String()})
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("kill", &goipyObject.BuiltinFunc{
		Name: "kill",
		Call: func(_ any, _ []goipyObject.Object, _ *goipyObject.Dict) (goipyObject.Object, error) {
			if c.Process != nil {
				c.Process.Kill()
			}
			return goipyObject.None, nil
		},
	})
	inst.Dict.SetStr("exitcode", goipyObject.NewInt(0))
	inst.Dict.SetStr("stdout", &goipyObject.Str{V: ""})
	inst.Dict.SetStr("stderr", &goipyObject.Str{V: ""})
	return inst
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func buildEnv(m map[string]string) []string {
	// Start with current environment and override.
	env := os.Environ()
	overrides := make(map[string]string, len(m))
	for k, v := range m {
		overrides[k] = v
	}
	result := make([]string, 0, len(env)+len(overrides))
	for _, e := range env {
		idx := strings.IndexByte(e, '=')
		if idx < 0 {
			result = append(result, e)
			continue
		}
		k := e[:idx]
		if _, found := overrides[k]; found {
			delete(overrides, k)
			result = append(result, k+"="+m[k])
		} else {
			result = append(result, e)
		}
	}
	for k, v := range overrides {
		result = append(result, k+"="+v)
	}
	return result
}

func dictToStringMap(v goipyObject.Object) map[string]string {
	d, ok := v.(*goipyObject.Dict)
	if !ok {
		return nil
	}
	keys, vals := d.Items()
	m := make(map[string]string, len(keys))
	for i, k := range keys {
		ks, ok2 := k.(*goipyObject.Str)
		vs, ok3 := vals[i].(*goipyObject.Str)
		if ok2 && ok3 {
			m[ks.V] = vs.V
		}
	}
	return m
}

func toStringList(obj goipyObject.Object) ([]string, error) {
	lst, ok := obj.(*goipyObject.List)
	if !ok {
		return nil, fmt.Errorf("expected a list")
	}
	result := make([]string, len(lst.V))
	for i, item := range lst.V {
		s, ok2 := item.(*goipyObject.Str)
		if !ok2 {
			return nil, fmt.Errorf("argv items must be str")
		}
		result[i] = s.V
	}
	return result, nil
}

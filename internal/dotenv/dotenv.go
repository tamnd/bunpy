// Package dotenv parses .env files and loads them into the process environment.
package dotenv

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// Parse reads key=value pairs from r.
// Lines starting with # and blank lines are ignored.
// Quoted values have their quotes stripped.
// The "export " prefix is stripped.
func Parse(r io.Reader) (map[string]string, error) {
	env := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = stripQuotes(v)
		env[k] = v
	}
	return env, sc.Err()
}

// Load parses path and calls os.Setenv for each key.
func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	env, err := Parse(f)
	if err != nil {
		return err
	}
	for k, v := range env {
		os.Setenv(k, v)
	}
	return nil
}

// LoadFiles loads multiple .env files in order; later files override earlier ones.
func LoadFiles(paths []string) error {
	for _, p := range paths {
		if err := Load(p); err != nil {
			return err
		}
	}
	return nil
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

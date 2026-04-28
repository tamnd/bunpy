package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupWhyProject seeds a project with two pinned wheels (one
// transitive). The wheels are written into a per-test cache so the
// Requires-Dist scan in `bunpy why` runs against real bytes.
func setupWhyProject(t *testing.T) (proj, cacheDir string) {
	t.Helper()
	proj = t.TempDir()
	cacheDir = t.TempDir()

	mustWrite(t, filepath.Join(proj, "pyproject.toml"), []byte(`[project]
name = "demo"
version = "0.1.0"
dependencies = [
    "alpha>=1.0",
]

[project.optional-dependencies]
tracing = ["beta>=1.0"]

[tool.bunpy]
peer-dependencies = []
`))

	mustWrite(t, filepath.Join(proj, "uv.lock"), []byte(`version = 1
requires-python = ">=3.12"

[[package]]
name = "alpha"
version = "1.0.0"
source = { registry = "https://pypi.org/simple" }

[[package.wheels]]
url = "https://files.example.com/alpha-1.0.0-py3-none-any.whl"
hash = "sha256:aaaa"
size = 0

[[package]]
name = "beta"
version = "1.0.0"
source = { registry = "https://pypi.org/simple" }
groups = ["optional:tracing"]

[[package.wheels]]
url = "https://files.example.com/beta-1.0.0-py3-none-any.whl"
hash = "sha256:bbbb"
size = 0

[[package]]
name = "gamma"
version = "2.2.3"
source = { registry = "https://pypi.org/simple" }

[[package.wheels]]
url = "https://files.example.com/gamma-2.2.3-py3-none-any.whl"
hash = "sha256:cccc"
size = 0
`))

	writeFakeWheel(t, cacheDir, "alpha", "1.0.0", "Requires-Dist: gamma>=2.0\n")
	writeFakeWheel(t, cacheDir, "beta", "1.0.0", "Requires-Dist: gamma>=2.1\n")
	writeFakeWheel(t, cacheDir, "gamma", "2.2.3", "")
	return proj, cacheDir
}

// writeFakeWheel synthesises a tiny zip with just the dist-info
// METADATA so wheel.LoadMetadata + ParseMetadata can read it.
func writeFakeWheel(t *testing.T, cacheDir, name, version, requiresDist string) {
	t.Helper()
	filename := fmt.Sprintf("%s-%s-py3-none-any.whl", name, version)
	dst := filepath.Join(cacheDir, "wheels", name, filename)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("Metadata-Version: 2.1\nName: %s\nVersion: %s\n%s", name, version, requiresDist)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(fmt.Sprintf("%s-%s.dist-info/METADATA", name, version))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, dst, buf.Bytes())
}

func TestWhyTreeShape(t *testing.T) {
	proj, cacheDir := setupWhyProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"why", "gamma", "--cache-dir", cacheDir}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("why gamma: code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "gamma 2.2.3") {
		t.Errorf("missing leaf header: %q", out)
	}
	if !strings.Contains(out, "via alpha 1.0.0") {
		t.Errorf("missing alpha edge: %q", out)
	}
	if !strings.Contains(out, "via beta 1.0.0") {
		t.Errorf("missing beta edge: %q", out)
	}
	if !strings.Contains(out, "project requirement (main)") {
		t.Errorf("missing main lane: %q", out)
	}
	if !strings.Contains(out, "project requirement (optional:tracing)") {
		t.Errorf("missing tracing lane: %q", out)
	}
}

func TestWhyTopOnly(t *testing.T) {
	proj, cacheDir := setupWhyProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"why", "gamma", "--top", "--cache-dir", cacheDir}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("why --top: code=%d err=%v", code, err)
	}
	got := strings.Fields(stdout.String())
	want := []string{"alpha", "beta"}
	if len(got) != len(want) {
		t.Fatalf("--top: got %v want %v", got, want)
	}
	for i, n := range want {
		if got[i] != n {
			t.Errorf("--top[%d]: %q want %q", i, got[i], n)
		}
	}
}

func TestWhyJSON(t *testing.T) {
	proj, cacheDir := setupWhyProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"why", "gamma", "--json", "--cache-dir", cacheDir}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("why --json: code=%d err=%v", code, err)
	}
	var parsed struct {
		Package string `json:"package"`
		Version string `json:"version"`
		Chains  []struct {
			Lane  string `json:"lane"`
			Edges []struct {
				Name string `json:"name"`
			} `json:"edges"`
		} `json:"chains"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("json: %v\nbody=%s", err, stdout.String())
	}
	if parsed.Package != "gamma" || parsed.Version != "2.2.3" {
		t.Errorf("header: %+v", parsed)
	}
	if len(parsed.Chains) != 2 {
		t.Fatalf("chains: got %d, want 2", len(parsed.Chains))
	}
}

func TestWhyDepthCap(t *testing.T) {
	proj, cacheDir := setupWhyProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"why", "gamma", "--depth", "1", "--cache-dir", cacheDir}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("why --depth 1: code=%d err=%v", code, err)
	}
	out := stdout.String()
	// depth=1 means each chain has at most 1 edge after the leaf,
	// i.e. either an immediate parent or the @project edge.
	if strings.Contains(out, "    via") {
		t.Errorf("depth cap not honoured: %q", out)
	}
}

func TestWhyMissingPin(t *testing.T) {
	proj, cacheDir := setupWhyProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"why", "ghost", "--cache-dir", cacheDir}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing pin")
	}
	if code == 0 {
		t.Error("expected non-zero exit")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error %q does not name the missing pin", err)
	}
}

func TestWhyLaneFilter(t *testing.T) {
	proj, cacheDir := setupWhyProject(t)
	chdirTo(t, proj)

	var stdout, stderr bytes.Buffer
	code, err := run([]string{"why", "gamma", "--lane", "main", "--cache-dir", cacheDir}, &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("why --lane main: code=%d err=%v", code, err)
	}
	out := stdout.String()
	if !strings.Contains(out, "via alpha 1.0.0") {
		t.Errorf("expected alpha chain: %q", out)
	}
	if strings.Contains(out, "via beta 1.0.0") {
		t.Errorf("did not expect beta chain in main lane: %q", out)
	}
}

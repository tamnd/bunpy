package editable

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) (source, target string) {
	t.Helper()
	source = t.TempDir()
	target = t.TempDir()
	pkg := filepath.Join(source, "widget")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "__init__.py"), []byte("VERSION = '1.0.0'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return source, target
}

func TestInstallLaysOutEditableProxy(t *testing.T) {
	source, target := setup(t)
	written, err := Install(Spec{Name: "widget", Version: "1.0.0", Source: source, Target: target})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(written) == 0 {
		t.Fatal("Install returned no paths")
	}

	pth := filepath.Join(target, "widget.pth")
	body, err := os.ReadFile(pth)
	if err != nil {
		t.Fatalf("read .pth: %v", err)
	}
	if strings.TrimSpace(string(body)) != source {
		t.Errorf(".pth = %q, want %q", string(body), source)
	}

	distInfo := filepath.Join(target, "widget-1.0.0.dist-info")
	if installer, err := os.ReadFile(filepath.Join(distInfo, "INSTALLER")); err != nil {
		t.Fatalf("read INSTALLER: %v", err)
	} else if strings.TrimSpace(string(installer)) != InstallerTag {
		t.Errorf("INSTALLER = %q, want %q", string(installer), InstallerTag)
	}
	if _, err := os.Stat(filepath.Join(distInfo, "METADATA")); err != nil {
		t.Errorf("METADATA missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(distInfo, "direct_url.json")); err != nil {
		t.Errorf("direct_url.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(distInfo, "RECORD")); err != nil {
		t.Errorf("RECORD missing: %v", err)
	}
}

func TestInstallRecordCoversEveryFile(t *testing.T) {
	source, target := setup(t)
	written, err := Install(Spec{Name: "widget", Version: "1.0.0", Source: source, Target: target})
	if err != nil {
		t.Fatal(err)
	}
	record, err := os.ReadFile(filepath.Join(target, "widget-1.0.0.dist-info", "RECORD"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(record)
	for _, rel := range written {
		if !strings.Contains(body, rel) {
			t.Errorf("RECORD missing %q:\n%s", rel, body)
		}
	}
}

func TestUninstallRemovesEverything(t *testing.T) {
	source, target := setup(t)
	if _, err := Install(Spec{Name: "widget", Version: "1.0.0", Source: source, Target: target}); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(target, "widget", "1.0.0"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "widget-1.0.0.dist-info")); !os.IsNotExist(err) {
		t.Errorf("dist-info still present: err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "widget.pth")); !os.IsNotExist(err) {
		t.Errorf(".pth still present: err=%v", err)
	}
}

func TestUninstallMissingIsNoop(t *testing.T) {
	target := t.TempDir()
	if err := Uninstall(target, "widget", "1.0.0"); err != nil {
		t.Errorf("Uninstall on missing: %v", err)
	}
}

func TestInstallRequiresAbsolutePaths(t *testing.T) {
	if _, err := Install(Spec{Name: "widget", Source: "rel", Target: "/abs"}); err == nil {
		t.Error("expected error: relative Source")
	}
	if _, err := Install(Spec{Name: "widget", Source: "/abs", Target: "rel"}); err == nil {
		t.Error("expected error: relative Target")
	}
}

func TestInstallRequiresExistingSource(t *testing.T) {
	target := t.TempDir()
	if _, err := Install(Spec{Name: "widget", Source: filepath.Join(target, "nope"), Target: target}); err == nil {
		t.Error("expected error: missing source")
	}
}

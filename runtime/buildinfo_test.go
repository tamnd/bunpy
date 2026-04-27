package runtime

import (
	"encoding/json"
	"testing"
)

func TestBuildInfoDevDefaults(t *testing.T) {
	if Version != "dev" {
		t.Errorf("default Version = %q, want %q", Version, "dev")
	}
	if Commit != "" {
		t.Errorf("default Commit = %q, want empty", Commit)
	}
	if BuildDate != "" {
		t.Errorf("default BuildDate = %q, want empty", BuildDate)
	}
}

func TestBuildPopulatesGoOSArch(t *testing.T) {
	b := Build()
	if b.Go == "" {
		t.Error("Build().Go is empty")
	}
	if b.OS == "" {
		t.Error("Build().OS is empty")
	}
	if b.Arch == "" {
		t.Error("Build().Arch is empty")
	}
}

func TestBuildInfoJSONShape(t *testing.T) {
	data, err := json.Marshal(Build())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"version", "go", "os", "arch"} {
		if _, ok := got[k]; !ok {
			t.Errorf("BuildInfo JSON missing key %q: %s", k, data)
		}
	}
	// On a dev build the omitempty-tagged fields should not appear.
	for _, k := range []string{"commit", "build_date", "goipy", "gocopy", "gopapy"} {
		if _, ok := got[k]; ok {
			t.Errorf("BuildInfo JSON has %q on dev build (defaults are empty): %s", k, data)
		}
	}
}

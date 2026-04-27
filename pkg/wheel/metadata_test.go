package wheel

import "testing"

func TestParseMetadataBasic(t *testing.T) {
	body := []byte("Metadata-Version: 2.1\n" +
		"Name: widget\n" +
		"Version: 1.0.0\n" +
		"Requires-Dist: gizmo>=2.0\n" +
		"Requires-Dist: lefty (>=1.0,<2)\n" +
		"Requires-Dist: dev-only; extra == \"dev\"\n" +
		"Requires-Dist: winonly; sys_platform == \"win32\"\n" +
		"\n" +
		"long description here\n")
	m, err := ParseMetadata(body)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "widget" || m.Version != "1.0.0" {
		t.Errorf("name/version: %q %q", m.Name, m.Version)
	}
	if len(m.RequiresDist) != 4 {
		t.Fatalf("requires-dist count: %d", len(m.RequiresDist))
	}
	if got := m.RequiresDist[0]; got.Name != "gizmo" || got.Spec != ">=2.0" || got.Marker != "" {
		t.Errorf("rd0 = %+v", got)
	}
	if got := m.RequiresDist[1]; got.Name != "lefty" || got.Spec != ">=1.0,<2" {
		t.Errorf("rd1 = %+v", got)
	}
	if got := m.RequiresDist[2]; got.Marker != `extra == "dev"` {
		t.Errorf("rd2 marker = %q", got.Marker)
	}
	if got := m.RequiresDist[3]; got.Marker != `sys_platform == "win32"` {
		t.Errorf("rd3 marker = %q", got.Marker)
	}
}

func TestParseRequiresDistExtras(t *testing.T) {
	rd, err := ParseRequiresDist("requests[security,socks] (>=2.0); python_version >= \"3.7\"")
	if err != nil {
		t.Fatal(err)
	}
	if rd.Name != "requests" {
		t.Errorf("name = %q", rd.Name)
	}
	if len(rd.Extras) != 2 || rd.Extras[0] != "security" || rd.Extras[1] != "socks" {
		t.Errorf("extras = %v", rd.Extras)
	}
	if rd.Spec != ">=2.0" {
		t.Errorf("spec = %q", rd.Spec)
	}
	if rd.Marker != `python_version >= "3.7"` {
		t.Errorf("marker = %q", rd.Marker)
	}
}

func TestParseRequiresDistBareName(t *testing.T) {
	rd, err := ParseRequiresDist("typing-extensions")
	if err != nil {
		t.Fatal(err)
	}
	if rd.Name != "typing-extensions" || rd.Spec != "" || rd.Marker != "" {
		t.Errorf("rd = %+v", rd)
	}
}

package resolver

import (
	"errors"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/version"
)

// fakeRegistry is an in-memory Registry. Versions are listed
// oldest-first; Highest reorders.
type fakeRegistry struct {
	pkgs map[string][]string
	deps map[string]map[string][]Requirement
}

func newFake() *fakeRegistry {
	return &fakeRegistry{
		pkgs: map[string][]string{},
		deps: map[string]map[string][]Requirement{},
	}
}

func (r *fakeRegistry) add(pkg string, vers []string, deps map[string][]Requirement) {
	r.pkgs[pkg] = vers
	r.deps[pkg] = deps
}

func (r *fakeRegistry) Versions(pkg string) ([]string, error) {
	vs, ok := r.pkgs[pkg]
	if !ok {
		return nil, errors.New("unknown package: " + pkg)
	}
	return vs, nil
}

func (r *fakeRegistry) Dependencies(pkg, ver string) ([]Requirement, error) {
	if m, ok := r.deps[pkg]; ok {
		return m[ver], nil
	}
	return nil, nil
}

func mustReq(t *testing.T, name, spec string) Requirement {
	t.Helper()
	s, err := version.ParseSpec(spec)
	if err != nil {
		t.Fatalf("ParseSpec(%q): %v", spec, err)
	}
	return Requirement{Name: name, Spec: s}
}

func TestSolveSinglePackage(t *testing.T) {
	reg := newFake()
	reg.add("widget", []string{"1.0.0", "1.1.0"}, nil)
	res, err := New(reg).Solve([]Requirement{mustReq(t, "widget", ">=1.0")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pins) != 1 || res.Pins[0].Name != "widget" || res.Pins[0].Version != "1.1.0" {
		t.Errorf("pins = %+v", res.Pins)
	}
}

func TestSolveTransitive(t *testing.T) {
	reg := newFake()
	reg.add("widget", []string{"1.0.0"}, map[string][]Requirement{
		"1.0.0": {mustReq(t, "gizmo", ">=2.0")},
	})
	reg.add("gizmo", []string{"2.0.0", "2.1.0"}, nil)
	res, err := New(reg).Solve([]Requirement{mustReq(t, "widget", "")})
	if err != nil {
		t.Fatal(err)
	}
	pins := indexPins(res.Pins)
	if pins["widget"] != "1.0.0" || pins["gizmo"] != "2.1.0" {
		t.Errorf("pins = %+v", pins)
	}
}

func TestSolveSharedConstraint(t *testing.T) {
	reg := newFake()
	reg.add("widget", []string{"1.0.0"}, map[string][]Requirement{
		"1.0.0": {mustReq(t, "gizmo", ">=2.0,<2.1")},
	})
	reg.add("gizmo", []string{"2.0.0", "2.0.5", "2.1.0"}, nil)
	res, err := New(reg).Solve([]Requirement{
		mustReq(t, "widget", ""),
		mustReq(t, "gizmo", ">=2.0"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if v := indexPins(res.Pins)["gizmo"]; v != "2.0.5" {
		t.Errorf("gizmo pin = %q, want 2.0.5", v)
	}
}

func TestSolveNoVersionConflict(t *testing.T) {
	reg := newFake()
	reg.add("widget", []string{"1.0.0"}, nil)
	_, err := New(reg).Solve([]Requirement{mustReq(t, "widget", ">=2.0")})
	if err == nil {
		t.Fatal("expected conflict")
	}
	var c *Conflict
	if !errors.As(err, &c) {
		t.Fatalf("expected *Conflict, got %T", err)
	}
	if !c.NoVersion || c.Package != "widget" {
		t.Errorf("conflict = %+v", c)
	}
	if !strings.Contains(c.Error(), "no candidate") {
		t.Errorf("error text: %s", c.Error())
	}
}

func TestSolveDirectVersusTransitiveConflict(t *testing.T) {
	reg := newFake()
	reg.add("widget", []string{"1.0.0"}, map[string][]Requirement{
		"1.0.0": {mustReq(t, "gizmo", "==2.0.0")},
	})
	reg.add("gizmo", []string{"2.0.0", "3.0.0"}, nil)
	_, err := New(reg).Solve([]Requirement{
		mustReq(t, "widget", ""),
		mustReq(t, "gizmo", ">=3.0"),
	})
	if err == nil {
		t.Fatal("expected conflict")
	}
	var c *Conflict
	if !errors.As(err, &c) {
		t.Fatalf("expected *Conflict, got %T", err)
	}
	if c.Package != "gizmo" {
		t.Errorf("conflict package = %q", c.Package)
	}
}

func TestSolveCycleSafe(t *testing.T) {
	reg := newFake()
	reg.add("a", []string{"1.0.0"}, map[string][]Requirement{
		"1.0.0": {mustReq(t, "b", "")},
	})
	reg.add("b", []string{"1.0.0"}, map[string][]Requirement{
		"1.0.0": {mustReq(t, "a", "")},
	})
	res, err := New(reg).Solve([]Requirement{mustReq(t, "a", "")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pins) != 2 {
		t.Errorf("pins = %+v", res.Pins)
	}
}

func TestSolveRespectsLocked(t *testing.T) {
	reg := newFake()
	reg.add("widget", []string{"1.0.0", "1.1.0"}, nil)
	s := New(reg)
	s.Locked = map[string]string{"widget": "1.0.0"}
	res, err := s.Solve([]Requirement{mustReq(t, "widget", ">=1.0")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pins) != 1 || res.Pins[0].Version != "1.0.0" {
		t.Errorf("locked 1.0.0 should pin: %+v", res.Pins)
	}
}

func TestSolveOverridesLockedWhenSpecForbids(t *testing.T) {
	reg := newFake()
	reg.add("widget", []string{"1.0.0", "1.1.0", "2.0.0"}, nil)
	s := New(reg)
	s.Locked = map[string]string{"widget": "1.0.0"}
	res, err := s.Solve([]Requirement{mustReq(t, "widget", ">=1.5")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Pins) != 1 || res.Pins[0].Version != "2.0.0" {
		t.Errorf("locked 1.0.0 forbidden by >=1.5; want 2.0.0: %+v", res.Pins)
	}
}

func TestTermStringPositive(t *testing.T) {
	tt, err := NewPositive("widget", ">=1.0")
	if err != nil {
		t.Fatal(err)
	}
	got := tt.String()
	if !strings.Contains(got, "widget") || !strings.Contains(got, ">=1.0") {
		t.Errorf("string = %q", got)
	}
}

func indexPins(pins []Pin) map[string]string {
	out := map[string]string{}
	for _, p := range pins {
		out[p.Name] = p.Version
	}
	return out
}

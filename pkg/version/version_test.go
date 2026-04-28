package version

import "testing"

func TestParseSpec(t *testing.T) {
	cases := []struct {
		in    string
		ops   []Op
		first string
	}{
		{"1.2.3", []Op{OpEQ}, "1.2.3"},
		{"==1.0", []Op{OpEQ}, "1.0"},
		{">=1.0", []Op{OpGE}, "1.0"},
		{">=1.0,<2.0", []Op{OpGE, OpLT}, "1.0"},
		{"~=1.4", []Op{OpCA}, "1.4"},
	}
	for _, tc := range cases {
		s, err := ParseSpec(tc.in)
		if err != nil {
			t.Errorf("ParseSpec(%q): %v", tc.in, err)
			continue
		}
		if len(s.Clauses) != len(tc.ops) {
			t.Errorf("ParseSpec(%q): got %d clauses, want %d", tc.in, len(s.Clauses), len(tc.ops))
			continue
		}
		for i, op := range tc.ops {
			if s.Clauses[i].Op != op {
				t.Errorf("ParseSpec(%q) clause %d op = %q, want %q", tc.in, i, s.Clauses[i].Op, op)
			}
		}
		if s.Clauses[0].Version != tc.first {
			t.Errorf("ParseSpec(%q) clause 0 version = %q, want %q", tc.in, s.Clauses[0].Version, tc.first)
		}
	}
}

func TestParseSpecErrors(t *testing.T) {
	for _, in := range []string{"==", ">=", "===1.0", "frob"} {
		if _, err := ParseSpec(in); err == nil {
			t.Errorf("ParseSpec(%q): expected error", in)
		}
	}
}

func TestCompareNumeric(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0", 0},
		{"1.0", "1.1", -1},
		{"1.0.1", "1.0", 1},
		{"2.0", "1.99", 1},
		{"1.0.0", "1.0", 0},
	}
	for _, tc := range cases {
		if got := Compare(tc.a, tc.b); got != tc.want {
			t.Errorf("Compare(%q,%q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestComparePrePostDev(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0a1", "1.0", -1},
		{"1.0a1", "1.0b1", -1},
		{"1.0b1", "1.0rc1", -1},
		{"1.0rc1", "1.0", -1},
		{"1.0", "1.0.post1", -1},
		{"1.0.dev1", "1.0a1", -1},
		{"1.0.dev1", "1.0", -1},
		{"1.0a1.dev1", "1.0a1", -1},
	}
	for _, tc := range cases {
		if got := Compare(tc.a, tc.b); got != tc.want {
			t.Errorf("Compare(%q,%q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestMatch(t *testing.T) {
	cases := []struct {
		spec, v string
		want    bool
	}{
		{"==1.0", "1.0", true},
		{"==1.0", "1.0.0", true},
		{"==1.0", "1.1", false},
		{">=1.0", "1.0", true},
		{">=1.0", "0.9", false},
		{"<2.0", "1.99", true},
		{"<2.0", "2.0", false},
		{">=1.0,<2.0", "1.5", true},
		{">=1.0,<2.0", "2.0", false},
		{"!=1.0", "1.1", true},
		{"!=1.0", "1.0", false},
		{"~=1.4", "1.4.5", true},
		{"~=1.4", "1.5", true},
		{"~=1.4", "2.0", false},
		{"~=1.4.5", "1.4.6", true},
		{"~=1.4.5", "1.5", false},
	}
	for _, tc := range cases {
		s, err := ParseSpec(tc.spec)
		if err != nil {
			t.Fatalf("ParseSpec(%q): %v", tc.spec, err)
		}
		if got := s.Match(tc.v); got != tc.want {
			t.Errorf("Spec(%q).Match(%q) = %v, want %v", tc.spec, tc.v, got, tc.want)
		}
	}
}

func TestHighest(t *testing.T) {
	candidates := []string{"1.0", "1.0.1", "2.0a1", "2.0", "1.5"}
	cases := []struct {
		spec string
		want string
	}{
		{"", "2.0"},
		{">=1.0", "2.0"},
		{">=1.0,<2.0", "1.5"},
		{"==1.0", "1.0"},
		{">=3.0", ""},
	}
	for _, tc := range cases {
		s, err := ParseSpec(tc.spec)
		if err != nil {
			t.Fatalf("ParseSpec(%q): %v", tc.spec, err)
		}
		if got := Highest(s, candidates); got != tc.want {
			t.Errorf("Highest(%q) = %q, want %q", tc.spec, got, tc.want)
		}
	}
}

func TestWildcardSpec(t *testing.T) {
	cases := []struct {
		spec string
		v    string
		want bool
	}{
		{"==1.*", "1.0", true},
		{"==1.*", "1.99", true},
		{"==1.*", "1.0.0", true},
		{"==1.*", "2.0", false},
		{"==1.*", "0.9", false},
		{"==1.2.*", "1.2.0", true},
		{"==1.2.*", "1.2.99", true},
		{"==1.2.*", "1.3", false},
		{"==1.2.*", "1.1.9", false},
		{"!=1.*", "2.0", true},
		{"!=1.*", "1.0", false},
		{"!=1.*", "1.99", false},
		{"==1.*,!=1.2.*", "1.3", true},
		{"==1.*,!=1.2.*", "1.2.0", false},
		{"==1.*,!=1.2.*", "2.0", false},
	}
	for _, tc := range cases {
		s, err := ParseSpec(tc.spec)
		if err != nil {
			t.Fatalf("ParseSpec(%q): %v", tc.spec, err)
		}
		if got := s.Match(tc.v); got != tc.want {
			t.Errorf("Spec(%q).Match(%q) = %v, want %v", tc.spec, tc.v, got, tc.want)
		}
	}
}

func TestEmptySpecMatchesAll(t *testing.T) {
	s := Spec{}
	if !s.Match("0.0.1") || !s.Match("99.99.99") {
		t.Error("empty Spec must match every version")
	}
}

package runtime

import (
	"sort"
	"testing"
)

func TestStdlibModulesNonEmpty(t *testing.T) {
	if StdlibCount() == 0 {
		t.Fatal("StdlibCount() == 0; expected the goipy stdlib index to be populated")
	}
}

func TestStdlibModulesSorted(t *testing.T) {
	mods := StdlibModules()
	if !sort.StringsAreSorted(mods) {
		t.Errorf("StdlibModules() is not sorted; first few: %v", head(mods, 10))
	}
}

func TestStdlibModulesDeduped(t *testing.T) {
	mods := StdlibModules()
	seen := make(map[string]struct{}, len(mods))
	for _, m := range mods {
		if _, ok := seen[m]; ok {
			t.Errorf("duplicate module %q in StdlibModules()", m)
		}
		seen[m] = struct{}{}
	}
}

func TestStdlibIncludesCriticalSet(t *testing.T) {
	want := []string{"math", "json", "re", "os", "sys", "io", "collections", "functools", "itertools"}
	have := make(map[string]struct{}, StdlibCount())
	for _, m := range StdlibModules() {
		have[m] = struct{}{}
	}
	for _, w := range want {
		if _, ok := have[w]; !ok {
			t.Errorf("critical stdlib module %q missing from index", w)
		}
	}
}

func TestStdlibCopyIsolation(t *testing.T) {
	a := StdlibModules()
	if len(a) == 0 {
		t.Skip("empty index")
	}
	a[0] = "scrambled"
	b := StdlibModules()
	if b[0] == "scrambled" {
		t.Error("StdlibModules() must return an isolated copy")
	}
}

func head(s []string, n int) []string {
	if len(s) < n {
		return s
	}
	return s[:n]
}

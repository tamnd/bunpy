package runtime

// StdlibModules returns the list of Python stdlib modules embedded
// in this bunpy binary. The list is sorted and deduplicated; the
// caller may modify the returned slice.
//
// The slice mirrors the goipy build pinned in scripts/sync-deps.sh
// at the time bunpy was built. Generated at sync time by
// scripts/sync-stdlib-index.sh.
func StdlibModules() []string {
	out := make([]string, len(Modules))
	copy(out, Modules)
	return out
}

// StdlibCount returns the number of embedded stdlib modules.
func StdlibCount() int {
	return len(Modules)
}

package runtime

import "runtime"

// Build-time metadata. Set via -ldflags "-X github.com/tamnd/bunpy/v1/runtime.<field>=...".
// Defaults distinguish a dev build from a release build: a dev build prints
// "bunpy dev" and hides commit/buildDate/toolchain lines so it cannot lie
// about identity.
//
// Source of truth for the toolchain pins is scripts/sync-deps.sh; the build
// pipeline reads that file via scripts/build-ldflags.sh and bakes the same
// commits into Goipy/Gocopy/Gopapy here.
var (
	Version   = "dev"
	Commit    = ""
	BuildDate = ""
	Goipy     = ""
	Gocopy    = ""
	Gopapy    = ""
)

// BuildInfo is the JSON-serialisable view of build metadata.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
	Goipy     string `json:"goipy,omitempty"`
	Gocopy    string `json:"gocopy,omitempty"`
	Gopapy    string `json:"gopapy,omitempty"`
	Go        string `json:"go"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Build returns the metadata baked into this binary.
func Build() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		Goipy:     Goipy,
		Gocopy:    Gocopy,
		Gopapy:    Gopapy,
		Go:        runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

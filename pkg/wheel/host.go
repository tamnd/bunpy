package wheel

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/tamnd/bunpy/v1/pkg/pypi"
)

// HostTagsOptions overrides parts of host tag detection. The zero
// value uses runtime.GOOS / GOARCH and best-effort libc probing.
type HostTagsOptions struct {
	GOOS    string
	GOARCH  string
	GlibcMa int    // 2 for manylinux2014; 0 to skip glibc detection
	GlibcMi int    // e.g. 28 for manylinux_2_28
	HasMusl bool   // emit musllinux tags
	MacosMa int    // macOS major version, e.g. 14
	MacosMi int    // macOS minor (used for older 10.x rows)
	PyTag   string // interpreter implementation tag, default "py3"
	PyMinor string // e.g. "py314"; falls back to runtime when empty
}

// HostTags returns the ranked host tag set: most specific first,
// `py3-none-any` last. The ranking is what wheel.Pick consumes.
func HostTags() []Tag {
	if env := os.Getenv("BUNPY_HOST_TAGS"); env != "" {
		if tags, ok := parseHostTagsEnv(env); ok {
			return tags
		}
	}
	return HostTagsWith(HostTagsOptions{})
}

// HostTagsWith is HostTags with detection knobs exposed for tests.
func HostTagsWith(opts HostTagsOptions) []Tag {
	if opts.GOOS == "" {
		opts.GOOS = runtime.GOOS
	}
	if opts.GOARCH == "" {
		opts.GOARCH = runtime.GOARCH
	}
	if opts.PyTag == "" {
		opts.PyTag = "py3"
	}
	if opts.PyMinor == "" {
		opts.PyMinor = "py314"
	}

	cpyMinor := strings.Replace(opts.PyMinor, "py", "cp", 1)
	pyTags := []string{cpyMinor, opts.PyMinor, "cp3", opts.PyTag}
	abiTags := []string{cpyMinor, "abi3", "none"}
	platTags := platformTagsFor(opts)

	var out []Tag
	for _, plat := range platTags {
		for _, py := range pyTags {
			for _, abi := range abiTags {
				out = append(out, Tag{Python: py, ABI: abi, Platform: plat})
			}
		}
	}
	out = append(out, Tag{Python: "py3", ABI: "none", Platform: "any"})
	return out
}

func platformTagsFor(opts HostTagsOptions) []string {
	switch opts.GOOS {
	case "linux":
		return linuxPlatformTags(opts)
	case "darwin":
		return darwinPlatformTags(opts)
	case "windows":
		return windowsPlatformTags(opts)
	}
	return []string{opts.GOOS + "_" + opts.GOARCH}
}

func linuxPlatformTags(opts HostTagsOptions) []string {
	arch := linuxArch(opts.GOARCH)
	var out []string
	if opts.GlibcMa == 0 {
		opts.GlibcMa, opts.GlibcMi = detectGlibc()
	}
	if opts.GlibcMa > 0 {
		for mi := opts.GlibcMi; mi >= 17; mi-- {
			out = append(out, fmt.Sprintf("manylinux_%d_%d_%s", opts.GlibcMa, mi, arch))
		}
		out = append(out,
			fmt.Sprintf("manylinux2014_%s", arch),
			fmt.Sprintf("manylinux2010_%s", arch),
			fmt.Sprintf("manylinux1_%s", arch),
		)
	}
	if opts.HasMusl || detectMusl() {
		for mi := 2; mi >= 0; mi-- {
			out = append(out, fmt.Sprintf("musllinux_1_%d_%s", mi, arch))
		}
	}
	out = append(out, fmt.Sprintf("linux_%s", arch))
	return out
}

func linuxArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	case "386":
		return "i686"
	}
	return goarch
}

func darwinPlatformTags(opts HostTagsOptions) []string {
	arch := darwinArch(opts.GOARCH)
	var out []string
	major := opts.MacosMa
	if major == 0 {
		if opts.GOARCH == "arm64" {
			major = 11
		} else {
			major = 10
		}
	}
	for ma := major; ma >= floor(opts); ma-- {
		out = append(out, fmt.Sprintf("macosx_%d_0_%s", ma, arch))
	}
	if opts.GOARCH == "amd64" {
		// 10.x rows for x86_64 hosts.
		for mi := 15; mi >= 9; mi-- {
			out = append(out, fmt.Sprintf("macosx_10_%d_%s", mi, arch))
		}
	}
	out = append(out, fmt.Sprintf("macosx_%d_0_universal2", major))
	return out
}

func darwinArch(goarch string) string {
	switch goarch {
	case "arm64":
		return "arm64"
	case "amd64":
		return "x86_64"
	}
	return goarch
}

func floor(opts HostTagsOptions) int {
	if opts.GOARCH == "arm64" {
		return 11
	}
	return 11
}

func windowsPlatformTags(opts HostTagsOptions) []string {
	switch opts.GOARCH {
	case "amd64":
		return []string{"win_amd64"}
	case "arm64":
		return []string{"win_arm64"}
	case "386":
		return []string{"win32"}
	}
	return []string{"win_" + opts.GOARCH}
}

// parseHostTagsEnv accepts a small set of canned profiles for tests.
// Format: "linux_amd64_glibc228" / "darwin_arm64_14" / "windows_amd64".
func parseHostTagsEnv(s string) ([]Tag, bool) {
	switch s {
	case "linux_amd64_glibc228":
		return HostTagsWith(HostTagsOptions{GOOS: "linux", GOARCH: "amd64", GlibcMa: 2, GlibcMi: 28}), true
	case "linux_amd64_musl":
		return HostTagsWith(HostTagsOptions{GOOS: "linux", GOARCH: "amd64", HasMusl: true}), true
	case "linux_arm64_glibc228":
		return HostTagsWith(HostTagsOptions{GOOS: "linux", GOARCH: "arm64", GlibcMa: 2, GlibcMi: 28}), true
	case "darwin_arm64_14":
		return HostTagsWith(HostTagsOptions{GOOS: "darwin", GOARCH: "arm64", MacosMa: 14}), true
	case "darwin_amd64_13":
		return HostTagsWith(HostTagsOptions{GOOS: "darwin", GOARCH: "amd64", MacosMa: 13}), true
	case "windows_amd64":
		return HostTagsWith(HostTagsOptions{GOOS: "windows", GOARCH: "amd64"}), true
	}
	return nil, false
}

// Pick chooses the best wheel from files for the given ranked tag
// set. Returns the file plus its rank (lower is better) so callers
// can break ties. Yanked or non-wheel files are ignored.
func Pick(files []pypi.File, tags []Tag) (pypi.File, bool) {
	bestRank := -1
	var best pypi.File
	for _, f := range files {
		if f.Kind != "wheel" || f.Yanked {
			continue
		}
		rank, ok := matchWheel(f.Filename, tags)
		if !ok {
			continue
		}
		if bestRank < 0 || rank < bestRank {
			bestRank = rank
			best = f
		}
	}
	if bestRank < 0 {
		return pypi.File{}, false
	}
	return best, true
}

// matchWheel parses filename's tag triple and returns the lowest
// rank against tags. PEP 427 allows dotted compressed sets in each
// of the three positions; we expand them.
func matchWheel(filename string, tags []Tag) (int, bool) {
	base := strings.TrimSuffix(filename, ".whl")
	parts := strings.Split(base, "-")
	if len(parts) < 5 {
		return 0, false
	}
	py := parts[len(parts)-3]
	abi := parts[len(parts)-2]
	plat := parts[len(parts)-1]
	pys := strings.Split(py, ".")
	abis := strings.Split(abi, ".")
	plats := strings.Split(plat, ".")
	bestRank := -1
	for _, p := range pys {
		for _, a := range abis {
			for _, pl := range plats {
				for i, t := range tags {
					if t.Python == p && t.ABI == a && t.Platform == pl {
						if bestRank < 0 || i < bestRank {
							bestRank = i
						}
					}
				}
			}
		}
	}
	if bestRank < 0 {
		return 0, false
	}
	return bestRank, true
}

// LoadMetadata reads the dist-info/METADATA bytes out of a wheel
// archive without extracting anything else.
func LoadMetadata(body []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("wheel: read zip: %w", err)
	}
	for _, f := range zr.File {
		name := f.Name
		if !strings.HasSuffix(name, "/METADATA") {
			continue
		}
		head := strings.TrimSuffix(name, "/METADATA")
		if !strings.HasSuffix(head, ".dist-info") {
			continue
		}
		if strings.Contains(head, "/") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("wheel: METADATA not found")
}

// detectGlibc reads the host's libc.so.6 for a GLIBC_x.y string.
// Returns (0, 0) on any failure so the caller can drop manylinux
// tags entirely.
func detectGlibc() (int, int) {
	candidates := []string{
		"/lib/x86_64-linux-gnu/libc.so.6",
		"/lib/aarch64-linux-gnu/libc.so.6",
		"/lib64/libc.so.6",
		"/lib/libc.so.6",
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			if ma, mi := scanGlibc(data); ma > 0 {
				return ma, mi
			}
		}
	}
	return 0, 0
}

func scanGlibc(data []byte) (int, int) {
	needle := []byte("GLIBC_")
	bestMa, bestMi := 0, 0
	for i := 0; i+len(needle) < len(data); i++ {
		if !bytes.Equal(data[i:i+len(needle)], needle) {
			continue
		}
		j := i + len(needle)
		ma := 0
		for j < len(data) && data[j] >= '0' && data[j] <= '9' {
			ma = ma*10 + int(data[j]-'0')
			j++
		}
		if j >= len(data) || data[j] != '.' {
			continue
		}
		j++
		mi := 0
		for j < len(data) && data[j] >= '0' && data[j] <= '9' {
			mi = mi*10 + int(data[j]-'0')
			j++
		}
		if ma > bestMa || (ma == bestMa && mi > bestMi) {
			bestMa, bestMi = ma, mi
		}
	}
	return bestMa, bestMi
}

func detectMusl() bool {
	candidates := []string{
		"/lib/ld-musl-x86_64.so.1",
		"/lib/ld-musl-aarch64.so.1",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

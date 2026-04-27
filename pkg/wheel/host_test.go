package wheel

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/pypi"
)

func TestHostTagsLinuxAMD64(t *testing.T) {
	tags := HostTagsWith(HostTagsOptions{GOOS: "linux", GOARCH: "amd64", GlibcMa: 2, GlibcMi: 28})
	plats := uniquePlatforms(tags)
	want := []string{"manylinux_2_28_x86_64", "manylinux2014_x86_64", "linux_x86_64", "any"}
	for _, w := range want {
		if !contains(plats, w) {
			t.Errorf("missing %q in %v", w, plats)
		}
	}
}

func TestHostTagsDarwinARM64(t *testing.T) {
	tags := HostTagsWith(HostTagsOptions{GOOS: "darwin", GOARCH: "arm64", MacosMa: 14})
	plats := uniquePlatforms(tags)
	want := []string{"macosx_14_0_arm64", "macosx_11_0_arm64", "any"}
	for _, w := range want {
		if !contains(plats, w) {
			t.Errorf("missing %q in %v", w, plats)
		}
	}
}

func TestHostTagsWindowsAMD64(t *testing.T) {
	tags := HostTagsWith(HostTagsOptions{GOOS: "windows", GOARCH: "amd64"})
	plats := uniquePlatforms(tags)
	want := []string{"win_amd64", "any"}
	for _, w := range want {
		if !contains(plats, w) {
			t.Errorf("missing %q in %v", w, plats)
		}
	}
}

func TestHostTagsTerminatesAtAny(t *testing.T) {
	tags := HostTagsWith(HostTagsOptions{GOOS: "linux", GOARCH: "amd64", GlibcMa: 2, GlibcMi: 28})
	last := tags[len(tags)-1]
	if last.Python != "py3" || last.ABI != "none" || last.Platform != "any" {
		t.Errorf("last tag = %+v, want py3-none-any", last)
	}
}

func TestPickPrefersPlatformOverUniversal(t *testing.T) {
	tags := HostTagsWith(HostTagsOptions{GOOS: "linux", GOARCH: "amd64", GlibcMa: 2, GlibcMi: 28})
	files := []pypi.File{
		{Filename: "widget-1.0-py3-none-any.whl", Kind: "wheel"},
		{Filename: "widget-1.0-cp314-cp314-manylinux2014_x86_64.whl", Kind: "wheel"},
	}
	got, ok := Pick(files, tags)
	if !ok || !strings.Contains(got.Filename, "manylinux2014") {
		t.Errorf("Pick = %+v, want manylinux", got)
	}
}

func TestPickFallsBackToUniversal(t *testing.T) {
	tags := HostTagsWith(HostTagsOptions{GOOS: "linux", GOARCH: "amd64", GlibcMa: 2, GlibcMi: 28})
	files := []pypi.File{
		{Filename: "widget-1.0-py3-none-any.whl", Kind: "wheel"},
	}
	got, ok := Pick(files, tags)
	if !ok || !strings.Contains(got.Filename, "py3-none-any") {
		t.Errorf("Pick = %+v, want py3-none-any", got)
	}
}

func TestPickRespectsTagOrder(t *testing.T) {
	tags := HostTagsWith(HostTagsOptions{GOOS: "linux", GOARCH: "amd64", GlibcMa: 2, GlibcMi: 28})
	files := []pypi.File{
		{Filename: "widget-1.0-cp314-cp314-manylinux2010_x86_64.whl", Kind: "wheel"},
		{Filename: "widget-1.0-cp314-cp314-manylinux_2_28_x86_64.whl", Kind: "wheel"},
	}
	got, ok := Pick(files, tags)
	if !ok || !strings.Contains(got.Filename, "manylinux_2_28") {
		t.Errorf("Pick = %+v, want manylinux_2_28", got)
	}
}

func TestPickIgnoresYanked(t *testing.T) {
	tags := HostTagsWith(HostTagsOptions{GOOS: "linux", GOARCH: "amd64", GlibcMa: 2, GlibcMi: 28})
	files := []pypi.File{
		{Filename: "widget-1.0-py3-none-any.whl", Kind: "wheel", Yanked: true},
	}
	if _, ok := Pick(files, tags); ok {
		t.Error("Pick returned a yanked file")
	}
}

func TestLoadMetadataReadsDistInfo(t *testing.T) {
	body := buildFakeWheel(t, "widget-1.0.0", "Metadata-Version: 2.1\nName: widget\nVersion: 1.0.0\n")
	got, err := LoadMetadata(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Name: widget") {
		t.Errorf("metadata: %q", got)
	}
}

func buildFakeWheel(t *testing.T, distName, metadata string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(distName + ".dist-info/METADATA")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(metadata)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func uniquePlatforms(tags []Tag) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range tags {
		if !seen[t.Platform] {
			seen[t.Platform] = true
			out = append(out, t.Platform)
		}
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

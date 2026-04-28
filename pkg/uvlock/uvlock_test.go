package uvlock_test

import (
	"os"
	"testing"

	"github.com/tamnd/bunpy/v1/pkg/uvlock"
)

func readGolden(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read testdata/%s: %v", name, err)
	}
	return data
}

func TestParseGolden(t *testing.T) {
	data := readGolden(t, "requests.lock")
	lock, err := uvlock.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if lock.Version != 1 {
		t.Errorf("Version = %d, want 1", lock.Version)
	}
	if lock.RequiresPython != ">=3.12" {
		t.Errorf("RequiresPython = %q, want >=3.12", lock.RequiresPython)
	}
	if len(lock.Packages) != 2 {
		t.Fatalf("len(Packages) = %d, want 2", len(lock.Packages))
	}

	// certifi
	certifi := lock.Packages[0]
	if certifi.Name != "certifi" {
		t.Errorf("Packages[0].Name = %q, want certifi", certifi.Name)
	}
	if certifi.Version != "2024.2.2" {
		t.Errorf("Packages[0].Version = %q, want 2024.2.2", certifi.Version)
	}
	if certifi.Source.Kind != "registry" {
		t.Errorf("Packages[0].Source.Kind = %q, want registry", certifi.Source.Kind)
	}
	if certifi.Sdist == nil {
		t.Error("Packages[0].Sdist = nil, want non-nil")
	}
	if len(certifi.Wheels) != 1 {
		t.Errorf("len(Packages[0].Wheels) = %d, want 1", len(certifi.Wheels))
	}

	// requests
	req := lock.Packages[1]
	if req.Name != "requests" {
		t.Errorf("Packages[1].Name = %q, want requests", req.Name)
	}
	if len(req.Dependencies) != 4 {
		t.Errorf("len(Packages[1].Dependencies) = %d, want 4", len(req.Dependencies))
	}
	if req.Dependencies[0].Name != "certifi" {
		t.Errorf("Packages[1].Dependencies[0].Name = %q, want certifi", req.Dependencies[0].Name)
	}
	if len(req.Metadata.RequiresDist) != 4 {
		t.Errorf("len(Packages[1].Metadata.RequiresDist) = %d, want 4", len(req.Metadata.RequiresDist))
	}
}

func TestBestWheel(t *testing.T) {
	data := readGolden(t, "requests.lock")
	lock, err := uvlock.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	for _, pkg := range lock.Packages {
		w := pkg.BestWheel()
		if w == nil {
			t.Errorf("%s: BestWheel = nil, want non-nil", pkg.Name)
			continue
		}
		if w.Hash == "" {
			t.Errorf("%s: BestWheel.Hash = empty", pkg.Name)
		}
		fn := w.Filename()
		if fn == "" || fn == "." {
			t.Errorf("%s: BestWheel.Filename() = %q", pkg.Name, fn)
		}
	}
}

func TestLockExists(t *testing.T) {
	dir := t.TempDir()

	if uvlock.LockExists(dir) {
		t.Error("empty dir: LockExists should be false")
	}

	if err := os.WriteFile(dir+"/uv.lock", []byte("version = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !uvlock.LockExists(dir) {
		t.Error("with uv.lock: LockExists should be true")
	}
}

func TestToBunpyLock(t *testing.T) {
	data := readGolden(t, "requests.lock")
	uv, err := uvlock.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	bl := uvlock.ToBunpyLock(uv)
	if len(bl.Packages) != 2 {
		t.Fatalf("len(Packages) = %d, want 2", len(bl.Packages))
	}

	names := map[string]bool{}
	for _, p := range bl.Packages {
		names[p.Name] = true
		if p.URL == "" {
			t.Errorf("%s: URL is empty", p.Name)
		}
		if p.Hash == "" {
			t.Errorf("%s: Hash is empty", p.Name)
		}
		if p.Version == "" {
			t.Errorf("%s: Version is empty", p.Name)
		}
	}
	if !names["certifi"] {
		t.Error("certifi missing from converted lock")
	}
	if !names["requests"] {
		t.Error("requests missing from converted lock")
	}
}

func TestFromBunpyLock(t *testing.T) {
	data := readGolden(t, "requests.lock")
	uv, err := uvlock.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	bl := uvlock.ToBunpyLock(uv)
	graph := map[string][]string{
		"requests": {"certifi", "urllib3"},
	}
	uv2 := uvlock.FromBunpyLock(bl, ">=3.12", graph, nil, nil)

	if uv2.RequiresPython != ">=3.12" {
		t.Errorf("RequiresPython = %q, want >=3.12", uv2.RequiresPython)
	}
	if len(uv2.Packages) != 2 {
		t.Fatalf("len(Packages) = %d, want 2", len(uv2.Packages))
	}

	for _, p := range uv2.Packages {
		if p.Source.Kind != "registry" {
			t.Errorf("%s: Source.Kind = %q, want registry", p.Name, p.Source.Kind)
		}
		if len(p.Wheels) == 0 {
			t.Errorf("%s: no wheels", p.Name)
		}
	}

	// Check dependency graph propagated
	var reqPkg *uvlock.UVPackage
	for i := range uv2.Packages {
		if uv2.Packages[i].Name == "requests" {
			reqPkg = &uv2.Packages[i]
		}
	}
	if reqPkg == nil {
		t.Fatal("requests not found in uv2.Packages")
	}
	if len(reqPkg.Dependencies) != 2 {
		t.Errorf("requests.Dependencies len = %d, want 2", len(reqPkg.Dependencies))
	}
}

func TestBytesParseRoundtrip(t *testing.T) {
	data := readGolden(t, "requests.lock")
	lock1, err := uvlock.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out := lock1.Bytes()
	lock2, err := uvlock.Parse(out)
	if err != nil {
		t.Fatalf("Parse(Bytes()): %v", err)
	}

	if len(lock1.Packages) != len(lock2.Packages) {
		t.Errorf("package count: %d != %d", len(lock1.Packages), len(lock2.Packages))
	}

	for i, p1 := range lock1.Packages {
		if i >= len(lock2.Packages) {
			break
		}
		p2 := lock2.Packages[i]
		if p1.Name != p2.Name {
			t.Errorf("[%d] Name: %q != %q", i, p1.Name, p2.Name)
		}
		if p1.Version != p2.Version {
			t.Errorf("[%d] Version: %q != %q", i, p1.Version, p2.Version)
		}
		if p1.Source.Kind != p2.Source.Kind {
			t.Errorf("[%d] Source.Kind: %q != %q", i, p1.Source.Kind, p2.Source.Kind)
		}
		if len(p1.Dependencies) != len(p2.Dependencies) {
			t.Errorf("[%d] Dependencies len: %d != %d", i, len(p1.Dependencies), len(p2.Dependencies))
		}
		if len(p1.Wheels) != len(p2.Wheels) {
			t.Errorf("[%d] Wheels len: %d != %d", i, len(p1.Wheels), len(p2.Wheels))
		}
	}
}

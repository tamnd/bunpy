package bunpy_test

import (
	"testing"

	goipyObject "github.com/tamnd/goipy/object"
	goipyVM "github.com/tamnd/goipy/vm"

	bunpyAPI "github.com/tamnd/bunpy/v1/api/bunpy"
)

func newInterp() *goipyVM.Interp {
	i := goipyVM.New()
	i.SetNativeModules(bunpyAPI.Modules())
	return i
}

func callFn(m *goipyObject.Module, name string, args ...goipyObject.Object) (goipyObject.Object, error) {
	v, ok := m.Dict.GetStr(name)
	if !ok {
		return nil, nil
	}
	fn, ok := v.(*goipyObject.BuiltinFunc)
	if !ok {
		return nil, nil
	}
	return fn.Call(nil, args, nil)
}

// --- bunpy.base64 ---

func TestBase64EncodeStr(t *testing.T) {
	m := bunpyAPI.BuildBase64(newInterp())
	got, err := callFn(m, "encode", &goipyObject.Str{V: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	want := "aGVsbG8="
	if s, ok := got.(*goipyObject.Str); !ok || s.V != want {
		t.Fatalf("encode('hello') = %v, want %q", got, want)
	}
}

func TestBase64EncodeBytes(t *testing.T) {
	m := bunpyAPI.BuildBase64(newInterp())
	got, err := callFn(m, "encode", &goipyObject.Bytes{V: []byte("hello")})
	if err != nil {
		t.Fatal(err)
	}
	want := "aGVsbG8="
	if s, ok := got.(*goipyObject.Str); !ok || s.V != want {
		t.Fatalf("encode(b'hello') = %v, want %q", got, want)
	}
}

func TestBase64DecodeRoundTrip(t *testing.T) {
	m := bunpyAPI.BuildBase64(newInterp())
	enc, _ := callFn(m, "encode", &goipyObject.Str{V: "round trip"})
	dec, err := callFn(m, "decode", enc)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := dec.(*goipyObject.Bytes)
	if !ok {
		t.Fatalf("decode returned %T, want Bytes", dec)
	}
	if string(b.V) != "round trip" {
		t.Fatalf("round trip mismatch: got %q", b.V)
	}
}

func TestBase64URLRoundTrip(t *testing.T) {
	m := bunpyAPI.BuildBase64(newInterp())
	data := &goipyObject.Bytes{V: []byte("\xff\xfe\xfd")}
	enc, err := callFn(m, "encode_url", data)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := callFn(m, "decode_url", enc)
	if err != nil {
		t.Fatal(err)
	}
	b := dec.(*goipyObject.Bytes)
	if string(b.V) != "\xff\xfe\xfd" {
		t.Fatalf("URL round trip mismatch")
	}
}

// --- bunpy.gzip ---

func TestGzipRoundTrip(t *testing.T) {
	m := bunpyAPI.BuildGzip(newInterp())
	original := []byte("hello gzip world")
	compressed, err := callFn(m, "compress", &goipyObject.Bytes{V: original})
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := callFn(m, "decompress", compressed)
	if err != nil {
		t.Fatal(err)
	}
	b := decompressed.(*goipyObject.Bytes)
	if string(b.V) != string(original) {
		t.Fatalf("gzip round trip mismatch: got %q", b.V)
	}
}

func TestGzipCompressLevelZero(t *testing.T) {
	m := bunpyAPI.BuildGzip(newInterp())
	data := &goipyObject.Bytes{V: []byte("test data")}
	level := goipyObject.NewInt(0)
	out, err := callFn(m, "compress", data, level)
	if err != nil {
		t.Fatal(err)
	}
	// decompress must recover the original
	dec, err := callFn(m, "decompress", out)
	if err != nil {
		t.Fatal(err)
	}
	if string(dec.(*goipyObject.Bytes).V) != "test data" {
		t.Fatal("level-0 compress/decompress mismatch")
	}
}

// --- top-level bunpy module ---

func TestBunpyModuleVersion(t *testing.T) {
	i := newInterp()
	m := bunpyAPI.BuildBunpy(i)
	v, ok := m.Dict.GetStr("__version__")
	if !ok {
		t.Fatal("bunpy.__version__ not set")
	}
	if _, ok := v.(*goipyObject.Str); !ok {
		t.Fatalf("bunpy.__version__ type %T, want Str", v)
	}
}

func TestBunpyModuleHasBase64(t *testing.T) {
	m := bunpyAPI.BuildBunpy(newInterp())
	if _, ok := m.Dict.GetStr("base64"); !ok {
		t.Fatal("bunpy.base64 sub-module missing")
	}
}

func TestBunpyModuleHasGzip(t *testing.T) {
	m := bunpyAPI.BuildBunpy(newInterp())
	if _, ok := m.Dict.GetStr("gzip"); !ok {
		t.Fatal("bunpy.gzip sub-module missing")
	}
}

func TestNativeModulesMap(t *testing.T) {
	mods := bunpyAPI.Modules()
	for _, name := range []string{"bunpy", "bunpy.base64", "bunpy.gzip"} {
		if _, ok := mods[name]; !ok {
			t.Errorf("Modules() missing %q", name)
		}
	}
}

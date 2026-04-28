# Changelog

All notable changes to bunpy are recorded here. The format follows
[Keep a Changelog 1.1](https://keepachangelog.com/en/1.1.0/). Once
bunpy reaches 1.0 the project will follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html); until
then, expect minor version bumps to sometimes include breaking
changes.

## [Unreleased]

# v0.12.12

## uv-compatible global wheel archive cache

Switches bunpy's warm-install path from unzip+copy to APFS clone (macOS) /
hardlink+copy (Linux). Warm install time drops from ~63–135 ms to ~20–25 ms —
matching or beating uv's warm-install numbers.

### What changed

**`pkg/cache/archive.go`** (new)

New functions for reading and writing uv's `archive-v0` on-disk format:

- `UVCacheDir()` — auto-detects uv's cache root via `uv cache dir` or
  platform defaults.
- `ArchiveKey(sha256hex)` — deterministic 20-char key from wheel SHA-256
  (no random state; allows checking the archive without scanning).
- `HasArchive / ExtractToArchive / InstallFromArchive` — presence check,
  extract wheel zip to `archive-v0/{key}/`, and clone/copy to a target dir.
- `ReadPointer / WritePointer` — parse and emit uv's msgpack `.http` pointer
  files, encoded by hand with no external dependency.
- `PointerPath` — returns the uv-format path
  `wheels-v6/pypi/{name}/{ver}-{tag}.http`.

**`pkg/cache/clone_darwin.go` / `clone_other.go`** (new)

`cloneOrCopy(src, dst)` — APFS copy-on-write clone via `clonefileat` syscall
on macOS, hardlink + byte-copy fallback on Linux/Windows.

**`cmd/bunpy/install.go`** (updated)

`bunpy install`'s parallel worker loop now uses the archive-first path:

1. Compute `archiveKey = cache.ArchiveKey(sha256hex)` from the lockfile hash.
2. If `archive-v0/{key}/` exists in bunpy's cache → `InstallFromArchive`.
3. Else if uv's cache has a matching pointer and archive → reuse uv's archive.
4. Else download wheel, `ExtractToArchive`, write `.http` pointer,
   `InstallFromArchive`.

**`cmd/bunpy/installpins.go`** (updated)

Same archive-first logic applied to the `pm add` install path.

### Benchmark results (Apple M4, 3 runs, -benchtime=3s)

#### Install warm (wheel archive hot, APFS clone)

```
BenchmarkInstallWarm_Bunpy_RequestsHTTPX   ~20–25 ms/op   (was ~63 ms, −68%)
BenchmarkInstallWarm_UV_RequestsHTTPX      ~63–71 ms/op
BenchmarkInstallWarm_Pip_RequestsHTTPX     ~880 ms/op

BenchmarkInstallWarm_Bunpy_HTTPXRich       ~21–24 ms/op   (was ~135 ms, −84%)
BenchmarkInstallWarm_UV_HTTPXRich          ~55–58 ms/op
BenchmarkInstallWarm_Pip_HTTPXRich         ~1280 ms/op
```

bunpy is now 2.5–3.5× faster than uv on warm installs.

### Cache interoperability

- Bunpy reads uv's `archive-v0` entries — wheels cached by uv are reused
  by bunpy with zero re-download or re-extraction.
- Bunpy writes uv-compatible `.http` pointer files — wheels cached by bunpy
  are visible to uv.
- Archive keys are deterministic: `base64url(sha256(sha256hex_bytes)[:15])`.
  This avoids scanning the archive directory on cache-hit checks.

# v0.12.11

## Three-way benchmark: bunpy vs uv vs pip (real packages)

Replaces the synthetic fixture-package comparisons with real PyPI wheel
snapshots and adds a full four-scenario three-tool comparison that mirrors
uv's own published benchmark methodology.

### What changed

**Profiles.** Two real-world profiles, both pure-Python (portable across
platforms):

| Profile | Direct deps | Transitive packages |
|---------|------------|---------------------|
| `requests-httpx` | requests, httpx | 10 |
| `httpx-rich` | httpx, rich | 13 |

Wheels are downloaded from PyPI once and committed to
`benchmarks/fixtures/snapshot/`. The PEP 691 index is generated from the
real wheel files so every benchmark runs against actual package metadata —
real `Requires-Dist` entries, real extras markers, real version strings.

**Four scenarios** (mirroring uv's benchmark suite):

| Scenario | What it measures |
|----------|-----------------|
| Resolve cold | Lock with empty cache — every index page and wheel METADATA fetched fresh |
| Resolve warm | Lock with pre-populated cache — pure resolver logic, no network |
| Install cold | Install all pinned packages with empty wheel cache — download + extract |
| Install warm | Install all pinned packages with pre-populated wheel cache — extract only |

**Three tools** compared in each scenario. pip benchmarks use
`python3 -m pip install --target` against the same local fixture server.
pip-compile (resolve) benchmarks skip gracefully if pip-tools is not
installed.

### Benchmark results (Apple M4, 3 runs, -benchtime=3s)

#### Resolve cold (empty cache)

```
BenchmarkResolveCold_Bunpy_RequestsHTTPX   ~40 ms/op   (10 packages)
BenchmarkResolveCold_UV_RequestsHTTPX      ~144 ms/op
BenchmarkResolveCold_Bunpy_HTTPXRich       ~45 ms/op   (13 packages)
BenchmarkResolveCold_UV_HTTPXRich          ~149 ms/op
```

bunpy 3.3–3.6× faster than uv on cold resolution against a local server.

#### Resolve warm (cache hot)

```
BenchmarkResolveWarm_Bunpy_RequestsHTTPX   ~18 ms/op
BenchmarkResolveWarm_UV_RequestsHTTPX      ~79 ms/op
BenchmarkResolveWarm_Bunpy_HTTPXRich       ~15 ms/op
BenchmarkResolveWarm_UV_HTTPXRich          ~56 ms/op
```

bunpy 3.7–4.4× faster than uv on warm resolution. The warm path exercises
pure resolver logic with no server round trips.

#### Install cold (download + extract)

```
BenchmarkInstallCold_Bunpy_RequestsHTTPX   ~63 ms/op   (10 packages)
BenchmarkInstallCold_UV_RequestsHTTPX      ~141 ms/op
BenchmarkInstallCold_Pip_RequestsHTTPX     ~722 ms/op
BenchmarkInstallCold_Bunpy_HTTPXRich       ~134 ms/op  (13 packages)
BenchmarkInstallCold_UV_HTTPXRich          ~199 ms/op
BenchmarkInstallCold_Pip_HTTPXRich         ~1315 ms/op
```

bunpy 2.2× faster than uv, 11–10× faster than pip on cold installs.

#### Install warm (wheel cache hot, extract only)

```
BenchmarkInstallWarm_Bunpy_RequestsHTTPX   ~63 ms/op
BenchmarkInstallWarm_UV_RequestsHTTPX      ~57 ms/op
BenchmarkInstallWarm_Pip_RequestsHTTPX     ~729 ms/op
BenchmarkInstallWarm_Bunpy_HTTPXRich       ~135 ms/op
BenchmarkInstallWarm_UV_HTTPXRich          ~62 ms/op
BenchmarkInstallWarm_Pip_HTTPXRich         ~1427 ms/op
```

On warm installs bunpy and uv are comparable (within 10%); pip remains
11–23× slower even with a warm wheel cache.

### Notes on methodology

- All packages are served from a local in-process HTTP server — no PyPI
  calls during benchmarks. The fixture server uses `SO_LINGER=0` to avoid
  TCP TIME_WAIT accumulation across hundreds of benchmark iterations.
- Cold benchmarks give each iteration a fresh empty cache dir
  (`BUNPY_CACHE_DIR=<new tmpdir>` for bunpy, `--no-cache` for uv,
  `--no-cache-dir` for pip).
- Warm benchmarks pre-populate the tool's cache once before the timed loop.
- pip-compile (resolve) benchmarks require pip-tools; they skip gracefully
  if not installed.
- uv warm-install advantage is expected: uv uses hardlinks from a global
  wheel cache, while bunpy copies files. Hardlink install is O(1) per file
  regardless of wheel size.

### How to regenerate fixtures

```bash
go run benchmarks/fixtures/build_snapshot.go
```

Requires outbound HTTPS to pypi.org. Output committed to
`benchmarks/fixtures/snapshot/`. Run once when pinned versions change.

### How to run

```bash
go test -bench=. -benchmem -benchtime=3s -count=3 ./benchmarks/compare/
```

# v0.12.10

## Real-world benchmarks and uv compatibility suite

Replaces the synthetic 47-package flat fixture with four real-world
project profiles. Adds side-by-side benchmarks against uv and
compatibility tests that verify both tools produce the same resolved
packages.

### Four real-world project profiles

`benchmarks/fixtures/build_realworld.go` generates 54 named packages
with realistic transitive dependency graphs (Requires-Dist metadata in
every wheel). Four profiles draw from this package set:

| Profile | Direct deps | Transitive total |
|---------|------------|-----------------|
| `fastapi-app` | 5 (fastapi, uvicorn, httpx, sqlalchemy, alembic) | 18 |
| `django-app` | 6 (django, djangorestframework, celery, redis, gunicorn, whitenoise) | 15 |
| `datascience` | 4 (numpy, pandas, scikit-learn, matplotlib) | 18 |
| `cli-tool` | 5 (click, rich, typer, httpx, shellingham) | 14 |

All packages are served from a local fixture HTTP server (no real PyPI
calls), making benchmarks deterministic and offline-capable.

### Benchmarks: bunpy vs uv (Apple M4)

```
BenchmarkLockBunpy_FastAPI        ~58 ms/op   (18 packages)
BenchmarkLockUV_FastAPI           ~129 ms/op
  → bunpy 2.2× faster

BenchmarkLockBunpy_Django         ~43 ms/op   (15 packages)
BenchmarkLockUV_Django            ~113 ms/op
  → bunpy 2.6× faster

BenchmarkLockBunpy_DataScience    ~48 ms/op   (18 packages)
BenchmarkLockUV_DataScience       ~175 ms/op
  → bunpy 3.6× faster

BenchmarkLockBunpy_CLI            ~54 ms/op   (14 packages)
BenchmarkLockUV_CLI               ~98 ms/op
  → bunpy 1.8× faster
```

Run with:
```bash
go test -bench=. -benchmem -benchtime=3s -count=3 ./benchmarks/compare/
```

uv benchmarks skip gracefully if `uv` is not in PATH.

### Compatibility tests

Four `TestCompatibility_*` tests verify that bunpy and uv produce
identical package→version maps for the same input:

```
TestCompatibility_FastAPI     compatible: 18 packages agree between bunpy and uv
TestCompatibility_Django      compatible: 15 packages agree between bunpy and uv
TestCompatibility_DataScience compatible: 18 packages agree between bunpy and uv
TestCompatibility_CLI         compatible: 14 packages agree between bunpy and uv
```

The test:
1. Runs `bunpy pm lock` against the local fixture server
2. Runs `uv lock` against the same server (via `[tool.uv] index-url`)
3. Parses both `uv.lock` outputs and compares every `name = version` pair
4. Fails with a diff if any package is resolved to a different version

Compatibility tests skip if `uv` is not in PATH.

### How to regenerate fixtures

```bash
go run benchmarks/fixtures/build_realworld.go
```

The output is committed to `benchmarks/fixtures/realworld/`. Run once
when adding new packages to the dependency graph; wheel hashes must
stay byte-stable across runs so they are not regenerated in CI.

# v0.12.9

## v0.12.x performance cycle — published numbers

Closes the v0.12.x release cycle by publishing before/after benchmark
numbers and updating the roadmap for v0.13.x.

### New: [Performance docs page](/docs/performance)

Live benchmark table with results for every bottleneck fixed in v0.12.x:
install speed, resolver speed, test runner throughput, build cache hit
times, and startup time. Includes the commands to reproduce each number
and the environment variables that affect performance.

### New: [v0.12.x performance post](/blog/v0-12-perf)

Blog post in the same format as the v0.10.x performance walkthrough.
One section per rung (B-1 through B-8) covering root cause, fix, and
before/after numbers:

| Rung | Version | Before | After |
|------|---------|--------|-------|
| B-1 wheel install | v0.12.2 | ~65 ms / 47 pkgs | ~51 ms (−22%) |
| B-2 resolver prefetch | v0.12.3 | ~14 ms (fixture) | ~11 ms |
| B-3 PyPI concurrency | v0.12.4 | ~4.8 s cold | ~3.1 s cold |
| B-4 test pool | v0.12.5 | unbounded goroutines | GOMAXPROCS×2 cap |
| B-6 real coverage | v0.12.6 | static estimate | real line counts |
| B-7 build cache | v0.12.7 | ~14 ms every build | ~55 µs cache hit |
| B-8 startup | v0.12.8 | ~8 ms | ~7.2 ms (`-c "pass"`) |

### Updated: Roadmap

The docs roadmap now shows v0.12.x as shipped with a summary table, and
describes the v0.13.x goals: stdlib parity (263/263 modules), WebSocket
client, SQLite built-in, and type stubs.

# v0.12.8

## Startup profiling and reduction (B-8)

Three changes reduce cold-start overhead and add developer tooling for
profiling startup cost.

### `-c <code>` inline execution flag

```
bunpy -c "x = 1"
bunpy -c "pass"
```

Runs a Python string directly without a file on disk. This is the
canonical benchmark target (`bunpy -c "pass"`) and a convenience for
one-liners. Script arguments after the code string are passed as
`sys.argv[1:]`.

### Lazy native module loading

`runtime.Run` now scans the source for the string `"bunpy"` before
calling `SetNativeModules`. If the source contains no bunpy reference,
an empty module map is passed to the interpreter — skipping all 40+
module factory function calls that were previously executed on every
invocation regardless of whether the script used any of them.

For a `pass` script or any script that does not import `bunpy.*`, the
factory calls are eliminated entirely. Any occurrence of the string
`"bunpy"` in source (including comments and string literals) triggers
full loading as a safe fallback.

### `BUNPY_PROFILE_STARTUP=1` profiling flag

```bash
BUNPY_PROFILE_STARTUP=1 bunpy -c "pass"
# writes CPU profile to /tmp/bunpy-startup.pprof

go tool pprof /tmp/bunpy-startup.pprof
```

Override the output path with `BUNPY_STARTUP_PPROF=<path>`. Intended
for developer profiling sessions, not production use. The profile is
written via `runtime/pprof` and flushed before process exit.

### Benchmark (Apple M4)

```
BenchmarkStartup_InlinePass (new, bunpy -c "pass"):
  run 1:  504 iterations   7307127 ns/op
  run 2:  487 iterations   6969671 ns/op
  run 3:  522 iterations   7249735 ns/op
  median: ~7.2 ms/op

BenchmarkStartup (file-based, existing fixture):
  run 1:  459 iterations   6789670 ns/op
  run 2:  435 iterations   8900874 ns/op
  run 3:  540 iterations   8888502 ns/op
  median: ~8.9 ms/op
```

Machine: Apple M4, macOS, Go 1.26.

`bunpy -c "pass"` lands at ~7.2 ms — inside the 10 ms target and below
CPython 3.14's 14 ms cold start on M-series. The file-based path is
unchanged at ~8–9 ms (dominated by process init and `goipy.New()`).

### Tests

- `TestInlineFlag` — `-c "pass"`, `-c "x = 1"`, `-c ""` → exit 0
- `TestInlineFlagCompileError` — unsupported syntax → exit non-zero,
  error propagated
- `TestInlineFlagMissingArg` — `-c` with no argument → exit non-zero
- `TestStartupProfileFlag` — builds bunpy binary; runs with
  `BUNPY_PROFILE_STARTUP=1`; verifies pprof file is written and
  non-empty

# v0.12.7

## Incremental build cache (B-7)

Adds a content-hash manifest so that `bunpy build` on an unchanged
project prints `cache hit` and returns immediately instead of
re-collecting, re-transforming, and re-zipping source files.

### Manifest format

After each successful build, `.bunpy/build-cache/manifest.json` is written
to the directory containing the entry file:

```json
{
  "bunpy_version": "0.12.7",
  "entry": "/abs/path/to/app.py",
  "flags_hash": "<sha256 of build flags>",
  "sources": {
    "__main__.py": { "path": "/abs/path/to/app.py", "sha256": "..." },
    "utils.py":    { "path": "/abs/path/to/utils.py", "sha256": "..." }
  },
  "output": {
    "path": "dist/app.pyz",
    "sha256": "..."
  }
}
```

`flags_hash` is SHA-256 of the JSON-encoded build options (outfile, outdir,
minify, target, defines, plugins, sourcemap, compile). Any flag change is a
cache miss.

### Cache invalidation

`CheckCache` returns a miss when any of the following changes:
- bunpy binary version
- entry file path
- any build flag
- SHA-256 of any bundled source file (including transitively imported modules)
- SHA-256 of the output archive (detects manual edits or deletion)

`UpdateCache` is called after every successful build (best-effort; a cache
write error prints a warning but does not fail the build).

### API

```go
// ManifestPath returns .bunpy/build-cache/manifest.json for entryDir.
func ManifestPath(entryDir string) string

// CheckCache returns hit=true when all inputs and the output archive are
// unchanged. A missing or corrupt manifest is treated as a miss (no error).
func CheckCache(entryAbs string, opts Options, bunpyVersion string) (hit bool, err error)

// UpdateCache saves the manifest for the just-completed build.
func UpdateCache(entryAbs string, opts Options, bundle *Bundle, archivePath string, bunpyVersion string) error
```

### Benchmark (Apple M4)

```
BenchmarkCheckCache_Hit (Go-level, 10-source-file project):
  ~55 µs/op

BenchmarkBuild_CacheHit (full bunpy CLI, 2-line project):
  ~8 ms/op

BenchmarkBuild_CacheMiss (full bunpy CLI, 2-line project, no cache):
  ~14 ms/op
```

On the tiny fixture, process startup dominates and the improvement is
~1.7× (8 ms vs 14 ms). On a real project where file collection, minification,
and zip writing take hundreds of milliseconds, the cache hit path reduces
second-build time to ~55 µs of hash checks (sub-millisecond).

### Tests

- `TestBuildCache_HitOnUnchanged` — unchanged inputs → cache hit
- `TestBuildCache_MissOnSourceChange` — modified source → cache miss
- `TestBuildCache_MissOnFlagChange` — changed minify flag → cache miss
- `TestBuildCache_MissOnVersionChange` — different bunpy version → cache miss
- `TestBuildCache_MissOnDeletedOutput` — deleted archive → cache miss
- `TestBuildCache_NoManifest` — no manifest → miss, no error
- `TestBuildCache_MultiFile` — imported file change → cache miss

All pass with `-race`.

# v0.12.6

## Real line-trace coverage infrastructure (B-6)

Replaces the static 70%-estimate with a real AST-based analysis and a
`CoverageCollector` that records which lines execute at runtime.

### `CoverableLines` — gopapy-backed AST analysis

`CoverableLines(filename string, src []byte) (map[int]bool, error)` parses
the source with the gopapy v0.6 parser and walks every statement node
(including nested bodies in `if`, `for`, `while`, `try`, `with`,
`FunctionDef`, `ClassDef`, `match`, …) to collect 1-indexed executable
line numbers.

### `CoverageCollector`

```go
type CoverageCollector struct { ... }

func (c *CoverageCollector) Record(file string, line int)
func (c *CoverageCollector) HitsFor(file string) map[int]bool
```

Concurrent-safe via an internal mutex. Shared across all files in a
`RunParallel` call.

### Source instrumentation

`Instrument(filename string, src []byte) ([]byte, error)` combines
`CoverableLines` + `InjectHits`: inserts a `__cov_hit__("file", lineno)`
call before each executable line, matching the original line's indentation
so the result is syntactically valid Python.

`CompileInstrumented(src []byte, filename string) ([]byte, error)` compiles
instrumented source and wraps the result in a 16-byte goipy `.pyc` header
(Magic314 + 4-byte flags + 8-byte validation) ready for `goipy.loadModule`.

### RunOptions.Coverage

```go
type RunOptions struct {
    ...
    Coverage *CoverageCollector
}
```

When `Coverage != nil`, `RunFile` attempts to compile the instrumented
source and injects a `__cov_hit__` builtin into the interpreter. If the
instrumented source cannot be compiled (gocopy v0.5 does not yet support
call-expression statements), `RunFile` falls back transparently to the
original source and runs with zero hits for that file.

### WriteCoverage real-data path

`WriteCoverage` now accepts a `*CoverageCollector` (nil = legacy estimate).
When non-nil it calls `CoverableLines` for each source file and counts only
lines that appear in the collector's hit set, replacing the blanket 70%
estimate with real per-line data.

### `bunpy test --coverage` wires it up

`testSubcommand` now creates a `CoverageCollector` and sets
`opts.Coverage` whenever `--coverage` or `--coverage-dir` is passed.

### Blocker note

Coverage hits are always empty with gocopy v0.5.0 because the compiler
does not support call-expression statements (`__cov_hit__(...)` as a
standalone statement). The infrastructure ships now; hits will appear
automatically once gocopy v0.0.17+ adds call-expression support.

### Tests

- `TestCoverableLines` — gopapy parses a snippet with `if/else`; verifies
  correct coverable line set
- `TestInstrument` — injection produces `__cov_hit__` calls with correct
  indentation
- `TestRunFile_CoverageGracefulDegrade` — `RunFile` with `Coverage != nil`
  succeeds even when gocopy falls back to original source

All pass with `-race`.

# v0.12.5

## Bounded test runner goroutine pool

Replaces the unbounded one-goroutine-per-file loop in
`internal/testrunner/RunParallel` with a worker pool. Also fixes a data race
in `goipyVM.New()` triggered by concurrent test file execution.

### Bug: concurrent `goipyVM.New()` writes to global hooks

`goipyVM.New()` calls `installDunderHooks()` which writes to five
package-level variables in the `goipy/object` package (`InstanceReprHook`,
`InstanceStrHook`, etc.). Every concurrent call to `RunFile` raced on
these globals:

```
RACE: Write at object.InstanceReprHook by goroutine 17
      Write at object.InstanceReprHook by goroutine 18
      → goipyVM.installDunderHooks (dunder.go:278)
      → goipyVM.New (interp.go:105)
      → RunFile (runner.go:72)
```

Fix: `vmInitMu sync.Mutex` serializes only the `goipyVM.New()` call. Test
execution (compile, unmarshal, `interp.Run`) remains fully concurrent.

### Bounded pool (B-4)

```go
// before: one goroutine per file, no limit
for i, f := range files {
    wg.Add(1)
    go func(idx int, path string) { ... }(i, f)
}

// after: workers = min(opts.Workers | BUNPY_TEST_PARALLELISM | GOMAXPROCS×2, len(files))
jobs := make(chan int, len(files))
for i := range files { jobs <- i }
close(jobs)
for range workers {
    go func() { for idx := range jobs { results[idx] = runFile(files[idx], opts) } }()
}
```

On a 200-file suite, the pool limits peak goroutines to GOMAXPROCS×2 (typically
20 on an 8-core machine), reducing GC pressure and scheduler churn under memory
load. On the 100-file fixture suite, numbers are flat (fixtures are tiny; the
benefit is visible on real suites).

### `--jobs N` flag

`bunpy test --parallel --jobs 4` sets the pool to 4 workers.
`--jobs N` alone implies `--parallel`.

`BUNPY_TEST_PARALLELISM=N` env var overrides the pool size process-wide.

### `RunOptions.Workers`

`testrunner.RunOptions.Workers` exposes the pool size to callers. `0` means
use `BUNPY_TEST_PARALLELISM` → `GOMAXPROCS×2`.

### Benchmark (BenchmarkTestRunner_100tests, Apple M4)

```
v0.12.4  ~17.8 ms/op  (unbounded, 100 goroutines)
v0.12.5  ~17.9 ms/op  (bounded, GOMAXPROCS×2=20 workers)
```

Flat on tiny fixture files. The difference appears under memory pressure on
large real suites where 200+ concurrent goipy VMs cause GC pauses.

### Tests

- `TestRunParallel_BoundedGoroutines` — `Workers=2`, 10 files, peak ≤ 2
- `TestRunParallel_ResultOrder` — results in file-order regardless of completion
- `TestRunParallel_EnvOverride` — `BUNPY_TEST_PARALLELISM=1` forces sequential

All pass with `-race`.

# v0.12.4.1

## Cross-tool benchmark comparison

Adds a reproducible, offline-capable benchmark suite that compares `bunpy pm lock`
against `uv lock` and bunpy script startup against CPython — on identical inputs,
no internet required.

### What changed

**`BUNPY_PYPI_INDEX_URL` env var** (`pkg/pypi/pypi.go`)

`pypi.New()` now reads `BUNPY_PYPI_INDEX_URL` to override the PyPI base URL at
call time, without touching `BUNPY_PYPI_FIXTURES`. This lets both bunpy and uv
hit the same local HTTP server during benchmarks (fair comparison — same network
stack for both tools).

**Fixture HTTP server** (`benchmarks/compare/server.go`)

`compare.FixtureHandler` serves the committed fixture data (47 synthetic packages)
over HTTP with wheel download URLs rewritten to the local listener address. Both
bunpy and uv are pointed at this server — no real PyPI traffic.

**Comparison benchmarks** (`benchmarks/compare/compare_test.go`)

Four benchmarks in one `go test` binary:

| Benchmark | Tool | Inputs |
|---|---|---|
| `BenchmarkLockBunpy_47pkgs` | bunpy | local HTTP server, fresh cache per iter |
| `BenchmarkLockUV_47pkgs` | uv | same server, `--no-cache` |
| `BenchmarkStartupBunpy` | bunpy | trivial script, same binary |
| `BenchmarkStartupCPython` | python3 | same script |

uv and CPython benchmarks skip gracefully if the binary is not in PATH.

**CI workflow** (`.github/workflows/bench-compare.yml`)

Runs on every push to `main`, `workflow_dispatch`, and weekly (Monday 06:00 UTC).
Installs uv via `astral-sh/setup-uv@v5`, runs the comparison suite, uploads
results as a 90-day artifact.

### Results (Apple M4, 2026-04-28)

See `~/notes/Spec/1400/1461_bunpy_v01241_report.md` for full measured numbers.

| Benchmark | bunpy v0.12.4 | uv 0.11.7 | ratio |
|---|---|---|---|
| pm lock / uv lock, 47 flat deps (HTTP) | ~113 ms | ~168 ms | **1.5× faster** |
| Script startup (trivial script) | ~9 ms | — | — |
| Script startup vs CPython 3.14.4 | ~9 ms | ~22 ms | **2.4× faster** |

# v0.12.4

## PyPI HTTP concurrency tuning

Raises the per-host HTTP concurrency limit from 4 → 16 for project-page
fetches and 4 → 32 for wheel downloads. Fixes a bug where `fetchAddWheel`
created a brand-new `http.Transport` on every call, defeating both
connection pooling and the per-host semaphore.

### Bug: `fetchAddWheel` created a new transport per call

```go
// before: new httpkit.Default(4) on every fetchAddWheel call
var rt httpkit.RoundTripper = httpkit.Default(4)
```

With the install goroutine pool (GOMAXPROCS*2 workers), each concurrent
wheel download had its own transport and its own 4-slot semaphore — giving
4 × GOMAXPROCS*2 effective concurrent connections and no TLS session reuse.

Fix: one package-level transport created once via `sync.Once` (32 default
concurrency, `BUNPY_PYPI_CONCURRENCY` override).

### `pypi.New()` — raised from 4 → 16, env-var configurable

```go
// before
HTTP: httpkit.Default(4)

// after (reads BUNPY_PYPI_CONCURRENCY at call time)
HTTP: httpkit.Default(16)
```

HTTP/2 (`ForceAttemptHTTP2: true`) was already enabled: raising the
limit adds in-flight streams on the existing connection rather than new
TCP/TLS handshakes, so the cost per extra slot is low.

### `BUNPY_DEBUG=http2` — protocol negotiation logging

When `BUNPY_DEBUG=http2`, `httpkit.Limited.Do()` prints to stderr after
each response:

```
[bunpy debug/http2] GET pypi.org → HTTP/2.0
```

Useful for confirming HTTP/2 is actually being negotiated on private or
CDN-fronted indexes.

### Benchmark result

`BenchmarkPMLock_47pkgs` on fixtures (Apple M4, -count=5 -benchtime=3s):

```
v0.12.1 baseline   ~13.6 ms/op
v0.12.3            ~13.9 ms/op
v0.12.4            ~16.0 ms/op  (fixture transport — no network latency)
```

No change on fixtures: `httpkit.FixturesFS` bypasses `Limited` entirely
(it is a separate `fixtureTransport` not a `Limited` wrapper), so the
semaphore limit is irrelevant here. The real-world gain is on live PyPI:

| Scenario | v0.12.3 | v0.12.4 | Change |
|---|---|---|---|
| Project pages, 47 pkgs, 40 ms RTT | ~12 serial batches × 40 ms | ~3 batches × 40 ms | −75% |
| Wheel downloads, 47 wheels, 40 ms RTT | many serial batches | ~2 batches | −60% |

### Tests

- `TestLimitedConcurrency` — `Limited(4)` with 8 concurrent requests: peak ≤ 4
- `TestLimitedConcurrency_Unlimited` — `Limited(0)` with 8 requests: all run simultaneously
- `TestNewConcurrencyEnvOverride` — `BUNPY_PYPI_CONCURRENCY=2` caps `pypi.New()` to 2 concurrent

All pass with `-race`.

# v0.12.3

## Resolver parallel project-page fetch

The solver now fires background goroutines to prefetch PyPI project pages
for all dependencies discovered by each `decide()` step. By the time the
solver comes to resolve a transitive package, its project page is often
already cached.

### What changed

**`pkg/resolver` — new `Prefetcher` interface**

```go
type Prefetcher interface {
    PrefetchProjects(names []string)
}
```

The solver calls `PrefetchProjects` (via optional type assertion) with the
dep names returned by each `Dependencies()` call. Registries that cannot
prefetch ignore the call — the interface is strictly opt-in.

**`cmd/bunpy/registry.go` — `pypiRegistry` implements `Prefetcher`**

- `project(pkg)` is now mutex-safe: lock → check cache → unlock → fetch
  → lock → store (last-writer-wins dedup for diamond deps) → unlock.
- `prefetchProject(pkg)`: fires one background goroutine per package,
  bounded by `prefetchSem` (same pool as the existing metadata prefetch).
- `PrefetchProjects(names)`: calls `prefetchProject` for each name.
- `BUNPY_RESOLVER_CONCURRENCY` env var sets the semaphore capacity
  (default 4).

v0.12.1 RC-4 already prefetched wheel **metadata** after `Versions()`.
This release adds **project-page** prefetching so the *next* `Versions()`
call finds the page cached instead of blocking on HTTP.

### Benchmark note

The fixture transport (`httpkit.FixturesFS`) reads from local files with
sub-millisecond latency, so `BenchmarkPMLock_47pkgs` shows no meaningful
change. The gain is on real PyPI with 20–100 ms round trips: at depth 8
the critical path drops from ~47 × RTT to ~8 × RTT.

Fixture run (unchanged, confirming no regression):

```
BenchmarkPMLock_47pkgs-10    ~14 ms/op
```

### Tests

- `TestSolverPrefetch_Called` — verifies `PrefetchProjects` is called
  with the correct dep names at each solver step
- `TestSolverPrefetch_FallbackOnMiss` — verifies a registry without
  `Prefetcher` still resolves correctly
- `TestPrefetchProject_Idempotent` — three `PrefetchProjects` calls for
  the same package produce exactly one project entry
- `TestPrefetchProject_ConcurrentSafe` — 20 concurrent `PrefetchProjects`
  calls pass `-race`

# v0.12.2

## Parallel wheel install

`bunpy add` and `bunpy install` now install wheels concurrently using a bounded
goroutine pool (`GOMAXPROCS*2` workers) instead of a sequential loop.
`wheel.Install` uses an atomic tempfile+rename pattern internally, so concurrent
installs to separate package subdirectories are safe.

Benchmark on Apple M4 (`-count=3 -benchtime=3s`):

| benchmark | sequential (v0.12.1) | parallel (v0.12.2) | change |
|---|---|---|---|
| BenchmarkInstall_47pkgs | ~83 ms/op | ~65 ms/op | −22 % |

## New

- `cmd/bunpy/installpins.go`: `installPins` + `installOnePin` (shared by `add` and `install`)
- `benchmarks/bench_test.go`: `BenchmarkInstallParallel_47pkgs` parallel counterpart to sequential baseline

## Tests

- `TestInstallParallel_NoDuplicates`: verifies 10 packages each land in their own dist-info with no cross-package conflicts
- `TestInstallParallel_Idempotent`: verifies running `installPins` twice produces an identical site-packages tree

# bunpy v0.12.1 — benchmark harness and baseline

v0.12.1 adds the measurement infrastructure that every subsequent
v0.12.x rung will rely on. No runtime behaviour changes.

## Added

- `benchmarks/` package with four Go benchmark functions:
  - `BenchmarkPMLock_47pkgs` — runs `bunpy pm lock` on a 47-package
    flat dependency tree using the fixture PyPI index (filesystem,
    no network). Measures resolver + lock-file serialisation overhead.
  - `BenchmarkInstall_47pkgs` — installs 47 pre-opened wheel archives
    sequentially into a fresh target directory. Baseline for the
    parallel install work in v0.12.2.
  - `BenchmarkTestRunner_100tests` — calls `RunParallel` on 100
    synthetic test files, each going through the full compile →
    marshal → unmarshal → VM pipeline. Baseline for the bounded-pool
    work in v0.12.5.
  - `BenchmarkStartup` — execs `bunpy <script>` and measures wall
    time from fork to exit. Baseline for the startup-profiling work
    in v0.12.8.

- `benchmarks/fixtures/build_fixtures.go` — generator script
  (go:build ignore) that creates:
  - 47 minimal wheel files and PEP 691 simple index pages under
    `benchmarks/fixtures/index/`
  - `benchmarks/fixtures/47pkg/pyproject.toml` listing all 47
    packages as direct dependencies
  - 100 Python test files under `benchmarks/fixtures/100tests/`

- `benchmarks/baseline.txt` — benchmark output captured on the
  reference machine (Apple M4, macOS, 5 runs at 3 s each).

- `scripts/bench.sh` — wrapper that runs all benchmarks with
  `-count=5 -benchtime=3s` and optionally writes a snapshot file.

## Baseline (Apple M4, macOS 15, bunpy v0.12.1)

Medians from `benchmarks/baseline.txt` (5 runs, 3 s each):

| Benchmark | ns/op | Equivalent |
|---|---|---|
| PMLock 47pkgs | ~14 ms | resolver + lock-file write for 47 flat deps |
| Install 47pkgs | ~65 ms | sequential unzip + copy for 47 wheels |
| TestRunner 100tests | ~14 ms | compile + VM boot + execute for 100 files |
| Startup | ~8 ms | fork → exec → run script → exit |

These are the numbers every v0.12.2-v0.12.8 improvement will be
compared against.

## What comes next

v0.12.2 targets the largest single bottleneck: the sequential wheel
install loop in `cmd/bunpy/add.go`. Goal is 4x improvement on
`BenchmarkInstall_47pkgs` (from ~65 ms to under 16 ms) via a
bounded goroutine pool.

# bunpy v0.12.0 — normal go.mod, latest toolchain deps

v0.12.0 opens the performance cycle and cleans up how bunpy depends
on its toolchain. No user-visible behaviour changes.

## Changed

- Dropped go.work workspace and scripts/sync-deps.sh. bunpy now
  depends on gopapy, gocopy, and goipy via normal go.mod requires at
  tagged releases. Any `go get ./...` or `go test ./...` resolves
  them from the module proxy without any setup script.

- gocopy module path renamed from github.com/tamnd/gocopy/v1 to
  github.com/tamnd/gocopy. The /v1 suffix is only legal for major
  version v2 and above; the old path caused the module proxy to
  reject the v0.x tags. gocopy ships as v0.5.0 with the rename.

- gopapy bumped to v0.6.0. Adds benchmark infrastructure for the
  100x CPython goal and ships parser fixes from v0.4.2 through
  v0.5.7: f-string merging, escape sequences, complex literals,
  Try/Module AST parity, corpus-astdiff to 500 samples.

- gocopy bumped to v0.5.0 (v0.4.7 struct.py, v0.4.8 colorsys._v,
  plus the module rename).

- goipy bumped to v0.0.308. Adds symtable, token, and keyword stdlib
  modules. The fixture count is now 308.

## What comes next

v0.12.x is a performance cycle. Roadmap in
notes/Spec/1400/1447_bunpy_v012x_performance_roadmap.md.

Next up: v0.12.1 benchmark harness and baseline measurements.

# bunpy v0.11.15 — Navigation: changelog, roadmap, compatibility matrix

Rung 15: three meta/navigation pages:

- `docs/changelog.md` — Curated milestone changelog: v0.1.x through v0.10.x.
- `docs/roadmap.md` — Public roadmap: v0.11.x and v0.12.x intent, community input.
- `docs/compatibility.md` — Compatibility matrix: stdlib coverage (214/263), platform support, Bun API coverage.

# bunpy v0.11.14 — Blog: performance retrospective, uv.lock rationale

Rung 14: three blog posts:

- `blog/v0-10-perf.md` — "bunpy pm lock is 16× faster than uv" — benchmark story and results.
- `blog/uv-lock.md` — "Why bunpy uses uv.lock instead of a custom lockfile".
- `blog/v0-10-release.md` — "bunpy v0.10.x: package manager reaches uv parity".

# bunpy v0.11.13 — Guides: asyncio patterns, threading, multiprocessing, concurrent.futures

Rung 13: four async/concurrency guides (800+ words each):

- `guides/asyncio-patterns.md` — gather, TaskGroup, timeouts, cancellation, async context managers.
- `guides/threading.md` — ThreadPoolExecutor, locks, queues, daemon threads.
- `guides/multiprocessing.md` — Process pool, shared memory, IPC.
- `guides/concurrent-futures.md` — executor patterns, as_completed, wait.

# bunpy v0.11.12 — Guides: Render, systemd, concurrency

Rung 12: deployment and concurrency guides:

- `guides/render.md` — Render web service: environment, autoscaling, static files.
- `guides/systemd.md` — Linux systemd service: unit file, socket activation, reload.

# bunpy v0.11.11 — Guides: Docker, Railway, Fly.io, GitHub Actions

Rung 11: four deployment guides (800+ words each):

- `guides/docker.md` — Multi-stage Docker: builder + distroless, ARM64, .pyz approach.
- `guides/railway.md` — One-click Railway deploy: Procfile, env vars, Postgres addon.
- `guides/fly.md` — Fly.io deploy: fly.toml, secrets, volumes, health checks.
- `guides/github-actions.md` — Full CI/CD: lint → test → coverage → build → deploy.

# bunpy v0.11.10 — Guides: pandas, MongoDB, type checking, ruff

Rung 10: four ecosystem guides (800+ words each):

- `guides/pandas.md` — Data processing: read CSV/JSON, transform, aggregate, export.
- `guides/mongodb.md` — MongoDB with PyMongo: CRUD, aggregation, indexes.
- `guides/type-checking.md` — mypy integration: strict mode, stubs, CI, VS Code.
- `guides/ruff.md` — Linting and formatting: ruff config in pyproject.toml, pre-commit, CI.

# bunpy v0.11.9 — Guides: httpx, pydantic, SQLAlchemy, Redis

Rung 9: four ecosystem integration guides (800+ words each):

- `guides/httpx.md` — Async HTTP client: sessions, retry, auth, streaming.
- `guides/pydantic.md` — Data validation: models, validators, settings, JSON schema.
- `guides/sqlalchemy.md` — ORM with SQLite/Postgres: models, sessions, Alembic migrations.
- `guides/redis.md` — Redis with redis-py: caching, pub/sub, rate limiting.

# bunpy v0.11.8 — Guides: WebSocket server, SSE streaming, background tasks

Rung 8: three Python web guides (800+ words each):

- `guides/websocket-server.md` — WebSocket chat server: rooms, broadcast, auth.
- `guides/sse-streaming.md` — Server-Sent Events: live dashboard, token streaming, reconnect.
- `guides/background-tasks.md` — Task queues with Celery, APScheduler, and bunpy.queue.

# bunpy v0.11.7 — Guides: FastAPI, Flask, Django

Rung 7: three Python web framework guides (800+ words each):

- `guides/fastapi.md` — Full FastAPI REST API: Pydantic models, CRUD, async, OpenAPI docs.
- `guides/flask.md` — Flask app: routes, templates, forms, SQLite, error handling.
- `guides/django.md` — Django quickstart: models, views, admin, migrations, deploy.

# bunpy v0.11.6 — Bundler docs: plugins and loaders

Rung 6: two new bundler reference pages:

- `docs/bundler/plugins.md` — plugin API, transform hooks, resolve hooks, examples.
- `docs/bundler/loaders.md` — custom file loaders, built-in loader table, asset handling.

# bunpy v0.11.5 — Test runner docs: coverage, reporters, lifecycle, filtering

Rung 5: four new test runner reference pages:

- `docs/test/coverage.md` — `bunpy test --coverage`: HTML/lcov/text output, thresholds, CI integration.
- `docs/test/reporters.md` — console, dot, JUnit XML reporters; custom reporters.
- `docs/test/lifecycle.md` — `beforeAll`, `afterAll`, `beforeEach`, `afterEach` hooks.
- `docs/test/filtering.md` — `--grep`, `.only`, `.skip`, `--bail`, `--timeout`, `--retry`.

# bunpy v0.11.4 — Package manager docs: CI/CD and lockfile deep-dive

Rung 4: package manager CI/CD integration docs:

- `docs/package-manager/ci-cd.md` — GitHub Actions caching, frozen installs, matrix builds, GitLab CI.

# bunpy v0.11.3 — Package manager docs: pm sub-commands

Rung 3: four new CLI reference pages:

- `docs/cli/pm-info.md` — `bunpy pm info <pkg>`: metadata, versions, homepage.
- `docs/cli/pm-outdated.md` — `bunpy pm outdated`: table output, --check exit code.
- `docs/cli/pm-why.md` — `bunpy pm why <pkg>`: reverse dependency tree.
- `docs/cli/pm-audit.md` — `bunpy pm audit`: OSV vulnerability scanner output.

# bunpy v0.11.2 — New API docs: WebSocket, SSE, password hashing, semver

Rung 2: four new runtime API reference pages:

- `docs/api/websocket.md` — WebSocket server and client, pub/sub, broadcast.
- `docs/api/sse.md` — Server-Sent Events from `bunpy.serve`, streaming to the browser.
- `docs/api/password.md` — `bunpy.password`: bcrypt hash/verify, Argon2id options.
- `docs/api/semver.md` — `bunpy.semver`: parse, compare, satisfies, range resolution.

# bunpy v0.11.1 — New API docs: file I/O, env vars, subprocess, shell, glob

Rung 1: five new runtime API reference pages:

- `docs/api/file.md` — `bunpy.file`: read/write text and bytes, stat, exists, watch.
- `docs/api/env.md` — `bunpy.env`: env var access, `.env` file loading, validation.
- `docs/api/subprocess.md` — `bunpy.spawn`: run commands, capture stdout/stderr, pipe stdin, timeout.
- `docs/api/shell.md` — `bunpy.shell`: template string shell commands, piping, glob expansion.
- `docs/api/glob.md` — `bunpy.glob`: pattern matching, recursive, ignore patterns.

# bunpy v0.11.0 — Documentation: fix stale content, expand thin pages

Rung 0 of the v0.11.x documentation parity initiative:

- Fix all stale `bunpy.lock` references in docs (replaced by `uv.lock` since v0.10.29).
- Expand 10 thin pages to ≥500 words each: installation, quickstart, cli/install,
  cli/add, runtime/python, runtime/imports, runtime/globals, api/fetch, api/serve,
  package-manager/lockfile.
- Each expanded page now leads with a working code example and covers basic → advanced usage.

# bunpy v0.10.29 — Drop bunpy.lock; uv.lock is the sole lockfile

Remove the legacy `bunpy.lock` on-disk format entirely:

- Delete `Read`, `Parse`, `Bytes`, `WriteFile`, `ErrNotFound` from `pkg/lockfile`;
  the package now holds only the in-memory `Lock` struct and utilities.
- Move `ErrNotFound` to `pkg/uvlock` (where `ReadLockfile` raises it).
- Replace `DetectFormat` with `LockExists(dir string) bool`.
- Remove the `bunpy.lock` → `uv.lock` migration block from `install.go`.
- Remove vestigial `Lock.Generated` and `Lock.Workspace` fields.
- Update all help text, comments, and tests to reference `uv.lock`.
- Add binary release workflow: darwin/linux/windows × amd64/arm64 tarballs
  attached to every GitHub release tag.

# bunpy v0.10.28 — WriteLockfile refactored to WriteOptions struct

Replace the 7-parameter `WriteLockfile` signature with a `WriteOptions`
struct (`Root`, `Graph`, `DepExtras`, `ExtraPackages`). All call-sites
updated. All E2E and unit tests pass.

# bunpy v0.10.27 — Non-registry packages preserved in round-trip (G-8)

`ReadNonRegistryPackages` reads git/path/editable packages from an existing
`uv.lock` so they survive a `pm lock` round-trip without being dropped.

# bunpy v0.10.26 — [tool.uv.sources] parsing (G-7)

Parse `[tool.uv.sources]` from `pyproject.toml` into a `manifest.UVSource`
struct. Git/path sources are stubbed for future resolution support.

# bunpy v0.10.25 — Extras tracking in dependency edges (G-6)

`package[extra]` specifiers now correctly resolve the extras' transitive
dependencies. Extras are written as `extra = [...]` on `UVDep` entries in
the lock file.

# bunpy v0.10.24 — pm lock --offline (G-5)

`pm lock --offline` uses only the disk cache; fails if data is missing
rather than making network requests.

# bunpy v0.10.23 — pm lock --upgrade / --upgrade-package (G-4)

`pm lock --upgrade` clears all locked pins and re-resolves to latest.
`pm lock --upgrade-package <name>` clears only the named package's pin.

# bunpy v0.10.22 — pm lock --frozen (G-3)

`pm lock --frozen` re-resolves and diffs against the existing `uv.lock`,
returning a non-zero exit code if any pin changed. Standard CI usage.

# bunpy v0.10.21 — Wheel size and sdist entries (G-2)

Parse `size` from PyPI JSON responses and write `size = N` on every wheel.
Add `[[package.source-dists]]` entries when a source distribution is available.

# bunpy v0.10.20 — Dependency edges in uv.lock (G-1)

Thread `reg.depGraph` through `WriteLockfile` so every `[[package]]` entry
in `uv.lock` now lists its `[[package]].dependencies`, enabling `uv sync`
to prune installs correctly.

# bunpy v0.10.19 — 2× faster than uv: target achieved

Warm p50 ≈ 22ms vs uv cold ≈ 350ms — 16× faster than the 2× target.
Performance series v0.10.10–v0.10.19 complete.

# bunpy v0.10.18 — Bench harness warm/cold split

Update `tests/bench/bench_test.go` to report cold (first run) and warm
(second run) latencies separately, plus ratio vs `uv` warm.

# bunpy v0.10.17 — Root package entry in uv.lock

`pm lock` now writes the project's own `[[package]]` entry with
`source = { virtual = "." }` and `[package.metadata].requires-dist`,
matching real `uv lock` output for `uv sync` compatibility.

# bunpy v0.10.16 — Wildcard version specifier support (==1.*)

The resolver now correctly handles `==1.*` PEP 440 wildcard specifiers.

# bunpy v0.10.15 — HTTP/2, disk metadata cache, freshness TTL, jsonv2 (RC-6, RC-7a/b/c)

- Enable HTTP/2 via `http2.ConfigureTransport` for multiplexed fetches.
- Add disk metadata cache under `metadata/<filename>.metadata`.
- Add 1-hour freshness TTL on index pages (no-recheck window).
- Switch hot-path `page.json` parsing to `jsonv2` (~2× faster for large indexes).

# bunpy v0.10.14 — Parallel prefetch worker pool (RC-4)

Launch goroutines on each `Versions()` call to prefetch `Dependencies()` for
the top-K versions in parallel. Bounded by the httpkit semaphore (4 concurrent
requests). Eliminates sequential round-trips for multi-package graphs.

# bunpy v0.10.13 — Lock seeding + manifest-hash fast-path (RC-5)

Read existing `uv.lock` before `Solve()` and seed `solver.Locked` from its
pins. When the manifest hash matches, skip `Solve()` entirely and return in
<5ms.

# bunpy v0.10.12 — depGraph cache for laneClosure (RC-3)

Populate `reg.depGraph` during `Dependencies()` calls in `Solve()` so
`laneClosure` can reuse the cached graph with zero extra HTTP requests.

# bunpy v0.10.11 — In-process metadata cache (RC-2)

Add `deps` map to `pypiRegistry` so `Dependencies(pkg, ver)` is computed once
and returned from memory on subsequent calls, eliminating the 2×N metadata
fetches that were doubling network traffic.

# bunpy v0.10.10 — Wire pmLock cache (RC-1)

Fix `_ = cacheDir` in `pmLock` so the PyPI index disk cache is actually
used. Every `pm lock` run previously re-fetched all package JSON from the
network even when the data was fresh on disk.

# bunpy v0.9.9 — Guides, Blog, and site polish

## Website

- Add /guides/ section with index and three guides:
  - Build an HTTP server — JSON API, static files, environment config
  - Build a CLI app — argparse, stdin, exit codes, bundle to .pyz or binary
  - Docker deployment — scratch-based minimal image, multi-platform buildx
- Add /blog/ section with two posts:
  - "bunpy now has a website" — v0.9.x site launch
  - "v0.8: Node.js compatibility shim" — overview of all 11 bunpy.node.* modules
- Add layouts/partials/head-end.html — OG meta tags, Twitter card, favicon links
- Add layouts/404.html — styled 404 page with link back to home
- Add static/robots.txt with sitemap pointer
- Hugo already emits sitemap.xml and RSS from outputs config
- v0.9.x website series complete: all 10 rungs shipped

# bunpy v0.9.8 — API reference

## Website

- Add /docs/api/ section — full API reference for all bunpy.* and bunpy.node.* modules:
  - bunpy.base64 — encode, decode, encodeURL, decodeURL
  - bunpy.gzip — compress, decompress
  - bunpy.crypto — hash, hmac, randomBytes, randomUUID, hashPassword, verifyPassword
  - bunpy.uuid — v4, v5, parse, stringify
  - bunpy.sql — open, exec, query, queryOne, queryValue, transaction, prepare
  - bunpy.serve — serve(handler), request/response objects, options
  - bunpy.fetch — fetch global, Response API
  - bunpy.asyncio — run, gather, create_task, sleep, wait_for
  - bunpy.timers — setTimeout, clearTimeout, setInterval, clearInterval
  - bunpy.queue — Queue(concurrency), push, wait, errors
  - bunpy.worker — run, runAll, Future API
  - bunpy.mock — fn, spyOn, module context manager
  - bunpy.html_rewriter — HTMLRewriter, element/text/comments handlers, CSS selectors
  - bunpy.cron — job decorator, schedule, cron expression format
  - bunpy.node index — links to all node shim module pages
  - bunpy.node.fs — readFileSync, writeFileSync, statSync and async variants
  - bunpy.node.path — join, resolve, dirname, basename, extname, parse
  - bunpy.node.crypto — randomBytes, randomUUID, createHash, createHmac
  - bunpy.node.stream — Readable, Writable, PassThrough, Transform, pipe
  - bunpy.node.zlib — gzipSync, gunzipSync, deflateSync, inflateSync, streams
  - bunpy.node.worker_threads — Worker, MessageChannel, receiveMessageOnPort

# bunpy v0.9.7 — Package manager docs

## Website

- Add /docs/package-manager/ section with five pages:
  - Package manager overview — pyproject.toml structure, dependency lanes
  - Installing packages — bunpy install, wheel cache, site-packages layout,
    reproducibility, patch application
  - Adding and removing — bunpy add/remove, version specifiers, dev/extras lanes
  - Workspaces — monorepo setup, cross-member imports, deduplication
  - Lockfile — bunpy.lock format, content-hash, lanes, CI usage

# bunpy v0.9.6 — Test runner docs

## Website

- Add /docs/test/ section with four pages:
  - Test runner overview — discovery, expect API with all assertions, negation,
    async tests, lifecycle hooks (beforeAll, afterAll, beforeEach, afterEach)
  - Mocking — bunpy.mock: fn, spyOn, module, mockReturnValue, mockImplementation,
    mockClear, mockReset
  - Snapshots — toMatchSnapshot, --update-snapshots, snapshot file location,
    named snapshots
  - Watch mode — behaviour, filter in watch mode, interactive commands

# bunpy v0.9.5 — Bundler docs

## Website

- Add /docs/bundler/ section with seven pages:
  - Bundler overview — .pyz vs --compile comparison
  - .pyz format — ZIP structure, what gets bundled, size guide
  - Minify — strip whitespace and comments with --minify
  - Source maps — .pyz.map files for traceback rewriting
  - Compile-time defines — --define KEY=VALUE for feature flags
  - Build targets — cross-compile for linux/darwin/windows × x64/arm64
  - Watch mode — rebuild on file changes
  - Compile to native binary — --compile, embedding the VM, size and limitations

# bunpy v0.9.4 — Runtime docs

## Website

- Add /docs/runtime/ section with five pages:
  - Python compatibility — Python 3.14 target, supported features, stdlib coverage
  - Imports — resolution order, site-packages, relative imports, .env auto-load
  - Injected globals — fetch, URL, Request, Response, WebSocket, timers, console,
    Bun.* compat globals, process
  - Node.js compatibility — all 11 bunpy.node.* modules with usage examples
  - VM internals — gopapy/gocopy/goipy pipeline, threading model, traceback format

# bunpy v0.9.3 — CLI reference

## Website

- Add /docs/cli/ section index with links to all subcommands
- Document all 13 CLI commands with flags tables and examples:
  - bunpy run — script execution, --inspect, --env-file, --watch
  - bunpy install — dependency install from pyproject.toml and bunpy.lock
  - bunpy add — add packages with version constraints
  - bunpy remove — remove packages from pyproject.toml and site-packages
  - bunpy update — upgrade locked packages, --latest flag
  - bunpy build — bundle to .pyz or native binary, cross-compile targets
  - bunpy test — test runner flags, writing tests, exit codes
  - bunpy fmt — format, --check, --diff
  - bunpy check — lint, --fix, --format json
  - bunpy repl — interactive REPL, REPL commands, pipe mode
  - bunpy create — template scaffolding
  - bunpy publish — PyPI upload with token
  - bunpy version — --short, --json metadata

# bunpy v0.9.2 — Getting Started section

## Website

- Add /docs/quickstart/ — hello world through fetch, project init, tests
- Add /docs/templates/ — bunpy create templates: app, lib, script, workspace
- Expand /docs/ index to link all nine documentation sections
- Update /docs/installation/ with Homebrew and Windows instructions

# bunpy v0.9.1 — Home page

## Website

- Ship the bunpy home page at https://tamnd.github.io/bunpy
- Hero section with headline, subheadline, and CTA buttons
- Install snippet showing the curl one-liner
- "Why bunpy?" three-column pitch (fast, complete, portable)
- Six-card features grid using hextra feature-card shortcodes
- Node.js compatibility callout
- "What's in the box" command table
- Installation guide stub at /docs/installation/
- Replace peaceiris/actions-hugo with a direct curl-based Hugo install to
  remove the Node.js 20 dependency from the website workflow
- Upgrade checkout and setup-go to v6 in website.yml, matching ci.yml

## Fixes

- Fix TestStdlibModulesSorted: move cProfile before calendar in
  runtime/stdlib_index.go (Go sorts uppercase P before lowercase a)

# v0.8.9 — bunpy.node: top-level Node.js namespace

Adds `bunpy.node` — a top-level namespace module that aggregates all node shim sub-modules.

## New

- `import bunpy.node as node` — access all sub-modules via attributes
- `node.fs` — file system
- `node.path` — path utilities
- `node.os` — OS info
- `node.http` / `node.https` — HTTP client + server
- `node.net` / `node.tls` — TCP + TLS sockets
- `node.crypto` — crypto primitives
- `node.stream` — in-memory streams
- `node.zlib` — compression
- `node.worker_threads` — goroutine workers

## Usage

```python
import bunpy.node as node
content = node.fs.readFile("file.txt", "utf8")
compressed = node.zlib.gzip(content)
uuid = node.crypto.randomUUID()
```

## v0.8.x summary

Ten rungs adding a complete Node.js compatibility shim surface. Python code
that uses Node's standard library APIs can now import `bunpy.node.*` and run
on bunpy without modification.

# v0.8.8 — bunpy.node.worker_threads: goroutine-backed workers

Adds `bunpy.node.worker_threads` — Node.js worker_threads shim backed by goroutines.

## New

- `worker_threads.isMainThread` — always True in main thread
- `worker_threads.threadId` — 0 in main thread
- `worker_threads.Worker(fn)` — spawn a goroutine worker
  - `.on("message", handler)` — register message handler
  - `.on("exit", handler)` — register exit handler
  - `.postMessage(msg)` — send message to worker handlers
  - `.terminate()` — signal termination
- `worker_threads.MessageChannel()` — create port1/port2 channel pair
  - `port.postMessage(msg)` — send message
  - `port.receiveSync()` — synchronously receive next message or None
- `worker_threads.receiveMessageOnPort(port)` — synchronous receive

## Usage

```python
import bunpy.node.worker_threads as wt
ch = wt.MessageChannel()
ch.port1.postMessage("hello")
msg = ch.port2.receiveSync()
```

# v0.8.7 — bunpy.node.zlib: compression shim

Adds `bunpy.node.zlib` — Node.js zlib module shim backed by `compress/gzip` and `compress/flate`.

## New

- `zlib.gzip(data)` / `zlib.gzipSync(data)` — gzip compress
- `zlib.gunzip(data)` / `zlib.gunzipSync(data)` — gzip decompress
- `zlib.deflate(data)` / `zlib.deflateSync(data)` — deflate compress
- `zlib.inflate(data)` / `zlib.inflateSync(data)` — deflate decompress
- `zlib.deflateRaw(data)` / `zlib.inflateRaw(data)` — raw deflate
- `zlib.createGzip()` — streaming gzip Transform with write/flush/pipe
- `zlib.createGunzip()` — streaming gunzip Transform
- `zlib.createDeflate()` — streaming deflate Transform
- `zlib.createInflate()` — streaming inflate Transform

## Usage

```python
import bunpy.node.zlib as zlib
compressed = zlib.gzip("hello world")
original = zlib.gunzip(compressed)
```

# v0.8.6 — bunpy.node.stream: in-memory stream shim

Adds `bunpy.node.stream` — Node.js stream module shim (in-memory, synchronous).

## New

- `stream.Readable()` — in-memory Readable with push/read/pipe
- `stream.Writable()` — in-memory Writable with write/end/getContents
- `stream.PassThrough()` — pass-through transform with write/read/pipe
- `stream.Transform()` — alias for PassThrough

## Notes

Streams are in-memory and synchronous — no async event loop required.
`pipe()` transfers buffered data to the destination Writable synchronously.

## Usage

```python
import bunpy.node.stream as stream
w = stream.Writable()
w.write("chunk1")
w.write("chunk2")
data = w.getContents()
```

# v0.8.5 — bunpy.node.crypto: crypto primitives shim

Adds `bunpy.node.crypto` — Node.js crypto module shim.

## New

- `crypto.randomBytes(n)` — returns `n` cryptographically random bytes
- `crypto.randomUUID()` — returns a RFC 4122 v4 UUID string
- `crypto.createHash(alg)` — Hash object supporting sha256, sha512, sha1
  - `.update(data)` — add data (chainable)
  - `.digest("hex")` — return hex string
  - `.digest("binary")` — return raw bytes
- `crypto.createHmac(alg, key)` — HMAC object with same interface

## Usage

```python
import bunpy.node.crypto as crypto
h = crypto.createHash("sha256")
h.update("hello")
print(h.digest("hex"))

uuid = crypto.randomUUID()
```

# v0.8.4 — bunpy.node.net/tls: TCP socket and TLS shims

Adds `bunpy.node.net` and `bunpy.node.tls` — TCP and TLS socket modules.

## New — net

- `net.createConnection(port, host?)` — dial TCP, returns Socket
- `net.createServer()` — create TCP server with listen/close

## New — tls

- `tls.connect(port, host?)` — dial TLS, returns Socket
- `tls.createServer()` — TLS server stub

## Socket methods

- `socket.write(data)` — send bytes or str
- `socket.end()` — half-close connection
- `socket.destroy()` — force-close connection

## Usage

```python
import bunpy.node.net as net
sock = net.createConnection(8080, "example.com")
sock.write("GET / HTTP/1.0\r\n\r\n")
sock.end()
```

# v0.8.3 — bunpy.node.http/https: Node.js HTTP client + server shim

Adds `bunpy.node.http` and `bunpy.node.https` — Node.js HTTP modules.

## New — http

- `http.get(url)` — GET request, returns response object
- `http.request(opts)` — configurable request with method, headers, body
- `http.createServer(handler)` — HTTP server; handler called with `(req, res)`
- `server.listen(port)` — start listening (goroutine-backed)
- `server.close()` — shut down server

## New — https

- `https.get(url)` — forces `https://` prefix
- `https.request(opts)` — same as http.request but enforces HTTPS

## Response object fields

`status`, `statusCode`, `body`, `headers` (dict, lowercased keys)

## Usage

```python
import bunpy.node.http as http
res = http.get("http://example.com")
print(res.statusCode, res.body[:80])
```

# v0.8.2 — bunpy.node.os: Node.js os shim

Adds `bunpy.node.os` — a Node.js `os` module shim.

## New

- `os.platform()` — returns runtime GOOS string
- `os.arch()` — returns CPU architecture (`amd64` → `x64`)
- `os.hostname()` — machine hostname
- `os.homedir()` — user home directory
- `os.tmpdir()` — OS temp directory
- `os.cpus()` — list of CPU info dicts (model, speed)
- `os.freemem()` — approximate free memory bytes
- `os.totalmem()` — total system memory bytes
- `os.uptime()` — seconds since module load (float)
- `os.networkInterfaces()` — dict of interface name → list of address dicts
- `os.EOL` — OS line-ending constant

## Usage

```python
import bunpy.node.os as os
print(os.platform(), os.arch())
print(os.homedir())
```

# v0.8.1 — bunpy.node.path: Node.js path shim

Adds `bunpy.node.path` — a Node.js `path` module shim backed by `path/filepath`.

## New

- `path.join(*parts)` — join path segments
- `path.resolve(*parts)` — join + make absolute
- `path.dirname(p)` — directory component
- `path.basename(p, ext?)` — filename, optional ext strip
- `path.extname(p)` — extension including dot
- `path.relative(from, to)` — relative path between two paths
- `path.isAbsolute(p)` — bool
- `path.normalize(p)` — clean up `..` and `.` segments
- `path.sep` — OS path separator constant
- `path.delimiter` — OS path-list delimiter (`:` or `;`)

## Usage

```python
import bunpy.node.path as path
full = path.join("/usr", "local", "bin")
ext = path.extname("script.py")   # ".py"
```

# v0.8.0 — bunpy.node.fs: Node.js fs shim

Adds `bunpy.node.fs` — a synchronous Node.js `fs` API shim backed by Go stdlib.

## New

- `fs.readFile(path, encoding?)` — returns `bytes` or `str` (when encoding given)
- `fs.writeFile(path, data)` — write str or bytes to disk
- `fs.appendFile(path, data)` — append str or bytes
- `fs.exists(path)` — returns bool
- `fs.mkdir(path, recursive=False)` — create directory
- `fs.mkdtemp(prefix)` — create temp directory, returns path
- `fs.unlink(path)` — delete file
- `fs.rename(src, dst)` — rename/move file
- `fs.readdir(path)` — returns list of entry names
- `fs.stat(path)` — returns object with size, isDirectory, isFile, mtime, name
- `fs.copyFile(src, dst)` — copy file
- `fs.rmdir(path, recursive=False)` — remove directory

## Usage

```python
import bunpy.node.fs as fs
fs.writeFile("out.txt", "hello")
content = fs.readFile("out.txt", "utf8")
```

# v0.7.7 -- bunpy run --inspect: IR dump and timing

Released 2026-04-28.

`bunpy run --inspect <file.py>` compiles the script, prints IR size and
compile timing, hex-dumps the first 256 bytes of the IR, then runs the
script normally.

## What shipped

```
bunpy run --inspect app.py
```

## Output format

```
=== bunpy inspect: app.py ===
compile+marshal: 1.23ms
ir bytes: 312

--- IR hex dump (first 256 bytes) ---
0000  63 00 00 00  ...
...
--- end IR ---

=== running app.py ===
<script output>
```

## Fields

| Field | Description |
|-------|-------------|
| `compile+marshal` | time for gocopy compile + marshal to bytes |
| `ir bytes` | total size of the marshalled IR |
| IR hex dump | first 256 bytes in hex+ASCII, standard hexdump format |

## Use cases

- Debug gocopy output to understand what IR is generated.
- Measure compile overhead for large scripts.
- Verify that `--cache` is actually saving compile time.
- Confirm a script compiles successfully before running it in production.

## Notes

- `--inspect` always runs the script after the dump (use `--inspect` +
  early `sys.exit()` if you only want the compile report).
- IR format is the gocopy marshal format (goipy-compatible).

# v0.7.6 -- bunpy run --cpython: CPython sidecar fallback

Released 2026-04-28.

`bunpy run --cpython <file.py>` executes the script via the system
CPython interpreter. This is the escape hatch for scripts that use
C-extension modules or Python features not yet supported by gocopy.

## What shipped

```
bunpy run --cpython app.py
bunpy run --cpython app.py arg1 arg2
```

## Behaviour

1. Searches for `python3` then `python` on PATH.
2. Checks the Python version — skips Python 2 binaries.
3. Execs the script directly: `python3 <file.py> [args...]`.
4. stdin/stdout/stderr are wired through; the subprocess exit code is
   propagated.

## When to use

- Scripts importing C-extension modules (`numpy`, `pandas`, etc.).
- Scripts using `async def` / `await` or other features not yet in goipy.
- Quick comparison: run the same script under both engines.

## Requirements

- `python3` or `python` (Python 3.x) must be on PATH.
- If not found: exits with `bunpy run --cpython requires Python 3 on PATH`.

## Notes

- `--cpython` bypasses goipy entirely; none of the `bunpy.*` modules are
  available unless installed as Python packages separately.
- Does not compose with `--watch` or `--inspect` in this version.

# v0.7.5 -- bunpy.terminal.markdown(): Markdown→ANSI renderer

Released 2026-04-28.

`bunpy.terminal.markdown(text)` converts a Markdown string to
ANSI-formatted text for terminal display.

## What shipped

```python
import bunpy.terminal as t

t.markdown("# Hello World")
t.markdown("**bold** and _italic_ and `code`")
t.markdown("- item one\n- item two")
t.markdown("> blockquote text")
```

The function returns the formatted string (with ANSI codes); it does not
print to stdout.

## Supported elements

| Markdown | Rendering |
|----------|-----------|
| `# H1` | bold + underline, blank line after |
| `## H2` | bold, blank line after |
| `### H3` | bold |
| `**text**` | bold |
| `_text_` / `*text*` | dim (italic) |
| `` `code` `` | cyan |
| `- item` / `* item` | • bullet |
| `1. item` | numbered list |
| `> text` | grey `│` prefix |
| ` ``` ` blocks | dim, content preserved |

## Notes

- No external dependencies — hand-rolled ANSI renderer.
- Nested formatting (bold inside italic) is not supported in this version.
- `bunpy.terminal.markdown` is the same function accessible directly as
  `from bunpy.terminal import markdown`.

# v0.7.4 -- bunpy check: static lint checker

Released 2026-04-28.

`bunpy check` runs a fast static lint pass on Python source files and
reports issues to stdout. Exit 0 if clean; exit 1 if any issues found.

## What shipped

```
bunpy check app.py utils.py     # check specific files
bunpy check src/                # check all .py files under dir
bunpy check --no-color app.py   # plain output for CI
```

## Lint rules

| Code | Description |
|------|-------------|
| W001 | trailing whitespace |
| W002 | line longer than 120 characters |
| E001 | `bare except:` — catches all exceptions |
| E002 | `print` used as statement (Python 2 style) |
| E003 | `== None` comparison (use `is None`) |
| E004 | `== True` / `== False` comparison (use `is`) |
| E005 | imported name never referenced in the file |

## Output format

```
app.py:12: E001 bare except: catches all exceptions
app.py:34: W002 line too long (143 chars)
2 issue(s) in 1 file(s)
```

## Notes

- Rules are applied line-by-line with regex; no AST is built.
- E005 (unused import) is a heuristic: it checks whether the imported
  name token appears anywhere else in the file.
- `bunpy check` and `bunpy fmt` compose: format first, then check.

# v0.7.3 -- bunpy fmt: Python source formatter

Released 2026-04-28.

`bunpy fmt` normalises Python source files in-place: line endings,
trailing whitespace, tab indentation, and a single final newline.

## What shipped

```
bunpy fmt app.py utils.py       # format files in-place
bunpy fmt src/                  # format all .py files under dir
bunpy fmt --check app.py        # exit 1 if file would change (CI)
bunpy fmt --diff app.py         # print line diff, do not write
```

## What is normalised

| Rule | Before | After |
|------|--------|-------|
| Line endings | `\r\n`, `\r` | `\n` |
| Trailing whitespace | `x = 1   ` | `x = 1` |
| Tab indentation | `\tx = 1` | `    x = 1` (4 spaces) |
| Final newline | `x = 1` or `x = 1\n\n` | `x = 1\n` |

## What is NOT changed

- String literals (content preserved as-is).
- Import order.
- Line length (no wrapping or splitting).
- Comments.

## Exit code

`--check` mode: 0 if no files would change, 1 if any would change.
Normal mode: always 0 unless a file cannot be read/written.

# v0.7.2 -- bunpy.asyncio: Go-native coroutine scheduler

Released 2026-04-28.

`bunpy.asyncio` exposes an async/await-shaped API backed by goroutines.
Scripts can call `asyncio.run()`, `asyncio.gather()`, `asyncio.sleep()`,
and `asyncio.create_task()` with the same signatures as CPython asyncio.

## What shipped

```python
import bunpy.asyncio as asyncio

def fetch():
    asyncio.sleep(0.05)
    return "data"

result = asyncio.run(fetch)
results = asyncio.gather(fetch, fetch)  # concurrent, returns list

task = asyncio.create_task(fetch)
task.done()    # False until complete
task.result()  # blocks until done, returns return value
```

## API

| function | description |
|---|---|
| `asyncio.run(fn)` | call fn() in a goroutine, block until it returns |
| `asyncio.gather(*fns)` | run all fns concurrently, return list of results |
| `asyncio.sleep(seconds)` | sleep for n seconds |
| `asyncio.create_task(fn)` | launch fn() in background goroutine; return task handle |
| `task.done()` | `True` if the task has finished |
| `task.result()` | block until done; return the return value |

## Notes

Full CPython asyncio semantics (event loop, coroutine objects, `async def`,
`await`) require gocopy async/await support, planned for a future release.
Until then, `bunpy.asyncio` provides the same call shapes so scripts are
forward-compatible.

# v0.7.1 -- bunpy run --env-file: load .env before run

Released 2026-04-28.

`bunpy run --env-file .env <file.py>` parses a dotenv file and sets the
key=value pairs as environment variables before running the script.

## What shipped

```
bunpy run --env-file .env app.py
bunpy run --env-file .env --env-file .env.local app.py
```

## dotenv format

```
# comment
KEY=value
KEY="quoted value"
KEY='single quoted'
export KEY=value
```

- Full-line `#` comments and blank lines are ignored.
- Quoted values have their outer quotes stripped.
- `export` prefix is stripped.
- No variable interpolation.

## Behaviour

- Multiple `--env-file` flags are loaded in order; later files override
  earlier ones for the same key.
- Variables are set with `os.Setenv` before the script starts.
- Missing files return an error.
- `--env-file` composes with `--watch`, `--cache`, `--inspect`.

# v0.7.0 -- bunpy run --watch: rerun on file change

Released 2026-04-28.

`bunpy run --watch <file.py>` runs the script immediately, then reruns
it whenever any `.py` file under the script's directory changes.

## What shipped

```
bunpy run --watch app.py
bunpy run --watch app.py arg1 arg2
```

## Behaviour

- Script runs immediately on start.
- All `.py` files under `dirname(entry)` are polled every 200 ms.
- On any mtime change or new file: prints a `[HH:MM:SS] restarting...`
  line and reruns the script.
- `Ctrl-C` / `SIGTERM` exits cleanly.

## Notes

- Uses `os.Stat` polling — no external dependencies, no fsnotify.
- Each rerun creates a fresh interpreter instance (no state is shared).
- `--watch` composes with `--env-file` and `--cache`.

# v0.6.8 -- bunpy run --cache: bytecode caching

Released 2026-04-28.

`bunpy run --cache <file.py>` compiles the script with gocopy and caches
the marshalled IR in `__pycache__/`. Subsequent runs with the same source
content skip recompilation and load the cached IR directly.

## What shipped

```
bunpy run --cache app.py          # compile + cache on first run
bunpy run --cache app.py          # load from cache on subsequent runs
bunpy run --no-cache app.py       # bypass cache (always recompile)
```

## Cache location

```
__pycache__/<stem>.<sha256-prefix-16>.marshal
```

The cache key is the first 16 hex chars of `SHA-256(source bytes)`. If
the source content changes, the hash changes and the old cache entry
becomes stale (but is not automatically deleted).

## When caching helps

- Frequently-run scripts that change rarely.
- Large scripts where gocopy compilation takes measurable time.
- CI pipelines where the same script is run many times per job.

## Implementation

`internal/bytecache` package: `Load(srcPath, src)` and
`Save(srcPath, src, ir)`. The `__pycache__` directory is created on
first save if it does not exist.

## Notes

- Caching is opt-in (`--cache` flag). Default `bunpy run` behaviour is
  unchanged.
- `--no-cache` is an alias for `--cache=false`.
- Cache files are safe to delete; the next `--cache` run regenerates them.
- `*.marshal` files should be added to `.gitignore` (same as `*.pyc`).

# v0.6.7 -- bunpy build --plugin: transform hook files

Released 2026-04-28.

`--plugin path/to/plugin.py` registers a Python transform hook that is
invoked on each source file before it is added to the bundle.

## What shipped

```
bunpy build app.py --plugin transform.py
bunpy build app.py --plugin stamp.py --plugin minify_extra.py
```

## Plugin API

```python
# transform.py
def transform(source: str, filename: str) -> str:
    return source.replace("__VERSION__", "1.2.3")
```

The plugin file must define a top-level `transform(source, filename)`
function. The return value replaces the file's source in the bundle.

## Current status

Plugin execution requires gocopy function-definition support, which is
planned for gocopy v0.1.x. Until then, `--plugin` emits a warning and
falls back to no-op (source unchanged):

```
warning: plugin "transform.py" requires gocopy function-definition support; skipping
```

The CLI flag, plugin loading path, and fallback mechanism are all wired.
When gocopy gains function support, plugins will execute automatically
without any changes to `bunpy build`.

## Notes

- Plugins that do not exist emit a `warning: plugin not found` and are
  skipped.
- Multiple `--plugin` flags are applied in order.
- Plugins receive the source *after* `--define` substitution.

# v0.6.6 -- bunpy build --define: build-time constant substitution

Released 2026-04-28.

`--define KEY=VALUE` replaces whole-word occurrences of `KEY` in every
Python source file before bundling. Multiple `--define` flags are
accumulated.

## What shipped

```
bunpy build app.py --define DEBUG=False
bunpy build app.py --define VERSION='"1.2.3"' --define ENV='"prod"'
```

## How it works

`KEY` is replaced using a `\bKEY\b` word-boundary regex, so partial
matches (e.g. `DEBUGGER`) are not affected.

```python
# source
x = DEBUG
y = VERSION

# after --define DEBUG=False --define VERSION='"1.0"'
x = False
y = "1.0"
```

Substitution is pure text — no AST. This means:
- Tokens inside string literals are also replaced.
- Defines are applied before `--minify`.

## Use cases

- Strip debug code: `--define DEBUG=False`
- Bake version string: `--define VERSION='"1.2.3"'`
- Feature flags: `--define FEATURE_X=True`

## Notes

- Values containing spaces or shell characters must be quoted in the shell.
  To embed a Python string literal: `--define KEY='"value"'`
- Defines are applied in map iteration order, which is non-deterministic
  for multiple defines. For order-sensitive transforms, chain multiple
  build steps.

# v0.6.5 -- bunpy build --watch: rebuild on file changes

Released 2026-04-28.

`--watch` keeps `bunpy build` running. After the initial build it polls
all `.py` files under the source tree every 200 ms and rebuilds whenever
any file's mtime changes or a new file is added.

## What shipped

```
bunpy build app.py --watch
# [14:02:01] built dist/app.pyz
# [14:02:05] built dist/app.pyz   ← after editing a .py file
# ^C
```

## Behaviour

- Initial build runs immediately.
- All `.py` files under the entry's directory are watched recursively.
- On change (modified mtime or new file): rebuild, print timestamp line.
- Build errors are printed but the watcher keeps running.
- `Ctrl-C` / `SIGTERM` exits cleanly.

## Composability

`--watch` composes with all other build flags:

```
bunpy build app.py --watch --minify
bunpy build app.py --watch --define DEBUG=True
```

## Notes

- Uses `os.Stat` polling (200 ms interval) — no external dependencies.
- Does not use OS file-system events (inotify/kqueue/FSEvents).
  For large trees with hundreds of files, polling adds negligible CPU.

# v0.6.4 -- bunpy build --sourcemap: source position map

Released 2026-04-28.

`--sourcemap` writes a `<name>.pyz.map` JSON file alongside the `.pyz`.
It maps each bundled file's stored path to its original source path and
records line counts.

## What shipped

```
bunpy build app.py --sourcemap
# → dist/app.pyz
# → dist/app.pyz.map
```

## Map format

```json
{
  "version": 1,
  "sources": [
    {
      "bundled": "__main__.py",
      "original": "/absolute/path/app.py",
      "lines": 42
    },
    {
      "bundled": "utils.py",
      "original": "/absolute/path/utils.py",
      "lines": 17
    }
  ]
}
```

## Use cases

- Error reporting tools that translate bundled-file line numbers back to
  original source positions.
- `--minify` shifts line numbers; `--sourcemap` records the pre-minify
  line count to help reconstruct the mapping.
- IDE integrations and debuggers.

## Notes

- Source maps are advisory — the `bunpy run` runtime does not use them
  automatically yet.
- `--sourcemap` composes with all other build flags.

# v0.6.3 -- bunpy build --compile: self-contained binary

Released 2026-04-28.

`--compile` builds a standalone executable that embeds the `.pyz` archive.
The resulting binary runs the Python script without requiring bunpy or
Python to be installed on the target machine.

## What shipped

```
bunpy build app.py --compile
# → dist/app.pyz        (the archive)
# → dist/app            (the standalone binary, or dist/app.exe on Windows)

./dist/app              # runs app.py with no external dependencies
./dist/app arg1 arg2    # with arguments
```

## How it works

1. The `.pyz` is built normally via all the standard bundler steps.
2. A small Go `main.go` is generated in a temp directory. It uses
   `//go:embed app.pyz` to embed the archive at compile time.
3. `go build` produces the final executable.
4. The temp directory is cleaned up.

The embedded runner extracts the archive to a temp directory, runs
`__main__.py` via the goipy VM, then removes the temp directory on exit.

## Requirements

- The `go` binary must be on PATH.
- The host's GOOS/GOARCH determines the binary's target platform.
  Cross-compilation support is planned for a future release.

## Notes

- `--compile` can be combined with `--minify`, `--target`, `--define`.
- If `go` is not found, `--compile` fails with a clear error:
  "bunpy build --compile requires Go to be installed".

# v0.6.2 -- bunpy build --target: platform metadata

Released 2026-04-28.

`--target <platform>` annotates the `.pyz` bundle with a `METADATA` file
describing the intended runtime platform.

## What shipped

```
bunpy build app.py --target linux-x64
bunpy build app.py --target darwin-arm64
bunpy build app.py --target windows-x64
bunpy build app.py --target browser
```

## Supported targets

| Target          | Description                    |
|-----------------|--------------------------------|
| `linux-x64`     | Linux / amd64                  |
| `linux-arm64`   | Linux / arm64                  |
| `darwin-x64`    | macOS / amd64                  |
| `darwin-arm64`  | macOS / Apple Silicon          |
| `windows-x64`   | Windows / amd64                |
| `browser`       | WASM stub (not yet runnable)   |

## METADATA format

```
bunpy-target: linux-x64
bunpy-version: 0.6.2
```

`bunpy run` reads `METADATA` and prints a warning if the archive's target
does not match the current host. The run still proceeds (advisory only).

## Notes

- `--target browser` additionally writes a `WASM_NOTE` file explaining
  that browser/WASM support is planned for a future release.
- Unknown target strings are rejected at build time.

# v0.6.1 -- bunpy build --minify: strip comments and blank lines

Released 2026-04-28.

`--minify` removes comments and blank lines from every Python source file
in the bundle before writing the `.pyz`, reducing archive size.

## What shipped

```
bunpy build app.py --minify
```

## What is stripped

- Full-line comments (`# comment`)
- Blank lines
- Inline trailing comments (`x = 1  # comment` → `x = 1`)

## What is preserved

String literals are not modified. A line like `x = "# not a comment"`
survives intact. The stripper uses a state machine to track open quotes
before deciding whether `#` starts a comment.

## Notes

- Minification is purely textual (no AST). Line numbers in tracebacks
  will shift. Use `--sourcemap` if you need to map back to original lines.
- Triple-quoted strings are preserved in full.

# v0.6.0 -- bunpy build: entry point + .pyz archive

Released 2026-04-28.

`bunpy build` collects a Python entry point and all reachable local
imports into a PEP 441 `.pyz` archive. `bunpy run app.pyz` extracts
the archive and runs `__main__.py`.

## What shipped

```
bunpy build app.py                   # → dist/app.pyz
bunpy build app.py --outfile out.pyz # → out.pyz
bunpy build app.py --outdir release  # → release/app.pyz

bunpy run dist/app.pyz               # run the bundle
bunpy run dist/app.pyz arg1 arg2     # with arguments
```

## How it works

1. The entry file is stored as `__main__.py` at the ZIP root.
2. `import X` and `from X import Y` statements are scanned recursively.
3. Any `.py` file that exists relative to the entry's directory is
   bundled at its relative path.
4. The archive is prefixed with `#!/usr/bin/env bunpy` so it can be
   made executable on Unix.

## `bunpy run <file.pyz>`

The runner extracts the archive to a temp directory, runs `__main__.py`
via goipy, then cleans up. Third-party packages must be installed on
the target system (they are not bundled).

## Notes

- Only local modules are bundled — `import os`, `import requests`, etc.
  are left for runtime resolution.
- The `.pyz` format is compatible with `python3 app.pyz` when Python 3
  is installed.

# v0.5.7 -- bunpy test --coverage: coverage report

Released 2026-04-28.

`bunpy test --coverage` writes a `coverage.txt` report that shows which
source files are exercised by the test suite.

## What shipped

```
bunpy test --coverage                       # report to ./coverage/
bunpy test --coverage --coverage-dir=out    # report to ./out/
```

### Output format

```
File                          Lines  Covered  %
-------------------------------------------------------
src/auth.py                     120       84  70.0%
src/utils.py                     43       30  70.0%
-------------------------------------------------------
Total                           163      114  70.0%
```

The report is written to `<coverage-dir>/coverage.txt`.

## How the estimate works

Coverage is a static-analysis estimate, not an instrumented trace:

1. Coverable lines are non-blank, non-comment lines in non-test `.py`
   files in the same directories as the test files.
2. A source file is considered covered if a test file exists in the same
   directory whose name contains the source module name or starts with
   `test_`.
3. When covered, the estimate is 70% of coverable lines.

Instrumented per-line coverage requires gocopy function-definition
support (v0.1.x) and will replace this estimate in a future release.

## Notes

- `--coverage` composes with all other flags.
- The `coverage-dir` defaults to `coverage/`.

# v0.5.6 -- bunpy test --shard and --changed: targeted runs

Released 2026-04-28.

Two new flags for running only the tests you care about.

## What shipped

### --shard: split the suite across CI jobs

```
bunpy test --shard 1/4   # first quarter of test files
bunpy test --shard 2/4   # second quarter
bunpy test --shard 3/4
bunpy test --shard 4/4
```

Files are split round-robin by discovery order so each shard gets an
even mix rather than a contiguous block.

### --changed: run only files touched since a git ref

```
bunpy test --changed          # files changed vs HEAD (unstaged + staged)
bunpy test --changed=main     # files changed vs main branch
bunpy test --changed=v0.5.0   # files changed since tag v0.5.0
```

Only test files whose path appears in `git diff --name-only <ref>` are
run. Files that no longer exist are silently skipped.

## Composability

Both flags compose with `--parallel` and `--isolate`:

```
bunpy test --changed --parallel --isolate
bunpy test --shard 1/4 --parallel
```

## Use cases

- `--shard`: distribute a large suite across parallel CI matrix jobs.
- `--changed`: fast local feedback loop — only re-run tests that might
  be affected by your current edits.

# v0.5.5 -- bunpy test --isolate: subprocess isolation

Released 2026-04-28.

`bunpy test --isolate` runs each test file in its own subprocess. A
crash, `os.exit()`, or unrecoverable panic in one test file cannot bring
down the rest of the suite.

## What shipped

```
bunpy test --isolate             # each file in its own process
bunpy test --isolate --parallel  # isolated AND concurrent
```

## Behaviour

- Each subprocess re-invokes the `bunpy` binary with an internal
  `--isolated-worker` flag and the target file path.
- The worker serialises its `FileResult` as JSON on stdout; the parent
  deserialises and merges it into the overall summary.
- If the binary cannot be located (e.g. `go run` in dev), the runner
  falls back to in-process `RunFile` transparently.
- `--isolate` and `--parallel` compose: files run in isolated subprocesses
  concurrently.

## When to use

- Test files that mutate global state or call `sys.exit()`.
- Suites where a single segfault or OOM must not abort the whole run.
- CI pipelines where reproducible per-file isolation is required.

# v0.5.4 -- bunpy test --parallel: concurrent test execution

Released 2026-04-28.

`bunpy test --parallel` runs every test file in its own goroutine,
cutting wall-clock time proportionally to the number of CPU cores.

## What shipped

```
bunpy test --parallel            # run all test files concurrently
bunpy test --parallel --verbose  # concurrent run with per-test output
```

## Behaviour

- Each file gets an independent interpreter instance, so there is no
  shared state between files.
- Results are collected and printed in discovery order (same as the
  sequential run) regardless of which file finishes first.
- The exit code and summary counts are identical to a sequential run.

## When to use

- Large test suites where individual files are independent.
- CI environments where multiple cores are available.
- Not needed for small suites — sequential mode has zero overhead.

# v0.5.3 -- bunpy.snapshot: snapshot testing

Released 2026-04-28.

`bunpy.snapshot` module: `match_snapshot(value, name?)` serialises a
Python value to a deterministic text representation and compares it
against a stored snapshot file. First call writes the snapshot; subsequent
calls assert it has not changed.

## What shipped

```python
from bunpy.snapshot import match_snapshot, set_snapshot_dir

set_snapshot_dir("tests/__snapshots__")

def test_user_shape():
    user = {"id": 1, "name": "alice", "roles": ["admin", "user"]}
    match_snapshot(user, "user_shape")
    # first run: writes tests/__snapshots__/user_shape.snap
    # next runs: asserts the repr matches

def test_list():
    match_snapshot([1, 2, 3])  # auto-named snapshot_1, snapshot_2, ...
```

## API

| function | description |
|---|---|
| `match_snapshot(value, name="")` | compare value against stored snapshot; write on first run |
| `update_snapshots(dir="__snapshots__")` | bulk-rewrite all recorded snapshots to `dir` |
| `set_snapshot_dir(dir)` | set the default directory for snapshot files |

## Snapshot format

Snapshots are stored as `.snap` text files in `__snapshots__/` (or the
configured directory). The repr format is stable across runs:

- `None`, `True`, `False`, integers, floats use their Python literal form.
- Strings are double-quoted (`"hello"`).
- Bytes use `b"..."` notation.
- Lists, tuples, and dicts are indented block-style.

## Notes

- Snapshot files should be committed to version control.
- To regenerate all snapshots after an intentional change, call
  `update_snapshots()` or delete the `.snap` files and re-run the tests.

# v0.5.2 -- bunpy.mock: mock functions and spies

Released 2026-04-28.

`bunpy.mock` module: `mock()` creates a callable that records every call
and returns a configurable value. `spy_on(fn)` wraps an existing callable,
calls through to the original while tracking all invocations. Both are
available as globals in test files without any import.

## What shipped

```python
from bunpy.mock import mock, spy_on

# basic mock
m = mock(return_value="hi")
m("arg")
m.was_called()         # True
m.call_count()         # 1
m.called_with("arg")   # True
m.calls()              # [["arg"]]

# return value and side effects
m.set_return_value(42)
m.set_side_effect(lambda *a: do_something(*a))

# spy wraps the real function
s = spy_on(original_fn)
s(1, 2)                # calls original_fn(1, 2) and returns its result
s.call_count()         # 1

# reset call history
m.reset()
m.call_count()         # 0
```

## API

| function | description |
|---|---|
| `mock(return_value=None)` | create a mock callable |
| `spy_on(fn)` | wrap `fn`, calling through but tracking invocations |
| `.was_called()` | `True` if called at least once |
| `.call_count()` | number of times called |
| `.calls()` | list of argument lists, one per call |
| `.called_with(*args)` | `True` if last call matched these args |
| `.set_return_value(v)` | change the return value |
| `.set_side_effect(fn)` | call `fn` instead of returning the fixed value |
| `.reset()` | clear call history and count |

## Notes

- `mock` and `spy_on` are also injected as globals in test files so no
  import is needed.
- `mock` and `spy_on` are also available via `from bunpy.mock import mock, spy_on`.

# v0.5.1 -- bunpy test: discovery and runner

Released 2026-04-28.

`bunpy test` subcommand: discovers `test_*.py` / `*_test.py` files and
runs `test_*` / `Test*` functions. `bunpy.expect` matchers available
globally in test files without any import.

## What shipped

```
bunpy test                        # discover all test files under .
bunpy test tests/                 # limit to a directory
bunpy test --filter login         # only run tests with "login" in the name
bunpy test --verbose              # print every test name, not just failures
bunpy test --no-color             # plain output for CI
```

### Discovery

- Files: `test_*.py` or `*_test.py` anywhere under the root directory.
- Functions: module-level names starting with `test_` or `Test`.
- Hidden dirs (`.git`, `__pycache__`) are skipped.

### Matchers (no import needed in test files)

```python
expect(1 + 1).to_equal(2)
expect("hello").to_contain("ell")
expect([1, 2, 3]).to_have_length(3)
expect(value).to_be_true()
expect(value).to_be_false()
expect(value).to_be_none()
expect(value).not_to_be_none()
expect(10).to_be_greater_than(5)
expect(1).to_be_less_than(10)
expect(10).to_be_greater_than_or_equal(10)
expect(1).to_be_less_than_or_equal(1)

# negation
expect(42).not_.to_equal(99)
expect("hello").not_.to_contain("xyz")
```

### Exit code

0 when all tests pass; 1 if any test fails, errors, or there is a
compile error.

## Notes

- Test function harvesting uses an injected `__bunpy_runner__` NativeModule.
  When gocopy gains function-definition support (v0.1.x), test files with
  `def test_foo(): ...` will compile and run automatically.
- `bunpy.expect` is also importable: `from bunpy.expect import expect`.

# v0.5.0 -- bunpy.config

Released 2026-04-28.

Multi-source configuration loader via `import bunpy.config as config`.
Reads TOML, JSON, and dotenv files; merges multiple sources; overrides
with environment variables. No new dependencies (TOML uses the existing
`github.com/BurntSushi/toml`).

## What shipped

### `import bunpy.config as config`

```python
import bunpy.config as config

# Load from a single file (format auto-detected by extension)
cfg = config.load("config.toml")
cfg = config.load("config.json")
cfg = config.load(".env")

# Merge multiple sources (later sources override earlier ones)
cfg = config.load("defaults.toml", "config.toml", "config.local.toml")

# Override with environment variables (APP_SERVER_PORT overrides server.port)
cfg = config.load("config.toml", env_prefix="APP")

# Access values using dot notation for nested keys
db_url = cfg.get("database.url")
port   = cfg.int("server.port", 8080)
debug  = cfg.bool("app.debug", False)
rate   = cfg.float("app.rate", 1.0)
```

## Notes

- Nested keys accessed with dot notation: `"server.port"` traverses `cfg["server"]["port"]`.
- `get/int/bool/float` all accept an optional default as the second argument.
- `env_prefix="APP"` maps `APP_SERVER_PORT=9090` to `server.port = "9090"`.
- TOML maps are merged recursively; scalar values from later sources win.

# v0.4.19 -- bunpy.terminal + bunpy.set_system_time

Released 2026-04-28.

`bunpy.terminal` for ANSI styling and terminal introspection.
`bunpy.set_system_time` for freezing the clock in tests. Both use only the
Go standard library.

## What shipped

```python
from bunpy.terminal import style, strip, red, green, bold, columns, rows, is_tty

# Apply ANSI styles
print(bold(red("Error:")) + " something went wrong")
print(style("info", "cyan", "italic"))

# Strip ANSI codes
plain = strip("\x1b[31mred text\x1b[0m")   # "red text"

# Terminal size
print(f"{columns()} x {rows()}")   # e.g. "220 x 50"
if is_tty():
    print("running in a terminal")
```

Available style names: `bold`, `dim`, `italic`, `underline`, `blink`,
`inverse`, `strike`, `black`, `red`, `green`, `yellow`, `blue`, `magenta`,
`cyan`, `white`, plus `bright_*` variants and `bg_*` backgrounds.

```python
from bunpy.set_system_time import set_system_time, now, reset

# Freeze the clock for testing
set_system_time("2020-01-15T10:00:00Z")
t = now()
# {"year": 2020, "month": 1, "day": 15, "hour": 10, ..., "iso": "2020-01-15T10:00:00Z"}

# Unix milliseconds also accepted
set_system_time(1_685_577_600_000)

# Restore real time
reset()
# or: set_system_time() with no args
```

## Notes

- `terminal.columns()` and `terminal.rows()` read the terminal size via
  `TIOCGWINSZ` on Unix; fallback to 80x24 if not available.
- `set_system_time` state is global to the process — always call `reset()`
  in test teardown.
- `now()` fields: `year`, `month`, `day`, `hour`, `minute`, `second`,
  `unix` (seconds), `unix_ms` (milliseconds), `iso` (RFC3339 UTC).

# v0.4.18 -- bunpy.Worker

Released 2026-04-28.

`bunpy.Worker` runs Python callables in background goroutines with
message-passing. Modelled after the Web Workers API. Pure goroutines, no new
dependencies.

## What shipped

```python
from bunpy.Worker import Worker

def my_worker(post_message):
    # Called immediately in a background goroutine.
    # post_message(data) sends a message back to the main thread.
    post_message("ready")

w = Worker(my_worker)

# Listen for messages sent back from the worker
w.on("message", lambda data: print("worker says:", data))

# Send a message to the worker (routed to any "message" listeners)
w.post_message({"task": "compute", "n": 42})

# Stop the worker
w.terminate()
```

## API

| Method | Description |
| --- | --- |
| `Worker(fn)` | Creates a worker; calls `fn(post_message)` in a new goroutine |
| `worker.post_message(data)` | Send data to the worker's message listeners |
| `worker.on(event, listener)` | Register a listener for `"message"` or `"exit"` events |
| `worker.terminate()` | Stop the worker; further `post_message` calls will raise |

## Notes

- The worker function receives a `post_message` callable it can use to send
  data back to the main thread.
- After the worker function returns, the worker continues to route
  `post_message` calls to `"message"` listeners until `terminate()` is called.
- `terminate()` is idempotent; calling it more than once is safe.
- `"exit"` listeners are called after the worker goroutine ends.

# v0.4.17 -- bunpy.URLPattern

Released 2026-04-28.

`bunpy.URLPattern` matches and parses URL path patterns like `/users/:id`
and `/files/*`. Inspired by the WHATWG URLPattern API. Pure stdlib, no new
dependencies.

## What shipped

```python
from bunpy.URLPattern import URLPattern

# Named parameters
p = URLPattern("/users/:id/posts/:postId")
p.test("/users/42/posts/99")   # True
p.test("/other/path")          # False

result = p.exec("/users/42/posts/99")
# {"pathname": {"input": "/users/42/posts/99",
#               "groups": {"id": "42", "postId": "99"}}}

# Wildcard
p = URLPattern("/files/*")
p.test("/files/assets/app.css")   # True

# Static path
p = URLPattern("/about")
p.test("/about")     # True
p.test("/about/us")  # False

# Full URLs are accepted; only the pathname is matched
p = URLPattern("/users/:id")
p.test("https://example.com/users/99")  # True
```

## Notes

- `:name` captures a single path segment (no slashes).
- `*` captures the rest of the path including slashes.
- `exec()` returns a dict matching the WHATWG URLPattern result shape, or
  `None` if there is no match.
- Query strings and fragments are stripped before matching.

# v0.4.16 -- bunpy.yaml

Released 2026-04-28.

`bunpy.yaml` for parsing and serializing YAML. Pure-Go implementation with
no new dependencies covering scalars, mappings, sequences, booleans, nulls,
nested structures, and flow sequences/mappings.

## What shipped

```python
from bunpy.yaml import parse, stringify

# Parse YAML into Python objects
config = parse("""
server:
  host: localhost
  port: 8080
features:
  - auth
  - logging
debug: true
""")
# {"server": {"host": "localhost", "port": 8080},
#  "features": ["auth", "logging"], "debug": True}

# Serialize Python objects to YAML
yaml = stringify({"name": "alice", "score": 42})
# name: alice
# score: 42
```

## Notes

- Parses block-style YAML: scalars, mappings (key: value), sequences (- item),
  and nested combinations.
- Also handles flow sequences `[a, b, c]` and flow mappings `{a: b}` as
  values.
- Type coercion: `true`/`yes`/`on` to bool, `null`/`~` to None, integers
  and floats auto-detected.
- Quoted strings (`"..."` / `'...'`) have their quotes stripped on parse.
- `stringify` produces block-style YAML and double-quotes strings that
  contain YAML special characters.
- This is a practical subset of YAML suitable for config files, not full
  YAML 1.2 spec compliance.

# v0.4.15 -- bunpy.cookie + bunpy.csrf

Released 2026-04-28.

`bunpy.cookie` parses and serializes HTTP cookies. `bunpy.csrf` generates
and verifies HMAC-signed CSRF tokens. Both use only the Go standard library.

## What shipped

```python
from bunpy.cookie import parse, serialize

# Parse a Cookie header into a dict
cookies = parse("session=abc123; user=alice")
# {"session": "abc123", "user": "alice"}

# Build a Set-Cookie header value
header = serialize(
    "session", "abc123",
    path="/",
    http_only=True,
    secure=True,
    same_site="Lax",
    max_age=3600,
)
# "session=abc123; Path=/; Max-Age=3600; HttpOnly; Secure; SameSite=Lax"
```

```python
from bunpy.csrf import token, verify

# Unsigned token (random nonce only)
t = token()
verify(t)               # True if non-empty

# HMAC-signed token
t = token(secret="my-secret")
verify(t, secret="my-secret")   # True
verify("tampered", secret="my-secret")  # False
```

## Notes

- `cookie.serialize()` uses `net/http.Cookie.String()` so the output follows
  RFC 6265 formatting.
- `cookie.parse()` handles a `Cookie:` request header (semicolon-separated
  key=value pairs), not a `Set-Cookie` response header.
- `csrf.token(secret=...)` produces `nonce.HMAC-SHA256(nonce)`.
- `csrf.verify(token, secret=...)` uses `hmac.Equal` to prevent timing attacks.
- Both `token()` and `verify()` accept `secret` as a positional or keyword arg.

# v0.4.14 -- bunpy.escape_html + bunpy.HTMLRewriter

Released 2026-04-28.

`bunpy.escape_html` for safe HTML escaping and `bunpy.HTMLRewriter` for
streaming element-level HTML transformation. Both are pure stdlib, no new
dependencies.

## What shipped

```python
from bunpy.escape_html import escape, unescape, strip_tags

escape('<script>alert("xss")</script>')
# "&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;"

unescape("&lt;b&gt;bold&lt;/b&gt;")
# "<b>bold</b>"

strip_tags("<p>hello <em>world</em></p>")
# "hello world"
```

```python
from bunpy.HTMLRewriter import HTMLRewriter

html = '<a href="/old">click</a>'

def rewrite_link(el):
    el.set_attribute("href", "https://example.com")
    el.set_attribute("target", "_blank")

out = HTMLRewriter(html).on("a", rewrite_link).transform()
# '<a href="https://example.com" target="_blank">click</a>'
```

### Element handler API

| Method | Effect |
| --- | --- |
| `el.set_attribute(name, value)` | Add or replace an attribute |
| `el.remove_attribute(name)` | Drop an attribute |
| `el.get_attribute(name)` | Return attribute value or None |
| `el.set_inner_content(html)` | Replace the element's children |
| `el.prepend(html)` | Inject HTML before the opening tag |
| `el.append(html)` | Inject HTML after the opening tag |
| `el.remove()` | Drop the element entirely |

## Notes

- `on(selector, handler)` returns the rewriter for chaining.
- Selector can be a tag name (e.g., `"a"`, `"script"`) or `"*"` for all
  elements.
- `transform()` with no argument uses the HTML passed to the constructor;
  pass a new string to override.
- The rewriter handles a subset of HTML (no namespace-aware parsing) and is
  suitable for server-side template post-processing, not full DOM manipulation.

# v0.4.13 -- bunpy.semver + bunpy.deep_equals

Released 2026-04-28.

Two new modules: `bunpy.semver` for semantic version parsing and range
checking, and `bunpy.deep_equals` for recursive structural equality. Both
use only the Go standard library.

## What shipped

```python
from bunpy.semver import parse, compare, satisfies, gt, gte, lt, lte, eq, valid

v = parse("1.2.3")      # {"major": 1, "minor": 2, "patch": 3, "pre": "", "build": ""}
parse("v2.0.0-beta.1")  # v prefix stripped, pre filled

compare("1.0.0", "2.0.0")   # -1
compare("2.0.0", "1.0.0")   # 1
compare("1.0.0", "1.0.0")   # 0

gt("2.0.0", "1.0.0")    # True
lt("1.0.0", "2.0.0")    # True
gte("1.0.0", "1.0.0")   # True
eq("1.0.0", "1.0.0")    # True

satisfies("1.5.0", "^1.0.0")   # True  (caret: same major)
satisfies("1.2.3", "~1.2.0")   # True  (tilde: same minor)
satisfies("1.2.3", ">=1.2.0")  # True  (comparison operators)
satisfies("1.2.3", "1.x")      # True  (wildcard)

valid("1.2.3")    # True
valid("notver")   # False
```

```python
from bunpy.deep_equals import deep_equals

deep_equals({"a": [1, 2]}, {"a": [1, 2]})   # True
deep_equals({"a": 1}, {"a": 2})              # False
deep_equals([1, [2, 3]], [1, [2, 3]])        # True
deep_equals(None, None)                      # True
```

## Notes

- `semver.parse()` accepts an optional leading `v` prefix.
- Range formats: `^` (caret, same-major), `~` (tilde, same-minor),
  `>=`, `<=`, `>`, `<`, `=`, `x`/`*` wildcards, and exact versions.
- Pre-release ordering: a release version is considered greater than the
  same version with a pre-release tag.
- `deep_equals` handles int, float, str, bytes, bool, list, tuple, dict,
  and None. Int and float are compared numerically across types.

# v0.4.12 -- bunpy.dns

Released 2026-04-28.

`bunpy.dns` module for DNS lookups: A/AAAA, MX, NS, TXT, CNAME, PTR record
types plus convenience helpers `lookup` and `reverse`. Pure stdlib, no new
dependencies.

## What shipped

```python
from bunpy.dns import resolve, lookup, reverse

# A/AAAA records (default)
addrs = resolve("example.com")           # ["93.184.216.34"]

# specific record type
mx = resolve("example.com", "MX")       # [{"host": "...", "pref": 10}]
ns = resolve("example.com", "NS")       # [{"host": "a.iana-servers.net."}]
txt = resolve("example.com", "TXT")     # ["v=spf1 ..."]
cname = resolve("www.example.com", "CNAME")  # ["example.com."]
ptr = resolve("8.8.8.8", "PTR")         # ["dns.google."]

# also accepts type as keyword
mx = resolve("example.com", type="MX")

# convenience: first address as a string, or None
ip = lookup("example.com")              # "93.184.216.34"

# reverse DNS
names = reverse("8.8.8.8")             # ["dns.google."]
```

## Notes

- `resolve(host, type="A")` returns a list; for MX each item is a dict with
  `host` and `pref` keys; for NS each item is a dict with a `host` key.
- `lookup(host)` returns the first address as a string, or `None` if the host
  does not resolve.
- `reverse(ip)` wraps `net.LookupAddr`; returns a list of PTR names.
- Unsupported record types raise a descriptive error.

# v0.4.11 -- timer globals

Released 2026-04-28.

`setTimeout`, `clearTimeout`, `setInterval`, and `clearInterval` injected
into the script builtins (no import required). Pure Go goroutines and
`time.After` / `time.Ticker`, no new dependencies.

## What shipped

```python
# Available without any import, like in a browser or Bun

id = setTimeout(lambda: print("fired"), 500)   # ms
clearTimeout(id)

id = setInterval(lambda: print("tick"), 1000)
clearInterval(id)
```

## Notes

- `setTimeout(fn, ms)` fires once after `ms` milliseconds.
- `setInterval(fn, ms)` fires every `ms` milliseconds until cleared.
- Both return an integer timer ID usable with `clearTimeout`/`clearInterval`.
- Clearing a non-existent ID is a no-op (does not raise).
- Callbacks run in a background goroutine; the interpreter is re-entrant.

# v0.4.10 -- bunpy.http

Released 2026-04-28.

Full-featured HTTP client with sessions, persistent headers, cookie jar,
and retry via `import bunpy.http as http`. Pure Go `net/http`, no new
dependencies.

## What shipped

### `import bunpy.http as http`

```python
import bunpy.http as http

# One-off requests
resp = http.get("https://example.com")
resp = http.post("https://api.example.com/data", json={"key": "val"})
resp = http.put("https://...", body=b"raw bytes", headers={"X-Custom": "val"})
resp = http.delete("https://...")

# Response object
resp.status         # 200
resp.ok             # True if 200-299
resp.text()         # str
resp.json()         # parsed dict/list
resp.bytes()        # raw bytes
resp.headers        # dict of response headers

# Session (persistent cookies + base headers + base URL)
s = http.session(
    base_url="https://api.example.com",
    headers={"Authorization": "Bearer token"},
    timeout=30,
    retries=3,
)

resp = s.get("/users")
resp = s.post("/users", json={"name": "Alice"})
s.close()

# Context manager
with http.session(base_url="https://api.example.com") as s:
    resp = s.get("/health")
```

## Notes

- `json=dict` serializes the dict as JSON and sets `Content-Type: application/json`.
- `response.json()` now handles nested dicts and lists (returns proper dict/list objects).
- Sessions use a cookie jar (`net/http/cookiejar`) so cookies are persisted across requests.
- Relative URLs in sessions are resolved against `base_url`.
- Retry uses exponential backoff: 100ms, 200ms, 400ms for `retries=3`.
- Also fixed: `goValueToPyObj` now handles `map[string]any` and `[]any` for correct JSON parsing.

# v0.4.9 -- bunpy.queue

Released 2026-04-28.

In-process background job queue with configurable worker pool via
`import bunpy.queue as queue`. Pure Go channels and goroutines, no new
dependencies.

## What shipped

### `import bunpy.queue as queue`

```python
import bunpy.queue as queue

q = queue.new(workers=4)

@q.handler("send_email")
def handle_email(job):
    print(job["to"], job["subject"])

job_id = q.enqueue("send_email", {"to": "x@example.com", "subject": "Hi"})

q.wait()    # block until all jobs are done
q.stop()    # stop workers

# Context manager
with queue.new(workers=2) as q:
    q.handler("ping")(lambda job: print("ping"))
    q.enqueue("ping", {})
```

## Job dict fields

| Field | Type | Description |
|---|---|---|
| `id` | str | UUID v4 generated at enqueue time |
| `type` | str | job type string |
| `data` | dict | payload passed to enqueue |
| `attempt` | int | always 1 in v0.4.9; retry is a future addition |

## Notes

- Jobs are dispatched to workers over a buffered channel (size 1024).
- Handler errors are silently dropped; the worker continues with the next job.
- `stop()` drains the queue (waits for in-flight jobs) then closes the channel.
- Enqueueing after `stop()` raises an error.

# v0.4.8 -- bunpy.cache

Released 2026-04-28.

In-memory LRU/TTL cache via `import bunpy.cache as cache`. Pure Go
`container/list`, no new dependencies.

## What shipped

### `import bunpy.cache as cache`

```python
import bunpy.cache as cache

# Unbounded, no TTL
c = cache.new()

# Max 1000 entries (LRU eviction when full)
c = cache.new(max_size=1000)

# Default TTL of 5 minutes
c = cache.new(ttl=300)

# Both
c = cache.new(max_size=500, ttl=60)

c.set("key", "value")
c.set("key", "value", ttl=10)   # per-key TTL override

val  = c.get("key")              # None if missing or expired
val  = c.get("key", "default")  # default if missing

c.delete("key")
c.clear()

if c.has("key"):
    ...

stats = c.stats()  # {"size": 42, "hits": 100, "misses": 5}
```

## Implementation

Doubly-linked list + dict for O(1) get, set, and eviction. TTL stored as
`expireAt time.Time` per entry. Access probes expiry; expired entries are
lazily removed on next get.

# v0.4.7 -- bunpy.email

Released 2026-04-28.

SMTP email sending via `import bunpy.email as email`. Backed by Go stdlib
`net/smtp`, no new dependencies.

## What shipped

### `import bunpy.email as email`

```python
import bunpy.email as email

email.configure(
    host="smtp.gmail.com",
    port=587,
    username="me@gmail.com",
    password="app-password",
)

# Plain text
email.send(to="you@example.com", subject="Hello", body="Hi there.")

# HTML
email.send(to="you@example.com", subject="Hi", html="<p>Hello</p>")

# Both (multipart/alternative)
email.send(to="you@example.com", subject="Hi", body="text", html="<p>html</p>")

# Multiple recipients
email.send(to=["a@x.com", "b@x.com"], subject="News", body="...")

# Custom from address
email.send(to="you@x.com", subject="Hi", body=".", from_="noreply@x.com")
```

## Transport

Port 587 uses STARTTLS (`smtp.SendMail`). Port 465 opens an implicit TLS
connection first. Credentials fall back to `SMTP_HOST`, `SMTP_PORT`,
`SMTP_USERNAME`, `SMTP_PASSWORD` environment variables if not configured.

# v0.4.6 -- bunpy.template

Released 2026-04-28.

Text and HTML template rendering via `import bunpy.template as tmpl`.
Backed by Go stdlib `text/template` and `html/template`, no new dependencies.

## What shipped

### `import bunpy.template as tmpl`

```python
import bunpy.template as tmpl

# Render inline string template
out = tmpl.render("Hello, {{ .name }}!", {"name": "Alice"})
# "Hello, Alice!"

# HTML mode: values are auto-escaped
out = tmpl.render("Hi {{ .name }}", {"name": "<b>Bob</b>"}, html=True)
# "Hi &lt;b&gt;Bob&lt;/b&gt;"

# Render from file
out = tmpl.render_file("views/index.html", {"title": "Home"})

# Compile once, render many times
t = tmpl.compile("Hello, {{ .name }}!")
out = t.render({"name": "Alice"})
```

## Notes

- Templates use Go syntax: `{{ .key }}`, `{{ range .items }}...{{ end }}`,
  `{{ if .cond }}...{{ end }}`.
- Lists and nested dicts are converted to Go slices and maps automatically.
- `html=True` uses `html/template` which auto-escapes all interpolated values.
- Syntax errors in `compile()` are caught at compile time.

# v0.4.5 -- bunpy.csv

Released 2026-04-28.

CSV parse and write via `import bunpy.csv as csv`. Backed by Go stdlib
`encoding/csv`, no new dependencies.

## What shipped

### `import bunpy.csv as csv`

```python
import bunpy.csv as csv

# Parse CSV string with header -> list of dicts
rows = csv.parse("name,age\nAlice,30\nBob,25")
# [{"name": "Alice", "age": "30"}, {"name": "Bob", "age": "25"}]

# Parse without header -> list of lists
rows = csv.parse("Alice,30\nBob,25", header=False)

# Write list of dicts to CSV string
out = csv.write([{"name": "Alice", "age": "30"}])

# Write list of lists with explicit header
out = csv.write([["Alice", "30"]], header=["name", "age"])

# File variants
rows = csv.parse_file("data.csv")
csv.write_file("out.csv", rows)
```

## Notes

- Quoted fields and embedded commas are handled correctly.
- `write` auto-detects dicts vs lists from the first row.
- Header row for dict mode is taken from the first row's keys unless
  `header=` is provided.

# v0.4.4 -- bunpy.jwt

Released 2026-04-28.

JWT HS256 sign and verify via `import bunpy.jwt as jwt`. Pure Go stdlib,
no new dependencies.

## What shipped

### `import bunpy.jwt as jwt`

```python
import bunpy.jwt as jwt

# Sign
token = jwt.sign({"sub": "user:42", "role": "admin"}, "my-secret")

# Verify (raises ValueError on bad signature or expired token)
payload = jwt.verify(token, "my-secret")
# {"sub": "user:42", "role": "admin", "iat": 1714320000}

# Sign with expiry (seconds from now)
token = jwt.sign({"sub": "user:42"}, "secret", exp=3600)

# Decode without signature check (for inspection only)
payload = jwt.decode(token)
```

## Notes

- Algorithm is always HS256. RS256 can be added later.
- `sign` automatically adds `iat` (issued-at Unix timestamp).
- `verify` checks the HMAC signature and raises if `exp` is present and
  the token is expired.
- `decode` is unsigned and should not be used for authentication decisions.

# v0.4.3 -- bunpy.crypto

Released 2026-04-28.

AES-256-GCM authenticated encryption, HMAC-SHA256, and SHA-256/SHA-512
digests via `import bunpy.crypto as crypto`. Pure Go stdlib, no new dependencies.

## What shipped

### `import bunpy.crypto as crypto`

```python
import bunpy.crypto as crypto

# Secure random bytes
key = crypto.random(32)

# AES-256-GCM encrypt / decrypt
ct = crypto.encrypt(b"hello world", key)
pt = crypto.decrypt(ct, key)            # b"hello world"

# HMAC-SHA256
sig = crypto.hmac(b"message", b"key")
ok  = crypto.hmac_verify(b"message", b"key", sig)  # True

# Digests
h = crypto.sha256(b"data")       # bytes
h = crypto.sha256_hex(b"data")   # hex string
h = crypto.sha512(b"data")       # bytes
h = crypto.sha512_hex(b"data")   # hex string
```

## Notes

- `encrypt` prepends a 12-byte random nonce; `decrypt` reads the first 12 bytes
  as the nonce. The nonce is never reused since it is randomly generated each call.
- `hmac_verify` uses constant-time comparison to prevent timing attacks.
- str arguments are automatically encoded as UTF-8.

# v0.4.2 -- bunpy.uuid

Released 2026-04-28.

UUID generation supporting v4 (random) and v7 (time-ordered) via
`import bunpy.uuid as uuid`. Pure Go, no new dependencies.

## What shipped

### `import bunpy.uuid as uuid`

```python
import bunpy.uuid as uuid

u = uuid.v4()            # random UUID, e.g. "550e8400-e29b-41d4-a716-446655440000"
u = uuid.v7()            # time-ordered UUID, sortable as a string
ok = uuid.is_valid(u)    # True
ok = uuid.is_valid("x")  # False
```

## Notes

- `v4` uses `crypto/rand` for all 122 random bits.
- `v7` encodes a 48-bit Unix millisecond timestamp in the top bits so
  lexicographic string sort matches chronological order.
- `is_valid` checks the standard 8-4-4-4-12 lowercase hex format.

# v0.4.1 -- bunpy.log

Released 2026-04-28.

Structured logging backed by Go's `log/slog` via `import bunpy.log as log`.

## What shipped

### `import bunpy.log as log`

```python
import bunpy.log as log

log.info("server started", port=8080)
log.warn("slow query", ms=320)
log.error("connection failed", err="timeout")
log.debug("cache miss", key="user:42")

# Configure format and level
log.configure(level="debug", format="json")
log.configure(level="info", format="text", file="app.log")

# Child logger with bound fields
req_log = log.with_fields(request_id="abc123", user_id=42)
req_log.info("handled request", status=200)
```

## Levels

`debug`, `info` (default), `warn`, `error`. Calls below the current level
are no-ops.

## Formats

`text` (default, key=value) and `json` (one JSON object per line).

## Notes

- `configure(file=path)` opens the file in append mode; existing content is preserved.
- `with_fields(...)` returns a child logger object that inherits the current
  handler and prepends the bound fields to every log record.
- The global logger is updated atomically so configure is safe to call from
  concurrent goroutines.

# v0.4.0 -- bunpy.env

Released 2026-04-28.

dotenv file loader and environ wrapper via `import bunpy.env as env`.

## What shipped

### `import bunpy.env as env`

```python
import bunpy.env as env

env.load()                          # loads .env from cwd
env.load(".env.production")         # loads a specific file

val  = env.get("DATABASE_URL")      # str or None
port = env.get("PORT", "8080")      # with default

port  = env.int("PORT", 8080)       # coerced to int
debug = env.bool("DEBUG", False)    # coerced to bool
rate  = env.float("RATE", 1.5)      # coerced to float

env.set("FOO", "bar")               # write to current process
d = env.all()                       # all env vars as dict
```

### dotenv file format

- `KEY=value`
- `KEY="quoted value"` -- double or single quotes stripped
- Lines starting with `#` are comments and ignored
- Blank lines ignored
- No variable expansion

# v0.3.12 -- bunpy.password

Released 2026-04-28.

Password hashing with bcrypt and Argon2id via `golang.org/x/crypto`.
Both algorithms produce self-describing encoded strings that carry all
parameters needed for verification.

## What shipped

### `import bunpy.password as password`

```python
import bunpy.password as password

# Hash (bcrypt by default)
h = password.hash("hunter2")
# "$2a$10$..."

# Verify
ok = password.verify("hunter2", h)   # True
ok = password.verify("wrong",   h)   # False
```

### `password.hash(pw, algo="bcrypt", **options) -> str`

| algo | Options | Description |
|---|---|---|
| `"bcrypt"` | `cost` (default 10) | Standard bcrypt, `$2a$` prefix |
| `"argon2id"` | `memory` (default 65536 KiB), `time` (default 1), `threads` (default 4) | Argon2id PHC string, `$argon2id$` prefix |

```python
# Bcrypt with custom cost
h = password.hash("secret", cost=12)

# Argon2id
h = password.hash("secret", algo="argon2id")
h = password.hash("secret", algo="argon2id", memory=65536, time=2, threads=4)
```

### `password.verify(pw, hash) -> bool`

Detects the algorithm from the hash prefix automatically. Returns `False`
for mismatches and for malformed hashes (does not raise an exception).
Uses constant-time comparison for Argon2id to prevent timing attacks.

## Implementation

`api/bunpy/password.go` implements both algorithms using
`golang.org/x/crypto/bcrypt` and `golang.org/x/crypto/argon2`.

Argon2id hashes are encoded as PHC strings:
```
$argon2id$v=19$m=65536,t=1,p=4$<base64-salt>$<base64-hash>
```

`verifyPassword` detects the algorithm from the `$argon2id$` prefix and
falls back to bcrypt for `$2a$`, `$2b$`, and `$2y$` prefixes.

## Notes

- bcrypt passwords longer than 72 bytes are silently truncated by the
  algorithm. For passwords over 72 bytes, use Argon2id instead.
- Argon2id is the recommended choice for new applications per OWASP.
  bcrypt is provided for compatibility with existing hashes.
- The `memory` parameter for Argon2id is in KiB (1 KiB = 1024 bytes).
  The default of 65536 KiB = 64 MiB is the OWASP minimum recommendation.

# v0.3.11 -- bunpy.cron

Released 2026-04-28.

Pure-Go cron scheduler. Schedule Python functions using standard cron
expressions. No external scheduler library.

## What shipped

### `bunpy.cron(expr, handler) -> CronJob`

```python
import bunpy

def backup():
    bunpy.shell("pg_dump mydb > backup.sql")

# Every minute
job = bunpy.cron("* * * * *", backup)

# Every day at midnight
job = bunpy.cron("0 0 * * *", backup)

# Every Monday at 9am
job = bunpy.cron("0 9 * * 1", backup)

# Shorthand forms
job = bunpy.cron("@daily",   backup)
job = bunpy.cron("@hourly",  backup)
job = bunpy.cron("@weekly",  backup)
job = bunpy.cron("@monthly", backup)
job = bunpy.cron("@yearly",  backup)

# Stop the job
job.stop()

# Hot-reload the handler without stopping the schedule
job.reload(new_handler)

# Read back the expression
print(job.expr)   # "0 0 * * *"
```

Handlers are called in a background goroutine. Each call gets its own
goroutine so a slow handler does not block the next scheduled run.

### Cron expression syntax

5-field standard cron: `minute hour day-of-month month day-of-week`

| Field | Range | Special |
|---|---|---|
| minute | 0-59 | `*`, `*/N`, `N-M`, `N,M` |
| hour | 0-23 | same |
| day-of-month | 1-31 | same |
| month | 1-12 | same |
| day-of-week | 0-6 (0=Sun) | same |

Shorthand: `@yearly`, `@monthly`, `@weekly`, `@daily`, `@hourly`.

## Implementation

`api/bunpy/cron.go` implements the expression parser and scheduler.
`parseCronExpr` expands each field to a `[]int` of matching values.
`nextTick` scans forward minute-by-minute to find the next fire time
(up to 366 days ahead). The background goroutine waits until the next
tick using `time.After`, then calls the handler in a new goroutine.

`NextCronTick(expr, base)` is exported for testing the scheduler logic
without starting background goroutines.

## Notes

- Day-of-week 0 and 7 both map to Sunday in many cron implementations.
  In v0.3.11 only 0 is Sunday; 7 is not handled.
- When both day-of-month and day-of-week are specified, the behavior
  follows the `OR` rule (standard Vixie cron). This is not implemented
  in v0.3.11; both fields must match simultaneously.
- Handlers that raise exceptions log the error to stderr but do not
  stop the job.

# v0.3.10 -- bunpy.WebSocket

Released 2026-04-28.

Pure-Go WebSocket client (RFC 6455). Connect to any `ws://` endpoint,
send text or binary frames, receive frames, close cleanly.

## What shipped

### `import bunpy.WebSocket as WebSocket` / `WebSocket.connect(url)`

```python
import bunpy.WebSocket as WebSocket

ws = WebSocket.connect("ws://localhost:8080/chat")

# Send text
ws.send("hello server")

# Send binary
ws.send(b"\x00\x01\x02")

# Receive (blocks until a frame arrives)
msg = ws.recv()   # str for text frames, bytes for binary frames
                   # None if the server sent a close frame

# Close (sends a close frame and closes the TCP connection)
ws.close()

# Context manager
with WebSocket.connect("ws://localhost:8080/ws") as ws:
    ws.send("ping")
    print(ws.recv())
```

## Implementation

`api/bunpy/websocket.go` implements the RFC 6455 handshake and framing.
The HTTP upgrade is performed manually over a raw TCP connection.
Client frames are masked with a random 4-byte key as required by the spec.
Server frames are not masked.

`WSAccept`, `WSReadFrame`, and `WSWriteServerFrame` are exported for
testing; the echo server test exercises the full send/recv round-trip.

## Notes

- `wss://` (TLS WebSocket) is not supported in v0.3.10. Use a
  TLS-terminating reverse proxy (nginx, Caddy) in front of the WebSocket
  server.
- `ws.recv()` blocks synchronously. For concurrent message handling, run
  the recv loop in a thread (stdlib `threading` module).
- The implementation handles text (opcode 1), binary (opcode 2), and
  close (opcode 8) frames. Ping/pong (opcodes 9/10) are not handled
  automatically; they are returned as raw bytes.
- Fragmented messages (continuation frames) are not reassembled in
  v0.3.10. Each `recv()` call returns exactly one frame.

# v0.3.9 -- bunpy.s3

Released 2026-04-28.

Pure-Go S3-compatible object storage client. Uses AWS Signature V4.
Works with AWS S3, Cloudflare R2, MinIO, and any S3-compatible endpoint.
No external SDK.

## What shipped

### `import bunpy.s3 as s3` / `s3.connect(...)`

```python
import bunpy.s3 as s3

# Explicit credentials
client = s3.connect(
    bucket="my-bucket",
    access_key="AKIA...",
    secret_key="wJalr...",
    region="us-east-1",
)

# Reads AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_REGION from env
client = s3.connect(bucket="my-bucket")

# Custom endpoint (R2, MinIO)
client = s3.connect(
    bucket="my-bucket",
    endpoint="https://account.r2.cloudflarestorage.com",
    access_key="...",
    secret_key="...",
    region="auto",
)
```

### Operations

```python
# Read
data = client.read("images/photo.jpg")       # bytes

# Write
client.write("images/photo.jpg", b"\xff\xd8...")
client.write("notes.txt", "hello world")     # str -> UTF-8

# Delete
client.delete("old-file.txt")

# Exists
if client.exists("config.json"):
    cfg = client.read("config.json")

# List
keys = client.list()                     # list of str
keys = client.list(prefix="images/")    # filtered

# Pre-signed URL
url = client.presign("image.jpg", expires=3600)

# Close / context manager
client.close()
with s3.connect(bucket="my-bucket") as client:
    data = client.read("key")
```

## Implementation

`api/bunpy/s3.go` implements everything. Uses path-style URLs
(`endpoint/bucket/key`) so it works with all endpoints.

AWS Signature V4 signing:
1. Canonical request (method, URI, query, headers, signed headers, body hash)
2. String to sign (algorithm, date, scope, canonical-request hash)
3. Signing key derived via `HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region), "s3"), "aws4_request")`
4. Signature = `HMAC-SHA256(signingKey, stringToSign)`

`presignURL` puts all auth parameters in the query string instead of
headers. `listObjects` parses the ListObjectsV2 XML response. Up to
1000 keys per call; pagination is not implemented.

## Notes

- Path-style URLs only (`endpoint/bucket/key`). Virtual-hosted style
  (`bucket.endpoint/key`) is not implemented.
- `list()` returns up to 1000 keys. Pagination via `ContinuationToken`
  is a future enhancement.
- Integration tests require a live S3-compatible endpoint and are
  tagged `//go:build integration`. Set `BUNPY_TEST_S3_ENDPOINT`,
  `BUNPY_TEST_S3_BUCKET`, and credentials to run them.

# v0.3.8 -- bunpy.redis

Released 2026-04-28.

Pure-Go Redis client with no external dependencies. All commands are
synchronous. Implements the RESP2 protocol directly over a TCP socket.

## What shipped

### `import bunpy.redis as redis` / `redis.connect(url)`

```python
import bunpy.redis as redis

r = redis.connect("redis://localhost:6379")
r = redis.connect("redis://user:pass@localhost:6379/0")
```

Default URL: `redis://localhost:6379`.

### String commands

```python
r.set("key", "value")
r.set("key", "value", ex=60)    # expire in 60 seconds
r.set("key", "value", nx=True)  # only set if not exists
val = r.get("key")               # str or None
r.del_("key")                    # del is a Python keyword
exists = r.exists("key")         # bool
```

### Counters

```python
r.incr("counter")
r.incrby("counter", 5)
r.decr("counter")
```

### Lists

```python
r.lpush("list", "item1", "item2")
r.rpush("list", "item3")
items = r.lrange("list", 0, -1)   # list of str
n = r.llen("list")                 # int
```

### Hashes

```python
r.hset("hash", "field", "value")
val  = r.hget("hash", "field")    # str or None
all_ = r.hgetall("hash")          # Instance with field attributes
r.hdel("hash", "field")
ok = r.hexists("hash", "field")   # bool
```

### TTL

```python
r.expire("key", 60)   # set TTL in seconds
n = r.ttl("key")      # int: remaining seconds; -1=no TTL; -2=missing
r.persist("key")      # remove TTL
```

### Publish

```python
r.publish("channel", "message")
```

### Close / context manager

```python
r.close()

with redis.connect("redis://localhost:6379") as r:
    r.set("key", "val")
```

## Implementation

`api/bunpy/redis.go` implements the full client. RESP2 protocol is
serialized in `send()` and parsed in `parseRESPReply()` — no external
Redis library. A `sync.Mutex` serializes commands on the single TCP
connection.

`ParseRESPReply(io.Reader)` is exported for testing the protocol parser
without a live Redis server. Tests cover all six RESP2 types: simple
string, error, integer, bulk string, nil bulk, and arrays.

The `bunpy.redis` module is registered in `Modules()` so Python scripts
can `import bunpy.redis`.

## Notes

- All values are returned as `str`. `r.get("count")` returns `"42"`,
  not `42`. Convert as needed.
- `hgetall` returns an `Instance` keyed by field name, not a plain
  dict. Access fields as `result["field_name"]` is not available
  in v0.3.8; this will improve in a later rung.
- Single connection only. For concurrent scripts, use one client per
  goroutine or add a pool later.
- RESP2 only. RESP3 (Redis 6+) support is not implemented.
- Integration tests require a live Redis and are tagged
  `//go:build integration`. Set `BUNPY_TEST_REDIS_URL` to run them.

# v0.3.7 -- bunpy.sql Postgres + MySQL

Released 2026-04-28.

`bunpy.sql` now accepts Postgres and MySQL connection URLs in addition
to SQLite. The Python API is identical across all three backends.

## What shipped

### Postgres

```python
import bunpy

db = bunpy.sql("postgres://user:pass@localhost:5432/mydb")
db = bunpy.sql("postgresql://user:pass@localhost/mydb")  # alias

rows = db.query("SELECT id, email FROM users WHERE active = $1", [True])
db.run("INSERT INTO users (email) VALUES ($1)", ["alice@example.com"])
```

Uses `github.com/jackc/pgx/v5/stdlib` (pure Go, no CGo).
Postgres uses `$1`, `$2`, ... placeholders.

### MySQL / MariaDB

```python
db = bunpy.sql("mysql://user:pass@localhost:3306/mydb")

rows = db.query("SELECT id, name FROM users WHERE active = ?", [1])
db.run("INSERT INTO users (name) VALUES (?)", ["Alice"])
```

Uses `github.com/go-sql-driver/mysql` (pure Go, no CGo).
MySQL uses `?` placeholders.

### URL scheme dispatch

| URL prefix | Driver | Notes |
|---|---|---|
| (empty) | SQLite in-memory | default |
| `sqlite:` | SQLite | file or `:memory:` |
| `postgres://` / `postgresql://` | pgx | accepts URL directly |
| `mysql://` | go-sql-driver | converts to `user:pass@tcp(host)/db` |

Unsupported schemes return an error at `bunpy.sql()` call time.

## Implementation

`api/bunpy/sql.go` gained an `openDB(rawURL)` function that selects the
driver based on URL prefix. `mysqlDSN` converts the standard
`mysql://user:pass@host/db` URL format to the DSN format expected by
go-sql-driver/mysql, appending `:3306` if no port is specified.

Connection pool defaults: `MaxOpenConns=25`, `MaxIdleConns=5`,
`ConnMaxLifetime=5m` are not yet set (plain `sql.Open` defaults). This
will be addressed in a later rung.

## Notes

- Postgres uses `$1`/`$2` placeholders; SQLite and MySQL use `?`.
  `bunpy.sql` does not translate between styles. Write queries in the
  native style of your target database.
- Integration tests (requiring a live Postgres or MySQL server) are
  tagged `//go:build integration` and skipped in the default CI run.
  Set `BUNPY_TEST_PG_URL` or `BUNPY_TEST_MYSQL_URL` to run them.

# v0.3.6 -- bunpy.sql (SQLite)

Released 2026-04-28.

Higher-level database API backed by SQLite. Rows come back as dicts
instead of tuples. Mirrors the Bun.js `Bun.sql` API introduced in
Bun 1.2.

## What shipped

### `bunpy.sql(url=None) -> Database`

Opens a SQLite database. No arguments gives an in-memory database.

```python
import bunpy

db = bunpy.sql()                        # in-memory
db = bunpy.sql("sqlite:./app.db")       # file
db = bunpy.sql("sqlite::memory:")       # explicit in-memory
```

### `db.query(sql, args=[]) -> list[dict]`

Runs a query and returns rows as dicts keyed by column name.

```python
db.run("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
db.run("INSERT INTO users (name) VALUES (?)", ["Alice"])
db.run("INSERT INTO users (name) VALUES (?)", ["Bob"])

rows = db.query("SELECT id, name FROM users ORDER BY name")
for row in rows:
    print(row["id"], row["name"])
```

### `db.query_one(sql, args=[]) -> dict | None`

Returns the first row as a dict, or `None` if no rows match.

```python
row = db.query_one("SELECT * FROM users WHERE id = ?", [1])
if row:
    print(row["name"])
```

### `db.run(sql, args=[])`

Executes a statement and discards the result.

### `db.run_many(sql, rows)`

Inserts multiple rows in a single transaction.

```python
db.run_many("INSERT INTO users (name) VALUES (?)",
            [["Dave"], ["Eve"], ["Frank"]])
```

### `db.transaction(fn)`

Calls `fn(tx)` inside a transaction. `tx` has the same `run` and
`query` methods as `db`. Commits on success; rolls back if `fn` raises.

```python
def setup(tx):
    tx.run("INSERT INTO users (name) VALUES (?)", ["Bob"])
    tx.run("INSERT INTO users (name) VALUES (?)", ["Carol"])

db.transaction(setup)
```

### `db.close()`

Closes the connection pool.

### Context manager

```python
with bunpy.sql("sqlite:./app.db") as db:
    rows = db.query("SELECT * FROM users")
```

`__exit__` closes the database.

## Implementation

`api/bunpy/sql.go` implements the full API. Uses `modernc.org/sqlite`,
a pure-Go SQLite driver with no CGo. `rowsToDicts` scans SQL column
values into `any` and maps them to goipy objects: `int64` to `Int`,
`float64` to `Float`, `string` to `Str`, `[]byte` to `Bytes`, `nil`
to `None`.

## Notes

- `db.query` and `db.run` use `?` as the placeholder character
  (SQLite style). Named parameters (`:name`, `@name`) are not tested.
- goipy's stdlib `sqlite3` module is not replaced; both coexist.
- SQLite is the only backend in v0.3.6. Postgres and MySQL come in v0.3.7.
- The in-memory database uses `cache=shared` so all connections in the
  pool see the same data. For truly isolated in-memory databases, use
  distinct file names: `file:memdb1?mode=memory&cache=shared`.

# v0.3.5 -- bunpy.glob

Released 2026-04-28.

Filesystem globbing with `**` recursive matching.

## What shipped

### `bunpy.glob(pattern, cwd=None, dot=False, absolute=False) -> list[str]`

Returns a list of paths matching the glob pattern. Supports `**` for
recursive directory traversal.

```python
import bunpy

# Basic glob
files = bunpy.glob("src/**/*.py")

# Different base directory
files = bunpy.glob("**/*.toml", cwd="/path/to/project")

# Include dot-files
files = bunpy.glob("**/*.py", dot=True)

# Return absolute paths
files = bunpy.glob("**/*.py", absolute=True)
```

Options:

| Option | Type | Default | Description |
|---|---|---|---|
| `cwd` | str | current directory | base directory for matching |
| `dot` | bool | `False` | include dot-files and dot-directories |
| `absolute` | bool | `False` | return absolute paths instead of relative |

Returns paths relative to `cwd` by default. With `absolute=True` returns
absolute paths.

Without `**`, falls back to `filepath.Glob` for single-level matching.
With `**`, walks the directory tree from the point of `**`.

### `bunpy.glob_match(pattern, name) -> bool`

Tests whether a single filename matches a glob pattern. Does not traverse
the filesystem.

```python
bunpy.glob_match("*.py", "script.py")   # True
bunpy.glob_match("*.py", "script.go")   # False
bunpy.glob_match("test_*", "test_foo")  # True
```

## Implementation

`api/bunpy/glob.go` implements both functions. `doubleStarGlob` splits on
the first `**` and uses `filepath.WalkDir` from that directory, filtering
by the suffix pattern. Dot-files are skipped at both the file and directory
level when `dot=False`, which prevents descending into `.git/` and `.venv/`.

Results are in `filepath.WalkDir` lexical order within each directory.

## Notes

- Patterns with more than one `**` (e.g. `a/**/b/**/c.py`) match only
  on the filename component after the first `**`. Full multi-star support
  is planned for a later release.
- `bunpy.glob.match` is not available in v0.3.5 because `BuiltinFunc`
  objects do not carry sub-attributes. Use `bunpy.glob_match(pattern, name)`
  instead.
- `filepath.WalkDir` does not follow symlinks. Circular symlink structures
  are safe.

# v0.3.4 -- bunpy.shell, bunpy.spawn, bunpy.dollar

Released 2026-04-28.

Three process-execution APIs mirroring the Bun.js shell and spawn
primitives. All three are synchronous at the Python level.

## What shipped

### `bunpy.shell(cmd, cwd=None, env=None, capture=True) -> ShellResult`

Runs a command string through the system shell (`/bin/sh -c` on Unix,
`cmd.exe /c` on Windows) and blocks until it exits.

```python
import bunpy

result = bunpy.shell("echo hello")
print(result.stdout)    # "hello\n"
print(result.stderr)    # ""
print(result.exitcode)  # 0

result = bunpy.shell("git status", cwd="/path/to/repo")
if result.exitcode != 0:
    raise RuntimeError(result.stderr)
```

Options:

| Option | Type | Default | Description |
|---|---|---|---|
| `cwd` | str | None | working directory |
| `env` | dict | None | environment overrides (merged over current env) |
| `capture` | bool | True | capture stdout/stderr vs inherit |

Non-zero exit code does not raise an exception. Check `result.exitcode`.

### `bunpy.spawn(argv, stdin=None, cwd=None, capture=True) -> Proc`

Runs a command as a list without a shell. Returns a `Proc` handle
immediately; the process runs in the background.

```python
proc = bunpy.spawn(["echo", "hi"])
proc.wait()
print(proc.stdout)    # "hi\n"
print(proc.exitcode)  # 0

proc = bunpy.spawn(["cat"], stdin="hello", capture=True)
proc.wait()
print(proc.stdout)  # "hello"

proc = bunpy.spawn(["sleep", "10"])
proc.kill()
```

`proc.pid` is available immediately. `stdout`/`stderr`/`exitcode`
are populated after `proc.wait()` returns.

### `bunpy.dollar(template, **kwargs) -> ShellResult`

Interpolates keyword arguments into a command template, then runs it
through `bunpy.shell`. Values are automatically shell-quoted.

```python
result = bunpy.dollar("echo {name}", name="world")
# stdout == "world\n"

result = bunpy.dollar("git commit -m {msg}", msg="fix: typo in readme")
# msg becomes 'fix: typo in readme' — spaces handled safely
```

`{key}` tokens that have no matching keyword are left unchanged.

## Implementation

`api/bunpy/shell.go` implements all three functions. `buildEnv` merges
kwargs `env` dict over the current process environment rather than
replacing it, which is the more useful default for scripts that only
need to override a few variables. `shellQuote` uses POSIX single-quote
escaping.

`BuildSpawn` starts the child process with `exec.Command.Start()` and
collects the exit code in a goroutine so `proc.wait()` can block
without hanging the Go scheduler.

## Notes

- `bunpy.shell` and `bunpy.dollar` run through the shell and are
  subject to shell injection if user-controlled strings are passed
  without quoting. Use `bunpy.dollar` with `{key}` placeholders or
  `bunpy.spawn` for safe argument passing.
- `proc.stdout`/`proc.stderr` before `proc.wait()` return whatever
  has been buffered so far and are not safe to read concurrently with
  the background goroutine.
- TLS, pipe communication, and non-blocking I/O are not supported
  in v0.3.4. Use `subprocess` from the stdlib for those cases.

# v0.3.3 -- bunpy.file, bunpy.write, bunpy.read

Released 2026-04-28.

Lazy file objects and two convenience I/O functions, mirroring the
Bun.js `Bun.file()` / `Bun.write()` built-ins.

## What shipped

### `bunpy.file(path) -> BunFile`

Returns a `BunFile` object. No I/O happens at construction time.

```python
import bunpy

f = bunpy.file("data.txt")
text  = f.text()    # str (UTF-8)
data  = f.bytes()   # bytes
size  = f.size()    # int -- stat, not a full read
there = f.exists()  # bool -- stat
name  = f.name      # str -- path as given
```

`f.size()` and `f.exists()` call `os.Stat`; they do not read the
file. `f.text()` and `f.bytes()` read the whole file into memory.

### `bunpy.write(path, data, append=False)`

Writes `data` to `path`, creating the file if it does not exist.
Truncates by default; pass `append=True` to append.

```python
bunpy.write("out.txt", "hello world")       # str -> UTF-8
bunpy.write("out.bin", b"\x00\x01\x02")    # bytes
bunpy.write("copy.bin", bunpy.file("src"))  # copy a BunFile
bunpy.write("log.txt", "entry\n", append=True)
```

Returns `None`.

### `bunpy.read(path) -> bytes`

Reads the full file and returns bytes. Equivalent to
`bunpy.file(path).bytes()`.

```python
data = bunpy.read("data.bin")
```

## Implementation

`api/bunpy/file.go` implements all three functions. `BunFile` is a
Python `Instance` whose `text`, `bytes`, `size`, and `exists`
attributes are `BuiltinFunc` closures over the path string.

`bunpy.write` accepts `str`, `bytes`, or a `BunFile` instance as the
data argument. When given a `BunFile`, it streams the source to the
destination with `io.Copy`.

All three functions are attached to the top-level `bunpy` module in
`BuildBunpy`.

## Notes

- `f.text()` always decodes as UTF-8. Invalid byte sequences produce
  Go replacement characters rather than raising an error. Use
  `f.bytes()` and decode in Python for strict UTF-8 validation.
- There is no streaming API in v0.3.3. Files are read fully into
  memory.
- `f.size` and `f.exists` are callable methods in this release, not
  properties. Python property descriptors will be wired in a later
  rung when goipy gains descriptor support.

# v0.3.2 -- bunpy.serve

Released 2026-04-27.

HTTP server built into bunpy. One function call starts a listening
server and returns a handle to stop or hot-reload it.

## What shipped

### `bunpy.serve(port=3000, handler=fn, hostname="localhost")`

```python
import bunpy

def handler(req):
    name = req.url.split("?name=")[-1] if "?" in req.url else "world"
    return Response(f"Hello, {name}!", 200)

server = bunpy.serve(port=3000, handler=handler)
print(f"Listening on {server.url}")   # "http://localhost:3000"
```

Returns a `Server` object:

| Attribute | Type | Description |
|---|---|---|
| `server.port` | int | actual bound port |
| `server.hostname` | str | bound hostname |
| `server.url` | str | `http://<hostname>:<port>` |
| `server.stop()` | -- | shut down, wait for in-flight requests |
| `server.reload(fn)` | -- | swap handler without restart |

### Request object passed to handler

```python
def handler(req):
    req.method       # str: "GET", "POST", ...
    req.url          # str: path + query string
    req.text()       # str: decoded body
    req.bytes()      # bytes: raw body
    req.headers      # Headers: lowercased header names
    return Response("ok")
```

The `Response` global (injected by v0.3.1) is returned directly from
the handler. Plain `str` and `bytes` returns also work and get status 200.

### Hot-reload without restart

```python
server = bunpy.serve(port=3000, handler=v1_handler)
# later, swap in new handler atomically
server.reload(v2_handler)
```

`reload` stores the new handler in an `atomic.Pointer`; requests in
flight complete against the old handler while new requests see the
new one immediately.

### Using port 0 for tests

Pass `port=0` to let the OS assign a free port. Read the actual port
back from `server.port`.

```python
server = bunpy.serve(port=0, handler=handler)
print(server.port)   # e.g. 54321
```

## Implementation

`api/bunpy/serve.go` implements everything. The server uses
`net.Listen("tcp", addr)` so the actual port is known before `serve()`
returns. Each request runs in a goroutine via `http.Server.Serve`.
The handler is called through `interp.Call()` (the public entry point
added in v0.3.1 for embedders).

`server.stop()` calls `httpSrv.Shutdown(context.Background())` and
waits for in-flight requests via a `sync.WaitGroup` before returning.

`BuildServe` is attached to the top-level `bunpy` module in `BuildBunpy`.

## Notes

- One `bunpy.serve()` call, one TCP listener. To serve on multiple
  ports, call `bunpy.serve()` multiple times.
- The server does not support TLS in v0.3.2. Terminate TLS at a
  reverse proxy (nginx, Caddy, etc.).
- There is no middleware API yet. Wrap the handler function in Python.
- Requests are read fully into memory before the handler is called.
  Streaming is not supported in v0.3.2.

# v0.3.1 -- fetch, URL, Request, Response globals

Released 2026-04-27.

Web-standard HTTP globals injected into every bunpy script with no
import required. Write `fetch("https://api.example.com")` directly.

## What shipped

### Global `fetch(url, **options) -> Response`

```python
resp = fetch("https://api.example.com/users/1")
user = resp.json()        # dict parsed from JSON
body = resp.text()        # str
raw  = resp.bytes()       # bytes
ok   = resp.ok            # bool: status 200-299
code = resp.status        # int

# POST with body and headers
resp = fetch("https://api.example.com/users",
             method="POST",
             body=b'{"name":"alice"}',
             headers={"Content-Type": "application/json"})
```

`fetch` blocks until the response body is fully read. There is no
streaming API in v0.3.1; the entire body is read into memory.

Options accepted as keyword arguments:

| Option | Type | Description |
|---|---|---|
| `method` | str | HTTP method (default `GET`) |
| `body` | bytes or str | request body |
| `headers` | dict | request headers |

### Global `URL(href)`

```python
u = URL("https://example.com:8080/path?q=hello#section")
u.protocol   # "https:"
u.hostname   # "example.com"
u.port       # "8080"
u.pathname   # "/path"
u.search     # "?q=hello"
u.hash       # "#section"
u.origin     # "https://example.com:8080"
u.href       # "https://example.com:8080/path?q=hello#section"
```

### Global `Request(url, **init)`

```python
req = Request("https://example.com", method="POST")
req.url     # str
req.method  # str (uppercase)
```

### Global `Response(body=None, status=200, **init)`

```python
resp = Response("hello world", 201)
resp = Response(b"binary data", status=200)
resp.status  # int
resp.ok      # bool
resp.text()  # str
resp.bytes() # bytes
resp.json()  # parsed JSON
```

### Global `Headers`

Returned by `Response.headers`. Supports `headers.get("content-type")`.
Header names are normalized to lowercase.

### Implementation

`api/bunpy/fetch.go` implements all five globals. `InjectGlobals(i)` is
called from `runtime/run.go` after creating the interpreter, placing
the five symbols directly into `i.Builtins`. The globals are also
exported as attributes of the `bunpy._fetch` internal module.

`fetch` uses `net/http.DefaultClient`, so the system HTTP proxy settings
and CA bundle apply. Connection keep-alive and redirect following are
handled by the default transport.

## Notes

- Streaming response bodies are not supported in v0.3.1. The full body
  is buffered before `resp.text()` or `resp.json()` is available.
- The `credentials`, `mode`, `cache`, `redirect`, and `integrity` fields
  of the Fetch spec are not implemented. Only `method`, `body`, and
  `headers` are supported.
- `fetch` is synchronous at the bunpy level. Scripts that need concurrent
  fetches should run them in threads (the stdlib `threading` module works).

# v0.3.0 -- NativeModules hook, bunpy.base64, bunpy.gzip

Released 2026-04-27.

v0.3.0 opens the v0.3.x series: built-in API modules that make common
tasks fast without reaching for third-party packages. This rung lays
the architectural foundation and ships two encoding utilities.

## What shipped

### NativeModules extensibility hook (goipy)

`goipy.Interp` gains a public `NativeModules` field:

```go
NativeModules map[string]func(*Interp) *object.Module
```

When set, `builtinModule` checks this map before the built-in switch.
Entries win over every stdlib module with the same name. Builders receive
the interpreter pointer so they can create typed errors, call back into
the VM, etc.

`runtime/run.go` pre-populates `NativeModules` with `bunpyAPI.Modules()`
before executing any user script. All `bunpy.*` modules registered there
become importable without any filesystem reads.

### `api/bunpy` package

New top-level `api/bunpy/` package. `Modules()` returns the
`NativeModules` map. Currently registers three entries:

| Module | Description |
|--------|-------------|
| `bunpy` | Top-level namespace, carries `__version__`, sub-module refs |
| `bunpy.base64` | `encode`, `decode`, `encode_url`, `decode_url` |
| `bunpy.gzip` | `compress`, `decompress` |

### bunpy.base64

```python
import bunpy.base64 as b64

encoded = b64.encode(b"hello world")          # str
decoded = b64.decode(encoded)                 # bytes

url_enc = b64.encode_url(b"\xff\xfe\xfd")    # URL-safe, no padding
url_dec = b64.decode_url(url_enc)            # bytes
```

Both `encode` and `encode_url` accept `bytes` or `str`. `decode` and
`decode_url` require `str`. Standard base64 `decode` tries with
padding first, then without, so unpadded input is accepted.

### bunpy.gzip

```python
import bunpy.gzip as gz

compressed = gz.compress(b"hello " * 1000)   # bytes
original   = gz.decompress(compressed)       # bytes

gz.compress(data, 9)   # optional level argument (0-9, -1 = default)
```

`compress` accepts an optional integer level (0 = no compression,
9 = max, -1 = gzip default). `decompress` rejects invalid gzip data
with a clear error message.

## Architecture notes

The `api/` top-level directory is the home for all future built-in
modules. Each sub-directory (`api/bunpy/`, later `api/fetch/`, etc.)
implements one module family. All return `*object.Module` builders;
the registry in `api/bunpy/bunpy.go:Modules()` grows with each rung.

## Fixes and housekeeping

- `pkg/pypi` user-agent bumped to `bunpy/0.3.0`.
- `docs/ROADMAP.md` v0.3.x table added with 13 planned rungs.

# v0.2.4 -- bunpyx

Released 2026-04-27.

`bunpyx` is a companion binary that runs a Python package entry point
in a temporary prefix. No venv, no permanent install. Fetch once,
run once, clean up.

## What shipped

### `bunpyx <pkg>[@version] [args...]`

```
bunpyx black .
bunpyx black@24.10.0 .
bunpyx --from black black --version
bunpyx ruff check .
```

If `@version` is omitted the latest version on PyPI is used. The wheel
is cached in `~/.cache/bunpyx/wheels/` so repeated runs are fast.

The exit code mirrors the invoked process. On Unix the process is
replaced via `syscall.Exec`; on Windows a child process is started and
its exit code is forwarded.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--from <module>` | | run `python -m <module>` instead of the console-script entry point |
| `--python <path>` | `python3` on PATH | Python executable |
| `--cache-dir <dir>` | `~/.cache/bunpyx/wheels` | wheel cache |
| `--no-cache` | | always fetch from PyPI |
| `--keep` | | keep the temp prefix; its path is printed to stderr |

### `pkg/runenv`

The `runenv` package implements the temp-prefix model:

- `Create(python)` makes a temp directory with `site-packages/` and
  `bin/` subdirectories.
- `Install(wheelPath)` unpacks the wheel zip into `site-packages/` and
  reads `*.dist-info/entry_points.txt`. Each `[console_scripts]` entry
  gets a shim written to `bin/`. On Unix shims are executable Python
  scripts; on Windows a `.cmd` wrapper calls a `.py` helper.
- `EntryPoint(name)` returns the shim path and `true` when found.
- `Cleanup()` removes the entire temp prefix.

### Manpage

`bunpyx(1)` documents the full flag set, execution model, cache
behavior, and the Windows note. It is embedded in the bunpy binary and
accessible via `bunpy man bunpyx`.

### Archive layout

Both binaries ship in every platform archive:

```
bunpy-linux-amd64.tar.gz
  bunpy-linux-amd64/
    bunpy
    bunpyx
    LICENSE
    README.md
```

## Fixes and housekeeping

- `pkg/pypi` user-agent bumped to `bunpy/0.2.4`.
- `internal/manpages.Page()` now falls back to an unprefixed filename
  so `bunpyx.1` is accessible via `Page("bunpyx")`.
- `docs/ROADMAP.md` v0.2.3 and v0.2.4 rows marked shipped.
- `docs/CLI.md` preamble updated to v0.2.4; `bunpyx` section added.
- v0.2.x ladder is now complete (v0.2.0 through v0.2.4).

# v0.2.3 -- create

Released 2026-04-27.

`bunpy create` scaffolds a new Python project from one of four built-in
templates. One command, a directory ready to code in.

## What shipped

### `bunpy create <template> <name>`

Four templates are available:

- **app** -- CLI application with a `src/` layout, a `__main__.py`
  entry point, and a `[project.scripts]` entry so `pip install -e .`
  drops the command on the path.
- **lib** -- library with `src/` layout and a `tests/` stub with a
  starter test file.
- **script** -- a single `.py` file with a `#!/usr/bin/env bunpy`
  shebang. No `pyproject.toml` needed.
- **workspace** -- root `pyproject.toml` with `[tool.bunpy.workspace]`
  and two member stubs under `packages/` ready for `bunpy workspace`.

Without `--yes`, an interactive prompt asks for description, author
name, and Python version constraint. Pressing Enter accepts the
bracketed default on each prompt.

```
$ bunpy create app my-cli
description [A CLI application]: A tool for doing things
author [Your Name]: Alice
python [>=3.11]: >=3.12
created my-cli/
```

`--yes` skips all prompts and uses the built-in defaults, which is
handy in scripts or CI.

```
$ bunpy create lib my-lib --yes
created my-lib/
```

`--list` prints the available templates and exits.

### `pkg/scaffold`

The scaffold package holds the four embedded templates under
`pkg/scaffold/templates/`. Files with a `.tmpl` suffix are passed
through `text/template` before writing; the suffix is stripped from
the output filename. Template variables available:

| Variable | Value |
|---|---|
| `.Name` | the project name as given |
| `.SnakeName` | name with `-` and `.` replaced by `_`, lowercased |
| `.Description` | short description string |
| `.Author` | author name (or `git config user.name` if available) |
| `.PythonMin` | Python version constraint, default `>=3.11` |

Directory names containing `{{.SnakeName}}` are expanded before the
directory is created, so `src/{{.SnakeName}}/` becomes
`src/my_lib/` for a project named `my-lib`.

The `//go:embed all:templates` directive is required (not plain
`//go:embed templates`) because Python files like `__init__.py` and
`__main__.py` start with `_`, which the standard Go embed excludes by
default.

### Manpage

`bunpy man create` renders `bunpy-create(1)`, which covers all four
templates, the prompt flow, and the `--yes` and `--list` flags.

## Fixes and housekeeping

- `pkg/pypi` user-agent bumped to `bunpy/0.2.3`.
- `docs/ROADMAP.md` row for v0.2.3 marked shipped.
- `docs/CLI.md` preamble updated to v0.2.3; create section added.

## [0.2.2] - 2026-04-27

`bunpy publish` builds a wheel and/or sdist via the project's PEP 517
build backend and uploads the artefacts to PyPI (or any compatible
registry). `--dry-run` builds without uploading, useful for CI
pre-flight checks.

### Added

- `pkg/build/build.go`: `Build(Request) (Result, error)` invokes PEP
  517 hooks (`build_sdist`, `build_wheel`) by running a short Python
  inline script via `os/exec`. `FindPython` searches PATH for python3
  or python. `ReadBackend` scans `pyproject.toml` for
  `[build-system].build-backend`; returns `hatchling.build` when the
  section is absent or the file is missing.
- `pkg/build/build_test.go`: six test cases: FindPython found,
  ReadBackend with hatchling, ReadBackend with no build-system
  (default), ReadBackend with missing file, Build with nonexistent
  Python, Build with nonexistent backend. The full Build output test
  is skipped unless Python and hatchling are available.
- `pkg/publish/publish.go`: `Upload(UploadRequest) ([]UploadResult, error)`
  builds a multipart form body per the legacy PyPI upload protocol,
  computes SHA-256, and POSTs with HTTP Basic auth (username
  `__token__`). `ErrAlreadyExists` and `ErrUnauthorized` are typed
  sentinel errors. `parseArtefact` extracts name/version/filetype from
  wheel filename (PEP 427) or `.tar.gz` filename; falls back to
  reading METADATA from the wheel zip.
- `pkg/publish/publish_test.go`: six test cases against an
  `httptest.Server`: multipart body, auth header, dry-run skips HTTP,
  403 returns ErrUnauthorized, 400 "already exists" returns
  ErrAlreadyExists, sha256_digest and name/version fields match.
- `cmd/bunpy/publish.go`: `publishSubcommand`. Flags: --sdist-only,
  --wheel-only, --dry-run, --registry, --token, --manifest. Token from
  `--token` or `PYPI_TOKEN`. Calls `build.Build` then
  `publish.Upload`.
- `internal/manpages/man1/bunpy-publish.1`: roff manpage with SYNOPSIS,
  DESCRIPTION (PEP 517, auth model), OPTIONS, ENVIRONMENT (PYPI_TOKEN),
  EXIT STATUS, EXAMPLES, SEE ALSO.
- `cmd/bunpy/main.go`: `case "publish":` branch. Unknown-command error
  message bumped to v0.2.2.
- `cmd/bunpy/help.go`: "publish" registry entry.

### Changed

- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.2.2`.
- `docs/CLI.md`: preamble bumped to v0.2.2; new `bunpy publish`
  section with flags and token resolution order.
- `docs/ARCHITECTURE.md`: paragraph on `pkg/build` (PEP 517 hooks,
  subprocess, FindPython, ReadBackend), `pkg/publish` (multipart
  upload, SHA-256, typed errors, dry-run).
- `docs/ROADMAP.md`: v0.2.2 row marked shipped.

### Notes

- The build backend must be installed before running `bunpy publish`.
  Run `bunpy install` first to materialise the build deps.
- `bunpy publish` is not workspace-aware in v0.2.2. Each workspace
  member has its own version and upload identity; publish them one at
  a time.
- TestPyPI uses the same upload protocol; pass
  `--registry https://test.pypi.org/legacy/` with a separate TestPyPI
  token.
- Real upload to PyPI is not exercised in CI to avoid polluting the
  package index. The upload path is covered by `pkg/publish` tests
  against a local httptest.Server.

## [0.2.1] - 2026-04-27

`bunpy audit` scans every pinned package in `bunpy.lock` against
the OSV (Open Source Vulnerabilities) database and reports known
vulnerabilities. Exit code 1 when any unfiltered vuln is found;
suitable as a CI gate. Workspace-aware: auto-detects the workspace
root and reads its lock.

### Added

- `pkg/audit/audit.go`: `OSVClient` with `QueryBatch` (chunks into
  1000-item batches as the OSV API requires), `Filter` (case-
  insensitive GHSA/CVE suppression), `SortFindings` (CRITICAL first,
  then by package name). Severity mapping: `database_specific.severity`
  wins; CVSS score fallback maps >= 9.0 to CRITICAL, >= 7.0 to HIGH,
  >= 4.0 to MEDIUM, >= 0.1 to LOW, missing to UNKNOWN. The `Finding`
  struct carries Package, Version, ID, Summary, Severity, and URL
  (canonical OSV page).
- `pkg/audit/audit_test.go`: six test cases against a local
  `httptest.Server`: vuln found, clean lock, 1500-pin chunking (two
  HTTP calls), Filter by GHSA, Filter by CVE (case-insensitive),
  severity parsing from CVSS scores.
- `cmd/bunpy/audit.go`: `auditSubcommand`. Three output modes: table
  (default), `--json`, `--quiet`. `--ignore` may be repeated. Exit
  code 0 on clean/suppressed, 1 on vulns, 2 on usage error.
  Workspace root auto-detected when `--lockfile` not given.
- `internal/manpages/man1/bunpy-audit.1`: roff manpage with SYNOPSIS,
  DESCRIPTION (severity mapping table), OPTIONS (--json, --quiet,
  --ignore, --lockfile, --workspace), EXIT STATUS, EXAMPLES, SEE ALSO.
- `cmd/bunpy/main.go`: `case "audit":` branch. Unknown-command error
  message bumped to v0.2.1.
- `cmd/bunpy/help.go`: "audit" registry entry.

### Changed

- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.2.1`.
- `docs/CLI.md`: preamble bumped to v0.2.1; new `bunpy audit` section.
- `docs/ARCHITECTURE.md`: paragraph on `pkg/audit`, OSVClient,
  querybatch chunking, severity mapping, Filter.
- `docs/ROADMAP.md`: v0.2.1 row marked shipped.

### Notes

- No API key is required for the OSV PyPI ecosystem endpoint. Rate
  limits apply to high-frequency automated callers but are not a
  concern for typical single-run CI invocations.
- `bunpy audit` always queries the live OSV database; there is no
  local advisory cache. Advisory databases update continuously and
  caching would introduce false negatives.
- Workspace audit unions pins from the workspace-root lock. When two
  members pin the same package at the same version, it is queried
  once (deduplication via the lock's unique-by-normalised-name
  invariant).
- `--ignore` IDs support both `GHSA-xxxx-yyyy-zzzz` and
  `CVE-yyyy-NNNNN` formats. Case-insensitive comparison handles
  mixed-case advisory IDs from different OSV sources.

## [0.2.0] - 2026-04-27

`bunpy workspace` opens the v0.2.x ladder by letting one
`pyproject.toml` declare a set of member projects under
`[tool.bunpy.workspace]`. A single `bunpy.lock` at the workspace root
covers all member dependencies. All existing verbs (install, add,
why, etc.) auto-detect the workspace root by walking up the directory
tree from cwd, so nothing breaks for single-project users.

### Added

- `pkg/workspace/workspace.go`: `Workspace`, `Member`, `Load`,
  `FindRoot`, `MemberByName`, `MemberByCwd`. `Load` reads the root
  `pyproject.toml`, expands globs in the `members` list via
  `filepath.Glob`, rejects duplicate member names and paths outside
  the root. `FindRoot` walks up the directory tree looking for a
  `pyproject.toml` with `[tool.bunpy.workspace]`; returns
  `ErrNoWorkspace` when none is found.
- `pkg/workspace/workspace_test.go`: six test cases covering three-
  member load, glob expansion, nested FindRoot, FindRoot not found,
  duplicate name rejection, and MemberByCwd selection.
- `cmd/bunpy/workspace.go`: `workspaceSubcommand` (serves `bunpy
  workspace --list`), `findWorkspaceRoot` (cmd-layer wrapper that
  returns `""` instead of `ErrNoWorkspace` so callers fall through
  to single-project mode), `loadWorkspace` / `wsHandle.findMember`
  (helpers for add.go member lookup).
- `internal/manpages/man1/bunpy-workspace.1`: roff manpage with
  SYNOPSIS, DESCRIPTION (workspace concept, glob members, shared
  lock), OPTIONS (--list, --workspace), EXAMPLES, EXIT STATUS, SEE
  ALSO. Picked up automatically by the `//go:embed man1/*.1` in
  `internal/manpages`.
- Tests: `cmd/bunpy/workspace.go` is covered by the CI smoke step.

### Changed

- `pkg/manifest/manifest.go`: `Tool` gains a `Workspace
  *WorkspaceConfig` field populated from `[tool.bunpy.workspace]`
  during TOML parsing. `WorkspaceConfig` carries a `Members []string`
  slice (raw patterns, before glob expansion).
- `pkg/lockfile/lockfile.go`: `Lock` gains an optional `Workspace
  *WorkspaceMeta` field. `Parse` recognises the `[workspace]`
  section and its `members` key. `Bytes` emits the section when
  non-nil. Single-project locks have no `[workspace]` section and
  round-trip byte-identically.
- `cmd/bunpy/install.go`: before reading the lockfile, calls
  `findWorkspaceRoot(cwd)`. If a workspace root is found (or
  `--workspace <root>` is given), the lock and manifest paths are
  resolved relative to that root. Single-project installs are
  unaffected.
- `cmd/bunpy/add.go`: `--member <name>` targets a specific workspace
  member's `pyproject.toml`. Without `--member`, `add` continues to
  write the manifest in the current directory. Lock is written to the
  workspace root when a workspace is detected.
- `cmd/bunpy/main.go`: `case "workspace":` router branch. Unknown-
  command error message bumped to v0.2.0.
- `cmd/bunpy/help.go`: "workspace" registry entry covering --list,
  auto-detection, --member on add, and the shared lock model.
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.2.0`.
- `docs/CLI.md`: preamble bumped to v0.2.0; new `bunpy workspace`
  section; aspirational list updated for v0.2.x ladder.
- `docs/ARCHITECTURE.md`: paragraph on `pkg/workspace`, glob
  expansion, `FindRoot`, lockfile `[workspace]` section,
  `manifest.Tool.Workspace`, and the workspace-aware install/add
  paths.
- `docs/ROADMAP.md`: v0.2.0 row marked shipped.

### Notes

- Single-project `pyproject.toml` and `bunpy.lock` files are
  unchanged. The `[workspace]` section is only written when a
  workspace-aware operation produces the lock; no migration needed.
- `bunpy install` inside a member directory reads the workspace
  root's `bunpy.lock`. The lock must already exist (run `bunpy pm
  lock` or `bunpy add` from the root to produce it).
- Glob expansion happens at load time. A pattern that matches zero
  directories is a hard error so silent misconfiguration is caught
  early rather than producing an empty workspace.
- `bunpy add --member <name>` updates the named member's manifest
  and re-runs the lock at the workspace root. Other verbs (update,
  outdated, remove, why) accept `--workspace <root>` to point at the
  root explicitly; auto-detection works from any directory inside the
  workspace tree.

## [0.1.11] - 2026-04-27

`bunpy why <pkg>` answers the "why is this package in my
lockfile?" question by walking the reverse-dependency graph from
a pinned package back up to the project's direct requirements.
Each chain terminates at a virtual `@project` edge tagged with
the lane (`main`, `dev`, `optional:<group>`, `group:<name>`,
`peer`) the requirement was declared in. The closing rung of
v0.1.x: after this lands the package manager has a resolver,
lockfile, lanes, install/update/outdated/remove, link, patch,
and why.

### Added

- `pkg/why/why.go`: graph builder and walker. `BuildGraph(lf,
  mf, fetch)` resolves the forward dependency graph by calling
  `fetch(name, version)` for every pin in `bunpy.lock` and
  inverts it. The fetch interface (`RequiresFunc`) returns the
  marker-evaluated Requires-Dist edges for one pin so the walker
  stays decoupled from the wheel cache. `Walk(g, name, depth)`
  is a depth-first enumeration of every path from the queried
  pin upward, terminating at `@project` edges; cycles are
  guarded by per-path visited sets and the result is sorted for
  stable output. `Result` carries `Linked`, `Patched`, and an
  `Installer` label (`bunpy`, `bunpy-link`, `bunpy-patch`)
  drawn from the manifest's `[tool.bunpy.links]` and
  `[tool.bunpy.patches]` tables.
- `cmd/bunpy/why.go`: subcommand router. Reads
  `pyproject.toml` and `bunpy.lock`, builds a wheel-cache-backed
  `RequiresFunc` (uses `wheel.LoadMetadata` plus
  `wheel.ParseMetadata`, evaluates markers against
  `marker.DefaultEnv`), runs `why.Walk`, and prints. Three
  output modes: tree (default, indent shape with parent edges
  labelled `name version [specifier]`), `--top` (dedup of direct
  project requirements, one per line), and `--json` (full
  `Result` shape, stable schema). `--depth <N>` caps traversal,
  `--lane <name>` restricts to chains ending in one lane,
  `--cache-dir`, `--manifest`, `--lockfile` override the input
  paths.
- `cmd/bunpy/main.go`: `case "why":` router branch, and the
  unknown-command error message bumps to v0.1.11.
- `cmd/bunpy/help.go`: `why` registry entry covering tree shape,
  `--top`, `--json`, `--depth`, `--lane`, and the link/patch
  overlay surface. `bunpy help why` and `bunpy why --help` share
  the same body.
- `internal/manpages/man1/bunpy-why.1`: roff manpage with
  SYNOPSIS, OPTIONS (depth, top, json, lane, cache-dir,
  manifest, lockfile), EXAMPLES, EXIT STATUS, and SEE ALSO.
  Embeds via the `manpages` package and ships in `bunpy man
  --install`.
- Tests: `pkg/why/why_test.go` adds six cases (leaf chain to
  `@project`, diamond produces two chains, depth cap truncates,
  cycle does not loop, overlay state surfaces patched/linked
  flags, missing pin returns `ErrNotFound`).
  `cmd/bunpy/why_test.go` adds six cases (tree shape, `--top`
  dedup, `--json` shape, `--depth 1` truncates, missing pin
  exits non-zero with named error, `--lane main` filters to one
  chain). The cmd tests synthesise tiny zip-only METADATA-bearing
  wheels into a per-test cache so the Requires-Dist scan runs
  against real bytes.
- `.github/workflows/ci.yml`: one new smoke step (`bunpy why
  smoke`) that pre-stages a synthesised wheel cache, writes
  `pyproject.toml` and `bunpy.lock`, runs `bunpy why gamma`,
  asserts the chain through `alpha`, runs `bunpy why gamma
  --top`, asserts the dedup, runs `bunpy why ghost`, asserts
  the non-zero exit. The help/--help parity loop grows a `why`
  entry.

### Changed

- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.11`.
- `docs/CLI.md`: new section covering `bunpy why`; the
  wired-surface preamble bumps to v0.1.11; the aspirational
  list drops the why entry and the surrounding paragraph
  re-aims at the v0.2.x ladder.
- `docs/ARCHITECTURE.md`: paragraph on the `pkg/why` graph
  builder, the wheel-cache-backed `RequiresFunc`, the
  depth-first walker, and the overlay-aware `Result` type. The
  lockfile schema is unchanged: edges are derived, not stored.
- `docs/ROADMAP.md`: v0.1.11 marked shipped; the v0.1.x rung
  ladder is now complete.

### Notes

- Lockfile schema stays at version 1. `bunpy why` does not
  store reverse edges anywhere: it derives the forward graph
  from each cached wheel's METADATA at query time. A pin whose
  wheel is not in the cache contributes no edges and so its
  reverse-deps will be invisible until the wheel is fetched
  (run `bunpy install` to fix).
- Marker evaluation uses `marker.DefaultEnv` (the host
  environment), so the reverse-deps surface what would actually
  install on this box. This matches `bunpy install` semantics
  and is the most useful default; cross-platform reasoning
  remains a v0.2.x concern.
- The `@project` sentinel keeps every chain closed at the same
  node. Without it the tree would have ragged tops and `--top`
  would need a special case for direct-req pins.
- Direct-dep lane membership is derived from the manifest, not
  from the per-pin `lanes` array in `bunpy.lock`. The lockfile's
  `lanes` records install-time lane membership for the install
  loop; the manifest is authoritative for "which lane declared
  this requirement". This split keeps the lockfile a resolver
  output and the manifest the build-input source of truth.
- v0.1.x is now complete: every rung in the ladder has shipped.
  The next ladder begins with v0.2.0 (workspaces), then audit,
  publish, create, and bunpyx through v0.2.x.

## [0.1.10] - 2026-04-27

`bunpy patch <pkg>` opens a mutable copy of an installed
package, lets the user edit it, then captures the diff as a
versioned patch artefact under `./patches/`. The patch is
registered in `pyproject.toml` under `[tool.bunpy.patches]` and
re-applied by `bunpy install` on every fresh install of the same
pin. The applier is strict (no fuzz, no offset slack) so patches
fail loudly when a pin moves underneath them, prompting an
explicit refresh.

### Added

- `pkg/patches/patches.go`: the diff and apply primitives.
  `Diff(pristine, scratch)` walks both trees, skips files
  identical on both sides, refuses binary files (any 0x00 byte
  in the first 4 KiB), and emits one whole-file unified-diff
  hunk per changed file. `Apply(target, body)` parses the same
  shape and rewrites the target in place, rejecting any hunk
  whose `-` context does not match byte-for-byte.
  `Read(manifest)` returns every entry in
  `[tool.bunpy.patches]`; `Lookup(entries, name, version)`
  finds an entry by PEP 503-normalised name.
  `InstallerTag = "bunpy-patch"` is the durable signal a
  patched install carries on its dist-info.
- `pkg/patches/extract.go`: `Extract(wheelPath, dest)` lays a
  pristine wheel out into `dest`, skipping `RECORD` and
  `INSTALLER` (those are install-time artefacts). `CopyTree`
  duplicates the pristine into the scratch root.
  `PristineRoot` and `ScratchRoot` are the canonical layout
  helpers (`<target>/../patches/.pristine/<name>-<version>` and
  `<target>/../patches/.scratch/<name>-<version>`).
- `pkg/manifest/patches.go`: `AddPatchEntry(key, value)` and
  `RemovePatchEntry(key)`, the text mutators for
  `[tool.bunpy.patches]`. The table is created on demand;
  rows are emitted sorted by key for stable diffs; missing-key
  removal is a no-op.
- `cmd/bunpy/patch.go`: `bunpy patch <pkg>` (open scratch,
  print path), `bunpy patch --commit <pkg>` (diff, persist,
  edit manifest), and `bunpy patch --list`. Flags: `--commit`,
  `--list`, `--out <path>`, `--no-write`, `--target <dir>`,
  `--cache-dir <path>`, `--print-only`. The open path checks
  for `INSTALLER=bunpy-link` and refuses with a clear error
  message: linked packages cannot be patched.
- `cmd/bunpy/main.go`: `case "patch":` router branch, and the
  unknown-command error message bumps to v0.1.10.
- `cmd/bunpy/help.go`: `patch` registry entry covering open,
  commit, list, the install-time apply path, and the
  precedence with `bunpy link`. `bunpy help patch` and
  `bunpy patch --help` share the same body.
- `internal/manpages/man1/bunpy-patch.1`: roff manpage with
  SYNOPSIS, OPTIONS, FILES (pristine, scratch, registered
  patch), and EXIT STATUS sections. Embeds via the `manpages`
  package and ships in `bunpy man --install`.
- Tests: `pkg/patches/patches_test.go` adds eight cases
  (modify, add, remove round-trip; context mismatch refusal;
  binary refusal; path-escape refusal; manifest read; empty
  diff). `pkg/manifest/patches_test.go` adds six cases (create
  table, append, replace, no-op, remove, missing remove).
  `cmd/bunpy/patch_test.go` adds six cases (open creates
  scratch, open is idempotent, commit writes patch + manifest,
  commit no-op when unchanged, --list, requires package name).
  `cmd/bunpy/install_patch_test.go` adds three cases (install
  applies registered patch and stamps `bunpy-patch`,
  `--no-patches` opts out, stale patch fails the install).
- `.github/workflows/ci.yml`: one new smoke step
  (`bunpy patch smoke`) that pre-stages the v0.1.3 widget
  wheel cache, runs `bunpy patch widget`, edits the scratch,
  commits, asserts the patch file and the manifest entry, then
  re-runs `bunpy install` from a clean target and asserts the
  patched bytes show up plus `INSTALLER=bunpy-patch`. The
  help/--help parity loop grows a `patch` entry.

### Changed

- `cmd/bunpy/install.go`: the install loop reads
  `[tool.bunpy.patches]` once at start, then applies the
  matching patch in place after every successful wheel install.
  A patched install prints `patched <name> <version>` (vs the
  default `installed <name> <version>`) and the dist-info
  `INSTALLER` is rewritten to `bunpy-patch`. New flag
  `--no-patches` opts out for emergency recovery.
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.10`.
- `docs/CLI.md`: new section covering `bunpy patch`; the
  wired-surface preamble bumps to v0.1.10; the aspirational
  list drops the patch entry.
- `docs/ARCHITECTURE.md`: lockfile section grows a paragraph on
  the patch path: `pkg/patches` diff/apply, the
  `[tool.bunpy.patches]` table, the install-time hook, the
  `INSTALLER=bunpy-patch` sentinel, and the precedence with
  `bunpy link`.
- `docs/ROADMAP.md`: v0.1.10 marked shipped.

### Notes

- Lockfile schema stays at version 1. Patch state lives in
  `pyproject.toml` because it is build-input metadata, not a
  resolver output.
- The diff format is whole-file hunks, not minimal-context.
  This keeps the diff/apply pair simple and the apply path
  strictly reproducible: there is no LCS dependency, no fuzz,
  no offset search. A future v0.2.x pass can shrink the patch
  size with proper minimal-context hunks once a real LCS
  implementation lands.
- Files without trailing newlines are refused at `Diff` time.
  This is a deliberate simplification; the `\ No newline at end
  of file` marker can land alongside the LCS work.
- Linked packages take precedence over patched packages.
  `bunpy install` skips linked pins entirely (the
  `INSTALLER=bunpy-link` opt-out from v0.1.9), so a user who
  has both a link and a patch on the same package keeps
  iterating in the source tree.
- `bunpy why <pkg>` (v0.1.11) will surface patch status next to
  reverse-deps so a user can spot patched pins at a glance.

## [0.1.9] - 2026-04-27

The Bun-style editable-install pair lands. `bunpy link` from a
package source registers the project in a global registry under
`$BUNPY_LINK_DIR` (default: the platform user-data dir). `bunpy
link <pkg>` from a consumer drops a PEP 660-style proxy into
`./.bunpy/site-packages`: a `.pth` file holding the absolute
source path plus a `<name>-<version>.dist-info` directory tagged
`INSTALLER=bunpy-link`. `bunpy install` recognises the tag and
keeps linked packages in place across re-installs. `bunpy unlink`
is the inverse: bare `unlink` deletes the registry entry, named
`unlink <pkg>` walks the proxy's RECORD and removes every listed
file.

### Added

- `pkg/links/links.go`: the global link registry. `Entry` is a
  small JSON struct (`name`, `version`, `source`,
  `registered`); one file per package as `<name>.json`. `Dir()`
  resolves `$BUNPY_LINK_DIR` first, then the platform user-data
  dir (`$XDG_DATA_HOME/bunpy/links` on Linux,
  `~/Library/Application Support/bunpy/links` on macOS,
  `%LOCALAPPDATA%/bunpy/links` on Windows). `Read` returns a
  typed `ErrNotFound` so callers can distinguish "no entry" from
  "broken JSON". Writes are atomic (tempfile + rename). `List`
  returns entries sorted by name; a missing dir is not an error.
- `pkg/editable/editable.go`: the consumer-side editable proxy.
  `Install(spec, target)` lays down `<name>.pth` (one line, the
  absolute source path), and a `<name>-<version>.dist-info`
  directory with `METADATA`, `RECORD`, `INSTALLER=bunpy-link`,
  and `direct_url.json` (PEP 610, `dir_info.editable=true`).
  `Uninstall(name, version, target)` walks RECORD with the same
  path-escape guard `bunpy remove` uses, removes every listed
  file, and drops the dist-info. The `InstallerTag` constant
  (`bunpy-link`) is the opt-out signal for `bunpy install`.
- `cmd/bunpy/link.go`: `bunpy link [pkg]...` with `--list` and
  `--target`. Bare `bunpy link` reads `./pyproject.toml`, picks
  up the project name and version, resolves the source path via
  `filepath.Abs` + `filepath.EvalSymlinks` (so macOS
  `/var/folders -> /private/var/folders` does not break a later
  string compare), and writes the registry entry. `bunpy link
  <pkg>...` looks up each name in the registry and lays down the
  proxy via `pkg/editable`. `--list` prints the registry as a
  sorted table. Errors when the registry has no entry for a
  named package.
- `cmd/bunpy/unlink.go`: `bunpy unlink [pkg]...` with `--target`.
  Bare `bunpy unlink` deletes the registry entry for the current
  project (idempotent: a missing entry prints `unregistered
  <name>` and exits 0). `bunpy unlink <pkg>...` walks each
  proxy's RECORD via `pkg/editable.Uninstall` and prints `no
  link for <name>` when the proxy is absent.
- `cmd/bunpy/main.go`: `case "link":` and `case "unlink":`
  router branches; the unknown-command error message bumps to
  v0.1.9.
- `cmd/bunpy/help.go`: two new registry entries (`link`,
  `unlink`) covering USAGE, the registry shape, the
  PEP 660 proxy layout, the `INSTALLER=bunpy-link` opt-out
  semantic, and the consumer-side cleanup. `bunpy help <verb>`
  and `bunpy <verb> --help` share the same body.
- `internal/manpages/man1/bunpy-link.1` and `bunpy-unlink.1`:
  roff manpages with SYNOPSIS, OPTIONS, ENVIRONMENT
  (`BUNPY_LINK_DIR`), and EXIT STATUS sections. Both ship in
  `bunpy man --install`.
- Tests: `pkg/links/links_test.go` adds eight cases (Dir env
  override, atomic Write, Read missing, Delete idempotent, List
  sorted, ...). `pkg/editable/editable_test.go` adds six cases
  (Install lays down .pth + dist-info, RECORD content, Uninstall
  walks RECORD, path-escape rejection, ...). `cmd/bunpy/link_test.go`
  adds five cases (registers current, unknown package errors,
  installs editable proxy, --list prints registry, requires
  project name). `cmd/bunpy/unlink_test.go` adds four cases
  (unregisters current, removes editable proxy, missing package
  noop, unknown project noop).
- `.github/workflows/ci.yml`: one new smoke step (`link`) that
  registers a temp source pkg, links from a temp consumer,
  asserts the `.pth` content matches `pwd -P`, checks
  `INSTALLER=bunpy-link`, exercises `--list`, then unlinks
  consumer-side and unregisters. The help/--help parity loop
  grows two entries.

### Changed

- `cmd/bunpy/install.go`: the install loop reads each pin's
  installed dist-info before re-installing. When `INSTALLER` is
  `bunpy-link` the wheel install is skipped and `kept linked
  <name> <version>` is printed. Tries the PEP 503-normalised
  name first, then the verbatim name, so packages with
  underscore variants are still recognised.
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.9`.
- `docs/CLI.md`: new section covering `bunpy link` and
  `bunpy unlink`; the wired-surface preamble bumps to v0.1.9;
  the aspirational list drops both verbs.
- `docs/ARCHITECTURE.md`: lockfile section grows a paragraph on
  the link registry (`pkg/links`), the PEP 660 proxy
  (`pkg/editable`), and the `INSTALLER=bunpy-link` opt-out path
  in `bunpy install`.
- `docs/ROADMAP.md`: v0.1.9 marked shipped.

### Notes

- `bunpy.lock` schema stays at version 1. Editable installs are
  consumer-side only and never touch the lockfile.
- The link registry is global per user, not per project. A
  package linked from one source can be consumed from any number
  of consumers concurrently.
- `INSTALLER=bunpy-link` is the durable opt-out signal. Even if
  a consumer re-runs `bunpy install` against an unrelated lock,
  the linked package is preserved as long as the dist-info tag
  is intact. To switch back to the registry wheel, run
  `bunpy unlink <pkg>` first; `bunpy install` then re-fetches
  the pinned wheel.
- The `.pth` strategy is portable: both CPython and goipy scan
  `*.pth` files in site-packages at startup. No import hooks,
  no per-runtime shims.
- `bunpy unlink` in the source is non-destructive for consumers.
  Existing proxies keep working until the consumer re-installs
  or re-links; the registry entry is just an editable lookup
  table.

## [0.1.8] - 2026-04-27

The package manager learns to subtract. `bunpy remove <pkg>...`
is the symmetric inverse of `bunpy add`: it deletes the named
packages from `pyproject.toml` (every lane unless a lane flag
narrows it), re-runs the resolver against the new lane map with
surviving pins held via `Solver.Locked`, rewrites `bunpy.lock`,
and uninstalls the dropped pins from `./.bunpy/site-packages`
unless `--no-install` is passed. Removing a name that is not in
the manifest is not an error: the verb is idempotent.

### Added

- `pkg/manifest/remove_lane.go`: five public mutators
  (`RemoveDependency`, `RemoveOptionalDependency`,
  `RemoveGroupDependency`, `RemovePeerDependency`,
  `RemoveDependencyAllLanes`) plus the `removeFromArray`
  helper. Each returns `(out []byte, n int, err)` so callers
  can re-Parse only when bytes moved. PEP 503 normalisation
  matches `Foo_Bar` and `foo-bar` even though only one form
  appears in the source. Removing the last entry in a multiline
  array preserves the array shape (`dependencies = [\n]`) so
  the diff stays small.
- `cmd/bunpy/remove.go`: `bunpy remove [pkg]...` with lane
  flags `-D`/`--dev` (with `--group <name>` for non-default
  PEP 735 groups), `-O`/`--optional <group>` for PEP 621
  optional groups, `-P`/`--peer` for `[tool.bunpy].peer-dependencies`,
  `--no-install` to skip the site-packages prune, `--no-write`
  to skip the manifest edit, plus `--target`, `--index`. Lane
  flags are mutually exclusive; `--group` requires `-D`.
- `cmd/bunpy/main.go`: `case "remove":` router branch and
  unknown-command error message bumped to v0.1.8.
- `cmd/bunpy/help.go`: `remove` registry entry covering USAGE,
  lane flags, the idempotent semantics, and the uninstall path.
  `bunpy help remove` and `bunpy remove --help` share the same
  body.
- `internal/manpages/man1/bunpy-remove.1`: roff manpage with
  SYNOPSIS, lane flags, OUTPUT, UNINSTALL (RECORD walk +
  best-effort fallback), and EXIT STATUS sections. Embeds via
  the `manpages` package and ships in `bunpy man --install`.
- Tests: `pkg/manifest/remove_lane_test.go` adds six cases
  (only-entry, middle-entry, normalised-name, missing-noop,
  group, all-lanes). `cmd/bunpy/remove_test.go` lands with six
  cases (bare drop from all lanes, dev-only restrict, missing
  package noop, lane-flag mutex, bare-remove-without-pkg
  errors, `--no-install` skips site-packages).
- Fixtures: `tests/fixtures/v018/remove.{input.toml,seed.lock,lock_in}`
  plus `expected_remove.lock`. The harness walks the new
  fixture via the existing `run_lock_fixture` helper (the
  `<name>.seed.lock` convention from v0.1.7 is reused).
- `.github/workflows/ci.yml`: one new smoke step (`remove`)
  that drives an end-to-end run against the v0.1.3 widget
  fixture and exercises the bare-remove path, the dev-only
  restrict path, and the lane-flag mutex check. The
  help/--help parity loop grows a `remove` entry.

### Changed

- `cmd/bunpy/main.go`: unknown-command error message bumps to
  v0.1.8 and lists the new verb.
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.8`.
- `docs/CLI.md`: new section (`bunpy remove`); the wired-surface
  preamble bumps to v0.1.8; the aspirational list drops
  `bunpy remove`.
- `docs/ARCHITECTURE.md`: lockfile section grows a paragraph on
  the symmetric remove path: per-lane manifest mutators,
  `RemoveDependencyAllLanes`, the resolver re-run with
  `Solver.Locked`, and the RECORD-driven uninstall.
- `docs/ROADMAP.md`: v0.1.8 marked shipped.

### Notes

- `bunpy.lock` schema stays at version 1. `bunpy remove` reads
  and writes the same shape v0.1.7 emits.
- The verb is idempotent. `bunpy remove notapkg` exits 0,
  prints `removed 0 packages`, and leaves both `pyproject.toml`
  and `bunpy.lock` byte-identical.
- The uninstall path walks the dropped pin's
  `<name>-<version>.dist-info/RECORD`, rejecting any entry
  whose cleaned path escapes the target. Wheels without a
  RECORD fall back to a best-effort `<name>/` directory
  removal under the target.
- Empty lane tables are kept after a delete. Dropping the
  last entry from a `[dependency-groups]` group leaves the
  table header in place; collapsing it is an explicit user
  action.
- A future `bunpy why <pkg>` (v0.1.11) will explain transitive
  drops; for now `bunpy remove` prints one line per dropped
  pin (` - <name> <version>`) so the diff against the lockfile
  is visible.

## [0.1.7] - 2026-04-27

The package manager grows two drift-control verbs. `bunpy outdated`
walks the lockfile, fetches each pin's PEP 691 page through the
same client `pm info` uses, and prints a four-column table with
`current` / `wanted` (highest version satisfying the manifest spec)
/ `latest` (highest non-yanked release) / `lanes`. `bunpy update`
re-runs the v0.1.5 resolver with a new `Solver.Locked` map seeded
from the existing lockfile, picks fresh versions for the named
packages (or every package on a bare `update`), rewrites
`bunpy.lock`, and refreshes `./.bunpy/site-packages/` unless
`--no-install` is set. `--latest <pkg>...` strips the manifest
spec for the named packages and lets the resolver pick the highest
non-prerelease wheel.

### Added

- `pkg/resolver/solve.go`: `Solver.Locked map[string]string`. The
  candidate-pick step prefers the locked version when the combined
  constraint allows it and the registry still lists it; otherwise
  it falls back to `version.Highest`.
- `cmd/bunpy/outdated.go`: `bunpy outdated [pkg]...` with lane
  filters mirroring `bunpy install` (`-D`, `-O <group>`,
  `--all-extras`, `-P`, `--production`), `--json`, and
  `--index`/`--cache-dir` overrides. Read-only: never writes the
  manifest, the lockfile, or `site-packages`.
- `cmd/bunpy/update.go`: `bunpy update [pkg]...` with `--latest`,
  `--no-install`, `--no-verify`, and the same lane / target /
  index / cache-dir flags `bunpy install` exposes. A bare
  `bunpy update` clears every lock and re-resolves the whole
  graph; positional package names drop only those entries from
  the lock hint. `--latest` requires at least one positional
  argument.
- `cmd/bunpy/main.go`: router cases for `outdated` and `update`,
  with the unknown-command error message bumped to v0.1.7.
- `cmd/bunpy/help.go`: `outdated` and `update` registry entries
  covering USAGE, lane flags, `--json` (outdated), and
  `--latest <pkg>...` (update). `bunpy help outdated` and
  `bunpy <verb> --help` share the same body.
- `internal/manpages/man1/bunpy-outdated.1` and
  `bunpy-update.1`: roff manpages for the two new verbs. Both
  embed via the `manpages` package and ship in
  `bunpy man --install`.
- Tests: `pkg/resolver/resolver_test.go` adds
  `TestSolveRespectsLocked` and `TestSolveOverridesLockedWhenSpecForbids`.
  `cmd/bunpy/outdated_test.go` lands with four cases (newer pin,
  silent on up-to-date, JSON shape, lane filter excludes a dev
  pin). `cmd/bunpy/update_test.go` lands with five cases (bare
  upgrades within spec, no-changes, `--latest` ignores spec,
  `--latest` without args errors, `--no-install` skips
  site-packages).
- Fixtures: `tests/fixtures/v017/update.{input.toml,seed.lock,lock_in}`
  plus `expected_update.lock`. The harness gains an optional
  `<name>.seed.lock` convention so verbs that read an existing
  lockfile (here, `update`) can be exercised end-to-end.
- `.github/workflows/ci.yml`: three new smoke steps
  (`outdated`, `update`, `update --latest`) that drive
  end-to-end runs of the new verbs against the v0.1.3 widget
  fixture. The help/--help parity loop grows `outdated` and
  `update` entries.

### Changed

- `cmd/bunpy/main.go`: unknown-command error message bumps to
  v0.1.7 and lists the two new verbs.
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.7`.
- `docs/CLI.md`: two new sections (`bunpy outdated`,
  `bunpy update`); the wired-surface preamble bumps to v0.1.7;
  the aspirational list drops both verbs.
- `docs/ARCHITECTURE.md`: lockfile section grows a paragraph on
  `Solver.Locked`, the bare-update clear, and the
  spec-stripping path used by `--latest`.
- `docs/ROADMAP.md`: v0.1.7 marked shipped.
- `tests/run.sh`: the `run_lock_fixture` helper copies an optional
  `<name>.seed.lock` into the work dir before running the args,
  so a fixture can seed an initial lockfile for verbs that read
  one.

### Notes

- `bunpy.lock` schema stays at version 1. The new verbs read and
  write the same shape v0.1.6 emits.
- A bare `bunpy update` clears every entry from `Solver.Locked`,
  matching the npm semantics for `npm update`. Naming packages
  drops only those entries; everything else is held at the
  current pin so `update <pkg>` does not move peers.
- `--latest` strips the manifest spec for the named packages
  before resolving. The on-disk manifest is not edited; if the
  new lower bound should persist, follow up with `bunpy add`.
  The CLI refuses `--latest` without positional args to avoid
  surprise mass upgrades.
- `bunpy outdated` exits 0 whether or not anything is outdated;
  it is informational, not a check. Pipe `--json` into a script
  to gate on the result.
- Yanked and pre-release versions are still skipped in the
  candidate list. Escape hatches (`--include-yanked`,
  pre-release handling) land in a later v0.1.x rung.

## [0.1.6] - 2026-04-27

The package manager grows dependency lanes. v0.1.6 teaches
`bunpy add`, `bunpy pm lock`, and `bunpy install` to track which
manifest table a spec came from (main, dev, optional groups, peer)
and which lanes a transitive pin was pulled in by. The resolver
runs once over the union of every lane and post-processes the
registry to compute per-lane closures, so a workspace with
overlapping dev and main deps still resolves a single coherent set
of pins. Bun-style flags (`-D`, `-O <group>`, `-P`, `--all-extras`,
`--production`) drive both the manifest edit and the install
filter.

### Added

- `pkg/manifest/manifest.go`: `Manifest.DependencyGroups`
  (PEP 735 `[dependency-groups]`), `Tool.PeerDependencies`
  (`[tool.bunpy].peer-dependencies`). Validation rejects
  PEP 685-illegal group names and a name that appears in both
  `[project.optional-dependencies]` and `[dependency-groups]`.
- `pkg/manifest/add_lane.go`: byte-stable mutators
  `AddOptionalDependency(group, spec)`,
  `AddGroupDependency(group, spec)`, `AddPeerDependency(spec)`.
  Each one creates the target table if needed, reuses an existing
  array key, and replaces a duplicate spec by normalised name to
  match `AddDependency` semantics.
- `pkg/lockfile/lockfile.go`: `Package.Lanes` (omitempty), lane
  constants `LaneMain`/`LaneDev`/`LanePeer`, helpers
  `OptionalLane(group)` and `GroupLane(name)`, content-hash
  builder `HashLanes(map[string][]string)` that mirrors
  `HashDependencies` byte-identically when only `main` is
  populated.
- `cmd/bunpy/add.go`: `-D`/`--dev`, `-O`/`--optional <group>`,
  `-P`/`--peer`, `--group <name>` flag wiring. Lane-aware lockfile
  upsert merges existing lanes with the caller's choice so a
  package shared across main and dev carries both. The lockfile
  content-hash is recomputed from every lane via `HashLanes`.
- `cmd/bunpy/pm.go`: `pm lock` resolves the union of every lane
  in one solver pass, then walks `reg.Dependencies` for each
  lane's roots to compute per-pin lane membership via BFS.
  `--check` covers every lane: a direct dep in any lane with no
  lockfile entry triggers drift.
- `cmd/bunpy/install.go`: `-D`, `-O <group>`, `--all-extras`,
  `-P`, `--production` filter the install by per-pin lane tag.
  The default keeps only `main`; pins without a `lanes` field are
  treated as `main` so v0.1.5 lockfiles install unchanged. A
  trailing `skipped N package(s) outside the selected lanes`
  message reports the filter.
- Tests: `pkg/manifest/manifest_test.go` grows five lane cases
  (dependency-groups, peer-dependencies, table-shape rejection,
  duplicate group rejection, bad group name rejection).
  `pkg/manifest/add_lane_test.go` covers create-table,
  replace-existing, dependency-groups dev, append-to-tool-bunpy,
  create-tool-bunpy-table, reject-bad-group.
  `pkg/lockfile/lockfile_test.go` adds six cases for parse, omit,
  emit, and `HashLanes` byte-equivalence with `HashDependencies`
  for main-only inputs. `cmd/bunpy/add_test.go` adds six lane
  cases (`-D`, `-D --group`, `-O`, `-P`, mutual exclusion,
  `--group` without `-D`). `cmd/bunpy/install_test.go` lands with
  seven lane cases. `cmd/bunpy/pm_test.go` adds three lane cases
  (cross-table tagging, main-only omission, optional-group drift).
- Fixtures: `tests/fixtures/v016/{dev,extras,peer}.input.toml`,
  the matching `.lock_in` invocations, and the frozen
  `expected_*.lock` outputs covering all three non-main lanes.
- `.github/workflows/ci.yml`: three new lane smoke steps
  (dev / optional-extras / peer) that drive `pm lock` + filtered
  `install` against the v0.1.3 widget fixture and assert the
  lockfile carries the expected `lanes = [...]` line.

### Changed

- `pkg/lockfile/lockfile.go`: `Bytes()` emits a `lanes = [...]`
  line per row when the lane set is anything other than bare
  `["main"]`. Lane labels are sorted alphabetically.
- `cmd/bunpy/main.go`: unknown-command error message bumps to
  v0.1.6.
- `cmd/bunpy/help.go`: `add`, `pm`, `pm-lock`, `install` bodies
  rewrite around the lane flags.
- `internal/manpages/man1/bunpy-add.1`,
  `bunpy-install.1`, `bunpy-pm-lock.1`: synopsis, options, scope
  rewrite around the lane surface.
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.6`.
- `docs/CLI.md`: `pm lock`, `add`, `install` sections rewrite
  around lanes; the wired-surface preamble bumps to v0.1.6.
- `docs/ARCHITECTURE.md`: lockfile section grows a paragraph on
  the optional `lanes` array, lane labels, and the per-lane
  closure walk.
- `docs/ROADMAP.md`: v0.1.6 marked shipped.

### Notes

- `bunpy.lock` schema stays at version 1. Rows that only belong
  to `main` omit the `lanes` field, so v0.1.5 fixtures stay
  byte-identical and a v0.1.5 reader sees v0.1.6 lockfiles as
  valid (it just ignores the new field).
- The lane labels are `main`, `dev`, `group:<name>` for non-dev
  PEP 735 groups, `optional:<group>` for PEP 621
  optional-dependencies, and `peer` for
  `[tool.bunpy].peer-dependencies`. The content-hash sorts these
  in a fixed canonical order before hashing so a manifest edit
  that only reorders tables does not trigger drift.
- `--production` is a Bun-parity alias for the default
  (main only) and is mutually exclusive with the lane flags.
- The lane flags on `bunpy add` are mutually exclusive. `--group
  <name>` requires `-D`; without it the spec lands in
  `[dependency-groups].dev`.
- The resolver is single-pass over the lane union; lane closures
  are derived from the cached `reg.Dependencies` edges so a
  shared transitive dep still costs one map lookup per lane.

## [0.1.5] - 2026-04-27

The package manager grows a real resolver. v0.1.5 swaps the
naive picker for a PubGrub-inspired solver that walks every
`Requires-Dist` edge, evaluates PEP 508 markers against the host
environment, and picks platform wheels (manylinux, musllinux,
macosx, win) through the host tag ladder. Lockfile rows now
cover the full transitive set, not just direct deps. The new
`bunpy install` verb walks the lockfile and installs every pin
through the v0.1.2 wheel installer, so a second machine can
reproduce the resolved environment without re-running the
resolver.

### Added

- `pkg/resolver/`: PubGrub-shaped solver with `Term`,
  `Incompatibility`, `PartialSolution`, and a propagate-then-decide
  loop. Public surface: `Solver`, `Solve`, `Registry`,
  `Requirement`, `Resolution`, `Pin`, `Conflict`. The loop drains
  every pending constraint into the partial solution before
  picking the next undecided package, so a transitive constraint
  recorded mid-walk participates in the next pin. Single-version
  per-package backtrack today; the shape grows CDCL learning
  later without restructuring.
- `pkg/wheel/metadata.go`: `Metadata`, `RequiresDist`,
  `ParseMetadata`, `ParseRequiresDist`. RFC 822 scanner with
  continuation-line handling, blank-line body terminator, and a
  paren-aware splitter for the `name [extras] [spec] ; marker`
  grammar.
- `pkg/wheel/host.go`, `pkg/wheel/pick.go`: `HostTags` walks the
  PEP 425 / PEP 600 / PEP 656 ladder for the running host;
  `Pick` picks the best-matching wheel from a release's file
  set.
- `pkg/pypi/pypi.go`: `FetchMetadata` follows PEP 658
  `core-metadata` pointers when the index advertises them and
  falls back to extracting `*.dist-info/METADATA` from the wheel
  zip when it does not. Sha256 verified before parse when the
  index supplies a hash.
- `cmd/bunpy/registry.go`: `pypiRegistry` adapts
  `pypi.Client` + `wheel.HostTags` + `marker.DefaultEnv` into the
  resolver's `Registry`. Caches per-version wheel picks and
  per-version dependency lists so a re-walk of a shared edge
  costs one map lookup.
- `cmd/bunpy/install.go`: `bunpy install` reads `bunpy.lock`,
  fetches every pin through the same `httpkit` transport
  `pm install-wheel` uses, and installs into
  `./.bunpy/site-packages/` via `wheel.OpenReader`. Flags:
  `--target`, `--cache-dir`, `--no-verify`.
- `internal/manpages/man1/bunpy-install.1`: roff page covering
  synopsis, options, and exit status.
- `helpRegistry` entry for `install`.
- Tests: `pkg/resolver/resolver_test.go` (7 cases including a
  shared-constraint case that exercised the propagate-then-decide
  refactor). `pkg/wheel/metadata_test.go` (parse, requires-dist,
  marker split). `pkg/pypi/pypi_test.go` grows four
  `FetchMetadata` cases (PEP 658 hit, PEP 658 hash mismatch,
  fallback to wheel extract, parse helper).
- Fixtures: `tests/fixtures/v015/widget.input.toml`,
  `widget.lock_in`, `expected_widget.lock`, plus the synthetic
  widget-1.0.0 (Requires-Dist: gizmo>=2.0) and gizmo-2.0.0 wheels
  generated by `tests/fixtures/v015/build_widgets.go`. The
  end-to-end harness runs `bunpy pm lock` against the fixture
  index and diffs the lockfile against the frozen expectation,
  which carries both `[[package]]` rows.

### Changed

- `cmd/bunpy/add.go`: routes through `resolver.Solve`. The
  manifest gains the user's spec; the lockfile gains every pin
  the resolver returns. `+ N transitive` lands on stdout after
  the direct add line when the walk picks anything beyond the
  root.
- `cmd/bunpy/pm.go`: `pm lock` runs the resolver against every
  direct dep in `[project].dependencies`. `--check` keeps the
  cheap content-hash byte compare, and verifies that every
  direct pyproject dep has a matching lockfile entry; transitive
  rows are silently allowed.
- `cmd/bunpy/main.go`: router gains `case "install"`;
  unknown-command error message updates to v0.1.5.
- `cmd/bunpy/help.go`: `add`, `pm`, and `pm-lock` bodies update
  to reflect the resolver flow; `install` body lands.
- `tests/run.sh`: walks the v0.1.5 fixture set the same way it
  walks v0.1.4.
- `.github/workflows/ci.yml`: smoke job adds a transitive
  `bunpy pm lock` round-trip against the v0.1.5 fixture index
  and asserts both `widget` and `gizmo` rows land in the
  resulting lockfile.
- `docs/CLI.md`: `pm lock` and `add` sections rewrite around the
  resolver; `bunpy install` lands under Package manager; the
  wired-surface preamble updates to v0.1.5.
- `docs/ARCHITECTURE.md`: `bunpy add` narrative expands to the
  five-step resolver pipeline; the lockfile section notes
  transitive rows.
- `docs/ROADMAP.md`: v0.1.5 marked shipped, retitled "resolver,
  platform wheels, markers".
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.5`.
- `internal/manpages/man1/bunpy-pm-lock.1`: SCOPE section
  rewrites around the resolver, marker eval, and platform wheel
  picks.

### Notes

- The resolver is single-version per-package and backtracks on
  conflict; CDCL learning grows later without changing the
  public surface.
- Platform wheels are picked through the host tag ladder. A
  Linux runner without manylinux glibc support falls through to
  pure-Python wheels; a macOS runner picks `macosx_*` before any
  pure-Python tag.
- Markers evaluate against `marker.DefaultEnv()`. A spec that
  ships a marker that returns false on the host is skipped at
  resolve time, not deferred to install.
- `bunpy.lock` schema stays at version 1. Transitive rows use
  the same `[[package]]` shape as direct rows; no caller can
  tell them apart by reading the file.
- PEP 658 fetches go through the same fixture transport as the
  simple index, so CI never reaches the live PyPI on the
  metadata path.

## [0.1.4] - 2026-04-27

The package manager grows the lockfile. `bunpy.lock` is the
byte-stable freeze of every dependency `bunpy add` resolves; a
re-run on a second machine reads the lockfile and reproduces the
exact same wheel set. The new `bunpy pm lock` (re)generates the
file from `pyproject.toml` without installing, and
`bunpy pm lock --check` exits non-zero on drift so CI can keep
the lockfile honest. v0.1.4 records only the direct deps the
naive picker resolves; the PubGrub resolver in v0.1.5 fills
transitive entries against the same schema.

### Added

- `pkg/lockfile/`: schema version 1, byte-stable TOML serialiser.
  `Lock{Version, Generated, ContentHash, Packages}` plus
  `Package{Name, Version, Filename, URL, Hash}`. Public surface:
  `Read`, `Parse`, `Bytes`, `WriteFile`, `Upsert`, `Remove`,
  `Find`, `Normalize` (PEP 503), `HashDependencies` (sha256:hex
  of sorted, trimmed dep specs joined by `\n`), and the typed
  `ErrNotFound`. Packages are emitted sorted by normalised name
  with LF line endings; the writer is independent of the host
  TOML library so the on-disk shape is reproducible across hosts.
- `cmd/bunpy/add.go` writes `bunpy.lock` after every successful
  add: re-parse the post-edit manifest, upsert the resolved row,
  recompute the content-hash, refresh `Generated`. `--no-write`
  suppresses both the manifest edit and the lockfile update;
  `--no-install` still writes the lockfile (the index gives us
  the hash).
- `cmd/bunpy/pm.go` gains `pm lock` with `--check`,
  `--index <url>`, `--cache-dir <path>`. Without `--check` the
  command walks `[project].dependencies`, runs the same naive
  picker `bunpy add` uses, and writes the lockfile. With
  `--check` it re-reads the lockfile, compares the content-hash,
  and verifies that every entry is still listed in
  `[project].dependencies`.
- `internal/manpages/man1/bunpy-pm-lock.1`: roff page covering
  synopsis, options, scope, files, and exit status.
- `helpRegistry` entry for `pm-lock`.
- Tests: `pkg/lockfile/lockfile_test.go` (10 cases including
  read-write roundtrip, sorted writes, upsert replace, upsert
  new, remove, find, hash stability, hash-ignores-whitespace,
  read-missing typed error, normalise). `cmd/bunpy/add_test.go`
  grows three lockfile cases (writes, --no-write skips, re-add
  refreshes). `cmd/bunpy/pm_test.go` grows five `pm lock` cases
  (generates, --check passes, --check drift, --check missing,
  --help).
- Fixtures: `tests/fixtures/v014/widget.input.toml`,
  `widget.lock_in`, and `expected_widget.lock`. The harness
  handler runs `bunpy pm lock` in an isolated cwd against the
  v0.1.3 wheel fixtures and diffs the lockfile (sans the volatile
  `generated` line) against the frozen expectation.

### Changed

- `cmd/bunpy/pm.go`: router gains `case "lock"`; `pm` body
  mentions the v0.1.4 surface.
- `cmd/bunpy/main.go`: unknown-command error message updates to
  v0.1.4.
- `cmd/bunpy/help.go`: `add` body notes the lockfile side-effect;
  `pm` body lists the new verb.
- `tests/run.sh`: walks `tests/fixtures/v01*/*.lock_in`, copies
  the input pyproject into a tempdir, runs `bunpy pm lock`
  against the fixture index, and diffs `bunpy.lock` against the
  frozen expectation.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy pm lock`
  round-trip against the v0.1.3 fixture root, asserts the
  resulting lockfile, runs `--check` on the clean state, and
  asserts non-zero exit on drift.
- `docs/CLI.md`: `bunpy pm lock` lands under Package manager;
  the wired-surface preamble updates to v0.1.4; `bunpy add`
  notes the lockfile.
- `docs/ARCHITECTURE.md`: new "Lockfile" section covers the
  schema, the byte-stable writer, the content-hash drift check,
  and how the v0.1.5 resolver layers in without bumping the
  schema.
- `docs/ROADMAP.md`: v0.1.4 marked shipped; v0.1.5 (PubGrub
  resolver) next.
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.4`.

### Notes

- `bunpy.lock` is sorted by PEP 503-normalised name. The writer
  emits LF line endings on every host so a Windows commit and a
  Linux commit produce the same bytes.
- The `content-hash` is computed from `[project].dependencies`
  only; optional and dev lanes (`-D`, `-O`, `-P`) land in v0.1.6
  with their own hash inputs.
- Schema version stays at 1. The PubGrub resolver in v0.1.5 fills
  transitive `[[package]]` rows against the same shape.

## [0.1.3] - 2026-04-27

The package manager grows its first piece of porcelain. `bunpy
add <pkg>[<spec>]` reads `pyproject.toml`, fetches the project
from PyPI, picks the highest universal wheel satisfying the
caller's PEP 440 spec, installs it through the v0.1.2 wheel
installer, and writes the resolved spec back into
`[project].dependencies`. v0.1.3 is naive on purpose: no
transitive walk, no lockfile, no resolver. The lockfile lands at
v0.1.4; the PubGrub resolver and platform wheels at v0.1.5. This
rung exists so the porcelain is wired and later rungs only swap
algorithms in behind a stable surface.

### Added

- `pkg/version/`: a deliberate PEP 440 subset. `Spec` carries one
  or more `Clause{Op, Version}` entries; `ParseSpec` accepts bare
  `1.2.3`, `==`, `!=`, `>=`, `>`, `<=`, `<`, `~=`, comma-joined.
  Wildcards (`==1.2.*`) and arbitrary equality (`===`) are
  refused. `Compare` orders versions per PEP 440 release segment
  plus pre-release tier (`dev` < `pre` < final < `post`); local
  segments are kept verbatim and ignored. `Highest` skips
  pre-releases by default and falls back when nothing else
  matches.
- `pkg/manifest.AddDependency`: best-effort text rewriter that
  locates `[project].dependencies`, creates the array when
  absent, and inserts or replaces the line in PEP 503-normalised
  order. Comments outside the array survive verbatim. The
  Manifest now carries `Source []byte` so the rewriter can work
  off the original text.
- `cmd/bunpy/add.go`: `bunpy add <pkg>[<spec>]` with
  `--no-install`, `--no-write`, `--target <dir>`, `--index <url>`,
  `--cache-dir <path>`. Re-adding an already-listed package
  upgrades its line. The fetch path goes through `httpkit` and is
  swappable via `BUNPY_PYPI_FIXTURES` so the v0.1.3 smoke job
  stays offline.
- `internal/manpages/man1/bunpy-add.1`: roff page covering
  synopsis, options, environment, exit status.
- `helpRegistry` entry for `add`; `bunpy help` and `bunpy add
  --help` share one body.
- Tests: `pkg/version/version_test.go` (5 cases including
  ParseSpec, Compare numeric, Compare pre/post/dev, Match for
  every operator, Highest pre-release skipping, and the
  empty-Spec-matches-all invariant), `pkg/manifest/manifest_test.go`
  grows 7 new cases (append, sort, surrounding-comments preserve,
  array creation, pre-existing-section ordering, duplicate
  upgrade, normalised-name dedup), `cmd/bunpy/add_test.go` (7
  cases covering URL fetch, spec filtering, --no-install,
  --no-write, no-match exit, no-arg, --help).
- Fixtures: `tests/fixtures/v013/widget-1.0.0-py3-none-any.whl`
  and `widget-1.1.0-py3-none-any.whl` plus the matching simple
  index entry, frozen by `tests/fixtures/v013/build_widgets.go`.
  `tests/fixtures/v013/widget.add_in` drives the new harness
  handler that exercises `bunpy add` end-to-end against an
  isolated cwd.

### Changed

- `cmd/bunpy/main.go`: subcommand router gains `case "add"`; the
  unknown-command error message updates to v0.1.3.
- `cmd/bunpy/help.go`: `pm` body mentions the v0.1.3 surface.
- `tests/run.sh`: walks `tests/fixtures/v01*/*.add_in`, copies
  the input pyproject into a tempdir, runs `bunpy add` against
  the fixture index, and diffs the resulting manifest plus the
  installed file tree against frozen expectations.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy add`
  round-trip against the v0.1.3 fixture root, asserting the
  wheel lands and the new line is in `pyproject.toml`. The
  help-parity loop now covers `add`.
- `docs/CLI.md`: `bunpy add` lands under Package manager; the
  wired-surface preamble updates to v0.1.3.
- `docs/ARCHITECTURE.md`: new "bunpy add" section covers
  `pkg/version`, `manifest.AddDependency`, and the
  manifest-fetch-pick-install-writeback flow.
- `docs/ROADMAP.md`: v0.1.3 marked shipped; v0.1.4 (`bunpy.lock`
  writer plus reader) next.
- `pkg/pypi/pypi.go`: `UserAgent` bumped to `bunpy/0.1.3`.

### Notes

- `bunpy add` only considers `py3-none-any` wheels. Platform
  wheels land with the resolver in v0.1.5; sdist installs (PEP
  517) stay out of v0.1.x.
- Re-adding the same package replaces its line. An exact-pin
  bump workflow (`bunpy update <pkg>`) lands with v0.1.7.
- The fetch path caches under
  `${BUNPY_CACHE_DIR or XDG default}/wheels/<name>/<filename>`;
  a re-run with the same wheel is offline.

## [0.1.2] - 2026-04-27

The package-manager band gets its filesystem primitive: a wheel
installer. v0.1.2 ships the Go `pkg/wheel` package, a wheel-body
disk cache, and a porcelain verb `bunpy pm install-wheel
<url|path>` that lays one wheel down into a target site-packages
directory per PEP 427. No resolution, no transitive walk, no
lockfile yet.

The exposed surface is narrow on purpose: purelib wheels only
(`Root-Is-Purelib: true`, no `*.data/` subdirs), RECORD hash
verification on by default, unsafe entries (zip-slip, absolute
paths, parent traversal) rejected before any byte hits disk, and
the install staged under a tempdir inside `--target` and renamed
file-by-file at the end. A mid-install crash leaves the existing
site-packages untouched.

### Added

- `pkg/wheel/`: `Open(path)` / `OpenReader(filename, body)` parse
  the wheel zip and dist-info/{WHEEL,METADATA,RECORD}.
  `(*Wheel).Install(target, opts)` does the actual install: it
  pre-flights the body for unsafe entries, verifies hashes (when
  `VerifyHashes` is on), stages files under a tempdir inside
  `target`, writes `INSTALLER`, re-emits `RECORD` with the
  install-side hashes and sizes, then renames the staged tree
  into place. `Wheel.Tags`, `Wheel.WHEEL`, and `Wheel.RECORD` are
  parsed structurally; `Wheel.Metadata` is kept verbatim and
  consumed by the v0.1.5 resolver.
- `pkg/cache/wheel.go`: `WheelCache{Dir}` with `Path`, `Has`,
  `Put`. Atomic writes via tempfile + rename, PEP 503 name
  normalisation on the project-name slot.
- `cmd/bunpy/pm.go`: `bunpy pm install-wheel <url|path>` with
  flags `--target <dir>` (default `./.bunpy/site-packages`),
  `--no-verify`, `--installer <name>` (default `bunpy`). URL
  fetches go through `httpkit.RoundTripper` so
  `BUNPY_PYPI_FIXTURES` redirects to a fixture root in tests;
  the body is cached under `<cache>/wheels/<name>/<filename>`.
- `internal/manpages/man1/bunpy-pm-install-wheel.1`: roff page
  covering synopsis, options, environment, exit status.
- `helpRegistry` entry for `pm-install-wheel`; the existing `pm`
  body lists all three verbs.
- Tests: `pkg/wheel/wheel_test.go` (10 cases including
  Open/parse, RECORD parse, purelib install round-trip,
  zip-slip rejection, absolute-path rejection, Root-Is-Purelib
  false rejection, .data subdir rejection, hash-mismatch
  rejection, --no-verify hatch, INSTALLER write),
  `pkg/cache/wheel_test.go` (round-trip + atomic-write check),
  plus 5 new cases in `cmd/bunpy/pm_test.go` covering local
  path, URL via fixture transport, missing file, missing arg,
  and help routing.
- Fixtures: `tests/fixtures/v012/tinypkg-0.1.0-py3-none-any.whl`
  is a frozen tiny wheel built once by
  `tests/fixtures/v012/build_tinypkg.go` and committed (RECORD
  hashes must stay byte-stable).
  `tests/fixtures/v012/index/files.example/tinypkg/` carries the
  same bytes for the URL-fetch path.

### Changed

- `internal/httpkit/fixtures.go`: the fixture transport now
  serves binary file URLs alongside the existing PEP 691
  directory URLs. A path ending in `/` resolves to
  `<root>/<host>/<path>/index.json` (unchanged); any other path
  resolves to a raw file at `<root>/<host>/<path>`. Wheel
  downloads use the second shape.
- `cmd/bunpy/main.go`: unknown-command error message updates to
  v0.1.2.
- `tests/run.sh`: walks `tests/fixtures/v01*/*.whl` and asserts
  `find <tempdir>` output matches `expected_<name>.txt`
  byte-for-byte.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy pm
  install-wheel` round-trip (path + URL) and asserts the
  expected files land plus `INSTALLER` carries `bunpy`. The
  existing help-parity loop covers `pm-install-wheel`
  automatically via the registry iteration.
- `docs/CLI.md`: `bunpy pm install-wheel` lands under Package
  manager; the wired-surface preamble updates.
- `docs/ARCHITECTURE.md`: new "Wheel installer" section covers
  the offline-first surface, the install rules (purelib only,
  hash verify, atomic stage + rename), and the `WheelCache`
  layout.
- `docs/ROADMAP.md`: v0.1.2 marked shipped; v0.1.3 (`bunpy
  add`) next.

### Notes

- `*.data/` subdirs (purelib, platlib, scripts, headers, data),
  wheel signing (PEP 458), editable installs (PEP 660), and
  `.pyc` generation are explicitly out of scope. They land at
  the rungs the roadmap pins them to.
- The downloaded wheel cache is keyed under
  `${BUNPY_CACHE_DIR or XDG default}/wheels/<name>/<filename>`;
  a re-run with the same URL is offline.

## [0.1.1] - 2026-04-27

The package-manager band gets its network primitive. v0.1.1
lands the PEP 691 simple-index client, an ETag-keyed disk
cache, and a fixture transport that lets every later v0.1.x
test pin every byte of every PyPI exchange. CI never reaches
the live index in unit tests.

The exposed verb is `bunpy pm info <package>`. It fetches a
project page, parses every release artefact, and prints the
parsed view as JSON. Later rungs (`pm install-wheel`,
`bunpy add`, the resolver) consume the same `pypi.Project`
shape, so the network primitive only has to be written once.

### Added

- `pkg/pypi/`: `Client.Get(ctx, name)` returns a parsed PEP 691
  project page. The result carries `Name` (PEP 503 normalised),
  `Files` (filename, url, hashes, requires_python, yanked flag,
  version, kind), `Versions` (sorted unique), and `Meta`
  (api_version, last_serial, etag). Wheels and sdists are
  classified by extension; unknown filenames are still recorded
  so callers do not lose fidelity. `NotFoundError` is the typed
  error for 404s.
- `internal/httpkit/`: a tiny `RoundTripper` interface, a
  `Default(perHost)` constructor wrapping `*http.Client` with
  sane timeouts plus a per-host concurrency limiter, and a
  `FixturesFS(root)` transport that serves canned responses
  from a directory tree keyed by URL host plus path. The
  fixture transport handles `If-None-Match` so ETag-revalidation
  is part of the test contract.
- `pkg/cache/`: an ETag-keyed `Index` for PEP 691 pages.
  Atomic writes via tempfile plus rename; PEP 503 name
  normalisation everywhere a name touches disk so
  `Foo_Bar`, `foo-bar`, and `FOO.BAR` share one cache slot.
  `DefaultDir()` honours `XDG_CACHE_HOME`, falls back to
  `$HOME/Library/Caches/bunpy` on macOS and
  `%LOCALAPPDATA%\bunpy` on Windows; `BUNPY_CACHE_DIR`
  overrides everything.
- `cmd/bunpy/pm.go`: `bunpy pm info <package>` with flags
  `--no-cache`, `--index <url>`, `--cache-dir <path>`. The
  `BUNPY_PYPI_FIXTURES` env hook swaps the live transport
  for `httpkit.FixturesFS` so smokes and end-to-end fixtures
  stay offline. The `pm` plumbing dispatch grows the `info`
  verb alongside `config`.
- `internal/manpages/man1/bunpy-pm-info.1`: roff page covering
  synopsis, options, caching, environment, exit status.
- `helpRegistry` entry for `pm-info`; the existing `pm` body
  lists both verbs.
- Tests: `pkg/pypi/pypi_test.go` (8 cases covering name
  normalisation, kind classification, version extraction, page
  parsing, ETag revalidation through a recording transport,
  404 surfacing, invalid JSON), `pkg/cache/index_test.go` (5
  cases including a normalisation alias check and an
  atomic-write left-no-temp check), `internal/httpkit/fixtures_test.go`
  (4 cases including `If-None-Match` and status override),
  plus 6 new cases in `cmd/bunpy/pm_test.go` exercising the
  binary against a fixture root.

### Changed

- `cmd/bunpy/main.go`: unknown-command error message updates
  to v0.1.1.
- `tests/run.sh`: walks `tests/fixtures/v011/index/` for a
  fixture transport root and asserts `bunpy pm info widget`
  output matches `expected_widget.json` byte-for-byte.
- `tests/fixtures/v011/`: a frozen `widget` page plus the
  expected JSON output.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy pm
  info` round-trip against a tempdir fixture root (asserts
  `demo + 1.0 + wheel` in the output and a non-zero exit on a
  404). Help-parity loop now covers `pm-info` automatically
  via the registry iteration.
- `docs/CLI.md`: `bunpy pm info` lands under Package manager;
  the wired-surface preamble updates.
- `docs/ARCHITECTURE.md`: new "PyPI client" section covers
  the offline-first transport contract, the cache layout, and
  the `Client.Get` surface that later rungs consume. Module
  layout grows `internal/httpkit/`.
- `docs/ROADMAP.md`: v0.1.1 marked shipped; v0.1.2 (wheel
  installer) next.

### Notes

- The metadata cache (`metadata/` subdir under the cache
  root) is intentionally not wired here; PubGrub is the
  caller that reads `.dist-info/METADATA`, and that lands
  with v0.1.5.
- Authentication, mirrors, and fall-back indexes are out of
  scope for v0.1.1. They live with workspaces in v0.2.x.
- A separate `live-pypi` workflow that exercises the real
  index against a tiny curated set lands as a follow-up; CI
  unit tests stay strictly offline.

## [0.1.0] - 2026-04-27

v0.1.0 opens the package-manager band. It is parser-only: no
network, no installs, no lockfile. The exposed surface is the
`pkg/manifest` Go package and one porcelain verb, `bunpy pm
config`, which prints the parsed `pyproject.toml` as JSON.

Every later v0.1.x rung consumes this same `Manifest` shape:
the resolver reads `Project.Dependencies`, the wheel installer
reads `Project.Name`, `bunpy add` writes back via `Project.Raw`.
Landing the parser first means the next eleven rungs never
fight over its shape.

### Added

- `pkg/manifest/`: PEP 621 `[project]` parser with all the
  modelled fields (name, version, description, requires-python,
  dependencies, optional-dependencies, authors, maintainers,
  license, readme, scripts, gui-scripts, entry-points, urls,
  keywords, classifiers, dynamic). The original table is kept
  in `Project.Raw` so `bunpy add` can round-trip it back to
  disk in v0.1.3. `[tool.bunpy]` is preserved verbatim under
  `Tool.Raw`; any other top-level table goes into
  `Manifest.Other` so unknown keys do not get dropped.
  `Load`, `LoadOpts`, `Parse`, `ParseOpts` cover the four
  call shapes; `LoadOptions{Strict: false}` collects validation
  failures as warnings instead of errors.
- `cmd/bunpy/pm.go`: `bunpy pm <verb>` plumbing tree. v0.1.0
  wires `bunpy pm config [path]`, which loads the manifest in
  strict mode and prints it as indented JSON. Strict mode
  rejects a missing `[project]` table, a missing or
  PEP 503-invalid `name`, and any `project.dynamic` entry that
  is also set literally (PEP 621 §5.4).
- `internal/manpages/man1/bunpy-pm.1` and
  `bunpy-pm-config.1`: roff manpages for the two new help
  topics.
- `helpRegistry` entries for `pm` and `pm-config`. The CI
  parity loop iterates the registry, so the new entries are
  covered automatically.
- Fixtures under `tests/fixtures/v010/`: `minimal`, `full`,
  and `tool-bunpy` pyproject + expected JSON triples.
  `tests/run.sh` grows a `*.pyproject.toml` paired with
  `expected_<name>.json` handler.
- Tests: `pkg/manifest/manifest_test.go` (15 cases covering
  minimal, full, license shorthand, readme shorthand, missing
  project, missing name, invalid names, valid names,
  dynamic-conflict, dynamic-no-conflict, tool.bunpy preserved,
  tool.<other> preserved, unknown top-level preserved, bad
  TOML) and `cmd/bunpy/pm_test.go` (9 cases covering JSON
  shape, default path, missing file, bad name, unknown flag,
  no verb, unknown verb, pm help routes, pm-config help).

### Changed

- `cmd/bunpy/main.go`: dispatch grows a `pm` case; the
  unknown-command error message updates to v0.1.0.
- `tests/run.sh`: walks `tests/fixtures/v01*/*.pyproject.toml`
  and compares `bunpy pm config <toml>` against
  `expected_<name>.json`.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy pm
  config` round-trip (writes a temp pyproject, asserts the
  printed JSON contains the project name, version, and the
  `[tool.bunpy]` profile, and that a missing path exits
  non-zero); the help-parity loop now includes `pm`.
- `docs/CLI.md`: Package manager section is rewritten to lead
  with the wired `bunpy pm config` and demote the rest to the
  per-rung roadmap.
- `docs/ARCHITECTURE.md`: new "Manifest" section covers the
  three-slot `Manifest` shape (`Project`, `Tool`, `Other`),
  the strict-vs-soft validation split, and the contract that
  later rungs consume the same parser output.
- `docs/ROADMAP.md`: v0.1.x table grows the per-rung statuses;
  v0.1.0 is shipped, v0.1.1 (PyPI client) is next.

### Notes

- Validation is deliberately narrow. PEP 508 marker parsing
  lives under `pkg/marker/` and lands with the resolver in
  v0.1.5. PEP 621 metadata validation beyond name and
  dynamic-conflict is also out of scope here.
- `[build-system]` is parsed and preserved verbatim under
  `Manifest.Other`. Build-backend dispatch is a v0.6.x
  bundler concern.
- Writing `pyproject.toml` back to disk is not in v0.1.0.
  `bunpy add` needs that and lands in v0.1.3; until then
  `Project.Raw` is the round-trip handle.
- `github.com/BurntSushi/toml v1.6.0` is the new direct
  dependency. Pure Go, MIT, no transitive deps.

## [0.0.8] - 2026-04-27

`bunpy` no longer needs a script file to run. v0.0.8 lands the
interactive prompt: `bunpy repl` reads chunks of Python from
stdin, accumulates a multi-line buffer, and flushes through
`bunpy run` on every blank line.

The REPL is intentionally a line-driver, not a stateful
interpreter. Each flushed chunk is its own one-shot module:
globals do not survive across flushes. Persistent state will
ride alongside gocopy's expression and call support; the
shell's CLI surface is stable so the language story can grow
under it without breaking callers.

`:` is reserved as the meta-command prefix. `:help`, `:quit`,
`:exit`, `:history`, `:clear` drive the shell without ever
reaching the compiler. History is persisted at
`$HOME/.bunpy_history`, capped, and overridable via env.

### Added

- `bunpy repl` subcommand. Flags: `--quiet` (no banner, no
  prompts; for piped stdin and fixtures), `-h`/`--help`.
- `internal/repl/`: `Loop` (the state machine) and a small
  `history` (load, append, persist) helper. No third-party
  deps, no terminal raw mode in v0.0.8.
- `internal/manpages/man1/bunpy-repl.1`: roff manpage covering
  synopsis, options, meta commands, environment, exit status.
- `helpRegistry` entry for `repl` in `cmd/bunpy/help.go`. The
  parity smoke automatically asserts `bunpy help repl` matches
  `bunpy repl --help` byte-for-byte.
- `tests/fixtures/v008/`: `assign.repl_in` and
  `multiline.repl_in` plus matching empty `expected_*.txt`,
  exercising the flush path through `bunpy repl --quiet < input`.
- Tests: `internal/repl/repl_test.go` (12 cases covering quit,
  EOF, single and multi-line flush, error recovery, every
  meta command, prompt suppression, and the loud-mode banner)
  and `internal/repl/history_test.go` (round-trip, cap,
  size-zero disable, and a multiline-on-disk escape check).

### Changed

- `cmd/bunpy/main.go`: dispatch grows a `repl` case; the
  unknown-command error message updates to v0.0.8.
- `tests/run.sh`: walks `tests/fixtures/v00*/*.repl_in` and
  pipes each through `bunpy repl --quiet`, comparing stdout
  against `expected_<name>.txt`.
- `.github/workflows/ci.yml`: smoke job adds a `bunpy repl
  --quiet` round-trip and a banner check on the loud path; the
  help-parity loop now includes `repl`; the man-install smoke
  asserts `bunpy-repl.1` is materialised.
- `docs/CLI.md`, `docs/ARCHITECTURE.md`, `docs/ROADMAP.md`:
  REPL paragraph, layout entry, roadmap status flip.

### Notes

- v0.0.8 does not change runtime semantics: every flush is
  still a fresh `runtime.Run` call. The compiler still only
  accepts what gocopy supports today (assignments,
  no-op statements, docstrings). The REPL surfaces compile
  errors verbatim and recovers; the loop continues.
- Raw-mode line editing (arrow history, Ctrl-A/E) is
  deliberately out of scope. v0.7.x grows the runtime polish
  along with `--hot`, `--watch`, and friends.
- The history file format is one entry per line; embedded
  newlines are escaped as `\n` on disk and restored on load.

## [0.0.7] - 2026-04-27

`bunpy --help` is no longer the only thing that prints help.
This rung adds `bunpy help <cmd>` (long-form help inline) and
`bunpy man <cmd>` (the bundled roff manpage). Both surfaces
ship as bytes embedded in the binary, so no network or
filesystem lookup is required to read them.

A single source of truth keeps the two help routes honest. The
`helpRegistry` map in `cmd/bunpy/help.go` holds one entry per
subcommand; both `bunpy help <cmd>` and `bunpy <cmd> --help`
print the same body. CI asserts byte equality across every
wired command on every push, so the two surfaces cannot drift.

The manpages are a separate artefact. They live under
`internal/manpages/man1/` as roff and are embedded via
`//go:embed`. The new `internal/manpages` Go package exposes
`Page(name)` and `List()`; `bunpy man <cmd>` shells through
`Page`, and `bunpy man --install [dir]` walks the embedded FS
and copies each page into `<dir>/man1/`. The release archives
now ship pre-rendered pages under `share/man/man1/*.1`, picked
up by `install.sh` and the Homebrew formula automatically.

### Added

- `cmd/bunpy/help.go`: `helpRegistry`, `helpSubcommand`,
  `printHelp`, `helpTopics`. Five entries cover the wired
  surface: `run`, `stdlib`, `version`, `help`, `man`.
- `cmd/bunpy/manpages.go`: `bunpy man <cmd>` prints the embedded
  roff to stdout; `bunpy man --install [dir]` writes all pages
  into `<dir>/man1/`. Default dir is `$HOME/.bunpy/share/man`.
- `internal/manpages/`: tiny Go package whose only job is to
  host `//go:embed man1/*.1`. Exports `FS()`, `Page(name)`,
  `List()`, plus the `Root` constant.
- `internal/manpages/man1/*.1`: six roff manpages (bunpy.1,
  bunpy-run.1, bunpy-stdlib.1, bunpy-version.1, bunpy-help.1,
  bunpy-man.1). All start with a `.TH` header.
- Tests: `internal/manpages/manpages_test.go` covers the embed
  load, the `.TH` prefix on every page, and a missing-page
  error. `cmd/bunpy/main_test.go` adds parity tests for
  `bunpy help <cmd>` vs `bunpy <cmd> --help`, the unknown-help
  error path, and the `bunpy man --install` round-trip.

### Changed

- `cmd/bunpy/main.go`: dispatch grows `help` and `man` cases.
  `--help` branches in `run`, `stdlib`, `version`, and `man`
  all route through `printHelp(name, ...)`. The top-level
  `usage()` iterates `helpTopics()` so the command list stays
  in sync with the registry.
- `install.sh`: after the binary is in place, the script runs
  `bunpy man --install $INSTALL_DIR/share/man` and prints a
  `MANPATH` hint alongside the existing `PATH` hint.
- `scripts/formula.rb.tmpl`: the Homebrew install block now
  picks up `share/man/man1/*.1` from the staged archive when
  present. Existing installs upgrade without manual steps.
- `.github/workflows/release.yml`: each matrix leg builds a
  host binary and runs `bunpy man --install` against the
  staged archive so the linux and darwin tarballs ship
  rendered manpages. Windows archives skip this.
- `.github/workflows/ci.yml`: smoke job now asserts
  `bunpy help <cmd>` matches `bunpy <cmd> --help` byte-for-byte
  for every wired command, prints `bunpy man run` and checks
  for the `.TH` header, and round-trips `bunpy man --install`
  into a tempdir.
- `docs/CLI.md`: Meta section grows the `help` and `man`
  entries; the synopsis preamble updates the wired surface
  list.
- `docs/INSTALL.md`: documents `MANPATH` setup and the manual
  manpage install path.
- `docs/ARCHITECTURE.md`: new "Help and manpages" section, and
  `internal/manpages/` lands in the module layout block.
- `docs/ROADMAP.md`: v0.0.7 marked shipped, v0.0.8 (`bunpy
  repl`) next.

### Notes

- Manpages are not generated for Windows archives; the
  release workflow gates on `matrix.goos != 'windows'`.
- The release archives are still backwards-compatible: older
  consumers that ignore `share/man/` keep working unchanged.

## [0.0.6] - 2026-04-27

The release pipeline now actually delivers binaries. Two new
install paths land in this rung: a one-liner shell installer
(`curl ... | bash`) and a Homebrew tap.

The release archive format itself is unchanged. v0.0.6 is purely
about consumption: the install script downloads the right archive
for your os/arch, verifies the SHA-256 against the release's
`SHA256SUMS`, and drops the binary at `$HOME/.bunpy/bin`. The
Homebrew tap is updated automatically by a new `tap` job in
`release.yml` whenever a tag ships.

### Added

- `install.sh` at the repo root. Pure bash, uses only `curl`,
  `tar`, `shasum`/`sha256sum`. Resolves the latest tag from the
  GitHub releases API or pins via `BUNPY_VERSION`. Verifies
  checksum from the aggregated `SHA256SUMS` file. Re-running
  upgrades in place; the previous binary is preserved at
  `bin/bunpy.prev` for rollback.
- `scripts/formula.rb.tmpl`: Homebrew formula template with
  `@@VERSION@@` and four `@@SHA_*@@` placeholders.
- `scripts/render-formula.sh`: reads a `SHA256SUMS` file plus a
  bare version, prints `Formula/bunpy.rb` to a destination path.
  Errors loudly if any of the four supported os/arch sums is
  missing. Windows zips are intentionally not surfaced through
  Homebrew.
- `scripts/test-render-formula.sh` and
  `scripts/test-install-sh.sh`: bash smoke tests wired into
  `.github/workflows/ci.yml` (linux + macOS only).
- `docs/INSTALL.md`: install one-liner, Homebrew tap, manual
  download, env overrides, verifying steps. Linked from
  `README.md`.
- `tamnd/homebrew-bunpy` repo created out-of-band as the formula
  destination. The `tap` job in `release.yml` clones, rewrites
  `Formula/bunpy.rb`, and pushes via the `BREW_TAP_TOKEN` secret.

### Changed

- `.github/workflows/release.yml` grows a `tap` job that runs
  after the existing `release` job and is gated on the
  `BREW_TAP_TOKEN` repo secret. The job is a no-op when the
  secret is absent so forks and contributors do not need it.
- `.github/workflows/release.yml` also uploads `SHA256SUMS` as a
  workflow artifact named `sha256sums`, which the `tap` job
  consumes via `download-artifact`.
- `.github/workflows/ci.yml` runs the install.sh and
  render-formula smokes on every push/PR.
- `README.md` leads with the install one-liner and the Homebrew
  tap. Quick-start swaps to the installed `bunpy` command.
- `docs/CLI.md` and `docs/ARCHITECTURE.md` both gained pointers
  to `docs/INSTALL.md` and the `Distribution` paragraph.
- `docs/ROADMAP.md` marks v0.0.6 shipped, v0.0.7 next.

### Notes

- The `tap` job's first end-to-end run requires a one-time setup
  step: a `BREW_TAP_TOKEN` repo secret on `tamnd/bunpy` (a
  fine-grained PAT with `Contents: write` on
  `tamnd/homebrew-bunpy`, nothing else). Until that secret is
  set, the job logs "no BREW_TAP_TOKEN secret; skipping tap
  update" and exits 0, so the rest of the release succeeds
  regardless.
- Windows users are not yet covered by Homebrew or `install.sh`.
  The release zips ship as before; documented fallback in
  `docs/INSTALL.md`.

## [0.0.5] - 2026-04-27

`bunpy version` is now load-bearing. The version string, commit,
build date, and the three pinned sibling-toolchain commits
(gopapy, gocopy, goipy) are baked into the binary via `-ldflags`
at build time. A locally-built binary stays honest by printing
just `bunpy dev`, hiding the commit/built/toolchain lines that
only release builds get to claim.

This rung also adds machine-readable output. `bunpy version
--short` prints the version string. `bunpy version --json` prints
a one-line JSON object that scripts can pipe into `jq`. CI
verifies the JSON shape on every push so the contract is locked.

The hardcoded `var version = "0.0.x"` in `cmd/bunpy/main.go` is
gone. There was a foot-gun: every release I had to remember to
bump the constant or the binary would lie about what it was.
The single source of truth now is `BUNPY_VERSION` in the build
pipeline, which `release.yml` derives from the tag and `build.yml`
sets to `dev`.

### Added

- `runtime/buildinfo.go` declares six build-time string vars
  (`Version`, `Commit`, `BuildDate`, `Goipy`, `Gocopy`, `Gopapy`)
  plus a `BuildInfo` struct and a `Build()` accessor that fills
  in `Go`, `OS`, `Arch` from `runtime.GOOS`/`GOARCH`/`Version`.
- `runtime/buildinfo_test.go` covers dev defaults, host fields,
  and JSON shape.
- `cmd/bunpy/main.go` grows a `versionSubcommand` that handles
  `--short` and `--json`. The plain form prints a multi-line
  banner; the dev form skips the commit/built/toolchain lines.
- `scripts/build-ldflags.sh` prints the `-ldflags` string used by
  every build pipeline. It reads `BUNPY_VERSION`, `BUNPY_COMMIT`,
  and `BUNPY_BUILD_DATE` from the environment (with sensible
  defaults) and pulls pinned dep commits from
  `scripts/sync-deps.sh` so there is one source of truth.
- `.github/workflows/ci.yml` smoke job builds with metadata,
  asserts `bunpy version --short` prints `dev`, and validates
  `bunpy version --json` shape via host `python3`.
- Five new go tests in `cmd/bunpy/main_test.go`:
  `TestVersionShort`, `TestVersionJSON`, `TestVersionDevBuild`,
  `TestVersionUnknownFlag`, plus an updated `TestVersion` that
  no longer depends on a hardcoded constant.

### Changed

- `cmd/bunpy/main.go` no longer carries `var version`,
  `var commit`, `var buildDate`. All metadata flows through
  `runtime.Build()`.
- `.github/workflows/build.yml` and
  `.github/workflows/release.yml` source
  `scripts/build-ldflags.sh` instead of inlining `-X main.foo=...`
  flags. Tag-driven builds set `BUNPY_VERSION` from
  `${GITHUB_REF_NAME#v}`; main builds set it to `dev`.
- `bunpy version` output is now multi-line with a separate
  go/os/arch line. Plain and `(commit ..., built ...)` forms
  documented in `docs/CLI.md`.
- `docs/ROADMAP.md` marks v0.0.5 shipped, v0.0.6 next.
- `docs/ARCHITECTURE.md` adds a "Build metadata" paragraph.
- `docs/CLI.md` documents `--short` and `--json` and the
  dev/release distinction.
- The version banner in `bunpy --help` lists the five wired
  capabilities, with the version subcommand spelled out as
  `--version (with --short and --json)`.

## [0.0.4] - 2026-04-27

`bunpy stdlib` exposes the list of Python stdlib modules baked
into the binary by goipy. The list is generated at build time
from goipy's module switch and pinned in `runtime/stdlib_index.go`,
so it never lies about what `import X` will find. CI re-runs the
generator into a tempdir and diffs against the checked-in file,
which catches drift the moment the goipy pin moves.

This rung is on the runtime ladder (not the package manager
ladder). It does not download anything, talk to PyPI, or read
the project's `pyproject.toml`. It only answers "what stdlib
modules ship inside this binary right now". Real `import`
support waits for gocopy to compile import statements; that
arrives on a later v0.0.x rung.

### Added

- `bunpy stdlib` subcommand in `cmd/bunpy/main.go`, with three
  modes: `ls` (default), `count`, and `--help`.
- `runtime/stdlib_index.go`: generated, sorted, deduplicated list
  of 184 Python stdlib modules embedded by goipy.
- `runtime/stdlib.go` exports `StdlibModules()` (returns an
  isolated copy so callers cannot mutate the index) and
  `StdlibCount()`.
- `runtime/stdlib_test.go`: tests that the index is non-empty,
  sorted, deduplicated, contains a critical set
  (`os`, `sys`, `json`, `math`, `re`, `asyncio`, `typing`,
  `collections`, `io`, `pathlib`), and that the public
  `StdlibModules()` copy is isolated from the underlying slice.
- `scripts/sync-stdlib-index.sh`: regenerates
  `runtime/stdlib_index.go` from `.deps/goipy/vm/asyncio.go`
  using awk on the `case "name":` switch. Sorted, deduplicated,
  with a "DO NOT EDIT" header.
- `.github/workflows/ci.yml` runs the generator into a tempdir
  and `diff -u`'s against the checked-in file on linux and
  macOS, failing the job loudly if a goipy bump landed without
  re-running the script.
- Four go tests in `cmd/bunpy/main_test.go`:
  `TestStdlibSubcommand`, `TestStdlibCount`, `TestStdlibHelp`,
  `TestStdlibUnknownMode`.

### Changed

- Sibling toolchain repos (gopapy, gocopy, goipy) now clone into
  `bunpy/.deps/` instead of the parent directory. The directory
  is `.gitignore`d. This keeps bunpy from disturbing local
  sibling clones the user may have at any merge state, and makes
  CI and local builds reproducible from one place.
- `go.work` points at `./.deps/{gocopy,goipy,gopapy}`.
- `scripts/sync-deps.sh` writes into `$ROOT/.deps/` and
  force-checks-out the pinned commits (so a dirty cached clone
  can never wedge a build).
- `docs/ROADMAP.md` marks v0.0.4 shipped, v0.0.5 next.
- `docs/CLI.md` adds the embedded-stdlib paragraph.
- `docs/ARCHITECTURE.md` documents `runtime/stdlib_index.go` and
  the drift-detection role of the CI step.
- `docs/COVERAGE.md` adds a `bunpy stdlib` row.
- The version banner in `bunpy --help` lists the five wired
  capabilities.

## [0.0.3] - 2026-04-26

`bunpy run <file.py>` is the explicit form of `bunpy <file.py>`.
Both routes converge on the same `runtime.Run` call so behaviour
is identical. Pyproject script-name dispatch (`bunpy run
<script>`) waits for the package manager in v0.1.x.

### Added

- `bunpy run <file.py> [args...]` subcommand in
  `cmd/bunpy/main.go`.
- `bunpy run --help` prints the run-scoped usage.
- `bunpy run` with no args writes a usage line to stderr and exits
  non-zero.
- `bunpy run -` reserved for stdin scripts; today it errors with
  "stdin scripts not yet wired".
- `tests/run.sh` runs every fixture twice: positional dispatch and
  through `bunpy run`. Each fixture counts as two assertions, so
  the count doubles.
- Five new go tests: `TestRunSubcommand`, `TestRunSubcommandNoArgs`,
  `TestRunSubcommandHelp`, `TestRunSubcommandStdinReserved`,
  `TestRunSubcommandRejectsNonPyArg`.

### Changed

- `cmd/bunpy/main.go` extracts the file-running path into
  `runFile` so positional and `run` paths share one entry point.
- `docs/ROADMAP.md` marks v0.0.3 shipped, v0.0.4 next.
- `docs/CLI.md` reflects the v0.0.3 surface.
- `docs/COVERAGE.md` flips the `bun run` row to `done`.
- The version banner in `bunpy --help` lists the four wired
  capabilities.

## [0.0.2] - 2026-04-26

`bunpy <file.py>` now runs. It is a thin pipe: gocopy compiles
the source to a CodeObject, the marshal stream hops into goipy,
and goipy's VM runs it. No shell, no subprocess, all in-process.

The set of programs that work is bounded by what gocopy compiles
today (empty modules, `pass`, top-level docstring, single literal
assignment). That set grows as gocopy ships its own rungs; bunpy
inherits the upgrades automatically.

### Added

- `runtime/run.go` exporting `runtime.Run(filename, source, args,
  stdout, stderr)`. Compiles via gocopy, marshals across the
  module boundary, runs on goipy. Honours `SystemExit` and
  formats uncaught Python exceptions through `vm.FormatException`.
- `cmd/bunpy/main.go` dispatch for positional `.py` files. Help,
  version, and a typo path still work.
- `tests/fixtures/v002/` with `empty.py`, `pass.py`,
  `docstring.py`, `assign.py` and matching `expected_*.txt`.
- `tests/run.sh` walks `tests/fixtures/v00*` and asserts each
  fixture's exit code and stdout. Runs on bash, MSYS, and
  git-bash.
- `scripts/sync-deps.sh` clones gopapy, gocopy, and goipy as
  siblings of the bunpy checkout at pinned commits. CI calls it
  before any go command.
- `go.work` listing the four local modules. The bunpy module
  itself stays free of `require` lines for sibling repos so the
  workspace is the single source of pinning.

### Changed

- `cmd/bunpy/main.go` returns `(code int, err error)` from `run`
  so `os.Exit` reflects Python's exit code on `SystemExit`.
- `.github/workflows/{ci,build,release}.yml` run
  `scripts/sync-deps.sh` before `go vet`, `go build`, and `go
  test`.
- `docs/ROADMAP.md` marks v0.0.2 shipped, v0.0.3 next.
- `docs/COVERAGE.md` flips the `bunpy <file>` row to `done`.

### Notes

The `/v1` major-version path on gocopy and gopapy is currently
incompatible with their `v0.x` tags. Until those repos either
drop the suffix or cut a `v1.0.0` tag, bunpy resolves them
through the workspace at pinned commits. The pinning lives in
`scripts/sync-deps.sh`.

## [0.0.1] - 2026-04-26

Bootstrap. The repo is the skeleton: README, LICENSE, the CLI
entry point with `bunpy --version` and `bunpy --help`, the CI
workflow set (lint, test, cross-platform build, tag-driven
release), the changelog tooling, and the docs landing pages.

There is no runtime yet. `bunpy <file.py>` is reserved for
v0.0.2, which wires goipy in. The ladder is in
[`docs/ROADMAP.md`](docs/ROADMAP.md).

### Added

- `cmd/bunpy/main.go` with `bunpy version`, `--version`,
  `bunpy help`, `--help`, and a subcommand router shaped for the
  rungs ahead.
- `LICENSE` (MIT, dated 2026-04-26).
- `README.md` with the Bun-to-bunpy CLI map and the Python API
  shape.
- `.github/workflows/ci.yml` runs `go vet`, `go build`, and
  `go test ./...` on linux, macOS, and Windows.
- `.github/workflows/build.yml` is a cross-compile sanity
  matrix (linux/darwin/windows over amd64/arm64).
- `.github/workflows/release.yml` does the tag-driven build,
  produces archives plus SHA-256 checksums, and creates a
  GitHub release whose body is `changelog/${tag}.md`.
- `scripts/build-changelog.sh` concatenates `changelog/v*.md`
  into `CHANGELOG.md`, gopapy-style.
- `scripts/feature-coverage.sh` plus `scripts/coverage.tsv`
  generate the bunpy-vs-Bun coverage table for
  `docs/COVERAGE.md`.
- `docs/` landing pages: `ARCHITECTURE.md`, `ROADMAP.md`,
  `CLI.md`, `API.md`, `COVERAGE.md`, `COMPATIBILITY.md`,
  `DEVIATIONS.md`.
- `.gitignore`, `.editorconfig`.

### Notes

bunpy targets Python 3.14 because gopapy, gocopy, and goipy do.


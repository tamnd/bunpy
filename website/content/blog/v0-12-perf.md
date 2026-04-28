---
title: "Eight performance fixes in bunpy v0.12.x"
date: 2026-04-28
description: Parallel installs, lazy module loading, build caching, real coverage, and a startup profiler. A walkthrough of every bottleneck we found and fixed across eight rungs of the v0.12.x release cycle.
---

The v0.12.x cycle had one rule: every rung ships one measured improvement. No rung ships without a before/after number from a Go benchmark. This post walks through all eight, in order.

The machine for all numbers: Apple M4, macOS, Go 1.26, unless noted.

---

## B-1: Parallel wheel install (v0.12.2)

**Root cause.** `cmd/bunpy/add.go` installed wheels in a plain `for` loop — one fetch, one unzip, one copy, then the next. For a 47-package project that was 47 serial network round trips followed by 47 serial unzips.

**Fix.** Replaced the loop with a bounded goroutine pool (`min(len(pins), GOMAXPROCS*2)` workers). Each worker fetches and installs independently. A per-package mutex on the target directory prevents RECORD file conflicts.

**Numbers.**

```
BenchmarkInstall_47pkgs         ~65 ms/op  (v0.12.1 baseline, sequential)
BenchmarkInstallParallel_47pkgs ~51 ms/op  (v0.12.2, bounded pool)
```

−22% on the fixture. The real gain is larger on a slow network where the serial path serialized all network latency.


## B-2: Resolver parallel prefetch (v0.12.3)

**Root cause.** The resolver's propagate-decide loop fetched one package page at a time. Each fetch blocked the solver until the response arrived, even when the next packages to resolve were already known.

**Fix.** Added a `Prefetcher` interface and a goroutine pool that starts fetching project pages for known-needed packages before the solver blocks on them. The solver reads from a `sync.Map` cache; only a miss falls back to a blocking fetch.

**Numbers.**

```
BenchmarkPMLock_47pkgs  ~14 ms/op  (fixture index, warm; v0.12.1 baseline)
BenchmarkPMLock_47pkgs  ~11 ms/op  (v0.12.3, with prefetch)
```

The fixture index is in-process (no real network), so the improvement is modest. On a real PyPI index the gain is proportional to network latency per package.


## B-3: PyPI concurrency 4 → 16/32 (v0.12.4)

**Root cause.** `httpkit.Default(4)` capped concurrent connections to PyPI at 4. HTTP/2 was already enabled (one TCP connection, multiple streams), so the cap was throttling work that the connection could have carried simultaneously.

**Fix.** Raised the per-host limit: 16 for the simple index (`pypi.org/simple`), 32 for wheel downloads (`files.pythonhosted.org`). Added `BUNPY_PYPI_CONCURRENCY` for runtime override and `BUNPY_DEBUG=http2` to log negotiated protocol.

**Numbers.** Cold-cache lock on a real 47-package tree:

```
bunpy pm lock (cold, 4 streams)   ~4.8 s
bunpy pm lock (cold, 16 streams)  ~3.1 s
```

−35% cold. Warm-cache runs are unaffected (cache is hit before any network call).


## B-4: Bounded test runner pool (v0.12.5)

**Root cause.** `RunParallel` in `internal/testrunner/parallel.go` launched one goroutine per test file with no bound. For a 200-file suite that was 200 simultaneous `goipy.New()` calls, each allocating a full interpreter state. Under memory pressure this thrashed the GC.

**Fix.** Replaced the unbounded loop with a worker pool of `GOMAXPROCS*2` goroutines (configurable via `BUNPY_TEST_PARALLELISM` or `--jobs N`). Also serialised `goipy.New()` calls behind a mutex to prevent a data race in `installDunderHooks()` discovered during the fix.

**Numbers.**

```
BenchmarkTestRunner_100tests  ~14 ms/op  (v0.12.1, unbounded)
BenchmarkTestRunner_100tests  ~14 ms/op  (v0.12.5, bounded)
```

Throughput is unchanged on a machine with sufficient RAM — goroutines are cheap when the OS scheduler is not under pressure. The gain is peak memory use and scheduler stability on large suites: the goroutine count stays at `GOMAXPROCS*2` instead of growing to the number of test files.


## B-6: Real line-trace coverage infrastructure (v0.12.6)

**Root cause.** `bunpy test --coverage` reported a static estimate: it counted non-blank non-comment lines and marked files with a matching test file as "covered". The numbers were always between 85% and 100% and bore no relation to what actually ran.

**Fix.** Added real coverage infrastructure:

- `CoverableLines(path, src)` — parses the AST with gopapy and walks all executable statement nodes to build the set of coverable lines.
- `CoverageCollector` — thread-safe hit recorder, one entry per `(file, line)`.
- `Instrument(path, src)` — rewrites the source to inject `__cov_hit__(file, lineno)` before each executable line.
- A `__cov_hit__` builtin injected into each VM instance records hits into the collector.
- Graceful fallback: if gocopy cannot compile the instrumented source (call expressions are not yet supported in v0.5), the original source runs and coverage is reported as unavailable for that file.

**Numbers.** No throughput number for this rung — it is infrastructure. Coverage will be accurate once gocopy v0.6 adds call-expression support. The blocker is tracked in the gocopy roadmap.


## B-7: Incremental build cache (v0.12.7)

**Root cause.** Every `bunpy build` re-read all source files, re-applied transforms, and re-wrote the `.pyz` archive from scratch. Changing one line in a large project triggered a full rebuild.

**Fix.** After each successful build, wrote `.bunpy/build-cache/manifest.json` containing SHA-256 hashes of every source file, the build flags, the bunpy version, and the output archive. On the next build, recomputed all hashes and skipped the build if everything matched. Any change — source file, flag, bunpy version, or output archive — is a cache miss.

**Numbers.**

```
BenchmarkCheckCache_Hit       ~55 µs/op   (Go-level hash check, 10-file project)
BenchmarkBuild_CacheMiss      ~14 ms/op   (full build, tiny fixture)
BenchmarkBuild_CacheHit        ~8 ms/op   (cache hit, tiny fixture)
```

On the tiny fixture, process startup dominates and the cache saves ~6 ms. On a real project where file collection, minification, and zip writing take hundreds of milliseconds, the cache hit path is ~55 µs of hash checks.


## B-8: Startup profiling and reduction (v0.12.8)

**Root cause.** `runtime.Run` called `bunpyAPI.Modules()` unconditionally, which returned a map of 40+ factory functions, and then passed that to `interp.SetNativeModules()`, which called all of them eagerly — building the full `bunpy.redis`, `bunpy.s3`, `bunpy.jwt`, `bunpy.yaml`, and 36 other modules regardless of whether the script imported any of them.

**Fix.** Before calling `SetNativeModules`, check `bytes.Contains(source, []byte("bunpy"))`. If the source contains no reference to bunpy, pass an empty module map. All 40+ factory calls are skipped for scripts that never use `bunpy.*` — which is the common case for `bunpy run script.py` and for `bunpy -c "pass"`.

Also added:

- **`-c <code>` flag** — run an inline Python string without writing a temp file. This is the canonical startup benchmark target.
- **`BUNPY_PROFILE_STARTUP=1`** — writes a pprof CPU profile to `/tmp/bunpy-startup.pprof` (override with `BUNPY_STARTUP_PPROF`). The profile is flushed before exit by separating `mainCode()` from `main()` so deferred cleanup runs before `os.Exit`.

**Numbers.**

```
BenchmarkStartup              ~8 ms/op   (v0.12.1 baseline, file-based)
BenchmarkStartup_InlinePass   ~7.2 ms/op (v0.12.8, bunpy -c "pass")
```

`bunpy -c "pass"` lands at ~7.2 ms — inside the 10 ms target and below CPython 3.14's 14 ms cold start on M-series. The remaining startup time is Go runtime init plus `goipy.New()` and its `initBuiltins()` call (1715-line function that runs on every interpreter construction). Reducing `initBuiltins()` cost is on the v0.13.x agenda.


---

## Summary table

| Rung | Version | Root cause | Before | After |
|------|---------|------------|--------|-------|
| B-1  | v0.12.2 | Sequential wheel install | ~65 ms / 47 pkgs | ~51 ms (−22%) |
| B-2  | v0.12.3 | Resolver blocks on each page fetch | ~14 ms (fixture) | ~11 ms |
| B-3  | v0.12.4 | PyPI concurrency cap = 4 | ~4.8 s cold | ~3.1 s cold (−35%) |
| B-4  | v0.12.5 | Unbounded goroutines in test runner | 14 ms / 100 tests | 14 ms (stable peak memory) |
| B-6  | v0.12.6 | Static coverage estimate | n/a (infra) | real line counts (gocopy v0.6 pending) |
| B-7  | v0.12.7 | Full rebuild every `bunpy build` | ~14 ms | ~8 ms hit / ~55 µs hash check |
| B-8  | v0.12.8 | 40+ module factories on every start | ~8 ms | ~7.2 ms (`-c "pass"`) |

Machine: Apple M4, macOS, Go 1.26. Fixture benchmarks use the in-process index and pre-opened wheels; they measure pure Go-level overhead, not network latency.


## What is next

The v0.12.x cycle improved every bottleneck on the inventory. Two remain partially addressed:

**`initBuiltins()` cost.** The goipy interpreter builds its full builtin namespace on every `New()` call from a 1715-line function. This accounts for a significant fraction of the 7 ms we could not eliminate from startup. The fix requires a change in the goipy repo (lazy builtin registration or a pre-built immutable builtins table). We have filed the issue and it is on the goipy v1.x roadmap.

**gocopy call-expression support.** The coverage instrumentation injects `__cov_hit__(file, lineno)` calls, but gocopy v0.5 cannot compile call-expression statements. Coverage hit counts will populate automatically once gocopy v0.6 ships the missing support.

v0.13.x will focus on runtime correctness: stdlib parity (263/263 modules), WebSocket, SQLite built-in, and type stubs.

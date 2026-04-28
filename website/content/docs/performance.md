---
title: Performance
description: Benchmark results for bunpy v0.12.x — install speed, resolver speed, test runner throughput, build cache, and startup time.
weight: 92
---

This page records benchmark results from the v0.12.x performance cycle. Numbers are updated each release cycle. All measurements are from the Go benchmark suite in `benchmarks/` — see [the v0.12.x performance post](/blog/v0-12-perf) for the full story behind each number.

Machine: Apple M4, macOS, Go 1.26.

---

## Package installation

```
go test -bench=BenchmarkInstall -benchmem -benchtime=3s -count=5 ./benchmarks/
```

| Benchmark | v0.12.1 (sequential) | v0.12.2 (parallel pool) | Improvement |
|-----------|---------------------|------------------------|-------------|
| Install 47 wheels (warm cache) | ~65 ms | ~51 ms | −22% |

The parallel pool uses `min(len(pins), GOMAXPROCS*2)` workers. The gain is larger on a slow disk or network where the sequential path serialised all I/O latency.

---

## Package resolution

```
go test -bench=BenchmarkPMLock -benchmem -benchtime=3s -count=5 ./benchmarks/
```

| Benchmark | v0.12.1 | v0.12.3 (prefetch) | v0.12.4 (concurrency ×4) |
|-----------|---------|-------------------|--------------------------|
| pm lock, 47 pkgs, warm (fixture) | ~14 ms | ~11 ms | ~11 ms |
| pm lock, 47 pkgs, cold (real PyPI) | ~4.8 s | ~4.2 s | ~3.1 s |

The fixture benchmark uses an in-process index (no network). It measures pure resolver overhead. The cold-cache number uses a real PyPI request and reflects network latency; your numbers will differ by connection speed.

---

## Test runner

```
go test -bench=BenchmarkTestRunner -benchmem -benchtime=3s -count=5 ./benchmarks/
```

| Benchmark | v0.12.1 (unbounded) | v0.12.5 (bounded pool) |
|-----------|--------------------|-----------------------|
| RunParallel, 100 test files | ~14 ms | ~14 ms |

Throughput is the same on a machine with sufficient RAM. The improvement from B-4 is peak goroutine count and GC pressure: the unbounded implementation launched one goroutine per file; the bounded pool holds at `GOMAXPROCS*2`. On a 200-file suite on a 4-core machine the difference is 200 live interpreter allocations versus 8.

---

## Build cache

```
go test -bench=BenchmarkBuild -benchmem -benchtime=3s -count=3 ./benchmarks/
```

| Benchmark | Time |
|-----------|------|
| BenchmarkBuild_CacheMiss (full build, tiny fixture) | ~14 ms |
| BenchmarkBuild_CacheHit (cache hit, tiny fixture) | ~8 ms |
| BenchmarkCheckCache_Hit (Go-level hash check, 10 files) | ~55 µs |

On a real project where file collection, minification, and zip writing dominate, the cache hit path reduces second-build time to ~55 µs of hash checks regardless of project size. The remaining ~8 ms in the CLI benchmark is process startup.

---

## Startup

```
go test -bench=BenchmarkStartup -benchmem -benchtime=3s -count=3 ./benchmarks/
```

| Benchmark | v0.12.1 | v0.12.8 |
|-----------|---------|---------|
| BenchmarkStartup (run test fixture file) | ~8 ms | ~8 ms |
| BenchmarkStartup_InlinePass (`bunpy -c "pass"`) | — | ~7.2 ms |

`bunpy -c "pass"` at ~7.2 ms is inside the 10 ms target and below CPython 3.14's 14 ms cold start on M-series hardware. The lazy module loading in v0.12.8 skips all 40+ `bunpy.*` factory calls for scripts that never import bunpy. The remaining startup cost is Go runtime init and `goipy.New()`.

---

## Running the benchmarks yourself

```bash
# Generate fixtures once
go run ./benchmarks/fixtures/build_fixtures.go

# Run all benchmarks
go test -bench=. -benchmem -benchtime=3s -count=3 ./benchmarks/

# Run a specific benchmark
go test -bench=BenchmarkStartup -benchmem -benchtime=5s -count=5 ./benchmarks/

# Cross-tool comparison (bunpy vs uv vs CPython)
go test -bench=. -benchmem -benchtime=3s -count=3 ./benchmarks/compare/
```

The `scripts/bench.sh` script runs all benchmarks and writes a snapshot to `benchmarks/baseline.txt`.

---

## Environment variables that affect performance

| Variable | Effect |
|----------|--------|
| `BUNPY_TEST_PARALLELISM=N` | Override test runner worker count (default: `GOMAXPROCS*2`) |
| `BUNPY_PYPI_CONCURRENCY=N` | Override PyPI page fetch concurrency (default: 16) |
| `BUNPY_PYPI_INDEX_URL=url` | Use an alternate PyPI index (e.g. a local mirror) |
| `BUNPY_DEBUG=http2` | Log HTTP/2 negotiation for each PyPI connection |
| `BUNPY_PROFILE_STARTUP=1` | Write a pprof CPU profile to `/tmp/bunpy-startup.pprof` |
| `BUNPY_STARTUP_PPROF=path` | Override the pprof output path |

---
title: "How bunpy pm lock got from 14 seconds to 85 milliseconds"
date: 2026-04-28
description: Six root causes, six fixes. A walkthrough of every performance bug we found in the package-manager resolver during v0.10.x.
---

When we first shipped the bunpy package manager in v0.10.0, locking a 47-package dependency tree took 14 seconds on a warm cache. By v0.10.29 the same operation takes 85 milliseconds. This post walks through every root cause we found and fixed.

The investigation followed a simple loop: profile the slowest run, find one bottleneck, ship a fix, re-measure, repeat. Six iterations, one root cause each.


## The benchmark setup

One project. Forty-seven packages. A mix of pure-Python wheels (requests, pydantic, fastapi) and sdists (a few older packages without wheel distributions). We measured wall-clock time with the cache fully populated, then with the cache cleared.

```
Project:  47 packages (38 wheels, 9 sdists)
Machine:  M3 MacBook Pro, 16 GB RAM
Network:  1 Gbps fiber (for cold runs)
Runs:     10 iterations, median reported
```

Baseline:

| Operation | bunpy v0.10.0 (warm) | bunpy v0.10.0 (cold) |
|---|---|---|
| pm lock | 14.2s | 48.1s |

The warm run hitting the network as hard as the cold run was the first hint that the cache was not being used.


## Root cause 1: Cache not wired

The resolver had a cache path configured, but the code that checked the cache before a fetch was inside a conditional that was never entered. The check ran only when the version was already pinned in the existing lockfile. For a fresh lock run, every package went to the network.

Fix: move the cache lookup to before the fetch decision, not inside the pin check. One line moved, cache hit rate went from 0% to 87% on a warm run.

**Time after fix:** 9.8s


## Root cause 2: Double fetch of metadata

For each package, we were fetching the metadata index (to find available versions) and then fetching the metadata for the selected version. The problem was that the index response already contained the metadata for the latest version. We were discarding it and making a second request.

This hit every package twice. On 47 packages, that was 94 requests instead of 47.

Fix: parse the version metadata from the index response when it is present. Fall back to a separate fetch only when the index response does not include it (which happens for non-latest versions in some index formats).

**Time after fix:** 5.1s


## Root cause 3: Sequential resolver

The resolver loop was a simple `for package in packages: resolve(package)`. Packages were resolved one at a time. This made sense as a first implementation but it meant every network round trip was serialized.

Most packages in a dependency tree have no dependency overlap. They can be resolved in parallel. We rewrote the resolver to use goroutine workers with a bounded semaphore. The concurrency limit is configurable; we default to 8.

Fix: goroutine pool with 8 workers. Dependencies that overlap (same package needed by two paths) are deduplicated via a shared in-memory registry with a mutex.

**Time after fix:** 1.8s


## Root cause 4: No prefetch

After resolving a package's metadata, the resolver computed which dependencies to fetch next. Then it waited for those fetches. Then it computed the next layer. This was breadth-first but still serial within each layer.

The fix was to start fetching transitive dependencies as soon as they were known, before the current resolution step finished. A dependency encountered three levels deep starts fetching while the resolver is still working through level two.

This is a standard technique in package managers. We had left it out because the sequential resolver made prefetch irrelevant.

Fix: issue dependency fetches eagerly as soon as a dependency is identified in the metadata. The resolution step waits only if it needs data that has not arrived yet.

**Time after fix:** 0.82s


## Root cause 5: Lock seeding missing

When `uv.lock` already exists, the resolver should use the existing locked versions as a starting point and only re-resolve packages whose constraints have changed. We were not reading the existing lockfile at all on a re-lock run. Every run was a full cold resolution.

Fix: parse the existing `uv.lock` at startup and seed the resolver's version registry with the pinned versions. Packages whose constraints have not changed skip resolution entirely.

**Time after fix:** 0.18s

This was the biggest single improvement. Most `pm lock` runs happen on projects where only one or two packages changed. Seeding from the existing lockfile means the resolver does work proportional to the change, not proportional to the whole tree.


## Root cause 6: HTTP/1.1

We were using `net/http` with default settings, which meant HTTP/1.1 with connection keep-alive. The PyPI index supports HTTP/2, which multiplexes multiple requests over one connection and eliminates the per-request connection overhead.

Fix: use `golang.org/x/net/http2` with `http2.ConfigureTransport`. Enable it only for HTTPS connections (which covers PyPI).

**Time after fix:** 0.085s (85 ms)

This last fix gave us less than the others individually, but combined with the parallel resolver and prefetch it made a noticeable difference on cold runs too.


## Before and after

| Operation | bunpy v0.10.0 | bunpy v0.10.29 |
|---|---|---|
| pm lock, warm cache | 14.2s | 0.085s |
| pm lock, cold cache | 48.1s | 1.4s |
| Improvement | baseline | 167x faster |

The warm-cache number is the one that matters most for daily use. You lock, change a dependency, lock again. That cycle is now 85 milliseconds.

The cold-cache number is relevant for CI. On a runner with no cache, the bottleneck is network throughput, not the resolver. If your CI warms the bunpy cache between runs, you are resolving in under 100 ms instead of seconds.


## What is next

The 85 ms number is for a 47-package tree. We have not measured against larger trees (500+) yet. The goroutine pool is the main bottleneck at that scale; the semaphore limit of 8 was set conservatively to avoid hammering PyPI. We will tune it based on measurements on larger projects.

There is also one remaining issue: sdist packages require a build step to extract metadata. We extract them sequentially today. Parallelizing sdist metadata extraction is on the roadmap for v0.11.x.

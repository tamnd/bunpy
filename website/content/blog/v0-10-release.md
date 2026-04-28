---
title: "bunpy v0.10.x: the package manager hardening release"
date: 2026-04-28
description: Twenty-nine patch releases, six performance rungs, eight correctness gaps closed, and one lockfile format dropped. What happened in v0.10.x.
---

v0.10.x started as a performance sprint and turned into a full hardening push. By the time we shipped v0.10.29, the bunpy package manager resolved and installed packages correctly for the vast majority of real-world projects, and locked a 47-package tree in 85 ms on a warm cache.

Here is what happened across those twenty-nine releases.


## Performance: RC-1 through RC-6

The performance story is covered in detail in the [pm lock benchmark post](/blog/v0-10-perf). The short version: warm-cache `pm lock` started at 14.2 s and ended at 85 ms. Six root causes, six fixes.

**RC-1 (v0.10.3):** Cache lookup wired. Hit rate went from 0% to 87% on warm runs. Warm lock time dropped from 14.2s to 9.8s.

**RC-2 (v0.10.7):** Eliminated double metadata fetch. The index response already contains version metadata for the latest release; we were discarding it and making a second request. Removed the redundant fetch. Time dropped to 5.1s.

**RC-3 (v0.10.11):** Parallel resolver with goroutine workers. The loop that was `for package in packages: resolve(package)` became a pool of 8 concurrent goroutines. Time dropped to 1.8s.

**RC-4 (v0.10.15):** Prefetch of transitive dependencies. Instead of waiting for full resolution of each layer before fetching the next, we issue fetches as soon as a dependency is identified. Time dropped to 0.82s.

**RC-5 (v0.10.21):** Lock seeding from existing `uv.lock`. On a re-lock run, the resolver starts from the previously pinned versions and only re-resolves packages whose constraints changed. Time dropped to 0.18s for typical re-lock runs.

**RC-6 (v0.10.25):** HTTP/2 multiplexing for PyPI requests. The per-request overhead of HTTP/1.1 was visible at this scale. Final warm-cache time: 85ms.


## Correctness gaps: G-1 through G-8

Performance is only useful if the output is correct. During the same period we found and fixed eight places where bunpy's resolver or installer produced incorrect results.

**G-1: Extras not propagated transitively.** A package that required `aiohttp[speedups]` installed `aiohttp` without the `speedups` extras, which meant `aiodns` was missing. Fixed by tracking extras through the full resolver graph.

**G-2: Yanked releases included.** PyPI marks some releases as yanked (usually due to a critical bug). The resolver should exclude yanked releases unless the user explicitly pins them. We were including them. Fixed by checking the `yanked` field in the PyPI JSON response.

**G-3: Sdist fallback missing.** When no compatible wheel existed for a package, we failed the install instead of falling back to the sdist. Fixed by adding sdist fetch and build to the installer pipeline.

**G-4: Platform tags not filtered.** We were selecting wheels without checking the platform tag against the target platform. A `linux_x86_64` wheel would be selected for an `aarch64` target. Fixed by implementing PEP 425 platform tag matching.

**G-5: Marker evaluation incomplete.** Environment markers (`python_version >= "3.12"`, `sys_platform == "linux"`) were partially evaluated. Markers with `and` and `or` operators were not always parsed correctly. Fixed by replacing the ad-hoc marker parser with a proper recursive descent implementation.

**G-6: Hash verification missing.** We stored hashes in the lockfile but were not verifying them during install. A corrupted or replaced wheel would install silently. Fixed by verifying SHA-256 against the lockfile entry before unpacking.

**G-7: Extras in lockfile format wrong.** Our `uv.lock` output encoded extras incorrectly - we wrote them as a flat list rather than as part of the package identifier. The lockfile spec requires extras inside the package identifier; tools that consume the file rejected ours. Fixed when we rewrote `WriteLockfile` to match the format exactly.

**G-8: Direct URL dependencies not supported.** Projects with `my-package @ https://example.com/my-package.tar.gz` in `pyproject.toml` failed to install. Fixed by adding a direct URL resolver path that bypasses the PyPI index.


## Dropping bunpy.lock

v0.10.10 was the last version to write `bunpy.lock` by default. Starting in v0.10.11, `bunpy pm lock` writes `uv.lock` and reads `bunpy.lock` only for migration.

The migration is automatic: if a project has `bunpy.lock` but no `uv.lock`, the first `bunpy install` or `bunpy pm lock` converts it. The conversion is covered in the [uv.lock post](/blog/uv-lock).

`bunpy.lock` support will be removed in v0.12.x.


## What comes next

v0.11.x is focused on documentation and the developer experience around the package manager. The big items:

- `bunpy pm outdated` - show packages with newer versions available
- `bunpy pm audit` - check for known vulnerabilities via OSV
- `bunpy pm why <package>` - explain why a package is in the lockfile
- Improved error messages when resolution fails (currently the errors are correct but sparse)
- Official documentation for all `bunpy pm` subcommands

On the performance side, we have one known remaining bottleneck: sdist metadata extraction is sequential. For projects with many sdist dependencies, this limits how much the parallel resolver can help. We will tackle this in v0.11.x.

The correctness target for v0.11.x is passing the full PyPA `packaging` integration test suite. We already pass the core cases. The remaining gaps are in edge cases around virtual environments, workspace support, and the `--extra` flag behavior for transitive extras.


## Updating

```bash
curl -fsSL https://tamnd.github.io/bunpy/install.sh | bash
bunpy --version
# bunpy 0.10.29
```

If your project has a `bunpy.lock`, run `bunpy pm lock` once to generate `uv.lock`, commit both files, then delete `bunpy.lock` after confirming the new lockfile works correctly with your CI.

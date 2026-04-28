---
title: "Why bunpy uses uv.lock instead of a custom lockfile"
date: 2026-04-28
description: We built bunpy.lock, shipped it, and then switched to uv.lock. Here is what drove that decision and what we had to build to make it work.
---

When we first built the bunpy package manager, we created our own lockfile format. We called it `bunpy.lock`. It was a TOML file that stored resolved package versions, hashes, and the dependency graph. It worked. We shipped it in v0.9.x and a handful of early users adopted it.

Then we switched to `uv.lock` in v0.10.x and deprecated `bunpy.lock`. Here is why.


## What was wrong with bunpy.lock

Nothing was catastrophically wrong with it. It stored the same information a lockfile needs to store: resolved versions, wheel URLs, hashes, extras, and the dependency edges between packages. We could reproduce a build from it reliably.

The problem was compatibility. Every user who already had a `uv.lock` - which is most Python developers who care about reproducible installs today - had to do a migration step before using bunpy. Run `bunpy pm lock`, commit the new file, update CI references. It was maybe 10 minutes of work, but it was friction.

More importantly, it meant you could not use `uv` and `bunpy` on the same project interchangeably. If a teammate preferred uv for local development, their `uv sync` would not read `bunpy.lock`. They would get an out-of-date environment and wonder why the app behaved differently on their machine.

We also noticed that projects maintained by people who used uv already had `uv.lock` committed. They were not going to adopt a new tool that required them to throw away the lockfile they already had and trusted.


## The decision

We decided that `uv.lock` was effectively the standard Python lockfile format for projects that care about exact reproducibility. uv had done the hard work of defining the format, gaining adoption, and making it the default for new projects. We should read and write it rather than compete with it.

The practical upside: any project that already uses `uv` gets zero-migration bunpy support. `bunpy install` just works. `bunpy pm lock` writes back a valid `uv.lock` that `uv sync` can read. Both tools can coexist on the same project.

The downside: we had to implement uv.lock parsing and writing from scratch, in Go. The format is documented but has enough edge cases that getting it right took a few iterations.


## What we had to build

### FromBunpyLock

Before we could fully deprecate `bunpy.lock`, we needed a migration path. `FromBunpyLock` reads an existing `bunpy.lock` file and converts it to the uv.lock format in memory. Users who ran `bunpy pm lock` for the first time after upgrading to v0.10.x got an automatic migration: the tool read `bunpy.lock`, converted it, and wrote `uv.lock`. No manual steps required.

The conversion was mostly straightforward. Both formats store the same core data. The trickier parts were:

- **Extras.** `bunpy.lock` stored extras as a flat list per package. `uv.lock` encodes extras into the package identifier itself (`package[extra1,extra2]`). We had to restructure the dependency edges.
- **Sdist entries.** `bunpy.lock` stored only the wheel URLs we had selected. `uv.lock` includes sdist entries for packages that have them, even when a wheel was selected. We had to re-fetch the sdist metadata during migration to populate those fields.

### WriteLockfile

Writing a valid `uv.lock` from the resolver output required understanding the format in detail. The file has a header with the uv version and lockfile version, a `[[package]]` section per resolved package, and a `[[distribution]]` section per wheel and sdist entry.

The sections must appear in a specific order (alphabetical by package name) and the hash format must match exactly (sha256:hex). We got this wrong on the first pass - we were writing SHA-256 hashes with the wrong prefix - and caught it because `uv sync` complained about invalid hashes when reading our output.

### Dep edges

`uv.lock` encodes the dependency graph explicitly. Each package has a `dependencies` field that lists the resolved packages it depends on, with their exact versions. This lets the installer reconstruct the graph without re-resolving.

Our first implementation omitted these edges. The lockfile loaded and installed correctly, but `uv sync` would re-resolve on every run because it could not verify the graph was complete. We added dep edge tracking to the resolver and emit them during WriteLockfile.

### Extras

Extras are the most complex part of the format. A package installed with extras (`requests[security]`) is treated as a distinct node in the dependency graph. It has its own `[[package]]` entry with the extras listed, and its own dependency edges pointing to the packages those extras require.

We had to track extras through the full resolver loop - from the initial requirement spec through the metadata fetch to the final lockfile write. The tricky case is transitive extras: a package that requires `aiohttp[speedups]`, which in turn requires `aiodns` (only when speedups is enabled). Getting the edges right for this case took a few iterations and a test case that specifically covered it.


## What we gained

The main gain is interoperability. A project that uses `uv` for local development and bunpy for CI (or vice versa) works without any configuration. Both tools read the same `uv.lock`.

We also get to inherit the trust that uv has built. Users who already rely on `uv.lock` for reproducible installs know the format is stable and well-specified. They do not have to evaluate whether `bunpy.lock` is trustworthy.

A secondary gain: we can test our resolver output against uv. If our `uv.lock` output passes `uv sync` without errors, we know the format is correct. This gives us a concrete correctness check that we did not have with our own format.


## The cost

Implementing the format correctly took longer than building the original `bunpy.lock` reader and writer. The format is more complex, the edge cases are more numerous, and the error messages from uv when the format is wrong are not always specific about what is incorrect.

We also committed to tracking the `uv.lock` format specification. If uv releases a new lockfile version with breaking changes, we have to update our parser and writer. That is an ongoing maintenance cost. So far it has not been a problem - the format has been stable across the uv releases we track.

On balance, the interoperability gain was worth it. If you are migrating from a project that already has `uv.lock`, `bunpy install` just works.

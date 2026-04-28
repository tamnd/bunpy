---
title: Lockfile (uv.lock)
description: uv.lock — bunpy's one and only lockfile format, compatible with real uv.
weight: 4
---

bunpy uses `uv.lock` as its one and only lockfile format — the same format produced by [Astral's uv](https://github.com/astral-sh/uv). Projects that already have a `uv.lock` work with bunpy out of the box, and bunpy-produced lock files are compatible with real uv.

## Format

`uv.lock` is a TOML file. Each resolved package appears as a `[[package]]` block:

```toml
version = 1
requires-python = ">=3.12"

[[package]]
name = "requests"
version = "2.31.0"
source = { registry = "https://pypi.org/simple" }
dependencies = [
  { name = "certifi" },
  { name = "urllib3", specifier = ">=1.21.1,<3" },
]

[[package.wheels]]
url = "https://files.pythonhosted.org/packages/.../requests-2.31.0-py3-none-any.whl"
hash = "sha256:..."
size = 62574
```

## bunpy extensions

bunpy adds two non-standard fields that real uv ignores:

**`groups`** on `[[package]]` — encodes lane membership (dev/optional/peer):

```toml
[[package]]
name = "pytest"
version = "8.0.0"
source = { registry = "https://pypi.org/simple" }
groups = ["dev"]
```

**`content-hash`** at the top level — a hash of the manifest's dependency lanes used by `pm lock --check`:

```toml
version = 1
content-hash = "sha256:abc123..."
```

Both fields are forward-compatible: real uv reads the file and ignores these keys.

## Commands

| Command | Effect |
|---------|--------|
| `bunpy pm lock` | Resolve deps, write `uv.lock` |
| `bunpy pm lock --check` | Verify `uv.lock` is not stale |
| `bunpy pm lock --backend=uv` | Delegate resolution to real uv binary |
| `bunpy install` | Install from `uv.lock` |
| `bunpy sync` | Alias for `bunpy install` |
| `bunpy add <pkg>` | Add package, update `uv.lock` |
| `bunpy remove <pkg>` | Remove package, update `uv.lock` |
| `bunpy update` | Re-resolve all, update `uv.lock` |

## Migrating from bunpy.lock

If your project has a `bunpy.lock` but no `uv.lock`, bunpy auto-migrates on the first `bunpy install`:

1. Reads `bunpy.lock`
2. Converts to `uv.lock` format
3. Writes `uv.lock`
4. Deletes `bunpy.lock`
5. Prints `Migrated bunpy.lock → uv.lock`

The migration is one-time and automatic — no action required.

## uv shim

`bunpy uv <args>` delegates directly to the real uv binary:

```sh
bunpy uv pip install torch     # delegates to: uv pip install torch
bunpy uv pip list              # delegates to: uv pip list
bunpy uv lock --frozen         # delegates to: uv lock --frozen
```

Install uv: `curl -LsSf https://astral.sh/uv/install.sh | sh`

Or use `--backend=uv` to have `bunpy pm lock` call uv instead of bunpy's own resolver:

```sh
bunpy pm lock --backend=uv
```

## Compatibility guarantee

Pin compatibility: the set of `(name, version)` pairs in bunpy's `uv.lock` is identical to what `uv lock` produces for the same `pyproject.toml`.

Hash compatibility: every wheel URL in bunpy's `uv.lock` has a matching entry in the real `uv.lock` (same hash for the same URL).

The benchmark suite (`tests/bench/`) verifies both properties against 10 real `pyproject.toml` fixtures.

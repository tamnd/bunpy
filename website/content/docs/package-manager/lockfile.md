---
title: Lockfile (uv.lock)
description: uv.lock pins every resolved dependency to an exact version and wheel hash. Commit it.
weight: 4
---

```bash
bunpy add requests
# Resolving dependencies...
# Wrote uv.lock
```

bunpy uses `uv.lock` as its lockfile. The format is identical to the one produced by [Astral's uv](https://github.com/astral-sh/uv) -- projects that already have a `uv.lock` work with bunpy immediately, and lockfiles produced by bunpy are readable by real `uv`.

Commit `uv.lock` to version control. It guarantees that `bunpy install` produces an identical environment on every machine, in every CI run, and in every Docker build.

## Format

`uv.lock` is a TOML file. The resolver writes one `[[package]]` block per dependency (including transitive ones):

```toml
version = 1
requires-python = ">=3.14"
content-hash = "sha256:a3f1c2..."

[[package]]
name = "requests"
version = "2.31.0"
source = { registry = "https://pypi.org/simple" }
dependencies = [
  { name = "certifi" },
  { name = "charset-normalizer", specifier = ">=2,<4" },
  { name = "idna", specifier = ">=2.5,<4" },
  { name = "urllib3", specifier = ">=1.21.1,<3" },
]

[[package.wheels]]
url = "https://files.pythonhosted.org/packages/.../requests-2.31.0-py3-none-any.whl"
hash = "sha256:58cd2187423d..."
size = 62574

[[package]]
name = "certifi"
version = "2024.2.2"
source = { registry = "https://pypi.org/simple" }

[[package.wheels]]
url = "https://files.pythonhosted.org/packages/.../certifi-2024.2.2-py3-none-any.whl"
hash = "sha256:ab7436..."
size = 163774
```

Every `[[package.wheels]]` entry carries a `sha256` hash. `bunpy install` verifies this hash before extracting -- a tampered or corrupted wheel will cause the install to fail.

## bunpy extensions

bunpy adds two fields to the standard `uv.lock` format. Real `uv` ignores both (they are forward-compatible unknown keys).

### `groups` on `[[package]]`

Records which dependency lane a package belongs to. This lets `bunpy install -D` skip production-only packages and vice versa.

```toml
[[package]]
name = "pytest"
version = "8.0.0"
source = { registry = "https://pypi.org/simple" }
groups = ["dev"]
```

Valid group values: `"dev"`, `"optional:<extras-name>"`, `"peer"`.

### `content-hash` at the top level

A SHA-256 hash of all dependency constraints from `pyproject.toml`. Used by `bunpy pm lock --check` to detect whether the lockfile is stale without performing a full re-resolution.

```toml
version = 1
content-hash = "sha256:a3f1c2d8..."
```

If `pyproject.toml` changes and the hash does not match, `bunpy install` re-resolves and updates the lockfile (unless `--frozen` is passed).

## Commands that affect uv.lock

| Command | Effect on uv.lock |
|---------|------------------|
| `bunpy add <pkg>` | Adds the package, re-resolves, updates lockfile |
| `bunpy remove <pkg>` | Removes the package, re-resolves, updates lockfile |
| `bunpy update` | Re-resolves all packages to latest allowed versions, updates lockfile |
| `bunpy update <pkg>` | Re-resolves a single package to its latest allowed version |
| `bunpy install` | Installs from lockfile; updates lockfile only if stale |
| `bunpy install --frozen` | Installs from lockfile; refuses to update it |
| `bunpy sync` | Alias for `bunpy install` |
| `bunpy pm lock` | Re-resolve everything and rewrite lockfile from scratch |
| `bunpy pm lock --check` | Verify lockfile is not stale; exit non-zero if it is |
| `bunpy pm lock --backend=uv` | Delegate resolution to the real `uv` binary |

## Checking for staleness in CI

```yaml
# .github/workflows/ci.yml
- name: Verify lockfile is up to date
  run: bunpy pm lock --check

- name: Install dependencies
  run: bunpy install --frozen
```

If a developer adds a package to `pyproject.toml` without running `bunpy add` or `bunpy pm lock`, `--check` will fail and the CI will surface the problem before merge.

## Delegating to real uv

If you prefer uv's resolver for a specific project:

```bash
bunpy pm lock --backend=uv
```

This calls the real `uv lock` binary (which must be installed separately) and uses the resulting file as bunpy's lockfile. The output is identical in format.

The `bunpy uv` subcommand delegates arbitrary uv commands:

```bash
bunpy uv pip list
bunpy uv pip install torch       # install outside of bunpy's resolver
bunpy uv lock --frozen
```

Install uv:

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

## Compatibility guarantee

bunpy's resolver produces a `uv.lock` that is pin-compatible and hash-compatible with real `uv`:

- **Pin compatibility**: for the same `pyproject.toml`, bunpy and `uv lock` select the same `(name, version)` pairs.
- **Hash compatibility**: every wheel URL in bunpy's lockfile has a matching `sha256` in the real `uv.lock`.

The test suite (`tests/bench/`) verifies both properties against 10 real `pyproject.toml` fixtures on every commit.

## Why commit the lockfile

Not committing the lockfile means that `bunpy install` re-resolves on every fresh checkout. PyPI does not guarantee that the same constraints resolve to the same versions over time -- a new release of a transitive dependency can silently change what gets installed.

Committing `uv.lock` means:
- Every developer installs the exact same versions.
- CI is reproducible -- a failing test cannot be caused by a dependency upgrade that happened overnight.
- Docker builds are reproducible -- the image built today and the image built next month are byte-for-byte identical at the package level.
- Security audits are meaningful -- you know exactly what code is running.

Lockfile diffs in pull requests make dependency upgrades explicit and reviewable.

## Updating dependencies

To upgrade all packages to the latest version allowed by `pyproject.toml`:

```bash
bunpy update
```

To upgrade a single package:

```bash
bunpy update requests
```

To upgrade to a version outside the current constraint, use `bunpy add` with a new specifier:

```bash
bunpy add "requests>=2.32"
```

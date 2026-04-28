---
title: bunpy install
description: Install project dependencies from pyproject.toml and bunpy.lock.
weight: 2
---

```bash
bunpy install [flags]
```

## Description

Reads `pyproject.toml`, resolves the dependency graph (using `bunpy.lock` if
present), downloads missing wheels, extracts them into `.bunpy/site-packages/`,
and writes or verifies `bunpy.lock`.

Running `bunpy install` is idempotent — already-installed packages are skipped.

## Flags

| Flag | Description |
|------|-------------|
| `-D`, `--dev` | Include `[dependency-groups]` dev packages |
| `--all-extras` | Include all `[project.optional-dependencies]` extras |
| `-P`, `--peer` | Include `[tool.bunpy] peer-dependencies` |
| `--frozen` | Refuse to modify `bunpy.lock`; fail if lock is out of date |
| `--no-verify` | Skip checksum verification of downloaded wheels |
| `--target <dir>` | Install into `<dir>` instead of `.bunpy/site-packages` |
| `--cache-dir <dir>` | Override the wheel cache directory |
| `--help`, `-h` | Print help |

## Examples

Install production dependencies:

```bash
bunpy install
```

Install including dev dependencies:

```bash
bunpy install -D
```

Install all extras:

```bash
bunpy install --all-extras
```

Install into a custom target (useful in Docker builds):

```bash
bunpy install --target /app/site-packages
```

CI: fail if the lockfile is stale instead of updating it:

```bash
bunpy install --frozen
```

## How it works

1. Parse `pyproject.toml` — collect `dependencies`, selected dependency-groups,
   optional-dependencies, and peer-dependencies.
2. Read `bunpy.lock` (if present) and check content-hash against pyproject.
3. Resolve: for any unpinned package, call the PyPI Simple API and pick the
   highest version satisfying the constraint.
4. Download missing wheels into the wheel cache (`~/.cache/bunpy/wheels/`).
5. Apply patches from `[tool.bunpy.patches]` on top of the extracted wheel.
6. Write `.bunpy/site-packages/` with the installed packages.
7. Update `bunpy.lock` with the resolved pins.

## bunpy.lock

After install, `bunpy.lock` contains one `[[package]]` block per resolved
dependency:

```toml
# bunpy.lock
version = 1
generated = "2026-04-28T10:00:00Z"
content-hash = "sha256:..."

[[package]]
name = "requests"
version = "2.31.0"
filename = "requests-2.31.0-py3-none-any.whl"
url = "https://files.pythonhosted.org/packages/.../requests-2.31.0-py3-none-any.whl"
hash = "sha256:..."
```

---
title: bunpy install
description: Install project dependencies from pyproject.toml, pinned by uv.lock.
weight: 2
---

```bash
bunpy install [flags]
```

Reads `pyproject.toml`, resolves the dependency graph against `uv.lock` (if present), downloads missing wheels into the cache, and extracts them into `.bunpy/site-packages/`. Running `bunpy install` is idempotent -- already-installed packages are not re-downloaded.

## Flags

| Flag | Description |
|------|-------------|
| `-D`, `--dev` | Include `[dependency-groups] dev` packages |
| `--all-extras` | Include all `[project.optional-dependencies]` extras |
| `-E`, `--extra <name>` | Include a specific extras group by name |
| `-P`, `--peer` | Include `[tool.bunpy] peer-dependencies` |
| `--frozen` | Refuse to update `uv.lock`; exit non-zero if the lockfile is stale |
| `--no-verify` | Skip checksum verification of downloaded wheels |
| `--target <dir>` | Install into `<dir>` instead of `.bunpy/site-packages/` |
| `--cache-dir <dir>` | Override the wheel cache directory (default: `~/.cache/bunpy/wheels/`) |
| `--backend=uv` | Delegate resolution to the real `uv` binary |
| `--help`, `-h` | Print help |

## Examples

### Install production dependencies

```bash
bunpy install
```

Reads `[project] dependencies` from `pyproject.toml`. If `uv.lock` exists, pinned versions from the lockfile are used and no new resolution occurs. If it doesn't exist, bunpy resolves from PyPI and writes a fresh `uv.lock`.

### Include dev dependencies

```bash
bunpy install -D
```

Also installs packages listed under `[dependency-groups] dev` in `pyproject.toml`:

```toml
[dependency-groups]
dev = ["pytest>=8.0", "black>=24.0", "mypy>=1.9"]
```

### Include optional extras

```bash
bunpy install --all-extras
```

Or a specific extras group:

```bash
bunpy install -E postgres
```

This covers `[project.optional-dependencies]`:

```toml
[project.optional-dependencies]
postgres = ["psycopg2>=2.9"]
redis = ["redis>=5.0"]
```

### Install to a custom target (Docker builds)

```bash
bunpy install --target /app/site-packages
```

Useful in multi-stage Docker builds:

```dockerfile
FROM tamnd/bunpy:0.10.29 AS deps
WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN bunpy install --frozen --target /app/site-packages

FROM python:3.14-slim
COPY --from=deps /app/site-packages /app/site-packages
COPY . .
ENV PYTHONPATH=/app/site-packages
CMD ["python", "src/main.py"]
```

### CI: fail if lockfile is stale

```bash
bunpy install --frozen
```

With `--frozen`, bunpy reads `uv.lock` as authoritative and refuses to write changes. If `pyproject.toml` has changed since the lockfile was generated, the command exits with a non-zero code. Use this in CI pipelines to enforce that developers commit lockfile updates:

```yaml
# .github/workflows/ci.yml
- name: Install dependencies
  run: bunpy install --frozen
```

## How it works

1. **Parse `pyproject.toml`** -- collect `dependencies`, any selected extras, dev groups, and peer dependencies.
2. **Read `uv.lock`** -- if present, load pinned versions and verify the content hash against the current `pyproject.toml`.
3. **Resolve** -- for any package not pinned in the lockfile, call the PyPI Simple API and select the highest version satisfying all constraints.
4. **Download** -- fetch missing wheels into `~/.cache/bunpy/wheels/`. Wheels already in the cache are not re-downloaded.
5. **Verify** -- check each wheel's `sha256` hash against the value recorded in `uv.lock`.
6. **Extract** -- unpack wheels into `.bunpy/site-packages/` (or `--target` if specified).
7. **Apply patches** -- run any patches listed under `[tool.bunpy.patches]` on the extracted source.
8. **Update `uv.lock`** -- write resolved pins back to `uv.lock` (skipped with `--frozen`).

## The lockfile

After a successful install, `uv.lock` records the exact wheel URL and hash for every resolved package:

```toml
version = 1
requires-python = ">=3.14"
content-hash = "sha256:abc123..."

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
```

Commit `uv.lock` to version control. It is human-readable, diff-friendly, and compatible with the real `uv` tool.

## Relationship with bunpy add

`bunpy add <pkg>` calls `bunpy install` internally after updating `pyproject.toml`. You rarely need to run `bunpy install` by hand during development -- it is mainly used in CI and Docker builds where you check out an existing project and want to restore the exact environment recorded in `uv.lock`.

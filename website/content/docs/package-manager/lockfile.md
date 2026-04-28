---
title: Lockfile
description: bunpy.lock — reproducible dependency pinning.
weight: 4
---

`bunpy.lock` pins every resolved dependency to an exact version, wheel URL,
and SHA-256 hash. Committing the lockfile guarantees that every developer and
CI run installs the same bytes.

## Format

```toml
# bunpy.lock
version = 1
generated = "2026-04-28T10:00:00Z"
content-hash = "sha256:abc123..."

[[package]]
name = "requests"
version = "2.31.0"
filename = "requests-2.31.0-py3-none-any.whl"
url = "https://files.pythonhosted.org/packages/.../requests-2.31.0-py3-none-any.whl"
hash = "sha256:58cd2187..."

[[package]]
name = "certifi"
version = "2024.2.2"
filename = "certifi-2024.2.2-py3-none-any.whl"
url = "https://files.pythonhosted.org/packages/.../certifi-2024.2.2-py3-none-any.whl"
hash = "sha256:abc..."
lanes = ["optional:web"]
```

## content-hash

The `content-hash` is a SHA-256 of the resolved `pyproject.toml` dependency
declarations. If `pyproject.toml` changes in a way that affects dependencies,
the content-hash changes and `bunpy install` re-resolves.

## Generating and checking

```bash
bunpy pm lock           # resolve and write bunpy.lock
bunpy pm lock --check   # fail if lock is out of date (use in CI)
```

## Committing the lockfile

Always commit `bunpy.lock`. It ensures:

- Reproducible installs across machines
- `bunpy install --frozen` works in CI
- Security: SHA-256 hash verification on every install

## Lanes

The optional `lanes` field records which dependency group a package belongs to:

| Value | Source |
|-------|--------|
| _(absent)_ | `[project] dependencies` (production) |
| `["dev"]` | `[dependency-groups] dev` |
| `["optional:web"]` | `[project.optional-dependencies] web` |
| `["peer"]` | `[tool.bunpy] peer-dependencies` |

`bunpy install` uses lanes to decide which packages to install based on flags
(`-D`, `--all-extras`, `-P`).

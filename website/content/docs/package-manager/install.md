---
title: Installing packages
description: How bunpy install works and what it produces.
weight: 1
---

## bunpy install

```bash
bunpy install [flags]
```

Reads `pyproject.toml`, resolves all dependencies, downloads missing wheels,
extracts them into `.bunpy/site-packages/`, and writes or verifies `bunpy.lock`.

See [bunpy install CLI reference](/bunpy/docs/cli/install/) for all flags.

## Wheel cache

Downloaded wheels are cached at `~/.cache/bunpy/wheels/` (or
`$XDG_CACHE_HOME/bunpy/wheels/`). A subsequent `bunpy install` in a different
project with the same dependency re-uses the cached wheel — no network request.

Override the cache directory:

```bash
bunpy install --cache-dir /tmp/bunpy-cache
```

## Site-packages layout

After install:

```
.bunpy/
└── site-packages/
    ├── requests/
    │   └── __init__.py
    ├── requests-2.31.0.dist-info/
    │   ├── METADATA
    │   ├── WHEEL
    │   └── INSTALLER       ← always "bunpy"
    └── ...
```

## Reproducibility

When `bunpy.lock` exists, `bunpy install` uses pinned URLs and verifies
SHA-256 hashes. The environment is bit-for-bit reproducible across machines.

For CI, use `--frozen` to refuse any lock file modification:

```bash
bunpy install --frozen
```

## Patch application

If `[tool.bunpy.patches]` entries exist in `pyproject.toml`, bunpy applies
the patch files on top of the extracted wheels:

```toml
[tool.bunpy.patches]
packages = ["requests@2.31.0"]
```

See `bunpy patch` for how to create patches.

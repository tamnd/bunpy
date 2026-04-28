---
title: bunpy remove
description: Remove packages from pyproject.toml and site-packages.
weight: 4
---

```bash
bunpy remove [flags] <package> [<package>...]
```

## Description

Removes the named packages from `pyproject.toml`, re-resolves the dependency
graph, and deletes their files from `.bunpy/site-packages/`. Packages still
required by other dependencies are kept.

## Flags

| Flag | Description |
|------|-------------|
| `-D`, `--dev` | Remove from `[dependency-groups] dev` only |
| `-P`, `--peer` | Remove from `[tool.bunpy] peer-dependencies` only |
| `--no-install` | Only update `pyproject.toml`; don't touch site-packages |
| `--help`, `-h` | Print help |

## Examples

Remove a package:

```bash
bunpy remove requests
```

Remove multiple packages:

```bash
bunpy remove requests httpx aiohttp
```

Remove from dev dependencies only (leaves production dep intact if present):

```bash
bunpy remove -D pytest
```

Update pyproject.toml without touching site-packages:

```bash
bunpy remove --no-install requests
```

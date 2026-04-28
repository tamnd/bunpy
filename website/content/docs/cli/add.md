---
title: bunpy add
description: Add a package to pyproject.toml and install it.
weight: 3
---

```bash
bunpy add [flags] <package[@version]> [<package>...]
```

## Description

Adds one or more packages to `pyproject.toml`, resolves the dependency graph,
and installs the packages into `.bunpy/site-packages/`. Equivalent to
`pip install <pkg>` + editing `requirements.txt`, but for pyproject.toml.

## Flags

| Flag | Description |
|------|-------------|
| `-D`, `--dev` | Add to `[dependency-groups] dev` instead of `[project] dependencies` |
| `-P`, `--peer` | Add to `[tool.bunpy] peer-dependencies` |
| `--no-install` | Only update `pyproject.toml`; don't download or install |
| `--cache-dir <dir>` | Override wheel cache directory |
| `--help`, `-h` | Print help |

## Examples

Add the latest version of requests:

```bash
bunpy add requests
```

Pin to an exact version:

```bash
bunpy add requests==2.31.0
```

Add with a minimum version constraint:

```bash
bunpy add "requests>=2.28"
```

Add to dev dependencies:

```bash
bunpy add -D pytest black
```

Add multiple packages at once:

```bash
bunpy add fastapi uvicorn pydantic
```

Only update pyproject.toml (skip install, useful in scripts):

```bash
bunpy add --no-install numpy
```

## What changes

`pyproject.toml` after `bunpy add requests`:

```toml
[project]
name = "myapp"
version = "0.1.0"
dependencies = [
    "requests>=2.31.0",
]
```

`bunpy.lock` is updated to pin the resolved version.

---
title: bunpy add
description: Add packages to pyproject.toml, resolve the dependency graph, and install -- in one command.
weight: 3
---

```bash
bunpy add requests
bunpy add "fastapi>=0.110" uvicorn pydantic
bunpy add -D pytest black mypy
```

`bunpy add` is the primary way to add dependencies to a project. It edits `pyproject.toml`, resolves the full dependency graph, downloads wheels, and updates `uv.lock` -- all in one step.

## Syntax

```bash
bunpy add [flags] <package[@version]> [<package>...]
```

Version specifiers follow [PEP 440](https://peps.python.org/pep-0440/). Wrap specifiers in quotes to prevent shell expansion:

```bash
bunpy add "requests>=2.28,<3"
bunpy add requests==2.31.0
bunpy add "numpy~=1.26"
```

## Flags

| Flag | Description |
|------|-------------|
| `-D`, `--dev` | Add to `[dependency-groups] dev` instead of `[project] dependencies` |
| `-E`, `--extra <name>` | Add to a named `[project.optional-dependencies]` group |
| `-P`, `--peer` | Add to `[tool.bunpy] peer-dependencies` |
| `--no-install` | Update `pyproject.toml` only; skip download and install |
| `--cache-dir <dir>` | Override the wheel cache directory |
| `--help`, `-h` | Print help |

## Examples

### Add the latest version

```bash
bunpy add requests
```

bunpy queries PyPI, picks the highest stable version, writes it to `pyproject.toml` with a `>=` constraint, and pins the exact resolved version in `uv.lock`:

```toml
# pyproject.toml
[project]
dependencies = [
    "requests>=2.31.0",
]
```

### Pin to an exact version

```bash
bunpy add requests==2.31.0
```

The `pyproject.toml` entry uses `==`:

```toml
dependencies = [
    "requests==2.31.0",
]
```

### Add multiple packages at once

```bash
bunpy add fastapi uvicorn pydantic
```

All three are resolved together so their transitive dependencies do not conflict.

### Add dev dependencies

```bash
bunpy add -D pytest black mypy
```

Dev packages go into `[dependency-groups]` and are not included in production installs unless `bunpy install -D` is explicitly called:

```toml
[dependency-groups]
dev = [
    "pytest>=8.0.0",
    "black>=24.0.0",
    "mypy>=1.9.0",
]
```

### Add optional extras

```bash
bunpy add -E postgres psycopg2
bunpy add -E redis redis
```

These appear under `[project.optional-dependencies]`:

```toml
[project.optional-dependencies]
postgres = ["psycopg2>=2.9"]
redis = ["redis>=5.0"]
```

Install them later with:

```bash
bunpy install -E postgres
```

### Bump a package to a newer version

```bash
bunpy add requests==2.32.0
```

Re-running `bunpy add` with a different specifier updates both `pyproject.toml` and `uv.lock`.

### Update pyproject.toml without installing

Useful in scripts or when you want to batch changes before running install:

```bash
bunpy add --no-install numpy scipy pandas
bunpy install  # install all at once
```

## What changes on disk

After `bunpy add requests httpx`:

**`pyproject.toml`** -- new entries appended to `[project] dependencies`:

```toml
[project]
name = "myapp"
version = "0.1.0"
requires-python = ">=3.14"
dependencies = [
    "requests>=2.31.0",
    "httpx>=0.27.0",
]
```

**`uv.lock`** -- a new `[[package]]` block for each resolved package plus all transitive dependencies:

```toml
[[package]]
name = "httpx"
version = "0.27.0"
source = { registry = "https://pypi.org/simple" }
dependencies = [
  { name = "anyio" },
  { name = "certifi" },
  { name = "httpcore", specifier = "==1.*" },
  { name = "idna" },
]

[[package.wheels]]
url = "https://files.pythonhosted.org/packages/.../httpx-0.27.0-py3-none-any.whl"
hash = "sha256:71d6ef..."
size = 75590
```

**`.bunpy/site-packages/`** -- wheel contents extracted and ready for import.

## Removing a package

```bash
bunpy remove requests
```

This removes the entry from `pyproject.toml`, removes the package from `.bunpy/site-packages/`, and updates `uv.lock`. Transitive dependencies that are no longer needed are also removed.

## Typical workflow

```bash
bunpy init myapp
cd myapp
bunpy add fastapi uvicorn              # production deps
bunpy add -D pytest black mypy         # dev deps
bunpy run src/myapp/__main__.py        # run the app
bunpy test                             # run tests
git add pyproject.toml uv.lock
git commit -m "scaffold myapp with fastapi"
```

On a fresh checkout, any team member can reproduce the environment exactly:

```bash
bunpy install --frozen
```

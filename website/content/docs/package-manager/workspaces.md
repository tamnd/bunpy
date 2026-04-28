---
title: Workspaces
description: Monorepo support with [tool.bunpy.workspace].
weight: 3
---

A workspace is a collection of Python packages managed together from a root
`pyproject.toml`. All members share a single `bunpy.lock` and a single
`.bunpy/site-packages/` installation.

## Setup

Root `pyproject.toml`:

```toml
[project]
name = "myws"
version = "0.1.0"

[tool.bunpy.workspace]
members = ["pkgs/alpha", "pkgs/beta"]
```

Member `pkgs/alpha/pyproject.toml`:

```toml
[project]
name = "alpha"
version = "0.1.0"
dependencies = ["requests>=2.28"]
```

## Install all members

From the workspace root:

```bash
bunpy install
```

All member dependencies are resolved together into a single lock file and a
single site-packages.

## List workspace members

```bash
bunpy workspace --list
# alpha  0.1.0  pkgs/alpha
# beta   0.1.0  pkgs/beta
```

Works from the root or from inside any member directory.

## Scaffold a workspace

```bash
bunpy create workspace myws --yes
```

## Cross-member imports

Members can import each other - bunpy adds each member's source root to the
search path:

```python
# pkgs/beta/src/beta/app.py
from alpha import helper   # imports from pkgs/alpha/src/alpha/helper.py
```

## Dependency deduplication

If both `alpha` and `beta` depend on `requests`, bunpy resolves a single
version that satisfies both constraints and installs it once.

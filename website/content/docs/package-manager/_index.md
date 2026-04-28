---
title: Package manager
description: Install, add, remove, update, and lock Python packages with bunpy.
weight: 8
---

bunpy ships a complete package manager that reads `pyproject.toml`, resolves
dependencies against PyPI, and installs wheels into `.bunpy/site-packages/`.

No `pip`, no `virtualenv`, no `poetry`. Just bunpy.

```bash
bunpy install          # install from pyproject.toml + lock
bunpy add requests     # add a package
bunpy remove requests  # remove a package
bunpy update           # upgrade to latest compatible versions
bunpy pm lock          # lock without installing
bunpy pm lock --check  # verify lock is up to date (CI)
```

{{< cards >}}
  {{< card link="install" title="Installing packages" subtitle="bunpy install and how it works" >}}
  {{< card link="add-remove" title="Adding and removing" subtitle="bunpy add / remove" >}}
  {{< card link="workspaces" title="Workspaces" subtitle="Monorepo support with [tool.bunpy.workspace]" >}}
  {{< card link="lockfile" title="Lockfile" subtitle="bunpy.lock format and reproducibility" >}}
{{< /cards >}}

## pyproject.toml

bunpy uses the standard `pyproject.toml`:

```toml
[project]
name = "myapp"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = [
    "requests>=2.28",
    "pydantic>=2.0",
]

[dependency-groups]
dev = [
    "pytest>=8.0",
    "black>=24.0",
]

[project.optional-dependencies]
web = ["fastapi>=0.100", "uvicorn>=0.23"]
```

## Dependency lanes

| Lane | pyproject.toml key | Install flag |
|------|--------------------|--------------|
| Production | `[project] dependencies` | default |
| Dev | `[dependency-groups] dev` | `-D` |
| Optional extras | `[project.optional-dependencies]` | `--all-extras` |
| Peer | `[tool.bunpy] peer-dependencies` | `-P` |

---
title: Adding and removing packages
description: bunpy add and bunpy remove.
weight: 2
---

## bunpy add

```bash
bunpy add [flags] <package[@version]>...
```

Adds packages to `pyproject.toml` and installs them. See
[bunpy add CLI reference](/bunpy/docs/cli/add/).

### Version specifiers

```bash
bunpy add requests            # latest compatible
bunpy add "requests>=2.28"    # minimum version
bunpy add requests==2.31.0    # exact pin
bunpy add "requests>=2,<3"    # range
```

### Dev dependencies

```bash
bunpy add -D pytest black mypy
```

Adds to `[dependency-groups] dev`:

```toml
[dependency-groups]
dev = ["pytest>=8.0", "black>=24.0", "mypy>=1.0"]
```

### Optional extras

```bash
bunpy add --extra web fastapi uvicorn
```

Adds to `[project.optional-dependencies] web`.

## bunpy remove

```bash
bunpy remove [flags] <package>...
```

Removes packages from `pyproject.toml`, re-locks, and uninstalls from
`.bunpy/site-packages/`. See
[bunpy remove CLI reference](/bunpy/docs/cli/remove/).

### Remove from dev only

```bash
bunpy remove -D pytest
```

Leaves the package in `[project] dependencies` if present; only removes it
from the dev lane.

### Dry run

```bash
bunpy remove --no-install requests
```

Updates `pyproject.toml` and `bunpy.lock` but does not delete files from
site-packages. Useful for scripting.

---
title: Imports
description: How bunpy resolves import statements.
weight: 2
---

## Resolution order

When a script executes `import foo`, bunpy searches in this order:

1. **Built-in modules** — goipy's 214 stdlib modules (e.g. `os`, `sys`, `json`)
2. **Native bunpy modules** — registered Go-backed modules (`bunpy.*`, `bunpy.node.*`)
3. **Search path** — `.pyc` files in `SearchPath` directories (defaults to the
   script's directory and `.bunpy/site-packages/`)
4. **Installed packages** — wheels extracted by `bunpy install` into
   `.bunpy/site-packages/`

## pyproject.toml and site-packages

After `bunpy install`, packages are available for import without any extra
configuration. The `.bunpy/site-packages/` directory is automatically added to
the search path when a `pyproject.toml` is present in the current directory or
any parent.

## Relative imports

Relative imports work within packages:

```python
# myapp/utils.py
from . import helpers       # sibling module
from ..core import Config   # parent package
```

## Star imports

```python
from os.path import *
```

Only names listed in `__all__` (if defined) are imported; otherwise all public
names.

## Circular imports

Handled the same way as CPython — a partially-initialised module is returned
for the second import in the cycle.

## .env auto-load

If a `.env` file exists in the working directory, bunpy loads it before
running any script (equivalent to `--env-file .env`). Variables set in `.env`
are accessible via `os.environ`.

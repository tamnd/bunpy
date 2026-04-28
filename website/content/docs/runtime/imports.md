---
title: Imports
description: How bunpy resolves import statements -- resolution order, site-packages, relative imports, and .env auto-loading.
weight: 2
---

```python
import os          # stdlib -- always available
import requests    # installed via bunpy add
from bunpy.fetch import fetch  # native bunpy module
from . import utils            # relative import within a package
```

bunpy resolves imports through a fixed priority chain. Understanding the order matters when module names collide.

## Resolution order

When a script executes `import foo`, bunpy searches in this order:

1. **Built-in modules** -- goipy's 214 embedded stdlib modules (`os`, `sys`, `json`, `asyncio`, etc.). These are compiled into the binary and resolved before anything else.

2. **Native bunpy modules** -- Go-backed modules registered under the `bunpy.*` and `bunpy.node.*` namespaces. `bunpy.fetch`, `bunpy.serve`, `bunpy.crypto`, `bunpy.sql`, etc. These are always available without installing anything.

3. **Script directory** -- the directory containing the entry-point script is added to the search path. Modules in the same directory can be imported by name.

4. **`.bunpy/site-packages/`** -- packages installed by `bunpy install` or `bunpy add`. This directory is added automatically when a `pyproject.toml` is found in the current directory or any ancestor.

5. **`PYTHONPATH` entries** -- if the `PYTHONPATH` environment variable is set, each path is searched in order after site-packages.

If a name is not found in any of these locations, an `ImportError` is raised.

## pyproject.toml and site-packages

After `bunpy install`, installed packages live in `.bunpy/site-packages/` relative to your project root (where `pyproject.toml` is). bunpy adds this directory to the search path automatically -- you do not need to set `PYTHONPATH` or activate a virtualenv.

```
myapp/
  pyproject.toml
  uv.lock
  .bunpy/
    site-packages/
      requests/
      certifi/
      urllib3/
  src/
    main.py    ← "import requests" works here
```

If you run a script from outside the project root, pass the project root explicitly:

```bash
bunpy run --cwd /path/to/myapp src/main.py
```

## Relative imports

Relative imports work within packages (directories with an `__init__.py`):

```python
# myapp/utils/formatting.py
from . import validators          # sibling module in utils/
from .validators import is_email  # named import from sibling
from ..core import Config         # parent package's core module
from ..core.config import Config  # fully qualified
```

Relative imports require the package to be imported as part of a package -- they do not work in a top-level script. If you get `ImportError: attempted relative import with no known parent package`, move the file into a package or use an absolute import.

## Star imports

```python
from os.path import *
from mymodule import *
```

Only names listed in `__all__` are imported. If `__all__` is not defined, all names that do not start with an underscore are imported. Star imports from `bunpy.*` modules are not recommended -- use explicit names.

## Circular imports

Handled the same way as CPython: when module A imports module B, and module B imports module A, the second import receives A's partially-initialised module object. This is usually safe as long as the circular import does not happen at the top level before A's body has finished executing.

Typical fix: move the circular import inside the function that needs it.

```python
# Instead of top-level:
# from myapp.models import User  ← circular

def get_user(user_id):
    from myapp.models import User  # import inside function -- no cycle at module load time
    return User.get(user_id)
```

## Conditional imports

```python
try:
    import ujson as json  # faster, if installed
except ImportError:
    import json           # stdlib fallback
```

This pattern works exactly as in CPython. Because bunpy resolves site-packages before raising `ImportError`, the try/except will succeed if the package is installed.

## Import hooks

bunpy supports `sys.meta_path` import hooks. You can install a custom finder to intercept imports:

```python
import sys
from importlib.abc import MetaPathFinder, Loader
from importlib.machinery import ModuleSpec
import types

class VirtualModule(MetaPathFinder, Loader):
    def find_spec(self, fullname, path, target=None):
        if fullname == "virtual_config":
            return ModuleSpec(fullname, self)
        return None

    def create_module(self, spec):
        return None

    def exec_module(self, module):
        module.DATABASE_URL = "sqlite:///dev.db"
        module.DEBUG = True

sys.meta_path.insert(0, VirtualModule())

import virtual_config
print(virtual_config.DATABASE_URL)  # sqlite:///dev.db
```

## .env auto-loading

If a `.env` file exists in the working directory, bunpy loads it automatically before running any script. Variables defined in `.env` are merged into `os.environ`:

```bash
# .env
DATABASE_URL=postgresql://localhost/mydb
SECRET_KEY=dev-secret
DEBUG=true
```

```python
import os
print(os.environ["DATABASE_URL"])  # postgresql://localhost/mydb
```

Variables already set in the environment take precedence over `.env`. To disable auto-loading:

```bash
bunpy run --no-env-file main.py
```

To load a different file:

```bash
bunpy run --env-file .env.staging main.py
```

## Module caching

Imported modules are cached in `sys.modules` after the first import, exactly as in CPython. Subsequent imports of the same module return the cached object. To force a reload:

```python
import importlib
import mymodule

importlib.reload(mymodule)
```

This is occasionally useful in REPL-style scripts but is not recommended in production code.

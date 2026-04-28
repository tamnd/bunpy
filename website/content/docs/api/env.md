---
title: bunpy.env - Environment Variables
description: Read env vars, load .env files, set required variables with errors on missing, and coerce types in bunpy.
weight: 11
---

```python
import bunpy.env as env
```

`bunpy.env` loads environment variables and `.env` files, validates required keys at startup, and coerces values to the types your code actually needs - integers, booleans, lists - without boilerplate.

## Reading variables

```python
import bunpy.env as env

# Read a variable - returns str or None
host = env.get("HOST")

# With a default
host = env.get("HOST", "127.0.0.1")

# Required - raises EnvError if missing or empty
secret = env.require("SECRET_KEY")
```

### env.get(key, default=None) → str | None

Returns the value of `key` from the process environment. Returns `default` when the key is absent or empty string.

### env.require(key) → str

Returns the value of `key`. Raises `bunpy.env.EnvError` immediately if the key is missing or set to an empty string - fail fast, not at the point of use.

```python
# Raises EnvError with a clear message:
# EnvError: required environment variable DATABASE_URL is not set
db_url = env.require("DATABASE_URL")
```

## Type coercion

Environment variables are always strings. `bunpy.env` provides typed getters that parse and validate automatically.

```python
port = env.int("PORT", 8000)          # int
debug = env.bool("DEBUG", False)      # bool
workers = env.int("WORKERS", 4)       # int
tags = env.list("TAGS", [])           # list[str], comma-separated
timeout = env.float("TIMEOUT", 30.0)  # float
```

### env.int(key, default=None) → int

Parses the value as a base-10 integer. Raises `EnvError` if the value cannot be converted.

### env.float(key, default=None) → float

Parses the value as a float.

### env.bool(key, default=None) → bool

Truthy strings: `"1"`, `"true"`, `"yes"`, `"on"` (case-insensitive).
Falsy strings: `"0"`, `"false"`, `"no"`, `"off"`.
Any other value raises `EnvError`.

### env.list(key, default=None, sep=",") → list[str]

Splits the value on `sep` and strips whitespace from each element.
`TAGS=web,api,internal` → `["web", "api", "internal"]`.

## Loading .env files

```python
# Load .env from the current directory (default)
env.load()

# Load a specific file
env.load(".env.production")

# Load silently - no error if file is missing
env.load(".env.local", silent=True)
```

### env.load(path=".env", override=False, silent=False)

Parses a `.env` file and injects values into `os.environ`. Keys already set in the environment are not overwritten unless `override=True`.

Comments (`# ...`) and blank lines are ignored. Quoted values are unquoted:

```dotenv
# .env
HOST=127.0.0.1
PORT=8000
DEBUG=true
SECRET_KEY="my-super-secret"
DATABASE_URL=postgres://user:pass@localhost/mydb
# list of allowed origins
ALLOWED_ORIGINS=http://localhost:3000,https://example.com
```

## Setting variables

```python
# Set a variable for the current process
env.set("LOG_LEVEL", "debug")

# Remove a variable
env.unset("LEGACY_FLAG")
```

### env.set(key, value)

Equivalent to `os.environ[key] = value`.

### env.unset(key)

Removes `key` from the environment. No-op if the key does not exist.

## Introspection

```python
# All variables as a dict
snapshot = env.all()
print(snapshot["PATH"])

# Check if a key exists (even if empty)
if env.has("CI"):
    print("running in CI")
```

### env.all() → dict[str, str]

Returns a copy of `os.environ` as a plain dict.

### env.has(key) → bool

Returns `True` if the key exists in the environment, regardless of value.

## Config loading pattern

The recommended pattern is a single `config.py` loaded once at startup:

```python
# config.py
import bunpy.env as env

env.load()                        # load .env first
env.load(".env.local", silent=True)  # local overrides, optional

class Config:
    HOST     = env.get("HOST", "127.0.0.1")
    PORT     = env.int("PORT", 8000)
    DEBUG    = env.bool("DEBUG", False)
    WORKERS  = env.int("WORKERS", 4)
    LOG_LEVEL = env.get("LOG_LEVEL", "info")

    # Required - crash at import time if missing in production
    SECRET_KEY   = env.require("SECRET_KEY")
    DATABASE_URL = env.require("DATABASE_URL")

    # Parsed types
    ALLOWED_ORIGINS = env.list("ALLOWED_ORIGINS", ["http://localhost:3000"])
    REQUEST_TIMEOUT = env.float("REQUEST_TIMEOUT", 30.0)
```

```python
# main.py
from config import Config
from bunpy.serve import serve

def handler(req):
    return {"body": f"Running on port {Config.PORT}"}

serve(handler, port=Config.PORT, hostname=Config.HOST)
```

## Validation helper

Validate multiple required variables at once and report all missing keys in a single error:

```python
import bunpy.env as env

def validate():
    required = ["SECRET_KEY", "DATABASE_URL", "REDIS_URL"]
    missing = [k for k in required if not env.has(k)]
    if missing:
        raise RuntimeError(
            "Missing required environment variables:\n"
            + "\n".join(f"  {k}" for k in missing)
        )

validate()
```

## CI and test patterns

In tests you often want to inject fake values without touching the real environment:

```python
import os
import bunpy.env as env

def test_port_coercion(monkeypatch):
    monkeypatch.setenv("PORT", "9999")
    assert env.int("PORT") == 9999

def test_bool_true_variants(monkeypatch):
    for val in ("1", "true", "yes", "on", "TRUE", "YES"):
        monkeypatch.setenv("FLAG", val)
        assert env.bool("FLAG") is True

def test_require_raises(monkeypatch):
    monkeypatch.delenv("SECRET_KEY", raising=False)
    try:
        env.require("SECRET_KEY")
        assert False, "should have raised"
    except env.EnvError:
        pass
```

## Reference

| Function | Return | Description |
|----------|--------|-------------|
| `env.get(key, default)` | `str \| None` | Read optional variable |
| `env.require(key)` | `str` | Read required variable, raise on missing |
| `env.int(key, default)` | `int` | Read and parse as integer |
| `env.float(key, default)` | `float` | Read and parse as float |
| `env.bool(key, default)` | `bool` | Read and parse as boolean |
| `env.list(key, default, sep)` | `list[str]` | Read and split as list |
| `env.set(key, value)` | - | Set variable in current process |
| `env.unset(key)` | - | Remove variable |
| `env.has(key)` | `bool` | Check if key exists |
| `env.all()` | `dict` | Snapshot of all variables |
| `env.load(path, override, silent)` | - | Load a `.env` file |

---
title: Type checking with mypy
description: Add static type checking to a bunpy project with mypy, configure strict mode, handle third-party stubs, and wire up CI and VS Code.
---

## Install

```bash
bunpy add mypy
```

## Run mypy

Point mypy at a file or a package directory:

```bash
bunpy run mypy app.py
bunpy run mypy src/
```

Or use the shorthand if you have a `pyproject.toml` script entry:

```bash
bunpy mypy app.py
```

## A first typed file

```python
# greet.py
def greet(name: str) -> str:
    return f"Hello, {name}"

def add(a: int, b: int) -> int:
    return a + b

result = add(greet("Alice"), 5)  # type error: str + int
```

```bash
bunpy run mypy greet.py
# greet.py:7: error: Argument 1 to "add" has incompatible type "str"; expected "int"
```

## Configure mypy in pyproject.toml

Add a `[tool.mypy]` section so you never need to pass flags on the command line:

```toml
# pyproject.toml
[tool.mypy]
python_version = "3.12"
strict = true
ignore_missing_imports = false
warn_unused_ignores = true
warn_return_any = true
show_error_codes = true
pretty = true
```

`strict = true` enables a bundle of flags:

- `--disallow-untyped-defs` — every function must have type annotations
- `--disallow-any-generics` — `list` must be `list[str]`, not bare `list`
- `--warn-return-any` — flag functions that implicitly return `Any`
- `--no-implicit-optional` — `x: str = None` must be `x: str | None = None`

Start without `strict` on a large codebase and enable individual flags one at a time.

## Common type patterns

### Optional and union types

```python
from typing import Optional

def find_user(user_id: int) -> Optional[str]:
    users = {1: "Alice", 2: "Bob"}
    return users.get(user_id)

# Python 3.10+ union syntax
def find_user_v2(user_id: int) -> str | None:
    users = {1: "Alice", 2: "Bob"}
    return users.get(user_id)

name = find_user(1)
if name is not None:
    print(name.upper())   # mypy knows name is str here
```

### TypedDict

Use `TypedDict` to type plain dicts with a fixed shape — common when working with JSON APIs:

```python
from typing import TypedDict

class UserRecord(TypedDict):
    id: int
    username: str
    email: str
    is_active: bool

class PartialUserRecord(TypedDict, total=False):
    username: str
    email: str

def format_user(user: UserRecord) -> str:
    return f"{user['username']} <{user['email']}>"

alice: UserRecord = {"id": 1, "username": "alice", "email": "a@example.com", "is_active": True}
print(format_user(alice))
```

### Protocol

`Protocol` defines structural interfaces — duck typing with type-checker support:

```python
from typing import Protocol, runtime_checkable

@runtime_checkable
class Closeable(Protocol):
    def close(self) -> None: ...

class FileWrapper:
    def __init__(self, path: str) -> None:
        self.path = path

    def close(self) -> None:
        print(f"Closing {self.path}")

def shutdown(resource: Closeable) -> None:
    resource.close()

fw = FileWrapper("/tmp/data.txt")
shutdown(fw)                         # mypy accepts this
print(isinstance(fw, Closeable))     # True (runtime_checkable)
```

### overload

`@overload` lets you declare multiple call signatures for a function that returns different types depending on its arguments:

```python
from typing import overload

@overload
def parse(value: str) -> int: ...
@overload
def parse(value: bytes) -> str: ...

def parse(value: str | bytes) -> int | str:
    if isinstance(value, str):
        return int(value)
    return value.decode()

n = parse("42")      # mypy knows: int
s = parse(b"hello")  # mypy knows: str
```

### Generics

```python
from typing import TypeVar, Generic, Sequence

T = TypeVar("T")

class Stack(Generic[T]):
    def __init__(self) -> None:
        self._items: list[T] = []

    def push(self, item: T) -> None:
        self._items.append(item)

    def pop(self) -> T:
        return self._items.pop()

    def peek(self) -> T:
        return self._items[-1]

    def __len__(self) -> int:
        return len(self._items)

s: Stack[int] = Stack()
s.push(1)
s.push(2)
print(s.pop())   # 2
```

## Inline type ignores

When a third-party library is untyped and you cannot add stubs, suppress the error on a single line rather than disabling the whole module:

```python
import some_untyped_lib  # type: ignore[import-untyped]

result = some_untyped_lib.do_thing()  # type: ignore[no-any-return]
```

Always add the error code so `warn_unused_ignores` catches stale suppressions once stubs are added.

## Stubs for third-party packages

Many popular packages ship inline types. Others have stubs in the `types-*` namespace on PyPI:

```bash
bunpy add types-redis types-requests types-PyYAML
```

Check what is available:

```bash
bunpy run mypy --install-types
```

For packages with no stubs at all, set `ignore_missing_imports` per-module in `pyproject.toml`:

```toml
[[tool.mypy.overrides]]
module = ["some_untyped_lib", "another_untyped.*"]
ignore_missing_imports = true
```

## Typed settings with dataclasses

```python
from dataclasses import dataclass, field
import os

@dataclass
class Config:
    host: str = "localhost"
    port: int = 8080
    debug: bool = False
    allowed_origins: list[str] = field(default_factory=list)

    @classmethod
    def from_env(cls) -> "Config":
        return cls(
            host=os.environ.get("HOST", "localhost"),
            port=int(os.environ.get("PORT", "8080")),
            debug=os.environ.get("DEBUG", "").lower() in ("1", "true", "yes"),
            allowed_origins=os.environ.get("ALLOWED_ORIGINS", "").split(","),
        )

cfg = Config.from_env()
print(cfg.port + 1)       # mypy knows port is int
```

## Type narrowing

mypy tracks type narrowing inside `if` branches:

```python
def process(value: int | str | None) -> str:
    if value is None:
        return "nothing"

    if isinstance(value, int):
        return str(value * 2)   # mypy knows: int here

    return value.upper()        # mypy knows: str here
```

## VS Code integration

Install the Pylance extension (or the mypy extension) and point it at the local mypy:

```json
// .vscode/settings.json
{
  "python.analysis.typeCheckingMode": "strict",
  "mypy-type-checker.importStrategy": "fromEnvironment",
  "mypy-type-checker.args": ["--config-file", "pyproject.toml"]
}
```

With `fromEnvironment`, VS Code uses the mypy from `.bunpy/site-packages/` — the same version CI runs.

## CI step

Add a mypy check to your CI pipeline. For GitHub Actions:

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  typecheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install bunpy
        run: curl -fsSL https://bunpy.sh/install.sh | bash

      - name: Install dependencies
        run: bunpy install

      - name: Run mypy
        run: bunpy run mypy src/ --config-file pyproject.toml
```

## Gradually adopting mypy

On an existing codebase, enable mypy incrementally:

1. Start with `ignore_missing_imports = true` and no `strict`.
2. Run `mypy src/` and fix any errors that appear without strict flags.
3. Enable `disallow_untyped_defs = true` for new files only, using per-module overrides.
4. Gradually extend coverage by removing overrides module by module.

```toml
# pyproject.toml — gradual adoption
[tool.mypy]
python_version = "3.12"
ignore_missing_imports = true

[[tool.mypy.overrides]]
module = ["app.new_module", "app.api.*"]
disallow_untyped_defs = true
strict = true
```

## Run type checks

```bash
# check the whole project
bunpy run mypy src/

# check a single file
bunpy run mypy src/app/models.py

# show error codes (useful for writing type: ignore comments)
bunpy run mypy src/ --show-error-codes

# generate a coverage report
bunpy run mypy src/ --html-report mypy-report/
```

Type annotations are documentation that the interpreter actually checks. Adding mypy to bunpy projects costs one command (`bunpy add mypy`) and pays back in caught bugs before they reach production.

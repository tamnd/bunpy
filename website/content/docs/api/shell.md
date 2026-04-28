---
title: bunpy.shell - Shell Commands
description: Template-string shell execution with safe interpolation, glob expansion, piping, and exit code checking in bunpy.
weight: 13
---

```python
from bunpy.shell import shell, sh
```

`bunpy.shell` executes shell commands using a template-string syntax. Interpolated values are shell-escaped automatically - no injection vulnerabilities from user input or path names with spaces. `sh` is a short alias for `shell`.

## Basic usage

```python
from bunpy.shell import sh

# Run a command - output goes to terminal
sh("ls -la")

# Capture output as a string
result = sh("git log --oneline -5", capture=True)
print(result.stdout)

# Interpolate Python values - automatically shell-escaped
filename = "my file with spaces.txt"
sh(f"wc -l {filename!r}")   # → wc -l 'my file with spaces.txt'
```

## Template interpolation

Use `${}` syntax (or plain f-strings) to embed Python values. `bunpy.shell` uses `shlex.quote` on every interpolated value so special characters cannot break the command.

```python
from bunpy.shell import sh

branch = "feat/new-feature"
message = "Release: v1.2.0"

# Safe - branch and message are quoted automatically
sh(f"git checkout -b {branch!r}")
sh(f'git commit -m {message!r}')

# Lists expand to space-separated quoted args
files = ["src/main.py", "src/utils.py", "tests/test_main.py"]
sh(f"black {' '.join(files)!r}")
```

For multi-value interpolation, use `shell.args()` to join a list safely:

```python
from bunpy.shell import sh, args

targets = ["dist/", "build/", ".mypy_cache/"]
sh(f"rm -rf {args(targets)}")
# → rm -rf 'dist/' 'build/' '.mypy_cache/'
```

## Options

```python
result = sh(
    "pytest tests/ --tb=short",
    cwd=".",               # working directory
    env={"CI": "true"},    # extra env vars (merged)
    capture=True,          # return stdout/stderr instead of printing
    check=True,            # raise ShellError on non-zero exit
    timeout=60,            # kill after N seconds
)
```

| Option | Default | Description |
|--------|---------|-------------|
| `cwd` | `None` | Working directory |
| `env` | `None` | Extra env vars merged into process environment |
| `capture` | `False` | Capture stdout/stderr and return them |
| `check` | `False` | Raise `ShellError` on non-zero exit code |
| `timeout` | `None` | Timeout in seconds |
| `quiet` | `False` | Suppress stdout/stderr - discard output silently |

## Return value

```python
result = sh("git status", capture=True)

result.stdout      # str - captured standard output
result.stderr      # str - captured standard error
result.returncode  # int - exit code
result.ok          # bool - True when returncode == 0
```

When `capture=False` (default), output goes directly to the terminal and `stdout`/`stderr` are empty strings.

## Exit code checking

```python
from bunpy.shell import sh, ShellError

# Manual check
result = sh("git push")
if not result.ok:
    print(f"push failed (exit {result.returncode})")

# Automatic exception
try:
    sh("git push origin main", check=True)
except ShellError as e:
    print(e.stderr)
```

## Piping

Chain commands with standard shell pipe syntax in the command string:

```python
from bunpy.shell import sh

# Pipe in the shell string
result = sh("ps aux | grep python | wc -l", capture=True)
print(result.stdout.strip())

# Count errors in a log file
result = sh("grep ERROR app.log | wc -l", capture=True)
error_count = int(result.stdout.strip())
```

For pipelines where the left side is Python data, pipe via stdin:

```python
from bunpy.shell import sh

data = "unsorted\nlines\nalpha\nbeta"
result = sh("sort | uniq", capture=True, stdin=data)
print(result.stdout)
```

## Glob expansion

```python
from bunpy.shell import sh

# Shell globs work as expected
sh("rm -f dist/*.whl")
sh("cp src/**/*.py build/")

# Or use bunpy.glob to get the list in Python, then pass to sh
import bunpy.glob as glob

py_files = glob.find("src/**/*.py")
sh(f"wc -l {args(py_files)}", capture=True)
```

## Build script examples

```python
from bunpy.shell import sh, args

VERSION = "1.4.2"

def clean():
    sh("rm -rf dist/ build/ *.egg-info")

def lint():
    sh("ruff check src/", check=True)
    sh("mypy src/", check=True)

def test():
    sh("pytest tests/ -x -q", check=True)

def build():
    sh("python -m build", check=True)

def publish():
    wheels = sh("ls dist/*.whl", capture=True).stdout.split()
    sh(f"twine upload {args(wheels)}", check=True)

def tag():
    sh(f'git tag -a v{VERSION!r} -m "Release v{VERSION}"', check=True)
    sh("git push --tags", check=True)

clean()
lint()
test()
build()
tag()
publish()
```

## Git workflow automation

```python
from bunpy.shell import sh

def current_branch() -> str:
    return sh("git rev-parse --abbrev-ref HEAD", capture=True).stdout.strip()

def sync_main():
    sh("git fetch origin", check=True)
    sh("git checkout main", check=True)
    sh("git merge --ff-only origin/main", check=True)

def create_release(version: str):
    branch = f"release/{version}"
    sh(f"git checkout -b {branch!r}", check=True)

    # Bump version in pyproject.toml
    sh(f"sed -i '' 's/version = .*/version = \"{version}\"/' pyproject.toml")

    sh(f'git add pyproject.toml')
    sh(f'git commit -m {f"chore: bump version to {version}"!r}', check=True)
    sh(f'git push -u origin {branch!r}', check=True)

create_release("2.0.0")
```

## File operations

```python
from bunpy.shell import sh

src_dir = "src/mypackage"
dst_dir = "dist/mypackage"

# Copy directory tree
sh(f"cp -r {src_dir!r} {dst_dir!r}")

# Compress
archive = "mypackage-1.0.tar.gz"
sh(f"tar -czf {archive!r} {dst_dir!r}")

# Check archive contents
result = sh(f"tar -tzf {archive!r}", capture=True)
for entry in result.stdout.splitlines():
    print(entry)
```

## Async shell

```python
import asyncio
from bunpy.shell import ash

async def parallel_lint():
    results = await asyncio.gather(
        ash("ruff check src/", capture=True),
        ash("mypy src/", capture=True),
        ash("bandit -r src/ -q", capture=True),
    )
    for r in results:
        if not r.ok:
            print(r.stdout)
            print(r.stderr)

asyncio.run(parallel_lint())
```

### ash(command, **options) → Coroutine[ShellResult]

Async version of `sh`. Same signature and return type. Runs the shell command without blocking the event loop.

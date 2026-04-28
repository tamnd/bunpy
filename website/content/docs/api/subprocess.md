---
title: bunpy.spawn - Subprocess
description: Run subprocesses, capture output, pipe stdin/stdout, set timeouts and environment, and use async subprocess in bunpy.
weight: 12
---

```python
from bunpy.spawn import spawn
```

`bunpy.spawn` runs external programs as subprocesses. It is lower-level than `bunpy.shell` - you get explicit control over argument lists, file descriptors, environment, and working directory without going through a shell.

## Basic usage

```python
from bunpy.spawn import spawn

# Run a command and wait for it to finish
result = spawn(["git", "status"])
print(result.stdout)   # captured stdout as str
print(result.returncode)

# Check for failure
result = spawn(["git", "push"])
if result.returncode != 0:
    print("push failed:", result.stderr)
```

### spawn(args, **options) → CompletedProcess

`args` is a list of strings (preferred) or a single string when `shell=True`.
Blocks until the process exits and returns a `CompletedProcess`.

| CompletedProcess field | Type | Description |
|------------------------|------|-------------|
| `returncode` | int | Exit code - 0 means success |
| `stdout` | str | Captured standard output |
| `stderr` | str | Captured standard error |
| `args` | list | The command that was run |
| `ok` | bool | `True` when `returncode == 0` |

## Options

```python
result = spawn(
    ["python", "-m", "pytest", "tests/"],
    cwd="/path/to/project",
    env={"PYTHONPATH": "src"},
    timeout=120,        # seconds
    stdin="y\n",        # pipe this string to stdin
    capture=True,       # capture stdout/stderr (default True)
    check=True,         # raise SpawnError on non-zero exit
)
```

| Option | Default | Description |
|--------|---------|-------------|
| `cwd` | `None` (inherit) | Working directory for the subprocess |
| `env` | `None` (inherit) | Environment variables - merged with current env unless `env_replace=True` |
| `env_replace` | `False` | Replace environment entirely instead of merging |
| `timeout` | `None` | Kill after this many seconds; raises `TimeoutError` |
| `stdin` | `None` | String or bytes piped to stdin; `None` inherits parent stdin |
| `capture` | `True` | Capture stdout and stderr; set `False` to let output go to terminal |
| `check` | `False` | Raise `SpawnError` if exit code is non-zero |
| `shell` | `False` | Pass `args` to the shell (`/bin/sh -c`) |

## Git workflow examples

```python
from bunpy.spawn import spawn

def git(*args) -> str:
    result = spawn(["git", *args], check=True)
    return result.stdout.strip()

branch = git("rev-parse", "--abbrev-ref", "HEAD")
sha    = git("rev-parse", "HEAD")
log    = git("log", "--oneline", "-10")

print(f"Branch: {branch}")
print(f"SHA:    {sha}")
print(log)
```

```python
# Check if the working tree is clean before deploying
def is_clean() -> bool:
    result = spawn(["git", "status", "--porcelain"])
    return result.stdout.strip() == ""

if not is_clean():
    raise SystemExit("Uncommitted changes - aborting deploy")
```

## Capturing stderr separately

```python
from bunpy.spawn import spawn

result = spawn(["mypy", "src/"], capture=True)
if not result.ok:
    print("Type errors:")
    for line in result.stderr.splitlines():
        print(" ", line)
```

## Piping stdin

```python
from bunpy.spawn import spawn

# Feed data to a process's stdin
result = spawn(["wc", "-l"], stdin="line1\nline2\nline3\n")
print(result.stdout.strip())   # "3"

# Use a subprocess to format code
with open("ugly.py") as f:
    source = f.read()

result = spawn(["black", "-"], stdin=source, capture=True)
formatted = result.stdout
```

## Running Python tools

```python
from bunpy.spawn import spawn
import sys

python = sys.executable   # same interpreter as bunpy

# Run a module
spawn([python, "-m", "pip", "install", "httpx"], check=True)

# Run a script with arguments
result = spawn([python, "scripts/migrate.py", "--env", "production"], check=True)
print(result.stdout)
```

## Timeout

```python
from bunpy.spawn import spawn, TimeoutError

try:
    result = spawn(["sleep", "60"], timeout=5)
except TimeoutError:
    print("Process took too long and was killed")
```

## Async subprocess

```python
import asyncio
from bunpy.spawn import aspawn

async def run_tests():
    # Run multiple test suites in parallel
    results = await asyncio.gather(
        aspawn(["pytest", "tests/unit/"]),
        aspawn(["pytest", "tests/integration/"]),
    )
    for r in results:
        if not r.ok:
            print("Failure:", r.stderr)

asyncio.run(run_tests())
```

### aspawn(args, **options) → Coroutine[CompletedProcess]

Same signature as `spawn`. Returns a coroutine that resolves to `CompletedProcess`.
The process runs in the asyncio event loop - no thread-pool blocking.

## Piping multiple commands

`bunpy.spawn` does not do shell piping natively. Use `Popen` for multi-stage pipelines, or reach for `bunpy.shell` for one-liners.

```python
from bunpy.spawn import Popen, PIPE

# ps aux | grep python
ps  = Popen(["ps", "aux"], stdout=PIPE)
grep = Popen(["grep", "python"], stdin=ps.stdout, stdout=PIPE)
ps.stdout.close()
output, _ = grep.communicate()
print(output.decode())
```

### Popen

`bunpy.spawn.Popen` is a thin alias for `subprocess.Popen` from the standard library. All standard-library options apply.

## Environment manipulation

```python
from bunpy.spawn import spawn
import os

# Add a variable while keeping the rest of the environment
result = spawn(
    ["node", "server.js"],
    env={"PORT": "9000"},    # merged - other env vars are preserved
)

# Fully isolated environment
result = spawn(
    ["env"],
    env={"HOME": os.environ["HOME"], "PATH": os.environ["PATH"]},
    env_replace=True,
)
```

## Error handling

```python
from bunpy.spawn import spawn, SpawnError

try:
    spawn(["git", "push", "origin", "main"], check=True)
except SpawnError as e:
    print(f"Command failed (exit {e.returncode})")
    print(e.stderr)
```

| Exception | When raised |
|-----------|-------------|
| `SpawnError` | `check=True` and exit code is non-zero |
| `TimeoutError` | Process exceeded `timeout` seconds |
| `FileNotFoundError` | The executable was not found in `PATH` |

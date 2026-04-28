---
title: bunpy.glob — Glob Patterns
description: Find files by glob pattern, recursive search, ignore rules, and async glob scanning in bunpy.
weight: 14
---

```python
import bunpy.glob as glob
```

`bunpy.glob` finds files and directories by pattern. It handles recursive `**` globbing, ignore patterns, and a streaming async interface for large directory trees — without spawning a subprocess or importing `pathlib` boilerplate.

## Finding files

```python
import bunpy.glob as glob

# All Python files in the current directory
files = glob.find("*.py")

# Recursive — all .py files under src/
files = glob.find("src/**/*.py")

# Multiple patterns
files = glob.find(["src/**/*.py", "tests/**/*.py"])
print(files)
# ["/project/src/main.py", "/project/src/util.py", ...]
```

### glob.find(pattern, cwd=None, ignore=None, dot=False) → list[str]

Returns a sorted list of absolute paths matching `pattern`.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `pattern` | — | A glob string or list of glob strings |
| `cwd` | `None` (process cwd) | Base directory to resolve relative patterns from |
| `ignore` | `None` | Pattern, list of patterns, or path to `.gitignore`-format file to exclude |
| `dot` | `False` | Include dotfiles (names starting with `.`) |

`**` matches zero or more directory components. `?` matches a single character. `[abc]` matches a character class.

## Checking a single path

```python
import bunpy.glob as glob

# Test whether a path matches a pattern
if glob.match("*.py", "script.py"):
    print("it's a Python file")

if glob.match("tests/**", "tests/unit/test_api.py"):
    print("in tests tree")
```

### glob.match(pattern, path) → bool

Returns `True` if `path` matches `pattern`. Patterns follow the same `**` rules as `glob.find`.

## Ignoring patterns

```python
import bunpy.glob as glob

# Exclude specific directories
files = glob.find(
    "**/*.py",
    ignore=["__pycache__/**", "*.pyc", ".venv/**", "dist/**"],
)

# Exclude by .gitignore rules
files = glob.find(
    "**/*.py",
    ignore=".gitignore",
)

# Combine — pass a list with a file path and extra patterns
files = glob.find(
    "**/*.py",
    ignore=[".gitignore", "scratch/**"],
)
```

When `ignore` is a string ending in `.gitignore` or an actual file path that exists, the file is parsed as a `.gitignore`-format ignore list.

## Scanning a directory tree

```python
import bunpy.glob as glob

# All files in a directory (non-recursive)
entries = glob.scan("src/")
for entry in entries:
    print(entry.name, entry.is_file, entry.size)

# Recursive scan — yields DirEntry objects
for entry in glob.scanAll("src/"):
    if entry.is_file and entry.name.endswith(".py"):
        print(entry.path)
```

### glob.scan(directory) → list[DirEntry]

Lists the immediate children of `directory`.

### glob.scanAll(directory, ignore=None) → Iterator[DirEntry]

Walks the entire tree under `directory` depth-first.

| DirEntry field | Type | Description |
|---------------|------|-------------|
| `name` | str | File or directory name (no path) |
| `path` | str | Absolute path |
| `is_file` | bool | `True` if a regular file |
| `is_dir` | bool | `True` if a directory |
| `size` | int | File size in bytes (`0` for directories) |
| `mtime` | float | Last-modified Unix timestamp |

## Async glob

```python
import asyncio
import bunpy.glob as glob

async def index_project():
    # Async find — does not block event loop
    py_files = await glob.afind("src/**/*.py")
    print(f"Found {len(py_files)} Python files")

    # Async scan — yields entries as they arrive
    async for entry in glob.ascanAll("src/"):
        if entry.is_file:
            process(entry.path)

asyncio.run(index_project())
```

### glob.afind(pattern, **options) → Coroutine[list[str]]

Async version of `glob.find`. Runs the directory walk off the event loop thread.

### glob.ascanAll(directory, ignore=None) → AsyncIterator[DirEntry]

Async generator that yields `DirEntry` objects as the walk progresses.

## Real-world examples

### Find all Python files, exclude caches

```python
import bunpy.glob as glob

source_files = glob.find(
    "**/*.py",
    ignore=[
        "__pycache__/**",
        "*.pyc",
        ".venv/**",
        ".git/**",
        "dist/**",
        "build/**",
    ],
)

print(f"{len(source_files)} source files")
for f in source_files:
    print(f)
```

### Collect test files and run them

```python
import bunpy.glob as glob
from bunpy.shell import sh

test_files = glob.find("tests/**/test_*.py", ignore=".gitignore")

if not test_files:
    print("No tests found")
else:
    print(f"Running {len(test_files)} test files")
    from bunpy.shell import args
    sh(f"pytest {args(test_files)} -v", check=True)
```

### Scan directory tree and report large files

```python
import bunpy.glob as glob

LIMIT = 1 * 1024 * 1024   # 1 MB

large = [
    e for e in glob.scanAll(".")
    if e.is_file and e.size > LIMIT
]

large.sort(key=lambda e: e.size, reverse=True)
for e in large[:10]:
    mb = e.size / 1024 / 1024
    print(f"{mb:.1f} MB  {e.path}")
```

### Watch a glob pattern for new files

```python
import bunpy.glob as glob
import bunpy.file as file
import time

seen = set(glob.find("uploads/*.csv"))

print(f"Watching uploads/ for new CSV files ({len(seen)} existing)…")
while True:
    current = set(glob.find("uploads/*.csv"))
    new = current - seen
    for path in sorted(new):
        print(f"New file: {path}")
        process_csv(path)
    seen = current
    time.sleep(2)
```

### Build a file manifest

```python
import bunpy.glob as glob
import bunpy.file as file
import hashlib

def sha256(path: str) -> str:
    data = file.read(path, encoding=None)
    return hashlib.sha256(data).hexdigest()

sources = glob.find("src/**/*.py", ignore=".gitignore")
manifest = {
    path: sha256(path)
    for path in sources
}

file.writeJSON("dist/manifest.json", manifest)
print(f"Manifest written for {len(manifest)} files")
```

### Async parallel file processing

```python
import asyncio
import bunpy.glob as glob
import bunpy.file.async_ as afile

async def count_lines(path: str) -> tuple[str, int]:
    text = await afile.read(path)
    return path, text.count("\n")

async def main():
    files = await glob.afind("src/**/*.py", ignore=".gitignore")
    tasks = [count_lines(f) for f in files]
    results = await asyncio.gather(*tasks)

    total = sum(n for _, n in results)
    for path, n in sorted(results, key=lambda x: -x[1])[:5]:
        print(f"{n:6d}  {path}")
    print(f"Total: {total} lines across {len(files)} files")

asyncio.run(main())
```

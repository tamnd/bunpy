---
title: bunpy.file — File I/O
description: Read, write, stat, watch, and stream files from bunpy with a minimal, fast API built on Python 3.14.
weight: 10
---

```python
import bunpy.file as file
```

`bunpy.file` wraps Python 3.14's native I/O with convenience methods for the common 95 %: read a file in one call, write bytes or text atomically, stream large files without loading them into memory, and watch paths for changes.

## Reading files

```python
import bunpy.file as file

# Read entire file as text
text = file.read("README.md")

# Read as bytes
raw = file.read("image.png", encoding=None)

# Read JSON — parses and returns a dict/list
config = file.readJSON("pyproject.toml")
```

### file.read(path, encoding="utf-8") → str | bytes

Returns the full contents of `path`. Pass `encoding=None` to get raw `bytes`.
Raises `FileNotFoundError` if the path does not exist.

### file.readJSON(path) → dict | list

Reads the file and calls `json.loads`. Raises `json.JSONDecodeError` on bad input.

### file.readLines(path, encoding="utf-8") → list[str]

Returns a list of lines with newlines stripped.

```python
lines = file.readLines("access.log")
for line in lines:
    print(line)
```

## Writing files

```python
# Write text — creates parent dirs automatically
file.write("dist/output.txt", "hello world\n")

# Write bytes
file.write("dist/data.bin", b"\x00\x01\x02")

# Write JSON — pretty-printed by default
file.writeJSON("dist/report.json", {"ok": True, "count": 42})

# Append to an existing file
file.append("app.log", "[INFO] server started\n")
```

### file.write(path, data, encoding="utf-8")

Writes `data` (str or bytes) to `path`. Missing parent directories are created.
The write is atomic on POSIX — the file is written to a temp path and renamed.

### file.writeJSON(path, data, indent=2)

Serialises `data` with `json.dumps(indent=indent)` then writes to `path`.

### file.append(path, data, encoding="utf-8")

Opens `path` in append mode and writes `data`. Creates the file if absent.

## Checking existence and stat

```python
if file.exists("config.json"):
    cfg = file.readJSON("config.json")
else:
    cfg = {}

info = file.stat("large_dataset.csv")
print(info.size)       # bytes
print(info.mtime)      # float — Unix timestamp
print(info.is_file)    # bool
print(info.is_dir)     # bool
```

### file.exists(path) → bool

Returns `True` if the path exists (file or directory).

### file.stat(path) → StatResult

| Field | Type | Description |
|-------|------|-------------|
| `size` | int | File size in bytes |
| `mtime` | float | Last-modified Unix timestamp |
| `ctime` | float | Creation (Windows) or metadata-change (POSIX) timestamp |
| `is_file` | bool | `True` if a regular file |
| `is_dir` | bool | `True` if a directory |
| `mode` | int | POSIX permission bits |

## Streaming large files

Loading a 2 GB CSV into memory with `file.read` will exhaust RAM.
Use `file.stream` instead — it returns an iterator of chunks.

```python
import bunpy.file as file

# Stream 64 KB chunks
with file.stream("large_dataset.csv", chunk_size=65536) as stream:
    for chunk in stream:
        process(chunk)

# Stream lines — useful for log parsing
with file.streamLines("access.log") as lines:
    for line in lines:
        if "ERROR" in line:
            print(line)
```

### file.stream(path, chunk_size=65536, encoding=None) → ContextManager[Iterator[bytes | str]]

Yields successive `chunk_size`-byte chunks. Pass `encoding="utf-8"` to decode each chunk to str.

### file.streamLines(path, encoding="utf-8") → ContextManager[Iterator[str]]

Yields one decoded line at a time without loading the whole file.

## Async file operations

All synchronous functions have async counterparts in `bunpy.file.async_`:

```python
import asyncio
import bunpy.file.async_ as afile

async def build():
    config = await afile.readJSON("pyproject.toml")
    name = config["project"]["name"]

    await afile.write(f"dist/{name}.txt", f"built: {name}\n")
    print("done")

asyncio.run(build())
```

The async API mirrors the sync API exactly — every function name is the same.
Reads and writes are dispatched to a thread-pool executor so they never block the event loop.

## Watching for changes

```python
import bunpy.file as file

# Watch a single file
with file.watch("config.json") as watcher:
    for event in watcher:
        print(event.path, event.kind)   # "modify" | "create" | "delete"
        reload_config()

# Watch a directory (recursive)
with file.watch("src/", recursive=True) as watcher:
    for event in watcher:
        if event.path.endswith(".py"):
            run_tests()
```

### file.watch(path, recursive=False) → ContextManager[Iterator[WatchEvent]]

Wraps the OS-native watcher (FSEvents on macOS, inotify on Linux, ReadDirectoryChangesW on Windows).

| WatchEvent field | Type | Description |
|-----------------|------|-------------|
| `path` | str | Absolute path of the changed file |
| `kind` | str | `"create"`, `"modify"`, or `"delete"` |
| `is_dir` | bool | `True` if the event target is a directory |

## Real-world examples

### Read a JSON config with fallback defaults

```python
import bunpy.file as file

DEFAULTS = {"host": "127.0.0.1", "port": 8000, "debug": False}

def load_config(path="config.json") -> dict:
    if not file.exists(path):
        return DEFAULTS
    cfg = file.readJSON(path)
    return {**DEFAULTS, **cfg}

config = load_config()
print(config["port"])
```

### Write a rotating log file

```python
import bunpy.file as file
from datetime import date

def log(message: str):
    path = f"logs/{date.today()}.log"
    file.append(path, message + "\n")

log("server started")
log("request received")
```

### Stream and transform a large CSV

```python
import bunpy.file as file
import csv
import io

total = 0
with file.stream("sales.csv", encoding="utf-8") as chunks:
    buf = ""
    for chunk in chunks:
        buf += chunk
        while "\n" in buf:
            line, buf = buf.split("\n", 1)
            row = next(csv.reader([line]))
            if row and row[0] != "date":     # skip header
                total += float(row[2])

print(f"Total sales: {total:.2f}")
```

### Hot-reload config on change

```python
import bunpy.file as file

config = file.readJSON("config.json")

with file.watch("config.json") as watcher:
    print("Watching config.json for changes…")
    for event in watcher:
        if event.kind in ("modify", "create"):
            config = file.readJSON("config.json")
            print("Config reloaded:", config)
```

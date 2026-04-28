---
title: bunpy.node.fs
description: Node.js-compatible file system module.
---

```python
from bunpy.node import fs
from bunpy.node.fs import readFileSync, writeFileSync
```

## Sync functions

| Function | Description |
|----------|-------------|
| `readFileSync(path, encoding?)` | Read file; returns `bytes` or `str` if encoding given |
| `writeFileSync(path, data)` | Write string or bytes to file |
| `appendFileSync(path, data)` | Append to file |
| `existsSync(path)` | Return `True` if path exists |
| `mkdirSync(path, recursive=False)` | Create directory |
| `rmdirSync(path, recursive=False)` | Remove directory |
| `unlinkSync(path)` | Delete file |
| `renameSync(src, dst)` | Rename/move file |
| `copyFileSync(src, dst)` | Copy file |
| `readdirSync(path)` | List directory entries |
| `statSync(path)` | Return stat object |
| `lstatSync(path)` | Like statSync, but for symlinks |
| `realpathSync(path)` | Resolve symlinks |

## Async functions

Each sync function has an async counterpart without the `Sync` suffix:

```python
content = await fs.readFile("data.txt", "utf8")
await fs.writeFile("out.txt", "hello")
entries = await fs.readdir(".")
```

## Stat object

```python
st = fs.statSync("file.txt")
st.size       # file size in bytes
st.isFile()   # True
st.isDirectory()  # False
st.mtime      # modification time (datetime)
```

## Examples

```python
from bunpy.node import fs

# Read
content = fs.readFileSync("config.json", "utf8")

# Write
fs.writeFileSync("output.txt", "hello world\n")

# List directory
for entry in fs.readdirSync("."):
    print(entry)

# Create directory
fs.mkdirSync("logs", recursive=True)
```

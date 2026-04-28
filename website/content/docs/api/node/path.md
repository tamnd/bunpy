---
title: bunpy.node.path
description: Node.js-compatible path manipulation module.
---

```python
from bunpy.node import path
from bunpy.node.path import join, resolve, dirname, basename
```

## Functions

| Function | Description | Example |
|----------|-------------|---------|
| `join(*parts)` | Join path segments | `join("/a", "b", "c")` → `/a/b/c` |
| `resolve(*parts)` | Resolve to absolute path | `resolve("../foo")` |
| `dirname(p)` | Directory part | `dirname("/a/b.txt")` → `/a` |
| `basename(p, ext?)` | File name part | `basename("/a/b.txt")` → `b.txt` |
| `extname(p)` | Extension | `extname("file.txt")` → `.txt` |
| `parse(p)` | Parse into parts dict | `{"root": "/", "dir": "/a", ...}` |
| `format(parts)` | Format from parts dict | inverse of `parse` |
| `isAbsolute(p)` | Is absolute path | `isAbsolute("/tmp")` → `True` |
| `normalize(p)` | Normalize `..` and `.` | `normalize("/a/../b")` → `/b` |
| `relative(from, to)` | Relative path between two paths | |
| `sep` | Path separator (`/` on POSIX, `\` on Windows) | |
| `delimiter` | PATH delimiter (`:` or `;`) | |

## Examples

```python
from bunpy.node import path

print(path.join("/usr", "local", "bin"))   # /usr/local/bin
print(path.dirname("/home/user/file.txt")) # /home/user
print(path.basename("/home/user/file.txt")) # file.txt
print(path.extname("archive.tar.gz"))      # .gz
print(path.isAbsolute("relative/path"))    # False
```

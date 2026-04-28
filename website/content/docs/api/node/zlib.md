---
title: bunpy.node.zlib
description: Node.js-compatible compression module.
---

```python
from bunpy.node import zlib
```

## Sync functions

| Function | Description |
|----------|-------------|
| `gzipSync(data)` | Gzip-compress bytes |
| `gunzipSync(data)` | Gzip-decompress bytes |
| `deflateSync(data)` | zlib deflate |
| `inflateSync(data)` | zlib inflate |
| `deflateRawSync(data)` | Raw deflate (no zlib header) |
| `inflateRawSync(data)` | Raw inflate |

## Stream factories

| Function | Description |
|----------|-------------|
| `createGzip()` | Gzip Transform stream |
| `createGunzip()` | Gunzip Transform stream |
| `createDeflate()` | Deflate Transform stream |
| `createInflate()` | Inflate Transform stream |

## Examples

```python
from bunpy.node import zlib

data = b"hello world" * 100
compressed = zlib.gzipSync(data)
original = zlib.gunzipSync(compressed)
assert original == data

# Using streams
gz = zlib.createGzip()
gz.write(data)
gz.flush()
result = gz.read()
```

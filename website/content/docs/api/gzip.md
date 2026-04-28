---
title: bunpy.gzip
description: Gzip compression and decompression.
---

```python
import bunpy.gzip as gzip
from bunpy.gzip import compress, decompress
```

## Functions

### compress(data, level=6) → bytes

Gzip-compress bytes or string. `level` is the zlib compression level (1–9).

```python
gzip.compress(b"hello world")
gzip.compress("hello world", level=9)
```

### decompress(data) → bytes

Decompress gzip-compressed bytes.

```python
original = gzip.decompress(compressed)
```

## Example

```python
import bunpy.gzip as gzip

data = b"hello " * 1000
compressed = gzip.compress(data)
print(f"{len(data)} → {len(compressed)} bytes")

assert gzip.decompress(compressed) == data
```

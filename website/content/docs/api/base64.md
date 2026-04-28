---
title: bunpy.base64
description: Base64 encoding and decoding.
---

```python
import bunpy.base64 as b64
# or
from bunpy.base64 import encode, decode, encodeURL, decodeURL
```

## Functions

### encode(data) → str

Encode bytes or string to standard Base64.

```python
b64.encode(b"hello world")       # "aGVsbG8gd29ybGQ="
b64.encode("hello world")        # "aGVsbG8gd29ybGQ="
```

### decode(s) → bytes

Decode a Base64 string to bytes.

```python
b64.decode("aGVsbG8gd29ybGQ=")   # b"hello world"
```

### encodeURL(data) → str

URL-safe Base64 (uses `-` and `_` instead of `+` and `/`, no padding).

```python
b64.encodeURL(b"\xfb\xff")   # "-_8"
```

### decodeURL(s) → bytes

Decode URL-safe Base64.

```python
b64.decodeURL("-_8")   # b"\xfb\xff"
```

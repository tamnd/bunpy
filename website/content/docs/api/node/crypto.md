---
title: bunpy.node.crypto
description: Node.js-compatible crypto module.
---

```python
from bunpy.node import crypto
from bunpy.node.crypto import randomBytes, randomUUID, createHash, createHmac
```

## Functions

### randomBytes(n) → bytes

```python
token = crypto.randomBytes(32)
```

### randomUUID() → str

```python
crypto.randomUUID()   # "550e8400-e29b-41d4-a716-446655440000"
```

### createHash(algorithm) → Hash

```python
digest = crypto.createHash("sha256").update("hello").digest("hex")
# "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
```

Hash methods:

| Method | Description |
|--------|-------------|
| `.update(data)` | Feed data (chainable) |
| `.digest("hex")` | Return hex-encoded digest |
| `.digest("binary")` | Return raw bytes |

### createHmac(algorithm, key) → Hmac

```python
mac = crypto.createHmac("sha256", "my-secret")
mac.update("message")
tag = mac.digest("hex")
```

## Supported algorithms

`sha256`, `sha512`, `sha1`, `md5`

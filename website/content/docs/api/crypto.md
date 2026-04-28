---
title: bunpy.crypto
description: Cryptographic hashing, HMAC, and password utilities.
---

```python
import bunpy.crypto as crypto
```

## Functions

### crypto.hash(algorithm, data) → str

Hash data and return a hex-encoded digest.

```python
crypto.hash("sha256", "hello")
# "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

crypto.hash("sha512", b"data")
crypto.hash("sha1", "legacy")
crypto.hash("md5", "legacy")
```

### crypto.hmac(algorithm, key, data) → str

Compute an HMAC and return a hex-encoded digest.

```python
crypto.hmac("sha256", "my-secret-key", "message")
```

### crypto.randomBytes(n) → bytes

Generate `n` cryptographically secure random bytes.

```python
token = crypto.randomBytes(32)
```

### crypto.randomUUID() → str

Generate a RFC 4122 v4 UUID.

```python
crypto.randomUUID()   # "550e8400-e29b-41d4-a716-446655440000"
```

### crypto.hashPassword(password) → str

Hash a password with bcrypt (cost 12).

```python
hashed = crypto.hashPassword("my-password")
```

### crypto.verifyPassword(password, hash) → bool

Verify a password against a bcrypt hash.

```python
ok = crypto.verifyPassword("my-password", hashed)
```

## Supported algorithms

`sha256`, `sha512`, `sha1`, `md5`, `sha384`, `sha3_256`, `sha3_512`

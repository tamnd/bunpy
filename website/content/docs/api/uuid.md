---
title: bunpy.uuid
description: UUID generation.
---

```python
import bunpy.uuid as uuid
from bunpy.uuid import v4, v5
```

## Functions

### uuid.v4() → str

Generate a random UUID v4.

```python
uuid.v4()   # "550e8400-e29b-41d4-a716-446655440000"
```

### uuid.v5(namespace, name) → str

Generate a deterministic UUID v5 (SHA-1 namespace hash).

```python
uuid.v5("dns", "example.com")
```

Standard namespace constants: `uuid.NAMESPACE_DNS`, `uuid.NAMESPACE_URL`,
`uuid.NAMESPACE_OID`, `uuid.NAMESPACE_X500`.

### uuid.parse(s) → bytes

Parse a UUID string into 16 bytes.

```python
uuid.parse("550e8400-e29b-41d4-a716-446655440000")
```

### uuid.stringify(b) → str

Format 16 bytes as a UUID string.

```python
uuid.stringify(b"\x55\x0e\x84\x00...")
```

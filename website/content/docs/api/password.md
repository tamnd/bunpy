---
title: bunpy.password - Password Hashing
description: Bcrypt and Argon2id password hashing, verification, timing-safe compare, and migration from legacy hashes in bunpy.
weight: 17
---

```python
import bunpy.password as password
```

`bunpy.password` provides secure password hashing with bcrypt and Argon2id. Both algorithms are slow by design, making brute-force attacks expensive. Use Argon2id for new projects - it is the winner of the Password Hashing Competition and the current OWASP recommendation.

## Quick start

```python
import bunpy.password as password

# Hash a password
hashed = password.hash("hunter2")

# Verify later
ok = password.verify("hunter2", hashed)   # True
bad = password.verify("wrong", hashed)    # False
```

## Bcrypt

### password.bcrypt.hash(plain, cost=12) → str

Hashes `plain` with bcrypt and returns a 60-character string starting with `$2b$`.

```python
import bunpy.password as password

hashed = password.bcrypt.hash("my-password")
# "$2b$12$eImiTXuWVxfM37uY4JANjQ..."

# Higher cost = slower = harder to brute-force
hashed_slow = password.bcrypt.hash("my-password", cost=14)
```

| Cost | Approx. time on modern hardware |
|------|---------------------------------|
| 10   | ~65 ms |
| 12   | ~250 ms (default) |
| 13   | ~500 ms |
| 14   | ~1 s |

Cost 12 is the OWASP-recommended minimum as of 2024.
Increase cost as hardware improves - old hashes with lower cost remain valid and can be re-hashed on next login.

### password.bcrypt.verify(plain, hashed) → bool

Returns `True` if `plain` matches `hashed`. The comparison is constant-time - it takes the same amount of time regardless of where the strings differ, preventing timing attacks.

```python
ok = password.bcrypt.verify("my-password", hashed)  # True
```

### password.bcrypt.needsRehash(hashed, cost=12) → bool

Returns `True` if the hash was created with a cost factor below `cost`.
Use this to transparently upgrade hashes at login time.

```python
if password.bcrypt.needsRehash(user.password_hash, cost=13):
    user.password_hash = password.bcrypt.hash(plain_password, cost=13)
    db.save(user)
```

## Argon2id

Argon2id is memory-hard - attackers need large amounts of RAM per attempt, not just fast CPUs.
It is the recommended algorithm for new applications.

### password.argon2.hash(plain, time_cost=3, memory_cost=65536, parallelism=4) → str

Hashes `plain` and returns a PHC-format string.

```python
import bunpy.password as password

hashed = password.argon2.hash("my-password")
# "$argon2id$v=19$m=65536,t=3,p=4$..."

# Tune for your hardware
hashed = password.argon2.hash(
    "my-password",
    time_cost=4,       # iterations
    memory_cost=131072, # 128 MB in KiB
    parallelism=4,
)
```

| Parameter | Default | Meaning |
|-----------|---------|---------|
| `time_cost` | 3 | Number of iterations |
| `memory_cost` | 65536 | Memory usage in KiB (64 MB default) |
| `parallelism` | 4 | Parallel threads |

OWASP recommendation: `time_cost=3, memory_cost=64MB, parallelism=4` as a minimum.

### password.argon2.verify(plain, hashed) → bool

Returns `True` if `plain` matches `hashed`. Constant-time comparison.

```python
ok = password.argon2.verify("my-password", hashed)  # True
```

### password.argon2.needsRehash(hashed) → bool

Returns `True` if the hash parameters differ from the current defaults.

## Top-level convenience API

If you do not care which algorithm is used, the top-level `password.hash` and `password.verify` use Argon2id by default:

```python
import bunpy.password as password

hashed = password.hash("my-password")           # Argon2id
ok = password.verify("my-password", hashed)     # True

# Switch algorithm
hashed_bcrypt = password.hash("my-password", algorithm="bcrypt")
ok = password.verify("my-password", hashed_bcrypt)  # True - auto-detected
```

`password.verify` detects the algorithm from the hash string prefix (`$2b$` for bcrypt, `$argon2id$` for Argon2id).

## Timing-safe compare

For comparing tokens, API keys, or any secret string where you must not leak information through timing:

```python
import bunpy.password as password

# Compare two strings in constant time
if password.timingSafeEqual(provided_token, expected_token):
    grant_access()
```

### password.timingSafeEqual(a, b) → bool

Compares `a` and `b` character by character in constant time.
Returns `False` immediately only after processing all characters - it never short-circuits on a mismatch.
Both `str` and `bytes` are accepted; a `str` is encoded to UTF-8 before comparison.

## User registration

```python
import bunpy.password as password
import bunpy.sql as sql

db = sql.open("app.sqlite")
db.exec("""
    CREATE TABLE IF NOT EXISTS users (
        id       INTEGER PRIMARY KEY,
        email    TEXT UNIQUE NOT NULL,
        password TEXT NOT NULL
    )
""")

def register(email: str, plain: str):
    if len(plain) < 8:
        raise ValueError("Password must be at least 8 characters")

    hashed = password.hash(plain)   # Argon2id by default
    db.exec("INSERT INTO users (email, password) VALUES (?, ?)", email, hashed)
    print(f"User {email} registered")

register("alice@example.com", "correct-horse-battery-staple")
```

## Login verification

```python
import bunpy.password as password
import bunpy.sql as sql

db = sql.open("app.sqlite")

def login(email: str, plain: str) -> bool:
    user = db.queryOne("SELECT * FROM users WHERE email = ?", email)
    if user is None:
        # Run hash anyway to prevent user-enumeration via timing
        password.hash("dummy-work")
        return False

    ok = password.verify(plain, user["password"])
    if ok and password.argon2.needsRehash(user["password"]):
        # Transparently upgrade to current params
        new_hash = password.hash(plain)
        db.exec("UPDATE users SET password = ? WHERE id = ?", new_hash, user["id"])

    return ok

if login("alice@example.com", "correct-horse-battery-staple"):
    print("Welcome!")
else:
    print("Invalid credentials")
```

## Migration from MD5 / SHA-1

Legacy systems often store passwords as unsalted MD5 or SHA-1 hashes.
Migrate them transparently at login time:

```python
import hashlib
import bunpy.password as password
import bunpy.sql as sql

db = sql.open("legacy.sqlite")

def is_md5(h: str) -> bool:
    return len(h) == 32 and all(c in "0123456789abcdef" for c in h)

def login_with_migration(email: str, plain: str) -> bool:
    user = db.queryOne("SELECT * FROM users WHERE email = ?", email)
    if user is None:
        password.hash("dummy")   # constant-time dummy
        return False

    stored = user["password"]

    if is_md5(stored):
        # Legacy MD5 check
        md5 = hashlib.md5(plain.encode()).hexdigest()
        if not password.timingSafeEqual(md5, stored):
            return False
        # Migrate to Argon2id
        new_hash = password.hash(plain)
        db.exec("UPDATE users SET password = ? WHERE id = ?", new_hash, user["id"])
        print(f"Migrated {email} from MD5 to Argon2id")
        return True

    return password.verify(plain, stored)
```

## Reference

| Function | Description |
|----------|-------------|
| `password.hash(plain, algorithm="argon2id")` | Hash with Argon2id (default) or bcrypt |
| `password.verify(plain, hashed)` | Verify, auto-detect algorithm |
| `password.timingSafeEqual(a, b)` | Constant-time string comparison |
| `password.bcrypt.hash(plain, cost=12)` | Hash with bcrypt |
| `password.bcrypt.verify(plain, hashed)` | Verify bcrypt hash |
| `password.bcrypt.needsRehash(hashed, cost=12)` | Check if rehash needed |
| `password.argon2.hash(plain, ...)` | Hash with Argon2id |
| `password.argon2.verify(plain, hashed)` | Verify Argon2id hash |
| `password.argon2.needsRehash(hashed)` | Check if params outdated |

---
title: bunpy.asyncio
description: Async/await utilities on top of goroutines.
---

```python
import bunpy.asyncio as asyncio
```

bunpy's asyncio module wraps the standard library `asyncio` with
goroutine-backed primitives. It is API-compatible with `asyncio` for the
common subset used in web and I/O applications.

## Functions

### asyncio.run(coro)

Run a coroutine and return its result. Entry point for async code.

```python
async def main():
    return 42

result = asyncio.run(main())
```

### asyncio.gather(*coros)

Run coroutines concurrently and collect results.

```python
async def fetch_all(urls):
    return await asyncio.gather(*[fetch_one(u) for u in urls])
```

### asyncio.create_task(coro)

Schedule a coroutine as a background task.

```python
task = asyncio.create_task(background_job())
result = await task
```

### asyncio.sleep(seconds)

Suspend the current coroutine for `seconds` (float allowed).

```python
await asyncio.sleep(0.5)
```

### asyncio.wait_for(coro, timeout)

Run a coroutine with a timeout; raises `asyncio.TimeoutError` if exceeded.

```python
try:
    result = await asyncio.wait_for(slow_op(), timeout=5.0)
except asyncio.TimeoutError:
    print("timed out")
```

## Example

```python
import bunpy.asyncio as asyncio

async def fetch_user(uid):
    resp = fetch(f"https://api.example.com/users/{uid}")
    return resp.json()

async def main():
    users = await asyncio.gather(
        fetch_user(1),
        fetch_user(2),
        fetch_user(3),
    )
    for u in users:
        print(u["name"])

asyncio.run(main())
```

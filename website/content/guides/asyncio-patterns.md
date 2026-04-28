---
title: Async patterns with asyncio
description: Write concurrent Python with async/await, gather parallel tasks, use TaskGroup, handle timeouts and cancellation, and run sync code in a thread executor.
---

## Basics: async def and await

An `async def` function is a coroutine. Calling it returns a coroutine object — it does not run until you `await` it or schedule it on an event loop:

```python
import asyncio

async def fetch_price(symbol: str) -> float:
    await asyncio.sleep(0.1)   # simulate I/O
    prices = {"AAPL": 182.5, "GOOG": 175.3, "MSFT": 415.1}
    return prices.get(symbol, 0.0)

async def main() -> None:
    price = await fetch_price("AAPL")
    print(f"AAPL: ${price}")

asyncio.run(main())
```

`asyncio.run` creates a new event loop, runs the coroutine to completion, and closes the loop. Always use it as the entry point — never call `loop.run_until_complete` manually in new code.

## gather: run tasks in parallel

`asyncio.gather` schedules multiple coroutines concurrently and collects their return values in order:

```python
import asyncio
import time

async def fetch(symbol: str, delay: float) -> tuple[str, float]:
    await asyncio.sleep(delay)
    prices = {"AAPL": 182.5, "GOOG": 175.3, "MSFT": 415.1, "TSLA": 245.0}
    return symbol, prices.get(symbol, 0.0)

async def main() -> None:
    start = time.perf_counter()

    # All four fetches run concurrently — total time is max(delays), not sum
    results = await asyncio.gather(
        fetch("AAPL", 0.3),
        fetch("GOOG", 0.1),
        fetch("MSFT", 0.2),
        fetch("TSLA", 0.15),
    )

    elapsed = time.perf_counter() - start
    for symbol, price in results:
        print(f"{symbol}: ${price}")
    print(f"Done in {elapsed:.2f}s")   # ~0.3s, not ~0.75s

asyncio.run(main())
```

If one coroutine raises, `gather` cancels the others by default. Pass `return_exceptions=True` to get exceptions as values instead:

```python
import asyncio

async def might_fail(n: int) -> int:
    if n == 2:
        raise ValueError(f"Bad number: {n}")
    return n * 10

async def main() -> None:
    results = await asyncio.gather(
        might_fail(1), might_fail(2), might_fail(3),
        return_exceptions=True,
    )
    for result in results:
        if isinstance(result, Exception):
            print(f"Error: {result}")
        else:
            print(f"OK: {result}")

asyncio.run(main())
```

## TaskGroup (Python 3.11+)

`asyncio.TaskGroup` is the structured concurrency alternative to `gather`. All tasks are cancelled if any task raises, and errors are reported together as an `ExceptionGroup`:

```python
import asyncio

async def fetch_user(user_id: int) -> dict:
    await asyncio.sleep(0.05)
    return {"id": user_id, "name": f"User {user_id}"}

async def main() -> None:
    users: list[dict] = []

    async with asyncio.TaskGroup() as tg:
        tasks = [tg.create_task(fetch_user(i)) for i in range(1, 6)]

    # all tasks are done here
    for task in tasks:
        users.append(task.result())

    print(users)

asyncio.run(main())
```

TaskGroup enforces that you do not leak background tasks — the `async with` block does not exit until every task created inside it has finished or been cancelled.

## Timeout with asyncio.timeout

```python
import asyncio

async def slow_operation() -> str:
    await asyncio.sleep(5)
    return "done"

async def main() -> None:
    try:
        async with asyncio.timeout(1.0):
            result = await slow_operation()
    except TimeoutError:
        print("Operation timed out after 1s")

asyncio.run(main())
```

For Python 3.10 and earlier, use `asyncio.wait_for`:

```python
import asyncio

async def main() -> None:
    try:
        result = await asyncio.wait_for(slow_operation(), timeout=1.0)
    except asyncio.TimeoutError:
        print("Timed out")

asyncio.run(main())
```

## Cancellation

Tasks can be cancelled from outside. Always clean up in a `finally` block or by catching `asyncio.CancelledError`:

```python
import asyncio

async def worker(name: str) -> None:
    try:
        print(f"{name}: starting")
        while True:
            await asyncio.sleep(0.5)
            print(f"{name}: tick")
    except asyncio.CancelledError:
        print(f"{name}: cancelled, cleaning up")
        raise   # always re-raise CancelledError
    finally:
        print(f"{name}: done")

async def main() -> None:
    task = asyncio.create_task(worker("background"))

    await asyncio.sleep(1.8)
    task.cancel()

    try:
        await task
    except asyncio.CancelledError:
        pass

asyncio.run(main())
```

## Async context managers

Implement `__aenter__` and `__aexit__` (or use `@asynccontextmanager`) for resources that need async setup and teardown:

```python
import asyncio
from contextlib import asynccontextmanager
from typing import AsyncGenerator

@asynccontextmanager
async def managed_connection(host: str) -> AsyncGenerator[dict, None]:
    print(f"Connecting to {host}")
    conn = {"host": host, "alive": True}   # simulated connection object
    try:
        yield conn
    finally:
        conn["alive"] = False
        print(f"Disconnected from {host}")

async def main() -> None:
    async with managed_connection("db.example.com") as conn:
        print(f"Using connection: {conn}")
        await asyncio.sleep(0.1)   # do some work

asyncio.run(main())
```

## Async generators

Async generators yield values lazily, one at a time, from an async source:

```python
import asyncio
from typing import AsyncGenerator

async def paginate_api(endpoint: str, pages: int = 3) -> AsyncGenerator[list[dict], None]:
    for page in range(1, pages + 1):
        await asyncio.sleep(0.05)   # simulate network call
        yield [{"page": page, "item": i} for i in range(5)]

async def main() -> None:
    total = 0
    async for batch in paginate_api("/items", pages=4):
        total += len(batch)
        print(f"Received {len(batch)} items, total so far: {total}")

asyncio.run(main())
```

Use `async for` to consume them. Use `aiostream` or manual buffering if you need to fan them out.

## Running sync code in an executor

Blocking I/O or CPU-bound work inside a coroutine blocks the event loop. Offload it with `loop.run_in_executor`:

```python
import asyncio
import time
from concurrent.futures import ThreadPoolExecutor, ProcessPoolExecutor

def blocking_read(path: str) -> str:
    time.sleep(0.2)   # simulate slow disk
    return f"contents of {path}"

def cpu_intensive(n: int) -> int:
    return sum(i * i for i in range(n))

async def main() -> None:
    loop = asyncio.get_running_loop()

    # thread pool for I/O-bound blocking calls
    with ThreadPoolExecutor(max_workers=4) as thread_pool:
        results = await asyncio.gather(
            loop.run_in_executor(thread_pool, blocking_read, "file1.txt"),
            loop.run_in_executor(thread_pool, blocking_read, "file2.txt"),
            loop.run_in_executor(thread_pool, blocking_read, "file3.txt"),
        )
    print(results)

    # process pool for CPU-bound work (bypasses the GIL)
    with ProcessPoolExecutor(max_workers=4) as proc_pool:
        heavy_results = await asyncio.gather(
            loop.run_in_executor(proc_pool, cpu_intensive, 1_000_000),
            loop.run_in_executor(proc_pool, cpu_intensive, 2_000_000),
        )
    print(heavy_results)

asyncio.run(main())
```

## Real-world: parallel API calls

Fetch data from multiple GitHub endpoints concurrently, with per-request timeouts and error handling:

```python
import asyncio
import httpx
import os
from typing import Any

GITHUB_TOKEN = os.environ.get("GITHUB_TOKEN", "")

async def github_get(client: httpx.AsyncClient, path: str) -> dict[str, Any]:
    async with asyncio.timeout(10.0):
        response = await client.get(path)
        response.raise_for_status()
        return response.json()

async def fetch_repo_stats(repos: list[str]) -> None:
    async with httpx.AsyncClient(
        base_url="https://api.github.com",
        headers={
            "Authorization": f"Bearer {GITHUB_TOKEN}",
            "Accept": "application/vnd.github+json",
        },
    ) as client:
        tasks = [github_get(client, f"/repos/{repo}") for repo in repos]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        for repo, result in zip(repos, results):
            if isinstance(result, Exception):
                print(f"{repo}: ERROR — {result}")
            else:
                print(f"{repo}: {result['stargazers_count']} stars, {result['open_issues_count']} open issues")

repos = [
    "python/cpython",
    "pallets/flask",
    "tiangolo/fastapi",
    "django/django",
]

asyncio.run(fetch_repo_stats(repos))
```

## Real-world: async database connection pool

Using `asyncpg` for PostgreSQL (the pattern works the same with `aiosqlite`, `aiomysql`, etc.):

```bash
bunpy add asyncpg
```

```python
import asyncio
import asyncpg

async def main() -> None:
    pool = await asyncpg.create_pool(
        dsn="postgresql://user:pass@localhost/mydb",
        min_size=2,
        max_size=10,
    )

    async def fetch_user(user_id: int) -> asyncpg.Record | None:
        async with pool.acquire() as conn:
            return await conn.fetchrow("SELECT id, username FROM users WHERE id = $1", user_id)

    # fetch 20 users concurrently, sharing the connection pool
    results = await asyncio.gather(*[fetch_user(i) for i in range(1, 21)])
    for row in results:
        if row:
            print(row["username"])

    await pool.close()

asyncio.run(main())
```

## Semaphore: limit concurrency

When making many outbound requests, a semaphore prevents opening thousands of connections at once:

```python
import asyncio
import httpx

async def fetch_with_limit(
    client: httpx.AsyncClient,
    sem: asyncio.Semaphore,
    url: str,
) -> int:
    async with sem:
        response = await client.get(url, timeout=10.0)
        return response.status_code

async def main() -> None:
    urls = [f"https://httpbin.org/delay/0?n={i}" for i in range(20)]
    sem = asyncio.Semaphore(5)   # at most 5 concurrent requests

    async with httpx.AsyncClient() as client:
        tasks = [fetch_with_limit(client, sem, url) for url in urls]
        statuses = await asyncio.gather(*tasks)

    print(statuses)

asyncio.run(main())
```

## Run the examples

```bash
bunpy parallel_api.py
bunpy async_queue.py
```

asyncio's model is single-threaded cooperative multitasking: only one coroutine runs at a time, but any coroutine can yield control at an `await` point. That means shared state is safe within a single event loop — but blocking for more than a few milliseconds without `await` stalls everything else.

---
title: Concurrency with concurrent.futures
description: Run parallel I/O and CPU-bound work with ThreadPoolExecutor and ProcessPoolExecutor, track progress with as_completed, and process files and images in parallel.
---

## Overview

`concurrent.futures` is in the standard library — no install needed. It gives you two executors:

- `ThreadPoolExecutor` — for I/O-bound work (network calls, disk reads, database queries). Threads share memory and the GIL, so they do not speed up CPU-bound Python code.
- `ProcessPoolExecutor` — for CPU-bound work (image processing, number crunching, compression). Separate processes bypass the GIL but have higher startup cost and cannot share memory.

## ThreadPoolExecutor basics

```python
from concurrent.futures import ThreadPoolExecutor, as_completed
import time

def fetch_price(symbol: str) -> tuple[str, float]:
    time.sleep(0.2)   # simulate network I/O
    prices = {"AAPL": 182.5, "GOOG": 175.3, "MSFT": 415.1, "TSLA": 245.0}
    return symbol, prices.get(symbol, 0.0)

symbols = ["AAPL", "GOOG", "MSFT", "TSLA"]

start = time.perf_counter()
with ThreadPoolExecutor(max_workers=4) as executor:
    results = list(executor.map(fetch_price, symbols))

elapsed = time.perf_counter() - start
for symbol, price in results:
    print(f"{symbol}: ${price}")
print(f"Done in {elapsed:.2f}s")   # ~0.2s, not ~0.8s
```

`executor.map` preserves input order. Results are yielded as each completes but collected in the original order.

## ProcessPoolExecutor basics

```python
from concurrent.futures import ProcessPoolExecutor
import time

def count_primes(limit: int) -> int:
    """Return the count of primes below limit (CPU-bound)."""
    sieve = bytearray([1]) * limit
    sieve[0] = sieve[1] = 0
    for i in range(2, int(limit ** 0.5) + 1):
        if sieve[i]:
            sieve[i * i::i] = bytearray(len(sieve[i * i::i]))
    return sum(sieve)

limits = [1_000_000, 2_000_000, 3_000_000, 4_000_000]

start = time.perf_counter()
with ProcessPoolExecutor(max_workers=4) as executor:
    counts = list(executor.map(count_primes, limits))

elapsed = time.perf_counter() - start
for limit, count in zip(limits, counts):
    print(f"Primes below {limit:,}: {count:,}")
print(f"Done in {elapsed:.2f}s")
```

Because `ProcessPoolExecutor` uses `multiprocessing` under the hood, all arguments and return values must be picklable. Lambdas and local functions are not — define them at module level.

## as_completed: process results as they arrive

`as_completed` yields futures in the order they finish, not the order they were submitted. Use it when you want to display progress or handle errors as soon as they happen:

```python
from concurrent.futures import ThreadPoolExecutor, as_completed
import httpx

def check_url(url: str) -> tuple[str, int]:
    try:
        response = httpx.get(url, timeout=5.0, follow_redirects=True)
        return url, response.status_code
    except Exception as exc:
        return url, -1

urls = [
    "https://python.org",
    "https://pypi.org",
    "https://docs.python.org",
    "https://github.com",
    "https://httpbin.org",
    "https://this-does-not-exist.invalid",
]

with ThreadPoolExecutor(max_workers=6) as executor:
    future_to_url = {executor.submit(check_url, url): url for url in urls}

    for future in as_completed(future_to_url):
        url, status = future.result()
        status_str = str(status) if status > 0 else "ERROR"
        print(f"{status_str:>5}  {url}")
```

## wait: block until a subset finishes

`wait` gives finer control than `as_completed`. You can wait for the first result, the first exception, or all results:

```python
from concurrent.futures import ThreadPoolExecutor, wait, FIRST_COMPLETED, ALL_COMPLETED
import time

def task(name: str, delay: float) -> str:
    time.sleep(delay)
    return f"{name} finished"

with ThreadPoolExecutor(max_workers=4) as executor:
    futures = {
        executor.submit(task, "alpha", 1.0),
        executor.submit(task, "beta", 0.3),
        executor.submit(task, "gamma", 0.7),
        executor.submit(task, "delta", 0.1),
    }

    done, pending = wait(futures, return_when=FIRST_COMPLETED)
    print(f"First finished: {done.pop().result()}")
    print(f"Still running: {len(pending)}")

    done, _ = wait(pending, return_when=ALL_COMPLETED)
    for f in done:
        print(f.result())
```

## Cancel pending futures

Submit work, then cancel tasks that have not started yet — useful for timeouts or early-exit patterns:

```python
from concurrent.futures import ThreadPoolExecutor, as_completed
import time

def slow_task(n: int) -> int:
    time.sleep(2)
    return n * n

with ThreadPoolExecutor(max_workers=2) as executor:
    futures = [executor.submit(slow_task, i) for i in range(10)]

    # cancel all but the first two (which are already running)
    for future in futures[2:]:
        cancelled = future.cancel()
        print(f"Future {futures.index(future)} cancelled: {cancelled}")

    for future in as_completed(futures[:2]):
        print(f"Result: {future.result()}")
```

`cancel()` returns `True` only if the task has not started. Running tasks cannot be cancelled — that is a fundamental limit of threads.

## map with timeout

`executor.map` accepts a `timeout` parameter. If any result takes longer than `timeout` seconds to retrieve, `TimeoutError` is raised:

```python
from concurrent.futures import ThreadPoolExecutor
import time

def fetch(n: int) -> int:
    time.sleep(n * 0.1)
    return n * 10

with ThreadPoolExecutor(max_workers=4) as executor:
    try:
        for result in executor.map(fetch, [1, 2, 3, 15, 4], timeout=1.0):
            print(result)
    except TimeoutError:
        print("One of the tasks took too long.")
```

## Real-world: parallel file processing

Read and summarize a directory of log files concurrently:

```python
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
import re

def analyze_log(path: Path) -> dict:
    error_count = 0
    warn_count = 0
    lines = 0

    with path.open() as f:
        for line in f:
            lines += 1
            if "ERROR" in line:
                error_count += 1
            elif "WARN" in line:
                warn_count += 1

    return {
        "file": path.name,
        "lines": lines,
        "errors": error_count,
        "warnings": warn_count,
    }

def summarize_logs(log_dir: str, workers: int = 8) -> list[dict]:
    paths = list(Path(log_dir).glob("*.log"))
    results = []

    with ThreadPoolExecutor(max_workers=workers) as executor:
        future_to_path = {executor.submit(analyze_log, p): p for p in paths}

        for future in as_completed(future_to_path):
            path = future_to_path[future]
            try:
                result = future.result()
                results.append(result)
                print(f"Processed {result['file']}: {result['errors']} errors")
            except Exception as exc:
                print(f"Failed to process {path.name}: {exc}")

    return sorted(results, key=lambda r: r["errors"], reverse=True)

# usage
# results = summarize_logs("/var/log/myapp")
# for r in results[:5]:
#     print(r)
```

## Real-world: parallel API calls with a thread pool

Pull data from a REST API for a list of IDs, honoring a rate limit:

```python
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import Semaphore
import httpx
import time

RATE_LIMIT = 10   # max 10 concurrent requests

sem = Semaphore(RATE_LIMIT)

def fetch_item(item_id: int) -> dict:
    with sem:
        response = httpx.get(
            f"https://jsonplaceholder.typicode.com/posts/{item_id}",
            timeout=10.0,
        )
        response.raise_for_status()
        return response.json()

def fetch_all(ids: list[int]) -> list[dict]:
    results: list[dict] = []

    with ThreadPoolExecutor(max_workers=20) as executor:
        future_to_id = {executor.submit(fetch_item, id_): id_ for id_ in ids}

        for future in as_completed(future_to_id):
            id_ = future_to_id[future]
            try:
                data = future.result()
                results.append(data)
            except Exception as exc:
                print(f"Item {id_} failed: {exc}")

    return results

start = time.perf_counter()
items = fetch_all(list(range(1, 51)))
elapsed = time.perf_counter() - start
print(f"Fetched {len(items)} items in {elapsed:.2f}s")
```

## Real-world: image resizing with processes

Image processing is CPU-bound — use `ProcessPoolExecutor` to use all cores:

```bash
bunpy add pillow
```

```python
from concurrent.futures import ProcessPoolExecutor, as_completed
from pathlib import Path
from PIL import Image

def resize_image(src: str, dest: str, width: int, height: int) -> str:
    img = Image.open(src)
    img_resized = img.resize((width, height), Image.LANCZOS)
    img_resized.save(dest, optimize=True, quality=85)
    return dest

def resize_all(input_dir: str, output_dir: str, size: tuple[int, int] = (800, 600)) -> None:
    src_dir = Path(input_dir)
    out_dir = Path(output_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    images = list(src_dir.glob("*.jpg")) + list(src_dir.glob("*.png"))
    if not images:
        print("No images found.")
        return

    jobs = [
        (str(p), str(out_dir / p.name), size[0], size[1])
        for p in images
    ]

    with ProcessPoolExecutor() as executor:
        futures = {
            executor.submit(resize_image, src, dest, w, h): src
            for src, dest, w, h in jobs
        }

        completed = 0
        for future in as_completed(futures):
            src = futures[future]
            try:
                dest = future.result()
                completed += 1
                print(f"[{completed}/{len(jobs)}] {Path(src).name} -> {dest}")
            except Exception as exc:
                print(f"Failed {src}: {exc}")

# resize_all("photos/", "photos_resized/", size=(1280, 720))
```

## Exception handling

Exceptions are stored on the future and re-raised when you call `.result()`:

```python
from concurrent.futures import ThreadPoolExecutor

def divide(a: int, b: int) -> float:
    if b == 0:
        raise ZeroDivisionError("cannot divide by zero")
    return a / b

with ThreadPoolExecutor(max_workers=2) as executor:
    futures = [
        executor.submit(divide, 10, 2),
        executor.submit(divide, 5, 0),
        executor.submit(divide, 8, 4),
    ]

for i, future in enumerate(futures):
    try:
        print(f"Result {i}: {future.result()}")
    except ZeroDivisionError as exc:
        print(f"Result {i}: error — {exc}")
```

## Choosing workers count

A rough starting point:

```python
import os
from concurrent.futures import ThreadPoolExecutor, ProcessPoolExecutor

cpu_count = os.cpu_count() or 4

# I/O-bound: more threads than CPUs is fine because threads spend most time waiting
io_workers = min(32, cpu_count * 4)

# CPU-bound: one process per core; more just adds context-switch overhead
cpu_workers = cpu_count

with ThreadPoolExecutor(max_workers=io_workers) as executor:
    pass

with ProcessPoolExecutor(max_workers=cpu_workers) as executor:
    pass
```

## Run the examples

```bash
bunpy parallel_api.py
bunpy resize_images.py
bunpy log_analyzer.py
```

`concurrent.futures` is the right tool when you want straightforward parallelism without the complexity of raw `threading` or `multiprocessing`. For async I/O (aiohttp, httpx async), prefer `asyncio` with `asyncio.gather` — threads add overhead that async avoids entirely when the whole stack is async-native.

---
title: Threading with bunpy
description: Spawn threads, run concurrent I/O with ThreadPoolExecutor, protect shared state with Lock, and build producer-consumer pipelines with Queue.
weight: 14
---

## Quick start: spawn a thread

```python
import threading
import time

def fetch_data(name: str, delay: float) -> None:
    time.sleep(delay)
    print(f"[{name}] done after {delay}s")

threads = [
    threading.Thread(target=fetch_data, args=(f"worker-{i}", i * 0.1))
    for i in range(5)
]

for t in threads:
    t.start()
for t in threads:
    t.join()

print("All threads finished")
```

Run it:

```bash
bunpy main.py
```

All five threads start nearly simultaneously. Because each sleeps for a different duration, they finish out of order - which is exactly the point. Threads are the right tool when your work spends most of its time waiting on I/O: network calls, disk reads, database queries. For CPU-bound work that needs true parallelism, see the [multiprocessing guide](../multiprocessing).

## Threading.Thread in detail

`Thread` accepts `target`, `args`, and `kwargs`. Set `daemon=True` when the thread should not block the process from exiting:

```python
import threading
import time

results: dict[str, int] = {}
lock = threading.Lock()

def compute(key: str, value: int) -> None:
    time.sleep(0.05)   # simulate work
    with lock:
        results[key] = value ** 2

threads = [
    threading.Thread(target=compute, args=(f"key-{i}", i), daemon=True)
    for i in range(8)
]

for t in threads:
    t.start()
for t in threads:
    t.join()   # wait even for daemon threads when you need their results

print(results)
```

`join()` blocks the calling thread until the target thread exits. Without `join()` the main thread may exit before the workers finish - safe only when they are daemons doing background bookkeeping you do not need to wait for.

## ThreadPoolExecutor for concurrent I/O

`concurrent.futures.ThreadPoolExecutor` is the higher-level API over raw `Thread`. It handles pool sizing, queuing, and exception propagation:

```python
from concurrent.futures import ThreadPoolExecutor, as_completed
import httpx
import time

URLS = [
    "https://httpbin.org/delay/1",
    "https://httpbin.org/delay/1",
    "https://httpbin.org/delay/1",
    "https://httpbin.org/uuid",
    "https://httpbin.org/ip",
]

def fetch(url: str) -> dict:
    response = httpx.get(url, timeout=10.0)
    response.raise_for_status()
    return {"url": url, "status": response.status_code, "size": len(response.content)}

t0 = time.perf_counter()
with ThreadPoolExecutor(max_workers=5) as executor:
    future_to_url = {executor.submit(fetch, url): url for url in URLS}

    for future in as_completed(future_to_url):
        url = future_to_url[future]
        try:
            result = future.result()
            print(f"{result['status']}  {result['size']:>6} bytes  {url}")
        except Exception as exc:
            print(f"ERROR  {url}: {exc}")

elapsed = time.perf_counter() - t0
print(f"\nFinished {len(URLS)} requests in {elapsed:.2f}s")
```

Five requests that would each take ~1 second sequentially complete together in just over 1 second. `as_completed` yields each future as it resolves so you can display progress or handle errors without waiting for the slowest request.

## Lock for shared state

Threads share the process's memory, which means a dict or counter updated from multiple threads can corrupt without synchronization. `threading.Lock` is the basic primitive:

```python
import threading

class SafeCounter:
    def __init__(self) -> None:
        self._value = 0
        self._lock = threading.Lock()

    def increment(self, n: int = 1) -> None:
        with self._lock:
            self._value += n

    @property
    def value(self) -> int:
        with self._lock:
            return self._value


counter = SafeCounter()

def worker(counter: SafeCounter, iterations: int) -> None:
    for _ in range(iterations):
        counter.increment()

threads = [threading.Thread(target=worker, args=(counter, 100_000)) for _ in range(10)]
for t in threads:
    t.start()
for t in threads:
    t.join()

print(f"Final count: {counter.value:,}")   # 1,000,000
```

`with lock:` is equivalent to `lock.acquire()` / `lock.release()` in a try/finally block. Always use the context manager form - it releases even if an exception is raised inside the block.

For read-heavy access patterns where writes are rare, `threading.RLock` (reentrant) or `threading.Event` may suit better, but `Lock` covers the majority of real-world cases.

## RLock and multiple lock levels

`RLock` (reentrant lock) allows the same thread to acquire the lock multiple times without deadlocking. Useful when a method that holds a lock calls another method that also acquires it:

```python
import threading

class Cache:
    def __init__(self) -> None:
        self._store: dict[str, str] = {}
        self._lock = threading.RLock()

    def get(self, key: str) -> str | None:
        with self._lock:
            return self._store.get(key)

    def set(self, key: str, value: str) -> None:
        with self._lock:
            self._store[key] = value

    def get_or_set(self, key: str, default: str) -> str:
        with self._lock:               # outer acquire
            existing = self.get(key)   # inner acquire - safe with RLock
            if existing is None:
                self.set(key, default)
                return default
            return existing

cache = Cache()
cache.set("color", "blue")
print(cache.get_or_set("color", "red"))    # blue
print(cache.get_or_set("size", "large"))   # large
```

## Producer-consumer with Queue

`queue.Queue` is designed for multi-threaded use. Unlike `list`, every method is thread-safe without a separate lock:

```python
import threading
import queue
import time
import random

def producer(q: queue.Queue, n: int) -> None:
    for i in range(n):
        item = f"item-{i}"
        q.put(item)
        time.sleep(random.uniform(0.01, 0.05))
    q.put(None)   # sentinel

def consumer(q: queue.Queue, name: str) -> None:
    processed = 0
    while True:
        item = q.get()
        if item is None:
            q.put(None)   # pass sentinel to next consumer
            break
        time.sleep(0.02)   # simulate processing
        processed += 1
        q.task_done()
    print(f"[{name}] processed {processed} items")

NUM_CONSUMERS = 3
q: queue.Queue = queue.Queue(maxsize=20)   # back-pressure: block when full

prod = threading.Thread(target=producer, args=(q, 30))
consumers = [
    threading.Thread(target=consumer, args=(q, f"consumer-{i}"))
    for i in range(NUM_CONSUMERS)
]

prod.start()
for c in consumers:
    c.start()

prod.join()
for c in consumers:
    c.join()

print("Pipeline done")
```

`maxsize` sets a cap on queue depth, which prevents the producer from outrunning consumers and exhausting memory. `q.task_done()` / `q.join()` let you wait until all items have been fully processed rather than just dequeued.

## Daemon threads for background work

A daemon thread does not prevent the process from exiting. Use daemons for heartbeats, log flushers, and metric collectors - work that should stop when the main program is done, not delay it:

```python
import threading
import time

def heartbeat(interval: float) -> None:
    while True:
        print(f"[heartbeat] alive at {time.strftime('%H:%M:%S')}")
        time.sleep(interval)

hb = threading.Thread(target=heartbeat, args=(2.0,), daemon=True)
hb.start()

# main work
for i in range(5):
    print(f"Main: step {i}")
    time.sleep(1)

print("Main: done - daemon thread exits automatically")
```

If `daemon=False` (the default), the process stays alive until every non-daemon thread finishes. Missing a `join()` on a non-daemon thread that never exits is a common source of hung processes.

## Thread-local storage

`threading.local()` gives each thread its own isolated copy of a variable. This is the standard pattern for per-thread database connections or request contexts in web frameworks:

```python
import threading
import time

local_data = threading.local()

def setup_and_work(name: str) -> None:
    local_data.name = name
    local_data.start = time.time()
    time.sleep(0.1)   # another thread runs here
    elapsed = time.time() - local_data.start
    print(f"[{local_data.name}] elapsed: {elapsed:.3f}s")

threads = [threading.Thread(target=setup_and_work, args=(f"t{i}",)) for i in range(4)]
for t in threads:
    t.start()
for t in threads:
    t.join()
```

Each thread sees its own `local_data.name` and `local_data.start` - writes from one thread never appear in another.

## Choosing between threading and asyncio

| Scenario | Recommended |
|---|---|
| Many concurrent HTTP requests | `asyncio` + `httpx` async |
| Wrapping a blocking library (psycopg2, boto3) | `ThreadPoolExecutor` |
| Existing sync codebase you cannot rewrite | `threading` |
| CPU-bound parallel work | `multiprocessing` |
| Mix of sync and async | `asyncio.run_in_executor` |

The GIL means Python threads do not accelerate CPU-bound work - two threads computing primes will not run faster than one. They do accelerate I/O-bound work because a thread waiting for a network response releases the GIL, allowing other threads to run.

## Run the examples

```bash
bunpy concurrent_fetch.py
bunpy safe_counter.py
bunpy producer_consumer.py
```

For higher-level pool management with both thread and process backends, see the [concurrent.futures guide](../concurrent-futures). For async I/O without threads, see the [asyncio patterns guide](../asyncio-patterns).

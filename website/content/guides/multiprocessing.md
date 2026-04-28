---
title: Multiprocessing with bunpy
description: Spawn processes, parallelize CPU-bound work with Pool, share state with Value and Queue, and bypass the GIL on bunpy.
weight: 13
---

## Quick start: spawn a process

```python
from multiprocessing import Process
import os

def worker(name: str) -> None:
    print(f"[{name}] PID={os.getpid()} running")

if __name__ == "__main__":
    p = Process(target=worker, args=("alpha",))
    p.start()
    p.join()
    print("Done")
```

Run it:

```bash
bunpy main.py
```

Each `Process` maps to a dedicated goroutine under the hood. bunpy does not use OS threads for the GIL workaround — it forks a new Python interpreter state per process, so CPU-bound work scales across all cores without contention.

## Pool.map for parallel computation

`Pool` manages a fixed set of worker processes. `map` splits work across them and collects results in order, making it the shortest path from a list of inputs to a list of outputs:

```python
from multiprocessing import Pool
import time

def is_prime(n: int) -> bool:
    if n < 2:
        return False
    if n == 2:
        return True
    if n % 2 == 0:
        return False
    for i in range(3, int(n ** 0.5) + 1, 2):
        if n % i == 0:
            return False
    return True

def count_primes_in_range(args: tuple[int, int]) -> int:
    start, end = args
    return sum(1 for n in range(start, end) if is_prime(n))

if __name__ == "__main__":
    # split 0..10_000_000 into 8 chunks
    chunk_size = 1_250_000
    ranges = [(i, i + chunk_size) for i in range(0, 10_000_000, chunk_size)]

    t0 = time.perf_counter()
    with Pool() as pool:
        counts = pool.map(count_primes_in_range, ranges)
    elapsed = time.perf_counter() - t0

    total = sum(counts)
    print(f"Primes below 10,000,000: {total:,}")
    print(f"Time: {elapsed:.2f}s")
```

`Pool()` without an argument creates one worker per logical CPU. All arguments and return values must be picklable — define worker functions at module level, not inside `if __name__ == "__main__"`.

## Pool.starmap and imap

`starmap` unpacks each item as positional arguments, removing the need to pack into a tuple. `imap` is the lazy version — results are yielded one by one rather than collected in full:

```python
from multiprocessing import Pool
import math

def distance(x1: float, y1: float, x2: float, y2: float) -> float:
    return math.sqrt((x2 - x1) ** 2 + (y2 - y1) ** 2)

if __name__ == "__main__":
    pairs = [
        (0.0, 0.0, 3.0, 4.0),
        (1.0, 1.0, 4.0, 5.0),
        (2.0, 3.0, 5.0, 7.0),
        (0.0, 0.0, 1.0, 1.0),
    ]

    with Pool(processes=4) as pool:
        results = pool.starmap(distance, pairs)

    for args, d in zip(pairs, results):
        print(f"({args[0]},{args[1]}) -> ({args[2]},{args[3]}): {d:.4f}")
```

For very large datasets, prefer `pool.imap(func, iterable, chunksize=500)` to avoid materializing all results at once.

## Queue for inter-process communication

`multiprocessing.Queue` is a process-safe FIFO backed by a pipe. It is the standard way to pass messages between a producer and one or more consumers:

```python
from multiprocessing import Process, Queue
import time

def producer(q: Queue, items: list[str]) -> None:
    for item in items:
        q.put(item)
        time.sleep(0.05)
    q.put(None)   # sentinel: signal consumers to stop

def consumer(q: Queue, name: str) -> None:
    while True:
        item = q.get()
        if item is None:
            q.put(None)   # pass sentinel along for other consumers
            break
        print(f"[{name}] processed: {item}")

if __name__ == "__main__":
    q: Queue = Queue()
    tasks = [f"task-{i}" for i in range(20)]

    prod = Process(target=producer, args=(q, tasks))
    cons1 = Process(target=consumer, args=(q, "worker-1"))
    cons2 = Process(target=consumer, args=(q, "worker-2"))

    prod.start()
    cons1.start()
    cons2.start()

    prod.join()
    cons1.join()
    cons2.join()
```

Queues serialize data with pickle on the send side and unpickle on the receive side. For high-throughput scenarios, keep messages small or consider `multiprocessing.Pipe` for a direct channel between two processes.

## Shared memory with Value and Array

`multiprocessing.Value` wraps a single typed value in shared memory. Multiple processes can read and write it without copying data through a queue:

```python
from multiprocessing import Process, Value, Lock
import ctypes
import time

def increment(counter: Value, lock: Lock, n: int) -> None:
    for _ in range(n):
        with lock:
            counter.value += 1

if __name__ == "__main__":
    counter = Value(ctypes.c_int, 0)
    lock = Lock()

    workers = [
        Process(target=increment, args=(counter, lock, 250_000))
        for _ in range(4)
    ]

    t0 = time.perf_counter()
    for w in workers:
        w.start()
    for w in workers:
        w.join()
    elapsed = time.perf_counter() - t0

    print(f"Final count: {counter.value:,}")   # 1,000,000
    print(f"Time: {elapsed:.2f}s")
```

`Value` accepts any `ctypes` type — `c_int`, `c_double`, `c_bool`, and so on. Always guard reads and writes with a `Lock` unless the operation is atomic at the OS level.

`Array` works the same way for fixed-length sequences:

```python
from multiprocessing import Array, Process
import ctypes

def fill_chunk(arr: Array, start: int, end: int) -> None:
    for i in range(start, end):
        arr[i] = i * i

if __name__ == "__main__":
    size = 1000
    arr = Array(ctypes.c_int, size)

    half = size // 2
    p1 = Process(target=fill_chunk, args=(arr, 0, half))
    p2 = Process(target=fill_chunk, args=(arr, half, size))

    p1.start(); p2.start()
    p1.join();  p2.join()

    print(arr[0], arr[1], arr[2])    # 0 1 4
    print(arr[999])                  # 998001
```

## Manager for complex shared state

`Value` and `Array` cover primitives. For shared dicts, lists, or custom objects, use a `Manager`. It starts a server process that owns the data and proxies access over a socket:

```python
from multiprocessing import Manager, Pool

def worker_task(args: tuple) -> None:
    results, key, value = args
    results[key] = value * value

if __name__ == "__main__":
    with Manager() as manager:
        shared_results = manager.dict()

        tasks = [(shared_results, f"x{i}", i) for i in range(10)]

        with Pool(processes=4) as pool:
            pool.map(worker_task, tasks)

        for k in sorted(shared_results):
            print(f"{k} = {shared_results[k]}")
```

Managers add network overhead compared to `Value`/`Array`. Use them only when you genuinely need a shared mutable container. For read-heavy workloads, consider writing results to a `Queue` and aggregating them in the main process instead.

## GIL bypass and performance notes

Python's Global Interpreter Lock prevents two threads from running Python bytecode at the same time. `multiprocessing` sidesteps this entirely: each child has its own interpreter with its own GIL, so all cores can run simultaneously.

On bunpy, each `Process` corresponds to a goroutine with an isolated interpreter state. Startup is fast because bunpy reuses pre-warmed interpreter memory, but there is still a cost to spawning and a cost to serializing data across the process boundary.

Rules of thumb:

- Use `multiprocessing` when the work is genuinely CPU-bound (numerical computation, compression, parsing, image manipulation) and each task takes at least a few milliseconds. The serialization overhead dominates for tasks shorter than that.
- Use `threading` or `asyncio` for I/O-bound work. Threads share memory directly and avoid the pickle round-trip.
- Use `Pool.imap` with a `chunksize` larger than 1 when mapping over thousands of small items to batch the pickle overhead.

## Run the examples

```bash
bunpy primes.py
bunpy shared_counter.py
bunpy producer_consumer.py
```

For I/O-bound concurrency, see the [threading guide](../threading) or the [asyncio patterns guide](../asyncio-patterns). For a higher-level API over `ProcessPoolExecutor`, see the [concurrent.futures guide](../concurrent-futures).

---
title: bunpy.worker
description: Goroutine-backed background workers.
---

```python
import bunpy.worker as worker
```

## worker.run(fn, *args) → Future

Run a function in a background goroutine and return a `Future`.

```python
future = worker.run(expensive_computation, data)
result = future.result()   # blocks until complete
```

## worker.runAll(*fns) → list

Run multiple functions concurrently and collect results in order.

```python
results = worker.runAll(
    lambda: compute_a(),
    lambda: compute_b(),
    lambda: compute_c(),
)
```

## Future API

| Method | Description |
|--------|-------------|
| `.result()` | Block and return the result; re-raise any exception |
| `.done()` | Return `True` if the goroutine has finished |
| `.cancel()` | Signal cancellation (cooperative) |

## Example

```python
import bunpy.worker as worker

def fetch_page(url):
    return fetch(url).text()

urls = [
    "https://example.com/page1",
    "https://example.com/page2",
    "https://example.com/page3",
]

futures = [worker.run(fetch_page, url) for url in urls]
pages = [f.result() for f in futures]
```

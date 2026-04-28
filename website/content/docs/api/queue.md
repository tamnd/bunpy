---
title: bunpy.queue
description: Concurrent task queue backed by goroutines.
---

```python
import bunpy.queue as queue
```

## Creating a queue

```python
q = queue.Queue(concurrency=4)
```

`concurrency` sets the number of goroutines processing tasks in parallel.

## Enqueuing tasks

```python
q.push(my_function, arg1, arg2)
q.push(another_function)
```

## Waiting for completion

```python
q.wait()   # blocks until all queued tasks complete
```

## Error handling

If a task raises an exception, the error is recorded and all remaining tasks
are still processed. After `wait()`, check:

```python
errors = q.errors()
if errors:
    for err in errors:
        print(err)
```

## Example

```python
import bunpy.queue as queue

results = []

def process(item):
    results.append(item * 2)

q = queue.Queue(concurrency=8)
for i in range(100):
    q.push(process, i)
q.wait()

print(sorted(results)[:5])  # [0, 2, 4, 6, 8]
```

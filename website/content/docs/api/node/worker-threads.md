---
title: bunpy.node.worker_threads
description: Node.js-compatible worker_threads module.
---

```python
from bunpy.node import worker_threads
from bunpy.node.worker_threads import Worker, MessageChannel, isMainThread, threadId
```

## isMainThread

`True` in the main script, `False` in a worker.

```python
print(worker_threads.isMainThread)   # True
```

## threadId

Integer thread identifier. `0` for the main thread.

```python
print(worker_threads.threadId)   # 0
```

## Worker

```python
w = Worker(lambda: print("hello from worker"))
w.on("exit", lambda code: print(f"worker exited {code}"))
```

### Methods

| Method | Description |
|--------|-------------|
| `on(event, handler)` | Register event handler (`"message"`, `"exit"`) |
| `postMessage(data)` | Send a message to the worker |
| `terminate()` | Signal the worker to stop |

## MessageChannel

Bi-directional channel between two goroutines:

```python
ch = worker_threads.MessageChannel()
port1 = ch.port1
port2 = ch.port2

port1.postMessage("hello")
msg = worker_threads.receiveMessageOnPort(port2)
print(msg["message"])   # "hello"
```

## receiveMessageOnPort(port)

Synchronously receive one message from a port. Returns `None` if no message
is available.

```python
msg = worker_threads.receiveMessageOnPort(port)
if msg is not None:
    print(msg["message"])
```

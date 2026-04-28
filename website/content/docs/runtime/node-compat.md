---
title: Node.js compatibility
description: bunpy.node.* - a Node.js standard library shim for Python.
weight: 4
---

bunpy 0.8+ ships `bunpy.node` - a Go-backed shim of the Node.js standard
library that lets Python code use the same APIs as Node.js scripts.

## Available modules

| Module | Node.js equivalent | Key exports |
|--------|-------------------|-------------|
| `bunpy.node.fs` | `fs` / `fs/promises` | readFile, writeFile, mkdir, readdir, stat, ... |
| `bunpy.node.path` | `path` | join, resolve, dirname, basename, extname, ... |
| `bunpy.node.os` | `os` | platform, arch, hostname, homedir, tmpdir, cpus, ... |
| `bunpy.node.http` | `http` | createServer, request, IncomingMessage, ServerResponse |
| `bunpy.node.https` | `https` | createServer, request (TLS) |
| `bunpy.node.net` | `net` | createServer, createConnection, Socket |
| `bunpy.node.tls` | `tls` | createServer, connect, TLSSocket |
| `bunpy.node.crypto` | `crypto` | randomBytes, randomUUID, createHash, createHmac |
| `bunpy.node.stream` | `stream` | Readable, Writable, PassThrough, Transform |
| `bunpy.node.zlib` | `zlib` | gzip, gunzip, deflate, inflate + Sync variants |
| `bunpy.node.worker_threads` | `worker_threads` | Worker, MessageChannel, isMainThread, threadId |

## Import pattern

```python
from bunpy.node import fs, path, crypto

# or individual imports
from bunpy.node.crypto import randomUUID, createHash
```

## Examples

### fs

```python
from bunpy.node import fs

content = fs.readFileSync("data.txt", "utf8")
fs.writeFileSync("out.txt", "hello world")

entries = fs.readdirSync(".")
```

### crypto

```python
from bunpy.node.crypto import createHash, randomUUID

digest = createHash("sha256").update("hello").digest("hex")
print(digest)  # 2cf24dba...

print(randomUUID())  # e.g. 550e8400-e29b-41d4-a716-446655440000
```

### worker_threads

```python
from bunpy.node.worker_threads import Worker, isMainThread, threadId

print(isMainThread)  # True
print(threadId)      # 0

w = Worker(lambda: print("running in worker"))
w.on("exit", lambda code: print(f"exited {code}"))
```

### zlib

```python
from bunpy.node import zlib

compressed = zlib.gzipSync(b"hello world")
original = zlib.gunzipSync(compressed)
print(original)  # b'hello world'
```

## Top-level namespace

The `bunpy.node` module exposes all sub-modules as attributes:

```python
import bunpy.node as node
node.fs.writeFileSync("x.txt", "hello")
node.path.join("/a", "b", "c")  # /a/b/c
```

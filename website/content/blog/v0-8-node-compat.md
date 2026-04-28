---
title: "v0.8: Node.js compatibility shim"
date: 2026-04-20
description: bunpy 0.8 ships bunpy.node.* — 11 Node.js standard library modules backed by Go.
---

bunpy 0.8 ships a full Node.js standard library shim: `bunpy.node.*`.

You can now write Python code using the same API shapes as Node.js:

```python
from bunpy.node.fs import readFileSync, writeFileSync
from bunpy.node.path import join, dirname
from bunpy.node.crypto import createHash, randomUUID
```

## What's included

11 modules covering the Node.js standard library:

- **`bunpy.node.fs`** — file system: readFile, writeFile, readdir, stat, mkdir, ...
- **`bunpy.node.path`** — join, resolve, dirname, basename, extname, ...
- **`bunpy.node.os`** — platform, arch, hostname, homedir, tmpdir, cpus, ...
- **`bunpy.node.http`** / **`https`** — createServer, request, IncomingMessage
- **`bunpy.node.net`** / **`tls`** — TCP sockets with TLS
- **`bunpy.node.crypto`** — randomBytes, randomUUID, createHash, createHmac
- **`bunpy.node.stream`** — Readable, Writable, PassThrough, Transform
- **`bunpy.node.zlib`** — gzip, gunzip, deflate, inflate + Sync variants
- **`bunpy.node.worker_threads`** — Worker, MessageChannel (goroutine-backed)

## Why

bunpy targets Python developers who also work with Node.js. The `bunpy.node.*`
shim lets you port Node.js scripts to bunpy line-by-line, and use your Node.js
mental model for I/O operations.

The entire shim is backed by Go stdlib — no C dependencies, no npm, no Node.js
binary on the host.

## Shipped over 10 releases

- v0.8.0: `bunpy.node.fs`
- v0.8.1: `bunpy.node.path`
- v0.8.2: `bunpy.node.os`
- v0.8.3: `bunpy.node.http` / `https`
- v0.8.4: `bunpy.node.net` / `tls`
- v0.8.5: `bunpy.node.crypto`
- v0.8.6: `bunpy.node.stream`
- v0.8.7: `bunpy.node.zlib`
- v0.8.8: `bunpy.node.worker_threads`
- v0.8.9: `bunpy.node` top-level namespace

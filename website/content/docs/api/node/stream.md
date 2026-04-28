---
title: bunpy.node.stream
description: Node.js-compatible stream module.
---

```python
from bunpy.node import stream
from bunpy.node.stream import Readable, Writable, PassThrough
```

## Readable

```python
r = stream.Readable()
r.push(b"chunk 1")
r.push(b"chunk 2")
r.push(None)   # signal EOF

data = r.read()   # returns all buffered bytes
```

## Writable

```python
w = stream.Writable()
w.write(b"hello ")
w.write(b"world")
w.end()

content = w.getContents()   # b"hello world"
```

## PassThrough

Bidirectional passthrough — data written to it can be read out:

```python
pt = stream.PassThrough()
pt.write(b"data")
pt.end()
out = pt.read()   # b"data"
```

## pipe

```python
src = stream.Readable()
src.push(b"hello")
src.push(None)

dst = stream.Writable()
src.pipe(dst)

print(dst.getContents())   # b"hello"
```

## Transform

`stream.Transform` is an alias for `PassThrough`. Override `_transform` for
custom behaviour:

```python
class UpperCase(stream.Transform):
    def _transform(self, chunk, encoding, callback):
        self.push(chunk.upper())
        callback()
```

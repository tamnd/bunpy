---
title: bunpy.serve - WebSocket
description: WebSocket upgrade from bunpy.serve, send and receive messages, broadcast, rooms, and connection lifecycle handling.
weight: 15
---

```python
from bunpy.serve import serve
```

WebSocket connections are handled inside the same `bunpy.serve` handler as HTTP requests. When the client sends an `Upgrade: websocket` header, call `req.upgrade()` to switch protocols and return the resulting `WebSocket` object from your handler.

## Upgrade from HTTP

```python
from bunpy.serve import serve

def handler(req):
    if req.headers.get("upgrade") == "websocket":
        ws = req.upgrade()
        return ws

    return "Not a WebSocket request"

serve(handler, port=3000)
```

`req.upgrade()` performs the handshake and returns a `WebSocket` instance.
The handler must return the `WebSocket` - bunpy uses the return value to track the live connection.

## Sending and receiving messages

```python
from bunpy.serve import serve

def handler(req):
    if req.headers.get("upgrade") != "websocket":
        return {"status": 400, "body": "WebSocket only"}

    ws = req.upgrade()

    # Receive loop
    for msg in ws:
        print(f"Received: {msg.text}")    # msg.text for str, msg.bytes for bytes
        ws.send(f"Echo: {msg.text}")      # send text back

    return ws
```

### WebSocket methods

| Method | Description |
|--------|-------------|
| `ws.send(data)` | Send a text or bytes message |
| `ws.sendText(text)` | Send a UTF-8 text frame explicitly |
| `ws.sendBytes(data)` | Send a binary frame explicitly |
| `ws.close(code=1000, reason="")` | Close the connection |
| `ws.ping(data=b"")` | Send a ping frame |

### WebSocket properties

| Property | Type | Description |
|----------|------|-------------|
| `ws.id` | str | Unique connection ID (UUID) |
| `ws.closed` | bool | `True` after the connection is closed |
| `ws.remoteAddress` | str | Client IP address |
| `ws.data` | Any | Arbitrary data you attach to this connection |

### Message object

Iterating over `ws` yields `Message` objects:

| Field | Type | Description |
|-------|------|-------------|
| `msg.text` | str | Message decoded as UTF-8 (text frames) |
| `msg.bytes` | bytes | Raw message bytes (binary frames) |
| `msg.is_text` | bool | `True` for text frames |
| `msg.is_binary` | bool | `True` for binary frames |

## Broadcast to all clients

Track connections yourself and broadcast to all of them:

```python
from bunpy.serve import serve
import threading

clients: set = set()
lock = threading.Lock()

def handler(req):
    if req.headers.get("upgrade") != "websocket":
        return "use ws://"

    ws = req.upgrade()
    with lock:
        clients.add(ws)

    try:
        for msg in ws:
            # Broadcast to everyone
            with lock:
                dead = set()
                for client in clients:
                    if client.closed:
                        dead.add(client)
                        continue
                    client.send(msg.text)
                clients -= dead
    finally:
        with lock:
            clients.discard(ws)

    return ws

serve(handler, port=3000)
```

## Rooms / pub-sub

```python
from bunpy.serve import serve
import json
import threading
from collections import defaultdict

rooms: dict[str, set] = defaultdict(set)
lock = threading.Lock()

def broadcast(room: str, event: dict, exclude=None):
    with lock:
        dead = set()
        for ws in rooms[room]:
            if ws.closed:
                dead.add(ws)
                continue
            if ws is exclude:
                continue
            ws.send(json.dumps(event))
        rooms[room] -= dead

def handler(req):
    if req.headers.get("upgrade") != "websocket":
        return {"status": 400, "body": "WebSocket only"}

    room = req.query.get("room", "general")
    ws = req.upgrade()
    ws.data = {"room": room, "user": req.query.get("user", "anon")}

    with lock:
        rooms[room].add(ws)

    broadcast(room, {"type": "join", "user": ws.data["user"]}, exclude=ws)
    ws.send(json.dumps({"type": "joined", "room": room}))

    try:
        for msg in ws:
            event = {"type": "message", "user": ws.data["user"], "text": msg.text}
            broadcast(room, event)
    finally:
        with lock:
            rooms[room].discard(ws)
        broadcast(room, {"type": "leave", "user": ws.data["user"]})

    return ws

serve(handler, port=3000)
print("Chat server on ws://localhost:3000?room=general&user=alice")
```

## Connection lifecycle

```python
from bunpy.serve import serve

def handler(req):
    if req.headers.get("upgrade") != "websocket":
        return "not ws"

    ws = req.upgrade()
    print(f"[{ws.id}] connected from {ws.remoteAddress}")

    try:
        for msg in ws:
            if msg.is_text:
                print(f"[{ws.id}] text: {msg.text!r}")
                ws.send("ack")
            elif msg.is_binary:
                print(f"[{ws.id}] binary: {len(msg.bytes)} bytes")
    except ConnectionError as e:
        print(f"[{ws.id}] connection error: {e}")
    finally:
        print(f"[{ws.id}] disconnected")

    return ws

serve(handler, port=3000)
```

## Ping / pong keepalive

```python
from bunpy.serve import serve
import threading
import time

def keepalive(ws, interval=30):
    def loop():
        while not ws.closed:
            try:
                ws.ping()
            except Exception:
                break
            time.sleep(interval)
    threading.Thread(target=loop, daemon=True).start()

def handler(req):
    if req.headers.get("upgrade") != "websocket":
        return "not ws"

    ws = req.upgrade()
    keepalive(ws)

    for msg in ws:
        ws.send(msg.text)

    return ws

serve(handler, port=3000)
```

## Async handler

```python
import asyncio
from bunpy.serve import serve

async def handler(req):
    if req.headers.get("upgrade") != "websocket":
        return "use ws://"

    ws = await req.upgrade_async()

    async for msg in ws:
        await ws.asend(f"echo: {msg.text}")

    return ws

serve(handler, port=3000)
```

## Live dashboard example

```python
from bunpy.serve import serve
import json
import threading
import time
import psutil

viewers: set = set()
lock = threading.Lock()

def metrics_loop():
    while True:
        data = {
            "cpu": psutil.cpu_percent(),
            "mem": psutil.virtual_memory().percent,
            "ts": time.time(),
        }
        payload = json.dumps(data)
        with lock:
            dead = {ws for ws in viewers if ws.closed}
            viewers.difference_update(dead)
            for ws in viewers:
                try:
                    ws.send(payload)
                except Exception:
                    pass
        time.sleep(1)

threading.Thread(target=metrics_loop, daemon=True).start()

def handler(req):
    if req.headers.get("upgrade") == "websocket":
        ws = req.upgrade()
        with lock:
            viewers.add(ws)
        for _ in ws:   # drain messages, ignore them
            pass
        with lock:
            viewers.discard(ws)
        return ws

    return {
        "headers": {"Content-Type": "text/html"},
        "body": "<script>const ws=new WebSocket('ws://localhost:3000');"
                "ws.onmessage=e=>console.log(JSON.parse(e.data))</script>",
    }

serve(handler, port=3000)
```

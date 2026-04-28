---
title: Server-Sent Events (SSE)
description: Stream SSE responses from bunpy.serve using async generators, with event types, retry, and client reconnect support.
weight: 16
---

```python
from bunpy.serve import serve, SSEResponse
```

Server-Sent Events (SSE) let a server push a stream of text events to a browser over a plain HTTP connection. `bunpy.serve` turns an async generator into an SSE response — the generator `yield`s events, and bunpy handles framing, flushing, and connection teardown.

## Basic SSE response

```python
from bunpy.serve import serve, SSEResponse
import asyncio

async def count_up():
    for i in range(10):
        yield {"data": str(i)}
        await asyncio.sleep(1)

def handler(req):
    return SSEResponse(count_up())

serve(handler, port=3000)
```

Open `http://localhost:3000` in a browser and watch the numbers arrive one per second.

### SSEResponse(generator, headers=None)

Wraps an async generator and streams each yielded value as an SSE event.

The generator yields dicts with optional keys:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `data` | str | — | The event payload (required) |
| `event` | str | `None` | Named event type |
| `id` | str | `None` | Event ID for client reconnect |
| `retry` | int | `None` | Client reconnect delay in milliseconds |

## Event types

Use the `event` key to dispatch named events on the client:

```python
from bunpy.serve import serve, SSEResponse
import asyncio
import json

async def live_feed():
    # Unnamed event — client listens with onmessage
    yield {"data": "connected"}

    for tick in range(100):
        await asyncio.sleep(0.5)
        if tick % 10 == 0:
            # Named event — client listens with addEventListener("heartbeat", ...)
            yield {"event": "heartbeat", "data": str(tick)}
        else:
            yield {
                "event": "update",
                "data": json.dumps({"tick": tick, "value": tick * 3.14}),
            }

def handler(req):
    return SSEResponse(live_feed())

serve(handler, port=3000)
```

Client-side JavaScript:

```javascript
const es = new EventSource("http://localhost:3000");
es.onmessage = e => console.log("message:", e.data);
es.addEventListener("heartbeat", e => console.log("heartbeat:", e.data));
es.addEventListener("update", e => console.log("update:", JSON.parse(e.data)));
```

## Retry and reconnect

Tell the browser how long to wait before reconnecting after a dropped connection:

```python
async def resilient_stream():
    # First event sets the retry delay to 3 seconds
    yield {"data": "start", "retry": 3000}

    event_id = 0
    while True:
        event_id += 1
        yield {"id": str(event_id), "data": f"event-{event_id}"}
        await asyncio.sleep(1)
```

When the browser reconnects after a drop it sends `Last-Event-ID` in the request headers. Resume from that point:

```python
from bunpy.serve import serve, SSEResponse

async def resumable(last_id: int):
    i = last_id + 1
    while True:
        yield {"id": str(i), "data": f"event-{i}"}
        i += 1
        await asyncio.sleep(0.5)

def handler(req):
    last_id_header = req.headers.get("last-event-id", "0")
    last_id = int(last_id_header)
    return SSEResponse(resumable(last_id))

serve(handler, port=3000)
```

## Live log streamer

```python
from bunpy.serve import serve, SSEResponse
import asyncio
import bunpy.file.async_ as afile

async def tail_log(path: str, last_id: int = 0):
    # Read existing lines up to `last_id`, then watch for new ones
    lines = (await afile.read(path)).splitlines()
    for i, line in enumerate(lines[last_id:], start=last_id):
        yield {"id": str(i), "event": "log", "data": line}

    line_count = len(lines)
    # Poll for new lines
    while True:
        await asyncio.sleep(0.25)
        current = (await afile.read(path)).splitlines()
        for i, line in enumerate(current[line_count:], start=line_count):
            yield {"id": str(i), "event": "log", "data": line}
        line_count = len(current)

def handler(req):
    log_path = req.query.get("file", "app.log")
    last_id = int(req.headers.get("last-event-id", "0"))

    if not log_path.endswith(".log"):
        return {"status": 400, "body": "bad file"}

    return SSEResponse(tail_log(log_path, last_id))

serve(handler, port=3000)
```

## Token streaming for LLM output

Stream LLM tokens as they arrive — the pattern used by ChatGPT, Claude, and every modern AI chat UI:

```python
from bunpy.serve import serve, SSEResponse
import asyncio
import json

async def stream_tokens(prompt: str):
    # Stand-in for an actual LLM client
    import anthropic
    client = anthropic.Anthropic()

    yield {"event": "start", "data": json.dumps({"prompt": prompt})}

    full_text = ""
    with client.messages.stream(
        model="claude-sonnet-4-6",
        max_tokens=1024,
        messages=[{"role": "user", "content": prompt}],
    ) as stream:
        for token in stream.text_stream:
            full_text += token
            yield {"event": "token", "data": json.dumps({"token": token})}

    yield {"event": "done", "data": json.dumps({"full_text": full_text})}

def handler(req):
    if req.method == "POST":
        body = req.json()
        prompt = body.get("prompt", "")
        return SSEResponse(stream_tokens(prompt))

    return {
        "headers": {"Content-Type": "text/html"},
        "body": open("chat.html").read(),
    }

serve(handler, port=3000)
```

## Stock ticker

```python
from bunpy.serve import serve, SSEResponse
import asyncio
import json
import random

SYMBOLS = ["AAPL", "GOOG", "MSFT", "TSLA", "AMZN"]
prices = {s: round(100 + random.random() * 400, 2) for s in SYMBOLS}

async def ticker(symbols: list[str]):
    yield {"retry": 2000, "data": json.dumps({s: prices[s] for s in symbols})}

    while True:
        await asyncio.sleep(0.5)
        symbol = random.choice(symbols)
        delta = round(random.uniform(-2, 2), 2)
        prices[symbol] = round(max(1, prices[symbol] + delta), 2)
        yield {
            "event": "tick",
            "data": json.dumps({"symbol": symbol, "price": prices[symbol], "delta": delta}),
        }

def handler(req):
    requested = req.query.get("symbols", "AAPL,MSFT").split(",")
    valid = [s for s in requested if s in SYMBOLS]
    return SSEResponse(ticker(valid or SYMBOLS))

serve(handler, port=3000)
print("Stock ticker on http://localhost:3000?symbols=AAPL,TSLA")
```

## Headers and CORS

Pass additional headers to `SSEResponse` — useful for CORS when the client is on a different origin:

```python
from bunpy.serve import serve, SSEResponse

CORS = {
    "Access-Control-Allow-Origin": "*",
    "Cache-Control": "no-cache",
}

async def gen():
    yield {"data": "hello"}

def handler(req):
    if req.method == "OPTIONS":
        return {"status": 204, "headers": CORS}
    return SSEResponse(gen(), headers=CORS)

serve(handler, port=3000)
```

## Connection cleanup

The generator is automatically cancelled when the client disconnects.
Use `try/finally` to release resources:

```python
from bunpy.serve import serve, SSEResponse
import asyncio

async def stream_with_cleanup():
    resource = acquire_resource()
    try:
        while True:
            data = resource.read_next()
            yield {"data": data}
            await asyncio.sleep(0.1)
    finally:
        resource.release()
        print("client disconnected — resource released")

def handler(req):
    return SSEResponse(stream_with_cleanup())

serve(handler, port=3000)
```

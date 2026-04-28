---
title: Stream data with Server-Sent Events
description: Push live updates from server to browser using SSE — log streaming, LLM token output, and EventSource reconnect handling.
---

## SSE vs WebSocket

Server-Sent Events (SSE) are a simple, HTTP-native way for the server to push a stream of text events to the browser. Unlike WebSockets, SSE is unidirectional — the server writes, the client reads. That simplicity makes it the right choice for:

- Live log tails
- Progress updates for long-running jobs
- LLM token streaming
- Real-time dashboards with read-only data

Use WebSockets when you need the client to send data back over the same channel.

## How SSE works

The server responds with `Content-Type: text/event-stream` and keeps the connection open, writing lines in this format:

```
data: your payload here\n\n
```

Optional fields:

```
id: 42\n
event: custom-event-name\n
data: {"key": "value"}\n
retry: 3000\n
\n
```

A blank line (`\n\n`) ends each event. The browser's built-in `EventSource` API handles reconnection automatically.

## Basic SSE endpoint with bunpy.serve

```python
import time
from bunpy.serve import serve


def handler(req):
    if req.path == "/events":
        return sse_handler(req)
    return {"status": 404, "body": "Not found"}


def sse_handler(req):
    def stream():
        for i in range(10):
            yield f"data: tick {i}\n\n"
            time.sleep(1)

    return {
        "status": 200,
        "headers": {
            "Content-Type": "text/event-stream",
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",   # disable nginx buffering if behind a proxy
        },
        "body": stream(),
    }


serve(handler, port=3000)
```

Run it and curl the stream:

```bash
bunpy server.py &

curl -N http://localhost:3000/events
# data: tick 0
# data: tick 1
# ...
```

## Live log streamer

This example tails a log file and pushes each new line to every connected browser. It is the server-side equivalent of `tail -f`.

```python
from __future__ import annotations

import json
import os
import time
from pathlib import Path
from bunpy.serve import serve

LOG_FILE = Path(os.environ.get("LOG_FILE", "/tmp/app.log"))


def tail_file(path: Path, poll_interval: float = 0.25):
    """Generator that yields new lines appended to a file."""
    with path.open("r") as f:
        f.seek(0, 2)          # seek to end
        while True:
            line = f.readline()
            if line:
                yield line.rstrip("\n")
            else:
                time.sleep(poll_interval)


def log_stream(req):
    def events():
        try:
            for line in tail_file(LOG_FILE):
                payload = json.dumps({"line": line, "ts": time.time()})
                yield f"data: {payload}\n\n"
        except GeneratorExit:
            pass   # client disconnected

    return {
        "status": 200,
        "headers": {
            "Content-Type": "text/event-stream",
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
        "body": events(),
    }


def handler(req):
    if req.path == "/logs":
        if not LOG_FILE.exists():
            return {"status": 404, "body": f"{LOG_FILE} not found"}
        return log_stream(req)

    if req.path == "/":
        return {
            "status": 200,
            "headers": {"Content-Type": "text/html"},
            "body": CLIENT_HTML,
        }

    return {"status": 404, "body": "Not found"}


serve(handler, port=3000)
```

Write test lines to the log from another terminal:

```bash
while true; do echo "$(date) - request processed" >> /tmp/app.log; sleep 1; done
```

## LLM token streaming

Streaming tokens from an LLM API as they arrive makes the interface feel responsive. The pattern is identical to the log streamer — yield each chunk as a separate SSE event.

```python
import json
import anthropic
from bunpy.serve import serve

client = anthropic.Anthropic()


def llm_stream(req):
    prompt = req.query.get("prompt", "Tell me a short story.")

    def events():
        with client.messages.stream(
            model="claude-opus-4-5",
            max_tokens=512,
            messages=[{"role": "user", "content": prompt}],
        ) as stream:
            for text in stream.text_stream:
                payload = json.dumps({"token": text})
                yield f"data: {payload}\n\n"

        # Signal completion so the client knows to stop the spinner
        yield "event: done\ndata: {}\n\n"

    return {
        "status": 200,
        "headers": {
            "Content-Type": "text/event-stream",
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
        "body": events(),
    }


def handler(req):
    if req.path == "/generate":
        return llm_stream(req)
    return {"status": 404, "body": "Not found"}


serve(handler, port=3000)
```

```bash
bunpy add anthropic
ANTHROPIC_API_KEY=sk-... bunpy server.py
curl -N "http://localhost:3000/generate?prompt=Hello"
# data: {"token": "Hello"}
# data: {"token": "!"}
# ...
# event: done
# data: {}
```

## Client JavaScript (EventSource)

Paste this into any HTML page to consume the log stream. The same pattern works for the LLM streamer — just swap the URL and the `onmessage` handler.

```html
CLIENT_HTML = """
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Live Logs</title>
  <style>
    body { font-family: monospace; background: #0f172a; color: #e2e8f0; padding: 1rem; }
    #log { white-space: pre-wrap; font-size: 0.85rem; line-height: 1.6; }
    .ts  { color: #64748b; }
  </style>
</head>
<body>
  <h2 style="font-family:system-ui;color:#94a3b8">Live log stream</h2>
  <div id="log"></div>
  <script>
    const log = document.getElementById("log");

    function append(text, cls) {
      const span = document.createElement("span");
      if (cls) span.className = cls;
      span.textContent = text + "\\n";
      log.appendChild(span);
      window.scrollTo(0, document.body.scrollHeight);
    }

    const es = new EventSource("/logs");

    es.onmessage = (e) => {
      const { line, ts } = JSON.parse(e.data);
      const date = new Date(ts * 1000).toISOString().slice(11, 19);
      append(`[${date}] ${line}`);
    };

    es.onerror = () => {
      append("Connection lost — retrying…", "ts");
    };

    es.addEventListener("done", () => {
      append("Stream ended.", "ts");
      es.close();
    });
  </script>
</body>
</html>
"""
```

## Reconnect and retry behavior

`EventSource` reconnects automatically after a network drop. The server can control the reconnect delay by sending a `retry:` field (milliseconds):

```python
def events():
    # Tell the client to wait 5 s before reconnecting if the connection drops
    yield "retry: 5000\n\n"

    event_id = 0
    for line in tail_file(LOG_FILE):
        event_id += 1
        payload = json.dumps({"line": line})
        yield f"id: {event_id}\ndata: {payload}\n\n"
```

On reconnect, the browser sends a `Last-Event-ID` header. Read it on the server to resume from where the client left off:

```python
def log_stream(req):
    last_id = int(req.headers.get("Last-Event-ID", "0"))
    # skip or replay events with id <= last_id
    ...
```

## CORS for cross-origin clients

If your frontend is on a different origin, add the appropriate header:

```python
return {
    "status": 200,
    "headers": {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        "Access-Control-Allow-Origin": "*",
    },
    "body": events(),
}
```

## Bundle and deploy

```bash
bunpy build server.py -o logstream.pyz
./logstream.pyz
```

Behind nginx, add these directives to avoid response buffering:

```nginx
location /logs {
    proxy_pass         http://127.0.0.1:3000;
    proxy_buffering    off;
    proxy_cache        off;
    proxy_set_header   X-Accel-Buffering no;
    proxy_read_timeout 3600s;
}
```

## What to add next

- **Authentication**: check a bearer token or session cookie before opening the stream.
- **Named events**: send `event: metric\n` so clients can subscribe with `es.addEventListener("metric", handler)`.
- **Fan-out**: use a `threading.Event` or `asyncio.Queue` to push the same event to multiple simultaneous SSE clients without running one generator per client.

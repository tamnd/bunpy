---
title: Build a WebSocket chat server
description: Room-based WebSocket chat with bunpy.serve, connection tracking, broadcast, token auth, and a browser client.
---

## Overview

WebSockets let the server push data to connected clients without polling. This guide builds a room-based chat server where clients join named rooms, broadcast messages to everyone in the same room, and get notified when others leave. Authentication happens via a query-parameter token checked on the initial handshake.

## Create the project

```bash
bunpy create --template minimal chat-server
cd chat-server
```

No extra packages required — `bunpy.serve` handles the WebSocket upgrade natively.

## Server: connection state

Create `server.py`. Start with the data structures that track open connections.

```python
from __future__ import annotations

import json
import logging
from collections import defaultdict
from typing import Any

from bunpy.serve import serve, WebSocket

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger(__name__)

# room_name -> set of connected WebSocket objects
rooms: dict[str, set[WebSocket]] = defaultdict(set)

# WebSocket -> {"room": str, "username": str}
connections: dict[WebSocket, dict[str, str]] = {}

# Simple token allow-list. In production, verify a JWT or session cookie.
VALID_TOKENS: set[str] = {"token-alice", "token-bob", "token-carol", "dev-token"}
```

## Server: helpers

```python
def send(ws: WebSocket, event: str, data: Any) -> None:
    """Send a JSON-encoded event to a single connection."""
    try:
        ws.send(json.dumps({"event": event, "data": data}))
    except Exception:
        pass  # connection already closed


def broadcast(room: str, event: str, data: Any, exclude: WebSocket | None = None) -> None:
    """Send an event to every connection in a room, optionally skipping one."""
    dead: list[WebSocket] = []
    for ws in list(rooms[room]):
        if ws is exclude:
            continue
        try:
            ws.send(json.dumps({"event": event, "data": data}))
        except Exception:
            dead.append(ws)
    for ws in dead:
        _remove(ws)


def _remove(ws: WebSocket) -> None:
    """Clean up a disconnected socket from all state tables."""
    meta = connections.pop(ws, None)
    if meta:
        rooms[meta["room"]].discard(ws)
        if not rooms[meta["room"]]:
            del rooms[meta["room"]]
    try:
        ws.close()
    except Exception:
        pass
```

## Server: WebSocket handler

```python
def handler(req):
    # -----------------------------------------------------------------------
    # Authenticate on the handshake
    # -----------------------------------------------------------------------
    token = req.query.get("token", "")
    if token not in VALID_TOKENS:
        return {"status": 401, "body": "Unauthorized"}

    username = req.query.get("username", "anonymous")
    room = req.query.get("room", "general")

    # -----------------------------------------------------------------------
    # Upgrade to WebSocket
    # -----------------------------------------------------------------------
    ws: WebSocket = req.upgrade()

    # Register connection
    connections[ws] = {"room": room, "username": username}
    rooms[room].add(ws)

    log.info("JOIN  room=%s user=%s", room, username)

    # Greet the joiner
    send(ws, "welcome", {
        "message": f"Welcome to #{room}, {username}!",
        "members": [connections[c]["username"] for c in rooms[room] if c is not ws],
    })

    # Announce to everyone else in the room
    broadcast(room, "joined", {"username": username}, exclude=ws)

    # -----------------------------------------------------------------------
    # Message loop
    # -----------------------------------------------------------------------
    try:
        for raw in ws:
            try:
                msg = json.loads(raw)
            except json.JSONDecodeError:
                send(ws, "error", {"message": "Invalid JSON"})
                continue

            event = msg.get("event", "")

            if event == "message":
                text = str(msg.get("data", {}).get("text", "")).strip()
                if not text:
                    continue
                log.info("MSG   room=%s user=%s text=%r", room, username, text[:80])
                broadcast(room, "message", {
                    "username": username,
                    "text": text,
                })

            elif event == "switch_room":
                new_room = str(msg.get("data", {}).get("room", "general")).strip()
                if new_room == room:
                    continue

                # Leave old room
                broadcast(room, "left", {"username": username}, exclude=ws)
                rooms[room].discard(ws)
                if not rooms[room]:
                    del rooms[room]

                # Join new room
                room = new_room
                connections[ws]["room"] = room
                rooms[room].add(ws)

                send(ws, "switched", {"room": room})
                broadcast(room, "joined", {"username": username}, exclude=ws)
                log.info("SWITCH user=%s -> room=%s", username, room)

            elif event == "ping":
                send(ws, "pong", {})

            else:
                send(ws, "error", {"message": f"Unknown event: {event}"})

    except Exception as exc:
        log.warning("ERROR user=%s: %s", username, exc)

    finally:
        # -----------------------------------------------------------------------
        # Teardown
        # -----------------------------------------------------------------------
        log.info("LEAVE room=%s user=%s", room, username)
        broadcast(room, "left", {"username": username})
        _remove(ws)


serve(handler, port=3000)
```

Run the server:

```bash
bunpy server.py
# Listening on http://localhost:3000
```

## Client: browser chat page

Save this as `client.html` and open it directly in a browser (`file://` works fine for local testing).

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>bunpy chat</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: system-ui, sans-serif; display: flex; flex-direction: column; height: 100vh; }
    #toolbar { display: flex; gap: 0.5rem; padding: 0.75rem; background: #f1f5f9; border-bottom: 1px solid #e2e8f0; }
    #toolbar input { flex: 1; padding: 0.4rem 0.6rem; border: 1px solid #cbd5e1; border-radius: 4px; }
    #toolbar button { padding: 0.4rem 1rem; background: #3b82f6; color: #fff; border: none; border-radius: 4px; cursor: pointer; }
    #log  { flex: 1; overflow-y: auto; padding: 1rem; display: flex; flex-direction: column; gap: 0.25rem; }
    .msg  { padding: 0.25rem 0.5rem; border-radius: 4px; background: #f8fafc; }
    .msg .user { font-weight: 600; color: #3b82f6; }
    .sys  { color: #64748b; font-style: italic; }
    #form { display: flex; gap: 0.5rem; padding: 0.75rem; border-top: 1px solid #e2e8f0; }
    #form input  { flex: 1; padding: 0.5rem; border: 1px solid #cbd5e1; border-radius: 4px; }
    #form button { padding: 0.5rem 1.25rem; background: #3b82f6; color: #fff; border: none; border-radius: 4px; cursor: pointer; }
  </style>
</head>
<body>
  <div id="toolbar">
    <input id="usernameInput" placeholder="Username" value="alice">
    <input id="tokenInput"    placeholder="Token"    value="token-alice">
    <input id="roomInput"     placeholder="Room"     value="general">
    <button onclick="connect()">Connect</button>
    <button onclick="switchRoom()">Switch room</button>
  </div>
  <div id="log"></div>
  <form id="form" onsubmit="sendMessage(event)">
    <input id="msgInput" placeholder="Type a message…" autocomplete="off" disabled>
    <button type="submit" disabled id="sendBtn">Send</button>
  </form>

  <script>
    let ws = null;

    function log(html, cls = "msg") {
      const div = document.createElement("div");
      div.className = cls;
      div.innerHTML = html;
      document.getElementById("log").appendChild(div);
      div.scrollIntoView();
    }

    function connect() {
      if (ws) { ws.close(); }
      const username = document.getElementById("usernameInput").value;
      const token    = document.getElementById("tokenInput").value;
      const room     = document.getElementById("roomInput").value;
      const url = `ws://localhost:3000/?token=${encodeURIComponent(token)}&username=${encodeURIComponent(username)}&room=${encodeURIComponent(room)}`;
      ws = new WebSocket(url);

      ws.onopen = () => {
        document.getElementById("msgInput").disabled = false;
        document.getElementById("sendBtn").disabled  = false;
        log(`<span class="sys">Connected as <strong>${username}</strong></span>`, "sys");
      };

      ws.onmessage = (e) => {
        const { event, data } = JSON.parse(e.data);
        if (event === "welcome") {
          log(`<span class="sys">${data.message} Members online: ${data.members.join(", ") || "none"}</span>`, "sys");
        } else if (event === "message") {
          log(`<span class="user">${data.username}</span>: ${data.text}`);
        } else if (event === "joined") {
          log(`<span class="sys">${data.username} joined the room.</span>`, "sys");
        } else if (event === "left") {
          log(`<span class="sys">${data.username} left the room.</span>`, "sys");
        } else if (event === "switched") {
          log(`<span class="sys">Switched to #${data.room}</span>`, "sys");
          document.getElementById("roomInput").value = data.room;
        } else if (event === "error") {
          log(`<span class="sys" style="color:red">Error: ${data.message}</span>`, "sys");
        }
      };

      ws.onclose = () => {
        document.getElementById("msgInput").disabled = true;
        document.getElementById("sendBtn").disabled  = true;
        log(`<span class="sys">Disconnected.</span>`, "sys");
        ws = null;
      };
    }

    function sendMessage(e) {
      e.preventDefault();
      const input = document.getElementById("msgInput");
      const text = input.value.trim();
      if (!text || !ws) return;
      ws.send(JSON.stringify({ event: "message", data: { text } }));
      input.value = "";
    }

    function switchRoom() {
      const room = document.getElementById("roomInput").value.trim();
      if (!ws || !room) return;
      ws.send(JSON.stringify({ event: "switch_room", data: { room } }));
    }
  </script>
</body>
</html>
```

Open `client.html` in two browser tabs with different usernames and tokens. Messages typed in one tab appear instantly in the other.

## Test with wscat

```bash
# Install wscat globally once
npm install -g wscat

wscat -c "ws://localhost:3000/?token=dev-token&username=bot&room=general"
# Connected (press CTRL+C to quit)
> {"event":"message","data":{"text":"hello from wscat"}}
< {"event":"message","data":{"username":"bot","text":"hello from wscat"}}
```

## Bundle the server

```bash
bunpy build server.py -o chat.pyz
./chat.pyz
```

## What to add next

- **Persistent history**: store messages in SQLite and replay the last 50 on `welcome`.
- **Typing indicators**: broadcast a `typing` event with a debounce timer on the client.
- **JWT auth**: replace the token allow-list with `PyJWT` verification on the handshake.
- **Horizontal scaling**: replace the in-process `rooms` dict with a Redis pub/sub channel so multiple server instances share state.

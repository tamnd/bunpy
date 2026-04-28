---
title: Background tasks and job queues
description: Run work outside the request cycle with bunpy.queue, Celery + Redis, scheduled cron jobs, and graceful shutdown.
---

## Why background tasks

Some operations take too long to run inside an HTTP request — sending email, resizing images, calling third-party APIs, generating reports. Move them out of the request cycle so the server responds immediately and the work happens in parallel.

bunpy gives you three tools for this:

- `bunpy.queue` — lightweight in-process task queue backed by threads, zero dependencies
- Celery + Redis — production-grade distributed task queue with retries, results, and monitoring
- `bunpy.cron` — cron-style scheduler for recurring jobs

## In-process queue with bunpy.queue

`bunpy.queue` is the right choice for a single-process server with moderate task volume. Tasks share memory with the main process, so you can access globals and open database connections directly.

```python
from bunpy.queue import Queue
from bunpy.serve import serve
import json
import time

# Create a queue with 4 worker threads
q = Queue(workers=4)


# ---------------------------------------------------------------------------
# Define tasks
# ---------------------------------------------------------------------------

@q.task
def send_welcome_email(user_id: int, email: str) -> None:
    """Simulate sending an email. Replace with smtplib or an API call."""
    time.sleep(0.5)   # network I/O
    print(f"[email] sent welcome to {email} (user {user_id})")


@q.task
def resize_image(path: str, width: int, height: int) -> None:
    """Simulate image processing."""
    time.sleep(1.0)
    print(f"[image] resized {path} to {width}x{height}")


# ---------------------------------------------------------------------------
# HTTP server that enqueues tasks
# ---------------------------------------------------------------------------

def handler(req):
    if req.path == "/register" and req.method == "POST":
        body = req.json()
        user_id = body.get("user_id")
        email   = body.get("email")

        # Enqueue — returns immediately
        send_welcome_email.delay(user_id, email)

        return {
            "status": 202,
            "headers": {"Content-Type": "application/json"},
            "body": json.dumps({"status": "queued"}),
        }

    return {"status": 404, "body": "Not found"}


serve(handler, port=3000)
```

```bash
bunpy server.py &

curl -X POST http://localhost:3000/register \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "email": "alice@example.com"}'
# {"status": "queued"}
# [email] sent welcome to alice@example.com (user 1)   <- printed ~0.5 s later
```

The queue drains in the background. Call `q.join()` before process exit to wait for all pending tasks to finish.

## Celery with a Redis broker

Use Celery when you need retries, persistent task history, multiple worker processes or machines, and a dashboard (Flower).

### Install dependencies

```bash
bunpy add celery redis
```

Start Redis locally (Docker is the quickest path):

```bash
docker run -d -p 6379:6379 redis:7-alpine
```

### Define tasks

Create `tasks.py`:

```python
from celery import Celery
import time

app = Celery(
    "tasks",
    broker="redis://localhost:6379/0",
    backend="redis://localhost:6379/1",
)

app.conf.update(
    task_serializer="json",
    result_serializer="json",
    accept_content=["json"],
    timezone="UTC",
    enable_utc=True,
    task_track_started=True,
    # Retry up to 3 times, waiting 60 s between attempts
    task_acks_late=True,
    task_reject_on_worker_lost=True,
)


@app.task(bind=True, max_retries=3, default_retry_delay=60)
def send_welcome_email(self, user_id: int, email: str) -> dict:
    try:
        time.sleep(0.5)   # replace with real SMTP / SendGrid call
        print(f"[email] sent to {email}")
        return {"status": "sent", "email": email}
    except Exception as exc:
        raise self.retry(exc=exc)


@app.task(bind=True, max_retries=2)
def generate_report(self, report_id: int) -> dict:
    try:
        time.sleep(2.0)   # replace with real report generation
        path = f"/tmp/report_{report_id}.pdf"
        print(f"[report] generated {path}")
        return {"status": "done", "path": path}
    except Exception as exc:
        raise self.retry(exc=exc)
```

### Start a worker

```bash
bunpy run -m celery -A tasks worker --loglevel=info --concurrency=4
```

### Enqueue tasks from a server

Create `server.py`:

```python
from bunpy.serve import serve
from tasks import send_welcome_email, generate_report
import json


def handler(req):
    if req.path == "/register" and req.method == "POST":
        body = req.json()
        result = send_welcome_email.delay(body["user_id"], body["email"])
        return {
            "status": 202,
            "headers": {"Content-Type": "application/json"},
            "body": json.dumps({"task_id": result.id}),
        }

    if req.path.startswith("/tasks/") and req.method == "GET":
        task_id = req.path.removeprefix("/tasks/")
        from celery.result import AsyncResult
        res = AsyncResult(task_id)
        return {
            "status": 200,
            "headers": {"Content-Type": "application/json"},
            "body": json.dumps({
                "task_id": task_id,
                "status": res.status,
                "result": res.result if res.ready() else None,
            }),
        }

    return {"status": 404, "body": "Not found"}


serve(handler, port=3000)
```

```bash
# Terminal 1 — worker
bunpy run -m celery -A tasks worker --loglevel=info

# Terminal 2 — server
bunpy server.py

# Terminal 3 — test
curl -X POST http://localhost:3000/register \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "email": "alice@example.com"}'
# {"task_id": "a1b2c3d4-..."}

curl http://localhost:3000/tasks/a1b2c3d4-...
# {"status": "SUCCESS", "result": {"status": "sent", "email": "alice@example.com"}}
```

## Scheduled tasks with bunpy.cron

`bunpy.cron` lets you declare recurring jobs alongside your application code, without a separate process like cron or APScheduler.

```python
from bunpy.cron import cron
from bunpy.serve import serve
import time


@cron("*/5 * * * *")   # every 5 minutes
def cleanup_old_sessions() -> None:
    print(f"[cron] cleaning up expired sessions at {time.time():.0f}")
    # delete from sessions where expires_at < now()


@cron("0 8 * * 1-5")   # 08:00 weekdays
def send_daily_digest() -> None:
    print("[cron] sending daily digest emails")


@cron("0 0 * * 0")     # midnight Sunday
def weekly_report() -> None:
    print("[cron] generating weekly report")


def handler(req):
    return {"status": 200, "body": "ok"}


serve(handler, port=3000)
```

Cron jobs run in background threads managed by bunpy. They survive request traffic and log errors without crashing the server.

## Long-running background thread

Sometimes you want a permanent background loop — polling a queue, watching a directory, or keeping a WebSocket connection to an upstream service alive.

```python
import threading
import time
import queue
import signal
from bunpy.serve import serve

_stop = threading.Event()
_work_queue: queue.Queue = queue.Queue()


def worker_loop() -> None:
    """Pull items off the queue and process them until told to stop."""
    while not _stop.is_set():
        try:
            item = _work_queue.get(timeout=1.0)
            process(item)
            _work_queue.task_done()
        except queue.Empty:
            continue
        except Exception as exc:
            print(f"[worker] error: {exc}")


def process(item: dict) -> None:
    time.sleep(0.1)   # simulate work
    print(f"[worker] processed {item}")


# Start the thread before accepting requests
worker_thread = threading.Thread(target=worker_loop, daemon=True, name="background-worker")
worker_thread.start()


def handler(req):
    if req.path == "/jobs" and req.method == "POST":
        item = req.json()
        _work_queue.put(item)
        return {"status": 202, "body": "queued"}
    return {"status": 404, "body": "not found"}


serve(handler, port=3000)
```

## Graceful shutdown with signal handling

When your process receives `SIGTERM` (Kubernetes pod eviction, `docker stop`, Ctrl+C), you need to finish in-flight tasks before exiting.

```python
import signal
import sys
import threading
import time
from bunpy.queue import Queue
from bunpy.serve import serve

q = Queue(workers=4)
_shutdown = threading.Event()


@q.task
def slow_task(n: int) -> None:
    time.sleep(2)
    print(f"[task] done {n}")


def _handle_shutdown(signum, frame):
    print(f"\n[server] received signal {signum}, draining queue…")
    _shutdown.set()
    q.join()           # wait for all running tasks to finish
    print("[server] queue drained, exiting")
    sys.exit(0)


signal.signal(signal.SIGTERM, _handle_shutdown)
signal.signal(signal.SIGINT,  _handle_shutdown)


def handler(req):
    if req.path == "/work" and req.method == "POST":
        body = req.json()
        slow_task.delay(body.get("n", 0))
        return {"status": 202, "body": "queued"}
    return {"status": 404, "body": "not found"}


serve(handler, port=3000)
```

Test graceful shutdown:

```bash
bunpy server.py &
SERVER_PID=$!

# Enqueue some tasks
for i in 1 2 3 4 5; do
  curl -s -X POST http://localhost:3000/work \
    -H "Content-Type: application/json" \
    -d "{\"n\": $i}" &
done

# Shut down — tasks in flight complete before exit
kill -TERM $SERVER_PID
# [server] received signal 15, draining queue…
# [task] done 1
# [task] done 2
# ...
# [server] queue drained, exiting
```

## Flower: Celery monitoring dashboard

```bash
bunpy add flower
bunpy run -m celery -A tasks flower --port=5555
```

Open `http://localhost:5555` to see active workers, task history, retry rates, and queue depths.

## What to add next

- **Priorities**: Celery supports named queues (`@app.task(queue="high")`) and dedicated workers per queue.
- **Rate limiting**: `@app.task(rate_limit="10/m")` prevents a task from running more than 10 times per minute per worker.
- **Result expiry**: `app.conf.result_expires = 3600` clears finished results from Redis after one hour.
- **Dead-letter queue**: route tasks that exhaust retries to a `failed` queue for manual inspection.

---
title: Redis with bunpy
description: Connect to Redis, cache API responses, implement rate limiting, manage sessions, and use pub/sub with redis-py.
---

## Install

```bash
bunpy add redis
```

Make sure Redis is running locally:

```bash
# macOS
brew install redis && brew services start redis

# Docker
docker run -d -p 6379:6379 redis:7-alpine
```

## Connect

```python
import redis

r = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)
print(r.ping())  # True
```

`decode_responses=True` tells redis-py to return Python strings instead of bytes. For binary values (images, pickled objects) leave it off.

Use a connection URL when deploying to Redis Cloud or Heroku:

```python
import redis
import os

r = redis.from_url(os.environ.get("REDIS_URL", "redis://localhost:6379/0"), decode_responses=True)
```

## String set/get with TTL

```python
import redis

r = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)

# Set a key that expires after 60 seconds
r.set("greeting", "hello", ex=60)

value = r.get("greeting")
print(value)          # hello
print(r.ttl("greeting"))  # ~60

# setex is equivalent
r.setex("token:abc123", 3600, "user:42")

# check existence
if r.exists("token:abc123"):
    print("token is valid")

# atomic increment
r.set("page_views", 0)
r.incr("page_views")
r.incr("page_views")
print(r.get("page_views"))  # 2
```

## Hash operations

Hashes map string fields to string values under one key - ideal for storing records:

```python
import redis

r = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)

# store a user record
r.hset("user:1", mapping={
    "username": "alice",
    "email": "alice@example.com",
    "role": "admin",
})

# read individual field
print(r.hget("user:1", "username"))   # alice

# read all fields
user = r.hgetall("user:1")
print(user)   # {'username': 'alice', 'email': '...', 'role': 'admin'}

# update one field
r.hset("user:1", "role", "editor")

# delete a field
r.hdel("user:1", "role")

# check field existence
print(r.hexists("user:1", "email"))   # True
```

## List as a queue

Redis lists support O(1) push and pop on both ends - a natural fit for task queues:

```python
import redis
import json

r = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)

QUEUE = "jobs:email"

def enqueue(job: dict) -> None:
    r.rpush(QUEUE, json.dumps(job))

def dequeue(timeout: int = 5) -> dict | None:
    result = r.blpop(QUEUE, timeout=timeout)
    if result:
        _, raw = result
        return json.loads(raw)
    return None

# producer
enqueue({"to": "alice@example.com", "subject": "Welcome"})
enqueue({"to": "bob@example.com", "subject": "Reset password"})

# consumer
while True:
    job = dequeue(timeout=2)
    if job is None:
        print("Queue empty.")
        break
    print(f"Sending email to {job['to']}: {job['subject']}")
```

`blpop` blocks until an item arrives or the timeout expires - more efficient than polling.

## Pub/Sub pattern

Use pub/sub for broadcasting events to multiple subscribers:

```python
# publisher.py
import redis
import json
import time

r = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)

events = [
    {"type": "order.created", "order_id": 101},
    {"type": "order.shipped", "order_id": 101},
    {"type": "order.delivered", "order_id": 101},
]

for event in events:
    r.publish("orders", json.dumps(event))
    print(f"Published: {event['type']}")
    time.sleep(1)
```

```python
# subscriber.py
import redis
import json

r = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)

pubsub = r.pubsub()
pubsub.subscribe("orders")

print("Listening for order events...")
for message in pubsub.listen():
    if message["type"] == "message":
        event = json.loads(message["data"])
        print(f"Received: {event['type']} for order {event['order_id']}")
```

Run subscriber in one terminal and publisher in another:

```bash
bunpy subscriber.py &
bunpy publisher.py
```

## Rate limiting

A sliding-window rate limiter using a sorted set. Each request logs a timestamp; expired entries are pruned on every check:

```python
import redis
import time

r = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)

def is_allowed(user_id: str, limit: int = 10, window_seconds: int = 60) -> bool:
    key = f"ratelimit:{user_id}"
    now = time.time()
    window_start = now - window_seconds

    pipe = r.pipeline()
    pipe.zremrangebyscore(key, "-inf", window_start)   # remove old timestamps
    pipe.zadd(key, {str(now): now})                    # add current request
    pipe.zcard(key)                                    # count requests in window
    pipe.expire(key, window_seconds)
    results = pipe.execute()

    request_count = results[2]
    return request_count <= limit

# simulate 15 requests from user "42"
for i in range(15):
    allowed = is_allowed("42", limit=10, window_seconds=60)
    print(f"Request {i+1}: {'OK' if allowed else 'RATE LIMITED'}")
```

The pipeline batches all four commands into one round trip, keeping the check atomic enough for most applications. For strict atomicity, use a Lua script with `r.eval()`.

## API response caching

Cache expensive API calls with a simple wrapper:

```python
import redis
import httpx
import json
import hashlib
import os

r = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)

def cached_get(url: str, params: dict | None = None, ttl: int = 300) -> dict:
    cache_key = "cache:" + hashlib.sha256(f"{url}:{params}".encode()).hexdigest()

    cached = r.get(cache_key)
    if cached:
        print(f"Cache HIT for {url}")
        return json.loads(cached)

    print(f"Cache MISS for {url}")
    response = httpx.get(url, params=params, timeout=10.0)
    response.raise_for_status()
    data = response.json()

    r.set(cache_key, json.dumps(data), ex=ttl)
    return data

# first call hits the network; second call returns from Redis
data1 = cached_get(
    "https://api.github.com/repos/python/cpython",
    headers={"Accept": "application/vnd.github+json"},
)
data2 = cached_get(
    "https://api.github.com/repos/python/cpython",
    headers={"Accept": "application/vnd.github+json"},
)

print(data1["stargazers_count"])
```

## Session storage pattern

Store user sessions as hashes with an expiry. This is the same approach used by Flask-Session, Django's Redis cache backend, and most production web frameworks:

```python
import redis
import uuid
import json
import time

r = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)

SESSION_TTL = 3600  # 1 hour

def create_session(user_id: int, metadata: dict | None = None) -> str:
    session_id = str(uuid.uuid4())
    key = f"session:{session_id}"

    data = {
        "user_id": str(user_id),
        "created_at": str(time.time()),
        **(metadata or {}),
    }
    r.hset(key, mapping=data)
    r.expire(key, SESSION_TTL)
    return session_id


def get_session(session_id: str) -> dict | None:
    key = f"session:{session_id}"
    data = r.hgetall(key)
    if not data:
        return None
    r.expire(key, SESSION_TTL)   # sliding expiry: refresh TTL on access
    return data


def delete_session(session_id: str) -> None:
    r.delete(f"session:{session_id}")


# create a session after login
sid = create_session(user_id=42, metadata={"ip": "192.168.1.1", "ua": "Mozilla/5.0"})
print(f"Session created: {sid}")

# retrieve on subsequent requests
session = get_session(sid)
print(f"User: {session['user_id']}, IP: {session['ip']}")

# destroy on logout
delete_session(sid)
print(f"Session {sid} deleted. Exists: {bool(get_session(sid))}")
```

## Using a connection pool

For long-running applications, create a pool once at startup rather than reconnecting on every request:

```python
import redis

pool = redis.ConnectionPool(
    host="localhost",
    port=6379,
    db=0,
    decode_responses=True,
    max_connections=20,
)

def get_redis() -> redis.Redis:
    return redis.Redis(connection_pool=pool)

r = get_redis()
r.set("app:status", "running")
print(r.get("app:status"))
```

## Run the examples

```bash
bunpy rate_limiter.py
bunpy cache.py
bunpy session.py
```

Redis pipelines batch multiple commands into one network round trip - always use them when you need to fire several commands together. For operations that must be atomic, reach for Lua scripts via `r.eval()`.

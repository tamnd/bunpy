---
title: Setup and teardown
description: Run code before and after tests with lifecycle hooks.
weight: 6
---

bunpy's test runner provides four lifecycle hooks that control when setup and cleanup code runs relative to your tests. The hooks map directly to xUnit-style fixtures and are familiar to anyone who has used Jest, Mocha, or pytest.

## Hook summary

| Hook | Runs | Scope |
|------|------|-------|
| `beforeAll` | Once before any test in the file | Per file (suite) |
| `afterAll` | Once after all tests in the file | Per file (suite) |
| `beforeEach` | Before every individual test | Per file |
| `afterEach` | After every individual test | Per file |

Import all four from `bunpy.test`:

```python
from bunpy.test import test, expect, beforeAll, afterAll, beforeEach, afterEach
```

## beforeAll and afterAll

`beforeAll` runs once before the first test in the file. `afterAll` runs once after the last test, even if some tests failed. Use them for expensive setup that would be wasteful to repeat per test: database connections, test servers, large fixture files.

```python
from bunpy.test import test, expect, beforeAll, afterAll

db = None

@beforeAll
def start_database():
    global db
    db = Database.connect("postgresql://localhost/test_myapp")
    db.run_migrations()

@afterAll
def stop_database():
    db.drop_all_tables()
    db.disconnect()

@test("creates a user")
def _():
    user = db.users.create(name="Alice", email="alice@example.com")
    expect(user.id).not_.to_be_none()

@test("finds a user by email")
def _():
    db.users.create(name="Bob", email="bob@example.com")
    found = db.users.find_by_email("bob@example.com")
    expect(found.name).to_be("Bob")
```

The database connection is opened once and reused across both tests.

## beforeEach and afterEach

`beforeEach` runs before every test in the file. `afterEach` runs after every test, including failing ones. Use them to reset shared state so tests do not bleed into each other.

```python
from bunpy.test import test, expect, beforeEach, afterEach

db = None

@beforeEach
def reset_tables():
    db.truncate_all()

@afterEach
def log_test_result():
    # useful for debugging — runs even on failure
    pass

@test("counter starts at zero")
def _():
    counter = db.counters.create()
    expect(counter.value).to_be(0)

@test("counter increments")
def _():
    counter = db.counters.create()
    counter.increment()
    expect(counter.value).to_be(1)
```

Each test sees a clean set of tables because `beforeEach` truncates before each run. Without this, the second test would see the counter created by the first test.

## Database fixture pattern

A complete database test pattern combining all four hooks:

```python
import psycopg2
from bunpy.test import test, expect, beforeAll, afterAll, beforeEach

conn = None
cursor = None

@beforeAll
def connect():
    global conn, cursor
    conn = psycopg2.connect(
        dbname="test_myapp",
        user="postgres",
        host="localhost",
    )
    conn.autocommit = False
    cursor = conn.cursor()
    cursor.execute(open("schema.sql").read())
    conn.commit()

@afterAll
def disconnect():
    cursor.close()
    conn.close()

@beforeEach
def begin_transaction():
    # Wrap each test in a transaction that is rolled back after
    conn.rollback()

@test("inserts a product")
def _():
    cursor.execute(
        "INSERT INTO products (name, price) VALUES (%s, %s) RETURNING id",
        ("Widget", 9.99),
    )
    row = cursor.fetchone()
    expect(row[0]).to_be_greater_than(0)

@test("lists products")
def _():
    cursor.execute("INSERT INTO products (name, price) VALUES (%s, %s)", ("Gadget", 19.99))
    cursor.execute("SELECT COUNT(*) FROM products")
    count = cursor.fetchone()[0]
    expect(count).to_be(1)   # only the row from this test, rolled back after
```

The `begin_transaction` hook rolls back any changes from the previous test before starting the next one. Each test runs in isolation without the cost of re-creating the schema or reconnecting.

## HTTP server lifecycle

For integration tests that need a live HTTP server:

```python
import threading
import urllib.request
from http.server import HTTPServer, BaseHTTPRequestHandler
from bunpy.test import test, expect, beforeAll, afterAll

server = None
server_thread = None
base_url = ""

class EchoHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        body = b"OK"
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *args):
        pass  # suppress access log in test output

@beforeAll
def start_server():
    global server, server_thread, base_url
    server = HTTPServer(("127.0.0.1", 0), EchoHandler)  # port 0 = OS assigns
    port = server.server_address[1]
    base_url = f"http://127.0.0.1:{port}"
    server_thread = threading.Thread(target=server.serve_forever, daemon=True)
    server_thread.start()

@afterAll
def stop_server():
    server.shutdown()
    server_thread.join(timeout=2)

@test("server responds 200")
def _():
    resp = urllib.request.urlopen(f"{base_url}/ping")
    expect(resp.status).to_be(200)

@test("server returns OK body")
def _():
    resp = urllib.request.urlopen(f"{base_url}/ping")
    body = resp.read()
    expect(body).to_be(b"OK")
```

The server starts once, handles both tests, and shuts down cleanly. Using `port=0` lets the OS assign a free port, avoiding conflicts when multiple test workers run in parallel.

## Temporary directory cleanup

Use `afterEach` to remove temporary files created during a test:

```python
import os
import shutil
import tempfile
from bunpy.test import test, expect, beforeEach, afterEach

tmp_dir = None

@beforeEach
def make_tmpdir():
    global tmp_dir
    tmp_dir = tempfile.mkdtemp(prefix="bunpy_test_")

@afterEach
def remove_tmpdir():
    shutil.rmtree(tmp_dir, ignore_errors=True)

@test("writes and reads a file")
def _():
    path = os.path.join(tmp_dir, "hello.txt")
    with open(path, "w") as f:
        f.write("hello")
    with open(path) as f:
        expect(f.read()).to_be("hello")
```

`ignore_errors=True` in `rmtree` ensures cleanup does not mask the original test failure when the test itself fails after creating files.

## Async lifecycle hooks

All four hooks support `async def`:

```python
import asyncio
from bunpy.test import test, expect, beforeAll, afterAll

client = None

@beforeAll
async def connect():
    global client
    client = await AsyncDatabaseClient.connect("postgresql://localhost/test")

@afterAll
async def disconnect():
    await client.close()

@test("async query")
async def _():
    result = await client.fetch("SELECT 1 AS n")
    expect(result[0]["n"]).to_be(1)
```

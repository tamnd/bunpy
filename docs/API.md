# Python API surface (`bunpy.*`)

Every Bun namespace from the canonical `llms.txt` index has a
Python counterpart with snake_case names. The `bunpy` package is
registered as a built-in into the runtime; no install needed.

```python
import bunpy
```

## Server

```python
server = bunpy.serve(
    port=3000,
    routes={"/api/users/:id": handle_user},
    fetch=fallback_handler,           # called when no route matches
    websocket={"open": ..., "message": ...},
    tls={"cert": ..., "key": ...},
)
```

## File I/O

```python
bunpy.file("README.md").text()
bunpy.file("data.bin").bytes()
bunpy.file("config.json").json()
bunpy.write("out.txt", "hello")
bunpy.read("README.md")               # alias for file().text()
```

## SQL

```python
db = bunpy.sql("sqlite://app.db")     # or postgres://, mysql://
rows = db("SELECT * FROM users WHERE id = ?", 1).all()
one  = db("SELECT * FROM users WHERE id = ?", 1).one()
for row in db("SELECT * FROM users").iter():
    ...
```

## Redis / S3 / Shell

```python
r = bunpy.redis("redis://localhost:6379")
r.set("key", "value")

s3 = bunpy.s3({"region": "us-east-1", "bucket": "logs"})
url = s3.presign("path/to/object", method="GET", expires_in=3600)

result = bunpy.shell("ls", "-la").text()
out = bunpy.dollar("git status").text()    # identifier-safe form of bunpy.$
```

## Spawn / Glob / Cron

```python
proc = bunpy.spawn(["python", "-c", "print(1)"])

for path in bunpy.glob("**/*.py"):
    print(path)

@bunpy.cron("*/5 * * * *")
def heartbeat():
    print("alive")
```

## Web-platform globals

```python
resp = await fetch("https://example.com")
req  = Request("https://example.com", method="POST", body=b"hi")
url  = URL("https://example.com/?x=1")

setTimeout(lambda: print("later"), 1000)

ws = WebSocket("wss://echo.example")
ws.send("hi")
```

`bunpy --no-globals` opts out. The PEP 8 contingent gets `from
bunpy.web import fetch, Request, Response`.

## Crypto / hash / password

```python
hashed = bunpy.password.hash("hunter2")
ok     = bunpy.password.verify("hunter2", hashed)

key  = bunpy.generate_crypto_key()
uuid = bunpy.random_uuid_v7()
sha  = bunpy.hash("blake3", b"data")
```

## Compression / encoding

```python
bunpy.gzip(b"hello world")
bunpy.gunzip(g)
bunpy.deflate(b"...")
bunpy.inflate(d)
bunpy.base64.encode(b"abc")
bunpy.base64.decode("YWJj")
```

## YAML / HTML / cookies / CSRF

```python
data = bunpy.YAML.parse(open("config.yaml").read())
text = bunpy.YAML.stringify(data)

safe   = bunpy.escape_html("<script>")
parsed = bunpy.cookie.parse("a=1; b=2")
token  = bunpy.CSRF.generate(secret="...")
```

## Workers

```python
w = bunpy.Worker("worker.py")
w.post_message({"job": 1})
w.on_message(lambda msg: print(msg))
```

## FFI

```python
lib = bunpy.dlopen("libfoo.so", {
    "add": {"args": ["i32", "i32"], "returns": "i32"},
})
print(lib.add(1, 2))
```

## Build (also exposed in Python for build-time scripts)

```python
bunpy.build(entry="app.py", compile=True, target="darwin-arm64")
bunpy.plugin({"name": "my-loader", "setup": lambda b: ...})
```

## Test (used inside test files)

```python
from bunpy import describe, it, expect, beforeEach, mock, spy_on

def test_hashes_password():
    u = User("alice", "hunter2")
    expect(u.password_hash).not_to_equal("hunter2")
    expect(bunpy.password.verify("hunter2", u.password_hash)).to_be_truthy()
```

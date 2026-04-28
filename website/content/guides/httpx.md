---
title: HTTP requests with httpx
description: Make sync and async HTTP requests, handle sessions, retries, streaming, file uploads, and GitHub API pagination with httpx.
---

## Install

```bash
bunpy add httpx tenacity
```

## Basic GET request

```python
import httpx

response = httpx.get("https://httpbin.org/get")
print(response.status_code)   # 200
print(response.json())
```

Pass query parameters as a dict — httpx URL-encodes them automatically:

```python
import httpx

params = {"q": "python", "per_page": 5}
response = httpx.get("https://api.github.com/search/repositories", params=params)
data = response.json()
print(f"Total results: {data['total_count']}")
for repo in data["items"]:
    print(repo["full_name"], "—", repo["stargazers_count"], "stars")
```

## POST with JSON body

```python
import httpx

payload = {"title": "Hello", "body": "World", "userId": 1}
response = httpx.post("https://jsonplaceholder.typicode.com/posts", json=payload)
print(response.status_code)   # 201
print(response.json())
```

## Session with base URL and default headers

A `Client` is the right tool when you make multiple requests to the same host. It reuses the underlying TCP connection and lets you set shared headers once:

```python
import httpx
import os

GITHUB_TOKEN = os.environ.get("GITHUB_TOKEN", "")

with httpx.Client(
    base_url="https://api.github.com",
    headers={
        "Authorization": f"Bearer {GITHUB_TOKEN}",
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
    },
    timeout=10.0,
) as client:
    user = client.get("/user").json()
    print(f"Logged in as: {user['login']}")

    repos = client.get("/user/repos", params={"per_page": 5}).json()
    for repo in repos:
        print(repo["full_name"])
```

## Paginate GitHub API results

GitHub paginates most list endpoints. Keep fetching until there is no `next` link:

```python
import httpx
import os

def get_all_repos(org: str, token: str) -> list[dict]:
    repos: list[dict] = []
    url = f"https://api.github.com/orgs/{org}/repos"
    params: dict = {"per_page": 100, "type": "public"}

    with httpx.Client(
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/vnd.github+json",
        },
        timeout=15.0,
    ) as client:
        while url:
            response = client.get(url, params=params)
            response.raise_for_status()
            repos.extend(response.json())

            # Follow the Link header for the next page
            link = response.headers.get("link", "")
            next_url = None
            for part in link.split(","):
                if 'rel="next"' in part:
                    next_url = part.split(";")[0].strip().strip("<>")
            url = next_url
            params = {}  # next URL already has query string baked in

    return repos

token = os.environ["GITHUB_TOKEN"]
repos = get_all_repos("python", token)
print(f"Found {len(repos)} repos")
```

## Handle rate limits

The GitHub API returns a `Retry-After` or `X-RateLimit-Reset` header when you are throttled. Check for 429 and 403 and back off:

```python
import httpx
import time
import os

def github_get(client: httpx.Client, url: str, **kwargs) -> httpx.Response:
    while True:
        response = client.get(url, **kwargs)

        if response.status_code == 403:
            reset_at = int(response.headers.get("X-RateLimit-Reset", 0))
            sleep_for = max(reset_at - int(time.time()), 1)
            print(f"Rate limited. Sleeping {sleep_for}s")
            time.sleep(sleep_for)
            continue

        if response.status_code == 429:
            retry_after = int(response.headers.get("Retry-After", 5))
            print(f"Too many requests. Sleeping {retry_after}s")
            time.sleep(retry_after)
            continue

        response.raise_for_status()
        return response

with httpx.Client(
    headers={"Authorization": f"Bearer {os.environ['GITHUB_TOKEN']}"},
    timeout=10.0,
) as client:
    data = github_get(client, "https://api.github.com/repos/python/cpython").json()
    print(data["stargazers_count"])
```

## Retry with tenacity

For transient network failures — connection resets, 500 errors — wrap requests with tenacity:

```python
import httpx
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type

@retry(
    retry=retry_if_exception_type((httpx.TransportError, httpx.HTTPStatusError)),
    stop=stop_after_attempt(5),
    wait=wait_exponential(multiplier=1, min=1, max=30),
)
def fetch_with_retry(url: str) -> dict:
    response = httpx.get(url, timeout=10.0)
    response.raise_for_status()
    return response.json()

data = fetch_with_retry("https://api.github.com/repos/python/cpython")
print(data["name"])
```

## Upload a file

```python
import httpx

with open("report.csv", "rb") as f:
    response = httpx.post(
        "https://httpbin.org/post",
        files={"file": ("report.csv", f, "text/csv")},
        data={"description": "Monthly sales report"},
    )

print(response.status_code)
print(response.json()["files"])
```

## Custom authentication

Subclass `httpx.Auth` to attach credentials to every request:

```python
import httpx

class APIKeyAuth(httpx.Auth):
    def __init__(self, api_key: str) -> None:
        self.api_key = api_key

    def auth_flow(self, request: httpx.Request):
        request.headers["X-Api-Key"] = self.api_key
        yield request

with httpx.Client(auth=APIKeyAuth("secret-key-here")) as client:
    response = client.get("https://httpbin.org/headers")
    print(response.json()["headers"].get("X-Api-Key"))
```

## Streaming response

Stream large downloads instead of loading the whole body into memory:

```python
import httpx

url = "https://speed.hetzner.de/100MB.bin"

with httpx.stream("GET", url, timeout=60.0) as response:
    response.raise_for_status()
    total = int(response.headers.get("content-length", 0))
    downloaded = 0

    with open("large_file.bin", "wb") as f:
        for chunk in response.iter_bytes(chunk_size=65536):
            f.write(chunk)
            downloaded += len(chunk)
            if total:
                pct = downloaded / total * 100
                print(f"\r{pct:.1f}%", end="", flush=True)

print("\nDone.")
```

## Async client

Use `httpx.AsyncClient` inside an `async def` function. This is the right choice when you are already in an async context (FastAPI, asyncio scripts, etc.):

```python
import asyncio
import httpx

async def fetch_repos(usernames: list[str]) -> dict[str, int]:
    results: dict[str, int] = {}

    async with httpx.AsyncClient(
        base_url="https://api.github.com",
        timeout=10.0,
    ) as client:
        tasks = [client.get(f"/users/{u}/repos", params={"per_page": 100}) for u in usernames]
        responses = await asyncio.gather(*tasks)

        for username, response in zip(usernames, responses):
            response.raise_for_status()
            results[username] = len(response.json())

    return results

counts = asyncio.run(fetch_repos(["torvalds", "gvanrossum", "antirez"]))
for user, count in counts.items():
    print(f"{user}: {count} repos")
```

## Async streaming

```python
import asyncio
import httpx

async def stream_download(url: str, dest: str) -> None:
    async with httpx.AsyncClient(timeout=60.0) as client:
        async with client.stream("GET", url) as response:
            response.raise_for_status()
            with open(dest, "wb") as f:
                async for chunk in response.aiter_bytes(chunk_size=65536):
                    f.write(chunk)

asyncio.run(stream_download("https://httpbin.org/bytes/1024", "output.bin"))
print("Downloaded.")
```

## Timeout configuration

Set separate timeouts for connect, read, write, and pool:

```python
import httpx

timeout = httpx.Timeout(
    connect=5.0,   # time to establish TCP connection
    read=30.0,     # time to receive data
    write=10.0,    # time to send request body
    pool=5.0,      # time to acquire a connection from the pool
)

with httpx.Client(timeout=timeout) as client:
    response = client.get("https://httpbin.org/delay/2")
    print(response.status_code)
```

## Run the examples

```bash
# Set your token once
export GITHUB_TOKEN=ghp_...

# Run any script directly
bunpy github_repos.py
```

httpx keeps the same familiar requests-style API for synchronous code while offering a true async client. The session (`Client`) approach is almost always the right default: it handles connection pooling, keeps headers DRY, and makes retry wrappers straightforward to add.

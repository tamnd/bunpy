---
title: Watch mode
description: Re-run tests automatically when source files change.
weight: 3
---

```bash
bunpy test --watch
```

Watch mode monitors the source files of every test file and re-runs the
affected tests on each save. Useful during TDD cycles.

## Behaviour

- On startup, all tests run once.
- When a test file changes, only that file's tests re-run.
- When a non-test source file changes, all tests that import it (directly or
  transitively) re-run.
- The terminal clears and shows fresh results on each re-run.
- A summary line shows total pass / fail counts and elapsed time.

## Filter in watch mode

```bash
bunpy test --watch --filter "auth"
```

Only tests whose name contains "auth" are collected and re-run.

## Exit

Press `Ctrl+C` to stop the watcher.

## Interactive commands

While the watcher is running, pressing:

| Key | Action |
|-----|--------|
| `a` | Re-run all tests immediately |
| `f` | Show only failing tests |
| `q` | Quit |

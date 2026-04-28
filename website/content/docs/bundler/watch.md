---
title: Watch mode
description: Rebuild automatically when source files change.
weight: 6
---

```bash
bunpy build --watch app.py
```

Watch mode polls for changes to the entry point and all imported modules.
On any change, it rebuilds and prints the result:

```
[watch] built app.pyz (12 KB, 3.2 ms)
[watch] rebuilt app.pyz - 1 change (2.8 ms)
[watch] error in mylib/utils.py:7 - SyntaxError: invalid syntax
```

Errors do not stop the watch process - bunpy waits for the next save and
tries again.

## Combining with --compile

```bash
bunpy build --compile --watch app.py
```

Watch + compile is slower per rebuild (full VM link each time) but useful
when testing the compiled binary.

## Exit

Press `Ctrl+C` to stop the watcher.

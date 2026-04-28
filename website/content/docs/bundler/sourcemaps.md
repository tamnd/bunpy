---
title: Source maps
description: Map bundle line numbers back to original source files.
weight: 3
---

```bash
bunpy build --sourcemap app.py
# → app.pyz
# → app.pyz.map
```

The `.pyz.map` file is a JSON source map that bunpy uses to rewrite
tracebacks. When a bundled script raises an exception, bunpy looks up the
`.pyz.map` alongside the `.pyz` and prints original file names and line
numbers instead of bundle-internal offsets.

## Source map format

```json
{
  "version": 1,
  "sources": ["app.py", "mylib/utils.py"],
  "mappings": "..."
}
```

## Deploying with source maps

Keep the `.pyz.map` next to the `.pyz` on the deployment host. bunpy
auto-discovers it by appending `.map` to the bundle path.

If the map file is absent, tracebacks show bundle-internal line numbers.

---
title: Minify
description: Strip whitespace, comments, and docstrings from the bundle.
weight: 2
---

```bash
bunpy build --minify app.py
```

The `--minify` flag strips:

- Blank lines and indentation (byte-safe — preserved inside strings)
- Single-line `#` comments
- Module-level and function-level docstrings

The minified source is embedded in the `.pyz`. The bytecode compiled from
the minified source is functionally identical to the original.

## Combining with --compile

```bash
bunpy build --compile --minify app.py -o dist/myapp
```

Minification reduces the embedded source size in the binary, not the binary
size itself (the VM is the dominant factor).

## When to use

Minification is primarily useful when distributing `.pyz` files where source
confidentiality is a concern. It is not a security measure — the bytecode is
still human-readable with a disassembler.

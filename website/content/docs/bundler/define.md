---
title: Compile-time defines
description: Replace named constants at bundle time with --define.
weight: 4
---

```bash
bunpy build --define DEBUG=False --define VERSION=1.2.3 app.py
```

`--define KEY=VALUE` replaces occurrences of the bare name `KEY` in the
compiled source with the literal `VALUE` before compilation. This is useful for:

- Feature flags (`DEBUG`, `BETA`, `ENABLE_NEW_UI`)
- Compile-time version strings
- Environment-specific configuration without runtime overhead

## Example

```python
# app.py
DEBUG = True   # will be replaced

if DEBUG:
    print("debug mode")
```

```bash
bunpy build --define DEBUG=False app.py
```

The compiled bytecode is equivalent to:

```python
DEBUG = False

if DEBUG:   # constant fold → dead branch eliminated
    print("debug mode")
```

## Limitations

- Only top-level bare name occurrences are replaced (not attribute access)
- VALUE is inserted as a Python literal; use quotes for strings:
  `--define GREETING="'hello'"` → `GREETING = 'hello'`
- Multiple `--define` flags can be combined

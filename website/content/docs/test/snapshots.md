---
title: Snapshots
description: Assert complex values with stored snapshots.
weight: 2
---

Snapshot testing lets you assert that a complex value matches a previously
stored "snapshot". On the first run, the snapshot is written to disk. On
subsequent runs, bunpy compares the current value to the stored snapshot and
fails the test if they differ.

## Basic usage

```python
from bunpy.test import test
from bunpy.snapshot import toMatchSnapshot

@test("renders the right HTML")
def _():
    html = render_page(title="Hello")
    toMatchSnapshot(html)
```

First run (no snapshot exists):

```
✓ renders the right HTML  [snapshot created]
```

Second run (snapshot exists):

```
✓ renders the right HTML  [snapshot matched]
```

If the output changed:

```
✗ renders the right HTML  [snapshot mismatch]
  - Expected: <h1>Hello</h1>
  + Received: <h1>Hello!</h1>
```

## Updating snapshots

To accept new output as the new baseline:

```bash
bunpy test --update-snapshots
```

## Snapshot file location

Snapshots are stored in `__snapshots__/` next to the test file:

```
tests/
├── test_render.py
└── __snapshots__/
    └── test_render.py.snap
```

The snapshot file is JSON and should be committed to version control.

## Named snapshots

```python
toMatchSnapshot(result, name="with-title")
```

Names are used to distinguish multiple snapshots within the same test.

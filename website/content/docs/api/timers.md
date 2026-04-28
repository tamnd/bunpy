---
title: bunpy.timers
description: setTimeout, setInterval, and their clear counterparts.
---

Timers are available as globals in every bunpy script. They are also
importable from `bunpy.timers`.

```python
from bunpy.timers import setTimeout, clearTimeout, setInterval, clearInterval
```

## setTimeout(fn, delay_ms) → id

Call `fn` after `delay_ms` milliseconds. Returns a timer ID.

```python
tid = setTimeout(lambda: print("fired!"), 1000)
```

## clearTimeout(id)

Cancel a pending timeout.

```python
clearTimeout(tid)
```

## setInterval(fn, interval_ms) → id

Call `fn` every `interval_ms` milliseconds. Returns a timer ID.

```python
count = 0
iid = setInterval(lambda: (count := count + 1), 100)
```

## clearInterval(id)

Stop a repeating interval.

```python
clearInterval(iid)
```

## Notes

- Timers run on goroutines. The callback is called from a separate goroutine.
- Use `threading.Lock` if the callback mutates shared state.
- A script exits when the main goroutine finishes, even if timers are pending.
  Use `time.sleep` or an event to keep the main goroutine alive.

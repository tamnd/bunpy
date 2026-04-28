---
title: bunpy.cron
description: Cron-style task scheduling.
---

```python
import bunpy.cron as cron
```

## Scheduling a task

```python
@cron.job("0 * * * *")   # every hour at :00
def hourly_report():
    print("generating report...")
```

Or without a decorator:

```python
def my_task():
    print("running")

cron.schedule("*/5 * * * *", my_task)   # every 5 minutes
```

## Cron expression format

```
┌───────────── minute (0–59)
│ ┌───────────── hour (0–23)
│ │ ┌───────────── day of month (1–31)
│ │ │ ┌───────────── month (1–12)
│ │ │ │ ┌───────────── day of week (0–6, Sunday=0)
│ │ │ │ │
* * * * *
```

Common patterns:

| Expression | Meaning |
|------------|---------|
| `* * * * *` | Every minute |
| `0 * * * *` | Every hour |
| `0 0 * * *` | Every day at midnight |
| `0 9 * * 1` | Every Monday at 9:00 |
| `*/15 * * * *` | Every 15 minutes |

## Listing scheduled jobs

```python
jobs = cron.jobs()
for j in jobs:
    print(j.expression, j.last_run, j.next_run)
```

## Stopping a job

```python
job = cron.schedule("* * * * *", my_task)
job.stop()
```

## Notes

- Jobs run on goroutines. The callback is called from a separate goroutine.
- bunpy keeps the process alive as long as at least one cron job is active.
  Call `cron.stop_all()` to exit the scheduler.

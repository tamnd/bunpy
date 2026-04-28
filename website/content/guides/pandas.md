---
title: Data processing with pandas
description: "Read, filter, group, merge, and export tabular data with pandas. Real-world example: compute monthly sales totals and export a report."
---

## Install

```bash
bunpy add pandas openpyxl pyarrow
```

`openpyxl` is needed for reading/writing `.xlsx` files. `pyarrow` enables Parquet support.

## Read data

```python
import pandas as pd

# CSV
df = pd.read_csv("sales.csv")

# JSON (records format)
df = pd.read_json("sales.json")

# Excel
df = pd.read_excel("sales.xlsx", sheet_name="Sheet1")

# inspect
print(df.head())
print(df.shape)       # (rows, columns)
print(df.dtypes)
print(df.describe())  # summary statistics for numeric columns
```

Generate sample data to follow along:

```python
import pandas as pd
import numpy as np

rng = np.random.default_rng(42)

df = pd.DataFrame({
    "date": pd.date_range("2024-01-01", periods=200, freq="D"),
    "region": rng.choice(["North", "South", "East", "West"], size=200),
    "product": rng.choice(["Widget A", "Widget B", "Widget C"], size=200),
    "units": rng.integers(1, 50, size=200),
    "unit_price": rng.uniform(10.0, 200.0, size=200).round(2),
})
df["revenue"] = (df["units"] * df["unit_price"]).round(2)

df.to_csv("sales.csv", index=False)
print(f"Generated {len(df)} rows")
```

## Filter and select

```python
import pandas as pd

df = pd.read_csv("sales.csv", parse_dates=["date"])

# boolean indexing
north = df[df["region"] == "North"]
print(f"North rows: {len(north)}")

# multiple conditions
high_value = df[(df["revenue"] > 500) & (df["region"].isin(["North", "East"]))]

# select columns
summary = df[["date", "region", "revenue"]]

# query string (more readable for complex filters)
q = df.query("revenue > 1000 and product == 'Widget A'")
print(q.shape)

# date filtering
jan = df[df["date"].dt.month == 1]
print(f"January rows: {len(jan)}")
```

## Add and transform columns

```python
import pandas as pd

df = pd.read_csv("sales.csv", parse_dates=["date"])

df["month"] = df["date"].dt.to_period("M")
df["quarter"] = df["date"].dt.quarter
df["year"] = df["date"].dt.year

# apply a function to a column
df["revenue_tier"] = pd.cut(
    df["revenue"],
    bins=[0, 200, 600, float("inf")],
    labels=["low", "medium", "high"],
)

# string operations
df["region_lower"] = df["region"].str.lower()

print(df[["date", "month", "quarter", "revenue_tier"]].head())
```

## groupby and aggregate

```python
import pandas as pd

df = pd.read_csv("sales.csv", parse_dates=["date"])
df["month"] = df["date"].dt.to_period("M")

# total revenue per region
by_region = df.groupby("region")["revenue"].sum().sort_values(ascending=False)
print(by_region)

# multiple aggregations at once
summary = df.groupby(["region", "product"]).agg(
    total_units=("units", "sum"),
    total_revenue=("revenue", "sum"),
    avg_price=("unit_price", "mean"),
    order_count=("revenue", "count"),
).round(2)
print(summary)

# monthly totals
monthly = df.groupby("month").agg(
    revenue=("revenue", "sum"),
    orders=("revenue", "count"),
).reset_index()
monthly["month"] = monthly["month"].astype(str)
print(monthly)
```

## Pivot tables

```python
import pandas as pd

df = pd.read_csv("sales.csv", parse_dates=["date"])
df["month"] = df["date"].dt.month_name()

pivot = pd.pivot_table(
    df,
    values="revenue",
    index="region",
    columns="product",
    aggfunc="sum",
    fill_value=0,
).round(2)

print(pivot)
```

## Merge DataFrames

```python
import pandas as pd

# sales data
sales = pd.DataFrame({
    "order_id": [1, 2, 3, 4],
    "customer_id": [10, 20, 10, 30],
    "revenue": [150.0, 300.0, 200.0, 450.0],
})

# customer data
customers = pd.DataFrame({
    "customer_id": [10, 20, 30],
    "name": ["Alice", "Bob", "Carol"],
    "tier": ["gold", "silver", "gold"],
})

# inner join (only matching rows)
merged = sales.merge(customers, on="customer_id", how="inner")
print(merged)

# left join (keep all sales, fill missing customer info with NaN)
merged_left = sales.merge(customers, on="customer_id", how="left")

# aggregate after merge
by_tier = merged.groupby("tier")["revenue"].agg(["sum", "mean", "count"]).round(2)
print(by_tier)
```

## Sort, rank, and deduplicate

```python
import pandas as pd

df = pd.read_csv("sales.csv")

# sort by multiple columns
df_sorted = df.sort_values(["region", "revenue"], ascending=[True, False])

# rank within group
df["rank_in_region"] = df.groupby("region")["revenue"].rank(method="dense", ascending=False)

# top 3 per region
top3 = df[df["rank_in_region"] <= 3].sort_values(["region", "rank_in_region"])
print(top3[["region", "product", "revenue", "rank_in_region"]])

# drop duplicates (keep first occurrence)
df_unique = df.drop_duplicates(subset=["region", "product"])
```

## Export to CSV, JSON, and Parquet

```python
import pandas as pd

df = pd.read_csv("sales.csv", parse_dates=["date"])
df["month"] = df["date"].dt.to_period("M").astype(str)

report = df.groupby(["month", "region", "product"]).agg(
    total_revenue=("revenue", "sum"),
    total_units=("units", "sum"),
    order_count=("revenue", "count"),
).round(2).reset_index()

# CSV
report.to_csv("monthly_report.csv", index=False)

# JSON (records - one object per row)
report.to_json("monthly_report.json", orient="records", indent=2)

# Excel with formatting
with pd.ExcelWriter("monthly_report.xlsx", engine="openpyxl") as writer:
    report.to_excel(writer, sheet_name="Monthly Summary", index=False)

# Parquet - best for large datasets (columnar, compressed)
report.to_parquet("monthly_report.parquet", index=False)

print("Exported monthly_report.csv / .json / .xlsx / .parquet")
```

## Memory-efficient chunked reading

When a CSV is too large to fit in memory, process it in chunks:

```python
import pandas as pd

chunk_size = 50_000
totals: dict[str, float] = {}

for chunk in pd.read_csv("large_sales.csv", chunksize=chunk_size):
    group = chunk.groupby("region")["revenue"].sum()
    for region, rev in group.items():
        totals[region] = totals.get(region, 0.0) + rev

for region, total in sorted(totals.items()):
    print(f"{region}: ${total:,.2f}")
```

## Real-world: monthly sales report

A complete script that reads raw sales data, computes monthly totals by region and product, and writes a formatted Excel report:

```python
import pandas as pd
import numpy as np
from pathlib import Path

# --- Generate sample data (replace with your real CSV) ---
rng = np.random.default_rng(0)
df_raw = pd.DataFrame({
    "date": pd.date_range("2024-01-01", "2024-12-31", freq="D").repeat(3)[:365],
    "region": rng.choice(["North", "South", "East", "West"], 365),
    "product": rng.choice(["Widget A", "Widget B", "Widget C"], 365),
    "units": rng.integers(1, 100, 365),
    "unit_price": rng.uniform(10, 300, 365).round(2),
})
df_raw["revenue"] = (df_raw["units"] * df_raw["unit_price"]).round(2)
df_raw.to_csv("sales_2024.csv", index=False)

# --- Load and process ---
df = pd.read_csv("sales_2024.csv", parse_dates=["date"])
df["month"] = df["date"].dt.to_period("M")
df["month_name"] = df["date"].dt.strftime("%B %Y")

# monthly totals
monthly = (
    df.groupby(["month", "month_name", "region", "product"])
    .agg(units=("units", "sum"), revenue=("revenue", "sum"), orders=("revenue", "count"))
    .round(2)
    .reset_index()
    .sort_values(["month", "region", "product"])
)

# overall monthly summary
monthly_summary = (
    df.groupby(["month", "month_name"])
    .agg(total_revenue=("revenue", "sum"), total_orders=("revenue", "count"))
    .round(2)
    .reset_index()
)

# best product per month
best_product = (
    df.groupby(["month", "product"])["revenue"]
    .sum()
    .reset_index()
    .sort_values(["month", "revenue"], ascending=[True, False])
    .groupby("month")
    .first()
    .reset_index()[["month", "product", "revenue"]]
    .rename(columns={"product": "top_product", "revenue": "top_revenue"})
)

# --- Export ---
output = Path("report_2024.xlsx")
with pd.ExcelWriter(output, engine="openpyxl") as writer:
    monthly_summary.to_excel(writer, sheet_name="Monthly Summary", index=False)
    monthly.to_excel(writer, sheet_name="Detail", index=False)
    best_product.to_excel(writer, sheet_name="Top Products", index=False)

print(f"Report written to {output}")
print(monthly_summary.to_string(index=False))
```

## Run the script

```bash
bunpy sales_report.py
```

pandas holds large DataFrames in RAM. For datasets above a few gigabytes, consider chunked reading (shown above), DuckDB (`bunpy add duckdb`), or Polars (`bunpy add polars`), which follow the same general API but use lazy evaluation and multithreading by default.

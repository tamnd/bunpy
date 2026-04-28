#!/usr/bin/env bash
# bench.sh: run all benchmarks and optionally write a snapshot.
#
# Usage:
#   bash scripts/bench.sh                  # print results to stdout
#   bash scripts/bench.sh benchmarks/baseline.txt   # also write snapshot
#
# Fixtures must already exist. If they don't, generate them first:
#   go run benchmarks/fixtures/build_fixtures.go
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUTFILE="${1:-}"

# Run with count=5 and benchtime=3s for stable medians.
CMD=(go test
  -bench=.
  -benchmem
  -count=5
  -benchtime=3s
  ./benchmarks/
)

if [ -n "$OUTFILE" ]; then
  mkdir -p "$(dirname "$OUTFILE")"
  "${CMD[@]}" | tee "$OUTFILE"
  echo ""
  echo "snapshot written to $OUTFILE"
else
  "${CMD[@]}"
fi

#!/usr/bin/env bash
# merge_coverage.sh — Merge multiple Go coverage profiles, taking max count per block.
# Usage: bash scripts/merge_coverage.sh tmp_cov/*.out > merged.out
set -euo pipefail

echo "mode: set"

# For each unique block (file:startLine.startCol,endLine.endCol numStmts),
# take the max count across all profiles.
for f in "$@"; do
  tail -n +2 "$f"
done | awk '{
  key = $1 " " $2
  count = $3 + 0
  if (!(key in maxcount) || count > maxcount[key]) {
    maxcount[key] = count
    line[key] = $1 " " $2 " " count
  }
}
END {
  for (k in line) print line[k]
}' | sort

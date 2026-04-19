#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "Usage: $0 <goos> <goarch> <version> <output>" >&2
  exit 1
fi

goos="$1"
goarch="$2"
version="$3"
output="$4"

export CGO_ENABLED="1"
export GOOS="$goos"
export GOARCH="$goarch"

mkdir -p "$(dirname "$output")"

echo "[build] GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED"
if [[ -n "${CC:-}" ]]; then
  echo "[build] CC=$CC"
fi
if [[ -n "${CXX:-}" ]]; then
  echo "[build] CXX=$CXX"
fi

go build \
  -ldflags "-X main.version=${version}" \
  -o "$output" \
  ./cmd/server/

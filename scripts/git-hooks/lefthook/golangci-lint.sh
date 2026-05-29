#!/usr/bin/env bash
# golangci-lint en mode rapide. Skippé silencieusement si absent du PATH (CI s'en charge).
set -euo pipefail

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "[skip] golangci-lint absent du PATH (voir CI)"
  exit 0
fi
cd apps/go-api || exit 0
golangci-lint run --fast ./...

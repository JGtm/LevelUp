#!/usr/bin/env bash
# Scan des vulnérabilités Go (govulncheck). Skippé si absent du PATH (CI s'en charge).
set -euo pipefail

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "[skip] govulncheck absent du PATH (voir CI)"
  exit 0
fi
cd apps/go-api || exit 0
govulncheck ./...

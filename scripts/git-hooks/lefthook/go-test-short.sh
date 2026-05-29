#!/usr/bin/env bash
# Tests unitaires Go rapides (avant push).
set -euo pipefail
cd apps/go-api || exit 1
go test -short ./... -timeout 120s

#!/usr/bin/env bash
# go vet sur les packages hors cmd/server (CGO désactivé pour éviter DuckDB).
# Non bloquant (|| true) — parité avec l'ancien hook pre-commit.
cd apps/go-api || exit 0
pkgs=$(go list ./... 2>/dev/null | grep -v "cmd/server" | tr "\n" " ")
CGO_ENABLED=0 go vet $pkgs 2>&1 || true

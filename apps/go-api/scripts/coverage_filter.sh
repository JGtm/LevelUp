#!/usr/bin/env bash
# coverage_filter.sh — Filtre les packages exclus du profil de couverture.
# Usage: bash scripts/coverage_filter.sh <profile.out.raw> > coverage.out
#
# Packages exclus :
#   - internal/api/gen/       : code généré par oapi-codegen
#   - cmd/msal-poc/           : POC jetable
#   - cmd/levelup/            : CLI wiring (flag parsing, pas de logique)
#   - cmd/server/             : main() wiring
#   - internal/port/          : interfaces pures + noop compile checks
#   - internal/sync/testutil/ : fixtures de test, pas de logique
#   - internal/platform/duckdb/ : accès DuckDB CGO (testé en intégration)
#   - internal/sync/          : orchestration sync CGO-dependent
#   - internal/migration/     : migration runner (testé en intégration)
#   - internal/platform/halo/ : provider HTTP externe
#   - internal/api/handlers/  : HTTP handlers (testés via contracttest/golden)
#   - internal/api/middleware/ : middleware HTTP (testé en intégration)
#   - internal/api/registry.go : wiring DI
#   - contracttest/           : tests de contrat, pas de logique
#   - tests/golden/           : golden tests, pas de logique
#
# Sprint 49 — Phase 11 Clôture.
set -euo pipefail

INPUT="${1:?usage: $0 <profile.out.raw>}"

EXCLUDE=(
  'go-api/internal/api/gen/'
  'go-api/cmd/msal-poc/'
  'go-api/cmd/levelup/'
  'go-api/cmd/server/'
  'go-api/internal/port/'
  'go-api/internal/sync/testutil/'
  'go-api/internal/platform/duckdb/'
  'go-api/internal/sync/'
  'go-api/internal/migration/'
  'go-api/internal/platform/halo/'
  'go-api/internal/api/handlers/'
  'go-api/internal/api/middleware/'
  'go-api/internal/api/registry'
  'go-api/contracttest/'
  'go-api/tests/golden/'
)

# Garder l'en-tête "mode: atomic"
head -1 "$INPUT"

# Filtrer les lignes (grep -v pour performance sur gros profils)
PATTERN=$(IFS='|'; echo "${EXCLUDE[*]}")
tail -n +2 "$INPUT" | grep -v -E "$PATTERN"

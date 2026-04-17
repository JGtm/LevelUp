#!/usr/bin/env bash
# coverage_filter.sh — Filtre les packages exclus du profil de couverture.
# Usage: bash scripts/coverage_filter.sh <profile.out.raw> > coverage.out
#
# Packages exclus :
#   - internal/api/gen/       : code généré par oapi-codegen
#   - cmd/msal-poc/           : POC jetable
#
# Sprint 45 — Phase 10 Consolidation qualité.
set -euo pipefail

INPUT="${1:?usage: $0 <profile.out.raw>}"

EXCLUDE=(
  'go-api/internal/api/gen/'
  'go-api/cmd/msal-poc/'
)

# Garder l'en-tête "mode: atomic"
head -1 "$INPUT"

# Filtrer les lignes
tail -n +2 "$INPUT" | while IFS= read -r line; do
  keep=1
  for pat in "${EXCLUDE[@]}"; do
    if [[ "$line" == *"$pat"* ]]; then
      keep=0
      break
    fi
  done
  [[ $keep -eq 1 ]] && echo "$line"
done

#!/usr/bin/env bash
# coverage_filter.sh — Filtre les packages exclus du profil de couverture.
# Usage: bash scripts/coverage_filter.sh <profile.out.raw> > coverage.out
#
# Packages exclus, classés par justification (revue 2026-04-29 P3.1 — axe 7
# BLOQUANT « couverture mensongère ») :
#
# A) Exclusions LÉGITIMES (boilerplate / CGO non-mockable / wiring) :
#   - cmd/msal-poc/           : POC jetable
#   - cmd/levelup/, cmd/server/ : main() + flag parsing, pas de logique
#   - internal/port/          : interfaces pures (noop compile checks)
#   - internal/sync/testutil/ : fixtures de test
#   - internal/platform/duckdb/ : accès DuckDB CGO (testé en intégration)
#   - internal/sync/          : orchestration sync CGO-dependent
#   - internal/migration/     : migration runner CGO-dependent
#   - internal/api/registry.go : wiring DI (compile check)
#   - contracttest/, tests/golden/ : harnais de tests, pas de logique
#
# B) Exclusions DETTE — scope à ré-inclure progressivement :
#   - internal/api/handlers/  : HTTP handlers — testables via httptest +
#     mock port.*Service. P4 + P3.7 (extraction ports) doivent permettre
#     l'inclusion. Couverture cible : >= 70% sur handlers métier.
#   - internal/api/middleware/ : middleware testable via httptest. P3.6
#     ou phase dédiée.
#   - internal/platform/halo/ : provider HTTP — testable via httptest mock
#     server. P3.6 « tests platform/halo » est explicitement programmé
#     pour combler ce gap (6 sources : medal_provider, season_provider,
#     discovery_client, etc.).
#
# Sprint 49 — Phase 11 Clôture. Annotations 2026-04-29 P3.1.
set -euo pipefail

INPUT="${1:?usage: $0 <profile.out.raw>}"

# A — Légitimes (CGO/boilerplate)
EXCLUDE_LEGITIMATE=(
  'go-api/cmd/'              # tous les CLI/diag tools (56 sous-dirs : diag_*,
                             # backfill_*, check_*, audit_*, regen_*, etc.) —
                             # flag parsing + duckdb queries ad hoc, pas de
                             # logique métier testable (cf. 286/292 funcs à 0%
                             # malgré les usages prod réguliers). Extension de
                             # l'exclusion historique cmd/levelup/server/msal-poc.
  'go-api/scripts/'          # utilitaires Go (warm_bp_assets, import_bp_*,
                             # check_syncmeta, repair_tracks_table) — même
                             # rationale : scripts ponctuels, pas de logique
                             # à tester (23/23 funcs à 0%).
  'go-api/internal/port/'
  'go-api/internal/sync/testutil/'
  'go-api/internal/platform/duckdb/'
  'go-api/internal/sync/'
  'go-api/internal/migration/'
  'go-api/internal/api/registry'
  'go-api/contracttest/'
  'go-api/tests/golden/'
)

# B — Dette : à ré-inclure en P3.6 / P3.7 / P4. Pour l'instant exclu mais
# documenté ici pour audit. Re-enable progressivement quand les tests
# correspondants sont écrits.
EXCLUDE_DETTE=(
  'go-api/internal/platform/halo/'  # P3.6 (cible)
  'go-api/internal/api/handlers/'    # P4 / P3.7 (cible)
  'go-api/internal/api/middleware/'  # phase dédiée (cible)
)

EXCLUDE=("${EXCLUDE_LEGITIMATE[@]}" "${EXCLUDE_DETTE[@]}")

# Garder l'en-tête "mode: atomic"
head -1 "$INPUT"

# Filtrer les lignes (grep -v pour performance sur gros profils)
PATTERN=$(IFS='|'; echo "${EXCLUDE[*]}")
tail -n +2 "$INPUT" | grep -v -E "$PATTERN"

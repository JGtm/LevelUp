#!/usr/bin/env bash
# scripts/check_lease_enforcement.sh — vérifie que les services et handlers
# n'ouvrent pas de connexion DuckDB en write directement (sans lease).
#
# Référence : ADR 0013 — Leased writer enforcement.
#
# Garde-fou contre la régression de la garantie compile-time différée
# (cf. plan §commit 8) : un futur PR qui ajouterait `duckdb.OpenReadWrite`
# dans internal/service ou internal/api/handlers sans passer par le système
# de lease échouerait ce check.
#
# Sortie :
#   - exit 0 si aucune violation détectée
#   - exit 1 si une OpenReadWrite est trouvée dans un fichier non whitelisté
#
# Usage :
#   bash apps/go-api/scripts/check_lease_enforcement.sh
#
# Le script tolère :
#   - les sites historiques du sync engine (utilise déjà sync.AcquireLeaseCtx)
#   - les utilitaires ops/migration/scripts qui ouvrent ponctuellement la DB
#   - les fichiers _test.go (tests d'intégration peuvent ouvrir librement)
#   - le pool lui-même (internal/platform/duckdb/db.go, pool.go : c'est l'origine)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
GO_API="$REPO_ROOT/apps/go-api"

# Patterns interdits dans les paths sensibles (services + handlers).
PATTERNS=(
  "duckdb\.OpenReadWrite\("
  "duckdb\.OpenReadWriteShared\("
)

# Whitelist : fichiers qui ont une raison documentée d'ouvrir la DB en write
# directement. Toute addition à cette liste doit être justifiée dans le PR.
WHITELIST_REGEX='(_test\.go$|/db\.go$|/pool\.go$|/pool_writers\.go$|/persist_sink\.go$|/bootstrap_repo\.go$|api/prestige_setup\.go$|api/registry_notifications\.go$|api/registry\.go$|/migration/|/ops/|/cmd/|/scripts/|/sync/|/assets/|/platform/auth/|/platform/duckdb/|/api/server\.go$)'

SCAN_DIRS=(
  "$GO_API/internal/service"
  "$GO_API/internal/api/handlers"
)

violations=0
for dir in "${SCAN_DIRS[@]}"; do
  [[ -d "$dir" ]] || continue
  for pattern in "${PATTERNS[@]}"; do
    while IFS= read -r line; do
      [[ -z "$line" ]] && continue
      file="${line%%:*}"
      # Ignorer les fichiers whitelistés
      if [[ "$file" =~ $WHITELIST_REGEX ]]; then
        continue
      fi
      echo "❌ Lease enforcement violation:"
      echo "   $line"
      echo "   → Toute écriture DuckDB depuis service/ ou handlers/ doit"
      echo "     passer par un *LeasedWriter (cf. ADR 0013)."
      violations=$((violations + 1))
    done < <(grep -rEn "$pattern" "$dir" 2>/dev/null || true)
  done
done

if [[ $violations -gt 0 ]]; then
  echo ""
  echo "❌ $violations violation(s) trouvée(s)."
  echo ""
  echo "Pour résoudre :"
  echo "  - injecter l'écriture via un service qui acquiert un lease"
  echo "    (cf. notifications.WithWriterAcquirer, social.WithWriterAcquirer,"
  echo "    media.WithMediaWriterAcquirer, ou un wrapper LazyXxxService)."
  echo "  - si l'ouverture est légitime (tests, migration, script CLI),"
  echo "    ajouter le path à la WHITELIST_REGEX avec justification."
  exit 1
fi

echo "✅ Aucune violation : tous les writes service/handlers passent par lease."
exit 0

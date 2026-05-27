#!/usr/bin/env bash
# scripts/check_coverage_ratchet.sh — ADR 0021 Gap 4 (Phase 5.1 strict).
#
# Mesure la coverage des fichiers critiques shared_social et FAIL si une
# fonction ratchet-baselinée descend SOUS son seuil.
#
# Baseline versionnée : scripts/coverage_baseline.txt (format: fichier:fonction TAB seuil%).
# Plan original demandait ≥ 90% sur pool.go + shared_social_persister.go.
# Réalité 2026-05-27 : ~80% — ratchet baseline pour ne pas bloquer
# immédiatement le CI, avec objectif progressif 90%.
#
# Usage :
#   ./scripts/check_coverage_ratchet.sh                # mesure + check
#   ./scripts/check_coverage_ratchet.sh --update-baseline  # met à jour la baseline

set -euo pipefail

cd "$(dirname "$0")/.."

UPDATE_BASELINE=0
if [ "${1:-}" = "--update-baseline" ]; then
    UPDATE_BASELINE=1
fi

BASELINE_FILE="scripts/coverage_baseline.txt"
COVERAGE_FILE="coverage_ratchet.out"
GO_API_DIR="apps/go-api"

# Tests qui exercent les fichiers ciblés.
TESTS='TestOpenSharedSocial|TestCheckpointSharedSocial|TestSet.*PersistsAfter|TestToggle.*PersistsAfter|TestMediaService_|TestWALOrphanRepro|TestQuarantineOrphanWAL|TestCommitWithCheckpoint|TestE2E_KillBrutal_WALOrphanRecoveryWorks|TestIsWALReplayFailure|TestRequireSocialPersister|TestErrSocialPersisterNotWired|TestSocialReceiverLabel|TestSentinel|TestNoATTACHOnSocialDB|TestNoUnauthorizedSharedSocialMention|TestMigrationIdempotent'

COVERPKG='./internal/platform/duckdb/...,./internal/persist/...,./internal/platform/dblease/...'

echo "[coverage-ratchet] running tests with coverage..."
cd "$GO_API_DIR"
CGO_ENABLED=1 go test -count=1 \
    -coverprofile="../../$COVERAGE_FILE" \
    -covermode=atomic \
    "-coverpkg=$COVERPKG" \
    "-run=$TESTS" \
    ./internal/platform/duckdb/ \
    ./internal/platform/dblease/ \
    2>&1 | tail -5

cd ../..

# Fichiers critiques surveillés (les fonctions exposées au runtime).
# Format : chemin Go du fichier (sans le préfixe module).
WATCHED_FILES=(
    "internal/platform/duckdb/pool_shared_social_recovery.go"
    "internal/platform/dblease/writer.go"
    "internal/persist/shared_social_persister.go"
)

echo ""
echo "[coverage-ratchet] coverage des fichiers critiques :"
TMP_CURRENT=$(mktemp)
cd "$GO_API_DIR"
for f in "${WATCHED_FILES[@]}"; do
    go tool cover -func="../../$COVERAGE_FILE" 2>/dev/null \
        | grep -E "$f:" \
        | awk '{print $1 "\t" $NF}' \
        | sed 's/%$//' >> "$TMP_CURRENT" || true
done
cd ../..

cat "$TMP_CURRENT"

# Si --update-baseline, écrire le snapshot actuel comme nouveau baseline.
if [ "$UPDATE_BASELINE" -eq 1 ]; then
    cp "$TMP_CURRENT" "$BASELINE_FILE"
    echo ""
    echo "[coverage-ratchet] baseline mise à jour : $BASELINE_FILE"
    rm -f "$TMP_CURRENT" "$COVERAGE_FILE"
    exit 0
fi

# Mode check : compare current vs baseline.
if [ ! -f "$BASELINE_FILE" ]; then
    echo ""
    echo "[coverage-ratchet] baseline absente — la créer avec :"
    echo "  ./scripts/check_coverage_ratchet.sh --update-baseline"
    rm -f "$TMP_CURRENT" "$COVERAGE_FILE"
    exit 0
fi

REGRESSIONS=0
while IFS=$'\t' read -r KEY BASELINE_PCT; do
    [ -z "$KEY" ] && continue
    CURRENT=$(grep -F "$KEY" "$TMP_CURRENT" | head -1 | awk -F'\t' '{print $2}' || true)
    if [ -z "$CURRENT" ]; then
        echo "  [warn] $KEY : absent du snapshot courant (test renommé/supprimé ?)"
        continue
    fi
    # Compare en awk (gérer décimales).
    if awk -v cur="$CURRENT" -v base="$BASELINE_PCT" 'BEGIN { exit !(cur + 0 < base + 0) }'; then
        echo "  [FAIL] $KEY : $CURRENT% < baseline $BASELINE_PCT%"
        REGRESSIONS=$((REGRESSIONS + 1))
    fi
done < "$BASELINE_FILE"

rm -f "$TMP_CURRENT" "$COVERAGE_FILE"

if [ "$REGRESSIONS" -gt 0 ]; then
    echo ""
    echo "[coverage-ratchet] $REGRESSIONS régression(s) de coverage détectée(s) — fail."
    echo "  Soit (a) ajouter des tests pour rétablir le niveau baseline,"
    echo "  soit (b) si la baisse est volontaire/documentée :"
    echo "        ./scripts/check_coverage_ratchet.sh --update-baseline"
    exit 1
fi

echo ""
echo "[coverage-ratchet] OK — aucune régression vs baseline."
echo "  Objectif long terme : 90% (cf. ADR 0021 Phase 5.1)."
exit 0

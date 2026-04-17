#!/usr/bin/env bash
# coverage_check.sh — Vérifie le ratchet de couverture.
# Usage: bash scripts/coverage_check.sh <profile.out> <baseline.txt>
#
# Règles :
#   - Lit le pourcentage global depuis le profil filtré.
#   - Compare avec la première ligne de coverage_baseline.txt.
#   - Échec si coverage < baseline - 0.1 (tolérance flakiness).
#   - Sur $CI + branche main : met à jour le baseline si coverage > baseline.
#
# Sprint 45 — Phase 10 Consolidation qualité.
set -euo pipefail

PROFILE="${1:?usage: $0 <profile.out> <baseline.txt>}"
BASELINE_FILE="${2:?usage: $0 <profile.out> <baseline.txt>}"

# Extraire le pourcentage global
current=$(go tool cover -func="$PROFILE" | awk '/^total:/ { gsub("%",""); print $3 }')

if [[ -z "$current" ]]; then
  echo "❌ Impossible de lire la couverture depuis $PROFILE"
  exit 1
fi

# Lire le baseline
if [[ ! -f "$BASELINE_FILE" ]]; then
  echo "⚠️  Baseline $BASELINE_FILE introuvable — création avec valeur actuelle $current%"
  echo "$current" > "$BASELINE_FILE"
  echo "✅ Baseline initialisé à ${current}%"
  exit 0
fi

baseline=$(head -1 "$BASELINE_FILE" | tr -d '%' | xargs)

echo "Coverage actuel  : ${current}%"
echo "Coverage baseline: ${baseline}%"

# Vérifier le ratchet (tolérance 0.1%)
awk -v c="$current" -v b="$baseline" 'BEGIN {
  if (c + 0.1 < b) {
    printf "❌ Coverage %.1f%% < baseline %.1f%% (ratchet violation)\n", c, b
    exit 1
  }
  printf "✅ Coverage %.1f%% >= baseline %.1f%%\n", c, b
  exit 0
}'

# Sur CI en main : mettre à jour le baseline si amélioré
if [[ "${CI:-false}" == "true" ]] && [[ "${GITHUB_REF:-}" == "refs/heads/main" ]]; then
  improved=$(awk -v c="$current" -v b="$baseline" 'BEGIN { exit (c <= b) }' && echo "yes" || echo "no")
  if [[ "$improved" == "yes" ]]; then
    echo "$current" > "$BASELINE_FILE"
    echo "📈 Baseline mis à jour : ${baseline}% → ${current}%"
  fi
fi

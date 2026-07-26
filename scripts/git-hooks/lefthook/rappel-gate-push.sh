#!/usr/bin/env bash
# Rappel NON BLOQUANT au pre-push depuis main : penser à `make gate-push`.
#
# Contexte (2026-07-26) : sur 188 jobs CI rouges analysés sur 14 jours, 149
# venaient de 3 gates qui n'ont aucun équivalent local (baseline de tests,
# ratchet golangci-lint, ratchet de couverture) — d'où les « arrivées rouges »
# sur main. `make gate-push` les rejoue, mais il dure ~25 min : le brancher en
# hook BLOQUANT serait contourné (--no-verify) en une semaine. Ce script se
# contente donc d'afficher un rappel, et sort TOUJOURS 0.
#
# Le rappel ne s'affiche que depuis main, seule branche dont le push déclenche le
# déploiement prod (workflow projet : on merge sur main en local puis on pousse).

set -euo pipefail

branch=$(git branch --show-current 2>/dev/null || echo "")

if [ "$branch" = "main" ]; then
  echo ""
  echo "[rappel] Push depuis main = DÉPLOIEMENT PROD automatique."
  echo "[rappel] Les 3 gates CI les plus bruyants n'ont pas été joués par ce hook :"
  echo "[rappel]   baseline de tests, ratchet golangci-lint, ratchet de couverture."
  echo "[rappel] Filet local (~25 min) : make gate-push"
  echo ""
fi

exit 0

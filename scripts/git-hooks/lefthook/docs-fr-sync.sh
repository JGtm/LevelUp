#!/usr/bin/env bash
# docs-fr-sync.sh — WARNING (non bloquant) : liste les guides majeurs bilingues
# (docs/X.md <-> docs/FR/X.md) modifies d'UN seul cote dans le commit stage.
#
# Politique CLAUDE.md regle 15 : toute modif EN d'un guide majeur inclut la MAJ FR
# (et inversement). Les ADRs et runbooks sont EN-only (hors scope). Ce hook ne
# BLOQUE PAS le commit -- il rappelle la paire a synchroniser (C8, DEC-5).
set -euo pipefail

# Guides majeurs bilingues (cf. CLAUDE.md regle 15).
guides="FOUNDATIONS_GUIDE COMMANDS SYNC_GUIDE ARCHITECTURE_V6"

staged="$(git diff --cached --name-only)"

warned=0
for g in $guides; do
  en="docs/${g}.md"
  fr="docs/FR/${g}.md"
  en_staged=0
  fr_staged=0
  if printf '%s\n' "$staged" | grep -qxF "$en"; then en_staged=1; fi
  if printf '%s\n' "$staged" | grep -qxF "$fr"; then fr_staged=1; fi
  if [ "$en_staged" != "$fr_staged" ]; then
    if [ "$warned" = 0 ]; then
      echo "[docs-fr-sync] guides bilingues desynchronises (CLAUDE.md regle 15, warning non bloquant) :"
      warned=1
    fi
    if [ "$en_staged" = 1 ]; then
      echo "  - ${en} modifie SANS ${fr}"
    else
      echo "  - ${fr} modifie SANS ${en}"
    fi
  fi
done

if [ "$warned" = 1 ]; then
  echo "  -> synchroniser la paire dans ce commit (ou justifier si volontaire)."
fi

exit 0

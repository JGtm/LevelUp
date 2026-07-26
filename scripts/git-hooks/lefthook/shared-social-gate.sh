#!/usr/bin/env bash
# Gate ADR 0021 (invariants shared_social) en pre-push.
#
# Pourquoi ce script existe (2026-07-26) : le gate n'était branché dans AUCUN hook
# local. Il vivait dans scripts/git-hooks/pre-commit, installé par
# `make install-git-hooks` — mais cette cible copiait le hook PAR-DESSUS le shim
# lefthook, et depuis le passage à lefthook le hook historique a été renommé en
# .git/hooks/pre-commit.old : il ne s'exécutait plus. Résultat : les invariants
# anti-corruption ART n'étaient vérifiés qu'en CI, après le push.
#
# Stage pre-push (et non pre-commit) : le gate dure ~2 min (tests -race,
# sous-process kill-brutal) — trop lent pour chaque commit, conforme à la
# doctrine de lefthook.yml (« pre-push = checks coûteux »).
#
# Commande : make go-api-test-shared-social-gate — MÊME intention que le job CI
# shared-social-gate.yml. Delta assumé (la CI reste le gate d'autorité) : la CI
# ajoute ./internal/platform/dblease/, les tests de cmd/rebuild_shared_social/ et
# le ratchet de couverture (scripts/check_coverage_ratchet.sh).
#
# Bypass exceptionnel : LEFTHOOK=0 git push (à documenter dans le message de commit).

set -euo pipefail

# CGO obligatoire (DuckDB). Sur Windows, gcc vient de msys64/ucrt64 et n'est pas
# toujours dans le PATH du shell git.
if ! command -v gcc >/dev/null 2>&1 && [ -d /c/msys64/ucrt64/bin ]; then
  export PATH="/c/msys64/ucrt64/bin:$PATH"
fi

if ! command -v gcc >/dev/null 2>&1; then
  echo "[shared-social-gate] gcc introuvable — CGO/DuckDB impossible en local."
  echo "[shared-social-gate] Gate NON joué ici ; le job CI shared-social-gate.yml fait foi."
  exit 0
fi

exec make go-api-test-shared-social-gate

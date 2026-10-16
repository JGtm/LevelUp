#!/usr/bin/env bash
# Scan des vulnérabilités Go (govulncheck), INDICATIF : le rapport s'affiche mais ce
# hook ne bloque jamais (exit 0, y compris si govulncheck échoue à charger les paquets).
# L'autorité est le job `govulncheck` de .github/workflows/ci.yml, bloquant
# (garde-rail : archlint/ci_workflows_security_test.go). Un rapport rouge ici se corrige
# en montant la dépendance fautive ou la version `go` de apps/go-api/go.mod.
set -euo pipefail

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "[skip] govulncheck absent du PATH (voir CI)"
  exit 0
fi

# DuckDB nécessite CGO + gcc (toolchain msys64 sur Windows). Sans eux
# govulncheck ne peut pas charger les packages CGO → erreur de build.
export CGO_ENABLED=1
if [[ -d "/c/msys64/ucrt64/bin" ]]; then
  export PATH="/c/msys64/ucrt64/bin:$PATH"
elif [[ -d "C:/msys64/ucrt64/bin" ]]; then
  export PATH="C:/msys64/ucrt64/bin:$PATH"
fi

cd apps/go-api || exit 0
# || true : govulncheck exit 3 = "vulns affecting code". Le hook local reste indicatif
# (le scan dépend d'une base distante et d'un toolchain CGO) ; c'est la CI qui refuse.
govulncheck ./... || true

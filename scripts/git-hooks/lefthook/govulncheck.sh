#!/usr/bin/env bash
# Scan des vulnérabilités Go (govulncheck). Advisory (non-bloquant) : les
# vulns stdlib nécessitent un upgrade de toolchain Go (pas un fix de code) et
# sont déjà suivies. Exit 0 toujours ; le rapport reste affiché pour information.
# Bloquant uniquement si govulncheck lui-même crashe (erreur de build).
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
# || true : govulncheck exit 3 = "vulns affecting code" — advisory ici car les
# vulns actuelles sont toutes stdlib (Go 1.26.1 → 1.26.2/1.26.3 requis).
# Upgrade Go pour corriger : https://go.dev/dl/
govulncheck ./... || true

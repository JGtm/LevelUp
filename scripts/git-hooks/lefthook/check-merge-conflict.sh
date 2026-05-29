#!/usr/bin/env bash
# Détecte les marqueurs de conflit Git non résolus dans les fichiers stagés.
set -euo pipefail

if git diff --cached | grep -qE "^\+(<<<<<<<|>>>>>>>)"; then
  echo "Conflit Git non resolu detecte dans les fichiers stages."
  exit 1
fi
exit 0

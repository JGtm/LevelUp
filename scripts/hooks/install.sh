#!/usr/bin/env bash
# Installation des hooks Git versionnés depuis scripts/hooks/.
# Usage : bash scripts/hooks/install.sh

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
HOOKS_SRC="${REPO_ROOT}/scripts/hooks"
HOOKS_DST="${REPO_ROOT}/.git/hooks"

echo "Installation des hooks Git depuis ${HOOKS_SRC}..."

for hook in "${HOOKS_SRC}"/*; do
    name="$(basename "$hook")"
    [[ "$name" == "install.sh" ]] && continue
    cp "$hook" "${HOOKS_DST}/${name}"
    chmod +x "${HOOKS_DST}/${name}"
    echo "  ✅ ${name} installé dans .git/hooks/"
done

echo ""
echo "Hooks installés. Pour vérifier : ls -la .git/hooks/"
echo "Pour tester manuellement : bash scripts/test_deploy_dry_run.sh"

#!/bin/bash
# Wrapper pour exécuter le launcher avec le bon Python Windows
# Usage: ./run.sh [args...]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_PYTHON="$SCRIPT_DIR/.venv/Scripts/python.exe"

if [ ! -f "$VENV_PYTHON" ]; then
    echo "Erreur: .venv non trouve"
    echo "Active d'abord l'environnement: source activate_env.sh"
    exit 1
fi

exec "$VENV_PYTHON" "$SCRIPT_DIR/launcher.py" "$@"

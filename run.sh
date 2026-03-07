#!/bin/bash
# Wrapper pour exécuter le launcher avec le bon Python Windows
# Usage: ./run.sh [args...]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_PYTHON="$SCRIPT_DIR/.venv/Scripts/python.exe"

if [ ! -f "$VENV_PYTHON" ]; then
    echo "Erreur: .venv non trouve"
    echo "Lance d'abord: python launcher.py setup"
    exit 1
fi

exec "$VENV_PYTHON" "$SCRIPT_DIR/launcher.py" "$@"

#!/bin/sh
# Wrapper léger pour lancer LevelUp depuis un terminal Unix (macOS / Linux / Git Bash).
# Usage: ./run.sh [args...]
#
# Sur macOS/Linux : utilise .venv/bin/python
# Sur Windows Git Bash : utilise .venv/Scripts/python.exe si disponible, sinon bin/python

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Choisir le bon chemin Python selon l'OS
if [ -f "$SCRIPT_DIR/.venv/Scripts/python.exe" ]; then
    # Windows (Git Bash / MSYS2)
    VENV_PYTHON="$SCRIPT_DIR/.venv/Scripts/python.exe"
elif [ -f "$SCRIPT_DIR/.venv/bin/python" ]; then
    # macOS / Linux
    VENV_PYTHON="$SCRIPT_DIR/.venv/bin/python"
else
    echo "Erreur: .venv non trouvé."
    echo "Premier lancement :"
    echo "  macOS/Linux → ./LevelUp.sh"
    echo "  Windows     → double-cliquez sur LevelUp.bat"
    exit 1
fi

exec "$VENV_PYTHON" "$SCRIPT_DIR/launcher.py" "$@"

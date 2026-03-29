#!/bin/sh
# Wrapper léger pour lancer LevelUp depuis un terminal Unix (macOS / Linux / Git Bash).
# Usage: ./run.sh [args...]
#
# Sur macOS/Linux : utilise .venv/bin/python
# Sur Windows Git Bash : utilise .venv/Scripts/python.exe
#
# Note : MSYS2/MinGW Python natif (pacman) est interdit — seul le .venv Windows
# natif est supporté. Ce script utilise le .venv directement, donc pas de risque
# si run.sh est exécuté depuis Git Bash (MINGW64/MINGW32).

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Git Bash sur Windows crée un fichier "nul" si une commande Windows (tasklist,
# chcp, where…) utilise ">nul" depuis ce shell — supprimer silencieusement.
if [ -f "$SCRIPT_DIR/nul" ]; then
    rm -f "$SCRIPT_DIR/nul"
fi

# S'assurer que LevelUp.sh est exécutable (peut arriver après extraction d'un zip)
chmod +x "$SCRIPT_DIR/LevelUp.sh" 2>/dev/null || true

# ── Si le venv n'existe pas → déléguer au setup automatique (LevelUp.sh) ──────
if [ ! -f "$SCRIPT_DIR/.venv/Scripts/python.exe" ] && [ ! -f "$SCRIPT_DIR/.venv/bin/python" ]; then
    echo "  Venv absent — démarrage du setup automatique..."
    exec "$SCRIPT_DIR/LevelUp.sh" "$@"
fi

# ── Choisir le bon chemin Python selon l'OS ────────────────────────────────────
if [ -f "$SCRIPT_DIR/.venv/Scripts/python.exe" ]; then
    # Windows (Git Bash)
    VENV_PYTHON="$SCRIPT_DIR/.venv/Scripts/python.exe"
elif [ -f "$SCRIPT_DIR/.venv/bin/python" ]; then
    # macOS / Linux
    VENV_PYTHON="$SCRIPT_DIR/.venv/bin/python"
fi

# ── Sanity check : interpréteur vivant et imports critiques présents ───────────
if ! "$VENV_PYTHON" -c "import sys; sys.exit(0)" 2>/dev/null; then
    echo "  ⚠  Interpréteur du venv inaccessible (Python désinstallé ?)."
    echo "     Relancez le setup : ./LevelUp.sh --reinstall"
    exit 1
fi

if ! "$VENV_PYTHON" -c "import streamlit, duckdb, polars" 2>/dev/null; then
    echo "  ⚠  Environnement incomplet ou corrompu."
    echo "     Relancez le setup : ./LevelUp.sh --reinstall"
    exit 1
fi

exec "$VENV_PYTHON" "$SCRIPT_DIR/launcher.py" "$@"

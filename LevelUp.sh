#!/bin/sh
# =============================================================================
# LevelUp - Lanceur macOS / Linux
#
# Double-cliquez ou lancez : ./LevelUp.sh
# Au premier lancement, l'environnement est configuré automatiquement.
#
# Écrit en POSIX sh (pas bash) pour fonctionner sur macOS (bash 3.2),
# Ubuntu/Debian (dash), et toute distribution Linux moderne.
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR" || exit 1

VENV_PY="$SCRIPT_DIR/.venv/bin/python"

# ── Venv déjà présent → on lance directement ──────────────────────────────────
if [ -f "$VENV_PY" ]; then
    exec "$VENV_PY" "$SCRIPT_DIR/launcher.py" run
fi

# ── Premier lancement ─────────────────────────────────────────────────────────
echo ""
echo "  ╔══════════════════════════════════════════╗"
echo "  ║     LevelUp - Premier lancement          ║"
echo "  ╚══════════════════════════════════════════╝"
echo ""

# ── Trouver Python 3.10+ ──────────────────────────────────────────────────────
find_python() {
    # Binaires versionnnés en priorité (Homebrew macOS, apt Linux)
    for minor in 12 13 11 10; do
        candidate="python3.$minor"
        if command -v "$candidate" > /dev/null 2>&1; then
            echo "$candidate"
            return 0
        fi
    done

    # Homebrew macOS — chemins explicites (Intel + Apple Silicon)
    for prefix in /opt/homebrew /usr/local; do
        for minor in 12 13 11 10; do
            candidate="$prefix/bin/python3.$minor"
            if [ -x "$candidate" ]; then
                echo "$candidate"
                return 0
            fi
        done
    done

    # Générique python3 / python
    for name in python3 python; do
        if command -v "$name" > /dev/null 2>&1; then
            ver=$("$name" -c "import sys; print(sys.version_info.minor)" 2>/dev/null)
            if [ -n "$ver" ] && [ "$ver" -ge 10 ] 2>/dev/null; then
                echo "$name"
                return 0
            fi
        fi
    done

    return 1
}

PY=$(find_python)

if [ -z "$PY" ]; then
    echo "  ❌ Python 3.10+ introuvable sur ce système."
    echo ""
    OS=$(uname -s)
    case "$OS" in
        Darwin)
            echo "  → macOS — installez Python avec Homebrew :"
            echo "       brew install python@3.12"
            echo "  → Ou téléchargez depuis https://www.python.org/downloads/"
            ;;
        Linux)
            if command -v apt-get > /dev/null 2>&1; then
                echo "  → Ubuntu/Debian :"
                echo "       sudo apt-get install python3.12 python3.12-venv"
            elif command -v dnf > /dev/null 2>&1; then
                echo "  → Fedora/RHEL :"
                echo "       sudo dnf install python3.12"
            elif command -v pacman > /dev/null 2>&1; then
                echo "  → Arch Linux :"
                echo "       sudo pacman -S python"
            else
                echo "  → Téléchargez depuis https://www.python.org/downloads/"
            fi
            ;;
        *)
            echo "  → Téléchargez depuis https://www.python.org/downloads/"
            ;;
    esac
    echo ""
    read -r _dummy
    exit 1
fi

# Vérifier que python3-venv est disponible (Linux peut nécessiter un paquet séparé)
if ! "$PY" -m venv --help > /dev/null 2>&1; then
    echo "  ❌ Le module 'venv' est absent de $PY."
    echo ""
    if command -v apt-get > /dev/null 2>&1; then
        VER=$("$PY" -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')")
        echo "  → Ubuntu/Debian : sudo apt-get install python${VER}-venv"
    fi
    echo "  → Ou installez Python depuis https://www.python.org/downloads/"
    echo ""
    read -r _dummy
    exit 1
fi

echo "  Python : $PY ($("$PY" --version 2>&1))"
echo ""

# ── Créer le venv ─────────────────────────────────────────────────────────────
echo "  Création de l'environnement virtuel..."
"$PY" -m venv "$SCRIPT_DIR/.venv"
if [ $? -ne 0 ]; then
    echo "  ❌ Impossible de créer le venv."
    read -r _dummy
    exit 1
fi

# ── Installer les dépendances ─────────────────────────────────────────────────
echo "  Installation des dépendances (peut prendre quelques minutes)..."
"$VENV_PY" -m pip install --upgrade pip -q
"$VENV_PY" -m pip install -e ".[spnkr]" -q
if [ $? -ne 0 ]; then
    echo "  ❌ L'installation des dépendances a échoué."
    read -r _dummy
    exit 1
fi

echo "  ✓ Environnement prêt."
echo ""

# ── Lancer le dashboard ───────────────────────────────────────────────────────
exec "$VENV_PY" "$SCRIPT_DIR/launcher.py" run

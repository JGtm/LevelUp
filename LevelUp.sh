#!/bin/sh
# =============================================================================
# LevelUp - Lanceur macOS / Linux
#
# Usage : ./LevelUp.sh [options] [args launcher...]
#
#   --reinstall   Force la recréation du venv (.venv supprimé et recréé)
#   --no-spnkr    Install légère sans les dépendances API Halo (spnkr)
#   --offline     Interdit l'accès PyPI (nécessite un cache pip local)
#
# Compatible : macOS (bash 3.2+, zsh), Ubuntu/Debian (dash), Arch, Fedora,
#              WSL2 (avertissement si le projet se trouve sur /mnt/...).
# Écrit en POSIX sh strict — pas de bashismes.
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR" || exit 1

VENV_PY="$SCRIPT_DIR/.venv/bin/python"
LOG_DIR="$SCRIPT_DIR/data/logs"
INSTALL_LOG="$LOG_DIR/install.log"
PYPROJECT="$SCRIPT_DIR/pyproject.toml"
PYPROJECT_HASH_FILE="$SCRIPT_DIR/.venv/.pyproject_hash"

# ── Parser les options LevelUp.sh (les autres sont transmis à launcher.py) ───
OPT_REINSTALL=0
OPT_NO_SPNKR=0
OPT_OFFLINE=0
PASS_ARGS=""

for _arg in "$@"; do
    case "$_arg" in
        --reinstall) OPT_REINSTALL=1 ;;
        --no-spnkr)  OPT_NO_SPNKR=1 ;;
        --offline)   OPT_OFFLINE=1 ;;
        *)           PASS_ARGS="${PASS_ARGS:+$PASS_ARGS }$_arg" ;;
    esac
done

# ── Préparer le dossier de logs ───────────────────────────────────────────────
mkdir -p "$LOG_DIR" 2>/dev/null || true

# ── Avertissement WSL2 avec projet sur filesystem Windows ─────────────────────
if [ -f /proc/version ] && grep -qi "microsoft" /proc/version 2>/dev/null; then
    case "$SCRIPT_DIR" in
        /mnt/*)
            echo "  ⚠  WSL2 : projet sur un chemin Windows ($SCRIPT_DIR)."
            echo "     Les performances I/O seront dégradées."
            echo "     Recommandé : déplacez le projet dans ~/LevelUp (ext4)."
            echo ""
            ;;
    esac
fi

# ── Flag --reinstall ───────────────────────────────────────────────────────────
if [ "$OPT_REINSTALL" = "1" ] && [ -d "$SCRIPT_DIR/.venv" ]; then
    echo "  🔄 Suppression du venv (--reinstall)..."
    rm -rf "$SCRIPT_DIR/.venv"
fi

# ── Venv déjà présent → validation et lancement ────────────────────────────────
if [ -f "$VENV_PY" ]; then
    # 1. Interpréteur vivant ? (peut pointer vers un Python désinstallé)
    if ! "$VENV_PY" -c "import sys; sys.exit(0)" 2>/dev/null; then
        echo "  ⚠  Interpréteur du venv inaccessible (Python désinstallé ?), recréation..."
        rm -rf "$SCRIPT_DIR/.venv"
    # 2. Imports critiques présents ?
    elif ! "$VENV_PY" -c "import streamlit, duckdb, polars" 2>/dev/null; then
        echo "  ⚠  Environnement incomplet détecté, réinstallation..."
        rm -rf "$SCRIPT_DIR/.venv"
    else
        # 3. pyproject.toml modifié → mettre à jour les dépendances
        _current_hash=""
        if command -v md5sum > /dev/null 2>&1; then
            _current_hash=$(md5sum "$PYPROJECT" 2>/dev/null | awk '{print $1}')
        elif command -v md5 > /dev/null 2>&1; then
            _current_hash=$(md5 -q "$PYPROJECT" 2>/dev/null)
        fi
        _stored_hash=""
        [ -f "$PYPROJECT_HASH_FILE" ] && _stored_hash=$(cat "$PYPROJECT_HASH_FILE" 2>/dev/null)

        if [ -n "$_current_hash" ] && [ "$_current_hash" != "$_stored_hash" ]; then
            echo "  🔄 pyproject.toml modifié — mise à jour des dépendances..."
            _install_extra=".[spnkr]"
            [ "$OPT_NO_SPNKR" = "1" ] && _install_extra="."
            _pip_opts="--disable-pip-version-check"
            [ "$OPT_OFFLINE" = "1" ] && _pip_opts="$_pip_opts --no-index"
            # shellcheck disable=SC2086
            if "$VENV_PY" -m pip install -e "$_install_extra" $_pip_opts >> "$INSTALL_LOG" 2>&1; then
                echo "$_current_hash" > "$PYPROJECT_HASH_FILE"
                echo "  ✓ Dépendances à jour."
            else
                echo "  ⚠  Mise à jour partielle (détails : $INSTALL_LOG)"
            fi
        fi

        # shellcheck disable=SC2086
        exec "$VENV_PY" "$SCRIPT_DIR/launcher.py" $PASS_ARGS
    fi
fi

# ── Setup : premier lancement ou venv recréé ──────────────────────────────────
echo ""
echo "  ╔══════════════════════════════════════════╗"
echo "  ║     LevelUp - Premier lancement          ║"
echo "  ╚══════════════════════════════════════════╝"
echo ""

# ── Trouver Python 3.10–3.13 ──────────────────────────────────────────────────
find_python() {
    # Binaires versionnés — du plus récent au plus ancien (13→10)
    for _minor in 13 12 11 10; do
        _candidate="python3.$_minor"
        if command -v "$_candidate" > /dev/null 2>&1; then
            echo "$_candidate"
            return 0
        fi
    done

    # Homebrew macOS — chemins explicites (Intel /usr/local + Apple Silicon /opt/homebrew)
    for _prefix in /opt/homebrew /usr/local; do
        for _minor in 13 12 11 10; do
            _candidate="$_prefix/bin/python3.$_minor"
            if [ -x "$_candidate" ]; then
                echo "$_candidate"
                return 0
            fi
        done
    done

    # Générique python3 / python — vérifier que c'est bien >= 3.10
    for _name in python3 python; do
        if command -v "$_name" > /dev/null 2>&1; then
            _ver=$("$_name" -c "import sys; print(sys.version_info.minor)" 2>/dev/null)
            _maj=$("$_name" -c "import sys; print(sys.version_info.major)" 2>/dev/null)
            if [ "$_maj" = "3" ] && [ -n "$_ver" ] && [ "$_ver" -ge 10 ]; then
                echo "$_name"
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
    case "$(uname -s)" in
        Darwin)
            echo "  → macOS : brew install python@3.12"
            echo "  → Ou    : https://www.python.org/downloads/"
            ;;
        Linux)
            if   command -v apt-get > /dev/null 2>&1; then
                echo "  → Ubuntu/Debian : sudo apt-get install python3.12 python3.12-venv"
            elif command -v dnf     > /dev/null 2>&1; then
                echo "  → Fedora/RHEL   : sudo dnf install python3.12"
            elif command -v pacman  > /dev/null 2>&1; then
                echo "  → Arch Linux    : sudo pacman -S python"
            else
                echo "  → https://www.python.org/downloads/"
            fi
            ;;
        *) echo "  → https://www.python.org/downloads/" ;;
    esac
    echo ""
    read -r _dummy
    exit 1
fi

# Vérifier que le module venv est disponible (Linux peut nécessiter un paquet séparé)
if ! "$PY" -m venv --help > /dev/null 2>&1; then
    echo "  ❌ Le module 'venv' est absent de $PY."
    echo ""
    if command -v apt-get > /dev/null 2>&1; then
        _ver=$("$PY" -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')")
        echo "  → sudo apt-get install python${_ver}-venv"
    fi
    echo "  → Ou : https://www.python.org/downloads/"
    echo ""
    read -r _dummy
    exit 1
fi

echo "  Python : $PY  ($("$PY" --version 2>&1))"
echo ""

# macOS : lever la quarantaine Gatekeeper (fréquent après extraction d'un .zip)
if [ "$(uname -s)" = "Darwin" ] && command -v xattr > /dev/null 2>&1; then
    xattr -cr "$SCRIPT_DIR" 2>/dev/null || true
fi

# ── Créer le venv ─────────────────────────────────────────────────────────────
echo "  Création de l'environnement virtuel..."
{
    echo "=== $(date) - Création venv avec $PY ==="
    "$PY" -m venv "$SCRIPT_DIR/.venv"
} >> "$INSTALL_LOG" 2>&1

if [ ! -f "$VENV_PY" ]; then
    echo "  ❌ Impossible de créer le venv. Détails : $INSTALL_LOG"
    read -r _dummy
    exit 1
fi

# ── Mettre à jour pip ─────────────────────────────────────────────────────────
echo "  Mise à jour de pip..."
if ! "$VENV_PY" -m pip install --upgrade pip --disable-pip-version-check -q >> "$INSTALL_LOG" 2>&1; then
    echo "  ⚠  pip non mis à jour (réseau ou proxy). Poursuite avec la version installée."
fi

# ── Installer les dépendances ─────────────────────────────────────────────────
_install_extra=".[spnkr]"
[ "$OPT_NO_SPNKR" = "1" ] && _install_extra="."
_pip_opts="--disable-pip-version-check"
[ "$OPT_OFFLINE" = "1" ] && _pip_opts="$_pip_opts --no-index"

echo "  Installation des dépendances (quelques minutes à la première exécution)..."
# shellcheck disable=SC2086
if ! "$VENV_PY" -m pip install -e "$_install_extra" $_pip_opts >> "$INSTALL_LOG" 2>&1; then
    echo ""
    echo "  ❌ Échec de l'installation. Causes possibles :"
    echo "     - Pas de connexion internet"
    echo "     - Dossier en lecture seule (déplacez LevelUp dans ~/Documents)"
    echo "     - Espace disque insuffisant (df -h)"
    echo "     Détails : $INSTALL_LOG"
    read -r _dummy
    exit 1
fi

# Enregistrer le fingerprint de pyproject.toml pour les prochains lancements
_hash=""
if command -v md5sum > /dev/null 2>&1; then
    _hash=$(md5sum "$PYPROJECT" 2>/dev/null | awk '{print $1}')
elif command -v md5 > /dev/null 2>&1; then
    _hash=$(md5 -q "$PYPROJECT" 2>/dev/null)
fi
[ -n "$_hash" ] && echo "$_hash" > "$PYPROJECT_HASH_FILE"

echo "  ✓ Environnement prêt."
echo ""

# shellcheck disable=SC2086
exec "$VENV_PY" "$SCRIPT_DIR/launcher.py" $PASS_ARGS

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

# ── Détection langue système ─────────────────────────────────────────────────
_locale="${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}"
_lang_code=$(echo "$_locale" | cut -c1-2 | tr '[:upper:]' '[:lower:]')
case "$_lang_code" in
    [a-z][a-z]) : ;;
    *)           _lang_code="" ;;
esac
case "$_lang_code" in
    fr) SCRIPT_LANG="fr" ;;
    *)  SCRIPT_LANG="en" ;;
esac

# ── Messages localisés ───────────────────────────────────────────────────────
if [ "$SCRIPT_LANG" = "fr" ]; then
    MSG_WSL_WARNING="  ⚠  WSL2 : projet sur un chemin Windows"
    MSG_WSL_PERF="     Les performances I/O seront dégradées."
    MSG_WSL_RECOMMEND="     Recommandé : déplacez le projet dans ~/LevelUp (ext4)."
    MSG_REINSTALL="  🔄 Suppression du venv (--reinstall)..."
    MSG_VENV_DEAD="  ⚠  Interpréteur du venv inaccessible (Python désinstallé ?), recréation..."
    MSG_VENV_INCOMPLETE="  ⚠  Environnement incomplet détecté, réinstallation..."
    MSG_DEPS_CHANGED="  🔄 pyproject.toml modifié — mise à jour des dépendances..."
    MSG_DEPS_OK="  ✓ Dépendances à jour."
    MSG_DEPS_PARTIAL="  ⚠  Mise à jour partielle"
    MSG_FIRST_LAUNCH_TITLE="     LevelUp - Premier lancement"
    MSG_PYTHON_NOT_FOUND="  ❌ Python 3.10+ introuvable sur ce système."
    MSG_VENV_MODULE_MISSING="  ❌ Le module 'venv' est absent de"
    MSG_CREATING_VENV="  Création de l'environnement virtuel..."
    MSG_VENV_FAIL="  ❌ Impossible de créer le venv."
    MSG_PIP_UPDATE="  Mise à jour de pip..."
    MSG_PIP_WARN="  ⚠  pip non mis à jour (réseau ou proxy). Poursuite avec la version installée."
    MSG_INSTALLING="  Installation des dépendances (quelques minutes à la première exécution)..."
    MSG_INSTALL_FAIL="  ❌ Échec de l'installation. Causes possibles :"
    MSG_INSTALL_FAIL_NETWORK="     - Pas de connexion internet"
    MSG_INSTALL_FAIL_READONLY="     - Dossier en lecture seule (déplacez LevelUp dans ~/Documents)"
    MSG_INSTALL_FAIL_DISK="     - Espace disque insuffisant (df -h)"
    MSG_READY="  ✓ Environnement prêt."
    MSG_PYTHON_LABEL="  Python :"
    MSG_INSTALL_SUGGEST_MAC="  → macOS : brew install python@3.12"
    MSG_INSTALL_SUGGEST_MAC_ALT="  → Ou    : https://www.python.org/downloads/"
    MSG_INSTALL_SUGGEST_DEB="  → Ubuntu/Debian : sudo apt-get install python3.12 python3.12-venv"
    MSG_INSTALL_SUGGEST_FED="  → Fedora/RHEL   : sudo dnf install python3.12"
    MSG_INSTALL_SUGGEST_ARCH="  → Arch Linux    : sudo pacman -S python"
    MSG_INSTALL_SUGGEST_OTHER="  → https://www.python.org/downloads/"
    MSG_INSTALL_SUGGEST_VENV="  → sudo apt-get install python"
else
    MSG_WSL_WARNING="  ⚠  WSL2: project on a Windows path"
    MSG_WSL_PERF="     I/O performance will be degraded."
    MSG_WSL_RECOMMEND="     Recommended: move the project to ~/LevelUp (ext4)."
    MSG_REINSTALL="  🔄 Removing venv (--reinstall)..."
    MSG_VENV_DEAD="  ⚠  Venv interpreter inaccessible (Python uninstalled?), recreating..."
    MSG_VENV_INCOMPLETE="  ⚠  Incomplete environment detected, reinstalling..."
    MSG_DEPS_CHANGED="  🔄 pyproject.toml changed — updating dependencies..."
    MSG_DEPS_OK="  ✓ Dependencies up to date."
    MSG_DEPS_PARTIAL="  ⚠  Partial update"
    MSG_FIRST_LAUNCH_TITLE="     LevelUp - First launch"
    MSG_PYTHON_NOT_FOUND="  ❌ Python 3.10+ not found on this system."
    MSG_VENV_MODULE_MISSING="  ❌ The 'venv' module is missing from"
    MSG_CREATING_VENV="  Creating virtual environment..."
    MSG_VENV_FAIL="  ❌ Unable to create venv."
    MSG_PIP_UPDATE="  Updating pip..."
    MSG_PIP_WARN="  ⚠  pip not updated (network or proxy). Continuing with installed version."
    MSG_INSTALLING="  Installing dependencies (this may take a few minutes on first run)..."
    MSG_INSTALL_FAIL="  ❌ Installation failed. Possible causes:"
    MSG_INSTALL_FAIL_NETWORK="     - No internet connection"
    MSG_INSTALL_FAIL_READONLY="     - Read-only folder (move LevelUp to ~/Documents)"
    MSG_INSTALL_FAIL_DISK="     - Insufficient disk space (df -h)"
    MSG_READY="  ✓ Environment ready."
    MSG_PYTHON_LABEL="  Python:"
    MSG_INSTALL_SUGGEST_MAC="  → macOS: brew install python@3.12"
    MSG_INSTALL_SUGGEST_MAC_ALT="  → Or   : https://www.python.org/downloads/"
    MSG_INSTALL_SUGGEST_DEB="  → Ubuntu/Debian: sudo apt-get install python3.12 python3.12-venv"
    MSG_INSTALL_SUGGEST_FED="  → Fedora/RHEL  : sudo dnf install python3.12"
    MSG_INSTALL_SUGGEST_ARCH="  → Arch Linux   : sudo pacman -S python"
    MSG_INSTALL_SUGGEST_OTHER="  → https://www.python.org/downloads/"
    MSG_INSTALL_SUGGEST_VENV="  → sudo apt-get install python"
fi

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
            echo "$MSG_WSL_WARNING ($SCRIPT_DIR)."
            echo "$MSG_WSL_PERF"
            echo "$MSG_WSL_RECOMMEND"
            echo ""
            ;;
    esac
fi

# ── Flag --reinstall ───────────────────────────────────────────────────────────
if [ "$OPT_REINSTALL" = "1" ] && [ -d "$SCRIPT_DIR/.venv" ]; then
    echo "$MSG_REINSTALL"
    rm -rf "$SCRIPT_DIR/.venv"
fi

# ── Venv déjà présent → validation et lancement ────────────────────────────────
if [ -f "$VENV_PY" ]; then
    # 1. Interpréteur vivant ? (peut pointer vers un Python désinstallé)
    if ! "$VENV_PY" -c "import sys; sys.exit(0)" 2>/dev/null; then
        echo "$MSG_VENV_DEAD"
        rm -rf "$SCRIPT_DIR/.venv"
    # 2. Imports critiques présents ?
    elif ! "$VENV_PY" -c "import streamlit, duckdb, polars" 2>/dev/null; then
        echo "$MSG_VENV_INCOMPLETE"
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
            echo "$MSG_DEPS_CHANGED"
            _install_extra=".[spnkr]"
            [ "$OPT_NO_SPNKR" = "1" ] && _install_extra="."
            _pip_opts="--disable-pip-version-check"
            [ "$OPT_OFFLINE" = "1" ] && _pip_opts="$_pip_opts --no-index"
            # shellcheck disable=SC2086
            if "$VENV_PY" -m pip install -e "$_install_extra" $_pip_opts >> "$INSTALL_LOG" 2>&1; then
                echo "$_current_hash" > "$PYPROJECT_HASH_FILE"
                echo "$MSG_DEPS_OK"
            else
                echo "$MSG_DEPS_PARTIAL ($INSTALL_LOG)"
            fi
        fi

        # shellcheck disable=SC2086
        exec "$VENV_PY" "$SCRIPT_DIR/launcher.py" $PASS_ARGS
    fi
fi

# ── Setup : premier lancement ou venv recréé ──────────────────────────────────
echo ""
echo "  ╔══════════════════════════════════════════╗"
echo "  ║     $MSG_FIRST_LAUNCH_TITLE          ║"
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
    echo "$MSG_PYTHON_NOT_FOUND"
    echo ""
    case "$(uname -s)" in
        Darwin)
            echo "$MSG_INSTALL_SUGGEST_MAC"
            echo "$MSG_INSTALL_SUGGEST_MAC_ALT"
            ;;
        Linux)
            if   command -v apt-get > /dev/null 2>&1; then
                echo "$MSG_INSTALL_SUGGEST_DEB"
            elif command -v dnf     > /dev/null 2>&1; then
                echo "$MSG_INSTALL_SUGGEST_FED"
            elif command -v pacman  > /dev/null 2>&1; then
                echo "$MSG_INSTALL_SUGGEST_ARCH"
            else
                echo "$MSG_INSTALL_SUGGEST_OTHER"
            fi
            ;;
        *) echo "$MSG_INSTALL_SUGGEST_OTHER" ;;
    esac
    echo ""
    read -r _dummy
    exit 1
fi

# Vérifier que le module venv est disponible (Linux peut nécessiter un paquet séparé)
if ! "$PY" -m venv --help > /dev/null 2>&1; then
    echo "$MSG_VENV_MODULE_MISSING $PY."
    echo ""
    if command -v apt-get > /dev/null 2>&1; then
        _ver=$("$PY" -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')")
        echo "$MSG_INSTALL_SUGGEST_VENV${_ver}-venv"
    fi
    echo "$MSG_INSTALL_SUGGEST_OTHER"
    echo ""
    read -r _dummy
    exit 1
fi

echo "$MSG_PYTHON_LABEL $PY  ($($PY --version 2>&1))"
echo ""

# macOS : lever la quarantaine Gatekeeper (fréquent après extraction d'un .zip)
if [ "$(uname -s)" = "Darwin" ] && command -v xattr > /dev/null 2>&1; then
    xattr -cr "$SCRIPT_DIR" 2>/dev/null || true
fi

# ── Créer le venv ─────────────────────────────────────────────────────────────
echo "$MSG_CREATING_VENV"
{
    echo "=== $(date) - Création venv avec $PY ==="
    "$PY" -m venv "$SCRIPT_DIR/.venv"
} >> "$INSTALL_LOG" 2>&1

if [ ! -f "$VENV_PY" ]; then
    echo "$MSG_VENV_FAIL Details : $INSTALL_LOG"
    read -r _dummy
    exit 1
fi

# ── Mettre à jour pip ─────────────────────────────────────────────────────────────────
echo "$MSG_PIP_UPDATE"
if ! "$VENV_PY" -m pip install --upgrade pip --disable-pip-version-check -q >> "$INSTALL_LOG" 2>&1; then
    echo "$MSG_PIP_WARN"
fi

# ── Installer les dépendances ─────────────────────────────────────────────────
_install_extra=".[spnkr]"
[ "$OPT_NO_SPNKR" = "1" ] && _install_extra="."
_pip_opts="--disable-pip-version-check"
[ "$OPT_OFFLINE" = "1" ] && _pip_opts="$_pip_opts --no-index"

echo "$MSG_INSTALLING"
# shellcheck disable=SC2086
if ! "$VENV_PY" -m pip install -e "$_install_extra" $_pip_opts >> "$INSTALL_LOG" 2>&1; then
    echo ""
    echo "$MSG_INSTALL_FAIL"
    echo "$MSG_INSTALL_FAIL_NETWORK"
    echo "$MSG_INSTALL_FAIL_READONLY"
    echo "$MSG_INSTALL_FAIL_DISK"
    echo "     Details : $INSTALL_LOG"
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

echo "$MSG_READY"
echo ""

# shellcheck disable=SC2086
exec "$VENV_PY" "$SCRIPT_DIR/launcher.py" $PASS_ARGS

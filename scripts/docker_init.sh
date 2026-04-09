#!/usr/bin/env bash
# docker_init.sh — Prépare l'environnement hôte pour docker compose up.
#
# Usage:
#   bash scripts/docker_init.sh
#
# Ce script crée les fichiers et dossiers requis par les bind mounts
# de docker-compose.yml, pour éviter que Docker ne crée des DOSSIERS
# à la place des fichiers attendus (comportement par défaut si absents).
#
# IMPORTANT (VPS) : lancer ce script en tant qu'utilisateur `deploy` (pas root).
# Si lancé en root, les fichiers seront possédés par root et le container appuser (UID 10001)
# ne pourra pas écrire dans data/. En cas de doute : chown -R deploy:deploy /opt/levelup/data
#
# Idempotent : peut être relancé sans risque.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== Initialisation Docker pour LevelUp ==="

# --- Fichiers de configuration ---

if [ ! -f "$REPO_ROOT/db_profiles.json" ]; then
    echo '{"profiles": {}}' > "$REPO_ROOT/db_profiles.json"
    echo "✓ db_profiles.json créé (vide)"
else
    echo "· db_profiles.json existe déjà"
fi

if [ ! -f "$REPO_ROOT/app_settings.json" ]; then
    echo '{}' > "$REPO_ROOT/app_settings.json"
    echo "✓ app_settings.json créé (vide)"
else
    echo "· app_settings.json existe déjà"
fi

# --- Dossiers de données ---

mkdir -p "$REPO_ROOT/data/players"
mkdir -p "$REPO_ROOT/data/warehouse"
mkdir -p "$REPO_ROOT/data/cache"
mkdir -p "$REPO_ROOT/data/logs"
mkdir -p "$REPO_ROOT/data/demo/players/DEMO"
echo "· Dossiers data/{players,warehouse,cache,logs,demo} OK"

# --- Stubs bind-mount pour levelup-demo ---
# IMPORTANT : Docker crée automatiquement un RÉPERTOIRE quand il rencontre un bind-mount fichier
# dont la source n'existe pas. Ces stubs évitent ce piège ; ils seront remplacés par le
# vrai contenu lors du premier regen (scripts/prepare_demo_data.py).
for demo_file in "$REPO_ROOT/data/demo/db_profiles.json" "$REPO_ROOT/data/demo/app_settings.json"; do
    if [[ ! -e "$demo_file" ]]; then
        echo '{}' > "$demo_file"
        echo "✓ $(basename "$demo_file") demo créé (stub)"
    else
        echo "· $(basename "$demo_file") demo existe déjà"
    fi
done

# --- Fichiers de données optionnels ---

if [ ! -f "$REPO_ROOT/data/xuid_aliases.json" ]; then
    echo '{}' > "$REPO_ROOT/data/xuid_aliases.json"
    echo "✓ data/xuid_aliases.json créé (vide)"
else
    echo "· data/xuid_aliases.json existe déjà"
fi

echo ""
echo "=== Prêt ! Lancez : docker compose up --build ==="

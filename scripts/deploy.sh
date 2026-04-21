#!/usr/bin/env bash
# deploy.sh — Script de mise à jour LevelUp sur VPS Ionos
#
# Usage (depuis le VPS) :
#   /opt/levelup/scripts/deploy.sh
#
# Usage (depuis le poste local) :
#   ssh deploy@212.227.206.42 '/opt/levelup/scripts/deploy.sh'
#
# Prérequis : docker, docker compose, git installés sur le VPS.

set -euo pipefail

DEPLOY_DIR="/opt/levelup"

echo "[deploy] Répertoire : $DEPLOY_DIR"
cd "$DEPLOY_DIR"

# 0. Corriger les permissions git (idempotent — protège contre les écrits root via Docker)
find .git/objects -not -writable -exec chmod u+w {} \; 2>/dev/null || true

# 1. Récupérer les derniers commits depuis main (force, ignore les changements locaux)
echo "[deploy] git fetch + reset --hard origin/main..."
git fetch origin main
git reset --hard origin/main
git clean -fd --exclude=data/ --exclude=.env.local --exclude=app_settings.json --exclude=db_profiles.json

# 2a. Créer les fichiers stub pour les bind-mounts "fichier" de levelup-demo
# sans ça Docker crée automatiquement des RÉPERTOIRES à la place des fichiers attendus,
# ce qui fait planter docker compose up avec "not a directory".
# Les stubs sont remplacés par le vrai contenu lors du regen demo.
mkdir -p "$DEPLOY_DIR/data/demo" "$DEPLOY_DIR/data/logs"
# Si un répertoire fantôme existe à la place d'un fichier (docker compose mount raté),
# on le supprime avant de créer le stub — sinon docker compose up replante.
for _stub in db_profiles.json app_settings.json; do
    _path="$DEPLOY_DIR/data/demo/$_stub"
    [[ -d "$_path" ]] && { rm -rf "$_path"; echo "[deploy] 🧹 Répertoire fantôme supprimé: $_stub"; }
done
[[ -f "$DEPLOY_DIR/data/demo/db_profiles.json" ]] \
    || echo '{"profiles":{}}' > "$DEPLOY_DIR/data/demo/db_profiles.json"
[[ -f "$DEPLOY_DIR/data/demo/app_settings.json" ]] \
    || echo '{}' > "$DEPLOY_DIR/data/demo/app_settings.json"
echo "[deploy] ✅ Stubs demo OK"

# 2b. Stopper proprement les anciens containers (évite les conflits de noms
# quand Docker essaie de recréer un container qui existe déjà)
echo "[deploy] docker compose down..."
docker compose down --remove-orphans || true

# 2c. Rebuilder et redémarrer les services
echo "[deploy] docker compose up --build..."
docker compose up -d --build

# 3. Nettoyer les images orphelines
echo "[deploy] Nettoyage des images obsolètes..."
docker image prune -f

# Helper : attendre qu'un port Streamlit soit prêt (retry sur /_stcore/health)
_wait_for_http() {
    local url="$1" label="$2" max_seconds="${3:-60}"
    local elapsed=0
    echo "[deploy] Attente démarrage $label (max ${max_seconds}s)..."
    while ! curl -sf "$url" >/dev/null 2>&1; do
        sleep 2
        elapsed=$((elapsed + 2))
        if [[ $elapsed -ge $max_seconds ]]; then
            echo "[deploy] ⚠️  $label n'a pas répondu après ${max_seconds}s"
            return 1
        fi
    done
    echo "[deploy] ✅ $label prêt (${elapsed}s)"
}

# 4. Healthcheck DB — attendre que Streamlit soit prêt, puis vérifier les DB
_wait_for_http "http://127.0.0.1:8501/_stcore/health" "levelup" 60

HC_LOG="$DEPLOY_DIR/data/logs/healthcheck_deploy.log"
mkdir -p "$(dirname "$HC_LOG")"
{
    echo "=== Deploy $(date '+%Y-%m-%d %H:%M:%S') — $(git log -1 --oneline) ==="
    docker compose exec -T levelup python scripts/healthcheck_db.py --verbose 2>&1
    echo ""
} >> "$HC_LOG"

# Lire le statut depuis une passe JSON pour détecter les erreurs
HC_STATUS=$(docker compose exec -T levelup python scripts/healthcheck_db.py --json 2>/dev/null \
    | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    errors = [r['db_name'] for r in data if r['status'] == 'error']
    warnings = [r['db_name'] for r in data if r['status'] in ('warning', 'repaired')]
    if errors:
        print('ERROR:' + ','.join(errors))
    elif warnings:
        print('WARNING:' + ','.join(warnings))
    else:
        print('OK')
except Exception as e:
    print('UNKNOWN:' + str(e))
" 2>/dev/null || echo "UNKNOWN:exec failed")

case "$HC_STATUS" in
    OK)
        echo "[deploy] ✅ DB healthcheck OK"
        ;;
    WARNING:*)
        echo "[deploy] ⚠️  DB healthcheck — warnings sur : ${HC_STATUS#WARNING:}"
        echo "[deploy]    Détails : $HC_LOG"
        ;;
    ERROR:*)
        echo "[deploy] ❌ DB healthcheck — erreurs sur : ${HC_STATUS#ERROR:}"
        echo "[deploy]    Détails : $HC_LOG"
        ;;
    *)
        echo "[deploy] ⚠️  DB healthcheck non concluant (${HC_STATUS})"
        ;;
esac

# 5. Vérifier que le container demo (port 8502) est également prêt
if ls "$DEPLOY_DIR"/data/demo/warehouse/*.duckdb >/dev/null 2>&1; then
    _wait_for_http "http://127.0.0.1:8502/_stcore/health" "levelup-demo" 60 \
        || echo "[deploy] ⚠️  levelup-demo lent — logs : docker compose logs levelup-demo"
else
    echo "[deploy] ⚠️  data/demo/warehouse/ vide ou absent — demo désactivé"
    echo "[deploy]    Pour générer les données : python scripts/prepare_demo_data.py"
fi

echo "[deploy] Déployé avec succès : $(git log -1 --oneline)"

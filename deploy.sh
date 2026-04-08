#!/usr/bin/env bash
# deploy.sh — Script de mise à jour LevelUp sur VPS Ionos
#
# Usage (depuis le VPS) :
#   /opt/levelup/deploy.sh
#
# Usage (depuis le poste local) :
#   ssh deploy@212.227.206.42 '/opt/levelup/deploy.sh'
#
# Prérequis : docker, docker compose, git installés sur le VPS.

set -euo pipefail

DEPLOY_DIR="/opt/levelup"

echo "[deploy] Répertoire : $DEPLOY_DIR"
cd "$DEPLOY_DIR"

# 1. Récupérer les derniers commits depuis main (force, ignore les changements locaux)
echo "[deploy] git fetch + reset --hard origin/main..."
git fetch origin main
git reset --hard origin/main
git clean -fd --exclude=data/ --exclude=.env.local --exclude=app_settings.json --exclude=db_profiles.json

# 2. Rebuilder et redémarrer les services (sans downtime des autres)
echo "[deploy] docker compose up --build..."
docker compose up -d --build --no-deps levelup levelup-demo

# 3. Nettoyer les images orphelines
echo "[deploy] Nettoyage des images obsolètes..."
docker image prune -f

# 4. Healthcheck DB — attendre que Streamlit soit prêt, puis vérifier les DB
echo "[deploy] Attente démarrage Streamlit (20s)..."
sleep 20

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

echo "[deploy] Déployé avec succès : $(git log -1 --oneline)"

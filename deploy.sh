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

# 1. Récupérer les derniers commits depuis main
echo "[deploy] git pull origin main..."
git pull origin main

# 2. Rebuilder et redémarrer uniquement le service levelup (sans downtime des autres)
echo "[deploy] docker compose up --build..."
docker compose up -d --build --no-deps levelup

# 3. Nettoyer les images orphelines
echo "[deploy] Nettoyage des images obsolètes..."
docker image prune -f

echo "[deploy] Déployé avec succès : $(git log -1 --oneline)"

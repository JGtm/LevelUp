#!/usr/bin/env bash
# deploy.sh — Mise à jour LevelUp (serveur Go natif) sur VPS Ionos
#
# Usage (depuis le VPS) :
#   /opt/levelup/scripts/deploy.sh
#
# Usage (depuis le poste local) :
#   ssh deploy@<VPS_HOST> '/opt/levelup/scripts/deploy.sh'
#
# Prérequis : docker, docker compose, git installés sur le VPS.
# Déclenché automatiquement par .github/workflows/deploy.yml sur push main.
#
# Stack : binaire Go (Dockerfile multi-stage Vite + Go + Debian slim), port 8000,
# healthcheck HTTP natif `GET /health`. PLUS de Python ni Streamlit dans l'image
# (migration Go) — l'ancien healthcheck `scripts/healthcheck_db.py` + l'attente
# du port Streamlit 8501/8502 ont été retirés car ils faisaient échouer le deploy.

set -euo pipefail

DEPLOY_DIR="/opt/levelup"
APP_PORT=8000   # service levelup       (compose: 127.0.0.1:8000:8000)
DEMO_PORT=8001  # service levelup-demo  (compose: 127.0.0.1:8001:8000)

echo "[deploy] Répertoire : $DEPLOY_DIR"
cd "$DEPLOY_DIR"

# 0. Corriger les permissions git (idempotent — protège contre les écrits root via Docker)
find .git/objects -not -writable -exec chmod u+w {} \; 2>/dev/null || true

# 1. Récupérer les derniers commits depuis main (force, ignore les changements locaux)
echo "[deploy] git fetch + reset --hard origin/main..."
git fetch origin main
git reset --hard origin/main
git clean -fd --exclude=data/ --exclude=.env.local --exclude=app_settings.json --exclude=db_profiles.json

# 1b. Rappel garde-fou prod (non bloquant). Le serveur Go refuse de booter si
# LEVELUP_ENV=production avec une config non sûre ; hors production il démarre en
# posture dev (CORS/CSRF localhost, ownership/secret au défaut) avec un simple
# WARN. On le signale ici de façon visible — cf. bloc PRODUCTION de .env.local.example.
if [[ -f .env.local ]] && grep -Eq '^[[:space:]]*LEVELUP_ENV=production' .env.local; then
    echo "[deploy] ✅ .env.local : LEVELUP_ENV=production (garde-fou prod armé)"
else
    echo "[deploy] ⚠️  .env.local : LEVELUP_ENV != production — serveur en posture DEV"
    echo "[deploy]    (CORS/CSRF limités à localhost, secret/ownership possiblement au défaut)"
    echo "[deploy]    Renseigner le bloc PRODUCTION de .env.local.example si déploiement exposé."
fi

# 2a. Créer les fichiers stub pour les bind-mounts "fichier" de levelup-demo.
# Sans ça, Docker crée des RÉPERTOIRES à la place des fichiers attendus et
# `docker compose up` plante avec "not a directory". Les stubs sont remplacés
# par le vrai contenu lors du regen demo (job deploy-demo de deploy.yml).
mkdir -p "$DEPLOY_DIR/data/demo" "$DEPLOY_DIR/data/logs"
for _stub in db_profiles.json app_settings.json; do
    _path="$DEPLOY_DIR/data/demo/$_stub"
    [[ -d "$_path" ]] && { rm -rf "$_path"; echo "[deploy] Répertoire fantôme supprimé: $_stub"; }
done
[[ -f "$DEPLOY_DIR/data/demo/db_profiles.json" ]] \
    || echo '{"version":"2.1","warehouse_path":"data/warehouse","profiles":{}}' > "$DEPLOY_DIR/data/demo/db_profiles.json"
[[ -f "$DEPLOY_DIR/data/demo/app_settings.json" ]] \
    || echo '{}' > "$DEPLOY_DIR/data/demo/app_settings.json"
echo "[deploy] Stubs demo OK"

# 2b. Garde pré-build : espace disque libre sur / avant de lancer le build.
# Incident 2026-07-23 : cache BuildKit à 44,75 Go, disque saturé PENDANT le build (avant
# même d'atteindre le bornage post-build de l'étape 3b). Si < 10 Go libres, purge
# d'urgence du cache builder AVANT le build, avec une borne plus agressive (2 Go) que le
# nettoyage post-deploy habituel (5 Go). Jamais silencieux : log explicite dans tous les cas.
_avail_gb="$(df --output=avail -BG / | tail -1 | tr -dc '0-9')" || _avail_gb=""
echo "[deploy] Espace disque libre sur / : ${_avail_gb:-?}G"
if [[ "${_avail_gb:-0}" -lt 10 ]]; then
    echo "[deploy] WARN: espace disque < 10 Go — purge d'urgence du cache builder avant build (max-used-space=2GB)..."
    docker builder prune -f --max-used-space=2GB || {
        _emerg_prune_rc=$?
        echo "[deploy] WARN: emergency builder prune failed (exit ${_emerg_prune_rc})"
    }
fi

# 2c. Builder les images AVANT de toucher aux containers (Dockerfile = build Vite +
# Go CGo/DuckDB), pendant que l'ancienne prod tourne encore. Incident 2026-07-23 : avec
# l'ancien ordre (down PUIS `up --build`), un build en échec (disque/réseau/code) laissait
# la prod DOWN jusqu'à intervention manuelle. Ici un échec de build ne coupe rien — les
# containers actuels ne sont ni arrêtés ni recréés. `set -euo pipefail` ferait déjà sortir
# le script sur un `docker compose build` en échec, mais on capte explicitement l'erreur
# pour un message sans ambiguïté (et pour documenter l'invariant : rien n'a bougé).
echo "[deploy] docker compose build (levelup + levelup-demo)..."
if ! docker compose build; then
    echo "[deploy] ERREUR: docker compose build a échoué — prod NON touchée (anciens containers toujours actifs)"
    echo "[deploy]    Corriger la cause (disque/réseau/code) puis relancer scripts/deploy.sh"
    exit 1
fi
echo "[deploy] Build OK"

# 2d. Version de l'app pour les notifications "nouvelle version" (in-app app_release
# + Discord). Le serveur ne notifie que si cfg.AppVersion est un vrai semver (≠ "dev") ;
# on lui passe le dernier tag atteignable via l'env LEVELUP_APP_VERSION (interpolé par
# docker-compose). L'anti-spam (major/minor, last_notified_version) est géré côté Go :
# entre deux tags, la même valeur => aucune re-notification. Fallback "dev" (dépôt sans
# tag) => notifications gardées OFF, comportement inchangé.
# --match 'v*.*.*' : ne retenir QUE les tags de release semver (release.yml se
# déclenche sur ce motif) et ignorer les tags de travail (ex. "Shared-social-fixed").
git fetch --tags --quiet origin 2>/dev/null || true
LEVELUP_APP_VERSION="$(git describe --tags --abbrev=0 --match 'v*.*.*' 2>/dev/null || echo dev)"
export LEVELUP_APP_VERSION
echo "[deploy] LEVELUP_APP_VERSION=$LEVELUP_APP_VERSION"

# 2e. Basculer : le build (2c) a réussi, donc on peut arrêter les anciens containers puis
# redémarrer avec les images tout juste construites. PAS de --build sur le `up` : les
# images sont déjà prêtes (un `--build` ici referait un build pour rien et réintroduirait
# une fenêtre de risque après le down). Fenêtre d'indisponibilité réduite au seul temps du
# restart (down + up), il n'y a plus de fenêtre de build entre les deux.
echo "[deploy] docker compose down..."
docker compose down --remove-orphans || true

echo "[deploy] docker compose up -d (images déjà construites à l'étape 2c)..."
docker compose up -d

# 3. Nettoyer les images orphelines (garder les images < 24h : rollback rapide possible
# le jour même en re-taguant l'image N-1 si le nouveau déploiement pose problème).
echo "[deploy] Nettoyage des images obsolètes (> 24h)..."
docker image prune -f --filter "until=24h"

# 3b. Borner le cache de build BuildKit. Sans ça il croît sans limite à chaque
# deploy (chaque build empile ses couches) et finit par saturer le disque du VPS
# — incident disque 2026-06-27 : 33 Go de cache accumulé. On garde 5 Go de cache
# récent pour des builds incrémentaux rapides ; au-delà, BuildKit évince le plus ancien.
# PIÈGE (incident 2026-07-13, disque 100%, prod down) : `docker buildx prune` vide le
# cache du builder BUILDX, mais `docker compose build` passe par le builder du DAEMON —
# deux stores distincts. L'éviction ne touchait donc jamais le bon cache (46 Go
# accumulés en 2 semaines). `docker builder prune` cible le builder du daemon.
# PIÈGE (incident 2026-07-23, disque saturé, cache BuildKit à 44,75 Go) : `--keep-storage`
# est déprécié depuis Docker 29.4.0 / BuildKit 0.29 et silencieusement remappé vers
# `--reserved-space`, qui est un PLANCHER d'espace réservé (jamais purgé en dessous) et
# NON un plafond — les deux prunes ci-dessous étaient donc des no-op silencieux depuis la
# montée de version. Le vrai plafond est `--max-used-space`. On ne masque plus l'échec
# avec `|| true` : la sortie de la commande (dont le `Total:` récupéré) reste visible dans
# les logs de déploiement, et un échec est loggé explicitement — sans faire échouer un
# déploiement déjà basculé (les services tournent déjà à ce stade, cf. étape 2e).
echo "[deploy] Bornage du cache de build Docker (max-used-space 5GB)..."
docker builder prune -f --max-used-space=5GB || {
    _builder_prune_rc=$?
    echo "[deploy] WARN: builder prune failed (exit ${_builder_prune_rc})"
}
echo "[deploy] Bornage du cache buildx (max-used-space 1GB)..."
docker buildx prune -f --max-used-space=1GB || {
    _buildx_prune_rc=$?
    echo "[deploy] WARN: buildx prune failed (exit ${_buildx_prune_rc})"
}

# Helper : attendre qu'un endpoint HTTP réponde (retry jusqu'à max_seconds)
_wait_for_http() {
    local url="$1" label="$2" max_seconds="${3:-90}"
    local elapsed=0
    echo "[deploy] Attente démarrage $label (max ${max_seconds}s)..."
    while ! curl -sf "$url" >/dev/null 2>&1; do
        sleep 3
        elapsed=$((elapsed + 3))
        if [[ $elapsed -ge $max_seconds ]]; then
            echo "[deploy] ⚠️  $label n'a pas répondu après ${max_seconds}s"
            return 1
        fi
    done
    echo "[deploy] ✅ $label prêt (${elapsed}s)"
}

# 4. Healthcheck serveur Go — bloquant. `GET /health` ouvre metadata + shared en
# read-only et renvoie le nb de matchs + la version DuckDB : un 200 confirme donc
# à la fois que le binaire est up ET que les DBs s'ouvrent correctement.
HC_LOG="$DEPLOY_DIR/data/logs/healthcheck_deploy.log"
mkdir -p "$(dirname "$HC_LOG")"
if _wait_for_http "http://127.0.0.1:${APP_PORT}/health" "levelup (Go)" 90; then
    {
        echo "=== Deploy $(date '+%Y-%m-%d %H:%M:%S') — $(git log -1 --oneline) ==="
        curl -s "http://127.0.0.1:${APP_PORT}/health" || true
        echo ""
    } >> "$HC_LOG"
    echo "[deploy] ✅ /health OK — détails : $HC_LOG"
else
    echo "[deploy] ❌ levelup (Go) ne répond pas sur :${APP_PORT}/health — deploy en échec"
    echo "[deploy]    Logs : docker compose logs --tail=80 levelup"
    docker compose logs --tail=80 levelup || true
    exit 1
fi

# 5. Vérifier le container demo (port 8001) — warn-only (non bloquant).
if ls "$DEPLOY_DIR"/data/demo/warehouse/*.duckdb >/dev/null 2>&1; then
    _wait_for_http "http://127.0.0.1:${DEMO_PORT}/health" "levelup-demo (Go)" 60 \
        || echo "[deploy] ⚠️  levelup-demo lent/absent — logs : docker compose logs levelup-demo"
else
    echo "[deploy] ⚠️  data/demo/warehouse/ vide ou absent — demo non vérifié"
    echo "[deploy]    Regen demo : job deploy-demo de .github/workflows/deploy.yml"
fi

echo "[deploy] Déployé avec succès : $(git log -1 --oneline)"

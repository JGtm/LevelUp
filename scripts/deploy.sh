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

# 2b. Stopper proprement les anciens containers (évite les conflits de noms
# quand Docker recrée un container qui existe déjà)
echo "[deploy] docker compose down..."
docker compose down --remove-orphans || true

# 2c. Démarrer les services. Chemin nominal : PULL de l'image pré-buildée en CI
# (GHCR, job build-image de deploy.yml) ; l'image est identique pour prod et démo
# (même Dockerfile, aucun build-arg divergent) → on la tague sous les deux noms
# compose par défaut puis `up --no-build`. Un apt-get / npm ci / go build cassé est
# alors attrapé en CI AVANT ce point (au lieu d'échouer ici, en pleine prod).
#
# FALLBACK build local = KILL-SWITCH TRANSITOIRE (bascule par défaut 2026-07-17).
#   Raison d'être : au premier déploiement de ce mécanisme, le VPS n'a pas encore
#   de login GHCR → `docker pull` d'un package privé échoue ; le build local prend
#   le relais et la prod reste servie. Idem si GHCR/réseau indisponible.
#   Retrait cible : 2026-Q4 (après activation du login GHCR côté VPS —
#   docs/RUNBOOK_GO_LIVE.md § "Activation GHCR pull").
#   Critère mesurable de retrait : sur ≥ 4 déploiements consécutifs, la ligne
#   "[deploy] Pull GHCR échoué" n'apparaît PLUS dans les logs de deploy
#   (grep sur data/logs/healthcheck_deploy.log + sortie CI) → le pull est fiable,
#   le fallback devient du code mort et se supprime (avec ce bloc de commentaire).
_project="$(basename "$DEPLOY_DIR")"   # nom de projet compose (→ images "<projet>-<service>")
_image_pulled=false
if [[ -n "${LEVELUP_IMAGE:-}" ]]; then
    echo "[deploy] Image CI demandée : ${LEVELUP_IMAGE} — tentative docker pull..."
    if docker pull "${LEVELUP_IMAGE}"; then
        # Tague l'image GHCR sous les noms compose par défaut (prod + démo) : les
        # commandes `docker compose` ultérieures (dont le job deploy-demo) la
        # retrouvent sans rebuild.
        docker tag "${LEVELUP_IMAGE}" "${_project}-levelup:latest"
        docker tag "${LEVELUP_IMAGE}" "${_project}-levelup-demo:latest"
        _image_pulled=true
        echo "[deploy] Pull GHCR OK — démarrage sans build local"
    else
        echo "[deploy] Pull GHCR échoué (login absent / réseau / image privée) — fallback build local"
    fi
else
    echo "[deploy] LEVELUP_IMAGE absent — build local (comportement historique / fallback)"
fi

if [[ "${_image_pulled}" == true ]]; then
    # Belt-and-suspenders : si `up --no-build` échoue malgré le tag (nom d'image
    # imprévu), on retombe sur le build local plutôt que de laisser la prod down.
    echo "[deploy] docker compose up --no-build (image GHCR)..."
    if ! docker compose up -d --no-build; then
        echo "[deploy] WARN up --no-build a échoué — fallback build local"
        docker compose up -d --build
    fi
else
    echo "[deploy] docker compose up --build (build local)..."
    docker compose up -d --build
fi

# 3. Nettoyer les images orphelines
echo "[deploy] Nettoyage des images obsolètes..."
docker image prune -f

# 3b. Borner le cache de build BuildKit. Sans ça il croît sans limite à chaque
# deploy (chaque build empile ses couches) et finit par saturer le disque du VPS
# — incident disque 2026-06-27 : 33 Go de cache accumulé. On garde 5 Go de cache
# récent pour des builds incrémentaux rapides ; au-delà, BuildKit évince le plus ancien.
# PIÈGE (incident 2026-07-13, disque 100%, prod down) : `docker buildx prune` vide le
# cache du builder BUILDX, mais `docker compose build` passe par le builder du DAEMON —
# deux stores distincts. L'éviction ne touchait donc jamais le bon cache (46 Go
# accumulés en 2 semaines). `docker builder prune` cible le builder du daemon.
echo "[deploy] Bornage du cache de build Docker (keep 5GB)..."
docker builder prune -f --keep-storage=5GB || true
docker buildx prune -f --keep-storage=1GB || true

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

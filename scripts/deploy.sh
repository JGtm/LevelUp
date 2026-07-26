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
# nettoyage post-deploy habituel (12 Go, étape 3b) : ici on sacrifie sciemment la vitesse
# du build pour ne pas saturer le disque. Jamais silencieux : log explicite dans tous les cas.
_avail_gb="$(df --output=avail -BG / | tail -1 | tr -dc '0-9')" || _avail_gb=""
echo "[deploy] Espace disque libre sur / : ${_avail_gb:-?}G"
if [[ "${_avail_gb:-0}" -lt 10 ]]; then
    echo "[deploy] WARN: espace disque < 10 Go — purge d'urgence du cache builder avant build (max-used-space=2GB)..."
    docker builder prune -f --max-used-space=2GB || {
        _emerg_prune_rc=$?
        echo "[deploy] WARN: emergency builder prune failed (exit ${_emerg_prune_rc})"
    }
fi

# 2c. Version de l'app pour les notifications "nouvelle version" (in-app app_release
# + Discord). Le serveur ne notifie que si cfg.AppVersion est un vrai semver (≠ "dev").
#
# V721-15 (2026-07-25) : la version N'EST PLUS bakée dans le binaire. Elle transite
# UNIQUEMENT par l'environnement d'exécution (LEVELUP_APP_VERSION), persisté dans
# `.env` juste en dessous. Raison : en faire une entrée de build obligeait à
# recompiler tout le Go à chaque tag, et faisait diverger les couches prod/démo —
# donc deux compilations CGO simultanées, d'où l'épuisement mémoire du VPS au
# deploy v7.2.0. Le `.env` couvre le cas qui avait motivé le bakage (env runtime
# perdu quand la regen démo recrée un conteneur depuis une session ssh nue) : docker
# compose le lit pour TOUS les services et à TOUTE commande. Cf. Dockerfile.
#
# L'anti-spam (major/minor, last_notified_version) est géré côté Go : entre deux
# tags, la même valeur => aucune re-notification. Fallback "dev" (dépôt sans tag)
# => notifications gardées OFF, comportement inchangé.
# --match 'v*.*.*' : ne retenir QUE les tags de release semver (release.yml se
# déclenche sur ce motif) et ignorer les tags de travail (ex. "Shared-social-fixed").
git fetch --tags --quiet origin 2>/dev/null || true
LEVELUP_APP_VERSION="$(git describe --tags --abbrev=0 --match 'v*.*.*' 2>/dev/null || echo dev)"
export LEVELUP_APP_VERSION
echo "[deploy] LEVELUP_APP_VERSION=$LEVELUP_APP_VERSION (env runtime uniquement — plus bakée au build depuis V721-15)"
# Persister dans .env (gitignoré, lu automatiquement par docker compose) : tout
# `docker compose up` hors de ce script (regen démo dans deploy.yml, ops manuelles
# ssh, reboot) recréerait sinon le conteneur avec le défaut "dev" — c'est arrivé
# au deploy v7.1.0 (le job demo-regen redémarre la prod dans une session SSH sans
# la variable). L'export shell ci-dessus reste prioritaire sur .env au `up` de 2e.
# Depuis V721-15 c'est le SEUL porteur de la version (plus de repli baké dans le
# binaire) : l'étape 4b ci-dessous vérifie donc que la version réellement servie
# correspond, pour qu'un .env perdu échoue bruyamment au lieu de désactiver
# silencieusement les notifications de version.
touch .env
grep -v '^LEVELUP_APP_VERSION=' .env > .env.tmp || true
printf 'LEVELUP_APP_VERSION=%s\n' "$LEVELUP_APP_VERSION" >> .env.tmp
mv .env.tmp .env

# 2d. Builder les images AVANT de toucher aux containers (Dockerfile = build Vite +
# Go CGo/DuckDB), pendant que l'ancienne prod tourne encore. Incident 2026-07-23 : avec
# l'ancien ordre (down PUIS `up --build`), un build en échec (disque/réseau/code) laissait
# la prod DOWN jusqu'à intervention manuelle. Ici un échec de build ne coupe rien — les
# containers actuels ne sont ni arrêtés ni recréés. `set -euo pipefail` ferait déjà sortir
# le script sur un `docker compose build` en échec, mais on capte explicitement l'erreur
# pour un message sans ambiguïté (et pour documenter l'invariant : rien n'a bougé).
# SÉRIALISÉ service par service : un `docker compose build` nu construit les deux
# images EN PARALLÈLE, et deux builds CGO simultanés sur 2 vCPU / 2 Go suffisent à
# épuiser la mémoire (incident 2026-07-25, VPS gelé). Depuis V721-15 les deux
# services n'ont plus AUCUN build-arg, donc leur couche go-builder est identique par
# construction et le 2e build est un cache-hit garanti — la sérialisation reste
# néanmoins en place : elle ne coûte rien et borne la mémoire même si une divergence
# réapparaissait un jour (nouvelle dépendance de build, changement de contexte).
# NB : une modif de CE script ne prend effet qu'au deploy suivant (bash retient
# l'ancien inode pendant que le git reset remplace le fichier).
#
# Visibilité mémoire avant build (2026-07-26) — DIAGNOSTIC SEUL, aucune garde bloquante.
# Les 3 gels machine des 25-26/07 se sont produits exactement ici : à court de RAM et sans
# swap, le noyau évince les pages exécutables plutôt que de tuer le build, et tout le
# système part en livelock (aucun OOM-kill dans les logs du boot précédent — la machine ne
# meurt pas, elle bat la page). Un swap de 2 Go a été posé à la main sur le VPS le 26/07 ;
# ces deux lignes permettent de corréler un futur incident avec l'état mémoire réel au
# moment du build. Volontairement PAS de seuil qui refuserait le déploiement : MemAvailable
# est une lecture instantanée qui varie avec les cycles d'auto-sync en cours, un refus
# serait arbitraire. `|| true` sur les deux : une sonde de diagnostic ne doit jamais faire
# échouer un déploiement (procps absent, /proc/meminfo illisible — cf. set -euo pipefail).
awk '/MemAvailable/ {printf "[deploy] MemAvailable avant build: %d Mo\n", $2/1024}' /proc/meminfo || true
echo "[deploy] Mémoire/swap avant build (free -m, ligne Swap) :"
free -m | tail -1 || true
echo "[deploy] docker compose build levelup (puis levelup-demo, sérialisé)..."
if ! docker compose build levelup; then
    echo "[deploy] ERREUR: build levelup a échoué — prod NON touchée (anciens containers toujours actifs)"
    echo "[deploy]    Corriger la cause (disque/réseau/code) puis relancer scripts/deploy.sh"
    exit 1
fi
if ! docker compose build levelup-demo; then
    echo "[deploy] ERREUR: build levelup-demo a échoué — prod NON touchée (anciens containers toujours actifs)"
    echo "[deploy]    Corriger la cause (disque/réseau/code) puis relancer scripts/deploy.sh"
    exit 1
fi
echo "[deploy] Build OK"

# 2e. Basculer : le build (2d) a réussi, donc on peut arrêter les anciens containers puis
# redémarrer avec les images tout juste construites. PAS de --build sur le `up` : les
# images sont déjà prêtes (un `--build` ici referait un build pour rien et réintroduirait
# une fenêtre de risque après le down). Fenêtre d'indisponibilité réduite au seul temps du
# restart (down + up), il n'y a plus de fenêtre de build entre les deux.
echo "[deploy] docker compose down..."
docker compose down --remove-orphans || true

echo "[deploy] docker compose up -d (images déjà construites à l'étape 2d)..."
# --remove-orphans aussi sur le up (2026-07-26) : le down qui précède le porte déjà, mais
# si le down échoue partiellement (démon occupé) ou qu'un conteneur a été créé hors compose
# entre les deux, le up nu échoue sur « container name already in use » (incident
# 2026-07-23 16:30, run 30025386880 : /levelup-levelup-demo-1 déjà pris). Idempotent.
docker compose up -d --remove-orphans

# 3. Nettoyer les images orphelines (garder les images < 24h : rollback rapide possible
# le jour même en re-taguant l'image N-1 si le nouveau déploiement pose problème).
echo "[deploy] Nettoyage des images obsolètes (> 24h)..."
docker image prune -f --filter "until=24h"

# 3b. Borner le cache de build BuildKit. Sans ça il croît sans limite à chaque
# deploy (chaque build empile ses couches) et finit par saturer le disque du VPS
# — incident disque 2026-06-27 : 33 Go de cache accumulé. Au-delà du plafond, BuildKit
# évince les entrées les moins récemment utilisées.
#
# DIMENSIONNEMENT DU PLAFOND (2026-07-26) — règle contre-intuitive, à ne pas « optimiser »
# à la baisse : le plafond doit être SUPÉRIEUR au working-set d'UN build complet. En
# dessous, la purge d'un deploy évince le cache dont le deploy SUIVANT a besoin — chaque
# déploiement repart alors d'un build à FROID (compilation CGO/DuckDB complète : ~7 min sur
# 2 vCPU contre 2-4 min en incrémental) avec un pic mémoire que la machine (1,8 Go de RAM,
# ~950 Mo résidents pour les 2 conteneurs + l'OS) n'absorbe pas. C'est la cause racine des
# 3 gels du VPS des 25-26/07, tous survenus pendant l'étape 2d : machine morte par
# épuisement mémoire — sans swap le noyau part en livelock au lieu d'OOM-killer le build.
# Un build complet mesuré = ~5,7 Go de cache (`docker system df`) : l'ancien plafond de
# 5 Go était donc STRUCTURELLEMENT plus bas que le working-set, il garantissait l'éviction
# à chaque deploy. Son commentaire annonçait pourtant « 5 Go de cache récent pour des
# builds incrémentaux rapides » — doc inversée (anti-patron n°9) : le plafond détruisait
# exactement ce qu'il prétendait préserver.
# 12 Go = deux générations de build + marge, et reste très loin des 33-46 Go des incidents
# disque (disque de 79 Go à 50 % d'occupation : cache plein ≈ 15 % du disque). Ne pas
# ré-abaisser cette valeur sans avoir re-mesuré le working-set d'un build.
# Ce plafond couvre AUSSI les cache mounts Go du Dockerfile (`type=cache` : module cache +
# cache de compilation), comptés dans le même store et purgés par la même commande — c'est
# voulu et sans danger, 12 Go les couvre largement.
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
echo "[deploy] Bornage du cache de build Docker (max-used-space 12GB)..."
docker builder prune -f --max-used-space=12GB || {
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

# 4b. Vérifier que la version SERVIE correspond à celle qu'on vient de déployer.
#
# Depuis V721-15 la version n'est plus bakée dans le binaire : elle ne vient QUE de
# LEVELUP_APP_VERSION (persisté en .env à l'étape 2c). Sans ce contrôle, un .env
# perdu ou un conteneur recréé sans la variable ferait retomber l'app sur "dev" et
# désactiverait SILENCIEUSEMENT les notifications de nouvelle version — exactement
# le défaut qui avait été constaté aux deploy v7.0.0 et v7.1.0, et qui avait motivé
# le bakage. On le rend donc bruyant plutôt que de le prévenir par redondance.
#
# Warn-only et non bloquant : à ce stade la prod tourne déjà et répond au
# healthcheck ; faire échouer le déploiement pour un libellé de version serait
# disproportionné. Le message doit en revanche être impossible à rater dans les logs.
if [[ "$LEVELUP_APP_VERSION" != "dev" ]]; then
    # Clé exacte du payload : "app_version" (domain.HealthResponse, json:"app_version").
    _served_version="$(curl -sf "http://127.0.0.1:${APP_PORT}/health" 2>/dev/null \
        | grep -o '"app_version"[[:space:]]*:[[:space:]]*"[^"]*"' \
        | head -1 | sed 's/.*"\([^"]*\)"$/\1/')"
    if [[ -z "$_served_version" ]]; then
        echo "[deploy] ⚠️  Version servie illisible depuis /health — contrôle ignoré"
    elif [[ "$_served_version" != "$LEVELUP_APP_VERSION" ]]; then
        echo "[deploy] ❌ VERSION SERVIE INCOHÉRENTE : /health annonce '$_served_version'," \
             "attendu '$LEVELUP_APP_VERSION'."
        echo "[deploy]    Cause probable : .env perdu ou conteneur recréé sans LEVELUP_APP_VERSION."
        echo "[deploy]    Conséquence : les notifications « nouvelle version » ne partiront PAS."
        echo "[deploy]    Correctif : vérifier /opt/levelup/.env puis 'docker compose up -d levelup'."
    else
        echo "[deploy] ✅ Version servie conforme : $_served_version"
    fi
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

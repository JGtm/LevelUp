# ============================================================================
# Stage 1 — Build React (Vite)
# ============================================================================
FROM node:22-slim AS web-builder

WORKDIR /build/web

# Cache npm : installer les dépendances séparément du code source
# .npmrc : conserve `legacy-peer-deps=true` (workaround TS6 vs
# openapi-typescript@7.x peer dep — cf. apps/web/.npmrc).
COPY apps/web/package.json apps/web/package-lock.json apps/web/.npmrc ./
# Cache mount BuildKit (2026-07-26, même motivation que le stage go-builder) : /root/.npm
# est le cache npm par défaut ici (l'image node tourne en root et ne redéfinit pas
# npm_config_cache). Il survit à l'invalidation de la couche, ce qui rend `--prefer-offline`
# réellement offline : une invalidation du lockfile — ou une éviction de couche par la purge
# post-deploy — ne repart plus d'un téléchargement complet des dépendances.
RUN --mount=type=cache,target=/root/.npm \
    npm ci --prefer-offline

# Garde-fou binaire natif lightningcss : il est livré en dépendance OPTIONNELLE
# (optionalDependencies, os/cpu-gated). Combiné à legacy-peer-deps (.npmrc),
# `npm ci` saute silencieusement le binaire natif linux → `npm run build` plante
# en MODULE_NOT_FOUND (.../lightningcss/node/index.js). On vérifie que le binaire
# charge et, au besoin, on installe le paquet de la plateforme à la version EXACTE
# de lightningcss, sans toucher au lockfile ni à package.json. Idempotent (no-op si
# déjà présent). Cf. npm/cli legacy-peer-deps + optional-deps cross-platform.
RUN node -e "require('lightningcss')" 2>/dev/null \
 || npm install --no-save --no-package-lock \
      "lightningcss-linux-x64-gnu@$(node -p "require('lightningcss/package.json').version")"

# Code source
COPY apps/web/ ./

# Build Vite (output : /build/web/dist)
RUN npm run build

# Garde-rail assets publics : chaque fichier de public/ doit se retrouver dans
# dist/ (Vite copie public/ tel quel). Echec du build sinon, avec la liste des
# manquants — prévention d'une régression FUTURE (option `copyPublicDir`
# désactivée, chemin public/ déplacé/renommé...), PAS le correctif d'un bug
# prod déjà survenu : aucune investigation n'a confirmé d'image absente en
# production (l'hypothèse initiale — .dockerignore strippant public/ — était
# fausse : un motif .dockerignore est ancré à la racine du contexte, `*.png`
# ne traverse pas les sous-dossiers ; cf. .dockerignore + thought_log
# 2026-07-26).
RUN node scripts/verify-public-in-dist.mjs

# Le front compilé se coupe en deux (ADR 0045, amendement L24) : les images des titres (dist/titles, identiques
# à public/titles, garde ci-dessus) font partie du PAQUET DE RESSOURCES (stage pack-builder) ; le reste (JS, CSS,
# index.html : couplé au code) est celui du MOTEUR. Le paquet se copie dans l'image AVANT le moteur : un
# déploiement qui ne change que le moteur ne retire que les couches du moteur.
RUN mkdir -p /build/engine-web \
 && cp -a dist/. /build/engine-web/ \
 && rm -rf /build/engine-web/titles

# ============================================================================
# Stage 1b — Paquet de ressources : manifeste et empreinte (ADR 0045, amendement L24)
# ============================================================================
# Le paquet hébergé = ce que l'image embarque et que les mises à jour du moteur ne changent presque jamais :
# static/ et les images des titres. Le référentiel du rejeu (data/titles/*/reference) vit dans le volume ./data
# de la prod, pas dans l'image. Un seul générateur de manifeste (cmd/resources-manifest, le même que celui de
# l'installeur Windows) ; le stage ne dépend QUE du paquet et de ce générateur : changer le moteur ne le
# recompile ni ne le recalcule. Son empreinte entre dans le binaire serveur (stage go-builder) et le démarrage
# la compare au manifeste de l'image : une image mal assemblée refuse de démarrer.
FROM golang:1.26.11-bookworm AS pack-builder

WORKDIR /build/apps/go-api
COPY apps/go-api/go.mod apps/go-api/go.sum ./
COPY apps/go-api/cmd/resources-manifest/ ./cmd/resources-manifest/
COPY apps/go-api/internal/resourcepack/ ./internal/resourcepack/

# Chemins de l'image : /app (racine de ressources), front sous apps/web/dist.
COPY static /pack/static
COPY apps/web/public/titles /pack/apps/web/dist/titles

# GOWORK=off : ce stage ne copie que le module apps/go-api, sans go.work ni briques ; le générateur n'importe
# que la bibliothèque standard et golang.org/x/crypto.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOWORK=off CGO_ENABLED=0 go run ./cmd/resources-manifest -dir /pack > /pack-fingerprint \
 && test "$(wc -c < /pack-fingerprint)" -eq 65

# ============================================================================
# Stage 2 — Build Go (CGo activé pour DuckDB bindings)
# ============================================================================
# Image épinglée sur le correctif exact (pas la branche flottante `1.26`) :
# `scripts/deploy.sh` construit sans `--pull`, donc un tag flottant resterait sur la
# version déjà présente sur l'hôte. Même version que la ligne `go` de
# apps/go-api/go.mod, de go.work et de chaque libs/*/go.mod ; les monter ensemble (garde-rail :
# internal/archlint/dockerfile_go_version_test.go).
FROM golang:1.26.11-bookworm AS go-builder

# Espace de travail Go (ADR 0041) : le serveur est le module apps/go-api, relié par go.work aux
# briques sous libs/ (aucune directive `replace`). La racine de l'espace de travail est /build ;
# les `go build` se lancent depuis /build/apps/go-api, où go.work est trouvé en remontant.
WORKDIR /build

# Dépendances Go (cache Docker — séparé du code source). Ordre des couches : go.work, puis les
# go.mod/go.sum de chaque module de l'espace de travail, AVANT `go mod download` ; tout module
# ajouté à go.work doit avoir sa ligne COPY ici (go.work refuse un `use` sans go.mod).
COPY go.work go.work.sum ./
COPY apps/go-api/go.mod apps/go-api/go.sum ./apps/go-api/
COPY libs/film/go.mod libs/film/go.sum ./libs/film/
COPY libs/gamefiles/go.mod libs/gamefiles/go.sum ./libs/gamefiles/
COPY libs/haloapi/go.mod libs/haloapi/go.sum ./libs/haloapi/
COPY libs/himap/go.mod libs/himap/go.sum ./libs/himap/

# Cache mounts BuildKit (2026-07-26) — présents sur les 4 RUN Go de ce stage :
#   /go/pkg/mod           = GOMODCACHE par défaut (l'image officielle fixe GOPATH=/go)
#   /root/.cache/go-build = GOCACHE par défaut (l'image tourne en root, donc HOME=/root)
#
# Ces caches survivent à l'invalidation des COUCHES. Une couche invalidée (code modifié)
# ou évincée (purge post-deploy, cf. scripts/deploy.sh étape 3b) ne coûte plus une
# compilation CGO/DuckDB complète : le compilateur retrouve ses objets et un rebuild
# « à froid » redevient incrémental. Motivation : les 3 gels du VPS des 25-26/07 — le
# plafond de purge de 5 Go, inférieur aux ~5,7 Go de cache que produit un build, évinçait
# le cache à chaque déploiement et transformait donc tout deploy en build à froid, dont le
# pic mémoire est ingérable sur 1,8 Go de RAM. Le plafond est corrigé (12 Go), ces mounts
# sont la seconde ligne de défense : ils rendent une éviction de couche peu coûteuse.
#
# Les 4 RUN portent les MÊMES mounts et ce n'est PAS de la redondance : le contenu d'un
# cache mount n'est jamais écrit dans la couche d'image, donc les modules téléchargés ici
# ne sont visibles des `go build` suivants que s'ils montent le même cache.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Code source Go : le module de l'application, puis les briques (libs/ en entier : ooz y embarque
# ses sources C++, mapvar ses graines de test).
COPY apps/go-api/ ./apps/go-api/
COPY libs/ ./libs/
WORKDIR /build/apps/go-api

# La version N'EST PLUS une entrée de build (V721-15, 2026-07-25).
#
# Elle était bakée via `ARG VERSION` + `-X main.version`. Deux conséquences, toutes
# deux constatées en production :
#   1. Un simple tag de release invalidait cette couche et relançait une compilation
#      CGO complète — sur un VPS 2 vCPU / 2 Go, c'est le poste le plus lourd du
#      déploiement.
#   2. Les images prod et démo recevant des valeurs différentes, leurs couches
#      go-builder divergeaient et BuildKit lançait DEUX compilations CGO au lieu
#      d'une → épuisement mémoire, VPS gelé (incident 2026-07-25, deploy v7.2.0).
#      Aligner les valeurs corrigeait le symptôme par convention ; les retirer
#      supprime la divergence par construction.
#
# La version vient désormais UNIQUEMENT de l'environnement d'exécution
# (`LEVELUP_APP_VERSION`, cf. config.go). scripts/deploy.sh la persiste dans `.env`,
# que docker compose lit automatiquement pour TOUS les services et à TOUTE commande
# (`up`, `--force-recreate`, reboot, session ssh manuelle) — c'est ce qui couvre le
# cas ayant motivé le bakage à l'origine (env perdu au recreate de la regen démo).
# Effet de bord voulu : changer de version ne recompile plus rien.
#
# `main.version` (cmd/server/main.go) reste un repli utile pour un `go build` local
# avec ldflags ; il n'est simplement plus renseigné ici.

# `-s -w` (2026-07-26) : retire la table de symboles et les données DWARF des 3 binaires.
# Le LINK est le poste le plus gourmand en mémoire de tout le build — c'est la marge qui
# manquait sur les 1,8 Go du VPS pendant les gels des 25-26/07 — et les binaires produits
# sont nettement plus petits (image et transfert réduits d'autant).
# Aucune perte d'exploitabilité : les stack traces Go restent complètes avec fichier:ligne
# (elles proviennent de la pclntab, que `-w` ne touche pas — seul un débogueur type delve
# aurait besoin du DWARF), et aucun outillage du dépôt ne symbolise ces binaires : ni
# delve, ni profilage pprof exposé sur l'image de prod. Les binaires publiés par
# release.yml ont leurs propres ldflags, indépendants de ce Dockerfile.

# CLI levelup (seed, seed-demo, backfill…) — requis par le job deploy-demo de
# la CI (docker compose run levelup `levelup` seed-demo). Sans ce binaire, la
# regen démo échoue avec "executable file not found in $PATH".
# Mêmes cache mounts et mêmes `-s -w` que ci-dessus (justification au build du serveur).
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-s -w -extldflags '-static'" \
    -o /build/levelup \
    ./cmd/levelup/

# CLI backfill_objective_stats — ops prod v7.2 : backfill historique des stats
# d'objectifs (re-téléchargement API, serveur ARRÊTÉ pendant l'opération car il
# prend le lock RW de la shared DB). Binaire dédié hors CLI levelup.
# Mêmes cache mounts et mêmes `-s -w` que ci-dessus (justification au build du serveur).
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-s -w -extldflags '-static'" \
    -o /build/backfill_objective_stats \
    ./cmd/backfill_objective_stats/

# CLI backfill_milestone_dates — dates de franchissement des jalons (`earned_at`) recalculées en
# rejouant les matchs du joueur, Campagne exclue (`ops.BackfillMilestoneDates`). Serveur ARRÊTÉ
# (écrit les player DB). Binaire dédié hors CLI levelup, comme backfill_objective_stats.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-s -w -extldflags '-static'" \
    -o /build/backfill_milestone_dates \
    ./cmd/backfill_milestone_dates/

# Serveur : construit EN DERNIER, parce qu'il est le seul binaire à embarquer l'empreinte du paquet de ressources
# (`-X main.resourcesFingerprint`) : un changement du paquet ne rejoue que cette compilation (le cache de
# compilation est monté), jamais celle des CLI ci-dessus. Mêmes cache mounts et mêmes `-s -w` que ci-dessus.
# Une empreinte vide (build local sans paquet) désactive le contrôle de démarrage.
COPY --from=pack-builder /pack-fingerprint /build/pack-fingerprint
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-s -w -X main.resourcesFingerprint=$(cat /build/pack-fingerprint) -extldflags '-static'" \
    -o /build/levelup-server \
    ./cmd/server/

# ============================================================================
# Stage 3 — Runtime minimal (Debian slim)
# ============================================================================
FROM debian:bookworm-slim

# gosu : switch user non-root (même pattern que l'ancienne image Python).
# ffmpeg : REQUIS au runtime — le serveur exécute ffmpeg/ffprobe pour générer
# les miniatures (.webp animées) ET transcoder les clips multipistes en HLS à
# l'ingestion (cf. internal/ops/media_thumbnails.go, internal/media/hls.go).
# Sans lui : "ffprobe: executable file not found in $PATH" → médias récents
# illisibles + sans miniature (régression post-cutover Go, l'image Python
# l'embarquait). Debian fournit ffmpeg ET ffprobe dans le même paquet.
RUN apt-get update && apt-get install -y --no-install-recommends \
    gosu \
    ca-certificates \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# ----------------------------------------------------------------------------
# 1. PAQUET DE RESSOURCES (couches basses : elles ne changent presque jamais, le VPS ne les retire pas à
#    chaque déploiement). Assets statiques UI servis sous /static/ (maps, ranks, medals, commendations,
#    prestige, ~23 MB ; sans eux 404 sur tout /static/*, prod + demo), images des titres du front, et le
#    manifeste dont l'empreinte est comparée à celle du binaire au démarrage.
# ----------------------------------------------------------------------------
COPY --from=pack-builder /pack/static /app/static
COPY --from=pack-builder /pack/apps/web/dist/titles /app/apps/web/dist/titles
COPY --from=pack-builder /pack/resources.manifest.json /app/resources.manifest.json

# ----------------------------------------------------------------------------
# 2. MOTEUR (couches hautes : changent à chaque version)
# ----------------------------------------------------------------------------
# Config title-aware (mappings/fields.toml, assets, outcomes, capabilities).
# Lue au boot par fieldMappingsRegistry.LoadFromConfigDir + RankCatalog. Sans ce
# COPY, config/titles/ n'existe pas dans le conteneur → field_mappings_load_warning,
# semantic adapter + RankCatalog jamais chargés → rangs carrière en EN (prod+demo)
# et libellés métier non title-aware. ~117 KB.
COPY config /app/config

# Docs (RELEASE_NOTES.md, CHANGELOG.md, FR/) — lus au runtime par
# release_notes_service.go (docs/FR/RELEASE_NOTES.md) et changelog.go
# (docs/CHANGELOG.md). Sans ce COPY, /app/docs n'existe pas dans le conteneur →
# la page "Notes de version" (/help) renvoie 500 et /changelog 404 (les deux
# titres). Note : docs/ embarque aussi les captures d'écran (docs/screenshots/,
# ~37 PNG) — .dockerignore ne les strippe PAS (le motif `*.png` est ancré à la
# racine du contexte de build, il ne traverse pas les sous-dossiers ; cf.
# .dockerignore). Source globale à l'app (pas par-titre).
COPY docs /app/docs

# Front compilé du moteur (sans titles/, qui sont déjà en place dans le paquet ci-dessus).
COPY --from=web-builder /build/engine-web /app/apps/web/dist

# Scripts d'exploitation (backfill, seed, etc.)
COPY scripts /app/scripts

# Binaire Go + CLI : le serveur en dernier, c'est lui qui change à chaque déploiement.
# CLI levelup dans le PATH (seed/seed-demo/backfill via `levelup <cmd>`).
COPY --from=go-builder /build/levelup        /usr/local/bin/levelup
# Backfill objectifs (ops post-deploy v7.2, serveur arrêté pendant l'opération).
COPY --from=go-builder /build/backfill_objective_stats /usr/local/bin/backfill_objective_stats
# Dates des jalons (ops post-deploy v7.5, serveur arrêté pendant l'opération).
COPY --from=go-builder /build/backfill_milestone_dates /usr/local/bin/backfill_milestone_dates
COPY --from=go-builder /build/levelup-server /app/levelup-server

# Garde de construction : le manifeste de l'image et le binaire doivent porter la MÊME empreinte (le serveur le
# vérifie aussi au démarrage, mais un échec ici coûte une construction, pas un déploiement).
RUN fp="$(sed -n 's/.*"fingerprint": *"\([0-9a-f]*\)".*/\1/p' /app/resources.manifest.json | head -n 1)" \
 && test -n "$fp" \
 && grep -a -q "main.resourcesFingerprint=$fp" /app/levelup-server

# Stubs de config — écrasés au runtime par les volumes bind-mount
RUN echo '{"version":"2.1","warehouse_path":"data/warehouse","profiles":{}}' > /app/db_profiles.json \
    && echo '{}' > /app/app_settings.json

# Dossiers attendus par le runtime
RUN mkdir -p /app/data/players /app/data/warehouse /app/data/logs /app/data/cache /app/data/sessions

# --- Utilisateur non-root ---
# UID 1000 = même UID que le user "deploy" sur le VPS.
RUN adduser --disabled-password --gecos "" --uid 1000 appuser \
    && chown -R appuser:appuser /app

COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# PAS de USER appuser ici — l'entrypoint démarre root, drop vers appuser après chown

EXPOSE 8000

# Variables d'environnement par défaut (runtime Go)
# LEVELUP_API_HOST=0.0.0.0 : indispensable en conteneur — sans ça le serveur bind
# 127.0.0.1 (défaut dev) et le port publié + le reverse proxy ne l'atteignent pas.
# Les trois racines (ADR 0008, amendement « two roots ») : ressources livrées avec l'image
# (config/, static/, build web), données (volume ./data), journaux (sous le volume : ils survivent
# aux recréations du conteneur). Les anciennes LEVELUP_ROOT et LEVELUP_DATA, que rien ne lisait,
# sont retirées. LEVELUP_DATA_DIR posé = db_profiles.json, app_settings.json et .env.local se
# cherchent en tête de /app/data : le service levelup du compose garde ses fichiers montés sous
# /app par LEVELUP_DB_PROFILES et LEVELUP_APP_SETTINGS.
ENV LEVELUP_RESOURCES_DIR=/app \
    LEVELUP_DATA_DIR=/app/data \
    LEVELUP_LOGS_DIR=/app/data/logs \
    LEVELUP_WEB_DIST=/app/apps/web/dist \
    LEVELUP_API_HOST=0.0.0.0 \
    LEVELUP_LOG_JSON=true

# Healthcheck Go (endpoint natif)
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/app/levelup-server", "-health-check"]

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["/app/levelup-server"]

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

# ============================================================================
# Stage 2 — Build Go (CGo activé pour DuckDB bindings)
# ============================================================================
FROM golang:1.26-bookworm AS go-builder

WORKDIR /build/go

# Dépendances Go (cache Docker — séparé du code source)
COPY apps/go-api/go.mod apps/go-api/go.sum ./

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

# Code source Go
COPY apps/go-api/ ./

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
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-s -w -extldflags '-static'" \
    -o /build/levelup-server \
    ./cmd/server/

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

# Binaire Go + assets web
COPY --from=go-builder /build/levelup-server /app/levelup-server
# CLI levelup dans le PATH (seed/seed-demo/backfill via `levelup <cmd>`).
COPY --from=go-builder /build/levelup        /usr/local/bin/levelup
# Backfill objectifs (ops post-deploy v7.2, serveur arrêté pendant l'opération).
COPY --from=go-builder /build/backfill_objective_stats /usr/local/bin/backfill_objective_stats
COPY --from=web-builder /build/web/dist       /app/apps/web/dist

# Scripts d'exploitation (backfill, seed, etc.)
COPY scripts /app/scripts

# Config title-aware (mappings/fields.toml, assets, outcomes, capabilities).
# Lue au boot par fieldMappingsRegistry.LoadFromConfigDir + RankCatalog. Sans ce
# COPY, config/titles/ n'existe pas dans le conteneur → field_mappings_load_warning,
# semantic adapter + RankCatalog jamais chargés → rangs carrière en EN (prod+demo)
# et libellés métier non title-aware. ~117 KB.
COPY config /app/config

# Assets statiques UI servis sous /static/ par le serveur Go (maps, ranks,
# medals, commendations, prestige). ~23 MB. Sans ce COPY, /app/static n'existe
# pas dans le conteneur → 404 sur tout /static/* (prod + demo).
COPY static /app/static

# Docs (RELEASE_NOTES.md, CHANGELOG.md, FR/) — lus au runtime par
# release_notes_service.go (docs/FR/RELEASE_NOTES.md) et changelog.go
# (docs/CHANGELOG.md). Sans ce COPY, /app/docs n'existe pas dans le conteneur →
# la page "Notes de version" (/help) renvoie 500 et /changelog 404 (les deux
# titres). Note : docs/ embarque aussi les captures d'écran (docs/screenshots/,
# ~37 PNG) — .dockerignore ne les strippe PAS (le motif `*.png` est ancré à la
# racine du contexte de build, il ne traverse pas les sous-dossiers ; cf.
# .dockerignore). Source globale à l'app (pas par-titre).
COPY docs /app/docs

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
ENV LEVELUP_ROOT=/app \
    LEVELUP_DATA=/app/data \
    LEVELUP_WEB_DIST=/app/apps/web/dist \
    LEVELUP_API_HOST=0.0.0.0 \
    LEVELUP_LOG_JSON=true

# Healthcheck Go (endpoint natif)
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/app/levelup-server", "-health-check"]

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["/app/levelup-server"]

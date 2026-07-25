# ============================================================================
# Stage 1 — Build React (Vite)
# ============================================================================
FROM node:22-slim AS web-builder

WORKDIR /build/web

# Cache npm : installer les dépendances séparément du code source
# .npmrc : conserve `legacy-peer-deps=true` (workaround TS6 vs
# openapi-typescript@7.x peer dep — cf. apps/web/.npmrc).
COPY apps/web/package.json apps/web/package-lock.json apps/web/.npmrc ./
RUN npm ci --prefer-offline

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

# ============================================================================
# Stage 2 — Build Go (CGo activé pour DuckDB bindings)
# ============================================================================
FROM golang:1.26-bookworm AS go-builder

WORKDIR /build/go

# Dépendances Go (cache Docker — séparé du code source)
COPY apps/go-api/go.mod apps/go-api/go.sum ./
RUN go mod download

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
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-extldflags '-static'" \
    -o /build/levelup-server \
    ./cmd/server/

# CLI levelup (seed, seed-demo, backfill…) — requis par le job deploy-demo de
# la CI (docker compose run levelup `levelup` seed-demo). Sans ce binaire, la
# regen démo échoue avec "executable file not found in $PATH".
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-extldflags '-static'" \
    -o /build/levelup \
    ./cmd/levelup/

# CLI backfill_objective_stats — ops prod v7.2 : backfill historique des stats
# d'objectifs (re-téléchargement API, serveur ARRÊTÉ pendant l'opération car il
# prend le lock RW de la shared DB). Binaire dédié hors CLI levelup.
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-extldflags '-static'" \
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
# titres). Les images de docs/ sont strippées par .dockerignore (*.png/*.jpg) ;
# seuls les .md (légers) sont embarqués. Source globale à l'app (pas par-titre).
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

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

# Lire la version depuis VERSION (injectée via ldflags)
ARG VERSION=dev
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -extldflags '-static'" \
    -o /build/levelup-server \
    ./cmd/server/

# ============================================================================
# Stage 3 — Runtime minimal (Debian slim)
# ============================================================================
FROM debian:bookworm-slim

# gosu : switch user non-root (même pattern que l'ancienne image Python)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gosu \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Binaire Go + assets web
COPY --from=go-builder /build/levelup-server /app/levelup-server
COPY --from=web-builder /build/web/dist       /app/apps/web/dist

# Scripts d'exploitation (backfill, seed, etc.)
COPY scripts /app/scripts

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
ENV LEVELUP_ROOT=/app \
    LEVELUP_DATA=/app/data \
    LEVELUP_WEB_DIST=/app/apps/web/dist \
    LEVELUP_LOG_JSON=true

# Healthcheck Go (endpoint natif)
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/app/levelup-server", "-health-check"]

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["/app/levelup-server"]

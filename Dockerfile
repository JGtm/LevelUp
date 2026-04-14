# ============================================================================
# Stage 1 — Build React (Vite)
# ============================================================================
FROM node:22-slim AS web-builder

WORKDIR /build/web

# Cache npm : installer les dépendances séparément du code source
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci --prefer-offline

# Code source
COPY apps/web/ ./

# Build Vite (output : /build/web/dist)
RUN npm run build

# ============================================================================
# Stage 2 — Runtime Python + FastAPI
# ============================================================================
FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

# --- Étape 0 : Paquets système ---
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    gosu \
    && rm -rf /var/lib/apt/lists/*

# --- Étape 1 : Dépendances Python (cache Docker maximisé) ---
COPY pyproject.toml /app/
RUN mkdir -p /app/src && touch /app/src/__init__.py

RUN python -m pip install --no-cache-dir --upgrade pip \
    && python -m pip install --no-cache-dir -e ".[spnkr,api]"

# --- Étape 2 : Code Python ---
COPY src /app/src
COPY apps/api /app/apps/api
COPY scripts /app/scripts
COPY launcher.py /app/

# --- Étape 3 : Assets React (build Vite depuis stage 1) ---
COPY --from=web-builder /build/web/dist /app/apps/web/dist

# Stubs de config — écrasés au runtime par les volumes bind-mount
RUN echo '{"version":"2.1","warehouse_path":"data/warehouse","profiles":{}}' > /app/db_profiles.json \
    && echo '{}' > /app/app_settings.json

# Dossiers attendus par le runtime
RUN mkdir -p /app/data/players /app/data/warehouse /app/data/logs /app/data/cache

# --- Étape 4 : Utilisateur non-root ---
# UID 1000 = même UID que le user "deploy" sur le VPS.
RUN adduser --disabled-password --gecos "" --uid 1000 appuser \
    && chown -R appuser:appuser /app

COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# PAS de USER appuser ici — l'entrypoint démarre root, drop vers appuser après chown

EXPOSE 8000

# Variables d'environnement par défaut
ENV LEVELUP_DB="" \
    LEVELUP_ROOT=/app \
    LEVELUP_DATA=/app/data \
    LEVELUP_WEB_DIST=/app/apps/web/dist

# Healthcheck FastAPI
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD ["python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8000/api/v1/health').read()"]

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["python", "-m", "uvicorn", "apps.api.app.main:app", \
     "--host", "0.0.0.0", "--port", "8000", \
     "--proxy-headers", "--forwarded-allow-ips=*"]

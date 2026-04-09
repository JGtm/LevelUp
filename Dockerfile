FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

# --- Étape 0 : Paquets système ---
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# --- Étape 1 : Dépendances (cache Docker maximisé) ---
# On copie pyproject.toml en premier pour ne pas
# réinstaller à chaque changement de code source.
COPY pyproject.toml /app/
# setup.py stub minimal pour que pip install -e fonctionne sans le code src/
RUN mkdir -p /app/src && touch /app/src/__init__.py

RUN python -m pip install --no-cache-dir --upgrade pip \
    && python -m pip install --no-cache-dir -e ".[spnkr]"

# --- Étape 2 : Code et assets ---
COPY src /app/src
COPY static /app/static
COPY scripts /app/scripts
COPY .streamlit /app/.streamlit
COPY streamlit_app.py launcher.py /app/

# Données de référence embarquées (petits fichiers nécessaires à l'UI)
# Les playlists/modes sont dans metadata.duckdb (tables playlist_translations, mode_translations)

# Stubs de config — écrasés au runtime par les volumes bind-mount
# db_profiles.json est gitignored, on génère un stub valide pour le build
RUN echo '{"version":"2.1","warehouse_path":"data/warehouse","profiles":{}}' > /app/db_profiles.json \
    && echo '{}' > /app/app_settings.json

# Dossiers attendus par le runtime
RUN mkdir -p /app/data/players /app/data/warehouse /app/data/logs /app/data/cache

# --- Étape 3 : Utilisateur non-root ---
RUN adduser --disabled-password --gecos "" --uid 10001 appuser \
    && chown -R appuser:appuser /app

USER appuser

EXPOSE 8501

# Variables d'environnement par défaut
# LEVELUP_DB : optionnel, force un chemin DB précis
# LEVELUP_ROOT : indique la racine du projet au runtime
# LEVELUP_DATA : obligatoire en conteneur (pas de .venv → _is_dev_mode()=False)
ENV LEVELUP_DB="" \
    LEVELUP_ROOT=/app \
    LEVELUP_DATA=/app/data

# Healthcheck Streamlit (endpoint officiel /_stcore/health)
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
    CMD ["python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8501/_stcore/health').read()"]

CMD ["python", "-m", "streamlit", "run", "streamlit_app.py", \
     "--server.address=0.0.0.0", "--server.port=8501", "--server.headless=true"]

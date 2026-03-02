# Doppler — Guide d'intégration LevelUp

## Contexte

Les secrets LevelUp (tokens OAuth, webhook Discord) sont gérés via :
1. **Doppler** (recommandé pour la prod / partage d'équipe)
2. **`.env.local`** (fallback local — ne pas committer)

## Secrets à migrer

| Variable | Source actuelle | Description |
|----------|----------------|-------------|
| `SPNKR_AZURE_CLIENT_ID` | `.env.local` | Client ID Azure |
| `SPNKR_AZURE_CLIENT_SECRET` | `.env.local` | Client Secret Azure |
| `SPNKR_AZURE_REDIRECT_URI` | `.env.local` | Redirect URI OAuth |
| `SPNKR_OAUTH_REFRESH_TOKEN_JGTM` | `.env.local` | Refresh token joueur |
| `DISCORD_WEBHOOK_URL` | `.env.local` | Webhook Discord |

## Setup Doppler

### 1. Installer le CLI
```bash
# Windows (via winget)
winget install Doppler.doppler

# ou via scoop
scoop install doppler
```

### 2. S'authentifier
```bash
doppler login
```

### 3. Créer le projet
```bash
doppler projects create levelup
doppler setup  # sélectionner le projet "levelup" et la config "dev"
```

### 4. Importer les secrets
```bash
# Importer depuis .env.local
doppler secrets upload .env.local
```

### 5. Activer dans LevelUp
Éditer `app_settings.json` :
```json
{
  "doppler_enabled": true,
  "doppler_project": "levelup",
  "doppler_config": "dev"
}
```

## Comment ça marche

Au démarrage de l'app (via `streamlit_app.py`), si `doppler_enabled=true` :
1. `src/utils/secrets.py::load_doppler_secrets_to_env()` est appelé
2. Il exécute `doppler secrets download --format env --no-file`
3. Les secrets sont injectés dans `os.environ`
4. Tous les modules existants (`discord_notifier.py`, `profile_api_tokens.py`, etc.)
   continuent de lire `os.environ.get(...)` sans modification

Si Doppler est indisponible → fallback automatique sur `.env.local`.

## Lire un secret manuellement

```python
from src.utils.secrets import get_secret
webhook_url = get_secret("DISCORD_WEBHOOK_URL")
```

## Commandes utiles

```bash
# Voir les secrets
doppler secrets

# Ajouter un secret
doppler secrets set MON_SECRET=valeur

# Vérifier l'intégration
doppler run -- python -c "import os; print(os.environ.get('DISCORD_WEBHOOK_URL'))"
```

# Configuration Guide — LevelUp

French version: [FR/CONFIGURATION.md](FR/CONFIGURATION.md)

This doc explains how to configure SPNKr (Halo Infinite API) credentials and LevelUp settings.

## 1) Create `.env.local`

```bash
cp .env.example .env.local
```

## 2) Set SPNKr / Azure credentials

Edit `.env.local` and set:

```env
SPNKR_AZURE_CLIENT_ID=your_client_id
SPNKR_AZURE_CLIENT_SECRET=your_client_secret
SPNKR_AZURE_REDIRECT_URI=https://localhost
SPNKR_OAUTH_REFRESH_TOKEN=your_refresh_token
```

Generate / refresh the token using:

```bash
python scripts/spnkr_get_refresh_token.py
```

## 3) Player profiles (`db_profiles.json`)

LevelUp reads player profiles from `db_profiles.json` at the repo root.

If you need to add a player, add an entry with at least `gamertag`, `xuid`, and `db_path`.

## 4) App settings (`app_settings.json`)

App-level defaults live in `app_settings.json` at the repo root.

## Security notes

- Never commit `.env.local` (tokens). It must stay local.

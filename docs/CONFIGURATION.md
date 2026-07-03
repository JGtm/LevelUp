# Configuration Guide — LevelUp

French version: [FR/CONFIGURATION.md](FR/CONFIGURATION.md)

> Complete configuration guide for the Go backend (`apps/go-api`): player profiles
> (`db_profiles.json`), application settings (`app_settings.json`), environment
> variables, and the single source of auth tokens.

## Table of Contents

- [Player Profiles](#player-profiles)
- [Token Storage & Onboarding](#token-storage--onboarding)
- [Environment Variables](#environment-variables)
- [Application Settings](#application-settings)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

---

## Player Profiles

### File structure (`db_profiles.json`)

LevelUp reads player profiles from `db_profiles.json` at the repo root
(override the path with `LEVELUP_DB_PROFILES`). Since the multi-title refactor
(ADR 0008), the file is **v3**: profiles are grouped under a title slug
(`halo_infinite`, `halo_5`, …). The `xuid` is global cross-title — the same
player keeps the same XUID across every title section.

```json
{
  "version": "3.0",
  "profiles": {
    "halo_infinite": {
      "MyGamertag": {
        "db_path": "data/titles/halo_infinite/players/MyGamertag/stats.duckdb",
        "xuid": "2533274823110022",
        "waypoint_player": "MyGamertag"
      },
      "OtherPlayer": {
        "db_path": "",
        "xuid": "2535413181053876",
        "waypoint_player": "OtherPlayer",
        "auth_only": true
      }
    }
  }
}
```

### Entry properties

| Property | Type | Required | Description |
|----------|------|:--------:|-------------|
| `xuid` | string | Yes | Global Xbox identifier (16 digits). Required to address the token store and to call the Halo API (`/matches` needs `xuid(NNN)`, never the gamertag). |
| `db_path` | string | No | Path to the player's DuckDB enrichment DB (`data/titles/<slug>/players/<gamertag>/stats.duckdb`). Empty for `auth_only` profiles. |
| `waypoint_player` | string | No | Gamertag used for Halo Waypoint player-gated lookups. |
| `sync_enabled` | bool | No | `null`/`true` = active; `false` = sync paused (data kept). |
| `initial_max_matches` | int | No | Matches requested at onboarding (`0` = default). |
| `auth_only` | bool | No | Profile that exists only to hold auth tokens (no player DB, not synced). |

Unknown keys are preserved on round-trip — `db_profiles.json` has a single
writer (`internal/platform/dbprofiles`), so no field is silently dropped.

> The profile key is the gamertag. The `xuid` **must** be present before you can
> capture a token for that player (see below).

### Adding a new title

To scaffold the directory tree and a `db_profiles.json` section for a new game
title, use the Go CLI:

```bash
go run ./apps/go-api/cmd/levelup add-title --name "Halo MCC" --capabilities matchmaking,media
```

This creates `data/titles/<slug>/...`, an empty title section in
`db_profiles.json`, and prints the Go snippet to register the
`TitleDescriptor` in `registry.go` (the only remaining manual step).

### Finding your XUID

- Via the dashboard once a sync has run (resolved from the API).
- Via third-party sites (xboxgamertag.com, etc.).

---

## Token Storage & Onboarding

### Single source of truth (ADR 0023)

Auth tokens (the Microsoft OAuth refresh token + the serialized MSAL cache)
live in **one** place: the `MultiUserTokenStore`, one JSON file per user keyed
by XUID.

- Runtime path (title-namespaced): `data/titles/halo_infinite/auth/watcher_tokens/{xuid}.json`
- Legacy global path (read-only, copy-migrated at boot): `data/auth/watcher_tokens/{xuid}.json`
- The auth root is overridable via `LEVELUP_AUTH_DIR` (default `data/auth`).

Writes are atomic (write-to-temp + `os.Rename`), files are `0600`, the
directory is `0700`, and the XUID is validated against path traversal.

> Tokens are **not** stored in `stats.duckdb` / `sync_meta`, and `.env.local`
> is **not** a credential store. Any older documentation saying otherwise is
> obsolete (see ADR 0023). DuckDB and env-var refresh tokens are tolerated only
> as a transitional read fallback (warn-logged) and are copy-migrated into the
> store at first boot.

Each `{xuid}.json` holds the canonical `UserTokens` fields: `OAuthRefreshToken`
(raw Microsoft OAuth v2 refresh token), `MSALCacheJSON`, the derived
XSTS/Spartan tokens, and their expiry timestamps.

### Mode 1 — Xbox SSO (normal)

1. The user clicks "Sign in with Xbox" in the dashboard.
2. The Microsoft OAuth flow returns to `/auth/xbox/callback`.
3. The callback persists the refresh token to the store automatically.

No `.env.local` editing required. Requires the OAuth client to be configured
(`LEVELUP_OAUTH_CLIENT_ID` / `SPNKR_AZURE_CLIENT_SECRET`,
`LEVELUP_OAUTH_REDIRECT_URI`).

### Mode 2 — `token-capture` (advanced, Device Code)

```bash
cd apps/go-api && go run ./cmd/token-capture/ MyGamertag
```

Resolves the XUID from `db_profiles.json`, runs a Microsoft Device Code Flow
(prints a URL + short code), polls until the user authenticates, then writes
the refresh token directly into the store and invalidates the in-process token
cache. Restart the server and it works immediately.

### Mode 3 — `token-import` (advanced, RT from elsewhere)

```bash
cat token-mygt.txt | (cd apps/go-api && go run ./cmd/token-import/ MyGamertag)
```

Reads the refresh token from **stdin** (never argv, to keep it out of shell
history / `ps`) and writes it directly into the store.

### Common prerequisite

The player must already be declared in `db_profiles.json` **with its `xuid`**
before `token-capture` / `token-import` — without the XUID the store cannot
address the entry.

> After an external rotation injects a new RT, the in-process Halo token cache
> (~50 min) is invalidated for that XUID (`halo.InvalidateCachedPlayerTokens`),
> so the server re-derives Spartan tokens from the fresh chain. `token-capture`
> and `token-import` do this automatically.

---

## Environment Variables

### Configuration files

| File | Usage | Git |
|------|-------|-----|
| `.env.local` | Local overrides loaded into the process env at boot (idempotent: never overrides an already-set var) | Ignored |
| `.env` | Alternative to `.env.local` | Ignored |

`.env.local` is loaded from the repo root (resolved via `LEVELUP_REPO_ROOT` or
auto-detection) before any `os.Getenv` read.

### OAuth / Azure

| Variable | Description | Required |
|----------|-------------|:--------:|
| `LEVELUP_OAUTH_CLIENT_ID` | Azure OAuth client ID for SSO web + refresh + `token-capture`. Defaults to the bundled canonical app ID if unset. | No |
| `SPNKR_AZURE_CLIENT_SECRET` | Client secret for the canonical app (its redirect is a Web platform → confidential client → Microsoft requires the secret for the Authorization Code Flow). | For SSO |
| `LEVELUP_OAUTH_REDIRECT_URI` | Redirect URI for `/auth/xbox/login`. If empty, that route returns 500. | For SSO |

> The three OAuth paths (SSO web, server refresh, `token-capture`) read the same
> `LEVELUP_OAUTH_CLIENT_ID` so a captured refresh token always matches the
> client that will refresh it (a refresh token is bound to its issuing client).

### Server / runtime

| Variable | Description | Default |
|----------|-------------|---------|
| `LEVELUP_REPO_ROOT` | Repo root (resolves `db_profiles.json`, `app_settings.json`, `data/`). | Auto-detected |
| `LEVELUP_API_HOST` | HTTP bind host. | `127.0.0.1` |
| `LEVELUP_API_PORT` | HTTP listen port. | `8000` |
| `LEVELUP_DB_PROFILES` | Path to `db_profiles.json`. | `<root>/db_profiles.json` |
| `LEVELUP_APP_SETTINGS` | Path to `app_settings.json`. | `<root>/app_settings.json` |
| `LEVELUP_AUTH_DIR` | Auth data root (token store, sessions). | `<root>/data/auth` |
| `LEVELUP_SESSION_DIR` | Session store directory. | `<root>/data/sessions` |
| `LEVELUP_SESSION_SECRET` | Session signing secret. Must be overridden in production. | `CHANGE_ME_IN_PRODUCTION` |
| `LEVELUP_ENV` | `production` enables prod hardening (HTTPS-only cookies, etc.). | dev |
| `LEVELUP_CORS_ORIGINS` | Comma-separated allowed CORS origins. | (none) |
| `LEVELUP_AUTH_MODE` | Auth mode. | `none` |
| `LEVELUP_REGISTRATION` | Registration mode. | `invite` |
| `LEVELUP_COOKIE_SECURE` | Force/disable the `Secure` cookie flag. | auto |
| `LEVELUP_TRUST_PROXY_HEADERS` | Trust `X-Forwarded-*` (behind a reverse proxy). | `false` |
| `LEVELUP_INSTANCE_LOCKED` | Lock the instance to existing users. | `false` |
| `LEVELUP_RATE_LIMIT_RPM` | HTTP rate limit (requests/minute). | built-in default |
| `LEVELUP_WEB_DIST` | Path to the built frontend (`apps/web/dist`), set by the Docker image. | (none) |
| `LEVELUP_DEMO_MODE` | `true` enables demo mode. | `false` |
| `LEVELUP_LANG` | Default UI/CLI language. | `fr` |
| `LEVELUP_APP_VERSION` | Reported app version. | `dev` |
| `LEVELUP_USE_SHARED_PROVIDER` | Enable the SharedDBProvider RO↔RW swap (ADR 0016). | (off) |

### Logging

| Variable | Description | Default |
|----------|-------------|---------|
| `LEVELUP_LOG_LEVEL` | Log level (`debug`/`info`/`warn`/`error`). | `info` |
| `LEVELUP_LOG_FORMAT` | Console format. | text |
| `LEVELUP_LOGS_DIR` | Per-category log file directory. | `<root>/logs` |
| `LEVELUP_LOGS_ENABLED` | Set `false` to disable file logging. | enabled |

### Sync / feature flags

| Variable | Description | Default |
|----------|-------------|---------|
| `LEVELUP_PERSIST_BATCH_ASYNC` | Run the batch persister asynchronously (WAL + worker). | (off) |
| `LEVELUP_SYNC_PIPELINE` | Select the sync pipeline. | built-in |
| `LEVELUP_CSR_SEASON_ID` | Override CSR season id. | from `app_settings.json` |
| `MULTI_TITLE_API_ENABLED` | Expose the multi-title field-mappings/preview routes (override of `app_settings.json`). | `false` |
| `PRESTIGE_ENABLED` | Enable the Prestige module (override of `app_settings.json`). | `true` |

### Integrations

| Variable | Description |
|----------|-------------|
| `LEVELUP_DISCORD_WEBHOOK_URL` | Discord webhook (preferred over `DISCORD_WEBHOOK_URL` and over `app_settings.json:discord_webhook_url`). |
| `DISCORD_WEBHOOK_URL` | Discord webhook (legacy name, still read). |
| `STEAM_API_KEY` | Steam Web API key (Steam presence). |
| `RESTIC_REPOSITORY` / `RESTIC_PASSWORD` / `RESTIC_PASSWORD_FILE` | Restic backup target/credentials. |
| `LEVELUP_BACKUP_DIR` | Local backup directory. |

> Legacy `SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>` env vars are still read as a
> transitional fallback only (warn-logged, migrated into the token store at
> boot). Do not rely on them for new setups — use `token-capture` /
> `token-import` instead.

---

## Application Settings

### File `app_settings.json`

Copy the template and customize (repo root, or `LEVELUP_APP_SETTINGS`):

```bash
cp app_settings.example.json app_settings.json
```

### Available settings

Keys read by the Go backend from `app_settings.json` (some are not in the example template):

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `media_enabled` | bool | `false` | Enable media (Xbox captures) integration. |
| `media_captures_base_dir` | string | `""` | Path to the Xbox captures folder. |
| `media_buffer_minutes` | int | `1` | Tolerance window matching captures to games. |
| `media_watcher_enabled` | bool | `false` | Enable the media folder watcher. |
| `refresh_clears_caches` | bool | `false` | Clear caches on manual refresh. |
| `spnkr_refresh_with_backfill` | bool | `false` | Run backfill during sync. |
| `spnkr_refresh_backfill_medals` | bool | `false` | Backfill medals. |
| `spnkr_refresh_backfill_events` | bool | `false` | Backfill highlight events. |
| `spnkr_refresh_backfill_skill` | bool | `false` | Backfill MMR/skill. |
| `spnkr_refresh_backfill_personal_scores` | bool | `false` | Backfill personal scores. |
| `spnkr_refresh_backfill_performance_scores` | bool | `true` | Compute performance scores. |
| `spnkr_refresh_backfill_aliases` | bool | `false` | Backfill XUID→gamertag aliases. |
| `spnkr_refresh_backfill_lusr` | bool | `true` | Backfill LUSR skill rating. |
| `lang` | string | `"fr"` | UI language (`fr`, `en`). |
| `discord_lang` | string | `"fr"` | Discord notification language. |
| `discord_notifications_enabled` | bool | `false` | Enable Discord sync notifications. |
| `discord_notify_new_media` | bool | `true` | Notify on new media. |
| `discord_webhook_url` | string | `""` | Discord webhook URL (env vars take precedence). |
| `tailscale_enabled` | bool | `false` | Enable Tailscale Funnel remote access. |
| `user_timezone` | string | `"Europe/Paris"` | IANA timezone for display. |
| `watcher_presence_enabled` | bool | `true` | Enable presence watcher. |
| `career_top_exclude_btb` | bool | `false` | Exclude Big Team Battle from career tops. |
| `csr_season_id` | string | `"CsrSeason13-1"` | CSR season id (overridable via `LEVELUP_CSR_SEASON_ID`). |
| `backup_enabled` | bool | `false` | Enable scheduled DuckDB backups. |
| `backup_interval` | string | `"6h"` | Backup interval (Go duration). |
| `backup_keep_daily` | int | `7` | Daily backups retained. |
| `backup_keep_weekly` | int | `4` | Weekly backups retained. |
| `backup_keep_monthly` | int | `12` | Monthly backups retained. |
| `multi_title_api_enabled` | bool | `false` | Expose multi-title API routes (overridable via `MULTI_TITLE_API_ENABLED`). |
| `prestige_enabled` | bool | `true` | Enable Prestige module (overridable via `PRESTIGE_ENABLED`). |
| `instance_locked` | bool | `false` | Lock the instance to existing users (also via `LEVELUP_INSTANCE_LOCKED`). |

---

## Security

### Never commit these

- `.env.local` / `.env`
- `data/auth/` and `data/titles/*/auth/` (token store files)
- Any file containing tokens or secrets

They are already covered by `.gitignore`.

### Token rotation

Microsoft refresh tokens rotate on each refresh and expire after ~90 days of
inactivity. To re-provision a player, re-run:

```bash
cd apps/go-api && go run ./cmd/token-capture/ MyGamertag
```

Never re-capture a healthy token unnecessarily: the store is the source of
truth and rotation is persisted automatically by the server and the CLIs.

### Production

In production set `LEVELUP_ENV=production` and override `LEVELUP_SESSION_SECRET`.
The runtime opens `metadata.duckdb` and `shared_matches_v2.duckdb` read-only.

---

## Troubleshooting

### `invalid_grant` / `AADSTS70000`

The refresh token was already consumed or belongs to a stale store entry.
Re-provision the affected player:

```bash
cd apps/go-api && go run ./cmd/token-capture/ MyGamertag
```

If only some XUIDs fail with `AADSTS70000`, they are likely stale store entries
from an old app registration — they should be ignored/blacklisted, not
re-captured.

### `AADSTS90023` on refresh

A refresh token issued by a public (device-code) flow rejects the client
secret. The refresh path retries automatically without the secret — keep
`SPNKR_AZURE_CLIENT_SECRET` configured for the SSO (confidential) flow.

### Device code expired

The code is valid ~15 minutes. Re-run `token-capture` and complete sign-in
promptly.

### Career rank / customization 403

Player-gated Waypoint endpoints require that player's own token in the store.
Run `token-capture` for that gamertag so the store has its refresh token.

### Database not found

Check `db_path` in `db_profiles.json`. The Go CLI creates the title tree:

```bash
go run ./apps/go-api/cmd/levelup add-title --name "<Title>"
```

### Verify configuration

```bash
go run ./apps/go-api/cmd/levelup check-env
```

# Guide de Configuration — LevelUp

Version anglaise : [../CONFIGURATION.md](../CONFIGURATION.md)

> Guide complet de configuration du backend Go (`apps/go-api`) : profils joueurs
> (`db_profiles.json`), paramètres applicatifs (`app_settings.json`), variables
> d'environnement et source unique des tokens d'authentification.

## Table des Matières

- [Profils joueurs](#profils-joueurs)
- [Stockage des tokens & onboarding](#stockage-des-tokens--onboarding)
- [Variables d'environnement](#variables-denvironnement)
- [Paramètres applicatifs](#paramètres-applicatifs)
- [Sécurité](#sécurité)
- [Dépannage](#dépannage)

---

## Profils joueurs

### Structure du fichier (`db_profiles.json`)

LevelUp lit les profils joueurs depuis `db_profiles.json` à la racine du repo
(chemin surchargeable via `LEVELUP_DB_PROFILES`). Depuis la refonte multi-titres
(ADR 0008), le fichier est en **v3** : les profils sont groupés sous un slug de
titre (`halo_infinite`, `halo_5`, …). Le `xuid` est global cross-titres — un même
joueur garde le même XUID dans chaque section de titre.

```json
{
  "version": "3.0",
  "profiles": {
    "halo_infinite": {
      "MonGamertag": {
        "db_path": "data/titles/halo_infinite/players/MonGamertag/stats.duckdb",
        "xuid": "2533274823110022",
        "waypoint_player": "MonGamertag"
      },
      "AutreJoueur": {
        "db_path": "",
        "xuid": "2535413181053876",
        "waypoint_player": "AutreJoueur",
        "auth_only": true
      }
    }
  }
}
```

### Propriétés d'une entrée

| Propriété | Type | Requis | Description |
|-----------|------|:------:|-------------|
| `xuid` | string | Oui | Identifiant Xbox global (16 chiffres). Requis pour adresser le store de tokens et appeler l'API Halo (`/matches` exige `xuid(NNN)`, jamais le gamertag). |
| `db_path` | string | Non | Chemin de la DB DuckDB d'enrichissement du joueur (`data/titles/<slug>/players/<gamertag>/stats.duckdb`). Vide pour les profils `auth_only`. |
| `waypoint_player` | string | Non | Gamertag utilisé pour les lookups Halo Waypoint player-gated. |
| `sync_enabled` | bool | Non | `null`/`true` = actif ; `false` = sync en pause (données conservées). |
| `initial_max_matches` | int | Non | Matchs demandés à l'onboarding (`0` = défaut). |
| `auth_only` | bool | Non | Profil existant uniquement pour porter les tokens auth (pas de DB joueur, pas de sync). |

Les clés inconnues sont préservées au round-trip — `db_profiles.json` a un writer
unique (`internal/platform/dbprofiles`), aucun champ n'est perdu silencieusement.

> La clé du profil est le gamertag. Le `xuid` **doit** être présent avant de
> pouvoir capturer un token pour ce joueur (voir ci-dessous).

### Ajouter un nouveau titre

Pour créer l'arborescence disque et la section `db_profiles.json` d'un nouveau
titre de jeu, utiliser le CLI Go :

```bash
go run ./apps/go-api/cmd/levelup add-title --name "Halo MCC" --capabilities matchmaking,media
```

Cela crée `data/titles/<slug>/...`, une section de titre vide dans
`db_profiles.json`, et affiche le snippet Go pour enregistrer le
`TitleDescriptor` dans `registry.go` (seule étape manuelle restante).

### Trouver son XUID

- Via le dashboard une fois une première sync effectuée (résolu depuis l'API).
- Via des sites tiers (xboxgamertag.com, etc.).

---

## Stockage des tokens & onboarding

### Source unique de vérité (ADR 0023)

Les tokens d'authentification (refresh token OAuth Microsoft + cache MSAL
sérialisé) vivent à **un seul** endroit : le `MultiUserTokenStore`, un fichier
JSON par utilisateur indexé par XUID.

- Chemin runtime (namespacé titre) : `data/titles/halo_infinite/auth/watcher_tokens/{xuid}.json`
- Chemin legacy global (lecture seule, copy-migré au boot) : `data/auth/watcher_tokens/{xuid}.json`
- La racine auth est surchargeable via `LEVELUP_AUTH_DIR` (défaut `data/auth`).

Les écritures sont atomiques (temp + `os.Rename`), les fichiers en `0600`, le
répertoire en `0700`, et le XUID est validé contre le path traversal.

> Les tokens ne sont **pas** stockés dans `stats.duckdb` / `sync_meta`, et
> `.env.local` n'est **pas** un credential store. Toute ancienne documentation
> affirmant le contraire est obsolète (cf. ADR 0023). Les refresh tokens en
> DuckDB et en variables d'environnement ne sont tolérés qu'en fallback de
> lecture transitoire (warn-loggé) et sont copy-migrés dans le store au premier
> démarrage.

Chaque `{xuid}.json` porte les champs canoniques de `UserTokens` :
`OAuthRefreshToken` (refresh token OAuth v2 brut Microsoft), `MSALCacheJSON`,
les tokens XSTS/Spartan dérivés et leurs dates d'expiration.

### Mode 1 — SSO Xbox (normal)

1. L'utilisateur clique « Se connecter avec Xbox » dans le dashboard.
2. Le flow OAuth Microsoft revient sur `/auth/xbox/callback`.
3. Le callback persiste le refresh token au store automatiquement.

Aucune édition de `.env.local` nécessaire. Requiert le client OAuth configuré
(`LEVELUP_OAUTH_CLIENT_ID` / `SPNKR_AZURE_CLIENT_SECRET`,
`LEVELUP_OAUTH_REDIRECT_URI`).

### Mode 2 — `token-capture` (avancé, Device Code)

```bash
cd apps/go-api && go run ./cmd/token-capture/ MonGamertag
```

Résout le XUID depuis `db_profiles.json`, lance un Device Code Flow Microsoft
(affiche une URL + un code court), poll jusqu'à l'authentification de
l'utilisateur, puis écrit le refresh token directement dans le store et invalide
le cache de tokens en mémoire. Redémarrer le serveur, c'est tout.

### Mode 3 — `token-import` (avancé, RT venu d'ailleurs)

```bash
cat token-mongt.txt | (cd apps/go-api && go run ./cmd/token-import/ MonGamertag)
```

Lit le refresh token sur **stdin** (jamais en argv, pour le tenir hors de
l'historique shell / `ps`) et l'écrit directement dans le store.

### Prérequis commun

Le joueur doit déjà être déclaré dans `db_profiles.json` **avec son `xuid`**
avant `token-capture` / `token-import` — sans le XUID, le store ne peut pas
adresser l'entrée.

> Après une rotation externe injectant un nouveau RT, le cache process des tokens
> Halo (~50 min) est invalidé pour ce XUID (`halo.InvalidateCachedPlayerTokens`),
> pour que le serveur re-dérive les Spartan tokens depuis la chaîne fraîche.
> `token-capture` et `token-import` le font automatiquement.

### Fournisseur de tokens — SISU (seul fournisseur)

Le Device Code Flow d'onboarding est démarré par le fournisseur **SISU** (device-code
natif Xbox, scope MSA `service::user.auth.xboxlive.com::MBI_SSL`, client ID Xbox officiel) :
les self-hosters n'ont besoin d'**aucune inscription Azure**. L'ancien fournisseur MSAL a
été retiré le 2026-07-15 après validation de SISU bout-en-bout ; une valeur héritée
`auth_provider: "msal"` dans `app_settings.json` est ignorée avec un avertissement au
démarrage. Les refresh tokens Azure existants continuent de se rafraîchir via l'endpoint
OAuth v2 ; les refresh tokens SISU natifs passent par `login.live.com` (repli automatique
sur `invalid_grant`).

L'endpoint natif de démarrage est
`https://login.live.com/oauth20_connect.srf` ; un garde-rail réseau opt-in
(`go test -tags=integration ./internal/platform/auth/` avec
`LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK=1`) vérifie qu'il reste joignable — un changement côté
Microsoft se manifeste ainsi par un test en échec plutôt que par un spinner d'onboarding.

---

## Variables d'environnement

### Fichiers de configuration

| Fichier | Usage | Git |
|---------|-------|-----|
| `.env.local` | Surcharges locales chargées dans l'env du process au boot (idempotent : n'écrase jamais une var déjà définie) | Ignoré |
| `.env` | Alternative à `.env.local` | Ignoré |

`.env.local` est chargé depuis la racine du repo (résolue via `LEVELUP_REPO_ROOT`
ou auto-détection) avant toute lecture `os.Getenv`.

### OAuth / Azure

| Variable | Description | Requis |
|----------|-------------|:------:|
| `LEVELUP_OAUTH_CLIENT_ID` | Client ID Azure OAuth pour SSO web + refresh + `token-capture`. Défaut : l'ID de l'app canonique embarquée si absent. | Non |
| `SPNKR_AZURE_CLIENT_SECRET` | Secret client de l'app canonique (son redirect est en plateforme Web → client confidentiel → Microsoft exige le secret pour l'Authorization Code Flow). | Pour SSO |
| `LEVELUP_OAUTH_REDIRECT_URI` | Redirect URI pour `/auth/xbox/login`. Si vide, cette route renvoie 500. | Pour SSO |

> Les trois chemins OAuth (SSO web, refresh serveur, `token-capture`) lisent le
> même `LEVELUP_OAUTH_CLIENT_ID` pour qu'un token capturé matche toujours le
> client qui le rafraîchira (un refresh token est lié à son client émetteur).

### Serveur / runtime

| Variable | Description | Défaut |
|----------|-------------|--------|
| `LEVELUP_REPO_ROOT` | Racine du repo (résout `db_profiles.json`, `app_settings.json`, `data/`). | Auto-détecté |
| `LEVELUP_API_HOST` | Host de bind HTTP. | `127.0.0.1` |
| `LEVELUP_API_PORT` | Port d'écoute HTTP. | `8000` |
| `LEVELUP_DB_PROFILES` | Chemin de `db_profiles.json`. | `<root>/db_profiles.json` |
| `LEVELUP_APP_SETTINGS` | Chemin de `app_settings.json`. | `<root>/app_settings.json` |
| `LEVELUP_AUTH_DIR` | Racine des données auth (store de tokens, sessions). | `<root>/data/auth` |
| `LEVELUP_SESSION_DIR` | Répertoire du store de sessions. | `<root>/data/sessions` |
| `LEVELUP_SESSION_SECRET` | Secret de signature des sessions. À surcharger en production. | `CHANGE_ME_IN_PRODUCTION` |
| `LEVELUP_ENV` | `production` active le durcissement prod (cookies HTTPS-only, etc.). | dev |
| `LEVELUP_CORS_ORIGINS` | Origines CORS autorisées (séparées par virgule). | (aucune) |
| `LEVELUP_AUTH_MODE` | Mode d'authentification. | `none` |
| `LEVELUP_REGISTRATION` | Mode d'inscription. | `invite` |
| `LEVELUP_COOKIE_SECURE` | Force/désactive le flag cookie `Secure`. | auto |
| `LEVELUP_TRUST_PROXY_HEADERS` | Faire confiance aux `X-Forwarded-*` (derrière un reverse proxy). | `false` |
| `LEVELUP_INSTANCE_LOCKED` | Verrouille l'instance aux utilisateurs existants. | `false` |
| `LEVELUP_RATE_LIMIT_RPM` | Rate limit HTTP (requêtes/minute). | défaut interne |
| `LEVELUP_WEB_DIST` | Chemin du frontend buildé (`apps/web/dist`), posé par l'image Docker. | (aucun) |
| `LEVELUP_DEMO_MODE` | `true` active le mode démo. | `false` |
| `LEVELUP_LANG` | Langue UI/CLI par défaut. | `fr` |
| `LEVELUP_APP_VERSION` | Version applicative reportée. | `dev` |
| `LEVELUP_USE_SHARED_PROVIDER` | Active le swap RO↔RW du SharedDBProvider (ADR 0016). | (off) |

### Logging

| Variable | Description | Défaut |
|----------|-------------|--------|
| `LEVELUP_LOG_LEVEL` | Niveau de log (`debug`/`info`/`warn`/`error`). | `info` |
| `LEVELUP_LOG_FORMAT` | Format console. | texte |
| `LEVELUP_LOGS_DIR` | Répertoire des fichiers de log par catégorie. | `<root>/logs` |
| `LEVELUP_LOGS_ENABLED` | Mettre `false` pour désactiver les logs fichiers. | activé |
| `LEVELUP_LOGS_MAX_SIZE_MB` | Taille max de chaque `{catégorie}.log` avant rotation. `0` désactive la rotation (croissance illimitée). | `100` |
| `LEVELUP_LOGS_MAX_BACKUPS` | Archives conservées par catégorie (`{catégorie}.log.1..N`). `0` = aucune. | `3` |

### Sync / feature flags

| Variable | Description | Défaut |
|----------|-------------|--------|
| `LEVELUP_PERSIST_BATCH_ASYNC` | Exécute le persister batch en asynchrone (WAL + worker). Kill-switch : `0` pour revenir au chemin synchrone. Retrait cible >= 2026-Q4. | on |
| `LEVELUP_EVENTS_CONVERGENCE` | Passe de convergence des highlight_events (scheduler + trigger immédiat). Kill-switch : `0` pour désactiver. Retrait cible >= 2026-Q4. | on |
| `LEVELUP_EVENTS_CONVERGENCE_MAX` | Borne le nombre de matchs traités par tick de convergence. | `50` |
| `LEVELUP_CSR_SEASON_ID` | Surcharge l'id de saison CSR. | depuis `app_settings.json` |
| `PRESTIGE_ENABLED` | Active le module Prestige (override de `app_settings.json`). | `true` |

### Intégrations

| Variable | Description |
|----------|-------------|
| `LEVELUP_DISCORD_WEBHOOK_URL` | Webhook Discord (prioritaire sur `DISCORD_WEBHOOK_URL` et sur `app_settings.json:discord_webhook_url`). |
| `DISCORD_WEBHOOK_URL` | Webhook Discord (nom legacy, toujours lu). |
| `LEVELUP_PUBLIC_BASE_URL` | URL publique de base de l'app (optionnelle). Si définie, le relais coach Discord ajoute un lien « Ouvrir dans LevelUp » aux embeds. Vide → pas de lien. |
| `STEAM_API_KEY` | Clé Steam Web API (présence Steam). |
| `RESTIC_REPOSITORY` / `RESTIC_PASSWORD` / `RESTIC_PASSWORD_FILE` | Cible/credentials des backups Restic. |
| `LEVELUP_BACKUP_DIR` | Répertoire de backup local. |

> Les variables legacy `SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>` ne sont PLUS lues à
> l'exécution (ADR 0023 Phase 5, 2026-08-25). Seule la migration one-shot du boot
> les consulte encore, pour recopier une valeur résiduelle dans le store de
> tokens — retrait prévu le 2026-10-01. Pour semer un refresh token : SSO Xbox
> web, `token-capture` ou `token-import`.

---

## Paramètres applicatifs

### Fichier `app_settings.json`

Copier le template et l'adapter (racine du repo, ou `LEVELUP_APP_SETTINGS`) :

```bash
cp app_settings.example.json app_settings.json
```

### Paramètres disponibles

Clés lues par le backend Go depuis `app_settings.json` (certaines absentes du template d'exemple) :

| Paramètre | Type | Défaut | Description |
|-----------|------|--------|-------------|
| `auth_provider` | string | `""` (= `sisu`) | Clé héritée. SISU (natif Xbox, **aucune app Azure**) est le seul fournisseur depuis le retrait de MSAL le 2026-07-15 ; `msal` est ignoré avec un avertissement au démarrage. Voir « Fournisseur de tokens » ci-dessus. |
| `media_enabled` | bool | `false` | Active l'intégration média (captures Xbox). |
| `media_captures_base_dir` | string | `""` | Chemin du dossier de captures Xbox. |
| `media_buffer_minutes` | int | `1` | Fenêtre de tolérance pour associer captures et matchs. |
| `media_watcher_enabled` | bool | `false` | Active le watcher du dossier média. |
| `refresh_clears_caches` | bool | `false` | Vide les caches au refresh manuel. |
| `spnkr_refresh_with_backfill` | bool | `false` | Lance le backfill pendant la sync. |
| `spnkr_refresh_backfill_medals` | bool | `false` | Backfill des médailles. |
| `spnkr_refresh_backfill_events` | bool | `false` | Backfill des highlight events. |
| `spnkr_refresh_backfill_skill` | bool | `false` | Backfill MMR/skill. |
| `spnkr_refresh_backfill_personal_scores` | bool | `false` | Backfill des personal scores. |
| `spnkr_refresh_backfill_performance_scores` | bool | `true` | Calcule les performance scores. |
| `spnkr_refresh_backfill_aliases` | bool | `false` | Backfill des alias XUID→gamertag. |
| `spnkr_refresh_backfill_lusr` | bool | `true` | Backfill du rating LUSR. |
| `lang` | string | `"fr"` | Langue de l'UI (`fr`, `en`). |
| `discord_lang` | string | `"fr"` | Langue des notifications Discord. |
| `discord_notifications_enabled` | bool | `false` | Active les notifications de sync Discord (interrupteur maître de tout le canal Discord). |
| `discord_notify_coach` | bool | `false` | Relaie les proposals coach les plus fortes (signaux de progression) vers le webhook Discord. **OFF par défaut — opt-in** : requiert `discord_notifications_enabled` + un webhook. Émettre vers un service externe est une décision vie privée volontaire, jamais activée par défaut. Catégories relayées = catégories coach uniquement. |
| `discord_notify_new_media` | bool | `true` | Notifie sur nouveau média. |
| `discord_notify_disk` | bool | `true` | Alertes disque (warn > 80 % utilisés ou < 2 Go libres, critical > 90 % ou < 500 Mo) sur le volume data, envoyées au changement de statut + rappel quotidien + rétablissement. |
| `discord_webhook_url` | string | `""` | URL webhook Discord (les vars d'env priment). |
| `tailscale_enabled` | bool | `false` | Active l'accès distant Tailscale Funnel. |
| `user_timezone` | string | `"Europe/Paris"` | Timezone IANA pour l'affichage. |
| `watcher_presence_enabled` | bool | `true` | Active le watcher de présence. |
| `career_top_exclude_btb` | bool | `false` | Exclut le Big Team Battle des tops carrière. |
| `csr_season_id` | string | `"CsrSeason13-1"` | Id de saison CSR (surchargeable via `LEVELUP_CSR_SEASON_ID`). |
| `backup_enabled` | bool | `false` | Active les backups DuckDB planifiés. |
| `backup_interval` | string | `"6h"` | Intervalle de backup (durée Go). |
| `backup_keep_daily` | int | `7` | Backups quotidiens conservés. |
| `backup_keep_weekly` | int | `4` | Backups hebdomadaires conservés. |
| `backup_keep_monthly` | int | `12` | Backups mensuels conservés. |
| `prestige_enabled` | bool | `true` | Active le module Prestige (surchargeable via `PRESTIGE_ENABLED`). |
| `instance_locked` | bool | `false` | Verrouille l'instance aux utilisateurs existants (aussi via `LEVELUP_INSTANCE_LOCKED`). |
| `replay_sound_variation_percent` | int | `100` | Sons d'armes du rejeu 2D : variation du volume et de la hauteur à chaque tir, dans les fourchettes déclarées par le jeu. `100` = fourchettes du jeu telles quelles, `0` = toujours le même fichier. Réglage d'instance, modifié depuis Admin · Système. |
| `replay_sound_distance_percent` | int | `0` | Sons d'armes du rejeu 2D : effet de distance (atténuation + passe-bas). `0` = son pur, aucun nœud dans le chemin du signal. Réglage d'instance, modifié depuis Admin · Système. |

---

## Sécurité

### À ne jamais committer

- `.env.local` / `.env`
- `data/auth/` et `data/titles/*/auth/` (fichiers du store de tokens)
- Tout fichier contenant des tokens ou secrets

Déjà couverts par `.gitignore`.

### Rotation des tokens

Les refresh tokens Microsoft tournent à chaque refresh et expirent après ~90
jours d'inactivité. Pour re-provisionner un joueur, relancer :

```bash
cd apps/go-api && go run ./cmd/token-capture/ MonGamertag
```

Ne jamais re-capturer inutilement un token sain : le store est la source de
vérité et la rotation est persistée automatiquement par le serveur et les CLI.

### Production

En production, poser `LEVELUP_ENV=production` et surcharger
`LEVELUP_SESSION_SECRET`. Le runtime ouvre `metadata.duckdb` et
`shared_matches_v2.duckdb` en lecture seule.

---

## Dépannage

### `invalid_grant` / `AADSTS70000`

Le refresh token a déjà été consommé ou appartient à une entrée périmée du
store. Re-provisionner le joueur concerné :

```bash
cd apps/go-api && go run ./cmd/token-capture/ MonGamertag
```

Si seuls certains XUID échouent en `AADSTS70000`, ce sont probablement des
entrées de store périmées d'une ancienne app — à ignorer/blacklister, surtout
pas à re-capturer.

### `AADSTS90023` au refresh

Un refresh token émis par un flux public (device code) refuse le client secret.
Le chemin de refresh retente automatiquement sans secret — garder
`SPNKR_AZURE_CLIENT_SECRET` configuré pour le flux SSO (confidentiel).

### Device code expiré

Le code est valide ~15 minutes. Relancer `token-capture` et compléter la
connexion rapidement.

### 403 sur career rank / customisation

Les endpoints Waypoint player-gated exigent le token propre du joueur dans le
store. Lancer `token-capture` pour ce gamertag afin que le store ait son refresh
token.

### Base introuvable

Vérifier `db_path` dans `db_profiles.json`. Le CLI Go crée l'arborescence du
titre :

```bash
go run ./apps/go-api/cmd/levelup add-title --name "<Titre>"
```

### Vérifier la configuration

```bash
go run ./apps/go-api/cmd/levelup check-env
```

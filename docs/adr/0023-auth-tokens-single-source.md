# ADR 0023 — MultiUserTokenStore source unique des tokens auth

**Date** : 2026-05-26
**Statut** : Accepté
**Branche** : `refactor/auth-tokens-single-source`

## Contexte

Avant ce refactor, les tokens auth Microsoft (refresh_token OAuth + cache MSAL) vivaient dans **4 sources concurrentes** :

1. **`.env.local`** : `SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>=...` (seed bootstrap manuel)
2. **`sync_meta` DuckDB** (par joueur, table key/value) : clés `oauth_refresh_token` + `msal_token_cache`
3. **`data/auth/watcher_tokens/{xuid}.json`** (`MultiUserTokenStore`) : champ `MSALCacheJSON` (RT pas stocké)
4. **`data/auth/watcher_tokens.json`** (`TokenStore` mono-user legacy) : champ `RefreshToken`

Le code applicatif lisait ces sources avec des priorités incohérentes selon le chemin :

| Chemin | Source 1 | Source 2 | Source 3 |
|--------|---------|---------|---------|
| `registry.refreshTokensFromDB` (HTTP) | env var | DuckDB | — |
| `pool.Discovery.Scan` | DuckDB | env var | watcher store |
| `watcher_refresh.EnsureWatcherAccessToken` | mono-user store | env var | — |
| 4 CLI tools (refresh-metadata, etc.) | varie selon CLI | — | — |
| `auth_xbox_oauth.Callback` | aucune persistance | — | — |

## Incident déclencheur (2026-05-26)

**Madina97294** obtenait `invalid_grant AADSTS70000` immédiatement après avoir injecté un RT frais via `cmd/token-capture` :

1. Air hot-reload déclenchait un boot complet à chaque modif du serveur.
2. Le Pool lisait l'env var de Madina (DuckDB vide), appelait Microsoft, recevait `(access_token, rotated_RT)`, sauvait `rotated_RT` en DuckDB via `onRotated`. **Token original brûlé chez Microsoft.**
3. Millisecondes plus tard, une requête HTTP arrivait → `refreshTokensFromDB` → lisait **l'env var en premier** (l'original déjà consommé) → `invalid_grant`.

JGtm et Chocoboflor n'étaient pas affectés car leurs DuckDB avaient des RT valides maintenus depuis des mois — le Pool lisait DuckDB, pas l'env var.

## Diagnostic architectural

Au-delà du bug Madina, plusieurs défauts structurels :

- **DuckDB OLAP utilisé comme credential store** : anti-pattern. DuckDB est une base analytique, pas un secret manager.
- **4 CLI dupliquaient le pipeline auth** et ne persistaient PAS la rotation → un seul usage par RT, le suivant échoue.
- **`auth_xbox_oauth.Callback`** recevait un RT frais Microsoft et le jetait silencieusement.
- **`MultiUserTokenStore`** existait déjà avec le bon design (fichiers JSON atomiques, perms 0600/0700) mais n'était utilisé que pour le MSAL cache.

## Décision

`MultiUserTokenStore` (`data/auth/watcher_tokens/{xuid}.json`) devient la **source unique** des tokens auth. Toutes les autres sources deviennent legacy à migrer/supprimer.

### Champs canoniques de `UserTokens`

```go
type UserTokens struct {
    XUID              string
    Gamertag          string
    XSTSToken         string
    XSTSUserHash      string
    XSTSExpiresAt     time.Time
    AccessToken       string
    OAuthExpiresAt    time.Time
    OAuthRefreshToken string  // ADR 0023 — refresh_token OAuth v2 brut Microsoft
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

> Phase 5 (2026-08-25) : le champ `MSALCacheJSON` a été supprimé — MSAL n'est plus
> un provider depuis le 2026-07-15, `TrySilentRefresh` était un no-op. La clé JSON
> résiduelle des fichiers prod est ignorée au décodage et disparaît à la première
> rotation.

### Priorité de lecture (ordre canonique)

Pour tout chemin d'auth (HTTP handler, Pool/Discovery, watcher, CLI) :

1. **`MultiUserTokenStore`** — OAuth refresh du `OAuthRefreshToken` persisté.

Il n'existe plus AUCUNE source secondaire depuis la Phase 5 (2026-08-25) : ni
`sync_meta` DuckDB, ni env var `SPNKR_OAUTH_REFRESH_TOKEN_*`, ni le store
mono-user `data/auth/watcher_tokens.json`. Un joueur absent du store n'a pas de
token : il doit se reconnecter (SSO Xbox) ou passer par `cmd/token-capture`.

### Priorité d'écriture

- Au boot : `MigrateLegacyTokensAtBoot` copie env+DuckDB → store (idempotent) —
  SEULE lecture legacy encore câblée, sous kill-switch daté (cf. Phase 5).
- À chaque rotation : `UpdateOAuthRefreshToken(xuid, rt)` sur le store, point.
  (Le double-write `sync_meta` de compat a été retiré en Phase 5.)
- `cmd/token-capture` et `cmd/token-import` écrivent directement au store (plus de manipulation `.env.local`).
- `auth_xbox_oauth.Callback` persiste le RT post-SSO au store.

### Atomicité et sécurité

- Écritures atomiques via write-to-temp + `os.Rename`.
- Permissions `0600` sur les fichiers, `0700` sur le répertoire.
- XUID validé via `xuidIsSafe` (rejet path traversal).
- `UpdateOAuthRefreshToken` est read-modify-write protégé par `sync.Mutex`.

### Multi-titres (ADR 0008)

Le `xuid` est global cross-titres. Un seul fichier `{xuid}.json` couvre tous les titres pour un user. Pas de duplication par titre.

## Conséquences

### Positives

- **Bug Madina résolu** : Phase 3a inverse la priorité dans `refreshTokensFromDB` (store first), Phase 4 fait que `token-capture` écrit direct au store.
- **CLI tools cessent de brûler des tokens** : Phase 4bis ajoute la persistance de rotation au store dans le helper partagé `RefreshHaloTokensViaStoreFirst`.
- **SSO Xbox auto-reload** : Phase 4ter persiste le RT au store, plus besoin de re-prompter le user à chaque expiration.
- **Élimination de DuckDB comme credential store** : nettoyage architectural.
- **`.env.local` est revenu à son rôle de config-only** : Phase 5 (2026-08-25) a supprimé toute lecture runtime des `SPNKR_OAUTH_REFRESH_TOKEN_*`.

### Négatives / risques

- **Refactor large** : 13 commits sur la branche `refactor/auth-tokens-single-source`.
- **Période transitoire (CLOSE)** : elle a duré de mai à août 2026 — le code lisait et écrivait dans les deux endroits (store + DuckDB compat). Fermée par la Phase 5 le 2026-08-25.
- **Pas de tests E2E DuckDB** dans la branche : limitation environnementale du shell de dev (cgo + DuckDB native lib). Les tests unitaires couvrent la logique pure (auth package). Les tests DuckDB-dependent (pool/resolver, sync e2e) doivent être validés en prod.

## Phases livrées

| Phase | Description | Commit |
|-------|-------------|--------|
| 0 | Audit & inventaire exhaustif (`.ai/V7/AUDIT_TOKEN_STORAGE.md`) | `893706ca` |
| 1 | Extension `MultiUserTokenStore` (champ RT + helpers + 8 tests) | `16af4b04` |
| 2 | `MigrateLegacyTokens` boot-time + wiring main.go (14 tests) | `8b99ef3b` |
| 3a | `registry.refreshTokensFromDB` lit store first | `ab0ebefa` |
| 3b | `pool.Discovery.Scan` lit store first (RT + MSAL) | `9eb9b738` |
| 3c | `watcher_refresh.EnsureWatcherAccessToken` lit store | `5c7d87a8` |
| 3d | `scheduler/auto_sync` : no-op (déjà délégué au Pool) | `74b9755f` |
| 3bis | `halo.InvalidateCachedPlayerTokens(xuid)` pour rotation externe | `06ae0e69` |
| 4 | `token-capture` refait + `token-import` nouveau + onRotated→store | `d1c6cf43` |
| 4bis | 4 CLI tools via helper `RefreshHaloTokensViaStoreFirst` | `bde9f330` |
| 4ter | `auth_xbox_oauth.Callback` persiste RT au store | `e004b7c6` |
| 5 | Retrait des fallbacks legacy (+ Phase 6 absorbée) — 2026-08-25 | branche `refactor/adr0023-phase5` |

## Phase 5 — retrait des fallbacks legacy (livrée 2026-08-25)

**Critère d'armement (rempli)** : télémétrie prod `legacy_source_used` éteinte —
spam terminé le 25/07 (fix store-first post-sync achievements), puis seulement 3
occurrences isolées (31/07, 07/08, 11/08, toutes `source=duckdb_oauth`) et **0
depuis le 11/08**, soit 14 jours à zéro au moment du retrait (seuil : ≥ 7 j).
Store prod sain en parallèle : rotations du jour pour les 4 joueurs suivis + 5
comptes auth-only, aucun `reauth_required`/`AADSTS` depuis le 01/08.

**Supprimé** (lectures ET écritures) :

- `sync_meta.oauth_refresh_token` / `sync_meta.msal_token_cache` comme sources
  d'auth : `registry_auth.tryRefreshFromLegacy`, `pool.Discovery.adoptLegacySyncMeta`,
  `sync.readLegacyAuthInputs`, `worldenrich.loadLegacyInputs`, la lecture des ~13 CLI.
- env var `SPNKR_OAUTH_REFRESH_TOKEN_*` : tous les lecteurs runtime.
- store mono-user `data/auth/watcher_tokens.json` comme SOURCE de credentials
  (`pool.Discovery.legacyStore`, `watcher_refresh` fallback, `RefreshLoop`) — le
  fichier reste l'état propre du watcher RTA (access_token + XSTS), sans
  `refresh_token`.
- double-write DuckDB de la rotation (`onRotated`, probe admin, `registry_auth`)
  et `duckdb.WriteOAuthRefreshToken` / `ReadMSALCacheJSON*` /
  `ReadOAuthRefreshTokenFromSQL` devenus orphelins.
- `auth.LegacyAuthInputs`, `auth.RefreshTokenFromEnv`,
  `TokenProvider.TrySilentRefresh` (no-op SISU) et les champs `MSALCacheJSON`
  (`UserTokens`, `Attempt`, `CredentialSource.MSALCache`).
- télémétrie `observability/legacy_source.go` (plus aucun émetteur).

**Phase 6 absorbée** : le nettoyage de code mort prévu séparément est fait dans
la même passe (règle projet « 0 code mort »). Il ne reste de `queries_auth.go`
que `ReadOAuthRefreshToken`, pour la seule migration boot.

**Exception unique, sous kill-switch daté** : `MigrateLegacyTokens`
(`internal/platform/auth/migration.go`) + son wiring `migrateLegacyAuthTokensAtBoot`
(`cmd/server/main.go`) continuent de lire env + `sync_meta` au boot pour recopier
un RT resté en legacy vers le store.

- Bascule du défaut : **2026-08-25** (plus aucun fallback runtime).
- Date cible de retrait : **2026-10-01**.
- Critère mesurable : 0 token migré au boot sur 30 jours de logs prod, c.-à-d.
  `auth_migration: scan terminé` avec `rt_migrated=0` à chaque boot et aucun
  `auth_migration: RT migré vers store` (`grep 'auth_migration: RT migré' sync.log`).
- Au retrait : supprimer `migration.go` + son test, le wiring boot,
  `queries_auth.go` et les 2 entrées d'allowlist du sentinel. Le drop physique des
  colonnes `sync_meta` suit la recette ADR 0026 au prochain rebuild.

**Garde-rails posés** (`internal/platform/auth/sentinel_test.go`) : allowlist des
lecteurs d'env var réduite de ~30 entrées à 1 (la migration boot), nouveau guard
sur `duckdb.ReadOAuthRefreshToken` (2 sites + tests) ;
`internal/sync/no_legacy_source_used_test.go` interdit désormais tout littéral de
credential legacy dans le package sync ;
`pool/discovery_legacy_warn_test.go` et `discovery_priority_test.go` vérifient
qu'un résidu `sync_meta` ou une env var présente n'entre JAMAIS dans le pool.

**Impact contrat** : `PlayerTokenHealth.msal` renommé `access` (le champ mesurait
déjà l'expiration de l'access_token Microsoft, pas un cache MSAL) ;
`TokenProbeResult.has_msal_cache` supprimé ; codes d'erreur `msal_init_error` /
`msal_acquire_error` renommés `device_flow_init_error` /
`device_flow_acquire_error`. `openapi.yaml` + `generated.ts` régénérés.

## Alternatives considérées

### Alt 1 : Garder DuckDB comme source unique

Plus simple à court terme mais perpétue l'anti-pattern OLAP-as-credential-store. Rejeté.

### Alt 2 : Ajouter un wrapper qui présente une API unique au-dessus des 4 sources

Trop d'abstraction, masque le vrai problème (sources concurrentes). Rejeté.

### Alt 3 : Faire de `.env.local` la source unique

Pas thread-safe (env vars du process), pas mutable proprement, pas atomique. Rejeté.

### Alt 4 (choisie) : Promouvoir `MultiUserTokenStore` qui a déjà le bon design

Atomic rename, perms strictes, structure JSON par xuid, code déjà testé. Étendre avec un champ `OAuthRefreshToken` est trivial. Adopté.

## Onboarding utilisateur post-ADR

### Mode "normal" — SSO Xbox web

1. User clique "Se connecter avec Xbox" dans l'UI
2. Flow OAuth Microsoft → `/auth/xbox/callback`
3. Le callback persiste le RT au store automatiquement
4. **Aucune manipulation `.env.local` nécessaire**

### Mode "advanced" — CLI `token-capture`

```bash
go run ./cmd/token-capture/ Madina97294
# → ouvre browser, user s'authentifie, RT écrit direct dans data/auth/watcher_tokens/{xuid}.json
# → redémarrer le serveur, c'est tout
```

### Mode "advanced" — RT obtenu d'un autre outil

```bash
cat token-madina.txt | go run ./cmd/token-import/ Madina97294
# → lit le RT sur stdin (pas argv pour éviter exposition shell history)
# → écrit direct dans le store
```

### Pré-requis communs

Le joueur doit être déclaré dans `db_profiles.json` avec son `xuid` avant `token-capture` ou `token-import`. Sans xuid, le store ne peut pas adresser l'entrée correctement.

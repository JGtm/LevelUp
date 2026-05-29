# AUDIT_TOKEN_STORAGE — Refactor MultiUserTokenStore source unique

Date : 2026-05-26
Branche : `refactor/auth-tokens-single-source`
ADR cible (Phase 6) : `docs/adr/0023-auth-tokens-single-source.md`

## Contexte

Aujourd'hui les tokens auth (`refresh_token` OAuth Microsoft + cache MSAL) sont éparpillés sur 4 supports concurrents :

1. **`.env.local`** : `SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>=...` (seed bootstrap manuel)
2. **`sync_meta` DuckDB** (par joueur) : clés `oauth_refresh_token` + `msal_token_cache`
3. **`data/auth/watcher_tokens/{xuid}.json`** (`MultiUserTokenStore`) : champ `MSALCacheJSON` (RT pas encore stocké)
4. **`data/auth/watcher_tokens.json`** (`TokenStore` mono-user legacy) : champ `RefreshToken`

Conséquences observées :
- Bug Madina97294 (2026-05-26) : Air hot-reload → Pool brûle l'env var → DuckDB peuplé → handler HTTP lit env var stale → `invalid_grant`
- 4 CLI tools qui appellent directement `provider.Exchange` peuvent brûler le RT env.local et reproduire le bug
- DuckDB OLAP utilisé comme credential store — anti-pattern architectural

Cible : **`MultiUserTokenStore` est la source unique**. Tout le reste devient legacy à migrer / supprimer.

---

## Inventaire — Sites de lecture

| # | Fichier | Ligne | Fonction | Source lue | Action |
|---|---------|-------|----------|------------|--------|
| 1 | `internal/api/registry.go` | 944 | `refreshTokensFromDB` (chemin MSAL) | `duckdb.ReadMSALCacheJSON` | **Phase 3** : lire `MultiUserTokenStore.MSALCacheJSON` en premier |
| 2 | `internal/api/registry.go` | 962-964 | `refreshTokensFromDB` (chemin OAuth) | env var + `duckdb.ReadOAuthRefreshToken` | **Phase 3** : lire `MultiUserTokenStore.OAuthRefreshToken` en premier |
| 3 | `internal/api/registry.go` | 1156 | `oauthRefreshTokenForPlayer` | env var (`os.Getenv`) | **Phase 5** : supprimer |
| 4 | `internal/platform/auth/pool/discovery.go` | 109-110 | `Scan` (DuckDB) | `duckdb.ReadMSALCacheJSON` + `ReadOAuthRefreshToken` | **Phase 3** : lire `MultiUserTokenStore` en premier |
| 5 | `internal/platform/auth/pool/discovery.go` | 120, 205 | `Scan` (env fallback) | `readOAuthRefreshTokenFromEnv` | **Phase 5** : supprimer (mais garder migration Phase 2) |
| 6 | `internal/platform/auth/pool/discovery.go` | 134 | `Scan` (watcher store) | `multiUserStore.Load(xuid)` | **OK** : déjà sur la cible |
| 7 | `internal/platform/auth/pool/discovery.go` | 146 | `Scan` (legacy mono-user) | `legacyStore.Load()` | **Phase 6** : supprimer après migration |
| 8 | `internal/platform/auth/watcher_refresh.go` | 59-71 | `RefreshTokenFromEnv` | env var | **Phase 5** : supprimer |
| 9 | `internal/platform/auth/watcher_refresh.go` | 126-135 | `EnsureWatcherAccessToken` | `TokenStore.RefreshToken` + env fallback | **Phase 3** : lire `MultiUserTokenStore` en premier |
| 10 | `internal/scheduler/auto_sync.go` | `defaultTokenReader` | (à confirmer) | env + sync_meta | **Phase 3** : refactor |
| 11 | `internal/platform/duckdb/queries_auth.go` | 26 | `ReadMSALCacheJSON` | DuckDB direct | **Phase 6** : supprimer la fonction |
| 12 | `internal/platform/duckdb/queries_auth.go` | (autre) | `ReadOAuthRefreshToken` | DuckDB direct | **Phase 6** : supprimer la fonction |

## Inventaire — Sites d'écriture

| # | Fichier | Ligne | Fonction | Destination écrite | Action |
|---|---------|-------|----------|-------------------|--------|
| W1 | `cmd/server/main.go` | 1094 | `onRotated` callback (Pool) | `duckdb.WriteOAuthRefreshToken` | **Phase 4** : écrire dans `MultiUserTokenStore` (+ DuckDB transitoirement) |
| W2 | `internal/api/handlers/admin_auto_sync.go` | 168 | `onRotated` callback (admin) | `duckdb.WriteOAuthRefreshToken` | **Phase 4** : pareil |
| W3 | `internal/api/registry.go` | (hotfix actuel) | `refreshTokensFromDB` post-rotation | `duckdb.WriteOAuthRefreshToken` | **Phase 4** : écrire dans store |
| W4 | `cmd/token-capture/main.go` | 103 | Génère txt file | `os.WriteFile` | **Phase 4** : écrire direct dans `MultiUserTokenStore` + update `.env.local` supprimé |
| W5 | `internal/api/handlers/auth_xbox_oauth.go` | 190 | Callback SSO Xbox | (à vérifier) | **Phase 4ter** : confirmer que le RT obtenu va dans le store |
| W6 | `internal/scheduler/auto_sync_e2e_test.go` | 168, 286 | onRotated test | DuckDB | **Phase 4** : adapter tests |
| W7 | `internal/platform/duckdb/queries_auth.go` | 12 | `WriteOAuthRefreshToken` | sync_meta | **Phase 6** : supprimer la fonction |

## Inventaire — CLI tools indépendants

Ces CLI appellent `provider.Exchange` ou `provider.TryOAuth*` directement sans passer par Pool ni ServiceRegistry. Ils peuvent brûler des tokens sans persister la rotation.

| # | CLI | Lignes appel provider | Source RT actuelle | Action Phase 4bis |
|---|-----|---------------------|-------------------|-------------------|
| C1 | `cmd/refresh-metadata/main.go` | 391, 399 | env var probable | Lire/écrire `MultiUserTokenStore` |
| C2 | `cmd/refresh-career-ranks/main.go` | 175, 185 | env var probable | Pareil |
| C3 | `cmd/populate-career-rank-images/main.go` | 277, 286, 297 | env var probable | Pareil |
| C4 | `cmd/diag_emblem_colors/main.go` | 34, 84 | env var probable | Pareil |
| C5 | `cmd/levelup/cmd_sync.go` | 58, 258 | Pool (déjà) | OK, mais `onRotated` est `nil` (à fixer Phase 4) |
| C6 | `cmd/token-capture/main.go` | (refait Phase 4) | Device flow | Écrire dans store |

## Inventaire — Cache process

| # | Site | Fichier | TTL | Invalidation actuelle | Action |
|---|------|---------|-----|----------------------|--------|
| K1 | `halo.GetCachedPlayerTokens` / `SetCachedPlayerTokens` | `internal/platform/halo/player_token_cache.go` | 50 min | Auto-expire | **Phase 3bis** : invalider explicitement sur rotation détectée |
| K2 | `pool.Resolver.cache` | `internal/platform/auth/pool/resolver.go` | 3h30 | TTL + `Refresh()` | OK, géré par le Pool lui-même |
| K3 | `CareerLiveCache` | `internal/service/career_live_cache.go` | 5min / 6h | TTL singleflight | OK, indépendant des refresh tokens |

## Mapping cible

```
.env.local SPNKR_OAUTH_REFRESH_TOKEN_*  ─┐
                                          ├─ Phase 2 migration au boot ─→  data/auth/watcher_tokens/{xuid}.json
sync_meta.oauth_refresh_token  ──────────┤                                       │
sync_meta.msal_token_cache  ─────────────┤                                       │
data/auth/watcher_tokens.json (mono)  ──┘                                       │
                                                                                 ▼
                                                                          MultiUserTokenStore
                                                                          (UserTokens struct)
                                                                                 │
                          ┌──────────────────────────────────────────────────────┤
                          │                                                       │
                          ▼                                                       ▼
                  pool.Discovery.Scan                          registry.refreshTokensFromDB
                          │                                                       │
                          ▼                                                       ▼
                  Pool/Resolver                                  enrichWithHaloTokens
                          │                                                       │
                          └─────────────► onRotated callback ◄────────────────────┘
                                                  │
                                                  ▼
                                          MultiUserTokenStore.UpdateOAuthRefreshToken
                                          + halo.InvalidateCachedPlayerTokens(xuid)
```

## Contrats à respecter

1. **Atomicité des écritures** : `MultiUserTokenStore.Upsert` utilise write-to-temp + rename atomique. À conserver pour `UpdateOAuthRefreshToken`.
2. **Permissions** : 0600 fichiers / 0700 répertoire. À conserver.
3. **Idempotence migration** : Phase 2 doit pouvoir s'exécuter plusieurs fois sans corrompre le store.
4. **Pas d'appel HTTP en migration** : Phase 2 = copie de strings entre stores, aucun appel Microsoft.
5. **Multi-titres** : `xuid` est global (ADR 0008). Un seul token par xuid couvre tous les titres. Pas de duplication par titre.
6. **Anti-corruption** : si le store contient un token et que la migration trouve aussi un legacy, la migration **ne doit pas écraser** le store (le store est autoritaire).

## Périmètre — fichiers à toucher

### Création
- `internal/platform/auth/migration.go` (Phase 2)
- `cmd/token-import/main.go` (Phase 4)
- `docs/adr/0023-auth-tokens-single-source.md` (Phase 6)

### Modification
- `internal/platform/auth/multi_user_token_store.go` (Phase 1 : champ + helpers)
- `internal/platform/auth/multi_user_token_store_test.go` (Phase 1 : tests)
- `cmd/server/main.go` (Phase 2 wiring + Phase 4 onRotated)
- `internal/api/handlers/admin_auto_sync.go` (Phase 4 onRotated)
- `internal/api/registry.go` (Phase 3 read + Phase 4 write)
- `internal/api/registry_test.go` (Phase 3 tests adaptés)
- `internal/platform/auth/pool/discovery.go` (Phase 3 + Phase 5)
- `internal/platform/auth/pool/discovery_test.go` (Phase 3 tests adaptés)
- `internal/platform/auth/pool/resolver.go` (Phase 3bis cache invalidation)
- `internal/platform/auth/watcher_refresh.go` (Phase 3 + Phase 5)
- `internal/platform/auth/watcher_refresh_test.go` (Phase 3 tests adaptés)
- `internal/scheduler/auto_sync.go` (Phase 3 defaultTokenReader)
- `cmd/token-capture/main.go` (Phase 4 refait)
- `cmd/refresh-metadata/main.go` (Phase 4bis)
- `cmd/refresh-career-ranks/main.go` (Phase 4bis)
- `cmd/populate-career-rank-images/main.go` (Phase 4bis)
- `cmd/diag_emblem_colors/main.go` (Phase 4bis)
- `cmd/levelup/cmd_sync.go` (Phase 4 onRotated wiring)
- `internal/api/handlers/auth_xbox_oauth.go` (Phase 4ter)
- `internal/platform/halo/player_token_cache.go` (Phase 3bis ajouter `Invalidate(xuid)`)

### Suppression (Phase 5-6)
- Fonctions `duckdb.ReadOAuthRefreshToken` / `WriteOAuthRefreshToken` / `ReadMSALCacheJSON` / `WriteMSALCacheJSON` dans `internal/platform/duckdb/queries_auth.go`
- Lignes `SPNKR_OAUTH_REFRESH_TOKEN_*` dans `.env.local` (à documenter dans la doc onboarding)
- Code mort dans `discovery.go` (env fallback)
- Helper `oauthRefreshTokenForPlayer` dans `registry.go`
- Helper `RefreshTokenFromEnv` + `readOAuthRefreshTokenFromEnv` dans `auth/`

## Risques identifiés

| Risque | Mitigation |
|--------|-----------|
| Régression sur un chemin oublié | Audit exhaustif ci-dessus + grep anti-régression en Phase 5 |
| Race condition entre Pool boot et migration | Phase 2 s'exécute AVANT le Pool.Discovery (séquentiel) |
| Air restart pendant phase de migration | Migration idempotente + atomic rename du store |
| Tests legacy qui setupent env vars | Adapter tests Phase 3 pour utiliser un MultiUserTokenStore en mémoire |
| CLI tools cassés silencieusement | Phase 4bis : tests E2E minimaux par CLI |
| Watcher daemon mono-user en parallèle | Migration legacy `TokenStore` → store Phase 2 |

## Done definition (récapitulatif)

**Phase 0** (cette doc) : audit complet, branche worktree créée, plan validé par l'utilisateur.

**Phase 1** : `MultiUserTokenStore` étendu, tests verts (`go test ./internal/platform/auth/...`).

**Phase 2** : `MigrateLegacyTokensAtBoot` wired dans `main.go`, logs visibles au premier boot, idempotent au deuxième.

**Phase 3** : 4 sites de lecture migrés, tests passent, démarrage propre n'émet pas de warning `legacy_source_used`.

**Phase 3bis** : `halo.InvalidateCachedPlayerTokens(xuid)` ajouté + appelé sur rotation détectée.

**Phase 4** : tous les sites d'écriture vont vers store, `token-capture` écrit en store, `token-import` créé.

**Phase 4bis** : 4 CLI tools migrés, smoke test manuel par CLI.

**Phase 4ter** : OAuth callback Xbox SSO écrit dans store.

**Phase 5** : `grep "SPNKR_OAUTH_REFRESH_TOKEN_" apps/go-api/` retourne 0 hit hors `migration.go`.

**Phase 6** : ADR rédigé, CLAUDE.md mis à jour, fonctions DuckDB supprimées, cleanup `.env.local`.

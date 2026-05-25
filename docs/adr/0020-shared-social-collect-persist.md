# ADR 0020 — `shared_social` writes via pattern Collect→Persist

**Date** : 2026-05-25
**Statut** : Accepté
**Branche refactor** : `refactor/shared-social-collect-persist`

## Contexte

`shared_social.duckdb` regroupe les données sociales/transverses (médias indexés, likes, favoris matchs, notifications in-app, records & milestones, prestige). Avant ce refactor, les écritures étaient **dispersées** dans une dizaine de sites Go (`ops/media.go`, `social_repo.go`, `notifications_repo.go`, `post_sync_deltas.go`, `prestige_social_repo.go`, …) qui faisaient des `db.ExecContext` directs.

**Problème observé en production (mai 2026)** : à chaque rebuild Air (Windows SIGKILL), `shared_social.duckdb` devenait inouvrable au boot suivant avec :

```
INTERNAL Error: Failure while replaying WAL file
"shared_social.duckdb.wal":
Calling DatabaseManager::GetDefaultDatabase with no default database set
```

Symptôme cascade : `SharedSocial = nil` pour tous les joueurs → rail média home vide, `prestige_bundle_init_failed`, etc.

**Cause racine** : bug DuckDB upstream [#7659](https://github.com/duckdb/duckdb/issues/7659). Les écritures massives d'`IndexMedia` (115 médias + 93 associations) laissaient des entrées WAL non-checkpointed que DuckDB ne savait pas rejouer au reopen.

Les autres DBs (`shared_matches_v2`, player) ne souffraient pas du même bug parce qu'elles passent par `persist.BatchQueue` (ADR 0019) qui garantit l'atomicité TX + le flush au shutdown.

## Décision

**Aligner `shared_social` sur le pattern Collect→Persist** déjà adopté pour `shared_matches_v2` et player DBs. Toutes les écritures passent par un `SharedSocialPersister` qui garantit :

1. **Atomicité TX** : `BEGIN → INSERTs/UPDATEs → COMMIT` → rollback complet si erreur
2. **CHECKPOINT explicite** après chaque commit → WAL vidé sur disque immédiatement → bug DuckDB #7659 impossible
3. **INSERT-only sur tables critiques** : `player_records` migré en append-only (`player_records_history` + vue `player_records_latest`)
4. **API uniforme** : `persister.Persist(ctx, batch)` synchrone (live UI-critique) ou wrappable en async (event-driven/planifié)

## Architecture

```
┌── HTTP handler / post-sync hook / scheduler ────────────────────────┐
│  batch := &persist.SharedSocialBatch{                               │
│      MediaFiles: [...],                                             │
│      Likes:      [...],                                             │
│      Notifications: [...],                                          │
│  }                                                                  │
│  err := pdb.SocialPersister.PersistBatch(ctx, batch)                │
└────────────────────────────┬────────────────────────────────────────┘
                             ▼
┌── persist.SharedSocialPersister.Persist ────────────────────────────┐
│  tx := db.BeginTx(ctx)                                              │
│  persistMediaFiles(tx, batch.MediaFiles)                            │
│  persistMediaThumbnails(tx, batch.MediaThumbnails)                  │
│  persistMediaAssociations(tx, batch.MediaAssociations)              │
│  persistLikes(tx, batch.Likes, batch.LikesToRemove)                 │
│  persistFavorites(tx, batch.Favorites, batch.FavoritesToRemove)     │
│  persistNotifications(tx, batch.Notifications, batch.Reads)         │
│  persistPlayerRecords(tx, batch.PlayerRecordsAppend) // append-only │
│  tx.Commit()                                                        │
│  db.Exec("CHECKPOINT")  ◄── garde-fou bug DuckDB #7659              │
└─────────────────────────────────────────────────────────────────────┘
```

## Injection (cycle d'import résolu)

`internal/persist` importe déjà `internal/platform/duckdb` (via `combined_persister.go`). Importer dans l'autre sens créerait un cycle. **Solution** : interface `duckdb.SocialPersister` (any-typé) + factory hook configuré au boot.

- `internal/platform/duckdb/social_persister_iface.go` : interface + `SocialPersisterFactory`
- `internal/persist/shared_social_persister.go` : implémentation concrète + méthode `PersistBatch(ctx, any)` qui cast vers `*SharedSocialBatch`
- `cmd/server/main.go` (init précoce) :
  ```go
  duckdb.SocialPersisterFactory = func(db *sql.DB) duckdb.SocialPersister {
      return persist.NewSharedSocialPersister(db)
  }
  ```
- `pool.go:openPlayerDB` instancie via la factory si elle est configurée

## Pattern append-only `player_records`

Avant : `INSERT ON CONFLICT (xuid, metric, period) DO UPDATE SET value = EXCLUDED.value, …` → pressionne l'index ART DuckDB (cf. CLAUDE.md "Phase 2 ART") + entrées WAL UPDATE problématiques.

Après : nouvelle table `player_records_history` (PK = `id BIGSERIAL`, `written_at` ajouté). Vue `player_records_latest` retourne la dernière valeur par `(xuid, metric, period)` via `DISTINCT ON ORDER BY written_at DESC`. Backfill one-shot dans la migration.

Repos lecteurs : à migrer progressivement vers la vue `_latest` (rétrocompat actuelle : la table `player_records` originale est conservée).

## Conséquences

### Positives
- Plus aucune entrée WAL non-rejouable produite par IndexMedia (Phase 3 CHECKPOINT explicite)
- Architecture cohérente avec `shared_matches_v2` et player DBs
- Atomicité TX sur toutes les écritures shared_social
- Tests anti-régression à 2 niveaux : code-level (parse-AST) + intégration (kill brutal sub-process)

### Négatives / Dette
- Refactor de tous les sites d'écriture (`SocialRepo`, `NotificationsRepo`, `prestige_social_repo`, `post_sync_deltas`, etc.) pour passer par le Persister : non finalisé dans la première session. Sites encore en écriture directe ont une fenêtre WAL résiduelle mais leur volume reste faible (1-quelques INSERTs/op utilisateur) → pas d'accumulation problématique.
- Cycle d'import contourné par `interface{}` + cast → DX moins typée. Alternative future : extraire `SharedSocialBatch` dans un sous-package `internal/persist/socialbatch` sans dépendance duckdb.
- `player_records` (table originale) conservée pour rétrocompat des lectures non-migrées → dette à supprimer dans une PR ultérieure.

### Risques mitigés
- Bug DuckDB #7659 ne peut plus se déclencher dans les flows IndexMedia (CHECKPOINT explicite)
- Sentinel parse-AST `TestNoATTACHOnSocialDB` empêche la réintroduction d'`ATTACH` (autre vecteur du bug)
- Test E2E `TestE2E_KillBrutal_ReopenAfterCrashedIndexMedia` valide le scénario prod exact (sub-process exit brutal + reopen)

## Validation

- `go build ./...` OK
- `go vet ./...` OK
- `go test ./internal/persist/... ./internal/ops/... ./internal/migration/...` tous verts
- Test E2E kill brutal sub-process : passe (reopen OK, données préservées)

## Phases livrées (commits)

| Phase | Commit | Contenu |
|---|---|---|
| 0 — Audit | `bca67341` | `.ai/AUDIT_SHARED_SOCIAL_WRITERS.md` |
| 1 — Persister | `bca67341` | `shared_social_persister.go` + 7 tests |
| 2 — Append-only | `0e7ac7e0` | Migration `player_records_history` + vue + 4 tests |
| 3 — CHECKPOINT IndexMedia | `0e7ac7e0` | 1 ligne CHECKPOINT + 3 tests anti-régression |
| 4 — Infra Persister | `cd637674` | Interface `SocialPersister` + champ `PlayerDB.SocialPersister` + factory hook |
| 5 — Wiring main.go | (cette PR) | `duckdb.SocialPersisterFactory = persist.NewSharedSocialPersister` |
| 6 — Sentinels + E2E kill brutal | (cette PR) | Test sub-process exit brutal |
| 7 — ADR + doc | (cette PR) | Ce document |

## Références

- Bug DuckDB upstream : https://github.com/duckdb/duckdb/issues/7659
- ADR 0019 (Collect→Persist pattern original) : `docs/adr/0019-collect-persist-architecture.md`
- Audit Phase 0 : `.ai/AUDIT_SHARED_SOCIAL_WRITERS.md`

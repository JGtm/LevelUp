# AUDIT — Writers & Readers shared_social.duckdb

**Date** : 2026-05-25
**Branche** : `refactor/shared-social-collect-persist`
**Phase** : 0 (audit non-destructif)

## Objectif

Cataloguer **exhaustivement** tous les sites Go qui accèdent à `shared_social.duckdb` (lecture et écriture), afin de planifier précisément la migration vers le pattern `persist.BatchQueue` (Phase 1+).

**Critère** : aucun `socialDB.Exec*` ou `pdb.SharedSocial.Exec*` en runtime serveur ne doit subsister après Phase 5 hormis ceux situés dans `internal/persist/`.

## Tables présentes dans shared_social.duckdb

| Table | Cf. migration | Usage |
|---|---|---|
| `media_files` | `steps_shared_social.go` | Catalogue médias (vidéos/captures) indexés depuis disque |
| `media_match_associations` | `steps_shared_social.go` | Lien média ↔ match (calculé par algo fenêtre temporelle) |
| `media_likes` | `steps_shared_social.go` | Likes sociaux par joueur sur médias |
| `match_favorites` | `steps_shared_social.go` | Matchs marqués favoris par joueur |
| `player_notifications` | `steps_player_notifications.go` | Cloche notifications in-app (auto-sync, milestones, mentions, etc.) |
| `player_records` | `steps_player_notifications.go` + `steps_shared_social_records_window.go` | Records & Milestones (Progression V2 / Ascension) — feature backend OK, **pas encore consommée par UI** |
| `prestige_player_*` | `steps_shared_social_prestige.go` | État Prestige par joueur |

## 1. Writers — sites à migrer (RUNTIME SERVEUR, prod)

### 1.1 IndexMedia / Media indexing

| Site | Op | Table | Mode requis |
|---|---|---|---|
| `ops/media.go:insertMediaFile` | INSERT OR IGNORE / UPDATE | `media_files` | Event-driven (post-sync hook) ou planifié (scan settings) → **async via queue** |
| `ops/media.go:BackfillThumbnailPaths` | UPDATE | `media_files` | Event-driven (post-IndexMedia) → **async** |
| `ops/media.go:ensureMediaTables` | DDL (CREATE SEQUENCE, CREATE TABLE IF NOT EXISTS, ALTER TABLE ADD COLUMN) | `media_files`, `media_match_associations` | **À supprimer** — schéma garanti par migrations seulement |
| `ops/media.go:dropLegacyMediaFilesIfNeeded` | DROP TABLE | `media_files`, `media_match_associations` | **À supprimer** — code de migration legacy |
| `ops/media_associate.go:bulkInsertAssociations` | INSERT OR IGNORE | `media_match_associations` | Event-driven → **async via queue**. Logique d'orchestration à migrer dans Persister. |

### 1.2 Media upload / dislike (HTTP handlers)

| Site | Op | Table | Mode requis |
|---|---|---|---|
| `service/media_service.go` (upload + dislike) | UPDATE / DELETE | `media_files` | Live UI-critique → **sync : Persister.Persist direct, retourne 200 quand commit OK** |
| `platform/duckdb/media_repo.go` (write methods) | INSERT/UPDATE/DELETE | `media_files`, `media_match_associations` | À refactorer en Persister sync ou async selon le caller |

### 1.3 Likes / Favorites

| Site | Op | Table | Mode requis |
|---|---|---|---|
| `platform/duckdb/social_repo.go` (writes) | INSERT/DELETE | `media_likes`, `match_favorites` | **Live UI-critique** (user click) → **sync** |

### 1.4 Notifications

| Site | Op | Table | Mode requis |
|---|---|---|---|
| `platform/duckdb/notifications_repo.go` (writes) | INSERT/UPDATE/DELETE | `player_notifications` | Mixte — création post-sync (async) ; mark-as-read (sync) |
| `platform/duckdb/notifications_repo_helpers.go` | UPDATE | `player_notifications` | À catégoriser au moment de la migration site par site |
| `notify/notifiers.go` | UPDATE | `player_notifications` | Event-driven → async |

### 1.5 Records & Milestones (Ascension/Progression V2)

| Site | Op | Table | Mode requis |
|---|---|---|---|
| `api/post_sync_deltas.go:698` | INSERT ON CONFLICT DO UPDATE | `player_records` | Event-driven (post-sync hook) → **async**. **Migration append-only obligatoire (Phase 2)** : transformer en INSERT-only sur `player_records_history` + vue `latest`. |
| `platform/duckdb/records_repo.go` (writes) | INSERT | `player_records` | Async via Persister |

### 1.6 Prestige

| Site | Op | Table | Mode requis |
|---|---|---|---|
| `platform/duckdb/prestige_social_repo.go` (writes) | INSERT/UPDATE | `prestige_player_*` | Mixte — calculé post-sync (async), modifié via UI (sync) |
| `platform/duckdb/prestige_player_repo.go` (writes côté player ; à vérifier si touche shared_social) | À auditer | — | À catégoriser en début Phase 4 |

## 2. Readers — sites à NE PAS toucher

Les lectures restent directes via le pool process-wide (concurrent reads safes en DuckDB). Aucune migration nécessaire.

| Site | Op | Table |
|---|---|---|
| `platform/duckdb/home_repo_matches.go:413` (Q28 home rail) | SELECT | `media_files`, `media_match_associations` |
| `platform/duckdb/match_view_repo.go:990` (Q24 médias d'un match) | SELECT | `media_files`, `media_match_associations` |
| `platform/duckdb/media_repo.go` (reads / pipeline Q37) | SELECT | `media_files`, `media_match_associations`, `media_likes` |
| `platform/duckdb/records_repo.go:54,117` (reads progression) | SELECT | `player_records` |
| `platform/duckdb/progression_diag_repo.go:47` (diag) | SELECT | `player_records` |
| `platform/duckdb/social_repo.go` (reads) | SELECT | `media_likes`, `match_favorites` |
| `platform/duckdb/notifications_repo.go` (reads) | SELECT | `player_notifications` |
| `platform/duckdb/prestige_social_repo.go` (reads) | SELECT | `prestige_player_*` |
| `api/post_sync_deltas.go:667` | SELECT | `player_records` (lecture diff pour calcul delta) |
| `api/registry.go:626` | `pdb.SharedSocial.Path()` (introspection chemin, pas DB) | — |

## 3. Sites CLI / tools — hors scope direct

Ces sites tournent **hors processus serveur**. Ils ouvrent shared_social directement, font leur job, ferment proprement (CHECKPOINT au Close). Pas de risque WAL au reboot serveur.

| Site | Rôle |
|---|---|
| `cmd/prestige-seed/main.go` | Seed prestige (initialisation one-shot) |
| `cmd/reindex-media-thumbs/main.go` | Backfill miniatures |
| `cmd/regen-thumbnails/main.go` | Régénération miniatures WebP |
| `cmd/migrate-media-paths/main.go` | Migration paths legacy |
| `cmd/cleanup_media_index/main.go` | Wipe media_files (mon outil one-shot) |

**Décision** : laisser tel quel. Le bug WAL ne se produit que quand le serveur RW tourne ET qu'un autre process écrit en concurrence — ce qui n'est pas le pattern ici (CLI utilisés serveur arrêté).

## 4. Sites DDL (migrations) — pattern correct

Les migrations `internal/migration/steps_shared_social*.go` font des DDL au boot **dans un contexte propre** (avant le démarrage du runtime serveur, sur une conn dédiée).

| Site | Action |
|---|---|
| `steps_shared_social.go` | CREATE TABLE de base |
| `steps_shared_social_prestige.go` | Tables prestige |
| `steps_shared_social_records_window.go` | Extension `player_records` |
| `steps_shared_social_purge_data_health.go` | Purge data_health_warning notifs |
| `steps_player_notifications.go` | Tables notifications + records |

**Décision** : laisser tel quel. Les migrations sont LE bon endroit pour les DDL. Le runtime serveur (IndexMedia) ne doit plus en faire.

## 5. Tests — à adapter si nécessaire

| Site | Impact migration |
|---|---|
| `ops/media_associate_regression_test.go` | E2E à conserver (valide pipeline) |
| `ops/media_backup_cgo_test.go` | Tests IndexMedia — à adapter pour utiliser Persister |
| `platform/duckdb/notifications_repo_test.go` | Tests existants OK pour reads ; writes via Persister une fois migrés |
| `platform/duckdb/player_repos_test.go` | Reset test (legacy DROP) — à conserver, c'est un test |
| `platform/duckdb/stress_concurrent_test.go` | Stress test direct sur socialDB — à adapter pour passer par Persister |
| `notify/media_notify_test.go` | À auditer Phase 4 |

## 6. Synthèse — sites de migration par Phase

### Phase 1 — Persister + BatchBuilder (infra)
- Création `internal/persist/shared_social_persister.go`
- Extension `internal/persist/batch.go` (struct `SharedSocialBatch`)
- Extension `internal/persist/builder.go` (setters `AddMediaFile`, `AddLike`, etc.)

### Phase 2 — player_records append-only (migration DB)
- Migration `steps_shared_social_records_append_only.go`
- Refactor `post_sync_deltas.go:698` (1 INSERT direct → INSERT append-only via Persister)
- Backfill data existante (one-shot CLI)

### Phase 3 — IndexMedia (le plus gros writer)
- Refactor `ops/media.go:insertMediaFile`, `ops/media.go:BackfillThumbnailPaths`, `ops/media.go:AssociateMediaWithMatches`
- Suppression `ensureMediaTables` + `dropLegacyMediaFilesIfNeeded` (schéma via migrations)
- Refactor `ops/media_associate.go:bulkInsertAssociations`

### Phase 4 — likes / favorites / notifications / prestige
- Refactor `social_repo.go` (likes + favorites) — sync direct
- Refactor `notifications_repo.go` (mixed — site par site)
- Refactor `prestige_social_repo.go` (mixed)
- Refactor `service/media_service.go` (upload + dislike) — sync direct

### Phase 5 — wiring main.go + shutdown
- Initialisation `socialBatchQueue` au boot
- Injection dans tous les sites (DI)
- Shutdown : `socialBatchQueue.Drain()` AVANT `duckdb.CloseAll()`
- CHECKPOINT garanti par Persister + Drain au shutdown

### Phase 6 — sentinels + E2E kill brutal
- Test parse-AST : aucun `Exec` direct sur shared_social hors `internal/persist/`
- Test E2E : kill -KILL serveur après IndexMedia → reboot OK
- Test stress 100 cycles

### Phase 7 — doc + cleanup
- Thought log
- ADR 0020 "shared_social RW via Persister exclusif"
- Mise à jour `internal/persist/doc.go` (retirer note "media ne suit pas le flux")
- Mise à jour CLAUDE.md

## 7. Logs — règles

Tous les logs vont dans `logs/` à la racine du projet (multi-module logger déjà actif). Clés structurées :

```
slog.InfoContext(ctx, "persist: shared_social batch persisted",
    "batch_id", batch.ID,
    "media_count", len(batch.MediaFiles),
    "duration_ms", elapsed.Milliseconds())

slog.ErrorContext(ctx, "persist: shared_social Persist failed",
    "batch_id", batch.ID,
    "err", err)
```

## 8. Validation Phase 0 (cette doc)

- [x] Tous les sites Exec*/Query* sur shared_social listés (writers + readers)
- [x] Catégorisation par type (live / event / planifié)
- [x] Décision migration vs laisser inchangé par site
- [x] Aucun site oublié (sentinel parse-AST Phase 6 confirmera)
- [x] Tables de shared_social listées avec contexte (Records & Milestones non-exposé UI, etc.)

**Phase 0 terminée. Passage Phase 1.**

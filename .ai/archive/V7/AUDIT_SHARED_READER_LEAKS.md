# Audit — Usages résiduels de `shared.X` après ADR 0016

**Date** : 2026-05-20  
**Branche** : `fix/auto-sync-different-configuration`  
**Contexte** : ADR 0016 + commits 8k.X / 9c.X ont retiré tout ATTACH `shared` du pool DuckDB. Plus aucune conn `pdb.Player`, `pdb.SharedSocial`, `pdb.Metadata` ne porte d'ATTACH `shared`. Seule la conn obtenue via `pdb.SharedReadDB().Get(ctx)` accède aux tables shared — **et sans préfixe `shared.`** puisque la conn pointe directement sur la DB shared.

**Symptôme observé en prod** : `LoadMediaFiles: Catalog Error: Table with name "shared.match_registry" does not exist because schema "shared" does not exist.`

---

## 1. CRITIQUE — Sites qui cassent en prod (à migrer en priorité)

Query exécutée sur `pdb.Player`, `pdb.SharedSocial`, ou `pdb.Metadata` avec un `shared.X` qui ne résout plus. Chaque ligne = un site à migrer vers `pdb.SharedReadDB().Get(ctx)` + retrait du préfixe `shared.`.

### 1.1 — Média (Q37) — phase P1

| Fichier | Lignes | Symbole | Conn |
|---|---|---|---|
| `internal/platform/duckdb/queries_home_citations.go` | 466-472 | `q37LegacyMediaFromClause`, `q37SharedSocialFromClause` | utilisés par `LoadMediaFiles`/`CountMediaFiles`/`LoadMediaFilterOptions` (×3) sur `r.socialDB()` = SharedSocial |

Particularité : Q37 mixe `media_files` (SharedSocial) + `media_match_associations` (SharedSocial) + `shared.match_registry`. Requiert un refactor en 2 phases Go (cf. plan P1).

### 1.2 — Post-sync hook — phase P2

| Fichier | Lignes | Fonction | Conn |
|---|---|---|---|
| `internal/api/post_sync_progression_queries.go` | 56-57 | `loadProgressionMatches` | `pdb.Player` |
| `internal/api/post_sync_progression_queries.go` | 134 | `loadPlayerStats` (count) | `pdb.Player` |
| `internal/api/post_sync_progression_queries.go` | 145-146 | `loadPlayerStats` (KDA join) | `pdb.Player` |
| `internal/api/post_sync_progression_queries.go` | 182-183 | `loadComebackContext` | `pdb.Player` |
| `internal/api/post_sync_deltas.go` | 280 | `SnapshotPlayerState` KD count | `pdb.ReadDB()` = Player |
| `internal/api/post_sync_deltas.go` | 299 | `SnapshotPlayerState` winrate | `pdb.ReadDB()` = Player |

Toutes ces queries lisent **uniquement** des tables shared → migration directe vers SharedReader, retrait du préfixe.

### 1.3 — Engagement score repo — phase P3

| Fichier | Lignes | Fonction | Conn |
|---|---|---|---|
| `internal/platform/duckdb/engagement_score_repo_queries.go` | 43-44 | `LoadMatchEngagementContext` (registry+participants) | `pdb.ReadDB()` |
| `internal/platform/duckdb/engagement_score_repo_queries.go` | 72 | `LoadMatchEngagementContext` (participants only) | `pdb.ReadDB()` |
| `internal/platform/duckdb/engagement_score_repo_queries.go` | 97 | `LoadEventsForMatch` (highlight_events) | `pdb.ReadDB()` |
| `internal/platform/duckdb/engagement_score_repo_queries.go` | 129 | `LoadTeamXUIDs` (participants) | `pdb.ReadDB()` |
| `internal/platform/duckdb/engagement_score_repo_queries.go` | 209-210 | `ListRecentPvPMatchIDs` (registry+participants) | `pdb.ReadDB()` |

### 1.4 — Progression / profile narrative — phase P4

| Fichier | Lignes | Fonction | Conn |
|---|---|---|---|
| `internal/progression/profile/queries.go` | 31-32 | `countMatchesInWindow` | `s.db` = `pdb.Player` |
| `internal/progression/profile/queries.go` | 93-94 | `computeRadarAxesBase` | `s.db` |
| `internal/progression/profile/queries.go` | 139 | `applyAwardsRadarAxes` (join `personal_score_awards` × `match_registry`) | `s.db` |
| `internal/progression/profile/queries.go` | 180-181 | `findFirstKillFirstDeathHighlights` (`highlight_events` × `match_registry`) | `s.db` |
| `internal/progression/profile/queries.go` | 204-205 | Autre query agrégat | `s.db` |

Particularité 139 : la query joint `personal_score_awards` (player) avec `match_registry` (shared) → **cross-DB**, à scinder en 2 phases Go.

### 1.5 — Match navigation (voisins) — phase P5

| Fichier | Lignes | Symbole | Conn |
|---|---|---|---|
| `internal/platform/duckdb/queries_match.go` | 328-330 | `Q25NeighborMatches` const | exécuté dans `match_view_repo.go:711` sur `r.pdb.ReadDB()` |
| `internal/platform/duckdb/queries_match.go` | 359-361 | `Q25NeighborMatchesTemplate` const | exécuté dans `match_view_repo.go:757` sur `r.pdb.ReadDB()` |
| `internal/analysis/match_filter.go` | 143 | `BuildNeighborsWhereClause` génère `EXISTS (SELECT 1 FROM shared.match_participants mp2 ...)` | injecté dans le template ci-dessus → CASSE par transitivité |

### 1.6 — Career repo — phase P5

| Fichier | Lignes | Fonction | Conn |
|---|---|---|---|
| `internal/platform/duckdb/career_repo.go` | (à compléter) | `GetTopEncountersGlobal`, `GetRivals` | à vérifier en P5 |

(Vérification en début de P5 — les commentaires lignes 370 et 468 mentionnent `shared.X` mais le code peut déjà être migré via SharedReader.)

---

## 2. À NETTOYER — préfixe `shared.` inutile sur conn SharedReader

Query déjà exécutée sur la conn shared (via `pdb.SharedReadDB().Get(ctx)`) mais garde le préfixe `shared.X`. Fonctionne en prod parce que DuckDB-Go résout par auto-attach interne (à confirmer), mais **non documenté et fragile**.

| Fichier | Lignes | Fonction |
|---|---|---|
| `internal/platform/duckdb/match_history_repo.go` | 246-247 | `LoadMapWinRates` |
| `internal/platform/duckdb/bootstrap_repo.go` | 92 | `GetPlayerCount` |

Action P6 : retirer le préfixe `shared.` dans ces deux queries.

---

## 3. SAFE — Commentaires uniquement (pas d'exécution)

Aucune query SQL — uniquement de la documentation. Aucune migration nécessaire.

- `internal/sync/citations.go` (lignes 6, 7, 9, 205, 336)
- `internal/sync/engagement.go` (lignes 8, 161, 360, 486)
- `internal/sync/session_recalc.go` (ligne 94)
- `internal/sync/skill_heal.go` (ligne 6)
- `internal/sync/engine.go` (lignes 553, 946)
- `internal/service/compare_service.go` (lignes 98, 179)
- `internal/service/match_exclusion_service.go` (ligne 40)
- `internal/service/openspartan_import_service.go` (ligne 353)
- `internal/service/media_index_service.go` (0 occurrence)
- `internal/ops/media.go` (0 occurrence)
- `internal/domain/compare.go`, `citations.go`, `match_exclusion.go`, `match_view.go`
- `internal/games/canonical/enums.go`
- `internal/openspartan/mapper/rows.go`
- `internal/api/registry_notifications.go:145` (commentaire ; le code l. 165 utilise `FROM match_participants` sans préfixe sur conn dédiée — OK)
- `internal/port/services.go`, `repository.go`, `highlight_events.go`, `engagement_score.go`
- `internal/migration/steps_engagement.go`
- `internal/platform/duckdb/career_repo.go` (lignes 370, 468 — commentaires en attendant vérif P5)

---

## 4. SAFE — `cmd/` one-shot avec ATTACH manuel

Outils CLI qui ouvrent leur propre conn DuckDB avec ATTACH explicite — hors scope du pool runtime.

- `cmd/backfill_registry_names/main.go`
- `cmd/cleanup_orphan_match/main.go`
- `cmd/diag_medals/main.go`
- `cmd/diag_orphan_session/main.go`
- `cmd/diag_weapon_citations/main.go`
- `cmd/diag_squad_weapons/main.go`
- `cmd/migrate-xuid-aliases-global/main.go`
- `cmd/populate-playlists-catalog/main.go`
- `cmd/repair_data_consistency/main.go`

---

## 5. Tests à corriger (mensonge de topologie)

Tests qui appellent `seedSharedDBSchema(t, social)` ou similaire — créent `CREATE SCHEMA shared` directement dans une conn qui ne devrait pas l'avoir. Faux positifs : les tests passent mais le bug existait en prod.

| Fichier | Faille |
|---|---|
| `internal/platform/duckdb/media_repo_filters_test.go:47` | `seedSharedDBSchema(t, social)` — la conn social a un faux `shared.match_registry` |
| `internal/platform/duckdb/player_repos_test.go:1395, 1416, 1429` | Tests media legacy — vérifier topologie |
| `internal/api/post_sync_progression_test.go:117, 124` | `INSERT INTO shared.X` — sur quelle conn ? À vérifier |
| `internal/progression/profile/build_profile_integration_test.go:117, 123` | idem |
| `internal/platform/duckdb/campaign_repo_test.go:33, 36, 91` | À vérifier |
| `internal/platform/duckdb/match_view_repo_meta_test.go:371` | À vérifier |
| `internal/platform/duckdb/match_view_repo_neighbors_filtered_test.go:55` | À vérifier |
| `internal/analysis/match_filter_test.go:193` | Test pur (string SQL) — pas de DB, OK |

---

## 6. Synthèse par phase

| Phase | Sites concernés | Effort |
|---|---|---|
| P0 | Audit (ce doc) + fix mock `media_repo_filters_test.go` + test sentinel | léger |
| P1 | Q37 média (3 méthodes) | lourd (cross-DB) |
| P2 | `post_sync_progression_queries.go` (4 fcts) + `post_sync_deltas.go` (1 fct) | léger |
| P3 | `engagement_score_repo_queries.go` (4 fcts) | léger |
| P4 | `progression/profile/queries.go` (~5 fcts) | moyen (1 query cross-DB) |
| P5 | `Q25NeighborMatches*` + `match_filter.go` + `career_repo.go` (à confirmer) | moyen |
| P6 | À nettoyer (2 sites) + test E2E + cleanup commentaires | léger |

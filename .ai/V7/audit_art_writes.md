# Audit complet — Écritures à risque ART dans apps/go-api/internal/

**Date** : 2026-05-24
**Méthode** : grep exhaustif `DELETE FROM`, `UPDATE`, `ON CONFLICT DO UPDATE`, `INSERT OR REPLACE` sur `apps/go-api/internal/` (hors `*_test.go`, hors `internal/migration/` boot-only, hors `internal/ops/seed*` cold-path).

---

## 0. Hypothèse de travail

Le bug DuckDB ART ("Failed to delete all rows from index") est déclenché par **toute opération qui touche un index lors d'un DELETE** :

- `DELETE FROM`
- `UPDATE` qui modifie une colonne indexée (incluant la PK)
- `INSERT ... ON CONFLICT DO UPDATE` (DuckDB l'internalise en DELETE + INSERT)
- `INSERT OR REPLACE` (DuckDB documente cela comme sucre syntaxique de `ON CONFLICT DO UPDATE`)

Ne sont **pas** à risque :

- `INSERT` pur
- `INSERT OR IGNORE` (skip silencieux, pas de DELETE)
- `INSERT ... ON CONFLICT DO NOTHING`
- `UPDATE` qui modifie uniquement des colonnes **non indexées** (mais le test pour s'en assurer est non trivial → considérer comme à risque par défaut)

Le bug est **non déterministe** : il ne se reproduit pas systématiquement en `:memory:`. Il dépend de la combinaison données + index ART + timing. Les logs de prod sont l'unique source fiable.

---

## 1. Corrections aux idées reçues (importantes)

### 1.1 — La phrase de CLAUDE.md est fausse

CLAUDE.md affirme :
> "les backfills 'UPDATE-style' qui ne touchent PAS shared.match_participants (LUSR, citations, engagement, etc.) peuvent rester UPDATE car non concernés par l'ART bug"

**Démenti par les logs de prod 2026-05-24 20:41:04** : crash ART exact sur `match_skill_rank` (table LUSR de la player DB de Chocoboflor), avec la signature canonique `Failed to delete all rows from index. Only deleted 0 out of 1 rows`. La phrase doit être supprimée de CLAUDE.md.

### 1.2 — Le pattern "DELETE batch + INSERT batch en TX" n'est PAS un fix

Le `PostSyncLUSRPersister.Upsert` ([post_sync_lusr_persister.go:121-168](../apps/go-api/internal/persist/post_sync_lusr_persister.go#L121)) fait précisément ce pattern :

```go
DELETE FROM match_skill_rank WHERE rating_type='LUSR' AND match_id IN (?, ?, ?)
INSERT INTO match_skill_rank ... (×N)
```

C'est CE chemin qui a crashé ce soir. Le `DELETE` reste un DELETE, indépendamment du batch ou de la TX. Donc proposer ce pattern comme remède pour CSR (cf. agent v1) est faux.

### 1.3 — Le test `art_upsert_patterns_test.go` n'a pas "prouvé" qu'INSERT OR REPLACE est buggué

Le test logge `0 errors` pour les 3 patterns en `:memory:`. Le test comment lui-même reconnaît :

> "Le bug ART n'est PAS facile à reproduire en :memory: — si l'un échoue, on a une indication forte, mais si tous passent, on ne peut pas conclure"

**Verdict prudent** : `INSERT OR REPLACE` est documenté par DuckDB comme sucre syntaxique pour `ON CONFLICT DO UPDATE`. À traiter comme **équivalent en risque** par précaution, même si non démontré ce soir.

### 1.4 — Le seul pattern garanti sans déclencher ART

Le pattern C du test (`SELECT-then-UPDATE-or-INSERT`) est garanti :

```go
SELECT 1 FROM t WHERE pk = ?           -- read, ne touche pas l'index ART
IF exists: UPDATE ... WHERE pk = ?     -- modifie une row in-place, pas la PK → safe
ELSE:      INSERT ... VALUES (...)     -- ajoute une entry, jamais delete
```

**Limite** : 2 round-trips par row, donc inadapté aux gros batchs.

### 1.5 — Pattern alternatif aussi pérenne : schema append-only

Au lieu de tenter d'éviter le DELETE par discipline (fragile), modifier le schema pour qu'aucun DELETE ne soit JAMAIS nécessaire :

- Ajout d'une colonne `written_at TIMESTAMP NOT NULL DEFAULT now()`
- PK élargie pour inclure `written_at` (plus de conflit possible → pas besoin de DELETE)
- Lecture via vue `*_latest` qui prend le `MAX(written_at)` par clé fonctionnelle

C'est le pattern le plus pérenne — bug ART **impossible par construction** sur ces tables.

---

## 2. Inventaire complet — hot paths runtime à risque

Sont listés : tous les sites qui s'exécutent pendant un sync, un post-sync, un handler HTTP, ou un scheduler. Sont exclus : migrations boot-only, seeds, scripts CLI.

### 2.1 — Player DB (`data/players/{gt}/stats.duckdb`)

| Fichier:ligne | Table | Pattern | Hot path | Persister ? | Risque |
|---|---|---|---|---|---|
| [persist/post_sync_lusr_persister.go:77](../apps/go-api/internal/persist/post_sync_lusr_persister.go#L77) | `match_skill_rank` | DELETE WHERE rating_type='LUSR' + INSERT batch en TX | post-sync LUSR | partial | **CRITIQUE** (crash confirmé 20:41:04) |
| [persist/post_sync_lusr_persister.go:142](../apps/go-api/internal/persist/post_sync_lusr_persister.go#L142) | `match_skill_rank` | DELETE IN(...) + INSERT batch en TX | post-sync LUSR (`Upsert` appelé via [skill_rating_postsync_persist.go:118](../apps/go-api/internal/sync/skill_rating_postsync_persist.go#L118)) | partial | **CRITIQUE** (chemin confirmé crashé) |
| [sync/csr_writes.go:179](../apps/go-api/internal/sync/csr_writes.go#L179) | `match_skill_rank` (CSR) | ON CONFLICT (match_id) DO UPDATE | sync per-match | non | **CRITIQUE** (même table que LUSR, même classe de bug) |
| [sync/performance.go:641](../apps/go-api/internal/sync/performance.go#L641) | `player_match_enrichment` | ON CONFLICT (match_id) DO UPDATE | post-sync legacy (`LEVELUP_POSTSYNC_INSERT_ONLY=0`) | non | HAUT |
| [sync/comeback.go:180](../apps/go-api/internal/sync/comeback.go#L180) | `player_match_enrichment` | ON CONFLICT (match_id) DO UPDATE SET dominance_flag | post-sync comeback legacy | non | HAUT (fallback si `LEVELUP_POSTSYNC_INSERT_ONLY=0`) |
| [sync/engagement.go:498,519](../apps/go-api/internal/sync/engagement.go#L498) | `player_match_enrichment` | ON CONFLICT (match_id) DO UPDATE | post-sync engagement legacy | non | HAUT (idem) |
| [sync/engagement_recompute.go:215](../apps/go-api/internal/sync/engagement_recompute.go#L215) | `engagement_scores` | ON CONFLICT (xuid, mode_category) DO UPDATE | post-sync | non | HAUT |
| [sync/achievements.go:178,238](../apps/go-api/internal/sync/achievements.go#L178) | `xbox_achievement_definitions` | ON CONFLICT (achievement_id) DO UPDATE | post-sync achievements | non | HAUT |
| [sync/achievements.go:146](../apps/go-api/internal/sync/achievements.go#L146) | `xbox_achievement_definitions` | DELETE WHERE id NOT IN (...) | post-sync achievements | non | HAUT |
| [sync/skill_rating_loaders.go:285](../apps/go-api/internal/sync/skill_rating_loaders.go#L285) | `match_skill_rank` (legacy non-batch) | ON CONFLICT (match_id) DO UPDATE | post-sync LUSR fallback | non | HAUT |
| [sync/skill_rating_loaders.go:358](../apps/go-api/internal/sync/skill_rating_loaders.go#L358) | `lusr_component_history` | ON CONFLICT (match_id, component_name) DO UPDATE | post-sync LUSR (best-effort) | non | MOYEN (commentaire `lusr_persister.go:33` dit "pas critique côté ART") — à valider |
| [sync/citations.go:509](../apps/go-api/internal/sync/citations.go#L509) | `match_citations` | DELETE WHERE match_id=? | post-sync citations | non | MOYEN (row-by-row, fréquence basse) |
| [sync/citations_backfill.go:124,260](../apps/go-api/internal/sync/citations_backfill.go#L124) | `match_citations` | DELETE WHERE match_id=? [AND name=?] | backfill CLI | non | FAIBLE (CLI ad-hoc) |
| [sync/writes.go:329](../apps/go-api/internal/sync/writes.go#L329) | `weapon_kills` | DELETE WHERE match_id=? AND xuid=? | sync per-match | non | MOYEN |
| [sync/writes.go:399](../apps/go-api/internal/sync/writes.go#L399) | `personal_score_awards` | DELETE WHERE match_id=? AND xuid=? | sync per-match | non | MOYEN |
| [sync/writes.go:67](../apps/go-api/internal/sync/writes.go#L67) | `match_registry` (player local) | ON CONFLICT (match_id) DO UPDATE | sync per-match | non | HAUT (note : si cette ligne touche la table player et non shared, à vérifier) |
| [sync/writes.go:131](../apps/go-api/internal/sync/writes.go#L131) | (à vérifier) | ON CONFLICT (match_id, xuid) DO UPDATE | sync per-match | non | HAUT |
| [sync/writes.go:284](../apps/go-api/internal/sync/writes.go#L284) | `player_match_enrichment` | UPDATE | sync per-match | non | HAUT |
| [sync/enrichments.go:75](../apps/go-api/internal/sync/enrichments.go#L75) | `player_match_enrichment` | UPDATE | post-sync | non | HAUT |
| [sync/friends_recompute.go:231](../apps/go-api/internal/sync/friends_recompute.go#L231) | `player_match_enrichment` | UPDATE | post-sync friends | non | HAUT (cascade confirmée dans les logs après FATAL LUSR) |
| [sync/career.go:218](../apps/go-api/internal/sync/career.go#L218) | `player_csr_snapshots` | INSERT OR REPLACE | sync career | non | HAUT |
| [api/post_sync_deltas.go:676](../apps/go-api/internal/api/post_sync_deltas.go#L676) | `player_records` (à confirmer) | ON CONFLICT (xuid, metric) DO UPDATE | post-sync | non | HAUT |

### 2.2 — Shared DB (`shared_matches_v2.duckdb`)

| Fichier:ligne | Table | Pattern | Hot path | Persister ? | Risque |
|---|---|---|---|---|---|
| [sync/csr_shared_writes.go:112](../apps/go-api/internal/sync/csr_shared_writes.go#L112) | `match_csrs` | ON CONFLICT (match_id, xuid) DO UPDATE | sync per-match | non | **CRITIQUE** |
| [sync/writes.go:372,509,522,568,581](../apps/go-api/internal/sync/writes.go) | `match_registry` | UPDATE | sync per-match | non | HAUT (5 sites) |
| [sync/writes.go:561](../apps/go-api/internal/sync/writes.go#L561) | `match_participants` | UPDATE | sync per-match | non | **CRITIQUE** (table déjà connue ART-sensible) |
| [sync/writes.go:214](../apps/go-api/internal/sync/writes.go#L214) | (à confirmer xuid_aliases ?) | ON CONFLICT (xuid) DO UPDATE | sync | non | MOYEN |
| [sync/writes.go:238](../apps/go-api/internal/sync/writes.go#L238) | (à confirmer) | ON CONFLICT (match_id) DO UPDATE | sync | non | HAUT |
| [sync/writes.go:483](../apps/go-api/internal/sync/writes.go#L483) | `killer_victim_pairs` | DELETE WHERE match_id=? | sync per-match | non | MOYEN |
| [sync/events_replay.go:225](../apps/go-api/internal/sync/events_replay.go#L225) | `match_registry` | UPDATE | events replay | non | HAUT |
| [sync/pve.go:303](../apps/go-api/internal/sync/pve.go#L303) | `match_registry` | UPDATE | sync PvE | non | HAUT |
| [sync/engagement.go:544](../apps/go-api/internal/sync/engagement.go#L544) | `match_registry` | UPDATE match_intensity | post-sync | non | HAUT |
| [sync/backfill_registry_names.go:142](../apps/go-api/internal/sync/backfill_registry_names.go#L142) | `match_registry` | UPDATE | backfill CLI | non | FAIBLE (ad-hoc) |

### 2.3 — PvE DB (`shared_pve.duckdb`)

| Fichier:ligne | Table | Pattern | Hot path | Persister ? | Risque |
|---|---|---|---|---|---|
| [sync/pve.go:267](../apps/go-api/internal/sync/pve.go#L267) | `pve_match_stats` | INSERT OR REPLACE batch | sync PvE | non | HAUT |

### 2.4 — Metadata / catalog / asset cache DB (`metadata.duckdb`)

| Fichier:ligne | Table | Pattern | Hot path | Risque |
|---|---|---|---|---|
| [service/catalog_fetcher_service.go:157,191,209,233,252](../apps/go-api/internal/service/catalog_fetcher_service.go) | `playlists_catalog`, `pair_links`, `maps_catalog`, `game_variants` | ON CONFLICT DO UPDATE | scheduler catalog | MOYEN (writes peu fréquents, mais runtime) |
| [service/catalog_fetcher_service.go:270,283](../apps/go-api/internal/service/catalog_fetcher_service.go#L270) | `catalog_fetch_queue` | UPDATE / DELETE | scheduler catalog | MOYEN |
| [platform/duckdb/metadata_repo.go:201,229,257](../apps/go-api/internal/platform/duckdb/metadata_repo.go) | `seasons_catalog`, `resources_meta` | ON CONFLICT DO UPDATE | runtime metadata | MOYEN |
| [platform/duckdb/metadata_repo.go:389](../apps/go-api/internal/platform/duckdb/metadata_repo.go#L389) | `medal_definitions` | UPDATE | runtime | MOYEN |
| [platform/duckdb/metadata_repo_assets.go:111,365](../apps/go-api/internal/platform/duckdb/metadata_repo_assets.go) | `asset_translations`, `maps_catalog` | ON CONFLICT DO UPDATE | runtime asset cache | MOYEN |
| [platform/duckdb/asset_cache_repo.go:63](../apps/go-api/internal/platform/duckdb/asset_cache_repo.go#L63) | `asset_cache` | ON CONFLICT DO UPDATE | runtime asset cache | MOYEN |
| [platform/duckdb/medal_cache_repo.go:69,88](../apps/go-api/internal/platform/duckdb/medal_cache_repo.go) | `medal_cache` | ON CONFLICT DO UPDATE | runtime medal cache | MOYEN |
| [platform/duckdb/map_cache_repo.go:66](../apps/go-api/internal/platform/duckdb/map_cache_repo.go#L66) | `map_cache` | ON CONFLICT DO UPDATE | runtime map cache | MOYEN |
| [platform/duckdb/persist_sink.go:699](../apps/go-api/internal/platform/duckdb/persist_sink.go#L699) | `asset_cache` | ON CONFLICT DO UPDATE | runtime | MOYEN |
| [platform/duckdb/persist_sink.go:290,297,417,423,488](../apps/go-api/internal/platform/duckdb/persist_sink.go) | `battlepass_*` | UPDATE | runtime BP | MOYEN |
| [platform/duckdb/prestige_metadata_repo.go:117,289,307](../apps/go-api/internal/platform/duckdb/prestige_metadata_repo.go) | prestige metadata | ON CONFLICT DO UPDATE | runtime | MOYEN |

### 2.5 — Social / Prestige DB (`social.duckdb`)

| Fichier:ligne | Table | Pattern | Hot path | Risque |
|---|---|---|---|---|
| [platform/duckdb/streaks_repo.go:70](../apps/go-api/internal/platform/duckdb/streaks_repo.go#L70) | `streaks` | ON CONFLICT (id) DO UPDATE | handler HTTP + sync | HAUT (table déjà observée en crash `list_streaks_error`) |
| [platform/duckdb/records_repo.go:91](../apps/go-api/internal/platform/duckdb/records_repo.go#L91) | `player_records` | ON CONFLICT (xuid, metric, period) DO UPDATE | post-sync records | HAUT |
| [platform/duckdb/prestige_player_repo.go:348](../apps/go-api/internal/platform/duckdb/prestige_player_repo.go#L348) | `player_progress` | ON CONFLICT (user_id, title_slug, metric) DO UPDATE | handler HTTP | MOYEN |
| [platform/duckdb/prestige_player_repo.go:92,95,103,118,222](../apps/go-api/internal/platform/duckdb/prestige_player_repo.go) | `challenge`, `arc` | UPDATE | handler HTTP | MOYEN |
| [platform/duckdb/prestige_social_repo.go:54,93,376](../apps/go-api/internal/platform/duckdb/prestige_social_repo.go) | `prestige_status`, `squad_challenge_participant` | ON CONFLICT DO UPDATE | handler HTTP | MOYEN |
| [platform/duckdb/prestige_social_repo.go:227](../apps/go-api/internal/platform/duckdb/prestige_social_repo.go#L227) | `squad_member` | DELETE | handler HTTP | FAIBLE |
| [platform/duckdb/match_exclusion_repo.go:40](../apps/go-api/internal/platform/duckdb/match_exclusion_repo.go#L40) | `match_exclusions` | ON CONFLICT (match_id) DO UPDATE | handler HTTP | MOYEN |
| [platform/duckdb/engagement_score_repo.go:230](../apps/go-api/internal/platform/duckdb/engagement_score_repo.go#L230) | `engagement_scores` | ON CONFLICT (xuid, mode_category) DO UPDATE | handler HTTP | MOYEN |
| [platform/duckdb/engagement_score_repo.go:157,180](../apps/go-api/internal/platform/duckdb/engagement_score_repo.go) | `player_match_enrichment` | UPDATE | handler HTTP | HAUT |
| [platform/duckdb/notifications_repo.go:417](../apps/go-api/internal/platform/duckdb/notifications_repo.go#L417) | `notification_prefs` | ON CONFLICT (xuid, category) DO UPDATE | handler HTTP | MOYEN |
| [platform/duckdb/notifications_repo.go:242,276,280,341](../apps/go-api/internal/platform/duckdb/notifications_repo.go) | `player_notifications` | UPDATE / DELETE | handler HTTP | MOYEN |
| [platform/duckdb/social_repo.go:50](../apps/go-api/internal/platform/duckdb/social_repo.go#L50) | `match_favorites` | DELETE | handler HTTP | FAIBLE |
| [platform/duckdb/media_repo.go:610,654,690,714,725,746,753](../apps/go-api/internal/platform/duckdb/media_repo.go) | `media_files`, `media_likes`, `media_match_associations` | UPDATE / DELETE / ON CONFLICT | handler HTTP | MOYEN |
| [platform/duckdb/privacy_state_repo.go:41](../apps/go-api/internal/platform/duckdb/privacy_state_repo.go#L41) | `privacy_state` | ON CONFLICT (xuid) DO UPDATE | handler HTTP | MOYEN |

### 2.6 — Auth / sync_meta

| Fichier:ligne | Table | Pattern | Hot path | Risque |
|---|---|---|---|---|
| [sync/writes.go:255](../apps/go-api/internal/sync/writes.go#L255) | `sync_meta` | ON CONFLICT (key) DO UPDATE | sync | MOYEN |
| [platform/duckdb/queries_auth.go:18](../apps/go-api/internal/platform/duckdb/queries_auth.go#L18) | (auth tokens) | ON CONFLICT(key) DO UPDATE | runtime auth | MOYEN |
| [api/notifications_boot.go:121](../apps/go-api/internal/api/notifications_boot.go#L121) | (config kv) | ON CONFLICT (key) DO UPDATE | boot | FAIBLE |

---

## 3. Sites déjà sûrs (confirmés)

- **`internal/persist/BatchBuilder` + `SharedPersister` + `PlayerPersister` + `PVEPersister` + `MetadataPersister`** : tous INSERT-only (`INSERT OR IGNORE` ou INSERT pur), cf. [persist/doc.go](../apps/go-api/internal/persist/doc.go)
- **ON CONFLICT DO NOTHING** : idempotent par construction, pas de DELETE déclenché
  - `xuid_aliases` ([shared_persister.go:317](../apps/go-api/internal/persist/shared_persister.go#L317))
  - `media_likes` insert ([media_repo.go](../apps/go-api/internal/platform/duckdb/media_repo.go))
  - `match_favorites` ([social_repo.go:42](../apps/go-api/internal/platform/duckdb/social_repo.go#L42))
  - `milestones_earned` ([milestones_earned_repo.go:62](../apps/go-api/internal/platform/duckdb/milestones_earned_repo.go#L62))
  - `citations` insert ([citations_repo.go:308](../apps/go-api/internal/platform/duckdb/citations_repo.go#L308))
- **Migrations** : `internal/migration/steps_*.go` — boot-only, pas exposé à la concurrence runtime
- **Seeds CLI** : `internal/ops/seed*.go` — exécution rare, hors hot path

---

## 4. Top 10 à traiter en priorité

Ordonné par criticité (crash observé d'abord, puis fréquence × tables critiques) :

| # | Site | Action recommandée |
|---|------|-------------------|
| 1 | `post_sync_lusr_persister.go` (LUSR DELETE+INSERT) | Schema append-only sur `match_skill_rank` (jamais de DELETE) |
| 2 | `csr_writes.go:179` (CSR player) | Idem — même table `match_skill_rank` |
| 3 | `csr_shared_writes.go:112` (CSR shared) | Schema append-only sur `match_csrs` |
| 4 | `sync/writes.go:561` (UPDATE match_participants) | Vérifier si chemin batchMode ou legacy. Si legacy, router par persister |
| 5 | `sync/performance.go:641` (ON CONFLICT player_match_enrichment legacy) | Supprimer le path legacy (forcer batch mode toujours) |
| 6 | `sync/comeback.go:180` (idem legacy) | Idem |
| 7 | `sync/engagement.go:498,519` (idem legacy) | Idem |
| 8 | `sync/career.go:218` (INSERT OR REPLACE `player_csr_snapshots`) | Schema append-only |
| 9 | `sync/pve.go:267` (INSERT OR REPLACE `pve_match_stats`) | Schema append-only ou routing PVEPersister |
| 10 | `platform/duckdb/streaks_repo.go:70` (ON CONFLICT streaks) | Pattern C (SELECT-then-UPDATE-or-INSERT) — basse fréquence handler |

---

## 5. Stratégie unifiée

Trois patterns d'action selon le contexte :

### Stratégie A — Schema append-only + vue latest
**Quand** : table avec ré-écritures fréquentes pendant le sync (LUSR, CSR, performance, comeback, engagement, pve, career).
**Recette** :
1. `ALTER TABLE x ADD COLUMN written_at TIMESTAMP NOT NULL DEFAULT now()`
2. Nouvelle PK incluant `written_at` (ou suppression de la PK + UNIQUE)
3. `CREATE VIEW x_latest AS SELECT … QUALIFY ROW_NUMBER() OVER (PARTITION BY <clé fonctionnelle> ORDER BY written_at DESC) = 1`
4. Toutes les écritures deviennent INSERT pur, jamais de DELETE/UPDATE
5. Toutes les lectures pointent sur `x_latest`

### Stratégie B — Routage via persist.BatchBuilder
**Quand** : besoin de cohérence transactionnelle avec d'autres tables, ou orchestration centralisée souhaitée.
**Recette** : créer `*Persister` côté `internal/persist/`, brancher via `BatchBuilder.Submit`. INSERT-only par contrat.

### Stratégie C — SELECT-then-UPDATE-or-INSERT
**Quand** : table handler HTTP basse fréquence (streaks, records, prestige), où la perte de perf 2× est négligeable.
**Recette** : remplacer `ON CONFLICT DO UPDATE` par check d'existence + branchement INSERT/UPDATE explicite (pattern C du test).

### Stratégie D — Garder ON CONFLICT DO NOTHING
**Quand** : sémantique idempotente, "écrire si absent". Pas de DELETE déclenché.
**Application** : pour les sites qui font de la déduplication pure (xuid_aliases, citations insert, etc.) — déjà appliquée.

---

## 6. Guard-rail anti-régression

Ajouter un test `apps/go-api/internal/sync/no_art_patterns_test.go` qui :
- Grep `ON CONFLICT.*DO UPDATE`, `INSERT OR REPLACE`, `DELETE FROM` dans tout `apps/go-api/internal/sync/`, `apps/go-api/internal/persist/`, `apps/go-api/internal/platform/duckdb/` (hors `*_test.go`)
- Compare la liste à une allowlist déclarée explicitement dans le test (sites volontairement laissés tels quels avec justification commentée)
- Fail si un nouveau site apparaît hors allowlist

Cela rend impossible l'ajout silencieux d'un nouveau pattern à risque sans documentation.

---

## 7. Récapitulatif

- **Total sites runtime à risque** : ~50 (hors metadata cache et social handler HTTP basse fréquence)
- **Sites CRITIQUES** (player DB hot path) : 9
- **Sites HAUT** (player/shared DB hot path) : ~20
- **Sites MOYEN** (cache metadata, handlers HTTP) : ~20
- **Sites déjà sûrs** (BatchBuilder, ON CONFLICT DO NOTHING) : ~15
- **Sites cold-path acceptables** (migrations, seeds, CLI) : ~50

L'éradication exhaustive du bug demande de traiter au minimum **les 9 CRITIQUES + les 20 HAUT** par les stratégies A ou B. Les MOYENS peuvent être traités plus tard par C, après audit volumétrique de chacun.

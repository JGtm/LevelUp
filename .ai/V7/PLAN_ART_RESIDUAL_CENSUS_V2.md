# PLAN — Surfaces ART résiduelles (census v2 exhaustif, 2026-06-20)

> Issu du workflow ultracode `art-residual-census-v2` (29 agents, 143 sites scannés, 95 tables
> indexées cartographiées, skeptiques adversariaux). Le fix prod (catalogue/battlepass + shared_social)
> est DÉJÀ DÉPLOYÉ et stable (commit `d434ed38c`). Ce qui suit = le DURCISSEMENT restant.
> Branche : `fix/metadata-art-battlepass-appendonly`.

## Cause racine (rappel)

Bug DuckDB amont **#23046** (régression 1.5.0, non corrigé en 1.5.4) : l'enforcement de contrainte
ART corrompt le heap sur DB **fichier**. Vecteurs CONFIRMÉS par les crashs réels :
1. `INSERT … ON CONFLICT DO UPDATE` (delete+insert interne sur l'index).
2. `INSERT OR REPLACE`.
3. `DELETE` per-row sur table à index ART (prouvé : compaction match_skill_rank → crash JGtm).
4. `UPDATE SET <col>` où `<col>` est **indexée** (prouvé : PME engagement-coefs crashe sur colonnes
   engagement_* indexées sous pression).
5. `INSERT` pur qui enforce une PK/UNIQUE métier **sous pression massive**.

**Sérialisé/mono-writer ≠ sûr** (réfuté par match_skill_rank). **Nuance** (jugement, non prouvé) : un
UPDATE NU sur une colonne **non-indexée** (WHERE = PK, point-update, basse fréquence) n'a PAS de preuve
directe de crash ; les skeptiques l'ont flaggé par extrapolation. Priorité moindre, mais à terme
append-only. Les UPDATE **bulk multi-row** (ex friends_recompute `IN(...)` 500 lignes) et **haute
pression** (PME) restent à convertir même sur colonne non-indexée.

## Work-list priorisée (blast-radius × confiance)

### P0 — shared_social / shared_matches (blast MAX) — vecteurs CERTAINS
- **media_files** (shared_social) — UPDATE mutant `file_path` (UNIQUE) et/ou `kind` (idx_mf_kind) :
  `ops/media.go:455` (insertMediaFile conversion), `ops/media_hls.go:106` (finalizeMediaHLS),
  `ops/media_reconcile.go:87`. → muter une colonne indexée = surface ART. Action : retirer la
  contrainte UNIQUE sur file_path + idx_mf_kind (rebuild) OU garder file_path immuable + état
  transcode/kind en colonne non-indexée / sidecar. (`media_hls.go:93` MarkTranscodeStatus =
  transcode_status non-indexé → P3-bis.) EFFORT moyen.
- **match_registry** `sync/writes.go:67` InsertRegistry ON CONFLICT (mut season_id indexé). Action :
  SELECT-then-write (no-op season_id si non-NULL) OU router via persister INSERT-only (déjà défaut).
- **match_participants** `sync/writes.go:132` insertParticipantRow ON CONFLICT (mut team_id indexé,
  concurrency-load-bearing — le batch persist INSERT-only est le défaut). `sync/writes.go:478`
  MarkSkillLoaded UPDATE backfill_bits (idx_mp_backfill). Action : persister INSERT-only / drop idx_mp_backfill.

### P1 — player DB, vecteurs CERTAINS (ON CONFLICT / UPDATE colonne indexée)
- **lusr_component_history** `sync/skill_rating_loaders.go:347` ON CONFLICT (match_id, component_name) ;
  3 index ART. Action : append-only id-seq + vue `_latest` (comme match_skill_rank) OU SELECT-then-INSERT. EFFORT moyen.
- **coach_proposal** `platform/duckdb/coach_proposal_repo.go:152/161/171/180` (MarkAccepted/Dismissed/
  Superseded/Obsoleted) UPDATE `status` indexé (idx_coach_proposal_user_status). Action : DROP INDEX
  (comme idx_ch_user_status) + garde-fou. EFFORT rapide.
- **engagement_coefficients** : a un index secondaire `idx_engagement_coefficients_xuid` (steps_engagement.go:85)
  → ma conversion SELECT-then-write garde un UPDATE qui re-touche cet index. Action : drop idx_xuid
  (la PK (xuid,mode_category) couvre déjà xuid) OU append-only. EFFORT rapide.
- **challenge** `prestige_player_repo.go:92` UpdateStatus (status), `:166` DetachFromArc (arc_id),
  `campaign_repo.go:141/144` LinkChallenge (campaign_id) : colonnes ÉTAIENT indexées → la campagne a
  DROPpé les index (drop_challenge_mutated_art_indexes_v1). Action : CONFIRMER que cette migration
  s'applique au boot AVANT tout write + garde anti-recréation ; sinon table d'association append-only. EFFORT rapide (validation).

### P2 — player DB, DELETEs reconcile (vecteur #3 certain)
- **match_citations** : `sync/citations.go:546` deleteCitationForMatch + `sync/citations_backfill.go:275`
  (DELETE-then-INSERT par match). Action : append-only `_history`+`_latest` ou recompute sans DELETE per-row. moyen.
- **weapon_kills** `sync/writes.go:329` InsertWeaponKills (DELETE WHERE match_id,xuid puis INSERT). Action : append-only versionné. moyen.
- **personal_score_awards** `sync/writes.go:399` (DELETE WHERE match_id,xuid puis INSERT). Action : append-only `_latest`. moyen.
- **challenge** `prestige_player_repo.go:173` DeleteByArc, **arc** `:243` Delete. Action : soft-delete tombstone + vue `_latest`. moyen.
- **player_skill_state_v2** `sync/lusr_full_recompute.go:34` reset watermark DELETE WHERE xuid (seul trou
  de la conversion append-only de cette table). Action : écrire une ligne watermark 'reset' (append) au lieu de DELETE. moyen.

### P3 — player_match_enrichment (PME, lourd, cf .ai/PLAN_PME_ART_HARDENING.md Option A)
Table à **4 index ART** (PK match_id + idx_pme_session + idx_pme_engagement_history + idx_pme_engagement_paces),
écrite par MULTIPLES writers incrémentaux : ON CONFLICT (`writes.go:254` UpsertPlayerEnrichment,
`match_exclusion_repo.go:52` SetExclusion) + UPDATE per-match/bulk (`convergence.go:148` psa_checked_at,
`enrichments.go:202` had_bot, `friends_recompute.go:245/319` is_with_friends BULK IN(500), engagement,
sessions). → append-only `_history` + vue `_latest` avec **merge-on-read** (dernière valeur non-NULL
par colonne). Palliatif transitoire pour les bulk UPDATE : N UPDATE row-by-row via
`PostSyncEnrichmentPersister` (1 entrée ART/statement, pattern ADR 0019 déjà en place). EFFORT lourd.

### P4 — match_registry completion bit-ledger (lourd)
`backfill_completed` / `events_loaded` UPDATE en place (colonnes non-indexées MAIS table à 3 index ART) :
`sync/writes.go:372/485/498`, `sync/pve.go:306`, `sync/events_replay.go:224`,
`persist/events_completion_persister.go:159/250`. Vecteur P4 (UPDATE nu non-indexé = extrapolation, mais
volume + handle partagé). Action : bit-ledger append-only (`_history` des bits + vue `_latest` agrégée OR).
EFFORT lourd (refonte du modèle de complétion).

### P5 — CLI (ops/, hors hot path mais vecteur sur handle partagé)
- **archive.go:189** DELETE match_participants — fenêtre maintenance offline OU archive-par-copie + table-swap.
- **restore.go:223** DELETE générique pré-restore — DROP TABLE + CREATE + INSERT read_parquet (index propre) + whitelist.

## Garde-fous à renforcer (gaps détectés)
- `TestNoBulkMultiRowUpdateOnCriticalTables` ne matche que `UPDATE…FROM (VALUES…)` → MANQUE la forme
  `UPDATE…WHERE … IN (?,…)` (friends_recompute, backfill_registry_names). Étendre la regex.
- Étendre `metadata_art_surface_guard` / `no_art_patterns` aux nouveaux index droppés (idx_coach_proposal_user_status, idx_engagement_coefficients_xuid, idx_mp_backfill).

## État (2026-06-20)
Fait ce cycle : metadata + shared_social (catalogue/battlepass/notifs/prefs/favorites/likes/media_assoc/
squad×2/user_prestige/ref-cache) DÉPLOYÉS ; match_skill_rank compaction retirée ; player basse-fréquence
(sync_meta, assists_model, engagement_coefficients ON CONFLICT) ; baseline/privacy. RESTE = ce plan (P0→P5).

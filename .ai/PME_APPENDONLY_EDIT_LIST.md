# PME append-only — liste d'édition consolidée + décisions design (2026-06-21)

> Source : census ultracode v3 (5 agents, run wf_09c14567-bac, verdict **GAPS_FOUND** — gaps rattrapés et intégrés ici).
> Branche : `refactor/art-pme-appendonly`. Conversion **ATOMIQUE** (writers+readers+fixtures+schémas+rebuilders+garde-fous en un bloc cohérent ; pas de vert intermédiaire car la conversion de `schema.go` cascade sur toutes les fixtures `EnsurePlayerSchema`).
> ⚠️ NE PAS merger sur `main` avant `go test ./...` vert + go/no-go. `main` auto-deploy (leçon incident).

## Décisions design (tranchées sur pièce)

1. **Stage `'live'` baseline** — le collect live (`player_persister.persistEnrichment`) émet UNE row `stage='live'` (PAS de split par stage). La vue merge devient, par colonne owner-stage `X` :
   `CASE WHEN has('X') THEN value('X') ELSE COALESCE(value('live'), value('legacy')) END`.
   Le live est une baseline multi-colonnes (comme `legacy` mais pour les matchs collectés après migration). Évite le split multi-stage du persister live (risque #1 du census) tout en préservant le reset-NULL par owner-stage.
2. **Ancre d'idempotence sous-batch** — `player_persister.Persist` garde EXISTS mais ciblée `WHERE match_id=? AND stage='live'` (au lieu de `WHERE match_id=?`). Préserve EXACTEMENT la sémantique batch (skip re-persist de skill_rank/lusr/citations/psa/career — sinon conflit PK match_citations + doublons PSA/career).
3. **`mode_category` reste engagement-owned** — seul writer ACTIF = `sync/engagement.go` (écrit toujours mode_category). `SaveEngagementScore` (HTTP) **n'a aucun caller prod** (port + mock seulement) ; converti en INSERT `stage='engagement'` portant `mode_category` via scalar-subquery depuis `_latest` (zéro régression si re-câblé). Les deux engagement writers écrivent alors le même jeu de colonnes.

## Ordre d'exécution (PME-A → PME-I)

### PME-A — Vue + migration (FONDATION, déjà 90% livrée)
- `steps_player_append_only_match_enrichment.go` : `buildPMELatestViewSQL()` → ajouter le fallback `COALESCE(live, legacy)` (au lieu de `legacy` seul) pour les colonnes owner-stage ET pour les baseline-only (known_teammates_count, friends_xuids). `pmeColumnStage` inchangé (mode_category reste 'engagement').
- `steps_player_append_only_match_enrichment_test.go` : ajouter test `stage='live'` baseline + owner override sur live. Re-valider les 8 tests existants (CGO).

### PME-B — Schémas naissance append-only
- `internal/sync/schema.go:37` (playerSchemaSQL / EnsurePlayerSchema) : réécrire le bloc PME sur modèle match_skill_rank (CREATE SEQUENCE pme_seq ; id BIGINT DEFAULT nextval PRIMARY KEY ; match_id VARCHAR NOT NULL ; toutes colonnes ; stage VARCHAR DEFAULT 'legacy' ; written_at TIMESTAMP NOT NULL DEFAULT now()). SUPPRIMER `idx_pme_session`. Ajouter `idx_pme_match_lookup(match_id, written_at)` + la vue `player_match_enrichment_latest`. PAS de PK(match_id), PAS des 3 index ART.
- `internal/migration/steps_player.go:21` (create_base_player_schema) : idem (id seq PK + match_id NOT NULL + stage + written_at, pas de PK match_id). Centraliser le DDL si possible.

### PME-C — Migrations index ART → DROP (empêcher toute re-pose)
- `steps_player.go:288` add_pme_session_index → `DROP INDEX IF EXISTS idx_pme_session` (no-op).
- `steps_engagement.go:62` → retirer createIndexSafe idx_pme_engagement_history (return nil après addColumn).
- `steps_engagement.go:171` → retirer createIndexSafe idx_pme_engagement_paces.

### PME-D — Writers → INSERT pur taggé stage (+ suppression stubs)
| Fichier:ligne | Symbole | Action |
|---|---|---|
| sync/writes.go:248 | UpsertPlayerEnrichment | si sig≠"" → INSERT stage='teammates' ; si "" → no-op (retirer le priming). Vérifier callers engine_fetch.go:238 / engine_process_match.go:152. |
| persist/player_persister.go:64 | Persist EXISTS guard | `WHERE match_id=? AND stage='live'` (décision #2). |
| persist/player_persister.go:186 | persistEnrichment | ajouter `stage='live'` à l'INSERT (1 row, pas de split — décision #1). |
| persist/post_sync_enrichment_persister.go:82 | BatchUpdateColumn | map colonne→stage + INSERT pur taggé (booléen explicite). |
| persist/post_sync_enrichment_persister.go:157 | BatchUpdateMulti | deriveStage(columns homogène) + INSERT pur taggé. |
| platform/duckdb/engagement_score_repo.go:158/181 | SaveEngagementScore | INSERT stage='engagement' (+ mode_category via subquery _latest) ; retirer skip RowsAffected==0. |
| platform/duckdb/match_exclusion_repo.go:50 | SetExclusion | INSERT stage='exclusion' valeur explicite. |
| sync/convergence.go:147 | convergePSA | INSERT stage='psa'. |
| sync/comeback_postsync_persist.go:58 | backfillDominanceFlagsBatch | SUPPRIMER le seed INSERT OR IGNORE ; garder BatchUpdateColumn (devient INSERT stage='dominance'). |
| sync/enrichments.go:202 | setHadBotFlag | INSERT stage='bot' valeur explicite (pré-filtrer via _latest pour éviter redondance). |
| sync/friends_recompute.go:244 | updateIsWithFriendsBatch | INSERT stage='friends' TRUE explicite (pré-filtrer _latest). |
| sync/friends_recompute.go:319 | demoteIsWithFriendsBatch | INSERT stage='friends' FALSE explicite. |
| sync/ensure_enrichment_rows.go:147 | ensurePlayerEnrichmentRows | SUPPRIMER fonction + appel engine_postsync.go:184 + déclencheurs. |
| platform/duckdb/fanout_repo.go:92 | InsertStubEnrichments | SUPPRIMER (writer + port/fanout.go + fanout_service.go:158 + LoadExistingEnrichments:60). |
| service/openspartan_post_import_service.go:166 | ensureEnrichmentRows | SUPPRIMER (méthode + appel l.96). |
| cmd/seed-medal/main.go:82 | main | INSERT stage='dominance' (remplace UPDATE+INSERT ON CONFLICT). |

### PME-E — Readers → player_match_enrichment_latest
**JOIN_1_1 / FANOUT / SELECT_SCALAR (needsLatestView=true, BLOQUANTS) :**
- platform/duckdb/shared_query_helpers.go:136 (LoadPlayerMatchEnrichments — hub, corrige 4 callers)
- queries_career.go:35, :66, :276 ; queries_home_citations.go:96, :112 ; queries_career_encounters.go:190
- queries_squad.go:102, :346 (AGG fan-out winrate squad) ; queries_match.go:271
- squad_repo.go:90 ; squad_repo_mapstats.go:129 ; patterns_repo.go:191
- match_exclusion_repo.go:156 (loadExcludedPMERows) ; aggregates.go:33 (mv_player_matches → propagation)
- engagement_score_repo.go:509, :343, :389 ; engagement.go:417, :445 ; engagement_recompute.go:179
- **api/post_sync_progression_queries.go:196 (GAP rattrapé — perf=0 page progression)**
- sync/citations_backfill.go:97, :101
- **DELTA_FILTER re-INSERT loop (CRITIQUE)** : convergence.go:88 (psa_checked_at IS NULL) ; engine_postsync_csr.go:147 (dominance IS NULL) ; engine_backfills.go:485 ; sessions_postsync_persist.go:81 (deltaSessionAssignments) ; session_append.go:159 ; performance.go:305 ; exclusion_filter.go:30 ; friends_recompute.go:266 ; engine_postsync.go:108
- ops/seed_demo_corpus.go:62/72 ; validation/compare.go:246/325 ; validation/gate.go:312
- cmd/ diag (non bloquant, aligner) : diag_session_map_winrate, diag_db_health, diag_orphan_session, inspect_engagement, engagement-validate, audit_coverage, diag_citation_counters, diag_highlight_match

**Robustes au fan-out (set/MAP/MAX — non bloquant, aligner par cohérence)** : compare_repo.go:117/158 (MAX) ; convergence.go:213 ; engine.go:732 ; v2/known_loader.go:69 ; engagement.go:445 (set) ; invariants I1/I12 (set).

### PME-F — Rebuilders append-only-aware
- steps_player_rebuild_match_enrichment.go:100 (RebuildPlayerMatchEnrichmentART) : si columnExists('id') → déléguer applyAppendOnlyMatchEnrichment + return ; sinon swap append-only (id seq) + SEUL idx_pme_match_lookup + vue. SUPPRIMER le replay loadSecondaryIndexDDL (ressuscite les ex-index ART).
- steps_player_repair_pk.go:50 : si columnExists('id') → return nil ; sinon déléguer applyAppendOnlyMatchEnrichment (PAS RebuildPME qui re-pose PK match_id). MAJ godoc.
- cmd/rebuild_pme_art, cmd/force_rebuild_art, cmd/levelup/cmd_rebuild_pme.go : héritent du fix rebuilder (MAJ godocs).

### PME-G — Garde-fous + invariants
- no_art_patterns_test.go:229 : retirer player_match_enrichment de criticalMatchTables.
- no_art_patterns_test.go:65 : ajouter player_match_enrichment à tablesProtegees.
- append_only_state_guard_test.go:26 : ajouter player_match_enrichment à appendOnlyStateTables.
- append_only_state_guard_test.go:49 : nouveau garde interdire `UPDATE player_match_enrichment` + `FROM/JOIN player_match_enrichment` brut (hors _latest/__appendonly/__rebuild, hors writers INSERT) dans hot path.
- metadata_art_surface_guard_test.go:50 : forbiddenIndexedColumns += player_match_enrichment:{session_id, mode_category, engagement_score_brut}.
- invariants/invariants.go:139 (I1), :202 (I4), :226 (I2), :432 (I12) → `_latest` (I4/I2 OBLIGATOIRE faux positifs ; I1/I12 set-robuste mais aligner).

### PME-H — Fixtures de test → schéma append-only
**DDL manuel à migrer (id/written_at/stage + vue, OU appliquer migration.RunForDB) :**
testutil/fixture.go:167 (NewInMemoryPlayer, CENTRAL) ; sync_pipeline_fixture_test.go:197 (buildPlayerDDL) ; platform/duckdb/player_repos_test.go:215 (seedPlayerSchema, CENTRAL package) ; aggregates_test.go:27 (+INSERT positionnels 74/96) ; engine_test.go:24/51/108/148 (+INSERT positionnels 109/149) ; engagement_recompute_test.go:50 ; convergence_backfill_events_integration_test.go:51 ; ensure_enrichment_rows_test.go:31 (test du stub → supprimer avec la fonction) ; exclusion_filter_test.go:22 ; friends_recompute_integration_test.go:31 ; performance_integration_test.go:42 ; recompute_after_art_rebuild_test.go:81 ; v2/e2e_test.go:62 ; v2/known_loader_test.go:31 ; post_sync_enrichment_persister_test.go:21 ; repos_coverage_test.go:18 (InsertStub → supprimer) ; repos_extra_test.go:211 (+INSERT positionnels 496/561/562, CREATE 474/493/524/552) ; engagement_score_repo_integration_test.go:34 ; patterns_repo_db_test.go:53 (+INSERT positionnel 56) ; restore_test.go:31 (+INSERT positionnel 40, CREATE 118) ; restore_noop_test.go:70.
**GAPS rattrapés (census manqués) :** backfill_missing_test.go:75 (openPlayerForAll) ; backfill_integration_test.go:64/126/138 ; engine_e2e_test.go:188/331 (countRows assertions → COUNT(DISTINCT match_id)) ; service/openspartan_post_import_service_test.go:38/293/330 (COUNT(*) par match → ajuster) ; schema_integration_test.go:21/133 (liste tables + assert vue) ; migration_test.go:356 ; invariants/invariants_violation_test.go:70/102 (réécrire cohérent _latest) ; match_repos_test.go:431 (DELETE cleanup, hors hot path).
**BON pattern (migration déjà appliquée, OK) :** sessions_postsync_persist_test.go ; post_sync_progression_test.go:117 (mais ON CONFLICT(match_id) → INSERT pur) ; comeback_test.go (newInMemoryDBs → faire appliquer migration) ; enrichments_test.go ; convergence_test.go ; convergence_report_integration_test.go ; seed_demo_integration_test.go ; player_persister_test.go (RÉFÉRENCE) ; engine_batch_path_test.go.
**NE PAS MODIFIER (by-design) :** steps_player_append_only_match_enrichment_test.go ; steps_player_rebuild_match_enrichment_test.go (arbitrer si rebuild retiré) ; steps_player_repair_pk_test.go (arbitrer) ; steps_player_perf_chain_test.go ; compare_cgo_test.go / gate_cgo_test.go / exporter_test.go (tables jetables, nom incident).
**Fixtures Python (Go-only project) :** tests/create_test_fixture.py:268, create_multititle_fixture.py:135 — DDL append-only + INSERT colonnes nommées. Coordonner / idéalement générer via Go.
**pool_migration_test.go:160/209 :** seed LEGACY VOLONTAIRE (teste l'upgrade) — vérifier que la chaîne migration exécutée convertit en append-only + adapter assertions post-migration.

### PME-I — Go/no-go
`go build ./...` + `go test ./...` (CGO, -tags integration) vert + `engagement-coefs --all --with-scores --force` ×3 sans crash sur COPIES Madina/JGtm + EXPLAIN de la vue sur copie.

## Risques de correctness à surveiller (census verify)
- player_persister.go:186 multi-stage → résolu par stage='live' baseline (décision #1).
- booléens NULL : is_with_friends/had_bot/is_excluded — toujours valeur EXPLICITE, jamais NULL.
- collision engagement mode_category → résolu (décision #3).
- re-INSERT en boucle (psa/dominance/session delta-filters) → readers _latest IMPÉRATIFS (PME-E DELTA_FILTER).
- schema.go = pivot : non converti → tous les helpers EnsurePlayerSchema restent legacy + writers cassent à l'exécution réelle (conflit PK match_id).

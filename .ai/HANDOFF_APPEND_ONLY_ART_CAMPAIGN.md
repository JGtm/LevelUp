# HANDOFF — Campagne éradication ART DuckDB (append-only / SELECT-then-write)

> Point d'entrée unique pour REPRENDRE. Dernière MAJ : 2026-06-21 (post-incident PME : substrat retiré de main).
> **Branche courante : `refactor/art-pme-appendonly`** (origin, push à jour). Lire en entier.

## 1. TL;DR — où on en est

- **DÉPLOYÉ EN PROD (sain)** : `origin/main` = `125042306`. Contenu = fix prod d'origine (catalogue/battlepass) + retrait compaction match_skill_rank + **P1** (coach_proposal/engagement_coefficients/lusr_component_history) + **P0 media_files** (drop UNIQUE file_path) + correctif user `e1f021cbb` (combat-profile). Conteneurs Up (healthy), **zéro FATAL**. ⚠️ **Le substrat PME a été déployé par erreur (11:04) puis RETIRÉ (commit `125042306`, 11:32) — ZÉRO dégât** (aucune DB joueur migrée, aucun sync dans la fenêtre ; cf. thought_log [2026-06-21] incident). main ne contient PLUS la migration PME.
- **EN COURS, NON sur main — branche `refactor/art-pme-appendonly` (origin, `29c4df6eb`)** : **P3 player_match_enrichment** (le plus lourd). Substrat append-only LIVRÉ + validé (voir §1bis). ⚠️ **NE PAS laisser les commits de cette branche atteindre main** tant que la conversion writers n'est pas faite : main AUTO-DÉPLOIE, et le substrat seul (PME append-only + writers encore ON CONFLICT/UPDATE match_id) casse la prod. Migration + writers + readers = **un seul bloc** (re-livraison groupée).
- **RESTE** : finir PME PR1 (conversion atomique writers/readers, **turnkey** dans `.ai/PLAN_PME_ART_HARDENING.md`), puis `.ai/PLAN_ART_RESIDUAL_CENSUS_V2.md` P2/P4/P5 + clôture.
- Pansements (reopen/ExecRecovered/WithReopenOnInvalidated) GARDÉS : filet tant que la campagne n'est pas close.
- **À investiguer (pré-existant, non bloquant)** : WARN `catalog_expand: enqueue enfant échoué` (catalog_fetch_queue ON CONFLICT sans PK après rebuild_catalog_fetch_queue_drop_art_indexes) — antérieur à la campagne PME, dégradation catalogue silencieuse.

### 1bis. P3 player_match_enrichment — état (branche `refactor/art-pme-appendonly` @ `29c4df6eb`)

Commits PME (UNIQUEMENT sur la branche, PAS sur main) : `a836dcc53` (spec vérifiée workflow 7 agents) · `882aad112` (**substrat** : migration append-only transactionnelle + vue merge-on-read **par-groupe**, 8 tests CGO verts — merge partiel/NULL-reset/toggle booléen/legacy/idempotence/orphan) · `e5e23bed4` (recette turnkey + finding) · `d404b0080` (handoff) · `29c4df6eb` (note incident).

- **Le design merge-on-read est PROUVÉ** (le `buildPMELatestViewSQL()` de la migration est la référence : dédup par (match_id,stage) puis `CASE WHEN has_stage THEN valeur_stage ELSE legacy`). Le piège `last_value IGNORE NULLS` (figerait engagement_score=NULL) est ÉVITÉ.
- **Finding ATOMICITÉ** : la conversion writers/readers ne peut PAS se faire en chunks verts (convertir le persister seul casse ~7 tests sync : les ~12 fixtures créent PME sans colonne `stage`). → migration + TOUS les writers + readers→`_latest` + TOUTES les fixtures = **1 bloc cohérent** (PR1 atomique).
- **INCIDENT 2026-06-21 (résolu, zéro dégât)** : le substrat (`882aad112`) avait fui sur main → auto-deploy → migration PME live ~28 min SANS jamais s'exécuter (aucun sync) → retirée de main (`125042306`) avant déclenchement. **LEÇON : ne pas pousser/merger les commits de cette branche sur main avant la conversion writers complète** (main auto-déploie ; la migration PME est destructrice tant que les writers font ON CONFLICT/UPDATE match_id).
- **Reprise** : suivre l'ANNEXE IMPLÉMENTATION de `.ai/PLAN_PME_ART_HARDENING.md` (pattern persister-dérive-stage *prouvé*, stage par writer, readers→`_latest`, fixtures à patcher, RebuildPlayerMatchEnrichmentART à adapter en ADD PK(id), garde-fous). Validation finale = `go test ./...` + `engagement-coefs --all --with-scores --force` ×3 sur copie Madina/JGtm + go/no-go, PUIS livraison groupée sur main (migration+writers+readers ensemble).

### Findings P2 (cartographiés — pas de ré-investigation ; après PME) :
- `personal_score_awards` (player, NO PK, idx_psa_match_xuid) : DELETE-then-INSERT `InsertPersonalScoreAwards` (writes.go:392), **vecteur ACTIF** (engine_process_match live + convergence + backfill) → append-only. **Reprise conseillée ICI.**
- `weapon_kills` (shared, NO PK, idx_wk_match_xuid + vue v_weapon_kills) : `InsertWeaponKills` (writes.go:312), callers = backfill_weapons (basse fréq). Drop-index exclu (table grosse, scan) → append-only.
- `match_citations` (player, A une PK) : deleteCitationForMatch (citations.go:546) + citations_backfill.go:275 → append-only/recompute.
- `player_skill_state_v2` watermark : déjà append-only ; le DELETE WHERE xuid (lusr_full_recompute.go:34 + skill_v2_shadow.go:95) est le gap, **sémantique reset subtile** (comprendre le runner v2 avant de toucher). Basse fréq.
- `writes.go` match_registry/match_participants ON CONFLICT (P0 plan) : **CONFIRMÉ off-default** (batch INSERT-only = défaut prod) → risque réel faible, dé-prioriser.

## 2. Cause racine + doctrine

Bug DuckDB amont **#23046** (régression 1.5.0, NON corrigé 1.5.4 ; upgrade ne sauve pas) : l'enforcement de contrainte ART corrompt le heap sur DB **fichier**. Vecteurs CONFIRMÉS par crashs réels :
1. `INSERT … ON CONFLICT DO UPDATE` (= delete+insert interne sur index).
2. `INSERT OR REPLACE`.
3. `DELETE` per-row sur table à index ART (prouvé : compaction match_skill_rank → crash JGtm).
4. `UPDATE SET <col>` où `<col>` est **indexée** (prouvé : PME engagement-coefs sur colonnes indexées sous pression).
5. `INSERT` pur enforçant une PK/UNIQUE **métier** sous pression massive.

**`sérialisé/mono-writer` ≠ SÛR** (réfuté par match_skill_rank en mono-writer + PK BIGINT). **`shared_social`/`metadata` = handle RW PARTAGÉ concurrent → 1 FATAL = TOUTE l'app** (blast MAX). `player stats.duckdb` = mono-writer lease (blast 1 joueur) **mais crashe quand même** (#23046). `shared_matches_v2` = surtout RO + writers legacy concurrents.

**Nuance (jugement, non prouvé)** : « tout UPDATE NU sur table à index ART = dangereux » est une EXTRAPOLATION des skeptiques. Les preuves directes ne couvrent QUE les 5 vecteurs ci-dessus. Un UPDATE nu sur colonne **non-indexée** (WHERE=PK, point-update, basse fréquence) = priorité moindre. Les UPDATE **bulk** (`…IN(500)`) et **haute pression** (PME) restent à convertir.

## 3. Patterns de conversion (recette)

- **Table d'ÉTAT (mutée/accumulée)** → **append-only** : table sœur `<t>_history` (PK technique `seq`/`id` BIGINT `nextval(seq)` + colonnes + flag d'état/tombstone `is_*` + `written_at`) ; vue `<t>_latest` (`QUALIFY ROW_NUMBER() OVER (PARTITION BY <clé> ORDER BY written_at DESC, seq DESC)=1` [+ `WHERE is_deleted=FALSE`]) ; writers = INSERT pur (create / carry-forward `INSERT…SELECT FROM _latest` pour mark/update / tombstone) ; readers → `_latest` (PIÈGE : re-grep TOUS les readers) ; backfill ; fixtures de test (ajouter `_history`+seq+vue) ; garde-fou `append_only_state_guard_test.go` (ajouter `<t>` + `<t>_history`). Migration calquée sur `steps_shared_social_notif_prefs_append_only.go` / `steps_shared_social_user_prestige_append_only.go`.
- **Table de RÉFÉRENCE/cache (latest-value, basse fréquence)** → **SELECT-then-write** : `(*duckdb.DB).UpsertNoConflict(ctx, selectQ, selectArgs, updateQ, updateArgs, insertQ, insertArgs)` (méthode existante, db.go:432) ; ou inline pour `*sql.DB` brut (`SELECT 1 … ; switch err {case nil: UPDATE ; case ErrNoRows: INSERT}`). **⚠️ L'UPDATE ne doit toucher AUCUNE colonne indexée** → sinon DROP l'index aussi.
- **Colonne indexée mutée** → **drop l'index** (migration `drop_*_art_surface_indexes_v*`) + RETIRER le `CREATE INDEX` d'origine (DBs fraîches) + étendre `forbiddenIndexedColumns` (metadata_art_surface_guard_test.go — couvre aussi les tables player). Précédents : v1→v4 metadata, drop_challenge_mutated_art_indexes_v1 (player).
- **DELETE reconcile** → append-only tombstone OU recompute sans DELETE per-row OU (CLI) DROP+CREATE+INSERT structurel.

## 4. Gotchas build/test/commit (IMPÉRATIF)

- Toolchain CGO : `export CGO_ENABLED=1 && export PATH="/c/msys64/ucrt64/bin:/c/msys64/mingw64/bin:$PATH"`. Build/test/commit depuis `apps/go-api`.
- **Commit AVEC l'env CGO** (sinon le hook lefthook `go-vet` échoue « build constraints exclude all Go files »). `gofmt -w` sur les .go modifiés avant commit. **Jamais `--no-verify`. Demander n'est plus requis** (user a accordé l'autonomie pour cette campagne ; il escalade seulement le DEPLOY).
- `go test -race` INCOMPATIBLE driver DuckDB (checkptr) → ne pas l'utiliser.
- **Piège `execScript: empty query`** : un commentaire SQL APRÈS le dernier `;` = statement vide → mettre les commentaires AVANT le CREATE (statement suivant) ou en `//` Go.
- **Placement migration dans `order.go canonicalOrder`** (no-op strict `TestSortByCanonicalIsNoOpOnCurrentRegistry`) : position = ordre d'enregistrement = ordre ALPHABÉTIQUE du nom de FICHIER (init() global). Placer le nom de migration au bon slot ; vérifier avec le test no-op (échoue sinon, indique le décalage). En cas de doute : fichier temporaire `zz_dump_order_test.go` qui logge `All()` indexé.
- Whitelist `internal/platform/duckdb/no_attach_on_social_test.go` : ajouter tout NOUVEAU fichier non-test mentionnant le littéral `shared_social`.
- Tests verts à viser : `./internal/{migration,persist,sync,platform/duckdb,service,api/...,notify,ops,prestige}`. Go/no-go avant deploy = `go test ./...` + `go vet ./...` COMPLETS (env CGO).

## 5. RESTE À FAIRE → voir `.ai/PLAN_ART_RESIDUAL_CENSUS_V2.md`

Ordre conseillé (blast-radius × confiance) :
- **P0** (shared DBs MAX) : `media_files` UPDATE file_path(UNIQUE)+kind (ops/media.go:455, media_hls.go:106, media_reconcile.go:87) ; `match_registry` ON CONFLICT (writes.go:67, season_id indexé) ; `match_participants` ON CONFLICT (writes.go:132, team_id) + backfill_bits (writes.go:478).
- **P1** (player, vecteurs certains) : `lusr_component_history` ON CONFLICT (skill_rating_loaders.go:347) → append-only ; `coach_proposal` status (coach_proposal_repo.go:152/161/171/180) → DROP idx_coach_proposal_user_status ; `engagement_coefficients` → DROP idx_engagement_coefficients_xuid (redondant avec PK) ; `challenge` status/arc_id/campaign_id → VALIDER que drop_challenge_mutated_art_indexes_v1 s'applique au boot avant tout write.
- **P2** (player DELETEs) : match_citations (citations.go:546, citations_backfill.go:275), weapon_kills (writes.go:329), personal_score_awards (writes.go:399), challenge/arc (prestige_player_repo.go:173/243), player_skill_state_v2 watermark (lusr_full_recompute.go:34 → écrire ligne reset au lieu de DELETE).
- **P3 LOURD** : `player_match_enrichment` (4 index ART ; ON CONFLICT writes.go:254 + match_exclusion:52 ; UPDATE convergence.go:148/enrichments.go:202/friends_recompute.go:245+319) → append-only merge-on-read. Cf `.ai/PLAN_PME_ART_HARDENING.md` (Option A). Palliatif bulk : N UPDATE row-by-row via `PostSyncEnrichmentPersister` (1 entrée ART/statement, ADR 0019).
- **P4 LOURD** : `match_registry` completion bit-ledger (backfill_completed/events_loaded : writes.go:372/485/498, pve.go:306, events_replay.go:224, events_completion_persister.go:159/250) → bit-ledger append-only.
- **P5 CLI** : archive.go:189 (DELETE match_participants), restore.go:223 (DELETE générique) → maintenance offline / DROP+CREATE+INSERT.

## 6. Clôture (quand P0→P5 faits)

1. Étendre garde-fous : `TestNoBulkMultiRowUpdateOnCriticalTables` MANQUE la forme `UPDATE…WHERE…IN(…)` (gap exploité par friends_recompute, backfill_registry_names) — étendre la regex. Ajouter les nouveaux index droppés au guard.
2. Re-lancer le census v2 (workflow `art-residual-census-v2`, scriptPath réutilisable) → confirmer ZÉRO résiduel.
3. **Retirer les pansements** (reopen/ExecRecovered/WithReopenOnInvalidated) — code mort une fois les écritures sûres.
4. Go/no-go COMPLET + merge `fix/...` → `origin/main` (fast-forward) = 2e auto-deploy. Sync `git branch -f main origin/main`. Vérifier VPS (`ssh lvelup` : containers healthy + zéro FATAL).

## 7. Pointeurs + notes

- Work-list détaillée : `.ai/PLAN_ART_RESIDUAL_CENSUS_V2.md`. PME (Phase 3) : `.ai/PLAN_PME_ART_HARDENING.md`. Historique décisions : `.ai/thought_log.md` entrées `[2026-06-19]`/`[2026-06-20]`. Mémoire : `project_append_only_eradication_campaign`.
- Census v2 output brut (réutilisable, scriptPath) : sous `…/756bbf95-…/tasks/wou349k7s.output` + le script workflow.
- **Deploy** = merge `fix/...` sur `origin/main` (fast-forward) → GitHub Action « Deploy to VPS » (git reset --hard origin/main → deploy.sh, ~2min) + « Regen demo ». Le user ESCALADE le deploy (outward-facing) — ne pas déployer sans son OK.
- **Note git** : un commit ART avait atterri par erreur sur `chore/i18n-playlist-to-selection` (petit travail i18n du user) → corrigé (cherry-pick sur fix/... + `git branch -f`). Toujours vérifier `git branch --show-current` avant de commiter ; rester sur `fix/...`.
- Garde-fous existants : `internal/migration/metadata_art_surface_guard_test.go` (forbiddenIndexedColumns, couvre player aussi), `internal/sync/{append_only_state_guard_test,no_art_patterns_test}.go`, `internal/platform/duckdb/no_attach_on_social_test.go`.

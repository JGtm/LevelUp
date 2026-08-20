# PLAN — `player_match_enrichment` append-only (éradication ART #23046)

> Statut : **DESIGN VÉRIFIÉ** (census + design + vérif adversariale ultracode, 7 agents, 2026-06-20).
> Branche : `refactor/art-pme-appendonly` (créée depuis `main`).
> Verdict des 2 red-teams : **GO-WITH-FIXES**, POC-first, **2 PRs**. Tous les fixes ci-dessous sont OBLIGATOIRES.
> Ce document remplace l'ancien plan « proposé » (Options A–D) : Option A (append-only merge-on-read) est RETENUE et spécifiée.

## Cause racine (rappel)

Bug DuckDB amont **#23046** (régression 1.5.0, non corrigé 1.5.4) : l'enforcement de contrainte PK/UNIQUE via l'index ART corrompt le heap sur DB **fichier** sous churn. `player_match_enrichment` = la table la PLUS écrite du projet, écrite par **étapes incrémentales partielles** (perf ≠ engagement ≠ friends ≠ session ≠ had_bot ≠ exclusion ≠ psa). Crash prod observé sur `engagement-coefs --with-scores` (réécrit `engagement_*` sur des centaines de matchs). Mitigation existante = `RebuildPlayerMatchEnrichmentART` (répare, ne prévient pas).

## Schéma cible (in-place rebuild, calqué match_skill_rank/media_files)

Table garde son nom. PK technique `id BIGINT` (séquence `pme_seq`) + `written_at`. **Drop** PK(match_id) + les **3 index ART** mutés : `idx_pme_session`(session_id), `idx_pme_engagement_history`(mode_category, engagement_score_brut), `idx_pme_engagement_paces`(mode_category). Garder seulement `idx_pme_match_lookup(match_id, written_at)`. Lecture via vue `player_match_enrichment_latest`.

Colonnes (23) groupées par ÉTAPE co-écrite (clé du merge) :
- **perf** : performance_score, performance_chain · writer : performance.go → persister
- **dominance** : dominance_flag · comeback_postsync_persist.go
- **session** : session_id, session_label · sessions_postsync_persist.go
- **friends** : is_with_friends (DEFAULT FALSE, bidirectionnel) · friends_recompute.go (promote/demote)
- **teammates** : teammates_signature · writes.go UpsertPlayerEnrichment (live)
- **bot** : had_bot_teammate (bidirectionnel) · enrichments.go setHadBotFlag
- **exclusion** : is_excluded (DEFAULT FALSE, bidirectionnel) · match_exclusion_repo.go (HTTP)
- **psa** : psa_checked_at (marqueur terminal) · convergence.go
- **engagement** : engagement_score (NULL légitime = insufficient_history), engagement_score_brut, engagement_score_confidence, mode_category, engagement_pace_* (4) · engagement.go + engagement_score_repo.go (doublon HTTP)
- **meta** : created_at (min), updated_at (max)
- **MORTES (aucun writer Go)** : known_teammates_count, friends_xuids (vestiges Python — ne pas câbler)

## ⚠️ CORRECTION DESIGN CRITIQUE — merge PAR GROUPE, pas par colonne

Le `last_value(col IGNORE NULLS)` **par colonne** est FAUX (red-team défaut #1, HIGH) : `engagement_score = NULL` est une **valeur cible légitime** (insufficient_history) ; IGNORE NULLS la fige à l'ancienne valeur à vie.

**Solution = merge par GROUPE de colonnes co-écrites** : chaque INSERT porte un discriminateur d'étape `stage VARCHAR` ('perf'|'session'|'friends'|'bot'|'exclusion'|'psa'|'engagement'|'teammates'|'dominance'|'live'). La vue prend, **par (match_id, stage), la DERNIÈRE row** (written_at DESC, id DESC) et pivote les groupes en colonnes par match_id. Ainsi le dernier INSERT d'un groupe fait foi **NULL inclus** (reset intra-groupe correct), tout en mergeant entre groupes. Esquisse :

```sql
CREATE OR REPLACE VIEW player_match_enrichment_latest AS
WITH latest_per_stage AS (
  SELECT *, ROW_NUMBER() OVER (PARTITION BY match_id, stage ORDER BY written_at DESC, id DESC) AS rn
  FROM player_match_enrichment
)
SELECT
  match_id,
  -- chaque colonne tirée de la dernière row de SON étape (NULL légitime préservé) :
  any_value(performance_score) FILTER (WHERE stage='perf')            AS performance_score,
  any_value(session_id)        FILTER (WHERE stage='session')         AS session_id,
  any_value(engagement_score)  FILTER (WHERE stage='engagement')      AS engagement_score,   -- NULL = cible
  COALESCE(any_value(is_with_friends) FILTER (WHERE stage='friends'), FALSE) AS is_with_friends,
  COALESCE(any_value(is_excluded)     FILTER (WHERE stage='exclusion'), FALSE) AS is_excluded,
  ... -- une ligne par colonne, FILTER sur son étape
  min(created_at) AS created_at, max(updated_at) AS updated_at
FROM latest_per_stage WHERE rn = 1
GROUP BY match_id;
```
(`any_value(...) FILTER (WHERE rn=1 AND stage=...)` = la valeur de l'unique dernière row de l'étape ; valider la syntaxe DuckDB exacte au POC — alternative : `MAX(... ) FILTER` sauf qu'il ignore NULL → utiliser `any_value`/`arg_max(col, written_at)` pour préserver NULL.) **À éprouver au POC.**

Règle à documenter (persist/doc.go) : « colonne bidirectionnelle (is_with_friends/had_bot/is_excluded) = écrire la VALEUR EXPLICITE (TRUE/FALSE), jamais NULL ».

## Migration (BLOCKER : transactionnelle, PAS le template skill_rank)

Le template `steps_player_append_only_match_skill_rank.go` fait DROP/RENAME/ADD PK **sans transaction ni garde row-count** → sur PME (table NON re-dérivable, grosses DB Madina 1183/JGtm 942), un SIGTERM deploy entre DROP et RENAME = **perte totale irrécupérable** (red-team défaut migration #2, BLOCKER).

→ Calquer sur **`RebuildPlayerMatchEnrichmentART`** (steps_player_rebuild_match_enrichment.go:76-112) ET sur `media_files_drop_filepath_unique_v1` (déjà livré ce cycle) : `BeginTx` → DROP IF EXISTS __appendonly → CTAS (id seq + colonnes + stage + written_at) → garde `rebuilt==before` → DROP+RENAME+ADD PK(id) dans la TX → Commit. Defer Rollback. ADD DEFAULT/CREATE VIEW/indexes hors TX (idempotents). recoverOrphan en tête.

## Fixes OBLIGATOIRES (red-team, 16 défauts)

### Writers → INSERT pur partiel (avec `stage`)
- `writes.go:248` UpsertPlayerEnrichment ON CONFLICT → INSERT pur (stage='teammates'). **Vecteur #1 live.**
- `post_sync_enrichment_persister.go:82/157` BatchUpdateColumn/Multi (UPDATE) → INSERT partiel (garder whitelist `allowedEnrichmentColumns`, TX, delta-filters).
- `engagement_score_repo.go:158/181` SaveEngagementScore UPDATE×2 (doublon HTTP, **non couvert par le garde-rail actuel**) → INSERT partiel (stage='engagement'). Supprimer le skip RowsAffected==0.
- `friends_recompute.go:244/319` UPDATE…IN bulk → INSERT (stage='friends', valeur explicite TRUE/FALSE).
- `enrichments.go:192` setHadBotFlag UPDATE…IN → INSERT (stage='bot', valeur explicite).
- `convergence.go:147` convergePSA UPDATE → INSERT (stage='psa').
- `match_exclusion_repo.go:49` ExcludeMatch ON CONFLICT → INSERT pur (stage='exclusion', valeur explicite TRUE/FALSE).
- `player_persister.go:186` persistEnrichment : déjà INSERT partiel ; **retirer la garde EXISTS(match_id)** (player_persister.go:64-73) — ré-insérer est voulu.
- **Supprimer les stubs/priming** (plus de pré-création de row) : `ensure_enrichment_rows.go:147`, `fanout_repo.go:80`, `comeback_postsync_persist.go:57`, `openspartan_post_import_service.go:166` (ON CONFLICT (match_id) DO NOTHING → casse car PK match_id disparaît).

### Delta-filters & readers → DOIVENT lire `player_match_enrichment_latest` (BLOCKERS — sinon volume explosif + casse silencieuse)
- **Idempotence/delta** (sinon ré-INSERT massif à chaque sync) : `setHadBotFlag` WHERE, `friends_recompute` promote/demote WHERE, `deltaSessionAssignments` (sessions_postsync_persist.go:76-113), `convergence.go:88` (psa_checked_at IS NULL → sinon re-fetch PSA **infini**). Tous → `_latest`.
- **Readers engagement** : engagement_score_repo.go LoadPlayerHistory:508, HasEngagementScore:341, LoadRatioSamples:382 → `_latest` (sinon doublons, LIMIT N tronque, coefs biaisés).
- **JOIN 1:1 / agrégats fan-out** (BLOCKER, casse SILENCIEUSE) : queries_career.go:35/66, queries_squad.go:102/342 (AVG/COUNT/games_together FAUSSÉS), queries_match.go:265/271, patterns_repo.go:188, compare_repo.go, shared_query_helpers.go:127 (hub central), match_exclusion_repo.go:154 loadExcludedPMERows (sinon match ré-inclus reste fantôme-exclu). Tous → `_latest`.
- **invariants.go** (4 occ) : COUNT(*)==COUNT(DISTINCT match_id) devient FAUX → réécrire vers `_latest` ou DISTINCT match_id (sinon le gate d'invariants bloque le sync).
- Périmètre : ~25-30 readers prod sur ~10 fichiers (total ~43 sites read recensés, pas 120). **Audit site-par-site** (mémoire « audit shared.X = constante × caller »).

### Réparateur & schéma
- **BLOCKER** : `RebuildPlayerMatchEnrichmentART` re-pose `ADD PRIMARY KEY (match_id)` + recrée les ex-index ART → **ré-introduit le vecteur**. Exposé via cmd/rebuild_pme_art, cmd/force_rebuild_art, cmd/levelup rebuild-pme. → l'adapter : si `columnExists('id')`, rebuild en mode append-only (ADD PK(id), seulement idx_pme_match_lookup, recrée la vue). `steps_player_repair_pk.go:65` devient no-op (hasPrimaryKey sur id → true).
- **2 sources de schéma** à patcher pour naître append-only : `internal/sync/schema.go:37-56` (cité) ET `internal/migration/steps_player.go:21-30` create_base_player_schema (OUBLIÉ — PRIMARY KEY (match_id), ordonné AVANT la migration). Centraliser le DDL.

### Garde-rails & tests
- `no_art_patterns_test.go` : AJOUTER `player_match_enrichment` à `tablesProtegees` (active le scan ON CONFLICT) ; RETIRER de `criticalMatchTables`.
- **NOUVEAU garde-rail** : interdire tout `UPDATE player_match_enrichment SET` / `DELETE FROM player_match_enrichment` ET tout `FROM/JOIN player_match_enrichment` (brut, hors `_latest`) dans le hot path (modèle drift-detector / TestNoJSONRouteBypassesHuma). Indispensable : les UPDATE nus row-by-row (engagement_score_repo) ne sont PAS couverts par le garde actuel (forme VALUES seule).
- Tests : (1) **merge-on-read central** — 3 versions partielles disjointes d'un match (perf seul / session seul / engagement seul) → `_latest` reconstitue les 3 + reset booléen (exclude TRUE puis FALSE → FALSE) + reset engagement_score (valeur puis NULL → NULL). (2) **JOIN fan-out** — squad perf_avg/games_together inchangés malgré N versions. (3) **idempotence delta** — 2 post-sync sans changement ⇒ 0 INSERT. (4) **idempotence migration** + garde anti-perte. (5) **rebuild post-migration** ne recrée pas PK(match_id).
- **Perf (BLOCKER mou)** : `EXPLAIN ANALYZE` la vue sur copie Madina/JGtm AVANT de figer. Prévoir **compaction périodique** (réutiliser RebuildPlayerMatchEnrichmentART comme compacteur : CTAS `_latest` → 1 ligne/match) ; matérialiser la vue si bench mauvais.

## Roadmap (POC-first, 2 PRs)

**Incrément 1 = POC (mergeable seul, dé-risque le cœur)** :
1. Migration append-only transactionnelle + vue merge-on-read par-groupe + colonne `stage` + patch des 2 schémas + adapter RebuildPlayerMatchEnrichmentART.
2. Convertir UN cluster chaud isolé = **engagement** (engagement.go + engagement_score_repo.go SaveEngagementScore → INSERT partiel stage='engagement') + ses 3 readers (LoadPlayerHistory/HasEngagementScore/LoadRatioSamples → `_latest`).
3. Laisser les autres writers en UPDATE résiduel TEMPORAIREMENT (sans PK match_id ni index ART muté, ils ne déclenchent plus #23046 — palier sûr).
4. Test merge-on-read + EXPLAIN sur copie DB.
5. **Critère de succès** : `engagement-coefs --all --with-scores --force` ×3 sans crash sur copie Madina/JGtm.

**PR2 = reste des writers/readers** (sessions, friends, perf, dominance, bot, exclusion, psa, teammates) par lots, même pattern. + garde-rail strict + compaction.

## Risques résiduels
- Croissance non bornée (chaque post-sync ré-INSÈRE) → delta-filters sur `_latest` IMPÉRATIFS + compaction.
- Perf vue window/pivot non mesurée → EXPLAIN au POC.
- written_at ties (now() figé par TX) → s'appuyer sur `id` comme tie-breaker (mono-writer sous lease KindPlayer = id causal).
- Migration sur DB tenue RW in-process → RW exclusif requis (piège B-swap « attached read-only »).

## Pointeurs
- Census/design/vérif brut (728k tokens, réutilisable) : workflow `pme-appendonly-census-design` (run wf_34f9440b-264), output sous `…/tasks/wkjz80ya6.output`.
- Précédents à calquer : `steps_player_lusr_components_append_only.go`, `steps_shared_social_media_files_drop_filepath_unique.go` (migration transactionnelle + test, livrés ce cycle), `steps_player_rebuild_match_enrichment.go` (swap transactionnel).

---

## ANNEXE IMPLÉMENTATION (2026-06-21) — substrat LIVRÉ + recette writers turnkey

### Substrat livré (commit `882aad112`, branche refactor/art-pme-appendonly) — ⚠️ NE PAS DÉPLOYER SEUL
- Migration `player_append_only_match_enrichment_v1` (steps_player_append_only_match_enrichment.go) + vue `player_match_enrichment_latest` merge-on-read **par-groupe** PROUVÉE par 8 tests d'intégration CGO (merge partiel, NULL-reset, toggle booléen, fallback/override legacy, idempotence, orphan-recovery). Le design merge est VALIDÉ — le `buildPMELatestViewSQL()` du fichier est la référence (dédup par (match_id,stage) puis `CASE WHEN has_stage THEN valeur_stage ELSE legacy`).

### ⚠️ La conversion writers/readers est ATOMIQUE (pas de chunks verts incrémentaux)
Dès que le persister INSÈRE avec `stage`, les ~12 fixtures de test qui créent player_match_enrichment en DDL MANUEL (PK match_id, sans colonne `stage`) cassent (« no column stage ») — vérifié : convertir le seul persister casse ~7 tests sync (comeback/dominance). Donc migration + TOUS les writers + readers→_latest + TOUTES les fixtures doivent livrer en UN SEUL bloc cohérent. Lister les fixtures via `grep -rn "CREATE TABLE.*player_match_enrichment" --include=*_test.go` et les passer au schéma append-only (id + stage + written_at + colonnes) OU leur faire appliquer la migration (RunForDB TargetPlayer).

### Recette persister (validée, prête à recoller) — minimise le ripple
Dans `post_sync_enrichment_persister.go` : ajouter le map colonne→stage + dériver le stage (interface caller INCHANGÉE), puis BatchUpdateColumn/Multi → INSERT :
```go
var enrichmentColumnStage = map[string]string{
  "performance_score":"perf","performance_chain":"perf","dominance_flag":"dominance",
  "session_id":"session","session_label":"session","is_with_friends":"friends",
  "had_bot_teammate":"bot","teammates_signature":"teammates",
  "engagement_score":"engagement","engagement_score_brut":"engagement",
  "engagement_score_confidence":"engagement","engagement_pace_player":"engagement",
  "engagement_pace_team":"engagement","engagement_pace_lobby":"engagement",
  "engagement_player_activity":"engagement","mode_category":"engagement",
}
// deriveEnrichmentStage(columns) → stage commun (erreur si mixte/inconnu).
// BatchUpdateColumn : INSERT INTO pme (match_id, <col>, stage) VALUES (?,?,?)
// BatchUpdateMulti  : INSERT INTO pme (match_id, <cols...>, stage) VALUES (?,...,?)
// (id/written_at/created_at/updated_at via DEFAULT ; totalAffected = len(rows))
```
Les callers BatchUpdateMulti écrivent déjà des colonnes mono-stage (perf={score,chain}, session={id,label}, engagement={engagement_*}) → deriveEnrichmentStage OK sans toucher les callers.

### Stage par writer (à appliquer dans le même bloc)
- writes.go UpsertPlayerEnrichment (ON CONFLICT) → INSERT (match_id, teammates_signature, ...) stage='teammates'.
- player_persister.go persistEnrichment (live, subset DYNAMIQUE) : décomposer en 1 INSERT par groupe présent dans EnrichmentRow (perf/engagement/session/teammates...), chacun avec son stage ; retirer la garde EXISTS(match_id). POINT DUR — vérifier ce que le caller live (engine_process_match/engine_fetch) peuple réellement.
- engagement_score_repo.go SaveEngagementScore (UPDATE×2) → INSERT stage='engagement' (supprimer skip RowsAffected==0).
- friends_recompute.go (UPDATE IN) → INSERT stage='friends' (valeur explicite TRUE/FALSE) ; delta lu sur _latest.
- enrichments.go setHadBotFlag (UPDATE IN) → INSERT stage='bot' (valeur explicite) ; delta sur _latest.
- convergence.go convergePSA (UPDATE) → INSERT stage='psa' ; le SELECT `psa_checked_at IS NULL` (convergence.go:88) → _latest (sinon re-fetch infini).
- match_exclusion_repo.go ExcludeMatch (ON CONFLICT) → INSERT stage='exclusion' (explicite) ; loadExcludedPMERows → _latest.
- comeback_postsync_persist.go : dominance via persister (stage='dominance') ; SUPPRIMER le seed INSERT OR IGNORE.
- Supprimer stubs : ensure_enrichment_rows.go, fanout_repo.go InsertStubEnrichments, openspartan_post_import_service.go ensureEnrichmentRows (ON CONFLICT match_id casse).

### Readers → vue _latest (BLOCKERS fan-out, cf. red-team)
Tout `FROM/JOIN player_match_enrichment` qui attend 1 ligne/match → `player_match_enrichment_latest` : queries_career.go:35/66, queries_squad.go:102/342, queries_match.go:265/271, patterns_repo.go:188, shared_query_helpers.go:127 (hub), compare_repo.go, match_exclusion_repo.go:154, engagement_score_repo.go (3), invariants.go (×4 : COUNT==DISTINCT → _latest), deltaSessionAssignments (sessions_postsync_persist.go:76). Garde-fou : interdire `FROM/JOIN player_match_enrichment\b` (brut) hors writers+vue.

### Réparateur + schémas + garde-fous
- RebuildPlayerMatchEnrichmentART : si columnExists('id') → ADD PK(id) (pas match_id), recréer SEULEMENT idx_pme_match_lookup + la vue (sinon ré-introduit l'ART).
- Patcher schema.go playerSchemaSQL + steps_player.go create_base_player_schema pour naître append-only (id+stage+written_at, pas de PK match_id).
- no_art_patterns_test.go : player_match_enrichment → tablesProtegees (retirer de criticalMatchTables) + nouveau garde interdisant UPDATE/FROM brut.

### Validation finale
go test ./... vert + critère `engagement-coefs --all --with-scores --force` ×3 sans crash sur copie Madina/JGtm + EXPLAIN de la vue.

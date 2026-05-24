# Plan Phase 4 — Refactor post-sync compute (INSERT-only)

**Noté** : 2026-05-24 suite au smoke test Phase 3 prod
**Statut** : Plan d'analyse — pas d'implémentation
**Effort estimé** : 6-10h (gros sprint)
**Prérequis** : Phase 2 (Collect→Persist insert path) livrée

---

## Découverte (smoke test 2026-05-24)

Le path `submitMatchAsBatch` (Phase 2) marche **parfaitement** en prod :
- 10 matchs persistés pour XxDaemonGamerxX sans aucun FATAL
- `persist_batch_committed_total=10`, `persist_shared_total_ok=10`, `persist_player_total_ok=10`
- Cycle XxDaemonGamerxX : 107s (sous le seuil 8min)

MAIS le **post-sync compute** déclenche encore le bug ART :

```
upsertLUSRRatings: exec LUSR update failed match_id=...
err="FATAL Error: Invalid Input Error: Failed to delete all rows…"

post-sync: sessions échoué : WriteSessionAssignments: FATAL Error: Invalid Input Error: ...
post-sync: perf scores échoué : batchComputePerformanceScores: FATAL Error: ...
post-sync: friends recompute échoué : updateIsWithFriendsBatch: FATAL Error: ...
```

→ Le bug ART n'est PAS supprimé en prod tant que ces UPDATE concurrents existent.

---

## Diagnostic : 7 sites UPDATE post-sync

Tous touchent `player_match_enrichment` ou `match_skill_rank` (player DB, par joueur). Lors d'un cycle multi-joueur en parallèle, ces UPDATEs concurrents stressent l'ART de la player DB.

| # | Fichier | Ligne | UPDATE cible |
|---|---|---|---|
| 1 | `comeback.go` | 181 | `player_match_enrichment SET dominance_flag` |
| 2 | `engagement.go` | 441, 479 | `player_match_enrichment SET engagement_*` |
| 3 | `enrichments.go` | 75 | `player_match_enrichment SET had_bot_teammate` |
| 4 | `friends_recompute.go` | 231 | `player_match_enrichment SET is_with_friends` |
| 5 | `performance.go` | 615 | `player_match_enrichment SET performance_score` |
| 6 | `session_recalc.go` | 115, 178 (via `WriteSessionAssignments`) | `player_match_enrichment SET session_id` |
| 7 | `skill_rating_loaders.go` | 301 (via `upsertLUSRRatings`) | `match_skill_rank SET rating_value, tier, ...` |

---

## Pourquoi le path Collect→Persist Phase 2 ne couvre pas ça

Phase 2 a refactor le path d'**insert per-match** :
1. fetchMatchData récupère 1 match depuis l'API
2. buildBatchFromFetchedMatch produit un MatchBatch
3. submitMatchAsBatch INSERT atomique 1 fois
4. `player_match_enrichment` reçoit une row PLACEHOLDER (juste `match_id`)

Le **post-sync compute** :
1. Tourne APRÈS la boucle d'insertion (sur tous les matchs nouveaux + matchs existants à recalculer)
2. Calcule perf_score, sessions, dominance, friends, LUSR, engagement
3. UPDATE `player_match_enrichment` row par row pour chaque match
4. Avec 4 joueurs en parallèle (pool_size=4), 4 UPDATE concurrents sur la même DB → bug ART

---

## Options de fix

### Option A — INSERT-then-DELETE (rebuild row)

Pour chaque match recalculé : INSERT une nouvelle row complète, puis DELETE l'ancienne. Atomic via TX.

✅ Pas de UPDATE = pas de DELETE-side ART stress
❌ DELETE + INSERT = même problème ART (cf. verdict empirique)
→ **Rejeté** : le DELETE déclenchera le bug.

### Option B — Drop-and-recompute la table complète

Pour chaque cycle post-sync : DROP `player_match_enrichment`, CREATE TABLE, INSERT toutes les rows recalculées.

✅ Pas de UPDATE ni DELETE par-row
✅ TX atomique pour swap (rename old→backup, rename new→live)
❌ Coût élevé : recompute TOUS les matchs à chaque cycle (potentiellement 1000+)
❌ Tables liées (FK potentiels) à gérer

### Option C — Append-only avec versioning (event sourcing light)

Une nouvelle table `player_match_enrichment_versions(match_id, version, ..., created_at)` qui n'est que INSERT. Une vue `player_match_enrichment` qui retourne la dernière version par match_id.

✅ Pure INSERT = pas d'ART stress
✅ Historique préservé (debug + rollback)
❌ Migration lourde (touche tous les readers)
❌ Volume DB augmente (versions accumulées → besoin compaction périodique)

### Option D — Sérialiser le post-sync compute par joueur (pas concurrent)

Garder UPDATE mais 1 seul worker post-sync à la fois (mutex global). Pas de concurrence = pas de stress ART.

✅ Modif minime (juste ajouter un mutex)
✅ UPDATE reste sémantiquement clean
❌ Perte de parallélisme : cycle 3 joueurs passe de 5min à ~15min
❌ Le bug ART peut quand même se déclencher si les UPDATEs intra-cycle sont nombreux

### Option E — Calculer EN MÉMOIRE puis INSERT batch unique

Recompute tous les enrichments en mémoire (Go), puis 1 seul INSERT batch atomique (DELETE table + INSERT all en TX). Variante de B avec batching strict.

✅ Pas de UPDATE par-row, pas de concurrence intra-TX
✅ Compatible avec le SharedPersister pattern existant
❌ Mémoire : tous les matchs en RAM le temps du compute (acceptable jusqu'à ~10k matchs/joueur)
❌ Atomicité : si compute crash mid-way, rien n'est persisté (acceptable, retry au prochain cycle)

---

## Recommandation

**Option E** — adapter le pattern Collect→Persist au post-sync compute :

1. Compute en RAM dans `internal/analysis/` (déjà des fonctions pures)
2. Construire un `PlayerEnrichmentBatch` (struct similaire à `MatchBatch` mais pour les enrichments calculés)
3. Persister via un nouveau `PostSyncPersister` qui DELETE+INSERT en TX (single writer = pas de concurrence ART)
4. Réutiliser le pattern WAL+Worker async pour décorréler du sync principal

**Effort** : ~6-10h (5 sites UPDATE à migrer + tests + smoke prod)

**Bénéfice** : élimine définitivement le bug ART en prod. Permet ensuite Phase 5 cleanup (suppression singleflight, CHECKPOINT, etc.) sans risque.

---

## Plan d'attaque proposé (révisé 2026-05-24 — Option B retenue par user)

### Phase 4.1 — Audit + design ✅ DONE

- 7 sites UPDATE identifiés (table ci-dessus)
- Option B retenue : `DELETE WHERE rating_type=X + INSERT batch en 1 TX`

### Phase 4.2 — PostSyncLUSRPersister ✅ DONE (pattern de référence)

- `internal/persist/post_sync_lusr_persister.go` livré
- 5 tests TDD GREEN (InsertsBatch, PreservesCSRRows, EmptyBatch_NoOp, AtomicityRollback, NeverUpdatesCSRRow)
- Pattern : 1 DELETE WHERE filtre + N INSERT en TX atomique
- Préserve les CSR (filtre `WHERE rating_type='LUSR'` sur le DELETE)

### Phase 4.3 — Intégration LUSR dans le sync engine ✅ DONE

**Livré** :
- `internal/sync/skill_rating_postsync_persist.go` — `upsertLUSRRatingsBatch` accumule les `LUSRRatingInsert` et appelle `PostSyncLUSRPersister.Upsert` en 1 single TX
- `internal/sync/skill_rating_loaders.go::upsertLUSRRatings` — dispatch via feature flag `LEVELUP_POSTSYNC_INSERT_ONLY=1`
- User correction critique : `Upsert` (DELETE WHERE match_id IN(...) AND rating_type='LUSR' + INSERT batch) au lieu de `Persist` (full replace) — préserve les autres LUSRs et les CSRs
- 2 tests TDD GREEN supplémentaires sur `PostSyncLUSRPersister.Upsert` (PreservesOtherLUSRRows, PreservesCSRForSameMatchID)

### Phase 4.4 — Refactor des sites post-sync UPDATE row-by-row ✅ DONE (5/7 sites — 2 non concernés)

**Décision de design** : au lieu de créer 6 persisters dédiés, **mutualisation via 2 persisters génériques** sur `player_match_enrichment` (whitelist colonnes anti SQL injection) :
- `PostSyncEnrichmentPersister.BatchUpdateColumn(col, rows)` — single col, multi-row via `UPDATE … FROM (VALUES …) AS v(match_id, val)`
- `PostSyncEnrichmentPersister.BatchUpdateMulti(rows)` — multi-col, multi-row via syntaxe étendue
- 7 tests TDD GREEN sur le persister (BatchUpdateColumn + BatchUpdateMulti + edge cases)

**Sites refactorés (chemin batch derrière `LEVELUP_POSTSYNC_INSERT_ONLY=1`)** :

| Site | Helper batch | Persister appelé | Status |
|---|---|---|---|
| `comeback.go::BackfillDominanceFlags` | `backfillDominanceFlagsBatch` | `BatchUpdateColumn("dominance_flag", rows)` | ✅ DONE |
| `engagement.go::batchComputeEngagementScores` | accumulation in-loop + flush | `BatchUpdateMulti` (4 ou 8 cols selon migration paces) | ✅ DONE |
| `performance.go::batchComputePerformanceScores` | accumulation `pendingUpdates` + flush | `BatchUpdateMulti` (`performance_score` + `performance_chain`) | ✅ DONE |
| `writes.go::WriteSessionAssignments` | `writeSessionAssignmentsBatch` | `BatchUpdateMulti` (`session_id` + `session_label`) | ✅ DONE |
| `skill_rating_loaders.go::upsertLUSRRatings` | `upsertLUSRRatingsBatch` | `PostSyncLUSRPersister.Upsert` | ✅ DONE (Phase 4.3) |

**Sites non concernés (déjà single SQL UPDATE batchée) — pas de refactor nécessaire** :

| Site | Pattern actuel | Raison |
|---|---|---|
| `enrichments.go::computeAndPersistHadBotTeammate` | `UPDATE player_match_enrichment SET had_bot_teammate=TRUE WHERE match_id IN (?,?,…) AND COALESCE(had_bot_teammate,FALSE)=FALSE` | Déjà 1 single UPDATE multi-row, pas le pattern row-by-row qui stresse l'ART |
| `friends_recompute.go::updateIsWithFriendsBatch` | `UPDATE player_match_enrichment SET is_with_friends=TRUE WHERE COALESCE(is_with_friends,FALSE)=FALSE AND match_id IN (?,?,…)` | Idem — boucle par chunks de N matchs mais chaque chunk = 1 single UPDATE |

→ Ces 2 sites pourraient être migrés vers `BatchUpdateColumn` pour cohérence stylistique mais **n'apportent aucun bénéfice fonctionnel** (déjà 1 single SQL UPDATE par appel, pas un loop UPDATE per-row). À garder en l'état pour minimiser le diff. Si bug ART persistent observé sur ces 2 sites après smoke test Phase 4.5, alors migration ; sinon laisser tel quel.

**Effort réel** : ~6h sur 1 session (vs estimation initiale ~6-10h).

### Phase 4.5 — Smoke test prod ✅ VALIDÉE 2026-05-24

**Itérations exécutées** :

1. **Cycle 1 (pre-rebuild)** : 2/4 joueurs en FATAL ART (Chocoboflor `writeSessionAssignmentsBatch` 410 rows, Madina97294 `upsertLUSRRatingsBatch` 21 rows). Diagnostic : ART pré-corrompue.
2. **`force_rebuild_art --all`** : rebuild `shared.match_participants` + `player_match_enrichment` (4 player DBs). Sessions OK post-rebuild, mais...
3. **Cycle 2 (post pme rebuild)** : sessions batch OK pour tous, **LUSR Upsert toujours FATAL** car `match_skill_rank` ART non rebuilt par force_rebuild_art (le CLI ne touchait que `player_match_enrichment`).
4. **Fix livré** : `RebuildMatchSkillRankART` ajouté à `internal/migration/`, intégré dans `force_rebuild_art` (rebuild PME puis MSR par player DB).
5. **Cycle 3 (post full rebuild pme+msr)** : 4/4 OK, **0 FATAL**, LUSR Upsert validé (Chocoboflor: 9 LUSR persistés).
6. **Cycles 4-5** : 4/4 OK chacun, **0 FATAL**.

**Résultat consolidé** : 3 cycles consécutifs × 4 joueurs = **12 syncs, 0 FATAL ART**.

**Insight critique 2026-05-24** : la Phase 4 batch path **n'élimine pas une corruption pré-existante** — elle PRÉVIENT les futures. Pré-requis avant déploiement : `force_rebuild_art --all true` (qui rebuild les 3 tables critiques : `shared.match_participants`, `player_match_enrichment`, `match_skill_rank`).

**Performance observée** (cycle 5, sequentiel 4 joueurs) :
- Madina97294 : 94s (1115 rows enrichment, 1109 LUSR/CSR)
- JGtm : 76s (832 rows)
- Chocoboflor : 20s (419 rows)
- XxDaemonGamerxX : 6s (32 rows)
- **Total cycle complet** : ~3 min 30 — sous le seuil 8 min visé

### Phase 4.6 — Cleanup post-validation (Phase 5 du plan initial) — 2h

Une fois Phase 4.5 validée, débloque Phase 5 cleanup :
- Cf. `.ai/PLAN_PHASE5_CLEANUP_ANTI_ART.md`

---

## Effort cumulé restant

| Étape | Effort | Status |
|---|---|---|
| 4.1 Audit + design | 1h | ✅ DONE |
| 4.2 PostSyncLUSRPersister + tests | 1h | ✅ DONE (commit) |
| 4.3 Intégration LUSR (Upsert + dispatch) | 1h | ✅ DONE |
| 4.4 Refactor 5 sites + 2 mutualisés (PostSyncEnrichmentPersister) | 6h | ✅ DONE |
| 4.5 Smoke test prod multi-cycles | 30min | ✅ DONE (3+4+5 = 12 syncs / 0 FATAL) |
| 4.5b RebuildMatchSkillRankART (fix bonus découvert pendant 4.5) | 20min | ✅ DONE |
| 4.6 Phase 5 cleanup anti-ART (4/7 items) | 2h | ✅ DONE — singleflight + CHECKPOINT + BootARTGuard auto-heal supprimés |
| 4.7 Closure : BatchQueue wiring + janitor + flip defaults | 1h30 | ✅ DONE — cycles 7+8 OK, default ON validé |
| 4.8 Item 6 PLAN_PHASE5 (revert acad4603) | 30min | ✅ DONE — 5 sites legacy reverted, 3 conflits résolus, cycle 9 OK |
| **Total restant** | **0h (sauf PR 2.5b auth backlog)** | |

---

## État actuel des plans (consolidé 2026-05-24 fin de journée)

| Plan | Statut |
|---|---|
| REFACTOR_COLLECT_PERSIST Phases 1-2 | ✅ Livré |
| Phase 3 activation | 🟡 **Partielle** — path insert OK, post-sync FATAL ART (corrigé par Phase 4) |
| Phase 4 post-sync refactor | ✅ Code livré — attend smoke test prod (4.5) |
| Phase 5 cleanup anti-ART | ⏳ Attend Phase 4.5 validation |
| PLAN_AUTH E.v1 (Discovery + watcher stores) | ✅ Livré |
| PLAN_AUTH E.v2 (callback push) | ⏳ Backlog |

**Activation Phase 4 en prod** : poser les 2 flags ensemble :

```bash
LEVELUP_PERSIST_BATCH=1            # Phase 2 INSERT batch (déjà actif)
LEVELUP_POSTSYNC_INSERT_ONLY=1     # Phase 4 post-sync batch (nouveau)
```

Sans `LEVELUP_POSTSYNC_INSERT_ONLY`, le chemin legacy UPDATE-then-INSERT reste actif (zéro régression — feature flag opt-in strict, dispatch en tête de chaque fonction concernée).

**Décision pragmatique** : Phase 4 est **code-complete**. Validation finale = smoke test multi-cycles en prod avec les 2 flags actifs. Si 0 FATAL ART observé sur 3-5 cycles consécutifs → Phase 5 cleanup débloqué.

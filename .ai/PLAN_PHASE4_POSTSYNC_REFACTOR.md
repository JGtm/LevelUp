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

### Phase 4.3 — Intégration LUSR dans le sync engine (RESTE À FAIRE)

**Effort** : ~1h
**Fichiers à toucher** :
- `internal/sync/skill_rating_loaders.go::upsertLUSRRatings` — route via PostSyncLUSRPersister si flag actif
- `internal/sync/skill_rating.go::batchComputeLUSR` — accumule les `LUSRRatingInsert` au lieu d'appeler upsertLUSRRatings row-by-row
- Feature flag `LEVELUP_POSTSYNC_INSERT_ONLY=1` opt-in (cohérent avec LEVELUP_PERSIST_BATCH)

### Phase 4.4 — Refactor 6 autres sites (PATTERN À REPRODUIRE)

Pour chaque site, créer un `PostSyncXxxPersister` similaire :
- `comeback.go:181` → `PostSyncDominancePersister`
- `engagement.go:441,479` → `PostSyncEngagementPersister`
- `enrichments.go:75` → `PostSyncBotFlagPersister`
- `friends_recompute.go:231` → `PostSyncFriendsPersister`
- `performance.go:615` → `PostSyncPerfPersister`
- `session_recalc.go:115,178` (via WriteSessionAssignments) → `PostSyncSessionsPersister`

Chacun fait : `DELETE WHERE filter + INSERT batch en TX`. Le filter dépend de la colonne (ex: `WHERE dominance_flag IS NOT NULL` pour Dominance, `WHERE session_id IS NOT NULL` pour Sessions).

**Effort** : ~1h par site = ~6h.

### Phase 4.5 — Smoke test prod — 30min

- Cycle complet 4 joueurs avec `LEVELUP_PERSIST_BATCH=1 LEVELUP_POSTSYNC_INSERT_ONLY=1`
- Vérifier : **0 FATAL ART** sur 3-5 cycles
- Vérifier : enrichments cohérents avec legacy (sanity SQL : sample row par site refactoré)

### Phase 4.6 — Cleanup post-validation (Phase 5 du plan initial) — 2h

Une fois Phase 4.5 validée, débloque Phase 5 cleanup :
- Cf. `.ai/PLAN_PHASE5_CLEANUP_ANTI_ART.md`

---

## Effort cumulé restant

| Étape | Effort | Status |
|---|---|---|
| 4.1 Audit + design | 1h | ✅ DONE |
| 4.2 PostSyncLUSRPersister + tests | 1h | ✅ DONE (commit) |
| 4.3 Intégration LUSR | 1h | ⏳ TODO |
| 4.4 6 autres sites | 6h | ⏳ TODO |
| 4.5 Smoke test prod | 30min | ⏳ TODO |
| 4.6 Phase 5 cleanup | 2h | ⏳ TODO |
| **Total restant** | **~10h** | |

---

## État actuel des plans (consolidé 2026-05-24)

| Plan | Statut |
|---|---|
| REFACTOR_COLLECT_PERSIST Phases 1-2 | ✅ Livré |
| Phase 3 activation | 🟡 **Partielle** — path insert OK, post-sync FATAL ART |
| Phase 4 post-sync refactor | ❌ NOUVEAU — ce document |
| Phase 5 cleanup anti-ART | ⏳ Attend Phase 4 |
| PLAN_AUTH E.v1 (Discovery + watcher stores) | ✅ Livré |
| PLAN_AUTH E.v2 (callback push) | ⏳ Backlog |

Tant que Phase 4 n'est pas faite, **garder `LEVELUP_PERSIST_BATCH` actif** :
- Le path insert (Phase 2) marche et apporte un bénéfice net (10 matchs persistés sans FATAL pour XxDaemonGamerxX)
- Le post-sync compute échoue avec ART mais ça n'empêche pas l'insert (le sync se termine "status=success" malgré les warnings post-sync)
- L'auto-heal BootARTGuard + RebuildART runtime continuent de masquer le problème dans le legacy

**Décision pragmatique** : Phase 3 est "déployable" dans un mode "partiel" (sync insert OK, post-sync compute dégradé). Le cleanup Phase 5 attend Phase 4.

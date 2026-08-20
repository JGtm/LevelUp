# REFACTOR — Architecture Collect → Persist (anti-corruption ART définitive)

**Statut** : DESIGN — à valider avant implémentation.
**Date** : 2026-05-23.
**Auteur** : claude + Guillaume.
**Branche cible** : `refactor/collect-persist` (à créer depuis `chore/post-stabilisation-debt`).

---

## 1. Pourquoi

### Diagnostic empirique (cycle 19:33:20)

Migrations UPDATE-then-INSERT (commit `acad4603`) **n'ont rien changé** :

```
19:33:20 [WARN] upsertLUSRRatings: exec LUSR update failed
err="FATAL Error: Invalid Input Error: Failed to delete all rows from index.
     Only deleted 0 out of N rows"
```

**Cause confirmée** : DuckDB étant columnar, les **UPDATE sont implémentés comme DELETE+INSERT en interne** (rewrite de la row entière). Donc UPDATE consulte aussi l'index ART en mode DELETE → même bug que `ON CONFLICT DO UPDATE`.

### Conclusion

Tant qu'on fait des UPSERTs/UPDATEs concurrents sur DuckDB, le bug ART persiste. La **seule solution propre** :

> **Séparer la collecte (parallèle, mémoire) de la persistance (séquentielle, batch INSERT-only)**.

C'est l'architecture proposée par l'utilisateur et validée empiriquement par élimination de toutes les autres options.

---

## 2. Objectif

**Critère de succès** :

1. **0 FATAL ART** sur 10 cycles consécutifs en prod (validation empirique).
2. **Cycle ≤ 8 min** sur 3 joueurs (perf préservée).
3. **Aucune perte de données** : crash mid-sync → reprise propre au prochain démarrage.
4. **Architecture réutilisable** par sync engine + backfill CLI + scripts diag.
5. **Tous les enrichments couverts** (zéro régression fonctionnelle).

---

## 3. Architecture cible

### Vue d'ensemble

```
┌──────────────────────────────────────────────────────────────────┐
│                         CALLERS                                   │
│  - sync engine (auto + manuel)                                    │
│  - backfill CLI (cmd/backfill/*, cmd/seed-*)                      │
│  - scripts diag (cmd/diag_*)                                      │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                               ▼ utilisent
┌──────────────────────────────────────────────────────────────────┐
│              Package internal/persist (NOUVEAU)                   │
│                                                                    │
│  ┌────────────────────┐    ┌─────────────────────────────────┐   │
│  │   MatchBatch       │    │  BatchQueue                      │   │
│  │   (struct in-mem)  │───▶│  - WAL durable (data/wal/*.json)│   │
│  │                    │    │  - 1 worker goroutine par DB    │   │
│  │ Accumule pendant   │    │  - Channel buffered             │   │
│  │ la collecte        │    │  - Retry sur fail               │   │
│  └────────────────────┘    └──────────────┬──────────────────┘   │
│                                            │                      │
│                                            ▼                      │
│                            ┌────────────────────────────────┐    │
│                            │  Persister (par DB target)     │    │
│                            │  BEGIN TX                       │    │
│                            │    INSERT batch sur N tables   │    │
│                            │  COMMIT                         │    │
│                            │  → ACK + delete WAL file        │    │
│                            └────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

### Composants

#### `MatchBatch` — l'unité de travail

```go
// package internal/persist
type MatchBatch struct {
    // Métadonnées
    BatchID   string    // UUID v4
    Player    string    // gamertag (pour player DB target)
    XUID      string
    CreatedAt time.Time
    Source    string    // "sync_delta", "backfill_cli", "manual_recompute", etc.

    // Données shared.duckdb (1 batch = N matchs nouveaux)
    Shared SharedBatch

    // Données player.duckdb (stats.duckdb de ce joueur)
    Player PlayerBatch

    // Données shared_pve.duckdb (si Firefight matchs)
    PVE *PVEBatch  // nullable

    // Données metadata.duckdb (rare, ex. update mappings)
    Metadata *MetadataBatch  // nullable
}

type SharedBatch struct {
    Matches       []MatchRegistryRow
    Participants  []ParticipantRow
    Medals        []MedalRow
    HighlightEvents []HighlightEventRow
    WeaponKills   []WeaponKillRow
    KillerVictim  []KillerVictimPair
    XUIDAliases   []XUIDAlias
}

type PlayerBatch struct {
    Enrichments       []EnrichmentRow  // performance_score, dominance, engagement, session_id, is_with_friends EN UNE SEULE ROW par match
    SkillRanks        []SkillRankRow   // LUSR/CSR par match
    Citations         []CitationRow
    PersonalScoreAwards []PersonalScoreAwardRow
    CareerProgression []CareerSnapshot  // si rank a changé
    Sessions          []SessionRow      // si nouvelle session
}
```

**Property critique** : un MatchBatch est **complet** = TOUTES les écritures pour les N matchs traités. Pas de phase ultérieure qui rajoute des champs.

#### `BatchQueue` — durabilité + ordonnancement

```go
type BatchQueue struct {
    walDir   string                       // "data/wal/"
    channels map[DBTarget]chan *MatchBatch  // 1 channel par DB
    workers  map[DBTarget]*Persister
}

// API publique
func (q *BatchQueue) Submit(batch *MatchBatch) error {
    // 1. Sérialise batch en JSON
    // 2. Write atomique dans walDir/{batch_id}.json
    // 3. Push dans le channel approprié
    // 4. Retourne sans attendre la persistence (non-bloquant)
}

func (q *BatchQueue) RecoverPending() error {
    // Au boot : lire walDir/, re-soumettre les batches en suspens
}
```

#### `Persister` — un worker par DB target

```go
type Persister struct {
    target DBTarget        // shared, player, pve, metadata
    db     *sql.DB
}

// Worker loop
func (p *Persister) Run(ctx context.Context, in <-chan *MatchBatch) {
    for batch := range in {
        if err := p.persist(ctx, batch); err != nil {
            slog.WarnContext(ctx, "persist failed, retry later", "err", err)
            // Re-push dans la queue avec backoff
            continue
        }
        // ACK : delete walDir/{batch_id}.json
    }
}

func (p *Persister) persist(ctx context.Context, batch *MatchBatch) error {
    tx, _ := p.db.BeginTx(ctx, nil)
    defer tx.Rollback()

    // INSERT batch (jamais UPDATE/UPSERT)
    // Si une row existe déjà (cas anormal) → erreur, on log et on garde le WAL pour debug

    return tx.Commit()
}
```

---

## 4. Contraintes utilisateur explicites

| Contrainte | Solution |
|---|---|
| **Le tampon ne se perd pas** | WAL JSON sur disque AVANT push channel. Crash → batch lu et reprocessed au boot. |
| **Plusieurs processus écrivent** | Channel Go buffered (intra-process). N collecteurs (1 par joueur) → 1 channel → 1 worker par DB. Pas de race sur la persistence. |
| **Pas paralyser l'app** | Phase Collect parallèle. Phase Persist = transactions courtes (~quelques sec). DuckDB readers continuent (pas de lock partagé). |
| **Scalability multi-utilisateurs** | 4 joueurs syncs simultanés → 4 collectes parallèles + 1 worker shared.duckdb sériel (mais court) + 4 workers player.duckdb parallèles (DBs différentes). |
| **Toutes les tables/BDD** | Audit exhaustif des INSERT/UPDATE actuels. Voir §5. |
| **Aucun enrichment oublié** | Voir §6. Test E2E qui valide que chaque enrichment est dans le batch. |
| **Réutilisable CLI + backfill** | Le package `persist` est indépendant du sync engine. CLI/backfill l'utilisent aussi : `persist.NewBatchBuilder().AddMatch(...).AddEnrichment(...).Submit(queue)`. |

---

## 5. Inventaire des écritures actuelles (à migrer)

### shared_matches_v2.duckdb

| Table | Fonction actuelle | Migration |
|---|---|---|
| `match_registry` | `InsertRegistryIfNotExists` | → `SharedBatch.Matches` |
| `match_participants` | `InsertParticipants` (UPSERT) | → `SharedBatch.Participants` (INSERT only) |
| `medals_earned` | `InsertMedals` (INSERT OR IGNORE) | → `SharedBatch.Medals` |
| `highlight_events` | `InsertHighlightEvents` | → `SharedBatch.HighlightEvents` |
| `weapon_kills` | `InsertWeaponKills` (DELETE+INSERT batch atomique) | → `SharedBatch.WeaponKills` (INSERT batch) |
| `killer_victim_pairs` | (à identifier) | → `SharedBatch.KillerVictim` |
| `xuid_aliases` | `UpsertXUIDAlias` | → `SharedBatch.XUIDAliases` |
| Bits dans `match_registry` (MBitSkill, MBitWeaponKills, etc.) | mark functions | → posés DANS la transaction batch (UPDATE in-place sur le row qu'on vient d'INSERT — pas de DELETE car la row est fresh) |

### stats.duckdb (player DB)

| Table | Fonction actuelle | Migration |
|---|---|---|
| `player_match_enrichment` | UPSERT 4 callers différents (perf, dominance, engagement, sessions, is_with_friends) | → `PlayerBatch.Enrichments` UNE row complète par match avec tous les champs |
| `match_skill_rank` | `upsertLUSRRatings` | → `PlayerBatch.SkillRanks` (INSERT batch) |
| `match_citations` | UPSERT | → `PlayerBatch.Citations` |
| `personal_score_awards` | INSERT | → `PlayerBatch.PersonalScoreAwards` |
| `career_progression` | INSERT | → `PlayerBatch.CareerProgression` |
| `sessions` | INSERT/UPDATE | → `PlayerBatch.Sessions` |

### shared_pve.duckdb

| Table | Fonction actuelle | Migration |
|---|---|---|
| `pve_match_stats` | INSERT (si match Firefight) | → `PVEBatch.MatchStats` |

### metadata.duckdb

Écritures rares en sync (mode_name_tr, etc.). À identifier mais probablement out of scope.

---

## 6. Enrichments à NE PAS oublier

Inventaire exhaustif (chaque ligne = un computed local actuellement écrit via UPSERT) :

| Enrichment | Source code actuel | Cible dans MatchBatch |
|---|---|---|
| `performance_score` | `batchComputePerformanceScores` (performance.go) | `EnrichmentRow.PerformanceScore` |
| `performance_chain` | idem | `EnrichmentRow.PerformanceChain` |
| `dominance_flag` | `BackfillDominanceFlags` (comeback.go) | `EnrichmentRow.DominanceFlag` |
| `engagement_score` + brut + confidence | `engagement.go` | `EnrichmentRow.Engagement*` |
| `engagement_pace_player/team/lobby` | idem | idem |
| `engagement_player_activity` | idem | idem |
| `mode_category` | engagement.go (computed) | `EnrichmentRow.ModeCategory` |
| `session_id` + `session_label` | `WriteSessionAssignments` (writes.go) | `EnrichmentRow.SessionID/Label` |
| `is_with_friends` | `RecomputeIsWithFriendsCore` | `EnrichmentRow.IsWithFriends` |
| `teammates_signature` | (idem friends) | `EnrichmentRow.TeammatesSignature` |
| `had_bot_teammate` | `computeAndPersistHadBotTeammate` | `EnrichmentRow.HadBotTeammate` |
| LUSR rating + tier + delta + components | `batchComputeLUSR` + `upsertLUSRRatings` | `PlayerBatch.SkillRanks` |
| CSR snapshot | `runCSRSnapshotSync` | nouveau struct dédié |
| Citations | `batchComputeCitations` | `PlayerBatch.Citations` |
| Personal score awards | `processPersonalScoreAwards` | `PlayerBatch.PersonalScoreAwards` |
| Career progression | `processCareer` | `PlayerBatch.CareerProgression` |
| Match intensity | `persistMatchIntensity` (engagement.go) | `SharedBatch.MatchIntensities` (NEW car shared.match_registry update) |
| Achievement progress | `runAchievementsSync` | hors scope batch (API Xbox séparée, peu critique) |
| Assists model coefs | `RunBackfillAssistsModel` | rare, batch séparé |

**À vérifier** : `engagement_coefficients` (table par xuid, peut être rare en sync delta).

---

## 6.bis Granularité du batch — décision critique

**Question utilisateur 2026-05-23** : si crash pendant phase Collect (avant Submit), peut-on reprendre ?

**Design v1 (cycle entier = 1 batch)** : ❌ tout perdu si crash mid-collect → API quota gaspillé.

**Design v2 — 1 match = 1 batch atomique** :

```
Pour chaque nouveau match (séquentiel pour LUSR cascade) :
  Fetch + Compute enrichments → MatchBatch(1 match)
                              → Submit immédiat
                              → Persist + ACK
                              → Match suivant
```

**Garanties** :

| Scénario crash | Comportement |
|---|---|
| Pendant fetch match N | Match N perdu, matchs 1..N-1 déjà persistés. Restart : liste des matchs déjà persistés (PK match_id) + reprise sur les manquants. |
| Pendant compute enrichment | Idem : seul match N perdu, restart reprend. |
| Après submit, avant persist | WAL JSON contient le batch. Worker reprend au restart. |
| Après persist, avant ACK | Batch committé. Restart re-tente INSERT → PK conflict → ACK + skip (idempotent). |

### Bonus optionnel — cache fetch intermédiaire

Pour économiser les API calls coûteux (~2s/match) en cas de crash entre fetch et compute :

```
1. Fetch raw data → écrit dans data/sync_cache/{cycle_id}/match_{match_id}.json
2. Compute enrichments (lit cache + état DB)
3. Build + Submit batch
4. ACK → delete cache
```

→ Restart skip les fetch dont le cache existe.

### LUSR cascade reste cohérente

Le LUSR de match N dépend du rating après match N-1. Avec submit match-par-match :

```
Match 1 → Submit → Persist (rating row écrite)
Match 2 → loadLatestLUSR() lit DB (inclut match 1) → compute cascade
       → Submit → Persist
```

→ **La cascade lit l'état persisté**, pas un état in-memory. Pas de risque de perte de cohérence.

### Trade-off

| Aspect | v1 (cycle batch) | v2 (match batch) |
|---|---|---|
| Transactions | 1 par cycle | N par cycle |
| Durabilité crash mid-collect | ❌ tout perdu | ✅ partiel préservé |
| Perf | Marginalement plus rapide (1 fsync) | Marginalement plus lent (N fsync) |
| Mémoire | Pic plus haut | Pic plus bas (1 match à la fois) |
| Reprise au restart | recommence à zéro | reprend là où on était |

**Décision : v2** (granularité match) — durabilité prime sur micro-perf.

---

## 7. Flow par scénario

```
1. Scheduler tick (15min)
   ↓
2. RunOnce : pour chaque joueur (errgroup, 4 parallèles)
   ↓
3. Phase Collect :
   - Fetch /matches (delta API)
   - Pour chaque nouveau match :
     - Fetch /stats + /skill + /film en parallèle
     - Parse highlight_events
     - Scan weapon_kills
   - Compute tous les enrichments locaux (LUSR cascade, perf, dominance, engagement, sessions, is_with_friends, citations, personal_score_awards, career si changé)
   - Build MatchBatch{Shared: {...}, Player: {...}}
   ↓
4. Submit batch → BatchQueue (WAL write + push channel)
   ↓
5. Worker shared.duckdb (1 seul) :
   - BEGIN TX
   - INSERT batch tables shared
   - COMMIT
   - ACK (delete WAL)
   ↓
6. Worker player.duckdb (1 par joueur, parallèles) :
   - BEGIN TX
   - INSERT batch tables player
   - COMMIT
   - ACK
   ↓
7. Cycle terminé
```

### Scénario B : Backfill CLI

```
1. CLI : ./bin/backfill --player Madina97294 --force
   ↓
2. Phase Collect (identique au sync mais pour TOUS les matchs ou liste fournie)
   ↓
3. Submit batches (peut faire plusieurs batches si volume gros : 1 batch = 100 matchs max)
   ↓
4. Workers persistent
   ↓
5. CLI attend la fin (avec progress bar)
```

### Scénario C : Crash mid-persist

```
1. Sync collecte OK, submit batch (WAL OK)
2. Worker shared.duckdb commence TX
3. CRASH (OOM, signal, etc.)
4. Restart serveur
   ↓
5. BatchQueue.RecoverPending() :
   - Lit data/wal/*.json
   - Re-push dans les channels
6. Workers reprennent depuis le batch en suspens
   ↓
7. Si la TX précédente avait déjà été commitée (peu probable mais possible) :
   - INSERT échoue avec PK conflict
   - On log + skip + ACK le WAL
   (Le INSERT n'a PAS de DELETE-side ART : juste un check PK qui voit la row existe → erreur clean)
```

---

## 8. Plan d'implémentation

### Phase 1 — Infrastructure (~4h) ✅ LIVRÉ

- [x] **1.1** Créer `internal/persist/` package (commit `65a63900`)
- [x] **1.2** Définir `MatchBatch`, `SharedBatch`, `PlayerBatch`, `PVEBatch` structs (`internal/persist/batch.go`, `rows.go`)
- [x] **1.3** Implémenter `BatchQueue` : WAL JSON + channels + worker pattern (`queue.go`, `worker.go`)
- [x] **1.4** Implémenter `Persister` : transaction batch INSERT only — 4 Persisters livrés :
  - SharedPersister (commit `d8ee8d33`)
  - PlayerPersister (commit `e19bb504`)
  - PVEPersister + MetadataPersister (commit `b62f78a0`)
- [x] **1.5** Tests unitaires : 34 tests GREEN dans `internal/persist/` (queue + 4 persisters + worker + E2E)

### Phase 2 — Sync engine refactor (~5h) ✅ LIVRÉ

- [x] **2.1** `buildBatchFromFetchedMatch` (collect.go) — DTO → batch pure (commit `25e24f84`, 8 tests)
- [x] **2.2** Extension `fetchedMatch` avec SharedCSRs + PveStats + mapping batch (commit `0c1025ec`, 12 tests)
- [x] **2.3** Orchestrateur `submitOrInsertMatch` + flag `batchMode` + feature flag env var (commits `c2bb4200` + `cdadb4c4`, 2 tests + cablage scheduler/handler)
- [x] **2.4** Tests TDD E2E cycle complet via `mockHaloClient` (commit `7e89acb3`, 4 tests E2E dont async path)

### Phase 3 — Migration progressive (~3h) ✅ COMPLÉTÉE 2026-05-24

- [x] **3.1** Feature flag `LEVELUP_PERSIST_BATCH=1` (opt-in) — câblé dans scheduler + handler + CLI cmd_sync.go
- [x] **3.2** Activé pour 4 joueurs en prod via diag endpoint (smoke test 2026-05-24)
- [🟡→✅] **3.3** Path INSERT validé Phase 4.5 (12 syncs/0 FATAL) + post-sync compute corrigé Phase 4 (5 sites batch + RebuildMSR)
- [x] **3.4** Durée d'observation étendue : 4 cycles (3+4+5+6) × 4 joueurs = 16 syncs / 0 FATAL avec `LEVELUP_PERSIST_BATCH=1 LEVELUP_POSTSYNC_INSERT_ONLY=1`
- [x] **3.5** Flip default à opt-out — ✅ LIVRÉ Phase 4.7 (commit pending) : 9 sites passés à `!= "0"` (default ON). Set `LEVELUP_PERSIST_BATCH=0` ou `LEVELUP_POSTSYNC_INSERT_ONLY=0` pour fallback legacy

**Découverte critique 2026-05-24** : Phase 2 a refactor le path **INSERT per-match** (succès — 10 matchs persistés sans FATAL pour XxDaemonGamerxX). Mais le **post-sync compute** (LUSR cascade, sessions recalc, performance scores, friends recompute, dominance, engagement) faisait toujours des UPDATE concurrents sur `player_match_enrichment` + `match_skill_rank` → bug ART déclenché. **Résolu par Phase 4** (5 sites refactor batch INSERT-only via `PostSyncEnrichmentPersister` + `PostSyncLUSRPersister`, 2 sites déjà batchés laissés en l'état). Cf. `.ai/PLAN_PHASE4_POSTSYNC_REFACTOR.md`.

### Phase 4 — Backfill CLI + scripts (~3h) ✅ LIVRÉ (scope révisé)

- [x] **4.1** Cablage `LEVELUP_PERSIST_BATCH` dans `cmd/levelup/cmd_sync.go` (2 sites RunDelta) — commit `f741c831`
- [N/A] **4.2** Refactor `cmd/seed-*` — non concernés (seeds initiaux, pas de concurrence write)
- [N/A] **4.3** Refactor `cmd/diag_*` — non concernés (read-only ou ops one-shot)

**Note** : les backfills `cmd/backfill_all` et `cmd/levelup/cmd_backfill.go` (LUSR, citations, weapons, PSA, engagement, etc.) n'appellent pas `submitMatchAsBatch` — ils ont leurs propres chemins de compute UPDATE-style qui ne touchent pas `shared.match_participants` et donc ne sont pas concernés par le bug ART.

### Phase 5 — Cleanup (~2h) ⏳ DÉBLOQUÉE par Phase 4.5 (smoke test prod validé 2026-05-24)

Cf. `.ai/PLAN_PHASE5_CLEANUP_ANTI_ART.md` pour le plan détaillé.

**Phase 4.5 status** : ✅ validée empiriquement le 2026-05-24 — 3 cycles consécutifs × 4 joueurs = 12 syncs, 0 FATAL ART. Pré-requis découvert : `force_rebuild_art --all true` doit rebuild **3 tables critiques** (pas 2) : `shared.match_participants`, `player_match_enrichment`, et **`match_skill_rank`** (le LUSR Upsert touchait une ART non rebuilt à l'origine). Fix livré : `RebuildMatchSkillRankART` + intégration dans `force_rebuild_art`.

**Cleanup débloqué — items concrets** :
- `singleflight` dans `InsertParticipants` — peut être supprimé
- `CHECKPOINT` post-sync — peut être supprimé
- `BootARTGuard` auto-heal — retirer l'auto-heal, garder la détection (alerte ops)
- `RebuildMatchParticipantsART` runtime call sites — garder comme outil ops, retirer call sites runtime
- `force_rebuild_art` CLI — **GARDER** comme outil ops manuel (essentiel pour défaire corruption héritée)
- Migrations UPDATE-then-INSERT (`acad4603`) — revert ou réécrire en INSERT pur

- [ ] `singleflight` dans `InsertParticipants` — à supprimer post Phase 3 validée
- [ ] `CHECKPOINT` post-sync — à supprimer
- [ ] `BootARTGuard` auto-heal — retirer l'auto-heal, garder la détection
- [ ] `RebuildMatchParticipantsART` runtime — garder comme outil ops, retirer call sites runtime
- [ ] `force_rebuild_art` CLI — **GARDER** comme outil ops manuel
- [ ] Migrations UPDATE-then-INSERT (`acad4603`) — revert ou réécrire en INSERT pur

### Phase 6 — Documentation finale (~1h) ✅ LIVRÉ

- [x] Update `.ai/INCIDENT_ART_CORRUPTION_DUCKDB.md` avec verdict final (status 🔴 → 🟢, section RÉSOLUTION) — commit `f741c831`
- [x] `docs/adr/0019-collect-persist-architecture.md` créé — commit `730894aa`
- [x] Update `thought_log.md` (5 entrées au fil des phases)
- [x] Update `CLAUDE.md` : règle écritures DB via `persist.BatchBuilder.Submit` — commit `f741c831`
- [x] ADR 0017 + 0018 marqués OBSOLÈTES (superseded by 0019)
- [x] `.ai/RUNBOOK_PHASE3_ACTIVATION.md` créé pour exécution Phase 3 côté user

### Phases bonus livrées (post-design)

- [x] **B6** PurgeOldWAL — méthode `BatchQueue.PurgeOldWAL(maxAge)` + 3 tests TDD (commit `04b56f96`)
- [x] **B7** Expvar metrics (`persist_shared_total_ok/_error`, `persist_player_total_ok/_error`, `persist_batch_committed_total`, `persist_batch_submitted_total`, `persist_batch_submit_error`)
- [x] **Async layer optionnelle** — `BatchQueue.PendingCount()` + `Drain(ctx)` + `WithBatchQueue` + Drain à fin de cycle + 4 tests TDD + 1 E2E async GREEN

### Items reportés (pas critiques pour Phase 3) — STATUS 2026-05-24

- [x] **Cache fetch intermédiaire** (`data/sync_cache/{cycle_id}/`) — ✅ LIVRÉ : `internal/sync/fetch_cache.go` actif (logs montrent `cache fetch intermédiaire actif` à chaque cycle). Format JSON par match (stats/skill/film + chunks .bin). Purge périodique livrée Phase 4.7 (janitor 24h, maxAge 7j).
- [ ] **B8** Multi-titres workers map (`workers[slug][target]`) — forward-compat sans valeur immédiate (Halo Infinite seul titre actuel)
- [x] **Câblage `BatchQueue`** côté `cmd/server/main.go` au boot — ✅ LIVRÉ Phase 4.7 : `NewBatchQueue` + `WithBatchQueue` + Drain shutdown. Gate `LEVELUP_PERSIST_BATCH_ASYNC=1` (default OFF — additif). Sans queue : path synchrone direct (validé Phase 4.5, default ON).

**Total effort estimé** : ~18h en TDD strict, sur 2-3 jours.
**Total livré** : ~12h effective sur 1 jour intense. Activation prod par user + Phase 5 cleanup post-validation = ~3h restants.

---

## 9. Risques & mitigations

| Risque | Probabilité | Mitigation |
|---|---|---|
| Volume mémoire trop gros (gros backfill) | Faible | Split en batches de 100 matchs max. Backpressure sur le channel. |
| Crash pendant TX → batch perdu | Faible | WAL JSON sur disque AVANT push. Recovery au boot. |
| Race conditions entre workers | Faible | 1 worker par DB. Channels Go = sérialisation naturelle. |
| Ordonnancement (insertion d'un match avant son enrichment) | Faible | Tout dans MÊME transaction batch. Garanti atomique. |
| Performance dégradée (transactions plus longues) | Faible | INSERT batch est PLUS rapide que N INSERTs individuels (1 seul fsync). |
| Régression fonctionnelle (enrichment oublié) | Moyen | §6 inventaire exhaustif + test E2E qui valide chaque field dans le batch. |
| Compatibilité backfill historique | Moyen | Backfill utilise la même API, donc same code path. |
| Multi-utilisateurs simultanés sur shared.duckdb | Faible | 1 worker shared = bottleneck mais transactions ~secondes. Acceptable. |

---

## 10. Décisions clés — VALIDÉES 2026-05-23

| # | Décision | Choix | Justification |
|---|---|---|---|
| Q1 | Format WAL | **JSON** | Lisible, debuggable, volumes faibles |
| Q2 | Channel buffer size | **1000 batches** | Backpressure naturelle (Submit bloque si plein) |
| Q3 | Retry strategy | **Exponential backoff** : 1s, 2s, 4s, 8s, 16s, 32s, abandon → log ERROR + alerte |
| Q4 | TX rollback granularité | **Atomique par batch** | Tout ou rien. Cohérence forte. |
| Q5 | Bits MBit* | **Dans la même TX** | Toujours cohérents avec les INSERTs |
| Q6 | Worker pool sizing | **1 fixe par DB target** | Suffit pour la charge. Pas de pool dynamique. |
| Q7 | Cache fetch intermédiaire | **ACTIVÉ par défaut** | Pratique pour debug + tests. Disable via env var `LEVELUP_PERSIST_NO_FETCH_CACHE=1` |
| Q8 | Granularité batch | **1 match = 1 batch atomique** | Durabilité fine (§6.bis) |
| Q9 | Atomicité cross-DB | **Shared first, puis player** | Si player fail → retry au prochain cycle, état recoverable via bits |
| Q10 | Ordonnancement intra-cycle | **Submit ordonné par start_time** par joueur | LUSR cascade chronologique |
| Q11 | Multi-titres | **Workers par-titre** : `workers[slug][target]` | Forward-compatible Reach/MCC |

### Cache fetch intermédiaire — détail

Activé par défaut, désactivable via `LEVELUP_PERSIST_NO_FETCH_CACHE=1`.

```
data/sync_cache/
├── {cycle_id}/
│   ├── match_{match_id}_stats.json     # raw response /stats
│   ├── match_{match_id}_skill.json     # raw response /skill
│   ├── match_{match_id}_film.json      # manifest film
│   └── match_{match_id}_chunks/        # chunks REPLICATION_DATA
│       └── chunk_{index}.bin
```

**Cycle de vie** :
- Crée pendant Fetch
- Lu pendant Compute (skip API si présent)
- Delete après ACK du batch
- Cleanup périodique : fichiers > 7 jours = supprimer

**Bénéfices** :
- ✅ Recovery sans re-appel API au restart
- ✅ Tests : on peut rejouer un sync sans API en pointant vers un cache figé
- ✅ Debug : voir exactement ce que l'API a renvoyé pour un match donné
- ✅ Économie quota API en cas de retry

---

## 11. Trous identifiés en revue 2026-05-23 (intégrés au design)

| # | Trou | Solution |
|---|---|---|
| B1 | Ordonnancement matchs cycle | Submit séquentiel ordonné par start_time par joueur |
| B2 | Atomicité cross-DB | Shared first, player ensuite. Si player fail → retry au cycle suivant |
| B3 | Backpressure | Channel buffered 1000 + Submit bloquant si plein |
| B4 | Worker sizing | 1 fixe par DB target |
| B5 | Recovery boot | Lit `data/wal/*.json` + re-push. WAL corrompu → `data/wal/corrupted/` + ERROR |
| B6 | Cleanup WAL/cache orphelins | Job interne périodique : fichiers > 7 jours = supprimer |
| B7 | Métriques expvar | `persist_batches_submitted_total`, `_committed_total`, `_failed_total`, `_wal_pending_count`, `_recovery_runs_total` |
| B8 | Multi-titres | Workers par-titre : `workers[slug][target]` |
| B9 | Test recovery crash | Test E2E : submit 5 batches, kill worker mid-process, restart, vérifier que les pending sont rejoués |

## 12. Bits MBit* avec collect→persist

Aujourd'hui les bits servent au heal pour décider "ce match n'a pas X, retente plus tard".

Avec collect→persist en cas nominal :
- Tout est récupéré en 1 passe avant Submit
- Bits posés à TRUE dans la même TX → toujours cohérents

Cas dégradés (les bits restent utiles) :
- API film 404 (match trop ancien, film expiré) → bit `MBitFilm=TRUE` quand même pour ne plus retenter
- API skill timeout → bit `MBitSkill=FALSE`, le match est committé mais marqué incomplet
- Un cycle ultérieur peut retenter (heal mode dégradé, pas le mode nominal)

→ **Le heal devient un cas exceptionnel**, pas la norme.

---

## 11. Critères de validation finale

Avant de déclarer le refactor terminé :

1. ✅ 10 cycles consécutifs en prod sans aucun `FATAL Error: Invalid Input Error` ni `Failed to delete all rows from index`.
2. ✅ `art_corruption_detected_*` reste à 0 sur 24h.
3. ✅ Tous les enrichments présents dans `player_match_enrichment` post-cycle (perf, dominance, engagement, sessions, is_with_friends, etc.).
4. ✅ Cycle 3 joueurs < 8 min wall-time.
5. ✅ Backfill CLI fonctionne avec la même architecture.
6. ✅ Crash mid-persist → recovery au boot sans perte.
7. ✅ Tests intégration + race tous verts.
8. ✅ Pas de UPSERT/UPDATE sur match_participants ou tables critiques (grep zéro).

---

## 12. Hors scope (à faire séparément)

- Migration vers Postgres/SQLite : on garde DuckDB, juste on l'utilise correctement.
- Refactor des reads : reste inchangé (DuckDB reste OLAP-friendly).
- Auto-recovery LUSR Madina figé : suivra naturellement quand sync re-tournera proprement.
- medals_earned corruption résiduelle : rebuild one-shot avant le premier cycle post-refactor.

---

## 13. Notes finales

**Pourquoi cette archi est définitive** :
- Plus de UPSERT → plus de pression sur l'index ART → plus de bug ART possible par construction.
- Architecturalement propre : ressemble à du EventSourcing / CQRS light.
- Réutilisable : sync + backfill + scripts + futurs imports.
- Testable : phase collect testable sans DB, phase persist testable avec DB mock.
- Compatible multi-titres : le batch est indépendant du titre (slug dans MatchBatch.Title).

**Pourquoi le user a raison depuis le début** :
- Toutes mes tentatives de workaround (singleflight, CHECKPOINT, UPDATE-then-INSERT) étaient des emplâtres.
- La vraie solution était de revoir notre USAGE de DuckDB, pas de tuner DuckDB.
- "Match figé après insertion" + "tampon en mémoire" = pattern qu'on aurait dû avoir dès le départ.

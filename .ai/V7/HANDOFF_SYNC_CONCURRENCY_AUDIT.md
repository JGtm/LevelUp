# HANDOFF — Audit préliminaire du plan stabilisation sync (4 agents parallèles)

**Date** : 2026-05-22.
**Lié à** : [`.ai/PLAN_SYNC_CONCURRENCY_STABILIZATION.md`](PLAN_SYNC_CONCURRENCY_STABILIZATION.md).
**Méthode** : 4 agents d'analyse lancés en parallèle pour valider/réfuter empiriquement les postulats du plan AVANT démarrage de l'implémentation.

---

## 0. Résumé exécutif

### Findings qui changent le plan

| # | Finding | Impact sur le plan |
|---|---|---|
| 1 | **`processWeaponKillsInline` est SÉQUENTIEL** dans le post-sync — boucle `for` sans goroutine. Madina passe **210-275s** dessus | **Nouvelle Phase 3.0 prioritaire** (pas dans le plan initial). Gain potentiel ~150-200s par cycle Madina. C'est le plus gros gain identifié. |
| 2 | **Network = 95% du temps**, compute = ~5% | Réfute le postulat du plan "compute + network sont le bottleneck". Le compute n'est PAS significatif. Phase 3.2 (move parse parallèle) à dépriorisera fortement (gain réel 100-500ms vs 10-30% annoncé) |
| 3 | **Pool RPS prod = 15 RPS** (`PerTokenRPS=5 × 3 tokens`), pas 30 RPS | Phase 3.3 (intra-match parallel API) ne gagnera que **200-300ms par match**, pas un RTT complet. ROI questionnable, à déprioriser |
| 4 | **DuckDB driver actuel = 1.5.2**, **v1.5.3 dispo depuis le 20 mai 2026** (corrige edge case index deletion). v1.4.1 changelog mentionne EXACTEMENT notre symptôme | Ajouter étape : tester bump driver vers v1.5.3 (faible coût, gain non garanti) avant d'investir lourd sur singleflight |
| 5 | **Un rebuild migration EXISTE déjà** : [`migration/steps_shared_rebuild_match_participants.go`](apps/go-api/internal/migration/steps_shared_rebuild_match_participants.go) avec sentinel one-shot | Phase 4.1 doit se distinguer du rebuild migration : path runtime sans sentinel persistant |
| 6 | **Singleflight = bon ciblage** pour `InsertParticipants` UPSERT. Recommande en plus `SetLimit(1)` sur les UPSERTs des heal loops | Le plan Phase 2.3 est validé. Ajustement : sérialiser les writes participants des heals, garder parallel uniquement sur downloads |
| 7 | **ART rebuild swap-table = SEULE option viable** (REINDEX inexistant, VACUUM FULL non-implémenté, DROP/CREATE de PK peut crasher) | Phase 4.1 confirmée mais à risque MOYEN-HAUT (lock writer ~1s pendant rebuild) |
| 8 | **Données déjà silencieusement perdues** : LUSR de Madina figé Argent IV au lieu de Platine | Le rebuild seul ne suffit pas — il faut re-tourner les batchs de recompute (LUSR, sessions, etc.) après rebuild |

### Nouvelle priorité recommandée par les agents

1. **P0 — Paralléliser `processWeaponKillsInline`** (Agent 3, NEW) — gain ~150-200s/cycle Madina
2. **P0 — Singleflight `InsertParticipants`** (Agent 2 confirmé) — fixe le crash
3. **P0 — Bump driver DuckDB v1.5.2 → v1.5.3** (Agent 2 NEW) — gain non garanti mais low cost
4. **P1 — ART rebuild runtime** (Agent 2 confirmé, MOYEN-HAUT risque) — recovery
5. **P1 — Parallel scheduler RunOnce** (Agent 1 confirmé) — gain 15min → 5-8min
6. **P2 — Fusionner events_heal + weapon_heal** (Agent 3 NEW) — saturate HTTP pool
7. **P2 — Bump `healParallelism` à 16-32** sur paths network-only (Agent 3 NEW)
8. **P3 — Phase 3.2 parse parallel** — dépriorisé (gain modeste)
9. **P3 — Phase 3.3 intra-match parallel API** — dépriorisé (ROI questionnable)

---

## 1. Agent 1 — Audit parallélisation actuelle

### Validation des 6 postulats du plan

| Postulat | Verdict | Détail |
|---|---|---|
| Downloads chunks parallèles entre matchs | **VRAI** | `errgroup.WithContext` sans SetLimit dans Phase 2 ([engine.go:340-366](apps/go-api/internal/sync/engine.go)). Throttling effectif via rate limiter. ROI parallélisation supplémentaire : NUL |
| Parse highlight_events séquentiel dans Phase 3 | **VRAI** | `ParseHighlightEvents` appelé dans `insertHighlightEventsFromData` (Phase 3 séquentiel). Gain réel : **100-500ms** sur 20 matchs (vs 10-30% du plan, surestimé) |
| GetMatchStats + GetMatchSkill + GetHighlightEventsChunk séquentiels intra-match | **VRAI** | 3 calls strictement séquentiels ([engine_fetch.go:50-119](apps/go-api/internal/sync/engine_fetch.go)). Gain réel : ~200-300ms/match (rate limiter cap, pas RTT complet) |
| Scheduler RunOnce séquentiel | **VRAI** | Boucle for-range stricte ([auto_sync.go:357-367](apps/go-api/internal/scheduler/auto_sync.go)). Gain : 15min → 5-8min cohérent |
| Heal loops parallèles dangereux | **VRAI DANGEREUX** | `InsertParticipants` UPSERT (heal stats + heal skill) = cause racine du crash ART |
| Pool 30 RPS théorique | **FAUX** | Config prod = `PerTokenRPS=5 × 3 tokens = 15 RPS effectif`. Burst=1 strict |

### Opportunités supplémentaires identifiées

- **O1** : `processWeaponKillsInline` séquentiel (détaillé par Agent 3 ci-dessous) — gain massif
- **O3** : post-sync 14 étapes strictement séquentielles. Certaines indépendantes (CSR snapshots, achievements Xbox) parallélisables vs aggregates — gain ~1-2s
- **O4** : `refreshAggregates` (player DB) + `refreshSharedViews` (shared DB) sur DBs différentes → parallélisables — gain 500ms-2s

### Risques techniques par optim

| Optim | Risque ART | Risque deadlock | Niveau global |
|---|---|---|---|
| Phase 3.2 parse parallel | Aucun (CPU pur) | Aucun | **BAS** |
| Phase 3.3 intra-match API parallel | Aucun | Aucun | **BAS** |
| Phase 3.4 parallel scheduler | **HAUT sans Phase 2.3**. AVEC singleflight : BAS | Aucun | MOYEN sans 2.3, BAS avec |
| Phase 2.3 singleflight | Cible le bug | Faible | **BAS** |
| Phase 4.1 ART rebuild swap | Critique si rate avec writes concurrents | Possible (lock writer ~1s) | **MOYEN-HAUT** |

---

## 2. Agent 2 — DuckDB ART corruption deep dive

### Validation des 5 postulats

| Postulat | Verdict | Notes |
|---|---|---|
| Corruption ART vient des UPSERTs concurrents (match_id, xuid) | **CONFIRMÉ avec nuance** | Concurrence nécessaire mais l'interaction concurrence × bug DuckDB amont produit la corruption. `healStatsForRecentMatches` et `healSkillForMissingMatches` = déclencheurs principaux |
| Singleflight par (match_id, xuid) résout à la source | **PARTIELLEMENT** | Résout 100% des races intra-process visibles. Limites : ne sérialise pas entre clés différentes, ne masque pas un UPSERT solitaire sur PK déjà corrompue |
| Swap-table = stratégie recovery saine | **CONFIRMÉ — SEULE option viable** | REINDEX inexistant (PostgreSQL only), VACUUM FULL non-implémenté ([issue #21154](https://github.com/duckdb/duckdb/issues/21154)), DROP CONSTRAINT peut crasher sur PK ART corrompu |
| BootARTGuard détecte mais ne corrige pas | **CONFIRMÉ après lecture intégrale** | [`art_probe.go`](apps/go-api/internal/platform/duckdb/art_probe.go) : 4 étapes — énumère PKs, échantillonne, compare `WHERE pk = ?` vs `WHERE pk \|\| '' = ?`, log WARN. Aucune mitigation |
| Bug upstream DuckDB, on ne peut que mitiger | **CONFIRMÉ, version-dépendant** | Driver actuel `duckdb/duckdb-go v2.10502.0` = DuckDB 1.5.2. Issue [#18782](https://github.com/duckdb/duckdb/issues/18782) (août 2025) reste OPEN. v1.5.3 sorti 20 mai 2026 corrige edge case index deletion. v1.4.1 changelog mentionne explicitement "ART index could omit rows non-deterministically when running on multiple threads" |

### Alternatives écartées au singleflight

- `sync.Mutex` global : trop coarse
- Transaction par UPSERT : ne résout rien, bug est dans l'ART pas le commit
- Retry sur Constraint Error : ne marche pas, crash C++ non-récupérable
- Batch INSERT : issue [#8147](https://github.com/duckdb/duckdb/issues/8147) confirme conflits intra-statement non supportés (aggrave)
- Sharding par match_id : équivalent fonctionnel, plus lourd

### Stratégie défense en profondeur (recommandation Agent 2)

1. **Mitigation préventive** : singleflight + `SetLimit(1)` sur heal UPSERTs
2. **Mitigation curative** : rebuild swap-table runtime sans sentinel persistant
3. **Upgrade tactique** : bump driver v1.5.2 → v1.5.3 (faible coût)
4. **Garde-fou** : conserver le crash `FatalException` comme signal, ne pas tenter de "continuer" après

### Risques résiduels après mitigation

1. Bug DuckDB persiste sur writes inter-clés si l'ART partage des nœuds racine
2. Corruption possible hors writes (DELETE+reinsert intra-txn, cf. issue [#16520](https://github.com/duckdb/duckdb/issues/16520))
3. SIGABRT peut tuer le process avant que le handler tourne
4. Rebuild swap pose lock writer ~1s, ~10s sur 500k rows
5. Sentinel `match_participants_rebuilt_v1` actuel bloque les re-runs ; runtime rebuild doit avoir mécanisme distinct
6. **Données déjà silencieusement perdues** : LUSR figé en Argent IV au lieu de Platine pour Madina (cf. commentaire `steps_shared_rebuild_match_participants.go:17`). Recompute LUSR/sessions/etc. nécessaire après rebuild
7. Si la corruption a AUGMENTÉ après l'upgrade driver récent → envisager REVERT vers v1.4.3 LTS

### Sources clés Agent 2

- [DuckDB Issue #18782 — ART concurrent UPSERT (open)](https://github.com/duckdb/duckdb/issues/18782)
- [DuckDB Issue #16520 — Duplicate key during data insert (closed, reproduced)](https://github.com/duckdb/duckdb/issues/16520)
- [DuckDB Issue #8147 — ON CONFLICT intra-statement (closed not planned)](https://github.com/duckdb/duckdb/issues/8147)
- [DuckDB v1.5.2 release notes (PR #21815 ART, #20804 race fixes backport)](https://github.com/duckdb/duckdb/releases/tag/v1.5.2)
- [DuckDB v1.4.3 LTS — Index deletion edge case](https://duckdb.org/2025/12/09/announcing-duckdb-143)

---

## 3. Agent 3 — Performance breakdown des cycles 22 mai

### Timeline Madina (durée totale 7m59s, 21 matchs insérés, 35 healed)

| Étape | Durée | Type dominant |
|---|---|---|
| Pagination + fetch + insert 21 matchs | 12.8s | Network |
| `healStatsForRecentMatches` (0 healed) | ~0s | - |
| `healSkillForMissingMatches` (35 matchs, parallel=8) | 8.3s | Network |
| `healEventsForRecentMatches` (20 no_film, parallel=8) | 4.2s | Network |
| `healWeaponKillsForRecentMatches` (10 healed, parallel=8) | **174.0s** | Network |
| `computeAndPersistHadBotTeammate` | ~0s | DB write |
| **Sessions + perf + engagement + assists + `processWeaponKillsInline` 21 films SÉRIEL + citations** | **274.1s** | **Network (par `processWeaponKillsInline`)** |
| CSR (skipped) | 0.5s | - |
| Friends recompute | <1s | Compute+DB |
| Aggregates + Achievements Xbox | ~5.2s | Network |
| **TOTAL** | **479.1s** | |

**Le gap noir Madina** (`had_bot_teammate` à 18:29:56 → CSR à 18:34:31 = 274s) est presque entièrement consommé par `processWeaponKillsInline` sériel × 21 films × ~10-13s/film.

### Validation des 5 postulats Agent 3

| Postulat | Verdict |
|---|---|
| Network = bottleneck dominant | **VALIDÉ** — 95% du temps Madina dans des paths network |
| Compute prend une part significative | **RÉFUTÉ** — ~5s cumulés sur tous les batchs compute pur (perf, LUSR, engagement, aggregates) |
| DB writes rapides (sub-second) | **VALIDÉ** — aucun "slow query" dans `duckdb.log`, writes noyés dans network |
| Post-sync >> pagination | **VALIDÉ très fortement** — Madina : 2.7% pagination vs 97.3% post-sync |
| heal* network-bound | **VALIDÉ + nuance** — weapon kills heal = path le plus lent (174s Madina) |

### Recommandation Agent 3 — où mettre l'effort

#### Priorité 1 — Paralléliser `processWeaponKillsInline` (NEW, pas dans le plan)

- [`backfill_weapons.go:232-255`](apps/go-api/internal/sync/backfill_weapons.go) — boucle `for matchID := range matchIDs` SANS goroutine
- 21 films × ~10-13s/match = **210-275s SÉRIEL** dans le post-sync Madina
- Pattern : reproduire `errgroup + SetLimit(healParallelism=8)` déjà en place dans `healWeaponKillsForRecentMatches`
- Gain estimé : Madina 210s → 40s sur cette étape (~170s économisés)

#### Priorité 2 — Bump `healParallelism` au-delà de 8 sur paths network-only

- `healSkillForMissingMatches` : un seul GetMatchSkill par match, write rapide → passer à 16-32 safe
- Gain : skill heal de 8s → 4s sur batches de 35

#### Priorité 3 — Fusionner events_heal + weapon_heal dans un seul errgroup multi-endpoint

- `healEventsForRecentMatches` (20 films) + `healWeaponKillsForRecentMatches` (10 films) téléchargent même type de ressource
- Sur Madina, en série : 4s + 174s = 178s
- Fusionnés : saturer le pool HTTP en continu → gain estimé 20-30%

#### Anti-priorité Agent 3

**NE PAS optimiser le compute Go ni les writes DB** — les ~10s cumulés de compute sur 1085 matchs sont déjà excellents. Effort à ce niveau déplace 5-10% d'une enveloppe qui n'est pas le bottleneck.

### Note opérationnelle critique (Agent 3)

JGtm a crashé entre `had_bot_teammate` et `weapon_kills` — soit dans `recalculateSessionsInline`, `batchComputePerformanceScores`, `batchComputeEngagementScores`, `batchRecomputeCoefficients`, `batchComputePlayerAssistsModel` ou `processWeaponKillsInline`. Le `recover()` du commit récent dans `runPostSyncPipeline` devrait désormais loguer le panic.

**41 matchs JGtm × `processWeaponKillsInline` sériel ~10-13s/match = ~8 minutes**. Couplé au scheduler 15min, crée un risque de blocage / OOM. Le paralléliser réduit aussi cette pression.

---

## 4. Agent 4 — Highlight events pipeline feasibility

### Pipeline actuel

#### Phase 2 (parallèle, errgroup pagination)
1. `GetMatchStats` — network rate-limited
2. `ExtractRegistry` + `ExtractParticipants` — compute pur
3. `GetMatchSkill` — network
4. `ExtractMedals` + `ExtractPersonalScoreAwards` — compute
5. `GetHighlightEventsChunk` — network (manifest + blob, inflaté par `downloadBlob` post-fix 9cc4c2bb)
   - Stocke `data []byte` brut dans `fm.HighlightData`

#### Phase 3 (séquentielle, `insertFetchedMatch`)
1. Inserts (registry, participants, medals)
2. **`ParseHighlightEvents(data, filmMajorVersion)`** ← compute lourd (zlib + bit-level scan)
3. `InsertHighlightEvents` (append-only)
4. `UpsertXUIDAlias` loop sur events
5. `MarkEventsLoaded`
6. `InsertKillerVictimPairsFromEvents` (compute O(N²) + DELETE+INSERT)
7. `MarkKillerVictimLoaded`

### Thread-safety du parser

- **Aucune variable de package mutable** dans `scanEvents`
- Globales lues : `medalSortingWeights` (map immuable post-init), `endMarker` (slice constantes), constantes `minXUID`/`maxXUID`/`eventWindowBits`
- État local par appel : `events []HighlightEvent`, `seenPositions map[int]bool` alloués
- `readBytesAtBit` alloue un nouveau `[]byte` par appel — pas de slice partagée
- **Verdict** : parser totalement re-entrant, OK pour 4-8 goroutines parallèles

### Gain estimé du move parse → Phase 2

- Parse time par match : ~30-150ms (sur 800KB inflaté)
- Sur page de 25 matchs avec films : ~2.5s wall-time bloquant Phase 3 actuel
- Phase 2 parallèle 4-8 goroutines : compute distribué sur N cœurs
- **Gain wall-time estimé : 1-1.5s par page de 25 matchs (5-15 % pagination total)**
- Réserve : plan annonce "10-30%", Agent 4 place plutôt dans la fourchette basse

### Blockers techniques

- Dépendances cross-match : **AUCUNE**
- Ordre d'insertion : préservé (Phase 3 itère dans l'ordre original)
- Globales mutables parser : **AUCUNE**
- Couplage `result *domain.SyncResult` dans le parser : à déplacer aussi (`fetchedMatch.HighlightParseAnomaly`)
- `observability.IncCounter` : thread-safe (sync.Int64 atomique)

### Verdict Agent 4

**FAISABLE — gain modéré confirmé**. Conditions :
1. Ajouter à `fetchedMatch` : `HighlightEvents []analysis.HighlightEvent` + `HighlightParseAnomaly bool`
2. Dans `fetchMatchData`, parse après le download, libérer `fm.HighlightData = nil`
3. Refactorer `insertHighlightEventsFromData` en version qui prend events déjà parsés
4. `ProcessHighlightEvents` (path standalone replay) reste intact

**Pré-requis pour mesure** : ajouter timing `parse_ms` par match avant le refactor.

---

## 5. Recalibrage des phases du plan

Les findings agents nécessitent une **réorganisation des priorités du plan**. Tableau de mapping plan → priorité réelle :

| Phase plan | Verdict agents | Action |
|---|---|---|
| Phase 2.1 — Cartographie writers | Validé, à conserver | À démarrer en premier |
| Phase 2.2 — ADR 0018 | Validé, à conserver | Idem |
| Phase 2.3 — Singleflight `InsertParticipants` | Validé, **fixe le crash** | **P0** |
| Phase 3.1 — Vérif download parallèle | DÉJÀ OK | À skipper (juste documenter) |
| Phase 3.2 — Move parse → Phase 2 | Faisable, gain modeste 100-500ms | **P3** dépriorisé |
| Phase 3.3 — Intra-match parallel API | Faisable, gain marginal 200-300ms/match | **P3** dépriorisé fortement |
| Phase 3.4 — Parallel scheduler RunOnce | Validé, gain 15min → 5-8min | **P1** |
| Phase 3.5 — Heal loops avec singleflight | Validé | Avec Phase 2.3 |
| Phase 4.1 — ART rebuild runtime | Validé, risque MOYEN-HAUT | **P1** mais pas avant 2.3 |
| Phase 4.2 — Signal handler SIGABRT | Validé, limites connues | P2 |
| Phase 4.3 — Métriques expvar | Validé | P2 |
| Phase 5.* — Tests régression | Validé, TDD requis | TDD parallèle aux phases |

### Phases NOUVELLES à ajouter au plan

| Nouvelle phase | Source | Gain estimé | Priorité |
|---|---|---|---|
| **3.0 — Paralléliser `processWeaponKillsInline`** | Agent 3 | ~150-200s/cycle Madina | **P0** (le plus gros gain identifié) |
| **3.6 — Bump `healParallelism` 8 → 16-32** sur paths network-only | Agent 3 | ~4s/cycle | P2 |
| **3.7 — Fusionner events_heal + weapon_heal** dans un errgroup unique | Agent 3 | ~20-30% sur ces deux étapes | P2 |
| **3.8 — Bump driver DuckDB v1.5.2 → v1.5.3** | Agent 2 | Non garanti, low cost | **P0** (à tester en premier — peut résoudre seul) |
| **4.4 — Parallel `refreshAggregates` + `refreshSharedViews`** (O4) | Agent 1 | 500ms-2s | P3 |

### Phases à RETIRER ou DÉPRIORISER

| Phase | Justification |
|---|---|
| Phase 3.2 (move parse) | Gain 100-500ms réel (vs 10-30% annoncé). Pas worth le refactor sauf si bonus |
| Phase 3.3 (intra-match parallel API) | Gain 200-300ms/match capped par rate limiter. ROI questionnable |

---

## 6. Nouvelle priorité d'exécution recommandée

1. **P0a — Bump driver DuckDB v1.5.2 → v1.5.3** (1h, NEW)
   - Modifier `go.mod`, lancer tests, observer si la corruption ART régresse spontanément
   - Si oui : énorme gain, on évite tout le reste de la stabilisation
   - Si non : on continue avec singleflight
2. **P0b — Phase 2.3 Singleflight `InsertParticipants` + Phase 5.1 stress test TDD** (4h)
   - Test stress concurrent écrit AVANT le fix
   - Singleflight wrapper sur `InsertParticipants`
   - `SetLimit(1)` sur UPSERTs heal stats + heal skill (mais 8 sur les downloads)
3. **P0c — NEW Phase 3.0 Paralléliser `processWeaponKillsInline`** (2h)
   - Pattern `errgroup.SetLimit(healParallelism=8)` reproduit depuis `healWeaponKillsForRecentMatches`
   - Gain le plus impactant : ~150-200s/cycle Madina
4. **P1a — Phase 4.1 ART rebuild runtime** (3h)
   - Trigger via `BootARTGuard` extension périodique (pas que au boot)
   - Path runtime sans sentinel persistant (sentinel migration existant intact)
5. **P1b — Phase 3.4 Parallel scheduler RunOnce** (1h)
   - Après singleflight validé en stress test
   - Gain 15min → 5-8min
6. **P2 — Phase 4.2 + 4.3** (signal handler + métriques expvar) (3h)
7. **P2 — NEW 3.6 + 3.7** (bump healParallelism + fusion heal loops) (2h)
8. **P3 — Phase 5.2-5.5** (tests restants : property-based, E2E, bench, ART rebuild) (8h)
9. **P3 — Phase 3.2 et 3.3 et NEW 4.4** (gains marginaux, à faire si temps)

**Total recalibré : ~24h** (vs ~21h plan initial — la P0c NEW ajoute 2h, mais on dépriorise 3.2 et 3.3)

---

## 7. Recompute post-rebuild critique (oublié dans le plan)

Agent 2 a relevé que **les données ont été silencieusement perdues** : LUSR de Madina figé Argent IV au lieu de Platine. Le rebuild swap-table récupère les rows mais les batchs computed sur ces rows (LUSR cascade, sessions, perf scores, citations, dominance) ont produit des résultats faux pendant la période corrompue.

**Action à ajouter au plan** : après le rebuild ART de Phase 4.1, lancer un recompute force=true pour les joueurs affectés :
- `batchComputeLUSR(playerDB, sharedDB, xuid, ..., force=true)`
- `batchComputePerformanceScores(..., force=true)`
- `BackfillDominanceFlags(...)` pour les matchs concernés
- `RecomputeIsWithFriendsCore(...)`

Sans ce recompute, les rapports LUSR / KPIs / sessions resteront faux même après le rebuild.

---

## 8. État du driver DuckDB et fenêtre d'opportunité

- Driver actuel : `github.com/duckdb/duckdb-go/v2 v2.10502.0` → DuckDB **1.5.2**
- Driver disponible : **v1.5.3** (sorti 20 mai 2026), corrige edge case index deletion
- v1.4.1 changelog : *"ART index could omit rows non-deterministically when running on multiple threads"* — symptôme exact observé

**Hypothèse à valider en premier** : la régression du 22 mai est-elle apparue APRÈS l'upgrade vers 1.5.2 ? Si oui, deux options :
- **Avancer** vers 1.5.3 (pari sur le fix non documenté)
- **Reculer** vers 1.4.3 LTS (stabilité garantie, perd 2 mois de features)

Recommandation : tenter 1.5.3 d'abord (effort minimal). Si toujours instable après 24h en prod, fallback 1.4.3 LTS.

---

## 9. Sources et fichiers analysés

### Fichiers projet
- `apps/go-api/internal/sync/engine.go`
- `apps/go-api/internal/sync/engine_fetch.go`
- `apps/go-api/internal/sync/engine_postsync.go`
- `apps/go-api/internal/sync/engine_process_match.go`
- `apps/go-api/internal/sync/engine_highlight_events.go`
- `apps/go-api/internal/sync/events_heal.go`
- `apps/go-api/internal/sync/skill_heal.go`
- `apps/go-api/internal/sync/stats_heal.go`
- `apps/go-api/internal/sync/backfill_weapons.go`
- `apps/go-api/internal/sync/writes.go`
- `apps/go-api/internal/sync/halo_client.go`
- `apps/go-api/internal/sync/pooled_client.go`
- `apps/go-api/internal/analysis/highlight_event_parser.go`
- `apps/go-api/internal/scheduler/auto_sync.go`
- `apps/go-api/internal/platform/duckdb/art_probe.go`
- `apps/go-api/internal/platform/auth/pool/pool.go`
- `apps/go-api/internal/migration/steps_shared_rebuild_match_participants.go`
- `apps/go-api/internal/migration/steps_shared_social_purge_data_health.go`
- `apps/go-api/cmd/server/main.go`
- `apps/go-api/go.mod`
- `logs/sync.log`, `logs/scheduler.log`, `logs/duckdb.log` (timings cycle 22 mai)

### Sources externes (date d'accès 22 mai 2026)
- [DuckDB Issue #18782 — ART concurrent UPSERT](https://github.com/duckdb/duckdb/issues/18782)
- [DuckDB Issue #16520 — Duplicate key during data insert](https://github.com/duckdb/duckdb/issues/16520)
- [DuckDB Issue #8147 — ON CONFLICT intra-statement](https://github.com/duckdb/duckdb/issues/8147)
- [DuckDB Issue #11102 — INSERT OR IGNORE multiple violations](https://github.com/duckdb/duckdb/issues/11102)
- [DuckDB Issue #21154 — VACUUM FULL not implemented](https://github.com/duckdb/duckdb/issues/21154)
- [DuckDB v1.5.1 release notes](https://duckdb.org/2026/03/23/announcing-duckdb-151)
- [DuckDB v1.5.2 release tag](https://github.com/duckdb/duckdb/releases/tag/v1.5.2)
- [DuckDB v1.4.3 LTS](https://duckdb.org/2025/12/09/announcing-duckdb-143)
- [DuckDB Concurrency docs](https://duckdb.org/docs/current/connect/concurrency)
- [DuckDB Constraints docs (ART auto-create)](https://duckdb.org/docs/lts/sql/constraints)
- [DuckDB VACUUM docs](https://duckdb.org/docs/current/sql/statements/vacuum)
- [DuckDB Indexes docs (no REINDEX)](https://duckdb.org/docs/current/sql/indexes)

---

## 10. Conclusion / prochaine étape

**Le plan initial était globalement bon, mais les chiffres de gain étaient surestimés et un gros gain a été oublié** (`processWeaponKillsInline`).

**Recommandation immédiate à l'utilisateur** :
1. Lire ce handoff + valider la recalibrage.
2. Décider :
   - On démarre par P0a (bump driver 1.5.3) — pari à 1h, peut résoudre seul
   - Ou directement par P0b+P0c (singleflight + parallel weapon_kills_inline)
3. Le plan `PLAN_SYNC_CONCURRENCY_STABILIZATION.md` sera amendé après validation de cette priorisation.

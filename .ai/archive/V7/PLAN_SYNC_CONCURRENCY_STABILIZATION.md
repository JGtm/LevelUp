# PLAN — Stabilisation sync : parallélisation network + safety concurrent + tests régression

**Statut** : Brouillon, **amendé 2026-05-22 après audit handoff** (4 agents parallèles), attente validation utilisateur. Pas une ligne de code écrite sur les phases ci-dessous.
**Auteur** : Claude (agent IA), session du 2026-05-22.
**Branche cible** : à définir (probablement nouvelle branche depuis `fix/media-paths-portable` ou main).
**Effort estimé** : **~24h** de travail focus (recalibré post-audit, +3h vs estimation initiale).
**Handoff prérequis** : [`HANDOFF_SYNC_CONCURRENCY_AUDIT.md`](HANDOFF_SYNC_CONCURRENCY_AUDIT.md) — DOIT être lu en parallèle de ce plan.

---

## 0. Philosophie et priorités (DIRECTIVES UTILISATEUR + RECALIBRAGE AUDIT)

**Le bottleneck c'est UNIQUEMENT le réseau, pas le compute ni les writes DB.**

L'audit du 22 mai (Agent 3, breakdown logs réels) a mesuré :
- **Network = 95%** du temps Madina (~456s sur 479s) — appels Halo API à 300-500ms RTT vers Azure US
- **Compute Go = ~5%** cumulés sur tous les batchs (perf scores, LUSR, engagement, aggregates) → **~5s total pour 1085 matchs**
- **DB writes = négligeable** (sub-second, jamais "slow query" loggué)

→ **Parallélisation à privilégier (P0/P1)** :
- Téléchargements API qui sont actuellement séquentiels — notamment **`processWeaponKillsInline`** (séquentiel sur 21 films × ~10-13s = 210-275s par cycle Madina, gros gain ~150-200s)
- Scheduler RunOnce séquentiel sur joueurs → parallèle (gain 15min → 5-8min)

→ **Parallélisation à NE PAS prioriser (correction du plan initial)** :
- **Writes DB** : DuckDB sérialise de toute façon. Wanting 8 goroutines qui UPSERT la même table = races ART pour zéro gain perf
- **Compute** : 5% du temps total. Optimiser le parse zlib / scanEvents / batchComputeLUSR déplace 5-10% d'une enveloppe qui n'est PAS le bottleneck. Phase 3.2 (move parse) et Phase 3.3 (intra-match parallel API) **dépriorisées P3** (ROI réel : 100-500ms et 200-300ms/match capped par rate limiter)

**Singleflight (Phase 2)** : sert à la SAFETY (empêcher 2 paths concurrents de UPSERT la même row simultanément, ce qui corrompt l'index ART), **PAS à la perf**.

**Configuration rate limiter prod (à connaître)** : `PerTokenRPS=5 × 3 tokens = 15 RPS effectif` (et non 30 RPS comme initialement supposé). Burst=1 strict. C'est une borne dure qui cap tous les gains de parallélisation API intra-match.

---

## 1. Contexte et historique

### 1.1 Le problème initial (20 mai 2026)

L'utilisateur a signalé 4 problèmes :
1. Sync auto qui n'insère plus de matchs depuis le 6 mai (14 jours de silence).
2. Sync lente + pas assez parallélisée.
3. Crashes silencieux sans stack trace.
4. Données incomplètes (dominance flags pas câblés).

### 1.2 Ce qui a été fait depuis (déjà dans HEAD)

| Fix | Statut | Notes |
|---|---|---|
| URL endpoint `/hi/players/xuid(NNN)/matches` | Validé live | 83 matchs insérés post-fix |
| Delta loop : `goto done` → `stopAfterFlush` | Validé live | Évite de drop les nouveaux matchs collectés avant un known |
| Log INFO "sync: 1er match retourné par API" | Passif | Sentinelle de fraîcheur API |
| Counter `ConsecutiveZeroInserts` + WARN seuil 6 | Passif | Alerte si N cycles sans insert |
| `debug.SetCrashOutput` vers `logs/server.crash.log` | Partiel | Capte panics Go, PAS les `duckdb::FatalException` C++ |
| `defer recover()` dans `runPostSyncPipeline` | Partiel | Idem, bypass C++ |
| Heartbeat goroutine 30s | Passif | Distingue crash / deadlock / kill externe |
| Dominance flags wiring dans post-sync (étape 1.7) | Validé | `BackfillDominanceFlags` câblée pour les nouveaux matchs |
| Cross-player dedup : `loadKnownMatchIDs` UNION `shared.match_participants WHERE xuid=?` | Validé tests | Choco skippe le fetch API quand Madina vient de syncer |
| Parallel heal loops (errgroup + SetLimit 8) | **Suspect** | Concurrent UPSERTs match_participants aggravent races ART |
| Double zlib `ParseHighlightEvents` rendu tolérant | Validé | Fix régression du 8 mai sur fresh downloads |
| ~~Scoreboard Q12 WHERE relâché~~ | **REVERTÉ** | N'a fixé qu'1 match sur 1610, pas worth le diff |

### 1.3 Le crash qui a déclenché ce plan

```
terminate called after throwing an instance of 'duckdb::FatalException'
  what():  "INTERNAL Error: Failed to append to PRIMARY_match_participants_match_id_xuid:
  Constraint Error: PRIMARY KEY or UNIQUE constraint violation:
  duplicate key 0941d737-1fb4-4a11-8a9a-169624911729, 2533274828226170"
```

Crash C++ levé depuis le binding cgo DuckDB. Bypass `recover()` Go et `debug.SetCrashOutput`. Le process meurt brutalement.

### 1.4 Le vrai bug racine — ART index corruption DuckDB

Diagnostic via l'agent d'audit :

- **ART (Adaptive Radix Tree)** = structure d'index interne que DuckDB utilise pour les PRIMARY KEY et UNIQUE constraints.
- Sur certains patterns d'UPSERT répétés avec concurrence, l'index ART se désynchronise de la table : la PK existe physiquement mais le lookup index la rate.
- Conséquences :
  - Queries `WHERE match_id = ?` renvoient 0-N rows incomplètes (data physiquement présente mais invisible via index).
  - INSERTs ultérieurs croient pouvoir ajouter (l'index dit "absent") puis explosent au commit avec `Constraint Error: duplicate key`.

**Détection actuelle** : `BootARTGuard` ([art_probe.go:251](apps/go-api/internal/platform/duckdb/art_probe.go)) existe déjà, log WARN au boot mais **ne corrige rien**.

**Quantification** :
- 8 matchs détectés ART-corrompus depuis le 22 mai 18:11 (incl. `0941d737-...` du crash).
- Sur 50 samples random : ~2% des matchs présentent `indexed != scan` (~32 matchs sur 1610 estimés).
- Match `de3cec8b-...` : 1 participant via index, 9 via scan → 8 invisibles côté scoreboard.

**5 queries vulnérables** (toutes `WHERE match_id = ?` sur match_participants) : Q12, Q16, Q17, Q26, Q28.

---

## 1bis. Phase 0 — Bump driver DuckDB v1.5.2 → v1.5.3 (NEW, P0a) — ✅ FAIT 2026-05-22 (commit 25b56846)

**Effort réel : ~30min. Pari low-cost en place.**

**Validation déploiement** : tests verts (sync 10.5s, analysis 4 pkgs, scheduler 1.1s, platform/duckdb 12.5s, validation, watcher). vet clean.

**À surveiller en prod (24h)** :
- Plus de `duckdb::FatalException` crash
- `art_corruption_detected_total` n'incrémente pas sur de NOUVEAUX matchs
- Si KO → continuer Phase 1 (singleflight). Si AGGRAVATION → revert v1.4.3 LTS.

**Source** : Audit Agent 2 (handoff §2).

**Rationale** :
- Driver actuel : `github.com/duckdb/duckdb-go/v2 v2.10502.0` → DuckDB **1.5.2**.
- **v1.5.3** sorti le 20 mai 2026, corrige un edge case d'index deletion.
- DuckDB v1.4.1 changelog mentionne **exactement notre symptôme** : *"ART index could omit rows non-deterministically when running on multiple threads"*.
- v1.5.2 a backporté des race condition fixes (PR #20804) mais pas tous (issue #18782 reste OPEN).

**Étapes** :
1. `go get github.com/duckdb/duckdb-go/v2@v2.10503.0` (vérifier le tag exact dispo).
2. `go mod tidy`, `go build ./...`, `go test ./...` — observer si la suite passe sans régression.
3. Déployer en dev, lancer un cycle sync complet, vérifier les compteurs `BootARTGuard` après 24h.
4. **Si la corruption ART disparaît** : on évite singleflight + rebuild runtime, gain énorme.
5. **Si la corruption persiste** : on continue avec Phase 1 (singleflight).
6. **Si la corruption AUGMENTE après bump** : REVERT vers v1.4.3 LTS au lieu d'avancer.

**Critères de complétion** :
- `go.mod` updated, build clean, tests verts.
- 24h en dev sans crash `duckdb::FatalException`.
- `art_corruption_detected_total` n'incrémente pas sur de NOUVEAUX matchs.

**Risque** : nouveau bug introduit par v1.5.3. Mitigation : revert v1.4.3 LTS si KO.

---

## 2. Phase 1 — Safety concurrent (protection ART)

Effort : ~4h. **Objectif : empêcher les races ART, PAS gagner de la perf sur les writes.**

### 2.1 Cartographier tous les writers de `match_participants` — ✅ FAIT 2026-05-23

Tableau exhaustif livré dans [ADR 0018 §Context](../../docs/adr/0018-concurrent-write-model.md#cartographie-des-writers-sur-sharedmatch_participants). 8 callers identifiés, dont 2 problématiques (heal stats + heal skill, errgroup 8 goroutines).

**Livrable** : tableau dans l'ADR 0018 (§2.2).

**Sources connues à auditer** :
- `sync/writes.go:103` — `InsertParticipants` (UPSERT générique, point d'entrée unique)
- `sync/writes.go:551` — `MarkSkillLoaded` (UPDATE backfill_bits)
- `sync/engine_fetch.go:141` — sync engine pagination (Phase 3 sequential, déjà OK)
- `sync/stats_heal.go:94` — heal stats (8 goroutines depuis Action B)
- `sync/skill_heal.go:116` — heal skill (8 goroutines depuis Action B)
- `sync/backfill_weapons.go:265` — backfill séquentiel (OK)
- `service/openspartan_import_service.go:313` — import OpenSpartan one-shot (OK)

### 2.2 ADR 0018 — Concurrent Write Model — ✅ FAIT 2026-05-23

Livré : [`docs/adr/0018-concurrent-write-model.md`](../../docs/adr/0018-concurrent-write-model.md). Status Proposed (Accepted après livraison Phase 1.3).

**Livrable** : `docs/adr/0018-concurrent-write-model.md`.

**Contenu** :
- Cartographie des tables shared et leur policy de concurrence.
- Pattern obligatoire : `dblease.AcquireWriter` + singleflight par clé naturelle pour les tables PK-indexées.
- Liste des tables avec policy explicite :
  - `shared.match_participants` : singleflight par `(match_id, xuid)` — bug ART connu.
  - `shared.match_registry` : `INSERT IF NOT EXISTS` suffit, déjà protégé.
  - `shared.medals_earned` : `INSERT OR IGNORE` suffit, déjà protégé.
  - `shared.weapon_kills` : append-only, pas besoin.
  - `shared.highlight_events` : append-only, pas besoin.
- Référence ADR 0016 (B-swap RO↔RW).

### 2.3 Singleflight par `(match_id, xuid)` pour `match_participants` — ✅ FAIT 2026-05-23 (commit aef47968)

**Livré** : `participantsSF singleflight.Group` package-level dans [`writes.go`](../../apps/go-api/internal/sync/writes.go). `InsertParticipants` dédupe par `"match_id|xuid"`. Logique SQL extraite dans `insertParticipantRow`. Sémantique : N appelants sur même clé → 1 SQL exec, autres reçoivent le résultat partagé.

**Validation TDD** : `TestStressUpsertParticipants_*` (3 tests) passent post-fix après avoir échoué baseline (49 + 1061 + 28 failures). Perfs : 8.3s → 0.7s sur même-clé grâce au dedupe.

**But** : SAFETY, pas perf. Empêcher 2 goroutines de UPSERT la même row simultanément (cause de la race ART).

**Design** : wrapper autour de `InsertParticipants` :
```go
// Pseudo-code à valider en impl.
var participantsSF singleflight.Group

func InsertParticipantsSafe(ctx context.Context, db *sql.DB, rows []ParticipantRow) error {
    for _, row := range rows {
        key := row.MatchID + "|" + row.XUID
        _, err, _ := participantsSF.Do(key, func() (any, error) {
            return nil, insertSingleParticipant(ctx, db, row)
        })
        if err != nil { return err }
    }
    return nil
}
```

**Tests** : voir Phase 4.1.

---

## 3. Phase 2 — Parallélisation network (le vrai gain perf)

Effort : **~9h** (recalibré, +4h vs estimation initiale).

**Priorité réelle après audit Agent 3** : paralléliser le path le plus coûteux du post-sync = `processWeaponKillsInline`. Les sous-phases 3.2 et 3.3 sont dépriorisées (gains marginaux confirmés par audit).

### 3.0 NEW — Paralléliser `processWeaponKillsInline` EN MODE TDD — ✅ FAIT 2026-05-23 (commit fc772f80)

**Livré** : [`backfill_weapons.go::processWeaponKillsInline`](../../apps/go-api/internal/sync/backfill_weapons.go) → errgroup.SetLimit(healParallelism=8), mu.Mutex sur compteurs, best-effort par-match, cancel propagé via ctx.Err().

**Tests TDD** ([`backfill_weapons_parallel_test.go`](../../apps/go-api/internal/sync/backfill_weapons_parallel_test.go)) : 4 tests écrits AVANT impl :
- `Concurrent_NoRace` (100 matchs, -race clean)
- `LatencyParallelFasterThanSequential` (16×100ms latence → ÉCHEC baseline 1.6s, PASSE post-impl)
- `Idempotent` (2 runs successifs)
- `CancelMidRun` (ctx.Cancel à mi-parcours)

**4 tests existants** `TestProcessWeaponKillsInline_*` toujours verts post-refactor.

**Source** : Audit Agent 3 (handoff §3). **C'est le gain le plus impactant identifié — ~150-200s économisés par cycle Madina.**

**Constat audit** :
- `processWeaponKillsInline` ([backfill_weapons.go:232-255](apps/go-api/internal/sync/backfill_weapons.go#L232)) est une boucle `for matchID := range matchIDs` **SANS goroutine, séquentielle**.
- Sur cycle Madina 22 mai : 21 films × ~10-13s/match = **210-275s** dans le "gap noir" entre `had_bot_teammate` (18:29:56) et CSR (18:34:31).
- Sur cycle JGtm : 41 films × ~10-13s = **~8 minutes potentielles** sériel. Crash JGtm probablement dans ce gap.
- Le pattern `errgroup + SetLimit(healParallelism=8)` existe DÉJÀ dans `healWeaponKillsForRecentMatches` ([events_heal.go:179-202](apps/go-api/internal/sync/events_heal.go#L179)) — à reproduire.

**Approche obligatoire : TDD STRICT** (directive utilisateur) :

#### 3.0.a — Écrire les tests AVANT de modifier le code (1h)

**Livrables** :
- `apps/go-api/internal/sync/backfill_weapons_test.go` (extension) : nouvelle suite `TestProcessWeaponKillsInline_*`.
- **Test du contrat de sortie** (le plus important) : doit définir précisément ce qu'on attend en sortie de `processWeaponKillsInline` avant de toucher au code :
  - Pour N matchs en entrée, retourne `(done int, noFilm int, err error)` avec valeurs prédictibles
  - Les writes en DB sur `weapon_kills` doivent être identiques bit-à-bit à la version séquentielle
  - L'ordre des writes par `(match_id, xuid)` n'importe pas (table append-only)
  - Pas de fuite goroutine après retour (vérif via `runtime.NumGoroutine()`)
- **Test idempotence** : 2 runs consécutifs sur les mêmes match_ids → même résultat, pas de doublons
- **Test annulation** : `ctx.Cancel` à mi-parcours → les matchs déjà traités restent committés, les autres annulés sans corruption
- **Test stress concurrent** : 100 matchs en parallèle, run avec `-race`, assert pas de panic ni race detected
- **Test fixture réaliste** : utiliser un mock client qui simule des latences variables (50-2000ms) pour vérifier que la parallélisation gagne effectivement du temps

**Critère de complétion 3.0.a** : tous les tests écrits ÉCHOUENT sur le code actuel (séquentiel) OU passent comme baseline. Documenter la baseline temps d'exécution sur N matchs.

#### 3.0.b — Implémenter la parallélisation (1h)

**Conditions** : tous les tests 3.0.a écrits AVANT d'éditer `backfill_weapons.go`.

**Pattern à appliquer** (à valider lors de l'impl, basé sur `healWeaponKillsForRecentMatches`) :
```go
// Pseudo-code
var mu sync.Mutex
eg, egCtx := errgroup.WithContext(ctx)
eg.SetLimit(healParallelism) // 8 ; const partagée events_heal.go
for _, matchID := range matchIDs {
    matchID := matchID
    eg.Go(func() error {
        if egCtx.Err() != nil { return egCtx.Err() }
        result, err := backfillSingleMatchWeaponKills(egCtx, ...)
        mu.Lock()
        if result.found { done++ } else { noFilm++ }
        mu.Unlock()
        return nil // best-effort
    })
}
_ = eg.Wait()
```

**Tests** : faire passer la suite 3.0.a au vert. Mesurer le gain réel via benchmark Phase 5.4.

**Critère de complétion 3.0.b** : tests verts, build clean, benchmark montre ≥ 3× speedup sur 20 matchs.

### 3.1 Vérification : highlight_events chunks téléchargés en parallèle entre matchs (0.5h)

**À vérifier** dans le code actuel :
- Phase 2 de la pagination (errgroup parallèle) appelle `fetchMatchData` par match.
- Dans `fetchMatchData` : `GetMatchStats` → `GetMatchSkill` → `GetHighlightEventsChunk` séquentiels.
- Le `GetHighlightEventsChunk` télécharge UN chunk par match → l'analyse parallèle se fait entre matchs (via le errgroup parent), pas dans-match.

**État actuel** : download chunk = parallèle entre matchs (via errgroup pagination).
→ **Déjà OK pour le téléchargement entre matchs**. La vérification confirme juste que ça marche bien.

### 3.1bis NEW — Paralléliser download chunks REPLICATION_DATA INTRA-FILM dans GetMatchFilm — ✅ FAIT 2026-05-23 (commit f00468c7)

**Source** : opportunité ratée dans le sprint initial 2026-05-22, reprise sur demande utilisateur explicite ("c'est tout le traitement et les calculs qu'il faut optimisier niveau perf").

**Constat** : `GetMatchFilm` ([halo_client.go:371](apps/go-api/internal/sync/halo_client.go#L371)) téléchargeait les chunks d'un film via une boucle `for` séquentielle. Un film typique = 10-30 chunks × 200-500ms RTT CDN = **3-15s wall-time par film**. Phase 3.0 avait parallélisé "1 goroutine par match" donc N films en parallèle, mais à l'intérieur de chaque match les chunks restaient sériels.

**Livré** : `errgroup.SetLimit(filmChunkParallelism=8)` télécharge les chunks REPLICATION_DATA d'un film en parallèle. Mesure TDD : 10 chunks × 100ms latence simulée = **1010ms baseline → 203ms parallèle (5× speedup)**.

**Architecture race-free sans mutex** :
1. Phase séquentielle : pré-filtre cache hits vs misses (cache check = accès fichier local rapide).
2. Phase parallèle : chaque goroutine écrit dans un slot pré-alloué de `dlResults[]` indexé par position dans `toDownload` (jamais par `chunk.Index` qui peut être sparse). Pas de map concurrent.
3. Phase séquentielle post-`eg.Wait()` : assemble le map final. Race-free par construction.

**Garde "traitement attend tous DL"** : `GetMatchFilm` ne retourne qu'après `eg.Wait()` + assemblage. Le caller `BackfillWeaponKillsForMatch` reçoit soit une erreur, soit un map ENTIÈREMENT rempli — impossible que `BuildWeaponTimelines`/`ScanFireEventsAll` voient un `rawChunks` partiel. Test `CompletesAllBeforeReturn` verrouille via compteur HTTP atomique == N au retour.

**6 tests TDD écrits AVANT impl** ([halo_client_film_parallel_test.go](../apps/go-api/internal/sync/halo_client_film_parallel_test.go)) :
- `ParallelDownloadFasterThanSequential` : test perf principal. ÉCHOUE baseline (1.01s), PASSE post-impl (0.20s).
- `PreservesAllChunks` : ordre + contenu + métadonnées preserves, détecte les swaps inter-goroutines.
- `CompletesAllBeforeReturn` : garde critique pour le caller.
- `OneChunkFails_ReturnsError` : errgroup propagation.
- `CancelMidDownload` : ctx.Cancel à mi-parcours.
- `NoRace` : 30 chunks parallèles `-race` clean.

4 tests existants (`BasicPrefix`, `MultiChunk`, `FilmAbsent`, `DownloadFails`) restent verts → **no impact**. Suite complète : unit 10.7s + race 1.2s + integration 38s tout vert.

**Gain attendu en prod** : sur cycle Madina 21 films × 6s gaspillés intra-film en série = **~120s économisés** en plus du gain Phase 3.0 (~150-200s). Combiné, on attaque les fondations du "cycle 15min".

### 3.2 ~~Move parse highlight_events vers la phase parallèle~~ **DÉPRIORISÉ P3** (1.5h)

**Source** : Audit Agent 1 + Agent 4 — gain réel mesuré beaucoup plus faible qu'estimé.

**Constat audit** :
- Parse time par match : 30-150ms (sur 800KB inflaté).
- Sur page de 25 matchs avec films : ~2.5s wall-time bloquant Phase 3 actuel.
- Gain réel parallélisé à 4-8 cœurs : **~100-500ms par page**, soit 5-15% du wall-time pagination (vs 10-30% annoncé initialement).
- Agent 3 a confirmé : **compute = 5% du temps total**. Optimiser ici déplace 5% d'une enveloppe non bottleneck.

**Verdict** : faisable techniquement (parser re-entrant, blockers nuls — cf. handoff §4), mais ROI faible. À faire en bonus si du temps reste après les P0-P2.

**Implémentation si retenu** : ajouter à `fetchedMatch` : `HighlightEvents []analysis.HighlightEvent` + `HighlightParseAnomaly bool`. Dans `fetchMatchData` parse après download + libérer `fm.HighlightData = nil`. Refactorer `insertHighlightEventsFromData` en version qui prend events déjà parsés.

### 3.3 ~~Intra-match : paralléliser GetMatchStats + GetMatchSkill + GetHighlightEventsChunk~~ **DÉPRIORISÉ P3** (1h)

**Source** : Audit Agent 1.

**Constat audit** :
- Rate limiter prod = `PerTokenRPS=5 × 3 tokens = 15 RPS effectif` (et non 30).
- Burst=1 strict → pas de "bouffée", 200ms strict entre tokens.
- Lancer 3 calls simultanés par match au lieu de 3 séquentiels n'augmente PAS le RPS effectif, ça change juste la latence wall-clock perçue par match.
- Gain wall-clock par match : **~200-300ms** (vs 1 RTT complet de 300-500ms annoncé).
- Sur 20 matchs : **~1-2 secondes max** (et non ~10s annoncés).

**Verdict** : ROI trop faible vs effort de refactor. À skipper sauf si on a un trou de planning.

### 3.4 Paralléliser le scheduler `RunOnce` — ✅ FAIT 2026-05-23 (commit b439f73e)

**Livré** : [`auto_sync.go::RunOnce`](../apps/go-api/internal/scheduler/auto_sync.go) — boucle `for` séquentielle remplacée par `errgroup.WithContext(ctx)` + `eg.SetLimit(poolSizeSafe())` (clamp à 1 si pool absent). Compteurs locaux `Synced/Skipped/Failed` protégés par `atomic.Int32`. Best-effort : un syncPlayer en échec n'annule pas les autres goroutines.

**Gain mesuré (test)** : 4 joueurs × 400ms latence simulée = 1.6s séquentiel → 0.42s parallèle (4× speedup). Sur prod 3 joueurs réels, gain estimé 15min → 5-8min par cycle.

**Safety** :
- `syncPlayer` → `recordOutcome` déjà protégé par `s.snapshotMu`.
- Writes `shared.match_participants` sérialisés par dblease.leaseMutex + singleflight phase 2.3 (commit aef47968).
- Pas de double-writer cgo DuckDB.

**4 tests TDD écrits AVANT impl** ([`auto_sync_parallel_test.go`](../apps/go-api/internal/scheduler/auto_sync_parallel_test.go)) :
- `LatencyFasterThanSequential` : baseline rouge 1.6s → vert post-impl 0.42s.
- `CountersPreserved` : 4 joueurs OK → Synced=4.
- `MixedOutcomes_Counted` : 2 OK + 1 FAIL + 1 SKIP → counts exactement préservés sous parallélisme.
- `CtxCancelDrainsProperly` : cancel mid-cycle, pas de crash, Total stable.

Suite scheduler complète verte avec `-race` (2.4s). Aucune régression sur les 12 tests `RunOnce*` existants.

### 3.5 Heal loops parallèles — keep network parallel, sérialiser les writes (1h)

**Décision philosophique** :
- Les heal loops font tous `GetMatch[Stats|Skill|HighlightEventsChunk]` (réseau lent) puis UPSERT (DB rapide).
- Garder les goroutines parallèles pour les CALLS API (Phase Action B).
- Mais via le singleflight Phase 2.3, les UPSERTs sur la même `(match_id, xuid)` se sérialisent naturellement — pas de race.

**État final attendu** :
- `healEventsForRecentMatches` : 8 goroutines parallèles (downloads films) + writes `highlight_events` (append-only, pas de race).
- `healWeaponKillsForRecentMatches` : idem (writes `weapon_kills` append-only).
- `healSkillForMissingMatches` : 8 goroutines parallèles + writes `match_participants` protégés par singleflight.
- `healStatsForRecentMatches` : idem.

### 3.6 NEW — Bump `healParallelism` 8 → 24 sur paths network-only — ✅ FAIT 2026-05-23 (commit 6e437986)

**Source** : Audit Agent 3 (handoff §3 priorité 2).

**Livré** : introduction de `healParallelismNetworkOnly = 24` dans `events_heal.go` à côté du `healParallelism = 8` existant. Bumped les 3 heal loops dont les writes touchent uniquement des tables append-only :
- `healEventsForRecentMatches` (writes `highlight_events`)
- `healWeaponKillsForRecentMatches` (writes `weapon_kills`)
- `processWeaponKillsInline` (writes `weapon_kills`, déjà parallélisé en phase 3.0)

**Conservé à 8** : `healSkillForMissingMatches` + `healStatsForRecentMatches` (UPSERT sur `match_participants` protégés par singleflight phase 2.3, mais opérations CGO plus lourdes).

**Throttle réel** = rate limiter HTTP du HaloAPIClient (~15 RPS effectif sur 3 tokens). Les goroutines supplémentaires attendent le slot ; pas de pression mémoire.

**Audit Agent 3** estime gain ~8s → 4s sur skill_heal (similaire sur events/weapon_kills, mais ici on bump uniquement les network-only car les writes sur match_participants doivent rester capped).

**Tests** : suite intégration sync complète verte (36s + race 2.8s). Le test `CancelMidRun` ajusté (20 → 100 matchs) pour rester pertinent avec parallelism=24 — sinon le cancel à 75ms arrivait après la fin de l'unique vague.

### 3.7 NEW — Fusionner events_heal + weapon_heal dans un errgroup unique (P2, 30min)

**Source** : Audit Agent 3 (handoff §3 priorité 3).

**Constat** : sur cycle Madina, `healEventsForRecentMatches` (4.2s) puis `healWeaponKillsForRecentMatches` (174s) sont en série. Les deux téléchargent le **même type de ressource** (films Halo). En série, le pool HTTP a des moments creux.

**Cible** : encapsuler les 2 dans un seul `errgroup` qui consomme les match_ids à parser pour chaque type. Saturer le pool HTTP en continu.

**Gain estimé** : 20-30% sur les deux étapes combinées (Madina : 178s → ~125-140s).

**Risque** : ordre d'insertion des heal types peut compter (events avant weapon_kills si un dépend de l'autre). À auditer avant impl.

---

## 4. Phase 3 — Recovery + observabilité (résilience)

Effort : ~6h.

### 4.1 ART rebuild incrémental — ✅ FAIT (boot uniquement) 2026-05-23 (commits d2ca98ce + 20c23eda)

**Livré (step 1/2)** : extraction `migration.RebuildMatchParticipantsART(ctx, db)` runtime, idempotente par design (pas de sentinel), réutilisable. La migration `applyRebuildMatchParticipants` délègue à cette fonction puis pose son sentinel par-dessus. 5 tests TDD verts + 6 tests migration existants verts.

**Livré (step 2/2)** : branchement dans [`cmd/server/main.go`](../apps/go-api/cmd/server/main.go) — après `BootARTGuard` sur shared, si une divergence sur `match_participants` est détectée ET `cfg.SharedProvider` non-nil (mode B-swap), déclenche :
1. `AcquireWriter` via Provider (drain readers, swap RO→RW)
2. `migration.RebuildMatchParticipantsART(ctx, w.DB())` — swap CTAS
3. `Release` (swap RW→RO)
4. Re-probe en RO pour confirmer la disparition de la divergence

Helpers : `hasMatchParticipantsARTDivergence(report)` + `tryAutoHealMatchParticipantsART(ctx, provider, reader)`. Non-bloquant : tout échec logge + métrique mais le serveur continue. Timeout boot étendu 30s → 60s.

**Métriques expvar ajoutées** :
- `art_rebuild_runs_total_attempts`
- `art_rebuild_runs_total_ok`
- `art_rebuild_runs_total_error_acquire`
- `art_rebuild_runs_total_error`
- `art_rebuild_runs_total_still_diverged`
- `art_rebuild_runs_total_skipped_legacy`

**Pattern swap-table** (déjà utilisé dans `migration/steps_shared_social_purge_data_health.go` pour `player_notifications`) :
```sql
BEGIN;
CREATE TABLE match_participants_rebuild AS SELECT * FROM match_participants;
-- index ART rebuild automatique sur la nouvelle table
DROP TABLE match_participants;
ALTER TABLE match_participants_rebuild RENAME TO match_participants;
COMMIT;
```

**Coût** : sur 50k rows, swap < 1s. Lock writer pendant cette fenêtre.

**Trigger** : ✅ automatic au boot. **Détection runtime périodique = sous-phase 4.1.b à venir** (plus complexe car requiert coordination avec scheduler de sync actif). Au boot, no concurrent writers → trivialement safe.

### 4.2 Détection du C++ FATAL DuckDB — ✅ FAIT 2026-05-23 (commit c7d343a6)

**Problème** : `terminate called after throwing duckdb::FatalException` est un crash C++ qui bypass `recover()` Go.

**Livré** : signal handler SIGABRT + SIGSEGV dans `main.go::installFatalSignalHandler` + helper testable `dumpFatalStack(w, sig)`. Le handler dump la stack de toutes les goroutines vers `server.crash.log` puis `os.Exit(2)` — sinon `abort()` libc ré-émet le signal après le retour du handler et le process meurt silencieusement.

**Architecture** :
- `crashFile` hoisté en variable de scope outer pour accessibilité par `debug.SetCrashOutput` (panic Go) ET le signal handler (fatal C++).
- `installFatalSignalHandler(crashFile)` : `signal.Notify(SIGABRT, SIGSEGV)` + goroutine dispatch.
- `dumpFatalStack(w io.Writer, sig os.Signal)` extrait pour testabilité (mock writer + signal).

**Note Windows** : `signal.Notify(SIGABRT)` compile mais ne fire pas (Windows utilise SEH, pas signaux POSIX). Code defensive cross-platform — no-op sur Windows, actif Linux/macOS.

**Tests** : 2 tests unitaires sur `dumpFatalStack` (header timestamp/signal/pid, marqueur fin, présence "goroutine"). `installFatalSignalHandler` non testé directement (os.Exit terminerait le test).

**Limitation connue** : SIGABRT depuis libc terminate() peut tuer le process avant que le handler tourne (race entre signal dispatch et abort re-raise). En pratique sur Linux glibc, le handler a généralement quelques ms pour s'exécuter. Si KO en prod, fallback : wrapper superviseur (overkill, pas urgent).

### 4.3 Métriques concurrence — ✅ FAIT 2026-05-23 (commit 3eceff44, complete les compteurs livres en 4.1/4.4.b)

**Livrable** : expvar publié sur `/debug/vars`.

Compteurs livrés (clés expvar plates, sans labels — convention Go stdlib) :
- `upsert_match_participants_total_ok` / `_error` (writes.go, commit 3eceff44).
- `singleflight_dedupe_total` (writes.go, commit 3eceff44) — incrément quand `singleflight.Do` retourne `shared=true`.
- `art_corruption_detected_<dbLabel>_<table>` (art_probe.go, déjà livré phase 1 BootARTGuard).
- `art_rebuild_runs_total_attempts` / `_ok` / `_error_acquire` / `_error` / `_still_diverged` / `_skipped_legacy` (main.go, phase 4.1).
- `art_autoheal_recompute_started` / `_finished` / `_player_ok` / `_player_error_open` / `_player_error_lease` / `_player_error_shared` / `_player_error` (main.go, phase 4.4.b).
- `highlight_events_parse_total_ok` / `_stale_cache` / `_invalid_data` (engine_highlight_events.go, commit 3eceff44).
- `highlight_events_parse_anomaly_total` (legacy, conservé pour compat dashboard existant).

**Notif data_health_warning** (NON livré, à brancher si besoin sur le canal Discord existant) : si `art_corruption_detected_*` > 0 sur 24h ET pas de `art_rebuild_runs_total_ok` correspondant. Pour l'instant l'utilisateur surveille manuellement /debug/vars au reboot.

### 4.4 NEW — Recompute force=true post-rebuild ART — ✅ FAIT 2026-05-23 (commits b65e0417 step 1/2 + 5d9984fb step 2/2 opt-in)

**Source** : Audit Agent 2 (handoff §2 risque résiduel #6).

**Problème critique non listé dans le plan initial** : pendant la période où l'ART était corrompue, les batchs computed sur les rows partiellement visibles ont produit des résultats FAUX. Exemple documenté dans [`steps_shared_rebuild_match_participants.go:17`](apps/go-api/internal/migration/steps_shared_rebuild_match_participants.go#L17) : **LUSR de Madina figé à Argent IV au lieu de Platine**.

**Conséquence** : le rebuild swap-table de Phase 4.1 récupère les rows en DB, mais les valeurs dérivées (LUSR cascade, performance scores, sessions, citations, dominance flags) restent figées sur l'état corrompu.

**Livré step 1/2** : nouvelle fonction publique [`sync.RecomputeAfterARTRebuild`](../apps/go-api/internal/sync/recompute_after_art_rebuild.go) qui orchestre les 4 cascades en séquence, best-effort par étape :
1. `BatchComputeLUSR(force=true)` — nouveau wrapper public exposé (medal map nil).
2. `BatchComputePerformanceScores(force=true)` — wrapper public existant.
3. `BackfillDominanceFlags` sur la liste complète des match_ids du joueur (chargés via `WHERE xuid || '' = ?` pour court-circuiter l'ART).
4. `RecomputeIsWithFriendsCore` (skip si friend list vide).

API :
```go
func RecomputeAfterARTRebuild(
    ctx context.Context,
    playerDB, sharedDB *sql.DB,
    xuid string,
    friendGamertags []string,
) (RecomputeAfterARTRebuildReport, error)
```

Best-effort : chaque cascade peut échouer sans bloquer les suivantes (erreurs accumulées dans `report.Errors`). Erreur globale uniquement si TOUTES les cascades ont échoué (`allCascadesFailed`).

Logging riche : 1 INFO par étape + 1 INFO summary final avec counts + duration_ms + errors_count.

**Step 2/2 livré (4.4.b) — opt-in auto-recompute post-rebuild boot** (commit `5d9984fb`) : `tryAutoHealMatchParticipantsART` retourne maintenant un `bool` (rebuild OK?) ; si vrai ET env var `LEVELUP_ART_AUTOHEAL_RECOMPUTE=1` posée ET >=1 joueur configuré → spawn `runPostRebuildRecompute` en background. La goroutine itère sur les joueurs, GetOrOpen + AcquirePlayerWriter via pool DuckDB + sharedReader RO, appelle `RecomputeAfterARTRebuild`. Best-effort par joueur. ctx = `context.Background()` pour survivre à un shutdown serveur.

**Métriques expvar** : `art_autoheal_recompute_started`, `_finished`, `_player_ok`, `_player_error_open`, `_player_error_lease`, `_player_error_shared`, `_player_error`.

**Par défaut** (sans `LEVELUP_ART_AUTOHEAL_RECOMPUTE=1`) : aucun changement de comportement, rebuild Phase 4.1 fonctionne tel quel sans toucher aux valeurs dérivées. L'utilisateur peut activer le recompute quand il valide le comportement sur son setup.

**4 tests TDD écrits AVANT impl** ([`recompute_after_art_rebuild_test.go`](../apps/go-api/internal/sync/recompute_after_art_rebuild_test.go)) :
- `ProducesAllCascadeOutputs` : 15 matchs > threshold perf, vérifie LUSR + performance + dominance produits.
- `EmptyData_NoOp` : DB vide, no-op gracieux, counts à 0.
- `Idempotent` : 2 passes successives produisent les mêmes counts.
- `SkipsFriendsWhenEmpty` : friend list nil → FriendXUIDsCount = 0.

Critère TDD : baseline rouge (undefined symbol), verts post-impl. Suite intégration sync complète reste verte (36s).

**Risque** : Long sur grosses player DBs (LUSR cascade O(N), perf cascade O(N×window)). Sur 1000+ matchs ça peut prendre plusieurs minutes. Le caller (CLI tool ou auto-trigger) DOIT lancer en background.

### 4.5 NEW — Paralléliser refreshAggregates + refreshSharedViews — ✅ FAIT 2026-05-23 (commit 5a35a07a)

**Source** : Audit Agent 1 (handoff §1, opportunité O4).

**Livré** : étape 4 du post-sync (`runPostSyncPipeline` dans `engine_postsync.go`) — `refreshAggregates` (player DB) et `refreshSharedViews` (shared DB) tournent maintenant en parallèle via `errgroup.Group`. DBs différentes → pas de conflit. Compteur `r.ViewsRefreshed` via `atomic.Int32`. Best-effort idem ancien comportement (chaque error logguée WARN, ne propage pas).

**Gain estimé** : 500ms-2s par cycle. Marginal mais quasi-gratuit.

---

## 5. Phase 4 — Tests qui bloquent les régressions

Effort : ~8h.

### 5.1 Stress test concurrent `match_participants` — ✅ FAIT 2026-05-23 (commit aef47968)

**Livré** : [`concurrent_upsert_stress_test.go`](../../apps/go-api/internal/sync/concurrent_upsert_stress_test.go) — 3 tests intégration `-race` : SameKey_NoCrash_OneRow, DifferentKeys_AllPresent, BatchPerCall. **TDD strict appliqué** : tests écrits AVANT le singleflight, ont échoué baseline puis passent post-fix.

**Livrable** : `apps/go-api/internal/sync/concurrent_upsert_stress_test.go`.

```go
func TestStressUpsertMatchParticipants_NoCrash(t *testing.T) {
    // Spawn 50 goroutines × 1000 UPSERTs sur la même (match_id, xuid)
    // Assert : zéro panic, table contient exactement 1 row à la fin
    // Run avec -race
}
```

**C'est le test qui ÉCHOUERAIT aujourd'hui**. Une fois Phase 2.3 implémentée, il passe.

### 5.2 Property-based test idempotence — ✅ FAIT 2026-05-23 (commit 968e23d5)

**Livré** : [`concurrent_upsert_property_test.go`](../apps/go-api/internal/sync/concurrent_upsert_property_test.go).

2 tests :
- `TestProperty_ConcurrentUpsertsIdempotent` : K=8 matchs × M=12 xuids × N=20 UPSERTs concurrents (1920 calls). Property : `count(rows) == K*M`, `count(match_id, xuid) == 1` partout, aucune paire absente. Valeurs randomisées avec seed=42.
- `TestProperty_SamePairManyConcurrent_OneRow` : 200 UPSERTs concurrents sur la MÊME clé → exactement 1 row finale (stress singleflight).

Tests verts sous `-race` (1.5s).

### 5.3 E2E concurrent multi-player sync — ✅ FAIT 2026-05-23 (commit 639cd62f)

**Livré** : [`concurrent_multiplayer_e2e_test.go`](../apps/go-api/internal/sync/concurrent_multiplayer_e2e_test.go).

Scénario réel 3 joueurs (Alice, Bob, Carol) × 5 matchs partagés + 5 solos chacun = 30 paires uniques `(match_id, xuid)`. 3 goroutines parallèles, 100 cycles consécutifs. 4 invariants vérifiés par cycle :
1. `count(match_participants) == 30`
2. Chaque match partagé a exactement 3 xuids distincts
3. Chaque match solo a exactement 1 row
4. Aucun doublon `(match_id, xuid)` via `HAVING COUNT(*) > 1`

Tests verts (~7.8s normal, ~9.3s sous `-race`). Couvre le pattern Halo matchmaking réel que Phase 5.2 randomisée ne ciblait pas.

### 5.4 Benchmark perf cycle complet — ✅ FAIT 2026-05-23 (commit 67fe6031)

**Livré** : [`bench_perf_test.go`](../apps/go-api/internal/sync/bench_perf_test.go) — 2 micro-benchmarks ciblés sur les hot paths effectivement parallélisés (alternative au `BenchmarkAutoSync_3Players_20Matches` complet, trop lourd à setup avec pool+scheduler mocks).

| Benchmark | Wall-time mesuré | Baseline théorique séquentielle | Speedup |
|---|---|---|---|
| `BenchmarkProcessWeaponKillsInline_16Matches` (16 × 100ms) | ~102ms/op | 1.6s | **15.7×** |
| `BenchmarkGetMatchFilm_20Chunks` (20 × 50ms) | ~156ms/op | 1s | **6.4×** |

### 5.5 ART corruption detection + rebuild regression test — ✅ FAIT 2026-05-23 (commit 88a19b65)

**Livré** : [`art_rebuild_regression_test.go`](../apps/go-api/internal/sync/art_rebuild_regression_test.go) — 2 tests E2E chaînage `probe → rebuild → re-probe` :
- `TestART_RebuildRegression_ProbeCleanBeforeAndAfter` (10 matchs × 8 participants) : probe pré-rebuild clean, rebuild swap CTAS, row count préservé, PK active, v_gamertag_lookup recréée, re-probe clean.
- `TestART_RebuildRegression_PreservesRowsPerMatch` (20 × 6) : invariant fine-grain par match (détecte les pertes silencieuses où le total est OK mais la distribution faussée).

Bloquent toute régression future du flow auto-heal.

---

## 6. Hors scope explicite

Ces points ne sont PAS dans ce plan. Notés ailleurs si applicable.

| Sujet | Statut | Où ? |
|---|---|---|
| `roster` / `nemesis_duels` dead code | **À noter dans BACKLOG.md** | Décision produit (implémenter vs retirer du schéma OpenAPI). Action explicite à ajouter. |
| `weapon_kills` empty pour match `de3cec8b` | À diag séparé | Probablement xuid sans kills dans ce match. Pas lié à la concurrence. |
| Fix LUSR full-scan load | Hors scope | Push filter en SQL. Optimisation perf compute (incremental load). À considérer après stabilisation. |
| MV atomic rename pattern | Hors scope | `recreateMaterializedView` non-atomique. Indépendant de l'ART. |
| Backfill one-shot dominance pour les 83 matchs hier | Hors scope | CLI à lancer manuellement après stabilisation. |
| Optimisation parse `scanEvents` bit-level | Hors scope | Profile + optim CPU si benchmark Phase 5.4 montre que c'est le bottleneck. |

---

## 7. Récap chiffré (RECALIBRÉ post-audit)

| Phase | Effort | Bénéfice principal |
|---|---|---|
| **0 NEW — Bump driver v1.5.3** | 1h | Pari low-cost. Peut résoudre tout seul si le bug ART est fixé upstream. |
| 2 — Safety concurrent (singleflight + ADR) | 4h | Plus de race ART au niveau applicatif. SAFETY, pas du perf. |
| 3 — Parallélisation network | **9h** (+4h) | Sync 14 min → ~3-5 min. **§3.0 NEW = gain le plus impactant (~150-200s/cycle)**. §3.2/§3.3 dépriorisés. §3.6/§3.7 NEW. |
| 4 — Recovery + observabilité | **9h** (+3h) | Auto-heal corruptions ART, recompute post-rebuild (LUSR/perf force=true), capture crashes C++. |
| 5 — Tests régression | 8h | CI bloque les régressions futures (stress + property + E2E + bench). |
| **Total** | **~24h** (+3h vs initial) | Plan pérenne, validé par audit. |

---

## 8. Ordre d'exécution recommandé (RECALIBRÉ post-audit)

**P0 — Stop le crash + gain perf le plus impactant** :
1. **Phase 0** (bump driver v1.5.3) — 1h, pari low-cost. Si la corruption ART disparaît → on saute la moitié du reste.
2. **Phase 2.1 + 2.2** (cartographie + ADR 0018) — avant toute écriture de code, on aligne sur le contract.
3. **Phase 5.1** (stress test concurrent UPSERT) — écrit AVANT la Phase 2.3 pour validation TDD. Le test échoue d'abord.
4. **Phase 2.3** (singleflight) — fix le crash, pass le test 5.1.
5. **Phase 3.0** (paralléliser `processWeaponKillsInline`) — **TDD obligatoire** (tests d'abord, output attendu défini avant code). Gain ~150-200s/cycle Madina.

**P1 — Recovery + parallélisation scheduler** :
6. **Phase 4.1** (ART rebuild runtime) — recovery automatique pour les ~32 matchs déjà corrompus.
7. **Phase 4.4** (recompute force=true post-rebuild) — recalcule LUSR/perf/sessions/dominance pour les joueurs affectés (sinon Madina reste figée en Argent IV).
8. **Phase 3.4** (parallel scheduler RunOnce) — après singleflight validé. Gain 15min → 5-8min.
9. **Phase 3.5** (heal loops avec safety singleflight).

**P2 — Observabilité + optims marginales** :
10. **Phase 4.2 + 4.3** (signal handler SIGABRT + métriques expvar).
11. **Phase 3.6 + 3.7** (bump healParallelism + fusion heal loops).

**P3 — Tests régression + optims bonus** :
12. **Phase 5.2 → 5.5** (property test + E2E concurrent + bench + ART rebuild test).
13. **Phase 3.1** (vérif parallel download, juste confirmation).
14. **Phase 4.5** (parallel refreshAggregates + refreshSharedViews) — gain 500ms-2s.
15. **Phase 3.2** (move parse) — DÉPRIORISÉ, à faire si du temps reste.
16. **Phase 3.3** (intra-match parallel API) — DÉPRIORISÉ FORTEMENT, ROI marginal.

---

## 9. Critères de succès

| Critère | Mesure |
|---|---|
| Plus de `duckdb::FatalException` | 7 jours de sync continue sans crash |
| Sync 3 joueurs × 20 matchs | < 5 min wall-time |
| Couverture tests concurrence | `go test -race -tags=integration ./internal/sync/...` passe |
| Détection + recovery ART | Au moins 1 test E2E qui force une corruption et la résout |
| ART corruption en prod | 0 matchs corrompus détectés sur 7 jours après rebuild initial |
| Parse highlight_events parallèle | Phase 3 d'`insertFetchedMatch` ne fait plus de zlib decompress (juste write) |

---

## 10. Risques identifiés

| Risque | Probabilité | Mitigation |
|---|---|---|
| Bump driver v1.5.3 introduit nouveau bug | Faible | Revert v1.4.3 LTS si KO. Tests verts requis avant déploiement. |
| Bump driver ne corrige pas l'ART | Moyenne | Plan B = singleflight (Phase 2). Pas un risque, c'est le scenario nominal. |
| `singleflight` ne résout pas TOUTES les races (ex: 2 process distincts) | Faible | DuckDB single-writer process via B-swap (ADR 0016) |
| `SIGABRT` arrive trop tard pour être catché en Go | Moyenne | Tester sur Windows + Linux. Si KO, fallback wrapper superviseur |
| Rebuild ART échoue sur grosse table | Faible | Tester sur shared 50k+ rows. Si KO, batch par chunk de match_ids |
| Tests stress trop lents pour CI | Faible | Tag `slow` ou run hebdo |
| Phase 3.4 (parallel scheduler) introduit deadlocks dblease | Moyenne | Phase 5.3 (E2E concurrent multi-player) doit le catcher |
| **Phase 3.0 (parallel processWeaponKillsInline) introduit régression silencieuse** | Moyenne | **TDD obligatoire — tests écrits AVANT code. Contrat de sortie défini avant impl.** Stress test + idempotence + cancel + race. |
| Recompute force=true post-rebuild très long (Phase 4.4) | Moyenne | Lancer en background, ne pas bloquer le boot. Métrique exposée. |
| Move parse vers Phase 2 alourdit les goroutines | Faible | Le parse reste rapide (ms) vs network (~500ms). Pas de pression mémoire significative. (DÉPRIORISÉ de toute façon) |

---

## 11. État de la décision

**À ce stade** : plan en attente de validation utilisateur. Pas une ligne de code écrite sur ce plan.

**Q ouvertes pour l'utilisateur** :
1. L'ordre des phases convient-il ?
2. Budget temps à respecter (split sur plusieurs jours / sprints) ?
3. Branche cible : nouvelle ou continuer sur `fix/media-paths-portable` ?
4. Pour `roster` / `nemesis_duels` : entry à ajouter dans `BACKLOG.md` (priorité ? deadline ?).

---

## 12. Handoff — résultats d'analyse préliminaire des 4 agents

Avant le démarrage de l'implémentation, 4 agents d'analyse ont été lancés en parallèle pour **valider/réfuter empiriquement les postulats de ce plan**. Leurs livrables sont consolidés dans le document compagnon :

→ [`.ai/HANDOFF_SYNC_CONCURRENCY_AUDIT.md`](HANDOFF_SYNC_CONCURRENCY_AUDIT.md)

Ce handoff couvre :

1. **Audit parallélisation actuelle** : validation/refus des 6 postulats du plan sur l'état réel du sync engine (download chunks parallèles, parse séquentiel, intra-match séquentiel, scheduler séquentiel, heal loops parallèles depuis Action B, rate limiter).
2. **Deep dive ART corruption DuckDB** : validation que singleflight est la bonne stratégie ou alternatives, statut upstream du bug DuckDB, recommandation finale de recovery.
3. **Performance breakdown réel** : timeline parsée depuis les logs du 22 mai (Madina 8min, Choco 4min) avec décomposition network/compute/DB write — valide ou refute le postulat "writes DB pas le bottleneck".
4. **Highlight events pipeline** : faisabilité du move parse → Phase 2 parallèle, thread-safety du parser, gain estimé.

**Ce handoff DOIT être lu avant de démarrer l'implémentation.** Il peut amender certaines phases (priorisation, scope, blockers découverts).

---

## 13. Vérification UI en direct via Chrome DevTools MCP

**État** : **MCP Chrome DevTools DISPONIBLE depuis le 2026-05-22 (post-restart utilisateur)**. Tools disponibles : `mcp__chrome-devtools__*` (navigate_page, take_snapshot, list_network_requests, evaluate_script, performance_start_trace, take_screenshot, lighthouse_audit, etc.).

**Pourquoi en avoir besoin pour ce plan** : pouvoir charger la page match-view dans Chrome, inspecter le réseau (XHR / fetch), voir quelles requêtes API échouent ou retournent du vide, et corréler avec les fixes en cours. Sans ça, on est aveugle sur le rendu front-end et on doit demander à l'utilisateur de faire l'inspection manuellement à chaque tour.

**Étapes du plan où le MCP sera utile** :
- Phase 3.4 (parallel scheduler) : vérifier que la page match-view reflète bien le sync des 3 joueurs concurrents sans glitch UI.
- Phase 4.1 (ART rebuild) : confirmer en direct que le scoreboard repopulate après un rebuild.
- Phase 5.3 (E2E concurrent test) : compléter le test Go par une vérification UI réelle.
- Tout fix de bug front signalé par l'utilisateur (ex: sections vides match-view, graphiques absents).

---

## Annexe A — Lectures recommandées avant de démarrer

- ADR 0016 — SharedDBProvider B-swap (RO↔RW)
- `apps/go-api/internal/platform/duckdb/art_probe.go` — BootARTGuard existant
- `apps/go-api/internal/migration/steps_shared_social_purge_data_health.go` — pattern swap-table déjà utilisé pour `player_notifications`
- DuckDB issue tracker : "ART", "INSERT ON CONFLICT", "concurrent UPSERT"
- `.ai/thought_log.md` entrées 20/21/22 mai pour le contexte des fixes précédents

# Plan — Logging contextuel & couverture tests : `apps/go-api`

> **Branche** : `copilot/add-api-watcher-for-match-sync`  
> **Date** : 2026-04-23  
> **Statut** : À implémenter (reverts en place, aucune modif en attente)

---

## Contexte

La branche `copilot/add-api-watcher-for-match-sync` a introduit le watcher de présence
(`internal/watcher/`) et le moteur de sync Go (`internal/sync/`). Deux reverts successifs ont
été effectués sur des corrections partielles (`lease context-aware` et
`matchExistsInRegistry`). Ce plan regroupe **toutes** les améliorations ciblées à implémenter
en une seule passe cohérente.

---

## État des lieux

| Zone | Fichiers prod | Logging | Tests |
|------|--------------|---------|-------|
| `internal/sync/engine.go` | 1 | ✅ Complet (49 logs slog) | ✅ — **manque fast-path `matchExistsInRegistry`** |
| `internal/sync/coordinator.go` | 1 | ⚠️ 4 logs sans `slog.*Context` | ✅ Bien couvert |
| `internal/sync/lease.go` | 1 | ✅ | ✅ — **manque `AcquireLeaseCtx`** |
| `internal/watcher/` | 5 | ✅ Complet | ✅ — quelques manques ciblés |
| `internal/api/handlers/sync_handler.go` | 1 | ❌ 0 log | ⚠️ 6 tests — **manque 409, delta dispatch** |
| `internal/api/handlers/backfill.go` | 1 | ❌ 0 log | ✅ scope — **manque HTTP 400/404/409** |
| `internal/domain/sync.go` | 1 | n/a | ✅ — **manque champ `MatchesReused`** |

---

## Changements prévus

### Étape 1 — `internal/domain/sync.go` : champ `MatchesReused`

**Pourquoi** : le fast-path (match déjà en `match_registry`, pas d'appel API) doit être
tracé séparément des insertions classiques (`MatchesInserted`) pour donner une visibilité
sur la déduplication inter-joueurs.

**Ce qui change** :
- Ajouter `MatchesReused int` dans `SyncResult`
- Mettre à jour le log final dans `engine.go` : `"reused", result.MatchesReused`

**Test** : `TestSyncResult_MatchesReused_Field` dans `domain/domain_test.go`

---

### Étape 2 — `internal/sync/lease.go` : `AcquireLeaseCtx`

**Pourquoi** : `AcquireLease(path, timeout)` utilise un timeout fixe (10 s). Dans les
handlers HTTP, si le client annule sa requête, le polling doit s'arrêter immédiatement
pour éviter de tenir le CPU inutilement.

**Ce qui change** :
```go
// Nouvelle fonction — boucle TryLock + select ctx.Done()
func AcquireLeaseCtx(ctx context.Context, path string) (func(), error)
```

**Tests** (`lease_test.go`, tag `integration`) :

| Nom | Scénario |
|-----|---------|
| `TestAcquireLeaseCtx_Basic` | acquire + release normaux |
| `TestAcquireLeaseCtx_CancelledCtx` | ctx déjà annulé → erreur immédiate, 0 goroutine créée |
| `TestAcquireLeaseCtx_CancelDuringWait` | annulation pendant le polling → retour < 500 ms |
| `TestAcquireLeaseCtx_SequentialAfterRelease` | deux acquisitions successives réussissent |

---

### Étape 3 — `internal/sync/engine.go` : fast-path + leases ctx

**3a. Helper `matchExistsInRegistry`**

Avant tout appel API dans `processMatch`, vérifier si le match existe déjà dans
`match_registry` (shared DB). Si oui : écrire uniquement `player_match_enrichment`,
incrémenter `MatchesInserted` **et** `MatchesReused`, retourner sans appel réseau.

```go
func matchExistsInRegistry(db *sql.DB, matchID string) (bool, error)
```

**3b. Leases contextuelles dans `run()`**

Remplacer `AcquireLease(e.playerDBPath, leaseTimeout)` et
`AcquireLease(e.sharedDBPath, leaseTimeout)` par `AcquireLeaseCtx(ctx, path)`.
Supprimer la constante `leaseTimeout` devenue inutile.

**3c. Log final enrichi**

```go
slog.InfoContext(ctx, "sync: terminé", ..., "reused", result.MatchesReused, ...)
```

**Tests** :

| Fichier | Nom | Scénario |
|---------|-----|---------|
| `engine_test.go` (tag `integration`) | `TestMatchExistsInRegistry_NotFound` | match absent → false, nil |
| `engine_test.go` (tag `integration`) | `TestMatchExistsInRegistry_Found` | match présent → true, nil |
| `engine_e2e_test.go` | `TestProcessMatch_FastPath` | pré-insère en registry, vérifie 0 appel API, `MatchesReused == 1`, row dans `player_match_enrichment`, pas de doublon en `match_registry` |

---

### Étape 4 — `internal/sync/coordinator.go` : logging contextualisé

**Pourquoi** : les 4 appels `slog.Info` / `slog.Error` n'utilisent pas
`slog.InfoContext` / `slog.ErrorContext` alors que `ctx` est disponible. Les
champs `in_flight` et `parallel_cap` manquent pour le diagnostic opérationnel.

**Ce qui change** (refactoring pur — comportement identique) :

```go
// Avant
slog.Info("coordinator: sync déjà en cours, requête ignorée", "gamertag", req.Gamertag)

// Après
slog.InfoContext(ctx, "coordinator: sync déjà en cours — requête ignorée",
    "gamertag", req.Gamertag, "in_flight", c.InFlightCount())
```

Même traitement pour `démarrage sync`, `sync échoué`, `sync terminé`.

**Aucun nouveau test** : les `TestCoordinator_*` existants continuent de valider le
comportement.

---

### Étape 5 — `internal/api/handlers/sync_handler.go` : logging + 2 tests

**5a. Logging**

Ajouter `slog.InfoContext` / `slog.WarnContext` sur :
- Requête reçue (player_slug, mode, max_matches)
- Chaque décision de validation (400 / 403 / 404 / 409)
- Job créé (job_id)
- Erreur interne 500

**5b. Tests** (`sync_handler_test.go`) :

| Nom | HTTP | Scénario |
|-----|------|---------|
| `TestSyncHandler_InitialSync_409_DuplicateJob` | 409 | Créer un job actif, relancer → conflict |
| `TestSyncHandler_DeltaSync_404_PlayerNotFound` | 404 | `player_slug` inconnu dans `db_profiles.json` |

---

### Étape 6 — `internal/api/handlers/backfill.go` : logging + 3 tests

**6a. Logging**

Ajouter logs sur :
- Début du handler (player_slug, scope flags activés)
- Job créé (job_id)
- Dry-run court-circuit (total détecté)
- Phase weapons sans tokens (warn)
- Fin du job (total, inserted)

**6b. Tests** (`backfill_test.go`) :

| Nom | HTTP | Scénario |
|-----|------|---------|
| `TestStartBackfill_400_EmptySlug` | 400 | `player_slug` vide dans le body |
| `TestStartBackfill_404_PlayerNotFound` | 404 | Joueur absent de `db_profiles.json` |
| `TestStartBackfill_409_AlreadyActive` | 409 | Job backfill déjà actif pour ce joueur |

---

## Récapitulatif fichiers touchés

| Fichier | Type | Delta |
|---------|------|-------|
| `internal/domain/sync.go` | prod | +1 champ |
| `internal/domain/domain_test.go` | test | +1 test |
| `internal/sync/lease.go` | prod | +1 fonction |
| `internal/sync/lease_test.go` | test | +4 tests |
| `internal/sync/engine.go` | prod | +1 helper, fast-path, leases ctx, log enrichi, -1 const |
| `internal/sync/engine_test.go` | test | +2 tests |
| `internal/sync/engine_e2e_test.go` | test | +1 test |
| `internal/sync/coordinator.go` | prod | logging ctx (refactoring mineur) |
| `internal/api/handlers/sync_handler.go` | prod | +logging |
| `internal/api/handlers/sync_handler_test.go` | test | +2 tests |
| `internal/api/handlers/backfill.go` | prod | +logging |
| `internal/api/handlers/backfill_test.go` | test | +3 tests |

**Total : ~12 nouveaux tests, 0 test supprimé.**

---

## Ce qui n'est pas inclus (hors périmètre)

- `internal/service/*` : couche algorithmique pure — pas de log (conforme à la règle `analysis/` vs `services/`)
- `internal/assets/*` : package stable avec sa propre politique de logging
- `internal/platform/*` : logging géré par couche auth/duckdb séparément
- `internal/watcher/state_machine.go` : `transition()` appelle `slog.Info` sans ctx car le lock est tenu et ctx n'est pas disponible — comportement acceptable documenté

---

---

## Étape 7 — Couverture logging & non-régression DuckDB concurrence

Cette étape est distincte des étapes 1 à 6 : elle ne couvre pas des fonctionnalités
métier mais les **invariants de la couche d'accès DuckDB** introduits par
`fix-duckdb-conflicts-plan.md`. Elle doit être implémentée **après** les étapes 1 à 6
et **après** que le plan de concurrence est appliqué.

### 7.1 — Logging : surface DuckDB

Les éléments de log suivants n'existent pas aujourd'hui et doivent être ajoutés dans les
fichiers correspondants lors de l'implémentation du plan de concurrence :

| Surface | Fichier | Log à ajouter |
|---|---|---|
| `AcquireLeaseCtx` — début attente | `dblease/lease.go` | `slog.DebugContext(ctx, "dblease: attente du lease", "db", path)` |
| `AcquireLeaseCtx` — ctx annulé | `dblease/lease.go` | `slog.WarnContext(ctx, "dblease: contexte annulé pendant attente", "db", path, "err", ctx.Err())` |
| `AcquireLeaseCtx` — lease acquis | `dblease/lease.go` | `slog.DebugContext(ctx, "dblease: lease acquis", "db", path, "wait_ms", elapsed.Milliseconds())` |
| Consumer `PersistSink` — queue pleine | `persist_sink.go` | `slog.Warn("persist_sink: queue pleine, job abandonné", "xuid", s.XUID, "kind", kind)` |
| Consumer `PersistSink` — lease player timeout | `persist_sink.go` | `slog.Warn("persist_sink: lease player timeout", "xuid", s.XUID, "path", s.PlayerPath)` |
| Consumer `PersistSink` — lease metadata timeout | `persist_sink.go` | `slog.Warn("persist_sink: lease metadata timeout", "xuid", s.XUID, "path", s.MetaPath)` |
| Consumer `PersistSink` — job traité | `persist_sink.go` | `slog.Debug("persist_sink: job traité", "xuid", s.XUID, "kind", job.kind)` |
| `openPlayerDB` — PlayerRO ouvert | `pool.go` | `slog.Debug("pool: player db RO ouvert", "gamertag", cfg.Gamertag)` |
| `CloseAll` — drain sink | `pool.go` | `slog.Debug("pool: drain sink avant fermeture", "gamertag", pdb.Gamertag)` |
| `matchExistsInRegistry` — fast path pris | `engine.go` | `slog.DebugContext(ctx, "engine: fast path — match déjà en shared", "match_id", matchID, "gamertag", e.gamertag)` |
| `RunBackfill` — lease attendu | `engine.go` | `slog.InfoContext(ctx, "engine: backfill — attente du lease", "gamertag", e.gamertag)` |

Tous les logs de niveau `Warn` et `Error` dans la couche DuckDB doivent inclure au minimum
`"db", path` et, si disponible, `"gamertag"` ou `"xuid"`.

### 7.2 — Tests unitaires : lease et cache DuckDB

#### Fichier : `internal/platform/dblease/lease_test.go` (nouveau)

| Nom | Scénario | Résultat attendu |
|---|---|---|
| `TestAcquireLease_Basic` | acquire + release normaux | lock acquis, release libère sans erreur |
| `TestAcquireLease_Timeout` | path déjà locké, timeout court | erreur de timeout en < (timeout + 50ms) |
| `TestAcquireLease_DifferentPaths` | deux chemins distincts | pas de blocage mutuel |
| `TestAcquireLeaseCtx_AlreadyCancelled` | ctx annulé avant l'appel | erreur immédiate, aucune goroutine créée |
| `TestAcquireLeaseCtx_CancelDuringWait` | annulation pendant polling | retour < 500 ms, erreur ctx.Canceled |
| `TestAcquireLeaseCtx_AcquiredAfterRelease` | lock relâché pendant l'attente ctx | acquisition réussie |
| `TestAcquireLease_SingletonMutexMap` | deux appels `AcquireLease` sur le même path dans deux goroutines | le second attend le premier, ordre FIFO approximatif |
| `TestAcquireLease_NoLeakOnTimeout` | goroutine de poll doit s'arrêter après timeout | `goleak.VerifyNone(t)` passe |

#### Fichier : `internal/platform/duckdb/pool_test.go` (nouveau ou existant)

| Nom | Scénario | Résultat attendu |
|---|---|---|
| `TestPlayerDB_HasPlayerRO` | `GetOrOpen` sur un joueur | `pdb.PlayerRO != nil` et `pdb.PlayerRO.Path() == pdb.Player.Path()` |
| `TestPlayerRO_IsDifferentHandle` | comparer les handles | `pdb.PlayerRO.SQLDb() != pdb.Player.SQLDb()` (clés `ro:` vs `rw:`) |
| `TestCloseAll_DrainsSinkFirst` | mock sink, appel `CloseAll` | `sink.Close()` appelé avant `pdb.Player.Close()` |

### 7.3 — Tests de non-régression : coexistence RO/RW (WAL DuckDB)

Ces tests vérifient que le modèle WAL DuckDB fonctionne comme attendu dans le processus.
Ils constituent la garantie que `PlayerRO` ne bloque pas et ne se fait pas bloquer.

#### Fichier : `internal/platform/duckdb/wal_concurrency_test.go` (nouveau, tag `integration`)

| Nom | Scénario | Résultat attendu |
|---|---|---|
| `TestWAL_ReadWhileWriteInProgress` | ouvrir un handle RW, lancer un write long (100 rows en boucle), ouvrir un handle RO en parallèle et faire un SELECT | le SELECT RO retourne sans erreur dans < 200 ms, il voit les rows déjà committées avant le write en cours |
| `TestWAL_MultipleROConcurrent` | 10 goroutines lisent sur `ro:path` simultanément pendant qu'une goroutine écrit | aucune erreur, chaque lecteur termine < 500 ms |
| `TestWAL_RWAfterROClose` | ouvrir RO, fermer RO, ouvrir RW | aucune erreur, refcount cache = 0 pour `ro:path` après fermeture |
| `TestWAL_RODoesNotSeeUncommittedWrite` | écrire sans COMMIT, lire via RO | SELECT RO ne voit pas la row non committée |

### 7.4 — Tests de non-régression : PersistSink

#### Fichier : `internal/platform/duckdb/persist_sink_test.go` (nouveau ou existant, tag `unit`)

| Nom | Scénario | Résultat attendu |
|---|---|---|
| `TestPersistSink_EnqueueSingleJob` | créer un sink, envoyer 1 job BattlePass | job traité par le consumer, log debug émis, pas d'erreur |
| `TestPersistSink_QueueFull_Drop` | saturer la queue (16 jobs), envoyer un 17e | 17e job abandonné, `slog.Warn` émis, pas de panic |
| `TestPersistSink_Close_DrainsQueue` | envoyer 5 jobs puis appeler `Close()` | tous les jobs sont traités avant que `Close()` retourne |
| `TestPersistSink_Close_Idempotent` | appeler `Close()` deux fois | pas de panic, pas de double-fermeture du channel |
| `TestPersistSink_SingleConsumer` | envoyer 20 jobs en rafale | vérifier via compteur atomique que le consumer traite 1 job à la fois (pas d'exécution parallèle) |
| `TestPersistSink_NoLeakAfterClose` | `Close()` + `goleak.VerifyNone(t)` | aucune goroutine orpheline |

### 7.5 — Tests de non-régression : sync concurrent multi-joueurs

Ces tests couvrent le scénario central du plan : deux joueurs avec des matchs communs, un
sync manuel et un auto-sync en parallèle.

#### Fichier : `internal/sync/engine_e2e_test.go` (extension, tag `integration`)

| Nom | Scénario | Résultat attendu |
|---|---|---|
| `TestConcurrentSync_TwoPlayers_SharedMatch` | joueur A et joueur B ont un match commun ; syncs lancés en parallèle | `match_registry` contient 1 seule row pour ce match ; `player_match_enrichment` en contient 2 (une par joueur) ; pas d'erreur de lock DuckDB |
| `TestConcurrentSync_LeaseBlocksBackfill` | `run()` démarre, `RunBackfill()` lancé en parallèle sur le même joueur | `RunBackfill()` attend la libération du lease sans erreur ; les deux opérations terminent avec des données cohérentes |
| `TestConcurrentSync_ROReadDuringWrite` | `run()` écrit dans `stats.duckdb` (handle RW) ; requête Home lisant via `PlayerRO` lancée en parallèle | la requête Home retourne en < 500 ms, sans attendre la fin du sync |
| `TestConcurrentSync_PersistSinkDuringRun` | `run()` tient le lease player ; un `PersistBattlePass()` est appelé | le job est mis en queue ; le consumer attend le lease ; pas de corruption ; le consumer écrit après libération par `run()` |
| `TestAutoSync_DoesNotConflictWithManualSync` | auto-sync planifié déclenché pendant un sync manuel actif | le `Coordinator` bloque le deuxième sync (`409` ou skip interne) ; pas de double-écriture |

### 7.6 — Tests de non-régression : shutdown propre

#### Fichier : `internal/platform/duckdb/shutdown_test.go` (nouveau, tag `integration`)

| Nom | Scénario | Résultat attendu |
|---|---|---|
| `TestShutdown_DrainsSinksBeforeClose` | 3 joueurs avec un PersistSink chargé ; appel `CloseAll()` | tous les sinks drainent avant fermeture des handles DB ; pas d'erreur "write on closed DB" |
| `TestShutdown_CloseAllReleasesRefCount` | ouvrir player RW + player RO ; appel `CloseAll()` | les deux clés `rw:` et `ro:` sont supprimées de `openDBs`; `openDBs` est vide |
| `TestShutdown_NoLeakAfterCloseAll` | `CloseAll()` + `goleak.VerifyNone(t)` | aucune goroutine PersistSink en cours |

### 7.7 — Récapitulatif fichiers touchés par l'étape 7

| Fichier | Type | Delta |
|---|---|---|
| `internal/platform/dblease/lease_test.go` *(nouveau)* | test | +8 tests |
| `internal/platform/duckdb/pool_test.go` *(nouveau ou existant)* | test | +3 tests |
| `internal/platform/duckdb/wal_concurrency_test.go` *(nouveau)* | test intégration | +4 tests |
| `internal/platform/duckdb/persist_sink_test.go` *(nouveau ou existant)* | test | +6 tests |
| `internal/sync/engine_e2e_test.go` | test intégration | +5 tests |
| `internal/platform/duckdb/shutdown_test.go` *(nouveau)* | test intégration | +3 tests |
| `internal/platform/dblease/lease.go` | prod | logs debug/warn (§7.1) |
| `internal/platform/duckdb/persist_sink.go` | prod | logs warn/debug (§7.1) |
| `internal/platform/duckdb/pool.go` | prod | logs debug (§7.1) |
| `internal/sync/engine.go` | prod | logs debug/info (§7.1) |

**Total étape 7 : ~29 nouveaux tests (dont 12 intégration), 0 test supprimé.**

### 7.8 — Dépendances externes requises

- `go.uber.org/goleak` : détection de goroutines orphelines dans les tests de non-régression
  (ajout dans `go.mod` si absent)
- Tags de build retenus : `unit` (sans fichier DuckDB physique, mocks), `integration`
  (fichier DuckDB temporaire via `t.TempDir()`)

---

## Récapitulatif global fichiers touchés

| Fichier | Type | Delta |
|---------|------|-------|
| `internal/domain/sync.go` | prod | +1 champ |
| `internal/domain/domain_test.go` | test | +1 test |
| `internal/sync/lease.go` | prod | +1 fonction |
| `internal/sync/lease_test.go` | test | +4 tests |
| `internal/sync/engine.go` | prod | +1 helper, fast-path, leases ctx, log enrichi, -1 const |
| `internal/sync/engine_test.go` | test | +2 tests |
| `internal/sync/engine_e2e_test.go` | test | +6 tests (1 existant + 5 nouveaux §7.5) |
| `internal/sync/coordinator.go` | prod | logging ctx (refactoring mineur) |
| `internal/api/handlers/sync_handler.go` | prod | +logging |
| `internal/api/handlers/sync_handler_test.go` | test | +2 tests |
| `internal/api/handlers/backfill.go` | prod | +logging |
| `internal/api/handlers/backfill_test.go` | test | +3 tests |
| `internal/platform/dblease/lease.go` | prod | +logs §7.1 |
| `internal/platform/dblease/lease_test.go` *(nouveau)* | test | +8 tests |
| `internal/platform/duckdb/pool.go` | prod | +logs §7.1 |
| `internal/platform/duckdb/pool_test.go` *(nouveau)* | test | +3 tests |
| `internal/platform/duckdb/persist_sink.go` | prod | +logs §7.1 |
| `internal/platform/duckdb/persist_sink_test.go` *(nouveau)* | test | +6 tests |
| `internal/platform/duckdb/wal_concurrency_test.go` *(nouveau)* | test intégration | +4 tests |
| `internal/platform/duckdb/shutdown_test.go` *(nouveau)* | test intégration | +3 tests |

**Total global : ~41 nouveaux tests, 0 test supprimé.**

---

## Ordre d'implémentation recommandé

1. `domain/sync.go` (champ `MatchesReused`) — prérequis pour les étapes suivantes
2. `sync/lease.go` (`AcquireLeaseCtx`) — prérequis pour l'étape 3b
3. `sync/engine.go` (fast-path + leases ctx)
4. `sync/coordinator.go` (logging ctx)
5. `handlers/sync_handler.go` + tests
6. `handlers/backfill.go` + tests
7. **Après** implémentation de `fix-duckdb-conflicts-plan.md` :
   - `dblease/lease_test.go` (§7.2)
   - `duckdb/wal_concurrency_test.go` (§7.3) — valider le WAL en isolation
   - `duckdb/persist_sink_test.go` (§7.4) — valider drain + queue
   - `duckdb/pool_test.go` (§7.2)
   - `duckdb/shutdown_test.go` (§7.6)
   - `sync/engine_e2e_test.go` extension (§7.5) — tests multi-joueurs en dernier, car ils dépendent de tout ce qui précède

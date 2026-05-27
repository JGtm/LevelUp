# Plan — Fix Sync Reliability (2026-05-24)

**Branche cible** : `fix/sync-reliability-2026-05-24` (depuis `refactor/collect-persist`)
**Effort total estimé** : 4h30 fixes P0+P1, +2h optionnel P2, +8h tests (non négociable).
**Stratégie de tests** : voir [PLAN_FIX_SYNC_TESTS_STRATEGY_2026-05-24.md](PLAN_FIX_SYNC_TESTS_STRATEGY_2026-05-24.md) — pipeline complète couverte avec les 942 matchs réels (film_chunks + manifests) + 69 batches WAL locaux.
**Auteur** : audit logs auto-sync cycle 20:33-20:38 (4 joueurs, observation des trois bugs simultanés).

---

## Objectif

Restaurer la fiabilité de l'auto-sync sur la branche `refactor/collect-persist` avant le merge vers `main`. Trois bugs observés en cycle prod simultanément, chacun avec un effet d'amplification sur les deux autres. Le critère de succès est un cycle complet (4 joueurs) **sans** :

- `Connection Error: different configuration` sur metadata ou player DB
- `json: unsupported value: NaN` sur le batch marshal
- Inflation factice de `MatchesInserted` pour les joueurs inactifs
- Post-sync qui tourne sur des matchs déjà persistés
- Durée > 60s pour un joueur sans matchs nouveaux

---

## Critères go/no-go (delivery-checklist)

Avant merge vers `main` :

1. 3 cycles auto-sync consécutifs propres (4 joueurs, 0 erreur Persist, 0 NaN, 0 metadata conflict)
2. Sync d'un joueur inactif (`Madina97294` après son delta vide) < 5s end-to-end
3. `player_match_enrichment` à jour pour tous les matchs ingérés du cycle
4. `match_citations` non vide après post-sync (citations_computed > 0)
5. ART probe shared+metadata clean au boot ET en fin de cycle
6. Tests Go full pass (`go test ./...`)
7. Entrée `.ai/thought_log.md` avec décision technique + résultats observés

---

## Contexte — les bugs corrélés

| # | Symptôme | Site | Effet |
|---|---|---|---|
| 1 | `Can't open a connection with a different configuration` | [combined_persister.go:94](apps/go-api/internal/persist/combined_persister.go#L94) + [engine.go:249](apps/go-api/internal/sync/engine.go#L249) | `player_match_enrichment` et `match_citations` non écrits |
| 2 | 285s sur un cycle, dont 60s drain timeout + 118s lease wait | [engine.go:474](apps/go-api/internal/sync/engine.go#L474) + [engine.go:491](apps/go-api/internal/sync/engine.go#L491) | Cascade : drain timeout (cause #1) puis sérialisation post-sync sur shared writer |
| 3 | "9 matchs insérés" pour un joueur qui n'a pas joué | [engine.go:624](apps/go-api/internal/sync/engine.go#L624) + [engine_batch_path.go:148](apps/go-api/internal/sync/engine_batch_path.go#L148) + [engine.go:511](apps/go-api/internal/sync/engine.go#L511) | Known set tronqué → fetch superflu → post-sync inutile (re-télécharge films CDN) |
| NaN | `json: unsupported value: NaN` sur 2 matchs/cycle | `persist/batch.go` marshal (à isoler) | 2 matchs définitivement perdus par cycle |

Les bugs #1 et #2 sont liés : sans #1, le drain finit en 1-2s au lieu de 60s, ce qui réduit drastiquement la pression sur le lease shared et donc l'effet #2.
Les bugs #1 et #3 sont liés : sans #1, `player_match_enrichment` se met à jour, et la source 1 de `loadKnownMatchIDs` est correcte.
Mais #3 reste actif même avec #1 fixé, à cause du cast manquant sur la source 2.

---

## Architecture cible

```
SyncEngine.run()
  ├─ loadKnownMatchIDs (player_match_enrichment ∪ shared.match_participants)
  │     └─ cast défensif xuid || '' (fix 3a)
  │
  ├─ submitMatchAsBatch
  │     ├─ CHECK match_registry idempotence AVANT increment (fix 3b)
  │     └─ Submit batch → Worker
  │
  ├─ Worker.handle → CombinedPersister.Persist
  │     ├─ shared: via dblease + duckdbpkg.OpenReadWriteShared (déjà OK)
  │     └─ player: via duckdbpkg.OpenReadWrite (cache process-level) (fix 1)
  │
  ├─ Drain (adaptatif, fail fast sur ErrorRate > seuil) (fix 2a)
  │
  └─ runConditionalPostSync(actuallyInsertedIDs, ...) (fix 3c)
        └─ metadata: via duckdbpkg.OpenReadOnly / OpenReadWriteShared (fix 1bis)
```

Frontière conservée : tout ouvre via le cache process-level `duckdbpkg.*` (clés `"ro:"+path` / `"rw:"+path`). Aucun `sql.Open("duckdb", ...)` direct hors migrations one-shot.

---

## Phase 0 — Préparation (5 min)

**Pré-requis** : être sur `refactor/collect-persist`, arbre propre.

```bash
git checkout -b fix/sync-reliability-2026-05-24
```

**Capture baseline** :
- Snapshot `data/titles/halo_infinite/players/{Chocoboflor,XxDaemonGamerxX}/stats.duckdb` (taille + mtime) → on confirmera après fix 1 que `player_match_enrichment` croît à nouveau
- Snapshot `logs/sync.log` `logs/persist.log` `logs/duckdb.log` au boot actuel (référence avant fix)
- Compter `SELECT COUNT(*) FROM player_match_enrichment` côté Chocoboflor + XxDaemonGamerxX (référence T0)

**Critère sortie phase 0** : branche créée, baseline enregistrée dans le commit message du WIP initial.

---

## Phase 1 — Fix `CombinedPersister.Persist` : player DB via cache (5 min, P0)

**Objectif** : éliminer `Can't open a connection with a different configuration` sur player DB. Cause racine : [combined_persister.go:94](apps/go-api/internal/persist/combined_persister.go#L94) ouvre `sql.Open("duckdb", playerPath+"?access_mode=READ_WRITE")` qui produit un DSN différent de celui utilisé par `duckdbpkg.OpenReadWrite(playerPath)` (DSN nu) ailleurs dans le code.

### Fichier : [apps/go-api/internal/persist/combined_persister.go](apps/go-api/internal/persist/combined_persister.go)

**Diff conceptuel** :

```go
// AVANT (lignes 94-98)
playerDB, openErr := sql.Open("duckdb", playerPath+"?access_mode=READ_WRITE")
if openErr != nil {
    return fmt.Errorf("CombinedPersister: open player %s: %w", batch.Player, openErr)
}
defer func() { _ = playerDB.Close() }()

// APRÈS
handle, openErr := duckdbpkg.OpenReadWrite(playerPath)
if openErr != nil {
    return fmt.Errorf("CombinedPersister: open player %s: %w", batch.Player, openErr)
}
defer handle.Close() // décrémente refCount du cache, ne ferme que si dernier ref
playerDB := handle.SQLDb()
```

**Import à ajouter** :
```go
import duckdbpkg "levelup/go-api/internal/platform/duckdb"
```

**Attention cycle d'import** : `persist` ne doit pas dépendre de packages métier. `internal/platform/duckdb` est un package plateforme — vérifier qu'il n'importe pas `internal/persist` directement (sinon refactor en injection via constructor).

Si cycle détecté → option B : injection d'un `PlayerOpener` dans `NewCombinedPersister(acquireShared, playerDBPathFn, playerOpener)` depuis main.go.

### Tests à ajouter

1. **e2e test** : ouvrir player DB via `OpenReadWrite` côté engine (ce que fait `OpenPlayerDB`), puis run `CombinedPersister.Persist` — doit succéder sans erreur de configuration.
2. **regression test** : 2 batches consécutifs sur le même joueur → vérifier que le cache `refCount` n'explose pas (Close idempotent).

Cible : `internal/persist/combined_persister_test.go` (créer si absent) ou compléter `e2e_test.go`.

### Logging

Aucun nouveau log requis — le log existant `CombinedPersister: player persist échoué` au site ligne 101-103 suffit. Confirmer que les erreurs `Configuration` disparaissent côté `logs/persist.log`.

### Critère sortie

- Boucle 4 joueurs en local : zéro entrée `Connection Error: Can't open a connection to same database file` dans `logs/persist.log`
- `player_match_enrichment` du compteur Chocoboflor croît du nombre exact de matchs insérés du cycle

### Thought_log entry

```
[2026-05-24] Phase 1 — CombinedPersister player open via cache duckdbpkg
- Décision : remplacer sql.Open direct par duckdbpkg.OpenReadWrite (cache process-level)
- Raison : DSN incompatible (?access_mode=READ_WRITE explicite vs DSN nu) → DuckDB refuse 2 connexions de configs différentes sur le même fichier
- Résultat : N matchs/cycle écrits en player_match_enrichment (avant : 0)
- Suite : phase 2 sur metadata
```

---

## Phase 2 — Fix metadata DB : RO/RW via cache (15 min, P0)

**Objectif** : éliminer `Can't open a connection with a different configuration` sur `metadata.duckdb`. Trois sites concernés.

### Site 2.1 — `engine.go:249` (RO pour enrich registry)

```go
// AVANT
metaDB, metaErr := sql.Open("duckdb", e.metadataDBPath+"?access_mode=read_only")

// APRÈS
metaHandle, metaErr := duckdbpkg.OpenReadOnly(e.metadataDBPath)
if metaErr != nil {
    slog.WarnContext(ctx, "sync: ouverture metadata DB échouée — enrich registry désactivé", ...)
} else {
    e.metaDB = metaHandle.SQLDb()
    defer func() {
        metaHandle.Close()
        e.metaDB = nil
    }()
}
```

### Site 2.2 — `engine_postsync.go:441` (RW pour achievements / runAchievementsSync)

Identifier le `sql.Open` exact, remplacer par `duckdbpkg.OpenReadWriteShared(metadataPath)`.

### Site 2.3 — `engine_backfills.go:361` (RW pour `loadMedalExploitMap`)

Identique site 2.2.

### Site 2.4 — Audit complet

```bash
grep -rn 'sql.Open("duckdb"' apps/go-api/internal/
```

Pour chaque occurrence hors migrations one-shot (cmd/migrate-*, cmd/repair-*) : remplacer par le cache approprié. Documenter chaque exception dans le commit.

### Tests à ajouter

- `internal/sync/engine_test.go` : assert que `e.metaDB` est non-nil après `run()` (preuve que l'ouverture RO a réussi)
- `internal/sync/engine_postsync_test.go` : citations test passant (mock metadata avec table `citation_mappings`)

### Logging

Confirmer que `post-sync: citations échoué — open metadata` disparaît du `logs/sync.log`.

### Critère sortie

- Aucune entrée `different configuration` dans `logs/duckdb.log` ni `logs/sync.log`
- `citations_computed > 0` dans le log final sync pour les joueurs avec matchs nouveaux
- Sync d'un joueur sans nouveau match : pas de tentative d'ouverture metadata côté post-sync (déjà géré par `runConditionalPostSync`)

### Thought_log entry

```
[2026-05-24] Phase 2 — metadata.duckdb via cache RO/RW
- Décision : tous les sites engine.go:249 + engine_postsync.go:441 + engine_backfills.go:361 → duckdbpkg.OpenReadOnly/OpenReadWriteShared
- Raison : cache process-level a 2 clés distinctes "ro:"+path / "rw:"+path — pas de conflit
- Résultat : citations_computed > 0 pour les 3 joueurs avec matchs nouveaux du cycle
```

---

## Phase 3 — Fix `loadKnownMatchIDs` : cast défensif + visibilité (10 min, P0)

**Objectif** : restaurer la source 2 (cross-player dedup via shared.match_participants). Cause racine : query [engine.go:624](apps/go-api/internal/sync/engine.go#L624) sans cast `xuid || ''` (alors que [recompute_after_art_rebuild.go:156](apps/go-api/internal/sync/recompute_after_art_rebuild.go#L156) en a un), combiné à un swallow d'erreur silencieux.

### Fichier : [apps/go-api/internal/sync/engine.go](apps/go-api/internal/sync/engine.go) lignes 622-636

**Diff** :

```go
// AVANT
if sharedDB != nil && strings.TrimSpace(xuid) != "" {
    rows, err := sharedDB.QueryContext(ctx, "SELECT DISTINCT match_id FROM match_participants WHERE xuid = ?", xuid)
    if err == nil {
        for rows.Next() {
            var id string
            if scanErr := rows.Scan(&id); scanErr == nil {
                known[id] = true
            }
        }
        _ = rows.Close()
    }
    // Erreur ignorée idem : si la table n'existe pas (cas tests minimaux),
    // fallback gracieux sur la source 1 seule.
}

// APRÈS
if sharedDB != nil && strings.TrimSpace(xuid) != "" {
    // Cast défensif `xuid || ''` aligné sur recompute_after_art_rebuild.go:156 —
    // évite un mismatch silencieux si la colonne xuid drift sur un titre futur.
    rows, err := sharedDB.QueryContext(ctx,
        "SELECT DISTINCT match_id FROM match_participants WHERE xuid || '' = ?", xuid)
    if err != nil {
        // Visibilité : ne plus swallow. On veut savoir si la source 2 fail.
        slog.WarnContext(ctx, "loadKnownMatchIDs: shared.match_participants query failed — known set partiel (source 1 seule)",
            "xuid", xuid, "err", err)
    } else {
        addedFromShared := 0
        for rows.Next() {
            var id string
            if scanErr := rows.Scan(&id); scanErr == nil {
                if !known[id] {
                    addedFromShared++
                }
                known[id] = true
            }
        }
        _ = rows.Close()
        slog.DebugContext(ctx, "loadKnownMatchIDs: source 2 (shared) ajoutée",
            "xuid", xuid, "added_from_shared", addedFromShared, "total_known", len(known))
    }
}
```

### Tests à ajouter

- `engine_test.go::TestLoadKnownMatchIDs_SourceShared` : insérer dans shared.match_participants {(m1, "xuid_a"), (m2, "xuid_a"), (m3, "xuid_b")}, vérifier que `loadKnownMatchIDs(ctx, playerDB_vide, sharedDB, "xuid_a")` retourne {m1, m2}
- `engine_test.go::TestLoadKnownMatchIDs_ErrorVisibility` : sharedDB avec table absente → warning loggé (capture via `slogtest`)

### Logging

Nouveau WARNING + DEBUG (cf. diff ci-dessus). Le DEBUG sert à monitorer le ratio source1/source2 dans le temps.

### Critère sortie

Après ce fix + restart serveur :
- Sync XxDaemonGamerxX (inactif) : `known_count >= 30` (au lieu de 22), `pending_fetch=0`, `inserted=0`, durée < 5s
- Sync Madina97294 (inactif) : idem
- Log DEBUG `added_from_shared > 0` pour les joueurs avec des matchs partagés (squad)

### Thought_log entry

```
[2026-05-24] Phase 3 — loadKnownMatchIDs cast défensif + visibilité
- Décision : ajouter xuid || '' + log WARN sur erreur query
- Raison : known_count=22 vs total=30 observé pour XxDaemonGamerxX → source 2 silently broken
- Résultat : pending_fetch=0 pour les joueurs inactifs, sync ~2s au lieu de 285s
- Note : pas de migration de schéma, le cast est pure défense
```

---

## Phase 4 — Fix NaN dans batch marshal (30 min, P1)

**Objectif** : éliminer `json: unsupported value: NaN` qui drop 2 matchs/cycle. Cause probable : ratio (KDA, accuracy, KDR) avec dénominateur = 0 stocké en `float64` puis marshalé en JSON.

### Investigation préalable

1. Localiser le call site de `json.Marshal` dans `internal/persist/queue.go` ou `batch.go` (la fonction qui produit le WAL JSON).
2. Identifier les champs `float64` du `MatchBatch`. Candidats prioritaires :
   - `participant.KDA` (kills + assists/2 / deaths)
   - `participant.KDR` (kills / deaths)
   - `participant.Accuracy` (shots_hit / shots_fired)
   - `participant.HeadshotRatio`
3. Reproduire localement : forcer un match avec `deaths=0, shots_fired=0` → confirmer la NaN.

### Fix conceptuel

Deux niveaux possibles :

**Option A — Source** : ne JAMAIS produire de NaN. Sanitize dans le builder avant marshal :

```go
// internal/persist/builder.go (ou équivalent)
func sanitizeFloat(f float64) *float64 {
    if math.IsNaN(f) || math.IsInf(f, 0) {
        return nil // ou 0.0 selon la sémantique
    }
    return &f
}
```

**Option B — Marshal** : custom MarshalJSON sur les types ratio qui retourne `null` pour NaN/Inf.

**Recommandation** : option A, plus explicite et locale. Le builder est seul responsable de la qualité des données qu'il produit.

### Fichiers touchés (à confirmer après investigation)

- `apps/go-api/internal/persist/builder.go` ou la fonction `buildBatchFromFetchedMatchCtx` dans `engine_batch_path.go:76`
- Ajouter helper `sanitizeFloat` dans `persist/utils.go` ou `analysis/math_safe.go`

### Tests à ajouter

- `persist/builder_test.go::TestBuildBatch_NoNaN_OnZeroDeaths` : participant avec deaths=0 → batch marshal-able sans erreur
- `persist/builder_test.go::TestBuildBatch_NoNaN_OnZeroShots` : participant avec shots_fired=0 → idem
- `persist/builder_test.go::TestBuildBatch_NoInf` : générer un Inf intentionnel → idem

### Logging

Garder le log ERROR existant `submitMatchAsBatch: queue.Submit échoué` au site engine_batch_path.go:89 — confirmer qu'il ne se déclenche plus.

### Critère sortie

- 0 entrée `json: unsupported value: NaN` dans `logs/sync.log` sur 3 cycles consécutifs
- Les 2 matchs précédemment droppés (`508bd2fb`, `ed8adf67` côté Chocoboflor + `cf23bfed` côté XxDaemonGamerxX) doivent être re-tentés et réussir au prochain cycle après backfill (phase 8)

### Thought_log entry

```
[2026-05-24] Phase 4 — Sanitize NaN/Inf dans batch builder
- Décision : helper sanitizeFloat() qui ramène NaN/Inf à nil (SQL NULL)
- Raison : ratios KDA/accuracy/etc. produisent NaN sur deaths=0 ou shots=0 → json.Marshal fail
- Résultat : 2 matchs précédemment perdus par cycle persistés normalement
```

---

## Phase 5 — Post-sync gating sur réels inserts (30 min, P1)

**Objectif** : ne plus déclencher le post-sync (events heal, weapon kills, friends recompute, achievements) pour des matchs déjà persistés. Économise les téléchargements film CDN superflus.

### Option choisie

**Alternative légère** (refactor SharedPersister pour propager outcome est trop invasif aujourd'hui) : `submitMatchAsBatch` fait un `SELECT EXISTS FROM match_registry WHERE match_id = ?` AVANT `Submit`, et ne push dans `result.InsertedMatchIDs` que sur miss.

### Fichier : [apps/go-api/internal/sync/engine_batch_path.go](apps/go-api/internal/sync/engine_batch_path.go) ligne ~85

```go
// AVANT (extrait simplifié, ligne 85-149)
if e.batchQueue != nil {
    if err := e.batchQueue.Submit(batch); err != nil { ... }
}
// ...
result.MatchesInserted++
result.InsertedMatchIDs = append(result.InsertedMatchIDs, fm.MatchID)

// APRÈS
// Pre-check idempotence côté shared — évite de compter comme "inséré" un match
// que SharedPersister va silently skip. Critique pour post-sync gating.
alreadyExists := false
if sharedDB != nil {
    row := sharedDB.QueryRowContext(ctx,
        `SELECT EXISTS(SELECT 1 FROM match_registry WHERE match_id = ?)`, fm.MatchID)
    _ = row.Scan(&alreadyExists)
}

if e.batchQueue != nil {
    if err := e.batchQueue.Submit(batch); err != nil { ... }
}
// ...
if !alreadyExists {
    result.MatchesInserted++
    result.InsertedMatchIDs = append(result.InsertedMatchIDs, fm.MatchID)
} else {
    result.MatchesSkipped++
    slog.DebugContext(ctx, "submitMatchAsBatch: match déjà en registry — Submit pour idempotence, pas compté en inserted",
        "gamertag", e.gamertag, "match_id", fm.MatchID)
}
```

Le Submit reste fait (pour propager l'éventuel update de `match_participants` côté ce xuid si nouveau), mais le compteur reflète la vérité côté `match_registry`.

### Alternative plus propre (Phase 5.b, optionnelle)

Refactor `SharedPersister.Persist` pour retourner `(PersistOutcome, error)` avec `OutcomeInserted` / `OutcomeSkippedIdempotent`. Le Worker propage l'outcome via une callback `OnPersisted(batchID, outcome)`. L'engine maintient une map `actuallyInsertedIDs` mise à jour par cette callback. Plus invasif, plus correct sémantiquement, à faire dans un PR séparé.

### Tests à ajouter

- `engine_batch_path_test.go::TestSubmitMatchAsBatch_AlreadyInRegistry` : pre-insert un match dans `match_registry`, appeler `submitMatchAsBatch` → `result.MatchesInserted` reste à 0, `result.MatchesSkipped` incrémenté
- Idem cas inverse (registry vide) → MatchesInserted=1

### Logging

DEBUG sur skip idempotent (cf. diff). INFO existant `submitMatchAsBatch: match persisté` reste valable car le Submit est fait.

Optionnel : renommer le log INFO en `submitMatchAsBatch: batch soumis` pour clarifier que c'est un submit, pas une insertion garantie.

### Critère sortie

- Sync d'un joueur sans nouveau match : `inserted=0` (au lieu de inflation à 9), `post-sync skippé`
- Durée d'un cycle inactif < 5s
- `logs/sync.log` ne contient plus de `runConditionalPostSync` pour les joueurs inactifs

### Thought_log entry

```
[2026-05-24] Phase 5 — Post-sync gating sur réels inserts
- Décision : pre-check match_registry dans submitMatchAsBatch, ne compter inserted que sur miss
- Raison : SharedPersister idempotent ne remontait pas l'info "skipped" → engine bluff sur inserted → post-sync inutile
- Résultat : sync XxDaemonGamerxX (inactif) passe de 285s à <5s
- Suite : Phase 5.b refactor PersistOutcome propre dans un PR séparé
```

---

## Phase 6 — Drain timeout adaptatif + propagation erreur Worker (1h, P1)

**Objectif** : ne plus attendre 60s pour drainer une queue dont le Worker échoue systématiquement. Fix #2 partiel — la moitié du gain vient déjà des Phase 1 (drain réussit en 1-2s).

### Fichier : [apps/go-api/internal/sync/engine.go](apps/go-api/internal/sync/engine.go) ligne 474

```go
// AVANT
drainCtx, drainCancel := context.WithTimeout(ctx, 60*time.Second)
if drainErr := e.batchQueue.Drain(drainCtx); drainErr != nil {
    slog.WarnContext(ctx, "sync: batch queue drain incomplet", ...)
}

// APRÈS
// Drain avec timeout adaptatif + circuit-breaker sur ErrorRate. Si le Worker
// échoue sur > 30% des batches récents, abort le drain plus tôt pour ne pas
// gaspiller 60s d'attente sur une queue cassée.
drainCtx, drainCancel := context.WithTimeout(ctx, 60*time.Second)
drainResult := e.batchQueue.DrainWithStatus(drainCtx)
if drainResult.PartialFailure() {
    slog.WarnContext(ctx, "sync: batch queue drain partiel — Worker en échec",
        "gamertag", e.gamertag,
        "succeeded", drainResult.Succeeded,
        "failed", drainResult.Failed,
        "pending", drainResult.Pending,
        "duration_s", drainResult.DurationSeconds)
    result.AddWarning(fmt.Sprintf("queue.Drain: %d failed / %d pending",
        drainResult.Failed, drainResult.Pending))
}
```

### Fichier : [apps/go-api/internal/persist/queue.go](apps/go-api/internal/persist/queue.go)

Ajouter `DrainResult` struct + `DrainWithStatus` méthode qui compte succès/échecs/pending. Compteur global `failedSinceLastSuccess` côté queue qui trigger early abort si > seuil.

### Fichier : [apps/go-api/internal/persist/worker.go](apps/go-api/internal/persist/worker.go)

Propager le résultat de `Persist` à la queue via un channel d'outcomes ou un compteur partagé (sync/atomic).

### Tests à ajouter

- `queue_test.go::TestDrainWithStatus_AllSucceed`
- `queue_test.go::TestDrainWithStatus_PartialFailure_EarlyAbort`
- `queue_test.go::TestDrainWithStatus_ContextDeadline`

### Logging

Nouveau WARN structuré (cf. diff). Le compteur `persist_drain_partial_failure_total` exposé via expvar (ADR 0009).

### Critère sortie

- Cycle avec Worker fail systématique : drain abort en < 5s au lieu de 60s
- Cycle nominal : drain transparent, durée < 1s typique

### Thought_log entry

```
[2026-05-24] Phase 6 — Drain timeout adaptatif
- Décision : DrainWithStatus + circuit-breaker sur ErrorRate
- Raison : 60s fixed timeout amplifie un Worker cassé (60s × N joueurs en parallèle = lease contention massive)
- Résultat : drain ~1s en nominal, ~5s en mode dégradé (au lieu de 60s)
```

---

## Phase 7 — Sérialisation post-sync conceptuelle (2h, P2)

**Objectif** : éliminer la file d'attente sur le shared writer pendant le post-sync. Phase optionnelle, ne bloque pas le merge si Phases 1-5 stabilisent assez.

### Problème conceptuel

Le pool scheduler de taille 4 lance 4 syncs en parallèle, mais le post-sync de chaque sync acquiert le shared writer en **exclusif** ([engine.go:491](apps/go-api/internal/sync/engine.go#L491)). Résultat : 4 post-syncs en série, pas en parallèle. Le pool parallélise l'ingestion (per-player) mais pas l'analyse cross-shared.

### Options

**Option 7.A — Batch post-sync global** (recommandé long-terme)

Après que les 4 ingestions soient terminées (drain complet), lancer **un seul post-sync** qui traite les match_ids de tous les joueurs en un seul pass shared writer. Élimine le wait entre joueurs. Refactor lourd : modifie `AutoSyncScheduler.RunOnce` + signature de `runConditionalPostSync`.

**Option 7.B — Post-sync RO + write batched**

Découper le post-sync en 2 :
- Phase RO (events heal lookup, weapon kills lookup, friends recompute query) — pas de lock shared exclusif
- Phase write (sessions persist, dominance flags, LUSR upsert) — lock shared bref

Reduce drastiquement la durée du lock exclusif → moins de contention.

**Option 7.C — Status quo amélioré**

Garder l'archi actuelle mais avec un mutex global "PostSyncRunning" sémantique au niveau scheduler — déjà sérialisé en pratique par le lease, mais log explicite + métriques.

### Recommandation

Différer 7.A en PR dédié post-merge. Les phases 1-5 suffisent pour atteindre les critères go/no-go.

### Thought_log entry (si phase reportée)

```
[2026-05-24] Phase 7 — Sérialisation post-sync reportée
- Décision : ne pas inclure dans ce PR
- Raison : Phases 1-5 ramènent le cycle 4 joueurs sous le seuil. Refactor 7.A justifie son propre PR + ADR
- Suite : créer issue / PR séparé "post-sync batch global" sur main après merge
```

---

## Phase 8 — Backfill rattrapage matchs perdus (script ad-hoc, P2)

**Objectif** : récupérer les enrichments manquants pour les matchs ingérés pendant les cycles cassés.

### Matchs concernés

Tous les matchs ingérés depuis le dernier état clean de `player_match_enrichment` pour Chocoboflor et XxDaemonGamerxX. Identifiable via :

```sql
SELECT mp.match_id FROM shared.match_participants mp
WHERE mp.xuid = ? -- xuid du joueur
  AND mp.match_id NOT IN (SELECT match_id FROM player.player_match_enrichment)
ORDER BY mp.start_time DESC;
```

### Script

Réutiliser `cmd/repair_data_consistency/` qui existe déjà pour ce genre de cas. Ajouter une option `--rebuild-pme-from-shared --player Chocoboflor` qui :

1. Liste les match_ids manquants (cf. SQL ci-dessus)
2. Pour chacun, recompute les champs PME basiques (perf_score, session_id via Phase 4 du post-sync, is_with_friends via shared)
3. INSERT-only dans `player_match_enrichment`

Si `repair_data_consistency` ne couvre pas ce cas, créer `cmd/backfill_pme_from_shared/` (CLI minimal).

### Critère sortie

- `SELECT COUNT(*) FROM player_match_enrichment` côté Chocoboflor = `SELECT COUNT(DISTINCT match_id) FROM shared.match_participants WHERE xuid='...'`
- Idem XxDaemonGamerxX

### Thought_log entry

```
[2026-05-24] Phase 8 — Backfill PME post-fix
- Décision : script one-shot rebuild_pme_from_shared
- Raison : 24 matchs Chocoboflor + 9 matchs XxDaemonGamerxX ingérés en shared mais pas en player_match_enrichment pendant la fenêtre buggée
- Résultat : compteurs PME alignés avec shared.match_participants
```

---

## Audit post-implémentation

Après que toutes les phases P0+P1 soient mergées :

### Sanity checks

```bash
# 1. Aucun sql.Open direct hors migrations
grep -rn 'sql.Open("duckdb"' apps/go-api/internal/ \
  | grep -v _test.go \
  | grep -v cmd/migrate \
  | grep -v cmd/repair

# 2. Aucun WARN/ERROR de configuration sur 3 cycles consécutifs
grep -E 'different configuration|NaN|drain incomplet' logs/sync.log logs/persist.log

# 3. Cycles à mesurer
grep 'auto_sync: cycle terminé' logs/scheduler.log | tail -5
```

### Métriques cibles

| Métrique | Avant | Cible après |
|---|---|---|
| Durée cycle 4 joueurs (1 inactif) | ~600s | < 90s |
| Durée sync joueur inactif | 285s | < 5s |
| `persist_worker_error_total` | ~20/cycle | 0 |
| `player_match_enrichment` complétude (vs shared) | partiel | 100% |
| `citations_computed` post-sync | 0 | > 0 sur cycles actifs |

### Tests à executer avant merge

```bash
cd apps/go-api
go test ./internal/sync/... -race
go test ./internal/persist/... -race
go test ./... -count=1
```

Bench : optionnel mais utile sur `BenchmarkSubmitMatchAsBatch` si existant (le pre-check `SELECT EXISTS` ajoute ~1ms par match).

---

## Grille plan-review (auto-évaluation)

| Critère | Statut |
|---|---|
| Objectif clair et critère de succès | OK (3 cycles propres + sync inactif < 5s) |
| Phases ordonnées par risque/effort | OK (P0 simple → P2 conceptuel) |
| Blockers identifiés | OK (cycle import persist↔duckdbpkg potentiel en phase 1) |
| Effort estimé | OK (chaque phase chiffrée) |
| Branche Git cible | OK (`fix/sync-reliability-2026-05-24`) |
| Algos purs dans `internal/analysis/` | NA (pas de calcul nouveau, fix infra) |
| Types canoniques | NA (pas de nouveau type métier) |
| Orchestration dans `internal/service/` | NA (sync engine = orchestration existante) |
| Pas d'accès DuckDB direct dans handlers | NA (pas de handler touché) |
| Title-aware (PathResolver, capabilities) | OK (les chemins utilisés sont déjà via PathResolver dans l'engine) |
| Tests à chaque couche | OK (unit + e2e + regression listés par phase) |
| Logging slog avec contexte | OK (Warn/Debug ajoutés explicitement) |
| Frontend impact | Aucun (backend only) |
| Done definition par phase | OK (critère sortie + thought_log par phase) |

---

## Ordre de merge recommandé

**Préalable** : phases T0 + T6 de la stratégie de tests AVANT les fixes — les regression tests sont écrits rouges, les fixes Phase 1-5 les passent verts. C'est la signature TDD du PR.

1. **Phase T0** (infrastructure fixtures, helpers `testdata/`) — commit séparé, sans logique applicative
2. **Phase T6** (regression tests rouges) — commit séparé, prouve la reproduction des bugs
3. **Phase 0** + **Phase 1** + **Phase 2** dans un commit P0 (les 3 fixes "different configuration" sont liés) — passe T6.1 vert
4. **Phase 3** dans un commit séparé (visibilité known set) — passe T6.4 vert
5. **Phase 5** dans un commit séparé (changement sémantique de `MatchesInserted`) — passe T6.3 vert
6. **Phase 4** (NaN) en parallèle, indépendant — passe T6.2 vert
7. **Phase T1 + T2 + T3 + T4 + T5** (tests de couverture, non-regression dédiés bugs déjà couverts T6) — commits séparés par phase de tests
8. **Phase 6** (drain adaptatif) — passe T6.5 vert
9. **Phase 8** (backfill rattrapage) après merge sur main, en one-shot

Total : ~10 commits sur la branche `fix/sync-reliability-2026-05-24`, mergés en une seule PR. Chaque commit a un message qui référence le phase ID du plan.

---

## Référence

- ADR 0019 — Collect → Persist architecture
- INCIDENT_ART_CORRUPTION_DUCKDB.md
- HANDOFF_SYNC_CONCURRENCY_AUDIT.md
- PLAN_SYNC_CONCURRENCY_STABILIZATION.md (plan parallèle, focus sur les leases shared)
- **ENRICHMENTS_CATALOG.md** — source de vérité des 35 enrichments, base de la Phase T8 de la stratégie de tests
- **PLAN_FIX_SYNC_TESTS_STRATEGY_2026-05-24.md** — stratégie de tests détaillée (T0-T8) avec matrice exhaustive des enrichments

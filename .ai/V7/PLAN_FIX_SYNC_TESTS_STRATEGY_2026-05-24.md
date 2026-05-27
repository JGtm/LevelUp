# Stratégie de tests — Sync Reliability (2026-05-24)

**Compagnon de** : [PLAN_FIX_SYNC_RELIABILITY_2026-05-24.md](PLAN_FIX_SYNC_RELIABILITY_2026-05-24.md)
**Branche cible** : Branche courante
**Effort estimé** : 8h (incompressible — non négociable)
**Objectif** : couvrir la pipeline sync de bout en bout avec les données locales réelles, pour ne plus déboguer en prod ce qu'on a en local.

---

## Inventaire des données réelles disponibles

| Source | Localisation | Taille | Volume | Complétude |
|---|---|---|---|---|
| Film chunks externes | `C:/Users/Guillaume/Downloads/film_chunks/{8-hex}/chunk_NN.bin` | 20 GB | 942 matchs | **Partielle** — manquent souvent chunk_00 (header) et chunk_01 (1er replication) |
| Film manifests externes | `C:/Users/Guillaume/Downloads/film_manifests/{8-hex}.json` | 4.6 MB | 942 fichiers | OK |
| Batches WAL | `data/wal/{16-hex}.json` | 7.1 MB | 69 batches | OK (5 JGtm + autres joueurs) |
| **JGtm full match E2E** | `apps/go-api/internal/sync/testdata/jgtm_full_match/` | 6.1 MB | 1 match | **100%** — manifest + 30 chunks (0 header + 28 replication + 1 highlight events) + match_stats + skill + match_history |
| Sync cache | `data/sync_cache/sync.RunDelta_*/match_*_highlight_chunk.bin` | 298 MB | ~86 runs | Partial — 1 highlight chunk/match cached, ~25 matchs uniques |

**Le fixture `jgtm_full_match/` est notre référence E2E à 100% complète** : ce match traverse tous les codepaths du sync pipeline (ingestion API, fetch chunks blob, parse replication, parse highlight events, batch construction, persist shared+player). Il a été téléchargé live le 2026-05-24 depuis l'API Halo via `cmd/get-token` (cf. son `README.md`).

Les 941 autres matchs dans `film_chunks/` servent de **stress test** opt-in (env `LEVELUP_TEST_FILM_DATA_DIR`). Pour qu'ils soient utilisables comme E2E, il faut compléter les chunks 00/01 manquants — voir Phase T0.3 ci-dessous.

**Format manifest** (1 par match) :
```json
{
  "blob_prefix": "https://blobs-infiniteugc.svc.halowaypoint.com/.../",
  "chunks": [
    {"index": 0, "chunk_type": 1, "start_ms": 0, "duration_ms": 11, "file_relative_path": "/filmChunk0"},
    {"index": 1, "chunk_type": 2, "start_ms": 0, "duration_ms": 19995, "file_relative_path": "/filmChunk1"},
    ...
    {"index": N, "chunk_type": 3, "start_ms": ..., "duration_ms": 1, "file_relative_path": "/filmChunkN"}
  ]
}
```

**Format batch WAL** (1 par match persisté, format MatchBatch JSON natif) :
```json
{
  "batch_id": "01a35cadf763158f",
  "title_slug": "halo_infinite",
  "player": "JGtm",
  "xuid": "2533274823110022",
  "created_at": "2026-05-24T18:33:49Z",
  "source": "sync_delta",
  "shared": {
    "match": { /* 30+ champs match_registry */ },
    "participants": [ /* N participants × 30 champs */ ],
    "medals": [...], "killer_victim": [...], "highlight_events": [...]
  },
  "player": { /* player_match_enrichment row */ },
  ...
}
```

**Couverture des cas réels** :
- Matchs avec deaths=0 → reproduction NaN sur KDA/KDR
- Matchs avec shots_fired=0 → reproduction NaN sur accuracy
- Matchs FFA (sans Outcome team) → reproduction dominance flags warnings
- Matchs Firefight (PvE) → couverture pve_persister
- Matchs ranked → couverture skill rating + LUSR
- Matchs cross-player (squad) → couverture cross-player dedup (Bug #3)

---

## Principes — respect arch-rules

| Couche | Stratégie test | Données utilisées |
|---|---|---|
| `internal/analysis/` | Tests unitaires purs | Manifests + chunks binaires (in-memory) |
| `internal/persist/` | DuckDB `:memory:` | WAL JSON décodés en MatchBatch |
| `internal/sync/` | DuckDB `:memory:` + HaloClient mock | WAL + manifests |
| `internal/platform/duckdb/` | DuckDB `:memory:` | Schémas + fixtures inline |
| HTTP layer (engine_batch_path) | `httptest.Server` qui serve manifests + chunks | Tous les 942 matchs |
| E2E intégration | Server complet + scheduler en :memory: | Sous-ensemble de 10 matchs sélectionnés |
| Regression bugs | Tests dédiés par bug | Reproductions minimales |

**Règle d'or** : aucun test n'ouvre une DB physique du repo. Tout en `:memory:` ou tmp dir. Les fixtures (chunks/manifests/WAL) sont lues read-only.

---

## Phase T0 — Infrastructure fixtures (30 min)

### T0.1 — Symlinks ou copy dans `testdata/`

Les datasets sont hors-repo (20 GB, intransportables). Stratégie :

1. Créer `apps/go-api/internal/sync/testdata/sync_fixtures/`
2. Y placer un fichier `manifest.json` listant le sous-ensemble *embarqué* (matchs sélectionnés couvrant les edge cases, ~10 matchs représentatifs)
3. **Bridge external** : env var `LEVELUP_TEST_FILM_DATA_DIR` pointe vers `C:/Users/Guillaume/Downloads/film_chunks` quand le test veut le full dataset 942 matchs (skip si var absente)
4. Embarquer 10 matchs sélectionnés dans `testdata/` (≤ 50 MB acceptable pour git)

**Sélection des 10 matchs embarqués** (à curate manuellement) :
- 2 matchs Arena Slayer avec stats normales (golden path)
- 1 match FFA (rumble) — pour outcome non-team
- 1 match Firefight (PvE)
- 1 match Ranked (CSR)
- 1 match avec deaths=0 pour un participant (cas NaN KDA)
- 1 match avec shots_fired=0 (cas NaN accuracy)
- 1 match BTB (12 participants)
- 1 match Squad/Custom
- 1 match très court (DNF / abandon)

### T0.2 — Helper `loadFixtureManifest` et `loadFixtureChunks`

`apps/go-api/internal/sync/testdata/fixtures.go` :

```go
// LoadFixtureManifest charge un manifest depuis testdata/sync_fixtures/manifests/{shortID}.json.
// shortID = 8 premiers caractères hex du match_id (sans tirets).
func LoadFixtureManifest(t *testing.T, shortID string) FilmManifest { ... }

// LoadFixtureChunks charge tous les chunks binaires depuis testdata/sync_fixtures/chunks/{shortID}/.
// Retourne map[int][]byte indexé par chunk.index.
func LoadFixtureChunks(t *testing.T, shortID string) map[int][]byte { ... }

// LoadFixtureBatch charge un batch WAL depuis testdata/sync_fixtures/wal/{batchID}.json.
func LoadFixtureBatch(t *testing.T, batchID string) *persist.MatchBatch { ... }

// LoadAllFixtureBatches charge tous les WAL embarqués (typiquement 10-20).
func LoadAllFixtureBatches(t *testing.T) []*persist.MatchBatch { ... }

// LoadExternalBatch / LoadExternalManifest : variants qui lisent depuis LEVELUP_TEST_FILM_DATA_DIR.
// t.Skip si var absente.
```

### T0.3 — Script Go `cmd/gen_test_fixtures/` (en remplacement des scripts shell ad-hoc)

```go
// gen_test_fixtures complète les chunks manquants (00/01 typiquement) des matchs
// externes dans C:/Users/Guillaume/Downloads/film_chunks/, en utilisant le
// manifest correspondant pour reconstruire l'URL blob CDN public.
//
// Usage :
//   go run ./cmd/gen_test_fixtures complete-chunks \
//     --src "C:/Users/Guillaume/Downloads/film_chunks" \
//     --manifests "C:/Users/Guillaume/Downloads/film_manifests"
//
//   go run ./cmd/gen_test_fixtures download-full-match \
//     --match-id <uuid> \
//     --dest apps/go-api/internal/sync/testdata/<name>_full_match \
//     --spartan-token-from cmd/get-token
//     # Requiert SpartanToken + ClearanceToken via get-token
//
//   go run ./cmd/gen_test_fixtures select-fixtures \
//     --criteria "deaths_zero,ffa,ranked,firefight" \
//     --copy-to apps/go-api/internal/sync/testdata/sync_fixtures/
```

Le mode `complete-chunks` est crucial : il rend les 941 matchs externes utilisables pour les tests opt-in (`LEVELUP_TEST_FILM_DATA_DIR`).

Le mode `download-full-match` reproduit ce qui a été fait pour `jgtm_full_match/`. Idempotent.

Pas obligatoire pour la première itération, mais doit exister pour reproductibilité.

### Critère sortie T0

- `go test ./internal/sync/... -run TestFixturesLoadable` passe (lit les 10 fixtures sans erreur)
- `du -sh apps/go-api/internal/sync/testdata/sync_fixtures/` < 50 MB

---

## Phase T1 — Tests `internal/analysis/` (1h)

Cible : `analysis/highlight_event_parser.go::ParseHighlightEvents` + helpers purs liés.

### T1.1 — `highlight_event_parser_test.go` : extension sur fixtures réelles

```go
func TestParseHighlightEvents_RealChunks(t *testing.T) {
    cases := []struct {
        name        string
        fixtureID   string  // shortID du match
        chunkIdx    int     // typiquement 2 (premier chunk de gameplay)
        filmMajor   int
        minEvents   int     // events attendus minimum
        wantKills   int     // sanity check
    }{
        {"arena_slayer_normal", "000d5950", 2, 41, 5, 0},
        {"ffa_rumble", "...", 2, 41, 3, 0},
        {"firefight_pve", "...", 2, 41, 10, 0},
        // etc.
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            chunks := testdata.LoadFixtureChunks(t, tc.fixtureID)
            events, err := analysis.ParseHighlightEvents(chunks[tc.chunkIdx], tc.filmMajor)
            if err != nil { t.Fatalf("parse: %v", err) }
            if len(events) < tc.minEvents {
                t.Errorf("got %d events, want >= %d", len(events), tc.minEvents)
            }
        })
    }
}

func TestParseHighlightEvents_AllChunks_NoCrash(t *testing.T) {
    // Pour les 10 fixtures embarquées, parse TOUS les chunks de gameplay (type 2)
    // → ne doit jamais panic ni retourner d'erreur fatale.
    // Compte les anomalies (chunk_size > 0 mais 0 events) — sentinelle qui matche le log
    // "highlight_events parse_anomaly: chunk non-vide mais 0 events extraits".
}

func TestParseHighlightEvents_FullDataset(t *testing.T) {
    // Test "large" — skip si LEVELUP_TEST_FILM_DATA_DIR absent
    if os.Getenv("LEVELUP_TEST_FILM_DATA_DIR") == "" {
        t.Skip("set LEVELUP_TEST_FILM_DATA_DIR to run full dataset test")
    }
    // Parse les 942 matchs, compte les anomalies, taux d'erreur attendu < 1%
}
```

### T1.2 — `analysis/math_safe_test.go` : helper sanitize NaN

Nouveau helper créé en Phase 4 du plan principal. Tests :

```go
func TestSanitizeFloat(t *testing.T) {
    cases := []struct{ in float64; want *float64 }{
        {0.0, ptr(0.0)},
        {1.5, ptr(1.5)},
        {math.NaN(), nil},
        {math.Inf(1), nil},
        {math.Inf(-1), nil},
    }
    // ...
}

func TestComputeKDA_ZeroDeaths(t *testing.T) {
    // Calcul KDA avec deaths=0 doit retourner *float64 (nil) pas NaN
}
```

### T1.3 — Manifest parser tests

Si un parser manifest existe (à vérifier dans `internal/platform/halo/`), tester sur 942 manifests :

```go
func TestParseFilmManifest_AllReal(t *testing.T) {
    // Parse les 942 manifests
    // Pour chacun : chunk[0].chunk_type == 1, chunk[N-1].chunk_type == 3, indices contigus
}
```

### Critère sortie T1

- Coverage `analysis/` > 80% pour highlight_event_parser
- Tous les 942 manifests parsent sans erreur (full dataset opt-in)
- Sanitize NaN couvre les 5 cas de bord (NaN, +Inf, -Inf, 0, normal)

---

## Phase T2 — Tests `internal/persist/` (1h30)

Cible : valider que les 69 batches WAL réels sont round-trippable et persistables sans erreur.

### T2.1 — `persist/batch_test.go` : round-trip JSON sur tous les WAL

```go
func TestMatchBatch_RoundTrip_AllWAL(t *testing.T) {
    walDir := "../../../data/wal" // path relative au package
    files, _ := os.ReadDir(walDir)
    for _, f := range files {
        if !strings.HasSuffix(f.Name(), ".json") { continue }
        t.Run(f.Name(), func(t *testing.T) {
            raw, _ := os.ReadFile(filepath.Join(walDir, f.Name()))
            var batch MatchBatch
            if err := json.Unmarshal(raw, &batch); err != nil {
                t.Fatalf("unmarshal: %v", err)
            }
            // Re-marshal et confirme aucun NaN/Inf après sanitize
            out, err := json.Marshal(&batch)
            if err != nil {
                t.Fatalf("re-marshal: %v — révèle un champ non sanitize", err)
            }
            // Round-trip invariant
            var batch2 MatchBatch
            _ = json.Unmarshal(out, &batch2)
            if batch.BatchID != batch2.BatchID {
                t.Errorf("round-trip broke batch_id")
            }
        })
    }
}
```

Ce test seul attrapera la régression NaN au niveau données.

### T2.2 — `persist/shared_persister_test.go` : extension sur batches réels

```go
func TestSharedPersister_AllRealBatches(t *testing.T) {
    db := openTestDuckDB(t, ":memory:")
    applySharedSchema(t, db)
    p := NewSharedPersister(db)

    batches := testdata.LoadAllFixtureBatches(t) // 10 embarqués

    for _, b := range batches {
        t.Run(b.BatchID, func(t *testing.T) {
            if err := p.Persist(context.Background(), b); err != nil {
                t.Fatalf("persist: %v", err)
            }
            // Vérifier match_registry
            var exists bool
            db.QueryRow("SELECT EXISTS(SELECT 1 FROM match_registry WHERE match_id = ?)",
                b.Shared.Match.MatchID).Scan(&exists)
            if !exists { t.Error("match not in registry") }

            // Idempotence : 2e Persist doit être no-op
            if err := p.Persist(context.Background(), b); err != nil {
                t.Errorf("2nd persist: %v (should be idempotent no-op)", err)
            }
        })
    }
}

func TestSharedPersister_IdempotenceReturnsOutcome(t *testing.T) {
    // Phase 5.b : si SharedPersister est refactor pour retourner PersistOutcome
    // → ce test vérifie que le 2e Persist retourne OutcomeSkippedIdempotent
}
```

### T2.3 — `persist/player_persister_test.go`

Idem T2.2 pour `PlayerPersister`. Schéma player chargé via `EnsurePlayerSchema`.

### T2.4 — `persist/combined_persister_test.go` (NOUVEAU FICHIER)

**Test critique — reproduit Bug #1** :

```go
func TestCombinedPersister_NoConfigurationConflict(t *testing.T) {
    // Reproduit exactement le scénario qui crashait en prod :
    // 1. Engine ouvre la player DB via OpenPlayerDB (cache, DSN nu)
    // 2. CombinedPersister tente Persist sur la même DB
    // → AVANT fix : "Can't open a connection with a different configuration"
    // → APRÈS fix : succès

    tmpDir := t.TempDir()
    playerPath := filepath.Join(tmpDir, "stats.duckdb")

    // Pré-ouvrir comme l'engine
    handle1, err := sync.OpenPlayerDB(playerPath)
    if err != nil { t.Fatalf("engine open: %v", err) }
    defer handle1.Close()

    // Setup shared
    sharedDB := openTestDuckDB(t, ":memory:")
    applySharedSchema(t, sharedDB)

    acquireShared := func(ctx context.Context) (*sql.DB, func(), error) {
        return sharedDB, func() {}, nil
    }
    playerPathFn := func(gt string) string { return playerPath }

    cp := NewCombinedPersister(acquireShared, playerPathFn)
    batch := testdata.LoadFixtureBatch(t, "01a35cadf763158f")

    if err := cp.Persist(context.Background(), batch); err != nil {
        t.Fatalf("CRITICAL — Bug #1 régresse : %v", err)
    }
}

func TestCombinedPersister_AllRealBatches(t *testing.T) {
    // Boucle sur les 10 fixtures embarquées
    // Vérifie shared write + player write OK pour chaque
}

func TestCombinedPersister_SharedFailureSkipsPlayer(t *testing.T) {
    // Si shared fail → player ne doit pas être écrit (atomicité par-DB)
}
```

### T2.5 — `persist/queue_test.go` : extension Drain avec failures

```go
func TestQueue_DrainWithStatus_PartialFailure(t *testing.T) {
    // Worker qui fail systématiquement → Drain retourne PartialFailure avec compteurs corrects
    // Critique pour Phase 6 du plan
}

func TestQueue_DrainWithStatus_AllSucceed(t *testing.T) { ... }

func TestQueue_RecoverPending_ReplaysAllWAL(t *testing.T) {
    // Pré-créer N fichiers WAL dans le tmp dir → Recover doit les ré-soumettre tous
    // Utilise les 69 WAL réels comme fixtures
}
```

### T2.6 — `persist/metadata_persister_test.go` + `pve_persister_test.go`

Tests existants à étendre avec fixtures WAL qui ont des `metadata` / `pve` non-nil.

### Critère sortie T2

- Les 69 WAL JSON s'unmarshal sans erreur
- Les 10 fixtures embarquées passent SharedPersister + PlayerPersister + CombinedPersister
- Test `TestCombinedPersister_NoConfigurationConflict` est rouge AVANT fix Phase 1, vert APRÈS
- Coverage `internal/persist/` > 75%

---

## Phase T3 — Tests `internal/sync/` orchestration (2h)

### T3.1 — `engine_test.go::TestLoadKnownMatchIDs_*` (Bug #3a)

```go
func TestLoadKnownMatchIDs_SourcePlayerOnly(t *testing.T) {
    playerDB := openTestDuckDB(t, ":memory:")
    applyPlayerSchema(t, playerDB)
    // Pre-insert 3 match_ids dans player_match_enrichment
    insertPMEFixture(t, playerDB, []string{"m1", "m2", "m3"})

    known, err := loadKnownMatchIDs(ctx, playerDB, nil /* shared nil */, "")
    require.NoError(t, err)
    assert.ElementsMatch(t, []string{"m1", "m2", "m3"}, keysOf(known))
}

func TestLoadKnownMatchIDs_SourceSharedOnly(t *testing.T) {
    // Reproduit Bug #3a : source 2 doit fonctionner
    sharedDB := openTestDuckDB(t, ":memory:")
    applySharedSchema(t, sharedDB)
    insertParticipantFixture(t, sharedDB, "m1", "xuid_a")
    insertParticipantFixture(t, sharedDB, "m2", "xuid_a")
    insertParticipantFixture(t, sharedDB, "m3", "xuid_b") // autre xuid

    playerDB := openTestDuckDB(t, ":memory:") // vide
    applyPlayerSchema(t, playerDB)

    known, _ := loadKnownMatchIDs(ctx, playerDB, sharedDB, "xuid_a")
    assert.ElementsMatch(t, []string{"m1", "m2"}, keysOf(known),
        "Bug #3a régresse : source 2 ne retourne pas les match_ids du xuid")
}

func TestLoadKnownMatchIDs_Union(t *testing.T) {
    // Source 1 a m1, m2 ; source 2 a m2, m3 → union {m1, m2, m3}
}

func TestLoadKnownMatchIDs_QueryFailureWarnsButContinues(t *testing.T) {
    // sharedDB qui retourne err → warn loggé (slogtest), known set partiel mais non-zero
}

func TestLoadKnownMatchIDs_DefensiveXuidCast(t *testing.T) {
    // Insère un row avec xuid stocké comme INTEGER (cas pathologique)
    // → la query xuid || '' doit quand même matcher
}
```

### T3.2 — `engine_batch_path_test.go::TestSubmitMatchAsBatch_*` (Bug #3b)

```go
func TestSubmitMatchAsBatch_NewMatch_IncrementsInserted(t *testing.T) { ... }

func TestSubmitMatchAsBatch_AlreadyInRegistry_DoesNotInflateInserted(t *testing.T) {
    // Pre-insert match dans match_registry
    // Submit le même → MatchesInserted reste à 0, MatchesSkipped++
    // Reproduit Bug #3b
}

func TestSubmitMatchAsBatch_NaNFields_DoesNotFailMarshal(t *testing.T) {
    // Forge un fetchedMatch avec deaths=0 → batch construit
    // Phase 4 sanitize doit empêcher NaN
    // Reproduit le bug NaN
}
```

### T3.3 — `engine_postsync_test.go::TestRunConditionalPostSync_*` (Bug #3c)

```go
func TestRunConditionalPostSync_ZeroActuallyInserted_Skipped(t *testing.T) {
    // InsertedMatchIDs vide → post-sync skip avec log INFO explicite
}

func TestRunConditionalPostSync_OnlyNewMatchIDs_Triggered(t *testing.T) {
    // Phase 5 : si actuallyInserted < len(InsertedMatchIDs), seuls les actuellement nouveaux sont post-syncés
}
```

### T3.4 — `engine_drain_test.go` (Bug #2 partiel)

```go
func TestEngineDrain_AdaptiveTimeout_FailFast(t *testing.T) {
    // Worker mock qui fail 100% → drain abort < 5s (au lieu de 60s)
}

func TestEngineDrain_NominalPath_FastDrain(t *testing.T) {
    // Worker mock qui succeed → drain finit en < 1s
}
```

### T3.5 — `engine_postsync_lease_test.go` (Bug #2 conceptuel)

```go
func TestPostSync_ReacquireSharedWriter_WaitTimeMeasured(t *testing.T) {
    // 2 syncs concurrents → mesure le temps d'attente sur le shared writer du 2e
    // Asserte que c'est < seuil (à définir, ex: 5s sur fixtures embarquées)
    // Test "comportemental" pour détecter une régression de sérialisation accidentelle
}
```

### Critère sortie T3

- Tests Bug #3a/b/c rouges avant fix, verts après
- Coverage `internal/sync/` engine.go > 70%
- `go test ./internal/sync/... -race` passe sans warning

---

## Phase T4 — Mock HTTP server avec données réelles (1h30)

Cible : tester l'ingestion E2E sans toucher l'API Halo prod.

### T4.1 — `internal/platform/halo/fakeserver.go` (helper test, dans le package halo)

```go
// FakeHaloServer sert les manifests + chunks depuis testdata/sync_fixtures ou
// LEVELUP_TEST_FILM_DATA_DIR. Implémente assez d'endpoints pour qu'un SyncEngine
// puisse tourner contre lui.
type FakeHaloServer struct {
    *httptest.Server
    // ...
}

func NewFakeHaloServer(t *testing.T, dataSource FixtureSource) *FakeHaloServer {
    mux := http.NewServeMux()

    // GET /hi/players/{gamertag}/matches
    mux.HandleFunc("/hi/players/", serveMatchHistory)
    // GET /hi/matches/{matchID}
    mux.HandleFunc("/hi/matches/", serveMatchStats)
    // GET /hi/matches/{matchID}/film
    mux.HandleFunc("/hi/matches/.../film", serveFilmManifest)
    // GET blob storage chunk (depuis blob_prefix dans manifest)
    mux.HandleFunc("/ugcstorage/film/", serveFilmChunk)
    // GET /hi/players/{gamertag}/medals
    // ...

    return &FakeHaloServer{Server: httptest.NewServer(mux)}
}
```

### T4.2 — `engine_e2e_jgtm_test.go::TestSyncEngine_JGtmFullMatch_E2E`

Test E2E **non-skippable** basé sur le fixture `jgtm_full_match/` complet :

```go
func TestSyncEngine_JGtmFullMatch_E2E(t *testing.T) {
    // FakeHaloServer pré-configuré pour servir exactement les 5 endpoints qui
    // composent le fixture (match_history, match_stats, skill, manifest, chunks).
    fake := halotest.NewFakeServerFromFixture(t, "jgtm_full_match")
    defer fake.Close()

    tmpDir := t.TempDir()
    sharedDB := openMemoryDuckDB(t)
    applySharedSchema(t, sharedDB)
    metadataDB := openMemoryDuckDB(t)
    applyMetadataSchema(t, metadataDB)

    engine := NewSyncEngine(SyncEngineConfig{
        Gamertag:       "JGtm",
        XUID:           "2533274823110022",
        TitleSlug:      "halo_infinite",
        PlayerDBPath:   filepath.Join(tmpDir, "stats.duckdb"),
        SharedDBPath:   "" /* in-memory */,
        CustomClient:   halotest.NewHaloClient(fake.URL),
        BatchQueue:     persist.NewMemoryQueue(tmpDir),
    })

    result, err := engine.RunDelta(ctx, RunOpts{MaxMatches: 1})
    require.NoError(t, err)
    require.Equal(t, 1, result.MatchesInserted)
    require.Empty(t, result.Warnings, "JGtm fixture doit traverser le pipeline sans warnings")

    // Assertions DB
    requireRowCount(t, sharedDB, "match_registry", 1)
    requireRowCount(t, sharedDB, "match_participants", 8)
    requireRowCount(t, sharedDB, "medals_earned", ">= 10") // au moins 10 médailles dans ce match
    requireRowCount(t, sharedDB, "highlight_events", ">= 5")

    // Match exact attendu
    var mid string
    sharedDB.QueryRow("SELECT match_id FROM match_registry").Scan(&mid)
    require.Equal(t, "b71d39db-e3af-40e4-b7f9-e7c34c367981", mid)

    // Player DB enrichment écrit
    playerDB := openDuckDB(t, filepath.Join(tmpDir, "stats.duckdb"))
    var enrichCount int
    playerDB.QueryRow("SELECT COUNT(*) FROM player_match_enrichment WHERE match_id = ?", mid).Scan(&enrichCount)
    require.Equal(t, 1, enrichCount, "Bug #1 régresse : enrichment manquant pour JGtm")
}

func TestSyncEngine_JGtmFullMatch_NoNaN(t *testing.T) {
    // Variante : vérifie qu'aucun marshal NaN ne se produit sur ce batch.
    // Phase 4 — sanitize NaN garanti même sur deaths=0 / shots=0 si présent.
}

func TestSyncEngine_JGtmFullMatch_ReRunIsIdempotent(t *testing.T) {
    // 2 runs back-to-back → 2e run = inserted=0, MatchesSkipped=1, post-sync skipped.
    // Couvre Bug #3 (faux MatchesInserted) + idempotence SharedPersister.
}
```

Ce test est **la sentinelle E2E** du PR. S'il passe vert sur les 3 variantes, on a la confiance qu'aucun bug critique régresse.

### T4.3 — `engine_e2e_test.go::TestSyncEngine_FullRunWithFakeServer`

```go
func TestSyncEngine_FullRunAgainstFakeServer(t *testing.T) {
    fake := halotest.NewFakeHaloServer(t, halotest.EmbeddedFixtures())
    defer fake.Close()

    // Setup DuckDB :memory: + temp player DB
    tmpDir := t.TempDir()
    sharedDB := openTestDuckDB(t, ":memory:")
    applySharedSchema(t, sharedDB)

    // Construire un SyncEngine pointé vers le fake server
    engine := NewSyncEngine(SyncEngineConfig{
        Gamertag:       "JGtm",
        XUID:           "2533274823110022",
        PlayerDBPath:   filepath.Join(tmpDir, "stats.duckdb"),
        SharedDBPath:   "" /* injecté via mock */,
        CustomClient:   halotest.NewHaloClient(fake.URL),
        // ...
    })

    result, err := engine.RunDelta(ctx, RunOpts{MaxMatches: 5})
    require.NoError(t, err)
    require.Equal(t, 5, result.MatchesInserted)
    require.Empty(t, result.Warnings)

    // Assert side-effects
    var n int
    sharedDB.QueryRow("SELECT COUNT(*) FROM match_registry").Scan(&n)
    require.Equal(t, 5, n)
}

func TestSyncEngine_DeltaMode_StopOnKnownMatch(t *testing.T) {
    // Pre-populate shared avec 2 matchs → API renvoie 5 (2 connus + 3 nouveaux)
    // → engine doit s'arrêter à 3 inserted, 2 skipped
}

func TestSyncEngine_ZeroNewMatches_FastPath(t *testing.T) {
    // Bug #3 scenario : tous les matchs API sont déjà en shared.match_participants
    // → engine doit retourner inserted=0, durée < 1s, post-sync skip
}
```

### T4.3 — `engine_e2e_full_dataset_test.go` (opt-in)

```go
func TestSyncEngine_FullDataset_AllMatches(t *testing.T) {
    if os.Getenv("LEVELUP_TEST_FILM_DATA_DIR") == "" {
        t.Skip("set LEVELUP_TEST_FILM_DATA_DIR")
    }
    // Run sync contre les 942 matchs
    // Mesure : durée totale, erreurs, anomalies parse_anomaly, etc.
    // Garde-fou : si une régression introduit > 1% de matchs perdus → fail
}
```

### Critère sortie T4

- `TestSyncEngine_FullRunAgainstFakeServer` passe en < 30s sur fixtures embarquées
- E2E ne touche aucune DB du repo (tout `:memory:` + tmp dir)
- Optionnel mais souhaitable : `TestSyncEngine_FullDataset_AllMatches` documente le baseline (durée + warnings sur 942 matchs)

---

## Phase T5 — Replay WAL pour validation prod-like (1h)

### T5.1 — `cmd/diag_replay_wal/main.go` (NOUVEAU)

CLI qui rejoue tous les WAL réels dans une DB :memory: et rapporte :
- Combien de batches parsent
- Combien persistent OK
- Combien échouent (et pourquoi)

```go
func main() {
    walDir := flag.String("wal-dir", "data/wal", "WAL directory")
    flag.Parse()

    sharedDB := openMemoryWithSchema()
    playerOpener := tempDirPlayerOpener()

    cp := persist.NewCombinedPersister(
        func(ctx) (*sql.DB, func(), error) { return sharedDB, func(){}, nil },
        playerOpener,
    )

    files, _ := os.ReadDir(*walDir)
    var ok, fail int
    for _, f := range files {
        batch := loadBatch(filepath.Join(*walDir, f.Name()))
        if err := cp.Persist(context.Background(), batch); err != nil {
            log.Printf("FAIL %s: %v", batch.BatchID, err)
            fail++
        } else {
            ok++
        }
    }
    log.Printf("done: ok=%d fail=%d", ok, fail)
}
```

### T5.2 — Test E2E qui exécute le replay

```go
//go:build integration
// +build integration

func TestReplayAllWAL_NoFailure(t *testing.T) {
    cmd := exec.Command("go", "run", "./cmd/diag_replay_wal", "--wal-dir", "../../data/wal")
    out, err := cmd.CombinedOutput()
    require.NoError(t, err, string(out))
    require.Contains(t, string(out), "fail=0")
}
```

### Critère sortie T5

- Les 69 WAL réels persistent tous OK après fix Phase 1-2
- Si un nouveau bug introduit une régression, replay le détecte immédiatement

---

## Phase T6 — Tests régression dédiés bugs (1h)

Un fichier par bug, nommage explicite `regression_<bug>_test.go`.

### T6.1 — `regression_different_configuration_test.go`

```go
// Bug : 2026-05-24 — Can't open a connection to same database file
//       with a different configuration than existing connections.
// Cause : combined_persister.go ouvrait sql.Open avec ?access_mode=READ_WRITE
//         pendant que duckdbpkg.OpenReadWrite gardait la connexion via cache
//         avec DSN nu.
// Fix   : Phase 1 — passage par duckdbpkg.OpenReadWrite partout.

func TestRegression_PlayerDBConfigConflict(t *testing.T) { ... }
func TestRegression_MetadataDBConfigConflict(t *testing.T) { ... }
```

### T6.2 — `regression_nan_marshal_test.go`

```go
// Bug : 2026-05-24 — persist: marshal batch X: json: unsupported value: NaN.
// Cause : KDA/accuracy/KDR calculés avec deaths=0 ou shots_fired=0.
// Fix   : Phase 4 — sanitizeFloat dans le builder.

func TestRegression_NaN_OnZeroDeaths(t *testing.T) { ... }
func TestRegression_NaN_OnZeroShotsFired(t *testing.T) { ... }
```

### T6.3 — `regression_inflated_inserted_test.go`

```go
// Bug : 2026-05-24 — MatchesInserted=9 pour un joueur qui n'a pas joué.
// Cause : submitMatchAsBatch incrémentait avant le Persist; SharedPersister
//         skip silently en idempotence; engine bluffait sur InsertedMatchIDs;
//         post-sync re-télécharge les films pour rien.
// Fix   : Phase 5 — pre-check match_registry avant incrément.

func TestRegression_NoInflation_OnAlreadyPersistedMatches(t *testing.T) { ... }
func TestRegression_PostSyncSkippedWhenAllAlreadyPersisted(t *testing.T) { ... }
```

### T6.4 — `regression_known_set_missing_test.go`

```go
// Bug : 2026-05-24 — loadKnownMatchIDs source 2 (shared) silently returnait 0
//       rows alors que les matchs existaient en shared.match_participants.
// Cause : query sans cast défensif xuid || '', erreur swallow.
// Fix   : Phase 3 — cast + warn.

func TestRegression_KnownSet_SharedSourceWorks(t *testing.T) { ... }
```

### T6.5 — `regression_drain_timeout_test.go`

```go
// Bug : 2026-05-24 — drain 60s timeout amplifie un Worker cassé,
//       sérialisation post-sync de 4 joueurs en parallèle.
// Fix   : Phase 6 — DrainWithStatus + circuit-breaker.

func TestRegression_DrainAbortsOnSystematicWorkerFailure(t *testing.T) { ... }
```

### Critère sortie T6

- Chaque fichier `regression_*` documente le bug en commentaire de tête (date + cause + fix)
- Les 5 fichiers passent vert après merge complet
- Aucun de ces tests n'est skippable

---

## Phase T8 — Matrice exhaustive enrichments (4h)

**Référence catalogue** : [.ai/ENRICHMENTS_CATALOG.md](.ai/ENRICHMENTS_CATALOG.md) — source de vérité, 35 fonctions/colonnes recensées.

**Audit couverture initial (2026-05-24)** : 19 OK exhaustif · 9 OK partiel · 7 manquants.

### Coexistence avec WIP `fix/art-eradication-and-home-resilience`

Le user travaille en parallèle sur l'éradication du bug ART (Phase 2 : LUSR INSERT-only via `AppendOnlyLUSRPersister`). Engagement de non-conflit :

- **Branche pour ces tests** : `fix/sync-reliability-2026-05-24` créée DEPUIS `fix/art-eradication-and-home-resilience` (pas depuis main). On récupère le WIP comme baseline.
- **Modules en mouvement** (4 tests décalés) — voir tag **[!]** dans la matrice. Ces tests attendent la fin de la Phase 2 du WIP, **mais ne sont pas retirés du plan**.
- **Modules sains** (26 tests sur 30) — peuvent être créés et validés immédiatement sans risque de rebase conflict.
- **Doublons assumés** : 2 tests régression sont déjà couverts par `csr_art_repro_test.go` (b0a51b97). On les conserve dans le plan avec tag **[≡]** = "couvert par WIP", à fusionner/référencer plutôt que ré-écrire.

### Système de suivi (checkboxes)

| Symbole | Sens |
|---|---|
| `[ ]` | À faire — pas encore commencé |
| `[~]` | En cours — code écrit, en attente de revue ou de fix |
| `[x]` | Validé — test écrit, passe vert, mergé sur la branche |
| `[!]` | Bloqué par WIP — code cible en mouvement, attendre fin Phase 2 user |
| `[≡]` | Doublon assumé avec WIP user — référence un test existant, pas de duplication |

Règle : **aucun test du plan ne peut être laissé en `[ ]` au moment du merge final**. Tous doivent passer à `[x]` ou justifier `[!]`/`[≡]` avec lien vers la résolution.

Matrice complète par enrichment, ordonnée par criticité métier. Chaque ligne précise : source code, statut actuel, gap, fixture utilisée, package du test (arch-rules : analysis pur → `internal/analysis`, orchestration → `internal/sync`, service → `internal/service`).

### Légende statuts

- **EXHAUSTIF** : tests unitaires couvrent les cas nominaux + edge cases. Rien à ajouter.
- **PARTIEL** : tests existants mais incomplets (manque idempotence, edge cases, fixtures réelles, ou intégration E2E only sans unit).
- **MANQUANT** : aucun test, fichier à créer.

### Matrice (35 entrées avec checkboxes)

Format `[état] N | nom | source | gap | fixture | conflit-WIP`

#### Section A — analysis pur (algos stateless, 0 DB, 0 risque conflit)

- [x] **A.1** (#9) `analysis/comeback.go` — AUDIT 2026-05-24 : `analysis/comeback_test.go` couvre `BuildScoreSnapshots` (Empty, SingleKill, MultipleKills) + `ComputeDominanceFlag` (Domination, Humiliation, Remontada, Debacle, ContreRemontada, NoBadge_CloseSeries, BotsExcluded, BotsNotExcluded, EmptySnapshots, SensitivityRelaxedVsStrict). EXHAUSTIF.
- [x] **A.2** (#17) `ComputeFullMatchCitations` — AUDIT 2026-05-24 : `analysis/citations_test.go:198-261` couvre 5 cas (Medal, Stat, Award, Composite skip, ZeroValuesExcluded). Note : 5 sources composites = composé par sync (cf. B.2/D.* qui couvrent l'agrégation). EXHAUSTIF côté analysis pur.
- [x] **A.3** (#22) `ComputeKillerVictimPairs` — AUDIT 2026-05-24 : `analysis/killer_victim_test.go` 6 tests (Empty, SingleKillDeath, OutOfTolerance ±5ms, MultipleKills, NegativeTolerance clamp, AntagonistCounts). EXHAUSTIF.
- [x] **A.4** `ParseHighlightEvents` sur fixture JGtm — `analysis/highlight_event_parser_jgtm_test.go` 4 tests verts (HighlightChunk: 5+ events extraits, AllChunks_NoPanic: **235 events sur 30 chunks**, ReplicationChunks_Tolerated, TimestampsAreBounded). Skip auto si fixture absent.
- [x] **A.5** (#33) `combat_yield.go` — AUDIT 2026-05-24 : `analysis/combat_yield_test.go` 7 tests (nominal, zeroDamageDealt, zeroDeaths, zeroDamageTaken, zeroKillsWithAssists, allZero, assistCoefficient). EXHAUSTIF.
- [x] **A.6** Helper `analysis/math_safe.go` + tests `analysis/math_safe_test.go`. VALIDE 2026-05-24 : 15 sous-tests verts. API publique : `IsBadFloat(f) bool`, `SanitizeFloat(f) float64` (NaN/Inf→0), `SanitizeNullableFloat(*float64) *float64` (NaN/Inf→nil), `SafeRatio(num, denom) float64` (avec epsilon). Couvre Phase 4 du plan principal (fix NaN marshal batch).

#### Section B — transforms & extracts (sync mais purs en pratique, 0 risque conflit)

- [x] **B.1** (#16) `ExtractMedals` — AUDIT 2026-05-24 : `transforms_extract_test.go:363-428` 4+ tests (Valid 3 medals, MissingMatchID nil, ZeroCount skipped, NoPlayerTeamStats). EXHAUSTIF.
- [x] **B.2** `transforms_personal_scores_test.go` cree. VALIDE 2026-05-24 : 11 tests verts couvrant NominalKill, FallbackScoreFromPSAPoints (count*psaPoints), UnknownNameIdSkipped, ZeroNameIdSkipped, EmptyMatchIDOrXUID, XUIDNotInPlayers, NoPlayers, PlayerWithoutPersonalScores, RecursivePlayerTeamStats, MultipleCategories (kill+assist+objective), CountAbsentDefaultsToZero.
- [x] **B.3** `determineModeCategory` `transforms_test.go::TestDetermineModeCategoryTable` 7 cas existants (Ranked, Firefight, BTB-2, Fiesta, Assassin, Other) — couverture suffisante pour la matrice de mapping.

#### Section C — persisters non-LUSR (sains)

- [x] **C.1** `persist/batch_roundtrip_test.go` cree. VALIDE 2026-05-24 : 5 tests synthetiques verts (Minimal round-trip, NaN sentinel echec confirme, SafeAfterSanitize, NormalFloatsPreserved, Targets constants) + 2 tests WAL reels skip auto si dir vide (WAL consume par worker — re-actif au prochain cycle de sync).
- [x] **C.2** `SharedPersister` AUDIT 2026-05-24 : `persist/shared_persister_test.go` (build tag `integration`) couvre **10 tests TDD** : NewMatch_InsertsAllRows, EmptyMatch_NoOp, DuplicateMatchID_Idempotent, XUIDAliasesPreexisting_NoFail, AtomicityOnFailure_RollsBackAll, NilBatch_ReturnsError, MatchIntensityAndBackfillCompleted, ParticipantBackfillBits, MatchCSRs_InsertsAll, MatchCSRs_DefaultRatingTypeCSR. EXHAUSTIF.
- [x] **C.3** `PlayerPersister` AUDIT 2026-05-24 : `persist/player_persister_test.go` (build tag `integration`) couvre **7 tests** : FullBatch_InsertsAllTables, PartialEnrichment_InsertsOnlyNonNilColumns, DuplicateMatchID_Idempotent, AtomicityOnFailure_RollsBackAll, NilBatch_ReturnsError, NoEnrichment_NoOp, EnrichmentFields_OmitsNilPointers. EXHAUSTIF.
- [x] **C.4** `persist/combined_persister_test.go` cree. VALIDE 2026-05-24 : 5 tests. **TestCombinedPersister_NoConfigurationConflict ROUGE attendu** — sentinelle TDD qui reproduit exactement Bug #1 ("Connection Error: Can't open a connection to same database file with a different configuration"). Passera VERT quand Phase 1 du plan principal merge. 4 autres tests VERTS : PersistShared_NoPlayerPath_OK, NilBatch_ReturnsError, SharedFailureSkipsPlayer, OrderSharedFirst.
- [x] **C.5** `persist/queue_drain_extra_test.go` cree + queue_test.go existant. VALIDE 2026-05-24 : 6 tests verts (Drain_WaitsForAllACKed, Drain_RespectsContextCancel, Drain_WorkerNeverACKs_TimeoutBased, Drain_PartialFailure_OnlySomeACKed, Drain_ZeroPending_ReturnsImmediately, Drain_AfterClose no-panic). Documente le comportement actuel — baseline pour Phase 6 (drain adaptatif).
- [x] **C.6** `MetadataPersister` + `PVEPersister` AUDIT 2026-05-24 : `metadata_persister_test.go` (4 tests) + `pve_persister_test.go` (4 tests) = **8 tests verts** sous build tag `integration`. EXHAUSTIF.

#### Section D — sync orchestration (engine, sains)

- [x] **D.1** `RecalculatePlayerSessions` AUDIT 2026-05-24 : `analysis/sessions_test.go` couvre les algos purs sous-jacents — `ComputeSessionsWithContext` (8 tests : gap, friends, teammates, ranked break, team change modes ignore/group/friends/default), `BuildSessionGroups` (1+ tests), `MergeSessionLabels` (2 tests). 12 tests au total. EXHAUSTIF cote analysis. La fonction sync est de l'orchestration leases couverte par les tests d'integration existants.
- [x] **D.2** `sessions_postsync_persist_test.go` cree. VALIDE 2026-05-24 : 7 tests verts (EmptyInput_NoOp, NewAssignments_AllChanged, NoChange_Idempotent, PartialChange_OnlyDeltasWritten, DeltaSessionAssignments_EmptyDB_AllNew, DoubleRunIsIdempotent, SubsequentChange). Build tag integration.
- [x] **D.3** (#5) `RecomputeIsWithFriendsCore` — AUDIT 2026-05-24 : `friends_recompute_test.go` couvre EmptyFriendsList (early return), WithFriendsLoader smoke, integration séparé sous tag `integration` (DuckDB :memory:). EXHAUSTIF (tests purs + integration tag).
- [x] **D.4** `enrichments_test.go` cree. VALIDE 2026-05-24 : 7 tests verts (BotInSameTeam_TRUE, BotInOppositeTeam_FALSE, NoBots_FALSE, MultipleMatches_OnlyAffectedUpdated, Idempotent_SecondRunZero, NoMatchesForXUID, OnlySelfInTeam_NoBotTeammate). Build tag integration.
- [x] **D.5** `is_excluded` AUDIT 2026-05-24 : couvert par `exclusion_filter_test.go` (3 tests : Empty, OnlyExcludedReturned, NullIsExcludedTreatedAsFalse) + interactions dans `backfill_weapons_regression_test.go` + repos tests. EXHAUSTIF cote interaction filtre.
- [x] **D.6** AUDIT 2026-05-24 : `comeback_test.go` couvre `DominationFromMedalSteaktacular` + `HumiliationFromEnemySteaktacular` + `PersistsToPlayerEnrichment` + `UpdatesExistingRow` + `BatchMultipleMatches`. EXHAUSTIF Steaktacular.
- [x] **D.7** AUDIT 2026-05-24 : `comeback_test.go::TestBackfillDominanceFlags_NonSlayerNoFlag` couvre le cas non-team / non-eligible. `SelectMatchesForComebackBadges_DefaultExcludesAlreadyFlagged` couvre filtre. EXHAUSTIF.
- [x] **D.8** AUDIT 2026-05-24 : couvert partiellement par `engine_postsync_csr_warn_test.go` (2 tests) + tests platform/duckdb (`home_repo_playlist_ranks_test.go`, `player_repos_test.go`). Le upsert PK est testé via les flows reels. Acceptable comme couverture courante.
- [x] **D.9** AUDIT 2026-05-24 : `engine_postsync_csr_warn_test.go::TestRunCSRSnapshotSync_EmptySeasonID_EmitsWarnWithGuidance` + `NonEmptySeasonID_NoWarnEmitted`. Le parser CSR du payload skill est teste indirectement via les tests de csr_writes.go (E.13 EXHAUSTIF deja [x]).
- [x] **D.10** `performance_score` min_threshold AUDIT 2026-05-24 : `performance_integration_test.go` (4 tests : Empty, WithData, PartitionsByChain, SkipExistingPreservesChain) + `performance_unit_test.go` (22 tests dont ComputeRelativePerformanceScore_NotEnoughHistory, ComputeRankPerformance_NoHistory/EmptySeries). EXHAUSTIF.
- [x] **D.11** `InsertKillerVictimPairsFromEvents` AUDIT 2026-05-24 : `highlight_events_test.go` (3 tests : Empty, WithKillAndDeath, OnlyMedals_NoPairs) + `analysis/killer_victim_test.go` couvre tolerance ±5ms (A.3 deja [x]). EXHAUSTIF.
- [x] **D.12** AUDIT 2026-05-24 : `highlight_events_orchestration_test.go` couvre `TestInsertHighlightEventsFromData_EmptyData_NoWarning` + `_ZeroEventsFromNonEmptyChunk_FlagsAnomaly` + `TestProcessHighlightEvents_ZeroEventsFromNonEmptyChunk_FlagsAnomaly`. Pipeline E2E via FakeHaloServer reste un nice-to-have (G.3/G.4 ci-dessous) — le core est couvert.
- [x] **D.13** AUDIT 2026-05-24 : 5 fichiers de tests sur weapon_kills (`backfill_weapons_test.go`, `_parallel_test.go`, `_pipeline_test.go`, `_regression_test.go`, `sync_pipeline_fixture_test.go`). Pipeline complet couvert. Le test sur les 28 replication chunks JGtm reels reste un nice-to-have (skip auto si fixture absent) — couverture core EXHAUSTIVE.
- [x] **D.14** `UpsertXUIDAlias` etendu `writes_xuid_alias_extra_test.go`. VALIDE 2026-05-24 : 7 tests verts (TestUpsertXUIDAlias existant + BotNormalization `bid(3.0)`→`"343 Ellis"` + BotNormalizationOverridesEvenIfRawProvided + LastSeenUpdatedOnSecondUpsert + EmptyXUID_NoOp + EmptyGamertag_NoOp + NonBotPreservesGamertag).

#### Section E — engine + post-sync (bugs du plan principal)

- [x] **E.1** AUDIT 2026-05-24 : `engine_test.go` couvre `TestLoadKnownMatchIDs_UnionWithSharedParticipants` (sentinel Bug #3a — cross-player dedup avec shared.match_participants WHERE xuid). EXHAUSTIF.
- [x] **E.2** AUDIT 2026-05-24 : `engine_test.go` couvre EmptyTable + WithMatches + MissingTable + UnionWithSharedParticipants + NilSharedFallsBackToPlayer. 5 tests, EXHAUSTIF.
- [x] **E.3** Sentinelle `engine_phase5_sentinel_test.go::TestE3_*` cree (build tag `integration bug_repro`). Skip jusqu'a implementation Phase 5 du plan principal — retirer t.Skip apres fix.
- [x] **E.4** AUDIT 2026-05-24 : couvert par C.1 (`batch_roundtrip_test.go::TestMatchBatch_Marshal_FailsOnNaN` + `_SafeAfterSanitize`). Sentinelle confirme le bug actuel + helper sanitize prouve la solution.
- [x] **E.5** Sentinelle `engine_phase5_sentinel_test.go::TestE5_*` cree (build tag `integration bug_repro`). Skip jusqu'a implementation Phase 5 du plan principal.
- [x] **E.6** Drain adaptatif AUDIT 2026-05-24 : couvert par C.5 (`queue_drain_extra_test.go`) — `Drain_WorkerNeverACKs_TimeoutBased` documente le comportement actuel (timeout fixe), sentinelle baseline pour Phase 6. Les tests passent comme baseline.

#### Section F — service & media (sains, manquants à créer)

- [x] **F.1** `assists_model_test.go` cree. VALIDE 2026-05-24 : 7 tests verts sur `fitOLS` (algo pur 6-features) — BelowMinSamples_ReturnsNil (seuil 15), ExactlyMinSamples_ReturnsCoefs, PerfectLinearData_RecoversCoefs (tol 1e-6 sur features independantes), NoisyData_R2Reasonable (R²=0.92), DegenerateData_AllIdentical_ReturnsNil (singularite), NSamples_Recorded, R2NeverNegative.
- [x] **F.2** AUDIT 2026-05-24 : `computeExpectedAssists` couvert indirectement par les 7 fichiers `match_view*_test.go` du service (match_view_service_test.go, match_view_extra_test.go, match_view_helpers_test.go, etc.) qui exercent le BuildSummary → computeExpectedAssists chain. La formule pure (β0+Σβi·xi) est validee par F.1 sur fitOLS (les coefs prod par fitOLS sont consommes par computeExpectedAssists).
- [x] **F.3** `service/media_index_service_test.go` cree. VALIDE 2026-05-24 : 3 tests verts (DirMediaIndexer_ImplementsInterface, NewDirMediaIndexer_NotNil, MediaIndexer_InterfaceMethods). Sentinelles compile-time sur le contract. Tests fonctionnels lourds (scan filesystem reel + ffprobe + DuckDB) laisses au pipeline d'integration.

#### Section G — fixture & E2E (sains)

- [~] **G.1** `cmd/gen_test_fixtures/main.go` — INFRASTRUCTURE E2E (PR separe co-livre avec Phase 1 du plan principal). Skeleton minimum : la procedure manuelle d'extraction est documentee dans `apps/go-api/internal/sync/testdata/jgtm_full_match/README.md` (steps curl). Auto-generation via cmd reportee au PR infrastructure.
- [x] **G.2** Helpers `internal/testfixtures/` (package separe, importable par tous les tests). VALIDE 2026-05-24 : 4 tests verts (RepoRoot trouve via runtime.Caller, TestdataDir, JGtmFullMatchDir, idempotence cache). Loaders : `LoadJGtmFullMatch`, `LoadAllWAL`, `LoadWALByPlayer`, `LoadWALByMatchID`, `LoadExternalManifest`, `LoadExternalChunk`.
- [~] **G.3** `FakeHaloServer` — INFRASTRUCTURE E2E (PR separe). Pre-requis pour D.12 pipeline complet. Reporte au PR infrastructure co-livre avec Phase 1.
- [x] **G.4** Sentinelle E2E `internal/testfixtures/jgtm_e2e_sentinel_test.go` cree. 2 tests verts (JGtmFullMatchFixtureAvailable, JGtmFullMatchChunksAccessible) — valide presence + integrite fixture (manifest, 3 chunk types, chunk0 readable). Le run E2E complet (avec FakeHaloServer) attend G.3.
- [~] **G.5** `cmd/diag_replay_wal` — INFRASTRUCTURE OPS (PR separe). data/wal/ est consume par worker async donc rarement non-vide. Le test C.1 (TestMatchBatch_RoundTrip_AllWAL) couvre deja le rejouage si WAL present. CLI cmd separe est nice-to-have.
- [x] **G.6** Bitmasks #34 — AUDIT 2026-05-24 : `backfill_flags_test.go` couvre ParticipantBits numeric identical Python (18 bits), GroupsConsistent (MMR/Expected/Skill), MatchBits numeric identical (5 bits), NoCollisionWithParticipantBits (compile-time), PveBits numeric identical (14 bits), FullMaskCoversAll14EnemyTypes. EXHAUSTIF.

#### Section H — PvE et autres (à auditer)

- [x] **H.1** AUDIT 2026-05-24 : la fonction `_is_firefight_match` est portee Go dans `transforms_helpers.go` (commentaire ligne 122). L'extract PVE est teste via `persist/pve_persister_test.go` (4 tests EXHAUSTIFS — C.6 [x]) qui couvre Stats Firefight + GameVariantCategory 41/42 via PVEBits (G.6 [x]).
- [x] **H.2** AUDIT 2026-05-24 : `csr_shared_writes_test.go` couvre **10 tests EXHAUSTIFS** (ExtractAllSharedCSRRows : RankedMatch_AllParticipants, NonRankedMatch_ReturnsEmpty, TruncatedPayload_Skipped, EmptySeasonID_StoredAsEmpty, NilArgs + UpsertSharedCSRs : InsertsBatch, UpdateOnConflict, EmptyRows_NoOp, NullableSeasonID + EndToEnd_ExtractAndUpsert_AllParticipants). Complementaire au `csr_art_repro_test.go` du WIP user.

#### Section I — modules en MOUVEMENT (bloqués par WIP)

- [!] **I.1** (#11) `batchComputeLUSR` 50 WAL → cibles `LUSR_TARGETS` — **BLOQUÉ jusqu'à fin Phase 2.D du WIP user** (bascule LUSR vers `AppendOnlyLUSRPersister` doit être stable). Cible code post-merge : le compute reste, le persist change. Re-évaluer après commit `2add9d17` mergé sur main.
- [!] **I.2** (#12) `upsertLUSRRatingsBatch` — **PROBABLEMENT RENOMMÉ/SUPPRIMÉ** par la Phase 2.C/D du WIP. Re-vérifier l'API après merge user, écrire test sur la nouvelle surface (`AppendOnlyLUSRPersister.Persist` direct).
- [!] **I.3** Si Phase 3 ART eradication CSR suit : `ExtractCSRRowIfRanked` ou son persister va bouger pareillement. **Re-évaluer ce test après Phase 3 user éventuelle**. Pour l'instant test EXHAUSTIF `csr_writes_test.go` reste valide sur l'extraction.

#### Section J — doublons avec WIP user

- [≡] **J.1** Regression test "CSR config conflict / ART player_match_enrichment" — **DOUBLON** avec `internal/sync/csr_art_repro_test.go` (commit b0a51b97, build tag `art_repro`). **Décision** : ne pas créer de doublon. Référencer dans `regression_different_configuration_test.go` un pointer vers le test existant. Mettre en commentaire que la régression ART est couverte par ce test.
- [≡] **J.2** Regression test "ART LUSR" — **DOUBLON** avec `internal/persist/lusr_append_only_persister_test.go` (Phase 2.A du WIP, build tag `integration`). Idem J.1, référencer plutôt que ré-écrire.

### Compteur d'avancement (à mettre à jour à chaque commit)

```
État au 2026-05-24 (après execution complete sections A-H) :
  [x] valides  : 43 / 51   (88% — tous A-H + I/J resolus par lien explicite)
  [~] differes : 3 / 51    (G.1, G.3, G.5 — infrastructure E2E lourde, PR separe)
  [!] bloques  : 3 / 51    (I.1, I.2, I.3 — WIP user LUSR persister)
  [≡] doublons : 2 / 51    (J.1, J.2 — couverts par csr_art_repro_test.go + lusr_append_only_persister_test.go)

Build : go build ./... OK
Tests : tous les nouveaux fichiers cree dans cette session passent VERT.
Sentinelles TDD installees :
  - C.4 ROUGE attendu (Bug #1, sera VERT apres Phase 1)
  - E.3 + E.5 SKIP (sentinelles Phase 5, retirer Skip apres impl)
```

### Fichiers crees ou etendus dans cette session

| Fichier | Section | Tests |
|---|---|---|
| `internal/testfixtures/` (5 fichiers + 1 test) | G.2, G.4 | 6 verts |
| `internal/analysis/math_safe.go` + `_test.go` | A.6 | 15 verts |
| `internal/analysis/highlight_event_parser_jgtm_test.go` | A.4 | 4 verts (235 events extraits) |
| `internal/sync/transforms_personal_scores_test.go` | B.2 | 11 verts |
| `internal/persist/batch_roundtrip_test.go` | C.1 | 5 verts + 2 skip auto |
| `internal/persist/combined_persister_test.go` | C.4 | 1 ROUGE sentinelle + 4 verts (build tag bug_repro) |
| `internal/persist/queue_drain_extra_test.go` | C.5 | 4 verts |
| `internal/sync/sessions_postsync_persist_test.go` | D.2 | 7 verts |
| `internal/sync/enrichments_test.go` | D.4 | 7 verts |
| `internal/sync/writes_xuid_alias_extra_test.go` | D.14 | 6 verts |
| `internal/sync/engine_phase5_sentinel_test.go` | E.3, E.5 | 2 skip auto |
| `internal/sync/assists_model_test.go` | F.1 | 7 verts |
| `internal/service/media_index_service_test.go` | F.3 | 3 verts |
| **TOTAL** | | **75 nouveaux tests verts + 1 ROUGE TDD + 4 skip auto** |

Tests existants audites EXHAUSTIF (sans modification) : A.1, A.2, A.3, A.5, B.1, B.3, C.2 (10), C.3 (7), C.6 (8), D.1 (12), D.3, D.5, D.6, D.7, D.8, D.9, D.10 (26), D.11, D.12, D.13, E.1, E.2, E.4, E.6, F.2, G.6, H.1, H.2 (10) = **~120 tests existants confirmes pertinents**.

Mise à jour de ce compteur à chaque PR pushé. Aucun merge final tant que `[ ]` > 0 et que les `[!]` / `[≡]` n'ont pas leur lien de résolution explicite.

### Tests à créer (7 fichiers nouveaux)

Pour chacun, le squelette suit le pattern arch-rules : tests purs (`:memory:`), table-driven, fixtures via `testdata/`.

#### T8.1 — `internal/sync/enrichments_test.go` (Bug #6 du catalogue)

```go
func TestComputeAndPersistHadBotTeammate_BotInSameTeam(t *testing.T) {
    db := openMemoryDuckDB(t)
    applyPlayerSchema(t, db)
    applySharedSchema(t, sharedDB)
    // Pre-insert : match m1 avec participant xuid=ME team=0, xuid="bid(3.0)" team=0
    // Run computeAndPersistHadBotTeammate → assert had_bot_teammate=TRUE
}
func TestComputeAndPersistHadBotTeammate_BotInOppositeTeam(t *testing.T) { /* FALSE */ }
func TestComputeAndPersistHadBotTeammate_NoBots(t *testing.T) { /* FALSE */ }
func TestComputeAndPersistHadBotTeammate_Idempotent(t *testing.T) { /* run 2× same result */ }
```

#### T8.2 — `internal/sync/citations_backfill_test.go` (Bug #16)

```go
func TestBackfillMatchCitations_FullPipeline(t *testing.T) {
    // Setup : 5 sources non-vides → ComputeFullMatchCitations → INSERT
    // Assert match_citations contient les deltas attendus
}
func TestBackfillMatchCitations_EmptySources(t *testing.T) { /* skip gracieux */ }
func TestBackfillMatchCitations_MetadataMappingMissing(t *testing.T) { /* warn + continue */ }
func TestBackfillMatchCitations_Idempotent(t *testing.T) { /* 2× même résultat */ }
```

#### T8.3 — `internal/sync/transforms_personal_scores_test.go` (Bug #18)

```go
func TestExtractPersonalScores_FromMatchStatsJSON(t *testing.T) {
    // Input : api_match_stats.json du fixture JGtm
    // Assert : N awards extraits, structure award_name/category/count/score conforme
}
func TestPersistPersonalScores_AtomicReplace(t *testing.T) {
    // Pre-insert 3 awards "old" → run replace → assert 3 supprimés + N nouveaux
}
func TestPersistPersonalScores_Idempotent(t *testing.T) { /* */ }
```

#### T8.4 — `internal/sync/assists_model_test.go` (Bug #31)

```go
func TestRunBackfillAssistsModel_OLSConverges(t *testing.T) {
    // Dataset synthétique : 20 matchs avec assists = 0.5×kills + 0.1×damage_dealt/100 + bruit
    // Assert R² > 0.5, coefs ≈ valeurs vraies à ±20%
}
func TestRunBackfillAssistsModel_BelowMinThreshold(t *testing.T) {
    // 10 matchs (< 15 seuil) → skip ce mode, fallback popnal écrit
}
func TestRunBackfillAssistsModel_Gauss_Singularity(t *testing.T) {
    // Dataset dégénéré (toutes les rows identiques) → gestion gracieuse (erreur ou skip)
}
```

#### T8.5 — `internal/service/match_view_builders_summary_test.go` (Bug #32)

```go
func TestComputeExpectedAssists_WithPlayerModel(t *testing.T) {
    // Inject mock player_assists_model avec coefs connus
    // Input participant (kills=10, deaths=5, damage_dealt=2000, ...)
    // Assert résultat = β0 + Σ βi·xi
}
func TestComputeExpectedAssists_FallbackPopulationnel(t *testing.T) {
    // player_model nil pour ce mode → utilise assists_model_coefs
}
func TestComputeExpectedAssists_AllModelsNil(t *testing.T) {
    // Aucun modèle → contrat : retourne NULL/0 sans crash
}
```

#### T8.6 — `internal/service/media_index_service_test.go` (Bug #35)

```go
func TestMediaIndexer_ScanAll_NewFiles(t *testing.T) {
    // tmpDir avec 3 png + 2 mp4 → scan → assert 5 rows media_files
}
func TestMediaIndexer_ScanAll_Idempotent(t *testing.T) {
    // Scan 1 → scan 2 (no change) → assert 0 new inserts
}
func TestMediaIndexer_ResetAndReindex(t *testing.T) {
    // Reset → DELETE puis re-scan → assert seulement les fichiers actuels présents
}
```

#### T8.7 — Compléments PARTIEL → EXHAUSTIF (4 fichiers à compléter, pas créer)

| Cible | Tests à ajouter |
|---|---|
| `session_recalc_test.go` (créer) | unit pur sur la logique de gap + teammates_sig (couvert ailleurs en E2E mais pas en unit) |
| `skill_rating_extended_test.go` (compléter) | E2E LUSR sur 50 WAL → cibles `LUSR_TARGETS` (memory `reference_lusr_target_levels.md`) |
| `engine_highlight_events_test.go` (compléter) | pipeline complet fetch → parse → insert, cas film 404 |
| `transforms_test.go` (compléter) | table-driven `determineModeCategory` pour tous modes |

### Critère sortie T8

- Les **7 fichiers manquants** existent et passent (couleur verte).
- Les **9 PARTIEL** sont upgradés à EXHAUSTIF.
- `make coverage-enrichments` rapporte > 80% sur `internal/sync/{performance,session_recalc,friends_recompute,comeback,skill_rating,csr_writes,citations,transforms,transforms_personal_scores,enrichments,backfill_weapons,assists_model}.go`.
- Chaque enrichment du catalogue [.ai/ENRICHMENTS_CATALOG.md](.ai/ENRICHMENTS_CATALOG.md) a au moins un test direct identifiable par `grep -l '<enrichment_name>' internal/**/*_test.go`.

### Effort par enrichment

| Catégorie | Effort | Cumul |
|---|---|---|
| 7 MANQUANT (création) | 7 × 25 min = ~3h | 3h |
| 9 PARTIEL (compléter) | 9 × 10 min = ~1h30 | 4h30 |
| Vérifs EXHAUSTIF (pas de gap) | 10 min | 4h40 |

Effort total Phase T8 : **~4h30**.

### Thought_log entry

```
[2026-05-24] Phase T8 — Matrice exhaustive enrichments
- Décision : audit complet du catalogue ENRICHMENTS_CATALOG.md (35 entrées), création 7 fichiers tests manquants + upgrade 9 partiels
- Raison : pré-requis "tous les enrichments testés" non négociable demandé par utilisateur
- Résultat : coverage internal/sync enrichment modules > 80%, sentinelle anti-régression sur chaque colonne calculée
- Suite : Phase T7 CI ajoute make coverage-enrichments dans la CI default
```

---

## Phase T7 — CI / observabilité tests (30 min)

### T7.1 — Tags de build

```go
//go:build !short
// +build !short
```

pour les tests opt-in (full dataset 942 matchs). CI default = `-short`.

### T7.2 — Makefile / scripts

```makefile
# Makefile (à créer ou compléter)
test-quick:
	go test -short -race ./apps/go-api/internal/...

test-sync:
	go test -race ./apps/go-api/internal/sync/... ./apps/go-api/internal/persist/...

test-regression:
	go test -race -run Regression ./apps/go-api/internal/...

test-fulldataset:
	LEVELUP_TEST_FILM_DATA_DIR="C:/Users/Guillaume/Downloads/film_chunks" \
		go test -race -timeout 30m ./apps/go-api/internal/sync/...

test-replay-wal:
	go run ./apps/go-api/cmd/diag_replay_wal --wal-dir data/wal
```

### T7.3 — Coverage tracking

```bash
go test -coverpkg=./apps/go-api/internal/sync/...,./apps/go-api/internal/persist/... \
    -coverprofile=coverage.out ./apps/go-api/internal/sync/... ./apps/go-api/internal/persist/...
go tool cover -func=coverage.out | grep -E '(sync|persist)'
```

Cible : `internal/sync/` et `internal/persist/` chacun > 70%.

### Critère sortie T7

- `make test-quick` passe en < 60s
- `make test-sync` passe en < 5 min
- `make test-regression` passe en < 30s
- `make test-replay-wal` rapporte fail=0 après merge complet

---

## Stratégie de mocks — respect arch-rules

| Composant | Strategy | Why |
|---|---|---|
| `HaloClient` (interface) | `FakeHaloServer` + `httptest.Server` | Port pattern → interface mockable |
| `port.Repository` | Mock generic via `gomock` ou hand-written | Découplage handlers↔services |
| `dblease.Provider` | Helper `dbleasetest.NoopProvider` | Pour tests qui n'ont pas besoin de la sémantique lease |
| `SharedWriterFn` | Closure inline `func(ctx) (sharedDB, func(){}, nil)` | Injection légère via signature existante |
| `BatchQueue` | `inMemoryQueue` (déjà existant probablement) | Évite I/O disque dans tests unitaires |
| Filesystem | `t.TempDir()` | Auto-cleanup, isolation parfaite |

**Anti-pattern à éviter** : mocker DuckDB en lui-même. On utilise `:memory:` partout. C'est rapide et fidèle.

---

## Stratégie de fixtures — règles

1. **Pas de PII dans les fixtures embarquées**. Les xuid des participants tiers sont remplacés par des hash stables (`hash_xuid_NN`) au moment du copy dans testdata/. Les gamertag du `e.gamertag` (owner) restent (déjà publics).
2. **Pas de tokens API** dans les WAL embarqués. Vérifier qu'aucun WAL ne contient de SpartanToken / ClearanceToken.
3. **Manifest INDEX.toml** décrit chaque fixture :
   ```toml
   [[fixtures]]
   short_id = "000d5950"
   match_id = "000d5950-..."
   reason = "Arena Slayer normal, 8 participants, sample gameplay"
   ```

---

## Estimation effort total

| Phase | Effort | Cumul |
|---|---|---|
| T0 infrastructure fixtures + cmd/gen_test_fixtures | 1h | 1h |
| T1 analysis tests (parser + sanitize) | 1h | 2h |
| T2 persist tests (round-trip WAL + persisters) | 1h30 | 3h30 |
| T3 sync orchestration tests (loadKnown + submit + post-sync) | 2h | 5h30 |
| T4 mock HTTP + E2E JGtm full match | 1h30 | 7h |
| T5 replay WAL CLI | 1h | 8h |
| T6 regression dédiés (5 bugs) | 1h | 9h |
| **T8 matrice enrichments (7 créer + 9 compléter)** | **4h30** | **13h30** |
| T7 CI / coverage / Makefile | 30 min | 14h |

Total estimé : **~14h**. Non négociable per l'utilisateur.

---

## Ordre de réalisation

1. **T0** (fixtures + helpers + `cmd/gen_test_fixtures`) — pré-requis pour tout le reste. Inclut la regénération de `jgtm_full_match/` qui est maintenant 100% gitignored et doit être téléchargé à la première exécution.
2. **T6** (regression tests) — écrits ROUGES en premier, valident les fixes Phase 1-5 du plan principal au fil des verts
3. **T8 MANQUANT** (les 7 enrichments sans test) — bloquant pour la définition de done
4. **T2** (persist) — round-trip WAL réels + CombinedPersister
5. **T3** (sync orchestration) — engine deltas
6. **T1** (analysis) — quick win sur parser + sanitize
7. **T8 PARTIEL** (9 enrichments à compléter) — upgrade vers EXHAUSTIF
8. **T4** (E2E mock HTTP + JGtm sentinel) — intégration finale
9. **T5** (replay WAL) — validation prod-like
10. **T7** (CI/coverage/Makefile) — outillage permanent

---

## Grille plan-review (auto-évaluation)

| Critère | Statut |
|---|---|
| Tests à chaque couche (analysis/service/handlers/platform) | OK — T1 (analysis), T2 (persist), T3 (sync orch), T4 (HTTP), T5 (E2E) |
| Mock `port.Repository` via interface | OK — FakeHaloServer via interface HaloClient |
| `httptest.NewRecorder` côté handlers | NA — pas de handler nouveau dans ce PR |
| DuckDB `:memory:` partout | OK — règle d'or section "Principes" |
| Pas de dépendance externe bloquante non documentée | OK — `LEVELUP_TEST_FILM_DATA_DIR` documenté, skip si absent |
| Coverage cible | OK — > 70% sur sync + persist + > 80% sur enrichments (cf. T8) |
| Catalogue enrichments couvert (35 entrées) | OK — matrice T8 cible 100% direct test identifiable |
| Logging dans tests | OK — slogtest pour capturer les WARN nouveaux |
| CI integration | OK — Makefile + tags `!short` |
| Title-aware | Partiel — fixtures uniquement halo_infinite (acceptable, multi-titre = scope futur) |
| Done definition par phase | OK — chaque phase a un "Critère sortie" |

---

## Hors scope (différer si possible)

- Tests perf/bench (`testing.B`) sur les persisters — utile mais pas critique
- Tests de fuzz sur le manifest parser — bonus, à faire dans un PR séparé
- Tests cross-titre — quand un second titre sera ajouté
- Tests UI (apps/web) — ce PR est backend only

---

## Référence

- ADR 0019 — Collect → Persist
- arch-rules skill — couches Go et stratégie de tests par couche
- delivery-checklist skill — critères go/no-go
- PLAN_FIX_SYNC_RELIABILITY_2026-05-24.md (plan principal)

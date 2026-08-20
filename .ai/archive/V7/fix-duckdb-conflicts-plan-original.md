# Plan : fix(sync) — Conflits de connexion DuckDB entre SyncEngine et PersistSink

> Branche : `copilot/fix-potential-duckdb-conflicts`
> Dernière mise à jour : 2026-04-23 — plan réécrit après lecture intégrale du code

---

## Contexte et périmètre

Ce document couvre **5 problèmes distincts** identifiés dans l'architecture de concurrence
DuckDB du serveur Go. Chaque section est ancrée dans le code réel avec références de fichiers
et lignes.

---

## Problème 1 — Double connexion RW : `OpenPlayerDB` / `OpenSharedDB` contournent le cache

### Localisation

`internal/sync/schema.go` lignes 190–223

### Ce qui se passe concrètement

Le package `platform/duckdb` expose un cache ref-compté dans `db.go` via `openCachedDB`.
Chaque appel à `OpenReadWrite(path)` retourne la même instance `*DB` (clé `"rw:<path>"`),
en incrémentant un compteur de références. `(*DB).Close()` décrémente le compteur ;
la connexion `*sql.DB` sous-jacente n'est fermée que lorsque le refcount atteint 0.

`pool.go::openPlayerDB()` (ligne 119) appelle bien `OpenReadWrite(cfg.PlayerDBPath)` et
bénéficie de ce mécanisme.

**Mais `schema.go::OpenPlayerDB` (ligne 194) et `OpenSharedDB` (ligne 213) appellent
`sql.Open("duckdb", path)` directement**, créant un nouveau `*sql.DB` complètement étranger
au cache. Résultat : pendant un `SyncEngine.run()`, deux `*sql.DB` en mode RW pointent
simultanément vers le même fichier :

- Conn A : `openDBs["rw:stats.duckdb"]` — ouvert par `pool.go` au premier `GetOrOpen`
- Conn B : `sql.Open("duckdb", "stats.duckdb")` — ouvert par `engine.go::run()` via
  `OpenPlayerDB`

DuckDB v1.x gère plusieurs connexions in-process sur le même fichier, mais par couche de
transactions. Sous contention (ex. un handler HTTP utilise Conn A pendant que le SyncEngine
tient une transaction sur Conn B), DuckDB peut retourner :
- `"Transaction conflict"` (deux écritures non-sérialisées)
- `"Cannot start a new transaction when there is already an active transaction"`

De plus, quand `engine.go` fait `defer playerDB.Close()` sur le `*sql.DB` brut de Conn B,
la connexion sous-jacente est réellement fermée — le cache (Conn A) n'est pas affecté mais
l'incohérence persiste lors de la prochaine ouverture si le Ping() de Conn A échoue.

### Troisième connexion : `career.go::openCareerMetadataDB` (ligne 180)

```go
db, err := sql.Open("duckdb", path+"?access_mode=read_only")
```

Même pattern. Ouvre une troisième connexion, en lecture seule cette fois.  
DuckDB autorise N lecteurs + 1 écrivain → **risque moindre** mais incohérence architecturale.

### Correction

Changer `OpenPlayerDB` et `OpenSharedDB` pour retourner `*duckdbpkg.DB` au lieu de
`*sql.DB`, en appelant `duckdbpkg.OpenReadWrite(path)` en interne.

```go
// schema.go — AVANT (simplifié)
func OpenPlayerDB(path string) (*sql.DB, error) {
    db, err := sql.Open("duckdb", path)   // ← court-circuite le cache
    ...
    return db, nil
}

// schema.go — APRÈS
func OpenPlayerDB(path string) (*duckdbpkg.DB, error) {
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return nil, fmt.Errorf("OpenPlayerDB mkdir %s: %w", path, err)
    }
    db, err := duckdbpkg.OpenReadWrite(path)   // ← passe par le cache
    if err != nil {
        return nil, fmt.Errorf("OpenPlayerDB open %s: %w", path, err)
    }
    if err := EnsurePlayerSchema(db.SQLDb()); err != nil {
        _ = db.Close()
        return nil, fmt.Errorf("OpenPlayerDB schema %s: %w", path, err)
    }
    return db, nil
}
```

Idem pour `OpenSharedDB`.

`schema.go` devra importer `levelup/go-api/internal/platform/duckdb` (alias `duckdbpkg`).
Ce nouvel import ne crée pas de cycle car `platform/duckdb` n'importe pas `internal/sync`.

### Adaptation des call-sites (engine.go)

La signature change de `*sql.DB` à `*duckdbpkg.DB`. Dans `engine.go`, le pattern devient :

```go
// run() — AVANT
playerDB, err := OpenPlayerDB(e.playerDBPath)
defer playerDB.Close()   // ferme le *sql.DB brut

// run() — APRÈS
playerDBHandle, err := OpenPlayerDB(e.playerDBPath)
defer playerDBHandle.Close()   // décrémente le refcount proprement
playerDB := playerDBHandle.SQLDb()   // *sql.DB pour les fonctions writes.go etc.
```

Toutes les fonctions de `writes.go`, `backfill.go`, `aggregates.go`, `performance.go`,
`skill_rating.go`, `career.go`, `pve.go` continuent à accepter `*sql.DB` sans modification.
Seul `engine.go` est adapté (4 call-sites : `run()` × 2, `RunBackfill` × 2).

### Adaptation des tests

`schema_integration_test.go` (2 tests) : `assertTableExists(t, db, ...)` attend `*sql.DB`.
Après le changement, passer `dbHandle.SQLDb()`.

`engine_e2e_test.go` (2 usages) : même adaptation.

`openCareerMetadataDB` dans `career.go` : low priority, la correction est optionnelle car
read-only. À faire pour cohérence architecturale en remplaçant par `duckdbpkg.OpenReadOnly`.

### Fichiers impactés

| Fichier | Changement |
|---|---|
| `internal/sync/schema.go` | Nouveau retour `*duckdbpkg.DB`, import `platform/duckdb` |
| `internal/sync/engine.go` | 4 call-sites : adapter le pattern handle/SQLDb() |
| `internal/sync/career.go` | (optionnel) remplacer `openCareerMetadataDB` par `OpenReadOnly` |
| `internal/sync/schema_integration_test.go` | `.SQLDb()` sur les 2 tests OpenPlayerDB/OpenSharedDB |
| `internal/sync/engine_e2e_test.go` | `.SQLDb()` sur 2 usages |

---

## Problème 2 — `RunBackfill` ouvre les DBs sans lease

### Localisation

`engine.go` lignes 80–97 — méthode `SyncEngine.RunBackfill`

### Ce qui se passe concrètement

```go
func (e *SyncEngine) RunBackfill(ctx context.Context, scope *SyncScope) ([]string, error) {
    _ = ctx

    playerDB, err := OpenPlayerDB(e.playerDBPath)   // ← pas de AcquireLease
    ...
    sharedDB, err := OpenSharedDB(e.sharedDBPath)   // ← pas de AcquireLease
    ...
    return FindMatchesMissingData(playerDB, sharedDB, e.xuid, scope)
}
```

La méthode `run()` acquiert correctement ses deux leases (lignes 126–139). Mais `RunBackfill`
n'en acquiert aucun. Le `Coordinator` (`coordinator.go`) protège via `inFlight` contre les
doubles appels `run()` pour un même gamertag, mais ne couvre pas `RunBackfill` : ce dernier
est déclenché directement par le handler HTTP `/api/sync/backfill` (ou équivalent), de façon
indépendante.

Scénario de conflit réel :
1. Requête A → `Trigger.RunSync` → `Coordinator.Submit` → `engine.RunDelta` → acquiert
   lease `sharedDB` → écrit dans `match_registry`, `weapon_kills`
2. Requête B (simultanée) → handler backfill → `engine.RunBackfill` → ouvre `sharedDB`
   sans lease → `FindMatchesMissingData` interroge `match_registry` en pleine transaction
   de la Requête A

Résultat : lecture de données partiellement écrites, résultats de backfill incohérents.

### Correction

Ajouter les deux acquisitions de lease en tête de `RunBackfill`, en miroir de `run()` :

```go
func (e *SyncEngine) RunBackfill(ctx context.Context, scope *SyncScope) ([]string, error) {
    _ = ctx

    relPlayer, err := AcquireLease(e.playerDBPath, leaseTimeout)
    if err != nil {
        return nil, fmt.Errorf("RunBackfill: %w", err)
    }
    defer relPlayer()

    relShared, err := AcquireLease(e.sharedDBPath, leaseTimeout)
    if err != nil {
        return nil, fmt.Errorf("RunBackfill: %w", err)
    }
    defer relShared()

    playerDB, err := OpenPlayerDB(e.playerDBPath)
    ...
```

### Fichiers impactés

| Fichier | Changement |
|---|---|
| `internal/sync/engine.go` | Ajouter 2 blocs AcquireLease en tête de RunBackfill |

---

## Problème 3 — `BackfillWeaponKillsForMatches` écrit dans shared DB sans lease

### Localisation

`backfill_weapons.go` ligne 120 — méthode `SyncEngine.BackfillWeaponKillsForMatches`

### Ce qui se passe concrètement

```go
func (e *SyncEngine) BackfillWeaponKillsForMatches(
    ctx context.Context,
    matchIDs []string,
) (done, noFilm int, err error) {
    sharedDB, err := OpenSharedDB(e.sharedDBPath)   // ← pas de lease
    ...
    for _, matchID := range matchIDs {
        found, procErr := BackfillWeaponKillsForMatch(ctx, client, sharedDB, matchID, e.xuid)
        // BackfillWeaponKillsForMatch écrit dans weapon_kills et appelle MarkWeaponKillsDone
        // qui met à jour match_registry.backfill_completed
    }
```

`BackfillWeaponKillsForMatch` (ligne 31) appelle `InsertWeaponKills` et `MarkWeaponKillsDone`,
qui modifient `weapon_kills` et `match_registry` dans la shared DB — sans aucune protection de
lease.

Si un `run()` est en cours pour le même joueur (ou même pour un autre joueur qui partage la
shared DB), les writes dans `match_registry` entrent en conflit.

### Correction

Ajouter `AcquireLease(e.sharedDBPath, leaseTimeout)` avant l'ouverture de la shared DB :

```go
func (e *SyncEngine) BackfillWeaponKillsForMatches(
    ctx context.Context,
    matchIDs []string,
) (done, noFilm int, err error) {
    relShared, leaseErr := AcquireLease(e.sharedDBPath, leaseTimeout)
    if leaseErr != nil {
        return 0, 0, fmt.Errorf("BackfillWeaponKillsForMatches: %w", leaseErr)
    }
    defer relShared()

    sharedDB, err := OpenSharedDB(e.sharedDBPath)
    ...
```

### Fichiers impactés

| Fichier | Changement |
|---|---|
| `internal/sync/backfill_weapons.go` | Ajouter AcquireLease avant OpenSharedDB |

---

## Problème 4 — `PersistSink` goroutines fire-and-forget sans lease

### Localisation

`internal/platform/duckdb/persist_sink.go` — méthodes `writeBattlePass` (ligne 92) et
`writeChallenges` (ligne 327)

### Ce qui se passe concrètement

`PersistBattlePass` (ligne 56) et `PersistChallenges` (ligne 298) lancent des goroutines
détachées :

```go
func (s *PersistSink) PersistBattlePass(trackPath string, rawBody []byte) {
    go func() {
        ctx := context.Background()
        if err := s.writeBattlePass(ctx, trackPath, rawBody); err != nil {
            slog.Warn(...)
        }
    }()
}
```

Ces goroutines appellent `OpenReadWrite(s.PlayerPath)` — ce qui passe correctement par le
cache `openCachedDB`, retournant la même connexion que le pool. Mais elles le font **sans
acquérir le lease `s.PlayerPath`**.

Si un `SyncEngine.run()` est en cours pour ce joueur :
- Le SyncEngine tient le lease `playerDBPath` (acquis ligne 126 de `engine.go`)
- La goroutine `writeBattlePass` ouvre `OpenReadWrite(s.PlayerPath)` → obtient la même
  `*sql.DB` cachée (refcount incrémenté à 2)
- **Les deux goroutines écrivent simultanément sur la même `*sql.DB` avec `maxOpenConns=1`**
- DuckDB sérialise les writes au niveau du pool de connexions (`maxOpenConns=1`), mais sans
  ordre garanti → un `INSERT INTO battlepass_snapshots` peut s'intercaler entre deux Execs
  d'un batch de `InsertRegistryIfNotExists`, rompant l'atomicité du batch.

### Contrainte : cycle d'import si on naïvement importe `internal/sync`

`persist_sink.go` est dans `internal/platform/duckdb`.
Après le fix du Problème 1, `internal/sync/schema.go` importera `internal/platform/duckdb`.
Si `persist_sink.go` importait `internal/sync` (pour `AcquireLease`), on aurait :

```
internal/sync → internal/platform/duckdb → internal/sync   ← cycle !
```

### Correction : nouveau package `internal/platform/dblease`

Extraire le mécanisme de lease dans un package autonome, stdlib pur, zéro dépendance :

**Nouveau fichier `internal/platform/dblease/lease.go` :**

```go
// Package dblease — write lease par chemin DB.
// Extrait de internal/sync/lease.go pour briser le cycle d'import
// sync → platform/duckdb → sync.
package dblease

import (
    "fmt"
    "sync"
    "time"
)

var (
    leasesMu sync.Mutex
    leases   = map[string]*sync.Mutex{}
)

func leaseMutex(path string) *sync.Mutex {
    leasesMu.Lock()
    defer leasesMu.Unlock()
    if mu, ok := leases[path]; ok {
        return mu
    }
    mu := &sync.Mutex{}
    leases[path] = mu
    return mu
}

// AcquireLease tente d'acquérir le verrou d'écriture pour un chemin DB.
// Même sémantique que sync.AcquireLease.
func AcquireLease(path string, timeout time.Duration) (func(), error) {
    mu := leaseMutex(path)
    deadline := time.Now().Add(timeout)
    for {
        if mu.TryLock() {
            return func() { mu.Unlock() }, nil
        }
        if time.Now().After(deadline) {
            return nil, fmt.Errorf("write lease timeout (%v) pour %s", timeout, path)
        }
        time.Sleep(5 * time.Millisecond)
    }
}
```

**`internal/sync/lease.go` — déléguer vers `dblease` :**

```go
// AcquireLease délègue vers dblease.AcquireLease.
// Conservé pour ne pas casser les callers dans engine.go, backfill_weapons.go, etc.
func AcquireLease(path string, timeout time.Duration) (func(), error) {
    return dblease.AcquireLease(path, timeout)
}
```

**`persist_sink.go` — acquérir le lease dans `writeBattlePass` et `writeChallenges` :**

Sémantique fire-and-forget préservée : si le lease n'est pas disponible en 5 secondes,
on loggue un warning et on abandonne proprement (pas de retour d'erreur, la goroutine se
termine).

```go
import "levelup/go-api/internal/platform/dblease"

func (s *PersistSink) writeBattlePass(ctx context.Context, trackPath string, body []byte) error {
    // ... ouverture et écriture meta (pas de lease nécessaire : metadata.duckdb
    //     est partagé en RW mais les writes battlepass sont idempotents et rares)

    if s.PlayerPath == "" || s.XUID == "" {
        return nil
    }

    // Acquérir le lease player avant toute écriture dans stats.duckdb.
    releasePlayer, err := dblease.AcquireLease(s.PlayerPath, 5*time.Second)
    if err != nil {
        slog.Warn("persist_sink: timeout lease player DB — battlepass drop",
            "xuid", s.XUID, "path", s.PlayerPath)
        return nil   // fire-and-forget : on abandonne sans faire échouer la requête HTTP
    }
    defer releasePlayer()

    pdb, err := OpenReadWrite(s.PlayerPath)
    ...
}
```

Idem dans `writeChallenges`.

### Graphe d'imports après correction (pas de cycle)

```
internal/sync
  ├── internal/platform/duckdb   (pour OpenReadWrite dans schema.go)
  └── internal/platform/dblease  (pour AcquireLease dans lease.go)

internal/platform/duckdb
  └── internal/platform/dblease  (pour AcquireLease dans persist_sink.go)

internal/platform/dblease
  └── (stdlib uniquement)
```

### Fichiers impactés

| Fichier | Changement |
|---|---|
| `internal/platform/dblease/lease.go` *(nouveau)* | Extraction du mécanisme de lease |
| `internal/sync/lease.go` | Remplacer l'implémentation par une délégation vers `dblease` |
| `internal/platform/duckdb/persist_sink.go` | Importer `dblease`, ajouter `AcquireLease` dans `writeBattlePass` et `writeChallenges` |

---

## Problème 5 — Bug `writeChallenges` : `defer db.Close()` dans la branche erreur

### Localisation

`persist_sink.go` lignes 346–365

### Ce qui se passe concrètement

```go
if s.MetaPath != "" {
    db, err := OpenReadWrite(s.MetaPath)
    if err != nil {
        slog.Warn("persist_sink: open meta rw for challenges failed", "err", err)
        defer db.Close()   // ← LIGNE 350 : defer sur db nil/invalide dans le chemin erreur
    } else {
        defer db.Close()   // ← LIGNE 352 : defer correct dans le chemin succès
        ...
    }
}
```

Quand `err != nil`, `db` peut être nil (ou un `*DB` dans un état invalide). `(*DB).Close()`
gère `nil` (retourne nil sans panic), donc il n'y a pas de crash. Mais :

1. Sémantiquement, mettre un `defer` dans le chemin d'erreur est incorrect et trompeur.
2. La symétrie trompeuse `if err != nil { defer }` / `else { defer }` suggère une inversion
   logique à quelqu'un qui relit le code.

### Correction

Réécrire avec early return propre, qui est le pattern Go standard :

```go
if s.MetaPath != "" {
    if metaDB, metaErr := OpenReadWrite(s.MetaPath); metaErr != nil {
        slog.Warn("persist_sink: open meta rw for challenges failed", "err", metaErr)
    } else {
        defer metaDB.Close()
        if err := upsertWaypointAsset(ctx, metaDB, ...); err != nil {
            slog.Warn(...)
        }
    }
}
```

### Fichiers impactés

| Fichier | Changement |
|---|---|
| `internal/platform/duckdb/persist_sink.go` | Restructurer `writeChallenges` lignes 346–365 |

---

---

## Problème 6 — Timeouts de lease uniformes pour player et shared DB (Solution C de la discussion)

### Localisation

`internal/sync/engine.go` — constante unique `leaseTimeout = 10 * time.Second`

### Ce qui se passe concrètement

```go
const leaseTimeout = 10 * time.Second   // ← même durée pour les deux DBs
...
relPlayer, err := AcquireLease(e.playerDBPath, leaseTimeout)
...
relShared, err := AcquireLease(e.sharedDBPath, leaseTimeout)
```

La shared DB (`shared_matches_v2.duckdb`) est partagée par **tous les joueurs**. Avec
`maxParallel > 1` dans le `Coordinator`, plusieurs joueurs peuvent tenter d'acquérir la
shared DB simultanément. Si le joueur A a un sync long (500 matchs, médailles, participants),
son lease sur la shared peut prendre 30–60s. Le joueur B, qui tente d'acquérir la shared DB
10 secondes après, obtient un timeout et sa sync échoue **entièrement**.

La player DB (`stats.duckdb`) a un profil très différent : elle n'est écrite que par UN seul
joueur, les writes sont courts (player_match_enrichment, performance_score), et le Coordinator
garantit qu'un seul SyncEngine tient ce lease à la fois. 5s est amplement suffisant.

### Correction

Remplacer la constante unique par deux constantes distinctes :

```go
const (
    // playerLeaseTimeout — timeout pour stats.duckdb d'un joueur.
    // Court : la player DB n'est écrite que par un seul SyncEngine à la fois.
    playerLeaseTimeout = 5 * time.Second

    // sharedLeaseTimeout — timeout pour shared_matches_v2.duckdb.
    // Long : la shared DB est partagée par tous les joueurs ; un sync de 500 matchs
    // peut prendre 30–60s et le deuxième joueur doit pouvoir attendre.
    sharedLeaseTimeout = 45 * time.Second
)
```

**Mise à jour de tous les call-sites :**

| Appel | Constante à utiliser |
|---|---|
| `engine.go::run()` — `AcquireLease(e.playerDBPath, ...)` | `playerLeaseTimeout` |
| `engine.go::run()` — `AcquireLease(e.sharedDBPath, ...)` | `sharedLeaseTimeout` |
| `engine.go::RunBackfill` — `AcquireLease(e.playerDBPath, ...)` (Problème 2) | `playerLeaseTimeout` |
| `engine.go::RunBackfill` — `AcquireLease(e.sharedDBPath, ...)` (Problème 2) | `sharedLeaseTimeout` |
| `backfill_weapons.go` — `AcquireLease(e.sharedDBPath, ...)` (Problème 3) | `sharedLeaseTimeout` |
| `persist_sink.go` — `AcquireLease(s.PlayerPath, ...)` (Problème 4) | `playerLeaseTimeout` |

### Fichiers impactés

| Fichier | Changement |
|---|---|
| `internal/sync/engine.go` | Remplacer `leaseTimeout` par `playerLeaseTimeout` + `sharedLeaseTimeout` |
| `internal/sync/backfill_weapons.go` | Utiliser `sharedLeaseTimeout` (via délégation dblease) |
| `internal/platform/duckdb/persist_sink.go` | Utiliser `playerLeaseTimeout` (via `dblease.PlayerLeaseTimeout` — constante exportée) |

Note : les constantes `playerLeaseTimeout` / `sharedLeaseTimeout` restent dans
`engine.go`. Pour `persist_sink.go`, qui ne doit pas importer `internal/sync`,
on exporte deux constantes depuis `dblease` :

```go
// dblease/lease.go
const (
    PlayerLeaseTimeout = 5 * time.Second
    SharedLeaseTimeout = 45 * time.Second
)
```

---

## Problème 7 — `PersistSink` est recréé par requête HTTP, pas par joueur (Solution D de la discussion)

### Localisation

`internal/api/registry.go` lignes 227, 247 — `NewPersistSink` appelé à chaque requête

### Ce qui se passe concrètement

```go
// HomeCtxWithAuth — appelé à CHAQUE requête authentifiée
func (r *ServiceRegistry) HomeCtxWithAuth(ctx context.Context, slug string) (...) {
    pdb, _ := r.resolve(ctx, slug)
    sink := duckdb.NewPersistSink(pdb.Metadata.Path(), pdb.Player.Path(), pdb.XUID)  // ← nouveau sink par requête
    ...
}
```

Deux problèmes indépendants :

**Problème 7a — Goroutines orphelines sous forte concurrence**

Si 5 requêtes HTTP arrivent simultanément pour le même joueur (ex. home + season pass
+ live_refresh ticker), 5 `PersistSink` distincts sont créés, chacun capable de lancer ses
propres goroutines `go func()`. Résultat : jusqu'à 5 × 2 = 10 goroutines concurrentes
écrivent dans `stats.duckdb` du joueur, toutes voulant acquérir le même lease.

Avec la Solution A (Problème 4), le lease les sérialise — mais :
- 9 d'entre elles attendent dans une boucle de polling (5ms de sleep, `playerLeaseTimeout = 5s`
  → jusqu'à 1000 itérations par goroutine)
- Certaines vont timeout si le SyncEngine tient déjà le lease → drops silencieux

**Problème 7b — Aucun mécanisme de drain au shutdown**

`PersistSink.writeBattlePass` lance une goroutine détachée sans contexte d'annulation et
sans `sync.WaitGroup`. Au shutdown (`CloseAll()`), les goroutines en cours sont killées
si le process se termine avant qu'elles aient fini.

### Correction : déplacer PersistSink dans `PlayerDB`, une instance par joueur

**Étape 7.1 — Ajouter `persistJob` et le channel dans `PersistSink`**

```go
// persistJob est une tâche de persistance en queue.
type persistJob struct {
    kind      string // "battlepass" | "challenges"
    trackPath string
    rawBody   []byte
}

type PersistSink struct {
    MetaPath   string
    PlayerPath string
    XUID       string
    queue      chan persistJob    // buffered channel, capacité 16
    cancel     context.CancelFunc
    done       chan struct{}       // fermé quand le consumer s'est arrêté
}
```

**Étape 7.2 — `NewPersistSink` démarre le consumer**

```go
func NewPersistSink(metaPath, playerPath, xuid string) *PersistSink {
    ctx, cancel := context.WithCancel(context.Background())
    s := &PersistSink{
        MetaPath:   metaPath,
        PlayerPath: playerPath,
        XUID:       xuid,
        queue:      make(chan persistJob, 16),
        cancel:     cancel,
        done:       make(chan struct{}),
    }
    go s.consume(ctx)
    return s
}
```

**Étape 7.3 — `consume` est le seul goroutine à écrire dans player DB**

```go
func (s *PersistSink) consume(ctx context.Context) {
    defer close(s.done)
    for {
        select {
        case job, ok := <-s.queue:
            if !ok {
                return  // channel fermé → drain terminé
            }
            // Acquérir le lease AVANT chaque écriture pour ne pas entrer
            // en conflit avec SyncEngine.run().
            rel, err := dblease.AcquireLease(s.PlayerPath, dblease.PlayerLeaseTimeout)
            if err != nil {
                slog.Warn("persist_sink: timeout lease player DB — job drop",
                    "xuid", s.XUID, "kind", job.kind)
                continue
            }
            switch job.kind {
            case "battlepass":
                if err := s.writeBattlePass(ctx, job.trackPath, job.rawBody); err != nil {
                    slog.Warn("persist_sink: battlepass write failed", "xuid", s.XUID, "err", err)
                }
            case "challenges":
                if err := s.writeChallenges(ctx, job.rawBody); err != nil {
                    slog.Warn("persist_sink: challenges write failed", "xuid", s.XUID, "err", err)
                }
            }
            rel()
        case <-ctx.Done():
            return
        }
    }
}
```

**Étape 7.4 — `PersistBattlePass` et `PersistChallenges` envoient au channel (plus de `go func()`)**

```go
func (s *PersistSink) PersistBattlePass(trackPath string, rawBody []byte) {
    if s.MetaPath == "" || trackPath == "" || len(rawBody) == 0 {
        return
    }
    select {
    case s.queue <- persistJob{kind: "battlepass", trackPath: trackPath, rawBody: rawBody}:
    default:
        // Queue pleine (16 jobs) → drop. Acceptable : snapshot BP est idempotente.
        slog.Warn("persist_sink: queue pleine, battlepass drop", "xuid", s.XUID)
    }
}

func (s *PersistSink) PersistChallenges(rawBody []byte) {
    if s.PlayerPath == "" || len(rawBody) == 0 {
        return
    }
    select {
    case s.queue <- persistJob{kind: "challenges", rawBody: rawBody}:
    default:
        slog.Warn("persist_sink: queue pleine, challenges drop", "xuid", s.XUID)
    }
}
```

**Étape 7.5 — Ajouter `Close()` pour le drain propre**

```go
func (s *PersistSink) Close() {
    s.cancel()       // signale le consumer
    close(s.queue)   // débloque le select
    <-s.done         // attend que le consumer soit sorti
}
```

**Étape 7.6 — Stocker le `Sink` dans `PlayerDB` et fermer dans `CloseAll`**

```go
// pool.go — PlayerDB
type PlayerDB struct {
    Player       *DB
    Shared       *DB
    SharedSocial *DB
    Metadata     *DB
    Sink         *PersistSink   // ← ajout : une instance par joueur
    XUID         string
    Gamertag     string
    TitleSlug    string
}
```

```go
// pool.go — openPlayerDB
func openPlayerDB(ctx context.Context, cfg PlayerPoolConfig) (*PlayerDB, error) {
    ...
    pdb := &PlayerDB{...}
    pdb.Sink = NewPersistSink(cfg.MetaDBPath, cfg.PlayerDBPath, cfg.XUID)
    ...
}

// pool.go — CloseAll
func CloseAll() {
    globalPool.Range(func(key, value any) bool {
        pdb := value.(*PlayerDB)
        if pdb.Sink != nil {
            pdb.Sink.Close()   // drain + arrêt du consumer
        }
        _ = pdb.Player.Close()
        ...
        return true
    })
}
```

**Étape 7.7 — `registry.go` lit `pdb.Sink` au lieu de créer un nouveau sink**

```go
// AVANT
sink := duckdb.NewPersistSink(pdb.Metadata.Path(), pdb.Player.Path(), pdb.XUID)

// APRÈS
sink := pdb.Sink   // déjà initialisé dans openPlayerDB, partagé entre les requêtes
```

Idem dans `SeasonPassCtxWithAuth`. Le `watcher/live_refresh.go` conserve son champ
`sink *PersistSink` — il est initialisé depuis `pdb.Sink` dans `registry.go` ou dans
le code qui construit le watcher.

### Bénéfices de la solution complète

| Problème | Solution A seule | Solutions A + D |
|---|---|---|
| SyncEngine ↔ PersistSink race | ✅ lease sérialise | ✅ idem |
| 10 goroutines concurrentes / requête | ⚠️ sérialisées par lease mais 9 en attente | ✅ 1 seul consumer |
| Drops sous contention | ⚠️ si timeout (5s) dépassé | ✅ jobs queués, drops seulement si queue pleine (16 jobs) |
| Drain propre au shutdown | ❌ goroutines orphelines | ✅ `Close()` bloquant avec drain |
| Snapshot BF/challenges dans l'ordre | ❌ ordre aléatoire selon scheduling | ✅ FIFO strict |

### Fichiers impactés

| Fichier | Changement |
|---|---|
| `internal/platform/duckdb/persist_sink.go` | Réécriture complète : `persistJob`, channel, `consume`, `Close`, suppression des `go func()` |
| `internal/platform/duckdb/pool.go` | Ajout `Sink *PersistSink` dans `PlayerDB` ; init dans `openPlayerDB` ; `Sink.Close()` dans `CloseAll` |
| `internal/api/registry.go` | `HomeCtxWithAuth` + `SeasonPassCtxWithAuth` : utiliser `pdb.Sink` au lieu de `NewPersistSink` |

---

## Problème 8 — `defaultTokenReader` ouvre stats.duckdb sans le cache (Risque 4 de la discussion)

### Localisation

`internal/scheduler/auto_sync.go` ligne 350 — `defaultTokenReader`

### Ce qui se passe concrètement

```go
func defaultTokenReader(ctx context.Context, dbPath string, gamertag string, provider auth.TokenProvider) (string, error) {
    db, err := sql.Open("duckdb", dbPath)   // ← 3e connexion bare, bypasse le cache
    if err != nil {
        return "", err
    }
    db.SetMaxOpenConns(1)
    defer db.Close()   // ← ferme la connexion brute, indépendamment du cache
    ...
    // SELECT queries seulement
}
```

Cette connexion est :
1. Une 4e connexion distincte (après `openDBs["ro:stats.duckdb"]` du pool, `openDBs["rw:stats.duckdb"]` du SyncEngine via Problème 1, et `openDBs["ro:stats.duckdb"]` du career sync)
2. Ouverte sans `access_mode=read_only` → DuckDB l'interprète comme RW (même si on ne fait que des SELECT)

Atténuant : `ActivityChecker.IsPlayerActive` (ligne 208 de `auto_sync.go`) empêche l'auto-sync
pendant la présence active du joueur → en pratique, ce code ne s'exécute pas pendant un sync
actif du watcher. Mais :
- L'auto-sync programmé (ticker) peut coïncider avec un sync manuel déclenché via handler HTTP
- Le scheduler Go ne garantit pas l'ordre des goroutines

### Correction

Remplacer `sql.Open` par `duckdbpkg.OpenReadOnly` (passe par le cache, accès explicitement
read-only) :

```go
import duckdbpkg "levelup/go-api/internal/platform/duckdb"

func defaultTokenReader(...) (string, error) {
    dbHandle, err := duckdbpkg.OpenReadOnly(dbPath)
    if err != nil {
        return "", err
    }
    defer dbHandle.Close()   // décrémente le refcount, ne ferme pas si partagé
    db := dbHandle.SQLDb()
    ...
    // Les SELECT queries restent identiques, sur db (*sql.DB)
}
```

Import `"database/sql"` reste (pour `sql.Row`/`sql.ErrNoRows`), mais `sql.Open` est retiré.
Le `_ "github.com/duckdb/duckdb-go/v2"` dans `auto_sync.go` est retiré car le driver est
maintenant chargé via `duckdbpkg`.

### Fichiers impactés

| Fichier | Changement |
|---|---|
| `internal/scheduler/auto_sync.go` | Remplacer `sql.Open` + import `platform/duckdb`, retirer import driver |

---

## Ordre d'implémentation recommandé

L'ordre ci-dessous minimise les conflits d'édition et permet de valider étape par étape :

1. **Problème 4a / Problème 6** — Créer `internal/platform/dblease/lease.go` avec les
   constantes `PlayerLeaseTimeout` / `SharedLeaseTimeout` et l'implémentation `AcquireLease`

2. **Problème 4b** — `sync/lease.go` délègue vers `dblease`

3. **Problème 1** — `schema.go` : `OpenPlayerDB`/`OpenSharedDB` → `*duckdbpkg.DB` via cache

4. **Problème 2 + 6** — `engine.go` : leases dans `RunBackfill` + timeouts différenciés

5. **Problème 3 + 6** — `backfill_weapons.go` : lease + `sharedLeaseTimeout`

6. **Addendum metadata** — `main.go`, `pool.go`, `lab/provider.go`, `.air.toml`
    : metadata en read-only par défaut + `duckdb.CloseAll()` + hot-reload Air

7. **Problème 7** — `persist_sink.go` : réécriture channel queue
    + `pool.go` : `Sink` dans `PlayerDB` + `Close` dans `CloseAll`
   + `registry.go` : utiliser `pdb.Sink`

8. **Problème 5** — `persist_sink.go` : bug `defer` dans `writeChallenges` (trivial,
   à faire dans le même commit que l'étape 6)

9. **Problème 8** — `auto_sync.go` : `defaultTokenReader` → `OpenReadOnly`

10. **Problème 1 / tests** — `schema_integration_test.go` + `engine_e2e_test.go` :
   adapter `.SQLDb()`

---

## Récapitulatif global des fichiers

| Fichier | Nature du changement | Solutions couvertes |
|---|---|---|
| `internal/platform/dblease/lease.go` *(nouveau)* | Mécanisme de lease + constantes `PlayerLeaseTimeout`/`SharedLeaseTimeout` | A, C, D |
| `internal/sync/lease.go` | Délégation vers `dblease` — implémentation retirée | A |
| `internal/sync/schema.go` | `OpenPlayerDB`/`OpenSharedDB` retournent `*duckdbpkg.DB` via cache | B |
| `internal/sync/engine.go` | 4 call-sites adaptés + 2 leases dans `RunBackfill` + timeouts différenciés | B, C + Risque 2 |
| `internal/sync/backfill_weapons.go` | Lease `sharedLeaseTimeout` dans `BackfillWeaponKillsForMatches` | C + Risque 2 |
| `internal/sync/career.go` | `openCareerMetadataDB` → `duckdbpkg.OpenReadOnly` | B |
| `internal/platform/duckdb/persist_sink.go` | Réécriture : channel queue, `consume`, `Close`, suppression `go func()`, lease dans consumer, fix defer bug | A, D |
| `internal/platform/duckdb/pool.go` | `Metadata` en RO par défaut + `Sink *PersistSink` ; init + `Close` dans `CloseAll` | D + Addendum metadata |
| `internal/api/registry.go` | Utiliser `pdb.Sink` au lieu de `NewPersistSink` (×2) | D |
| `apps/go-api/cmd/server/main.go` | metadata runtime en RO + `duckdb.CloseAll()` au shutdown | Addendum metadata |
| `apps/go-api/internal/platform/lab/provider.go` | `OpenReadOnly` sur les parcours metadata de lecture | Addendum metadata |
| `apps/go-api/.air.toml` | `kill_delay` / `stop_timeout` ajustés | Addendum metadata |
| `internal/scheduler/auto_sync.go` | `defaultTokenReader` : `sql.Open` → `duckdbpkg.OpenReadOnly` | Risque 4 |
| `internal/sync/schema_integration_test.go` | `.SQLDb()` sur 2 tests | B |
| `internal/sync/engine_e2e_test.go` | `.SQLDb()` sur 2 usages | B |

**Total minimal révisé : 1 fichier créé, 14 fichiers modifiés (dont 2 tests, hors tests additionnels metadata/hot-reload).**

---

## Addendum metadata — durcir le plan pour les locks `metadata.duckdb`

Les problèmes 1 à 8 couvrent bien les conflits intra-process du sync. En revanche, les logs
observés sur Windows montrent un angle mort supplémentaire autour de `metadata.duckdb` :

- `cmd/server/main.go` ouvre metadata en read-write au runtime bootstrap, avec retry ;
- `pool.go::openPlayerDB()` ouvre aussi metadata en read-write pour chaque `PlayerDB` ;
- `lab/provider.go` l'ouvre également en read-write sur des parcours de lecture ;
- le shutdown du serveur ne draine pas explicitement tout le pool global avant qu'Air ne
    relance `tmp/server.exe`.

### Ce qu'il faut ajouter au plan

0. **Inventorier explicitement les writers metadata restants**

    Avant de basculer metadata en read-only par defaut, produire un inventaire simple des
    chemins qui ecrivent encore via `OpenReadWrite(...Meta`, `MetaDBPath`, `pdb.Metadata.Exec`
    ou toute ouverture metadata hors `internal/platform/duckdb`. L'objectif est de separer
    noir sur blanc les writers legitimes (migrations, seed, `PersistSink`, commandes ops)
    des lecteurs a rebasculer en RO.

1. **Passer metadata en read-only par défaut**

     - `cmd/server/main.go` : après migrations, ouvrir metadata en `OpenReadOnly(metaPath)` ;
     - `pool.go` : `PlayerDB.Metadata` doit être read-only par défaut ;
     - `lab/provider.go` : `GetResources()` et `loadMedalGuards()` doivent utiliser
         `OpenReadOnly(...)`.

2. **Traiter metadata comme une base partagée par titre, pas comme une ressource par joueur**

     Le consumer `PersistSink` ne doit pas seulement prendre un lease sur `PlayerPath`, mais
     aussi sur `MetaPath` avant les writes metadata. Sinon deux joueurs différents peuvent
     encore se faire concurrence sur la même metadata.

3. **Fermer explicitement le pool global au shutdown**

     `cmd/server/main.go` doit appeler `duckdb.CloseAll()` pour drainer `PlayerDB` et `Sink`
     avant la sortie du process.

    Ordre retenu au shutdown : `cancelScheduler` -> `watcherDaemon.Stop()` ->
    `srv.Shutdown(...)` -> `duckdb.CloseAll()` -> fermeture des handles racine `sharedDB`
    et `metaDB` du `main`.

4. **Inclure le hot-reload Air dans la validation du plan**

     - revoir `.air.toml` (`kill_delay`, `stop_timeout`) pour laisser à l'ancien process le
         temps de relâcher `metadata.duckdb` ;
     - valider localement sous `make go-api-dev` en déclenchant un rebuild Air pendant des
         appels Home / Season Pass ;
     - vérifier qu'on ne retrouve plus la boucle `metadata verrouillée` / `server.exe~`.

    Valeurs de depart retenues : `kill_delay = "3000ms"` et `stop_timeout = 5000`.

### Fichiers supplémentaires impactés

| Fichier | Changement |
|---|---|
| `apps/go-api/cmd/server/main.go` | metadata runtime en RO + `duckdb.CloseAll()` au shutdown |
| `apps/go-api/internal/platform/duckdb/pool.go` | `PlayerDB.Metadata` en RO par défaut |
| `apps/go-api/internal/platform/duckdb/persist_sink.go` | lease supplémentaire sur `MetaPath` |
| `apps/go-api/internal/platform/lab/provider.go` | `OpenReadOnly` sur les parcours metadata de lecture |
| `apps/go-api/.air.toml` | `kill_delay` / `stop_timeout` ajustés |

---

## Validation

```bash
cd apps/go-api

# 0. Pas de cycle d'import
go mod tidy
go build ./...

# 1. Tests unitaires dblease (rapides, stdlib pur)
go test ./internal/platform/dblease/...

# 2. Tests intégration sync
go test -tags=integration ./internal/sync/...

# 3. Tests intégration platform/duckdb
go test -tags=integration ./internal/platform/duckdb/...

# 4. Tests lease (délégation dblease)
go test -tags=integration ./internal/sync/ -run TestAcquireLease

# 5. Tests scheduler
go test ./internal/scheduler/...
```

Validation runtime spécifique metadata / hot-reload :

```bash
# terminal A
make go-api-dev

# terminal B
curl -s --max-time 8 "http://127.0.0.1:8000/api/v1/players/<slug>/pages/home"
curl -s --max-time 8 "http://127.0.0.1:8000/api/v1/players/<slug>/pages/palmares/season-pass"

# puis modifier un fichier Go pour forcer Air à redémarrer
# vérifier l'absence de boucle :
#   metadata verrouillée, nouvelle tentative...
#   ouverture metadata échouée après ...
# Répétition minimale retenue : 10 rebuilds consécutifs sans warning/error metadata.
```

---

## Points de vigilance

### `leaseMutex` : une seule map, dans `dblease` uniquement

Après extraction vers `dblease` et délégation depuis `sync/lease.go`, il ne doit exister
QU'UNE instance de la map `leases`. Vider complètement l'implémentation de `sync/lease.go`,
ne garder que la délégation.

### `metadata.duckdb` n'est plus un RW implicite du runtime

Avec l'addendum metadata :

- les lecteurs runtime doivent préférer `OpenReadOnly` ;
- `PersistSink` doit prendre un lease sur `MetaPath` pour ses writes metadata ;
- le shutdown doit drainer `duckdb.CloseAll()` avant sortie.

Le grep d'inventaire metadata fait partie de cette vérification ; ce n'est pas une note
optionnelle à traiter plus tard.

Ne pas garder l'hypothèse précédente selon laquelle le lease `PlayerPath` suffit et que
le RW metadata implicite du pool est acceptable.

### Timeout du consumer : si le SyncEngine hold le lease > 45s, le job est drop

C'est le comportement attendu : mieux vaut dropper une snapshot BP qu'un timeout de
45s sur la goroutine consumer. La prochaine requête HTTP ou le prochain tick du watcher
(5 minutes) re-fetchera les données.

### `PlayerDB.Sink` vs tests : `seedLegacyPlayerDB` ne crée pas de Sink

Dans `pool_migration_test.go`, `seedLegacyPlayerDB` crée une PlayerDB directement sans
passer par `openPlayerDB`. Il faudra initialiser `Sink` à nil dans les helpers de test
et ajouter un guard `if pdb.Sink != nil` dans `CloseAll`. Ou mieux : fournir un helper
`NewTestPlayerDB` dans les tests qui ne lance pas le consumer.

### `registry.go` : `NewPersistSink` n'est plus appelé, le symbol reste exporté

`NewPersistSink` reste public pour les tests qui créent un sink directement. Ne pas
le supprimer — adapter uniquement `registry.go`.

# Plan consolide : durcissement du sync multi-joueurs et des acces DuckDB

> Statut : plan consolide, non implemente
> Date : 2026-04-23
> Branche cible proposee : `copilot/fix-potential-duckdb-conflicts`

---

## 1. Objectif

Ce plan regroupe en une seule passe coherente les correctifs necessaires pour fiabiliser
la concurrence autour du sync Go quand plusieurs sources peuvent toucher les memes bases
DuckDB :

- sync manuel, watcher et auto-sync ;
- backfill classique et backfill weapons ;
- persistance live battle pass / challenges via `PersistSink` ;
- lectures auxiliaires de `stats.duckdb` et `metadata.duckdb` pendant le post-sync.

Le document remplace les versions precedentes qui melangeaient plusieurs variantes de
solution, des hypotheses contradictoires sur les signatures, puis un second chantier
adjacent sur la deduplication inter-joueurs et les logs. Ici, tout ce qui est retenu
est present dans un seul plan, et tout ce qui est ecarte est explicite.

---

## 2. Perimetre exact

Ce plan couvre volontairement deux familles de problemes qui viennent de la meme topologie
runtime :

1. **Conflits d'acces DuckDB**
   SyncEngine, backfill, scheduler et PersistSink ouvrent aujourd'hui plusieurs connexions
   concurrentes sur les memes fichiers, parfois hors cache process-level, parfois hors lease.

2. **Contention multi-joueurs sur la shared DB**
   Quand plusieurs joueurs jouent ensemble ou en parallele, le deuxieme sync peut soit
   attendre trop peu longtemps, soit refaire un travail deja fait pour la partie shared.

Le plan **inclut donc** :

- l'unification des ouvertures DB via le cache `internal/platform/duckdb` ;
- l'extraction du mecanisme de lease dans un package neutre ;
- une attente de lease pilotee par contexte pour les vrais flux de sync ;
- la protection des chemins qui ecrivent sans lease aujourd'hui ;
- la serialisation de `PersistSink` par joueur ;
- la serialisation des writes `metadata.duckdb` par chemin de DB partage ;
- la reduction des ouvertures `metadata.duckdb` en read-write au strict necessaire ;
- le durcissement du shutdown et du hot-reload dev autour de `metadata.duckdb` ;
- la deduction rapide des matchs deja presents dans `match_registry` ;
- la couverture de tests et les logs strictement necessaires a ces changements.

Le plan **n'inclut pas** le backlog plus large de logging/handlers deja isole dans
`.ai/PLAN_logging_tests_go_api.md` : ce backlog reste distinct car il depasse largement
la concurrence DuckDB et la chaine de sync elle-meme.

---

## 3. Etat actuel verifie dans le code

### 3.1 Topologie de concurrence actuelle

| Surface | Symbole actuel | Fichier | Fichier DuckDB touche | Etat actuel | Probleme |
|---|---|---|---|---|---|
| Ouverture player DB RW | `OpenPlayerDB` | `apps/go-api/internal/sync/schema.go` | `stats.duckdb` | `sql.Open("duckdb", path)` direct | contourne le cache ref-compte |
| Ouverture shared DB RW | `OpenSharedDB` | `apps/go-api/internal/sync/schema.go` | `shared_matches_v2.duckdb` | `sql.Open("duckdb", path)` direct | contourne le cache ref-compte |
| Sync principal | `run()` | `apps/go-api/internal/sync/engine.go` | player + shared | tient un lease, mais ouvre les DB via helpers bare | expose deux connexions physiques distinctes |
| Detection backfill | `RunBackfill()` | `apps/go-api/internal/sync/engine.go` | player + shared | n'acquiert aucun lease | lecture concurrente sur ecritures en cours |
| Backfill weapons | `BackfillWeaponKillsForMatches()` | `apps/go-api/internal/sync/backfill_weapons.go` | shared | n'acquiert aucun lease | ecrit dans `weapon_kills` et `match_registry` sans coordination |
| Persistance live | `PersistBattlePass()` / `PersistChallenges()` | `apps/go-api/internal/platform/duckdb/persist_sink.go` | metadata + player | `go func()` detachees, pas de lease player | writes concurrentes et ordre non garanti |
| Creation du sink | `NewPersistSink(...)` appele par requete | `apps/go-api/internal/api/registry.go` | player | nouveau sink a chaque requete | multiplication des goroutines et absence de drain |
| Bootstrap runtime metadata | `main()` | `apps/go-api/cmd/server/main.go` | metadata | `OpenReadWrite` + retry au demarrage | ouvre metadata en RW meme pour des lectures bootstrap et amplifie les overlaps hot-reload |
| Pool joueur metadata | `openPlayerDB()` | `apps/go-api/internal/platform/duckdb/pool.go` | metadata | `OpenReadWrite` par `PlayerDB` | base partagee tenue RW meme sur des flux majoritairement read-only |
| Lectures Lab | `GetResources()` / `loadMedalGuards()` | `apps/go-api/internal/platform/lab/provider.go` | metadata | `OpenReadWrite` | augmente la surface de verrouillage metadata pour des ecrans de lecture |
| Lecture token scheduler | `defaultTokenReader()` | `apps/go-api/internal/scheduler/auto_sync.go` | player | `sql.Open` direct | nouvelle connexion hors cache, en RW implicite |
| Lecture metadata carriere | `openCareerMetadataDB()` | `apps/go-api/internal/sync/career.go` | metadata | `sql.Open(...?access_mode=read_only)` | lecture hors cache process-level |
| Sync inter-joueurs d'un meme match | `processMatch()` + `MatchQueue.Enqueue()` | `apps/go-api/internal/sync/engine.go`, `apps/go-api/internal/watcher/match_queue.go` | shared + player | dedup par `gamertag:match_id` seulement | le meme match peut etre fetch/reinsere pour plusieurs joueurs |
| Migration shared dans le pool joueur | `openPlayerDB()` | `apps/go-api/internal/platform/duckdb/pool.go` | `shared_matches_v2.duckdb` | `OpenReadWrite` temporaire pour `migration.RunForDB(rwShared...)` | cree une cle `rw:shared_path` dans le cache pendant que `run()` tient le lease shared — contention silencieuse non couverte par les autres correctifs |
| Migration shared_social dans le pool joueur | `openPlayerDB()` | `apps/go-api/internal/platform/duckdb/pool.go` | `shared_social.duckdb` | `OpenReadWrite` + `migration.RunForDB(socialDB...)` sans lease | ecritures sans coordination si un sync concurrent touche social |

### 3.2 Ce que le code fait deja correctement

Le point de depart n'est pas catastrophique. Le code actuel a deja plusieurs garde-fous utiles :

- `Coordinator` empeche deux syncs simultanes pour un meme `gamertag` via `inFlight` ;
- `MatchQueue` dedupe correctement **intra-joueur** avec la cle `gamertag + ":" + match_id` ;
- `pool.go` ouvre deja `stats.duckdb`, `metadata.duckdb` et `shared_matches_v2.duckdb`
  via le cache process-level de `internal/platform/duckdb/db.go` ;
- `PersistSink` utilise deja `OpenReadWrite`, donc il profite du cache ref-compte ;
- les ecritures shared utilisent des UPSERT / insertions idempotentes cote sync ;
- `pdb.Shared` est ouvert en `OpenReadOnly` avec `maxOpenConns=4` : les requetes HTTP
  qui lisent la shared DB sont non bloquantes entre elles et non bloquees par un writer
  concurrent — c'est deja le bon modele.

#### Note sur le modele de concurrence de DuckDB WAL

DuckDB v1.x en mode fichier local utilise un WAL (Write-Ahead Log). Ce modele garantit :

- **Un seul writer physique a la fois** par fichier (au niveau OS) ;
- **N lecteurs simultanes** sur le meme fichier dans le meme process, via des cles
  distinctes dans `openDBs` : `ro:path` et `rw:path` coexistent sans conflit ;
- **Les lecteurs RO ne sont pas bloques par un write actif** : ils voient un snapshot
  coherent des donnees committees avant l'ouverture de leur transaction ;
- **Entre process distincts** : un seul process peut tenir un handle RW. Si l'ancien
  `server.exe~` (Air hot-reload) est encore en vie, le nouveau process ne peut pas ouvrir
  le fichier en RW — c'est l'origine des warnings `metadata verrouillee` observes en dev.

Consequence directe pour ce plan : la bonne strategie pour eviter le blocage HTTP pendant
un sync n'est pas un orchestrateur global, mais d'**exploiter les deux cles distinctes**
du cache (`ro:` pour les lecteurs, `rw:` pour les writers). DuckDB gere leur coexistence
nativement via le WAL.

Le probleme n'est donc pas l'absence totale de protection. Le probleme est que plusieurs
surfaces importantes n'emploient pas **la meme** protection ni **la meme** abstraction.

### 3.3 Causes racines retenues

#### Cause racine A — double connexion physique sur le meme fichier

Le package `internal/platform/duckdb/db.go` maintient une map `openDBs` ref-comptee. Un
appel `OpenReadWrite(path)` renvoie la meme instance `*duckdb.DB` pour la cle `rw:<path>`.

Mais `apps/go-api/internal/sync/schema.go` ouvre encore :

```go
db, err := sql.Open("duckdb", path)
```

Donc, pendant un sync, on peut avoir :

- une connexion `stats.duckdb` deja ouverte par le pool joueur ;
- une seconde connexion `stats.duckdb` ouverte par `SyncEngine.run()` ;
- une troisieme connexion `stats.duckdb` ouverte par `defaultTokenReader()` ;
- une connexion `metadata.duckdb` ouverte hors cache par `openCareerMetadataDB()`.

Cette divergence suffit a reintroduire les conflits que le cache etait justement cense
eliminer.

#### Cause racine B — chemins d'ecriture qui ne tiennent pas tous le meme lease

`run()` tient un lease player et un lease shared, mais :

- `RunBackfill()` n'en tient aucun ;
- `BackfillWeaponKillsForMatches()` n'en tient aucun ;
- `PersistSink` n'en tient aucun pour `stats.duckdb` ;
- les lecteurs auxiliaires hors cache ne respectent pas non plus cette topologie.

Le resultat n'est pas seulement un risque d'erreur DuckDB ; c'est aussi un ordre de write
indetermine entre operations qui devraient etre serializees par fichier.

Note importante sur la race intra-handle : `PersistSink` ouvre `OpenReadWrite(s.PlayerPath)`,
ce qui passe par le cache et retourne la meme `*sql.DB` que le pool (`maxOpenConns=1`).
Meme sans double connexion physique, deux goroutines ecrivant sur le meme `*sql.DB`
avec `maxOpenConns=1` peuvent s'intercaler entre les Exec d'un batch : un
`INSERT INTO battlepass_snapshots` peut se glisser entre deux appels de `InsertRegistryIfNotExists`,
rompant l'atomicite du batch. C'est pourquoi un simple mutex externe ne suffit pas —
il faut un consumer unique par joueur (etape 7).

#### Cause racine C — attente de shared DB non adaptee a des syncs longs

Le `leaseTimeout = 10 * time.Second` actuel dans `engine.go` est uniforme. Or la shared DB
peut etre tenue pendant plusieurs minutes si un joueur a beaucoup de matchs a synchroniser.

Avec un timeout fixe court, le deuxieme joueur ne rencontre pas un vrai probleme metier ;
il rencontre un echec purement technique de contention.

#### Cause racine D — `PersistSink` a une granularite de vie trop fine

`registry.go` cree aujourd'hui un nouveau `PersistSink` par requete HTTP authentifiee :

- `HomeCtxWithAuth()` ;
- `SeasonPassCtxWithAuth()`.

Chaque sink peut ensuite lancer ses propres goroutines detachees. Sous charge,
plusieurs sinks du meme joueur peuvent donc se faire concurrence entre eux, meme si le
probleme `SyncEngine <-> PersistSink` etait deja resolu.

#### Cause racine E — absence de deduplication inter-joueurs dans la partie shared

Le watcher dedupe par joueur, pas globalement. Si deux joueurs ont joue ensemble :

- chaque watcher enfile sa propre requete ;
- le coordinator les traite separement ;
- `processMatch()` appelle `GetMatchStats(matchID)` dans les deux syncs ;
- les insertions shared restent idempotentes, mais l'appel API et le travail d'extraction
  sont refaits pour rien.

Cette partie n'est pas un conflit DuckDB a proprement parler, mais elle est directement
liee a la meme contention sur la shared DB et au meme scenario multi-joueurs qui a motive
le reste du plan. Elle est donc integree au document consolide.

#### Cause racine F — `metadata.duckdb` reste traitee comme un accessoire par joueur

`metadata.duckdb` est une base partagee au niveau du titre, pas au niveau du joueur.
Pourtant, l'etat courant la traite encore souvent comme une ressource annexee au `PlayerDB`
ou au handler courant :

- `pool.go` l'ouvre en read-write pour chaque `PlayerDB` ;
- `main.go` l'ouvre en read-write pour le runtime bootstrap ;
- `lab/provider.go` l'ouvre egalement en read-write sur des parcours de lecture ;
- `PersistSink` peut encore l'ecrire depuis plusieurs joueurs, alors que le lease actuel
  ne protege que `PlayerPath`.

Le resultat est une base partagee sur-ouverte en RW, avec une discipline de concurrence
encore centree sur le joueur alors que la bonne granularite est le chemin metadata.

#### Cause racine G — le cycle de vie process / hot-reload n'est pas explicite dans le plan initial

Le shutdown du serveur ferme aujourd'hui des handles racine, mais pas explicitement tout le
pool global des `PlayerDB`. En parallele, Air peut relancer rapidement `tmp/server.exe`
alors que l'ancien process n'a pas encore libere `metadata.duckdb`.

Ce n'est pas un simple detail d'outillage : tant que `server.exe~` peut rester en course
avec le nouveau process, un plan limite au seul intra-process laisse subsister une classe
entiere de warnings `metadata verrouillee` pendant le dev local.

#### Cause racine H — `pdb.Player` en RW tenu par le sync bloque les lectures HTTP

`pdb.Player` est ouvert en `OpenReadWrite` avec `maxOpenConns=1`. Tous les consommateurs
du handle — le sync (`run()`) et les services HTTP (Home, palmares, stats) — passent par
la **meme `*sql.DB`**. Le pool Go (`database/sql`) sérialise les connexions : si `run()` tient
la connexion pendant plusieurs minutes, toute requete HTTP qui tente de lire `stats.duckdb`
est bloquee dans la queue Go jusqu'a la liberation du handle.

Ce n'est pas un conflit DuckDB a proprement parler (pas de corruption), mais c'est une
**degradation de latence HTTP directement correctable** : les lecteurs n'ont pas besoin
d'un handle RW. DuckDB WAL permet un handle `ro:stats.duckdb` independant du `rw:stats.duckdb`
du sync, sans aucune contention entre eux.

Nota bene : le plan actuel corrige les races entre writers et previent la corruption. Sans
cette cause racine H, il ne corrige pas le blocage HTTP pendant un sync long. En usage
solo/duo, l'impact est faible (syncs courts). Avec plusieurs joueurs dont les syncs peuvent
durer plusieurs minutes, la page Home peut timeout.

---

## 4. Decisions retenues

Cette section fige les choix qui restaient contradictoires dans les versions precedentes.

### 4.1 Le package proprietaire du lease devient `internal/platform/dblease`

Le mecanisme de lease ne doit plus vivre dans `internal/sync`, car `PersistSink` doit
egalement s'en servir sans creer de cycle `sync -> platform/duckdb -> sync`.

Decision retenue :

- creer `apps/go-api/internal/platform/dblease/lease.go` ;
- y deplacer la map `leases`, `leaseMutex`, `AcquireLease(...)` ;
- y ajouter `AcquireLeaseCtx(ctx, path)` ;
- conserver `apps/go-api/internal/sync/lease.go` comme simple facade de compatibilite.

### 4.2 Les vrais flux de sync attendent le lease via le contexte, pas via un timeout fixe

Les variantes precedentes hesitaient entre :

- deux timeouts fixes differents (`player = 5/15s`, `shared = 45/60s`) ;
- ou une attente pilotee par `ctx.Done()`.

Decision retenue :

- `run()`, `RunBackfill()` et `BackfillWeaponKillsForMatches()` utilisent
  `AcquireLeaseCtx(ctx, path)` ;
- `AcquireLease(path, timeout)` reste present uniquement pour les chemins best-effort
  comme `PersistSink` et pour les tests a borne dure.

Raison : un sync shared peut legitiment durer plusieurs minutes. Un timeout fixe reste
donc une fausse protection. Le vrai garde-fou doit etre le contexte de la requete ou du
worker appelant.

### 4.3 Les helpers de `schema.go` redeviennent l'abstraction proprietaire de l'ouverture sync

Une autre contradiction apparue dans les brouillons etait la suivante :

- soit modifier `engine.go` pour contourner `OpenPlayerDB()` / `OpenSharedDB()` ;
- soit corriger directement ces helpers.

Decision retenue :

- corriger **les helpers eux-memes** ;
- `OpenPlayerDB(path)` et `OpenSharedDB(path)` retourneront `*duckdbpkg.DB` ;
- les call-sites utiliseront ensuite `handle.SQLDb()` lorsqu'une fonction attend encore
  `*sql.DB`.

Raison : ce sont les helpers proprietaires du package `sync`. Corriger seulement les
call-sites laisserait le mauvais comportement en place pour tout futur appelant.

### 4.4 `PersistSink` devient un writer queue par joueur

Les brouillons precedents proposaient soit :

- un simple lease dans `writeBattlePass()` / `writeChallenges()` ;
- soit une vraie file FIFO par joueur.

Decision retenue :

- retenir la version robuste, avec **un seul consumer par joueur** ;
- serialiser en plus les writes metadata par `MetaPath`, car `metadata.duckdb` est partagee
  entre tous les joueurs d'un titre ;
- stocker le sink dans `PlayerDB` ;
- faire en sorte que les appels HTTP ne creent plus de sink ephemere.

Raison : un lease seul supprime la collision DB, mais ne supprime pas les courses entre
plusieurs goroutines detachees ni l'absence de drain propre au shutdown. Et un consumer
par joueur ne suffit pas a lui seul a proteger une metadata commune a plusieurs joueurs.

### 4.5 La deduplication inter-joueurs est faite dans `processMatch()`, pas dans `MatchQueue`

Decision retenue :

- on ne modifie pas la cle de dedup de `MatchQueue` ;
- on ajoute un fast path dans `processMatch()` via `matchExistsInRegistry(sharedDB, matchID)` ;
- si le match est deja present en shared, on ecrit seulement l'enrichissement player.

Raison : un match deja present en shared peut etre nouveau pour un joueur donne. Il ne faut
donc pas le supprimer trop tot dans la file d'attente ; il faut seulement eviter de refaire
la partie shared du travail.

### 4.6 Les lecteurs auxiliaires passent eux aussi par le cache process-level

Decision retenue :

- `openCareerMetadataDB()` devient une ouverture read-only via `duckdbpkg.OpenReadOnly()` ;
- `defaultTokenReader()` fait de meme pour `stats.duckdb`.

Raison : garder un seul modele mental : toute ouverture DuckDB process-level passe par
`internal/platform/duckdb` sauf raison exceptionnelle explicite.

### 4.7 `metadata.duckdb` est serialisee par chemin partage, pas par joueur

Decision retenue :

- tout write `metadata.duckdb` pris en charge par `PersistSink` acquiert un lease sur
  `MetaPath` ;
- le lease player ne suffit pas a lui seul a proteger les writes metadata ;
- le plan n'accepte pas qu'un joueur A et un joueur B puissent ecrire `metadata.duckdb`
  en parallele simplement parce qu'ils ont deux queues distinctes.

Raison : la granularite de protection doit suivre la granularite reelle du fichier.

### 4.8 `metadata.duckdb` passe en read-only par defaut

Decision retenue :

- `cmd/server/main.go` ouvre metadata en read-only apres les migrations ;
- `pool.go` ouvre `PlayerDB.Metadata` en read-only par defaut ;
- `lab/provider.go` bascule en read-only sur les chemins purement consultatifs ;
- les ouvertures read-write restent reservees aux writers explicites : migrations, seed,
  `PersistSink` et autres commandes operationnelles equivalentes.

Raison : la meilleure maniere de reduire les conflits metadata reste de diminuer la surface
RW avant meme de discuter de mutex ou de retry.

### 4.9 Le shutdown et le hot-reload dev font partie du plan

Decision retenue :

- le shutdown du serveur doit appeler `duckdb.CloseAll()` pour drainer `PlayerDB` et
  fermer leurs handles avant la sortie du process ;
- la config Air (`kill_delay`, `stop_timeout`) fait partie des ajustements legitimes du plan ;
- la validation finale inclut un scenario de hot-reload local pendant des appels Home /
  Season Pass pour verifier la disparition des warnings metadata lies a `server.exe~`.

Raison : le symptome utilisateur observe sur metadata se produit justement dans ce cycle de
vie, donc l'acceptation du correctif doit le couvrir explicitement.

### 4.10 `PlayerDB` expose un handle player distinct en lecture seule pour les services HTTP

Decision retenue :

- `PlayerDB` gagne un champ `PlayerRO *DB` ouvert via `OpenReadOnly(cfg.PlayerDBPath)` ;
- les services HTTP (Home, palmares, stats, etc.) recus via `registry.go` utilisent
  `pdb.PlayerRO.SQLDb()` pour leurs requetes de lecture ;
- `pdb.Player` (RW, `maxOpenConns=1`) reste reserve au sync, au backfill et a `PersistSink`
  consumer ;
- `CloseAll()` ferme `pdb.PlayerRO` apres `pdb.Sink.Close()` et `pdb.Player.Close()`.

Raison : DuckDB WAL gere nativement la coexistence d'une cle `ro:path` et d'une cle
`rw:path` sur le meme fichier. Les lecteurs RO ne sont pas bloques par un write actif,
ils voient un snapshot coherent des donnees committees. Ce modele est deja en place sur
`pdb.Shared` (`OpenReadOnly`, `maxOpenConns=4`) ; il est coherent de l'appliquer egalement
a `pdb.Player`.

Consequence : une page Home appelee pendant un sync de plusieurs minutes repond sans
attendre la fin du sync.

---

## 5. Plan d'implementation detaille

### Etape 1 — Creer `internal/platform/dblease/lease.go`

#### Fichier cree

- `apps/go-api/internal/platform/dblease/lease.go`

#### Changement

Y deplacer l'implementation actuelle de `apps/go-api/internal/sync/lease.go` et y ajouter
la variante contextuelle.

Interface retenue :

```go
package dblease

const PlayerLeaseTimeout   = 5 * time.Second
const MetadataLeaseTimeout = 10 * time.Second
const SharedLeaseTimeout   = 45 * time.Second // best-effort / tests à borne dure sur shared DB

func AcquireLease(path string, timeout time.Duration) (func(), error)
func AcquireLeaseCtx(ctx context.Context, path string) (func(), error)
```

`SharedLeaseTimeout` est exporté pour les chemins qui ne peuvent pas utiliser un contexte
(tests à borne dure, futurs backfills autonomes). Il n'est plus utilisé dans `run()` ni
`RunBackfill` ni `BackfillWeaponKillsForMatches` qui passent a `AcquireLeaseCtx`, mais son
absence laisserait un trou d'API.

Implementation attendue :

- map globale `leases map[string]*sync.Mutex` ;
- `TryLock()` + polling ;
- ticker `5ms` dans `AcquireLeaseCtx` ;
- pas de `time.After(...)` recree dans la boucle ;
- erreur claire en timeout ;
- erreur claire si `ctx.Done()` se ferme pendant l'attente.

#### Pourquoi cette etape est prealable

Sans package neutre, `PersistSink` ne peut pas partager le meme lease que `SyncEngine`
sans recreer un cycle d'import.

#### Tests attendus

Deux options sont acceptables, mais la version retenue ici est la plus claire :

- garder `apps/go-api/internal/sync/lease_test.go` pour la facade `sync` ;
- ajouter `apps/go-api/internal/platform/dblease/lease_test.go` pour le coeur du package.

Les tests minimaux a couvrir :

- acquisition simple ;
- timeout borne ;
- chemins differents non bloquants ;
- pas de fuite de goroutines sur timeout ;
- `AcquireLeaseCtx` avec contexte deja annule ;
- `AcquireLeaseCtx` annule pendant l'attente ;
- reacquisition apres release.

### Etape 2 — Faire de `internal/sync/lease.go` une facade de compatibilite

#### Fichier modifie

- `apps/go-api/internal/sync/lease.go`

#### Changement

Le fichier ne doit plus contenir de map `leases` locale. Il doit seulement deleguer :

```go
func AcquireLease(path string, timeout time.Duration) (func(), error) {
    return dblease.AcquireLease(path, timeout)
}

func AcquireLeaseCtx(ctx context.Context, path string) (func(), error) {
    return dblease.AcquireLeaseCtx(ctx, path)
}
```

#### Point de vigilance

Il doit rester **une seule** map de mutex dans tout le process. La duplication de la map
entre `sync` et `dblease` casserait tout le but de l'extraction.

#### Graphe d'import resolu (pas de cycle)

```
internal/sync
  ├── internal/platform/duckdb   (pour OpenReadWrite dans schema.go)
  └── internal/platform/dblease  (pour AcquireLease dans lease.go)

internal/platform/duckdb
  └── internal/platform/dblease  (pour AcquireLease dans persist_sink.go)

internal/platform/dblease
  └── (stdlib uniquement)
```

### Etape 3 — Rebrancher `schema.go` sur le cache DuckDB process-level

#### Fichiers modifies

- `apps/go-api/internal/sync/schema.go`
- `apps/go-api/internal/sync/engine.go`
- `apps/go-api/internal/sync/backfill_weapons.go`
- `apps/go-api/internal/sync/schema_integration_test.go`
- `apps/go-api/internal/sync/engine_e2e_test.go`

#### Changement dans `schema.go`

`OpenPlayerDB()` et `OpenSharedDB()` deviennent les wrappers officiels sur les handles
ref-comptes du package `duckdb` :

```go
func OpenPlayerDB(path string) (*duckdbpkg.DB, error)
func OpenSharedDB(path string) (*duckdbpkg.DB, error)
```

Le flux attendu est :

1. `MkdirAll(filepath.Dir(path))` ;
2. `duckdbpkg.OpenReadWrite(path)` ;
3. `EnsurePlayerSchema(handle.SQLDb())` ou `EnsureSharedSchema(handle.SQLDb())` ;
4. `return handle`.

`db.SetMaxOpenConns(1)` et `db.SetMaxIdleConns(1)` ne sont plus du ressort de `schema.go`,
car `OpenReadWrite()` les applique deja.

#### Changement dans `engine.go` et `backfill_weapons.go`

Les call-sites doivent suivre le pattern :

```go
playerHandle, err := OpenPlayerDB(e.playerDBPath)
if err != nil { ... }
defer playerHandle.Close()
playerDB := playerHandle.SQLDb()
```

Idem pour la shared DB.

Les call-sites verifies aujourd'hui sont :

- `RunBackfill()` : 2 ouvertures ;
- `run()` : 2 ouvertures ;
- `BackfillWeaponKillsForMatches()` : 1 ouverture shared.

#### Correction de la collision `ro:` / `rw:` sur la shared DB dans `openPlayerDB`

Dans `openPlayerDB`, `shared_matches_v2.duckdb` est ouvert en read-only (`OpenReadOnly` → clé
`ro:path`) puis aussitot en read-write pour les migrations (`OpenReadWrite` → clé `rw:path`).
Le cache process-level traite ces deux clés comme **independantes**, ce qui peut créer deux
handles physiques simultanés sur le même fichier.

La correction fait partie du scope de cette étape :

- les migrations shared (`migration.RunForDB`) ne doivent plus être lancees depuis
  `openPlayerDB()` : cette responsabilite appartient a `runMigrations()` dans `main.go`,
  qui execute deja les migrations shared avant que tout `PlayerDB` soit ouvert ;
- le bloc `if rwShared, rwErr := OpenReadWrite(cfg.SharedDBPath); rwErr == nil { ... }` est
  supprime de `openPlayerDB()` ;
- meme correction pour `socialDB` si les migrations `TargetSharedSocial` sont deja
  couvertes par `runMigrations()`.

Si un chemin de migration social n'est pas encore dans `runMigrations()`, l'ajouter la
plutot que de conserver un `OpenReadWrite` ad hoc dans `openPlayerDB()`.

#### Changement dans les tests sync

`schema_integration_test.go` et `engine_e2e_test.go` utilisent aujourd'hui le retour
`*sql.DB` des helpers. Ils devront :

- recevoir un handle `*duckdb.DB` ;
- passer `handle.SQLDb()` aux helpers de verification qui attendent `*sql.DB` ;
- fermer le handle via `defer handle.Close()`.

### Etape 3-bis — Ajouter `PlayerRO` dans `PlayerDB` pour isoler les lectures HTTP

#### Fichiers modifies

- `apps/go-api/internal/platform/duckdb/pool.go`
- `apps/go-api/internal/api/registry.go`
- `apps/go-api/internal/api/` (services de lecture : home, palmares, stats, etc.)

#### Changement dans `pool.go`

`PlayerDB` gagne un champ supplementaire :

```go
type PlayerDB struct {
    Player       *DB  // stats.duckdb — RW, max 1 connexion, reserve au sync + PersistSink
    PlayerRO     *DB  // stats.duckdb — RO, max 4 connexions, pour les lectures HTTP
    Shared       *DB  // shared_matches_v2.duckdb — RO
    SharedSocial *DB  // shared_social.duckdb
    Metadata     *DB  // metadata.duckdb — RO apres migration (4.8)
    Sink         *PersistSink
    ...
}
```

Dans `openPlayerDB()`, apres l'ouverture du handle RW :

```go
playerRO, err := OpenReadOnly(cfg.PlayerDBPath)
if err != nil {
    _ = playerDB.Close()
    return nil, fmt.Errorf("pool: open player db read-only %s: %w", cfg.Gamertag, err)
}
```

Dans `CloseAll()`, fermer `pdb.PlayerRO` apres `pdb.Sink.Close()` et `pdb.Player.Close()`.

#### Changement dans `registry.go` et les services

Les services de lecture (repos qui font uniquement des SELECT sur `stats.duckdb`) doivent
recevoir `pdb.PlayerRO.SQLDb()` au lieu de `pdb.Player.SQLDb()`.

Les writers (sync, backfill, PersistSink consumer) conservent `pdb.Player.SQLDb()`.

#### Garantie DuckDB

DuckDB WAL assure que `ro:stats.duckdb` et `rw:stats.duckdb` coexistent sans corruption.
Les lectures RO voient les donnees committees au moment de l'ouverture de leur transaction.
Elles ne voient pas les ecritures en cours d'un batch non committe — ce qui est le
comportement attendu pour une page de dashboard.

#### Effet observable

Une requete HTTP Home appellee pendant un sync de plusieurs minutes retourne des donnees
(snapshot coherent) sans attendre la fin du sync. La latence HTTP n'est plus conditionnee
par la duree du sync.

#### Point de vigilance

Le lease applicatif (etape 4) ne protege que la coordination writer↔writer. Il n'y a pas de
lease a acquerir pour les lecteurs RO : le WAL DuckDB est le seul mecanisme necessaire.
Ne pas ajouter de lease sur `PlayerRO` — ce serait sur-ingenierie et recreerait exactement
le probleme de blocage que cette etape resout.

---

### Etape 4 — Passer les vrais flux de sync sur `AcquireLeaseCtx`

#### Fichiers modifies

- `apps/go-api/internal/sync/engine.go`
- `apps/go-api/internal/sync/backfill_weapons.go`

#### Changement dans `run()`

Les acquisitions actuelles :

```go
relPlayer, err := AcquireLease(e.playerDBPath, leaseTimeout)
relShared, err := AcquireLease(e.sharedDBPath, leaseTimeout)
```

deviennent :

```go
relPlayer, err := AcquireLeaseCtx(ctx, e.playerDBPath)
relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)
```

La constante `leaseTimeout` disparait du chemin principal de sync.

**Point de vigilance** : verifier qu'aucun test ne reference `leaseTimeout` directement
avant de la supprimer ou de la limiter aux chemins best-effort. Si des tests unitaires
utilisent cette constante comme valeur attendue, ils doivent être mis a jour pour utiliser
les valeurs de `dblease.PlayerLeaseTimeout` ou `dblease.MetadataLeaseTimeout`.

#### Changement dans `RunBackfill()`

`RunBackfill()` ne doit plus ignorer `ctx`. Il doit :

1. attendre le lease player via `AcquireLeaseCtx(ctx, e.playerDBPath)` ;
2. attendre le lease shared via `AcquireLeaseCtx(ctx, e.sharedDBPath)` ;
3. seulement ensuite ouvrir les DB et lancer `FindMatchesMissingData(...)`.

#### Changement dans `BackfillWeaponKillsForMatches()`

Avant d'ouvrir la shared DB, la methode doit attendre :

```go
relShared, err := AcquireLeaseCtx(ctx, e.sharedDBPath)
```

Cela aligne enfin le backfill weapons sur la meme discipline d'acces que le sync principal.

#### Pourquoi la variante contextuelle est retenue ici

Le but est de laisser la requete ou le worker appelant choisir sa vraie borne de temps.

Exemples :

- un handler HTTP peut utiliser un contexte borne a 10 minutes ;
- un watcher peut utiliser un contexte de daemon plus long ;
- un test peut utiliser `context.WithTimeout(...)` pour forcer un scenario d'attente.

#### Logs a adapter

Quand `AcquireLeaseCtx` remplace un timeout fixe :

- on conserve les logs de debut d'acquisition ;
- on retire tout champ obsolescent du type `"timeout", leaseTimeout` ;
- on loggue seulement le `db`, le `gamertag` et `err`.

### Etape 5 — Normaliser les lecteurs auxiliaires hors sync principal

#### Fichiers modifies

- `apps/go-api/internal/sync/career.go`
- `apps/go-api/internal/scheduler/auto_sync.go`
- `apps/go-api/internal/sync/engine.go`

#### `openCareerMetadataDB()`

Aujourd'hui, `career.go` ouvre `metadata.duckdb` en read-only via `sql.Open`. Le helper doit
utiliser le cache read-only du package DuckDB.

Deux implementations sont possibles ; la variante retenue est la plus coherente avec
`schema.go` :

```go
func openCareerMetadataDB(path string) (*duckdbpkg.DB, error)
```

Puis dans `runCareerSync()` :

```go
metaHandle, err := openCareerMetadataDB(e.metadataDBPath)
defer metaHandle.Close()
if err := enrichCareerRankFromMetadata(metaHandle.SQLDb(), data); err != nil { ... }
```

#### `defaultTokenReader()`

Le scheduler ne doit plus ouvrir `stats.duckdb` via `sql.Open("duckdb", dbPath)`.
Il doit faire :

```go
dbHandle, err := duckdbpkg.OpenReadOnly(dbPath)
defer dbHandle.Close()
db := dbHandle.SQLDb()
```

Le package garde `database/sql` pour `sql.ErrNoRows` et les scans, mais n'utilise plus
`sql.Open` directement.

Le blank import du driver DuckDB dans `auto_sync.go` doit etre retire si ce fichier ne
contient plus aucun `sql.Open("duckdb", ...)` apres la migration.

#### Raison de cette etape

Sans cela, le plan resterait incomplet : meme apres correction du SyncEngine, il resterait
des connexions bare sur `stats.duckdb` et `metadata.duckdb`.

### Etape 6 — Durcir `metadata.duckdb` comme ressource partagee globale

#### Fichiers modifies

- `apps/go-api/cmd/server/main.go`
- `apps/go-api/internal/platform/duckdb/pool.go`
- `apps/go-api/internal/platform/lab/provider.go`
- `apps/go-api/.air.toml`

#### Inventaire prealable des writers metadata

Le basculement de `metadata.duckdb` en read-only par defaut ne doit pas se faire a
l'aveugle. Avant le premier edit runtime, il faut produire un inventaire simple des
surfaces qui ecrivent encore metadata, classees en deux groupes :

- **writers legitimes a conserver en RW** : migrations, seed, `PersistSink`, commandes
  operationnelles explicites ;
- **lecteurs a rebasculer en RO** : runtime bootstrap, pool joueur, lab, autres helpers
  consultatifs.

Le controle minimal attendu est une recherche ciblee sur :

- `OpenReadWrite(...Meta` / `OpenReadWrite(metaPath` / `MetaDBPath` ;
- `pdb.Metadata.Exec` ;
- tout appel applicatif qui ouvre metadata hors `internal/platform/duckdb`.

Le plan n'est considere complet sur metadata que si cet inventaire est capture dans le diff
ou dans les notes de validation du chantier.

#### Changement dans `cmd/server/main.go`

Apres `runMigrations(...)`, le serveur runtime n'a pas besoin de garder metadata en
read-write pour ses lectures bootstrap. Le handle runtime doit donc passer a :

```go
metaDB, err := duckdb.OpenReadOnly(metaPath)
```

Le retry de demarrage autour de metadata doit etre reexamine avec cette nouvelle semantique :

- si l'ouverture read-only suffit et ne rencontre plus de lock, le retry peut etre simplifie ;
- si un retry reste utile pour Windows / Air, il doit etre conserve comme filet dev, pas
  comme masque d'une sur-ouverture RW persistante.

#### Changement dans `pool.go`

`PlayerDB.Metadata` devient read-only par defaut :

```go
metaDB, err := OpenReadOnly(cfg.MetaDBPath)
```

Les repos qui lisent `pdb.Metadata` restent compatibles. Les rares chemins qui ont besoin
d'ecrire metadata doivent ouvrir un handle RW dedie via leur propre abstraction de write,
au lieu de reutiliser un handle pool keep-alive ouvert en RW pour tous les joueurs.

#### Changement dans `lab/provider.go`

Les parcours `GetResources()` et `loadMedalGuards()` sont des parcours de lecture. Ils
doivent donc passer a `OpenReadOnly(...)`.

#### Changement dans le shutdown et le hot-reload

Le shutdown du serveur doit explicitement appeler `duckdb.CloseAll()` pour drainer les
`PlayerDB` et leurs sinks avant la sortie du process.

Ordre retenu au shutdown :

1. arreter la creation de nouveau travail (`cancelScheduler`) ;
2. stopper le watcher (`watcherDaemon.Stop()`) ;
3. executer `srv.Shutdown(...)` ;
4. drainer les sinks `PersistSink` : `CloseAll()` doit appeler `pdb.Sink.Close()` **avant**
   de fermer `pdb.Player` / `pdb.Metadata` — sans quoi le consumer du sink peut tenter
   d'ecrire dans un handle deja ferme. La sequence interne de `CloseAll()` doit donc etre :
   `sink.Close()` → attendre `done` → `pdb.Player.Close()` → `pdb.Metadata.Close()` ;
5. drainer le pool global via `duckdb.CloseAll()` ;
6. seulement ensuite fermer les handles racine `sharedDB` et `metaDB` du `main`.

Dans la meme passe, `.air.toml` doit etre revu pour laisser plus de temps a l'ancien
`server.exe~` pour liberer les handles DuckDB avant la relance du nouveau process.

Valeurs cibles de depart retenues pour Air :

- `kill_delay = "3000ms"` ;
- `stop_timeout = 5000`.

Si ces valeurs ne suffisent pas, l'acceptation du chantier impose de les ajuster plutot que
de garder une boucle de retry metadata masquant un overlap de process.

#### Pourquoi cette etape vient avant `PersistSink`

Elle retire d'abord une partie de la surface RW metadata et clarifie le cycle de vie
process. La serialisation restante des writes metadata par `PersistSink` devient ensuite
beaucoup plus simple a raisonner.

### Etape 7 — Transformer `PersistSink` en queue par joueur

#### Fichiers modifies

- `apps/go-api/internal/platform/duckdb/persist_sink.go`
- `apps/go-api/internal/platform/duckdb/pool.go`
- `apps/go-api/internal/api/registry.go`
- tests DuckDB relies a `PersistSink`

#### Nouveau modele retenu

`PersistSink` devient un composant vivant associe au `PlayerDB`, avec :

- une queue bufferisee ;
- un seul consumer goroutine ;
- un `Close()` bloquant pour le drain ;
- acquisition du lease player **avant** chaque write dans `stats.duckdb` ;
- acquisition d'un lease `MetaPath` **avant** chaque write dans `metadata.duckdb` ;
- politique de drop explicite quand la queue est pleine ou quand le lease n'est pas acquis
  dans le timeout best-effort.

#### Structure cible

```go
type persistJob struct {
    kind      string
    trackPath string
    rawBody   []byte
}

type PersistSink struct {
    MetaPath   string
    PlayerPath string
    XUID       string

    queue     chan persistJob
    cancel    context.CancelFunc
    done      chan struct{}
    closeOnce sync.Once
}
```

#### Creation du sink

`NewPersistSink()` initialise la queue et lance `consume(ctx)`.

Capacite de la queue retenue : **16 jobs**. Au-dela, le job est droppe apres log.
Cette capacite absorbe 16 refreshes live sans blocage tout en bornant la memoire.

Important : le consumer ne fait pas de logique metier speciale. Il appelle les methodes
de write existantes. On preserve donc les tests actuels sur `writeBattlePass()` autant que
possible.

#### `PersistBattlePass()` et `PersistChallenges()`

Ces methodes ne doivent plus lancer `go func()`. Elles doivent seulement essayer
d'enfiler un job :

- succes : retour immediat ;
- queue pleine : `slog.Warn(...)` puis drop best-effort ;
- sink ferme : drop silencieux ou warn leger, mais jamais panic.

#### Consumer

Avant tout write player, le consumer doit faire :

```go
release, err := dblease.AcquireLease(s.PlayerPath, dblease.PlayerLeaseTimeout)
```

Avant tout write metadata, le consumer doit aussi faire :

```go
releaseMeta, err := dblease.AcquireLease(s.MetaPath, dblease.MetadataLeaseTimeout)
```

#### Mapping complet des timeouts best-effort restants

Apres migration vers `AcquireLeaseCtx` pour les flux de sync, les seuls appels qui
utilisent encore `AcquireLease` avec timeout fixe sont :

| Surface | Chemin | Constante |
|---|---|---|
| consumer `PersistSink` — player DB | `s.PlayerPath` | `dblease.PlayerLeaseTimeout` (5s) |
| consumer `PersistSink` — metadata DB | `s.MetaPath` | `dblease.MetadataLeaseTimeout` (10s) |
| tests a borne dure | tout chemin | `dblease.SharedLeaseTimeout` (45s) ou constante locale |

Si l'un des leases n'est pas acquis :

- log `Warn` ;
- job abandonne ;
- pas d'erreur retournee au caller HTTP.

Ce comportement reste coherent avec la nature fire-and-forget de ces snapshots.

#### `Close()`

Le sink doit pouvoir etre ferme proprement au shutdown. La sequence exacte retenue :

```go
func (s *PersistSink) Close() {
    s.cancel()      // signale ctx.Done() au consumer
    close(s.queue)  // debloque le select si la queue est vide et le consumer attend
    <-s.done        // attend que le consumer soit sorti
}
```

`close(s.queue)` est obligatoire : si on fait uniquement `s.cancel()` + `<-s.done`,
le consumer bloque sur `case job := <-s.queue` quand la queue est vide —
le `ctx.Done()` dans le select ne peut pas gagner tant qu'un `case` non-default est
toujours selectionnable. Sans le `close`, le shutdown deadlock.

L'usage de `sync.Once` est retenu pour eviter double fermeture / panic.

#### Integration au pool joueur

`PlayerDB` gagne un champ :

```go
Sink *PersistSink
```

`openPlayerDB()` initialise ce sink avec les chemins du joueur et de metadata.

`CloseAll()` doit :

- fermer le sink si non nil ;
- seulement ensuite fermer les handles DB.

#### Integration a `registry.go`

`HomeCtxWithAuth()` et `SeasonPassCtxWithAuth()` ne doivent plus appeler `NewPersistSink(...)`.
Ils doivent reutiliser `pdb.Sink`.

#### Correctif inclus dans la meme etape

Le bug actuel de `writeChallenges()` :

```go
if err != nil {
    defer db.Close()
}
```

est corrige dans la meme passe. Le code doit etre recrit avec un flow normal `if/else`
propre, sans `defer` dans la branche erreur.

### Etape 8 — Ajouter le fast path `match deja en shared`

#### Fichiers modifies

- `apps/go-api/internal/domain/sync.go`
- `apps/go-api/internal/domain/domain_test.go`
- `apps/go-api/internal/sync/engine.go`
- `apps/go-api/internal/sync/engine_test.go`
- `apps/go-api/internal/sync/engine_e2e_test.go`

#### Changement de contrat

`SyncResult` gagne :

```go
MatchesReused int
```

Semantique retenue : match deja present dans `match_registry`, donc la partie shared est
reutilisee ; seul `player_match_enrichment` est ecrit pour le joueur courant.

#### Nouveau helper

Dans `engine.go` :

```go
func matchExistsInRegistry(sharedDB *sql.DB, matchID string) (bool, error)
```

avec un `SELECT COUNT(*) FROM match_registry WHERE match_id = ?`.

#### Fast path dans `processMatch()`

Le flow cible en tete de `processMatch()` est :

1. verifier si le match existe deja en shared ;
2. si la verification echoue : `Warn` et continuer sur le full path ;
3. si le match existe :
   - `UpsertPlayerEnrichment(playerDB, matchID, "")` ;
   - `result.MatchesInserted++` ;
   - `result.MatchesReused++` ;
   - `result.InsertedMatchIDs = append(result.InsertedMatchIDs, matchID)` ;
   - log `processMatch: fast path — match deja en shared` ;
   - retour sans `GetMatchStats()` ni insertions shared.

#### Pourquoi `MatchesInserted` reste incremente

Le match est bien **nouveau pour ce joueur**. Le compteur `MatchesReused` vient completer
`MatchesInserted`, pas le remplacer.

#### Log final du sync

Le log `sync: termine` doit ajouter :

```go
"reused", result.MatchesReused
```

#### Effet concret recherche

Si Joueur A et Joueur B ont joue le meme match :

- A fait le full path ;
- B fait seulement l'enrichissement player ;
- la shared DB n'est pas refaite ;
- l'appel `GetMatchStats(matchID)` n'est fait qu'une fois.

### Etape 9 — Logs cibles a ajouter ou ajuster

Le plan n'embarque pas un chantier general de logging. En revanche, certains logs font
partie de la definition meme du correctif et doivent etre integres.

#### Dans `dblease/lease.go`

Lors de l'implementation de l'etape 1, les points de log suivants doivent etre ajoutes
dans `AcquireLeaseCtx` :

- debug en debut d'attente du lock :
  `slog.DebugContext(ctx, "dblease: attente du lease", "db", path)` ;
- warn si le contexte est annule pendant l'attente :
  `slog.WarnContext(ctx, "dblease: contexte annule pendant attente", "db", path, "err", ctx.Err())` ;
- debug a l'acquisition reussie :
  `slog.DebugContext(ctx, "dblease: lease acquis", "db", path, "wait_ms", elapsed.Milliseconds())`.

Tout `Warn` et `Error` dans la couche `dblease` doit inclure au minimum `"db", path`.

#### Dans `pool.go`

Lors de l'implementation des etapes 3-bis et 6 :

- debug quand `PlayerRO` est ouvert dans `openPlayerDB()` :
  `slog.Debug("pool: player db RO ouvert", "gamertag", cfg.Gamertag)` ;
- debug avant drain du sink dans `CloseAll()` :
  `slog.Debug("pool: drain sink avant fermeture", "gamertag", pdb.Gamertag)`.

#### Dans `engine.go`

- warnings explicites si `matchExistsInRegistry()` echoue et que le full path est utilise ;
- suppression du champ `timeout` devenu faux sur les erreurs `AcquireLeaseCtx` ;
- champ `reused` dans le log final ;
- eventuellement un debug au moment ou le fast path est pris.

#### Dans `persist_sink.go`

- warn si la queue est pleine ;
- warn si le lease player timeoute ;
- warn si le lease metadata timeoute ;
- warn sur echec de write battle pass/challenges ;
- pas de log noise sur les enqueues reussis.

#### Dans `coordinator.go`

La version minimale retenue est de **laisser le backlog general hors de ce plan**.
En revanche, si le diff touche deja ce fichier pour la lecture multi-joueurs en debug,
le passage a `slog.*Context` est acceptable, mais il ne fait pas partie du coeur du plan.

---

## 6. Impacts fichier par fichier

| Fichier | Type de changement | Detail |
|---|---|---|
| `apps/go-api/internal/platform/dblease/lease.go` | nouveau | proprietaire du lease, timeout borne + contexte |
| `apps/go-api/internal/platform/dblease/lease_test.go` | nouveau | tests directs du package neutre |
| `apps/go-api/internal/sync/lease.go` | modifie | facade de compatibilite vers `dblease` |
| `apps/go-api/internal/sync/lease_test.go` | modifie | complete les tests contextuels ou valide la facade |
| `apps/go-api/internal/sync/schema.go` | modifie | `OpenPlayerDB` / `OpenSharedDB` -> handles `*duckdb.DB` caches |
| `apps/go-api/internal/sync/engine.go` | modifie | leases ctx, call-sites handles, fast path shared, log final enrichi |
| `apps/go-api/internal/sync/backfill_weapons.go` | modifie | lease ctx shared + adaptation handle |
| `apps/go-api/internal/sync/career.go` | modifie | `openCareerMetadataDB` via `OpenReadOnly` cache |
| `apps/go-api/internal/scheduler/auto_sync.go` | modifie | `defaultTokenReader` via `OpenReadOnly` cache + retrait du blank import driver si obsolete |
| `apps/go-api/cmd/server/main.go` | modifie | metadata runtime en read-only + `duckdb.CloseAll()` au shutdown |
| `apps/go-api/internal/domain/sync.go` | modifie | ajoute `MatchesReused` |
| `apps/go-api/internal/domain/domain_test.go` | modifie | test du nouveau champ `MatchesReused` |
| `apps/go-api/internal/sync/schema_integration_test.go` | modifie | utilise `handle.SQLDb()` |
| `apps/go-api/internal/sync/engine_test.go` | modifie | tests `matchExistsInRegistry` |
| `apps/go-api/internal/sync/engine_e2e_test.go` | modifie | test du fast path inter-joueurs |
| `apps/go-api/internal/platform/duckdb/persist_sink.go` | modifie fortement | queue, consumer, lease player + metadata, `Close()`, fix `writeChallenges` |
| `apps/go-api/internal/platform/duckdb/pool.go` | modifie | champ `PlayerRO` (RO handle), metadata read-only, champ `Sink`, init dans `openPlayerDB()`, drain dans `CloseAll()` ; suppression des `OpenReadWrite` shared/social des migrations dans `openPlayerDB()` |
| `apps/go-api/internal/api/registry.go` | modifie | reutilise `pdb.Sink` au lieu de recreer un sink |
| `apps/go-api/internal/platform/lab/provider.go` | modifie | `OpenReadOnly` sur les parcours metadata read-only |
| `apps/go-api/.air.toml` | modifie | `kill_delay` / `stop_timeout` ajustes pour le hot-reload |
| `apps/go-api/internal/platform/duckdb/battlepass_cache_test.go` | modifie | ferme explicitement le sink si `NewPersistSink()` demarre un consumer |
| `apps/go-api/internal/platform/duckdb/pool_migration_test.go` | possiblement modifie | verifier les helpers / nil guards avec `Sink` |
| `apps/go-api/internal/platform/duckdb/pool_test.go` | nouveau | 3 tests PlayerRO : handles distincts, ordre de fermeture |
| `apps/go-api/internal/platform/duckdb/persist_sink_test.go` | nouveau | 6 tests comportement queue : drain, drop, idempotence, single consumer, goleak |
| `apps/go-api/internal/platform/duckdb/wal_concurrency_test.go` | nouveau (tag `integration`) | 4 tests non-regression WAL : RO non bloque, multi-RO concurrent, snapshots isoles |
| `apps/go-api/internal/platform/duckdb/shutdown_test.go` | nouveau (tag `integration`) | 3 tests drain/refcount/goleak au shutdown |

---

## 7. Strategie de tests

### 7.1 Tests de lease

#### `internal/platform/dblease/lease_test.go`

Tests a couvrir :

- `TestAcquireLease_Basic`
- `TestAcquireLease_Timeout`
- `TestAcquireLease_DifferentPaths`
- `TestAcquireLease_NoGoroutineLeak`
- `TestAcquireLeaseCtx_Basic`
- `TestAcquireLeaseCtx_CancelledCtx`
- `TestAcquireLeaseCtx_CancelDuringWait`
- `TestAcquireLeaseCtx_SequentialAfterRelease`

#### `internal/sync/lease_test.go`

Option retenue : garder soit les tests existants en facade, soit les alleger si le package
`dblease` porte deja les vrais tests de comportement. L'important est d'eviter deux suites
copiees-collees qui testent exactement la meme chose.

### 7.2 Tests sync/schema

#### `internal/sync/schema_integration_test.go`

Doit toujours verifier :

- creation du fichier ;
- application du schema player/shared ;
- tables presentes.

La seule adaptation attendue est l'usage de `handle.SQLDb()`.

### 7.3 Tests sync/engine

#### `internal/sync/engine_test.go`

Ajouter au minimum :

- `TestMatchExistsInRegistry_NotFound`
- `TestMatchExistsInRegistry_Found`

#### `internal/sync/engine_e2e_test.go`

Ajouter au minimum :

- `TestProcessMatch_FastPath`

Verifications attendues :

- `GetMatchStats()` n'est pas appele ;
- `MatchesInserted == 1` ;
- `MatchesReused == 1` ;
- une ligne `player_match_enrichment` ;
- pas de doublon `match_registry`.

### 7.4 Tests PersistSink

Les tests existants sur `writeBattlePass()` ne doivent pas etre casses. En plus, il faut
ajouter des tests de comportement sur le nouveau modele de queue.

Minimum attendu :

- un test de drain simple : enqueue puis `Close()` ;
- un test que `CloseAll()` ne panic pas quand `Sink` est nil ;
- un test que `PersistBattlePass()` / `PersistChallenges()` ne lancent plus de goroutines
  ad hoc mais passent bien par la queue ;
- un test ou deux sinks distincts partagent le meme `MetaPath` et serialisent correctement
  les writes metadata via le lease par chemin ;
- ou, si l'assertion sur les goroutines est trop fragile, un test que deux enqueues
  successifs produisent bien des writes seriees sans doublon supplementaire.

Noms de tests retenus (fichier `internal/platform/duckdb/persist_sink_test.go`) :

- `TestPersistSink_EnqueueSingleJob`
- `TestPersistSink_QueueFull_Drop`
- `TestPersistSink_Close_DrainsQueue`
- `TestPersistSink_Close_Idempotent`
- `TestPersistSink_SingleConsumer`
- `TestPersistSink_NoLeakAfterClose` (requiert `goleak`)

### 7.5 Tests scheduler

Si le package `internal/scheduler` dispose deja de tests autour de `defaultTokenReader`,
ils doivent rester verts. Sinon, l'etape minimale est une revalidation du package entier.

### 7.6 Validation metadata / hot-reload

Le symptome a corriger existe en dev runtime, pas seulement en test unitaire. Il faut donc
ajouter au minimum :

- un test ou une verification d'integration que `duckdb.CloseAll()` draine correctement des
  `PlayerDB` deja ouverts ;
- une validation manuelle sous `make go-api-dev` : lancer des appels Home / Season Pass,
  modifier un fichier Go, verifier que le redemarrage ne boucle plus sur `metadata verrouillee`.

Critere d'acceptation retenu :

- 10 redemarrages Air consecutifs pendant des appels Home / Season Pass ;
- 0 occurrence des logs `metadata verrouillee, nouvelle tentative...` ;
- 0 occurrence de `ouverture metadata echouee apres ...`.

Noms de tests d'integration retenus (fichier `internal/platform/duckdb/shutdown_test.go`) :

- `TestShutdown_DrainsSinksBeforeClose` — 3 joueurs avec PersistSink charge ; `CloseAll()` doit traiter tous les jobs avant fermeture des handles, sans erreur "write on closed DB"
- `TestShutdown_CloseAllReleasesRefCount` — ouvrir Player RW + PlayerRO puis `CloseAll()` ; les deux cles `rw:` et `ro:` sont supprimees de `openDBs`
- `TestShutdown_NoLeakAfterCloseAll` — `CloseAll()` + `goleak.VerifyNone(t)` → aucune goroutine PersistSink residuelle

### 7.7 Tests pool PlayerRO et non-regression WAL

Ces tests valident les invariants de l'etape 3-bis (`PlayerRO`) et la garantie DuckDB
WAL sur laquelle repose l'isolation lecteurs/writers.

#### Fichier `internal/platform/duckdb/pool_test.go` (nouveau)

- `TestPlayerDB_HasPlayerRO` — apres `GetOrOpen` sur un joueur, `pdb.PlayerRO != nil` et `pdb.PlayerRO.Path() == pdb.Player.Path()`
- `TestPlayerRO_IsDifferentHandle` — `pdb.PlayerRO.SQLDb() != pdb.Player.SQLDb()` (cles `ro:` vs `rw:`)
- `TestCloseAll_DrainsSinkFirst` — mock sink ; `sink.Close()` appele avant `pdb.Player.Close()`

#### Fichier `internal/platform/duckdb/wal_concurrency_test.go` (nouveau, tag `integration`)

- `TestWAL_ReadWhileWriteInProgress` — handle RW lance 100 writes ; handle RO `SELECT` en parallele → retour < 200 ms, pas d'erreur, voit les rows committees
- `TestWAL_MultipleROConcurrent` — 10 goroutines lisent via `ro:path` pendant qu'une ecrit → aucune erreur, chaque lecteur termine < 500 ms
- `TestWAL_RWAfterROClose` — ouvrir RO, fermer RO, ouvrir RW → aucune erreur, `refcount(ro:path) == 0`
- `TestWAL_RODoesNotSeeUncommittedWrite` — ecrire sans COMMIT, lire via RO → SELECT ne voit pas la row non committee

### 7.8 Dependance externe requise

- `go.uber.org/goleak` : detection de goroutines orphelines dans les tests de non-regression
  (ajouter dans `go.mod` si absent).
- Tags de build retenus : `unit` (sans fichier DuckDB physique, mocks), `integration`
  (fichier DuckDB temporaire via `t.TempDir()`).

---

## 8. Validation recommandee

```bash
cd apps/go-api

# 0. Pas de cycle d'import apres extraction `dblease`
go mod tidy
go build ./...

# 1. Leases
go test ./internal/platform/dblease/...

# 2. Sync integration
go test -tags=integration ./internal/sync/...

# 3. DuckDB / PersistSink / pool
go test -tags=integration ./internal/platform/duckdb/...

# 4. Scheduler
go test ./internal/scheduler/...
```

Validation runtime specifique metadata / hot-reload :

```bash
# dans un autre terminal
make go-api-dev

# pendant que l'API sert des pages authentifiees
curl -s --max-time 8 "http://127.0.0.1:8000/api/v1/players/<slug>/pages/home"
curl -s --max-time 8 "http://127.0.0.1:8000/api/v1/players/<slug>/pages/palmares/season-pass"

# puis toucher un fichier Go pour forcer Air a redemarrer
# et verifier l'absence de boucle
#   metadata verrouillee, nouvelle tentative...
#   ouverture metadata echouee apres ...
# Repetition minimale retenue : 10 rebuilds consecutifs sans warning/error metadata.
```

Si la suite integration complete est trop lourde localement, l'ordre minimal de falsification
reste :

```bash
cd apps/go-api

go test ./internal/platform/dblease/...
go test -tags=integration ./internal/sync/... -run 'Test(OpenPlayerDB|OpenSharedDB|ProcessMatch_FastPath|AcquireLease)'
go test -tags=integration ./internal/platform/duckdb/... -run 'Test(PersistSink|GetOrOpen)'
go test ./internal/scheduler/...
go build ./...
```

---

## 9. Points de vigilance

### 9.1 Le contexte appelant doit etre borne

Avec `AcquireLeaseCtx`, la borne de temps ne vient plus d'un timeout interne. Il faut donc
que :

- les handlers HTTP utilisent des contextes correctement bornes ;
- les workers / watchers aient des contextes d'annulation realistes ;
- ou le lease metadata n'est pas pris dans `MetadataLeaseTimeout` ;
- les tests qui attendent une annulation utilisent `context.WithTimeout(...)`.

Sinon, un wait sur lease pourrait durer indefiniment par design.

### 9.2 `PersistSink` reste best-effort

Le sink queue supprime la concurrence sauvage, mais il ne transforme pas ces ecritures en
chemin critique. Si :

- la queue est pleine ;
- ou le lease player n'est pas pris dans `PlayerLeaseTimeout` ;

alors le job est droppe apres log. C'est volontaire.

La logique produit reste acceptable parce qu'un prochain refresh live rechargera ces
snapshots.

### 9.3 `metadata.duckdb` a maintenant une discipline propre

Le plan renforce explicitement metadata :

- ouverture read-only par defaut pour les lecteurs runtime ;
- lease par `MetaPath` pour les writes `PersistSink` ;
- drainage du pool global au shutdown.

Il faut donc verifier qu'aucun code applicatif n'utilise encore `pdb.Metadata` comme un
handle RW implicite. Si un vrai writer metadata apparait, il doit ouvrir son propre handle
RW explicite plutot que de supposer que le pool garde deja la base en ecriture.

### 9.4 Tests existants qui instancient `NewPersistSink()`

Meme si `registry.go` n'appelle plus `NewPersistSink(...)`, le constructeur doit rester
public pour les tests qui l'instancient directement.

Si `NewPersistSink()` lance maintenant un consumer, les tests existants qui appellent ce
constructeur devront le fermer explicitement :

```go
sink := NewPersistSink(...)
defer sink.Close()
```

Sans cela, la suite de tests peut fuiter des goroutines ou garder des handles ouverts plus
longtemps que prevu.

### 9.5 `PlayerDB.Sink` doit etre optionnellement nil-safe

Certains tests construisent ou seedent des `PlayerDB` partiels. `CloseAll()` et les repos
qui consultent `pdb.Sink` doivent donc rester nil-safe.

### 9.6 Le fast path shared ne doit pas masquer une erreur de verification

Si `matchExistsInRegistry()` echoue, le plan retient un comportement conservateur :

- log `Warn` ;
- continuer avec le full path.

Le but est d'eviter une optimisation qui ferait perdre des donnees en cas de faux negatif
ou de probleme transitoire de lecture.

---

## 10. Ordre d'implementation recommande

L'ordre ci-dessous minimise les allers-retours et permet une validation progressive.

1. `internal/platform/dblease/lease.go`
2. `internal/sync/lease.go`
3. `internal/sync/schema.go`
4. `internal/sync/engine.go` et `internal/sync/backfill_weapons.go` pour les handles caches
5. `internal/sync/career.go` et `internal/scheduler/auto_sync.go`
6. `internal/platform/duckdb/pool.go` : ajout `PlayerRO` + bascule services HTTP sur `PlayerRO.SQLDb()` dans `registry.go` et les repos de lecture
7. `apps/go-api/cmd/server/main.go`, complement `internal/platform/duckdb/pool.go`, `internal/platform/lab/provider.go` et `.air.toml` pour le durcissement metadata
8. `internal/platform/duckdb/persist_sink.go`, puis complement `pool.go` / `registry.go`
9. `internal/domain/sync.go` + fast path `processMatch()`
10. tests sync / dblease / duckdb / scheduler + scenario hot-reload metadata ; logs cibles integres dans les etapes precedentes
11. validation `go build ./...`

---

## 11. Resultat attendu une fois le plan implemente

Apres implementation complete :

- toute ouverture DuckDB process-level du backend Go passe par `internal/platform/duckdb` ;
- les vrais flux de sync attendent les leases aussi longtemps que leur contexte l'autorise ;
- `RunBackfill()` et `BackfillWeaponKillsForMatches()` ne contournent plus la discipline
  de verrouillage ;
- `metadata.duckdb` n'est plus tenue read-write par defaut sur des chemins de lecture ;
- `PersistSink` n'est plus recree par requete et n'ecrit plus en goroutines detachees
  concurrentes ;
- les writes metadata sont serialises par `MetaPath`, pas seulement par joueur ;
- le meme match vu par deux joueurs n'entraine plus deux appels `GetMatchStats()` ;
- les logs finaux du sync permettent de distinguer les matchs inseres des matchs reutilises ;
- la shared DB reste serializee fonctionnellement sans faux echecs a 10 secondes ;
- le shutdown draine les `PlayerDB` avant sortie et le hot-reload dev ne relance plus un
  nouveau `server.exe~` sur un ancien process tenant metadata ;
- les conflits DuckDB residuels sont ramenes a des cas transitoires attendus ou a des bugs
  de logique clairement identifiables, et non plus a une topologie de connexions incoherente.

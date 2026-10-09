# GO_ARCHITECTURE_RULES.md — Architecture logicielle formelle du backend Go

> [!IMPORTANT]
> Ce document est **contraignant**. Tout code Go qui viole ces règles doit être refusé
> en revue ou corrigé avant merge. Les exceptions sont tolérées uniquement si elles sont
> marginales, justifiées par un commentaire `// ARCH-EXCEPTION: <raison>` et documentées
> dans le thought_log.

> [!NOTE]
> Lire aussi [PROGRAM_CHARTER.md](PROGRAM_CHARTER.md) pour le cadre du programme,
> [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md) pour les types canoniques Halo,
> et [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md) pour la chaîne
> provider → canonical → DTO.

---

## 1. Architecture en couches et direction des dépendances

Le backend Go suit une **architecture hexagonale (ports & adapters)** à 5 couches.
La règle fondamentale : **les dépendances pointent vers l'intérieur, jamais vers l'extérieur**.

```text
┌─────────────────────────────────────────────────────────────┐
│                    cmd/ (composition root)                   │
│   Assemble les couches, injecte les dépendances, démarre.   │
└────────────────────────────┬────────────────────────────────┘
                             │ importe tout, n'est importé par personne
┌────────────────────────────▼────────────────────────────────┐
│               internal/api/ (couche transport)               │
│   Handlers HTTP, middleware, routing, DTO request/response.  │
│   Dépend de : service/ (interfaces uniquement)               │
│   Ne dépend PAS de : platform/, domain/ directement          │
└────────────────────────────┬────────────────────────────────┘
                             │ appelle les services via interfaces
┌────────────────────────────▼────────────────────────────────┐
│               internal/service/ (couche application)         │
│   Orchestration métier : combine repository + domain logic.  │
│   Dépend de : domain/, port interfaces (repository, halo)    │
│   Ne dépend PAS de : platform/ (implémentations concrètes)   │
└────────────┬───────────────────────────────────┬────────────┘
             │ utilise                           │ utilise
┌────────────▼──────────┐          ┌─────────────▼────────────┐
│  internal/domain/      │          │  internal/port/           │
│  (couche domaine)      │          │  (interfaces/contrats)    │
│  Logique métier pure : │          │  Go interfaces :          │
│  sessions, scores,     │          │  Repository, HaloClient,  │
│  citations, LUSR,      │          │  SyncEngine, TokenStore   │
│  performance, séries,  │          │  N'importe RIEN d'autre   │
│  bitmask, enrichments. │          │  que des types domaine.   │
│  0 import externe,     │          │                           │
│  0 IO, 0 framework.    │          │                           │
└────────────────────────┘          └─────────────────────────┘
             ▲                                  ▲
             │ types domaine                    │ implémente
┌────────────┴──────────────────────────────────┴─────────────┐
│               internal/platform/ (couche infrastructure)     │
│   Implémentations concrètes : DuckDB, HTTP Halo, MSAL,      │
│   filesystem, config, logging, telemetry.                    │
│   Dépend de : port/ (implémente les interfaces)              │
│   Dépend de : domain/ (utilise les types)                    │
│   N'est importé QUE par cmd/ (jamais par service/ ou api/)   │
└──────────────────────────────────────────────────────────────┘
```

### Règles de dépendances — matrice d'imports autorisés

| Package importateur →   | `domain/` | `port/` | `service/` | `api/` | `platform/` | `cmd/` |
|-------------------------|:---------:|:-------:|:----------:|:------:|:-----------:|:------:|
| **`domain/`**           |  self     |   ❌    |     ❌     |   ❌   |     ❌      |   ❌   |
| **`port/`**             |   ✅      |  self   |     ❌     |   ❌   |     ❌      |   ❌   |
| **`service/`**          |   ✅      |   ✅    |    self    |   ❌   |     ❌      |   ❌   |
| **`api/`**              |   ✅      |   ✅    |     ✅     |  self  |     ❌      |   ❌   |
| **`platform/`**         |   ✅      |   ✅    |     ❌     |   ❌   |    self     |   ❌   |
| **`cmd/`**              |   ✅      |   ✅    |     ✅     |   ✅   |     ✅      |  self  |

**Lecture** : `service/` peut importer `domain/` et `port/`, mais **jamais** `platform/` ni `api/`.

### Règle absolue

> **Aucun handler HTTP ne doit jamais appeler `duckdb.Query()`, `sql.DB.Query()` ou toute
> méthode d'accès base de données directement.** Les handlers appellent un service,
> le service appelle un repository via son interface.

---

## 2. Interfaces Go obligatoires (ports)

Ces interfaces vivent dans `internal/port/`. Elles sont l'équivalent Go exact des `typing.Protocol`
Python existants dans `src/ports/`. Un type concret ne doit **jamais** apparaître dans la signature
d'un service ou d'un handler — seulement ces interfaces.

### 2.1 `PlayerRepository` (équivalent Python `DataRepository`)

```go
// internal/port/repository.go
package port

import (
    "context"
    "time"

    "apps/go-api/internal/domain"
)

// PlayerRepository — contrat de lecture des données d'un joueur.
// Implémenté par platform/duckdb.PlayerRepo.
// Équivalent Python : src/ports/repository.py → DataRepository (Protocol)
type PlayerRepository interface {
    // Identité
    XUID() string
    DBPath() string

    // Matchs
    LoadMatches(ctx context.Context, filters domain.MatchFilters) ([]domain.MatchRow, error)
    LoadMatchesInRange(ctx context.Context, start, end time.Time) ([]domain.MatchRow, error)
    GetMatchCount(ctx context.Context) (int, error)

    // Médailles
    LoadTopMedals(ctx context.Context, matchIDs []string, topN int) ([]domain.MedalCount, error)
    LoadMatchMedals(ctx context.Context, matchID string) ([]domain.MedalCount, error)

    // Coéquipiers
    ListTopTeammates(ctx context.Context, matchIDs []string, topN int) ([]domain.TeammateStats, error)
}
```

### 2.2 `SharedRepository` (pas d'équivalent Python direct — lectures cross-joueur)

```go
// internal/port/shared_repository.go
package port

// SharedRepository — contrat pour les lectures sur shared_matches_v2.duckdb.
// Match participants, gamertag lookup, killer/victim, weapon kills.
type SharedRepository interface {
    ResolveGamertag(ctx context.Context, xuid string) (string, error)
    GetMatchParticipants(ctx context.Context, matchID string) ([]domain.Participant, error)
    GetMatchHistory(ctx context.Context, xuid string, cursor domain.Cursor, limit int) (*domain.MatchHistoryPage, error)
    GetKillerVictimPairs(ctx context.Context, matchID string) ([]domain.KillerVictimPair, error)
    GetWeaponKills(ctx context.Context, matchID string, xuid string) ([]domain.WeaponKill, error)
    // ... extensible par surface
}
```

### 2.3 `HaloClient` (équivalent Python `HaloAPIPort`)

Correspondance **1:1 complète** avec les 14 méthodes de `HaloAPIPort` dans `src/ports/api.py`.
L'interface est `io.Closer` pour nettoyer les transports HTTP (équivalent du context manager Python).

```go
// internal/port/halo_client.go
package port

import (
    "context"
    "io"

    "apps/go-api/internal/domain"
)

// HaloClient — contrat pour tout client API Halo (quel que soit le titre).
// Implémenté par platform/halo/haloinfinite.Client.
// Équivalent Python : src/ports/api.py → HaloAPIPort (Protocol, 14 méthodes)
type HaloClient interface {
    io.Closer  // __aexit__ Python — ferme le transport HTTP sous-jacent

    // ------------------------------------------------------------------
    // Historique et stats de matchs (5 méthodes)
    // ------------------------------------------------------------------

    // get_match_history(player, match_type, start, count)
    GetMatchHistory(ctx context.Context, player string, opts domain.HistoryOpts) ([]domain.MatchHistoryItem, error)

    // get_match_stats(match_id)
    GetMatchStats(ctx context.Context, matchID string) (*domain.RawMatchStats, error)

    // get_skill_stats(match_id, xuids)
    GetSkillStats(ctx context.Context, matchID string, xuids []uint64) (*domain.SkillResponse, error)

    // get_highlight_events(match_id)
    GetHighlightEvents(ctx context.Context, matchID string) ([]domain.HighlightEvent, error)

    // get_match_data(match_id, xuids, with_skill, with_highlight_events)
    // Composite : stats + skill (optionnel) + events (optionnel) en parallèle.
    GetMatchData(ctx context.Context, matchID string, xuids []uint64, opts domain.MatchDataOpts) (*domain.MatchData, error)

    // ------------------------------------------------------------------
    // Assets — maps, playlists, game variants (1 méthode)
    // ------------------------------------------------------------------

    // get_asset(asset_type, asset_id, version_id)
    GetAsset(ctx context.Context, assetType, assetID, versionID string) (*domain.Asset, error)

    // ------------------------------------------------------------------
    // Profil joueur (5 méthodes)
    // ------------------------------------------------------------------

    // get_career_rank_progression(xuid)
    GetCareerRankProgression(ctx context.Context, xuid string) (*domain.CareerRankData, error)

    // get_match_count(xuid)
    GetMatchCount(ctx context.Context, xuid string) (map[string]int, error)

    // get_player_customization(xuid)
    GetPlayerCustomization(ctx context.Context, xuid string) (*domain.PlayerCustomization, error)

    // get_user_by_gamertag(gamertag)
    GetUserByGamertag(ctx context.Context, gamertag string) (*domain.UserIdentity, error)

    // get_users_by_id(xuids)
    GetUsersByID(ctx context.Context, xuids []string) ([]domain.UserIdentity, error)

    // ------------------------------------------------------------------
    // Films / weapon extraction (2 méthodes)
    // ------------------------------------------------------------------

    // get_film_by_match_id(match_id)
    GetFilmByMatchID(ctx context.Context, matchID string) (*domain.Film, error)

    // download_film_chunk(url)
    DownloadFilmChunk(ctx context.Context, url string) ([]byte, error)
}

// HistoryOpts — options pour GetMatchHistory (défauts identiques au Python).
type HistoryOpts struct {
    MatchType string // "matchmaking" par défaut
    Start     int    // offset
    Count     int    // 25 par défaut
}

// MatchDataOpts — options pour GetMatchData.
type MatchDataOpts struct {
    WithSkill           bool // true par défaut
    WithHighlightEvents bool // true par défaut
}
```

### 2.3b `HaloClientFactory` (équivalent Python `create_api_client`)

La factory Python (`src/data/sync/api_factory.py`) centralise l'instanciation et permet
de switcher de backend sans modifier les consommateurs. Équivalent Go :

```go
// internal/platform/halo/factory.go
package halo

import (
    "fmt"

    "apps/go-api/internal/port"
    "apps/go-api/internal/platform/halo/haloinfinite"
)

// NewClient crée un HaloClient pour le backend demandé.
// Équivalent Python : src/data/sync/api_factory.py → create_api_client()
func NewClient(backend string, cfg ClientConfig) (port.HaloClient, error) {
    switch backend {
    case "spnkr", "":
        return haloinfinite.NewClient(cfg.Transport, cfg.TokenStore, cfg.RateLimit, cfg.Lang)
    default:
        return nil, fmt.Errorf("backend inconnu: %q", backend)
    }
}
```

**Règle** : les services et handlers ne connaissent que `port.HaloClient`.
Seul `cmd/` (composition root) appelle `halo.NewClient(backend, cfg)`.

### 2.4 `TokenStore` (pas d'équivalent Python unique — éclaté entre MSAL cache + sync_meta)

```go
// internal/port/token_store.go
package port

// TokenStore — contrat pour la persistance des tokens auth.
type TokenStore interface {
    LoadMSALCache(ctx context.Context) ([]byte, error)
    SaveMSALCache(ctx context.Context, data []byte) error
    LoadRefreshToken(ctx context.Context, gamertag string) (string, error)
    SaveRefreshToken(ctx context.Context, gamertag string, token string) error
}
```

### 2.5 `SyncEngine` (équivalent Python `_SyncProtocol` — contrat de service, pas de port)

```go
// internal/port/sync_engine.go
package port

// SyncEngine — contrat du moteur de synchronisation.
// Utilisé par les handlers de jobs et la CLI.
type SyncEngine interface {
    RunDeltaSync(ctx context.Context, gamertag string) (*domain.SyncResult, error)
    RunFullSync(ctx context.Context, gamertag string, maxMatches int) (*domain.SyncResult, error)
    RunBackfill(ctx context.Context, gamertag string, scope domain.SyncScope) (*domain.BackfillResult, error)
}
```

### Règle

> **Toute nouvelle frontière IO (nouveau provider, nouveau store, nouveau service externe)
> DOIT commencer par une interface dans `port/` AVANT d'écrire l'implémentation.**
> L'interface est le contrat ; l'implémentation vient après.

---

## 3. Injection de dépendances — constructor injection, zéro globales

### Principe

Chaque struct reçoit ses dépendances **par constructeur** (interface-typed). Aucun singleton,
aucun `var global`, aucun `sync.Once` pour des dépendances métier.

### Exemple canonique

```go
// internal/service/career.go
package service

import "apps/go-api/internal/port"

// CareerService orchestre la logique de la page Career.
type CareerService struct {
    playerRepo port.PlayerRepository
    sharedRepo port.SharedRepository
}

// NewCareerService — le seul point de construction.
func NewCareerService(pr port.PlayerRepository, sr port.SharedRepository) *CareerService {
    return &CareerService{playerRepo: pr, sharedRepo: sr}
}

func (s *CareerService) GetCareerOverview(ctx context.Context, xuid string) (*domain.CareerOverview, error) {
    // utilise s.playerRepo et s.sharedRepo — JAMAIS un accès DB direct
    // ...
}
```

```go
// internal/api/career_handler.go
package api

import "apps/go-api/internal/port"

type CareerHandler struct {
    career *service.CareerService  // ← typé sur le service, pas sur DuckDB
}
```

```go
// cmd/levelup/main.go  (composition root)
func main() {
    // Seul endroit où les types concrets apparaissent
    db := platform.NewDuckDBPool(cfg)
    playerRepo := platform.NewPlayerRepo(db)
    sharedRepo := platform.NewSharedRepo(db)
    haloClient := platform.NewHaloInfiniteClient(httpClient, tokenStore)

    careerSvc := service.NewCareerService(playerRepo, sharedRepo)
    careerHandler := api.NewCareerHandler(careerSvc)

    router := api.NewRouter(careerHandler, historyHandler, /* ... */)
    // ...
}
```

### Règles

1. **`cmd/` est le seul package qui instancie des types concrets** (composition root).
2. **Aucun `init()` ne doit créer ou configurer des dépendances métier.**
3. **Aucune variable de package** (`var db *sql.DB`) ne doit exister dans `service/`, `api/` ou `domain/`.
4. Les tests injectent des **mocks ou stubs** via les mêmes constructeurs — aucun monkey-patching.

### Ce qui est toléré dans les globales

- Logger : `slog.Default()` (stdlib, sans état métier).
- Constantes : `const DefaultPageSize = 25`.
- Rien d'autre.

---

## 4. Pare-feu architectural — linter `depguard`

Les règles de dépendances sont **enforced en CI**, pas seulement documentées.

### Configuration `.golangci.yml`

```yaml
linters:
  enable:
    - depguard

linters-settings:
  depguard:
    rules:
      domain-purity:
        files:
          - "**/internal/domain/**"
        deny:
          - pkg: "database/sql"
            desc: "domain/ ne doit pas importer de driver DB"
          - pkg: "net/http"
            desc: "domain/ ne doit pas importer de framework HTTP"
          - pkg: "**/internal/platform/**"
            desc: "domain/ ne doit pas importer d'implémentation concrète"
          - pkg: "**/internal/api/**"
            desc: "domain/ ne doit pas importer la couche transport"
          - pkg: "**/internal/service/**"
            desc: "domain/ ne doit pas importer la couche application"

      port-purity:
        files:
          - "**/internal/port/**"
        deny:
          - pkg: "database/sql"
            desc: "port/ ne doit pas importer de driver DB"
          - pkg: "**/internal/platform/**"
            desc: "port/ ne doit pas importer d'implémentation concrète"
          - pkg: "**/internal/api/**"
            desc: "port/ ne doit pas importer la couche transport"
          - pkg: "**/internal/service/**"
            desc: "port/ ne doit pas importer la couche application"

      service-no-platform:
        files:
          - "**/internal/service/**"
        deny:
          - pkg: "database/sql"
            desc: "service/ ne doit pas importer de driver DB directement"
          - pkg: "**/internal/platform/**"
            desc: "service/ ne doit pas importer d'implémentation concrète"
          - pkg: "**/internal/api/**"
            desc: "service/ ne doit pas dépendre de la couche transport"

      api-no-platform:
        files:
          - "**/internal/api/**"
        deny:
          - pkg: "database/sql"
            desc: "api/ ne doit pas importer de driver DB directement"
          - pkg: "**/internal/platform/**"
            desc: "api/ ne doit pas importer d'implémentation concrète"
```

### Gate CI

```yaml
# .github/workflows/go.yml (extrait)
- name: Architecture lint
  run: golangci-lint run --enable depguard
```

**Règle** : un build qui viole `depguard` est un build rouge. Pas d'exception sans `// ARCH-EXCEPTION`.

---

## 5. Mapping explicite Python ports → Go interfaces

Ce tableau est la **table de correspondance officielle**. Chaque interface Python a un équivalent Go
nommé, testé et enforced.

| Python (Protocol) | Fichier Python | Go interface | Fichier Go | Implémentation concrète Go |
|--------------------|----------------|--------------|------------|---------------------------|
| `DataRepository` | `src/ports/repository.py` | `PlayerRepository` | `internal/port/repository.go` | `internal/platform/duckdb/player_repo.go` |
| _(pas d'équivalent direct)_ | lectures cross-joueur dans repo | `SharedRepository` | `internal/port/shared_repository.go` | `internal/platform/duckdb/shared_repo.go` |
| `HaloAPIPort` (14 méthodes) | `src/ports/api.py` | `HaloClient` (14 méthodes) | `internal/port/halo_client.go` | `internal/platform/halo/haloinfinite/client.go` |
| `create_api_client` (factory) | `src/data/sync/api_factory.py` | `halo.NewClient` (factory) | `internal/platform/halo/factory.go` | dispatche vers `haloinfinite.NewClient` |
| `_SyncProtocol` | `src/data/sync/_protocol.py` | `SyncEngine` | `internal/port/sync_engine.go` | `internal/platform/sync/engine.go` |
| _(éclaté dans sync_meta + MSAL)_ | cache tokens dans `sync_meta` | `TokenStore` | `internal/port/token_store.go` | `internal/platform/auth/token_store.go` |

### Interfaces Go additionnelles (pas d'équivalent Python)

| Go interface | Fichier Go | Rôle | Justification |
|---|---|---|---|
| `MigrationRunner` | `internal/port/migration.go` | Contrat pour le registre de migrations DuckDB | En Python c'est du code procédural ; en Go on le formalise |
| `JobStore` | `internal/port/job_store.go` | Persistance des jobs (statut, résultat, reprise) | En Python c'est en mémoire + Streamlit ; en Go il faut une interface |
| `MediaIndexer` | `internal/port/media_indexer.go` | Contrat d'indexation médias (ffprobe, hash, thumbnails) | Isoler la dépendance `ffprobe` du reste |

---

## 6. Couche `domain/` — logique métier pure

Cette couche est le **cœur du système**. Elle concentre toute la logique de calcul sans aucune
dépendance sur l'IO, le framework HTTP ou le driver DB.

### Contenu

```text
internal/domain/
  models.go          # MatchRow, Participant, Medal, Weapon, Career, Session, etc.
  match_data.go      # MatchData, MatchHistoryItem, RawMatchStats (types retournés par HaloClient)
  match_stats_row.go # MatchStatsRow (48 colonnes), MatchParticipantRow (31 colonnes, incl. MMR)
  skill.go           # SkillResponse, SkillParticipantUpdate (MMR/CSR)
  career.go          # CareerRankData
  pve.go             # PveMatchStatsRow (kills par type d'ennemi)
  customization.go   # PlayerCustomization
  match_filters.go   # MatchFilters, Cursor, SortOrder, HistoryOpts, MatchDataOpts
  sync_scope.go      # SyncScope (96 champs — struct Go avec defaults)
  bitmask.go         # BackfillBitmask (BACKFILL_FLAGS + MatchBits, identité numérique stricte avec Python)
  outcome.go         # Outcome enum (Win=2, Loss=3, Tie=1, DNF=4)

  # Algorithmes métier — portage direct de src/analysis/
  performance.go     # Performance score (percentile relatif)
  lusr.go            # LUSR / TrueSkill2 adapté
  sessions.go        # Détection et labels de sessions
  citations.go       # Règles de citations custom
  series.go          # Détection de séries (win/loss streaks, sprees)
  enrichments.go     # Calculs d'enrichissement post-match
```

> **Correspondance Python** : les types `MatchData`, `MatchHistoryItem`, `MatchStatsRow`,
> `MatchParticipantRow`, `SkillParticipantUpdate`, `CareerRankData`, `PveMatchStatsRow`
> sont définis côté Python dans `src/data/sync/models.py` (~380 lignes).
> Le portage Go doit conserver les mêmes noms de champs et types pour la parité golden values.

### Contraintes

1. **Zéro import externe** (pas de `database/sql`, pas de `net/http`, pas de `encoding/json` sauf pour les types — et encore, préférer des méthodes séparées).
2. **Zéro IO** : aucune lecture fichier, aucun appel réseau, aucune écriture DB.
3. **Fonctions pures** autant que possible : `func ComputePerformanceScore(matches []MatchRow) float64`.
4. **Testable en isolation** : les tests de `domain/` n'ont besoin d'aucun mock, d'aucune connexion.

### Critère de décision : où placer du code ?

| Le code a besoin de… | Package |
|---|---|
| Rien (calcul pur, types) | `domain/` |
| Lire/écrire en DB (via interface) | `service/` |
| Appeler une API externe (via interface) | `service/` |
| Parser du HTTP (request/response) | `api/` |
| Implémenter une interface port (DuckDB, HTTP réel) | `platform/` |
| Assembler et démarrer | `cmd/` |

---

## 7. Testabilité — mocks et golden values

### Règle

> Tout service et tout handler DOIT être testable **sans base de données réelle**.

### Comment

1. Chaque interface dans `port/` est mockable via un struct de test :

```go
// internal/service/career_test.go
type mockPlayerRepo struct {
    matches []domain.MatchRow
}

func (m *mockPlayerRepo) LoadMatches(ctx context.Context, f domain.MatchFilters) ([]domain.MatchRow, error) {
    return m.matches, nil
}
// ... autres méthodes

func TestCareerOverview(t *testing.T) {
    repo := &mockPlayerRepo{matches: fixtures.CorporalMatches}
    svc := service.NewCareerService(repo, &mockSharedRepo{})
    result, err := svc.GetCareerOverview(ctx, "xuid123")
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

2. Les tests `domain/` sont des **tests purs** (pas de mock du tout) :

```go
func TestPerformanceScore(t *testing.T) {
    score := domain.ComputePerformanceScore(fixtures.GoldenMatches)
    assert.InDelta(t, 72.34, score, 0.01)
}
```

3. Les tests `platform/` sont des **tests d'intégration** (DB réelle, fixtures DuckDB) :

```go
func TestPlayerRepo_LoadMatches(t *testing.T) {
    db := testutil.OpenFixtureDB(t, "corporal")
    repo := platform.NewPlayerRepo(db)
    matches, err := repo.LoadMatches(ctx, domain.MatchFilters{})
    require.NoError(t, err)
    assert.Len(t, matches, 142)
}
```

### Pyramide de tests

| Niveau | Quoi | Mock | Quantité |
|--------|------|------|----------|
| `domain/` | Algorithmes purs | Aucun | Beaucoup |
| `service/` | Orchestration | Interfaces mockées | Moyen |
| `api/` | Handlers HTTP | Services mockés | Léger |
| `platform/` | Intégration DB/API | DB ou serveur réel | Ciblé |
| E2E | Playwright | Rien (binaire complet) | Peu |

---

## 8. Exceptions tolérées

Les seules dérogations acceptables à ces règles :

| Exception | Justification | Formalisme requis |
|---|---|---|
| `platform/` importe `platform/` (sous-packages entre eux) | Les implémentations concrètes peuvent se composer | Acceptable par défaut |
| `api/` importe directement un type `domain/` pour un DTO simple | Éviter une couche d'adaptation pour un mapping 1:1 trivial | `// ARCH-EXCEPTION: DTO=domain type` |
| Un `init()` enregistre un driver DB (`sql.Register`) | Idiome Go standard pour les drivers `database/sql` | Uniquement dans `platform/` |
| Constantes techniques dans `platform/` utilisées par `cmd/` | Config, feature flags au démarrage | Acceptable par défaut |

**Tout le reste est interdit.** Si un développeur ou un agent IA veut contourner ces règles,
il doit d'abord modifier CE document et le justifier.

---

## 9. Layout Go révisé (synthèse)

Ce layout remplace la structure proposée dans le PLAN principal.

```text
apps/go-api/
  cmd/
    levelup/              # Binaire unique à sous-commandes
      main.go             # Composition root — seul fichier qui importe platform/
      serve.go            # Sous-commande serve (API HTTP)
      sync.go             # Sous-commande sync/backfill
      migrate.go          # Sous-commande migrate (DuckDB)
      version.go          # --version, build info

  internal/
    domain/               # Logique métier pure — 0 IO, 0 import externe
      models.go           # Types métier : MatchRow, Participant, Medal, etc.
      match_data.go       # MatchData, MatchHistoryItem, RawMatchStats (retours HaloClient)
      match_stats_row.go  # MatchStatsRow (48 col), MatchParticipantRow (31 col, MMR)
      skill.go            # SkillResponse, SkillParticipantUpdate (MMR/CSR)
      career.go           # CareerRankData
      pve.go              # PveMatchStatsRow (kills par type d'ennemi)
      customization.go    # PlayerCustomization
      outcome.go          # Outcome IntEnum
      match_filters.go    # MatchFilters, HistoryOpts, MatchDataOpts, Cursor, SortOrder
      sync_scope.go       # SyncScope (96 champs)
      bitmask.go          # BackfillBitmask (BACKFILL_FLAGS + MatchBits)
      performance.go      # Algorithme performance score
      lusr.go             # Algorithme LUSR/TrueSkill2
      sessions.go         # Détection sessions
      citations.go        # Citations custom
      series.go           # Séries (streaks, sprees)
      enrichments.go      # Enrichissements post-match
      chart/              # Sous-package charting — logique pure
        types.go          # ChartData, MatchSeries, SingleSeriesChartData
        theme.go          # ChartTheme, HaloColors, ThemeColors, PlotConfig, palettes
        options.go        # PlotOptions, defaults
        layout.go         # ApplyHaloStyle(), annotations, records overlay
        downsample.go     # DownsampleForPlot()
        timeseries.go     # KDA, kills/deaths, combat timeseries
        performance.go    # Performance score, LUSR progression
        distributions.go  # Outcomes, histogrammes, score distributions
        maps.go           # Map winrate, mode breakdown
        teammates.go      # Friends impact heatmap, squad cadence
        antagonists.go    # Killer/victim bars, duels
        radar.go          # Participation radar
        matchview.go      # Match timeline, team dominance
        career.go         # Career XP, rank progression charts

    port/                 # Interfaces (ports) — seuls types domain/ autorisés
      repository.go       # PlayerRepository
      shared_repository.go # SharedRepository
      halo_client.go      # HaloClient
      chart_renderer.go   # ChartRenderer (domain/chart → DTO format)
      sync_engine.go      # SyncEngine
      token_store.go      # TokenStore
      job_store.go        # JobStore
      migration.go        # MigrationRunner
      media_indexer.go    # MediaIndexer

    service/              # Orchestration — consomme port/ et domain/
      career.go
      history.go
      explorer.go
      matchview.go
      bootstrap.go
      stats.go
      settings.go
      sync_orchestrator.go
      backfill.go
      charts/             # Orchestration charting — combine repo + domain/chart
        career_charts.go
        history_charts.go
        explorer_charts.go
        matchview_charts.go
        timeseries_charts.go

    api/                  # Transport HTTP — consomme service/ et domain/
      router.go           # Chi router, middleware chain
      middleware/
        auth.go
        cors.go
        request_id.go
        logging.go
      handlers/
        bootstrap.go
        career.go
        history.go
        explorer.go
        matchview.go
        settings.go
        sync.go
        jobs.go
        health.go
      dto/                # DTO OpenAPI request/response
        bootstrap.go
        career.go
        history.go
        chart.go          # PlotlyFigurePayload (contrat Plotly actuel, produit par adapter/plotly)
        # chart_recharts.go  # Futur : RechartsPayload (produit par adapter/recharts)
        # ...
      adapters/            # canonical → DTO (productisation)
        bootstrap_adapter.go
        career_adapter.go
        history_adapter.go

    platform/             # Implémentations concrètes — implémente port/
      duckdb/
        pool.go           # Pool read-only borné + write lease
        player_repo.go    # → PlayerRepository
        shared_repo.go    # → SharedRepository
        metadata_repo.go
        migrations.go     # → MigrationRunner
      halo/
        factory.go        # halo.NewClient(backend, cfg) → port.HaloClient (factory)
        transport.go      # HTTP client, retry, rate limit, circuit breaker
        auth/
          msal.go         # MSAL Go device code flow
          token_store.go  # → TokenStore (DuckDB sync_meta)
          exchange.go     # access_token → spartan_token + clearance
        haloinfinite/
          client.go       # → HaloClient (provider Halo Infinite)
          mapper.go       # payloads natifs → domain canonical
          endpoints.go    # registre d'endpoints 343i
      adapter/              # Adaptateurs de sortie (implémente port/ à la frontière UI)
        plotly/
          renderer.go     # → ChartRenderer : ChartPayload → PlotlyFigurePayload
          trace.go        # Conversion séries → traces Plotly
          layout.go       # Conversion theme/annotations → layout Plotly
        # recharts/        # Futur : ChartPayload → RechartsPayload (même interface)
        # nivo/            # Futur : ChartPayload → NivoPayload (même interface)
      sync/
          engine.go       # → SyncEngine
          transformers/   # normalisation, nettoyage batch
          writers/        # insert shared, player, medals, weapons
      jobs/
          store.go        # → JobStore
      media/
          indexer.go      # → MediaIndexer (ffprobe, hash, thumbs)
      config/
          config.go       # Struct Config + JSON + env vars
      logging/
          setup.go        # slog configuration
```

---

## 10. Checklist par sprint

Avant de merger tout sprint Go, vérifier :

- [ ] Aucun handler n'importe `database/sql` ni `platform/`
- [ ] Aucun service n'importe `platform/` ni `api/`
- [ ] Les nouveaux types métier sont dans `domain/`, pas dans `service/` ou `api/`
- [ ] Toute nouvelle frontière IO a une interface dans `port/`
- [ ] Les tests du service utilisent des mocks, pas une DB réelle
- [ ] Les tests du domain n'utilisent aucun mock
- [ ] `golangci-lint run --enable depguard` passe sans erreur
- [ ] `domain/chart/` et `service/charts/` compilent **sans** `adapter/plotly/` dans le module
- [ ] Pas de `var global` de dépendance métier dans les packages

---

## 11. Couche d'abstraction charting — graphiques et tableaux

### Inventaire Python actuel

Le backend Python ne se contente pas de transmettre des données brutes : il **calcule les figures
Plotly** côté serveur et envoie du JSON structuré au frontend. Cette couche représente :

| Module Python | Fichiers | LOC | Rôle |
|---|---|---|---|
| `src/visualization/` | 47 | ~12 000 | ~80 fonctions `plot_*` / `create_*` (figures Plotly) |
| `src/visualization/_chart_series.py` | 1 | ~300 | Data models : `ChartData`, `MatchSeries`, `SingleSeriesChartData`, `ChartTheme`, `PlotOptions` |
| `src/visualization/theme.py` | 1 | ~60 | `apply_halo_plot_style()` — dark theme Waypoint centralisé |
| `src/visualization/_compat.py` | 1 | ~80 | Polars ↔ Pandas boundary + `smart_scatter()` |
| `src/config.py` (partiel) | 1 | ~120 | `HaloColors`, `ThemeColors`, `OKABE_ITO_PALETTE`, `PlotConfig` |
| `src/ui/components/` | 13 | ~1 200 | Radars, annotations, KPI, médias |
| `apps/api/app/schemas/common.py` | 1 | ~10 | `PlotlyFigurePayload` (sérialisation API) |

**Ce n'est pas éliminé par la migration** — c'est **porté** : le backend Go calcule les séries,
seuil, annotations et layout ; le frontend React les rend via `react-plotly.js`.

### Architecture charting Go — 4 couches (port + adapter)

> [!IMPORTANT]
> Le domaine chart ne connaît **pas** Plotly. Les builders produisent un `ChartPayload`
> (port renderer-agnostic). Un adapter convertit ensuite ce payload au format cible
> (Plotly aujourd'hui, Recharts/Nivo/ECharts demain) **sans toucher le backend**.

```text
┌─────────────────────────────────────────────────────────────┐
│  api/dto/              : PlotlyFigurePayload JSON           │
│                          (contrat HTTP vers le frontend)     │
└────────────────────────────┬────────────────────────────────┘
                             │ produit par
┌────────────────────────────▼────────────────────────────────┐
│  adapter/plotly/       : PlotlyRenderer                     │
│    Render(ChartPayload) → PlotlyFigurePayload               │
│    Convertit séries/annotations/theme → {data, layout}      │
│    Plotly JSON. Demain : adapter/recharts/, adapter/nivo/    │
└────────────────────────────┬────────────────────────────────┘
                             │ consomme
┌────────────────────────────▼────────────────────────────────┐
│  service/charts/       : orchestration — combine données    │
│                          repo + algorithmes domain/chart    │
│                          → produit des ChartPayload          │
└────────────────────────────┬────────────────────────────────┘
                             │ utilise
┌────────────────────────────▼────────────────────────────────┐
│  domain/chart/         : LOGIQUE PURE — 0 IO                │
│  - port : ChartPayload (interface renderer-agnostic)        │
│  - port : ChartRenderer (interface de rendu)                │
│  - types : ChartData, MatchSeries, SingleSeriesChartData    │
│  - config : ChartTheme, PlotOptions, palette                │
│  - builders : BuildKDChart(), BuildPerformanceChart(), etc.  │
│  - layout : annotations, records overlay, thresholds        │
│  - downsample : DownsampleForPlot()                         │
└─────────────────────────────────────────────────────────────┘
```

### Port ChartPayload — contrat renderer-agnostic (dans `domain/chart/`)

```go
// domain/chart/payload.go
package chart

// ChartPayload — contrat de sortie des builders, agnostique du renderer.
// Contient les données calculées + metadata de présentation, mais AUCUNE
// référence à Plotly, Recharts, Nivo ou toute lib de visualisation.
//
// Tout builder renvoie un *ChartPayload ; un ChartRenderer le convertit
// au format attendu par le frontend.
type ChartPayload struct {
    // Identifier
    ChartType  ChartType  `json:"chart_type"`  // timeseries, bar, heatmap, radar, gauge, distribution
    Title      string     `json:"title"`

    // Données
    Series      []MatchSeries          `json:"series"`
    XLabels     []string               `json:"x_labels,omitempty"`
    BarMode     string                 `json:"barmode,omitempty"` // "group" | "overlay" | "categorical"
    IsNegative  bool                   `json:"is_negative,omitempty"`

    // Metadata de présentation (renderer-agnostic)
    Annotations  []ChartAnnotation               `json:"annotations,omitempty"`
    Thresholds   []ChartThreshold                 `json:"thresholds,omitempty"`
    Records      map[string]*float64              `json:"global_records,omitempty"`
    PerMapRecords map[string]map[string]*float64  `json:"per_map_records,omitempty"`

    // Options de rendu
    Options  PlotOptions `json:"options"`
}

// ChartType — type de graphique (enum string pour extensibilité).
type ChartType string

const (
    ChartTimeseries   ChartType = "timeseries"
    ChartBar          ChartType = "bar"
    ChartHeatmap      ChartType = "heatmap"
    ChartRadar        ChartType = "radar"
    ChartGauge        ChartType = "gauge"
    ChartDistribution ChartType = "distribution"
    ChartScatter      ChartType = "scatter"
)

// ChartAnnotation — annotation positionnée sur un graphique.
type ChartAnnotation struct {
    X     any    `json:"x"`
    Y     any    `json:"y"`
    Text  string `json:"text"`
    Style string `json:"style,omitempty"` // "record", "milestone", "label"
}

// ChartThreshold — ligne horizontale de référence (moyenne, seuil, record).
type ChartThreshold struct {
    Value float64 `json:"value"`
    Label string  `json:"label"`
    Style string  `json:"style"` // "average", "record", "zero"
}
```

### Port ChartRenderer — interface de rendu (dans `domain/chart/`)

```go
// domain/chart/renderer.go
package chart

// ChartRenderer — port de rendu, implémenté par chaque adapter.
// Le backend injecte l'implémentation concrète au démarrage (DI).
type ChartRenderer interface {
    // Render convertit un ChartPayload en DTO sérialisable pour le frontend.
    // Le type de retour est any car le DTO concret dépend de l'adapter.
    Render(payload *ChartPayload) (any, error)
}
```

### Data models internes (dans `domain/chart/`)

```go
// domain/chart/types.go
package chart

// MatchSeries — une série de données pour un graphique.
type MatchSeries struct {
    Name     string     `json:"name"`
    X        []int      `json:"x"`
    Y        []*float64 `json:"y"`
    Color    string     `json:"color"`
    MapNames []*string  `json:"map_names,omitempty"`
}

// SingleSeriesChartData — container pour un graphe solo (KDA, per-minute).
type SingleSeriesChartData struct {
    X       []any      `json:"x"`
    Y       []*float64 `json:"y"`
    YSmooth []*float64 `json:"y_smooth,omitempty"`
    Height  int        `json:"height"`
    Title   string     `json:"title"`
}

// ChartTheme — thème visuel centralisé (25+ champs, renderer-agnostic).
type ChartTheme struct {
    BgPlot       string `json:"bg_plot"`
    FontColor    string `json:"font_color"`
    GridColor    string `json:"grid_color"`
    SoloColor    string `json:"solo_color"`
    // ... 20+ champs supplémentaires identiques au Python
}

// PlotOptions — options de rendu d'un graphique.
type PlotOptions struct {
    Lang        string     `json:"lang"`
    Smooth      bool       `json:"smooth"`
    HeightPx    int        `json:"height_px"`
    ShowRecords bool       `json:"show_records"`
    IsNegative  bool       `json:"is_negative"`
    Theme       ChartTheme `json:"theme"`
}
```

### Adapter Plotly (dans `adapter/plotly/`)

```go
// adapter/plotly/renderer.go
package plotly

import "levelup/internal/domain/chart"

// Renderer — adapter Plotly : convertit ChartPayload → PlotlyFigurePayload.
// Implémente chart.ChartRenderer.
//
// Demain, un adapter/recharts/renderer.go ou adapter/nivo/renderer.go
// implémentera la même interface sans modifier le domain ni les services.
type Renderer struct {
    defaultConfig string // "clean" | "static"
}

func NewRenderer(defaultConfig string) *Renderer {
    return &Renderer{defaultConfig: defaultConfig}
}

// Render convertit un ChartPayload renderer-agnostic en PlotlyFigurePayload.
func (r *Renderer) Render(p *chart.ChartPayload) (any, error) {
    fig := buildPlotlyFigure(p) // data[] + layout{} + annotations
    return &PlotlyFigurePayload{
        Figure:    fig,
        ConfigKey: r.defaultConfig,
    }, nil
}

// buildPlotlyFigure — construit le JSON Plotly {data, layout} à partir du payload.
// Applique le thème Halo, les annotations, records, thresholds.
func buildPlotlyFigure(p *chart.ChartPayload) map[string]any {
    // ... conversion séries → traces Plotly, theme → layout, annotations → shapes
    return map[string]any{"data": traces, "layout": layout}
}
```

### DTO API (dans `api/dto/`)

```go
// api/dto/chart.go
package dto

// PlotlyFigurePayload — figure Plotly sérialisée pour react-plotly.js.
// Produit par adapter/plotly/Renderer, consommé par le frontend actuel.
//
// Si le frontend migre vers Recharts/Nivo, un nouveau DTO remplacera celui-ci
// sans impacter domain/chart/ ni service/charts/.
type PlotlyFigurePayload struct {
    Figure      map[string]any `json:"figure"`
    ConfigKey   string         `json:"config_key"`
    RevisionKey *string        `json:"revision_key,omitempty"`
}
```

### Injection de dépendance (dans `cmd/`)

```go
// cmd/server/main.go (extrait)
func main() {
    // Injection de l'adapter charting — changeable sans toucher le domain
    chartRenderer := plotly.NewRenderer("clean")
    chartService := charts.NewService(repos, chartRenderer)
    // ...
}
```

### Règles charting

1. **Les builders vivent dans `domain/chart/`** — logique pure, zéro IO, testable sans DB.
   Ils retournent `*ChartPayload`, jamais un format de rendu spécifique.

2. **`domain/chart/` ne connaît pas Plotly** — aucun import, aucune référence à des
   traces/layout Plotly. Le domain produit des séries + metadata + annotations
   en termes métier. La conversion en format de rendu est dans l'adapter.

3. **Un seul adapter actif est injecté au démarrage** (DI dans `cmd/`). Aujourd'hui :
   `adapter/plotly/`. Demain : `adapter/recharts/` ou `adapter/nivo/` sans
   toucher ni `domain/chart/` ni `service/charts/`.

4. **Le thème (`ChartTheme`, `HaloColors`, `ThemeColors`)** est dans `domain/chart/theme.go` —
   il exprime des couleurs et styles en termes abstraits. L'adapter les traduit
   en directives spécifiques au renderer (template Plotly, props Recharts, etc.).

5. **`PlotlyFigurePayload`** est le contrat HTTP actuel vers le frontend React.
   Il vit dans `api/dto/` et n'est connu que de `adapter/plotly/` et des handlers.
   Si le frontend migre, un nouveau DTO le remplace — le domain ne change pas.

6. **Le downsampling** (`ChartPayload.Downsample()`) est une méthode domain — pas un middleware.

7. **La palette Okabe-Ito** (5 couleurs daltonien-safe) est une constante `domain/chart/` :
   ```go
   var OkabeItoPalette = [5]string{"#56B4E9", "#E69F00", "#009E73", "#CC79A7", "#D55E00"}
   ```

8. **Pas d'import `go-plotly`** ni de lib Go de charting dans le backend.
   L'adapter Plotly construit des `map[string]any` JSON-compatibles.
   Le rendu visuel est 100% frontend.

9. **Test de découplage** : il doit être possible de compiler `domain/chart/` et
   `service/charts/` **sans** que `adapter/plotly/` soit dans le module. Si ce
   test échoue → couplage interdit, à corriger immédiatement.

### Couverture des ~80 fonctions Python

Les 80 fonctions `plot_*` Python sont regroupées par surface produit :

| Surface | Fonctions Python (exemples) | Go package |
|---|---|---|
| Timeseries (KDA, kills, deaths) | `plot_kd_timeseries`, `plot_kills_deaths_bars` | `domain/chart/timeseries.go` |
| Performance | `plot_performance_score`, `plot_lusr_progression` | `domain/chart/performance.go` |
| Distributions | `plot_outcome_distribution`, `plot_score_histogram` | `domain/chart/distributions.go` |
| Maps & modes | `plot_map_winrate`, `plot_mode_breakdown` | `domain/chart/maps.go` |
| Coéquipiers | `plot_friends_impact_heatmap`, `plot_squad_cadence` | `domain/chart/teammates.go` |
| Adversaires (K/V) | `plot_antagonist_bars`, `plot_antagonist_duels` | `domain/chart/antagonists.go` |
| Participation (radar) | `create_participation_radar` | `domain/chart/radar.go` |
| Match view | `plot_match_impact_timeline`, `plot_team_dominance` | `domain/chart/matchview.go` |
| Career / XP | `plot_career_xp_chart` | `domain/chart/career.go` |

Chaque fichier Go correspond à un module Python de `src/visualization/`.

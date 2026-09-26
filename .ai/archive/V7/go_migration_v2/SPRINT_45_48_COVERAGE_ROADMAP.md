# Sprints 45–48 — Montée à 70% de couverture (Phase 10)

> **Document d'exécution opérationnel** pour les sprints 45 à 48.
> Lisible par un humain ou une IA qui reprend sans contexte préalable.
>
> **Créé le** : 2026-04-17
> **Branche** : `feat/live-golden-values` (worktree `LevelUp-go-migration`)
> **Déclencheur** : audit 2026-04-17 — coverage global réel = **13.4%** (la CI affichait 50% parce qu'elle ne mesurait qu'un sous-ensemble CGO-free).
>
> **Vue d'ensemble dans la roadmap** : voir [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) section "Phase 10 — Consolidation qualité".

---

## 0. Contexte de départ (état au 2026-04-17)

### Couverture mesurée

```bash
cd apps/go-api
go tool cover -func=coverage.out | tail -1
# total: (statements) 13.4%
```

### Distribution par package (approximative, à reconfirmer en S45)

| Package | Coverage estimé | Taille | Priorité |
|---|---|---|---|
| `internal/domain/` (hors title) | ~80% | moyen | basse (déjà bon) |
| `internal/analysis/` | ~70% | gros | basse (bien couvert) |
| `contracttest/` | ~60% (test yaml) | petit | basse |
| `internal/config/` | ~50% | petit | basse |
| `internal/service/` | ~40% (partiel) | gros | moyenne |
| `internal/api/handlers/` | **~15%** (8/21 handlers) | très gros | **haute — S46** |
| `internal/api/middleware/` | **~20%** | moyen | **haute — S46** |
| `internal/sync/` (transforms, engine) | **~10%** | très gros | **haute — S47** |
| `internal/sync/writes.go` | **0%** | gros | **critique — S47** |
| `internal/migration/steps_*.go` | **0%** | très gros | **critique — S47** |
| `internal/platform/duckdb/*_repo.go` | **~20%** | très gros | **haute — S47** |
| `internal/validation/` | **0%** | moyen | **haute — S48** |
| `internal/ops/` | **0%** | petit | moyenne — S48 |
| `internal/notify/` | **0%** | petit | faible — S48 |
| `internal/domain/title/` | ~60% | petit | basse |
| `internal/api/gen/` | N/A | gros | **EXCLU** (code généré) |
| `cmd/msal-poc/` | 0% | petit | **EXCLU** (POC jetable) |
| `cmd/levelup/` | 0% | moyen | à décider en S45 |

### Raison du faux positif actuel

Fichier `.github/workflows/ci.yml` ligne ~244 :

```yaml
go test -coverprofile=coverage.out -covermode=atomic \
  ./internal/domain/... ./internal/analysis/... ./contracttest/... \
  -timeout 60s -count=1
```

→ mesure uniquement les 3 packages déjà bien couverts. Le seuil "50%" est donc mécaniquement atteint mais ne reflète rien.

### Ce que le plan **ne** change **pas**

- Les tests **golden** existants (`tests/golden/golden_test.go`) restent et deviendront des contributeurs de couverture via `-coverpkg=./...` (Sprint 45).
- Les tests **contrat** (`contracttest/`) restent inchangés.
- Les tests **E2E Playwright** frontend restent dans leur workflow dédié (hors scope Go coverage).
- Aucun test ne sera **supprimé** ni mocké pour gonfler la métrique.

---

## 1. Définitions de "test" utilisées dans ce plan

| Niveau | Ce que c'est dans ce repo | Où | Contribue à `go test -cover` ? |
|---|---|---|---|
| **Unitaire pur** | Fonction pure, zéro IO, zéro DB | `internal/**/*_test.go` | ✅ oui |
| **Unitaire avec DB in-memory** | DuckDB `:memory:` + schéma seed + SQL réel | `internal/**/*_test.go` avec fixture | ✅ oui |
| **Handler `httptest`** | Mock `port.Services`, lance le handler via `httptest.NewRecorder()` | `internal/api/handlers/*_test.go` | ✅ oui |
| **Contrat** | Parsing OpenAPI YAML + vérif routes chi | `contracttest/`, `internal/api/contract_test.go` | ✅ oui |
| **Golden / Non-régression** | Appel handler + diff JSON vs snapshot | `tests/golden/golden_test.go` | ✅ oui **via `-coverpkg=./...`** (Sprint 45) |
| **Intégration DB réelle** | Vrai fichier DuckDB (copie de fixture) | `internal/migration/*_test.go` avec `t.TempDir()` | ✅ oui |
| **E2E bout-en-bout** | Serveur `cmd/server` up + requêtes HTTP + frontend | `apps/web/e2e/` (Playwright) | ❌ non (hors périmètre Go) |

**Décision** : **on ne monte PAS à 70% en écrivant des tests E2E Go**. Le ROI est médiocre sur une API JSON stateless et ça demande un serveur + DB réel + fixtures lourdes. On atteint 70% avec unitaires + handler httptest + intégration DB in-memory + golden. Si un besoin E2E émerge plus tard, c'est un projet séparé.

---

## 2. Sprint 45 — Infra coverage réelle (3–4j)

### Objectif
Mesurer la vérité avant d'écrire le moindre test. Un chiffre honnête à 14% > chiffre faux à 50%.

### Livrable 1 : nouveau job `go-coverage` dans `ci.yml`

**Remplacer** le job actuel par :

```yaml
  go-coverage:
    name: Go Coverage (seuil ratchet Phase 10)
    runs-on: ubuntu-latest  # ou windows-latest si CGO Windows obligatoire — à trancher
    defaults:
      run:
        working-directory: apps/go-api
    steps:
      - uses: actions/checkout@v4

      - name: Install CGO toolchain (Linux)
        run: sudo apt-get install -y gcc libc6-dev

      - uses: actions/setup-go@v5
        with:
          go-version-file: apps/go-api/go.mod
          cache-dependency-path: apps/go-api/go.sum

      - name: Run all tests with coverage (CGO on, full tree)
        env:
          CGO_ENABLED: "1"
          LEVELUP_DEMO_MODE: "true"
        run: |
          go test -coverprofile=coverage.out.raw -covermode=atomic \
            -coverpkg=./... ./... -timeout 5m -count=1

      - name: Filter excluded packages
        run: |
          # Voir scripts/coverage_filter.sh — exclut gen/, cmd/msal-poc/, etc.
          bash scripts/coverage_filter.sh coverage.out.raw > coverage.out

      - name: Check ratchet threshold
        run: bash scripts/coverage_check.sh coverage.out coverage_baseline.txt

      - name: Generate HTML report
        if: always()
        run: go tool cover -html=coverage.out -o coverage.html

      - name: Upload coverage artifacts
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: go-coverage-report
          path: |
            apps/go-api/coverage.out
            apps/go-api/coverage.html
```

**Décision à prendre en S45** : runner `ubuntu-latest` (plus rapide, nécessite toolchain gcc) ou `windows-latest` (identique à prod mais 3× plus lent CI). Recommandation : `ubuntu-latest` pour le job coverage, `windows-latest` reste sur le job `go-build`/`go-test` pour catch les DLL issues.

### Livrable 2 : `scripts/coverage_filter.sh`

```bash
#!/usr/bin/env bash
# Filtre les packages exclus du profil de couverture.
# Usage: coverage_filter.sh <input.out> > <output.out>
set -euo pipefail
INPUT="${1:?usage: $0 <profile>}"

# Patterns exclus (1 par ligne, regex Go-path)
EXCLUDE=(
  'levelup/go-api/internal/api/gen/'      # Code généré oapi-codegen
  'levelup/go-api/cmd/msal-poc/'          # POC jetable
  # cmd/levelup/ — à décider en S45 selon discussion
)

# Garder l'en-tête "mode: atomic"
head -1 "$INPUT"

# Filtrer les lignes
tail -n +2 "$INPUT" | while IFS= read -r line; do
  keep=1
  for pat in "${EXCLUDE[@]}"; do
    if [[ "$line" == *"$pat"* ]]; then
      keep=0
      break
    fi
  done
  [[ $keep -eq 1 ]] && echo "$line"
done
```

### Livrable 3 : `scripts/coverage_check.sh` (ratchet)

```bash
#!/usr/bin/env bash
# Vérifie que le coverage actuel >= baseline.
# Si supérieur, met à jour le baseline (sur push main uniquement).
set -euo pipefail

PROFILE="${1:?profile required}"
BASELINE="${2:?baseline required}"

current=$(go tool cover -func="$PROFILE" | awk '/^total:/ { gsub("%",""); print $3 }')
baseline=$(cat "$BASELINE" | head -1 | tr -d '%')

echo "Coverage current : ${current}%"
echo "Coverage baseline: ${baseline}%"

# Tolérance 0.1% pour éviter les faux positifs de flakiness
awk -v c="$current" -v b="$baseline" 'BEGIN { exit (c + 0.1 < b) }' || {
  echo "❌ Coverage ${current}% < baseline ${baseline}% (ratchet violation)"
  exit 1
}

echo "✅ Coverage ${current}% ≥ baseline ${baseline}%"
```

### Livrable 4 : `apps/go-api/coverage_baseline.txt`

Format : première ligne = pourcentage global. Lignes suivantes : coverage par package (format `go tool cover -func=` grep par package). Commit initial = valeur exacte mesurée en S45.

### Livrable 5 : paliers intermédiaires

À chaque fin de sprint, update le baseline :

| Sprint | Baseline après | Seuil CI |
|---|---|---|
| Fin S45 | ~15% (reflète réalité après filtrage gen/) | 15% |
| Fin S46 | ~36% (handlers + middlewares couverts) | 35% |
| Fin S47 | ~56% (sync + migrations + platform couverts) | 55% |
| Fin S48 | ≥70% | **70%** |

### Tâches Sprint 45 (checklist)

- [ ] Écrire `scripts/coverage_filter.sh` + `scripts/coverage_check.sh`
- [ ] Réécrire `.github/workflows/ci.yml` job `go-coverage` (CGO=1, `-coverpkg=./...`, `./...`)
- [ ] Lancer localement : `CGO_ENABLED=1 go test -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./... -timeout 5m`
- [ ] Mesurer vrai baseline, committer dans `coverage_baseline.txt`
- [ ] Ouvrir PR avec seuil CI = baseline mesuré (probablement 14-15%)
- [ ] Documenter commandes dans `docs/testing.md` (à créer)
- [ ] Entrée `.ai/thought_log.md` avec baseline exact et packages les plus bas

### Gate Sprint 45
- [ ] CI mesure `./...` complet avec CGO activé
- [ ] `-coverpkg=./...` activé pour que golden/contract remontent dans les packages cibles
- [ ] Baseline honnête committé
- [ ] Ratchet en place (jamais de régression acceptée)

---

## 3. Sprint 46 — Handlers HTTP + middlewares (6–8j)

### Objectif
`internal/api/handlers/` ≥ **75%**, `internal/api/middleware/` ≥ **80%**.
Globalement : atteindre **35%**.

### Prérequis S45

- `port.Services` interface existe déjà (Sprint 37) — à confirmer en lisant `internal/port/services.go`.
- Si la surface de `port.Services` n'expose pas tous les services utilisés par les handlers → tâche blocante : l'étendre.

### Livrable 1 : `testutil` partagé

Fichier `internal/api/handlers/testutil/mock_services.go` :

```go
package testutil

import (
    "context"
    "testing"

    "levelup/go-api/internal/port"
    "levelup/go-api/internal/domain"
)

// MockServices implémente port.Services avec des fonctions-champs pour mock ciblé.
type MockServices struct {
    GetBootstrapFn    func(ctx context.Context, req domain.BootstrapRequest) (*domain.BootstrapResponse, error)
    GetHomeFn         func(ctx context.Context, req domain.HomeRequest) (*domain.HomeResponse, error)
    // ... une ligne par méthode de port.Services
}

func (m *MockServices) GetBootstrap(ctx context.Context, req domain.BootstrapRequest) (*domain.BootstrapResponse, error) {
    if m.GetBootstrapFn == nil {
        panic("GetBootstrapFn not set in test")
    }
    return m.GetBootstrapFn(ctx, req)
}
// ... pareil pour chaque méthode

// NewMockServices retourne un MockServices avec toutes les fonctions qui panic si appelées.
func NewMockServices() *MockServices {
    return &MockServices{}
}
```

Fichier `internal/api/handlers/testutil/http.go` :

```go
package testutil

import (
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func DoRequest(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
    t.Helper()
    var r io.Reader
    if body != nil {
        b, err := json.Marshal(body)
        if err != nil {
            t.Fatal(err)
        }
        r = bytes.NewReader(b)
    }
    req := httptest.NewRequest(method, path, r)
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, req)
    return rec
}

func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
    t.Helper()
    if rec.Code != want {
        t.Errorf("status = %d, want %d, body=%s", rec.Code, want, rec.Body.String())
    }
}

func AssertJSONField(t *testing.T, rec *httptest.ResponseRecorder, path string, want any) {
    t.Helper()
    var got map[string]any
    if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
        t.Fatalf("body is not JSON: %v — body=%s", err, rec.Body.String())
    }
    // walk path "a.b.c"
    current := any(got)
    for _, part := range strings.Split(path, ".") {
        m, ok := current.(map[string]any)
        if !ok {
            t.Fatalf("path %s: expected map at %q, got %T", path, part, current)
        }
        current = m[part]
    }
    if current != want {
        t.Errorf("field %s = %v, want %v", path, current, want)
    }
}
```

### Livrable 2 : 13 fichiers de test handler

**Pattern canonique** (exemple pour `sessions_test.go`) :

```go
package handlers_test

import (
    "context"
    "errors"
    "net/http"
    "testing"

    "levelup/go-api/internal/api/handlers"
    "levelup/go-api/internal/api/handlers/testutil"
    "levelup/go-api/internal/domain"
)

func TestListSessions_OK(t *testing.T) {
    svc := testutil.NewMockServices()
    svc.ListSessionsFn = func(ctx context.Context, req domain.SessionsRequest) (*domain.SessionsResponse, error) {
        return &domain.SessionsResponse{
            Sessions: []domain.Session{{ID: "s1", MatchCount: 4}},
        }, nil
    }
    h := handlers.NewSessionsHandler(svc)
    rec := testutil.DoRequest(t, h, "GET", "/sessions?player_slug=gc&limit=10", nil)
    testutil.AssertStatus(t, rec, http.StatusOK)
    testutil.AssertJSONField(t, rec, "sessions.0.id", "s1")
}

func TestListSessions_BadRequest(t *testing.T) {
    svc := testutil.NewMockServices()
    h := handlers.NewSessionsHandler(svc)
    rec := testutil.DoRequest(t, h, "GET", "/sessions", nil) // pas de player_slug
    testutil.AssertStatus(t, rec, http.StatusBadRequest)
}

func TestListSessions_NotFound(t *testing.T) {
    svc := testutil.NewMockServices()
    svc.ListSessionsFn = func(ctx context.Context, req domain.SessionsRequest) (*domain.SessionsResponse, error) {
        return nil, domain.ErrPlayerNotFound
    }
    h := handlers.NewSessionsHandler(svc)
    rec := testutil.DoRequest(t, h, "GET", "/sessions?player_slug=unknown", nil)
    testutil.AssertStatus(t, rec, http.StatusNotFound)
}

func TestListSessions_InternalError(t *testing.T) {
    svc := testutil.NewMockServices()
    svc.ListSessionsFn = func(ctx context.Context, req domain.SessionsRequest) (*domain.SessionsResponse, error) {
        return nil, errors.New("db exploded")
    }
    h := handlers.NewSessionsHandler(svc)
    rec := testutil.DoRequest(t, h, "GET", "/sessions?player_slug=gc", nil)
    testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}
```

**Les 13 handlers à couvrir** (matrice — cocher au fur et à mesure) :

| Handler | Fichier test | OK | 400 | 404 | 500 | Edge |
|---|---|:-:|:-:|:-:|:-:|:-:|
| sessions | `sessions_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| home | `home_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| citations | `citations_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| explorer | `explorer_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| media | `media_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| last_match | `last_match_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| teammates | `teammates_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| session_compare | `session_compare_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| timeseries | `timeseries_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| match_history | `match_history_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| bootstrap | `bootstrap_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| auth | `auth_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| jobs | `jobs_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |
| sync_handler | `sync_handler_test.go` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ |

### Livrable 3 : 5 tests middleware

| Middleware | Cas à couvrir |
|---|---|
| `request_id` | propagation header, génération si absent, format UUID |
| `cors` | Origin autorisée, refusée, preflight OPTIONS |
| `rate_limit` | sous seuil, au seuil, après reset |
| `session` | cookie valid, expired, tampered, missing |
| `shadow` | mode `"off"`, `"go-only"`, `"both"` — vérifier les appels backend |

### Tâches Sprint 46 (checklist)

- [ ] Lire `internal/port/services.go` pour lister les méthodes à mocker
- [ ] Créer `testutil/mock_services.go` + `testutil/http.go`
- [ ] 14 fichiers `*_test.go` (13 handlers nouveaux + étendre `stats_test.go` / `squad_test.go` / `gamertag_test.go` / `career_test.go` / `filters_test.go` / `match_view_test.go` qui existent déjà mais partiellement)
- [ ] 5 fichiers middleware `*_test.go` (⚠️ `csrf_test.go` et `security_audit_test.go` existent déjà)
- [ ] `helpers_test.go` pour `internal/api/handlers/helpers.go`
- [ ] Lancer `go test ./internal/api/...` → vérifier couverture package `handlers` ≥ 75%, `middleware` ≥ 80%
- [ ] Mise à jour `coverage_baseline.txt` → ~35-36%
- [ ] CI seuil remonté à 35%
- [ ] Entrée `.ai/thought_log.md`

### Gate Sprint 46
- [ ] Tous les handlers publics testés avec ≥4 cas chacun
- [ ] Tous les middlewares testés
- [ ] `internal/api/handlers/` ≥ 75%, `internal/api/middleware/` ≥ 80%
- [ ] Coverage global ≥ 35%
- [ ] Aucun test n'ouvre de vraie DB (tout mocké via `port.Services`)

---

## 4. Sprint 47 — Sync + migrations + platform/duckdb (8–10j)

### Objectif
Le cœur du risque : 0% → **≥ 70%** sur `sync/`, **≥ 75%** sur `migration/`, **≥ 70%** sur `platform/duckdb/`.
Globalement : atteindre **55%**.

### Stratégie : fixtures DuckDB in-memory

Utiliser `database/sql` avec driver duckdb + `":memory:"` + appliquer le schéma depuis `migration/steps_*.go` (qu'on teste donc par ricochet).

### Livrable 1 : `internal/sync/testutil/fixture.go`

```go
package testutil

import (
    "context"
    "database/sql"
    "testing"

    _ "github.com/marcboeker/go-duckdb" // ou le driver utilisé

    "levelup/go-api/internal/migration"
)

// NewInMemoryShared retourne un *sql.DB DuckDB in-memory avec le schéma shared_matches v6 appliqué
// et un match seed minimal.
func NewInMemoryShared(t *testing.T) *sql.DB {
    t.Helper()
    db, err := sql.Open("duckdb", "")
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { db.Close() })

    ctx := context.Background()
    if err := migration.ApplySharedSchema(ctx, db); err != nil {
        t.Fatalf("apply shared schema: %v", err)
    }
    if err := seedMinimalMatch(ctx, db); err != nil {
        t.Fatalf("seed match: %v", err)
    }
    return db
}

func seedMinimalMatch(ctx context.Context, db *sql.DB) error {
    // Insert 1 match_registry + 8 participants + 3 medals + 2 xuid_aliases
    // Valeurs fixées pour tests déterministes.
    // ...
}
```

### Livrable 2 : matrice des tests à écrire

**`internal/sync/writes_test.go`** — 9 fonctions à 0% actuellement :

| Fonction | Cas de test |
|---|---|
| `InsertRegistryIfNotExists` | insert neuf, doublon `match_id` ignoré, conflit metadata |
| `InsertParticipants` | batch 8, xuid canonique, MMR propagé, vide |
| `InsertMedals` | 3 médailles, idempotence re-run |
| `UpsertXUIDAlias` | insert, update, unicité |
| `UpsertPlayerEnrichment` | nouveau, mise à jour, performance_score calculé |
| `SetSyncMeta` | key neuve, overwrite valeur |
| `InsertWeaponKills` | batch 5, FK vérifiée, weapon_id UBIGINT |
| `MarkWeaponKillsDone` | bit 18 de `backfill_completed` bien posé |
| `nullStr` | nil, empty, value |

**`internal/sync/transforms_test.go`** — compléter les 11 fonctions à 0% :

| Fonction | Cas |
|---|---|
| `findCoreStats` | stats présentes / absentes / null |
| `isRankedPlaylist` | ranked, social, unknown, nil |
| `isFirefightMatch` | firefight, arena, BTB |
| `extractTeamScoresByID` | 2 teams, 4 teams, 0 teams |
| `asString`, `strPtr`, `coalesceStrPtr` | nil, empty, value |
| `intPtrFrom`, `floatPtrFrom`, `intFrom`, `int64From` | overflow, underflow, nil |

**`internal/migration/`** :

| Test | Couverture attendue |
|---|---|
| `TestApplySharedSchema` (steps_shared) | 36 steps sur DB vide → toutes tables + vues présentes |
| `TestApplySharedSchemaIdempotent` | 2× apply → no-op à la 2e exécution |
| `TestApplyPlayerSchema` (steps_player) | toutes tables player v6 présentes |
| `TestApplyPveSchema` (steps_shared_pve) | table `pve_match_stats` + index |

**`internal/platform/duckdb/`** — extension de `repo_test.go` existant :

| Test | Ce qu'on vérifie |
|---|---|
| `TestMatchHistoryRepo_List` | pagination, filtres, empty |
| `TestCareerRepo_Progression` | jalons rangs |
| `TestFiltersRepo_Playlists` | playlists actives + archivées |
| `TestGamertagRepo_Resolve` | xuid → gamertag, alias, not found |
| `TestExplorerRepo_SearchMatches` | combinaison filtres |
| `TestHomeRepo_Aggregate` | stats jour / semaine / mois |
| `TestSessionsRepo_List` | sessions avec / sans matchs |
| `TestStatsRepo_Snapshot` | cumul, moyenne, variance |
| `TestCitationsRepo` | list + recompute |
| `TestMediaRepo` | list + associations |

**`internal/sync/lease_test.go`** :

```go
func TestWriteLease_Exclusive(t *testing.T) {
    // 2 goroutines tentent AcquireLease sur même path
    // → exactement 1 réussit, l'autre timeout
}
```

**`internal/sync/engine_test.go`** :

Test bout-en-bout avec API Halo mockée (via `halo.Provider` interface — à vérifier qu'elle existe ou à introduire) + DB in-memory.

### Tâches Sprint 47 (checklist)

- [ ] Créer `internal/sync/testutil/fixture.go` (fonctions `NewInMemoryShared`, `NewInMemoryPlayer`, `SeedMatch`, etc.)
- [ ] Exposer `migration.ApplySharedSchema(ctx, *sql.DB)` si pas déjà public (refactor minimal nécessaire)
- [ ] 9 tests `writes_test.go`
- [ ] 11 tests étendus `transforms_test.go`
- [ ] 4 tests `migration/*_test.go` (schémas complets)
- [ ] 10 tests `platform/duckdb/*_repo_test.go`
- [ ] `lease_test.go` (concurrence write lease)
- [ ] `engine_test.go` (cycle sync minimal)
- [ ] `backfill_test.go` (5 matchs, vérifier bitmask)
- [ ] `aggregates_test.go`, `career_test.go`, `performance_test.go` (côté sync, pas analysis)
- [ ] Lancer localement : coverage `sync/` ≥ 70%, `migration/` ≥ 75%, `platform/duckdb/` ≥ 70%
- [ ] Durée totale suite : vérifier < 2 minutes (sinon `t.Parallel()` + réduire seeds)
- [ ] Mise à jour `coverage_baseline.txt` → ~55-56%
- [ ] CI seuil remonté à 55%
- [ ] Entrée `.ai/thought_log.md`

### Gate Sprint 47
- [ ] Toutes fonctions `writes.go` couvertes
- [ ] Migrations testées en apply+idempotence
- [ ] Write lease testé sous concurrence
- [ ] Coverage global ≥ 55%
- [ ] Suite complète < 2 min local, < 5 min CI

---

## 5. Sprint 48 — Validation + ops + gate 70% (5–7j)

### Objectif
Combler les derniers trous et franchir officiellement **70%**.

### Fonctions cibles (matrice à vider)

**`internal/validation/compare.go`** — 13 fonctions à 0% :

| Fonction | Test |
|---|---|
| `ComparePlayerDBs` | 2 DBs identiques (0 diff) / 2 DBs différentes (diff capturé) |
| `listTables` | DB avec N tables → retourne N |
| `countRows` | 0 / 1 / N lignes |
| `compareTableCounts` | égal / +/- / manquante |
| `classifyDelta` | OK / WARN / FAIL selon seuils |
| `compareBitmasks` | identique / bits différents / manquants |
| `compareMatchIDs` | jaccard 1.0, 0.8, 0.5, 0.0 |
| `loadMatchIDs` | DB vide, DB avec N matchs |
| `isReportOK` | tous OK, un FAIL, un WARN |
| `buildSummary` | format texte + JSON |
| `statusIcon` | OK→✅, WARN→⚠️, FAIL→❌ |
| `jaccardLabel` | seuils de qualification |

**`internal/validation/gate.go`** — 11 fonctions à 0% :

| Fonction | Test |
|---|---|
| `RunGateCheck4` | scénario complet pass + scénario fail (1 check KO) |
| `Format` | report formatté |
| `checkBinary` | binaire existe / absent |
| `checkDBAccessible` | DB ouvrable / corrompue |
| `checkSharedTables` | tables présentes / manquante |
| `checkSharedViews` | vues présentes / manquante |
| `checkMigrationsApplied` | version ≥ requise / inférieure |
| `checkPlayerDB` | DB joueur valide / absente |
| `checkDBProfiles` | fichier valide / manquant / mal formé |
| `checkDiscordNotify` | URL webhook valide / invalide / unreachable |
| `checkTablesExist` / `checkViewsExist` | helpers avec 2+3 cas |

**`internal/ops/restore.go`** — étendre le test existant :
- corrupt archive (checksum mismatch)
- version mismatch (v5 → v6 refusée)
- partial restore (file manquant au milieu)

**`internal/notify/version.go`** :
- `TestVersionParse` : semver valide / invalide
- `TestVersionCompare` : <, =, >, breaking, backwards-compat

**`internal/service/` restants** (à confirmer par inspection de `service_test.go` + `bootstrap_test.go` + `timeseries_service_test.go` qui existent déjà) :

| Service | Test manquant |
|---|---|
| `match_history_service` | ⬜ |
| `explorer_service` | ⬜ |
| `media_service` | ⬜ |
| `citations_service` | ⬜ |
| `last_match_service` | ⬜ |
| `teammates_service` | ⬜ |
| `session_compare_service` | ⬜ |
| `sessions_service` | ⬜ |

**`internal/analysis/` restants** (à confirmer en listant les `*.go` sans pendant `*_test.go`) :

| Fichier | Test manquant |
|---|---|
| `kill_attribution.go` | ⬜ |
| `weapon_data.go` | ⬜ |
| `weapon_parser.go` | ⬜ |
| `weapon_reconciliation.go` | ⬜ |
| `performance_score.go` | ⬜ (vérifier) |

### Tâches Sprint 48 (checklist)

- [ ] Vider la matrice `validation/compare_test.go` (13 fonctions)
- [ ] Vider la matrice `validation/gate_test.go` (11 fonctions)
- [ ] Étendre `ops/restore_test.go` (3 cas edge)
- [ ] Créer `notify/version_test.go`
- [ ] Compléter les 8 tests service manquants
- [ ] Compléter les tests analysis manquants
- [ ] Tests `domain/title/` : multi-titres + fallback HI-only
- [ ] Lancer coverage final : **≥ 70%** global
- [ ] Vérifier : aucun package `internal/` (hors `gen/`) < 50%
- [ ] Remonter seuil CI à **70%** dans `.github/workflows/ci.yml`
- [ ] Faire passer `golangci-lint run ./...` clean (dette résiduelle S44)
- [ ] Créer `docs/testing.md` si pas encore fait (commandes locale + guide contribution)
- [ ] Mettre à jour `project_map.md` : colonne coverage par package
- [ ] Entrée `.ai/thought_log.md` : bilan Phase 10 complet (avant/après, durée, leçons)
- [ ] Cocher dans SPRINT_ROADMAP.md Gate Phase 9 : "Couverture ciblée modules Sprint 44 ≥ 80%, couverture Go globale ≥ 50%" (désormais vraie et mesurable)

### Gate Sprint 48 (gate finale Phase 10)
- [ ] **Coverage global ≥ 70%**
- [ ] `internal/api/handlers/` ≥ 75%
- [ ] `internal/sync/` ≥ 70%
- [ ] `internal/migration/` ≥ 75%
- [ ] `internal/platform/duckdb/` ≥ 70%
- [ ] `internal/validation/` ≥ 70%
- [ ] Aucun package `internal/` (hors `gen/`) < 50%
- [ ] CI seuil = 70%
- [ ] `golangci-lint run ./...` clean
- [ ] Baseline committé à jour
- [ ] Rapport HTML disponible en artifact CI
- [ ] `docs/testing.md` documenté
- [ ] Suite < 5 min local, < 10 min CI

---

## 6. Risques & parades

| Risque | Parade |
|---|---|
| CGO + Windows runner trop lent | Ubuntu runner pour coverage, Windows reste sur `go-build` et `go-test-short` |
| Tests flaky à cause de timestamps / UUIDs | Injection d'une clock / uuidgen via interface ; fixtures déterministes |
| Suite > 10 min en CI | `t.Parallel()` partout où possible, limiter seeds à 1 match par test, sharding `go test -p 4` |
| `port.Services` incomplet (méthodes manquantes) | Extension interface dès S46, tâche 1 bloquante |
| Driver DuckDB Go ne supporte pas `:memory:` proprement | Fallback : fichier temp `t.TempDir()` — coût perf négligeable |
| Les tests golden deviennent non-déterministes après refactor tests | `tests/golden/` reste isolé, ne bouge pas tant que snapshots compatibles |
| Coverage 70% atteint mais tests shallow (faux positifs de mutation) | Pas de test "smoke qui couvre sans assert" — chaque test a au moins 1 `assert` métier |

---

## 7. Suivi d'exécution

À chaque fin de sprint :

1. Mesurer : `cd apps/go-api && CGO_ENABLED=1 go test -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./...`
2. Committer `coverage_baseline.txt` updaté
3. Ajouter entrée `.ai/thought_log.md` avec :
   - Coverage avant / après (chiffre exact)
   - Packages les plus bas restants
   - Blockers rencontrés
   - Écart durée estimée vs réelle
4. Cocher les tâches dans ce document (matrice + checklist)
5. Update le statut dans `SPRINT_ROADMAP.md` vue d'ensemble (⬜ → 🔄 → ✅)

---

## 8. Définition de "fini" pour la Phase 10

La Phase 10 est **fermée** uniquement quand tous ces points sont vrais :

- [ ] Coverage global ≥ 70% mesuré sur `./...` avec CGO activé
- [ ] CI applique ratchet 70% minimum
- [ ] Baseline = valeur réelle (pas gonflée par exclusions abusives)
- [ ] `docs/testing.md` permet à un nouveau contributeur de lancer + interpréter les tests en < 10 min
- [ ] Gate Phase 9 "Couverture ciblée ≥ 80%" redevient mesurable et passe
- [ ] Phase 10 est réellement terminée, pas "juste le chiffre atteint" — les *tests eux-mêmes* sont utiles (chaque test a une assertion métier, pas juste `assert handler != nil`)

Si l'un de ces points n'est pas vrai, la phase reste ouverte et le sprint 48 n'est pas coché.

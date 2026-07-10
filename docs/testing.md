# Guide de test — API Go LevelUp

> Document créé lors du Sprint 45 (Phase 10 — Consolidation qualité).
> Objectif Phase 10 : couverture globale ≥ **70%** mesurée sur `./...` avec CGO activé.

---

## Lancer les tests localement

### Tests rapides sans DuckDB (CGO=0)

```bash
cd apps/go-api
CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... \
  -timeout 60s -count=1 -v
```

Ces tests ne nécessitent pas DuckDB et tournent sur toutes les plateformes.

### Tests complets avec DuckDB (CGO=1)

```bash
cd apps/go-api
CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1 -v
```

Sur Windows, s'assurer que le compilateur MinGW est disponible :

```powershell
# MinGW via MSYS2
$env:CGO_ENABLED = "1"
$env:CC = "gcc"
go test ./... -timeout 5m -count=1
```

### Mesurer la couverture localement

```bash
cd apps/go-api

# 1. Générer le profil brut
CGO_ENABLED=1 LEVELUP_DEMO_MODE=true \
  go test -coverprofile=coverage.out.raw -covermode=atomic \
  -coverpkg=./... ./... -timeout 5m -count=1

# 2. Filtrer les packages exclus
bash scripts/coverage_filter.sh coverage.out.raw > coverage.out

# 3. Afficher le résumé
go tool cover -func=coverage.out | tail -5

# 4. Ouvrir le rapport HTML
go tool cover -html=coverage.out -o coverage.html
# Puis ouvrir coverage.html dans le navigateur

# 5. Vérifier le ratchet vs baseline
bash scripts/coverage_check.sh coverage.out coverage_baseline.txt
```

---

## Interpréter le rapport

### Packages exclus de la métrique

| Package | Raison |
|---------|--------|
| `cmd/msal-poc/` | POC jetable — remplacé par `platform/auth` |

### Seuils CI (ratchet progressif Phase 10)

| Sprint | Seuil CI | Packages cibles |
|--------|:--------:|-----------------|
| S45 (infra) | **15%** | Baseline honnête établi |
| S46 (handlers) | **35%** | `api/handlers/` ≥ 75%, `api/middleware/` ≥ 80% |
| S47 (sync+DB) | **55%** | `sync/` ≥ 70%, `migration/` ≥ 75%, `platform/duckdb/` ≥ 70% |
| S48 (validation) | **70%** | `validation/` ≥ 70%, tous `internal/` ≥ 50% |

### Règle ratchet

La couverture **ne peut jamais régresser**. Le fichier `coverage_baseline.txt` contient le
pourcentage de référence. Toute PR qui baisse la couverture de plus de 0.1% fail la CI.

Sur `main`, si la couverture monte, le baseline est automatiquement mis à jour.

---

## Contribuer un test qui bouge le baseline

### Pattern handlers HTTP (mock service)

```go
// Exemple : citations_test.go
package handlers_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "errors"

    "github.com/go-chi/chi/v5"
    "levelup/go-api/internal/api/handlers"
    "levelup/go-api/internal/domain"
    "levelup/go-api/internal/port"
)

type mockCitationsService struct {
    page     *domain.CitationsPageResponse
    pageErr  error
}

func (m *mockCitationsService) GetCitationsPage(_ context.Context) (*domain.CitationsPageResponse, error) {
    return m.page, m.pageErr
}
// ... autres méthodes

func TestCitationsHandler_OK(t *testing.T) {
    mock := &mockCitationsService{page: &domain.CitationsPageResponse{}}
    factory := func(_ context.Context, slug string) (port.CitationsService, string, string, error) {
        return mock, "xuid-1", "TestPlayer", nil
    }
    h := handlers.NewCitationsHandler(factory)
    r := chi.NewRouter()
    r.Post("/players/{player_slug}/pages/citations", h.GetCitations)

    req := httptest.NewRequest(http.MethodPost, "/players/test/pages/citations", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }
}
```

### Pattern DB in-memory (sync/writes)

```go
// Exemple : writes_test.go
package sync_test

import (
    "database/sql"
    "testing"

    _ "github.com/duckdb/duckdb-go/v2"
    "levelup/go-api/internal/sync/testutil"
)

func TestInsertRegistryIfNotExists(t *testing.T) {
    db := testutil.NewInMemoryShared(t)
    defer db.Close()

    // Tester l'insertion
    err := InsertRegistryIfNotExists(db, "match-1", ...)
    if err != nil { t.Fatal(err) }

    // Doublon ignoré
    err = InsertRegistryIfNotExists(db, "match-1", ...)
    if err != nil { t.Fatal(err) }
}
```

### Pattern validation gate

```go
// Exemple : gate_test.go
func TestCheckBinary(t *testing.T) {
    // binaire présent
    err := checkBinary(os.Args[0]) // binaire de test lui-même
    if err != nil { t.Fatal(err) }

    // binaire absent
    err = checkBinary("/nonexistent/binary")
    if err == nil { t.Fatal("expected error for missing binary") }
}
```

---

## Organisation des tests par couche

| Couche | Répertoire | Type |
|--------|-----------|------|
| Handlers HTTP | `internal/api/handlers/*_test.go` | httptest + mock service |
| Middlewares | `internal/api/middleware/*_test.go` | httptest |
| Services | `internal/service/*_test.go` | mock repo |
| Sync writes | `internal/sync/*_test.go` | DuckDB in-memory |
| Migrations | `internal/migration/*_test.go` | DuckDB in-memory |
| Platform DuckDB | `internal/platform/duckdb/*_test.go` | DuckDB in-memory |
| Validation | `internal/validation/*_test.go` | fixtures DuckDB |
| Ops | `internal/ops/*_test.go` | `t.TempDir()` |
| Domain/Analysis | `internal/domain/*_test.go`, `internal/analysis/*_test.go` | unitaire pur |
| Contrat API | `contracttest/` | parsing OpenAPI YAML |
| Golden | `tests/golden/` | snapshot JSON |

---

## Foire aux questions

**Q : Pourquoi CGO_ENABLED=1 est requis pour la couverture complète ?**

DuckDB requiert CGO. Sans CGO, les packages `platform/duckdb/`, `sync/`, `migration/` ne compilent pas.
Le job `go-build` garde un check CGO=0 rapide pour les packages purement algorithmiques.

**Q : Les tests golden comptent-ils dans la couverture ?**

Oui, grâce au flag `-coverpkg=./...`. Sans ce flag, les tests dans `tests/golden/` (package
`golden_test`) n'auraient pas instrumenté les handlers qu'ils exercent.

**Q : Comment exclure un nouveau package de la métrique ?**

Ajouter le pattern dans `scripts/coverage_filter.sh` dans le tableau `EXCLUDE`. Documenter la
raison dans ce fichier.

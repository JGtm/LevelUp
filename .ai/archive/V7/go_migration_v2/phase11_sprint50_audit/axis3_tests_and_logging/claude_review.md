# Axe 3 — Review Claude — Tests & logging

## Métadonnées du passage

| Champ | Valeur |
|-------|--------|
| Auteur LLM | **Claude** |
| Date du passage | 2026-04-18 |
| SHA Go | `93c3cd66` |
| SHA React | `93c3cd66` |
| SHA Python (référence) | `db638c09` |
| Source couverture Go | `go test -coverprofile=cov_sprint50.out ./internal/...` |
| Durée de l'analyse | ~1.5h |

## Synthèse exécutive

La couverture Go globale est de **58.0%** — sous la gate ≥70% (🔴 BLOCKER). Les couches métier (analysis, handlers, middleware, service, validation) sont bien couvertes (>85%), mais trois packages infrastructure sont sous-couverts : **platform/duckdb (0.6%)**, **sync (14.1%)** et **migration (0.0%)**. Le test Go dispose de 1 376 `Test*` functions dans 144 fichiers test, avec des golden values pour les 7 algorithmes cœur. Côté React : 12 tests unitaires (Vitest) et 16 tests E2E (Playwright), mais 6 features sans test unitaire. Le logging Go utilise `slog` structuré (66 appels, 4 niveaux) avec 10 résidus `log.Printf` dans `notify/`.

---

## A. Couverture Go — par package

### A.1 Tableau de couverture détaillé

| Package | Couverture | # Fonctions | Gate cible | Verdict | Classif |
|---------|:----------:|:-----------:|:----------:|:-------:|:-------:|
| **Global** | **58.0%** | all | ≥ 70% | 🔴 FAIL | 🔴 |
| `internal/analysis` | 90.9% | 142 | ≥ 80% | ✅ PASS | 🟢 |
| `internal/api/handlers` | 90.3% | 85 | ≥ 75% | ✅ PASS | 🟢 |
| `internal/api/middleware` | 93.7% | 32 | ≥ 80% | ✅ PASS | 🟢 |
| `internal/config` | 84.2% | 26 | variable | ✅ | 🟢 |
| `internal/domain` | 100.0% | 11 | ≥ 80% | ✅ PASS | 🟢 |
| `internal/domain/title` | 100.0% | 30 | ≥ 80% | ✅ PASS | 🟢 |
| `internal/migration` | **0.0%** | 24 | ≥ 75% | 🔴 FAIL | 🔴 |
| `internal/notify` | 71.8% | 30 | variable | ✅ | 🟢 |
| `internal/ops` | 68.6% | 39 | ≥ 70% | 🟡 −1.4% | 🟡 |
| `internal/platform/auth` | 71.9% | 18 | variable | ✅ | 🟢 |
| **`internal/platform/duckdb`** | **0.6%** | 81 | ≥ 70% | 🔴 FAIL | 🔴 |
| `internal/platform/halo` | 88.4% | 12 | variable | ✅ | 🟢 |
| `internal/platform/jobs` | 83.1% | 10 | variable | ✅ | 🟢 |
| `internal/platform/session` | 88.5% | 13 | ≥ 70% | ✅ PASS | 🟢 |
| `internal/platform/settings` | 92.8% | 7 | variable | ✅ | 🟢 |
| `internal/service` | 91.4% | 166 | ≥ 80% | ✅ PASS | 🟢 |
| **`internal/sync`** | **14.1%** | 136 | ≥ 70% | 🔴 FAIL | 🔴 |
| `internal/validation` | 88.5% | 24 | ≥ 70% | ✅ PASS | 🟢 |

### A.2 Analyse des zones critiques 🔴

| Zone | Couverture | Analyse | Action recommandée | Priorité |
|------|:----------:|---------|--------------------|---------:|
| `platform/duckdb` | 0.6% | **81 fonctions** — repos DuckDB, pool, write-ahead. Aucun test unitaire (SQL embedded = difficile à mocker). Les handlers testent les routes de bout en bout mais la couverture ne se propage pas au package DB. | Tests d'intégration avec DuckDB in-memory (`:memory:`) + mocks pour les requêtes complexes. Objectif sprint : 40% → 70% | P0 |
| `sync` | 14.1% | **136 fonctions** — engine, backfill, lease, writes. Seulement 99 `Test*` fonctions vs ~136 fonctions production. Tests existants couvrent flags/scope (26 tests), transforms (12), mais pas le cœur engine. | Tests du sync engine avec DuckDB fixture + mock API Halo. Objectif sprint : 40% → 70% | P0 |
| `migration` | 0.0% | **24 fonctions** — 3 registres `init()` + `RunMigrations()`. Non testé car exécution unique au bootstrap. | Tests avec DB temporaire vierge → `RunMigrations()` → vérifier schéma. Objectif sprint : 50% | P1 |
| `ops` | 68.6% | Juste sous la gate 70%. Tests existants couvrent diagnose/healthcheck mais pas backup/restore intégralement. | Ajouter 2-3 tests pour atteindre 70% | P2 |

### A.3 Packages sans fichier test

| Package | # fichiers .go | Impact | Classif |
|---------|:--------------:|--------|:-------:|
| `internal/api/gen` | 1 (auto-généré) | Nul — types OpenAPI auto-générés | 🟢 |
| `internal/port` | 2 (interfaces pures) | Nul — pas de logique à tester | 🟢 |
| `internal/sync/testutil` | 1 (helpers test) | Nul — utilitaires de test | 🟢 |

---

## B. Couverture React

### B.1 Inventaire des tests

| Catégorie | Fichiers | Détail |
|-----------|:--------:|--------|
| Tests unitaires (Vitest) | 12 | 10 pages/features + 2 stores |
| Tests E2E (Playwright) | 16 | Toutes les slices fonctionnelles |
| Tests composants UI | 0 | Composants `src/components/ui/` non testés individuellement |

### B.2 Features avec test unitaire

| Feature | Test unitaire ? | Test E2E ? | Classif |
|---------|:---------------:|:----------:|:-------:|
| HomePage | ✅ | ✅ (`slice-5`) | 🟢 |
| CareerPage | ✅ | ✅ (`slice-2`) | 🟢 |
| MatchHistoryPage | ✅ | ✅ (`slice-3`) | 🟢 |
| ExplorerPage | ✅ | ✅ (`slice-4`) | 🟢 |
| SynthesisPage | ✅ | ✅ (`slice-7`) | 🟢 |
| SquadPage | ✅ | ✅ (`slice-6`) | 🟢 |
| MediaPage | ✅ | ✅ (`slice-8`) | 🟢 |
| SettingsPage | ✅ | ✅ (`slice-1`) | 🟢 |
| SetupPage | ✅ | ✅ (`slice-9`) | 🟢 |
| Shell navigation | ✅ | ✅ (`slice-0a`) | 🟢 |
| CitationsPage | ❌ | ✅ (`slice-2b`) | 🟡 |
| TimeseriesPage | ❌ | ✅ (`slice-3b`) | 🟡 |
| SessionComparePage | ❌ | ✅ (`slice-3c`) | 🟡 |
| MatchViewPage | ❌ | ✅ (`slice-4b`) | 🟡 |
| LastMatchPage | ❌ | ✅ (`slice-4c`) | 🟡 |
| ChangelogPage | ❌ | ❌ | 🟠 |

### B.3 Stores testés

| Store | Test ? | Classif |
|-------|:------:|:-------:|
| `appShellStore` | ✅ | 🟢 |
| `globalFilterStore` | ✅ | 🟢 |
| Autres stores | ❌ | 🟡 |

---

## C. Golden values & tests algorithmiques

### C.1 Algorithmes Go avec golden values

| Algorithme | Fichier test | # tests | Golden values ? | Parité Python ? | Classif |
|------------|-------------|:-------:|:---------------:|:---------------:|:-------:|
| Performance score | `performance_score_test.go` | 23 | ✅ Fixtures codées | ✅ Aligné `_performance_relative.py` | 🟢 |
| Skill rating (LUSR/CSR) | `skill_rating_test.go` + `_extra_test.go` | 51 | ✅ Golden values | ✅ Aligné `skill_rating.py` | 🟢 |
| Sessions | `sessions_test.go` | 30 | ✅ Fixtures multi-mode | ✅ Aligné `sessions.py` | 🟢 |
| Citations | `citations_test.go` | 9 | ✅ Golden values | ✅ Aligné `citations/engine.py` | 🟢 |
| Killer/victim | `killer_victim_test.go` | 6 | ✅ | ✅ | 🟢 |
| Weapon parser | `weapon_parser_test.go` + `scanner_test.go` | 26 | ✅ | ✅ | 🟢 |
| Spawn / comeback | `spawn_detection_test.go` | 14 | ✅ | ✅ | 🟢 |
| **Squad analysis** | `squad_test.go` + `squad_timeseries_test.go` | 41 | ✅ | ✅ | 🟢 |
| Home (session stats) | `home_test.go` + `home_internal_test.go` | 35 | ✅ | ✅ | 🟢 |

**7/7 algorithmes cœur ont des golden values, avec parité Python confirmée.**

### C.2 Tests sync

| Sous-domaine sync | Fichier test | # tests | Couverture | Classif |
|-------------------|-------------|:-------:|:----------:|:-------:|
| SyncScope / flags | `scope_test.go` + `backfill_flags_test.go` | 34 | Bonne | 🟢 |
| Transforms | `transforms_test.go` | 12 | Bonne | 🟢 |
| Writes shared | `writes_test.go` | 8 | Partielle | 🟡 |
| Lease | `lease_test.go` | 6 | Bonne | 🟢 |
| Performance sync | `performance_test.go` | 10 | Bonne | 🟢 |
| Skill rating sync | `skill_rating_test.go` | 8 | Bonne | 🟢 |
| Career sync | `career_test.go` | 5 | Bonne | 🟢 |
| Engine orchestration | `engine_test.go` | 3 | **Faible** — seulement 3 tests pour l'orchestrateur principal | 🔴 |
| Backfill | `backfill_test.go` | 9 | Partielle — SQL complexe non couvert | 🟡 |
| Aggregates | `aggregates_test.go` | 4 | Partielle | 🟡 |

---

## D. Logging & observabilité

### D.1 Go — structure de logging

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| Framework | `log/slog` (stdlib Go 1.21+) — structuré, JSON-capable | 🟢 |
| Fichiers utilisant slog | 19 fichiers | 🟢 |
| Distribution niveaux | 39 Warn, 13 Info, 11 Debug, 3 Error | 🟢 — distribution raisonnable |
| Résidus `log.Printf` | **10 occurrences** dans `notify/discord.go` et `notify/notifiers.go` | 🟠 |
| `fmt.Printf` en prod | 0 détecté hors tests | 🟢 |
| Contexte structuré (attrs) | Utilisé via `slog.String()`, `slog.Int()`, `slog.Any()` | 🟢 |
| Middleware request logging | ✅ `api/middleware/logging.go` — méthode, path, status, durée | 🟢 |

### D.2 Résidus `log.Printf` — détail

| Fichier | # occurrences | Contexte | Action |
|---------|:-------------:|----------|--------|
| `notify/discord.go` | 5 | Log d'erreurs webhook Discord | Migrer vers `slog.Error()` |
| `notify/notifiers.go` | 5 | Log de flow sync (panic recover, skip, allégé) | Migrer vers `slog.Warn()`/`slog.Info()` |

### D.3 React — logging

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| `console.log` résiduel | **0** | 🟢 |
| `console.error` résiduel | **0** | 🟢 |
| Error reporting (Sentry etc.) | Non configuré | 🟡 — acceptable MVP |
| Structured logging | Aucun framework client-side | 🟡 — acceptable MVP |

### D.4 Python (référence) — comparaison

| Aspect | Python | Go | Parité ? |
|--------|--------|-----|:--------:|
| Framework logging | `logging` stdlib | `slog` stdlib | ✅ Équivalent |
| Structured logging | JSON formatter configuré | slog JSON handler | ✅ |
| Niveaux utilisés | DEBUG/INFO/WARNING/ERROR | Debug/Info/Warn/Error | ✅ |
| Anti-patterns `print()` | 0 (éradiqué v5) | 10 `log.Printf` résidus | 🟡 Go légèrement en retard |

---

## E. Qualité des tests

### E.1 Patterns de test Go

| Pattern | Utilisé ? | Classif |
|---------|:---------:|:-------:|
| Table-driven tests | ✅ Massivement (analysis, service, handlers) | 🟢 |
| Subtests (`t.Run`) | ✅ | 🟢 |
| `testify/assert` + `require` | ✅ | 🟢 |
| Test helpers / fixtures (`testutil/fixture.go`) | ✅ | 🟢 |
| Mocks (interfaces) | ✅ — via interfaces port | 🟢 |
| DuckDB in-memory pour tests | Partiellement — analysis oui, duckdb repos non | 🟡 |
| Benchmarks | 0 | 🟡 — pas critique pour MVP |
| Fuzz tests | 0 | 🟡 — pas critique pour MVP |

### E.2 Matrice couverture vs criticité

| Zone | Criticité business | Couverture | Adéquation ? | Classif |
|------|:------------------:|:----------:|:-------------:|:-------:|
| Analysis (algorithmes) | 🔴 Critique | 90.9% | ✅ Excellente | 🟢 |
| Handlers (API) | 🔴 Critique | 90.3% | ✅ Excellente | 🟢 |
| Service (orchestration) | 🟠 Haute | 91.4% | ✅ Excellente | 🟢 |
| Domain (entités) | 🟠 Haute | 100.0% | ✅ Parfaite | 🟢 |
| Middleware | 🟠 Haute | 93.7% | ✅ Excellente | 🟢 |
| Sync (engine) | 🔴 Critique | 14.1% | ❌ Insuffisante | 🔴 |
| DuckDB repos | 🟠 Haute | 0.6% | ❌ Quasi-nulle | 🔴 |
| Migration | 🟡 Moyenne | 0.0% | ❌ Absente | 🟠 |
| Validation | 🟠 Haute | 88.5% | ✅ Excellente | 🟢 |

---

## Tableau récapitulatif des écarts

| # | Zone | Description | Classif |
|--:|------|-------------|:-------:|
| 1 | Go | Couverture globale **58.0%** — gate ≥70% non atteinte | 🔴 BLOCKER |
| 2 | Go | `platform/duckdb` à **0.6%** — 81 fonctions non testées | 🔴 BLOCKER |
| 3 | Go | `sync` à **14.1%** — engine/backfill sous-couvert, engine_test = 3 tests seulement | 🔴 BLOCKER |
| 4 | Go | `migration` à **0.0%** — aucun test pour RunMigrations() | 🟠 |
| 5 | Go | `ops` à **68.6%** — juste sous la gate 70% | 🟡 |
| 6 | Go | 10 résidus `log.Printf` dans `notify/` — à migrer vers `slog` | 🟠 |
| 7 | React | 6 features sans test unitaire (citations, changelog, timeseries, session-compare, match-view, last-match) | 🟡 |
| 8 | React | Aucun ErrorBoundary — crash = page blanche, pas de fallback | 🟠 |
| 9 | React | ChangelogPage sans test unitaire ni E2E | 🟠 |
| 10 | React | Aucun error reporting client-side (Sentry, etc.) | 🟡 |
| 11 | Go | 0 benchmarks, 0 fuzz tests | 🟡 |
| 12 | Go | DuckDB in-memory non utilisé pour tester les repos platform/duckdb | 🟡 |

### Blockers pour la gate de release

1. **Couverture Go globale ≥ 70%** : Nécessite ~200 tests supplémentaires ciblant `platform/duckdb` (+~55%) et `sync` (+~56%)
2. **platform/duckdb 0.6% → ≥ 70%** : Stratégie recommandée — DuckDB in-memory (`:memory:`) avec fixtures SQL injectées, tester les 81 fonctions de repository
3. **sync 14.1% → ≥ 70%** : Stratégie recommandée — mock des interfaces Halo API + DuckDB fixture pour tester l'orchestration engine, backfill SQL, writes

---

**Fin de la review Claude — Axe 3.**

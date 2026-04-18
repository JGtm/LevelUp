# Axe 3 · Réconciliation — Tests & logging

> **Statut** : ✅ Complété
> **Date de réconciliation** : `2026-04-18`
> **Note critique** : les deux LLM ont utilisé des profils de couverture différents — voir §3 pour l'arbitrage.

## 1. Méthodologie

Cf. PROTOCOL.md §Étape 4. Les deux LLM ont utilisé des profils de couverture différents :
- **Claude** a généré `cov_sprint50.out` via `go test ./internal/...` **sans** `-tags=integration` ni CGO → 58.0% global
- **ChatGPT** a utilisé `coverage.out` (profil CI, avec `-tags=integration`, CGO activé) → 78.8% global

La différence s'explique structurellement : les repos `platform/duckdb` et les migrations nécessitent CGO + tags d'intégration pour exercer la vraie base. La mesure CI (`coverage.out`) est la référence officielle.

**Point critique additionnel** : `coverage.out` ne contient **aucune ligne** du package `internal/sync`. Le package sync est donc exclu du profil CI (non testé dans ce mode) — son taux réel en environnement complet reste à mesurer séparément.

## 2. Convergences

| Item | Section | Classif commune | Fichier:ligne | Note |
|------|---------|:---------------:|---------------|------|
| Pas de `console.log` résiduel React | D.4 | 🟢 | `apps/web/src/` | Convergence forte |
| MSW (mock service worker) en place | B.1 | 🟢 | `src/test/setup.ts`, `src/test/handlers.ts` | Convergence |
| 16 specs Playwright en CI | C.3 | 🟢 | `e2e/*.spec.ts`, `.github/workflows/ci.yml:380` | Convergence |
| Workflow CI coverage ratchetté (baseline 76.0) | E | 🟢 | `coverage_baseline.txt`, `scripts/coverage_check.sh` | Convergence |
| `handlers` coverage ≥ 75% ✅ | A.1 | 🟢 | `internal/api/handlers/` | Les deux LLM s'accordent |
| `middleware` coverage ≥ 80% ✅ | A.1 | 🟢 | `internal/api/middleware/` | Les deux LLM s'accordent |
| `validation` coverage ≥ 70% ✅ | A.1 | 🟢 | `internal/validation/` | Les deux LLM s'accordent |
| `match-view`, `last-match`, `session-compare`, `citations`, `timeseries` sans unit test React | B | 🟠/🟡 | `features/*/` | Les deux l'identifient |
| ErrorBoundary React absent | D.4 | 🟡 | `apps/web/src/` | Convergence (noté par les deux) |
| `pool.go` fonctions `GetOrOpen`, `openPlayerDB`, `attachShared`, `attachMeta` à 0% | A.2 | 🟠 | `platform/duckdb/pool.go:56,94,141,160` | Confirmé dans `coverage.out` |

## 3. Divergences de classification

| Item | Claude | ChatGPT | Finale | Justification |
|------|:------:|:-------:|:------:|---------------|
| **Couverture globale Go** | 58.0% → 🔴 FAIL | 78.8% → 🟢 PASS | **🟢 PASS (78.8%)** | Le profil CI (`coverage.out`, CGO + integration) est la référence officielle. Claude avait un profil partiel. Gate ≥70% : PASSÉE. |
| **`platform/duckdb` coverage** | 0.6% → 🔴 | 75.4% → 🟢 | **🟢 (75.4% avec integration)** | Même explication — DuckDB repos nécessitent CGO+integration pour être exercés |
| **`migration` coverage** | 0.0% → 🟠 | 81.1% → 🟢 | **🟢 (81.1% avec integration)** | Les migrations sont exercées par les tests d'intégration handlers |
| **`sync` coverage** | 14.1% → 🔴 | non mesuré → ⚪ | **🟡 inconnu** | `sync` est absent de `coverage.out` (profil CI). Besoin d'un profil dédié. Valeur sans CGO : 14.1% |
| `analysis` coverage | 90.9% → 🟢 | non mesuré → ⚪ | **🟢** | Claude a mesuré, ChatGPT ne contestait pas |
| `service` coverage | 91.4% → 🟢 | non mesuré → ⚪ | **🟢** | Idem |

## 4. Items identifiés par un seul LLM

| Item | Par qui | Vérif manuelle | Retenu ? | Classif |
|------|:-------:|----------------|:--------:|:-------:|
| `@vitest/coverage-v8` non installé → couverture React non mesurable | ChatGPT | `apps/web/package.json` : `@vitest/ui` présent mais pas `coverage-v8` | ✅ | 🟠 |
| `pollDeviceFlow` (auth) à 0% dans coverage.out | ChatGPT | `handlers/auth.go:136` confirmé 0% | ✅ | 🟠 |
| 10 résidus `log.Printf` dans `notify/discord.go` et `notifiers.go` | Claude | Confirmé par grep | ✅ | 🟠 |
| `ChangelogPage` sans test unitaire ni E2E | Claude | Confirmé — pas de spec ni test | ✅ | 🟠 |
| `internal/sync` absent du profil CI coverage.out | (vérification réconciliation) | Confirmé — `grep sync coverage.out` = 0 lignes | ✅ | 🟡 |
| Tests de concurrence write lease | ChatGPT | Non présents dans les test files | ✅ | 🟡 |
| Idempotence migrations (run 2x) non testée | ChatGPT | Plausible, non re-vérifié | ✅ | 🟡 |

## 5. Synthèse finale

| Niveau | Nombre | Descriptions |
|--------|:------:|---|
| 🔴 | 0 | Gate globale Go passée (78.8% ≥ 70%) |
| 🟠 | 5 | `@vitest/coverage-v8` manquant, `pollDeviceFlow` 0%, pool.go 4 fonctions 0%, 10 `log.Printf` dans notify/, ChangelogPage sans tests |
| 🟡 | 7 | sync coverage inconnu (exclu du profil CI), 6 features React sans unit test, ErrorBoundary, concurrence write lease non testée, idempotence migrations, temps CI non mesuré, tests accessibilité absents |
| 🟢 | 7+ | Gate globale ✅, handlers/middleware/validation ✅, MSW en place ✅, Playwright CI ✅, ratchet CI ✅, 1376 Test* fonctions ✅, golden values 7/7 ✅ |

## 6. Top 10 trous de couverture (consolidé)

| # | Zone | Coverage actuelle | Effort | Impact |
|--:|------|:-----------------:|:------:|--------|
| 1 | `@vitest/coverage-v8` | non installé | S | Rend la couverture React mesurable |
| 2 | `sync` (exclu profil CI) | 14.1% sans CGO | M | Package critique — engine sync non mesuré en CI |
| 3 | `pool.go` — `GetOrOpen`, `openPlayerDB`, `attachShared`, `attachMeta` | 0% | M | Ouverture DB — chemin critique prod |
| 4 | `pollDeviceFlow` (auth) | 0% | S | Auth onboarding non testé |
| 5 | `parseUploadedFiles` (media) | 0% | S | Chemin upload non testé |
| 6 | `MatchViewPage` | 0 unit test | S | Surface centrale sans unit test |
| 7 | `LastMatchPage` | 0 unit test | S | Surface visible sans unit test |
| 8 | `SessionComparePage` | 0 unit test | S | Surface visible sans unit test |
| 9 | `TimeseriesPage` | 0 unit test | S | Surface visible sans unit test |
| 10 | `CitationsPage` + `ChangelogPage` | 0 unit test | S | Surfaces sans test unitaire ni E2E pour Changelog |

## 7. Flux logging critiques manquants (consolidé)

| Flux | Fichier à modifier | Niveau attendu | Effort |
|------|--------------------|:--------------:|:------:|
| `log.Printf` dans discord.go (5 occurrences) | `internal/notify/discord.go:104-150` | `slog.Error/Warn` | S |
| `log.Printf` dans notifiers.go (5 occurrences) | `internal/notify/notifiers.go:36-58` | `slog.Warn/Info` | S |
| `pollDeviceFlow` — état résolution / cause échec | `internal/api/handlers/auth.go:136` | `slog.Error` | S |
| Ouverture/attach pool DuckDB — succès + erreur | `internal/platform/duckdb/pool.go:56-160` | `slog.Info/Error` | S |

## 8. Recommandation go / no-go pour l'axe 3

- [x] Gate globale Go ≥ 70% : **PASSÉE** (78.8%)
- [x] handlers/middleware/validation gates passées
- [ ] `sync` à mesurer avec profil complet CGO
- [ ] `@vitest/coverage-v8` à installer pour mesurer React
- [ ] 4 fonctions `pool.go` à 0% — plan de couverture Sprint 51

**Décision** : **GO conditionnel** — gate officielle passée (78.8%), mais le package `sync` est exclu du profil CI et sa couverture réelle reste inconnue. À résoudre en Sprint 51 avant bascule prod.

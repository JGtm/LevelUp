# Axe 3 · SCOPE — Tests & logging

## Objectif de l'axe

Vérifier que :
1. La couverture de tests unitaires est **excellente** côté Go et React (conforme aux gates Phase 10).
2. Les scénarios critiques validés côté Python ont un **test de non-régression** côté Go ou React.
3. Le **logging / observabilité** est complet : chaque flux critique (sync, writes, boundaries HTTP, erreurs provider) est traçable.

## Baseline

| Worktree | Chemin | Branche | SHA |
|----------|--------|---------|-----|
| Go | `LevelUp-go-migration/apps/go-api/` | `recovery/reapply-wip-s49-closure-2026-04-18` | `93c3cd66` |
| React | `LevelUp-go-migration/apps/web/` | idem | `93c3cd66` |
| Python (réf. non-régression) | `LevelUp/` | `v7/cockpit` | `db638c09` |

## Périmètre inclus

### Couverture Go

- Tout `apps/go-api/internal/**/*.go` hors `gen/` et POC
- Mesure réelle CGO activé (`go test -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./...`)
- Rapport HTML archivé en artifact
- Baseline `apps/go-api/coverage_baseline.txt` à jour

### Couverture React

- Tout `apps/web/src/**/*.{ts,tsx}` hors `*.test.*`, `*.stories.*`
- Mesure via Vitest coverage (v8 ou istanbul)
- Rapport HTML archivé

### Tests de non-régression

Scénarios critiques issus de l'historique Python (sessions, sync, backfill, media, squad, comeback, i18n, restore). Chaque scénario doit avoir :
- Son test Python de référence (fichier)
- Son test Go/React équivalent (fichier) OU une note explicite d'absence
- Idéalement : une golden value partagée

### Golden values

Les 7 algorithmes listés dans `SPRINT_ROADMAP.md §Rappels transverses`.

### E2E Playwright

Les 15 specs référencées Sprint 36.

### Logging

Côté Go :
- Démarrage, shutdown, requêtes, sync, writes, leases, migration, auth, jobs
- Format structuré (slog JSON de préférence)
- Request ID propagé
- Taxonomie d'erreur provider (`HALO_PROVIDER_ERROR_TAXONOMY.md`)

Côté React :
- Pas de `console.log` résiduels
- Error boundaries remontent les erreurs (Sentry ou équivalent si configuré)
- Traçage requêtes API

## Périmètre EXCLU

- Logging Python (l'ancien) — sauf comme référence pour ce qui existait
- Métriques (Prometheus, etc.) sauf mention explicite par ailleurs
- Infrastructure CI (sauf le job coverage lui-même)

## Critères mesurables

> **Cible globale 70 %** (Go ET React) — cible dure, exigée par l'utilisateur. Tout écart < 70 % = 🔴 bloquant tant que non justifié. Les seuils par package restent alignés sur Phase 10 (cf. PROTOCOL.md).

| Critère | Seuil |
|---------|:-----:|
| Coverage globale Go (CGO on) | ≥ 70 % |
| `internal/api/handlers/` | ≥ 75 % |
| `internal/api/middleware/` | ≥ 80 % |
| `internal/sync/` | ≥ 70 % |
| `internal/migration/` | ≥ 75 % |
| `internal/platform/duckdb/` | ≥ 70 % |
| `internal/validation/` | ≥ 70 % |
| Aucun package `internal/` (hors gen/) | ≥ 50 % |
| Coverage React (global, hors `*.stories.*` et `*.test.*`) | ≥ 70 % |
| Coverage React `src/features/` (logique métier) | ≥ 70 % |
| Scénarios critiques Python sans test Go/React équivalent | 0 |
| Golden values divergentes | 0 |
| 15 specs Playwright vertes en CI | 100 % |
| Flux Go sans log ERROR sur échec | 0 |
| Flux Go sans log INFO démarrage/fin | 0 (sur liste critique) |
| `console.log` en prod React | 0 |

## Méthodologie de mesure

```bash
# Coverage Go (worktree Go)
cd apps/go-api
go test -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./...
go tool cover -func=coverage.out | tee coverage_report.txt
go tool cover -html=coverage.out -o coverage.html

# Coverage React
cd apps/web
npm run test -- --coverage
# rapport dans coverage/index.html

# E2E
cd apps/web
npm run e2e
```

## Entrées pour le LLM

1. Ce `SCOPE.md`
2. La `CHECKLIST.md`
3. Le template vide `templates/axis3_tests_logging_template.md`
4. `coverage_report.txt` (Go) + `coverage/coverage-summary.json` (React) figés au SHA de l'audit
5. Accès lecture aux deux worktrees (pour lire les tests Python de référence)
6. `SPRINT_45_48_COVERAGE_ROADMAP.md` comme référence des objectifs Phase 10

## Sortie attendue

`claude_review.md` et `chatgpt_review.md` remplis, avec :
- Chaque cellule de couverture avec un nombre réel mesuré (pas « bon » / « à améliorer »)
- Chaque scénario non-régression avec oui/non + test path
- Chaque flux logging avec présent/absent + fichier:ligne

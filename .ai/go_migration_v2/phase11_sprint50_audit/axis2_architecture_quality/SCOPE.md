# Axe 2 · SCOPE — Architecture & qualité code

## Objectif de l'axe

Vérifier que le code Go + React respecte les principes d'architecture hexagonale, d'abstractions propres, et qu'il est exempt de redondances, de workarounds non justifiés et de fallbacks dangereux.

## Baseline

| Worktree | Chemin | Branche | SHA |
|----------|--------|---------|-----|
| Go | `LevelUp-go-migration/apps/go-api/` | `recovery/reapply-wip-s49-closure-2026-04-18` | `93c3cd66` |
| React | `LevelUp-go-migration/apps/web/` | idem | `93c3cd66` |

## Périmètre inclus

### Go (`apps/go-api/`)

- `cmd/` — points d'entrée, wire-up
- `internal/domain/` — entités pures
- `internal/port/` — interfaces (contrats)
- `internal/service/` — orchestration
- `internal/analysis/` — algorithmes purs
- `internal/api/handlers/` — adapters HTTP entrants
- `internal/api/middleware/` — cross-cutting concerns HTTP
- `internal/platform/duckdb/` — adapter DB
- `internal/platform/auth/` — adapter auth externe
- `internal/platform/jobs/` — gestion jobs async
- `internal/sync/` — moteur de sync
- `internal/migration/` — migrations DB
- `internal/ops/` — opérations (backup/restore)
- `internal/notify/` — notifications
- `internal/config/` — feature flags, config
- `internal/validation/` — gate + compare

### React (`apps/web/`)

- `src/app/` — providers, routes racine
- `src/components/ui/` — composants de design system
- `src/components/shell/` — navigation, layout
- `src/features/<domaine>/` — code par feature
- `src/routes/` — définitions routes TanStack
- `src/main.tsx`, `src/App.tsx`

## Périmètre EXCLU

- Code généré (`internal/api/gen/`, routes auto-générées TanStack)
- Fichiers de test (traités en axe 3)
- `cmd/msal-poc/` et autres POC jetables
- Assets (images, fonts, CSS)

## Grille hexagonale appliquée

### Règles de dépendance Go (ordre strict)

Convention : `A → B` signifie « A dépend de B » (A importe B). Dépendances autorisées :

```
api/handlers  →  service  →  port  →  domain
api/middleware →  service  →  port  →  domain
platform/duckdb  →  port  →  domain
platform/auth    →  port  →  domain
platform/jobs    →  port  →  domain

analysis  →  domain (uniquement) + primitives standard Go
```

**Règle d'or** : les flèches pointent toujours vers l'intérieur de l'hexagone (`domain` ne dépend de personne ; `port` ne dépend que de `domain` ; les adapters externes dépendent de `port`).

### Ce qui est interdit

- `domain/` importe `platform/`, `service/`, `api/`
- `port/` importe autre chose que `domain/`
- `analysis/` importe `database/sql`, `net/http`, `os` (à part via `os.Getenv` très encadré)
- `api/handlers/` importe directement `platform/duckdb/` (doit passer par `port.Services`)
- Cycle d'import entre packages `internal/`

### Ce qui est vérifié côté React

- `features/X/` n'importe pas `features/Y/`
- `components/ui/` ne contient pas de logique fetch
- `routes/` reste fin (juste la composition)
- Client API centralisé (pas de `fetch()` dispersé)

## Critères mesurables

| Critère | Seuil Go | Seuil React |
|---------|:--------:|:-----------:|
| Fichiers > 500 L | 0 (hors gen/) | 0 (hors stories auto) |
| Fonctions / composants > 80 L | 0 (hors `//nolint` motivé) | 0 |
| Fonctions > 5 arguments (hors struct/dataclass) | 0 (alignement CLAUDE.md §15) | 0 |
| Violations de dépendance hexagonale | 0 | 0 |
| Workarounds/fallbacks sans motivation ou sans TTL | 0 | 0 |
| Duplications ≥ 3 occurrences du même bloc logique | 0 | 0 |
| `interface{}` / `any` / TS `any` non justifiés | 0 | 0 |
| Handlers touchant directement `platform/duckdb` | 0 | N/A |
| Services avec > 5 dépendances constructeur non regroupées (struct deps) | 0 | N/A |
| Fichiers TODO/FIXME/HACK non datés ou sans ticket | 0 | 0 |

## Outils d'appui suggérés

- `gocyclo` / `gocognit` — complexité cyclomatique
- `go vet`, `golangci-lint run ./...`
- `go-arch-lint` ou grep manuel — vérifier imports inter-packages
- `depguard` ou lecture d'imports — vérifier couches hexagonales
- `eslint-plugin-boundaries` côté React pour les limites features
- Recherche grep : `map\[string\]any`, `interface\{\}`, `any` (TS), `// TODO`, `// FIXME`, `// HACK`, `// nolint`, `// noqa`

## Entrées pour le LLM

1. Ce `SCOPE.md`
2. La `CHECKLIST.md`
3. Le template vide `templates/axis2_quality_template.md`
4. Accès lecture au code Go + React
5. `GO_ARCHITECTURE_RULES.md` (référence)
6. `CLAUDE.md` des deux worktrees (règles qualité Python → à transposer)

## Sortie attendue

`claude_review.md` et `chatgpt_review.md` remplis, avec chaque ligne des sections A-G référencée à un fichier:ligne exact du code Go ou React.

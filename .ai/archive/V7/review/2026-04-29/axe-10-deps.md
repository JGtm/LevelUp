# Axe 10 — Dépendances & couplage tech

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Périmètre : apps/go-api/go.mod + apps/web/package.json + structure monorepo

## Synthèse

Stack saine et concentrée sur peu de libs majeures. Côté Go, runtime tient sur 12 deps directes ; couplage DuckDB excellent (driver concentré dans `internal/platform/duckdb/`), seul `database/sql` fuit massivement (112 fichiers) car volontairement utilisé dans tout le code repo/migration/sync/ops. Côté front, 12 deps runtime, alias `@/` bien configuré, mais 47 imports cross-`features/` violent la règle "pas de couplage horizontal". Pas de monorepo tooling (Turbo/pnpm-workspace) : l'orchestration repose sur le Makefile racine. Versions Go mismatch entre `go.mod` 1.26.1 et `Dockerfile` 1.24. Aucune dep deprecated, aucun replace.

## Compteurs / inventaire

- **Go version** : `go.mod` ligne 3 = `go 1.26.1` ; **Dockerfile ligne 21 = `golang:1.24-bookworm`** (mismatch)
- **Node version** : aucun `.nvmrc`, aucun `engines` dans `package.json` ; `Dockerfile` ligne 4 = `node:22-slim`
- **Dépendances Go directes** : 12 (toutes en versions taggées stables)
- **Dépendances Go indirectes** : 26 (dont 4 `v0.0.0-...` toutes sous-deps `golang.org/x/exp`, `x/telemetry`, `x/xerrors`, `pkg/browser` — toutes indirectes, normal)
- **`replace` clauses** : 0 (sain, aucun fork local)
- **Dépendances npm directes** : 12 runtime + 25 dev = 37 total
- **Lockfile front** : `apps/web/package-lock.json` (8433 lignes, présent et committé)
- **Dépendance orpheline déjà signalée (axe 9)** : `recharts ^3.8.1` dans `dependencies` — 0 import dans `apps/web/src/`

## Couplage par lib majeure

### Go

| Lib | Import path | Fichiers consommateurs | Concentration | Verdict |
|---|---|---|---|---|
| DuckDB driver | `github.com/duckdb/duckdb-go/v2` | 53 fichiers (dont 33 hors `platform/duckdb/`) | partiellement disséminé : 17 sync, 12 ops, 4 cmd, 2 service media, 2 migration | DETTE mineure |
| `database/sql` | stdlib | 112 fichiers | disséminé partout (repo, sync, ops, migration, services media) | OK (pattern Go assumé) |
| chi router | `github.com/go-chi/chi/v5` | 81 fichiers (handlers HTTP) | concentré dans `api/handlers/` + `api/server.go` | OK |
| chi cors/httprate | `github.com/go-chi/cors`, `httprate` | utilisé via middleware | concentré | OK |
| MSAL Azure | `github.com/AzureAD/microsoft-authentication-library-for-go` | 3 fichiers (`platform/auth/msal_*` + `cmd/msal-poc`) | concentré dans `platform/auth/` | OK (parfait) |
| websocket | `github.com/gorilla/websocket` | 2 fichiers (`internal/presence/`) | concentré | OK |
| OpenAPI runtime | `github.com/oapi-codegen/runtime` | 1 fichier (`internal/api/gen/types.gen.go`) | concentré (généré) | OK |
| YAML v3 | `gopkg.in/yaml.v3` | 3 fichiers (lab provider + 2 contract tests) | concentré | OK |
| TOML pelletier | `github.com/pelletier/go-toml/v2` | 5 fichiers (mappings loader + prestige catalog) | concentré dans `games/mappings/` + `prestige/` | OK |
| uuid | `github.com/google/uuid` | 6 fichiers | usage ponctuel | OK |
| `log/slog` (stdlib) | stdlib | 135+ fichiers | disséminé (pattern voulu, voir arch-rules) | OK |
| Halo provider | `internal/platform/halo` | 6 fichiers (server, registry, cmd/server, populate-assets, refresh-metadata, watcher live_refresh, home_service) | concentré | OK |
| Halo Infinite adapter | `internal/games/halo_infinite` | 6 fichiers (server, registry, services tests + parity) | concentré | OK |

### Front

| Lib | Import path | Fichiers consommateurs | Concentration | Verdict |
|---|---|---|---|---|
| ECharts (core+react) | `echarts`, `echarts-for-react` | 16 fichiers (12 dans `components/charts/` + 4 fuites features — déjà couvert axe 1) | partiellement concentré | DETTE (axe 1) |
| TanStack Query | `@tanstack/react-query` | 51 fichiers | disséminé via `features/{name}/queries.ts` (pattern voulu) | OK |
| TanStack Router | `@tanstack/react-router` | 86 fichiers | disséminé (chaque route + Link consommateur) | OK (idiomatique) |
| Zustand | `zustand` | 6 stores (`stores/` + `features/asset-drawer/`) | concentré | OK |
| Tailwind CSS | via `@tailwindcss/vite` | n/a (CSS) | n/a | OK |
| MSW | `msw` | dev/test only | concentré | OK |
| recharts | `recharts` | 0 imports | dep orpheline (axe 9) | DETTE (axe 9) |
| sonner | `sonner` | 0 imports trouvés | dep orpheline candidate | A VÉRIFIER |
| react-markdown | `react-markdown` | 2 fichiers (changelog + ReleaseNotesTab) | concentré | OK |
| intl-messageformat | `intl-messageformat` | 1 fichier (`lib/i18n/format.ts`) | concentré | OK |
| @iarna/toml | `@iarna/toml` | 1 fichier (`scripts/build_i18n_manifests.mjs`, build-time uniquement) | concentré | OK |

## Constats

### [BLOQUANT] Mismatch Go version go.mod 1.26.1 vs Dockerfile 1.24

`apps/go-api/go.mod:3` exige `go 1.26.1`. `Dockerfile:21` build-stage utilise `golang:1.24-bookworm`. Le module ne compilera pas en CI/prod si l'image Docker est utilisée. Aligner sur 1.24 (rétrograde `go.mod`) ou bumper Dockerfile à `golang:1.26-bookworm`.

### [BLOQUANT] Dépendance front `sonner` listée mais aucun import

`apps/web/package.json:34` déclare `"sonner": "^2.0.7"` en `dependencies`. Aucune occurrence `from "sonner"` ni `from 'sonner'` dans `apps/web/src/`. Soit la lib est utilisée via re-export (à vérifier), soit elle est orpheline. À ajouter à la liste de l'axe 9.

### [BLOQUANT] Mention obsolète `plotly.js` / `react-plotly.js` dans `.ai/project_map.md`

`.ai/project_map.md:154` affirme "`apps/web/package.json` : dépendance explicite `plotly.js`, requise au build par `react-plotly.js`". Aucune trace de `plotly.js` ou `react-plotly` dans `package.json` ni dans `apps/web/src/`. Le project_map.md est désynchronisé sur la stack charts (ECharts seul). À corriger.

### [DETTE] 47 imports cross-`features/` détectés (couplage horizontal)

`apps/web/src/features/{a}` importe `@/features/{b}/...` dans 25 fichiers, 47 occurrences. Top contrevenants :
- `features/career/CareerProgressionTab.tsx` : 3 imports vers compare/leaderboard
- `features/settings/SettingsPage.tsx` : 8 imports cross-features (notifications, setup)
- `features/squad/SquadLayout.tsx` : 4 imports cross-features
- `features/home/HomePage.tsx` : 2 imports vers prestige/match-history
- `features/explorer/ExplorerPage.tsx` : 2 imports vers compare
- `features/match-view/MatchViewPage.tsx` : 2 imports vers engagement/match-history

Selon la règle énoncée (axe 4), `features/{titre}/` ne doit pas importer un autre `features/{autre}/`. Les composants partagés type `CompareDrawer`, `LeaderboardBlock`, `EngagementMatchSection` devraient remonter dans `components/` ou `lib/`. À traiter par lots, non bloquant mais dette structurelle réelle.

### [DETTE] Driver DuckDB importé dans 33 fichiers hors `platform/duckdb/`

`github.com/duckdb/duckdb-go/v2` (CGo bindings) est importé dans :
- `internal/sync/` : 17 fichiers (intégration tests + writes/career/performance/aggregates/skill_rating ; le driver est déclaré pour son side-effect `_` import sur connection ouverte ; à vérifier si nécessaire ou si stdlib `database/sql` suffit)
- `internal/ops/` : 12 fichiers (backup/restore/diagnose/seed/healthcheck/media — accepts opt-in CGo)
- `internal/service/media_service.go`, `internal/service/media_index_service.go` : 2 services qui devraient passer par repo (déjà flag axe 3)
- `internal/migration/` : 2 fichiers de tests
- `cmd/` (binaires utilitaires) : 4 fichiers — légitime

Le pattern habituel Go est : `_ "github.com/duckdb/duckdb-go/v2"` dans le main pour enregistrer le driver auprès de `database/sql`, puis tout le reste passe par `*sql.DB`. Les 17 imports dans `sync/` et 12 dans `ops/` méritent un audit pour vérifier qu'il s'agit bien de blank-imports CGo et non d'un usage direct du driver. Si direct usage, à concentrer.

### [DETTE] `home_service.go` importe directement `internal/platform/duckdb`

`apps/go-api/internal/service/home_service.go:18` : `"levelup/go-api/internal/platform/duckdb"`. Le service couche métier ne devrait dépendre que de `port.*` interfaces. Constat déjà signalé axe 3, je le re-mentionne pour traçabilité dans cet axe (couplage entre services et infra). Idem `media_service.go` et `media_index_service.go`.

### [DETTE] Aucun `engines` ni `.nvmrc` côté front

`apps/web/package.json` n'a pas de champ `engines`, et la racine n'a pas de `.nvmrc`. Le Dockerfile fixe `node:22-slim` mais rien ne contraint le dev local à utiliser Node 22. Risque de "ça marche chez moi". Ajouter `"engines": { "node": ">=22.0.0", "npm": ">=10.0.0" }` ou un `.nvmrc`.

### [DETTE] Imports relatifs profonds résiduels (24 fichiers)

23 fichiers utilisent `../../...` ou plus profond, 1 fichier utilise `../../../`. Concentration dans `apps/web/src/lib/accessibility/` (tests internes au module — acceptable) et `features/squad/v2/components/` (acceptable, sous-arbre du même feature). Pas de `../../../` cross-feature détecté. Volume faible : non bloquant.

### [AMÉLIORATION] Pas de monorepo tooling

Aucun `pnpm-workspace.yaml`, `turbo.json`, `nx.json`, ni `package.json` racine. La structure `apps/go-api/` + `apps/web/` est traitée comme deux projets indépendants orchestrés par le `Makefile` racine. C'est défendable (deux stacks différentes Go/TS, pas de code TS partagé), mais empêche tout caching de build incrémental ou commande `pnpm -r ...`. Si le projet ajoute un futur `apps/cli/` ou `packages/shared-types/`, prévoir Turborepo.

### [AMÉLIORATION] OpenAPI workflow propre mais à document

`apps/web/package.json:20` script `generate-types` lit `../../apps/go-api/api/openapi.yaml` et écrit `src/lib/api/generated.ts`. Le YAML est donc source-of-truth ; reste à vérifier (axe 7 / parité 45 vs 102 routes) si toutes les routes Go sont bien dans le YAML. Aucun pré-commit hook ni script CI vérifiant la fraîcheur de `generated.ts` vs `openapi.yaml` détecté (pas exhaustif).

### [AMÉLIORATION] Versions exactes vs flottantes — front 100% caret

Toutes les deps front utilisent `^x.y.z` (zéro version exacte). Avec un lockfile committé, le risque de drift est borné mais à `npm install --no-package-lock` ou nouveau dev, des minor bumps peuvent introduire des régressions silencieuses. Pas critique tant que `npm ci` est utilisé (Dockerfile:11 le fait correctement).

## Constats hors-axe

- **Axe 1 (ECharts)** : 4 fuites `EChartsCoreOption` inline dans features confirmées (couplage à hisser dans `components/charts/`).
- **Axe 9 (dead deps)** : `recharts` confirmé orphelin ; `sonner` candidat orphelin à ajouter à la liste.
- **Doc** : `.ai/project_map.md` mentionne plotly et python venv (CLAUDE.md aussi), à mettre à jour pour Go+React.

## Suivi recommandé

1. **Aligner Go versions** (Dockerfile 1.24 -> 1.26 ou go.mod -> 1.24) et ajouter un check CI qui parse les deux. Trivial, à faire avant la prochaine livraison.
2. **Vérifier `sonner`** : soit l'utiliser pour toasts (en cohérence avec `features/notifications/toastBridge.tsx`), soit la retirer. Faire une passe rapide sur `dependencies` post-axe 9.
3. **Documenter cross-feature imports** : graph rapide des 47 cas, identifier les composants à hisser (`CompareDrawer`, `LeaderboardBlock`, `EngagementMatchSection`, `ChallengesCarousel`, `BattlePassRewardLightbox`, `MediaViewer/MediaLightbox`) dans `components/shared/` ou un `features/_shared/`.

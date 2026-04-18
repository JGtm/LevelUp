# Sprint Exploration — LevelUp Go Migration

> Ce fichier documente l'état d'exploration du codebase Go pour les agents IA.

## Référence principale

Le suivi des sprints est maintenu dans [`SPRINT_ROADMAP.md`](.ai/go_migration_v2/SPRINT_ROADMAP.md).

## État actuel (Phase 11 — Sprint 49)

### Packages Go compilables localement (sans CGO/DuckDB)

| Package | Description |
|---------|-------------|
| `internal/domain/...` | Types métier purs (0 import externe) |
| `internal/analysis/...` | Algorithmes d'analyse (sessions, performance) |
| `internal/domain/title/...` | Registre multi-titres |

### Packages nécessitant CGO (DuckDB)

| Package | Description |
|---------|-------------|
| `internal/platform/duckdb/...` | Pool de connexions DuckDB |
| `internal/service/...` | Services métier |
| `internal/api/handlers/...` | Handlers HTTP (transitif via config) |
| `internal/api/...` | Router chi, middleware, server |

### Architecture contractuelle

- **OpenAPI** : `api/openapi.yaml` — source de vérité
- **Test contrat** : `internal/api/contract_test.go` — vérifie alignement OpenAPI ↔ chi
- **Exemptions** : 0 (vidées au Sprint 49)
- **Routes match exclusion** : `PATCH /players/{player_slug}/matches/{match_id}/exclusion` + `GET /players/{player_slug}/match-exclusions` documentées dans OpenAPI et branchées dans le router chi
- **Auth Halo live** : Battle Pass / Challenges lisent désormais `HaloTokens` + `XUID` depuis `ctxkeys`, injectés par le middleware de session

### Points d'entrée clés

| Fichier | Rôle |
|---------|------|
| `cmd/server/main.go` | Point d'entrée du serveur |
| `internal/api/server.go` | Assembly du router chi |
| `internal/api/middleware/session.go` | Injection session HTTP + auth Halo dans le contexte |
| `internal/service/bootstrap_service.go` | Bootstrap du shell React |
| `internal/api/handlers/session_context.go` | Contexte session (titre, joueur, locale) |
| `internal/api/handlers/match_exclusion.go` | Exclusion manuelle de matchs au niveau joueur |
| `internal/platform/halo/provider.go` | Provider Halo live pour Battle Pass / Challenges |
| `internal/ctxkeys/ctxkeys.go` | Clés de contexte partagées titre + auth Halo |
| `internal/domain/title/registry.go` | Registre des titres supportés |

### Frontend web — shell joueur (2026-04-18)

- `apps/web/src/components/shell/AppShell.tsx` : shell désormais sans sidebar, avec header global sticky et zone de rendu centrée.
- `apps/web/src/components/shell/AppShellHeader.tsx` : identité produit, titre courant, liens utilitaires et sélecteur de joueur branché au router.
- `apps/web/src/components/shell/PlayerScopeNav.tsx` : navigation du scope joueur découpée entre parcours principaux et vues secondaires, désormais rendue en `nav` sémantique pour l'accessibilité et les tests E2E.
- `apps/web/src/components/shell/shellNavigation.ts` : définition des items de navigation et helper `buildPlayerDestination()` pour préserver la section active lors d'un switch joueur.
- `apps/web/src/routes/players/$playerSlug.tsx` : montage du nouveau scope joueur (`PlayerScopeNav` + `KPIBar` + contenu).
- `apps/web/src/features/home/HomePage.tsx` + `apps/web/src/components/shell/KPIBar.tsx` : correction du contrat KPI côté frontend (`win_rate` = ratio, `avg_accuracy` = pourcentage déjà normalisé), et passage des liens player-scoped en routes typées TanStack Router.
- `apps/web/src/components/ui/empty-state.tsx` + pages player-scoped (`home`, `career`, `timeseries`, `squad`, `citations`, `synthesis`, `sessions`, `explorer`) : harmonisation des états vides, avec messages explicites quand une payload manque ou qu'une section analytique ne peut pas être rendue.
- Validation locale finale : `npm run -s typecheck` OK, `npm run -s build` OK, `vitest` OK sur `shellNavigation.test.ts`, `playwright` OK sur `e2e/slice-0a-shell.spec.ts` (5/5).

### Validation locale Go + React (2026-04-18)

- `CGO_ENABLED=1 go test -tags=integration ./... -timeout 120s -count=1` : OK.
- `apps/web` : `npm run typecheck`, `npm run lint`, `npm run build`, `npm run test:run`, `npm run test:e2e` : OK.
- `apps/web` : revalidation ciblée empty states via `vitest` (`HomePage`, `CareerPage`, `SquadPage`, `SynthesisPage`, `ExplorerPage`) + `npm run build` : OK.
- Les corrections déterminantes sur cette passe ont porté sur les tests Go dupliqués, l'alignement `MatchHistoryRepo`/`Q5MatchHistory`, le hook conditionnel de `MediaPage`, et la mise à jour des specs Vitest obsolètes pour `SetupPage`, `SquadPage`, `SynthesisPage`, `MediaPage` et `HomePage`.

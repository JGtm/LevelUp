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

### Points d'entrée clés

| Fichier | Rôle |
|---------|------|
| `cmd/server/main.go` | Point d'entrée du serveur |
| `internal/api/server.go` | Assembly du router chi |
| `internal/service/bootstrap_service.go` | Bootstrap du shell React |
| `internal/api/handlers/session_context.go` | Contexte session (titre, joueur, locale) |
| `internal/domain/title/registry.go` | Registre des titres supportés |

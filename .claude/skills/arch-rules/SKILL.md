# Skill : arch-rules — Architecture Go, séparation des responsabilités, testabilité

## Couches internes (internal/)

```
api/handlers/    — HTTP : décode requête, appelle service via port interface, encode réponse
api/middleware/  — Cross-cutting : auth, CSRF, rate-limit, slog HTTP, title, ownership
port/            — Interfaces : StatsService, Repository… (découplage handlers↔services)
service/         — Orchestration : combine repo + analysis, implémente port.*Service
domain/          — Types métier purs : structs, enums — 0 DB, 0 HTTP
analysis/        — Algos purs : fonctions stateless — 0 DB, 0 HTTP, 0 side-effect
platform/duckdb/ — Infrastructure : implémente port.Repository, accès DuckDB
platform/auth/   — Tokens (MultiUserTokenStore, ADR 0023), échanges Halo
persist/         — Écritures per-match : BatchBuilder + Persisters INSERT-only (ADR 0019)
sync/ + sync/v2/ — Moteur de synchronisation (V2 = cycle orchestrator, ADR 0027)
games/           — Par titre : adapters + client + livesync (halo_infinite/, halo_5/) + canonical/
migration/       — DDL + migrations de schéma (dont append-only rebuild, ADR 0026)
config/          — Configuration + feature flags
ops/             — Backup, restore, seed demo — ATTENTION : certains chemins tournent in-process
```

**Note racine `api/`** : la racine (hors handlers/ et middleware/) est réservée à la DI
(registry/wire). Ne JAMAIS y ajouter de SQL, de logique métier ou de runner — c'est la
dérive n°1 identifiée par l'audit archi 2026-07.

## Règle : où va ce code ?

| Ce que fait la fonction | Package |
|---|---|
| Transformation pure / calcul | `analysis/` |
| Type partagé entre couches | `domain/` |
| Combine DB + algo → réponse | `service/` |
| Interface entre handler et service | `port/` |
| Décode HTTP → appelle service → encode JSON | `api/handlers/` |
| Requête DuckDB (lecture) | `platform/duckdb/` |
| Écriture per-match sur DB partagée | `persist/` (BatchBuilder → Persister, INSERT-only) |
| Code spécifique à un titre (parsing, client, film) | `games/{slug}/` |
| DDL / évolution de schéma | `migration/` |

## Port pattern — pourquoi et comment

```go
// port/services.go — interface définie ici
type StatsService interface {
    GetStatsPage(ctx context.Context, params StatsParams) (*domain.StatsPage, error)
}

// api/handlers/stats.go — consomme l'interface, jamais le type concret
type StatsHandler struct {
    newSvc ServiceFactory[port.StatsService]
}

// service/stats_service.go — implémente l'interface
```

Bénéfice : handler testable avec mock, service testable avec repo mock, analysis testable seul.

## Anti-patterns à éviter

```go
// Handler qui court-circuite les couches
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
    rows, _ := h.db.Query("SELECT ...")  // accès DB dans handler = non
}

// Service qui appelle un autre service (couplage horizontal)
func (s *StatsService) Get(...) {
    result := s.sessionSvc.GetSessions(...)  // non
}

// Logique métier dans un handler
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
    ratio := float64(kills) / float64(deaths)  // calcul = analysis/, pas ici
}
```

## Testabilité par couche

| Couche | Stratégie |
|---|---|
| `analysis/` | Tests unitaires purs, 0 dépendance externe |
| `service/` | Mock `port.Repository` via interface |
| `api/handlers/` | `httptest.NewRecorder` + mock `port.*Service` |
| `platform/duckdb/` | DuckDB `:memory:` |

## Logging — `log/slog` (stdlib)

```go
// Pattern standard : ctx + clés structurées
slog.DebugContext(ctx, "resolveCurrentSeason", "titleSlug", titleID)
slog.InfoContext(ctx, "match loaded", "match_id", matchID, "player", gamertag)
slog.ErrorContext(ctx, "db query failed", "err", err, "query", queryName)

// Interdit
fmt.Println("debug:", x)  // non structuré
log.Printf("...")          // ancien package log
```

Clés standard du projet : `"err"`, `"match_id"`, `"player"`, `"titleSlug"`, `"duration"`.

## Modularité

| Unité | Seuil | Action |
|---|---|---|
| Fichier `.go` | 500 lignes | Extraire `*_helpers.go`, `*_types.go`, ou sous-package |
| Fonction | 80 lignes | Extraire sous-fonctions nommées |
| Paramètres | 5 | Regrouper en struct |

Nommage : `1 verbe + 1 complément`. `computeKD` + `renderKD`, jamais `computeAndRenderKD`.

## Multi-titres — règles

### PathResolver : point unique pour les chemins
**Jamais** de `filepath.Join(repoRoot, "data", ...)` en dehors de `PathResolver`.

```go
// Tous les chemins physiques passent par PathResolver
paths.SharedDBPath(titleSlug)      // data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb
paths.PlayerDBPath(titleSlug, gt)  // data/titles/halo_infinite/players/GT/stats.duckdb
```

### TitleRegistry — source de vérité
`internal/domain/title/registry.go` — titre courant injecté dans le contexte par le middleware `TitleExtractor`.

```go
// Récupérer le titre courant dans un handler/service
slug := ctxkeys.TitleSlug(ctx)  // "halo_infinite" par défaut

// Vérifier une capability avant d'activer une feature
title := registry.Get(slug)
if !title.HasCapability(title.CapFirefight) {
    // page PvE non disponible pour ce titre
}
```

### Capabilities — brancher le code par fonctionnalité, pas par titre

```go
// Correct — brancher sur capability
if desc.HasCapability(title.CapRanked) { ... }

// Interdit — brancher sur le slug
if slug == "halo_infinite" { ... }  // couplage fort, non extensible
```

Capabilities disponibles : `CapMatchmaking`, `CapFirefight`, `CapForge`, `CapMedia`, `CapRanked`, `CapCareer`.

### Header HTTP
Le titre courant peut être passé par `X-LevelUp-Title` (clients API directs). Le middleware `TitleExtractor` le résout automatiquement (header → session → fallback `halo_infinite`).

### Adapters — pattern Phase B

Deux interfaces séparées (SRP) dans `internal/games/adapter.go` :

| Interface | Rôle | Source |
|---|---|---|
| `TitleDataAdapter` | Charge les données du titre → format canonique | DuckDB title-specific |
| `TitleSemanticAdapter` | Expose les libellés, assets, outcomes | TOML versionnés Git |

```go
// Un service ne dépend que de ce dont il a besoin
type MyService struct {
    data     games.TitleDataAdapter     // si besoin de données
    semantic games.TitleSemanticAdapter // si besoin de libellés
}

// Obtenir les adapters via Resolver (injecté à la DI au boot)
data, err := resolver.Data(titleSlug)
if err != nil { ... }

// Dégradation gracieuse — ne jamais paniquer sur une capability absente
summaries, err := data.LoadMatchSummaries(ctx, ids)
if errors.Is(err, games.ErrCapabilityNotSupported) {
    // dégrader proprement, ex: retourner une réponse partielle
}
```

**CapabilityKey produit** (plus fins que les `Capability` du TitleRegistry) :
`"match.history"`, `"match.detail.core"`, `"match.skill.snapshot"`, `"career.progression"`, `"pve.firefight_stats"`, `"analytics.timeseries"`.

### TOML mappings — `config/titles/{slug}/mappings/`

| Fichier | Contenu |
|---|---|
| `fields.toml` | Labels EN/FR, unit, format, display_order par FieldKey canonique |
| `assets.toml` | Assets par mode (icônes), tiers médailles, statuts challenges |
| `outcomes.toml` | Labels + color tokens par résultat (win/loss/tie/dnf) |

**Règle** : un champ absent du TOML = le titre ne supporte pas cette surface produit. Le service doit dégrader, jamais paniquer.

### Ajouter un nouveau titre

1. `internal/games/{new_title}/adapter_data.go` + `adapter_semantic.go` — implémenter les 2 interfaces
2. `config/titles/{new_title}/mappings/fields.toml` + `assets.toml` + `outcomes.toml`
3. Enregistrer dans `TitleRegistry` + `Resolver` au boot (`api/server.go`)
4. Brancher sur `HasCapability()` côté domain, jamais sur le slug

## Feature flags & kill-switches

- Lire les env vars au boot via `internal/config` et injecter — pas de `os.Getenv`
  dispersés dans les packages (risque de divergence entre deux lecteurs du même flag).
- Supprimer le flag et son branchement une fois la feature stabilisée en prod. Ne pas
  laisser de code mort conditionnel.
- Un kill-switch conservé volontairement porte dans son commentaire : date du basculement
  de défaut + date cible de retrait + critère mesurable. Modèle à copier :
  `platform/duckdb/shared_reader_legacy.go`.
- Quand le DÉFAUT d'un flag bascule : mettre à jour tous ses commentaires/doc dans le
  même commit (une doc inversée sur un flag anti-corruption est un incident en attente).

# Logging multi-module — `internal/observability/logging/`

Sprint B1 commit 16 — système de logs structurés avec dispatch console + fichiers par module, et traçabilité cross-module via `event_id`.

## Pourquoi

Le sync engine, le `SharedDBProvider`, le pool joueur, les handlers HTTP et l'auto_sync scheduler logguent tous via `slog`. Sans dispatch, tout part dans un seul flux stderr — difficile de :

- Diagnostiquer un sync planté (logs noyés dans le trafic HTTP).
- Suivre un événement business (un `RunDelta` qui déclenche un swap RW Provider qui notifie le pool) à travers les modules.
- Conserver l'historique en JSON parsable pour outils ops.

## Architecture

```
                                  ┌── stderr (console, format texte ou JSON)
                                  │   conserve le comportement pré-sprint
slog.Info/Warn/Error  ────────────┤
                                  │
                                  └── logs/{module}.log (JSON append-only)
                                      un fichier par module, créé lazy
```

Le `MultiModuleHandler` (`multi_module_handler.go`) wrappe le handler console (`ContextHandler` existant) et duplique chaque record vers `logs/{module}.log`.

Le module d'un record est résolu dans cet ordre :

1. Attribut explicite `module=...` sur le record (`slog.With("module", "sync")`).
2. Détection auto depuis le PC d'appel (cf. `module.go::detectModuleFromCaller`).
3. Fallback `ModuleGeneral` → `logs/general.log`.

## Configuration

| Variable | Défaut | Effet |
|---|---|---|
| `LEVELUP_LOGS_ENABLED` | `true` | Kill-switch global. `false` → comportement pré-sprint (console only). |
| `LEVELUP_LOGS_DIR` | `<repoRoot>/logs` ou `logs/` | Répertoire de destination des fichiers `{module}.log`. |
| `LEVELUP_LOGS_FILE_LEVEL` | `info` | Niveau minimal écrit dans les fichiers. `debug` capture tout, `warn` minimise le volume. |

Les variables console pré-existantes (`LEVELUP_LOG_JSON`, `LEVELUP_LOG_LEVEL`) restent inchangées.

## Modules

Constantes dans `module.go`. Liste non-exclusive — un module non listé reçoit son propre fichier sans configuration préalable.

| Module | Source typique | Fichier |
|---|---|---|
| `sync` | `internal/sync/` (engine, RunDelta, RunBackfill) | `logs/sync.log` |
| `provider` | `internal/platform/duckdb/sharedprovider/` (swap RO↔RW, drain) | `logs/provider.log` |
| `pool` | `internal/platform/auth/pool/` (token discovery, refresh) | `logs/pool.log` |
| `scheduler` | `internal/scheduler/` (auto_sync) | `logs/scheduler.log` |
| `handlers` | `internal/api/handlers/` (endpoints HTTP) | `logs/handlers.log` |
| `service` | `internal/service/` (orchestration métier) | `logs/service.log` |
| `duckdb` | `internal/platform/duckdb/` (OpenRO, dblease, pool joueur) | `logs/duckdb.log` |
| `auth` | `internal/platform/auth/` (MSAL, XSTS, watcher) | `logs/auth.log` |
| `assets`, `prestige`, `media`, `notifications`, `migration`, `health` | … | `logs/{module}.log` |
| `general` | fallback | `logs/general.log` |

## Référençabilité cross-module : `event_id`

Pour suivre une opération business (sync, swap RW, backfill HTTP) à travers les fichiers logs, le code instancie un identifiant d'événement au début de l'opération :

```go
import "levelup/go-api/internal/observability/logging"

func (e *SyncEngine) run(ctx context.Context, ...) {
    ctx, eventID := logging.WithEvent(ctx, "sync.RunDelta")
    slog.InfoContext(ctx, "sync: démarrage", "gamertag", gt)
    // ... toutes les fonctions appelées avec ce ctx auront `event_id=sync.RunDelta:abc123`
}
```

Le `ContextHandler` (chaîné avant `MultiModuleHandler`) ajoute automatiquement `event_id` sur chaque record via le ctx. Résultat : grep cross-module pour reconstituer un timeline :

```bash
# Tous les logs d'un sync donné, dans l'ordre chronologique
grep -h 'event_id="sync.RunDelta:abc123"' logs/*.log | jq -r '"\(.time) [\(.level)] \(.msg)"' | sort
```

## Helpers

```go
// Crée un nouvel event_id + le met dans ctx.
ctx, id := logging.WithEvent(ctx, "swap.RoToRw")

// Lit l'event_id courant (vide si absent).
id := logging.CurrentEvent(ctx)

// Surcharge manuelle du module pour un logger spécifique.
logger := slog.Default().With("module", "custom_module")
logger.Info("...") // → logs/custom_module.log
```

## Tests

```bash
go test -tags=integration -race -count=1 ./internal/observability/logging/
```

11 tests couvrent : dispatch console+fichier, routing par attribut module, propagation event_id, fallback general, level filtering, Close idempotent, sanitization de noms, env vars config.

## Limitations connues

- **Pas de rotation automatique** : les fichiers grossissent indéfiniment. À gérer en ops (logrotate, cron). Le helper `archivedLogPath` est prévu pour un futur sprint.
- **Détection auto fragile** : repose sur `runtime.FuncForPC` et le nom de package. Si un package est renommé ou réorganisé, le mapping (`module.go::mapPackageToModule`) doit être mis à jour. Override possible via `slog.With("module", ...)`.
- **Pas de compression** : les `.log` sont en texte brut UTF-8 JSON-line, ~150 octets/ligne.
- **Aucun upload distant** : strictement local. Pour cloud logging (Loki, Splunk, etc.), agent système séparé recommandé sur le format JSON existant.

## Ajouter un nouveau module

Trois options :

1. **Détection auto** : si le package vit sous `internal/foo/`, ajouter un case `"foo": return "foo"` dans `mapPackageToModule` (module.go). Aucun autre changement nécessaire — le fichier `logs/foo.log` est créé lazy au premier log.
2. **Override explicite** : `logger := slog.Default().With("module", "foo")` au top du package, utiliser ce logger partout. Indépendant de la détection auto.
3. **Ad-hoc** : `slog.With("module", "foo").Info("...")` ponctuellement. Le fichier est créé lazy.

## Voir aussi

- `internal/observability/context_handler.go` — chaîne `request_id` + `event_id` depuis le ctx.
- `internal/ctxkeys/ctxkeys.go` — clés `EventID` et `RequestID`.
- ADR 0016 — SharedDBProvider B-swap (modules les plus loggués).

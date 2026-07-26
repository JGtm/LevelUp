# Logging multi-module — `internal/observability/logging/`

Sprint B1 commits 16-20 — système de logs structurés avec :
- dispatch console + fichiers par module
- traçabilité cross-module via `event_id`
- format console compact (commit 20) avec tronquage + skip d'attrs verbeux

## Pourquoi

Le sync engine, le `SharedDBProvider`, le pool joueur, les handlers HTTP et l'auto_sync scheduler logguent tous via `slog`. Sans dispatch, tout part dans un seul flux stderr — difficile de :

- Diagnostiquer un sync planté (logs noyés dans le trafic HTTP).
- Suivre un événement business (un `RunDelta` qui déclenche un swap RW Provider qui notifie le pool) à travers les modules.
- Conserver l'historique en JSON parsable pour outils ops.

## Architecture

```
                                  ┌── stderr (console)
                                  │   format au choix : compact (défaut), text, json
slog.Info/Warn/Error  ────────────┤
                                  │
                                  └── logs/{module}.log (JSON append-only)
                                      un fichier par module, créé lazy
```

Le `MultiModuleHandler` (`multi_module_handler.go`) wrappe le handler console (`ContextHandler` → `ConsoleHandler` ou JSON/Text) et duplique chaque record vers `logs/{module}.log`.

### Format console "compact" (défaut)

```
14:30:08 [INFO]  sync.postSync: pipeline démarré matches_inserted=3
14:30:09 [WARN]  halo_api: GET HTTP error status=429 url=…/spnkr
14:30:12 [ERROR] provider.swap: reopen RO failed attempts=3 err=…
```

Conventions :
- **Time** : `HH:MM:SS` (date complète préservée dans les fichiers JSON).
- **Level** : `[INFO]`/`[WARN]`/`[ERROR]`/`[DEBUG]` padded à 7 chars pour alignement vertical de la colonne message.
- **Message** : brut, sans préfixe ni quotes.
- **Attrs** : `key=value` espace-séparés ; valeurs avec espace, `"` ou `=` automatiquement quotées.
- **Skip attrs console** : `event_id`, `request_id`, `source.function`, `source.file`, `source.line`, `source` sont **masqués sur console** (préservés dans les fichiers JSON pour grep cross-module).
- **Tronquage** : ligne > `LEVELUP_LOG_MAX_LINE` (défaut 200) suffixée `…` (1 rune Unicode).
- **Couleurs ANSI** : opt-in via `LEVELUP_LOG_COLOR=on` (off par défaut pour Windows cmd.exe).

Le module d'un record est résolu dans cet ordre :

1. Attribut explicite `module=...` sur le record (`slog.With("module", "sync")`).
2. Détection auto depuis le PC d'appel (cf. `module.go::detectModuleFromCaller`).
3. Fallback `ModuleGeneral` → `logs/general.log`.

## Configuration

### Fichiers (par module)

| Variable | Défaut | Effet |
|---|---|---|
| `LEVELUP_LOGS_ENABLED` | `true` | Kill-switch global. `false` → console only. |
| `LEVELUP_LOGS_DIR` | `<repoRoot>/logs` ou `logs/` | Répertoire de destination des fichiers `{module}.log`. |
| `LEVELUP_LOGS_FILE_LEVEL` | `info` | Niveau minimal écrit dans les fichiers. `debug` capture tout, `warn` minimise le volume. |
| `LEVELUP_LOGS_MAX_SIZE_MB` | `100` | Taille max d'un `{module}.log` avant rotation. `0` désactive la rotation (croissance illimitée). |
| `LEVELUP_LOGS_MAX_BACKUPS` | `3` | Archives `{module}.log.1..N` conservées. `0` = aucune archive. |

### Console

| Variable | Défaut | Effet |
|---|---|---|
| `LEVELUP_LOG_LEVEL` | `info` | Niveau minimal console : `debug`/`info`/`warn`/`error`. |
| `LEVELUP_LOG_FORMAT` | `compact` | `compact` (ConsoleHandler, défaut) / `text` (slog natif, verbeux) / `json` (prod). |
| `LEVELUP_LOG_MAX_LINE` | `200` | Tronquage console (caractères). `0` désactive. Format `compact` uniquement. |
| `LEVELUP_LOG_COLOR` | `off` | Codes ANSI couleur (`on`/`off`). `off` par défaut pour Windows cmd.exe. |
| `LEVELUP_LOG_JSON` | `false` | **Legacy** — équivalent à `LEVELUP_LOG_FORMAT=json` si `LEVELUP_LOG_FORMAT` non défini. |

### Exemples

```bash
# Dev (défaut) : console compact lisible, fichiers debug-complets
./server

# Dev verbeux : voir les attrs source.* / event_id / request_id directement en console
LEVELUP_LOG_FORMAT=text ./server

# Production : JSON stdout pour aggregator (Loki, ELK), pas de fichiers locaux
LEVELUP_LOG_FORMAT=json LEVELUP_LOGS_ENABLED=false ./server

# Debug terminal large (ultra-wide), 300 chars max
LEVELUP_LOG_MAX_LINE=300 LEVELUP_LOG_COLOR=on ./server

# Aucun tronquage (debug exhaustif en console)
LEVELUP_LOG_MAX_LINE=0 ./server
```

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

Suite couvre :
- `multi_module_handler_test.go` (11 tests) — dispatch console+fichier, routing par attribut module, propagation event_id, fallback general, level filtering, Close idempotent, sanitization, env vars config.
- `console_handler_test.go` (20+ tests) — format compact, padding levels, level filtering, skip attrs default + custom, tronquage UTF-8, quoting valeurs avec espace/=/quote, types numériques (int/float/bool/duration), WithAttrs, codes ANSI on/off, écriture concurrente, `parseConsoleFormat` (matrice format × legacy JSON), `parseIntEnv`.

## Rotation par taille (`rotation.go`)

Chaque `{module}.log` est roté **par taille**, uniformément pour toutes les catégories :

```
provider.log      (courant, < LEVELUP_LOGS_MAX_SIZE_MB)
provider.log.1    (archive la plus récente)
provider.log.2
provider.log.3    (la plus ancienne ; supprimée à la rotation suivante)
```

- Défaut : **100 Mo × 3 archives** → ~400 Mo par catégorie au pire.
- Les archives ne se terminent pas par `.log` : le viewer admin (`ops.ListLogModules`)
  et `logtail` les ignorent déjà, aucune n'apparaît comme faux module.
- Multi-process (serveur + CLIs `cmd/*`) : l'append reste atomique ; le writer qui perd
  la course au rename le détecte via `os.SameFile` et se contente de ré-ouvrir le fichier
  neuf, sans décaler les archives une seconde fois.
- Échec de rotation (disque plein, fichier verrouillé) : signalé sur stderr, écriture
  poursuivie dans le fichier courant, nouvelle tentative après 1 min de cooldown.

Historique : avant le 2026-07-26, aucune rotation n'existait côté Go (les `app.log.1/.2/.3`
vus en prod étaient des reliquats du `RotatingFileHandler` Python supprimé avec la
migration) — la prod avait accumulé 2,1 Go, dont 1,5 Go pour `provider.log`.

## Limitations connues
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

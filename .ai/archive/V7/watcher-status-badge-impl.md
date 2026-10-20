# Watcher Status Badge — Plan d'implémentation

> Feature : indicateur visuel en temps réel de l'état de sync dans la NavL1,
> à droite du sélecteur de joueur. Poll toutes les 30s. 100% Go + React/TS.

---

## États et rendu visuel

| `state`    | Symbole        | Couleur   | Animation | Signification end-user              |
|------------|----------------|-----------|-----------|-------------------------------------|
| `ok`       | Check (✓)      | Vert      | Aucune    | Tout synced, rien en attente        |
| `syncing`  | Flèches cercle | Blanc     | Rotation  | Sync en cours                       |
| `pending`  | Flèches cercle | Orange    | Rotation  | Match(s) détecté(s), sync imminente |
| `error`    | Flèches cercle | Rouge     | Rotation  | Dernier sync échoué                 |

**Contraintes visuelles :**
- Pas d'emojis — SVG inline uniquement
- Le badge est cliquable (futur : ouvre un popover de détail — prévu dans le type mais non implémenté en V1)
- Tooltip au survol avec message contextuel (ex: "Sync en cours…", "Erreur — consulter les logs")
- Aria-label dynamique selon l'état (accessibilité)
- Loading initial : aucun symbole (invisible) pendant le premier fetch

---

## Règle de dérivation de l'état (côté Go)

Priorité décroissante — premier match gagne :

```
1. Job actif (queued | running) pour ce joueur   → "syncing"
2. Matchs en attente (futur watcher) > 0         → "pending"  ← retourne "ok" jusqu'au sprint watcher
3. Dernier job terminal = failed | interrupted   → "error"
4. Sinon                                         → "ok"
```

Le `"pending"` est prévu dans les types mais retourne `"ok"` tant que le daemon
watcher (`internal/watcher/`) n'est pas implémenté. Pas de fausse erreur.

---

## Backend Go

### Fichiers à créer / modifier

| Fichier | Action | Rôle |
|---------|--------|------|
| `internal/domain/watcher.go` | Créer | Types + logique de dérivation |
| `internal/domain/watcher_test.go` | Créer | Tests unitaires table-driven |
| `internal/api/handlers/watcher_handler.go` | Créer | Handler HTTP + slog |
| `internal/api/handlers/watcher_handler_test.go` | Créer | Tests HTTP table-driven |
| `internal/api/server.go` | Modifier | Brancher la route |
| `internal/platform/jobs/store.go` | Modifier | Ajouter `FindLatestTerminalJob` |

---

### `internal/domain/watcher.go`

```go
package domain

import "time"

type WatcherState string

const (
    WatcherStateOK      WatcherState = "ok"
    WatcherStateSyncing WatcherState = "syncing"
    WatcherStatePending WatcherState = "pending"
    WatcherStateError   WatcherState = "error"
)

type WatcherStatus struct {
    PlayerSlug    string       `json:"player_slug"`
    State         WatcherState `json:"state"`
    PendingCount  int          `json:"pending_match_count"`
    LastSyncAt    *time.Time   `json:"last_sync_at,omitempty"`
    LastSyncState *JobStatus   `json:"last_sync_state,omitempty"`
    ActiveJobID   *string      `json:"active_job_id,omitempty"`
}

// WatcherStateProvider permet au futur daemon watcher d'injecter des matchs en attente.
type WatcherStateProvider interface {
    PendingMatchCount(playerSlug string) int
}

// NoopWatcherProvider : implémentation nulle utilisée jusqu'au sprint watcher.
// Retourne toujours 0 — aucun effet de bord.
var NoopWatcherProvider WatcherStateProvider = noopWatcherProvider{}

type noopWatcherProvider struct{}
func (noopWatcherProvider) PendingMatchCount(_ string) int { return 0 }

// DeriveWatcherStatus calcule WatcherStatus depuis la liste de tous les jobs du joueur
// + le nombre de matchs en attente fourni par le provider.
// jobs : liste des jobs pour ce playerSlug (déjà filtrée ou non — la fonction filtre elle-même).
func DeriveWatcherStatus(playerSlug string, jobs []*AsyncJobStatus, pendingCount int) WatcherStatus
```

**Tests unitaires `watcher_test.go` — cas à couvrir :**

| Scénario | Input | État attendu |
|----------|-------|--------------|
| Aucun job | `[]` | `ok` |
| Job running | job{status=running} | `syncing` + `active_job_id` |
| Job queued | job{status=queued} | `syncing` |
| pendingCount > 0, pas de job actif | pendingCount=3 | `pending` |
| Dernier job failed | job{status=failed} | `error` + `last_sync_state` |
| Dernier job interrupted | job{status=interrupted} | `error` |
| Job succeeded | job{status=succeeded} | `ok` + `last_sync_at` |
| Job actif + pending | job running + pending=2 | `syncing` (priorité job actif) |
| Plusieurs jobs, plus récent = failed | 2 jobs terminal | `error` sur le plus récent |
| Job d'un autre joueur ignoré | job.PlayerSlug != slug | `ok` |

---

### `internal/platform/jobs/store.go` — méthode à ajouter

```go
// FindJobsForPlayer retourne tous les jobs (terminaux ou non) pour un joueur,
// triés du plus récent au plus ancien. Utilisé par WatcherHandler.
func (s *Store) FindJobsForPlayer(playerSlug string) []*AsyncJobStatus
```

Implémentation : RLock, itérer `s.jobs`, filtrer par `PlayerSlug`, trier par `StartedAt` desc, retourner copies.

---

### `internal/api/handlers/watcher_handler.go`

```go
package handlers

// WatcherHandler gère GET /players/{player_slug}/watcher/status.
//
// Logging structuré (slog) à chaque appel :
//   - request_id (depuis contexte middleware)
//   - player_slug
//   - derived_state
//   - active_job_id (si présent)
//   - last_sync_state (si présent)
//   - duration_ms
//
// Réponse 200 : WatcherStatus JSON
// Réponse 400 : player_slug vide
// Pas de 404 : un joueur sans jobs retourne state="ok" (joueur connu = dans db_profiles)
type WatcherHandler struct {
    jobStore        *jobs.Store
    watcherProvider domain.WatcherStateProvider
}

func NewWatcherHandler(jobStore *jobs.Store, provider domain.WatcherStateProvider) *WatcherHandler

// GetStatus : GET /players/{player_slug}/watcher/status
func (h *WatcherHandler) GetStatus(w http.ResponseWriter, r *http.Request)
```

**Logging slog — chaque appel :**
```go
slog.InfoContext(ctx, "watcher.status",
    "request_id",     middleware.RequestIDFromCtx(r.Context()),
    "player_slug",    playerSlug,
    "derived_state",  status.State,
    "active_job_id",  status.ActiveJobID,   // nil si absent
    "last_sync_state", status.LastSyncState, // nil si absent
    "duration_ms",    time.Since(start).Milliseconds(),
)
```

**Tests `watcher_handler_test.go` — cas table-driven :**

| Scénario | Setup | HTTP status | `state` attendu |
|----------|-------|-------------|-----------------|
| Joueur sans jobs | store vide | 200 | `"ok"` |
| Job running | store avec job running | 200 | `"syncing"` |
| Job failed | store avec job failed | 200 | `"error"` |
| player_slug vide | URL sans slug | 400 | — |
| provider pending=2 | custom provider | 200 | `"pending"` |

Pattern de test : `httptest.NewRecorder()` + `chi.NewRouter()` + store mocké via `jobs.NewStore("")` avec injection manuelle de jobs.

---

### `internal/api/server.go` — modification

Dans le bloc `r.Route("/players/{player_slug}", ...)`, ajouter après les routes existantes :

```go
// Sprint watcher : état de sync en temps réel
watcherH := handlers.NewWatcherHandler(jobStore, domain.NoopWatcherProvider)
r.Get("/watcher/status", watcherH.GetStatus)
```

> Quand `internal/watcher/provider.go` sera implémenté, remplacer `domain.NoopWatcherProvider`
> par l'instance réelle injectée depuis `cmd/server/main.go`. Le handler ne change pas.

---

## Frontend TypeScript / React

### Fichiers à créer / modifier

| Fichier | Action | Rôle |
|---------|--------|------|
| `lib/api/types.ts` | Modifier | + type `WatcherStatus` |
| `components/ui/sync-status-icon.tsx` | Créer | SVG + couleur + animation (pur, sans réseau) |
| `components/ui/sync-status-icon.test.tsx` | Créer | Tests Vitest : 4 états + tooltip + aria |
| `components/shell/WatcherStatusBadge.tsx` | Créer | Poll 30s + orchestration état |
| `components/shell/WatcherStatusBadge.test.tsx` | Créer | Tests MSW : poll OK, error, pending |
| `components/shell/NavL1.tsx` | Modifier | + `<WatcherStatusBadge>` après le select |
| `test/handlers.ts` | Modifier | + handler MSW `GET /players/:slug/watcher/status` |

---

### `lib/api/types.ts` — ajout

```ts
export type WatcherState = 'ok' | 'syncing' | 'pending' | 'error'

export interface WatcherStatus {
  player_slug: string
  state: WatcherState
  pending_match_count: number
  last_sync_at?: string        // ISO 8601
  last_sync_state?: string     // "succeeded" | "failed" | "interrupted" | ...
  active_job_id?: string
}
```

---

### `components/ui/sync-status-icon.tsx`

**Props :**
```ts
interface SyncStatusIconProps {
  state: WatcherState
  /** Taille en px (défaut 16) */
  size?: number
  /** Désactive le tooltip (utile si le parent gère un popover) */
  hideTooltip?: boolean
}
```

**Rendu :**
- `ok` → SVG check mark, `text-green-500`
- `syncing` → SVG flèches circulaires, `text-white`, `animate-spin`
- `pending` → SVG flèches circulaires, `text-orange-400`, `animate-spin`
- `error` → SVG flèches circulaires, `text-red-500`, `animate-spin`

SVG flèches circulaires : `<path>` arrow-path (Heroicons `arrow-path`, viewBox 24×24).
SVG check : `<path>` check (Heroicons `check`, viewBox 24×24).

**Tooltip (via `title` natif ou `<InfoTooltip>` existant) :**
```
ok      → "Données à jour"
syncing → "Synchronisation en cours…"
pending → "Nouveau match détecté — sync imminente"
error   → "Sync échouée — consulter les logs"
```

**Aria :**
```tsx
<svg aria-label={tooltipText} role="img" ...>
```

**Tests `sync-status-icon.test.tsx` :**

| Test | Vérification |
|------|-------------|
| Rendu sans crash pour chaque état | `container.firstChild` truthy × 4 |
| `ok` → classe `text-green-500` présente | `querySelector` ou `getByRole` |
| `syncing/pending/error` → classe `animate-spin` | className check |
| `pending` → couleur orange, pas blanc | className check |
| `error` → couleur rouge | className check |
| aria-label présent et non vide | `getByRole('img')` + `aria-label` |
| tooltip text correct par état | `title` attribute |
| `hideTooltip=true` → pas de title | attribute absent |

---

### `components/shell/WatcherStatusBadge.tsx`

**Responsabilité :** poll API + gestion des états réseau → délègue le rendu à `SyncStatusIcon`.

```ts
interface WatcherStatusBadgeProps {
  playerSlug: string
}
```

**Logique :**
```ts
const { data, isError } = useQuery({
  queryKey: ['watcher-status', playerSlug],
  queryFn: () => api.get<WatcherStatus>(`/players/${playerSlug}/watcher/status`),
  refetchInterval: 30_000,
  enabled: !!playerSlug,
  retry: 2,
})

// Pendant le premier fetch (data undefined) → ne rien rendre (invisible)
if (!data && !isError) return null

// Si erreur réseau (API injoignable) → forcer state="error"
const state: WatcherState = isError ? 'error' : data.state
```

**Tests `WatcherStatusBadge.test.tsx` avec MSW :**

| Test | Handler MSW | Vérification |
|------|-------------|-------------|
| État `ok` | retourne `{state:"ok"}` | `getByRole('img', {name: /à jour/i})` |
| État `syncing` | retourne `{state:"syncing"}` | aria-label contient "cours" |
| État `pending` | retourne `{state:"pending"}` | aria-label contient "détecté" |
| État `error` (API) | retourne `{state:"error"}` | aria-label contient "logs" |
| Erreur réseau | handler retourne 500 | state forcé à `error` |
| `playerSlug` vide | — | ne pas rendre (null) |
| Pas de rendu pendant loading | délai MSW | rien dans le DOM |

Utiliser `server.use(http.get(..., () => HttpResponse.json(...)))` dans chaque test pour override le handler par défaut.

---

### `components/shell/NavL1.tsx` — modification

Dans le bloc du sélecteur joueur, après le `<select>` (ou le `<span>` si un seul joueur) :

```tsx
{/* ── Watcher Status ──────────────────────────────────────────── */}
{playerSlug && (
  <WatcherStatusBadge playerSlug={playerSlug} />
)}
```

Positionné entre le sélecteur joueur et le lien Paramètres, avec `ml-1 shrink-0`.

---

### `test/handlers.ts` — ajout dans `handlers[]`

```ts
// Watcher status (état par défaut : ok)
http.get(p(`/players/${SLUG}/watcher/status`), () =>
  HttpResponse.json({
    player_slug: 'test-player',
    state: 'ok',
    pending_match_count: 0,
    last_sync_at: null,
    last_sync_state: null,
    active_job_id: null,
  })
),
```

---

## Ordre d'implémentation recommandé

```
1. domain/watcher.go + watcher_test.go          ← base, pas de dépendance externe
2. platform/jobs/store.go → FindJobsForPlayer    ← méthode simple, tests existants à étendre
3. handlers/watcher_handler.go + _test.go        ← dépend de 1 + 2
4. api/server.go → brancher la route             ← 1 ligne
5. lib/api/types.ts → WatcherStatus              ← pur TS, trivial
6. components/ui/sync-status-icon.tsx + test     ← pur UI, pas de réseau
7. components/shell/WatcherStatusBadge.tsx + test ← dépend de 5 + 6
8. NavL1.tsx → intégration                       ← dépend de 7
9. test/handlers.ts → handler MSW               ← dépend de 5, permet aux autres tests de passer
```

---

## Déploiement — intégration dans l'infrastructure existante

### Architecture actuelle

```
VPS /opt/levelup/
├── docker-compose.yml     ← 1 service "levelup" (API Go + assets React)
├── deploy.sh              ← appelé par GitHub Actions sur push main
└── data/                  ← volume persistant (DuckDB, logs, cache)
```

Le deploy : push main → GH Actions → SSH VPS → `deploy.sh` → `docker compose up --build`.

### Ce qui change avec le watcher

Le daemon watcher est un **second binaire Go** (`cmd/watcher/`) compilé dans la même image Docker.
Il est lancé comme un **second service Docker** dans `docker-compose.yml`, partageant le volume
`./data` avec `levelup` pour accéder à `db_profiles.json` et persister son état dans
`data/watcher/`.

### Modifications `Dockerfile`

Ajouter un second `go build` dans le stage `go-builder` :

```dockerfile
# Stage 2 — Build Go (ajout watcher)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -extldflags '-static'" \
    -o /build/levelup-server \
    ./cmd/server/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION}" \
    -o /build/levelup-watcher \
    ./cmd/watcher/
```

> Le watcher n'utilise pas CGo (pas de DuckDB direct) — `CGO_ENABLED=0` → binaire plus léger.

Copier le binaire watcher dans le stage runtime :

```dockerfile
COPY --from=go-builder /build/levelup-watcher /app/levelup-watcher
```

### Modifications `docker-compose.yml`

Ajouter le service `levelup-watcher` **après** le service `levelup` :

```yaml
  levelup-watcher:
    build: .
    command: ["/app/levelup-watcher"]
    depends_on:
      levelup:
        condition: service_healthy   # attend que l'API soit prête
    environment:
      - LEVELUP_ROOT=/app
      - LEVELUP_DATA=/app/data
      # Tokens Xbox XSTS — mêmes secrets que l'API si nécessaire
    env_file:
      - .env.local
    volumes:
      # Partage le même volume data que l'API (db_profiles.json + data/watcher/)
      - ./data:/app/data
      - ./db_profiles.json:/app/db_profiles.json:ro
      - ./app_settings.json:/app/app_settings.json:ro
    restart: unless-stopped
    # Pas de port exposé — le watcher ne sert pas de HTTP
    healthcheck:
      test: ["CMD", "/app/levelup-watcher", "-health-check"]
      interval: 30s
      timeout: 5s
      start_period: 30s    # l'auth XSTS peut prendre quelques secondes
      retries: 3
```

**Points clés :**
- `depends_on: levelup: condition: service_healthy` → le watcher démarre uniquement après que l'API passe son healthcheck. Évite les race conditions au boot.
- Volume `./data:/app/data` partagé en **lecture/écriture** pour que le watcher lise `db_profiles.json` et écrive dans `data/watcher/{gamertag}/state.json`.
- `db_profiles.json` monté en **read-only** côté watcher (il ne doit pas le modifier).
- Pas de port exposé — la communication watcher → API se fera via l'interface `WatcherStateProvider` (in-process à terme, ou HTTP localhost si processus séparés).

### Modifications `deploy.sh`

Le `deploy.sh` actuel a deux artefacts Streamlit à corriger **de toute façon** (hors scope watcher) :

```bash
# AVANT (Streamlit — à supprimer)
_wait_for_http "http://127.0.0.1:8501/_stcore/health" "levelup" 60
docker compose exec -T levelup python scripts/healthcheck_db.py --verbose

# APRÈS (Go natif)
_wait_for_http "http://127.0.0.1:8000/api/v1/health" "levelup" 60
```

Pour le watcher, pas de healthcheck HTTP — le container Docker suffit :

```bash
# Vérifier que le watcher est bien démarré après le deploy
echo "[deploy] Vérification watcher..."
if docker compose ps levelup-watcher | grep -q "healthy\|running"; then
    echo "[deploy] ✅ levelup-watcher actif"
else
    echo "[deploy] ⚠️  levelup-watcher non démarré — logs : docker compose logs levelup-watcher"
fi
```

### Modifications `deploy.yml` (GitHub Actions)

Aucune modification nécessaire — le workflow SSH appelle `deploy.sh` qui fait le
`docker compose up --build`. Le rebuild inclut automatiquement les deux binaires.

En revanche, il faudra ajouter le secret `XBOX_REFRESH_TOKEN` (token OAuth permanent obtenu
lors du premier setup via `cmd/auth-setup`) dans les secrets GitHub Actions **et** dans `.env.local`
sur le VPS. Ce secret permet au watcher de s'authentifier sans intervention manuelle au redémarrage.

### Variables d'environnement watcher (`.env.local`)

```bash
# Watcher Xbox RTA — tokens persistés sur disque (data/auth/tokens.json)
# Seul le refresh_token est nécessaire au démarrage si tokens.json existe déjà.
WATCHER_XBOX_CLIENT_ID=...
WATCHER_XBOX_CLIENT_SECRET=...
WATCHER_XBOX_REFRESH_TOKEN=...   # obtenu via cmd/auth-setup, permanent
WATCHER_TOKEN_STORE_PATH=data/auth/tokens.json
```

### Procédure de premier déploiement avec le watcher

```
1. Implémenter cmd/auth-setup (auth XSTS interactive)
2. Lancer cmd/auth-setup en local → génère data/auth/tokens.json
3. Copier data/auth/tokens.json sur le VPS dans /opt/levelup/data/auth/
4. Ajouter WATCHER_XBOX_REFRESH_TOKEN dans .env.local du VPS
5. Push main → deploy normal → docker compose up lance le watcher automatiquement
```

Les déploiements suivants sont transparents — `deploy.sh` redémarre le watcher avec
`docker compose up -d --build`, le refresh_token dans `.env.local` permet le re-auth automatique.

---

## Intégration binaire unique (Option B — app locale)

Le watcher tourne comme goroutine dans `cmd/server/main.go`, pas comme processus séparé.

```go
// cmd/server/main.go
func main() {
    cfg := config.Load()
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    // Démarrer le watcher en goroutine (partage le même process)
    watcherProvider := watcher.NewProvider()
    go watcher.Start(ctx, cfg, watcherProvider)

    // Injecter le provider réel dans le router
    router := api.NewRouter(cfg, bootRepo, bootSvc, watcherProvider)
    startHTTPServer(ctx, cfg, router)
}
```

`watcher.NewProvider()` retourne un objet qui implémente `domain.WatcherStateProvider` ET
expose une méthode `SetPendingCount(playerSlug string, n int)` appelée par la FSM interne.
Le handler HTTP reçoit cette interface — il ne sait pas si c'est le noop ou le vrai provider.

**Avantage** : zéro IPC, zéro HTTP entre watcher et API. Le `PendingCount` est une lecture
mémoire directe. Fonctionne identiquement en mode local (binaire unique) et Docker
(si on choisit de fusionner les deux services à terme).

---

## Contraintes et règles à respecter

- **Séparation stricte** : `SyncStatusIcon` ne fait aucun appel réseau — état reçu en prop.
  `WatcherStatusBadge` ne contient aucune logique de rendu SVG — délègue à `SyncStatusIcon`.
- **Logging Go** : chaque appel à `GetStatus` loggué via `slog.InfoContext` avec les champs listés ci-dessus. Les erreurs (ex: player_slug invalide) loggées avec `slog.WarnContext`.
- **Pas de state global** : le badge gère son propre état via TanStack Query — pas d'ajout dans `appShellStore`.
- **Extensibilité** : `WatcherStateProvider` est une interface — `NoopWatcherProvider` en V1, provider réel injecté depuis `main.go` quand le watcher est implémenté. Le handler ne change jamais.
- **Pas de 404** : un joueur sans jobs retourne `state="ok"` — le badge est toujours affiché si `playerSlug` est non vide.
- **Animation CSS** : utiliser `animate-spin` de Tailwind (déjà présent dans le projet). Pas de JS pour l'animation.
- **Steam** : le badge ne distingue pas si l'état vient de RTA ou du fallback Steam — l'état dérivé (`syncing`, `pending`, etc.) est identique dans les deux cas.

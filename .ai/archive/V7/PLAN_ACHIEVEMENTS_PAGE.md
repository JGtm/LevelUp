# Plan — Page Xbox Achievements

> Créé : 2026-05-04
> Branche Git cible : `feat/achievements-page`
> Effort total estimé : 4-5 jours

---

## Contexte et objectif

Le backend achievements est **entièrement implémenté** :
- `internal/sync/achievements.go` — fetch bilingue EN+FR, upsert deux tables
- `internal/sync/xbox_client.go` — client HTTP Xbox Achievements v2 avec pagination
- `internal/sync/engine.go` → `runAchievementsSync()` — intégré dans le post-sync pipeline
- `internal/assets/kinds.go` → `KindAchievementImage` — pré-warming images
- Tables DuckDB :
  - `metadata.xbox_achievement_definitions` (référentiel bilingue, statique)
  - `stats/{gamertag}/player_achievements` (progression et unlock, par joueur)
- Tests unitaires + intégration déjà présents

**Ce qui manque entièrement :**
1. Backfill initial — les joueurs existants n'ont pas encore leurs achievements en DB
2. Couche lecture (repo, service, port, domain)
3. Handler HTTP + route + capability
4. Frontend (route, hook, composants)

**Critère de succès :** La page `/players/{gamertag}/achievements` affiche la liste complète
des achievements avec statut unlock, gamerscore et image. Les données de JGtm sont présentes
après le backfill CLI.

---

## Stratégie de séquencement

```
Commit 1 — CLI backfill         → exécuter immédiatement pour peupler les données
Commit 2 — Repo + ports         → couche lecture, testée en isolation
Commit 3 — Domain + Service     → merge cross-DB, stats calculées
Commit 4 — Capability + Handler → exposition HTTP, garde multi-titres
Commit 5 — Frontend             → page complète
Commit 6 — Tests intégration + thought_log
```

Chaque commit laisse le code dans un état cohérent (compile + tests verts).

---

## Commit 1 — CLI `sync-achievements` (backfill initial) `[rapide, 2-3 h]`

### Pourquoi en premier

Les achievements sont stables (définis à la sortie du jeu + DLC). Peupler les deux tables en
amont permet de tester la couche lecture dès le commit 2 sur des données réelles.

### Fichier

`apps/go-api/cmd/levelup/cmd_sync_achievements.go` (nouveau)

### Comportement

```
levelup sync-achievements --gamertag JGtm
levelup sync-achievements --all
levelup sync-achievements --all --dry-run   # log sans écriture
```

**Flags :**
- `--gamertag` : sync un joueur précis
- `--all` : itère sur tous les joueurs de `db_profiles.json`
- `--dry-run` : affiche le nombre d'achievements trouvés sans upsert

**Logique interne :**
1. Charger les profils joueurs depuis config (`loadPlayerSummary`)
2. Pour chaque joueur : OAuth refresh → XSTS token → client Xbox
3. Ouvrir `metadataDB` (RW) + `playerDB` (RW) via `PathResolver.MetadataDBPath(slug)` /
   `PathResolver.PlayerDBPath(gamertag)` — jamais de `filepath.Join` direct
4. Appeler `sync.SyncAchievements(ctx, client, resolver, metadataDB, playerDB, xuid)`
5. Log par joueur : gamertag, nb achievements trouvés, nb unlocked, durée

**Sérialisation DB — prérequis admin :**
Le CLI backfill est une **opération admin one-shot**. Il n'acquiert pas de `dblease` (le
serveur HTTP détient les leases tant qu'un sync est actif). Pour éviter toute collision :
- Lancer `--dry-run` d'abord pour vérifier la connectivité sans écriture
- Lancer le backfill uniquement quand aucun sync n'est actif (vérifier sur `/debug/vars` ou
  attendre la fin d'un cycle)
- Alternative propre si un sync concurrent est possible : utiliser `dblease.AcquireWriter`
  avec timeout court (ex: `MetadataLeaseTimeout=10s`) et retry si `ErrDBLocked`

**Logging :**
```go
slog.InfoContext(ctx, "achievements sync start", "gamertag", gt, "all", all)
slog.InfoContext(ctx, "achievements sync done",
    "gamertag", gt, "total", len(achievements), "unlocked", unlocked,
    "duration_ms", elapsed.Milliseconds())
slog.WarnContext(ctx, "achievements sync skipped",
    "gamertag", gt, "reason", "no_provider")
slog.ErrorContext(ctx, "achievements sync failed",
    "gamertag", gt, "err", err)
```

**Sérialisaion DB :** utiliser les context managers `duckdb_read_write()` existants —
le CLI s'exécute hors sync engine donc pas de conflit de lease.

**Done :** `levelup sync-achievements --gamertag JGtm` peuple `player_achievements` et
`xbox_achievement_definitions`. Log visible en console.

---

## Commit 2 — Repo lecture + ports `[moyen, 3-4 h]`

### Fichiers

**`apps/go-api/internal/port/achievements.go`** (nouveau)
```go
// AchievementsRepository lit la progression joueur (stats.duckdb).
type AchievementsRepository interface {
    GetPlayerAchievements(ctx context.Context) ([]domain.PlayerAchievementRow, error)
}

// MetadataAchievementsRepository lit le référentiel bilingue (metadata.duckdb).
// Interface séparée de MetadataRepository (interface segregation) pour que le service
// n'ait pas à dépendre des méthodes saisons/waypoint non liées aux achievements.
type MetadataAchievementsRepository interface {
    GetAchievementDefinitions(ctx context.Context) ([]domain.AchievementDefinitionRow, error)
}

type AchievementsService interface {
    GetAchievementsPage(ctx context.Context) (domain.AchievementsPageResponse, error)
}
```
`MetadataRepo` existant implémente `MetadataAchievementsRepository` via
`metadata_achievements_repo.go` (aucun nouveau type repo à créer).

**`apps/go-api/internal/domain/achievements.go`** (nouveau — types bruts repo)
```go
type PlayerAchievementRow struct {
    AchievementID   string
    Unlocked        bool
    UnlockedAt      *time.Time
    CurrentProgress *int
    TargetProgress  *int
}

type AchievementDefinitionRow struct {
    AchievementID  string
    NameEn         string
    NameFr         string
    DescriptionEn  string
    DescriptionFr  string
    LockedDescEn   string
    LockedDescFr   string
    Gamerscore     int
    ImageURL       string
    IsSecret       bool
    RarityCategory string
    RarityPercent  float64
}
```

**`apps/go-api/internal/platform/duckdb/achievements_repo.go`** (nouveau)
- `AchievementsRepo` lit `player_achievements` depuis `stats.duckdb`
- Méthode : `GetPlayerAchievements(ctx) ([]domain.PlayerAchievementRow, error)`
- Scan minimal, pas de tri (la couche service trie)

**`apps/go-api/internal/platform/duckdb/metadata_achievements_repo.go`** (nouveau fichier,
  **étend le type `MetadataRepo` existant** — même pattern que `medal_cache_repo.go`,
  `map_cache_repo.go` : nouveau fichier, récepteur `*MetadataRepo`, pas de nouveau type)
- Méthode : `(r *MetadataRepo) GetAchievementDefinitions(ctx) ([]domain.AchievementDefinitionRow, error)`
- Retourne toutes les définitions depuis `xbox_achievement_definitions` (quelques centaines au max)
- Constructeurs déjà existants : `NewMetadataRepo(pdb)` + `NewMetadataRepoFromDB(meta)` — rien à ajouter

**Logging :**
```go
slog.DebugContext(ctx, "achievements repo query",
    "table", "player_achievements", "player", playerID)
slog.ErrorContext(ctx, "achievements repo scan failed",
    "err", err, "table", "player_achievements")
```

**Tests :** `internal/platform/duckdb/achievements_repo_test.go`
- DuckDB `:memory:` + migration EnsureTable
- `GetPlayerAchievements` : 0 rows (joueur neuf), N rows peuplés, filtre par player
- `GetAchievementDefinitions` : 0 définitions, N définitions, vérifier scan bilingue complet

**Done :** deux repos compilent, tests verts sur DuckDB `:memory:`.

---

## Commit 3 — Domain response + Service `[moyen, 3-4 h]`

### Domain response (dans `domain/achievements.go`)

```go
type AchievementsPageResponse struct {
    Summary      AchievementsSummary `json:"summary"`
    Achievements []AchievementEntry  `json:"achievements"`
}

type AchievementsSummary struct {
    TotalCount       int     `json:"total_count"`
    UnlockedCount    int     `json:"unlocked_count"`
    TotalGamerscore  int     `json:"total_gamerscore"`
    EarnedGamerscore int     `json:"earned_gamerscore"`
    CompletionPct    float64 `json:"completion_pct"`
}

type AchievementEntry struct {
    AchievementID   string     `json:"achievement_id"`
    NameEn          string     `json:"name_en"`
    NameFr          string     `json:"name_fr"`
    DescriptionEn   string     `json:"description_en"`
    DescriptionFr   string     `json:"description_fr"`
    LockedDescEn    string     `json:"locked_desc_en,omitempty"`
    LockedDescFr    string     `json:"locked_desc_fr,omitempty"`
    Gamerscore      int        `json:"gamerscore"`
    ImageURL        string     `json:"image_url,omitempty"`
    IsSecret        bool       `json:"is_secret"`
    RarityCategory  string     `json:"rarity_category,omitempty"`
    RarityPercent   float64    `json:"rarity_percent,omitempty"`
    Unlocked        bool       `json:"unlocked"`
    UnlockedAt      *time.Time `json:"unlocked_at,omitempty"`
    CurrentProgress *int       `json:"current_progress,omitempty"`
    TargetProgress  *int       `json:"target_progress,omitempty"`
}
```

**Note sur les noms bilingues :** les deux langues sont servies dans la réponse. Le frontend
choisit `name_fr` ou `name_en` selon la locale de l'app (même pattern que les medals dans
d'autres features). Pas de logique Accept-Language côté serveur.

### Service

**`apps/go-api/internal/service/achievements_service.go`** (nouveau)

```go
type AchievementsService struct {
    repo     port.AchievementsRepository
    metaRepo port.MetadataAchievementsRepository
    titleSlug string
}

func NewAchievementsService(
    repo port.AchievementsRepository,
    metaRepo port.MetadataAchievementsRepository,
) *AchievementsService

func (s *AchievementsService) WithTitleSlug(slug string) *AchievementsService

func (s *AchievementsService) GetAchievementsPage(ctx context.Context) (domain.AchievementsPageResponse, error)
```

**Logique `GetAchievementsPage` :**
1. `playerRows, err := s.repo.GetPlayerAchievements(ctx)` → map par achievement_id
2. `defs, err := s.metaRepo.GetAchievementDefinitions(ctx)` → slice de définitions
3. Merge Go-side : pour chaque définition, lookup dans la map player
4. Calcul summary : total_count, unlocked_count, earned/total gamerscore, completion_pct
5. Tri final : unlocked en premier (UnlockedAt DESC), puis locked par gamerscore DESC
6. Retourner `AchievementsPageResponse{Summary, Achievements}`

**Cas edge :**
- Joueur sans données player_achievements (0 rows) → tous `unlocked=false`, `current_progress=nil`
- Définitions vides (backfill pas encore fait) → réponse vide valide (pas d'erreur)

**Logging :**
```go
slog.DebugContext(ctx, "achievements service merge",
    "definitions", len(defs), "player_rows", len(playerRows),
    "unlocked", summary.UnlockedCount)
slog.ErrorContext(ctx, "achievements service failed",
    "err", err, "player", s.titleSlug)
```

**Tests :** `internal/service/achievements_service_test.go`
- Mock `port.AchievementsRepository` + mock `port.MetadataAchievementsRepository`
- Cas nominal : 10 définitions + 3 unlocked → summary correct, tri correct
- Cas joueur neuf : 0 player rows → tous locked, gamerscore earned = 0
- Cas définitions vides : réponse vide, pas d'erreur
- Cas achievement_id orphelin (player row sans définition) → ignoré silencieusement

**Done :** service compile, tests verts avec mocks.

---

## Commit 4 — Capability + Handler + Route `[rapide, 2-3 h]`

### Capability

**`apps/go-api/internal/games/capabilities.go`** (ajouter constante) :
```go
CapAchievements CapabilityKey = "achievements"
```

**`apps/go-api/internal/games/halo_infinite/adapter_data.go`** (ajouter dans `Capabilities()`) :
```go
games.CapAchievements: games.CapSupported,
```

Pour les autres titres qui n'auraient pas d'achievements Xbox : la capability est absente
de leur map → `ErrCapabilityNotSupported` retourné par le middleware existant.

### Handler

**`apps/go-api/internal/api/handlers/achievements.go`** (nouveau)

```go
type AchievementsHandler struct {
    newSvc ServiceFactory[port.AchievementsService]
}

func NewAchievementsHandler(newSvc ServiceFactory[port.AchievementsService]) *AchievementsHandler

func (h *AchievementsHandler) GetAchievementsPage(w http.ResponseWriter, r *http.Request) {
    slug := chi.URLParam(r, "player_slug")
    svc, err := h.newSvc(r.Context(), slug)
    if err != nil {
        writeError(w, http.StatusNotFound, "player_not_found", err.Error())
        return
    }
    resp, err := svc.GetAchievementsPage(r.Context())
    if err != nil {
        writeError(w, http.StatusInternalServerError, "achievements_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, resp)
}
```

**Logging dans le handler :**
```go
slog.ErrorContext(r.Context(), "achievements handler failed",
    "err", err, "player", slug, "endpoint", "GET /pages/achievements")
```

### Route

**`apps/go-api/internal/api/server.go`** (ajouter dans le groupe player routes) :
```go
achievementsH := handlers.NewAchievementsHandler(reg.Achievements)
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireCapability(titleRegistry, games.CapAchievements))
    r.Get("/pages/achievements", achievementsH.GetAchievementsPage)
})
```

**Enregistrement ServiceFactory** dans `internal/api/server.go` (point d'assemblage central,
pattern identique aux autres services) : wirer `AchievementsRepo` (stats.duckdb) +
`MetadataRepo` (metadata.duckdb, constructeur existant `NewMetadataRepoFromDB`) +
`NewAchievementsService`.

**Tests :** `internal/api/handlers/achievements_test.go`
- `httptest.NewRecorder` + mock `port.AchievementsService`
- GET succès → 200 + JSON valide
- GET player inconnu → 404 + code `player_not_found`
- GET erreur service → 500 + code `achievements_error`
- GET sur titre sans capability → 405/501 (middleware existant)

**Done :** `curl /api/v1/players/JGtm/pages/achievements` retourne JSON avec
les achievements en DB.

---

## Commit 5 — Frontend `[lourd, 1-2 j]`

### Query key

**`apps/web/src/lib/query/keys.ts`** (ajouter) :
```typescript
achievements: (playerSlug: string) => ['achievements', playerSlug] as const,
```

### Types TS

**`apps/web/src/lib/api/types.ts`** (ajouter) :
```typescript
export interface AchievementEntry {
  achievement_id: string
  name_en: string
  name_fr: string
  description_en: string
  description_fr: string
  locked_desc_en?: string
  locked_desc_fr?: string
  gamerscore: number
  image_url?: string
  is_secret: boolean
  rarity_category?: string
  rarity_percent?: number
  unlocked: boolean
  unlocked_at?: string  // ISO 8601
  current_progress?: number
  target_progress?: number
}

export interface AchievementsSummary {
  total_count: number
  unlocked_count: number
  total_gamerscore: number
  earned_gamerscore: number
  completion_pct: number
}

export interface AchievementsPageResponse {
  summary: AchievementsSummary
  achievements: AchievementEntry[]
}
```

### Hook query

**`apps/web/src/features/achievements/queries.ts`** (nouveau) :
```typescript
export function useAchievementsPage(playerSlug: string) {
  return useQuery<AchievementsPageResponse>({
    queryKey: queryKeys.achievements(playerSlug),
    queryFn: () =>
      api.get<AchievementsPageResponse>(
        `/players/${playerSlug}/pages/achievements`
      ),
    enabled: !!playerSlug,
    staleTime: 10 * 60 * 1000,  // 10 min — données stables
  })
}
```

### Composants

**`apps/web/src/features/achievements/`** :

- `AchievementsPage.tsx` — page principale (skeleton + summary + grid)
- `AchievementsSummaryBar.tsx` — barre de progression gamerscore + compteur unlocked/total
- `AchievementCard.tsx` — carte individuelle :
  - Image via `image_url` (placeholder si absent)
  - Nom localisé (`locale === 'fr' ? name_fr : name_en`)
  - Gamerscore badge
  - Rarity % (si disponible)
  - Etat lock/unlock visuellement distinct (opacité, icône)
  - `unlocked_at` formaté si présent
- `AchievementsFilter.tsx` — filtres : All / Unlocked / Locked / par rarity_category

**Règles couleur :** aucun hex ni classe Tailwind de couleur dans les composants.
État "locked" → token sémantique `muted` ou `disabled`. Rarity → tokens prédéfinis si
existants, sinon commentaire justificatif.

### Route

**`apps/web/src/routes/players/$playerSlug/achievements.tsx`** (nouveau) :
```typescript
export const Route = createFileRoute('/players/$playerSlug/achievements')({
  loader: ({ params, context }) => {
    void context.queryClient.prefetchQuery({
      queryKey: queryKeys.achievements(params.playerSlug),
      queryFn: () =>
        api.get<AchievementsPageResponse>(
          `/players/${params.playerSlug}/pages/achievements`
        ),
    })
  },
  component: AchievementsPage,
})
```

### i18n

Ajouter dans le manifeste i18n (EN + FR obligatoires) :
```
achievements.page_title       → "Achievements" / "Succès"
achievements.summary.unlocked → "{n} / {total} débloqués" / "{n} / {total} unlocked"
achievements.summary.gamerscore → "{earned}G / {total}G"
achievements.filter.all       → "Tous" / "All"
achievements.filter.unlocked  → "Débloqués" / "Unlocked"
achievements.filter.locked    → "Verrouillés" / "Locked"
achievements.secret           → "Secret" / "Secret"
achievements.unlocked_at      → "Débloqué le {date}" / "Unlocked on {date}"
achievements.progress         → "{current} / {target}" / "{current} / {target}"
achievements.empty            → "Aucun succès trouvé" / "No achievements found"
```

**Navigation :** ajouter un lien vers `/achievements` dans la sidebar/nav joueur (même
niveau que Career, Palmares, etc.).

**Tests front :**
```bash
cd apps/web && npm run typecheck && npm run lint && npm run test
```

**`src/features/achievements/useAchievementsPage.test.ts`** (nouveau) :
- Mock `fetch` avec une réponse `AchievementsPageResponse` factice
- Vérifier que la query key `queryKeys.achievements(slug)` est utilisée
- Vérifier que le type de retour est bien `AchievementsPageResponse` (no `any`)
- Vérifier `staleTime = 10 min` (données stables)

- `AchievementCard` : typecheck correct sur `AchievementEntry`

**Done :** page visible sur `/players/JGtm/achievements` avec données réelles post-backfill.

---

## Commit 6 — Tests intégration + thought_log `[moyen, 2-3 h]`

### Tests intégration Go (build tag `integration`)

**`internal/service/achievements_integration_test.go`** :
- DuckDB `:memory:` + tables `player_achievements` + `xbox_achievement_definitions`
- Test merge cross-repo : 20 définitions + 5 player rows → summary.UnlockedCount = 5,
  earned_gamerscore = sum des 5, tri correct (unlocked first)
- Test joueur neuf (0 player rows) : CompletionPct = 0, tous unlocked = false
- Test définitions vides : réponse `{summary: zeros, achievements: []}`
- Test achievement_id orphelin : player row sans définition → ignoré proprement

### Tests end-to-end non-régression

Suite à valider manuellement :
- [ ] `levelup sync-achievements --gamertag JGtm` → log success, données en DB
- [ ] `GET /api/v1/players/JGtm/pages/achievements` → 200 avec achievements et summary corrects
- [ ] Page `/players/JGtm/achievements` s'affiche sans erreur console
- [ ] Filtres All/Unlocked/Locked fonctionnels
- [ ] Sync delta classique inchangé (non-régression `runAchievementsSync` toujours appelé)

### Thought log

Ajouter entrée dans `.ai/thought_log.md` :
```
[2026-05-04] feat/achievements-page — Complété
Décision : deux repos séparés (AchievementsRepo + MetadataAchievementsRepo) + merge Go-side.
Évite ATTACH cross-DB dans le repo, plus testable avec mocks.
Résultats : backfill CLI fonctionnel, page frontend complète.
Suite : surveiller fraîcheur des données (sync pipeline déjà intégré).
```

---

## Conformité aux règles projet

| Règle | Application dans ce plan |
|---|---|
| Couches Go | Acquisition lease dans `service/`, repos CRUD purs, handlers sans logique métier |
| `PathResolver` | Pas de `filepath.Join` direct — `PathResolver.PlayerDBPath(gamertag)` / `PathResolver.MetadataDBPath(slug)` via `titlePkg.NewPathResolver(repoRoot)` |
| Capabilities | `CapAchievements` ajouté dans `games.CapabilityMap`, middleware `RequireCapability` sur la route |
| `ErrCapabilityNotSupported` | Retourné automatiquement par le middleware pour titres sans la capability |
| Multi-titres | La feature est title-agnostic côté service ; seul `adapter_data.go` de HI déclare la capability |
| Logging | Snippets explicites par commit, clés standards `err`, `player`, `titleSlug` |
| Tests par couche | repo (DuckDB :memory:) + service (mocks) + handler (httptest) + intégration |
| `thought_log.md` | Mentionné dans commit 6 (obligatoire avant de rendre la main) |
| `docs/FR/` | Pas de nouveau doc architecture dans ce plan → pas de sync FR nécessaire |
| Taille fichiers ≤ 500 L / fonctions ≤ 80 L | Vérifier à chaque commit ; `GetAchievementsPage` doit rester < 80 L |
| Couleurs frontend | Aucun hex ni Tailwind de couleur dans `features/achievements/` |
| i18n | Strings EN + FR obligatoires dans les manifests |
| Query keys | Entrée `achievements` ajoutée dans `keys.ts` |
| `routeTree.gen.ts` | Non édité manuellement — généré automatiquement |
| Branche Git | 1 branche `feat/achievements-page`, 6 commits séquentiels |

---

## Blockers identifiés

| Blocker | Probabilité | Workaround |
|---|---|---|
| Token OAuth expiré pour JGtm au moment du backfill | Moyenne | Le CLI utilise le refresh_token comme le sync normal — si le token est expiré depuis trop longtemps, lancer d'abord `levelup sync-delta --gamertag JGtm` pour le rafraîchir |
| Xbox Achievements API rate-limit sur `--all` | Faible | L'API v2 Xbox est généreuse ; ajouter `time.Sleep(500ms)` entre joueurs si nécessaire |
| Définitions vides au moment du dev frontend | Nul | Commit 1 doit être exécuté avant le commit 5 pour avoir des données réelles |

---

## Done definition globale

- [ ] `levelup sync-achievements --all` : peuple toutes les tables pour tous les joueurs
- [ ] `GET /api/v1/players/{slug}/pages/achievements` : 200 + JSON valide
- [ ] Page `/players/{slug}/achievements` : affichage complet sans erreur console
- [ ] `go test ./...` verts sur la branche
- [ ] `npm run typecheck && npm run lint` verts
- [ ] Aucun fichier > 500 L, aucune fonction > 80 L dans les fichiers créés
- [ ] `thought_log.md` mis à jour
- [ ] Non-régression sync delta : `runAchievementsSync` toujours appelé en post-sync

---

## Références

- `internal/sync/achievements.go` — logique sync existante (ne pas modifier)
- `internal/sync/xbox_client.go` — client Xbox API existant
- `internal/sync/engine.go` → `runAchievementsSync()` — integration point existant
- `internal/migration/steps_metadata.go` → `add_xbox_achievement_definitions`
- `internal/migration/steps_player.go` → `add_player_achievements`
- `internal/assets/kinds.go` → `KindAchievementImage`
- `internal/api/handlers/career.go` — pattern handler de référence
- `internal/service/career_service.go` — pattern service de référence
- `apps/web/src/lib/query/keys.ts` — query keys existantes
- `apps/go-api/cmd/levelup/cmd_sync.go` — pattern CLI de référence

# Plan — Asset Drawer (Référentiel Visuel Maps & Armes)

**Date** : 2026-04-28  
**Branche Git** : Courante  
**Statut** : En attente validation (3 points ouverts)

---

## Objectif & critère de succès

Un panneau latéral fixe accessible depuis toutes les pages : l'utilisateur tape "breaker" ou "skewer", voit la miniature en < 200ms, referme sans perturber la page derrière. Remplace définitivement l'ancien hover-tableau Python.

---

## Principe architectural — AssetResolver

Le frontend ne doit jamais construire une URL d'asset lui-même. Le backend expose une interface `AssetResolver` qui résout `(titleID, type, id)` → URL opaque. Les callers (handlers, repo) injectent l'interface et ignorent le backend de stockage (local, CDN, S3...).

```go
// internal/port/asset_resolver.go
type AssetResolver interface {
    ResolveMapURL(ctx context.Context, titleID, mapID string) (string, error)
    ResolveWeaponURL(ctx context.Context, titleID, weaponID string) (string, error)
}
```

Implémentation locale dans `internal/platform/local/asset_resolver.go` — construit le chemin `/static/{titleID}/maps/{filename}`. Le frontend consomme uniquement l'URL opaque retournée par l'API.

Ce pattern résout B3 + B4 d'un coup : plus de `filepath.Join` direct dans les handlers, plus de question sur ce que couvre `PathResolver`.

---

## Blockers à vérifier AVANT de coder

### B1 — Correspondance filename ↔ map_id [CRITIQUE]
Les fichiers sont nommés par nom affiché (`Aquarius.png`, `Breaker.png`) pas par `map_id` (GUID).

**Vérif** :
```sql
SELECT map_id, local_path FROM map_images_registry LIMIT 20;
```
Si `local_path` contient bien le chemin vers `/static/maps/`, on est bons. L'`AssetResolver` local utilisera ce `local_path` comme source de vérité.

### B2 — Gap images armes [MODÉRÉ]
28 images dans `/static/weapons-assets/` vs ~50+ `weapon_id` distincts dans `match_participants`.

**Workaround** : `ResolveWeaponURL` retourne `("", ErrAssetNotFound)` — le frontend affiche un placeholder SVG silhouette. Ne bloque pas la feature.

### B3 — Multi-title : `/static/` non organisé par titre [RÉSOLU par AssetResolver]
L'`AssetResolver` local encapsule la logique de chemin. Réorganiser vers `/static/halo_infinite/maps/` + `/static/halo_infinite/weapons/` sans toucher aux handlers.

### B4 — PathResolver pour `/static/` [RÉSOLU par AssetResolver]
Plus besoin de savoir si `PathResolver` couvre `/static/`. L'`AssetResolver` est la seule couche qui connaît la structure de fichiers.

---

## Phase 0 — Audit & réorg assets (0,5 j)

### Phase 0.5 — AssetResolver interface (~2h)

**0.5.1** — Interface port :
Fichier : `apps/go-api/internal/port/asset_resolver.go` (nouveau)

```go
type AssetResolver interface {
    ResolveMapURL(ctx context.Context, titleID, mapID string) (string, error)
    ResolveWeaponURL(ctx context.Context, titleID, weaponID string) (string, error)
}

var ErrAssetNotFound = errors.New("asset not found")
```

**0.5.2** — Implémentation locale :
Fichier : `apps/go-api/internal/platform/local/asset_resolver.go` (nouveau)

```go
type LocalAssetResolver struct {
    pathResolver PathResolver   // G4 — pas de staticRoot string hardcodé
    db           *MetadataRepo  // pour lookup map_id → local_path
}

func (r *LocalAssetResolver) ResolveMapURL(ctx context.Context, titleID, mapID string) (string, error)
func (r *LocalAssetResolver) ResolveWeaponURL(ctx context.Context, titleID, weaponID string) (string, error)
```

`ResolveMapURL` construit le chemin via `PathResolver` (même contrat que le reste de l'archi — aucun chemin absolu baked-in). Retourne l'URL de service : `/api/v1/assets/{titleID}/maps/{mapID}/image`.

**0.5.3** — Refactor `handlers/assets.go` existant :
Remplacer le `filepath.Join` direct par injection de `port.AssetResolver`. Le handler devient :
```go
func (h *AssetHandler) GetMapImage(w http.ResponseWriter, r *http.Request) {
    url, err := h.resolver.ResolveMapURL(r.Context(), titleID, mapID)
    // ...
}
```

**0.5.4** — Tests :
- `LocalAssetResolver` avec DB `:memory:` + fixture `map_images_registry`
- Cas : map connue → URL correcte, map inconnue → `ErrAssetNotFound`

**Critère** : `GetMapImage` passe toujours, `LocalAssetResolver` testé unitairement. Entrée thought_log.md ajoutée.

---

**0.1** — Audit SQL :
```sql
-- map_images_registry couvre bien les 102 images ?
SELECT COUNT(*), title_id FROM map_images_registry GROUP BY title_id;
SELECT map_id, local_path FROM map_images_registry LIMIT 10;

-- weapon_id sans label
SELECT DISTINCT weapon_id FROM weapon_labels ORDER BY weapon_id;
```

**0.2** — Réorganiser `/static/` :
```
/static/halo_infinite/maps/      ← depuis /static/maps/
/static/halo_infinite/weapons/   ← depuis /static/weapons-assets/
```
Mettre à jour les chemins dans `apps/go-api/internal/platform/duckdb/metadata_repo_assets.go` et `apps/go-api/internal/api/handlers/assets.go`.

**Critère de complétion** : `GET /api/v1/assets/maps/halo_infinite/{map_id}/image` répond toujours 200.

---

## Phase 1 — Backend Go : endpoint "list assets" (1 j)

### 1.1 — Type canonique
Fichier : `apps/go-api/internal/games/canonical/assets.go` (nouveau)

```go
type AssetMeta struct {
    ID       string `json:"id"`
    NameEN   string `json:"name_en"`
    NameFR   string `json:"name_fr"`
    ImageURL string `json:"image_url"` // URL relative /api/v1/assets/...
}
```

Pas de stats (match_count, win_rate) — c'est un référentiel, pas un tableau de bord. Feature V2.

### 1.2 — Repository
Extension de `apps/go-api/internal/platform/duckdb/metadata_repo_assets.go` :

```go
func (r *MetadataRepo) ListMapsByTitle(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)
func (r *MetadataRepo) ListWeaponsByTitle(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)
```

SQL maps (paramétré) :
```sql
SELECT mir.map_id, at_en.name AS name_en, at_fr.name AS name_fr
FROM map_images_registry mir
JOIN asset_translations at_en
    ON at_en.asset_id = mir.map_id AND at_en.asset_type = 'map' AND at_en.lang = 'en'
LEFT JOIN asset_translations at_fr
    ON at_fr.asset_id = mir.map_id AND at_fr.asset_type = 'map' AND at_fr.lang = 'fr'
WHERE mir.title_id = $1
  AND ($2 = '' OR lower(at_en.name) LIKE lower('%' || $2 || '%'))
ORDER BY at_en.name
```

SQL weapons :
```sql
SELECT weapon_id::VARCHAR AS id, name_en, name_fr
FROM weapon_labels
WHERE ($1 = '' OR lower(name_en) LIKE lower('%' || $1 || '%'))
ORDER BY name_en
```

### 1.3 — Service (G1)
Fichier : `apps/go-api/internal/service/asset_service.go` (nouveau)

```go
type AssetService struct {
    repo     port.AssetMetaRepository
    resolver port.AssetResolver
}

func (s *AssetService) ListMaps(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)
func (s *AssetService) ListWeapons(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)
```

Le service appelle le repo, puis enrichit chaque item avec `resolver.ResolveMapURL(ctx, titleID, id)` pour construire l'`image_url`. Aucun SQL ici.

Interface repo à ajouter dans `internal/port/` :
```go
type AssetMetaRepository interface {
    ListMapsByTitle(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)
    ListWeaponsByTitle(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error)
}
```

### 1.4 — Handler (G2, G3)
Fichier : `apps/go-api/internal/api/handlers/assets_metadata.go` (nouveau)

```go
// GET /api/v1/assets/{title_id}/maps?q=
func (h *AssetMetadataHandler) ListMaps(w http.ResponseWriter, r *http.Request)

// GET /api/v1/assets/{title_id}/weapons?q=
func (h *AssetMetadataHandler) ListWeapons(w http.ResponseWriter, r *http.Request)
```

Aucune logique métier dans le handler. Gate `HasCapability("asset_images")` en entrée :
```go
if !h.caps.Has("asset_images") {
    writeError(w, http.StatusNotFound, ErrCapabilityNotSupported)
    return
}
```

Logging :
```go
slog.DebugContext(ctx, "list maps", "title", titleID, "q", q, "n", len(results))
slog.ErrorContext(ctx, "list maps failed", "err", err, "title", titleID)
```

Capability à déclarer dans `config/titles/halo_infinite/capabilities.toml` (ou équivalent) :
```toml
asset_images = true
```

### 1.5 — Routes
Dans `apps/go-api/internal/api/server.go` :
```go
r.Get("/assets/{title_id}/maps", assetMetaHandler.ListMaps)
r.Get("/assets/{title_id}/weapons", assetMetaHandler.ListWeapons)
```

### 1.6 — Tests (G2)
**Repo** — `internal/platform/duckdb/metadata_repo_assets_test.go` :
- Fixtures : 3 maps + 2 armes en DuckDB `:memory:`
- Cas : list sans filtre → tout, search "aqu" → filtre, title_id inconnu → slice vide

**Service** — `internal/service/asset_service_test.go` :
- Mock `port.AssetMetaRepository` + mock `port.AssetResolver`
- Cas : `ListMaps` enrichit bien chaque item avec `image_url`

**Handler** — `internal/api/handlers/assets_metadata_handler_test.go` :
- `httptest.NewRecorder` pour `GET /assets/halo_infinite/maps`
- Cas : 200 avec résultats, 200 liste vide (search sans match), 404 si capability absente

**Critère** : `curl /api/v1/assets/halo_infinite/maps` retourne JSON avec >= 10 maps. Entrée thought_log.md ajoutée.

---

## Phase 2 — Frontend React (2 j)

### Structure des fichiers
```
apps/web/src/features/asset-drawer/
├── AssetDrawer.tsx        ← container (tab fixe + panel)
├── AssetGrid.tsx          ← grille 2 cols, lazy images
├── AssetCard.tsx          ← thumbnail + label (map ou arme)
├── AssetSearch.tsx        ← input debounced 300ms
├── assetDrawer.store.ts   ← Zustand store (localStorage persist)
├── useAssetDrawer.ts      ← TanStack Query hooks
└── index.ts
```

### 2.1 — Store Zustand
Fichier : `apps/web/src/features/asset-drawer/assetDrawer.store.ts`

```typescript
interface AssetDrawerState {
  isOpen: boolean
  activeTab: 'maps' | 'weapons'
  search: string
  open: () => void
  close: () => void
  setTab: (tab: 'maps' | 'weapons') => void
  setSearch: (q: string) => void
}
```

Persister `isOpen` + `activeTab` via Zustand persist middleware. Ne pas persister `search`.

### 2.2 — Query keys
Dans `apps/web/src/lib/query/keys.ts` — ajouter :
```typescript
assetMaps: (title: string, q: string) => ['assets', title, 'maps', q],
assetWeapons: (title: string, q: string) => ['assets', title, 'weapons', q],
```

### 2.3 — Hook useAssetDrawer
```typescript
function useAssetMaps(title: string, search: string): UseQueryResult<AssetMeta[]>
function useAssetWeapons(title: string, search: string): UseQueryResult<AssetMeta[]>
```

`staleTime: 5 * 60 * 1000` — les listes de maps/armes ne changent pas pendant une session.

### 2.4 — AssetCard
- `loading="lazy"` sur l'image
- `onError` → placeholder SVG silhouette
- Label via locale courante (`name_en` ou `name_fr`)
- Zéro hex/Tailwind couleur directe → `tokenCssVar(...)` pour toutes les couleurs sémantiques

### 2.5 — Layout du drawer

```
[bord droit, position fixed, centré verticalement]
Tab 40px → clic → drawer 360px large, 70vh, slide-in

┌─────────────────────────────────────────┐
│  Maps  │  Armes                    [×]  │
├─────────────────────────────────────────┤
│  Rechercher...                          │
├─────────────────────────────────────────┤
│ [img][nom]   [img][nom]                 │
│ [img][nom]   [img][nom]   ← scroll Y   │
│ [img][nom]   [img][nom]                 │
└─────────────────────────────────────────┘
```

Pas de backdrop, pas de modal — l'app reste utilisable derrière. Z-index élevé sans overlay bloquant.  
Animation slide-in CSS pure : `transform: translateX()`, `transition: 200ms ease-out`.

### 2.6 — Intégration AppShell
Dans `apps/web/src/components/shell/AppShell.tsx` : ajouter `<AssetDrawer />` en dehors du `<main>`, toujours monté, état géré par le store.

### 2.7 — i18n
```
drawer.tab.maps            → "Maps"           / "Maps"
drawer.tab.weapons         → "Weapons"        / "Armes"
drawer.search.placeholder  → "Search..."      / "Rechercher..."
drawer.empty.maps          → "No map found."  / "Aucune map trouvée."
drawer.empty.weapons       → "No weapon found." / "Aucune arme trouvée."
```

**Critère** : drawer s'ouvre/ferme, grille affiche les maps de `halo_infinite`, recherche filtre en temps réel. Entrée thought_log.md ajoutée.

---

## Phase 3 — Polish & a11y (0,5 j)

- Fermeture au `Escape` (keydown handler dans le drawer)
- Focus trap non requis (drawer non modal)
- Responsive : décision ouverte — désactiver sur mobile ou bottom sheet 50vh

---

## Récapitulatif effort

| Phase | Durée |
|-------|-------|
| 0 — Audit + réorg `/static/` | 0,5 j |
| 0.5 — AssetResolver interface + impl locale (via PathResolver) + refactor handler | ~2 h |
| 1 — Backend Go (canonical + repo + service + handler + tests 3 couches) | 1,5 j |
| 2 — Frontend React (store + query + composants + i18n) | 2 j |
| 3 — Polish | 0,5 j |
| **Total** | **~5 jours** |

---

## Hors scope V1

- Stats (match_count, win_rate) dans les cards — query sur shared_matches, feature V2
- Bottom sheet mobile responsive
- Raccourcis clavier globaux
- Favoris maps/armes (requires store persistence + éventuellement user_preferences table)

## Dette technique V1 documentée

**G5 — `MetadataRepo` exposé directement via `port.AssetMetaRepository`** : les maps étant title-specific, l'accès devrait idéalement passer par `TitleDataAdapter.LoadAssets()`. Pour V1 (Halo Infinite uniquement), l'interface `port.AssetMetaRepository` est acceptable. À migrer vers l'adapter si un second titre est ajouté.

---

## Points ouverts à valider

~~1. **B1** — vérifier `map_images_registry.local_path`~~ → hors scope, géré par une autre équipe. `LocalAssetResolver` sera implémenté en supposant que `local_path` est valide.
~~2. **Réorg `/static/` multi-title**~~ → hors scope, géré par une autre équipe. L'`AssetResolver` encapsule les chemins, la réorg peut avoir lieu indépendamment sans toucher au code.
~~3. **Drawer non-modal**~~ → confirmé non-modal. Pas de backdrop, l'app reste utilisable avec ou sans le drawer ouvert.

**Aucun point ouvert restant — plan prêt à implémenter.**

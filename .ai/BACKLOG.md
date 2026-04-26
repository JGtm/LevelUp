— Tâches et TODO centralisés

> Mis à jour le 2026-04-26.

---

## 🔄 Aucune tâche en cours

---

## 📋 Backlog

---

### [Multi-titre] Couche canonique `weapon_family` cross-titres

**Noté le** : 2026-04-26 | **Priorité** : Basse — bloqué par arrivée d'un second titre réel

**Contexte** : Plan complet documenté dans `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` (référentiel `weapon_families` global + colonne `family_key` sur `weapon_labels` par-titre + TOML source-de-vérité). L'audit `.ai/AUDIT_WEAPONS_2026-04-25.md` a validé la faisabilité sur HI : 42 weapon_id seedés, 88 % mappables vers ~17 familles canoniques, ~32 familles totales prévues pour couvrir Halo CE→Infinite. Effort estimé : 2.5–3j en 3 phases (référentiel global + mapping HI + adapter/endpoint).

**Ce qui doit être fait** :
1. Phase 1 — créer `data/warehouse/canonical_metadata.duckdb` avec tables `weapon_families` + `weapon_family_translations` ; créer `config/canonical/weapon_families.toml` (~32 familles) ; script `tools/seed-weapon-families.go`.
2. Phase 2 — `ALTER TABLE weapon_labels ADD COLUMN family_key VARCHAR` côté HI ; créer `config/titles/halo_infinite/mappings/weapon_families.toml` (~37 lignes) ; seeder via `tools/seed-weapon-families-mapping.go`.
3. Phase 3 — étendre `TitleSemanticAdapter` avec `WeaponFamilies()` + `WeaponFamilyOf(weaponID)` ; handler `/api/v1/weapon-families` derrière flag `WEAPON_FAMILIES_API_ENABLED=false`.

**Ajustements vs plan d'origine** (cf. AUDIT §7.1) :
- compléter l'annexe §10 du plan avec 6 familles HI manquantes (`shock_rifle`, `stalker_rifle`, `heatwave`, `sentinel_beam`, `ravager`, `mutilator`) → passe de 26 à ~32 familles ;
- expliciter `family_key = NULL` comme valide pour les sentinelles (Grenade/Melee/Vehicle) et easter-eggs ;
- relever le seuil de couverture du test CI de 60 % à 85 % (HI réel à 88 %).

**Conditions de déblocage** :
1. ✅ Phase A multi-titres terminée (commit `aaccbe12`+) ;
2. ❌ second titre réel (Halo 5, MCC, ODST…) validé en pipeline produit.

**Documents liés** :
- `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` (plan complet)
- `.ai/AUDIT_WEAPONS_2026-04-25.md` (audit du référentiel HI)

---

### [Multi-titre] Migration `static/` vers une arborescence title-scoped

**Noté le** : 2026-04-22 | **Audit étoffé** : 2026-04-26 | **Priorité** : Basse — ne bloque pas Halo Infinite seul

**Contexte** : `static/maps/`, `static/medals/icons/`, `static/ranks/`, `static/weapons-assets/` sont flat (tous les assets HI au même niveau). Quand un 2e titre arrivera, conflits d'IDs identiques. À noter : `static/commendations/{h5g,hi}/` est **déjà** title-scoped (convention historique avec slugs courts), `data/cache/{kind}/{titleID}/` aussi via `LocalFSStore.Path()` — seule la couche `static/` overridable est restée flat.

**Inventaire FS actuel** (état au 2026-04-26) :
```
static/
├── bg_space.webp, logo.png, styles.css   ← UI globale (non-titre)
├── ui-icons/        (3 PNG)               ← UI globale (non-titre)
├── commendations/h5g/ + hi/  (1+26 PNG)   ← DÉJÀ title-scoped
├── maps/            (102 .png/.jpg)       ← FLAT, à migrer
├── medals/icons/    (166 PNG)             ← FLAT, à migrer
├── ranks/           (32 PNG, préfixés "120px-HINF-CSR_*")  ← FLAT, à migrer
└── weapons-assets/  (28 PNG)              ← FLAT, à migrer
```

**Audit exhaustif des points d'entrée à modifier** (Go + frontend + DB) :

#### A. Génération d'URLs `/static/...` côté Go (5 sites de hardcodage actifs)

| # | Fichier:ligne | Pattern | Remplacement |
|---|---|---|---|
| A1 | [internal/analysis/home.go:1081](apps/go-api/internal/analysis/home.go#L1081) | `"/static/maps/" + encoded + ext` | `assets.StaticAssetPath(StaticKindMap, titleID, encoded, ext)` |
| A2 | [internal/platform/duckdb/home_repo.go:400](apps/go-api/internal/platform/duckdb/home_repo.go#L400) | `"/static/ranks/120px-HINF-CSR_Onyx.png"` | `assets.StaticAssetPath(StaticKindRank, titleID, "120px-HINF-CSR_Onyx", ".png")` |
| A3 | [internal/platform/duckdb/home_repo.go:406](apps/go-api/internal/platform/duckdb/home_repo.go#L406) | `fmt.Sprintf("/static/ranks/120px-HINF-CSR_%s%d.png", ...)` | idem A2 avec id formaté |
| A4 | [internal/platform/duckdb/home_repo.go:833](apps/go-api/internal/platform/duckdb/home_repo.go#L833) | `fmt.Sprintf("/static/medals/icons/%d.png", rr.medalID)` | `assets.StaticAssetPath(StaticKindMedal, titleID, strconv.FormatUint(rr.medalID, 10), ".png")` |
| A5 | [cmd/migrate-static-maps/main.go:115](apps/go-api/cmd/migrate-static-maps/main.go#L115) | `fmt.Sprintf("/static/maps/%s", filename)` | injecter `titleID` dans le path |

#### B. Configuration de routage / FileServer (2 sites)

| # | Fichier:ligne | Pattern | Action |
|---|---|---|---|
| B1 | [internal/api/server.go:199](apps/go-api/internal/api/server.go#L199) | `StaticMapDir: filepath.Join(cfg.RepoRoot, "static", "maps")` | passer à `filepath.Join(..., "static", "maps", titleSlug)` ou retirer (la title-scoping passe par `StaticAssetPath`) |
| B2 | [internal/api/server.go:211-215](apps/go-api/internal/api/server.go#L211-L215) | `staticDir := filepath.Join(cfg.RepoRoot, "static")` + `r.Handle("/static/*", ...)` | inchangé (FileServer sert l'arbo, peu importe la profondeur) |

#### C. AssetResolver layer (override `KindMapImage` aujourd'hui)

| # | Fichier:ligne | Pattern | Action |
|---|---|---|---|
| C1 | [internal/assets/wire.go:25-43](apps/go-api/internal/assets/wire.go#L25-L43) | `cfg.StaticMapDir` → `WithRootOverride(KindMapImage, ...)` | l'override pointe vers `static/maps/{titleID}/` après migration. NB : `LocalFSStore.Path()` ajoute `{kind}/{titleID}` au root → quand `RootOverride` pointe sur `static/maps/`, le path résolu devient `static/maps/map-image/halo_infinite/<id>.png` qui ne correspond PAS à la réalité FS. **L'override actuel est en partie cassé** — c'est `mapStaticImagePath()` (A1) qui produit l'URL effective. À nettoyer : retirer l'override ou refondre `Path()` pour que l'override remplace le chemin **complet** au lieu du root seulement. |

#### D. Frontend (2 sites + fixtures de test)

| # | Fichier:ligne | Pattern | Action |
|---|---|---|---|
| D1 | [apps/web/src/features/home/HomePage.tsx:412](apps/web/src/features/home/HomePage.tsx#L412) | `src="/static/ranks/Unranked.png"` | injecter via prop ou helper React `staticAsset('ranks', 'Unranked.png', titleSlug)` |
| D2 | [apps/web/src/features/home/HomeRecentPlaylistsCard.tsx:77](apps/web/src/features/home/HomeRecentPlaylistsCard.tsx#L77) | idem | idem |
| D3 | [apps/web/src/features/home/HomePage.test.tsx:71,76,109,112](apps/web/src/features/home/HomePage.test.tsx) | fixtures `'/static/ranks/120px-HINF-CSR_Gold3.png'` | mettre à jour vers `'/static/ranks/halo_infinite/...'` |
| D4 | [apps/go-api/internal/analysis/home_test.go](apps/go-api/internal/analysis/home_test.go) + `player_repos_test.go` + `commendation_handler_test.go` + `store_localfs_test.go` + `store_duckdb_test.go` | fixtures `/static/...` | UPDATE en cohérence |

#### E. Stockage DB — colonnes contenant des chemins `/static/...`

| # | Table.colonne | Format actuel | Action UPDATE |
|---|---|---|---|
| E1 | `metadata.duckdb.map_images_registry.local_path` | `/static/maps/<filename>` | `UPDATE map_images_registry SET local_path = REPLACE(local_path, '/static/maps/', '/static/maps/halo_infinite/') WHERE title_id = 'halo_infinite' AND local_path LIKE '/static/maps/%'` |
| E2 | `metadata.duckdb.citation_mappings.image_path` | `static/commendations/h5g/<file>` (sans `/` initial — pour H5G uniquement, déjà title-scoped) | aucune action si on conserve `h5g`. Si on renomme → `halo_5_guardians`, UPDATE bulk |
| — | `metadata.duckdb.battlepass_seasons.{battlepass_image_path,background_image_path}` | URLs Waypoint externes | **PAS** concerné, hors `/static/` |
| — | `metadata.duckdb.medal_image_cache.local_path` | `data/cache/medal-image/halo_infinite/...` | **DÉJÀ** title-scoped, hors `static/` |

#### F. Commentaires / documentation à rafraîchir

- [internal/domain/media.go:239](apps/go-api/internal/domain/media.go#L239) (commentaire DTO `// /static/maps/X.png`)
- [internal/platform/duckdb/map_cache_repo.go:20](apps/go-api/internal/platform/duckdb/map_cache_repo.go#L20) (commentaire struct field)
- [internal/analysis/citation_snippets.go:96](apps/go-api/internal/analysis/citation_snippets.go#L96) (commentaire format `image_path`)
- [internal/assets/kinds.go:14](apps/go-api/internal/assets/kinds.go#L14) (commentaire `KindMapImage` mention `static/maps/`)

#### G. Cas H5G / `commendations/{h5g,hi}/` — décision verrouillée

**Décision** : unifier vers les slugs canoniques longs (Option 2). Les raccourcis `h5g`/`hi` sont un reliquat à éliminer.
- Renommer `static/commendations/h5g/` → `static/commendations/halo_5_guardians/`
- Renommer `static/commendations/hi/` → `static/commendations/halo_infinite/`
- UPDATE bulk de `metadata.duckdb.citation_mappings.image_path` :
  ```sql
  UPDATE citation_mappings
  SET image_path = REPLACE(image_path, 'static/commendations/h5g/', 'static/commendations/halo_5_guardians/')
  WHERE image_path LIKE 'static/commendations/h5g/%';
  -- idem pour 'hi/' → 'halo_infinite/'
  ```
- Effectuer dans **PR 1** (même commit que la migration FS HI), pas de PR séparée — éviter un état intermédiaire incohérent.

---

### Architecture modulaire proposée — séparation de couches stricte

**Principe** : 4 couches avec responsabilités strictement séparées (SRP). Chaque couche ignore tout des couches au-dessus.

```
┌──────────────────────────────────────────────────────────────────────┐
│ Couche 4 — Caller (services produit, handlers, frontend)             │
│   home.go, home_repo.go, citation_snippets.go, HomePage.tsx          │
│   Connaît UNIQUEMENT l'identifiant métier (mapName, medalID, tier)   │
│   Aucune connaissance de path, mountpoint, encodage, naming          │
└──────────────────┬───────────────────────────────────────────────────┘
                   ↓ adapter.MapImageURL("Aquarius")  → URL
┌──────────────────────────────────────────────────────────────────────┐
│ Couche 3 — TitleAssetURLAdapter (un par titre, derrière interface)   │
│   internal/games/halo_infinite/adapter_asset_urls.go                 │
│   Connaît : conventions HI, mapPNGNames, préfixe "120px-HINF-CSR_*", │
│   encodage espaces map names, fallback sur UUID/empty                │
│   Délègue la composition path à la couche 2                          │
└──────────────────┬───────────────────────────────────────────────────┘
                   ↓ static.URL(KindMap, "halo_infinite", encodedID, ext)
┌──────────────────────────────────────────────────────────────────────┐
│ Couche 2 — Static layout / URL composition (générique, sans titre)   │
│   internal/assets/static/{kinds.go, layout.go, urls.go}              │
│   Connaît : MountPoint, Folder per Kind, RelPath assembly            │
│   Pas de connaissance titre, pas de naming convention                │
│   Pure logique, zero I/O, tests table-driven triviaux                │
└──────────────────┬───────────────────────────────────────────────────┘
                   ↓ "/static/maps/halo_infinite/Aquarius.png"
┌──────────────────────────────────────────────────────────────────────┐
│ Couche 1 — FS serving (HTTP) — inchangé structurellement              │
│   internal/api/server.go : http.FileServer générique                 │
│   Sert l'arbo telle qu'elle est, ignore la sémantique                │
└──────────────────────────────────────────────────────────────────────┘
```

#### Couche 2 — `internal/assets/static/` (nouveau package, ~80 LoC)

Pure logique, zéro dépendance à un titre ou à une convention de naming.

```go
// internal/assets/static/kinds.go
package static

type Kind string
const (
    KindMap          Kind = "map"
    KindMedal        Kind = "medal"
    KindCSRRank      Kind = "csr-rank"
    KindWeapon       Kind = "weapon"
    KindCommendation Kind = "commendation"
)

func (k Kind) Valid() bool { /* … */ }

// internal/assets/static/layout.go
const MountPoint = "/static"

// Folder retourne le sous-dossier sous /static/ pour un Kind donné.
// Centralise la convention historique (medals → "medals/icons", etc.).
func Folder(k Kind) string {
    switch k {
    case KindMap:          return "maps"
    case KindMedal:        return "medals/icons"
    case KindCSRRank:      return "ranks"
    case KindWeapon:       return "weapons-assets"
    case KindCommendation: return "commendations"
    }
    return ""
}

// internal/assets/static/urls.go
// URL compose l'URL absolue d'un asset statique title-scopé.
// L'id doit être pré-encodé (path-safe). Le caller (couche 3) gère l'encodage
// selon les règles propres au Kind (espaces pour maps, ID numérique pour médailles).
// Retourne "" si k invalide ou titleSlug/id vide.
func URL(k Kind, titleSlug, id, ext string) string {
    if !k.Valid() || titleSlug == "" || id == "" { return "" }
    return path.Join(MountPoint, Folder(k), titleSlug, id+ext)
}

// internal/assets/static/fs.go (pour le FileServer + migration tooling)
// AbsRoot retourne le chemin absolu de la racine /static/ sur le filesystem.
func AbsRoot(repoRoot string) string { return filepath.Join(repoRoot, "static") }

// AbsKindRoot retourne le répertoire absolu d'un (kind, titleSlug).
// Utilisé par migrate-static-maps + tests.
func AbsKindRoot(repoRoot string, k Kind, titleSlug string) string {
    return filepath.Join(AbsRoot(repoRoot), Folder(k), titleSlug)
}
```

Tests : `static_test.go` table-driven (kind, slug, id, ext) → URL attendu, ~15 cas.

#### Couche 3 — interface `TitleAssetURLAdapter` + impl HI

**Ajout dans `internal/games/adapter.go`** :

```go
// TitleAssetURLAdapter expose la construction d'URLs d'assets statiques
// title-spécifiques. Convertit un identifiant métier (mapName, medalID, tier)
// en URL relative, en encapsulant les conventions de naming propres au titre.
//
// Implémente la couche 3 du plan static/ multi-titre. Les callers (home,
// home_repo, citation_snippets, frontend) reçoivent cet adapter via le
// Resolver et n'ont aucune connaissance du format path/encoding.
type TitleAssetURLAdapter interface {
    TitleSlug() string

    MapImageURL(mapName string) string         // "" si invalide/UUID
    MedalImageURL(medalID uint64) string
    CSRRankImageURL(tier string, subTier int) string
    CSRRankImageURLOnyx() string               // cas spécial (sans subTier)
    WeaponImageURL(weaponName string) string
    CommendationImageURL(relPath string) string  // pour citation_snippets
}
```

**Extension du `Resolver`** :

```go
type Resolver interface {
    Data(slug string) (TitleDataAdapter, error)
    Semantic(slug string) (TitleSemanticAdapter, error)
    AssetURL(slug string) (TitleAssetURLAdapter, error)  // ← nouveau
    DefaultSlug() string
}
```

**Impl HI : `internal/games/halo_infinite/adapter_asset_urls.go`** (nouveau)

```go
type AssetURLAdapter struct {
    titleSlug string
    mapPNGSet map[string]struct{} // ex-mapPNGNames de home.go (déplacé)
}

func NewAssetURLAdapter() *AssetURLAdapter {
    return &AssetURLAdapter{
        titleSlug: "halo_infinite",
        mapPNGSet: defaultMapPNGSet, // chargé depuis ranks_loader ou TOML
    }
}

func (a *AssetURLAdapter) TitleSlug() string { return a.titleSlug }

func (a *AssetURLAdapter) MapImageURL(mapName string) string {
    name := strings.TrimSpace(mapName)
    if name == "" || isUUID(name) { return "" }
    ext := ".jpg"
    if _, ok := a.mapPNGSet[name]; ok { ext = ".png" }
    return static.URL(static.KindMap, a.titleSlug, encodeMapName(name), ext)
}

func (a *AssetURLAdapter) MedalImageURL(medalID uint64) string {
    return static.URL(static.KindMedal, a.titleSlug,
        strconv.FormatUint(medalID, 10), ".png")
}

func (a *AssetURLAdapter) CSRRankImageURL(tier string, subTier int) string {
    id := fmt.Sprintf("120px-HINF-CSR_%s%d", tier, subTier)
    return static.URL(static.KindCSRRank, a.titleSlug, id, ".png")
}

func (a *AssetURLAdapter) CSRRankImageURLOnyx() string {
    return static.URL(static.KindCSRRank, a.titleSlug, "120px-HINF-CSR_Onyx", ".png")
}
// + WeaponImageURL, CommendationImageURL …
```

#### Couche 4 — refactor des callers

**`internal/analysis/home.go::buildMapImageURL`** :
```go
// Avant : connaît /static/maps/, mapPNGNames, encodage, fallback URL absolute
// Après : ne connaît que l'adapter
func buildMapImageURL(adapter games.TitleAssetURLAdapter, mapID, mapName, mapNameFR string) *string {
    if u := adapter.MapImageURL(mapName);   u != "" { return &u }
    if u := adapter.MapImageURL(mapNameFR); u != "" { return &u }
    if mapID == "" { return nil }
    url := fmt.Sprintf("/api/v1/assets/maps/%s/%s/image", adapter.TitleSlug(), mapID)
    return &url
}
```

**`internal/platform/duckdb/home_repo.go`** : injecter `assetURL games.TitleAssetURLAdapter` dans le constructeur du repo (ou via `MetadataRepo`). Les 3 sites de hardcodage URL deviennent :
```go
// L400  : path := r.assetURL.CSRRankImageURLOnyx()
// L406  : path := r.assetURL.CSRRankImageURL(normalizedTier, normalizedSubTier)
// L833  : ImageURL := r.assetURL.MedalImageURL(rr.medalID)
```

**`cmd/migrate-static-maps/main.go`** : utiliser `static.AbsKindRoot` pour scanner et `assetURL.MapImageURL(mapName)` pour générer les `local_path` à upserter.

**Frontend : `apps/web/src/lib/staticAssets/`**

Mêmes 4 couches côté React :

```
apps/web/src/lib/staticAssets/
├── kinds.ts          → type Kind = 'map' | 'medal' | 'csr-rank' | …  (couche 2)
├── layout.ts         → MOUNT_POINT, folderFor(kind), assetURL(...)   (couche 2)
├── halo_infinite.ts  → mapPNGSet, mapImageURL(name), csrRankURL(...) (couche 3)
└── useAssetURL.ts    → hook React qui retourne l'adapter du titre courant (couche 3 → 4)
```

```ts
// useAssetURL.ts
export function useAssetURL(): TitleAssetURLAdapter {
    const slug = useAppShellStore((s) => s.currentTitleSlug)
    return useMemo(() => makeAssetURLAdapter(slug), [slug])
}
// Dans HomePage.tsx (avant: src="/static/ranks/Unranked.png")
const assetURL = useAssetURL()
<img src={assetURL.csrRankImageURL('Unranked', 0)} />
```

---

### Bénéfices de l'architecture en couches

| Bénéfice | Détail |
|---|---|
| **SRP strict** | Chaque couche : une responsabilité, une raison de changer. |
| **Title-agnostic généralisé** | Onboarding H5G = ajouter `internal/games/halo_5_guardians/adapter_asset_urls.go` qui implémente `TitleAssetURLAdapter`. Aucun caller modifié. |
| **Testabilité par couche** | Couche 2 : table-driven sans I/O. Couche 3 : tests sur naming HI seulement. Couche 4 : mock du `TitleAssetURLAdapter`. |
| **Migration FS découplée du refactor** | PRs 0+1+2 introduisent l'archi sans changer les valeurs (titleSlug = `"halo_infinite"`, FS flat). PR 3 active la title-scoping en ajustant la couche 2 + déplace les fichiers. |
| **Bug `KindMapImage` override** | Disparaît : la couche 2 prend la place du `WithRootOverride` cassé. |
| **Convergence avec `assets.Resolver` cache-aside** | À terme, `TitleAssetURLAdapter` peut basculer sur le resolver cache-aside (cf. `PLAN_ASSETS_ABSTRACTION.md`) en un changement local — couche 4 ne change pas. |
| **Frontend symétrique** | Même découpage côté React, hook `useAssetURL()` masque la résolution titre courant. |

---

### Ordonnancement révisé (4+1 PRs)

#### PR 0 — couche 2 Go + frontend (générique, neutre)

- Créer `internal/assets/static/{kinds,layout,urls,fs}.go` + tests
- Créer `apps/web/src/lib/staticAssets/{kinds,layout}.ts` + tests Vitest
- Aucun caller modifié. Modules nus, prêts à l'emploi.
- Verts : `go test ./internal/assets/static/` + `npx vitest run src/lib/staticAssets/`

#### PR 1 — couche 3 (interface + impls HI Go & TS)

- Ajouter `TitleAssetURLAdapter` à `internal/games/adapter.go` + extension du `Resolver`
- Créer `internal/games/halo_infinite/adapter_asset_urls.go` (déplacer `mapPNGNames` depuis `analysis/home.go`)
- Créer `apps/web/src/lib/staticAssets/halo_infinite.ts` + `useAssetURL.ts`
- Câbler dans `api/server.go` : `reg.RegisterAssetURL(halo_infinite.NewAssetURLAdapter())`
- Tests : table-driven HI (mapName → URL, medalID → URL, tier → URL)

#### PR 2 — couche 4 (callers refactor — neutre, valeurs inchangées)

- `home.go::buildMapImageURL` : prend `TitleAssetURLAdapter` en argument
- `home_repo.go` (3 sites) : injecter `assetURL` via `MetadataRepo` constructor
- `migrate-static-maps/main.go` : utiliser `static.AbsKindRoot` + `assetURL.MapImageURL`
- Frontend : 2 composants utilisent `useAssetURL()` au lieu de littéraux
- Suppression de `mapStaticImagePath` + re-export public `MapStaticImagePath` (mort)
- Suppression de l'override cassé `WithRootOverride(KindMapImage, ...)` dans `wire.go`
- Aucune valeur ne change → fixtures de test inchangées
- Tests existants doivent rester verts (smoke ~30 fixtures)

#### PR 3 — migration FS Halo Infinite + commendations slugs longs

- `git mv` :
  - `static/maps/*.{png,jpg}` → `static/maps/halo_infinite/`
  - `static/medals/icons/*.png` → `static/medals/icons/halo_infinite/`
  - `static/ranks/*.png` → `static/ranks/halo_infinite/`
  - `static/weapons-assets/*.png` → `static/weapons-assets/halo_infinite/`
  - `static/commendations/h5g/` → `static/commendations/halo_5_guardians/`
  - `static/commendations/hi/` → `static/commendations/halo_infinite/`
- Activer `titleSlug != ""` dans la couche 2 (constante par défaut + assertion non-vide)
- UPDATE bulk DuckDB :
  - `map_images_registry.local_path` (E1)
  - `citation_mappings.image_path` (E2 unifié sur slugs longs)
- Mettre à jour les fixtures de test (D3, D4 dans audit + Go test files)
- Smoke navigateur : home (carte mode/map, badge CSR), recent playlists, citations, médailles. 5 matchs.

#### PR 4 — onboarding 2e titre (déclencheur)

- `static/{maps,medals/icons,ranks,weapons-assets,commendations}/halo_5_guardians/`
- `internal/games/halo_5_guardians/adapter_asset_urls.go` (impl `TitleAssetURLAdapter`)
- `apps/web/src/lib/staticAssets/halo_5_guardians.ts`
- `db_profiles.json` + registration dans `api/server.go`
- Smoke tests de bascule de titre dans le UI

---

### Risques et mitigations

| Risque | Impact | Mitigation |
|---|---|---|
| Mismatch FS ↔ DB après UPDATE | Images cassées sur les matchs existants | Transaction unique : `BEGIN; git mv; UPDATE; COMMIT;` ou script qui vérifie `os.Stat` avant UPDATE |
| Override `KindMapImage` cassé (cf. C1) | Comportement actuel masqué par `mapStaticImagePath` qui court-circuite l'asset resolver | Profiter de PR 0 pour soit retirer l'override, soit corriger `Path()` (override = chemin complet) |
| Slugs incohérents `hi`/`h5g` vs `halo_infinite` | Confusion future | Trancher Option 1 vs Option 2 **avant** PR 1 |
| Frontend SSR / cache CDN | URLs anciennes en cache | Bump du `cache-control` de `/static/*` ou hash dans le nom de fichier — hors scope ici |

### Conditions de déblocage

- Onboarding effectif d'un 2e titre dans `db_profiles.json` (déclencheur principal)
- OU décision préventive de refactorer `static_paths.go` (PR 0) seule — recommandé car coût faible et bénéfice immédiat (1 seul site à toucher pour la migration FS le moment venu)

---

---

### [Multi-titre/O10] Store / economy tracker

**Noté le** : 2026-04-18 | **Priorité** : Basse — backlog multi-titre, hors scope Halo Infinite

**Contexte** : Opportunité O10 identifiée lors de la revue des repos externes (SpartanRecord). Non pertinente pour Halo Infinite aujourd'hui (store en fin de cycle commercial, risque d'obsolescence avant livraison). Gardée en backlog car potentiellement utile si un nouveau titre Halo dispose d'une économie de store active (cosmétiques, battle pass, rotations de boutique).

**Référence** : `.ai/go_migration_v2/HALO_EXTERNAL_OPPORTUNITIES.md` §O10

**Conditions de déblocage** :
1. Onboarding d'un nouveau titre avec économie de store active confirmée
2. Signal utilisateur explicite sur l'intérêt du tracking store pour ce titre
3. Atterrissage UI validé comme module optionnel dans `Home`, jamais comme menu prioritaire

**Périmètre si débloqué** :
- Fetcher Waypoint pour les rotations de boutique du titre concerné
- Persistance dans `metadata.duckdb` avec `title_id` comme clé de partition (déjà prévu dans l'architecture O3/O8)
- Module compact `Home` scoped au titre — pas de navigation globale
- Jamais comme sous-produit autonome hors du scope analytics de LevelUp

**Point de vigilance** : ne pas ouvrir ce chantier sur Halo Infinite même sous pression — le store y est en déclin et le risque d'obsolescence est élevé.

---

### [Multi-titre/O11] Spartan Company / social layer

**Noté le** : 2026-04-18 | **Priorité** : Basse — backlog multi-titre, hors scope Halo Infinite

**Contexte** : Opportunité O11 identifiée lors de la revue des repos externes (SpartanRecord). Non pertinente pour Halo Infinite aujourd'hui (pas de signal utilisateur, dimension groupe déjà couverte partiellement par `Squad`). Gardée en backlog car potentiellement utile si un nouveau titre Halo dispose d'une dimension clan ou groupe native établie (guildes, escouades persistantes, companies).

**Référence** : `.ai/go_migration_v2/HALO_EXTERNAL_OPPORTUNITIES.md` §O11

**Conditions de déblocage** :
1. Onboarding d'un nouveau titre avec dimension groupe native confirmée
2. Ou signal utilisateur explicite sur le besoin de gestion de groupes dans LevelUp (hors `Squad` existant)
3. Dans tous les cas : valider l'atterrissage dans `Squad` avant toute autre surface

**Périmètre si débloqué** :
- Extension de la page `Squad` existante : groupes / cohortes sauvegardées scoped au titre
- Appels Waypoint vers les endpoints `Spartan Company` ou équivalent du nouveau titre
- Jamais comme rubrique de navigation autonome ni comme sous-produit social parallèle à LevelUp
- L'architecture multi-titre (`title_id` dans `xuid_aliases`, `match_participants`, etc.) le rend naturellement extensible sans restructuration

**Point de vigilance** : toute dérive vers une surface sociale indépendante de `Squad` est à refuser — la valeur de LevelUp est analytique, pas sociale.

---

### [Migration] Cible desktop Tauri web-first, sans réécriture Rust métier

**Noté le** : 2026-04-12 | **Priorité** : Moyenne (distribution simplifiée, non bloquante pour les slices MVP)

**Référence plan** : `.ai/MIGRATION_MASTER.md`, `.ai/migration/DECISIONS.md`

**Problème** : La migration React/FastAPI améliore l'UX et le déploiement web, mais ne résout pas à elle seule le cas utilisateur néophyte qui ne doit ni installer Python, ni lancer `pip`, ni manipuler un terminal. Il faut documenter une cible desktop installable qui n'abîme pas la stratégie web/VPS.

**Décision cible** : Conserver une architecture **web-first** (`apps/web` + `apps/api`) comme source de vérité produit, puis ajouter **Tauri comme coque desktop** optionnelle. Rust est explicitement **hors périmètre métier** : aucune logique de sync, auth Halo, DuckDB, filtres, agrégats, visualisations ou contrats API ne doit être réécrite en Rust.

**Solution** : Préparer un spike de packaging Tauri autour du frontend React existant et d'un backend FastAPI/Python local packagé, avec un contrat d'intégration minimal et réversible.

**Changements ciblés** :
1. Architecture : figer la règle `React navigateur d'abord`, `FastAPI canonique`, `Tauri simple shell desktop`
2. Packaging : définir comment lancer/arrêter proprement le backend Python local depuis l'app desktop, avec gestion des logs, ports, répertoires de données et erreurs de démarrage
3. Frontend : isoler les appels natifs desktop derrière une couche d'adaptation pour que l'app reste exécutable telle quelle sur navigateur et sur VPS
4. Données locales : cadrer les chemins Windows pour DuckDB, médias, cache et configuration utilisateur sans hardcoder de chemins machine
5. Distribution : évaluer installateur Windows, taille du bundle, temps de démarrage et absence de prérequis Python côté utilisateur final
6. Exploitation : préserver explicitement la cible VPS en interdisant toute dépendance produit au runtime Tauri/Rust
7. Go/no-go : définir les critères du spike (installation propre, backend embarqué stable, auth utilisable, fichiers locaux OK, perf de lancement acceptable)

**Point de vigilance** : Tauri implique mécaniquement une fine couche Rust côté shell. Ce point est acceptable uniquement comme détail d'enveloppe technique. Toute dérive vers des commandes Rust métier, un stockage canonique côté Tauri ou une divergence desktop-only dans les flux React/FastAPI doit être refusée.

---

### Script d'analyse des kills par arme pour un match donné (v8+)

**Noté le** : 2026-03-27
**Priorité** : Basse

**Contexte** : Outil de diagnostic/exploration permettant d'analyser en détail tous les kills d'un match donné, pour un joueur donné.

**Entrée** : `match_id` + `gamertag`

**Sortie** : Tableau avec, pour chaque kill :
- `match_id`
- Paire `killer` / `victim` (gamertag ou xuid si inconnu)
- `timestamp` en format `mm:ss`
- `weapon_id` (même si inconnu / non résolu)

**Ce que ça impliquerait** :
1. Requête sur `weapon_kills` (shared_matches_v2) jointure `killer_victim_pairs` + `xuid_aliases`
2. Résolution des gamertags via `v_gamertag_lookup`
3. Conversion `timestamp_ms` → `mm:ss`
4. Affichage : script CLI + éventuellement widget UI dans la page d'un match

**Complexité estimée** : Faible (données déjà disponibles dans `weapon_kills` + vues v6)

**Priorité** : Basse — outil de debug / exploration, non bloquant pour les features v7

---

### Kills environnementaux — catégorie dédiée (v8++)

**Contexte** : La médaille **Kong** (kill via baril projeté) est actuellement comptée dans `GRENADE_MEDALS` faute d'une meilleure catégorie. Ce classement est approximatif — il est impossible de savoir avec certitude si l'API inclut ces kills dans `GrenadeKills` ou non.

**Idée** : Créer une catégorie `environmental_kills` (ou `environmental`) pour regrouper les kills causés par l'environnement sans arme tenue :
- Baril projeté (médaille **Kong**)
- Potentiellement : chutes provoquées, explosions de véhicules, etc.

**Ce que ça impliquerait** :
1. Nouvelle colonne `environmental_kills` dans `match_participants` (migration DuckDB)
2. Nouveau bit `ParticipantBits.ENVIRONMENTAL_KILLS` dans `constants.py`
3. Retirer `Kong` de `GRENADE_MEDALS` → nouvel ensemble `ENVIRONMENTAL_MEDALS`
4. Logique de réconciliation filmshell dédiée dans `_weapon_kills_repo.py`
5. Backfill pour l'historique existant
6. Affichage UI éventuel

**Complexité estimée** : Moyenne (surtout le backfill + validation que l'API expose bien des compteurs séparés)

**Priorité** : Basse — les barrel kills sont extrêmement rares, l'impact sur les stats est négligeable. À faire uniquement si on veut une exhaustivité totale des catégories de kills.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-04-10 | **Score de forme individuel + escouade** : `compute_form_score_history()` (Polars rolling avg_14 - avg_90), `load_full_performance_history()` (DB query), `plot_form_score_history()` (Plotly multi-lignes + fill). Intégré en tête de l'onglet Résumé (Timeseries) et avant "Taux de victoires vs historique" (Teammates). st.metric + graphe historique avec points session surlignés. |
| 2026-04-06 | **Discord i18n — assets résolus par ID dans l'embed** : `fetch_last_match_info()` remonte `map_id`/`playlist_id`/`pair_id`/`game_variant_id` + libellés EN bruts ; `src/utils/_discord_embed.py` résout désormais les traductions via `asset_translations` selon `discord_lang`, avec fallback unique vers l'anglais en BDD. Les colonnes `*_fr` de `v_match_full` ne sont plus utilisées dans ce flux. Tests ciblés : 138 passés (`test_discord_notifier.py`, `test_translations.py`, `test_delta_sync.py`). |
| 2026-03-30 | **i18n — Table `asset_translations` peuplée dans `metadata.duckdb`** : 9 674 traductions (698 assets × 14 langues BCP-47). Script `populate_asset_translations.py` réécrit avec `_build_version_id_cache()` (version_id SPNKr requis, `""` → 404), parallélisme `asyncio.gather` sur les 14 langues, reprise possible. |
| 2026-03-30 | **Fix critique — `v_match_full` sans traductions en prod** : `_try_attach_meta_for_views()` cherchait `meta.maps` (table absente en v6) → toujours `None` → vue créée sans JOINs i18n. Fix : vérifier `meta.asset_translations`. `_create_v_match_full()` : suppression des 4 JOINs legacy (`meta.maps/playlists/playlist_map_mode_pairs/game_variants`), 8 JOINs `asset_translations` (en-US + fr-FR × 4 types). Vue recréée en prod : "Starboard"→"Tribord", "The Pit"→"La fosse", etc. |
| 2026-03-30 | **Docs — Renommage ARCHITECTURE_V5 → V6** : `git mv` + mise à jour contenu (titre, version 6.3.0, `shared_matches_v2.duckdb`). §6 asset_translations ajouté dans la version FR. Toutes les références mises à jour : `CLAUDE.md`, `README.md`, `README_FR.md`, `FR/README.md`, `FR/COMMANDS.md`, `.ai/project_map.md`, `.ai/START_HERE.md`. |
| 2026-03-30 | **Docs — CHANGELOG 6.3.0** : entrées EN + FR documentant `asset_translations`, refonte `v_match_full` v6, fix `_try_attach_meta_for_views`. |
| 2026-03-30 | **Normalisation des labels de modes de jeu (v6.2.1)** : `resolve_display_mode()` dans `src/analysis/mode_display.py`, colonne `canonical_category` dans `mode_prefix_names`, 29 overrides dans `mode_pair_overrides`, `translate_pair_name` délégue au resolver, fichier plat de contrôle généré et validé. |
| 2026-03-30 | **Audit KDA locaux → `efficiency` (v6.2.1)** : sémantiques séparées — `p.kda` API conservé per-match, agrégats session/carte/cumul renommés `efficiency`/`session_efficiency` ; clés i18n `efficiency`/`efficacité` ajoutées ; 6 modules `src/analysis/` mis à jour (`cumulative.py`, `stats.py`, `_performance_relative.py`, `_performance_relative_helpers.py`, `_performance_session.py`, `stats.py` domain model). |
| 2026-03-27 | **Bug — `index_media.py --force` levait `ConstraintError: Duplicate key`** : quand `force_rescan=True`, `existing` était laissé vide `{}` → toutes les entrées considérées "nouvelles" → INSERT sur des clés déjà présentes. Fix : `existing` est toujours chargé depuis la DB ; `force_rescan` contourne uniquement le filtre delta `mtime`. Ré-indexation JGtm (73 médias) exécutée avec succès après fix. |
| 2026-03-26 | **Bug critique — `mv_player_matches` recalcule le KDA au lieu de lire la valeur API** : vue recréait `(kills + assists/3)/deaths` au lieu de `COALESCE(p.kda, fallback)`. Fix : détection dynamique `has_kda_col` (même pattern `has_enemy_mmr`) + génération SQL conditionnelle. |
| 2026-03-26 | **UX — Score d'équipe supérieur aux scores individuels (En-tête Page Coéquipiers)** : carte équipe n'affichait pas les bonus collectifs. Fix : `_render_compact_team_card` calcule `bonus = score - base_avg` et affiche `"moy. X (+Y collectif)"` quand > 0. |
| 2026-03-26 | **Bug — Colonne "Dernière rencontre" incohérente (Page Match · Encounters)** : SQL `MAX(start_time)` incluait le match courant et les matchs futurs. Fix : `filter_past` CTE + `_fetch_match_start_time` helper + guard `days = max(0, delta.days)` + colonne renommée "Précédente rencontre" + "1ère rencontre" pour les nouvelles têtes. |
| 2026-03-26 | **Bug annexe — `datetime.utcnow()` déprécié dans `career_lusr.py`** : remplacé par `datetime.now(timezone.utc).replace(tzinfo=None)`. |
| 2026-03-26 | **Bug — Médias mal rattachés aux matchs (décalage fuseau horaire)** : `epoch(capture_end_utc)` → `epoch(timezone('UTC', capture_end_utc))` dans `associate_with_matches()` + EXIF naïf ignoré (heure locale caméra, pas UTC). Ré-indexation requise (faite pour JGtm le 2026-03-27). |
| 2026-03-26 | **Bug RÉCURRENT CRITIQUE — Session escouade absente du graphe "Évolution de la performance"** : root cause A (fanout ouvrait shared en R/W → conflit handle Streamlit) fixée via Phase J (`shared_read_only=True` dans `_engine_fanout.py`). Fix défensif LEFT JOIN dans `_performance_squad._join_perf_frames()`. Les deux chemins de fix documentés dans l'audit sont implémentés. |
| 2026-03-26 | **Bug — Stats coéquipiers absentes (Page Teammates)** : résolu par le fix fanout R/O (Phase J). La root cause était identique au bug session escouade — fanout silencieux → PME coéquipier non créées. À revalider sur la prochaine session de jeu. |
| 2026-03-26 | **Bug annexe — `get_sync_metadata` lit mauvaise DB** : `SELECT last_sync_at FROM meta.sync_meta WHERE xuid=?` → `SELECT value FROM sync_meta WHERE key='last_sync_at'` dans la player DB. Fix commité dans `_diagnostic_repo.py` (Phase F). |
| 2026-03-26 | **Piste — Crashes silencieux (Page Coéquipiers · Top medals)** : source principale (connexions zombies fanout R/W) supprimée par Phase J. Si non récurrent → archivé. |
| 2026-03-21 | **Bug — Frags vs. détail armes (double-comptage melee)** : melee kills filmés attribués à l'arme tenue + `melee_kills` API → double-comptage. Fix : remainder `api_total - film_kills` dans 3 fichiers + `load_total_kills_for_player()` + 2 nouveaux tests. |
| 2026-03-21 | **UI — Graphe stats/min escouade : morts sous l'axe** — `plot_per_minute_timeseries` : deaths tracées en négatif (`dpm_neg`), `customdata[5]` = valeur absolue, `hover_dpm_neg` i18n, ticks Y absolus via `build_symmetric_abs_ticks` (extrait dans `src/visualization/_permin_helpers.py`). `timeseries.py` à exactement 500L. |
| 2026-03-21 | **Maintenance — Nettoyage dossier `scripts/`** — 10 scripts investigation → `scripts/investigation/` + README ; `cleanup_legacy_tables.py` + `cleanup_player_dbs_v5.py` → `scripts/_archive/` ; `.tmp.*` supprimés. |
| 2026-03-21 | **CI — Scripts exclus par `.gitignore`** — `check_code_size.py` → `enforce_size_limits.py` ; `check_imports.py` → `validate_imports.py` ; stubs `test_page_router_smoke.py` + `test_page_router_regressions.py` créés. Références mises à jour dans `ci.yml`, `.pre-commit-config.yaml`, `test_code_quality.py`. |
| 2026-03-21 | **UI — Notation de session escouade (Page Coéquipiers)** — `compute_squad_performance_score()` dans `src/analysis/_performance_squad.py` ; `SQUAD_GRADE_THRESHOLDS` + `resolve_squad_grade()` dans `performance_config.py` ; `render_squad_session_header()` + `_render_squad_score_block()` dans `src/ui/components/performance.py` ; 7 clés i18n `squad_grade_*` dans `src/ui/i18n/pages/teammates.py` ; bloc tendance K/D remplacé dans `teammates.py` ; 18 tests unitaires. |
| 2026-03-21 | **Perf — `_MAX_CONCURRENT_CHUNKS`** : déjà à 50 en production (`weapon_extraction_service.py`). Tâche obsolète — objectif déjà atteint. |
| 2026-03-19 | **Medal definitions en BDD** — table `medal_definitions` dans `metadata.duckdb` (167 médailles, DB-first + JSON-fallback). Migration, script population, CLI `--medal-metadata`, `MedalsMixin.load_medal_definitions()` / `get_medal_label()`, UI DB-first dans `medals.py`, 16 tests unitaires + 4 intégration. Orphan `citations_{fr,en}.json` supprimés. |
| 2026-03-19 | **Phase 8 — Couche centralisée médailles** (`medal_definitions.py`) — `src/data/medal_definitions.py` source canonique unique ; `_medal_data.py` thin re-export ; `medals.py` wrapper `@st.cache_data` délégant ; `_medals_repo.py` délègue. 3 chemins DB indépendants → 1. Fallbacks JSON applicatifs supprimés de `medals.py`. JSON `static/medals/*.json` conservés (source pour `populate_medal_metadata.py`). 51 tests passent. Commit `88d5cf0`. |
| 2026-03-19 | **Migration `b5>>4`** — `scan_fire_events_b5` implémenté, `fire_seq%n_players` supprimé, `map_b2_to_player`/`group_events_by_pi`/`POV_PLAYER_INDEX` retirés, 25 nouveaux tests — 4968 tests passent. Relancer `--force-weapons --all` pour re-extraire. |
| 2026-03-19 | **Backfill enrichissement** JGtm + Madina97294 — 8 matchs du 18 mars rattrapés (performance_score, sessions, citations) |
| 2026-03-19 | **Fix 11 — Fan-out multi-joueurs** : `FanoutEnrichmentMixin` (`_engine_fanout.py`) + branchement dans `engine.py` après `_detach_shared_from_player_conn()`. Résout le manquement d'enrichissement local pour les joueurs qui ne sync pas eux-mêmes. |
| 2026-03-19 | **Fix 10 — Performance vs historique** : `performance_score` ajouté à `COLUMNS_COMMON` + JOIN `player_match_enrichment` dans `load_matches_as_polars` + `df_history` propagé dans `WinLossService` |
| 2026-03-19 | **Fix 9 — Radar escouade** : `radar_squad_ids` sauvegardé avant filtre UI ; DFs historiques séparés (`radar_me_df/f1/f2/f3`) passés à `render_trio_synergy_radar` |
| 2026-03-19 | **Fix 8 — Heatmap monochrome** : `compute_map_breakdown` lit `performance_score` depuis la colonne quand présente (fallback percentile supprimé pour les joueurs enrichis) |
| 2026-03-19 | **Fix 7 — Performance vue 1 coéquipier** : `enrich_with_performance_score` appelé pour `me_df` et `friend_df` dans `render_single_teammate_view` |
| 2026-03-19 | **Fix 6 — MediaFileStorageError icônes rang** : images rang converties en data URI base64 dans `career.py` (IDs Streamlit éphémères éliminés) |
| 2026-03-19 | **Fix 5 — Joueurs fantômes** : `_is_ghost_player` requiert la présence des clés stat + filtre appliqué uniquement dans `filter_encounter_xuids` (scoreboard non filtré — joueurs légitimes à 0 stats conservés) |
| 2026-03-19 | **Fix 4 — ratio=kda** : `ratio = pl.col("kda").alias("ratio")` dans `_finalize_polars_df` + `p.kda AS ratio` dans `_query_teammate_shared_stats` — source unique API, plus de recalcul |
| 2026-03-19 | **Fix 3 — Matrice d'impact** : `.unique(maintain_order=True)` dans `friends_impact_heatmap.py` |
| 2026-03-19 | **Fix 2 — Bots bid(33.0)** : `get_bot_name()` appelé dans `_build_encounter_rows` avant le fallback `xuid[:8]` |
| 2026-03-19 | **Fix 1 — ColumnNotFoundError map_name** : `mr.map_name` ajouté au SELECT de `load_friend_match_details` + `_FRIEND_DF_EMPTY_SCHEMA` mis à jour |
| 2026-03-19 | **Bonus — `resolve_weapon_display` fusion avant DB** : la fusion map est appliquée (étape 0) avant le lookup `weapon_labels`, évitant que M392 Bandit / Fuel Rod SPNKr contournent leur regroupement canonique |
| 2026-03-16 | Audit post-V6 : `weapon_kills` bit sync + logging, `v_gamertag_lookup` systématique, `shared_matches_v2.duckdb` production, LEGACY SyncScope supprimés, 17 nouveaux tests — 4799 tests passent |
| 2026-03-16 | Sprint refactor : splits fonctions/modules >80/500L, `_teammates_trio_helpers`, `_match_relations`, `_roster_loader` helpers, `render_trio_charts` DRY |
| 2026-03-15 | Phase 3 v6 : migration complète `duckdb_read_only` UI → repo — 7 fichiers migrés, 17 tests + 9 tests antagonistes, 4764 tests passent |
| 2026-03-15 | Phase 2 v6 : `career`, `career_lusr`, `explorer` migrés + `CareerMixin` créé |
| 2026-03-15 | Migration last_match : requêtes directes → DuckDBRepository (`load_player_match_enrichment`, `is_abandoned_match`) — 12 tests |
| 2026-03-15 | Fixes Phase 1 v6 : `player_provisioning.py` bare connect, `cache_filters.py` `_get_connection()` privé, `multiplayer.py` dead code — 6 tests |
| 2026-03-15 | Couche résolution gamertag→XUID : `lookup_xuid_for_gamertag()` dans `src/utils/xuid.py` + `GamertagResolverMixin` — 9 fichiers migrés, 11 tests |
| 2026-03-15 | v5.8 Wave 5 : nettoyage i18n playlists/modes obsolètes → `metadata.duckdb` |
| 2026-03-15 | v5.8 Wave 4 : suppression `highlight_events.gamertag` + helper `resolve_medal_name` |
| 2026-03-15 | v5.8 Wave 3 : nettoyage wrappers XUID + dead code outcomes → `Outcome` enum |
| 2026-03-15 | v5.8 Wave 2 : migration consommateurs directs (gamertags, KV pairs, assets) |
| 2026-03-15 | v5.8 Wave 1 : vues SQL `v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full` + `GamertagResolverMixin` |
| 2026-03-15 | Fix weapon-parser : corrélation globale — taux `fire_event` 15% → 95% |
| 2026-03-15 | Navigation last_match : boutons ◀/▶ entre matchs filtrés |
| 2026-03-13 | Couverture tests `migrations.py` (lacunes v5.5–v5.7) |
| 2026-03-13 | Conflit `shared_matches.duckdb` — sync depuis UI Streamlit |
| 2026-03-13 | [UI] Heatmap performance par joueur × carte — Page Teammates |
| 2026-03-13 | [UI] Performance par carte vs historique — vues escouade et joueur |
| 2026-03-08 | Bug #0 : match invisible post-sync — suppression `_filters_loaded_*` dans `_clear_app_caches()` |
| 2026-03-08 | Perf UI — vues matérialisées lazy, pagination SQL, projections fines, `@fragment_if_available` |
| 2026-03-28 | [v6.2] Badges Remontada / Débandade / Contre-Remontada — `DominanceFlag` 3-5, `comeback_analysis.py`, `comeback_backfill.py`, `--comeback-badges` CLI |
| 2026-03-28 | [v6.2] Unification vue coéquipier unique → vue escouade — `f2_xuid` optionnel, suppression `render_single_teammate_view` |
| 2026-03-28 | [v6.2] Graphe combiné Frags↑/Morts↓ — `plot_trio_kills_deaths()`, axe Y symétrique, `safe_chart_render()` |

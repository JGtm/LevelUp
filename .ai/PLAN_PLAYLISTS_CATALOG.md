# Plan — Catalogue global Playlists / Pairs / Maps

> Analyse réalisée le 2026-04-30. Conception d'un référentiel global Halo Infinite
> pour accélérer la cascade de filtres et écrémer l'UI aux options réellement utiles.
> Ce document couvre uniquement le design data + sync — pas l'UI React (cible d'un sprint suivant).

---

## 1. Contexte

Le système de filtres actuel ([FilterOmnibar.tsx](apps/web/src/components/shell/FilterOmnibar.tsx) → [filters_service.go](apps/go-api/internal/service/filters_service.go)) recharge **tous les matchs du joueur** à chaque toggle de checkbox via [filters_repo.go](apps/go-api/internal/platform/duckdb/filters_repo.go) `LoadMatchesForFilters()`, puis recalcule la cascade Expérience → Playlists → Modes → Maps en mémoire Go. Aucun cache de la hiérarchie n'existe — le `staleTime: 5min` côté TanStack Query ne suffit pas car chaque interaction utilisateur invalide la requête.

Sur un gros historique, le scan de `match_participants` jointé à `match_registry` à chaque toggle devient le principal goulot perçu côté UI. La latence n'est pas dramatique (200-500ms typique), mais elle empêche le sentiment d'instantanéité attendu sur un widget de filtre.

### Insight clé

Les playlists Halo Infinite sont **au niveau du jeu**, pas au niveau du joueur. Sources :

- [Halo Waypoint — Multiplayer Playlists](https://support.halowaypoint.com/hc/en-us/articles/17920041655188-Halo-Infinite-Multiplayer-Playlists) : liste des ~25-30 playlists permanentes (Ranked Arena, Quick Play, BTB, Firefight, Action Sack, etc.) + rotations hebdomadaires Ranked
- [den.dev — Halo Infinite Playlist Weights](https://den.dev/blog/halo-infinite-playlist-weights/) : structure API `discovery-infiniteugc.svc.halowaypoint.com/hi/playlists/{asset_id}/versions/{version_id}` avec `CustomData.PlaylistEntries[].MapModePairAssetId` + `Metadata.Weight`

La hiérarchie est donc **stable, énumérable, partagée entre tous les joueurs**. Elle évolue à la cadence des saisons (~3 mois) plus quelques rotations Ranked hebdomadaires. C'est exactement le profil d'un référentiel — pas d'une mv par joueur.

---

## 2. Audit de l'existant

| Composant | État | Référence |
|---|---|---|
| `shared.match_registry` | Stocke déjà `playlist_id`, `pair_id`, `map_id`, `game_variant_id` (UUID Asset) + noms PublicName + i18n FR | [schema.go:97-129](apps/go-api/internal/sync/schema.go#L97-L129) |
| `version_id` par asset | **Absent** — ni dans `match_registry`, ni extrait par les transforms | [transforms.go:114-122](apps/go-api/internal/sync/transforms.go#L114-L122) |
| Provider DiscoveryUGC Go | Existe avec retry exponentiel (4 tentatives, base 800ms × 2) | [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) |
| Types `AssetTypePlaylist` / `AssetTypePair` | Définis et fonctionnels, jamais appelés depuis `internal/sync/` | [discovery_types.go](apps/go-api/internal/platform/halo/discovery_types.go) |
| Cache générique d'assets bruts | `metadata.waypoint_assets_raw(title_id, asset_id, version_id, raw_json, content_hash)` | [steps_metadata.go:170-189](apps/go-api/internal/migration/steps_metadata.go#L170-L189) |
| Traductions FR/EN d'assets | `metadata.asset_translations(asset_id, asset_type, lang, name)` | [steps_metadata.go:20-28](apps/go-api/internal/migration/steps_metadata.go#L20-L28) |
| Notion `experience` | Dérivée heuristiquement à chaque requête depuis `is_ranked` + `is_firefight` (3 buckets : `PVP non classé`, `PVP classé`, `PVE`) | [filters_service.go:18-19](apps/go-api/internal/service/filters_service.go#L18-L19) |
| Tables dédiées catalogue | **Aucune** — pas de `playlists_catalog`, `playlist_pair_links`, ni `map_mode_pair_definitions` | — |

### Conclusion d'audit

80% du terrain est préparé : asset IDs déjà stockés en match, provider DiscoveryUGC déjà opérationnel, cache générique disponible, point d'injection sync clairement identifiable. Il reste à modéliser la **relation** entre playlists et pairs, persister `version_id`, et ajouter une stratégie de refresh.

---

## 3. Stratégie cible

### 3.1 Catalogue global dans `metadata.duckdb`

Source de vérité du catalogue Halo, **partagée entre tous les joueurs**. Quatre tables (cf. §4 pour le schéma SQL).

### 3.2 Stratégie de refresh — pas de worker par sync

Les playlists changent rarement. Trois mécanismes complémentaires, du plus paresseux au plus actif :

1. **Lazy detection au sync (zéro fetch immédiat)** — Lors de l'ingestion d'un match, si `playlist_id` ou `pair_id` est absent du catalogue, on enqueue dans `catalog_fetch_queue` sans bloquer l'ingestion ni déclencher d'appel HTTP. Coût : un INSERT OR IGNORE.
2. **Bootstrap initial (one-shot CLI)** — Commande `populate-playlists-catalog` qui (a) seed la queue depuis `SELECT DISTINCT playlist_id, pair_id FROM shared.match_registry`, (b) drain la queue en appelant DiscoveryUGC, (c) persiste dans le catalogue. Couvre 100% de ce que les joueurs ont déjà vu sans liste manuelle à maintenir.
3. **Refresh mensuel (cron / job planifié)** — Une fois par mois, drain la queue accumulée par la détection lazy + re-fetch les `playlists_catalog` où `is_active = true` pour détecter les changements de `version_id` (rotations Ranked, mises à jour de weights). Marquer `is_active = false` les playlists non vues depuis N matchs / X mois (à calibrer).

**Pourquoi pas un worker à chaque sync ?**
- Les rotations Ranked changent au pire toutes les semaines, pas plus
- La très grande majorité des sync delta ne rencontrent **aucun nouvel asset_id** (le joueur rejoue les mêmes playlists)
- L'API DiscoveryUGC a un rate limiter — multiplier les appels par sync est un mauvais investissement
- La latence d'enqueue dans une table DuckDB est ~µs ; la latence d'un round-trip HTTP est ~100ms. Pas de débat.

### 3.3 Écrémage UX — réponse à « garder l'utile »

Avec le catalogue global en place, la requête de filtres devient un `LEFT JOIN catalogue ↔ matchs_du_joueur` qui permet trois modes d'affichage :

| Mode | Comportement | Cas d'usage |
|---|---|---|
| **Joué** (défaut) | `WHERE match_count > 0` — ne montre que les playlists/maps/modes touchés | 95% des interactions ; menu réduit de ~80 à ~10-15 options sur un joueur typique |
| **Tous** (toggle) | Catalogue complet, options grisées avec `match_count = 0` | Découverte, comparaison entre joueurs |
| **Compteurs visibles** | `Ranked Slayer (24)` à côté de chaque option | Guide visuel pour prioriser les filtres riches en données |

C'est le vrai gain UX, plus important que la perf brute.

---

## 4. Schéma proposé (`metadata.duckdb`)

### 4.1 Insight cardinalités — citation Halo Waypoint

> « The same game mode and map combination can belong to more than one playlist. Every map can be used in many map-mode pairs, and every mode can, just like the maps, belong to more than one combo. »

Trois relations N-N à modéliser **proprement**, pas une seule :

| Relation | Cardinalité | Conséquence |
|---|---|---|
| `pair` ↔ `playlist` | N-N | Table d'association `playlist_pair_links` |
| `map` ↔ `pair` | 1-N | Une map référencée par plusieurs pairs → **table `maps_catalog` séparée**, sinon duplication massive de `map_name` / `map_image_url` |
| `game_variant` ↔ `pair` | 1-N | Idem pour les variants → **table `game_variants_catalog` séparée** |

→ Le pair ne porte que les **FK** vers map et game_variant, plus son nom propre (« Slayer on Aquarius » par exemple). Un changement de nom de map ou de mode se propage gratuitement à tous les pairs qui le référencent.

### 4.2 Tables (6 au total, toutes title-aware via `title_slug`)

```sql
-- Référentiel des playlists, par titre
CREATE TABLE playlists_catalog (
  title_slug          VARCHAR,
  playlist_asset_id   UUID,
  current_version_id  UUID,
  name_en             VARCHAR,
  name_fr             VARCHAR,
  experience          VARCHAR,        -- 'ranked' | 'social' | 'btb' | 'firefight' | 'action_sack' | 'limited_time' | 'custom_browser' | 'unknown'
  is_ranked           BOOLEAN,
  is_active           BOOLEAN DEFAULT TRUE,
  first_seen_at       TIMESTAMP,
  last_seen_at        TIMESTAMP,
  last_fetched_at     TIMESTAMP,
  PRIMARY KEY (title_slug, playlist_asset_id)
);

-- Référentiel des maps (séparé pour ne pas dupliquer name/image)
CREATE TABLE maps_catalog (
  title_slug          VARCHAR,
  map_asset_id        UUID,
  current_version_id  UUID,
  name_en             VARCHAR,
  name_fr             VARCHAR,
  image_url           VARCHAR,        -- déjà résolu via internal/assets si dispo
  last_fetched_at     TIMESTAMP,
  PRIMARY KEY (title_slug, map_asset_id)
);

-- Référentiel des game variants (séparé : un variant peut être dans N pairs)
CREATE TABLE game_variants_catalog (
  title_slug              VARCHAR,
  game_variant_asset_id   UUID,
  current_version_id      UUID,
  name_en                 VARCHAR,
  name_fr                 VARCHAR,
  mode_canonical          VARCHAR,    -- 'slayer' | 'ctf' | 'oddball' | 'koth' | 'strongholds' | 'extraction' | 'firefight_kotr' | 'fiesta' | ...
  game_variant_category   INTEGER,    -- code numérique Halo (équivalent GameVariantCategory)
  last_fetched_at         TIMESTAMP,
  PRIMARY KEY (title_slug, game_variant_asset_id)
);

-- Pair = jonction map + game_variant + nom composite, par titre
CREATE TABLE map_mode_pair_definitions (
  title_slug             VARCHAR,
  pair_asset_id          UUID,
  current_version_id     UUID,
  name_en                VARCHAR,     -- ex: "Arena:Slayer on Bazaar" (brut DiscoveryUGC)
  name_fr                VARCHAR,
  map_asset_id           UUID,        -- FK -> maps_catalog
  game_variant_asset_id  UUID,        -- FK -> game_variants_catalog
  -- Dérivations stockées au moment du fetch (cf. §4ter)
  mode_label_en          VARCHAR,     -- sortie NormalizeModeLabel(name_en, map_labels)
  mode_label_fr          VARCHAR,     -- sortie NormalizeModeLabel(name_fr, map_labels)
  mode_category          VARCHAR,     -- sortie InferModeCategoryFromPairName(name_en) — enum Go: 'Assassin' | 'Fiesta' | 'SuperFiesta' | 'HuskyRaid' | 'BTB' | 'Ranked' | 'Firefight' | 'Other'
  last_fetched_at        TIMESTAMP,
  PRIMARY KEY (title_slug, pair_asset_id)
);

-- Relation N-N playlist <-> pair, avec poids de tirage
CREATE TABLE playlist_pair_links (
  title_slug         VARCHAR,
  playlist_asset_id  UUID,
  pair_asset_id      UUID,
  weight             DOUBLE,           -- depuis CustomData.PlaylistEntries[].Metadata.Weight
  PRIMARY KEY (title_slug, playlist_asset_id, pair_asset_id)
);

-- File d'attente du fetcher (pattern Kinds, drain mensuel)
CREATE TABLE catalog_fetch_queue (
  title_slug    VARCHAR,
  asset_type    VARCHAR,             -- 'playlist' | 'pair' | 'map' | 'game_variant'
  asset_id      UUID,
  version_id    UUID,                -- nullable si on ne connaît pas encore
  enqueued_at   TIMESTAMP,
  attempts      INTEGER DEFAULT 0,
  last_error    VARCHAR,
  PRIMARY KEY (title_slug, asset_type, asset_id)
);
```

### 4.3 Décisions de modélisation

- **6 tables au lieu de 4** : la séparation maps / game_variants / pairs est imposée par les cardinalités N-N. Sans ça, la moindre correction de nom de map nécessiterait un UPDATE multi-lignes et on perdrait toute l'info image au passage.
- **`title_slug` PK composite partout** : aligné sur le pattern `waypoint_assets_raw(title_id, asset_id, version_id)` documenté dans le `project_map.md`. Permet à l'adapter `synthetic_title_b/` (corpus de tests) de cohabiter sans pollution croisée.
- **`experience` en VARCHAR + enum applicatif** : pas de table de référence séparée. La liste reste petite et stable par titre, et l'enum vit dans `internal/games/canonical/` pour la validation au compile-time. La règle de classification est portée par le `TitleCatalogAdapter` (cf. §5bis).
- **`weight` est conservé** : pas critique pour le filtre, mais ouvre la porte à une future feature « probabilité de tomber sur cette map dans Quick Play ».
- **`current_version_id` au lieu d'une table d'historique** : on ne garde que la version active. Si un audit historique est jamais demandé, `waypoint_assets_raw` garde déjà les snapshots bruts datés.
- **Pas de FK déclarées en DuckDB** : DuckDB supporte les FK mais la pratique du repo (cf. autres tables metadata) est de ne pas les imposer pour éviter les blocages au sync. La cohérence est garantie par le `CatalogFetcherService` (un fetch playlist hydrate aussi ses pairs / maps / variants).
- **`maps_catalog.image_url` peuplé dès Phase F** (décision actée 2026-04-30) : au moment du fetch catalogue, on appelle `assetResolver.Resolve(KindMapImage, map_asset_id)` qui retourne l'URL interne `/api/v1/assets/maps/...`. Cohérent avec le pattern déjà en place pour les assets Spartan ([home_repo.go](apps/go-api/internal/platform/duckdb/home_repo.go)). Si GameCMS est down → URL NULL, retry au refresh mensuel, frontend gère le fallback.

---

## 4bis. Intégration multi-titre

Le repo a déjà une couche multi-titre Go en place (cf. `project_map.md` §« Multi-titres — couche canonical + adapters + TOML mappings »). Le catalogue doit s'y aligner pour ne pas créer un silo Halo Infinite parallèle.

### 4bis.1 Pattern existant à réutiliser

| Composant | Rôle | Référence |
|---|---|---|
| `internal/games/{adapter,resolver}.go` | Interfaces `TitleDataAdapter` + `TitleSemanticAdapter` (SRP), `StaticResolver` injecté au boot | déjà en prod |
| `internal/games/halo_infinite/` | Implémentation Halo : `DataAdapter` (wrap `CareerSource`), `SemanticAdapter` (wrap `FieldMappingSet`) | déjà en prod |
| `internal/games/synthetic_title_b/` | Corpus synthétique pour tests d'isolation cross-titres | déjà en prod |
| `config/titles/{slug}/mappings/fields.toml` | Sources de vérité versionnées Git | déjà en prod |
| `MULTI_TITLE_API_ENABLED` | Feature flag qui gate les endpoints `/api/v1/titles/{slug}/...` | déjà en prod |
| `tools/mappings/CHANGELOG.md` | Historique des bumps de `schema_version` | déjà en prod |

### 4bis.2 Nouvelle interface à introduire

Une troisième interface, sœur des deux existantes, dans `internal/games/catalog_adapter.go` :

```go
type TitleCatalogAdapter interface {
    // Fetch une playlist depuis l'API du titre et retourne sa définition canonique.
    FetchPlaylist(ctx context.Context, assetID, versionID string) (CanonicalPlaylist, error)

    // Fetch un pair (map+mode) depuis l'API du titre.
    FetchPair(ctx context.Context, assetID, versionID string) (CanonicalPair, error)

    // Classifie l'experience d'une playlist (ranked/social/btb/firefight/...)
    // depuis ses attributs natifs (nom, tags, GameVariantCategory, etc.).
    ClassifyExperience(playlist CanonicalPlaylist) Experience

    // Mappe un game_variant_asset_id vers un mode canonique ('slayer', 'ctf', ...).
    CanonicalMode(variant CanonicalGameVariant) ModeCanonical
}
```

### 4bis.3 Implémentations

- **`internal/games/halo_infinite/catalog_adapter.go`** : enveloppe le [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) existant, parse `CustomData.PlaylistEntries`, applique les heuristiques de classification (préfixes `Ranked:`, `BTB:`, etc. — porter la logique de `src/analysis/playlist_groups.py` qui est déjà éprouvée).
- **`internal/games/synthetic_title_b/catalog_adapter.go`** : implémentation no-op ou fixtures, suit le pattern du corpus de tests.

### 4bis.4 Configuration TOML versionnée

Cohérent avec `config/titles/halo_infinite/mappings/fields.toml`, ajouter :

```
config/titles/halo_infinite/catalog/
├── experience_rules.toml      # Règles de classification ranked/social/btb/firefight
└── mode_canonical_map.toml    # Mapping game_variant_category INT -> mode_canonical
```

`experience_rules.toml` (extrait) :

```toml
schema_version = 1

[[rule]]
experience = "ranked"
match_any = { name_prefix = ["Ranked "], tag = ["Ranked"] }

[[rule]]
experience = "btb"
match_any = { name_contains = ["Big Team", "BTB"], pair_prefix = ["BTB:", "BigTeam:"] }

[[rule]]
experience = "firefight"
match_any = { game_variant_category = [22, 32, 40, 41, 42] }
```

Cette config est chargée par `halo_infinite/CatalogAdapter` au boot. Un changement de classification = un commit TOML + bump `schema_version`, pas de redéploiement de code.

### 4bis.5 Endpoint REST

Symétrie avec `/api/v1/titles/{slug}/field-mappings` :

```
GET /api/v1/titles/{slug}/catalog/playlists?xuid={xuid}&only_played={bool}
GET /api/v1/titles/{slug}/catalog/pairs?playlist_asset_id={uuid}&xuid={xuid}&only_played={bool}
GET /api/v1/titles/{slug}/catalog/maps?xuid={xuid}&only_played={bool}
```

Gated par `MULTI_TITLE_API_ENABLED=true`. Conserve l'endpoint legacy `POST /players/{slug}/filters/resolve` qui consomme le catalogue en interne sans changer son contrat (Phase G).

### 4bis.6 Service Go

```go
type CatalogService struct {
    repo     CatalogRepo                   // accès DuckDB metadata
    resolver games.Resolver                // récupère TitleCatalogAdapter par slug
    fetcher  *CatalogFetcherService        // drain de la queue
}
```

Le `CatalogService` n'a **aucune connaissance Halo** : il délègue tout au `TitleCatalogAdapter` résolu via le `StaticResolver`. C'est la même séparation que `MultiTitlePreviewHandler`.

---

## 4ter. Articulation avec la normalisation des modes existante

La feature de normalisation (skill `halo-modes`) **survit en intégralité** comme couche complémentaire. Elle n'est ni remplacée ni dépréciée par le catalogue — elle est **consommée à l'hydratation** et ses sorties sont stockées comme colonnes du catalogue.

### 4ter.1 Trois niveaux orthogonaux à ne pas confondre

| Niveau | Granularité | Source | Usage filtre |
|---|---|---|---|
| `experience` (playlist-level) | ~6-8 buckets | TOML `experience_rules.toml` (Phase D) | « je veux que du Ranked » |
| `mode_category` (pair-level) | ~7-8 buckets enum Go | [mode_category.go](apps/go-api/internal/analysis/mode_category.go) `InferModeCategoryFromPairName()` | « je veux que du Slayer-like » |
| `mode_label` (pair-level) | ~30-50 valeurs | [mode_label.go](apps/go-api/internal/analysis/mode_label.go) `NormalizeModeLabel()` | « je veux uniquement du Tactical Slayer » |

`experience` (playlist) et `mode_category` (pair) sont **deux dimensions parallèles** dans le filtre, pas hiérarchiques. L'utilisateur peut combiner « Ranked **et** Assassin », ou « Social **et** Fiesta ». Le `mode_label` est un sous-niveau optionnel de `mode_category`.

### 4ter.2 Qui produit quoi

```
                                                      ┌─────────────────────────────────────┐
   DiscoveryUGC API                                   │  internal/analysis/  (pure Go)       │
   (asset brut + name_en)                             │  - NormalizeModeLabel()              │
            │                                         │  - InferModeCategoryFromPairName()   │
            ▼                                         │                                      │
   ┌────────────────────┐                            └──────────────┬──────────────────────┘
   │ TitleCatalogAdapter│                                           │
   │ (Halo Infinite)    │   appelle au fetch                        │
   └────────┬───────────┘  ─────────────────────────────────────────┘
            │
            │ persiste les sorties dans :
            ▼
   map_mode_pair_definitions.{mode_label_en, mode_label_fr, mode_category}
   playlists_catalog.experience  (depuis TOML, autre dimension)
```

### 4ter.3 Tables existantes : input upstream, pas output runtime

- `metadata.mode_name_tr` (traductions EN→FR des sous-modes) → consommée par `NormalizeModeLabel()` au moment du fetch pour produire `mode_label_fr`. Reste utile.
- `metadata.mode_pair_overrides` (surcharges des paires aux noms cassés DiscoveryUGC) → consommée par l'adapter Halo pour patcher `name_en` / `name_fr` avant l'appel à `NormalizeModeLabel()`. Reste utile.
- `metadata.mode_prefix_names` / `mode_lang_settings` → potentiellement consommées par les helpers existants. À conserver comme input.

**Aucune dépréciation prévue.** Ces tables servent désormais d'input à un pipeline qui les fige dans le catalogue, plutôt que d'être interrogées à chaque requête utilisateur.

### 4ter.4 Distinction entre `mode_canonical` et `mode_category`

- `game_variants_catalog.mode_canonical` = **fait technique stable** au niveau du game variant atomique (`'slayer'`, `'ctf'`, `'firefight_kotr'`). Dérivé de `game_variant_category` numérique + heuristique TOML. Sert à des aggrégations métier stables, pas à l'UI.
- `map_mode_pair_definitions.mode_category` = **catégorie UX au niveau du pair** (`'Assassin'`, `'Fiesta'`, `'SuperFiesta'`, etc.). Sert au filtre et au regroupement visuel.

Un pair `Tactical:Slayer on Bazaar` a :
- `mode_label_en = "Slayer"`, `mode_label_fr = "Assassin"` (sortie de `NormalizeModeLabel`)
- `mode_category = "Assassin"` (sortie de `InferModeCategoryFromPairName`)
- `game_variants_catalog.mode_canonical = "slayer"` (fait technique du variant lié)

Trois représentations différentes, complémentaires.

### 4ter.5 Cardinalité réelle à observer

Une question reste ouverte : faut-il exposer `mode_label` comme dimension de filtre, ou le laisser comme « expand » optionnel d'une catégorie ? Cela dépend de :
- Combien de `mode_label` distincts par `mode_category` en pratique
- Distribution des matchs entre catégories (équilibrée ou Pareto-skewed ?)
- Couverture de la catégorie `Other` (si > 10% → dette de `_PREFIX_RULES` à boucher avant)

→ **Audit cardinalité prévu en §4quater** (à compléter après mesure sur la DB de production).

---

## 5. Plan d'implémentation par phases

### Phase A — Migration schéma metadata (1 commit)

- Ajouter une migration dans [steps_metadata.go](apps/go-api/internal/migration/steps_metadata.go) qui crée les **6 tables** title-aware : `playlists_catalog`, `maps_catalog`, `game_variants_catalog`, `map_mode_pair_definitions`, `playlist_pair_links`, `catalog_fetch_queue`.
- Tests : migration appliquée deux fois sans erreur, schéma matches, isolation `title_slug` vérifiée avec une seed `synthetic_title_b`.

### Phase B — Extraction `version_id` au sync (1 commit)

- Étendre [transforms.go:89-164](apps/go-api/internal/sync/transforms.go#L89-L164) `ExtractRegistry()` pour extraire `Playlist.VersionId`, `PlaylistMapModePair.VersionId` depuis le payload Halo.
- Ajouter colonnes `playlist_version_id`, `pair_version_id` dans `match_registry` ([schema.go:97-129](apps/go-api/internal/sync/schema.go#L97-L129)).
- Migration de backfill (NULL acceptés, hydratés au prochain sync).
- Tests : `transforms_test.go` couvre extraction sur fixtures réelles.

### Phase C — Interfaces multi-titre canoniques (1 commit)

- Créer `internal/games/catalog_adapter.go` avec l'interface `TitleCatalogAdapter` (cf. §4bis.2).
- Créer les types canoniques `CanonicalPlaylist`, `CanonicalPair`, `CanonicalMap`, `CanonicalGameVariant`, `Experience`, `ModeCanonical` dans `internal/games/canonical/` (à côté des `MatchType`, `Outcome` existants).
- Étendre le `StaticResolver` pour exposer `CatalogAdapter(titleSlug)`.
- Tests d'isolation cross-titres : un appel `Resolver.CatalogAdapter("synthetic_title_b")` ne doit jamais router vers Halo.

### Phase D — Adapter Halo Infinite + TOML experience + reuse normalisation (1 commit)

- Créer `internal/games/halo_infinite/catalog_adapter.go` qui enveloppe [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go).
- À chaque hydratation de pair, appeler les fonctions Go existantes [NormalizeModeLabel](apps/go-api/internal/analysis/mode_label.go) et [InferModeCategoryFromPairName](apps/go-api/internal/analysis/mode_category.go) — **ne pas réimplémenter** ni porter en TOML, ce sont déjà du code Go enum-like (cf. §4ter).
- Porter la logique de classification d'`experience` (playlist-level) depuis [src/analysis/playlist_groups.py](src/analysis/playlist_groups.py) (Python) vers le TOML `config/titles/halo_infinite/catalog/experience_rules.toml`. Le TOML couvre **uniquement** l'`experience` playlist, pas la `mode_category` pair.
- Loader TOML dans `internal/games/halo_infinite/` (réutiliser `pelletier/go-toml/v2` déjà en place).
- Tests : 25-30 playlists permanentes mappées correctement (experience), 50+ pair_names couverts par mode_category (snapshot), snapshot des règles TOML versionné.

### Phase E — Détection lazy + enqueue au sync (1 commit)

- Hook après `ExtractRegistry()`, avant `InsertRegistryIfNotExists()` ([writes.go:22-65](apps/go-api/internal/sync/writes.go#L22-L65)) : vérifier si `(title_slug, playlist_id)`, `(title_slug, pair_id)`, `(title_slug, map_id)`, `(title_slug, game_variant_id)` existent dans leurs tables respectives. Si absents → INSERT OR IGNORE dans `catalog_fetch_queue`.
- Le `title_slug` provient du contexte sync (déjà disponible — Halo Infinite est seul titre productif aujourd'hui).
- Tests : un sync delta avec asset inconnu enqueue les bonnes lignes ; un sync delta avec asset connu n'enqueue rien ; pas de blocage de l'ingestion en cas d'erreur DB sur la queue ; isolation `title_slug` vérifiée.

### Phase F — Drain de la queue via adapters (1 commit)

- Service `CatalogFetcherService` qui :
  1. SELECT les lignes `catalog_fetch_queue` triées par `title_slug`, `attempts` ASC, `enqueued_at` ASC
  2. Récupère le `TitleCatalogAdapter` via le `Resolver` selon `title_slug`
  3. Pour chaque playlist : `adapter.FetchPlaylist()` → parse → upsert `playlists_catalog` + `playlist_pair_links` + enqueue les pairs / maps / variants si inconnus
  4. Pour chaque pair : `adapter.FetchPair()` → upsert `map_mode_pair_definitions` + enqueue map et game_variant si inconnus
  5. Sur succès → DELETE ; sur erreur → `attempts++` + `last_error`
- Pas de worker auto à ce stade — exposé via CLI `drain-catalog-queue --title halo_infinite`.
- Tests : drain sur fixtures DiscoveryUGC mockées via `synthetic_title_b` ; gestion 404 → `is_active = false` ; gestion erreur transitoire → réessayable.

### Phase G — CLI bootstrap one-shot (1 commit)

- Commande Go `populate-playlists-catalog --title halo_infinite` qui :
  1. Seed la queue depuis `SELECT DISTINCT playlist_id, pair_id, map_id, game_variant_id FROM shared.match_registry WHERE playlist_id IS NOT NULL`
  2. Lance le drain via `CatalogFetcherService`
  3. Loggue les stats finales (X playlists, Y pairs, Z maps, W variants, E erreurs)
- Tests : sur une DB de test avec 5 matchs distincts, peuple correctement les 4 tables référentielles.

### Phase H — Endpoint REST title-aware (1 commit)

- Créer `internal/api/handlers/catalog.go` qui expose :
  - `GET /api/v1/titles/{slug}/catalog/playlists?xuid={xuid}&only_played={bool}`
  - `GET /api/v1/titles/{slug}/catalog/pairs?playlist_asset_id={uuid}&xuid={xuid}&only_played={bool}`
  - `GET /api/v1/titles/{slug}/catalog/maps?xuid={xuid}&only_played={bool}`
- Gated par `MULTI_TITLE_API_ENABLED=true` (cohérent avec [field_mappings.go](apps/go-api/internal/api/handlers/field_mappings.go)).
- ETag + Cache-Control comme les autres endpoints title-aware.
- Tests : payload conforme contrats, 404 si `MULTI_TITLE_API_ENABLED=false`, isolation cross-titres.

### Phase I — Migration `FiltersService` vers le catalogue (1 commit)

- Réécrire [filters_service.go](apps/go-api/internal/service/filters_service.go) `Resolve()` pour requêter `playlists_catalog` ⨝ `match_registry` au lieu de scanner `match_participants`. Le `title_slug` vient du contexte (HTTP middleware).
- Ajouter le toggle `mode_only_played` (défaut `true`) dans la signature de l'endpoint.
- Tests : parité de comportement avec l'ancien service sur fixtures réelles ; benchmark Go avant/après pour quantifier le gain.

### Phase J — Refresh mensuel (1 commit, optionnel selon cadence)

- Soit cron OS (instructions doc dans `docs/`), soit goroutine dans [server.go](apps/go-api/internal/api/server.go) avec `time.Ticker(30*24*time.Hour)`.
- Drain de la queue + re-fetch des `is_active = true` pour tous les `title_slug` enregistrés.

---

## 6. Décisions ouvertes

| # | Question | Options |
|---|---|---|
| 1 | Critère de désactivation `is_active = false` | (a) jamais (manuel) ; (b) pas vue depuis 3 mois ; (c) pas vue depuis 6 mois |
| 2 | Mécanisme du refresh mensuel | (a) cron OS documenté ; (b) goroutine Go avec ticker ; (c) endpoint admin déclenchable |
| 3 | Faut-il exposer `weight` dans l'API React | Pas pour le filtre, mais pour une future page « stats par carte/mode » oui |
| 4 | Format des règles de classification | **Tranché 2026-04-30** : TOML versionné `config/titles/{slug}/catalog/experience_rules.toml` pour `experience` (playlist) ; code Go enum-like existant (`mode_category.go`, `mode_label.go`) pour `mode_category` + `mode_label` (pair). Découpage assumé : TOML pour ce qui peut bouger entre saisons sans redéploiement, code Go pour ce qui est stable enum-like. |
| 5 | Faut-il garder un historique des `version_id` par playlist | Non au début (`waypoint_assets_raw` couvre l'audit forensique si besoin) |
| 6 | Enregistrement des `title_slug` connus | Implicite dans les tables (DISTINCT sur `playlists_catalog.title_slug`) ou table dédiée `titles_registry` ? La couche `internal/games/` les énumère déjà via le `StaticResolver` — probablement suffisant. |
| 7 | Image map (`maps_catalog.image_url`) | **Tranché 2026-04-30** : peuplement dès Phase F via `assetResolver.Resolve(KindMapImage, ...)`. Cohérent avec home_repo.go pour les assets Spartan. NULL si GameCMS down → retry au refresh mensuel. |
| 8 | Faut-il exposer `mode_label` comme dimension de filtre, ou seulement `mode_category` | Dépend de la cardinalité réelle — audit prévu en §4quater. |

---

## 7. Tests à prévoir

- **Unitaires Go** : extraction `version_id`, classification `experience`, parsing `CustomData.PlaylistEntries`
- **Intégration DuckDB** (`//go:build integration`) : migration applicable, INSERT/UPSERT, JOIN catalogue ↔ registry
- **Fixtures DiscoveryUGC** : capturer 2-3 réponses réelles (Quick Play, Ranked Arena, Firefight) dans `testdata/`
- **Vitest React** (Phase G+) : `useFiltersResolve` consomme bien le toggle `mode_only_played`
- **Benchmark perf** : `go test -bench=BenchmarkFiltersResolve` avant/après pour quantifier le gain

---

## 8. Hors scope (à reporter)

- Refonte UI du `FilterOmnibar` au-delà de l'ajout du toggle « Joué / Tous »
- Page de configuration admin du catalogue (édition manuelle d'`is_active`, etc.)
- Statistiques agrégées par playlist (matchs joués totaux, durée moyenne, etc.) — possible suite naturelle
- Intégration AMQP `lobby-hi.svc.halowaypoint.com` mentionnée dans l'article den.dev pour la liste autoritative des playlists actives — overkill pour le besoin actuel

---

## 9. Estimation ordre de grandeur

- Phases A-B : ~1 jour (migration schéma + extraction `version_id`)
- Phases C-D : ~1,5 jour (interfaces multi-titre + adapter Halo + TOML classification)
- Phases E-G : ~1,5 jour (lazy enqueue + drain + CLI bootstrap)
- Phase H : ~0,5 jour (endpoint REST title-aware)
- Phase I : ~1 jour (migration FiltersService + tests parité + benchmark)
- Phase J : ~0,5 jour (refresh mensuel)
- Total raisonnable : **5-6 jours-homme** pour un sprint dédié (vs 3-5 estimés initialement, surcoût lié au respect du pattern multi-titre).

---

## 10. Références

### Code à étendre
- [FilterOmnibar.tsx](apps/web/src/components/shell/FilterOmnibar.tsx) — UI actuelle des filtres
- [globalFilterStore.ts](apps/web/src/stores/globalFilterStore.ts) — Zustand store
- [filters_service.go](apps/go-api/internal/service/filters_service.go) — Logique cascade actuelle (cible Phase I)
- [filters_repo.go](apps/go-api/internal/platform/duckdb/filters_repo.go) — Accès données filtres
- [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) — Provider Halo DiscoveryUGC à wrapper dans l'adapter Halo
- [discovery_types.go](apps/go-api/internal/platform/halo/discovery_types.go) — Types `AssetTypePlaylist`, `AssetTypePair`
- [transforms.go](apps/go-api/internal/sync/transforms.go) — Extraction registry au sync
- [writes.go](apps/go-api/internal/sync/writes.go) — Persistance match_registry
- [schema.go](apps/go-api/internal/sync/schema.go) — Schéma `shared.match_registry`
- [steps_metadata.go](apps/go-api/internal/migration/steps_metadata.go) — Migrations metadata

### Couche multi-titre Go (à étendre)
- `apps/go-api/internal/games/{adapter,resolver}.go` — Interfaces `TitleDataAdapter`, `TitleSemanticAdapter`, `StaticResolver` (à étendre avec `TitleCatalogAdapter`)
- `apps/go-api/internal/games/canonical/` — 43 FieldKey + enums (à étendre avec `Experience`, `ModeCanonical`, `CanonicalPlaylist`, `CanonicalPair`, `CanonicalMap`, `CanonicalGameVariant`)
- `apps/go-api/internal/games/halo_infinite/` — `DataAdapter` + `SemanticAdapter` (à étendre avec `CatalogAdapter`)
- `apps/go-api/internal/games/synthetic_title_b/` — Corpus de tests d'isolation cross-titres
- `apps/go-api/internal/api/handlers/field_mappings.go` — Pattern endpoint title-aware avec ETag + gating `MULTI_TITLE_API_ENABLED`
- `apps/go-api/internal/api/handlers/multi_title_preview.go` — Pattern handler proof-of-concept
- `config/titles/halo_infinite/mappings/fields.toml` — Pattern config TOML versionnée (à dupliquer pour `catalog/`)
- `tools/mappings/CHANGELOG.md` — Pattern de versionnement TOML

### Couche normalisation modes Halo (à consommer, pas réinventer)
- [apps/go-api/internal/analysis/mode_label.go](apps/go-api/internal/analysis/mode_label.go) — `NormalizeModeLabel(raw, mapLabels...)` consommée par l'adapter Halo (Phase D)
- [apps/go-api/internal/analysis/mode_category.go](apps/go-api/internal/analysis/mode_category.go) — `InferModeCategoryFromPairName(pairName)` consommée par l'adapter Halo (Phase D), enum 8 valeurs
- Skill projet : `.claude/skills/halo-modes/SKILL.md` — conventions de normalisation (2 niveaux orthogonaux, divergences assumées vs Python v7)

### Code Python à porter (logique éprouvée)
- [src/analysis/playlist_groups.py](src/analysis/playlist_groups.py) — Heuristiques de classification `experience` (à porter vers TOML Halo)
- [src/data/domain/refdata.py](src/data/domain/refdata.py) — Enum `GameVariantCategory` (à porter vers `internal/games/canonical/`)

### Pattern Kinds (référence d'inspiration architecture)
- [resolver_default.go](apps/go-api/internal/assets/resolver_default.go), [kinds.go](apps/go-api/internal/assets/kinds.go), [fetcher_chain.go](apps/go-api/internal/assets/fetcher_chain.go)

### Sources externes
- [Halo Waypoint — Multiplayer Playlists](https://support.halowaypoint.com/hc/en-us/articles/17920041655188-Halo-Infinite-Multiplayer-Playlists)
- [den.dev — Halo Infinite Playlist Weights](https://den.dev/blog/halo-infinite-playlist-weights/)

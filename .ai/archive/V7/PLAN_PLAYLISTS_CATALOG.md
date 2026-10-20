# Plan — Catalogue global Playlists / Pairs / Maps

> Analyse réalisée le 2026-04-30. Conception d'un référentiel global Halo Infinite
> pour accélérer la cascade de filtres et écrémer l'UI aux options réellement utiles.
> Ce document couvre uniquement le design data + sync — pas l'UI React (cible d'un sprint suivant).
>
> **Branche Git cible : implémentation sur `docs/charts-specs` (branche courante).**
> Amendements 2026-04-30 : `match_context` solo/squad (Phase C), `CatalogRepo` port (Phase C),
> cascade preservation + session filter cross-DB + fallback guard (Phase I), migration tests, logging.

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

### 4.2 Tables (8 au total, toutes title-aware via `title_slug`)

> **Schéma final, intégrant les décisions §4sexies (multi-langues) et §4cinquies (auto-détection préfixes).**
> Les tables catalogue ne portent plus `name_en` + `name_fr` inline mais un seul `name_canonical` (EN). Toutes les langues sont dans `asset_translations` (existante, multi-lang). Les labels normalisés multi-langues sont dans `pair_mode_label_translations` (nouvelle, dédiée).

```sql
-- 1. Référentiel des playlists, par titre
CREATE TABLE playlists_catalog (
  title_slug          VARCHAR,
  playlist_asset_id   UUID,
  current_version_id  UUID,
  name_canonical      VARCHAR,       -- EN par défaut (debug + lookup rapide). Autres langues : asset_translations.
  experience          VARCHAR,        -- 'ranked' | 'social' | 'btb' | 'firefight' | 'action_sack' | 'limited_time' | 'custom_browser' | 'unknown'
  is_ranked           BOOLEAN,
  is_active           BOOLEAN DEFAULT TRUE,
  first_seen_at       TIMESTAMP,
  last_seen_at        TIMESTAMP,
  last_fetched_at     TIMESTAMP,
  PRIMARY KEY (title_slug, playlist_asset_id)
);

-- 2. Référentiel des maps (séparé pour ne pas dupliquer name/image)
CREATE TABLE maps_catalog (
  title_slug          VARCHAR,
  map_asset_id        UUID,
  current_version_id  UUID,
  name_canonical      VARCHAR,       -- EN par défaut. Autres langues : asset_translations.
  image_url           VARCHAR,       -- résolu via internal/assets (KindMapImage) au fetch (cf. §4.3)
  last_fetched_at     TIMESTAMP,
  PRIMARY KEY (title_slug, map_asset_id)
);

-- 3. Référentiel des game variants (séparé : un variant peut être dans N pairs)
CREATE TABLE game_variants_catalog (
  title_slug              VARCHAR,
  game_variant_asset_id   UUID,
  current_version_id      UUID,
  name_canonical          VARCHAR,    -- EN par défaut. Autres langues : asset_translations.
  mode_canonical          VARCHAR,    -- 'slayer' | 'ctf' | 'oddball' | 'koth' | 'strongholds' | 'extraction' | 'firefight_kotr' | 'fiesta' | ...
  game_variant_category   INTEGER,    -- code numérique Halo (équivalent GameVariantCategory)
  last_fetched_at         TIMESTAMP,
  PRIMARY KEY (title_slug, game_variant_asset_id)
);

-- 4. Pair = jonction map + game_variant + nom composite, par titre
CREATE TABLE map_mode_pair_definitions (
  title_slug             VARCHAR,
  pair_asset_id          UUID,
  current_version_id     UUID,
  name_canonical         VARCHAR,    -- ex: "Arena:Slayer on Bazaar" (EN brut DiscoveryUGC). Autres langues : asset_translations.
  map_asset_id           UUID,       -- FK logique -> maps_catalog
  game_variant_asset_id  UUID,       -- FK logique -> game_variants_catalog
  mode_category          VARCHAR,    -- sortie InferModeCategoryFromPairName — enum Go via TOML (cf. §4cinquies)
                                     -- Note : labels normalisés multi-langues dans pair_mode_label_translations
  last_fetched_at        TIMESTAMP,
  PRIMARY KEY (title_slug, pair_asset_id)
);

-- 5. Relation N-N playlist <-> pair, avec poids de tirage
CREATE TABLE playlist_pair_links (
  title_slug         VARCHAR,
  playlist_asset_id  UUID,
  pair_asset_id      UUID,
  weight             DOUBLE,           -- depuis CustomData.PlaylistEntries[].Metadata.Weight
  PRIMARY KEY (title_slug, playlist_asset_id, pair_asset_id)
);

-- 6. File d'attente du fetcher (pattern Kinds, drain mensuel)
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

-- 7. Labels normalisés multi-langues (sortie NormalizeModeLabel par langue)
CREATE TABLE pair_mode_label_translations (
  title_slug      VARCHAR,
  pair_asset_id   UUID,
  lang            VARCHAR,           -- 'en', 'fr', 'de', 'es-ES', 'es-MX', 'it', 'ja', 'ko', 'nl', 'pl', 'pt-BR', 'ru', 'zh-CN', 'zh-TW'
  label           VARCHAR,           -- sortie NormalizeModeLabel(name_in_lang, mapLabels)
  PRIMARY KEY (title_slug, pair_asset_id, lang)
);

-- 8. Auto-détection des préfixes inconnus (alerting sur nouvelles catégories candidates)
CREATE TABLE unknown_prefix_candidates (
  title_slug    VARCHAR,
  prefix        VARCHAR,             -- ex: "Mega Fiesta" si jamais ça émerge
  n_matches     INTEGER DEFAULT 1,
  first_seen_at TIMESTAMP,
  last_seen_at  TIMESTAMP,
  pair_examples VARCHAR[],           -- 3-5 pair_name observés
  PRIMARY KEY (title_slug, prefix)
);
```

### Tables existantes consommées (pas créées par ce plan)

| Table | Rôle |
|---|---|
| `metadata.asset_translations(asset_id, asset_type, lang, name, description, fetched_at)` | Stockage multi-langues des noms bruts d'asset (déjà multi-lang via colonne `lang`). Ce plan l'étend en lui poussant les noms de playlists/pairs/maps/variants multi-langues lors de l'hydratation. |
| `metadata.mode_name_tr` / `metadata.mode_pair_overrides` / `metadata.mode_prefix_names` | Input fallback consommé par `NormalizeModeLabel` quand DiscoveryUGC ne fournit pas la traduction. Conservées telles quelles. |

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

→ **Audit cardinalité réalisé en §4quater** (mesure 2026-04-30 sur la DB de production locale).

### 4ter.6 Comparatif concret AVANT / APRÈS sur 4 cas réels

Échantillon tiré de `shared_matches_v2.duckdb` (mesure 2026-04-30) :

| pair_name (brut DB) | n matchs | AUJOURD'HUI (sans catalogue) | DEMAIN (avec catalogue) |
|---|---:|---|---|
| `BTB:Slayer on Deadlock` | 42 | `mode_category='BTB'` ✅ persisté au sync, `pair_name_fr=NULL` ❌, label EN/FR recalculé à chaque vue UI | `map_mode_pair_definitions.name_canonical="BTB:Slayer on Deadlock"` + `mode_category="BTB"` ; `asset_translations` reçoit le nom brut dans 14 langues ; `pair_mode_label_translations(lang='fr', label='Assassin')` + autres langues, tout persisté une fois |
| `Super Fiesta:Slayer on Streets - Forge` | 23 | `mode_category='Fiesta'` (regroupement Python — cf. §4ter.7), `_fr=NULL`, suffixe `Forge` à stripper à chaque vue | `name_canonical="Super Fiesta:Slayer on Streets - Forge"`, **`mode_category="SuperFiesta"`** (Go via TOML §4cinquies), `pair_mode_label_translations(lang='fr', label='Assassin')` (suffixe stripé une fois au fetch) |
| `Event:Escalation Slayer on Streets` | 2 | `mode_category='Other'` (préfixe Event mappé Other), `_fr=NULL` | `name_canonical="Event:Escalation Slayer on Streets"`, `mode_category="Other"`, `pair_mode_label_translations(lang='en', label='Escalation Slayer')`, `(lang='fr', label='Assassin par escalade')` |
| `b51efd48-ffd9-4a03-...` (UUID) | 12 | `mode_category='Other'` par défaut, UUID brut affiché à l'utilisateur (= bug visuel) | Bootstrap CLI fetch DiscoveryUGC → résout l'UUID en (par exemple) `"Arena:Slayer on Bazaar"` → upsert `name_canonical` + `asset_translations` + `pair_mode_label_translations` pour toutes les langues. Label propre dans toutes les langues. |

État global mesuré (1545 matchs) :
- `mode_category` : **1545/1545 remplis** ✅ (calculé au sync)
- `pair_name_fr`, `map_name_fr`, `playlist_name_fr` : **0/1545 remplis** ❌ (calculés à la volée à chaque requête UI)
- `playlist_name` UUID brut non résolu : **333/1545 = 21.5%** ❌

### 4ter.7 Super Fiesta + Husky Raid : promotion en catégories distinctes

Tu as raison sur ce point — le code Python actuel regroupe abusivement :

```python
# src/analysis/mode_display.py L. 50-52
"Super Fiesta": {"category": "Fiesta", "qualifier": None},
"Husky Raid": {"category": "Fiesta", "qualifier": None},
"Super Husky Raid": {"category": "Fiesta", "qualifier": None},
```

Mesure réelle sur `shared_matches_v2.duckdb` (catégorie `Fiesta` actuelle = 307 matchs) :

| Préfixe pair_name | n matchs | Catégorie cible Go (correcte) |
|---|---:|---|
| `Super Fiesta:...` | 240 (78 %) | **SuperFiesta** |
| `Super Husky Raid:...` | 11 | **HuskyRaid** |
| `Husky Raid...` | 4 | **HuskyRaid** |
| `Fiesta:...` | 3 | Fiesta (correct) |
| `Castle Wars...` | 1 | Fiesta (correct) |
| Autre | 48 | à investiguer |

→ **78 % des matchs « Fiesta » sont en réalité du Super Fiesta**, gameplay distinct (modes random vs vraie Fiesta dés-driven). Le code Go a déjà fait la promotion (cf. skill `halo-modes` : « Super Fiesta et Husky Raid sont des catégories distinctes en Go (Python les regroupait sous Fiesta). Ne pas revenir en arrière. »).

**Conséquence pour le plan** :

- Au moment où le sync passera en Go (cible de la migration `LevelUp-go-migration`), `match_registry.mode_category` sera re-calculé pour les nouveaux matchs → catégories correctes.
- Pour les **matchs anciens déjà synchronisés en Python**, deux options :
  - **(a)** Migration one-shot : `UPDATE match_registry SET mode_category = 'SuperFiesta' WHERE pair_name LIKE 'Super Fiesta:%'` etc. — à intégrer dans une phase de migration.
  - **(b)** Les laisser en `Fiesta` historique et accepter la divergence — pas idéal pour le filtre (« Super Fiesta » groupe vide visuellement).
- **Reco** : (a) — migration explicite dans la phase qui passe le sync en Go, traitée hors scope de ce plan mais à signaler comme dépendance.

Le catalogue, lui, n'a pas de bricolage à faire : il consomme ce que le sync stocke. Si le sync stocke la bonne catégorie, le filtre l'expose proprement.

### 4ter.8 Traductions FR : du bricolage à la persistance

Constat brutal : **0 / 1545 matchs ont `pair_name_fr`, `map_name_fr` ou `playlist_name_fr` remplis** dans la DB actuelle. Ces colonnes existent dans le schéma (cf. migration `add_match_registry_i18n_columns`) mais ne sont jamais peuplées par le sync. Conséquence : à chaque vue UI qui affiche un mode/map/playlist en français, on déclenche un appel `resolve_display_mode()` ou équivalent qui consulte `mode_name_tr` + `mode_pair_overrides`. Ça tourne mais c'est du runtime gaspillé.

**Le catalogue règle ça naturellement** :

1. **`maps_catalog.name_fr`, `game_variants_catalog.name_fr`, `map_mode_pair_definitions.mode_label_fr`, `playlists_catalog.name_fr`** sont peuplés au moment du fetch DiscoveryUGC. L'API supporte le paramètre `lang` (cf. [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) `doGetWithLang`). Deux fetchs successifs `lang=en` puis `lang=fr` hydratent les deux versions.
2. **Fallback** : si DiscoveryUGC ne fournit pas la traduction FR pour un asset (cas connu pour les modes Forge UGC), on retombe sur `mode_name_tr` + `mode_pair_overrides` au moment du fetch — résultat persisté quand même.
3. **Durabilité** : les traductions ne sont calculées qu'**une fois** par asset, vivent dans le catalogue, et survivent à toutes les vues UI futures.

**Conséquence pour le plan** :

- **Phase D (adapter Halo)** : ajouter un appel `FetchAsset(..., lang="fr")` en plus de `lang="en"` pour chaque asset hydraté (playlist, pair, map, game_variant).
- **Phase F (drain queue)** : persister les deux versions dans les colonnes `name_en` / `name_fr`.
- **Phase D bis** : appliquer la normalisation FR (`resolve_display_mode` Python ou son équivalent Go) au résultat pour produire `mode_label_fr` propre.
- **Hors scope mais à signaler** : une fois le catalogue alimenté, on peut **dépeupler** les colonnes FR de `match_registry` (elles deviennent redondantes — info portée par le catalogue via FK). Cleanup ultérieur, pas bloquant.

→ **Pas de "bricolage" à craindre** : le catalogue convertit la résolution opportuniste actuelle en pipeline propre persisté.

---

## 4cinquies. Évolutivité des catégories : TOML + auto-détection

Le code Go actuel ([mode_category.go](apps/go-api/internal/analysis/mode_category.go)) hardcode 8 constantes (Assassin, Fiesta, SuperFiesta, HuskyRaid, BTB, Ranked, Firefight, Other) avec un mapping préfixe → catégorie en dur. Conséquence : ajouter une nouvelle catégorie (ex: futur « Mega Fiesta ») = ajouter une constante + modifier le mapping + recompiler. Friction inutile vu la cadence d'évolution Halo.

### 4cinquies.1 Cible : TOML + auto-détection (option (c) actée 2026-04-30)

**Volet 1 — Migrer les règles en TOML versionné**, cohérent avec `experience_rules.toml` :

```
config/titles/halo_infinite/catalog/
├── experience_rules.toml         # déjà prévu (ranked/social/btb/firefight/...)
└── mode_category_rules.toml      # nouveau : prefix -> category enum
```

`mode_category_rules.toml` (extrait) :

```toml
schema_version = 1

[[rule]]
category = "Assassin"
prefixes = ["Arena", "Tactical", "Assault", "Community"]

[[rule]]
category = "SuperFiesta"
prefixes = ["Super Fiesta"]
note = "Promu vs Python v7 (cf. skill halo-modes). 240 matchs historiques en attente de migration."

[[rule]]
category = "HuskyRaid"
prefixes = ["Husky Raid", "Super Husky Raid"]

[[rule]]
category = "BTB"
prefixes = ["BTB", "BTB Heavies"]

# ...
```

[mode_category.go](apps/go-api/internal/analysis/mode_category.go) devient un loader TOML + cache en mémoire au boot. Les helpers existants (`PairNamePrefixesForCategory`, `AllKnownPairNamePrefixes`) lisent depuis ce cache. Validation au boot : tous les enum Go doivent avoir une règle correspondante (sinon panic au démarrage = échec rapide).

**Volet 2 — Auto-détection des préfixes inconnus** :

Nouvelle table dans `metadata.duckdb` :

```sql
CREATE TABLE unknown_prefix_candidates (
  title_slug    VARCHAR,
  prefix        VARCHAR,
  n_matches     INTEGER DEFAULT 1,
  first_seen_at TIMESTAMP,
  last_seen_at  TIMESTAMP,
  pair_examples VARCHAR[],         -- 3-5 exemples concrets
  PRIMARY KEY (title_slug, prefix)
);
```

Le sync, après extraction du `pair_name` de chaque match, vérifie si le préfixe (partie avant `:` ou label complet si pas de `:`) matche une règle existante. Si non → INSERT OR REPLACE dans `unknown_prefix_candidates` avec incrément de `n_matches`.

**Workflow alerting** :
- Quand `n_matches >= 5` pour un préfixe non couvert → **log warning structuré** (niveau WARN, champ `event=unknown_prefix_candidate` pour filtrage côté observabilité).
- Endpoint admin `GET /api/v1/titles/{slug}/catalog/unknown-prefixes` qui liste les candidats triés par `n_matches DESC` pour décision humaine.
- Décision humaine : éditer `mode_category_rules.toml` pour ajouter le préfixe, ou ignorer (laisser en `Other`).
- **Pas de notification externe** (Discord, Slack, email). Le log + l'endpoint admin suffisent ; ajout d'un canal de notif réservé à une décision séparée si le besoin émerge.

### 4cinquies.2 Avantages

- **Édition sans recompil** : un nouveau préfixe = 1 commit TOML.
- **Pas de surveillance manuelle** : le système alerte quand un seuil est franchi.
- **Cohérence avec experience_rules.toml** : même mécanisme de loader + cache + validation au boot.
- **Traçabilité** : `unknown_prefix_candidates` archive l'historique des candidats — pas perdu si on décide plus tard.
- **Pas de rupture** : tant que `mode_category_rules.toml` charge les mêmes 8 catégories enum Go, le code consommateur est inchangé.

### 4cinquies.3 Tradeoff

- **Validation compile-time perdue** : aujourd'hui les constantes Go garantissent que tout le code utilise les bonnes catégories. Avec TOML, la validation se fait au boot (panic si désynchronisation enum vs TOML). Mitigation : test d'intégration qui charge le TOML et vérifie la couverture exhaustive des enum.
- **Migration des matchs Python historiques** : indépendant de cette décision (cf. décision ouverte #9). Reste à traiter dans la phase de bascule sync Python → Go.

---

## 4sexies. i18n catalogue : architecture multi-langues robuste

Sujet sensible (cf. retour utilisateur 2026-04-30 : « les agents font leur sauce et ne comprennent pas que les traductions sont à tels endroit et qu'elles existent bien »). L'architecture cible est documentée à 3 endroits pour discoverability :
- Ce plan (§4sexies, design)
- **Skill `.claude/skills/halo-i18n/SKILL.md`** (mémo opérationnel, anti-patterns, helpers à utiliser)
- Code (commentaires de schéma + docstrings d'adapter)

### 4sexies.1 Architecture en 3 couches

| Couche | Stockage | Contenu | Multi-lang |
|---|---|---|---|
| **Nom canonique** (debug + lookup rapide) | Inline `name_canonical VARCHAR` dans chaque table catalogue | Nom EN par défaut depuis DiscoveryUGC | Non (1 seule valeur) |
| **Noms bruts traduits** | Table partagée `metadata.asset_translations` (existante, multi-lang via colonne `lang`) | Toutes les langues hydratées depuis DiscoveryUGC | **Oui native** |
| **Labels normalisés** (sortie `NormalizeModeLabel`) | Nouvelle table `metadata.pair_mode_label_translations(pair_asset_id, lang, label)` | Sortie de la normalisation par langue | **Oui native** |

### 4sexies.2 Schéma révisé (impact §4.2)

Les 4 tables catalogue (`playlists_catalog`, `maps_catalog`, `game_variants_catalog`, `map_mode_pair_definitions`) **conservent un seul `name_canonical`** au lieu de `name_en` + `name_fr` :

```sql
-- AVANT (§4.2)
name_en  VARCHAR,
name_fr  VARCHAR,

-- APRÈS (§4sexies)
name_canonical  VARCHAR,    -- EN par défaut, pour debug + requêtes simples sans JOIN
```

Pour `map_mode_pair_definitions`, les colonnes `mode_label_en` et `mode_label_fr` ajoutées en §4ter.6 deviennent **redondantes** — elles partent dans `pair_mode_label_translations`. Le pair conserve uniquement `mode_category` (qui est un enum Go non traduit).

Schéma final de `pair_mode_label_translations` :

```sql
CREATE TABLE pair_mode_label_translations (
  title_slug      VARCHAR,
  pair_asset_id   UUID,
  lang            VARCHAR,        -- 'en', 'fr', 'de', 'es-ES', ...
  label           VARCHAR,        -- sortie de NormalizeModeLabel pour cette langue
  PRIMARY KEY (title_slug, pair_asset_id, lang)
);
```

`asset_translations` reste tel quel (déjà multi-lang).

### 4sexies.3 Hydratation multi-langues

**Liste des langues cibles** (à confirmer en lisant [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) `doGetWithLang` au moment de l'implémentation) :

```
en, fr, de, es-ES, es-MX, it, ja, ko, nl, pl, pt-BR, ru, zh-CN, zh-TW
```

**Stratégie de fetch** au moment de l'hydratation (Phase D) :

1. Pour chaque asset (playlist/pair/map/game_variant) à hydrater, l'adapter Halo loope sur la liste des langues cibles.
2. Pour chaque `lang`, appel `FetchAsset(ctx, assetType, titleID, assetID, versionID, lang)` → upsert dans `asset_translations`.
3. Pour les pairs uniquement : après chaque fetch `lang`, appel `NormalizeModeLabel(name_in_lang, mapLabels)` → upsert dans `pair_mode_label_translations`.
4. Si DiscoveryUGC retourne 404 pour une langue (cas connu : modes Forge UGC) → fallback sur `mode_name_tr` + `mode_pair_overrides` pour cette langue. Si fallback nul aussi → laisser absent (la lecture COALESCE sur `name_canonical`).

Coût marginal : 14 round-trips HTTP par asset au lieu de 2. Sur 250 pairs uniques + ~30 playlists + ~100 maps + ~30 variants = ~410 assets × 14 langues = ~5740 calls au bootstrap. Avec le rate limiter existant (cf. `discovery_client.go`), ça prend ~10-15 minutes une fois pour toutes. Acceptable pour un one-shot bootstrap.

### 4sexies.4 Lecture côté API

Endpoint REST title-aware accepte `?lang=` :

```
GET /api/v1/titles/halo_infinite/catalog/playlists?lang=fr
```

Requête typique côté Go :

```sql
SELECT
  p.playlist_asset_id,
  COALESCE(t.name, p.name_canonical) AS display_name,
  p.experience,
  ...
FROM playlists_catalog p
LEFT JOIN asset_translations t
  ON  t.asset_id   = p.playlist_asset_id
 AND  t.asset_type = 'playlist'
 AND  t.lang       = $1
WHERE p.title_slug = 'halo_infinite' AND p.is_active = TRUE;
```

Pour les labels normalisés, JOIN sur `pair_mode_label_translations` avec même pattern.

### 4sexies.5 Impact sur l'existant — plan de cleanup

| Aspect | État courant | Action requise | Phase |
|---|---|---|---|
| `match_registry.{pair_name_fr, map_name_fr, playlist_name_fr}` (0/1545 peuplées) | Colonnes existantes mais inutilisées | **Supprimer en cleanup final** une fois le catalogue alimenté et l'UI migrée vers les JOINs catalogue | Phase K (cleanup) |
| Code UI Go qui résout les FR à chaud | Lit `mode_name_tr` + `mode_pair_overrides` à chaque requête | **Migrer** vers JOIN sur `asset_translations` + `pair_mode_label_translations` | Phase I (migration FiltersService) |
| Tables `mode_name_tr`, `mode_pair_overrides`, `mode_prefix_names` | Source d'input | **Conserver** comme input fallback dans la pipeline d'hydratation. Pas de dépréciation. | — |
| Helpers Python `resolve_display_mode`, `resolve_weapon_display`, etc. | Calcul à chaud | **Conserver** en parallèle pendant la migration. À termes (post Go) → suppression. | Hors scope |

### 4sexies.6 Anti-patterns à proscrire (cf. skill halo-i18n)

À chaque revue de PR touchant aux traductions, vérifier :

1. ❌ Aucun ajout de colonne `_fr` inline dans une table métier (utiliser `asset_translations` ou `pair_mode_label_translations`)
2. ❌ Aucun calcul de traduction à chaud dans une boucle UI (utiliser le catalogue persisté)
3. ❌ Aucune nouvelle table `xxx_translations` sans justification : étendre `asset_translations` avec un nouveau `asset_type` à la place
4. ❌ Aucun string littéral français/anglais hardcodé pour une catégorie ou un mode (utiliser les constantes Go)
5. ❌ Aucun import direct de `mode_name_tr` dans le code consommateur — passer par les helpers / le catalogue

---

---

## 4quater. Audit cardinalité réelle (mesuré 2026-04-30)

Mesure faite sur `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb` (1545 matchs, 4 joueurs sync). Détail des requêtes : DuckDB CLI direct sur `match_registry` qui porte déjà la colonne `mode_category` (sortie de `InferModeCategoryFromPairName()` au sync).

### 4quater.1 Distribution `mode_category`

| Catégorie | Matchs | % | Pair_names distincts | Playlists distinctes |
|---|---:|---:|---:|---:|
| **Assassin** | 636 | 41.2 % | 250 | 8 |
| **BTB** | 493 | 31.9 % | 65 | 3 |
| **Fiesta** | 307 | 19.9 % | 73 | 6 |
| Other | 70 | 4.5 % | 21 | 6 |
| Ranked | 34 | 2.2 % | 19 | 3 |
| Firefight | 5 | 0.3 % | 5 | 3 |

3 catégories absorbent **93%** du volume (Assassin + BTB + Fiesta). Catégorie `Other` à 4.5% — sous le seuil de 10% qui aurait justifié un boucher prioritaire des `_PREFIX_RULES`. Les 70 matchs `Other` se répartissent entre UUID asset_id non résolus (~70%), `Event:Escalation Slayer`, et quelques variantes Community: exotiques.

### 4quater.2 Insight cardinalité — `mode_label` est nécessaire

34 `mode_label` distincts au total, 48 paires `(mode_category, mode_label)` distinctes. Détail Assassin (la plus grosse catégorie, 17 labels) :

| mode_label | n |
|---|---:|
| Team Slayer | 186 |
| Slayer | 186 |
| CTF | 89 |
| Strongholds | 56 |
| King of the Hill | 49 |
| Oddball | 17 |
| Neutral Flag CTF | 17 |
| Team Snipers | 8 |
| One Flag CTF | 7 |
| Escalation Slayer | 6 |
| Neutral Bomb | 4 |
| VIP | 3 |
| One Bomb | 3 |
| Attrition | 2 |
| Land Grab, FFA Slayer, Neutral Bomb Squad | 1 |

**Insight critique** : la catégorie `Assassin` ne contient PAS uniquement des Slayer-likes. Elle contient CTF, Strongholds, KoTH, Oddball, Bomb, VIP, etc. Ce sont les modes joués via les playlists Arena/Tactical/Assault/Community (cf. skill `halo-modes` §préfixes). Idem pour BTB qui mélange Slayer/CTF/Total Control/Stockpile/etc.

→ **`mode_category` et `mode_label` sont vraiment orthogonaux**, pas hiérarchiques. Un utilisateur peut vouloir « tous les Slayer peu importe la playlist » (= 186+186+189+? = au moins 561 matchs cumulés sur les 3 catégories majeures, soit **36%** du total). Cette intention n'est exprimable QUE via `mode_label`, pas via `mode_category`.

### 4quater.3 Insight playlists — gain immédiat du catalogue

Top playlists par volume :

| playlist | n |
|---|---:|
| Quick Play | 827 |
| Big Team Battle | 341 |
| **`<UUID non résolu>`** | **333** |
| Super Fiesta | 9 |
| Ranked Arena | 8 |
| Ranked Slayer | 7 |
| Team Snipers | 6 |
| ... (8 autres playlists < 5 matchs) | |

**21.5% des matchs ont une `playlist_name` qui est encore l'UUID brut** — DiscoveryUGC n'a pas été appelé au sync ou a échoué. C'est exactement ce que le catalogue résoudrait d'un coup au bootstrap. Gain immédiat de qualité d'affichage **gratuit**.

### 4quater.4 Décisions tranchées

1. **Exposer `mode_category` ET `mode_label` comme dimensions parallèles dans le filtre** (pas hiérarchiques). Confirmé par les données : 36% des matchs sont du « Slayer » réparti dans 3 catégories différentes — le filtre par catégorie seule ne permet pas de les regrouper.
2. **Ajouter `playlists_catalog` en bootstrap prioritaire** : 333 UUIDs à résoudre = 21.5% du volume. Le CLI Phase G les attrape tous d'un coup.
3. **Pas de boucher prioritaire `_PREFIX_RULES`** pour `Other` — 4.5% est tolérable, le catalogue résoudra la majorité (UUID).
4. **Cardinalité gérable côté UI** : 6 catégories × 21 playlists × 34 labels × 103 maps. Aucun risque de saturation. Le toggle « only_played » reste pertinent pour réduire la surface visible mais n'est pas strictement nécessaire pour la perf.

### 4quater.5 Schéma de filtre UI implicite

Trois facettes parallèles + une dimension de scope :

```
[Experience]      Ranked / Social / BTB / Firefight / Action Sack / Custom    (playlist-level via TOML)
[Mode Category]   Assassin / BTB / Fiesta / SuperFiesta / HuskyRaid / Ranked / Firefight / Other    (pair-level via Go)
[Mode Label]      Slayer / Team Slayer / CTF / Strongholds / KoTH / ...    (pair-level via NormalizeModeLabel)
[Map]             Aquarius / Live Fire / Bazaar / ...    (pair-level via maps_catalog)

[Scope]           only_played (défaut) | all
```

Toutes les combinaisons sont AND-ables. La cascade visible (« si je sélectionne BTB, montre-moi les playlists BTB ») reste pertinente comme aide à l'exploration mais n'est plus une nécessité technique.

---

## 5. Plan d'implémentation par phases

### Phase A — Migration schéma metadata (1 commit)

- Ajouter une migration dans [steps_metadata.go](apps/go-api/internal/migration/steps_metadata.go) qui crée les **8 tables** title-aware : `playlists_catalog`, `maps_catalog`, `game_variants_catalog`, `map_mode_pair_definitions`, `playlist_pair_links`, `catalog_fetch_queue`, `pair_mode_label_translations`, `unknown_prefix_candidates`.
- Vérifier que `asset_translations` (existante) supporte les `asset_type` 'playlist', 'pair', 'map', 'game_variant' (étendre l'enum si nécessaire).
- Tests : migration appliquée deux fois sans erreur, schéma matches, isolation `title_slug` vérifiée avec une seed `synthetic_title_b`.

### Phase B — Extraction `version_id` au sync (1 commit)

- Étendre [transforms.go:89-164](apps/go-api/internal/sync/transforms.go#L89-L164) `ExtractRegistry()` pour extraire `Playlist.VersionId`, `PlaylistMapModePair.VersionId` depuis le payload Halo.
- Ajouter colonnes `playlist_version_id`, `pair_version_id` dans `match_registry` ([schema.go:97-129](apps/go-api/internal/sync/schema.go#L97-L129)).
- Migration de backfill (NULL acceptés, hydratés au prochain sync).
- Tests : `transforms_test.go` couvre extraction sur fixtures réelles.

### Phase C — Interfaces multi-titre canoniques + port/CatalogRepo + FilterContextInput (1 commit)

- Créer `internal/games/catalog_adapter.go` avec l'interface `TitleCatalogAdapter` (cf. §4bis.2).
- Créer les types canoniques `CanonicalPlaylist`, `CanonicalPair`, `CanonicalMap`, `CanonicalGameVariant`, `Experience`, `ModeCanonical` dans `internal/games/canonical/`.
- Étendre le `StaticResolver` pour exposer `CatalogAdapter(titleSlug)`.
- **Ajouter `CatalogRepo` dans `internal/port/`** : interface lecture seule (le fetch DiscoveryUGC est exclusivement du ressort du `CatalogFetcherService`). Méthodes minimales : `PlaylistsByTitle(ctx, titleSlug, xuid, onlyPlayed bool)`, `PairsByPlaylist(ctx, titleSlug, playlistID string)`, `MapsByTitle(ctx, titleSlug, xuid string)`. Implémentation DuckDB dans `internal/platform/duckdb/catalog_repo.go`.
- **Ajouter le champ `MatchContext`** dans `domain.FilterContextInput` :
  ```go
  // "solo"  : is_with_friends = false
  // "squad" : is_with_friends = true
  // "all"   : pas de filtre supplémentaire (défaut)
  MatchContext string `json:"match_context,omitempty"`
  ```
  Pages escouade → `"squad"` ; pages stats solo → `"all"`. Consommé en Phase I.
- Tests d'isolation cross-titres : `Resolver.CatalogAdapter("synthetic_title_b")` ne doit jamais router vers Halo.

### Phase D — Adapter Halo Infinite + TOML (experience + mode_category) + multi-lang (1 commit)

- Créer `internal/games/halo_infinite/catalog_adapter.go` qui enveloppe [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go).
- **Multi-lang** : à chaque hydratation d'asset, looper sur la liste des langues cibles (~14 langues, cf. §4sexies.3) et appeler `FetchAsset(..., lang)` pour chacune. Upsert dans `asset_translations`. Coût : ~10-15 min au bootstrap initial, négligeable ensuite (refresh mensuel).
- À chaque hydratation de pair, appeler [NormalizeModeLabel](apps/go-api/internal/analysis/mode_label.go) pour chaque langue → upsert dans `pair_mode_label_translations`. Si DiscoveryUGC retourne 404 pour une langue → fallback sur `mode_name_tr` + `mode_pair_overrides`.
- **Migrer `mode_category` en TOML** (§4cinquies actée) : créer `config/titles/halo_infinite/catalog/mode_category_rules.toml`, refactorer [mode_category.go](apps/go-api/internal/analysis/mode_category.go) en loader + cache au boot. Validation au boot : tous les enum Go doivent avoir une règle (panic si désynchronisation).
- Porter la logique de classification d'`experience` (playlist-level) depuis [src/analysis/playlist_groups.py](src/analysis/playlist_groups.py) vers `config/titles/halo_infinite/catalog/experience_rules.toml`.
- **Auto-détection préfixes inconnus** (§4cinquies.1 volet 2) : à l'hydratation, si un préfixe pair_name n'est couvert par aucune règle TOML → INSERT OR REPLACE dans `unknown_prefix_candidates` avec incrément `n_matches`. Quand seuil ≥5 atteint → log warning structuré (`event=unknown_prefix_candidate`). Pas de notification externe.
- Tests : 25-30 playlists permanentes mappées correctement (experience), 50+ pair_names couverts par mode_category (snapshot), validation au boot que TOML couvre tous les enum Go, snapshot multi-lang sur 5 assets de référence, snapshot d'auto-détection sur préfixe inconnu.

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

#### Décision requise — cascade "parents actifs" : Go pur ou SQL ?

| Option | Description | Avantages | Inconvénients |
|---|---|---|---|
| **(A) Go pur** (reco) | `Resolve()` charge l'ensemble filtré depuis `match_registry` (xuid + match_context) puis exécute `buildAvailableOptions` en Go (logique actuelle) | `filters_cascade_test.go` (28 tests) réutilisable sans modification | Plus de lignes en mémoire si catalogue ne réduit pas assez |
| **(B) SQL catalogue** | Requêtes SQL successives par niveau ; `playlists_catalog ⨝ match_registry WHERE id IN (...)` parents actifs | Moins de data en mémoire, exploite les index DuckDB | Cascade dispersée en SQL, tests plus complexes |

**Reco** : Option (A). Phase J+1 si benchmarks insuffisants.

#### Session filter — implémentation cross-DB

`session_label` vit dans `stats.duckdb/player_match_enrichment`, pas dans `match_registry`. JOIN cross-DB explicite dans `FiltersRepo` :

```go
// internal/platform/duckdb/filters_repo.go (lorsque session_ids présents)
//   ATTACH 'data/players/{gamertag}/stats.duckdb' AS player_db (READ_ONLY);
//   SELECT DISTINCT mr.match_id
//   FROM shared.match_registry mr
//   JOIN player_db.player_match_enrichment pme ON pme.match_id = mr.match_id
//   WHERE pme.session_label IN (?) AND mr.title_slug = ? AND mr.xuid = ?
//   DETACH player_db;
```

#### Contexte solo/squad (`match_context`)

```go
switch input.MatchContext {
case "squad": // AND pme.is_with_friends = TRUE
case "solo":  // AND pme.is_with_friends = FALSE
default:      // "all" → pas de filtre
}
```

#### Garde de sécurité — catalogue vide

```go
if catalogCount == 0 {
    slog.ErrorContext(ctx, "catalog empty: falling back to legacy scan", "title_slug", titleSlug)
    return r.legacyLoadMatchesForFilters(ctx, input)
}
```

Fallback transitoire, suppression en Phase K après confirmation prod.

#### Migration des tests existants

Si Option (A) retenue : `filters_cascade_test.go` (28 tests) s'applique sans modification — la source de données change, la cascade Go reste identique. Si Option (B) : remplacer par tests d'intégration DuckDB `:memory:` + fixture catalogue. À mentionner dans le commit message.

#### Logging

- `slog.ErrorContext(ctx, "catalog empty: falling back to legacy scan", "title_slug", titleSlug)`
- `slog.InfoContext(ctx, "FiltersService.Resolve", "source", "catalog", "matches_loaded", n, "duration_ms", ms)`
- `slog.ErrorContext(ctx, "session filter cross-DB attach failed", "err", err, "gamertag", gamertag)`

- Tests : parité fixtures réelles, fallback catalogue vide, session cross-DB, `match_context` solo/squad/all, benchmark avant/après.

### Phase J — Refresh mensuel (1 commit, optionnel selon cadence)

- Soit cron OS (instructions doc dans `docs/`), soit goroutine dans [server.go](apps/go-api/internal/api/server.go) avec `time.Ticker(30*24*time.Hour)`.
- Drain de la queue + re-fetch des `is_active = true` pour tous les `title_slug` enregistrés (toutes langues).

### Phase K — Cleanup post-migration (1 commit, conditionnel)

À ouvrir uniquement après confirmation que l'UI consomme exclusivement le catalogue (typiquement quelques semaines après Phase I).

- Supprimer les colonnes `match_registry.{pair_name_fr, map_name_fr, playlist_name_fr}` (jamais peuplées, info portée par catalogue via FK).
- Auditer le code Python `resolve_display_mode`, `resolve_weapon_display`, etc. pour identifier les call sites encore actifs côté UI legacy. Migration ou suppression selon le statut Streamlit/Go de chaque page.
- Migration des matchs Python historiques `mode_category='Fiesta'` → `'SuperFiesta'` ou `'HuskyRaid'` selon le préfixe pair_name (cf. décision ouverte #9). UPDATE one-shot avec snapshot avant/après.
- Tests : aucune régression UI, parité d'affichage avant/après cleanup, comptes catégories cohérents.

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
| 8 | Faut-il exposer `mode_label` comme dimension de filtre, ou seulement `mode_category` | **Tranché 2026-04-30 (§4quater)** : exposer les deux comme dimensions parallèles. La catégorie n'est pas une "famille de mode" mais une "famille de playlist" (Arena/Tactical/Assault → Assassin) ; un filtre `mode_label = 'Slayer'` est nécessaire pour regrouper les ~561 matchs Slayer-likes répartis sur 3 catégories. |
| 9 | Migration des matchs Python "Fiesta" vers les bonnes catégories Go (SuperFiesta, HuskyRaid) | **Hors scope catalogue** mais dépendance à signaler. Cf. §4ter.7 : 78 % des `mode_category='Fiesta'` actuels sont en réalité Super Fiesta. Migration one-shot à intégrer dans la phase qui passe le sync en Go. |
| 10 | Hydratation FR au catalogue : double fetch `lang=en` + `lang=fr`, ou fetch EN seul + fallback sur `mode_name_tr`/`mode_pair_overrides` | **Tranché 2026-04-30 (§4sexies)** : multi-langues complet (~14 langues DiscoveryUGC), persistance dans `asset_translations` + `pair_mode_label_translations`. Schéma rendu DRY (une seule colonne `name_canonical` inline + JOIN sur tables traductions). |
| 11 | Format des règles `mode_category` : code Go enum-like vs TOML | **Tranché 2026-04-30 (§4cinquies)** : option (c) — TOML `mode_category_rules.toml` + auto-détection des préfixes inconnus via table `unknown_prefix_candidates` + log warning structuré au seuil ≥5 + endpoint admin pour consultation. Validation au boot pour cohérence enum Go ↔ TOML. Pas de notification externe (décision séparée si besoin émerge). |
| 12 | Architecture i18n : colonnes inline vs table dédiée | **Tranché 2026-04-30 (§4sexies)** : `name_canonical` (EN) inline pour debug + lookup rapide, autres langues dans `asset_translations` (existante, multi-lang native), labels normalisés dans `pair_mode_label_translations` (nouvelle, dédiée). Documenté dans skill `halo-i18n` pour discoverability. |
| 13 | Cleanup des colonnes `_fr` de `match_registry` | **Tranché 2026-04-30 (§4sexies.5)** : Phase K conditionnelle, après confirmation UI migrée. Pas bloquant pour le catalogue. |

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
- **Exposition `mode_label` / `mode_category` comme dimensions de filtre dans l'UI React** : ce sprint pose les fondations backend (Phase C, Phase H endpoints) ; intégration FilterOmnibar reportée au sprint UI suivant.
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
- [apps/go-api/internal/analysis/mode_category.go](apps/go-api/internal/analysis/mode_category.go) — `InferModeCategoryFromPairName(pairName)` consommée par l'adapter Halo (Phase D), refactoré en Phase D pour charger les règles depuis TOML
- Skill projet : `.claude/skills/halo-modes/SKILL.md` — conventions de normalisation (2 niveaux orthogonaux, divergences assumées vs Python v7)
- Skill projet : `.claude/skills/halo-i18n/SKILL.md` — **où sont les traductions, quelles langues, anti-patterns à éviter**. Référence prioritaire pour tout PR touchant aux traductions.

### Code Python à porter (logique éprouvée)
- [src/analysis/playlist_groups.py](src/analysis/playlist_groups.py) — Heuristiques de classification `experience` (à porter vers TOML Halo)
- [src/data/domain/refdata.py](src/data/domain/refdata.py) — Enum `GameVariantCategory` (à porter vers `internal/games/canonical/`)

### Pattern Kinds (référence d'inspiration architecture)
- [resolver_default.go](apps/go-api/internal/assets/resolver_default.go), [kinds.go](apps/go-api/internal/assets/kinds.go), [fetcher_chain.go](apps/go-api/internal/assets/fetcher_chain.go)

### Sources externes
- [Halo Waypoint — Multiplayer Playlists](https://support.halowaypoint.com/hc/en-us/articles/17920041655188-Halo-Infinite-Multiplayer-Playlists)
- [den.dev — Halo Infinite Playlist Weights](https://den.dev/blog/halo-infinite-playlist-weights/)

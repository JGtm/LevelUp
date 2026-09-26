# PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md — Plan de support multi-titres : adaptateurs structurels + mappings sémantiques TOML

> Plan rédigé le 2026-04-25 dans le contexte du Sprint 44 (multi-titres) déjà acté.
>
> Ce document complète les cadrages existants :
> - `.ai/go_migration_v2/HALO_CANONICAL_MODEL.md` (modèle canonique stable)
> - `.ai/go_migration_v2/HALO_INFINITE_CAPABILITY_MAP.md` (capabilities mono-titre HI)
> - `.ai/go_migration_v2/ADR_S44_MULTI_TITLE_NAMESPACE.md` (namespace par titre)
> - `.ai/go_migration_v2/HALO_PRODUCT_CONTRACT_ADAPTERS.md` (adapters produit côté HTTP)
>
> Il fixe la **couche pratique** au-dessus du canonique : comment, concrètement, on isole en code et en config les différences structurelles et sémantiques entre titres, sans casser Halo Infinite.

---

## TL;DR

1. **Trois couches strictement séparées** : Stockage DuckDB par titre / Adaptateurs Go par titre / Mapping sémantique TOML.
2. **Deux interfaces Go par titre** (SRP) : `TitleDataAdapter` (lecture canonique services) + `TitleSemanticAdapter` (libellés). Injectées à partir du `title_slug` du contexte session.
3. **TOML versionnés Git sous `config/titles/{slug}/mappings/`** (séparé de `data/` runtime) ; loader Go strict au boot, validation locales/formats/collisions/conversions d'unités.
4. **Endpoint `GET /api/v1/titles/{slug}/field-mappings?locale=fr`** (versionné OpenAPI, derrière feature flag en Phase A) que le frontend consomme au boot, avec hook `useFieldLabel(key)`.
5. **Bascule incrémentale endpoint par endpoint avec golden parity HI strict (diff = 0)**, ordre du moins risqué au plus risqué : `/career/encounters` → `/synthesis` → `/home` → `/match-view`.
6. **Corpus synthétique titre B (~1j)** pour valider l'agnosticité des services et l'isolation cross-titres.
7. **Logs slog** structurés à toutes les frontières + rate-limit borné + tests fuzz/property-based + lint CI anti-régression.
8. **Effort estimé** : 14–20 jours-personne sur ~3 sprints solo, ou ~1.5 sprint à 2 devs.
9. **Décisions tranchées** (§16, validées 2026-04-25). Plan prêt pour Phase A.

---

## 1. Problème posé

Le runtime LevelUp est aujourd'hui mono-titre `halo_infinite`. La décision multi-titres a été actée (ADR_S44) mais sans plan opératoire pour deux questions distinctes :

1. **Couche sémantique** : les libellés, descriptions, unités, formats d'affichage et ordres d'affichage des champs et assets sont aujourd'hui éparpillés dans le code Go, le i18n React et la metadata DuckDB. Si un futur titre rebaptise « frags » en « éliminations », il faut pouvoir le faire sans toucher au SQL ni à la logique.
2. **Couche structurelle** : les schémas de tables, les noms de colonnes, les types et les unités peuvent diverger entre titres. Halo Infinite stocke des durées en secondes ; un futur titre pourrait stocker en millisecondes. Halo Infinite expose `kills` ; un autre titre pourrait exposer `eliminations`.

Le canonique défini dans `HALO_CANONICAL_MODEL.md` couvre la **frontière entrée** (provider de titre → produit), pas la **lecture DuckDB** côté services produit. Il manque donc deux couches en aval :

1. la couche **adaptateur** qui transforme les données **stockées** (DuckDB de chaque titre) en schéma canonique consommé par les services produit ;
2. la couche **mapping sémantique** qui décrit, par titre, comment afficher chaque champ et chaque asset.

Et un constat opérationnel : aujourd'hui les libellés FR/EN proviennent d'une mixture (DB `mode_lang_settings`, `mode_name_tr`, `weapon_labels` + i18n React `apps/web/src/i18n/` + strings hardcodés Go). Cette dualité doit être tranchée — voir §16.

---

## 2. Périmètre

### 2.1. Inclus dans ce plan

1. Définition d'un schéma canonique stable côté **services Go** (au-delà du canonique provider).
2. Conception de deux interfaces Go par titre (`TitleDataAdapter` + `TitleSemanticAdapter`) avec implémentations `halo_infinite/`.
3. Conception d'un format TOML de mapping sémantique par titre, avec conversions d'unités et mappings d'enums.
4. Loader Go typé qui charge les TOML au boot et les valide.
5. Exposition au frontend via un endpoint OpenAPI versionné pour qu'il consomme les libellés sans hardcode.
6. Stratégie de logging structuré (slog) à toutes les frontières, avec rate-limit borné.
7. Stratégie de tests unitaires (fuzz + property-based) + non-régression + golden parity.
8. Plan de bascule progressive Halo Infinite -> couche adaptateur sans régression, derrière feature flags.

### 2.2. Exclu

1. Implémentation effective d'un second titre (corpus synthétique seulement, voir §11).
2. Refonte du modèle canonique provider (`HALO_CANONICAL_MODEL.md`) — déjà figé.
3. Sélecteur UI de titre (déjà tranché : plomberie sans bouton dans le S44).
4. Auth multi-titres (titre-agnostique par construction).
5. Migration des chemins de stockage (déjà couvert par `ADR_S44_MULTI_TITLE_NAMESPACE.md`).

---

## 3. Architecture cible

### 3.1. Vue d'ensemble

```text
+----------------------------------------------------------------+
| Frontend React                                                  |
|   - lit libellés via /api/v1/titles/{slug}/field-mappings       |
|   - aucune string hardcodée pour les champs métier              |
+----------------------+-----------------------------------------+
                       |
                       v
+----------------------------------------------------------------+
| API Go — handlers                                               |
|   - assemblent dépendances                                      |
|   - lisent uniquement le schéma canonique services              |
+----------------------+-----------------------------------------+
                       |
                       v
+----------------------------------------------------------------+
| Services produit (career, home, match view, ...)                |
|   - consomment le schéma canonique services                     |
|   - n'ont jamais de SQL spécifique à un titre                   |
+----------------------+-----------------------------------------+
                       |
            +----------+----------+
            v                     v
+---------------------+   +------------------------+
| TitleDataAdapter    |   | TitleSemanticAdapter   |
| (lecture canonique) |   | (libellés TOML)        |
+----------+----------+   +-----------+------------+
           |                          |
   +-------+--------+                 |
   v                v                 v
+--------+   +-------------+   +--------------------------------+
| DuckDB |   | Fetcher     |   | TOML mappings (versionnés Git)  |
| pool   |   | live (SPNKr |   | config/titles/{slug}/mappings/  |
| titre  |   | / sync)     |   |                                 |
+--------+   +-------------+   +--------------------------------+
```

Note : le **fetcher live** (provider Halo via SPNKr, etc.) reste séparé de la lecture DuckDB. L'adapter peut consommer les deux selon le service appelant : un service `home` lira majoritairement DuckDB, un service `live_challenges` ira chercher live.

### 3.2. Trois couches strictement séparées

| Couche | Responsabilité | Source de vérité |
|---|---|---|
| **Stockage** | Schémas DuckDB par titre, libres de diverger | `data/titles/{slug}/warehouse/*.duckdb` |
| **Adaptateur structurel** | Transforme du stockage vers le canonique services | `internal/games/{slug}/adapter.go` |
| **Mapping sémantique** | Libellés, unités, format, ordre d'affichage | `config/titles/{slug}/mappings/*.toml` (versionné Git) |

Règle d'or : **un service produit ne lit jamais de TOML directement et ne touche jamais à DuckDB directement**. Il passe toujours par un `TitleDataAdapter` (lecture canonique) ou `TitleSemanticAdapter` (libellés) injecté.

> **Note sur l'emplacement des TOML** : les mappings vivent sous `config/titles/` et **non** sous `data/titles/`. Raison : `data/` regroupe les artefacts runtime (DuckDB, caches, sessions) souvent gitignorés ; `config/` regroupe le code de configuration versionné. La migration `data/titles/{slug}/warehouse/` et `data/titles/{slug}/players/` reste sous `data/` (cf. ADR_S44).

---

## 4. Schéma canonique services (au-delà du provider)

Le canonique du provider (cf. `HALO_CANONICAL_MODEL.md`) couvre l'entrée des données. Il faut un canonique **services** : la forme stable consommée par les services produit après lecture DuckDB.

### 4.1. Emplacement

```
apps/go-api/internal/games/canonical/
  ├── identity.go        # PlayerStats, PlayerProfile, PlayerIdentity
  ├── match.go           # MatchSummary, MatchDetail, Participant
  ├── career.go          # CareerSnapshot, RankProgression
  ├── timeseries.go      # TimeBucket, MetricSeries
  ├── enums.go           # Outcome, MatchType, RatingType (canoniques)
  ├── scopes.go          # StatsScope, TimeseriesQuery (paramètres lecture)
  └── fields.go          # FieldKey enum (référencé par les TOML)
```

> **Compatibilité avec l'existant** : `internal/domain/` contient déjà des enums comme `Outcome`. La migration ré-exporte ces enums depuis `canonical/enums.go` et déprécie progressivement `domain/` pour ce qui est titre-aware. Pas de duplication ; `domain/` continue d'héberger les types non-titre-aware (sessions, performance score, citations, qui restent calculés au-dessus du canonique services).

### 4.2. Convention de nommage canonique

Un `FieldKey` est un identifiant stable, machine-friendly, en `snake_case`, agnostique du titre :

```go
type FieldKey string

const (
    FieldKills           FieldKey = "kills"
    FieldDeaths          FieldKey = "deaths"
    FieldAssists         FieldKey = "assists"
    FieldHeadshotKills   FieldKey = "headshot_kills"
    FieldDamageDealt     FieldKey = "damage_dealt"
    FieldDamageTaken     FieldKey = "damage_taken"
    FieldShotsFired      FieldKey = "shots_fired"
    FieldShotsHit        FieldKey = "shots_hit"
    FieldAccuracy        FieldKey = "accuracy"
    FieldDurationSeconds FieldKey = "duration_seconds"
    // ... voir annexe §17 pour la liste exhaustive initiale (~40 FieldKey couvrant Match View + Home + Career)
)
```

### 4.3. Liste initiale exhaustive

Pour éviter un bikeshedding sans fin pendant la Phase A, la liste initiale est figée en annexe §17. Elle couvre **uniquement** les champs aujourd'hui exposés par les endpoints `home`, `match-view`, `career`, `synthesis`. Tout ajout ultérieur passe par la règle §4.4.

### 4.4. Règle d'extension

L'ajout d'un nouveau `FieldKey` :

1. doit être motivé par un besoin produit, pas par un endpoint provider ;
2. doit être documenté dans `HALO_CANONICAL_MODEL.md` puis ici ;
3. doit avoir une entrée dans le TOML de **chaque** titre où il s'applique, ou être marqué `unsupported` ;
4. ne doit jamais réutiliser un nom déjà utilisé pour un autre concept (test garde-fou) ;
5. doit avoir un test couvrant son transform dans au moins un adapter.

---

## 5. Adapters par titre — deux interfaces séparées (SRP)

Pour respecter la règle de responsabilité unique (CLAUDE.md §17), deux interfaces distinctes plutôt qu'une `GameAdapter` monolithique.

### 5.1. `TitleDataAdapter` — accès données canoniques

```go
// internal/games/data_adapter.go
package games

type TitleDataAdapter interface {
    // Identité
    TitleSlug() string
    Capabilities() CapabilityMap

    // Lecture canonique (DuckDB ou live selon implémentation)
    LoadMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error)
    LoadMatchDetail(ctx context.Context, matchID string) (*canonical.MatchDetail, error)
    LoadPlayerStats(ctx context.Context, xuid string, scope canonical.StatsScope) (*canonical.PlayerStats, error)
    LoadCareerSnapshot(ctx context.Context, xuid string, opts canonical.CareerOptions) (*canonical.CareerSnapshot, error)
    LoadTimeseries(ctx context.Context, xuid string, query canonical.TimeseriesQuery) (*canonical.MetricSeries, error)
}
```

### 5.2. `TitleSemanticAdapter` — libellés et présentation

```go
// internal/games/semantic_adapter.go
package games

type TitleSemanticAdapter interface {
    TitleSlug() string
    SchemaVersion() int

    Fields() FieldMappingSet
    Assets() AssetMappingSet
    Outcomes() OutcomeMappingSet
}
```

### 5.3. `StatsScope`, `TimeseriesQuery`, `CareerOptions`

Définis dans `canonical/scopes.go`, immutables (`@dataclass(frozen)` équivalent Go : struct sans pointeurs, fonctions accessor).

```go
type StatsScope struct {
    From         time.Time
    To           time.Time
    PlaylistIDs  []string  // optionnel
    IncludePvE   bool
    IncludeRanked *bool    // nil = pas de filtre ; true/false = filtre actif
}

type TimeseriesQuery struct {
    Metric    canonical.FieldKey
    Bucket    canonical.Bucket  // day, week, month
    From, To  time.Time
    GroupBy   []canonical.GroupBy  // playlist, mode, ...
}

type CareerOptions struct {
    IncludeHistory bool
    HistoryLimit   int  // 0 = pas de limite raisonnable, default 50
}
```

### 5.4. Résolution via `TitleRegistry`

Le `TitleRegistry` (cf. `ADR_S44_MULTI_TITLE_NAMESPACE.md`) expose deux résolveurs :

```go
type Resolver interface {
    Data(titleSlug string) (TitleDataAdapter, error)
    Semantic(titleSlug string) (TitleSemanticAdapter, error)
    Default() string  // slug du titre default ("halo_infinite")
}
```

Les services produit ne connaissent jamais le slug : ils reçoivent les adapters via injection à partir du `title_slug` du contexte de session (middleware Go `internal/api/middleware/title_context.go`).

### 5.5. Implémentation Halo Infinite

```
apps/go-api/internal/games/halo_infinite/
  ├── data_adapter.go         # implémente TitleDataAdapter
  ├── semantic_adapter.go     # implémente TitleSemanticAdapter (wrappe le loader TOML)
  ├── queries_match.go        # SQL spécifique HI (déplacé depuis platform/duckdb)
  ├── queries_career.go
  ├── queries_home.go
  ├── transforms.go           # transformations colonnes -> FieldKey + conversions d'unités
  ├── transforms_test.go
  └── adapter_test.go
```

L'adapter encapsule **tout** ce qui aujourd'hui s'appelle `queries_match.go`, `queries_career.go`, etc. dans `apps/go-api/internal/platform/duckdb/`. Les tables et colonnes restent inchangées ; seul le possesseur des requêtes change.

### 5.6. Accès au pool DuckDB

Le `TitleDataAdapter` reçoit une référence au pool DuckDB **du titre courant** via DI (`PlayerDB` + `Metadata` du titre, déjà title-aware d'après ADR_S44). L'adapter ne crée jamais de connexion bare ; il passe par les helpers existants `duckdb_read_only(path)` / `duckdb_read_write(path)`.

### 5.7. Capability gating

Si un service appelle `LoadTimeseries()` sur un adaptateur dont les capabilities n'exposent pas la timeseries, l'adapter doit :

1. retourner une erreur typée `ErrCapabilityNotSupported` (variable sentinelle exportée) ;
2. logguer en `slog.Warn` avec `title_slug`, `capability`, `caller` (extrait via `runtime.Caller(1)`) ;
3. ne **jamais** retourner des données vides silencieuses.

C'est la transposition pratique de la règle « représenter les absences explicitement » du modèle canonique.

### 5.3. Implémentation Halo Infinite

```
apps/go-api/internal/games/halo_infinite/
  ├── adapter.go          # implémente GameAdapter
  ├── queries.go          # SQL spécifique HI (déplacé depuis platform/duckdb)
  ├── transforms.go       # transformations colonnes -> FieldKey
  ├── transforms_test.go  # tests unitaires des transforms
  └── adapter_test.go     # tests d'intégration sur fixtures
```

L'adapter encapsule **tout** ce qui aujourd'hui s'appelle `queries_match.go`, `queries_career.go`, etc. dans `apps/go-api/internal/platform/duckdb/`. Le pool DuckDB reste partagé ; ce qui change est qui possède les requêtes.

### 5.4. Capability gating

Si un service appelle `LoadTimeseries()` sur un adaptateur dont les capabilities n'exposent pas la timeseries, l'adapter doit :

1. retourner une erreur typée `ErrCapabilityNotSupported` ;
2. logguer en `slog.Warn` avec `title_slug`, `capability`, `caller` ;
3. ne **jamais** retourner des données vides silencieuses.

C'est la transposition pratique de la règle « représenter les absences explicitement » du modèle canonique.

---

## 6. Format TOML des mappings sémantiques

### 6.1. Localisation

```
config/titles/{slug}/mappings/
  ├── fields.toml        # libellés des FieldKey + format/unité/ordre
  ├── assets.toml        # libellés des assets (modes, playlists, ranks) — voir §6.7 pour les médailles
  ├── outcomes.toml      # libellés des Outcome enum
  └── capabilities.toml  # statuts capability map (mirror du go code, source-of-truth)
```

Ces fichiers sont **versionnés Git** sous `config/` (séparé de `data/` qui contient les artefacts runtime). Raisons :

1. diff Git lisible et reviewable ;
2. pas de cycle « modifier la DB pour redéployer le label » ;
3. reproductibilité builds : checkout de commit donne TOML cohérents avec le code.

> **Schema VSCode** : un `config/titles/_schema/fields.schema.json` accompagne les TOML pour autocompletion et validation à l'édition (extension Even Better TOML). Permet d'éditer sans avoir à relire ce doc.

### 6.2. Schéma `fields.toml`

```toml
# config/titles/halo_infinite/mappings/fields.toml

[meta]
title_slug     = "halo_infinite"
schema_version = 1

# Les sections sont indexées par FieldKey canonique.
# Une section absente = le titre ne supporte pas ce champ.

[fields.kills]
labels        = { en = "Kills", fr = "Éliminations" }
description   = { en = "Total kills in the match.", fr = "Total des éliminations dans le match." }
storage_unit  = "count"     # unité dans la colonne DuckDB
display_unit  = "count"     # unité après conversion vers le canonique
format        = "integer"
display_order = 10
group         = "combat"
icon          = "icons/combat/kills.svg"

[fields.deaths]
labels        = { en = "Deaths", fr = "Morts" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 20
group         = "combat"

[fields.accuracy]
labels        = { en = "Accuracy", fr = "Précision" }
storage_unit  = "ratio"      # 0..1 dans DuckDB
display_unit  = "percent"    # 0..100 après conversion
format        = "percent_1"
display_order = 30
group         = "combat"

[fields.duration_seconds]
labels        = { en = "Duration", fr = "Durée" }
storage_unit  = "seconds"
display_unit  = "seconds"
format        = "duration_hms"
display_order = 5
group         = "match"
```

**Champs obligatoires** : `labels.en`, `labels.fr`, `storage_unit`, `display_unit`, `format`, `display_order`, `group`.
**Champs optionnels** : `description`, `icon`, `description.fr`, `description.en` (mais si l'un est défini, les deux le sont).

L'adapter applique automatiquement la conversion `storage_unit -> display_unit` lors du transform vers le canonique. L'enum des conversions supportées vit dans `internal/games/mappings/units.go`.

### 6.3. Schéma `assets.toml`

```toml
[meta]
title_slug   = "halo_infinite"
schema_version = 1

# Indexé par asset_kind puis asset_id.

[assets.medal."ec5d10a7-..."]
labels = { en = "Killing Spree", fr = "Carnage" }
icon_url = "/static/medals/killing_spree.png"
tier     = "bronze"

[assets.game_mode."ctf"]
labels = { en = "Capture the Flag", fr = "Capture du drapeau" }
icon   = "icons/modes/ctf.svg"
```

### 6.4. Schéma `outcomes.toml`

```toml
[outcome.win]
labels = { en = "Win", fr = "Victoire" }
color_token = "outcome.positive"

[outcome.loss]
labels = { en = "Loss", fr = "Défaite" }
color_token = "outcome.negative"

[outcome.tie]
labels = { en = "Tie", fr = "Égalité" }
color_token = "outcome.neutral"
```

### 6.5. Pourquoi TOML et pas JSON / YAML

| Critère | TOML | JSON | YAML |
|---|:---:|:---:|:---:|
| Lecture humaine | excellent | moyen | bon |
| Diff Git | excellent | bon | piégeux (indentation) |
| Commentaires | oui | non | oui |
| Parsing Go strict | excellent (`pelletier/go-toml/v2`) | natif | risqué (espaces) |
| Edition non-dev | bonne | mauvaise | risquée |

TOML gagne pour ce cas : éditable par un PM ou un i18n owner sans casser le parsing.

### 6.6. Validation au chargement

Au démarrage Go, pour chaque titre :

1. parser via `go-toml/v2` dans une struct typée ;
2. vérifier que chaque section `[fields.X]` a un `FieldKey` valide enregistré ;
3. vérifier que les locales `en` et `fr` sont présentes pour chaque label (locales obligatoires, alignées sur le projet) ;
4. vérifier qu'aucun `display_order` ne se collisionne dans un même `group` ;
5. vérifier que tout `format` est dans l'enum (`integer`, `percent_1`, `percent_2`, `kdr_2`, `duration_hms`, `seconds`, `signed_int`, `string`) ;
6. vérifier que toute conversion `storage_unit -> display_unit` est dans `units.go` ;
7. vérifier que les `asset_id` référencés dans `assets.toml` existent dans la DB metadata correspondante (via `information_schema.tables`) — règle stricte pour modes/playlists, **lazy** pour les médailles (cf. §6.7).

Toute erreur = `slog.Error` + refus de boot du titre concerné (les autres titres restent disponibles).

### 6.7. Cas particulier des médailles

Halo Infinite a ~150 médailles, identifiées par UUID, déjà référencées dans `metadata.duckdb.medal_definitions`. Mettre toutes les médailles dans `assets.toml` est lourd à maintenir.

**Décision** : les médailles **restent en DB** (`medal_definitions` + `medal_translations` à créer si besoin). Le `TitleSemanticAdapter.Assets()` lit depuis la DB pour les médailles, depuis le TOML pour les modes/playlists/ranks.

**Pourquoi** :

1. les médailles sont énumérées par l'API Halo, pas définies par le produit ;
2. ajout d'une médaille = sync metadata, pas un PR Git ;
3. le wording produit (titre, description) est principalement descriptif, pas opinionné.

**Ce que TOML couvre quand même** : les `medal_tier_label` (`bronze`, `silver`, `gold`, `mythic`), les couleurs et icons-overlays, les **regroupements UX** (ex : `combat`, `objective`, `style`).

### 6.8. Articulation avec l'i18n DB legacy

Le projet a déjà des libellés en DB :

| Source | Contenu | Décision |
|---|---|---|
| `metadata.mode_lang_settings` | langue par mode | **migré vers TOML** (mode = asset stable) |
| `metadata.mode_name_tr` | traductions modes | **migré vers TOML** |
| `metadata.weapon_labels` | labels armes par filmshell ID | **reste en DB** (volume + provenance API) |
| `metadata.career_ranks` (titres) | titres rangs FR/EN | **migré vers TOML** (assets.toml `[assets.career_rank]`) |
| `medal_definitions` | médailles | **reste en DB** (cf. §6.7) |

La migration `mode_lang_settings`/`mode_name_tr` -> TOML est faite en Phase A par un script `tools/dump-modes-to-toml.go` qui lit la DB et émet le TOML initial. Suppression effective après bascule complète.

### 6.9. Articulation avec l'i18n React (apps/web/src/i18n/)

Le i18n React continue de couvrir **tout ce qui n'est pas un FieldKey/asset/outcome** : labels de boutons, messages d'erreur, tooltips, navigation, copy marketing, etc.

**Frontière nette** :

| Couvert par TOML backend | Couvert par i18n React |
|---|---|
| Libellé d'un FieldKey métier (`kills`, `deaths`) | Bouton "Sauvegarder" |
| Description de la médaille | Message d'erreur "Connexion perdue" |
| Nom d'un mode (`Capture the Flag`) | Titre de la page Settings |
| Outcome (`Victoire`, `Défaite`) | Tooltip "Cliquez pour filtrer" |

Un test CI (mode audit, cf. §12.7) détecte les chevauchements (clés présentes des deux côtés).

---

## 7. Loader Go typé

### 7.1. Emplacement

```
apps/go-api/internal/games/mappings/
  ├── loader.go         # parse + valide + construit FieldMappingSet
  ├── loader_test.go    # tests unitaires sur fixtures TOML valides + invalides
  ├── types.go          # FieldMappingSet, AssetMappingSet, OutcomeMappingSet
  ├── format.go         # implémentation des formats (integer, percent_1, ...)
  └── format_test.go    # tests unitaires des formatters
```

### 7.2. Contrat

```go
type FieldMappingSet interface {
    Get(key canonical.FieldKey) (FieldMapping, bool)
    All() []FieldMapping
    OrderedByGroup() map[string][]FieldMapping
}

type FieldMapping struct {
    Key           canonical.FieldKey
    Labels        map[string]string
    Descriptions  map[string]string
    StorageUnit   Unit
    DisplayUnit   Unit
    Format        Format
    DisplayOrder  int
    Group         string
    Icon          string
}

// Label retourne le libellé pour la locale demandée + un bool de fallback.
// usedFallback=true si la locale demandée n'existait pas et qu'on est tombé sur "en" ou la clé.
// Permet aux callers de logguer field_lookup_missing quand ça compte.
func (m FieldMapping) Label(locale string) (label string, usedFallback bool) { ... }

// FormatValue applique le Format après conversion StorageUnit -> DisplayUnit.
func (m FieldMapping) FormatValue(v any) (string, error) { ... }
```

### 7.3. Cache et hot-reload

1. Chargement unique au boot dans le `TitleRegistry`.
2. Mode dev : flag `GAMES_HOT_RELOAD=true` -> watcher fs + rechargement par titre + log structuré + invalidation cache HTTP via signal vers handler.
3. Mode prod : pas de hot-reload. Redéploiement requis pour changer les TOML.

> **Choix volontaire** : pas de TOML « live editing » en prod. Justification : la couche sémantique fait partie du contrat versionné — un changement passe par PR + golden parity. Coût : un redéploiement pour changer un libellé. Acceptable au regard du volume (quelques dizaines de FieldKey), pas acceptable pour un futur catalogue de centaines de cosmétiques (qui resteront en DB, cf. §6.7).

### 7.4. Exposition au frontend

Endpoint dédié, **versionné OpenAPI** :

```
GET /api/v1/titles/{slug}/field-mappings?locale=fr
```

Schéma déclaré dans `apps/go-api/internal/api/openapi/openapi.yaml` (section v1) avec discipline OpenAPI du projet (cf. `OPENAPI_MVP_P0_P1.md`). Toute évolution breaking de la réponse passe par `/api/v2/...` (rare, à éviter ; préférer extensions optionnelles).

Réponse :

```json
{
  "title_slug": "halo_infinite",
  "schema_version": 1,
  "locale": "fr",
  "fields": {
    "kills": {
      "label": "Éliminations",
      "description": "Total des éliminations dans le match.",
      "unit": "count",
      "format": "integer",
      "display_order": 10,
      "group": "combat",
      "icon": "icons/combat/kills.svg"
    }
  },
  "outcomes": { ... },
  "assets": { ... }
}
```

Le frontend consomme cet endpoint au boot et cache via TanStack Query (clé `["field-mappings", titleSlug, locale]`). Aucune string métier hardcodée côté React pour les titres ; un test ESLint custom (ou un grep CI) interdit `t('Kills')` au profit de `useFieldLabel('kills')`.

---

## 8. Logging structuré (slog)

### 8.1. Événements à émettre

| Événement | Niveau | Champs obligatoires |
|---|---|---|
| `adapter_loaded` | `Info` | `title_slug`, `provider`, `capabilities_count` |
| `adapter_load_failed` | `Error` | `title_slug`, `error`, `phase` |
| `mappings_loaded` | `Info` | `title_slug`, `fields_count`, `assets_count`, `schema_version` |
| `mappings_validation_failed` | `Error` | `title_slug`, `file`, `field`, `reason` |
| `mappings_hot_reloaded` | `Info` | `title_slug`, `file`, `duration_ms` |
| `field_lookup_missing` | `Warn` | `title_slug`, `field_key`, `caller` |
| `capability_not_supported` | `Warn` | `title_slug`, `capability`, `caller` |
| `field_mappings_served` | `Debug` | `title_slug`, `locale`, `cache_hit` |

### 8.2. Politique

1. Toute frontière adapter <-> service log un `Debug` avec `title_slug` injecté via context (helper `slog.With("title_slug", ...)`).
2. Tout fallback de label log un `Warn` une seule fois par couple `(title, key, locale)` puis suppress.
3. Aucune donnée joueur dans les logs adapter (pas de `xuid`, pas de `gamertag`) — déjà la règle générale du projet.

### 8.3. Rate-limiting des logs

Le rate-limit vit dans un `sync.Map` borné par titre (max 1024 entrées par titre, LRU lazy avec eviction sur 75% de remplissage). Au-delà, un compteur global émet périodiquement un `mappings_lookup_throttled` toutes les 5 minutes avec `dropped_count`. Évite le flooding tout en gardant la visibilité.

```go
type lookupRecorder struct {
    seen   sync.Map  // (title, key, locale) -> struct{}
    bound  int       // 1024
    dropped atomic.Int64
}
```

Test dédié : `recorder_test.go` vérifie que 10000 lookups sur la même clé émettent 1 seul `Warn`.

---

## 9. Tests unitaires

### 9.1. Adapters Halo Infinite

| Fichier | Couverture |
|---|---|
| `data_adapter_test.go` | chemins nominaux des `Load*` + erreurs typées (`ErrCapabilityNotSupported`, DB absente, ctx canceled) |
| `semantic_adapter_test.go` | exposition correcte de `Fields()`, `Assets()`, `Outcomes()` ; cohérence avec le loader |
| `transforms_test.go` | chaque transform colonne DuckDB -> FieldKey + conversion `storage_unit`->`display_unit`, edge cases (NULL, 0, types numériques limites) |
| `queries_test.go` | requêtes SQL via golden fixtures DuckDB, comparées à snapshots JSON |

Exigence : couverture **>= 85%** sur le package `internal/games/halo_infinite/`.

### 9.2. Loader TOML

| Fichier | Couverture cible |
|---|---|
| `loader_test.go` | parsing + validation : 1 fixture valide HI, 8 fixtures invalides (locale manquante, format inconnu, FieldKey inconnu, collision display_order, schema_version absent, fichier vide, conversion d'unité non supportée, asset_id orphelin) |
| `loader_fuzz_test.go` | fuzz Go natif (`go test -fuzz`) sur le parser : aucune entrée TOML ne doit panic, toutes les erreurs sont typées |
| `format_test.go` | chaque format : integer, percent_1, percent_2, kdr_2, duration_hms, seconds, signed_int, string, NULL/0/négatifs/Inf/NaN |
| `format_property_test.go` | property-based via [pgregory.net/rapid](https://pgregory.net/rapid) : `format(parse(format(x))) == format(x)` pour les formats numériques |

Exigence : couverture **>= 90%** (logique pure, peu d'IO).

### 9.3. FieldKey garde-fous

Test dédié `internal/games/canonical/fields_test.go` :

1. Liste exhaustive des `FieldKey` exposés.
2. Vérifie qu'aucune valeur ne change entre commits (regression test sur un golden file `fields.golden.txt`).
3. Vérifie qu'aucun titre n'introduit un FieldKey absent du canonique central.

### 9.4. Endpoint `/field-mappings`

| Type | Cas |
|---|---|
| Unitaire handler | locale connue, locale inconnue (fallback en), titre inconnu (404), titre dégradé |
| Contrat | snapshot JSON pour `halo_infinite` + locale `fr` et `en` (golden) |
| Cache | header `Cache-Control` correct, ETag stable tant que TOML inchangé |

### 9.5. Frontend

| Type | Cas |
|---|---|
| Vitest unitaire | hook `useFieldLabel('kills')` rend `Éliminations` en FR, `Kills` en EN, `kills` en fallback |
| Vitest integration | composant `<MatchScoreboard />` rend les colonnes dans l'ordre `display_order` |
| Lint custom | script Node `tools/lint-no-hardcoded-fields.mjs` parsé en CI : détecte les littéraux correspondant à des FieldKey (`'Kills'`, `'Eliminations'`, ...) en dehors des fichiers de test ; échoue avec un message pédagogique pointant vers `useFieldLabel(key)` |

Le système i18n React utilisé est **react-i18next** (cf. `apps/web/src/i18n/`). Les FieldKey/assets/outcomes ne doivent **pas** apparaître dans les fichiers `apps/web/src/i18n/locales/*.json` après bascule complète (vérifié par script CI).

---

## 10. Tests de non-régression

### 10.1. Golden parity Halo Infinite

Avant la bascule vers la couche adapter, capturer un snapshot de réponses pour les endpoints clés. **Les fixtures sont synthétiques** (gamertags `Player_A`, `Player_B`, ... ; matchs avec UUIDs déterministes) pour garantir reproductibilité et préserver la privacy :

```
testdata/golden/halo_infinite/
  ├── home_player_a.json
  ├── match_view_00000000-0000-0000-0000-000000000001.json
  ├── career_player_a.json
  ├── synthesis_player_a.json
  └── timeseries_player_a.json
```

Après bascule, un test `go test -tags=goldenparity` rejoue et **diff = 0** sur ces snapshots. Tout diff bloque le merge.

**Régénération** : si une modification légitime change le format (ex : nouveau champ ajouté), `go test -tags=goldenparity -update` régénère les fixtures. Le diff git de la régénération **doit être reviewé manuellement** (commit séparé `chore(golden): regen after <feature>`). Le test refuse `-update` en CI (vérification env var).

**Budget perf** : le golden parity tourne en `< 30s` total (fixture DB préchargée en RAM, pas de réseau). Au-delà, il sera désactivé en pratique → règle stricte sur les fixtures small.

### 10.2. Smoke E2E Playwright

Sur les pages clés (Home, Match View, Career, Synthesis), vérifier que :

1. les labels affichés correspondent exactement à `fields.toml` après chargement de l'endpoint mappings ;
2. l'ordre d'affichage du scoreboard match correspond à `display_order` ;
3. un changement de locale FR <-> EN ne casse aucune section (pas de `[object Object]`, pas de `undefined`).

### 10.3. Corpus synthétique « titre B »

Créer un titre fictif `synthetic_title_b/` avec :

1. un schéma DuckDB **différent** (colonnes renommées : `eliminations` au lieu de `kills`, `damage_inflicted` au lieu de `damage_dealt`, durées en millisecondes — `storage_unit = "milliseconds"`, `display_unit = "seconds"`) ;
2. un `data_adapter.go` + `semantic_adapter.go` réalisant les transforms ;
3. des TOML mappings traduisant en `FieldKey` canoniques avec mappings d'enums modes ;
4. fixtures de matchs synthétiques.

Tests :

1. `synthetic_title_b` retourne le **même schéma canonique services** que HI sur les FieldKey communs ;
2. les services produit **n'ont aucun code-path conditionnel** sur `title_slug` (sauf le bootstrap et le router de l'adapter) ;
3. l'isolement inter-titres : une session HI ne lit jamais de fichier de `synthetic_title_b` (vérifié par instrumentation des opens DuckDB en test).

Effort : **~0.5–1 jour** (déjà budgété dans S44_WORKPACKAGES).

### 10.4. Tests d'isolation cross-titres

Test dédié qui boote 2 titres en parallèle, fait 100 requêtes mélangées, et vérifie :

1. aucun écho cross-titres dans les caches in-memory ;
2. aucun adapter ne lit la DB de l'autre titre (instrumentation des chemins de fichiers).

### 10.5. Tests de migration TOML

Quand on incrémente `schema_version` dans un TOML :

1. test que l'ancienne version échoue à charger avec un message clair ;
2. test qu'une migration up + down est documentée sous `tools/mappings/migrate_v{N}_to_v{N+1}.go` (script exécutable + test sur fixtures avant/après) ;
3. supporter N et N-1 simultanément pendant la transition (fenêtre min 1 sprint), via un dispatcher dans le loader qui choisit la struct selon `schema_version`.

Format de la doc de migration : un `tools/mappings/CHANGELOG.md` liste chaque bump de version avec ses changements breaking et l'instruction de migration.

---

## 11. Plan de bascule incrémentale

### Phase A — Fondation (no nouveau comportement métier)

1. Créer `internal/games/canonical/` avec la liste exhaustive de FieldKey (annexe §17).
2. Créer `internal/games/mappings/` avec le loader, les types, les conversions d'unités.
3. Créer `config/titles/halo_infinite/mappings/fields.toml` couvrant les FieldKey de l'annexe.
4. Endpoint `/api/v1/titles/{slug}/field-mappings` introduit derrière feature flag `MULTI_TITLE_API_ENABLED=false` par défaut → endpoint visible mais 404 hors flag. **Pas un no-op pur** (introduit une surface API), mais comportement off par défaut.
5. Tests unitaires complets sur le loader et le format (couverture cible §9).

### Phase B — Adapter HI (toujours sans bascule)

1. Créer `internal/games/halo_infinite/adapter.go` qui **wrappe** les `platform/duckdb/queries_*.go` existants sans les réécrire.
2. Tests d'intégration : l'adapter produit le même résultat que les requêtes legacy sur des fixtures.
3. Adapter ouvert mais services produit **non migrés** vers lui.

### Phase C — Bascule progressive endpoint par endpoint

Ordre du moins risqué au plus risqué (basé sur volume de FieldKey, criticité, et dépendances aval) :

1. `/career/encounters` (peu de FieldKey, surfaces secondaires)
2. `/synthesis` (mid-volume, déjà bien couvert par tests existants)
3. `/home` (volume élevé mais composition de sous-blocs déjà testés isolément)
4. `/match-view/{id}` (cœur produit, le plus de FieldKey, **dernier** car le risque est concentré là)

Justification : le critère est le **blast radius en cas d'erreur**. Match View en dernier permet d'avoir tous les FieldKey du Match View déjà éprouvés ailleurs (Home les utilise en grande partie en preview). Mettre Match View en premier ferait porter à un seul endpoint l'ensemble du risque.

Pour chaque endpoint :

1. brancher l'adapter ;
2. golden parity diff = 0 ;
3. déployer derrière flag `MULTI_TITLE_<ENDPOINT>_ENABLED=true` ;
4. observer les logs `field_lookup_missing` au moins 48h ;
5. corriger les TOML si besoin avant de passer au suivant.

### Phase D — Frontend

1. Hook `useFieldLabel(key)` introduit + endpoint consommé au boot. **MVP côté frontend = `/career/encounters`** (cohérent avec l'ordre de Phase C : on consomme côté React le premier endpoint basculé).
2. Migration page par page, avec lint custom qui empêche les régressions.
3. Suppression des strings i18n en double avec les TOML — **bloquante** pour la fin de Phase D : on ne marque pas Phase D terminée tant que le diff `lint-no-hardcoded-fields` est vert sur tout le repo.

### Phase E — Corpus synthétique titre B

1. Implémentation `synthetic_title_b/` (~1j).
2. Tests d'isolation cross-titres.
3. Documentation d'un futur titre réel.

### Phase F — Cleanup

1. Suppression des `queries_*.go` legacy une fois leurs équivalents adapter validés.
2. Documentation finale dans `docs/ARCHITECTURE_V6.md` + mise à jour `project_map.md`.
3. Thought log d'archivage du chantier.

---

## 12. Idées complémentaires (au-delà du périmètre core)

> Les éléments core (conversion d'unités, mapping d'enums, schema versioning) sont déjà intégrés au corps du plan §6. Cette section liste les améliorations non bloquantes.

### 12.1. Index inversé pour la recherche / filtres

Le `FieldMappingSet` peut exposer un index `LabelToKey(locale, label) -> FieldKey` pour les filtres en langue naturelle (« je veux trier par éliminations » -> `kills`). Utile à l'Explorer. Décision à prendre quand l'Explorer aura migré.

### 12.2. Capability gating au niveau UI

Au lieu de `null` côté frontend pour un champ non supporté, le contrat field-mappings expose `capability_status: "supported" | "degraded" | "unsupported"`. Le composant React applique automatiquement un badge ou masque la colonne. Aligne la couche sémantique avec la capability map du provider.

### 12.3. Génération automatique de la documentation

Un script `tools/gen-mapping-docs.go` lit les TOML et émet un `docs/FIELDS.md` listant pour chaque titre les champs supportés, leurs unités, et les écarts entre titres. Utile pour onboarder un futur titre.

### 12.4. Test de cohérence cross-titres

Un test CI vérifie que l'union des `FieldKey` déclarés dans les TOML est un sous-ensemble du canonique central. Aucun TOML ne peut introduire un FieldKey orphelin.

### 12.5. Format `display_token` pour cohérence UI

Plutôt que de hardcoder des couleurs ou des badges côté React, les TOML peuvent référencer des **tokens** du design system (`outcome.positive`, `combat.critical`). Le frontend résout le token via son thème. Ça permet de changer la couleur d'un outcome dans tout le produit sans toucher au code.

### 12.6. Audit de cohérence labels <-> i18n existant

Au boot du loader, un mode `--audit` compare les libellés TOML aux clés i18n React existantes (`apps/web/src/i18n/locales/*.json`). Toute divergence détectée -> rapport. Permet d'éviter qu'un titre affiche « Éliminations » côté backend et « Kills » côté frontend.

---

## 13. Risques et mitigations

| Risque | Probabilité | Impact | Mitigation |
|---|:---:|:---:|---|
| Bascule HI casse un endpoint | Moyen | Haut | Golden parity strict + bascule endpoint par endpoint + observabilité |
| TOML deviennent ingérables (centaines de FieldKey) | Faible | Moyen | Découpage par groupe (`fields_combat.toml`, `fields_pve.toml`) + test de coverage |
| Frontend appelle l'endpoint à chaque rendu | Moyen | Moyen | Cache TanStack Query + ETag backend + assertion de un seul `fetch` au boot |
| Drift entre TOML et i18n React | Haut sans audit | Moyen | Mode audit au boot + lint CI |
| Loader trop rigide bloque tout au boot d'un nouveau titre | Faible | Haut | Validation par titre indépendante : un titre cassé ne bloque pas les autres |
| Adapter HI ré-implémente mal une requête | Moyen | Haut | Tests unitaires + golden parity + revue obligatoire |
| Conversion d'unités introduit des erreurs flottants | Moyen | Moyen | Tests sur valeurs limites + format pur de la couche `format.go` + property-based tests |
| Endpoint `/field-mappings` lent ou volumineux (100+ FieldKey × N locales) | Moyen | Moyen | Cache HTTP avec ETag basé sur hash du TOML + payload `gzip` négocié + benchmark Go vérifiant `< 5ms` p99 |
| Volume Go global qui explose (queries déplacées + adapters + canonical + mappings) | Moyen | Moyen | Respecter règle CLAUDE.md §13-14 (fonctions ≤ 80L, modules ≤ 500L) ; découpage `queries_*.go` par domaine |

---

## 14. Critères d'acceptation

Le chantier n'est livré que si :

1. les interfaces `TitleDataAdapter` et `TitleSemanticAdapter` sont implémentées pour `halo_infinite` et passent les tests unitaires (>= 85% sur le package `internal/games/halo_infinite/`, >= 90% sur `internal/games/mappings/`) ;
2. tous les endpoints clés (Home, Match View, Career, Synthesis, Timeseries, Explorer) ont basculé vers l'adapter ;
3. la golden parity HI affiche **diff = 0** sur les fixtures synthétiques ;
4. l'endpoint `/field-mappings` est consommé par le frontend pour **`/career/encounters`** au minimum (cohérent avec l'ordre Phase C/D) ;
5. les TOML couvrent 100% des FieldKey exposés par les endpoints migrés ;
6. les logs structurés `adapter_loaded`, `mappings_loaded`, `field_lookup_missing`, `capability_not_supported` sont présents et observables en prod ;
7. le corpus synthétique titre B existe et passe les tests d'isolation cross-titres ;
8. la documentation est à jour : `docs/ARCHITECTURE_V6.md`, `.ai/project_map.md`, `.ai/thought_log.md`, `tools/mappings/CHANGELOG.md` ;
9. `golangci-lint run` clean ;
10. `npm run typecheck && npm run lint && npm run build && npm run test:run && npm run test:e2e` clean ;
11. le script `tools/lint-no-hardcoded-fields.mjs` retourne 0 violation sur le repo.

---

## 15. Documents liés

1. `.ai/go_migration_v2/HALO_CANONICAL_MODEL.md` — modèle canonique provider.
2. `.ai/go_migration_v2/HALO_INFINITE_CAPABILITY_MAP.md` — capabilities mono-titre HI.
3. `.ai/go_migration_v2/ADR_S44_MULTI_TITLE_NAMESPACE.md` — namespace par titre.
4. `.ai/go_migration_v2/HALO_PRODUCT_CONTRACT_ADAPTERS.md` — adaptateurs produit côté HTTP.
5. `.ai/go_migration_v2/SPRINT_44_WORKPACKAGES.md` — découpage technique S44.
6. `.ai/go_migration_v2/OPENAPI_MVP_P0_P1.md` — contrats HTTP figés.
7. `docs/ARCHITECTURE_V6.md` — architecture cible (à enrichir avec ce plan).

---

## 16. Décisions tranchées (validées 2026-04-25)

Validation complète par Guillaume des 8 recommandations par défaut. Aucune dérive.

| # | Question | Décision | Référence |
|---|---|---|---|
| 1 | Médailles : DB ou TOML ? | **Hybride** — libellés et icons en DB (`medal_definitions`), tiers (`bronze/silver/gold/mythic`) + regroupements UX en TOML | §6.7 |
| 2 | Weapon labels : DB ou TOML ? | **Reste DB** — provenance API, volume, et récurrence cross-titres avec variantes mineures de noms (cf. note ci-dessous) | §6.8 |
| 3 | Format de fichier | **TOML** définitif | §6.5 |
| 4 | Locales obligatoires | **EN + FR** seulement, extensible plus tard | §6.6 |
| 5 | Hot-reload prod | **Jamais** — redéploiement obligatoire | §7.3 |
| 6 | Schema versioning N et N-1 simultanés | **Au cas par cas** avec déprécation documentée dans `tools/mappings/CHANGELOG.md` | §10.5 |
| 7 | Lint anti-hardcode FieldKey | **Script Node simple** (`tools/lint-no-hardcoded-fields.mjs`) | §9.5 |
| 8 | Cache endpoint `/field-mappings` | **ETag + Cache-Control** combinés | §13 |

### Note sur les armes (décision 2)

Observation produit : les armes Halo (BR, AR, Sniper, Rocket, Sword…) reviennent largement d'un titre à l'autre, avec parfois des variantes de nom (`MA37` vs `AR`, `BR55` vs `BR75`) et quelques nouveautés à chaque jeu. Implications pour le design :

1. à long terme, les `weapon_id` filmshell sont **par-titre** et leur table reste donc dans la metadata DuckDB du titre courant (cohérent avec namespace par titre acté en ADR_S44) ;
2. une couche supérieure de **canonical weapon family** (`weapon_family = "battle_rifle"`, `weapon_family = "assault_rifle"`) pourra être introduite plus tard si le produit veut comparer des armes équivalentes cross-titres — **hors scope de ce plan**, à acter quand le second titre réel arrivera ;
3. en attendant, le `TitleSemanticAdapter` lit `weapon_labels` depuis la metadata DuckDB du titre, comme aujourd'hui ; rien à changer côté schéma.

---

## 17. Annexe — Liste initiale des `FieldKey`

**Périmètre** : champs exposés par Home, Match View, Career, Synthesis. Tout ajout suit la règle §4.4.

### Combat (`group = "combat"`)

| FieldKey | Type | display_unit | Format |
|---|---|---|---|
| `kills` | int | count | integer |
| `deaths` | int | count | integer |
| `assists` | int | count | integer |
| `headshot_kills` | int | count | integer |
| `melee_kills` | int | count | integer |
| `grenade_kills` | int | count | integer |
| `power_weapon_kills` | int | count | integer |
| `damage_dealt` | int | count | integer |
| `damage_taken` | int | count | integer |
| `shots_fired` | int | count | integer |
| `shots_hit` | int | count | integer |
| `accuracy` | float | percent | percent_1 |
| `kdr` | float | ratio | kdr_2 |
| `kda` | float | ratio | kdr_2 |
| `max_killing_spree` | int | count | integer |

### Match (`group = "match"`)

| FieldKey | Type | display_unit | Format |
|---|---|---|---|
| `match_id` | string | — | string |
| `started_at_utc` | datetime | iso8601 | datetime |
| `duration_seconds` | int | seconds | duration_hms |
| `is_ranked` | bool | — | boolean |
| `is_pve` | bool | — | boolean |
| `outcome` | enum | — | enum (Outcome) |
| `personal_score` | int | count | integer |
| `team_score` | int | count | integer |
| `rank_in_match` | int | count | integer |

### Career (`group = "career"`)

| FieldKey | Type | display_unit | Format |
|---|---|---|---|
| `current_rank_id` | string | — | string |
| `current_xp` | int | count | integer |
| `xp_for_next_rank` | int | count | integer |
| `total_matches_played` | int | count | integer |
| `total_kills_career` | int | count | integer |
| `win_rate` | float | percent | percent_1 |

### Skill (`group = "skill"`)

| FieldKey | Type | display_unit | Format |
|---|---|---|---|
| `team_mmr` | float | — | signed_int |
| `enemy_mmr` | float | — | signed_int |
| `kills_expected` | float | count | percent_2 |
| `deaths_expected` | float | count | percent_2 |
| `csr_value` | int | count | integer |
| `lusr_value` | float | — | percent_2 |

### PvE Firefight (`group = "pve"`, `capability = supporte`)

| FieldKey | Type | display_unit | Format |
|---|---|---|---|
| `waves_completed` | int | count | integer |
| `bosses_killed` | int | count | integer |
| `grunt_kills` | int | count | integer |
| `elite_kills` | int | count | integer |
| `jackal_kills` | int | count | integer |
| `brute_kills` | int | count | integer |
| `hunter_kills` | int | count | integer |

**Total : ~38 FieldKey** au démarrage. Liste exécutoire — toute modification passe par PR sur ce document + bump `schema_version` du TOML concerné.

---

## 18. Estimation grossière de l'effort

> Les estimations sont en **jours-personne** sur un développeur familier du repo. À ajuster en review.

| Phase | Effort | Détails |
|---|:---:|---|
| Phase A (fondation + endpoint flag-off) | 3–4j | Loader + types + 38 FieldKey + TOML HI initial + tests unitaires + endpoint OpenAPI |
| Phase B (adapter HI wrapper) | 2–3j | Wrappe les `queries_*.go` existants sans réécriture, golden capture |
| Phase C (bascule 4 endpoints) | 4–6j | 1j par endpoint + observation logs + corrections |
| Phase D (frontend) | 3–4j | Hook + lint + migration page par page + nettoyage i18n React |
| Phase E (titre B synthétique) | 1j | Schéma DuckDB synthétique + adapter + isolation tests |
| Phase F (cleanup + doc) | 1–2j | Suppression legacy, docs, thought log |
| **Total** | **14–20j** | ~3 sprints sur un dev solo, ou ~1.5 sprint à 2 devs |

Risque de dérive : la suppression progressive des strings i18n React (Phase D) peut révéler des centaines de touches éparpillées. Budget une marge de 30% sur Phase D si l'inventaire i18n n'a pas été fait au préalable.

**Pré-requis non bloquants mais aidant** : l'audit des locales i18n React existantes (~0.5j) avant d'attaquer Phase A permet de calibrer Phase D plus finement.

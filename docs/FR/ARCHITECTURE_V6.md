# Architecture LevelUp v6 — DuckDB Shared Matches + i18n Assets

> **Date** : 2026-04-12
> **Version** : 6.3.0
> **Évolution depuis** : v5.1 (Shared Matches) → v6.0 (couche résolution ID) → v6.3 (asset_translations)

> Version anglaise : [ARCHITECTURE_V6.md](../ARCHITECTURE_V6.md)

---

## Vue d'Ensemble

LevelUp v6 étend l'architecture **Shared Matches** avec une **couche d'abstraction SQL** (vues de résolution) et un système centralisé d'**internationalisation des assets** (`asset_translations`). Les noms de maps, modes de jeu, playlists et variantes sont désormais stockés en base dans 14 langues et exposés via la vue `v_match_full`.

### Gains mesurés

| Métrique | v4 | v5.0 | v5.1 | Gain total |
|----------|----|----|------|------|
| Stockage par joueur | 200 MB | 30 MB | ~4 MB | **-98%** |
| Connexion DB | - | 80ms | <20ms | **-75%** |
| Première page UI | - | 1500ms | <800ms | **-47%** |
| SQLite runtime | 7 | 0 | 0 | **-100%** |
| Pandas métier | 7 | 7 | 0 | **-100%** |
| Tables obsolètes/joueur | 13 | 8 | 0 | **-100%** |
| Tests | 1065 | 2768 | 3323 | **+212%** |

### 7 Points Critiques v5.1

1. **`match_stats` n'existe plus** dans les player DBs
2. **`player_match_stats` n'existe plus** — colonnes MMR dans `shared.match_participants`
3. **`xuid_aliases` est dans shared uniquement**
4. **`player_match_enrichment` est la SEULE table match** dans les player DBs
5. **Coéquipiers chargés depuis shared**, pas les DBs individuelles
6. **Cleanup brutal intentionnel** : erreurs explicites si code résiduel accède aux tables supprimées
7. **Sync écrit dans player DBs** : `player_match_enrichment` + `personal_score_awards` uniquement

---

## Diagramme d'Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Streamlit UI                        │
│  (pages/, components/, visualization/)               │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────┐
│              DuckDBRepository                         │
│  (src/data/repositories/duckdb_repo.py)              │
│                                                      │
│  ┌─────────────────┐  ┌──────────────────────────┐   │
│  │ _get_match_source│  │ ATTACH shared READ_ONLY  │   │
│  │ (sous-requête)   │  │ ATTACH meta READ_ONLY    │   │
│  └─────────────────┘  └──────────────────────────┘   │
└──────┬───────────────────────┬───────────────────────┘
       │                       │
       ▼                       ▼
┌──────────────┐   ┌───────────────────────────────────┐
│ stats.duckdb │   │     shared_matches.duckdb          │
│ (par joueur) │   │     (data/warehouse/)              │
│              │   │                                    │
│ ┌──────────┐ │   │ ┌────────────────┐ ┌────────────┐ │
│ │enrichment│ │   │ │match_registry  │ │match_      │ │
│ │teammates │ │   │ │(1 ligne/match) │ │participants│ │
│ │antagonist│ │   │ └────────────────┘ │(tous joueur│ │
│ │citations │ │   │ ┌────────────────┐ └────────────┘ │
│ └──────────┘ │   │ │highlight_events│ ┌────────────┐ │
└──────────────┘   │ │(tous events)   │ │medals_     │ │
                   │ └────────────────┘ │earned      │ │
┌──────────────┐   │ ┌────────────────┐ └────────────┘ │
│metadata.duckdb│  │ │xuid_aliases   │                 │
│(référentiels) │  │ └────────────────┘                 │
└──────────────┘   └───────────────────────────────────┘
```

---

## Structure des Fichiers

```
data/
├── warehouse/
│   ├── metadata.duckdb            # Référentiels (asset_translations 14 langues, weapon_labels, career_ranks, citation_mappings, challenge_*)
│   ├── shared_matches.duckdb      # Base partagée - TOUS les matchs
│   │   ├── match_registry         # Registre central (1 ligne par match unique)
│   │   ├── match_participants     # Stats de TOUS les joueurs de TOUS les matchs
│   │   ├── highlight_events       # TOUS les événements filmés
│   │   ├── medals_earned          # Médailles de TOUS les joueurs
│   │   ├── xuid_aliases           # Mapping global xuid→gamertag
│   │   └── schema_version         # Versioning du schéma
│   └── shared_pve.duckdb          # Stats PvE Firefight — v5.2
│       └── pve_match_stats        # Waves, boss, kills par type d'ennemi (par joueur/match)
│
├── players/
│   └── {gamertag}/
│       ├── stats.duckdb           # Enrichissements PERSONNELS uniquement
│       │   ├── player_match_enrichment  # performance_score, session_id, is_with_friends
│       │   ├── personal_score_awards    # Awards objectifs (PersonalScores API)
│       │   ├── antagonists              # Rivalités (top killers/victimes)
│       │   ├── match_citations          # Citations calculées par match
│       │   ├── career_progression       # Historique rangs
│       │   ├── media_files              # Fichiers médias
│       │   ├── media_match_associations # Associations média↔match
│       │   ├── match_skill_rank         # Rating LUSR ou CSR par match — v5.3
│       │   └── challenge_snapshots      # Historique des défis joueur (active/completed/upcoming)
│       └── archive/               # Archives Parquet (saisons)
│
└── cache/                         # Cache temporaire (thumbnails, etc.)
```

---

## Composants Principaux

### 1. DuckDBRepository (`src/data/repositories/duckdb_repo.py`)

Le repository central d'accès aux données. En v5, il utilise **ATTACH** pour monter
plusieurs bases DuckDB simultanément :

```python
# Connexion multi-DB
conn = duckdb.connect(player_db_path)
conn.execute("ATTACH ? AS shared (READ_ONLY)", [shared_db_path])
conn.execute("ATTACH ? AS meta (READ_ONLY)", [metadata_db_path])
```

La sous-requête `_get_match_source()` abstrait la jointure entre `shared.match_registry`,
`shared.match_participants` et `player_match_enrichment`, exposant un alias `match_stats`
compatible avec toutes les pages UI existantes.

### 2. DuckDBSyncEngine (`src/data/sync/engine.py`)

Le moteur de synchronisation détecte automatiquement les matchs partagés :

- **Match connu** (`_process_known_match`) : Le match existe déjà dans `match_registry`.
  Seul l'enrichissement personnel est calculé (performance_score, session). Économie : 1-2 appels API.
- **Match nouveau** (`_process_new_match`) : Sync complète — insertion dans `match_registry`,
  `match_participants` (tous les joueurs), `highlight_events`, `medals_earned`.

### 3. Transformers (`src/data/sync/transformers.py`)

Fonctions d'extraction des données JSON de l'API Halo vers les structures DuckDB :

- `extract_match_registry_data()` : Données communes du match
- `extract_all_medals()` : Médailles de TOUS les joueurs (pas seulement le joueur courant)
- `extract_collective_stats()` : Stats de tous les participants

### 4. CitationEngine (`src/analysis/citations/engine.py`)

Moteur de calcul des citations (commendations) basé sur SQL :

- Lecture depuis `shared.medals_earned` et `shared.match_participants`
- Agrégation SQL performante (vs itérations row-by-row en v4)
- Stockage dans `match_citations` (player DB)
- 14 règles configurables via `citation_mappings` (metadata DB)

### 5. Factory (`src/data/repositories/factory.py`)

Pattern Factory pour créer des `DuckDBRepository` avec auto-détection des chemins :

- `shared_db_path` : Auto-détecté depuis `data/warehouse/shared_matches.duckdb`
- `metadata_db_path` : Auto-détecté depuis `data/warehouse/metadata.duckdb`
- Fallback v4 transparent si `shared_matches.duckdb` n'existe pas

### 6. asset_translations — Internationalisation des assets (v6.3)

La table `asset_translations` dans `metadata.duckdb` est la **source unique** de noms localisés pour les assets Halo (cartes, modes, playlists, variantes).

#### Peuplement

```bash
python scripts/populate_asset_translations.py
# Options
python scripts/populate_asset_translations.py --force        # réécrit tout
python scripts/populate_asset_translations.py --dry-run      # simule sans écrire
python scripts/populate_asset_translations.py --types map    # type spécifique
```

Le script utilise `_build_version_id_cache()` pour récupérer les `VersionId` requis par l'API SPNKr Discovery UGC avant de paralléliser les 14 langues via `asyncio.gather`.

#### Utilisation via v_match_full

```sql
-- Dans shared_matches_v2.duckdb (meta doit être ATTACHé)
SELECT match_id, map_name, map_name_fr, game_variant_name, game_variant_name_fr
FROM v_match_full
WHERE map_name_fr IS NOT NULL
LIMIT 10;
```

`DuckDBRepository` attache automatiquement `metadata.duckdb` en `meta` à l'ouverture de chaque connexion. Aucune configuration manuelle requise.

### 6bis. challenge_definitions / challenge_translations — Défis Halo (v7)

Les définitions CMS de défis Halo sont désormais historisées dans `metadata.duckdb` :

- `challenge_definitions` : versionnement par couple `(challenge_path, content_hash)` avec `category`, `difficulty`, `threshold_for_success`, `reward_xp`
- `challenge_translations` : titres et descriptions dans toutes les langues exposées par le CMS, normalisées en BCP-47 avec fallback `en-US`

Les états joueur vus en live (`/decks`) sont stockés dans `stats.duckdb` via `challenge_snapshots` en mode append-only dédupliqué, afin de conserver une timeline exploitable sans réécrire la même ligne à chaque refresh.

#### Langues disponibles

| Code BCP-47 | Langue |
|-------------|--------|
| `en-US` | Anglais (US) |
| `fr-FR` | Français |
| `de-DE` | Allemand |
| `es-ES` | Espagnol (ES) |
| `es-MX` | Espagnol (MX) |
| `it-IT` | Italien |
| `ja-JP` | Japonais |
| `ko-KR` | Coréen |
| `nl-NL` | Néerlandais |
| `pl-PL` | Polonais |
| `pt-BR` | Portugais (BR) |
| `ru-RU` | Russe |
| `zh-Hans` | Chinois simplifié |
| `zh-Hant` | Chinois traditionnel |

### 7. SyncScope (`src/data/sync/scope.py`)

**Nouveau en v5.2** — Dataclass centralisant les 30+ flags de données partagés
entre `sync.py`, `backfill_data.py` et leurs sous-modules (orchestrator, detection, engine).

```python
from src.data.sync.scope import SyncScope

# Tout activer
scope = SyncScope.make_all(max_matches=100)

# Depuis argparse
scope = SyncScope.from_cli_args(args)

# Sélection fine
scope = SyncScope(medals=True, events=True, force_medals=True)
scope.resolve()  # force_medals → medals=True automatiquement
```

**Rôle** : Remplace la copie de 30+ kwargs à travers la chaîne
`cli.py` → `backfill_data.py` → `orchestrator.py` → `detection.py` → `_backfill_with_api`.

**Registres internes** :
- `_ALL_DATA_FIELDS` : champs activés par `--all-data`
- `_FORCE_MAP` : implications `force_X` → `X`
- `_REQUESTED_TYPE_MAP` : mapping champ → clé bitmask `backfill_completed`

**Pour ajouter un nouveau type de données** :
1. Ajouter le champ booléen dans `SyncScope` + registres
2. Ajouter l'argument CLI dans `scripts/backfill/cli.py`
3. Implémenter la logique métier dans l'orchestrateur / engine

> **Note legacy** : Les fonctions `backfill_player_data`, `backfill_all_players`,
> `_backfill_with_api` et `find_matches_missing_data` conservent les 30+ kwargs
> individuels pour rétro-compatibilité (marqués `LEGACY` dans le code).
> Nouveau code : toujours passer `scope=SyncScope(...)`.

---

## Flux de Données

### Synchronisation (Sync)

```
API Halo (SPNKr)
     │
     ▼
DuckDBSyncEngine
     │
     ├── Match nouveau ────────────────────────────────┐
     │   1. Fetch match_stats (API)                    │
     │   2. Fetch skill (API)                          │
     │   3. Fetch events (API)                         ▼
     │   4. extract_match_registry_data()     shared_matches.duckdb
     │   5. extract_collective_stats()        ├── match_registry
     │   6. extract_all_medals()              ├── match_participants
     │                                        ├── highlight_events
     │                                        └── medals_earned
     │
     └── Match connu ──────┐
         1. Calcul perf    │
         2. Session detect ▼
                      stats.duckdb (joueur)
                      └── player_match_enrichment
```

### Lecture (UI)

```
Page UI (ex: timeseries.py)
     │
     ▼
DuckDBRepository.load_matches()
     │
     ├── _get_match_source()
     │   └── JOIN shared.match_registry
     │         + shared.match_participants (WHERE xuid = ?)
     │         + player_match_enrichment
     │   → alias "match_stats"
     │
     └── Résultat : Polars DataFrame
```

---

## Modules Applicatifs

```
src/
├── app/                          # Orchestration application
│   ├── state.py                  # Gestion session_state Streamlit
│   ├── routing.py                # Navigation entre pages
│   ├── sidebar.py                # Sidebar (filtres, profil, sync)
│   ├── page_router.py            # Routeur de pages
│   ├── helpers.py                # Fonctions utilitaires
│   ├── filters.py                # Logique des filtres
│   ├── profile.py                # Gestion profil joueur
│   ├── kpis.py                   # Calcul et affichage KPIs
│   └── data_loader.py            # Chargement données
│
├── config.py                     # Configuration & constantes
│
├── data/                         # Couche données
│   ├── repositories/
│   │   ├── duckdb_repo.py        # Repository DuckDB principal (ATTACH multi-DB)
│   │   ├── _match_queries.py     # Requêtes matchs (_get_match_source)
│   │   ├── _roster_loader.py     # Chargement roster depuis shared
│   │   └── factory.py            # Factory pattern
│   ├── services/                 # Services métier
│   │   ├── timeseries_service.py # Agrégats séries temporelles
│   │   ├── win_loss_service.py   # Bucketing V/D, breakdown cartes
│   │   └── teammates_service.py  # Stats coéquipiers multi-DB
│   ├── sync/                     # Synchronisation API
│   │   ├── api_client.py         # Client SPNKr
│   │   ├── engine.py             # Moteur de sync (v5 shared)
│   │   ├── scope.py              # SyncScope — flags sync/backfill centralisés
│   │   ├── transformers.py       # Transformations JSON→DB
│   │   ├── migrations.py         # Migrations de schéma
│   │   └── models.py             # Modèles de sync
│   └── query/
│       └── engine.py             # Query Engine DuckDB
│
├── analysis/                     # Logique métier
│   ├── citations/                # Système de citations
│   │   ├── engine.py             # CitationEngine (SQL)
│   │   ├── custom_rules.py       # Règles custom (objectifs)
│   │   └── models.py             # Modèles citations
│   ├── filters.py                # Filtres playlists/modes
│   ├── killer_victim.py          # Analyse confrontations
│   ├── antagonists.py            # Agrégation rivalités
│   ├── sessions.py               # Détection sessions
│   ├── stats.py                  # Calculs statistiques
│   ├── performance_score.py      # Score de performance
│   ├── playlist_groups.py        # 6 groupes Halo (ranked/arena/btb/tactical/social/fun) — v5.3
│   ├── skill_rating_config.py    # Constantes TrueSkill 2, tiers Bronze→Onyx — v5.3
│   ├── skill_rating.py           # Algorithme LUSR (PlayerState, Elo-style mu, batch) — v5.3
│   └── skill_rating_calibration.py # Calibration COMPOSITE_WEIGHTS (grid search) — v5.3
│
├── ui/                           # Interface utilisateur
│   ├── cache.py                  # Cache Streamlit
│   ├── medals.py                 # Affichage médailles
│   ├── translations.py           # Traductions FR
│   ├── sync.py                   # UI de synchronisation
│   ├── filter_state.py           # Filtres intent-based, persist JSON par joueur — v5.2
│   ├── components/               # Composants réutilisables
│   └── pages/                    # Pages du dashboard (23 pages)
│
├── visualization/                # Graphiques Plotly (15 modules)
│
├── utils/                        # Utilitaires (paths, xuid, profiles)
│   └── discord_notifier.py       # Notifications Discord post-sync/backfill (failsafe) — v5.3
└── visualization/                # Graphiques Plotly (palette Okabe-Ito v5.2, LUSR timeseries v5.3)
```

---

## Tests

La suite de tests v5 comprend **3323 tests** répartis en :

| Catégorie | Tests | Couverture |
|-----------|-------|-----------|
| Schéma migration | 45 | 95%+ |
| Sync shared matches | 33 | 70%+ |
| Repository v5 | 77 | 75%+ |
| Match queries v5 | 35 | 80%+ |
| UI pages (MockStreamlit) | 147 | 35-84% (selon page) |
| Utils purs | 72 | 90% |
| Sync/backfill | 338 | 84% (transformers), 99% (core) |
| Autres (viz, profile, etc.) | ~2000+ | Variable |

Framework de test :
- **MockStreamlit** : Fixture `conftest.py` pour tester les pages UI en mode headless
- **DuckDB `:memory:`** : Tests isolation complète sans fichier disque
- **Polars DataFrames** : Données synthétiques pour tous les tests

```bash
# Suite complète
python -m pytest

# Hors intégration (recommandé quotidien)
python -m pytest -q --ignore=tests/integration

# Avec couverture
python -m pytest --cov=src --cov-report=html
```

---

## Configuration

### Profils joueurs (`db_profiles.json`)

```json
[
  {
    "gamertag": "MonGamertag",
    "xuid": "1234567890",
    "db_path": "data/players/MonGamertag/stats.duckdb"
  }
]
```

### Paramètres application (`app_settings.json`)

Configuration de l'application Streamlit (thème, langue, options d'affichage).

---

## Différences v4 → v5

| Aspect | v4 | v5 |
|--------|----|----|
| **Stockage matchs** | Dupliqué dans chaque player DB | Centralisé dans `shared_matches.duckdb` |
| **Sync match connu** | Re-sync complète | Skip (enrichissement perso uniquement) |
| **Repository** | 1 connexion (player DB) | ATTACH multi-DB (player + shared + meta) |
| **Pages UI** | `FROM match_stats` | `FROM _get_match_source()` (sous-requête) |
| **Médailles** | Par joueur dans player DB | Tous joueurs dans `shared.medals_earned` |
| **Events** | Par joueur dans player DB | Tous events dans `shared.highlight_events` |
| **Citations** | Calcul à la volée | `CitationEngine` SQL + `match_citations` table |

---

## Voir aussi

- [SHARED_MATCHES_SCHEMA.md](SHARED_MATCHES_SCHEMA.md) — Schéma DDL complet
- [MIGRATION_V4_TO_V5.md](MIGRATION_V4_TO_V5.md) — Guide de migration
- [SYNC_OPTIMIZATIONS_V5.md](SYNC_OPTIMIZATIONS_V5.md) — Optimisations sync
- [TESTING_V5.md](TESTING_V5.md) — Stratégie de tests
- [ARCHITECTURE.md](ARCHITECTURE.md) — Architecture v4 (référence historique)

---

## Architecture Multi-Titre (Sprint 44)

LevelUp supporte plusieurs jeux (titres) via une **arborescence de données par titre**. Chaque titre possède son propre arbre isolé :

```text
data/
  titles/
    halo_infinite/          # titre par défaut
      warehouse/
        metadata.duckdb
        shared_matches_v2.duckdb
      players/
        {gamertag}/
          stats.duckdb
    halo_mcc/               # second titre (exemple)
      warehouse/
        ...
      players/
        ...
  warehouse/                # layout legacy flat (rétrocompatibilité)
  players/                  # layout legacy flat (rétrocompatibilité)
```

### Composants clés

| Composant | Rôle |
|-----------|------|
| `TitleRegistry` | Registre en mémoire des titres connus (slug, nom, statut, capacités) |
| `PathResolver` | Résolution de tous les chemins fichier par titre (`TitleDataDir`, `SharedDBPath`, `PlayerDBPath`, etc.) |
| Middleware `TitleExtractor` | Lit le header `X-LevelUp-Title` / session / fallback → injecte `title_slug` dans le contexte de requête |
| `db_profiles.json` v3 | Profils joueurs scopés par titre : `{ "version": "3.0", "profiles": { "<slug>": { "<gamertag>": {...} } } }` |

### Stratégie de routage

L'API utilise une sélection de titre **par header** (`X-LevelUp-Title`). Les URLs restent inchangées (`/api/v1/players/{slug}/...`). Le middleware injecte le titre dans le contexte, et tous les services en aval (PlayerResolver, ProfileService, etc.) l'utilisent pour scoper l'accès aux données.

### Rétrocompatibilité

- `PathResolver` fournit des méthodes `Legacy*` (`LegacySharedDBPath`, `LegacyPlayerDir`, etc.) pour le layout plat `data/warehouse/`
- Les fichiers `db_profiles.json` v2.1 sont auto-détectés et lus comme profils `halo_infinite` implicites
- `LoadPlayers()` sans filtre de titre retourne les joueurs de tous les titres

---

## Schéma canonique services + adaptateurs sémantiques (Phase A–E plan multi-titres)

Au-dessus du layout par titre, LevelUp expose un schéma canonique services et
deux adaptateurs par titre. Cela découple les services produit du schéma DuckDB
spécifique à un titre et de ses libellés/unités.

```text
handler HTTP → service produit → games.Resolver
                                    ├─ Data(slug)     → games.TitleDataAdapter
                                    └─ Semantic(slug) → games.TitleSemanticAdapter
```

### Packages

| Package | Rôle |
|---------|------|
| `internal/games/canonical/` | Enum `FieldKey` (43 clés), enums (`Outcome`, `MatchType`, `RatingType`, `Bucket`, `GroupBy`), scopes (`StatsScope`, `TimeseriesQuery`, `CareerOptions`), types match/career/timeseries — stables, agnostiques, consommés par les services |
| `internal/games/mappings/` | Loader TOML strict (`go-toml/v2`), validation (locales, formats, collisions `display_order`, conversions d'unités), `FieldMappingSet`, registry |
| `internal/games/halo_infinite/` | Implémentation HI : `DataAdapter` (wrap des repos existants), `SemanticAdapter` (wrap du `FieldMappingSet`) |
| `internal/games/synthetic_title_b/` | Corpus synthétique de tests d'isolation cross-titres uniquement — jamais référencé par le code de production |
| `internal/games/{adapter,resolver}.go` | Interfaces `TitleDataAdapter` + `TitleSemanticAdapter`, `StaticResolver` |

### Mappings TOML (versionnés Git)

```text
config/
  titles/
    halo_infinite/
      mappings/
        fields.toml           # 43 FieldKey × labels EN/FR + format + group + display_order
    synthetic_title_b/
      mappings/
        fields.toml           # corpus synthétique de tests
```

Chaque `fields.toml` porte un `[meta].schema_version` (cf. `tools/mappings/CHANGELOG.md`).

### API HTTP (derrière `MULTI_TITLE_API_ENABLED=true`)

- `GET /api/v1/titles/{slug}/field-mappings?locale=fr` — expose le
  `FieldMappingSet` d'un titre avec ETag + `Cache-Control: max-age=300`.
- `GET /api/v1/titles/{slug}/preview/career?xuid=...&locale=fr` — preuve
  end-to-end du pipeline canonique (data adapter + semantic adapter).
  Retourne `not_supported_reason` pour les capabilities `not_exposed` au lieu
  d'une erreur silencieuse.

### Hook frontend (Phase D)

```ts
import { useFieldLabel } from '@/lib/i18n/fieldMappings'

const label = useFieldLabel('kills') // → 'Éliminations' (FR) / 'Kills' (EN) / 'kills' (fallback)
```

Le hook lit `currentTitleSlug` et `locale` depuis `appShellStore`, fetche les
field mappings via TanStack Query (cache `staleTime: Infinity` — versionnés Git,
pas de hot-reload prod), et retombe gracieusement sur la `key` si l'endpoint
est absent (flag off, 404, etc.).

### Dégradation par capability

Chaque `TitleDataAdapter` expose une `Capabilities() games.CapabilityMap` qui
reflète le support produit par-titre. Un appel `Load*` sur une capability
marquée `not_exposed` retourne `games.ErrCapabilityNotSupported`, traduit en
aval par les services produit en un `not_supported_reason` explicite plutôt
qu'en payload vide silencieux.

Voir [`.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md`](../../.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md)
pour le rationale et [`tools/mappings/CHANGELOG.md`](../../tools/mappings/CHANGELOG.md)
pour l'historique de versioning des TOML.

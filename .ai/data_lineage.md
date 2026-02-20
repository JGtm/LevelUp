# Data Lineage - Traçabilité des Données Halo

> Ce fichier trace l'origine, les transformations et la destination de chaque flux de données.
> Mis à jour : 2026-02-17

## Architecture v5.1 - Shared Matches + Player Enrichments

```
┌─────────────────┐      ┌─────────────────────────────────────────────┐
│   API SPNKr     │      │              DuckDB Engine                  │
│  (Halo Infinite)│      │                                             │
└────────┬────────┘      │  ┌─────────────────────────────────────┐   │
         │               │  │  metadata.duckdb (global)           │   │
         ▼               │  │  - playlists, maps, game_modes      │   │
┌─────────────────┐      │  │  - medal_definitions, career_ranks  │   │
│  Pydantic v2    │      │  └─────────────────────────────────────┘   │
│  Validation     │      │                                             │
└────────┬────────┘      │  ┌─────────────────────────────────────┐   │
         │               │  │  shared_matches.duckdb (centralisée)│   │
         ▼               │  │  - match_registry (1 ligne/match)   │   │
┌─────────────────┐      │  │  - match_participants (31 col, MMR) │   │
│ DuckDBSyncEngine│──────│  │  - highlight_events                 │   │
│  Transformers   │      │  │  - medals_earned                    │   │
└────────┬────────┘      │  │  - killer_victim_pairs              │   │
         │               │  │  - xuid_aliases                     │   │
         │               │  └─────────────────────────────────────┘   │
         │               │                                             │
         └───────────────│  ┌─────────────────────────────────────┐   │
                         │  │  players/{gt}/stats.duckdb          │   │
                         │  │  - player_match_enrichment (SEULE)  │   │
                         │  │  - personal_score_awards             │   │
                         │  │  - antagonists, match_citations      │   │
                         │  │  - career_progression, sessions      │   │
                         │  │  - media_files, media_match_assoc    │   │
                         │  │  - mv_* (vues matérialisées)        │   │
                         │  └─────────────────────────────────────┘   │
                         └─────────────────────────────────────────────┘
                                            │
                                            ▼
                                   ┌─────────────────┐
                                   │   Streamlit UI  │
                                   │  (Polars DFs)   │
                                   └─────────────────┘
```

## Flux de Données Principaux

### 1. API Halo → DuckDB (Synchronisation v5.1)

```
Source: API Halo Infinite (via SPNKr)
     ↓
Client: SPNKrAPIClient (src/data/sync/api_client.py)
     ↓
Transformers: transform_match_stats(), extract_participants(), etc.
     ↓
Engine: DuckDBSyncEngine (src/data/sync/engine.py)
     ├─→ Match connu → enrichissement personnel uniquement (player_match_enrichment)
     └─→ Match nouveau → shared (registry + participants + events + medals)
     ↓
Destinations:
  - shared_matches.duckdb : matchs, participants, events, médailles, xuid_aliases
  - players/{gamertag}/stats.duckdb : enrichissements, awards uniquement
```

### 2. JSON → DuckDB (Référentiels)

```
Source: Fichiers JSON locaux
     ↓
Script: scripts/ingest_halo_data.py
     ↓
Destination: data/warehouse/metadata.duckdb
```

### 3. DuckDB → Parquet (Archive)

```
Source: DuckDB (match_stats)
     ↓
Script: scripts/archive_season.py
     ↓
Destination: data/players/{gamertag}/archive/matches_*.parquet
```

### 4. Parquet → DuckDB (Restore)

```
Source: Backup Parquet
     ↓
Script: scripts/restore_player.py
     ↓
Destination: data/players/{gamertag}/stats.duckdb
```

### 5. Dossiers médias → DuckDB (Onglet Médias)

```
Source: Dossiers configurés (Paramètres → media_screens_dir, media_videos_dir)
     ↓
MediaIndexer.scan_and_index() — scan delta récursif, ffprobe/EXIF
     ↓
media_files (status=active/deleted), media_match_associations (après associate_with_matches)
     ↓
Thumbnails: generate_thumbnails_for_new() → thumbs/ (GIF vidéo, miniatures images)
     ↓
UI: media_tab.py (load_media_for_ui → sections Mes captures / Captures de XXX / Sans correspondance)
```

Lancement : thread en arrière-plan au démarrage de l’app (`_background_media_indexing` dans streamlit_app.py).

## Tables et Cardinalité

### Métadonnées (metadata.duckdb)

| Table | Lignes | Description |
|-------|--------|-------------|
| `playlists` | ~14 | Définitions playlists |
| `game_modes` | ~313 | Modes de jeu (FR/EN) |
| `categories` | ~16 | Catégories de modes |
| `medal_definitions` | ~153 | Définitions médailles |
| `career_ranks` | 273 | Rangs (0-272) |
| `players` | Variable | Joueurs connus |

### shared_matches.duckdb (centralisée)

| Table | Cardinalité | Description |
|-------|-------------|-------------|
| `match_registry` | 1:1 par match | Registre central (données communes du match) |
| `match_participants` | N:1 par match | Stats de tous les joueurs (31 col, incl. MMR) |
| `highlight_events` | N:1 par match | Événements filmés |
| `medals_earned` | M:N | Médailles de tous les joueurs |
| `killer_victim_pairs` | N:1 par match | Paires killer→victim |
| `xuid_aliases` | 1:1 | Mapping global XUID→Gamertag |

### Données Joueur stats.duckdb (v5.1 — enrichissements uniquement)

> 8 tables supprimées : match_stats, match_participants, highlight_events,
> medals_earned, killer_victim_pairs, player_match_stats, xuid_aliases, teammates_aggregate

| Table | Cardinalité | Description |
|-------|-------------|-------------|
| `player_match_enrichment` | 1:N par joueur | performance_score, session_id, is_with_friends (**SEULE table match**) |
| `personal_score_awards` | M:N | Awards objectifs (PersonalScores API) |
| `antagonists` | 1:N | Rivalités agrégées |
| `match_citations` | 1:N | Citations calculées par match |
| `career_progression` | 1:N | Historique rangs |
| `sessions` | 1:N | Sessions groupées |
| `sync_meta` | 1:1 | Métadonnées sync |
| `media_files` | 1:N | Fichiers médias indexés (captures/vidéos), status active/deleted |
| `media_match_associations` | M:N | Association média ↔ match ↔ xuid |

### Vues Matérialisées

| Vue | Description | Rafraîchissement |
|-----|-------------|------------------|
| `mv_map_stats` | Stats par carte | Post-sync |
| `mv_mode_category_stats` | Stats par mode | Post-sync |
| `mv_session_stats` | Stats par session | Post-sync |
| `mv_global_stats` | Stats globales | Post-sync |

## Transformations Clés

| Donnée | Source | Formule |
|--------|--------|---------|
| `kda` | match_stats | `(kills + assists/3) / max(deaths, 1)` |
| `shots_fired` / `shots_hit` | API → match_stats, match_participants | `Players[].PlayerTeamStats[].Stats.CoreStats.ShotsFired` / `ShotsHit` ; joueur propriétaire dans match_stats, tous les joueurs dans match_participants. |
| `accuracy` | match_stats | `shots_hit / shots_fired * 100` (ou API si fourni) |
| `net_kills` | antagonists | `times_killed - times_killed_by` |
| `win_rate` | mv_global_stats | `wins / total_matches * 100` |
| `headshot_rate` | weapon_stats | `headshot_kills / total_kills * 100` |

### Rang dans le match

Le **rang d'un joueur lors d'un match** (position 1, 2, 3…) a deux origines :
- **Sync** : `match_stats.rank` ← API (`Players[].Rank`) via `transformers._extract_player_rank()`.
- **Vue match** : `MatchPlayerStats.rank` ← recalcul par `loaders.load_match_players_stats()` (tri par score, puis attribution 1, 2, 3…). Utilisé notamment pour le tie-breaker dans l’analyse killer/victim.

Détail : `.ai/DATA_MATCH_RANK.md`.

## Validations

### Pydantic v2

- [x] MatchStatsRow : Validation des champs matchs
- [x] PlayerMatchStatsRow : Validation MMR
- [x] HighlightEventRow : Validation événements
- [x] XuidAliasRow : Validation XUID (16 chiffres)
- [x] CareerRankData : Validation progression

### Contraintes DuckDB

- Clés primaires sur toutes les tables
- Index sur colonnes fréquemment filtrées (`start_time`, `playlist_id`)
- Colonnes GENERATED pour les calculs (`net_kills`, `accuracy`)

## Architecture Multi-Joueurs (v5.1)

En v5.1, les stats coéquipiers sont chargées depuis `shared.match_participants` :

```
1. Identifier match_id communs via shared.match_participants
      ↓
2. Charger stats coéquipier depuis shared.match_participants (xuid)
      ↓
3. Pas besoin d'accéder aux DBs individuelles des coéquipiers
```

Le sync écrit dans les player DBs : `player_match_enrichment` + `personal_score_awards` uniquement.

## Problèmes Connus

- Aucun problème majeur identifié

## Références

- `docs/SQL_SCHEMA.md` : Schémas complets
- `docs/SYNC_GUIDE.md` : Guide de synchronisation
- `.ai/ARCHITECTURE_ROADMAP.md` : Roadmap des phases
- `.ai/DATA_MATCH_RANK.md` : Rang d'un joueur lors d'un match (API vs recalcul)

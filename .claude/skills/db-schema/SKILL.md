# Skill : db-schema — Schéma DuckDB LevelUp

## Structure des chemins — multi-titres

Tous les chemins passent par `PathResolver` (`internal/domain/title/registry.go`).
**Ne jamais construire de chemin manuellement** avec `filepath.Join(repoRoot, "data", ...)`.

```go
// Correct
paths.SharedDBPath(titleSlug)         // data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb
paths.MetadataDBPath(titleSlug)       // data/titles/halo_infinite/warehouse/metadata.duckdb
paths.PlayerDBPath(titleSlug, gt)     // data/titles/halo_infinite/players/Chocoboflor/stats.duckdb
paths.GlobalXuidAliasesDBPath()       // data/global/xbox_aliases.duckdb (P5, ADR 0008 — global Microsoft)

// Interdit
filepath.Join(repoRoot, "data", "warehouse", "shared_matches_v2.duckdb")
```

## Quelle DB contient quoi ?

| DB | Chemin résolu | Contenu |
|---|---|---|
| `shared_matches_v2.duckdb` | `data/titles/{slug}/warehouse/` | Stats matchs de TOUS les joueurs |
| `metadata.duckdb` | `data/titles/{slug}/warehouse/` | Référentiels (modes, armes, rangs, médailles) |
| `shared_pve.duckdb` | `data/titles/{slug}/warehouse/` | Stats Firefight |
| `shared_social.duckdb` | `data/titles/{slug}/warehouse/` | Données sociales (followers, activité) |
| `stats.duckdb` | `data/titles/{slug}/players/{gamertag}/` | Enrichissements individuels uniquement |
| `xbox_aliases.duckdb` | `data/global/` | **Global** — mapping xuid→gamertag Xbox Services (P5, ADR 0008) |

## shared_matches_v2.duckdb

### match_registry — 1 ligne par match unique
Colonnes clés : `match_id`, `start_time`, `end_time`, `map_id`, `pair_name`, `playlist_id`, `team_game`

**Score et manches** : `team_0_score` / `team_1_score` = score du mode AFFICHÉ par le jeu
(`CoreStats.Score`). `team_0_rounds_won` / `team_1_rounds_won` / `rounds_total` = les MANCHES
(ADR 0032, 2026-08-29). Sur un mode qui se décide aux manches, le score est un CUMUL DE POINTS
qui peut donner la victoire au camp qui en a le moins : lire les manches, via
`analysis.ReadTeamScore` (jamais une comparaison de score à la main). `rounds_total` est le MAX
des deux camps. NULL = inconnu (ligne antérieure au backfill, FFA, titre sans la donnée) → on
retombe sur les points. Rattrapage : `cmd/backfill-team-rounds`.

**Piège des vues** : `v_match_full` est un `SELECT mr.*` — DuckDB FIGE l'étoile à la création.
Toute colonne ajoutée à `match_registry` ET lue par cette vue exige un second step de migration
qui recrée la vue (modèle : `refresh_views_after_team_rounds`).

### match_participants — stats de tous les joueurs (31 colonnes)
Colonnes clés : `match_id`, `xuid`, `gamertag`, `outcome` (1=Tie,2=Win,3=Loss,4=DNF), `kills`, `deaths`, `assists`, `shots_fired`, `shots_hit`, `damage_dealt`, `damage_taken`, `mmr`, `team_id`, `rank`

### medals_earned
Colonnes : `match_id`, `xuid`, `medal_id`, `count`, `total_personal_score`

### highlight_events
Colonnes : `match_id`, `xuid`, `event_type`, `timestamp`, `details_json`

### killer_victim_pairs
Colonnes : `match_id`, `killer_xuid`, `victim_xuid`, `count`, `weapon_id`

### xuid_aliases
Colonnes : `xuid`, `gamertag`, `last_seen`

## metadata.duckdb

| Table | Clé | Description |
|---|---|---|
| `career_ranks` | `rank_id` | Paliers et noms des rangs Halo |
| `citation_mappings` | `medal_id` | Mapping médaille→citation |
| `mode_name_tr` | `raw_name`, `lang` | Traductions des noms de modes EN→FR |
| `mode_pair_overrides` | `pair_name` | Surcharges manuelles de paires map/mode |
| `mode_prefix_names` | `prefix` | Préfixes canoniques de modes |
| `weapon_labels` | `weapon_id` (UBIGINT) | Labels EN/FR par weapon_id filmshell |

## shared_pve.duckdb

### pve_match_stats
Stats par joueur par match Firefight : `match_id`, `xuid`, `waves`, `boss_kills`,
`grunt_kills`, `elite_kills`, `jackal_kills`, `brute_kills`, `hunter_kills`,
`skimmer_kills`, `crawler_kills`, `soldier_kills`, `knight_kills`, `warden_kills`

## stats.duckdb (par joueur — data/titles/{slug}/players/{gamertag}/)

**Enrichissements uniquement** — les stats de matchs sont dans shared.

| Table | Description |
|---|---|
| `player_match_enrichment` | `performance_score`, `session_id`, `is_with_friends` |
| `personal_score_awards` | Awards objectifs (PersonalScores API) |
| `match_citations` | Citations calculées par match |
| `match_skill_rank` | Rating LUSR ou CSR par match — **append-only : lire via `match_skill_rank_latest`** |
| `career_progression` | Historique rangs |
| `sessions` | Sessions groupées |
| `media_files` | Fichiers médias indexés |
| `media_match_associations` | Associations médias↔matchs |
| `mv_player_matches` | Vue matérialisée matchs joueur |
| `mv_map_stats` | Vue matérialisée stats par map |

## Requête type — stats coéquipier

```sql
-- Stats d'un coéquipier sur matchs communs (depuis shared, pas sa DB)
SELECT mp.*
FROM shared.match_participants mp
WHERE mp.xuid = '{coequipier_xuid}'
  AND mp.match_id IN (
    SELECT match_id FROM shared.match_participants WHERE xuid = '{mon_xuid}'
  )
```

## Tables append-only + vues `_latest` (ADR 0026 — règle critique)

`match_skill_rank`, `match_csrs`, `player_csr_snapshots`, `pve_match_stats` sont
append-only (PK technique `id` + `written_at`). **Toute lecture applicative passe par la
vue `<table>_latest`** — une lecture de la table brute peut servir plusieurs versions
d'une même ligne (rating non déterministe). Écriture = INSERT pur via la couche
`internal/persist/` (jamais d'UPSERT).

Piège associé : `start_time` NULL dans `_latest` — joindre `match_registry` avec le
COALESCE timezone canonique si un tri temporel est nécessaire.

## Règle connexions (Go — modèle mono-process, ADR 0013/0016)

- Lecture shared : via `SharedProvider` / `SharedReader` (B-swap RO↔RW) — jamais
  `sql.Open` direct sur le fichier.
- Lecture d'une DB potentiellement tenue RW par le process : `OpenReadForQuery`
  (jamais `OpenReadOnly` forcé — erreurs « different configuration »).
- Écriture player : sous lease `AcquirePlayerWriterTimeout` (dblease).
- Requête ad hoc en dev : CLI `duckdb` (READ_ONLY) ou `go run apps/go-api/cmd/inspect_bp/main.go`
  — jamais en RW pendant que le serveur tourne.

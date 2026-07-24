# PLAN V72-03 — Stats objectifs par mode (CTF / Strongholds / KOTH / Oddball)

Chantier parent : PLAN_V72_NOTION_BATCH.md (item V72-03). Auteur : agent architecte
(Opus), 2026-07-24. Statut : plan posé, revue plan-review en cours, exécution non lancée.

## État des lieux (vérifié sur pièces par l'architecte)

1. **Les données sont DÉJÀ dans le payload `GetMatchStats` Halo Infinite, perdues au
   parsing.** `Players[].PlayerTeamStats[0].Stats` contient `CoreStats` (extrait) + blocs
   par mode `CaptureTheFlagStats`, `ZonesStats`, `OddballStats`, `StockpileStats`,
   `EliminationStats`, `ExtractionStats`, `InfectionStats` déclarés `json.RawMessage`
   dans `internal/openspartan/models.go:85-96` (`StatsBundle`) et jamais extraits
   (`internal/sync/transforms.go::ExtractParticipants` ne lit que `CoreStats`).
   Aucun nouvel endpoint requis. Aucune fixture ne contient ces champs → P0 obligatoire.
2. **Rien en DB** : `shared.match_participants` (~40 colonnes, `internal/domain/match_rows.go:89`)
   n'a aucune colonne objectif ; l'import OpenSpartan non plus. Précédent exact de table
   large par-joueur append-only : `pve_match_stats` (`internal/sync/pve.go:35`).
3. **Halo 5** : pas d'agrégat objectif dans le carnage (`dto_carnage.go::H5CarnagePlayer`).
   Dérivable partiellement par comptage d'impulses (`ingest/objective_impulses.go`) ;
   les durées (Time in Zone, Carrier Time) fragiles → capability `degraded`/`not_exposed`
   au départ.
4. **Backfill = re-fetch API** (pas de cache payloads bruts). Précédent :
   `cmd/backfill_kda_accuracy/main.go` (itère `SELECT DISTINCT match_id`, `GetMatchStats`,
   rate-limité `--rps`). Coût : 1 requête par match distinct (~5 req/s).
5. **Surfaces UI** : scoreboard (`match_view_repo_scoreboard.go`, `Q12MatchScoreboard`,
   colonnes gated `useCapability`), `synthesis_repo.go` + `squad_repo*.go` (agrégats),
   `domain/timeseries.go::TimeseriesMatchRow` (champs optionnels `omitempty`).

## Décision « lobby ou équipe » (transmise à l'utilisateur dans Notion)

Stockage TOUJOURS par joueur, tout le lobby : 1 ligne `(match_id, xuid)` comme
`match_participants`/`pve_match_stats`. Agrégats équipe/lobby calculés à la LECTURE
(SUM par `team_id` via JOIN `match_participants` ; SUM globale lobby). Pas de
pré-agrégat stocké (redondant, non ré-exploitable, contraire au pipeline).

## Design table (reco ferme)

`match_objective_stats` dans `shared_matches_v2.duckdb` — table LARGE nullable,
1 ligne par `(match_id, xuid)`, append-only (`id` PK seq + `written_at`) + vue
`match_objective_stats_latest` (QUALIFY ROW_NUMBER … ORDER BY written_at DESC, id DESC).

Colonnes (noms définitifs verrouillés en P0 sur payload réel) :
- CTF : `flag_captures, flag_capture_assists, flag_returns, flag_steals,
  flag_carriers_killed INT, time_as_flag_carrier_seconds DOUBLE`
- Zones (Strongholds ET KOTH partagent `ZonesStats` en Infinite) : `zone_captures,
  zone_secures, zone_offensive_kills, zone_defensive_kills INT, time_in_zones_seconds DOUBLE`
- Oddball : `kills_as_carrier, skull_carriers_killed, skull_grabs INT,
  total_carrier_time_seconds, best_carrier_time_seconds DOUBLE`
- `mode_family VARCHAR` ('ctf'|'zones'|'oddball'|…), extensible stockpile/extraction/
  elimination par ALTER ADD COLUMN nullable.

Rejeté : clé-valeur (vocabulaire fermé ~20-25 champs, pivots partout, incompatible
DTO typés/generate-types) ; table par famille (un match = un mode → tables creuses,
3-4 LEFT JOIN au scoreboard).

Conformité ART : recette `migration.ApplyAppendOnlyRebuild` + `SynthWrittenAt` + `ViewSQL`
(copie de `applyAppendOnlyPveMatchStats`, `steps_appendonly_misc.go:74`) ; ajouter la table
à `tablesProtegees` (`internal/sync/no_art_patterns_test.go:68`) et à l'allowlist
`append_only_state_guard_test.go`.

## Pipeline

- **Collecte** : `internal/sync/objective_stats.go::ExtractObjectiveStats` (pur, calqué
  sur `ExtractPveStats`, `parsePTDuration` pour durées, `mode_family` déduit du bloc).
  Projection canonique dans `internal/games/canonical/` (ADR 0011).
- **Persist** : extension du persister shared existant (PAS de nouveau Persister) :
  `fetchedMatch.ObjectiveStats` (peuplé dans `engine_fetch.go` ET `engine_v2bridge.go`,
  flag `opts.WithObjectiveStats`), `SharedBatch.ObjectiveStats`, helper
  `persistObjectiveStats` INSERT-only dans la transaction atomique (ADR 0019/0030).
- **Capability** : clé dédiée `CapMatchObjectiveStats = "match.objective.stats"`
  (constante `internal/games/adapter.go`, `AllCapabilityKeys()`, 3 `capabilities.toml` :
  infinite=supported, halo_5=degraded ou not_exposed, synthetic selon fixture).
  Gating par mode = data-driven (colonnes NULL), pas capability.
- **Backfill** : `cmd/backfill_objective_stats/main.go` cloné sur `backfill_kda_accuracy`
  MAIS écriture append-only via chemin persist. Bit `MBitObjectiveStats` + reprise.
  Sync natif AVANT backfill historique. Backfill PROD = go séparé au déploiement.
- **UI** : scoreboard (LEFT JOIN `_latest` dans Q12 + section objectif gated + totaux
  équipe/lobby à la lecture), Timeseries (`TimeseriesMatchRow` omitempty), Synthesis +
  Escouade (SUM). openapi.yaml MANUEL + `openapi_schema_drift_test` + `generate-types`.

## Phasage (contrat plan-execution)

- [ ] **P0 — Verrouiller le schéma source** : sonder payloads réels CTF / Strongholds /
      KOTH / Oddball, figer les noms de champs exacts, confirmer KOTH=ZonesStats.
      Gate : liste de champs figée, documentée ici.
- [ ] **P1 — Table + migration append-only.** Gate : `migration_test`,
      `append_only_state_guard_test`, `no_art_patterns_test` verts.
- [ ] **P2 — Extraction + persist + sync natif** (+ tests golden purs).
      Gate : e2e sync insère ; batch roundtrip WAL vert.
- [ ] **P3 — Capability + backfill local.** Gate : dry-run échantillon ; parité capabilities.
- [ ] **P4 — Exploitation UI** (Match view → Timeseries → Synthesis → Escouade).
      Gate : `openapi_schema_drift_test` vert, tests front, dégradation H5 vérifiée.

## Fichiers critiques

`internal/sync/pve.go` (patron extracteur+table), `internal/persist/shared_persister.go` +
`internal/persist/batch.go`, `internal/games/halo_infinite/migrations/steps_appendonly_misc.go`,
`cmd/backfill_kda_accuracy/main.go`, `internal/platform/duckdb/match_view_repo_scoreboard.go`,
`internal/domain/match_view_raw.go`, `config/titles/*/mappings/capabilities.toml`,
`apps/web/src/features/match-view/MatchScoreboard.tsx`.

## Journal

- 2026-07-24 : plan posé (architecte Opus). Revue plan-review déléguée à un agent.
  Reco lobby/équipe transmise dans Notion (champ réponse utilisateur ouvert).

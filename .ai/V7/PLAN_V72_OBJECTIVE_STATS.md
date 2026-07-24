# PLAN V72-03 — Stats objectifs par mode (CTF / Strongholds / KOTH / Oddball)

Chantier parent : PLAN_V72_NOTION_BATCH.md (item V72-03). Auteur : agent architecte
(Opus), 2026-07-24. Revue plan-review (agent Opus, 2026-07-25) : GO avec amendements —
INTÉGRÉS ci-dessous. Statut : prêt à exécuter, exécution non lancée.

Contrat : skill `plan-execution`. Branche : `feat/v7.2-notion-batch`. Statuts d'items
`[x]` fait / `[~]` couvert ailleurs (référence) / `[!]` non traité (justification) —
aucune case vide à la clôture. Reprise de session : reprendre à la première case non
statuée. Zéro fix hors périmètre. NB : tous les chemins Go ci-dessous s'entendent
préfixés `apps/go-api/`.

## État des lieux (vérifié sur pièces par l'architecte, contre-vérifié par la revue)

1. **Les données sont DÉJÀ dans le payload `GetMatchStats` Halo Infinite, perdues au
   parsing.** `Players[].PlayerTeamStats[0].Stats` contient `CoreStats` (extrait) + blocs
   par mode `CaptureTheFlagStats`, `ZonesStats`, `OddballStats`, `StockpileStats`,
   `EliminationStats`, `ExtractionStats`, `InfectionStats` déclarés `json.RawMessage`
   dans `internal/openspartan/models.go:85-96` (`StatsBundle`) et jamais extraits
   (`internal/sync/transforms.go:263` + `findCoreStats`). Aucun nouvel endpoint requis.
   Aucune fixture ne contient ces champs → P0 obligatoire.
2. **Rien en DB** : `shared.match_participants` (`internal/domain/match_rows.go:89`)
   n'a aucune colonne objectif. Précédent de table large par-joueur append-only :
   `pve_match_stats` (`internal/sync/pve.go:35`) — ATTENTION : ce précédent vit dans
   `shared_pve.duckdb` (`steps.go:783`) ; la nouvelle table vit dans
   **`shared_matches_v2.duckdb`** → l'enregistrer dans la chaîne de migration de
   `shared_matches_v2` (`steps_shared_core.go`), pas celle du PvE.
3. **Halo 5** : pas d'agrégat objectif dans le carnage (`dto_carnage.go:33`).
   Impulses partiels (`ingest/objective_impulses.go`) ; durées non fiables.
   **DÉCISION FERME : `halo_5 = "not_exposed"` au lancement** ; promotion `degraded`
   (agrégation d'impulses) = chantier ultérieur distinct.
4. **Import OpenSpartan : HORS PÉRIMÈTRE (décision ferme).** L'import legacy ne peuplera
   pas `match_objective_stats` (chemin d'import historique, pas de re-parse des blocs ;
   les matchs importés seront couverts par le backfill re-fetch comme les autres).
5. **Backfill = re-fetch API** (pas de cache payloads bruts). Précédent :
   `cmd/backfill_kda_accuracy/main.go` (`--rps` défaut 5 l.40, `GetMatchStats` l.89,
   `SELECT DISTINCT mp.match_id` l.135). Coût : 1 requête par match distinct.
6. **Surfaces UI** : scoreboard (`internal/platform/duckdb/match_view_repo_scoreboard.go`,
   `Q12MatchScoreboard`, colonnes gated `useCapability`), `synthesis_repo.go` +
   `squad_repo*.go` (agrégats), `internal/domain/timeseries.go::TimeseriesMatchRow`
   (champs optionnels `omitempty`).

## Décision « lobby ou équipe » (transmise à l'utilisateur dans Notion)

Stockage TOUJOURS par joueur, tout le lobby : 1 ligne `(match_id, xuid)` comme
`match_participants`/`pve_match_stats`. Agrégats équipe/lobby calculés à la LECTURE
(SUM par `team_id` via JOIN `match_participants` ; SUM globale lobby). Pas de
pré-agrégat stocké.

## Design table (reco ferme, amendée)

`match_objective_stats` dans `shared_matches_v2.duckdb` — table LARGE nullable,
1 ligne par `(match_id, xuid)`, **CRÉÉE DIRECTEMENT en forme append-only** (`id` PK seq
+ `written_at` + vue `match_objective_stats_latest` QUALIFY ROW_NUMBER … ORDER BY
written_at DESC, id DESC). NE PAS utiliser `ApplyAppendOnlyRebuild` (recette de
CONVERSION d'une table mutable existante, `steps.go:1383`) — s'inspirer seulement de la
forme finale de `applyAppendOnlyPveMatchStats` (`steps_appendonly_misc.go:74`).
**Index sur `match_id`** dès la création (modèle `idx_pve_match`, `steps.go:806`).

Colonnes (noms définitifs verrouillés en P0 sur payload réel) :
- CTF : `flag_captures, flag_capture_assists, flag_returns, flag_steals,
  flag_carriers_killed INT, time_as_flag_carrier_seconds DOUBLE`
- Zones (Strongholds ET KOTH partagent `ZonesStats` en Infinite) : `zone_captures,
  zone_secures, zone_offensive_kills, zone_defensive_kills INT, time_in_zones_seconds DOUBLE`
- Oddball : `kills_as_carrier, skull_carriers_killed, skull_grabs INT,
  total_carrier_time_seconds, best_carrier_time_seconds DOUBLE`
- **PAS de colonne `mode_family`** (amendement revue : redondante avec le pattern NULL
  et avec le mode déjà porté par `match_registry` — le discriminant se dérive à la
  lecture par jointure si besoin).
- Extensible stockpile/extraction/elimination par ALTER ADD COLUMN nullable.

Rejeté : clé-valeur (vocabulaire fermé ~20-25 champs, pivots partout) ; table par
famille (un match = un mode → tables creuses, jointures multiples).

Conformité ART : ajouter la table à `tablesProtegees`
(`internal/sync/no_art_patterns_test.go:68`) et à l'allowlist
`append_only_state_guard_test.go:54`.

## Pipeline

- **Collecte** : `internal/sync/objective_stats.go::ExtractObjectiveStats` (pur, calqué
  sur `ExtractPveStats`, `parsePTDuration` pour durées). Projection canonique dans
  `internal/games/canonical/` (ADR 0011) ; si nouveaux `FieldKey` canoniques → les
  déclarer dans `canonical/fields.go` ET les TOML de mapping.
- **Persist** : extension du persister shared existant : `fetchedMatch.ObjectiveStats`
  (peuplé dans `engine_fetch.go` ET `engine_v2bridge.go`, flag `opts.WithObjectiveStats`),
  `SharedBatch.ObjectiveStats`, helper `persistObjectiveStats` INSERT-only dans la
  transaction atomique (ADR 0019/0030).
- **Capability** : clé dédiée `CapMatchObjectiveStats = "match.objective.stats"`
  (constante `internal/games/adapter.go`, `AllCapabilityKeys()`, 3 `capabilities.toml` :
  infinite=`supported`, halo_5=`not_exposed` (ferme), synthetic_title_b=`not_exposed`).
  Miroir TS : ajouter la clé à `TITLE_CAPABILITIES` (garde-rail
  `capabilities_ts_mirror_test.go` désormais actif). Gating par mode = data-driven.
- **Backfill** : `cmd/backfill_objective_stats/main.go` cloné sur `backfill_kda_accuracy`
  MAIS écriture append-only via chemin persist. Bit `MBitObjectiveStats` + reprise.
  Sync natif AVANT backfill historique. Backfill PROD = go séparé au déploiement.
- **UI** : scoreboard (LEFT JOIN `_latest` dans Q12 + section objectif gated + totaux
  équipe/lobby à la lecture), Timeseries (`TimeseriesMatchRow` omitempty), Synthesis +
  Escouade (SUM). **i18n/labels obligatoires** : entrées `fields.toml`
  (`config/titles/halo_infinite/mappings/fields.toml`) + `useFieldLabel()` + strings
  FR ET EN dans les manifests — AUCUN label en dur. openapi.yaml MANUEL +
  `openapi_schema_drift_test` + `generate-types`.

## Phasage (contrat plan-execution)

- [ ] **P0 — Verrouiller le schéma source.** Source des payloads (amendement revue) :
      sélectionner en DB 1-2 `match_id` récents PAR MODE (CTF, Strongholds, KOTH,
      Oddball) via `match_registry` (filtre mode/pair_name), puis capturer les payloads
      réels via le client Halo existant avec les tokens du pool (CLI diag existante ou
      petit main jetable sous scratchpad — PAS commité). Figer les noms de champs exacts
      de chaque bloc, confirmer KOTH=`ZonesStats`, documenter le mapping ICI.
      Gate : liste de champs figée + payloads d'exemple sauvés en fixtures de test.
- [ ] **P1 — Table + migration (création directe append-only + index match_id).**
      Chaîne de migration `shared_matches_v2`. Gate : `migration_test`,
      `append_only_state_guard_test`, `no_art_patterns_test` verts.
- [ ] **P2 — Extraction + persist + sync natif** (+ tests golden purs sur fixtures P0).
      Gate : e2e sync insère ; batch roundtrip WAL vert.
- [ ] **P3 — Capability (Go + TS) + backfill local.** Gate : dry-run échantillon ;
      `capabilities_ts_mirror_test` + parité capabilities verts.
- [ ] **P4 — Exploitation UI** (Match view → Timeseries → Synthesis → Escouade) +
      fields.toml/i18n FR-EN. Gate : `openapi_schema_drift_test` vert, tests front,
      dégradation H5 vérifiée (not_exposed → pas de section, pas d'erreur),
      plan d'exécution du JOIN `_latest` au scoreboard vérifié (index utilisé).

## Fichiers critiques

(préfixe `apps/go-api/`) `internal/sync/pve.go`, `internal/persist/shared_persister.go` +
`internal/persist/batch.go`, `internal/games/halo_infinite/migrations/steps_shared_core.go`
(+ `steps_appendonly_misc.go` pour la forme), `cmd/backfill_kda_accuracy/main.go`,
`internal/platform/duckdb/match_view_repo_scoreboard.go`, `internal/domain/match_view_raw.go`,
`config/titles/*/mappings/capabilities.toml` + `fields.toml`,
`apps/web/src/features/match-view/MatchScoreboard.tsx`,
`apps/web/src/lib/capabilities/capabilities.ts`.

## Journal

- 2026-07-24 : plan posé (architecte Opus). Reco lobby/équipe transmise dans Notion.
- 2026-07-25 : revue plan-review (GO avec amendements) intégrée : chaîne de migration
  shared_matches_v2 explicitée, création directe append-only (pas de rebuild), H5
  not_exposed tranché, import OpenSpartan hors périmètre, i18n/fields.toml ajoutés à P4,
  source des payloads P0 documentée, mode_family supprimé, index match_id + gate perf.

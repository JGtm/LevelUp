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

Colonnes — **VERROUILLÉES EN P0 SUR PAYLOAD RÉEL** (cf. section « P0 — Mapping figé »
ci-dessous). Décision d'exécution : on stocke le JEU COMPLET des champs natifs présents
dans chaque bloc (pas le sous-ensemble curé de la reco initiale) — le backfill P3 coûte
1 requête API/match ; capturer tout MAINTENANT évite un second backfill re-fetch complet
pour ajouter un champ oublié. Toutes INT sauf durées en `_seconds DOUBLE`.
- CTF (11) : `flag_captures, flag_capture_assists, flag_grabs, flag_secures, flag_steals,
  flag_returns, flag_carriers_killed, flag_returners_killed, kills_as_flag_carrier,
  kills_as_flag_returner INT, time_as_flag_carrier_seconds DOUBLE`
- Zones (Strongholds ET KOTH partagent `ZonesStats` — CONFIRMÉ P0) (6) : `zone_captures,
  zone_secures, zone_offensive_kills, zone_defensive_kills, zone_scoring_ticks INT,
  time_in_zones_seconds DOUBLE`
- Oddball (6) : `kills_as_skull_carrier, skull_carriers_killed, skull_grabs,
  skull_scoring_ticks INT, time_as_skull_carrier_seconds, longest_time_as_skull_carrier_seconds DOUBLE`
- **PAS de colonne `mode_family`** (amendement revue : redondante avec le pattern NULL
  et avec le mode déjà porté par `match_registry` — le discriminant se dérive à la
  lecture par jointure si besoin ; noter aussi que KOTH se distingue de Strongholds à la
  lecture par `zone_scoring_ticks > 0`).
- Extensible stockpile/extraction/elimination par ALTER ADD COLUMN nullable.

Rejeté : clé-valeur (vocabulaire fermé ~20-25 champs, pivots partout) ; table par
famille (un match = un mode → tables creuses, jointures multiples).

Conformité ART : ajouter la table à `tablesProtegees`
(`internal/sync/no_art_patterns_test.go:68`) et à l'allowlist
`append_only_state_guard_test.go:54`.

## P0 — Mapping figé (payload réel GetMatchStats, 2026-07-25)

Chemin des blocs dans le payload : `Players[].PlayerTeamStats[0].Stats.<BlocMode>`
(même parent `Stats` que `CoreStats` — cf. `findCoreStats`). Blocs **mutuellement
exclusifs par mode** (CTF → seulement `CaptureTheFlagStats` ; Strongholds/KOTH → seulement
`ZonesStats` ; Oddball → seulement `OddballStats`). Un joueur sans aucun de ces 3 blocs
(Slayer, etc.) ne produit AUCUNE ligne. Durées ISO-8601 `PTxMyS` → secondes DOUBLE
(fractions préservées, parser dédié `parseObjectiveDurationSeconds`).

`CaptureTheFlagStats` (match CTF réel `fb1a1a72…`, joueur 0) :
| Champ natif API | Colonne DB | Type | Exemple |
|---|---|---|---|
| `FlagCaptures` | `flag_captures` | INT | 1 |
| `FlagCaptureAssists` | `flag_capture_assists` | INT | 0 |
| `FlagGrabs` | `flag_grabs` | INT | 9 |
| `FlagSecures` | `flag_secures` | INT | 7 |
| `FlagSteals` | `flag_steals` | INT | 3 |
| `FlagReturns` | `flag_returns` | INT | 2 |
| `FlagCarriersKilled` | `flag_carriers_killed` | INT | 1 |
| `FlagReturnersKilled` | `flag_returners_killed` | INT | 0 |
| `KillsAsFlagCarrier` | `kills_as_flag_carrier` | INT | 0 |
| `KillsAsFlagReturner` | `kills_as_flag_returner` | INT | 0 |
| `TimeAsFlagCarrier` | `time_as_flag_carrier_seconds` | DOUBLE | `PT2M20S` → 140.0 |

`ZonesStats` (Strongholds `696a9d7c…` ET KOTH `21ece4d8…` — MÊME bloc, CONFIRMÉ) :
| Champ natif API | Colonne DB | Type | Ex. Strongholds / KOTH |
|---|---|---|---|
| `StrongholdCaptures` | `zone_captures` | INT | 9 / 7 |
| `StrongholdSecures` | `zone_secures` | INT | 1 / 1 |
| `StrongholdOffensiveKills` | `zone_offensive_kills` | INT | 3 / 5 |
| `StrongholdDefensiveKills` | `zone_defensive_kills` | INT | 1 / 4 |
| `StrongholdScoringTicks` | `zone_scoring_ticks` | INT | 0 / 89 |
| `StrongholdOccupationTime` | `time_in_zones_seconds` | DOUBLE | `PT1M52.7S` → 112.7 |

Note : `StrongholdScoringTicks` distingue KOTH (>0, temps sur la colline) de Strongholds
(=0 observé) — discriminant de lecture, pas de colonne `mode_family` nécessaire.

`OddballStats` (match Oddball `d9781168…`, joueur 0) :
| Champ natif API | Colonne DB | Type | Exemple |
|---|---|---|---|
| `KillsAsSkullCarrier` | `kills_as_skull_carrier` | INT | 0 |
| `SkullCarriersKilled` | `skull_carriers_killed` | INT | 3 |
| `SkullGrabs` | `skull_grabs` | INT | 3 |
| `SkullScoringTicks` | `skull_scoring_ticks` | INT | 47 |
| `TimeAsSkullCarrier` | `time_as_skull_carrier_seconds` | DOUBLE | `PT49.1S` → 49.1 |
| `LongestTimeAsSkullCarrier` | `longest_time_as_skull_carrier_seconds` | DOUBLE | `PT11.7S` → 11.7 |

XUID extrait de `PlayerId` = `xuid(<digits>)` (helper `extractPlayerXUID`/`cleanXUID`
existant — bots `bid(...)` conservés, parité `match_participants`/`pve_match_stats`).

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

- [x] **P0 — Verrouiller le schéma source.** FAIT (2026-07-25). Match_ids par mode
      sélectionnés via `diag_q` (RO, serveur tournant), payloads réels capturés via
      `client.GetMatchStats` (auth JGtm store-first ADR 0023, RT vivant/roté ; main
      jetable `cmd/tmp_objcap` supprimé après capture — raison : les packages `internal/`
      ne s'importent pas depuis un module scratchpad externe). Champs figés + KOTH=ZonesStats
      CONFIRMÉ + durées ISO converties (cf. « P0 — Mapping figé »). Fixtures anonymisées
      sauvées : `internal/sync/testdata/objective_stats/{ctf,strongholds,koth,oddball}_match.json`.
- [x] **P1 — Table + migration (création directe append-only + index match_id).** FAIT
      (2026-07-25). Step `shared_create_objective_stats` (TargetShared) créé DIRECTEMENT en
      append-only (séquence `match_objective_stats_seq` + `id BIGINT PK DEFAULT nextval` +
      23 colonnes métier + `written_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP` + index
      `idx_match_objective_stats_match` + vue `match_objective_stats_latest` QUALIFY
      ROW_NUMBER). Fichiers : `internal/games/halo_infinite/migrations/steps_shared_objective_stats.go`
      (+ enregistré dans `steps.go` après commendations, + `order.go` EN FIN de canonicalOrder).
      Ajouté à `tablesProtegees` (no_art_patterns_test.go) + `appendOnlyStateTables`
      (append_only_state_guard_test.go). Gate VERT :
      `go test ./internal/games/halo_infinite/migrations/ -run 'TestSharedObjectiveStatsAppendOnlyShape|TestCanonicalCoversGlobalAndTitle|TestTitleStepsRunEndToEnd_Shared'` = ok ;
      `go test ./internal/sync/ -run 'TestNoARTPatternsOnProtectedTables|TestNoRawDeleteOnAppendOnlyTables|TestAllowlistJustifiesEverything|TestNoMutationOnAppendOnlyStateTables'` = PASS ;
      `go test ./internal/migration/ -run 'Order|Sort|Canonical'` = ok.
- [x] **P2 — Extraction + persist + sync natif** (+ tests golden purs sur fixtures P0).
      FAIT (2026-07-25). `ExtractObjectiveStats` (`internal/sync/objective_stats.go`, pur,
      calqué ExtractPveStats ; retourne `[]persist.ObjectiveStatsInsert` directement —
      décision DRY, parité producteurs kill_positions/commendations, pas de struct row +
      converter dupliqués) + parser durée fractionnaire `objectiveDurationSeconds`.
      `persist.ObjectiveStatsInsert` (rows.go, pointeurs = colonnes nullables) +
      `SharedBatch.ObjectiveStats` (batch.go) + `builder.AddObjectiveStats` (builder.go) +
      `persistObjectiveStats` INSERT-only dans la transaction shared (shared_persister.go,
      ADR 0019/0030). `fetchedMatch.ObjectiveStats` peuplé dans `engine_fetch.go` (sous
      `opts.WithObjectiveStats`, défaut true dans DefaultSyncOptions) ET `engine_v2bridge.go`
      (inconditionnel, parité PVE/PSA — ce chemin ne porte pas de SyncOptions). Câblage
      collect.go. Gate VERT :
      `go test ./internal/domain/ ./internal/persist/` = ok ;
      `go test ./internal/sync/` = ok (27.8s) ;
      golden `go test ./internal/sync/ -run 'TestExtractObjectiveStats|TestObjectiveDurationSeconds'` = PASS (6 tests) ;
      `go test -tags=integration -p 1 ./internal/persist/` = ok (roundtrip _latest + colonne autre-mode NULL + idempotence) ;
      `go test -tags=integration -p 1 ./internal/sync/` = ok (90.6s) ;
      re-run garde-rails ART (persist en place) = ok.
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
- 2026-07-25 : **P0 CLOS** (agent exécution Opus). 8 payloads réels capturés (2/mode) via
  diag_q + GetMatchStats (auth JGtm OK, RT roté). Schéma des 3 blocs figé sur payload réel
  (23 colonnes métier : CTF 11, Zones 6, Oddball 6). Décision d'exécution actée : stocker
  le JEU COMPLET des champs natifs (pas le sous-ensemble curé) pour éviter un 2e backfill
  re-fetch — colonnes ajoutées vs reco initiale : CTF +flag_grabs/flag_secures/
  flag_returners_killed/kills_as_flag_carrier/kills_as_flag_returner ; Zones +zone_scoring_ticks ;
  Oddball +skull_scoring_ticks + renommage total/best → time_as/longest_time_as (mapping natif).
  KOTH=ZonesStats CONFIRMÉ. Fixtures anonymisées commitées sous testdata/objective_stats/.
  Throwaway cmd/tmp_objcap supprimé.
- 2026-07-25 : **P1 CLOS** (agent exécution Opus). Table `match_objective_stats` créée
  directement append-only dans la chaîne shared_matches_v2 (modèle
  create_world_csr_leaderboard_snapshots). Enregistrée provider + canonicalOrder (EN FIN) +
  2 garde-rails ART. Gates verts (migrations end-to-end shared + audit ordre + 4 garde-rails
  ART sync + ordre canonique). Aucun fix hors périmètre.
- 2026-07-25 : **P2 CLOS** (agent exécution Opus). Extraction pure + persist INSERT-only +
  sync natif V1 (opts.WithObjectiveStats, défaut true) et V2 (inconditionnel, parité PVE/PSA).
  Décision d'exécution : ExtractObjectiveStats retourne directement `[]persist.ObjectiveStatsInsert`
  (DRY, parité producteurs kill_positions/commendations) au lieu d'un couple row+converter.
  Parser durée fractionnaire dédié (parsePTDuration tronque en int, colonnes DOUBLE). Tous
  les gates verts (unit domain/persist/sync ; golden 6 tests ; intégration -p 1 persist+sync ;
  re-run garde-rails ART avec persist en place). NB parallèle : `go build ./...` échoue dans
  `internal/service` (match_view `applyTeamNames`, fragdist) — travail d'agents parallèles
  (INTERDITS), hors périmètre, mes 4 paquets (domain/persist/sync/migrations) compilent et
  testent verts. RESTE (vague suivante) : P3 capability Go/TS + backfill ; P4 UI + fields.toml.

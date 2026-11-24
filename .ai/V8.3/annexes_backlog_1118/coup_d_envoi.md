# Coup d'envoi du match : audit T.0 (2026-11-18)

Item T.0 de `.ai/V8.3/PLAN_BACKLOG_2026-11-18.md` (piste T). Lecture seule : aucun code de production modifié, base lue
sur une COPIE (`shared_matches_v2.duckdb` copiée le 2026-11-18 à 12:14, original jamais ouvert, aucun serveur
LevelUp actif au moment de la copie), 1 455 rejeux du poste lus un fichier à la fois.

Deux notions à ne pas confondre dans tout le document :
- le **coup d'envoi** (T0) : l'instant où l'horloge du jeu démarre, en ms depuis le début canonique du match
  (`COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')`), compté **countdown compris** ;
- le **temps de jeu** d'un joueur (`match_participants.time_played_seconds`).

## 0. Ce qui a été lancé (rien d'autre n'est affirmé)

| Quoi | Comment | Sortie |
|---|---|---|
| Copie de la base | `cp` de `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb` vers `C:/t/verite/` (pas de `.wal` voisin) | 483 143 680 octets |
| Requêtes SQL | programme Go jetable `C:/t/duckq` (pilote `duckdb-go` du projet, `access_mode=read_only`, hors dépôt) ; requêtes `C:/t/verite/q_*.sql` | TSV et tableaux ci-dessous |
| Lecture des rejeux | `C:/t/verite/rejeux.js` (horloge de manche, `t0FilmMs`, première élimination, présence) et `C:/t/verite/m3all.js` (premier déplacement horizontal réel des pistes), un fichier à la fois | `rejeux.tsv`, `rejeux_m3.tsv` (1 455 lignes chacun) |
| Export de vérité | requêtes `q_part.sql`, `q_reg.sql`, `q_evt.sql` | `participants.tsv` (29 107 lignes), `registre.tsv` (2 045), `evenements_mode.tsv` (58) |

Limites de la copie : `match_registry` compte 2 045 matchs, 1 455 ont un rejeu (le nom du fichier est les 8 premiers
caractères du match, sans collision dans le registre). Le registre n'a pas de colonne « build » : seule
`sync_spnkr_version` existe, vide sur les matchs examinés ; le document de rejeu et son `.meta` ne portent pas de
build non plus. `highlight_events` n'a pas de `details_json` : le code d'un événement de mode est `type_hint` (10 sur
la grande majorité), le détail est dans `raw_json`.

## A. Inventaire fermé

### A.1 Sources du coup d'envoi (et de la durée) : qui écrit quoi

| Id | Source | fichier:ligne | Ce qu'elle écrit | Quand |
|---|---|---|---|---|
| S1 | Estimation par l'API : médiane ou minimum des `first_joined_time` des joueurs présents au début, hors bots | `internal/analysis/timeline/compute_t0.go:88` (`ComputeT0`) ; appel `internal/sync/transforms.go:123` via `computeMatchT0` `:152` | `real_start_time` = début + T0, seulement si la qualité est « calculée » (`transforms.go:123-126`) ; **`t0_quality` n'est jamais écrite** à la synchronisation : `persist/shared_persister.go:164` et `:192` n'insèrent que `real_start_time` | synchronisation d'un match (INSERT) |
| S1b | Même estimation, deuxième appel dans la fenêtre de jeu | `internal/sync/transforms.go:188-206` (`computeMatchWindow`, appel `:203`) | rien en base : alimente le calcul du temps de jeu (S6) | synchronisation |
| S2 | Report du film : `t0FilmMs` publié par le document de rejeu | `internal/sync/replayartifacts/t0film.go:94` (`reporterT0Film`), `:150` (`ecrireUnT0Film`) ; extraction `derivations.go:358` (`rapportsT0`), appel `:152` ; écriture `internal/persist/t0_film_persister.go:47-75` (`MarkT0Film`) | `real_start_time` = début + `t0FilmMs`, `t0_quality = film_movement` ; garde « déjà là » `t0film.go:167` | au fil de l'eau, 5 artefacts au plus par cycle (`replayartifacts/artifacts.go:62`), **chemin local seulement** (`.ai/REGISTRE_REPORTS.md:48` : le chemin `worker`, défaut de la prod, ne reporte rien) |
| S3 | Commande : estimation sur tout le corpus | `cmd/backfill_t0/main.go` (`ComputeT0` `:84`, écriture `commitResults` `:125`, UPDATE `:130` et `:135`, garde `ecarterLesFilms` `:165` + `loadFilmT0Matches` `:185`) | `real_start_time` (qualités calculées) et `t0_quality` ; les rejets n'écrivent que la qualité | manuel, `--commit`, serveur arrêté |
| S4 | Commande : report du film sur tout le parc de rejeux | `cmd/backfill_t0_film/registre.go:84-116` (UPDATE `:100`), détecteur `cmd/backfill_t0_film/artefacts.go:131` (`replay.DetectT0Film`), garde `main.go:167` | idem S2, **écriture directe hors `persist`** | manuel ; lancée avec `--commit` le 2026-11-18 (`.ai/thought_log.md:36`) |
| S5 | Commande : recalcul du temps de jeu | `cmd/backfill_time_played/main.go:197` (UPDATE de `time_played_seconds`), fenêtres `:122-152` | `match_participants.time_played_seconds` (qualité « ok » seulement) | manuel, `--commit` ; **jamais lancée depuis le report du 2026-11-18** (mesure B.4) |
| S6 | Temps de jeu au sync | `internal/sync/transforms.go:367` (valeur API `TimePlayed`), `:381-392` (remplacement par `ComputeTimePlayedFor`) ; formule `internal/analysis/timeline/compute_time_played.go:54-83` | `time_played_seconds` | synchronisation, sur le T0 de l'estimation (S1b) |
| S7 | Colonne héritée `film_match_start_ms` | `internal/games/halo_infinite/migrations/steps_shared_core.go:66` et `:164` | **aucun écrivain ni lecteur Go** ; 814 lignes non vides dans la copie (5 002 à 118 466 ms) | héritage Python |
| S8 | Fonction `EstimateFilmMatchStartMS` (+ `ScanFirstMovements`, `FindPeakActivityWindow`) | `internal/analysis/spawn_detection.go:86`, `:122`, `:238` | aucun appelant hors tests : code mort | - |
| S9 | Import OpenSpartan | `internal/openspartan/mapper/mapper.go:62-72`, `participants.go:88` | durée jouable de l'API et temps de jeu de l'API ; aucun coup d'envoi | import |
| S10 | Décodeur de film (hors dépôt, brique `halo-infinite-film-decoder`) | consommé par le document de rejeu : champs `originMs`, `t0FilmMs` (`internal/domain/replaydoc/document.go:24`), `roundClock[]` | publie deux datations du coup d'envoi, voir B.2 | cuisson d'un rejeu |
| S11 | Démo | `internal/persist/demo_seed_columns.go:30-31` | recopie `duration_seconds`, `playable_duration_seconds`, `real_start_time` | graine de démo |

Halo 5 : aucune source (A.7).

### A.2 Paramètres, seuils, qualités

| Nom | Valeur | Rôle | fichier:ligne |
|---|---|---|---|
| `t0SpreadThresholdMs` | 2 000 ms | écart maximal entre arrivées pour retenir le minimum ; au-delà : médiane | `compute_t0.go:57`, usage `:112` |
| `t0MaxPlausibleMs` | 120 000 ms | au-delà : rejet `suspicious_high` | `compute_t0.go:60`, usage `:124` |
| seuil négatif | 0 | T0 < 0 : rejet `negative` | `compute_t0.go:121` |
| filtre de présence | présent au début, non bot, `FirstJoinedTime` non nul | sélection des sources | `compute_t0.go:91-96` |
| `maxPerCycle` | 5 | artefacts reportés au registre par cycle | `replayartifacts/artifacts.go:62` |
| `TimePlayedQuality` | `ok`, `no_data`, `clamped_zero` | `clamped_zero` est stockable et vaut 0 s | `compute_time_played.go:14-26` |
| `perfFallbackDurationSeconds` | 600 s | durée supposée sans temps joué (note de performance) | `internal/sync/performance.go:64` |
| `teamWeightFloor` | 0,1 | plancher du poids LUSR d'un joueur | `internal/analysis/skill_v2/ep/match2team.go:73` |
| `botLateJoinIgnoreRatio` | 0,30 | bot arrivé en cours ignoré sous 30 % du match | `internal/sync/enrichments.go:44`, SQL `:140-155` |
| `ObjectiveScoreMinPlayedSeconds` | 30 s | joueur retenu pour le score d'objectif | `internal/analysis/narrative/objective_match_score.go:46` |
| plancher de durée LUSR | 30 s | `duration_seconds >= 30` | `skill_v2_shadow.go:454`, `skill_rating_loaders.go:76` |
| plancher de carrière | 180 s | `time_played_seconds >= 180` | `internal/platform/duckdb/queries_career.go:276` |
| plancher de la page match | 60 s | historique du mode retenu si durée > 60 s | `internal/service/match_view_builders_summary.go:188` |
| `LEAD_IN_MS` (web) | 1 000 ms | une seconde de préambule avant le coup d'envoi | `apps/web/src/features/match-replay/model/replayWindow.ts:81` |

Les sept valeurs de `t0_quality` (`compute_t0.go:14-38`) :

| Valeur | Écrite par | LUE par, pour décider quoi | Lignes dans la copie |
|---|---|---|---|
| `ok` | S3 (`ComputeT0`, ≥ 2 sources à moins de 2 s) | personne à l'exécution ; `backfill_t0` ne la lit pas ; sert seulement de libellé | 583 |
| `single_source` | S3 | personne | 0 |
| `spread_high` | S3 | personne | 0 |
| `no_data` | S3 (qualité seule, valeur non écrite) | personne | 0 |
| `negative` | S3 (qualité seule) | personne | 11 (mais `real_start_time` porte quand même une valeur, voir A.3 et B.1) |
| `suspicious_high` | S3 (qualité seule) | personne | 0 |
| `film_movement` | S2, S4 (jamais par `ComputeT0`) | garde « ne pas écraser le film » : `backfill_t0/main.go:185-197` et `:165` ; garde « déjà à jour » : `replayartifacts/t0film.go:159-167`, `backfill_t0_film/main.go:167` | 1 449 |
| (vide) | tout match synchronisé depuis le dernier passage de `backfill_t0` | - | 2 |

`Computed()` (`compute_t0.go:43`) est la seule lecture « décisionnelle » : `transforms.go:123`, `:203`, `backfill_t0/main.go:87`,
`:143`. **Aucun lecteur de production (service, API, web) ne lit `t0_quality`** : grep sur `apps/go-api/internal`,
`apps/go-api/cmd`, `apps/web/src`. Tous les lecteurs de A.5 lisent `real_start_time` sans regarder la qualité.

### A.3 Replis : un coup d'envoi ou une durée absents remplacés

| # | Repli | fichier:ligne |
|---|---|---|
| R1 | coup d'envoi absent => T0 = 0 (builders de chronologie) | `internal/analysis/timeline/build.go:30-36` (`BuildFromRegistry`), `:51` et `:65` (`derefInt64`), `:74` |
| R2 | `SQLT0Ms` rend NULL sans `real_start_time`, le consommateur retombe sur 0 | `internal/platform/duckdb/sqlfrag/sql_fragments.go:95-100` ; `match_events_source.go:65-73` (`t0Ms.Int64` = 0 si NULL) |
| R3 | durée de jeu du registre : `GREATEST(0, durée - T0)` avec `ELSE 0` quand pas de coup d'envoi | `platform/duckdb/queries_squad.go:89-97` ; `sync/skill/skill_v2_shadow.go:441-446` |
| R4 | T0 canonique nul => durée brute | `internal/games/canonical/match.go:84-92` |
| R5 | durée de l'en-tête : `durée - T0`, sinon durée jouable de l'API | `internal/service/match_view_builders_header.go:81-92` |
| R6 | durée de la Dominance : durée jouable de l'API d'abord, puis durée - T0, puis 0 | `internal/service/match_view_data_loaders.go:249-259` (`tugDurationMS`) |
| R7 | T0 absent dans le service de page => 0 | `internal/service/match_view_shared_build.go:50-54` |
| R8 | T0 d'en-tête affiché seulement s'il est > 0 | `match_view_builders_header.go:62-64` ; `service/teammates/teammates_squad_charts_medal_digest.go:34` |
| R9 | temps de jeu, sinon durée jouable, sinon rien | `platform/duckdb/queries_home_citations.go:81` |
| R10 | temps de jeu, sinon durée du match, sinon 0 | `internal/analysis/trends/match.go:153-161` (`secondsPlayed`) |
| R11 | temps de jeu absent => 600 s | `internal/sync/performance_helpers.go:52-56` (`effectivePlayedSeconds`), `:241` et `:254` (`COALESCE(mp.time_played_seconds, ?)`) |
| R12 | `COALESCE(time_played_seconds, 0)` | `post_sync_progression_store.go:67` ; `queries_squad.go:87`, `:156` ; `player_matches_repo.go:188`, `:313` ; `queries_career.go:276` ; `objective_team_rows_repo.go:40` ; `sync/enrichments.go:148-151` |
| R13 | poids LUSR : durée de jeu inconnue ou temps de jeu absent => poids 0, puis 1 côté solveur | `sync/skill/skill_v2_quit_penalty.go:354-361` ; `analysis/skill_v2/ep/match2team.go:77-90` |
| R14 | quitteur principal : horodatage de départ, sinon plus petit temps de jeu | `skill_v2_quit_penalty.go:80-140` |
| R15 | durée de jeu « observée » : médiane du temps de jeu des joueurs du début à la fin, sinon maximum | `platform/duckdb/elapsed_seconds.go:35-45` (sert le drapeau Prolongation, `match_history_service_enrich.go:121`, `match_view_builders_header.go:260`) |
| R16 | temps de jeu de l'API conservé quand le calcul est inexploitable ou la fenêtre inconnue | `sync/transforms.go:381-392`, `:188-206` |
| R17 | coup d'envoi non calculé à la synchronisation => `real_start_time` NULL | `sync/transforms.go:117-126` |
| R18 | fin de session = début + temps de jeu | `internal/sync/session_recalc.go:37-41` ; `platform/duckdb/queries_career_encounters.go:331-336` |
| R19 | web : coup d'envoi du film, sinon celui de l'API | `apps/web/src/features/match-replay/model/replayWindow.ts:126` |
| R20 | web : T0 d'en-tête non fini => 0 ; décalage du fil à 0 | `apps/web/src/lib/replay/matchClock.ts:142` ; `features/match-replay/model/replayModel.ts:158` ; `model/killFeedLogic.ts:148-149` |
| R21 | web : durée de jeu d'escouade, sinon durée | `apps/web/src/features/squad/SquadSynergyHistoryTable.tsx:325` |
| R22 | fenêtre jouée du rejeu : sans coup d'envoi mesuré, refus ; fin = durée du match | `internal/analysis/leadshare/window.go:26-41` |

### A.4 Durées : chaque lecteur

Quatre grandeurs concurrentes : durée du registre (`duration_seconds`, début à fin du film, countdown compris), durée
jouable de l'API (`playable_duration_seconds`), « durée - coup d'envoi » (calculée), durée observée (R15). Mesure : la
durée jouable égale la durée sur 1 561 matchs sur 2 045, diffère sur 484 ; elle égale la durée sur 18 des 34 matchs
classés (le plan la dit fausse en Arène classée).

| Lecteur | Grandeur lue | Ce qu'il en fait | fichier:ligne |
|---|---|---|---|
| Écriture sync | `Duration`, `PlayableDuration` de l'API | rangées telles quelles | `internal/sync/transforms.go:106-107` ; `persist/shared_persister.go:163`, `:191` |
| Lecteur de page match (méta) | durée, durée jouable, T0 | `MatchMetaRaw` | `platform/duckdb/queries_match.go:125-137`, `:151` ; `match_view_repo.go:218-236` |
| En-tête de match | durée - T0, sinon durée jouable | champ servi `playable_duration_seconds` (nom trompeur : c'est la durée - T0) | `service/match_view_builders_header.go:59`, `:81-92` |
| Dominance | durée jouable d'abord | bins de la dominance | `match_view_data_loaders.go:249-259` |
| Onglet résumé | durée du match | durée courante pour l'historique du mode | `match_view_data_loaders.go:305-306` ; `match_view_builders_summary.go:188` |
| Squad (Q30) | durée et durée de jeu SQL | `gameplay_duration_seconds` ; cadences par minute | `queries_squad.go:88-97` ; `squad_service_v2_history.go:91-96` ; `squad_service_v2_contributions.go:60-63` ; `squad_service_v2_compose.go:73` |
| Séries et sessions | durée de jeu en ms par match | dénominateur de l'intensité et de la cadence | `analysis/timeline/build.go:86-98` ; `service/timeseries_service.go:365-374` ; `service/session_page_service.go:377-379` |
| Fenêtre jouée du rejeu (Go) | durée du registre - origine | borne de fin | `analysis/leadshare/window.go:40` ; `sync/replayartifacts/leadshare.go:100-143` |
| Fenêtre jouée du rejeu (web) | `playable_duration_seconds` (donc durée - T0) | borne de fin ; le début lit `t0FilmMs` | `apps/web/src/features/match-replay/model/replayWindow.ts:116-129` |
| En-tête web | `playable_duration_seconds` | affichage de la durée | `apps/web/src/features/match-view/MatchHeader.card.tsx:259-273` |
| LUSR | durée - T0 | dénominateur du poids de chaque joueur | `sync/skill/skill_v2_shadow.go:441-446`, `:465`, `:650` |
| Prolongation | durée observée (R15) | drapeau | `platform/duckdb/elapsed_seconds.go` ; `match_history_repo.go:183-202` ; `match_view_repo.go:179-195` |
| Briefing, explorateur | `duration_seconds` | somme des durées d'un périmètre | `service/match_history_service_briefing.go:134-148` ; `platform/duckdb/queries_career.go:110` ; `patterns_repo.go:176` |
| Bots significatifs | `duration_seconds`, temps de jeu des bots | seuil 30 % | `sync/enrichments.go:140-160` |
| Comeback | `duration_seconds` | borne des événements | `sync/comeback.go:190` |
| Complétion du snapshot | `duration_seconds` | éligibilité | `sync/snapshot/snapshot_readiness_eval.go:200` |
| Web : tables et courbes | `duration_seconds` des lignes de séance, de l'explorateur, de l'escouade | affichage, écart d'engagement | `features/explorer/ExplorerMatchesTable.tsx:732` ; `features/session-detail/SessionMatchesTable.tsx:125`, `sessionPerMinute.logic.ts:35` ; `features/timeseries/TimeseriesEngagementGapTrend.tsx:59` ; `lib/charts/engagementGap.ts:26` |

Piège de nommage : dans les lignes de séance et de l'explorateur, `duration_seconds` est le **temps de jeu du joueur**,
pas la durée du match (`service/session_page_service.go:721`, `service/match_history_service_enrich.go:257`).

### A.5 Lecteurs du coup d'envoi

| Lecteur | Ce qu'il lit | Ce qu'il en fait | fichier:ligne |
|---|---|---|---|
| Fragment SQL unique | `real_start_time` | `SQLT0Ms`, `SQLRealStartOffsetMs` ; ratchet `archlint/no_raw_real_start_offset_test.go` | `platform/duckdb/sqlfrag/sql_fragments.go:80-100` ; délégué `player_matches_repo.go:158-161` |
| Lignes canoniques de joueur | `t0_ms` | `Summary.T0Ms` | `player_matches_repo.go:346-347`, `player_matches_projection.go:131`, `player_matches_scan.go:109`, `:206` ; `games/canonical/match.go:71` |
| Méta de match | `t0_ms` | `MatchMetaRaw.T0Ms` => en-tête, recalage des événements | `queries_match.go:151` ; `match_view_repo.go:236` ; `service/match_view_builders_header.go:62-63`, `:86-87` ; `match_view_shared_build.go:50-54` ; `domain/match_view.go:168-180` |
| Recalage des événements | T0 | `CorrectEventTime` (temps de film - T0) | `domain/match_timeline.go:98-100` ; `analysis/timeline/correct_events.go:35-149` ; `games/halo_infinite/events.go:42-78` ; `service/match_view_data_loaders.go:393-410` |
| Squad | `t0_ms` | chronologies d'escouade | `queries_squad.go:97` ; `squad_repo.go:281` ; `squad_service.go:81` ; `squad_service_v2.go:213-262` ; `teammates/teammates_squad_charts_impact_events.go:70` ; `..._intensity_perminute.go:170` ; `teammates_squad_impact.go:155` |
| Séries, sessions | chronologies | intensité par minute | `service/timeseries_service.go:365` ; `session_page_service.go:377` |
| Tactique | `t0_ms` | instant de la mort depuis le coup d'envoi | `platform/duckdb/tactical_repo_univers.go:98`, `:251`, `:308` ; `domain/tactical.go:320-324` ; `service/tactical_service_sequence.go:80` ; `tactical_service_cellule_horloge.go:29` ; `analysis/tactical/sequence.go:202-203` |
| LUSR | `real_start_time` via `SQLRealStartOffsetMs` | durée de jeu du match | `sync/skill/skill_v2_shadow.go:441-446` ; `skill_v2_quit_penalty.go:186-198` (commentaire : le repère des frags reste le début du film, pas le coup d'envoi) |
| Temps de jeu | `real_start_time` | `MatchTimeline` du recalcul | `sync/transforms.go:188-206` ; `cmd/backfill_time_played/main.go:122-152` |
| Fenêtre jouée (Go) | `t0FilmMs` du document | début de la fenêtre | `analysis/leadshare/window.go:34-39` ; `sync/replayartifacts/leadshare.go:143` |
| Champ de piste du rejeu | `t0FilmMs` | lu dans `TacticalPistes`, aucune lecture trouvée | `domain/tactical_equipe.go:213` |
| Web : horloge du match | `originMs`, `t0FilmMs` du document ; `t0_ms` de l'en-tête | conversions film / match | `apps/web/src/lib/replay/matchClock.ts:72-158` ; `features/match-replay/model/replayClock.ts:20-70` |
| Web : fenêtre de lecture | `t0FilmMs ?? t0_ms` pour le début, `t0_ms` + durée jouable pour la fin | image de début, de fin, préambule | `features/match-replay/model/replayWindow.ts:116-132` |
| Web : horloge affichée | début de la fenêtre | `displayClockMs` (0:00 au coup d'envoi) | `replayWindow.ts:143-146` ; `hooks/useReplayClock.ts:83` ; `hooks/useReplayTimeline.ts:169` ; `export/useReplayExport.ts:460` |
| Web : fil, médias, objectifs | `t0_ms` et `originMs` | recalage `event_time_ms + t0Ms - originMs` | `features/match-replay/model/killFeedLogic.ts:141-150`, `:256-305` ; `layers/objectivesLayer.ts:370` ; `model/replayMediaLogic.ts:11` ; `lib/replay/aliveBalanceLogic.ts:108` |
| Web : graphiques de match | `t0_ms` | ancre des courbes | `features/match-view/MatchScoreCurveChart.tsx:97`, `MatchScoreDeltaChart.tsx:78`, `MatchScoreEventsChart.tsx:90`, `MatchNumbersAdvantageChart.tsx:63`, `MatchViewPage.tsx:311` |
| Web : `roundClock` du document | schéma et normalisation seulement | **aucune lecture** (champ publié sans lecteur côté LevelUp ; légitime, règle 19 : décodeur public) | `lib/replay/replayDocumentSchema.ts:157` ; `replayNormalize.ts:241` |

### A.6 Temps de jeu d'un joueur (`time_played_seconds`)

Écrivains : S6 (au sync : valeur API puis formule ci-dessous), S5 (rattrapage), import OpenSpartan
(`openspartan/mapper/participants.go:88`, `service/openspartan_import_adapters.go:108-136`), Halo 5
(`games/halo_5/mapping_carnage.go:105`, `cmd/h5-roster-refetch/insert.go:73`), graine de démo
(`ops/seed_demo_synthetic_shared.go:190`). Colonne créée `steps_shared_core.go:90`, `:193`.

Formule du sync (`compute_time_played.go:54-83`) : `joined = max(first_joined, début + T0)`,
`left = min(last_leave ou fin, début + durée)`, `secondes = int(left - joined)` (troncature : 1 s de moins en
moyenne, mesuré B.4). `first_joined` absent : valeur de l'API conservée. Le T0 utilisé est celui de l'estimation
(S1b), pas celui du registre.

Lecteurs :

| Lecteur | Usage | fichier:ligne |
|---|---|---|
| Note de performance | durée de toutes les cadences par minute (600 s si absente) ; faits d'objectif | `sync/performance.go:87` ; `performance_helpers.go:45-67`, `:241-272` ; `sync/objectiveblend` |
| LUSR | numérateur du poids d'un joueur ; ordre des quitteurs en repli | `sync/skill/skill_v2_quit_penalty.go:354-361`, `:84-85`, `:122-140` ; `skill_v2_shadow.go:488`, `:501`, `:533` |
| Pages, cadences | par minute : score personnel, frags, séries | `service/stats_service.go:229-230` ; `timeseries_service_buckets.go:55-58`, `:89-90` ; `timeseries_service_tabs.go:123-124` ; `teammates/teammates_squad_charts_synergy.go:70` ; `analysis/explorer_target_stats.go:116-122` ; `analysis/stats_canonical.go:56` |
| Agrégats | totaux de temps joué | `service/synthesis_service_builders.go:203` ; `analysis/squad_breakdown_canonical.go:165` ; `service/session_compare_participation_helpers.go:130` ; `platform/duckdb/explorer_repo.go:148` |
| Objectifs | tenue = durée / temps de jeu ; seuil 30 s | `analysis/narrative/objective_participation.go:189-298` ; `objective_match_score.go:94` ; `objective_share.go:35` ; `platform/duckdb/objective_team_rows_repo.go:40` |
| Filtres | `MinTimePlayedSeconds` ; carrière >= 180 s | `platform/duckdb/player_matches_repo.go:187-189` ; `queries_career.go:276` |
| Sessions, rencontres | fin = début + temps de jeu | `sync/session_recalc.go:37-41` ; `queries_career_encounters.go:331-336` ; `domain/sessions.go:13` |
| Durée observée | médiane des joueurs du début à la fin | `platform/duckdb/elapsed_seconds.go:35-45` |
| Progression | cumul | `platform/duckdb/post_sync_progression_store.go:67` |
| Bots | ratio 0,30 | `sync/enrichments.go:140-160` |
| Rejeu (oracle) | durée officielle non datée sur l'horloge du film | `platform/duckdb/replay_oracle_repo.go:73-96` ; `domain/match_oracle.go:33` |

**Dépendance de la note LUSR et de la note de performance (vérifiée sur le code, mesurée sur la copie).**

1. La note de performance lit `time_played_seconds` (`performance_helpers.go:241`) : un changement de cette valeur
   change KPM, DPM, APM, PSPM et les faits d'objectif, donc la note. Elle ne lit pas le coup d'envoi.
2. Le LUSR (perimetre : matchs non classés, hors Firefight, `skill_v2_shadow.go:449-454`) lit **les deux** :
   poids du joueur = `min(1, temps de jeu / (durée - T0 du registre))` (`skill_v2_quit_penalty.go:354-361`), plancher
   0,1 (`match2team.go:85`), et le temps de jeu sert à ordonner les quitteurs en repli (`:122-140`). Donc un changement
   de `time_played_seconds` OU de `real_start_time` change le LUSR.
3. Conséquence mesurée (matchs LUSR non classés, joueurs non bots, `q_lusr.sql`) : sur les 13 283 couples
   joueur-match dont le coup d'envoi a changé avec le report du film, le poids passe d'une médiane de 0,999 à 1,000,
   le nombre de poids sous 0,99 passe de 2 065 à 4 048, l'écart de poids a pour médiane 0,002 et pour maximum 0,378.
   **Les notes stockées ont été calculées avant le report : tout recalcul (rejeu de révision, exclusion de match)
   changera déjà des notes, même sans toucher au temps de jeu.**
4. ADR 0047 : la révision est une constante de code comparée à la valeur stockée par joueur ; elle se bascule quand une
   formule ou un paramètre change, et l'ADR dit que « un changement de logique sans changement de nombre n'est pas
   détecté : la révision se change à la main » (`docs/adr/0047-scoring-revision-replay.md`, section « Known limits »).
   Un rattrapage de données (nouvelles valeurs de `time_played_seconds` ou de `real_start_time` en base, formule
   inchangée) ne relève pas du mécanisme : aucune révision ne change, le rejeu au démarrage n'a pas lieu, les notes
   restent périmées jusqu'au prochain recalcul. En revanche, **changer la formule qui produit le temps de jeu
   (T.2 à T.4) est un changement de logique d'une entrée des deux notes** : il appelle de monter `PerformanceScoreRev`
   (`sync/skill/scoring_revision.go`, valeur actuelle `perf-2026-11-15b`) et `LUSRRev` (`lusr-2026-11-15`) à la main,
   le garde-rail d'empreinte (`TestScoringParametersMatchRevision`) ne le voyant pas. C'est une lecture, pas une
   décision : à valider au modèle (T.1).

### A.7 Halo 5

| Notion | Ce que fait Halo 5 | fichier:ligne |
|---|---|---|
| Coup d'envoi | aucun : `T0Ms` toujours nul, pas de `real_start_time` ni de `t0_quality` | `games/halo_5/mapping.go:120` ; `platform/duckdb/halo5/halo5_match_history_source.go:206`, `:226` |
| Durée du match | durée de l'API, `end_time` = début + durée | `games/halo_5/mapping.go:110` ; `games/halo_5/ingest/registry.go:24`, `:47-49` |
| Durée jouable | jamais écrite (100 % NULL côté Halo 5) : d'où le repli de la Dominance sur durée - T0 (R6) | `match_view_data_loaders.go:243-259` |
| Temps de jeu | `TotalTimePlayed` de l'API, tel quel, jamais recalculé | `games/halo_5/mapping_carnage.go:105` ; `cmd/h5-roster-refetch/insert.go:73` |
| LUSR | même poids `temps de jeu / durée` (chaîne `games/halo_5/lusr_chain.go:11`), avec T0 = 0 | `skill_v2_shadow.go:441-446` (ELSE 0) |

Aucune donnée Halo 5 dans la copie examinée (dossier `data/titles/halo_5` non lu : hors périmètre de la demande).

## B. Mesures

### B.1 Répartition de `t0_quality` (copie, 2 045 matchs)

| `t0_quality` | Matchs | Avec `real_start_time` | Avec rejeu | Sans rejeu |
|---|---|---|---|---|
| `film_movement` | 1 449 | 1 449 | 1 447 | 2 |
| `ok` | 583 | 583 | 6 | 577 |
| `negative` | 11 | 11 | 0 | 11 |
| vide | 2 | 2 | 2 | 0 |
| `single_source`, `spread_high`, `no_data`, `suspicious_high` | 0 | - | - | - |
| Total | 2 045 | 2 045 | 1 455 | 590 |

Matchs sans aucune valeur (colonne `real_start_time` vide) : 0. Matchs sans qualité : 2, avec une valeur.

Valeur du T0 du registre, par qualité (secondes, `q_t0.sql`) :

| Qualité | Médiane | Centiles 5 et 95 | Min et max | Remarque |
|---|---|---|---|---|
| `film_movement` | 34,1 | 26,2 et 55,9 | 15,2 et 213,2 | 1 match au-delà de 120 s |
| `ok` | 0,0 | 0,0 et 67,7 | 0 et 118,5 | **350 des 583 sont entre 0 et 5 s** (dégénérés) ; 349 d'entre eux sont sans rejeu |
| `negative` | 14 400 | 7 200 et 14 400 | 7 200 et 14 400 | **les 11 portent un `real_start_time` décalé de 2 h à 4 h** (dette de fuseau historique, dates 2021-11 à 2023-12) ; les lecteurs le servent comme T0 de 2 à 4 heures (`SQLT0Ms` ne regarde pas la qualité) |
| vide | 24,5 | 18,2 et 30,8 | 17,5 et 31,5 | synchronisés depuis le dernier `backfill_t0` |

Matchs sans rejeu et sans film (577 `ok` + 11 `negative` + 2 `film_movement` = 590) : leur seule source est l'estimation
de l'API, dégénérée (T0 < 5 s) pour 349 d'entre eux, rejetée mais servie pour 11.

### B.2 Matchs avec rejeu : registre contre horloge de manche et premier mouvement

Définitions : l'horloge de manche est lue par `roundClock[]` (départ de la première lecture où l'horloge court, calcul
du script `pilote_mesure_horloge.js.txt`), le premier mouvement est `t0FilmMs`. Écart = coup d'envoi du registre moins la
datation (secondes ; `q_b2b.sql`).

| Qualité | n | Registre - horloge : centile 5 / médiane / centile 95 | min / max | Registre - premier mouvement |
|---|---|---|---|---|
| `film_movement` | 1 447 | 0,30 / 0,36 / 0,53 | -21,4 / +1,5 | 0 pour les 1 447 (le report est exact) |
| `ok` | 6 | -31,7 / -5,1 / -0,3 | -34,0 / +0,6 | pas de `t0FilmMs` |
| vide | 2 | -9,8 / -6,6 / -3,4 | -10,2 / -3,1 | pas de `t0FilmMs` |
| Total | 1 455 | 0,30 / 0,36 / 0,53 | -34,0 / +1,5 | - |

Sur les 1 455 rejeux : horloge calculable pour 1 455 ; `t0FilmMs` présent pour 1 447, absent pour 8 (refus
`burstTooSmall` du détecteur) ; l'écart « premier mouvement - horloge » vaut +0,30 / +0,36 / +0,53 s (centiles 5 /
50 / 95) sur les 1 425 rejeux sains, et dépasse 3 s en valeur absolue pour 22 rejeux (point B.3).
Pour les 8 refusés, le registre (estimation de l'API, en place faute de film) est à -34,0 / +0,6 s de l'horloge : médiane -5,1 s pour les 6 `ok`, -6,6 s pour les 2 sans qualité.

### B.3 Les 22 rejeux : l'horloge et le premier mouvement diffèrent de 17 s

**Verdict : l'horloge de manche a raison, le détecteur de premier mouvement a tort pour ces 22 rejeux.** Preuves
chiffrées, toutes indépendantes de l'horloge et du détecteur :

1. **Positions des pistes (référence indépendante).** Pour chaque rejeu j'ai relevé l'image du troisième joueur dont la
   position horizontale change de plus de 5 cm entre deux images (`m3all.js`). Sur les 22 rejeux : ce déplacement réel
   survient **0,17 à 0,84 s après le départ de l'horloge** (médiane 0,23 s) et **16,8 à 21,6 s après `t0FilmMs`**
   (médiane 17,03 s). `t0FilmMs` tombe donc 17 s avant que quiconque bouge. Sur les 1 425 rejeux sains, le même
   déplacement suit l'horloge de 0,13 à 3,7 s (médiane 0,36 s) et `t0FilmMs` de -0,23 à +3,1 s (médiane -0,03 s) : les
   22 rejeux sont les seuls où l'horloge et les positions s'accordent contre le détecteur. Sur les 1 455 rejeux,
   l'horloge est à moins de 2 s du premier mouvement réel dans 1 447 cas (1 403 à moins d'une seconde).
2. **Première élimination côté API** (`highlight_events`, type `kill`, `q_b3b.sql`) : médiane de 16,1 s après le départ
   de l'horloge et de 32,5 s après `t0FilmMs` pour les 22 ; référence des matchs sains : 13,2 s après l'horloge et
   12,8 s après `t0FilmMs`. L'horloge redonne l'ordre de grandeur normal, le détecteur le double.
3. **Arrivées des joueurs côté API** (`first_joined_time`, présents au début, hors bots) : médiane 0,2 s après le départ
   de l'horloge et 16,6 s après `t0FilmMs` pour les 22 (matchs sains : 5 s avant les deux).
4. **Fin de partie** (4 matchs sur 22 terminés par le temps, `rcLastLeft` < 2 s) : durée du match moins (début + limite)
   vaut 5,5 s en médiane avec l'horloge et 24,5 s avec le premier mouvement. Échantillon trop petit pour valoir seul.

Mécanisme observé : dans ces 22 rejeux, le détecteur date le premier mouvement à l'image 6 à 12 (8 films, tous de Big Team Battle) ou
à l'image 163 à 168 (le reste), alors qu'aucune piste ne bouge avant l'image 496 à 676. Ce n'est pas une propriété du
mode, de la date ou du nombre d'arrivées tardives :

| Caractéristique | 22 rejeux | Autres |
|---|---|---|
| Playlist | 11 Quick Play, 11 Big Team Battle | 1 296 Quick Play, 108 BTB, 13 classés |
| Dates | 22 matchs, de 2024-11-03 à 2026-06-09 | 2023-04 à 2026-11 |
| Limite de l'horloge | 719,9 s (12 Quick Play) ou 900,1 s (BTB) ; un KOTH à 179,9 s avec 35 démarrages (manches) | tous modes |
| Schéma du rejeu | 125 pour tous | 125 |
| Joueurs arrivés après 5 s | 0 à 7 | présents dans les deux groupes |
| Origine du film sur l'axe du match | 9,7 à 98,3 s (médiane 24,8 s) | 0 à 213 s (médiane 14,9 s) |
| Image du premier mouvement détecté | 6 à 168 | médiane 686 |

Build : non disponible (voir section 0). Cause du détecteur : non trouvée dans ce dépôt (le détecteur est dans la
brique du décodeur) ; ce que l'on peut dire est que, dans ces 22 films, une activité précède le départ de l'horloge de
17 s sans que les pistes publiées bougent.

Conséquences sur le registre : pour ces 22 matchs, `film_movement` a écrit un coup d'envoi trop tôt de 17 s ; la durée
de jeu, la fenêtre de rejeu (début) et la fenêtre jouée du partage de la main (`leadshare/window.go:39`) en héritent.

### B.4 Temps de jeu des joueurs présents du début à la fin

Population : joueurs non bots, présents au début ET à la fin, ni arrivés ni partis en cours, temps de jeu non vide.
Référence : « durée du registre moins coup d'envoi du registre » (`duration_seconds` ; la fin déduite de `end_time`
diffère de `duration_seconds` de plus d'une seconde sur 437 matchs : non investigué). « Estimation de l'API » est le
`ComputeT0` rejoué en SQL sur les mêmes participations (`q_b4.sql`). Écart = `time_played_seconds - (durée - T0)`,
en secondes.

| Qualité | Coup d'envoi du registre | Matchs | Joueurs | Écart avec le T0 du registre : médiane / centiles 5 et 95 / dans les 2 s | Écart avec l'estimation de l'API : médiane / dans les 2 s |
|---|---|---|---|---|---|
| `ok` (jamais reporté) | = estimation | 583 | 10 553 | -1,0 / -1,0 et -0,1 / 10 553 | -1,0 / 10 553 |
| `film_movement` | = estimation (le film retombe sur la même valeur) | 96 | 757 | -0,6 / -1,6 et +0,4 / 757 | -0,5 / 757 |
| **`film_movement`** | **changé par le film (avant le report : estimation ; après : film)** | **1 345** | **10 895** | **+5,1 / -22,4 et +45,0 / 796** | **-0,7 / 10 871** |
| `film_movement` | estimation de l'API non calculable | 1 | 23 | +66,8 / - / 0 | - |
| `negative` | `real_start_time` décalé de 2 à 4 h | 11 | 161 | +14 399 / - / 0 (le temps de jeu est la valeur de l'API : `time_played = durée`) | - |
| vide | = estimation | 2 | 11 | -0,5 / - / 11 | -0,5 / 11 |

Lecture : avant le report, le temps de jeu suivait le coup d'envoi de l'estimation à 1 s près pour tous (10 553 sur
10 553). Après le report, le temps de jeu des 10 895 joueurs des 1 345 matchs dont le coup d'envoi a changé est resté
sur l'ancienne valeur (10 871 sur 10 895 à l'estimation près) alors que le coup d'envoi du registre a bougé de +5,1 s
(médiane ; -22,4 à +45 s aux centiles 5 et 95) : seuls 796 sur 10 895 coïncident avec la nouvelle fenêtre.
La date du report n'est pas lisible dans la base (`updated_at` n'est pas touché par `MarkT0Film` ni par
`backfill_t0_film`) ; le groupe « changé » est donc défini par l'écart entre le registre et l'estimation rejouée
(plus d'une seconde), qui recoupe le compte annoncé du report (1 447 des 1 455 rejeux).

Deux autres constats de cette mesure : la troncature à la seconde fait perdre 1 s en moyenne (écart -1,0 du groupe `ok`) ;
dans les 350 matchs `ok` au T0 < 5 s, le temps de jeu inclut donc tout le countdown.

## C. Ce que l'inventaire impose au modèle cible (sans recommandation de conception)

À retirer, fusionner ou garder, chaque ligne renvoie à l'inventaire.

1. Une seule valeur de coup d'envoi par match : retirer le choix entre estimation de l'API et film aux lecteurs
   (R19, A.5 « Web : fenêtre de lecture ») ; fixer pour la source film la datation qui prime, le verdict de B.3 étant
   que l'horloge de manche date mieux que `t0FilmMs` sur 22 rejeux (B.3, A.5 S10) et que `t0FilmMs` est refusé sur 8
   (B.2).
2. Statuer sur les sources : S1 (estimation, dont 349 des 590 matchs sans rejeu à T0 < 5 s, B.1), S2 et S4 (report du film),
   S3 (estimation sur tout le corpus, protégée par `film_movement`) ; S7 et S8 sont du code et une colonne sans lecteur.
3. Statuer sur les sept qualités : aucune n'est lue à l'exécution (A.2) ; seule `film_movement` sert de garde entre
   écrivains (S2, S3, S4). Un diagnostic journalisé les remplacerait sans lecteur à rebrancher.
4. Statuer sur les 11 matchs `negative` dont `real_start_time` est servi comme T0 de 2 à 4 h (B.1, A.5 « Fragment SQL
   unique ») : la valeur aberrante est lue partout sans filtre.
5. Un seul lecteur du coup d'envoi : `SQLT0Ms` et ses sept consommateurs de A.5, plus les copies de la durée
   (R3, R4, R5, R6, R7, R18, R22), plus les copies web (R19 à R21).
6. Un seul repli ou aucun : R1 à R22 ; en particulier le « 0 » de R1, R2, R3, R7, R13 (coup d'envoi absent), le « 600 s »
   de R11, le « sinon durée jouable » de R9, le « sinon durée » de R10.
7. La durée jouable de l'API : écrite (S9, `transforms.go:107`), servie comme `playable_duration_seconds` d'en-tête alors
   que ce champ web est la durée moins le coup d'envoi (A.4 « En-tête de match ») ; lecteur de la Dominance (R6) ; absente
   côté Halo 5 (A.7) ; égale à la durée sur 1 561 matchs (A.4).
8. Les quatre durées (registre, jouable, moins coup d'envoi, observée R15) et le nom trompeur de `duration_seconds` des
   lignes de séance (A.4).
9. Écriture du coup d'envoi hors `persist` : S3, S4 (UPDATE directs) et S5 (UPDATE de `time_played_seconds`) ; seule S2
   passe par `T0FilmPersister` (A.1).
10. Temps de jeu : le T0 utilisé à l'écriture (S1b, estimation) n'est pas celui du registre (S2) ; aucun recalcul quand le
    coup d'envoi change (B.4 : 10 895 joueurs sur 1 345 matchs).
11. Lecteurs du temps de jeu à ne pas casser par un changement de valeur : note de performance, LUSR, cadences,
    objectifs, filtres (A.6) ; le LUSR lit aussi le coup d'envoi (A.6 point 2).
12. Révisions de notation : un changement de définition du temps de jeu ou de la durée jouable de la note est une
    modification de logique non vue par le garde-rail d'empreinte (A.6 point 4) ; rattrapage de données et rejeu de
    note ne sont pas liés par ADR 0047.
13. Chemin du report : limité au placement local, 5 artefacts par cycle (A.1 S2) ; le chemin `worker` de la prod ne
    reporte rien (`.ai/REGISTRE_REPORTS.md:48`).
14. Rejeu web : `roundClock` publié sans lecteur (A.5) ; le décodeur est public, ce n'est pas une dette (règle 19).
15. Halo 5 : pas de coup d'envoi (A.7) ; tout modèle commun garde un cas « inconnu » pour ce titre ou publie un coup
    d'envoi de remplacement.
16. Garde-rails existants à adapter ou conserver : `archlint/no_raw_real_start_offset_test.go`,
    `no_raw_start_time_literal_test.go`, `TestScoringParametersMatchRevision`, compteurs `postsync_replay_t0_film_*`
    (`replayartifacts/journal.go:92-98`).

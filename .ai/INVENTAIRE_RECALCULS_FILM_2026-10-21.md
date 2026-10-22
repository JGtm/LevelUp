# Inventaire : lectures de requête qui recalculent depuis le journal du film (2026-10-21)

Pièce du dossier du plan `.ai/PLAN_PERF_PAGES_V801_2026-10-21.md` (volet 2). Inventaire statique
(agent Explore, Sonnet, lecture seule) ; association service → route vérifiée par la liste des
routes, pas handler par handler (non vérifié pour Escouade v2, Teammates, Trends, Match-history).
À RE-VÉRIFIER sur pièces avant chaque bascule.

Abréviations : `KE` = `match_kill_events_latest` ; `KP` = `kill_positions_latest`.
Classes : **A** agrégat décomposable par match (candidat résumé) ; **B** lignes brutes nécessaires
(instants, coordonnées, médianes exactes) ; **C** un seul match, déjà borné ; **D** déjà un résumé
stocké, lu tel quel.

## Constats qui cadrent le volet 2

1. `match_kill_events` n'est pas « du film pur » : chaque passe fusionne la base crédit (API) et
   l'enrichissement du film. Deux producteurs écrivent des passes : `KillSourcePersister.PersistPass`
   (`internal/persist/kill_events_persister.go:188`, film) et `persistCreditKillEvents`
   (`internal/persist/kill_events_credit.go:274`, crédit, à chaque sync). La vue retient la dernière
   passe par `written_at`. Un résumé doit être rattaché à la passe source (`decode_pass`).
2. Le coût est la fenêtre `QUALIFY ... PARTITION BY match_id` de `KE` sur ~3,95 M lignes (chiffre
   cité dans `tactical_repo_univers.go`) ; tout le code ADR 0036 I2 existe pour la contourner.
3. Plusieurs vues du film sont DÉJÀ des résumés stockés (classe D) : `match_life_placement`,
   `match_vehicle_takes`, `match_bomb_stats`, `match_flag_grabs_net`, `match_pad_pickups_by_tier`,
   `match_usage_*`, `match_player_positions`, `kill_positions`, `match_death_context`,
   `match_death_mates`, `kill_openings`.
4. `killer_victim_pairs` ne peut PAS servir de résumé de paires : lignes par frag (46,5 % de
   doublons mesurés le 2026-08-02), écrite par DELETE puis INSERT, sans `publishable` ni arme ni
   assistant, alimentée par l'API (pas le film), en retrait programmé ; lue sur le chemin de
   requête seulement pour les noms (`v_gamertag_lookup`).

## Doublons (le cœur du volet 2)

| Comptage | Lecteurs qui le refont | Nombre |
|---|---|---|
| Frags entre deux joueurs (par match, tueur, victime) | Q23b `GetMatchEncounterStats` (vue match, historique) ; Q26 `GetTopEncountersGlobal` (Carrière, historique) ; Q27 `GetRivals` (Carrière, historique) ; Q28 `GetRelations` (Relations) ; Q30 `GetRivalTimeline` (Relations moments, Explorer) ; `GetEncounterStats` / `QKillsBetweenPlayers` (Comparaison, non borné) ; `GetKillerVictimBetween` (Explorer) | 7 |
| Assistances par paire (par match, assistant, tueur, tranches de part) | Q21d `GetMatchAssistPairs` (vue match) ; Q28c `GetRelationAssists` / `GetMatchEncounterAssists` (Relations, Explorer, vue match, historique) ; Q26l `LoadMatchAssistedFrags` (Accueil) ; `QCoordinationAppuis` `LoadAppuis` (Coordination sessions/timeseries) ; Q32d `LoadSquadAssistPairs` (Escouade) | 5 |
| Frags par arme (par match, tueur, `source_tag`) | `bulkWeaponKillsFromSource` (vue match) ; `GetTopWeaponsForMatches` (Explorer) ; `LoadFavoriteWeapon` (Accueil, historique) ; `LoadWeaponKillsAggregated` (Comparaison, Explorer, historique des matchs, sessions, synthèse, timeseries, Escouade v2) ; `LoadKillSourceCategoryKills` (vue match, sessions, Escouade v2) ; `qSquadVehicleFrags` (Emprise) ; `/monitoring/weapon-coverage` | 7 |
| « Le match a un journal publiable / une assistance lue » | `QTacticalUnivers.mesure` ; `qSoloJournalNonPubliable` ; portée de Q21d ; `matches_measured` de Q32d ; `porteuses` de Q28c et Q26l ; dénominateur de `match_range` ; `EventsRead` de l'Emprise | 8 |

## Classe B (restent à la demande)

- Instants et lignes individuelles : `GetMatchKVPairs` (Q20), `GetMatchKillSources` (Q21b),
  `GetMatchKillAssists` (Q21c), `LoadSquadKillLog` (Q32e, badge « Voleur »), replis
  `loadSyntheticKillEvents` / Q32c (seulement si `highlight_events` est vide).
- Cartes de chaleur et rejeu : `player_positions_repo.LoadMatch`, Tactique `KillPositions`,
  `MortsAvecContexte`, `ContextesDeMort`, `MatesDesMorts`, `JournalDesKills`, `MortsParCarte`.
- Distances (médianes et records exacts) : `LoadWeaponRange` / `LoadWeaponOpening`,
  `LoadMatchRangeKills` (sauf son dénominateur, couvert par la tête), `kill_distance_repo.LoadMatch`.

Candidats de second rang (grain par frag ou par vie, gain = suppression de la fenêtre seulement) :
`match_measured_kills` (pré-jointure KE × KP + contexte de mort : sert distances et Tactique) et
`match_life_windows` (vies près d'un coéquipier, `solo_lives_repo`). Doctrine en vigueur : on stocke
des coordonnées, jamais une distance (`kill_measured.go`).

## Le modèle : le résumé d'usage

- Schéma : `internal/migration/steps_shared_usage_summary.go` (`match_usage_players` l.132-177,
  tête `match_usage_films` l.180-217, vues `_latest` l.229-244 ; la passe courante se décide sur la
  tête ; piège de l'étoile figée : recréer la vue après tout `ADD COLUMN`).
- Projection pure `replay.BuildUsageSummary` ; persister `internal/persist/usage_summary_persister.go:59`
  (une transaction, INSERT purs, une passe par appel, ligne de tête toujours écrite).
- Déclencheur `Deriver` (`internal/sync/replayartifacts/derivations.go:119`) ; rattrapage par fichier
  de marque `DerivationsRev` (plafond 5 artefacts par cycle). `summary_rev` seul NE relance PAS le
  recalcul dans la boucle de sync (seulement le CLI `backfill-usage-summary`) : pour les nouvelles
  familles, un anti-join en base (motif `conditionBacklog`, `internal/sync/killcollector/postsync.go:424`)
  est préférable à un fichier de marque.

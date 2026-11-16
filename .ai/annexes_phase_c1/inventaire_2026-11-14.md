# Phase C, premier temps : état des lieux sur pièces (2026-11-14)

Relevé en lecture seule par un agent (rien exécuté, aucune base ouverte), sur le clone du décodeur
`halo-infinite-film-decoder-wt-vehicules` (main fd7303c, v1.3.8 + retrait des tests de recherche clos),
LevelUp (`release/v8.2.0`) et la copie locale de libfilmshell. Chemins relatifs au dépôt concerné.
Note : une seule copie de libfilmshell a été trouvée (`libfilmshell/src/`), alors que la mémoire en cite
deux (dossier imbriqué plus récent) : à vérifier avant toute comparaison (C0.2).

## 1. Présence des joueurs et propriétaire des vies
- Lu : table de 32 joueurs du chunk 0 (`internal/grammar/player_table.go:198`) ; type 8 décodé
  (`roster_type8.go:127`, `DecodeRoster`) mais sans appelant hors tests ; type 12 (`bot_metadata.go:128`,
  `killsource/botmeta.go:84`) ; entités `ti=9` au pas des images-clés (`player_entities.go`,
  `player_teams.go:142`) ; motif de xuid (`killsource/index_motif.go:75`, `grammar/player_index.go:56`) ;
  `unit-control` lu (`unit_control.go:20`, contrôleur sur 5 bits) mais observateur nul en production
  (`etats_mouvement_hooks.go:39,86`).
- Publié : `roster[]` (`replay/roster_entry.go:14`), `roster[].presence[]` (`roster_entry.go:97`),
  `tracks[].xuid` / `tracks[].bot` (`document_tracks.go:116-158`).
- LevelUp : `match_participants.{joined_in_progress,left_in_progress,first_joined_time,last_leave_time}`,
  `match_lives` ; écran : `presenceFeed.ts` (l'API prime sur le film).
- libfilmshell : `roster.rs` (chunk 0 + type 8 + type 12, premier et dernier listage), `holders.rs:31`
  (porteur d'un identifiant moteur à une image), `owners.rs` + `components.rs:2381` (contrôleur du bipède
  créé = propriétaire de la vie, effacé à la mort).
- Manque : propriétaire des vies par `unit-control`, arrivées et départs datés par le type 8.

## 2. Tableau des scores (statborg)
- Lu : `signaux/statborg.go` (jusqu'à 58 composants, `StatRecord.Comps`, `types/objectives.go:98`).
- Publié : composants 0 A, 1 B, 2 A/B, 3 A, 21 B, 23 A (`objectives/score.go:76-115`,
  `slotidentity.go:52-54`, `replay/hill_hold_ticks.go:57`) ; `scoreTimeline.teams[]/players[]`.
- Limite à 8 joueurs : `StatborgSlotMax=24`, `StatborgSlotEquipeMax=8` (`signaux/statborg.go:86-87`),
  `StatPlayerSlots=8` (`objectives/rosterfit.go:48,75`).
- libfilmshell : `gamestate.rs` (28 lignes x 2 valeurs par entité), `statlog.rs:19-125` (changements par
  image), index de joueur sans plafond, `model/scoreboard.rs:72-90` (14 emplacements nommés).

## 3. Tirs et touches
- Lu : `ScanFilmWeaponShots` (`grammar/weapon_hits.go:194`), `ScanFilmWeaponDamages`, `PairWeaponHits`
  (`weapon_hits.go:109`, fenêtre 1 s), exposé par la façade seulement (`decfilm.go:173`).
- Publié : `shots[]`, `bursts[]` ; aucune touche.
- LevelUp : `match_weapon_shots` et `match_weapon_hit_distance` sans lecteur applicatif ; précision du
  film retirée de l'exposition, jugée non fiable (`halo_infinite/adapter_data.go:190-200`).
- libfilmshell : `fire.rs`, `accuracy.rs` (une touche = un `damage_aftermath` à dégât d'arme par
  tireur, victime et image ; projectiles du bloc 41), `weapon_stats.rs:19-27` (arme par le tag de dégât,
  sinon dernier tir sous 2 s).

## 4. Dégâts en points
- Lu : parts du tueur et de l'assistant au message de mort (`killsource/assist.go:55-110`,
  `kill.go:56-67`) ; magnitude `MagClear` / `MagRaw` (`WeaponDamage`) ; composant 10 A du statborg lu,
  jamais nommé (c'est l'emplacement « 20 » de libfilmshell, dégâts subis).
- Publié : rien au document.
- LevelUp : `match_kill_events.killer_damage_pct` / `assist_damage_pct` ; vérité API
  `match_participants.damage_dealt` / `damage_taken`.
- libfilmshell : `Hit.shield` / `Hit.body` en points (`hit_damage.rs`, table `game/damage_data.rs`,
  478 lignes), `Stat::DamageTaken`.

## 5. Médailles et score par action
- Lu : `grammar.ParseHighlightEvents` (`decfilm.go:176`), noms `medalname/table.go`.
- Publié : faits seulement (`[]HighlightEvent`), rien au document.
- LevelUp : `highlight_events` (écrite par `sync/collect.go`), écran `killFeedLogic.ts:99-117`,
  `useReplayTimeline.ts:314-341` ; `medals_earned` ; `personal_score_awards` (API, joueur synchronisé
  seul, plus pénalités synthétiques de `sync/enrichment_discipline.go`).
- Décodeur : `objectives/awards.go:104` (`LabelPersonalScore`) sans appelant de production.
- libfilmshell : `highlights.rs`, `assemble.rs:748-783` (calage de l'horloge des temps forts),
  `game/medals.rs:58` (`personal_score` par médaille).

## 6. Horloge, réapparition, tirs alliés
- Lu : `player-respawn-timer` et `game-engine-round-timer` capturés (`vitality.go:144-175`,
  `capture.go:49,60`) sans appelant ; délai minimal de la variante (`profile/infos_de_match.go:64`) ;
  `player-last-betrayer` (hook seul) ; compte interne `Trahisons` (`placement_des_vies.go:162,381`).
- Publié : `originMs`, `t0FilmMs`.
- LevelUp : trahisons déduites dans `sync/enrichment_discipline.go` (même équipe).
- libfilmshell : `statlog.rs` (compte à rebours, horloge de manche), pénalité de trahison et drapeau
  `friendlyFire` de la variante.

## 7. Propriétés gérées (D11)
- Lu : `managedPropertyWalk` (`zone_state_scan.go:203,239`), `matchWorldObjectRecord`
  (`projectiles.go:402`).
- Publié : `zoneStates[]` (`spans`, `gauge`, `gaugeRamps`), `FlagSpan.returnProgress`, `flagReturnZone`
  (minuterie du manifeste du titre).
- libfilmshell : ne lit `ti=13` que pour sa largeur ; zones par les événements de script type 82
  (`zones.rs`) ; temps de retour du drapeau dans les options de variante (`CTFOptions_FlagReturnTime`).

## Vérités disponibles
- `match_participants` (API) : tirs, touches, précision, dégâts infligés et subis, score personnel,
  frags, morts, assistances, arrivées et départs en cours de match.
- `match_objective_stats` (API) : captures et sécurisations de zone, tics de score de zone, temps en
  zone, retours de drapeau, temps de porteur.
- `personal_score_awards_latest` (API, joueur synchronisé seul) ; `medals_earned`.
- Variante du film (`InfosDeMatch.Options`) : temps de retour du drapeau (hachage `0x2b16b76f`, présence
  dans un film réel non vérifiée), délai de réapparition.
- libfilmshell : comparaison croisée, pas une vérité.

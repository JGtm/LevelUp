# Appelants de lecture de film dans LevelUp : lesquels n'ont pas besoin de tout le film

Inventaire en lecture seule, fait le 2026-11-23.

- Application : `Scripts/LevelUp-wt-dec160/apps/go-api` (worktree sur `release/v8.3.0`).
- Décodeur : `Scripts/dec-v160`, arbre propre au tag `v1.5.0`.
- Version pinée par LevelUp : `github.com/JGtm/halo-infinite-film-decoder v1.5.0` (`go.mod`, ligne 14). Pas de `replace` : le code lu dans `dec-v160` est donc celui que LevelUp compile.
- Périmètre : fichiers `.go` hors `_test.go`, sous `internal/` et `cmd/`.

## Résumé

- **21 sites de production** lisent un film ou un de ses morceaux (tableau A).
  - **5** en ont besoin de tout (chargement complet ou décodage complet).
  - **16** n'en lisent qu'une partie.
- **Génération complète du rejeu** : 5 appelants de plus passent par la même entrée (`BuildFromFilmAvecFaits`, tableau B).
- **8 fichiers d'outils hors production** (tableau C). Plusieurs ne lisent qu'une partie, mais ce ne sont pas des chemins de l'application.
- **Dépréciation** : `DetectT0Film` n'a aucun appelant dans LevelUp, ni en production ni dans les tests. Aucune entrée dépréciée n'est utilisée par l'application.

Les sites qui ne lisent qu'une partie, avec ce qu'ils lisent réellement :

| Site | Ce qu'il lit du film |
|---|---|
| A2 `lireMorts` (replaybuild) | Le fil des morts seul (morceau des temps forts). |
| A4 `botsSansKillFeed` (replaybuild) | Les métadonnées de bots (BOT_METADATA), et seulement quand le film n'a pas de kill-feed. |
| A5 `statborgDuFilm` (replaybuild) | Les morceaux d'enregistrements d'entité (statborg) et les replis constatés. |
| A6 `statborgDuFilm` (replaybuild) | Les morceaux de type « jeu » : instants des rafales de capture (drapeau CTF). |
| A8 `ecarterSiCleInconnue` (replaybuild) | Le registre `chunk_00` seul (clé du film). |
| A10 `decodeFilmForMatch` (killcollector) | Le registre `chunk_00` seul (clé du film). |
| A12 `BuildWeaponShotsBatch` (killcollector) | Les morceaux REPLICATION_DATA seuls, en octets bruts : indice de réplication et identifiant d'arme des tirs. |
| A13 `resolvePlayerIndices` (killcollector) | Les morceaux REPLICATION_DATA seuls, en octets bruts : recherche du motif de chaque xuid. |
| A14 `lireLePontDuCollecteur` (killcollector) | Positions des bipèdes, téléportations, créations de bipède, fil des morts, index des joueurs, horloge d'origine. Pas le statborg, pas le kill-feed. |
| A15 `lireLeBordDuCollecteur` (killcollector) | Nuage des véhicules et créations de véhicules (via le contexte de film), plus l'occupation lue dans le résultat killsource. |
| A16 `filmMajorVersionDuCache` (sync/filmfetch) | L'en-tête de version dans `chunk_00`. |
| A17 à A19 (sync, highlights) | Le morceau des temps forts seul (événements de médaille et de mort). |
| A20 `eventsDuFilm` (ops, rattrapage médailles) | Le morceau des temps forts seul. |
| A21 `versionDuFilm` (commande de rattrapage médailles) | `chunk_00` seul (version). |

## A. Production (chemins de l'application et commandes de rattrapage)

Colonne « Classe » : « tout » = a besoin de tout le film ; « partie » = n'a besoin que d'une partie (laquelle est dans « Ce qu'il lit »).

| # | Fichier:ligne | Fonction appelante | Entrée du décodeur | À quoi ça sert | Ce qu'il lit réellement du résultat | Classe |
|---|---|---|---|---|---|---|
| A1 | `internal/replaybuild/filmload.go:71` | `chargerFilm` (appelée par `chargerLeFilmDeLaCuisson`, `filmfacts_cuisson.go:142`) | `decfilm.LoadDir` | Charge et découpe tous les morceaux du film, une fois, pour toute la cuisson. Conteneur : aucun résultat propre. | Le film entier, chargé en mémoire. | tout (conteneur) |
| A2 | `internal/replaybuild/filmload.go:108` (dans `lireMorts`, appelée à `filmfacts_cuisson.go:117`) | `lireMorts` | `decfilm.ScanDeaths` | Fil des morts pour les actions d'objectif et les frags sous effet actif. | La liste des morts et l'erreur de lecture. | partie : fil des morts (morceau des temps forts) |
| A3 | `internal/replaybuild/kills.go:57` (dans `decodeKillSource`, appelée par `entreesDeLaCuisson`, `filmfacts_cuisson.go:100-130`) | `decodeKillSource` | `decfilm.Decode` | Décodage de la source de dégât : morts neutres, références de frag, bots, diagnostics. Un seul décodage par match. | `res.Kills` (morts neutres, frags sous effet), `res.Roster` (bots), diagnostics, contexte de film. | tout (décodage du kill-feed) |
| A4 | `internal/replaybuild/kills.go:78` (dans `botsSansKillFeed`, kills.go:74) | `botsSansKillFeed` | `decfilm.ResultatDeBotsSansKillFeed` | Repli quand `Decode` refuse le film faute de kill-feed : garder le roster de bots. | Roster de bots (`res.Roster.Bots`, `UnpinnedBots`). | partie : métadonnées de bots (BOT_METADATA) |
| A5 | `internal/replaybuild/matchfacts.go:107` (dans `statborgDuFilm`) | `statborgDuFilm` | `decfilm.StatRecordsAvecReplis` | Courbes de score, objectifs, drapeaux : enregistrements d'entité du film. | `recs` (enregistrements), `truncated`, `replis`, diagnostics. | partie : morceaux d'enregistrements d'entité (statborg) |
| A6 | `internal/replaybuild/matchfacts.go:110` (dans `statborgDuFilm`) | `statborgDuFilm` | `decfilm.CaptureBurstTimes` | Portage du drapeau en CTF : instants des rafales de capture. | Liste d'instants en ms, rangée dans `BurstMS`. | partie : morceaux de type « jeu » |
| A7 | `internal/replaybuild/filmfacts_cuisson.go:204` (dans `documentDeLaCuisson`, appelée par `BuildBytes`, `replaybuild.go:269`) | `documentDeLaCuisson` | `replay.BuildFromFilmAvecFaits` (paquet `replay` du décodeur) | Génération complète du document de rejeu (une trentaine de balayages, cf. `replay/observe.go`). | Le document entier, sérialisé dans l'artefact. | tout (génération complète) |
| A8 | `internal/replaybuild/refus_de_cuisson.go:72` (dans `ecarterSiCleInconnue`, appelée à `filmfacts_cuisson.go:155`) | `ecarterSiCleInconnue` | `replay.CleDuFilm` | Porte de la clé du film : écarter avant toute lecture si la clé n'est pas dans la table de profils. | La clé (`Ecrite`, `Format`, `Build`) et `Refusee()`. | partie : registre `chunk_00` |
| A9 | `internal/sync/killcollector/bridge.go:131` (dans `FilmOf`, appelée à `collector_run.go:135` et `bridge.go:72`) | `FilmOf` | `decfilm.Load` | Conteneur de la passe de collecte des morts : monte le film en mémoire à partir des morceaux téléchargés. | Le film entier, chargé en mémoire. | tout (conteneur) |
| A10 | `internal/sync/killcollector/collector_run.go:144` (dans `decodeFilmForMatch`) | `decodeFilmForMatch` | `replay.CleDuFilm` | Même porte de clé que A8, côté collecteur. Un film écarté n'est ni décodé ni publié. | La clé et `Refusee()`. | partie : registre `chunk_00` |
| A11 | `internal/sync/killcollector/collector_run.go:215` (dans `decoderSousLaCarte`, appelée via `decoderLeFilm`, `collector_run.go:184`) | `decoderSousLaCarte` | `decfilm.Decode` | Passe des morts : écrit `match_kill_events` et alimente les positions et le bord. | `res.Kills` et `LineByLinePublishable()` (`collector_batch.go:39-47`), `res.Occupation` (`positions_bord.go:30`), roster de bots, diagnostics. | tout (décodage du kill-feed) |
| A12 | `internal/sync/killcollector/shots.go:88` (dans `BuildWeaponShotsBatch`, appelée à `collector_run.go:304`) | `BuildWeaponShotsBatch` | `decfilm.ScanFireEventsB5` | Ventilation des tirs par joueur et par arme, écrite dans `match_weapon_shots`. | Sur chaque morceau REPLICATION_DATA seulement (`ReplicationChunks`, `bridge.go:141`) : `FilmIndex5` et `WeaponBytes` de chaque événement de tir. Le temps n'est pas utilisé (estimateur à zéro). | partie : morceaux REPLICATION_DATA seuls |
| A13 | `internal/sync/killcollector/shots.go:139` (dans `resolvePlayerIndices`, shots.go:129) | `resolvePlayerIndices` | `chercherMotifs` (lecture des octets bruts, comme `decfilm` pour le motif xuid) | Associer chaque xuid de l'API à son indice de réplication, pour compter les tirs. | Recherche d'un motif xuid dans les morceaux REPLICATION_DATA seuls. | partie : morceaux REPLICATION_DATA seuls |
| A14 | `internal/sync/killcollector/positions.go:338` (dans `lireLePontDuCollecteur`, appelée à `positions.go:283`) | `lireLePontDuCollecteur` | `decfilm.ScanPontDIdentite` | Positions des joueurs pour l'isolement des vies et le contexte de mort. | `Positions` (bipèdes), `Translocations`, `Creations`, `Morts` (fil des morts), `Index` (index des joueurs), `OrigineUS` (horloge). | partie : positions, téléportations, créations de bipède, fil des morts, index, horloge (pas le statborg, pas le kill-feed) |
| A15 | `internal/sync/killcollector/positions_bord.go:26` (dans `lireLeBordDuCollecteur`, positions_bord.go:19) | `lireLeBordDuCollecteur` | `replay.LireLesVehicules` | Placement des occupants dans les véhicules et les postes fixes. | Nuage et créations de véhicules ; `res.Occupation` (du résultat de A11). | partie : nuage de véhicules et créations |
| A16 | `internal/sync/filmfetch/manifeste.go:81` (dans `filmMajorVersionDuCache`) | `filmMajorVersionDuCache` | `decfilm.FilmMajorVersionFromHeader` (sur `decfilm.Inflate` du registre) | Version du film, qui commande le découpage du gamertag dans les événements. | Un entier (version majeure), lu dans l'en-tête de `chunk_00`. | partie : `chunk_00` (en-tête) |
| A17 | `internal/sync/collect.go:115` | `collect` (sync) | `haloinfinite.ParseHighlightEvents` (wrapper de `decfilm.ParseHighlightEvents`, `internal/games/halo_infinite/film_highlight_events.go:23`) | Événements de médaille et de mort, insérés dans la base pendant la collecte. | Liste d'événements (`events`) et erreur de parse. | partie : morceau des temps forts seul |
| A18 | `internal/sync/engine_highlight_events.go:185` | `collectHighlightEvents` (sync, post-synchro) | `haloinfinite.ParseHighlightEvents` | Même usage que A17 pour la passe post-synchronisation. | `out.events`, nombre d'événements (anomalies comptées). | partie : morceau des temps forts seul |
| A19 | `internal/sync/convergence_backfill_events.go:272` | `processChunk` (convergence des événements) | `haloinfinite.ParseHighlightEvents` | Rattrapage des événements par lot, hors verrou. | `f.events`, erreur de parse. | partie : morceau des temps forts seul |
| A20 | `internal/ops/medal_feed_backfill.go:230` (dans `eventsDuFilm`, appelée par `traiterMatch`) | `eventsDuFilm` | `haloinfinite.ParseHighlightEvents` | Rattrapage des identités des médailles dans la base. | Événements, pour l'appariement sur (xuid, instant). | partie : morceau des temps forts seul |
| A21 | `cmd/levelup/cmd_backfill_medailles_feed.go:184` (dans `versionDuFilm`) | `versionDuFilm` | `decfilm.HighlightProfileFromHeader` (sur `decfilm.Inflate` du registre) | Version du film pour la commande de rattrapage des médailles. | `MajorVersion` seul. | partie : `chunk_00` (en-tête) |

Note sur les contextes sans lecture : `decfilm.NewFilmContextForMap` (kills.go:55, positions.go:335) et `horlogeDesChunks` (filmfacts_cuisson.go:377) ne lisent pas les octets du film. Le premier construit le contexte, le second lit les métadonnées du manifeste.

## B. Appelants de la génération complète (même entrée que A7)

Ces sites ne lisent pas le film directement : ils appellent la cuisson, qui lit tout.

| Fichier:ligne | Fonction | Usage |
|---|---|---|
| `internal/replaychild/replaychild.go:136` | `builder.BuildBytes` | Enfant de cuisson lancé par la synchronisation. |
| `cmd/replay-worker/job.go:313` | `builder.BuildBytes` | Cuisson en parallèle (worker). |
| `cmd/levelup/cmd_backfill_replay_child.go:109` | `builder.BuildMatch` | Rattrapage de cuisson, un match. |
| `cmd/replay-build/main.go:119` | `builder.BuildMatch` | Outil de cuisson (classé en C). |
| `cmd/replay-equiv/child.go:177` | `b.BuildBytes` | Outil de vérification d'équivalence (classé en C). |

Pour la synchronisation (`internal/sync/replayartifacts/buildone.go`), je suppose que la cuisson passe par `replaychild` (sous-processus). Je ne l'ai pas ouvert.

## C. Outils hors production (`cmd/`)

| Fichier:ligne | Fonction | Entrée | Ce qu'il lit | Classe |
|---|---|---|---|---|
| `cmd/killsource/main.go:224, 232` | `decoder` | `LoadDir`, `Decode` | Décodage seul, sous une carte donnée. Sortie : morts et bijection. | outil : tout |
| `cmd/zone-attribution/measure.go:91, 103, 242, 243, 276` | mesure | `filmcache.LoadFilm`, `BuildFromFilm`, `NamedEvents`, `SlotIdentityResolved`, `ScanDeaths` | Document complet, événements de zone, identité des slots, morts. | outil : tout, puis partie (zones, morts) |
| `cmd/oddball-terrain/decode.go:62, 69, 71, 128` | enfant de décodage | `LoadFilm`, `StatRecordsBornes`, `ScanDeaths`, `SeriesByRound` | Statborg et morts seulement. | outil : partie (statborg + morts) |
| `cmd/statnames-sweep/sweep.go:97, 104, 106, 128` | balayage des noms de statistiques | `LoadFilm`, `StatRecordsBornes`, `ScanDeaths`, `SeriesTotal` | Statborg et morts seulement. | outil : partie (statborg + morts) |
| `cmd/diag_film/main.go:103, 112, 116, 120-121` | diagnostic du scanner d'armes | `FindFramePositions`, `ScanFormulaA`, `ScanFormulaANS`, `TimestampEstimator`, `ScanFireEventsB5` | Octets bruts de chaque morceau, sans décodage des morts ni du statborg. | outil : partie (morceaux bruts) |
| `cmd/diag_weapons_v3/process.go:26, 38` | diagnostic des armes | `filmcache.LoadFilm`, `Extract` | Événements d'objectif et contrôle d'équipe (le film est chargé entier). | outil : tout chargé, lit objectifs et équipes |
| `cmd/diag_weapons_v3/positions.go:53` | diagnostic des positions | `DecodeKeyframePositions` | Positions seules. | outil : partie (positions) |
| `cmd/replay-equiv/child.go:177` et `cmd/replay-build/main.go:119` | cuisson de contrôle | `BuildBytes`, `BuildMatch` | Document complet (cf. B). | outil : tout |

## D. Entrées du décodeur sans appelant de production, et dépréciations

- **`DetectT0Film`** (et ses types `T0FilmTrack`, `T0FilmPoint`) : aucun appelant dans LevelUp, ni en production ni dans les tests. Le seul fichier qui le nomme est `internal/archlint/film_facade_surface_test.go`, dans son commentaire d'historique (retrait de la façade). Le décodeur le marque `Deprecated` (`replay/t0_film.go:330`). Votre hypothèse « encore utilisé » ne se vérifie pas : il n'est plus appelé nulle part.
- **`NomPrecisionParArmePasseSautee`**, **`NomDistancesDeToucheDesactivees`** (`decfilm/replis_obsoletes.go`, marquées `Deprecated`) : aucun appelant. Incohérence à trancher : `film_facade_surface_test.go:166` dit qu'elles ont quitté la façade en v1.5.0, mais le fichier est présent au tag `v1.5.0` de `dec-v160`.
- **Sans appelant de production** (seulement un commentaire, un outil, ou rien) : `ScanBipedPositions` appelé seul (il n'est utilisé que par `ScanPontDIdentite`), `DetectI0LayoutOf`, `ReadPlayerTable`, `ScanFilmDeaths`, `ScanFilmWeaponShots`, `ScanFilmWeaponDamages`, `FilmWeaponHitDistance`, `ScanBipedCreations`, `ReadFilmChunk`, `FilmChunkAt`, `FilmChunkNumbers`, `NewFilmContext` (sans `ForMap`), `NamedEvents`, `Extract`, `DecodeKeyframePositions` (outil seulement), `ScanFormulaA`/`ANS` (outil seulement).
- **`filmcache.LoadFilm`** : aucun appel de production (seulement des outils de `cmd/`). La production ouvre le manifeste et les morceaux par `filmcache.Open` / `OpenChunkDir` (`replaybuild/filmload.go:49`), puis `decfilm.LoadDir`.
- **Commentaire obsolète** : l'en-tête de `cmd/killsource/main.go` (lignes 3 à 8) affirme qu'aucun chemin de l'application n'utilise le décodeur killsource. C'est faux aujourd'hui (A3 et A11). À corriger, hors de cet inventaire.

## E. Ce que j'ai vérifié en ouvrant le code, et ce que je suppose

Vérifié (lecture des fichiers, pas seulement des recherches) :
- Liste des entrées exportées de `decfilm/decfilm.go` et de leurs fonctions internes (`internal/grammar`, `internal/source`, `replay`).
- Tous les sites de production de `decfilm.*`, `replay.BuildFromFilm*`, `replay.CleDuFilm`, `replay.LireLesVehicules`, `filmcache.*` (LoadFilm, Open, OpenChunkDir), et les parseurs de highlights, par recherche dans `internal/` et `cmd/` hors tests.
- Corps des lectures partielles : `ScanDeaths` (`internal/grammar/deaths_source.go`, lit le morceau des temps forts), `CaptureBurstTimes` (morceaux de type « jeu »), `ScanPontDIdentite` (`internal/grammar/pont_identite.go`, lit téléportations, créations, positions, fil des morts, index, horloge), `ReplicationChunks` et `resolvePlayerIndices` (octets bruts des morceaux REPLICATION_DATA).
- Le flux de cuisson `BuildBytes` → `entreesDeLaCuisson` → `documentDeLaCuisson` : le film n'est chargé que si les faits frais ne suffisent pas (`replaybuild/filmfacts_cuisson.go:100-160`).
- La passe de collecte `collect` (`collector_run.go:246-280`) : un seul chargement (`FilmOf`), puis morts, tirs et positions.
- Le dépôt du décodeur est propre au tag `v1.5.0` (`git status` vide).

Supposé ou non vérifié :
- Que `decfilm.Decode` (A3, A11) a besoin de tous les morceaux. Je n'ai pas tracé quels morceaux il lit. Si l'option « canaux au choix » doit réduire ce décodage, il faut le vérifier dans `internal/facts/killsource`.
- Que `BuildFromFilmAvecFaits` (A7) consomme tous les canaux. Je me suis fié à sa place (le document complet) et au nombre de balayages, pas à une trace par balayage.
- Que `CaptureBurstTimes` ne lit que les morceaux de type « jeu » : c'est ce que montre la boucle lue (`manifestChunks` filtré sur `chunkTypeJeu`), mais je n'ai pas lu `manifestChunks`.
- Que `horlogeDesChunks` et `chunksDuManifeste` ne lisent que le manifeste (aucun octet) : lu, mais pas tracé jusqu'au bout.
- Que `ecarterSiCleInconnue` (A8) et `decodeFilmForMatch` (A10) ne lisent que `chunk_00` : je l'ai vu pour `CleDuFilm` (`FilmRegistryChunk`), pas pour les autres appels.
- Que `buildone.go` passe par le sous-processus `replaychild` (non ouvert).
- Que les cinq appelants de la section B sont les seuls qui déclenchent la cuisson complète en production : je n'ai pas tracé les appels indirects via `cmd/levelup` (commandes de rattrapage) au-delà de ceux listés.
- La version pinée (`v1.5.0`) et le code lu sont identiques. Vérifié pour le tag, mais le `go.sum` et le cache de modules n'ont pas été contrôlés.

Aucun commit, aucun push, aucun fichier modifié dans LevelUp ni dans `dec-v160`. Ce fichier est le seul créé.

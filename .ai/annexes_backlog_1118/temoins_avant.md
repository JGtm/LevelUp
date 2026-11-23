# Témoins AVANT de la branche `chore/830-backlog-1118` (2026-11-18)

Étape 0.2 du plan `.ai/PLAN_BACKLOG_2026-11-18.md`. Aucun code de production n'a été modifié pour les obtenir : les lancements
sont ceux de la tête de la branche, les mesures du match `38900a5e` viennent de tests vitest jetables (supprimés, jamais versionnés ;
le script est gardé en `.txt` sous `sondes/`). Chaque chiffre vient d'une commande lancée ce jour-là.

## 1. Tests vitest ciblés (`--maxWorkers=2`)

Commande : `cd apps/web && npx vitest run <dossier> --maxWorkers=2`. « MR » du plan = `src/features/match-replay`.

| Dossier | Fichiers de test | Tests | Réussis | Échoués | Sautés | Durée |
|---|---|---|---|---|---|---|
| `src/features/match-replay` (MR) | 390 (387 réussis, 3 sautés) | 5 521 | 5 517 | 0 | 4 | 443,7 s |
| `src/components` | 141 | 1 208 | 1 208 | 0 | 0 | 144,2 s |
| `src/features/squad` | 111 | 999 | 999 | 0 | 0 | 102,1 s |

À rejouer au gate de chaque piste : les trois nombres de tests ne doivent que croître (aucun test retiré sans raison écrite) sauf retrait
annoncé (D.4 retire des tests de `ReplayCountersBadge`).

## 2. Stats par minute de l'Escouade (Go)

Commande : `cd apps/go-api && GOWORK=off GOCACHE="$LOCALAPPDATA/go-build-bl1118-0" go test ./internal/service/teammates/... -count=1 -run 'PerMinute|ParMinute|TemoinMinute|TemoinCadence' -v`
(les noms réels des tests des fichiers `per_minute_caracterisation_test.go` et `per_minute_temoins_test.go` ne contiennent pas `per_minute`
mais `TemoinCadence` et `TemoinMinute` ; le motif attrape aussi les tests voisins du paquet). Résultat : `ok levelup/go-api/internal/service/teammates`.

| Test | Fichier | Verdict |
|---|---|---|
| `TestTemoinCadence_StatsParMinuteDeLEscouade` | `per_minute_caracterisation_test.go` | PASS |
| `TestStatsParMinuteEscouade_MatchSansDureeHorsDuTauxEtDuCompte` | `per_minute_sans_duree_test.go` | PASS |
| `TestStatsParMinuteEscouade_CompteInconnuDUnCoequipierHorsDuTaux` | `per_minute_sans_duree_test.go` | PASS |
| `TestTemoinMinute_SynergyScorePerMinute` | `per_minute_temoins_test.go` | PASS |
| `TestPerMinuteEntry_RoundingAndMatchCount` | `teammates_squad_charts_test.go` | PASS |
| `TestBuildSquadPerMinuteStats_DistinctValuesPerTeammate` | `teammates_squad_charts_test.go` | PASS |
| `TestBuildSquadPerMinuteStats_NoSquadLoader_TeammatesEmpty` | `teammates_squad_charts_test.go` | PASS |
| `TestBuildSquadSynergyRadar_ScoreIsPersonalScorePerMinute` | `teammates_squad_charts_test.go` | PASS |
| `TestBuildSquadPerMinuteStats_DamageSumOverSumAndAbsent` | `teammates_squad_per_minute_test.go` | PASS |
| `TestBuildSquadPerMinuteStats_DamageIgnoresMatchWithoutSeconds` | `teammates_squad_per_minute_test.go` | PASS |
| `TestBuildSquadPerMinuteStats_NoLoaderMainWithoutDamage` | `teammates_squad_per_minute_test.go` | PASS |
| `TestBuildSquadPerMinuteStats_TitleWithoutDamageTaken` | `teammates_squad_per_minute_test.go` | PASS |

Les valeurs sont posées par les assertions (la caractérisation ne journalise rien en `-v`). Valeurs attendues, lues dans le test qui passe :
joueur principal, 3 matchs : frags 0,66, morts 0,31, assistances 0,12, dégâts infligés 300, subis 136,63 par minute ; ami, 2 matchs : 0,23 / 0,35 /
0,35 / 48,8 / 19,21. Le seul test qui journalise (`TestTemoinMinute_SynergyScorePerMinute`) rend, en `-v` :

| Témoin | Valeur |
|---|---|
| synergy.937 | 79.01814300960513 |
| synergy.59 | 78.30508474576271 |
| synergy.zero | 0 |
| synergy.negatif | 0 |

## 3. Garde-rail des couches (`apps/web/src/features/dismissableLayers.guard.test.ts`)

Compté par un test jetable qui rejoue le balayage du garde-rail (mêmes fonctions `scanSources`, `lineOf`, `visitAll` de `lib/archScan.testutil`, même
détecteur copié) ; le garde-rail lui-même passe (4 tests réussis).

| Grandeur | Valeur |
|---|---|
| Fichiers `.tsx` balayés sous `features/` (hors tests) | 591 |
| dont dans `features/match-replay/` (plafond 0) | 105 |
| dont hors rejeu | 486 |
| Fichiers du rejeu portant un motif | 0 |
| Fichiers hors rejeu portant un motif | 5 |
| Fichiers exemptés (`EXEMPTIONS`, chacun avec une raison datée) | 5 |
| Exemptions sans objet / motifs non exemptés | 0 / 0 |

Exemptés : `features/asset-drawer/AssetDockedPanel.tsx`, `features/tactical/TacticalAnalysisView.tsx`, `features/tactical/TacticalPlanTueurs.tsx`, `features/squad/SquadFocusStrip.tsx`, `features/squad/useSquadPresets.tsx`.

## 4. Match `38900a5e` : le fil entre 4:30 et 5:30 et la fiche du joueur A

### Ce qui a été passé aux vraies fonctions, et ce qui manque

SANS SERVEUR, LES FRAGS DE LA BASE (`context.highlight_events` de la vue match, lus par `collectKillEvents`), L'EN-TÊTE DU MATCH ET LA FEUILLE
DE SCORE DE L'API NE SONT PAS DISPONIBLES (aucune base ouverte, aucun serveur lancé). Le témoin est donc fait SANS les frags de l'API ; ce qui a été passé à la place :

- le rejeu réel `38900a5e` (schéma 125), lu par `fs` + `zlib`, normalisé par `normalizeReplayDocument` (`lib/replay/replayNormalize`) ;
- les joueurs : `buildPlayers(doc, [])` (feuille vide : les noms viennent du film, pas de la feuille de l'API) ; l'allégeance : `buildFilmAllegiance`
  du point de vue de JGtm ;
- un en-tête synthétique : `t0_ms` = `t0FilmMs` du document (33415 ms), `playable_duration_seconds` = 600
  (la fin de la fenêtre est donc la fin du film) ; la fenêtre de gameplay `replayWindow` en sort : image de départ 686 (22638 ms),
  image de fin 14943. L'horloge affichée est `displayClockMs` (l'instant du film moins le départ de la fenêtre) : 4:30 vaut l'image 8 879 à 8 880 ;
- les lignes d'entrée et de sortie : `presenceEntries` SANS drapeaux de l'API (pas de feuille, pas de `start_time`) : elles viennent donc du REPLI FILM (source
  `film`, `PRESENCE_DEPART_MS` = 20 s), non de `last_leave_time` ; sur la page réelle, la ligne « a quitté la partie » est posée à l'instant de l'API quand
  la feuille le donne ;
- le fil : `buildFeedEntries` + `mergeFeedWithPresence` (le `model.feed` de la page), puis `withEventLines` avec les mêmes sources d'événements que
  `useFeedEventEntries` (objectifs, fin de manche, crâne, colline, prises, armes au sol, véhicules), filtre « Tout » ;
- le texte est celui que rend le vrai composant `ReplayKillFeed` (testing-library, locale `fr`), lu ligne à ligne ; la famille et le joueur sont lus
  sur les entrées du fil. Le nom d'un joueur vient du film.

Deux variantes :
- **A. sans frags** (aucun frag passé, le témoin demandé) : toute fin de vie que le film prouve devient une ligne « mort » (mort neutre) ;
- **B. frags reconstitués** : les 69 lignes `killPersonalScore` (tueur, cible) de `scoreActions` du document, datées à l'image
  du score, sans arme. Ce n'est PAS le flux de l'API : c'est une approximation qui montre le fil avec des frags. Les lignes autres que les frags sont identiques à la variante A.

Les deux variantes montrent le point qui compte pour la piste A : à l'image 9785, la fin de vie du joueur A (corps supprimé, aucun lâcher, aucun compte à rebours) est
une ligne « mort » (« non prouvée ») ET une ligne « a quitté la partie », deux lignes à la même image.

### Fil de 4:30 à 5:30, variante A (sans frags), de la plus ancienne à la plus récente

| Horloge | Image | Famille | Joueur | Texte affiché |
|---|---|---|---|---|
| 4:30 | 8879 | mort neutre (prouvee) | joueur B | joueur B mort |
| 4:32 | 8953 | medaille | joueur C | joueur C Remote Detonation |
| 4:33 | 8963 | event:pickups | joueur D | joueur D a ramassé : Empaleur |
| 4:41 | 9215 | medaille | joueur D | joueur D Rifleman |
| 4:45 | 9324 | event:pickups | joueur B | joueur B a pris : Déchiqueteur |
| 4:45 | 9350 | event:pickups | joueur E | joueur E a pris : Hydra |
| 4:52 | 9539 | mort neutre (prouvee) | joueur A | joueur A mort |
| 4:54 | 9614 | mort neutre (prouvee) | joueur F | joueur F mort |
| 4:59 | 9749 | mort neutre (prouvee) | joueur E | joueur E mort |
| 5:00 | 9785 | mort neutre (non prouvee) | joueur A | joueur A mort |
| 5:00 | 9785 | presence:left(film) | joueur A | joueur A a quitté la partie |
| 5:03 | 9895 | mort neutre (prouvee) | JGtm | JGtm mort |
| 5:03 | 9895 | mort neutre (prouvee) | joueur B | joueur B mort |
| 5:12 | 10151 | event:pickups | joueur D | joueur D a pris : Bandit EVO |
| 5:13 | 10188 | presence:joined(film) | 343 Mickey | 343 Mickey a rejoint la partie |
| 5:21 | 10416 | event:pickups | joueur B | joueur B a pris : Crémateur |
| 5:25 | 10552 | medaille | joueur D | joueur D Snipe |
| 5:28 | 10631 | mort neutre (prouvee) | joueur B | joueur B mort |
| 5:29 | 10678 | mort neutre (prouvee) | joueur F | joueur F mort |

### Fil de 4:30 à 5:30, variante B (frags reconstitués depuis `scoreActions`)

| Horloge | Image | Famille | Joueur | Texte affiché |
|---|---|---|---|---|
| 4:30 | 8880 | frag | joueur G | joueur G [tueur] > joueur B [victime] |
| 4:32 | 8953 | medaille | joueur C | joueur C Remote Detonation |
| 4:33 | 8963 | event:pickups | joueur D | joueur D a ramassé : Empaleur |
| 4:41 | 9215 | medaille | joueur D | joueur D Rifleman |
| 4:45 | 9324 | event:pickups | joueur B | joueur B a pris : Déchiqueteur |
| 4:45 | 9350 | event:pickups | joueur E | joueur E a pris : Hydra |
| 4:52 | 9539 | frag | joueur D | joueur D [tueur] > joueur A [victime] |
| 4:54 | 9614 | frag | JGtm | JGtm [tueur] > joueur F [victime] |
| 4:59 | 9749 | frag | joueur C | joueur C [tueur] > joueur E [victime] |
| 5:00 | 9785 | mort neutre (non prouvee) | joueur A | joueur A mort |
| 5:00 | 9785 | presence:left(film) | joueur A | joueur A a quitté la partie |
| 5:03 | 9895 | frag | JGtm | JGtm [tueur] > joueur B [victime] |
| 5:03 | 9895 | frag | joueur B | joueur B [tueur] > JGtm [victime] |
| 5:12 | 10151 | event:pickups | joueur D | joueur D a pris : Bandit EVO |
| 5:13 | 10188 | presence:joined(film) | 343 Mickey | 343 Mickey a rejoint la partie |
| 5:21 | 10416 | event:pickups | joueur B | joueur B a pris : Crémateur |
| 5:25 | 10552 | medaille | joueur D | joueur D Snipe |
| 5:28 | 10631 | frag | joueur G | joueur G [tueur] > joueur B [victime] |
| 5:29 | 10678 | frag | joueur G | joueur G [tueur] > joueur F [victime] |

### Fiche du joueur A (état avant la piste A)

Lu par `playerStateAt(player, image, vitalityPresence(doc))` (`lib/replay/rosterLogic.ts`) ; compteurs par `playerCountersAt`. Dernières vies du joueur dans le
document : [4857, 5822] released ; [6067, 7939] released ; [8183, 9539] released ; [9783, 9785] deleted. Présence au tableau : de 0 à 9937.

| Image | État de la fiche | Images depuis la fin de la vie | Image de réapparition lue | Présence au tableau | Score / frags / morts / assistances (calque) |
|---|---|---|---|---|---|
| 9700 | mort | 161 | 9783 | ouverte | 400 / 4 / 8 / 0 |
| 9786 | mort | 1 | -1 | ouverte | 400 / 4 / 8 / 0 |
| 9900 | mort | 115 | -1 | ouverte | 400 / 4 / 8 / 0 |
| 9940 | mort | 155 | -1 | fermée | 400 / 4 / 8 / 0 |

Lecture : aux images 9786, 9900 et 9940 la fiche dit « mort » (`alive: false`, réapparition `-1`), même à 9940 où la présence est fermée depuis 3 images (9937) ;
« absent » n'existe pas dans le code d'aujourd'hui pour cet état. L'image 9700 est dans le compte à rebours de la vie précédente (réapparition lue : 9783).
Gate A : de l'image 9785 (fin de la dernière vie) à 9937 (fermeture de la présence), donc aux images 9786 et 9900, la fiche doit dire « a quitté la partie » ; à 9940 la place n'est plus occupée par lui ; et plus aucune ligne « mort » pour joueur A à l'image 9785.

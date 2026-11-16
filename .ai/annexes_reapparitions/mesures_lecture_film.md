# Réapparitions lues dans le film — première mesure (2026-11-15)

Contexte : câblage de `EntreeOccupationDeLArtefact` sur les champs du décodeur v1.3.9 (schéma 124) :
`tracks[].endReason`, `tracks[].releaseFrame`, `respawnCountdowns[]`, `identity.players[].filmIndex`.
Branche `feat/reapp-film-v139`, livraison 8.2.1.

## Méthode

- Cinq films cuits à la v1.3.9 par `cmd/replay-build`, un par un, dans une racine temporaire
  (`LEVELUP_REPO_ROOT` = dossier de travail de la session) : copies du manifeste, des morceaux du film et
  des faits de film de chaque match. Rien n'est écrit dans le dossier de données principal. Pic mémoire de
  0,36 à 0,65 Gio, 5 à 10 s par film.
- Lecture par une sonde de test temporaire (retirée, non versionnée) qui appelle
  `EntreeOccupationDeLArtefact`, puis `tactical.NouvelIndexVies(...).Spawns()` deux fois : avec les lectures
  du film, et sur une copie privée de ces lectures (le repli décide). Même calcul que
  `accordDefinitions.comparer` de `spawn-model evaluate`.
- Écarts mesurés : frame du lâcher (`releaseFrame`) et début de compte à rebours le plus proche, moins la
  fin de la piste, en millisecondes (pas de 33 ms).

## Résultats

| film | mode | pistes | lâchers | suppressions | comptes | réapp. film | débuts de manche | accord | film seul | repli seul (dont manche) |
|---|---|---|---|---|---|---|---|---|---|---|
| 000d5950 | Assassin | 99 | 95 | 0 | 95 | 91 | 0 | 91 | 0 | 0 (0) |
| 51ebbc0f | Oddball, 2 manches | 84 | 72 | 7 | 72 | 69 | 7 | 69 | 0 | 7 (7) |
| 60ae07c4 | Oddball classé | 169 | 166 | 1 | 168 | 160 | 1 | 158 | 2 | 0 (0) |
| e5adf7b2 | Grande bataille | 216 | 201 | 3 | 183 | 189 | 0 | 188 | 1 | 0 (0) |
| fb1a1a72 | Drapeau, plusieurs manches | 144 | 140 | 0 | 141 | 136 | 0 | 135 | 1 | 0 (0) |
| **total** | | 712 | 674 | 11 | 659 | 645 | 8 | 641 | 4 | 7 (7) |

- Comptes à rebours sans joueur : 0 sur les cinq films.
- Écart lâcher - fin de piste : p5 0 ms, p50 33 ms, p95 33 ms (99 ms en Grande bataille). Même
  distribution pour le début de compte à rebours. La tolérance de 66 ms avant / 198 ms après tient.
- `coverage.tracks.lifeEnds` : `filmEnd` 0 à 12 par film (vies closes par la fin du film), `open` 2 sur
  60ae07c4, `retaken` et `unread` à 0.

## Lecture

- Le repli sur l'écart de 4 à 15 s retrouve 641 des 645 réapparitions lues dans le film (99,4 %).
- Il en ajoute 7, toutes sur le film à deux manches : les débuts de manche, que le film lit comme des
  corps supprimés encore commandés.
- 4 réapparitions du film tombent hors de la fenêtre de 4 à 15 s du repli.
- Cinq films ne sont pas le parc : la mesure A.8 reste à faire sur le parc régénéré à la v1.3.9
  (cuisson en lot, accord du user requis), par `spawn-model evaluate` (bilan de lecture).

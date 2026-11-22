# Mesures de l'étape 0.3 (2026-11-18)

Plan `.ai/PLAN_BACKLOG_2026-11-18.md`, étape 0, item 0.3. Lecture seule des 1 455 rejeux du poste
(`data/cache/replays/halo_infinite/*.json.gz`, un fichier à la fois, aucun lot en mémoire, un seul processus de mesure à la
fois, aucune base ouverte, aucun serveur). Les mesures (b), (c) et (d) appellent les VRAIES fonctions du site depuis des tests
vitest jetables (supprimés ; leurs sources sont gardées en `.txt` sous `sondes/`) ; (a) est un script Node jetable (gardé aussi).
Tous les nombres ci-dessous viennent de ces lancements.

## (a) Départs en cours de match

Script : `sondes/mesure_departs_0_3a.js.txt` (reprend `pilote_mesure_departs.js.txt` et le complète). Définition d'un partant retenue pour
la mesure (celle de `mesure_departs.md`) : une entrée du tableau (`roster[]`) qui n'est pas un bot, dont la présence publiée se ferme
plus de 10 s avant la dernière image ; sa dernière vie est la piste de plus grand `endFrame`.

### Rappel confirmé (même parc, mêmes comptes)

| Grandeur | Valeur |
|---|---|
| Rejeux lus / erreurs / sans présence publiée | 1 455 / 0 / 5 |
| Rejeux avec au moins un départ | 729 |
| Départs | 1 386 |
| Dernière vie finie par un corps supprimé (`deleted`) | 875 |
| Dernière vie finie par un lâcher de commande (`released`) | 503 |
| Dernière vie sans raison publiée | 8 |
| Pistes finies par `deleted`, tout le parc | 1 284 (875 dernières vies de partant, 409 autres) |

### Écart entre la fin de la dernière vie et la fermeture de la présence

Valeurs non numériques écartées : 0 (une piste sans `endFrame` prend l'instant de son dernier point ; aucune n'a manqué des deux).
Aucun écart négatif (la présence ne se ferme jamais avant la fin de la dernière vie).

| Population | n | min | centile 5 | centile 25 | médiane | centile 75 | centile 95 | max |
|---|---|---|---|---|---|---|---|---|
| Tous les départs, en secondes | 1 386 | 0 | 0,26 | 2,90 | 6,11 | 10,07 | 25,54 | 212,22 |
| Tous les départs, en images | 1 386 | 0 | 8 | 88 | 185 | 305 | 774 | 6 431 |
| Dernières vies `deleted`, en secondes | 875 | 0 | 0,10 | 1,65 | 4,98 | 9,14 | 31,78 | 212,22 |
| Dernières vies `released`, en secondes | 503 | 0 | 2,51 | 5,25 | 7,36 | 11,81 | 21,62 | 191,93 |

64 écarts dépassent 30 s (l'annexe précédente disait 62 : le calcul de l'époque portait sur une liste contenant des valeurs non numériques).
Pour un corps supprimé, la présence se ferme plus tôt après la fin de la vie (médiane 5,0 s) que pour un lâcher de commande (médiane 7,4 s). Pour les
partants que la règle A.1 retient (corps supprimé), la durée pendant laquelle la fiche dirait « a quitté la partie » est de 1,65 s au centile 25, 5,0 s à la médiane,
9,1 s au centile 75 et 31,8 s au centile 95.

### Vérification de l'affirmation « aucune dernière vie de partant finie par `deleted` n'a de compte à rebours de réapparition du joueur qui commence dans les 10 images de sa fin »

**Vérifiée : 0 sur 875.** Trois lectures, le même résultat :

| Lecture du compte à rebours | Dernières vies `deleted` de partant avec un compte à rebours proche |
|---|---|
| `respawnCountdowns[]` du même `filmIndex`, début à 10 images au plus de la fin de la vie (ce qu'a mesuré le pilote) | 0 / 875 |
| Idem, clé par xuid comme `countdownStartsByXuid` (index de film porté par une seule entrée du tableau) | 0 / 875 |
| Tolérance de `lifeEndProof` (de -66 ms à +200 ms, soit de -2 à +6 images) | 0 / 875 |

Distance du compte à rebours le plus proche (début de compte moins fin de vie, en valeur absolue) pour ces 875 vies : jusqu'à 10 images, 0 ;
de 11 à 30 images, 1 (match `c349da69`, `piesden`, 20 images : hors de la tolérance de `lifeEndProof` comme de la fenêtre de 10 images) ; de 31 à 300 images, 237 ; au-delà de 300 images, 438 ; aucun
compte à rebours pour ce joueur dans tout le document, 199.
Pour mémoire, côté `released` : 464 des 503 ont un compte à rebours dans les 10 images (les 39 autres n'ont aucun compte à rebours à 10 images : non examinées, hors règle A.1) ; côté « sans raison publiée » : 0 des 8.

### Pistes finies par `deleted` qui ne sont PAS une dernière vie de partant : 409, classées

Règles, appliquées dans cet ordre, première qui tient : (1) piste de bot ; (2) fin dans les 10 dernières secondes du document ; (3) fin de manche :
au moins une autre piste `deleted` finit à 15 images ou moins (suppression groupée de tous les corps) ; (4) piste isolée, dont le joueur a une vie plus tard ;
(5) dernière vie d'un joueur dont la présence ne s'est pas fermée plus de 10 s avant la fin.

| Classe | Pistes | Exemples (match, joueur, image) |
|---|---|---|
| Fin de manche (suppression groupée), le joueur revient à la manche suivante | 201 | `19ef6b04` (4 joueurs à 5 images près, image 22 687 à 22 692) |
| Corps supprimé isolé, le joueur a une vie ensuite | 128 | `06dfe6d9` ZuKu18 image 7 634 (vie suivante 6 952 images plus tard) ; `084a804d` RapidTundra image 25 243 (+305) ; `111cb657` XxXKeNnETh12XxX image 6 082 (+247) |
| Bot | 36 | `03e3f9ea` image 2 729 ; `04d05635` image 2 502 ; `0762ae7c` image 1 356 |
| Dernières secondes du match (moins de 10 s avant la dernière image) | 35 | `008e1bba` Cptn Pirate Joe image 5 154 (0 s) ; `00ba2e1c` Madina97294 image 15 139 (0 s) ; `02205ed3` Cheesefaced image 15 782 (0 s) |
| Dernière vie d'un joueur dont la présence dure jusqu'à la marge de fin de partie (dans les 10 s de la fin) | 9 | `06a883f7` ComboTX20 image 14 440 (33,4 s avant la fin, présence fermée image 15 230) ; `087474d9` Oatmello5674 image 9 306 (17,6 s) ; `5da6fd30` WaiseAbgrund449 image 769 (13,2 s) |
| **Total** | **409** | |

Pour les corps supprimés isolés : la vie suivante du joueur commence à une médiane de 305 images (10 s), centiles 5 et 95 à 92 et 1 966 images, minimum 23, maximum 6 952.
Les 201 de fin de manche sont des suppressions de tous les corps à la fois : la règle A.1 (1) les exclut déjà (le joueur a une vie ensuite).

Deux constats pour A.1 :
- 24 des 875 dernières vies de partant finies par `deleted` appartiennent aussi à une suppression groupée (au moins une autre piste `deleted` à 15 images près) : le
  joueur est parti pendant la pause de fin de manche et ne revient pas. La règle (4) le décide par la présence, pas par le groupement : ils restent des départs.
- Le gate A du plan attend « 875 départs reconnus, 503 morts gardées » : la règle codée ne reconnaît un départ que sur `deleted` (condition 2) ; les 8 dernières vies sans raison
  publiée ne sont pas des départs et gardent le comportement actuel, soit 511 fins de vie gardées sur les 1 386 (503 lâchers de commande + 8), pas 503. À écrire au gate A pour éviter un faux rouge.

## (b) Score à l'instant : calque contre somme des points par action

Ce qui est comparé, pour chaque joueur publié par le calque, à chaque image de changement de l'une ou l'autre série :
- le score du calque : `playerCountersAt(scoreTimelineOf(doc), xuid, image).score` (valeur gardée par la garde d'horloge, `lib/replay/scoreTimeline.ts`) ;
- la somme des points par action : `readInstantEstimatesSeries` puis `playerInstantEstimatesAt(...).score` (`MR/model/playerInstantEstimates.ts`), les paliers cumulés de
  `scoreActions` par destinataire, lus par les vraies fonctions. La garde de feuille (la somme finale égale le score de la ligne de feuille de l'API) n'est PAS
  mesurable sans serveur : la feuille est remplacée par la somme finale, qui la fait passer. Une confrontation a été faite entre la lecture réelle et une copie indépendante des
  paliers (somme cumulée par image) : 12 126 joueurs, 0 écart. Les deux premières gardes de la fiche (couverture `scoreActions.complete` et horloge recalée) sont, elles, appliquées :
  « garde du document tenue » ci-dessous.

| Grandeur | Valeur |
|---|---|
| Rejeux avec un calque de score lisible (garde d'horloge passée, au moins un joueur) | 1 449 sur 1 455 |
| dont sans aucune action de score (le calque s'affiche, la fiche n'a pas de somme) | 82 rejeux, 1 908 joueurs |
| Rejeux comparables (calque ET actions) | 1 367 |
| Joueurs comparables | 12 126 |
| Rejeux où au moins un joueur diffère à au moins une image | 1 272 sur 1 367 |
| Joueurs où les deux séries diffèrent à au moins une image | 6 539 sur 12 126 |
| Images (du film) où les deux diffèrent, tous joueurs | 9 088 545 sur 190 866 187, soit 4,8 % |
| Points de changement où les deux diffèrent | 20 747 sur 203 036 |
| Écart absolu à ces points, en points : 1 à 9 / 10 à 49 / 50 à 99 / 100 et plus | 19 / 2 293 / 4 483 / 13 952 |
| Sens : calque plus haut / calque plus bas | 13 568 / 7 179 |
| Joueurs dont les TOTAUX FINAUX diffèrent | 648 (écart final : min 10, médiane 100, centile 95 1 150, max 31 701) |

La différence est de deux natures, nettement séparées par la longueur des plages d'écart (une plage = images consécutives où les deux valeurs diffèrent) :

| Population | Joueurs | Plages de 1 à 2 images | de 3 à 10 | de 11 à 30 | de 31 à 300 | plus de 300 | Joueurs avec au moins une plage > 30 images | Totaux finaux différents |
|---|---|---|---|---|---|---|---|---|
| Garde du document tenue (`coverage.scoreActions.complete` et horloge recalée), 943 rejeux | 8 096 | 7 771 (3 845 joueurs) | 7 | 4 | 93 | 807 | 781 | 166 |
| Garde du document refusée (la fiche ne montre aucun score), 424 rejeux | 4 030 | 3 445 (1 681 joueurs) | 13 | 8 | 42 | 818 | 759 | 482 |

- **Décalage d'une image** : 7 762 des 7 771 plages courtes de la population gardée durent 1 image (neuf durent 2). Le calque date le point une image après l'action
  (exemple : `000d5950`, LORD PEINX13, image 5 836, calque 550, somme 650 : le point à 650 tombe à l'image 5 836 dans les actions et 5 837 dans le calque).
  Sur les 182 471 valeurs du calque que la somme atteint aussi, 180 706 sont datées à une image près (écart de date de -1 à +1 image) ; 1 765 le sont davantage ; 5 234 valeurs du calque ne sont jamais atteintes par la somme du même joueur.
- **Écart durable** (plus de 30 images, soit 1 s) : 900 plages, 781 joueurs, 2 401 618 images dans la population gardée (1,9 % des images de cette population) ; ampleur : 1 à 9 points, 2 ; 10 à 49, 98 ;
  50 à 99, 59 ; 100 et plus, 741. Exemples : `0076ebdc` DownfieldBook42 (image 6 777, 242 images, calque 800, somme 900) ; `0076ebdc` DiedLikeHeLived (image 13 720, 1 106 images,
  calque 1 050, somme 1 150) ; `0215fe6b` MarSshaL7848 (image 7 576, 3 846 images, calque 400, somme 500) ; `02205ed3` Ricolatra (image 1 492, 1 589 images, calque 100, somme 150).
  Hors garde du document : 860 plages durables, 759 joueurs, 6 675 305 images (10,2 % des images de cette population) ; ampleur : 1 à 9 points, 1 ; 10 à 49, 124 ; 50 à 99, 163 ; 100 et plus, 572 (exemple : `00ba2e1c`,
  LoveFrawg, image 1 955, 13 185 images, calque 50, somme 75). La couverture y est déclarée incomplète : c'est la raison d'être de la garde.

Lecture : (b) est strictement supérieur à zéro, sans ambiguïté, même en écartant le décalage d'une image et en ne gardant que les documents que la fiche accepte.
Non mesuré : combien de ces écarts la garde de feuille de la fiche (somme finale contre score de l'API) refuserait, faute de feuille (pas de serveur, pas de base).

## (c) Lâchers d'équipement datés à fin de vie + 2 dont la place est reprise à + 1 ou + 2

Le champ qui date un lâcher est `equipmentPlacements[].t0`, la place est `equipmentPlacements[].owner` (`at: { slot: owner, t: t0 }` dans `readEquipmentGestures`, `MR/model/equipmentGestures.ts`) ;
`gestureOwner` résout un lâcher par `ownerAtFrame(slot, t - 1)` puis `ownerAtFrameOrLast(slot, t)`. Mesure : tous les lâchers (`kind: 'dropped'`) du parc, résolus par la vraie fonction `gestureOwner` sur
`buildSlotOwnership(buildPlayers(doc, []))` ; pour chacun, la dernière vie de cette place terminée avant l'image du lâcher donne son écart « image du lâcher moins fin de vie ».

| Grandeur | Valeur |
|---|---|
| Rejeux avec au moins un lâcher | 1 245 sur 1 455 |
| Lâchers | 28 540 |
| Sans poseur (`owner` négatif) | 0 |
| Attribués par `gestureOwner` aujourd'hui / non attribués | 28 513 / 27 |
| Écart image du lâcher moins fin de vie : + 1 | 16 049 |
| + 2 | 1 040 |
| + 3 | 69 |
| de + 4 à + 10 | 28 |
| plus de + 10 | 233 |
| aucune vie de la place terminée avant l'image (lâcher en cours de vie : échange) | 11 121 |
| **Lâchers à + 2 dont la place est reprise par un autre joueur à + 1 ou + 2** | **0** |
| Lâchers, tous écarts, dont la place est reprise par un autre joueur à + 1 ou + 2 | 0 |

Le zéro n'est pas un hasard de l'échantillon : sur le parc entier, 4 240 passages d'une place d'un joueur à un autre joueur (vies consécutives de la même place) ont
tous un écart entre la fin de la vie précédente et le début de la suivante d'au moins 101 images (aucun à 1 ou 2 images, aucun à 3 à 100) ; 379 autres vies consécutives de la même place sont du même
joueur. Une place n'est jamais reprise dans les deux images qui suivent une fin de vie.
Les 27 lâchers non attribués ne sont pas des lâchers à + 2 repris : à signaler, non traités.

## (d) Durée de `readInstantDocSeries`

Test jetable dans vitest (Node 24, environnement `node`, le poste de travail), 5 exécutions consécutives par rejeu dans le même processus, première exécution comprise (à froid), temps par
`performance.now()` autour de l'appel seul, après lecture et normalisation du rejeu. Les joueurs sont ceux de `buildPlayers` avec une feuille de score égale à la somme des points par action (pour que la
lecture du score s'exécute comme en vrai quand la couverture le permet).

| Cas | Rejeu | Joueurs | Images | Tirs / touches / dégâts | Score par action complet | 5 exécutions (ms) | Médiane (ms) |
|---|---|---|---|---|---|---|---|
| 4 contre 4 | `7863ad21` | 8 | 15 276 | 2 509 / 1 662 / 1 717 | oui | 5,25 ; 2,27 ; 2,60 ; 9,26 ; 1,43 | **2,60** |
| Grande bataille | `81019bcb` | 26 | 17 668 | 1 805 / 1 290 / 2 033 | non | 9,66 ; 2,07 ; 2,12 ; 2,17 ; 4,82 | **2,17** |
| Match long (le plus long du parc, 39 051 images) | `1c4c63c2` | 24 | 39 051 | 9 103 / 1 262 / 1 987 | non | 7,61 ; 4,26 ; 4,00 ; 13,43 ; 2,95 | **4,26** |
| Complément : 4 contre 4 le plus long | `4a93f0e2` | 8 | 30 179 | 3 393 / 1 952 / 2 095 | non | 4,39 ; 3,64 ; 2,89 ; 1,81 ; 2,17 | 2,89 |
| Complément : Grande bataille, score complet | `9dfd3c24` | 29 | 17 722 | 2 659 / 1 251 / 1 751 | oui | 5,41 ; 2,37 ; 3,96 ; 2,54 ; 12,24 | 3,96 |
| Complément : Grande bataille longue, score complet | `58d09c44` | 26 | 33 009 | 7 540 / 4 304 / 5 347 | oui | 8,82 ; 5,95 ; 3,19 ; 5,65 ; 2,40 | 5,65 |

Les compléments couvrent la lecture du score active (couverture complète) sur une Grande bataille et sur une Grande bataille longue : les trois cas demandés ne l'ont qu'une fois. Maximum de
toutes les exécutions : 13,43 ms. Non mesuré : la durée dans un navigateur (pas de navigateur lancé) ; le rapport entre Node et un navigateur n'est pas connu, la marge de 50 ms contre 5,65 ms laisse un facteur de près de neuf.

## Seuils du plan, statués

| Item | Seuil | Mesure | Décision |
|---|---|---|---|
| D.5 (lâcher daté à fin de vie + 2) | s'exécute si (c) > 0 | (c) = 0 sur 28 540 lâchers, et aucune place reprise à moins de 101 images | **ne s'exécute pas** ; item clos sur la mesure |
| D.6 (séries de la fiche gardées hors de l'instance) | s'exécute si (d) dépasse 50 ms sur un des trois | médianes 2,60 / 2,17 / 4,26 ms (maximum d'une exécution : 13,43 ms ; maximum d'une médiane des six cas : 5,65 ms) | **ne s'exécute pas** ; item clos sur la mesure |
| D.7 (un seul score à l'instant) | s'exécute si (b) > 0 | (b) > 0 : 6 539 joueurs sur 12 126, dont 781 avec un écart de plus d'une seconde dans les rejeux dont les gardes de couverture et d'horloge tiennent | **s'exécute** (la branche « (b) = 0 : l'égalité devient un test » est exclue) |

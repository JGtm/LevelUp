# Santé et bouclier dans le rejeu : où en est la lecture, et à quel niveau de fiabilité

Mesure faite le 2026-11-23, sans code de production. Marques : **vérifié** = relu dans le code ou mesuré sur les films ; **déduit** = raisonné, avec ce qui le vérifierait.

## Réponse courte

- Le décodeur lit déjà la santé et le bouclier à chaque trame du film (environ toutes les 17 ms). **Vérifié.**
- Le document ne publie pas cette lecture. Il ne publie que les points retirés par chaque coup (`damage[]`) et, sur les points de piste, `sh` et `hp`, qui viennent du seul record de position. **Vérifié.**
- Ce qui est publié est exact quand il est présent, mais incomplet : environ la moitié des lectures de santé n'arrive pas dans les points de piste, et `sh` est borné à 1 (le surbouclier disparaît). **Vérifié** pour les comptes, **déduit** pour la cause.
- Pour mettre santé et bouclier au niveau des munitions : publier la série déjà lue, par vie, en ne gardant que les changements. Aucune lecture nouvelle à écrire dans le décodeur.

## Hypothèses

- « Niveau des munitions » = valeur connue à chaque changement, au pas de la trame du film.
- Trois films récents du dossier de référence, carte donnée par le catalogue des bornes (pas d'aide de test) : `03b7bbb6` (8 joueurs, Slayer), `221dd082` (8 joueurs, Slayer), `51101d1d` (13 joueurs, Drapeau). Échantillon petit : les chiffres disent un ordre de grandeur, pas une garantie sur tout le parc.
- Une « vie » est un bipède de joueur ou de bot qui a une piste, du premier enregistrement de sa création à la fin de piste que le film écrit (sinon sa suppression, sinon la fin du film). Les vies commencées avant le film : aucune sur ces trois films.
- Sonde jetable : fichier de test sous étiquette `research`, non suivi par git, dans le worktree de la branche `piste/portage-mes`. Cette branche est **33 commits derrière** la tête de la version en cours (`feat/v1.6.0`) et 0 devant. Les trois fichiers qui portent le canal (`internal/grammar/canal_des_vitalites.go`, `replay/vie_des_bipedes.go`, `replay/degats_en_points.go`) sont **identiques** à la tête et à la branche `piste/p-tirs` (diff vide, vérifié). Le retard ne change donc pas la mesure. `document_damage.go` diffère sur `piste/p-tirs` : cette branche ajoute la désignation par piste et les messages à zéro point, sans toucher au calcul des points.

## Ce que fait le code (vérifié)

- `internal/grammar/canal_des_vitalites.go` : un canal de la marche des trames. Il range, par vie (slot et génération), chaque lecture des composants santé (de -1 à 1) et bouclier (de 0 à 4, 1 = plein). Une création de bipède est une lecture à part : la vie naît pleine, rien n'est écrit à la naissance.
- Consommateur unique : `replay/degats_en_points.go` (calcul des points par message de dégât) via `replay/vie_des_bipedes.go`. Aucune publication directe de la série.
- Publié : `damage[]` (points retirés au bouclier et au corps, au dixième) ; `tracks[].points[].sh` et `.hp`, lus dans le record de position du même instant, gardés au premier échantillon de chaque case de 33 ms (`replay/tracks_publication.go`). `sh` est écrêté à 1 (commentaire de `equipment_episodes.go`).
- Compteur de couverture propre au canal : `coverage.hits.vitalsUnread` (trames dont la liste n'est pas lisible). Valeur mesurée ici : 0 sur les trois films. Il n'existe pas de compteur de couverture des lectures de santé ou de bouclier par joueur.

## 1. Cadence (vérifié)

| Film | minutes de vie | lectures de bouclier par minute de vie | lectures de santé par minute | changements de bouclier par minute |
|---|---|---|---|---|
| 03b7bbb6 | 42,9 | 516 | 17,0 | 37,5 |
| 221dd082 | 59,3 | 868 | 17,6 | 67,1 |
| 51101d1d | 19,4 | 811 | 23,9 | 62,1 |

- Écart médian entre deux lectures : 17 ms (une trame) pour le bouclier, sur les trois films. Santé : médiane 17 ms, 90e centile 0,17 à 0,34 s. Écart maximal : 30 à 43 s (bouclier), 37 à 88 s (santé), c'est-à-dire un joueur qui n'est pas touché.
- Environ 92 % des lectures de bouclier répètent la valeur précédente (par exemple 20 651 sur 22 260 sur `03b7bbb6`, creations et lectures de santé seule comprises). La cause de ces répétitions n'est pas établie. **Déduit** : le jeu réécrit le composant tant que le bouclier est dans un état actif. À vérifier en croisant ces répétitions avec l'état de recharge du bouclier.
- Recharge : écart médian entre deux lectures qui montent = 17 ms, pas médian = 2,2 points, soit un cran de 140/64 = 2,19 points. La recharge est donc écrite à chaque trame, cran par cran.

## 2. Couverture (vérifié)

| Film | vies avec au moins une lecture (bouclier / santé) | temps de vie avec une valeur écrite depuis moins de 2 s (bouclier) | idem santé | idem, la naissance comptée comme lecture |
|---|---|---|---|---|
| 03b7bbb6 | 105 sur 105 / 105 sur 105 | 16,8 % | 4,8 % | 25,0 % |
| 221dd082 | 95 sur 95 / 95 sur 95 | 27,4 % | 4,0 % | 32,8 % |
| 51101d1d | 35 sur 35 / 35 sur 35 | 25,3 % | 5,9 % | 31,2 % |

Lecture : toutes les vies ont des lectures, mais une valeur écrite depuis moins de 2 s ne couvre qu'un quart du temps de vie au mieux. Le reste est une valeur tenue depuis le dernier changement. C'est exact si le film écrit chaque changement (voir 3). Ce n'est pas une preuve en soi.

Ce que le document publie aujourd'hui sur les pistes (même mesure sur `tracks[].points`) :

| Film | points `sh` par minute de piste | points `hp` par minute | temps couvert < 2 s par `sh` | par `hp` |
|---|---|---|---|---|
| 03b7bbb6 | 274 | 8,1 | 17,6 % | 3,5 % |
| 221dd082 | 438 | 7,6 | 28,0 % | 3,1 % |
| 51101d1d | 466 | 12,7 | 27,9 % | 4,8 % |

Le document garde donc environ 43 à 53 % des lectures de santé du canal (7,6 à 12,7 contre 17,0 à 23,9 par minute). Pour le bouclier, la comparaison par minute n'a pas de sens, car 92 % des lectures sont des répétitions. La part du temps couverte à moins de 2 s est presque la même que celle du canal. **Déduit** : ce qui manque au document, ce sont des changements isolés, pas du temps. Non mesuré : combien de changements de santé tombent dans une case de 33 ms sans point `hp`.

## 3. Fiabilité

### Bouclier et corps contre les messages de dégât (vérifié)

Méthode : pour chaque scène (une unité blessée à un instant de trame) dont **tous** les coups sont chiffrés par la table des effets (pas par la vie lue, ce qui serait circulaire), comparer la perte prévue à la perte lue (vie juste avant l'instant contre vie à l'instant). Tolérance : **3 points**, soit un cran de lecture (2,19) plus l'arrondi. Résultat identique à 6 points.

| Film | scènes de bouclier comparées | accord | scènes de corps comparées | accord |
|---|---|---|---|---|
| 03b7bbb6 | 269 | 268 (99,6 %) | 122 | 121 (99,2 %) |
| 221dd082 | 993 | 991 (99,8 %) | 308 | 308 (100 %) |
| 51101d1d | 343 | 343 (100 %) | 139 | 138 (99,3 %) |
| Total | 1 605 | **1 602 (99,8 %)** | 569 | **567 (99,6 %)** |

- Les 3 désaccords de bouclier : la lecture est plus faible que la prévision (aucun cas de lecture plus forte). Lecture à l instant ou à la prochaine lecture : même résultat. Les 2 désaccords de corps n ont pas été classés.
- Limite 1 : les scènes avec au moins un coup mesuré sont exclues (186, 82 et 9 scènes). Elles sont nombreuses sur `03b7bbb6` (186 contre 363 retenues). L'accord ne vaut que pour les coups à montant fixe.
- Limite 2 : pour le corps, la règle « tir à la tête qui achève » s'appuie sur la vie lue : cas partiellement circulaire.
- Baisses de bouclier sans message de dégât au même instant : 11 sur 410, 359 sur 1 411, 143 sur 490. Sur les deux derniers films, 349 sur 359 et 138 sur 143 partent d'un bouclier au-dessus de 1 : c'est la décroissance du surbouclier, qui n'est pas un coup. Reste 26 baisses sur 2 311 (1,1 %) sans explication. Cause non cherchée.

### Santé contre les points de dégât (vérifié)

Vies avec au moins une lecture de santé et aucun coup mesuré : la dernière santé lue est à moins de 3 points (sur 90) de la santé impliquée par les points de dégât dans 93 cas sur 96 (96,9 %).

### À la mort (vérifié)

Mort = la fin de piste que le film écrit (calage des débuts de piste sur les créations : écart médian de -45 à +1 ms sur les trois films).

- La dernière lecture de santé **avant** la fin de piste n'est pas basse de façon fiable. Sur 218 morts : 102 (47 %) n'ont aucune lecture de santé avant la mort. Sur les 116 qui en ont une : 47 % sont à 0,25 ou moins, 61 % à 0,5 ou moins, 39 % au-dessus de 0,5.
- La raison : l'écriture de la santé à 0 arrive **après** la fin de piste. Première lecture de santé à 0 ou moins : médiane 37 à 39 ms après (90e centile 53 ms sur deux films, 367 ms sur le troisième). Fenêtre de +0,5 s : 215 morts sur 218 (98,6 %) ont une santé à 0 ou moins ; fenêtre de +5 s : 218 sur 218.
- Autre cause (**déduit**) : le joueur passe de la pleine santé à zéro d'un seul coup quand le bouclier tombe en même temps. Aucune lecture intermédiaire n'est alors écrite.
- Arbitre (points de dégât, vies à coups tous chiffrés par la table, 88 morts) : santé impliquée à 0 ou moins pour 67 (76 %), entre 0 et 0,25 pour 18 (20 %), au-dessus de 0,5 pour 2. Sélection : 130 morts exclues pour un coup mesuré.

## 4. Ce qui manque pour « à l'instant »

- Le film écrit la santé et le bouclier **entre les images-clés**, dans les trames delta : propriétés `object-body-vitality` et `object-shield-vitality` du bipède (**vérifié**, code du canal et mesure : 17 ms d'écart médian). Aucune dépendance aux images-clés.
- Le décodeur les lit déjà, à chaque trame, sans les publier en série (**vérifié**).
- Ce qui manque est donc une **publication** : une série par vie (frame, bouclier brut de 0 à 4, santé de -1 à 1), compressée aux changements. Ordre de grandeur : 1 609 changements de bouclier et 731 lectures de santé pour 105 vies sur `03b7bbb6` (**vérifié**), soit quelques milliers de lignes par film.
- Deux limites de lecture à connaître : la santé finale (0) arrive environ 38 ms après la fin de piste ; le bouclier de recharge se lit trame par trame, mais 92 % des lectures sont des répétitions, qu'il faudra filtrer.
- Non mesuré : les builds anciennes et les builds plus récentes que ces trois films, les véhicules (leurs unités n'ont pas de lecture : 152 scènes sur `03b7bbb6`, 35 sur `221dd082`, 0 sur `51101d1d`).

## Copie du contributeur externe (lecture seule, vérifié)

Point de départ : tag `v1.3.6` du dépôt d'origine.

- Les fichiers du décodeur qui parlent de vitalité, de santé ou de bouclier ont été comparés au tag par empreinte. `internal/grammar/vitality.go` est **identique**. Huit fichiers du même ensemble sont modifiés (dont `replay/tracks_publication.go`, `internal/grammar/offline_biped.go` et `replay/filmfacts*.go`), mais **aucune ligne modifiée ne porte sur la santé ou le bouclier** (aucune ligne de diff ne cite vitalité, bouclier, santé, `Hp` ou `Sh`). Aucun fichier neuf ne parle de vitalité.
- La publication des pistes à pleine cadence du contributeur ne porte pas `sh` ni `hp` (colonnes de position et de visée seulement).
- Le contributeur a ajouté du travail côté lecteur 3D, pas côté décodeur : un modèle de barre de bouclier reconstruit sur le client à partir des échantillons `sh` (`shieldHud`, avec un modèle de recharge qui remonte au rythme du jeu et tient la valeur sans limite pour la santé), des impacts de bouclier, une présence « santé / bouclier lisible » (`vitalityPresence`). Cela compense la rareté des échantillons publiés.
- Le plan de portage ne mentionne aucun changement de lecture de la santé ou du bouclier : **confirmé** (il cite seulement les dégâts en points et le lecteur `shieldImpacts3d.ts` comme consommateur du calque des dégâts). Le README du contributeur ne dit rien de la santé ni du bouclier.
- Réserve (**déduit**) : les autres fichiers modifiés n'ont pas été relus ligne à ligne ; une modification de la lecture des positions partagée avec les vitalités n'est pas exclue par ce contrôle.

## Contrôles et limites de la mesure

- Calage des pistes sur les créations : écart médian -45 à +1 ms ; 10e centile jusqu'à -989 ms (certaines créations précèdent d'une seconde le début de piste).
- La sonde répète certaines lignes de sortie (insertion dans une boucle) : les valeurs sont celles d'un calcul unique, dédoublonnées au relevé.
- Les mesures de santé à la mort et de couverture reposent sur la fin de piste écrite par le film. Le canal des morts d'objets de la marche est vide dans cette configuration (aucune mort lue) : il n'a pas servi.
- Seuil « bas » choisi par moi : 0,25 et 0,5 de la santé (soit 22 et 45 points sur 90).

## En bref

Santé et bouclier sont **lus à l'instant** par le décodeur (une écriture par trame, 17 ms) mais **publiés par moments** : seulement dans les points de piste, ce qui garde environ la moitié des lectures de santé, et le bouclier écrêté à 1. La lecture est fiable : accord de 99,8 % sur le bouclier et 99,6 % sur le corps avec le calcul des dégâts par la table (3 films, tolérance 3 points), accord de 96,9 % sur la santé. Pour passer au niveau des munitions, il suffit de publier la série déjà lue, aux changements.

Décisions attendues :
- **Publier une série santé et bouclier par vie** (changements seulement, bouclier brut jusqu'à 4) : c'est ce qui met santé et bouclier au niveau des munitions, sans nouvelle lecture.
- **Dire si la mesure doit s'étendre** à d'autres builds et à des films de véhicules : ces trois films ne les couvrent pas.

Attention : à la mort, la santé à 0 n'arrive qu'environ 38 ms après la fin de piste ; un lecteur qui prend la dernière valeur avant la mort lira souvent une santé haute ou rien.

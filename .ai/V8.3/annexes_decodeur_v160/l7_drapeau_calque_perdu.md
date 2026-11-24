# Causes des reculs du calque des portages du drapeau (parc 1 455 rejeux, v1.4.1 vers v1.6.1)

Méthode : les deux documents publiés (schéma 125 avant, 127 après) comparés, puis décodage des trois films aux tags
v1.4.1, v1.5.0, v1.6.0 et v1.6.1 dans un clone séparé (carte donnée par le catalogue de bornes, mêmes entrées que la
production : lecture du tableau des scores, rafales de capture, pont par les morts, socles du catalogue d'objectifs pour
les deux films d'arène, aucun socle pour le grand format, comme en production). Les chiffres du décodage reproduisent
ceux des documents publiés (1 puis 5 captures, 19 puis 0 portages, 34 puis 33 portages) : la reproduction est
vérifiée. Le tag v1.5.0 et le tag v1.6.0 sont des squashs : la bascule est localisée au tag, pas au commit.

Les joueurs sont désignés par un rang (par temps de porteur de l'API, décroissant) ; aucun identifiant n'est recopié.

## 1. Film be758198 (Grande bataille, Fiesta CTF, build 1.11, 24 joueurs)

Verdict : DÉFAUT (une lecture juste existe et une règle de reconnaissance du mode la bloque). Vérifié.

### 1.1 Cause prouvée (vérifié)

- Le calque n'est construit que si le film est reconnu comme du drapeau : `IsFlagFilm`
  (`internal/facts/objectives/flagfilm.go:78-80`) exige rafales > 0, captures > 0, vols > 0 ET captures <= rafales.
  Cette règle n'a pas changé entre v1.4.1 et v1.6.1.
- Ce qui a changé est son entrée. Jusqu'à v1.5.0, `EntreeDuDrapeau` (`replay/porteurs_entrees.go:94-97`) ne passait
  que les huit premiers sièges (`objectives.RecordsDeLaBandeSure`). Sur ce film à 24 joueurs, le film ne montrait que
  1 capture et 1 vol (compteur `seatsBeyondBand` = 16 dans le document d'avant). Le test passait : 1 <= 3.
- La v1.6.0 (tag, schéma 127, squash 2fe2f49) supprime la bande (`RecordsDeLaBandeSure` et `SiegesAuDelaDeLaBandeSure`
  n'existent plus, `porteurs_entrees.go:94-96` passe `recs` entier). Le film se lit désormais sur tous les sièges :
  5 captures et 8 vols. 5 > 3 : le test échoue, `flagFilm` passe à faux, le calque est vide.
- Bissection par décodage de ce seul film : v1.4.1 et v1.5.0 donnent 19 portages ; v1.6.0 et v1.6.1 donnent 0
  (captures 1 puis 5, vols 1 puis 8). La bascule est dans v1.6.0.
- Les deux replis `repli_index_drapeau_zero_pour_tous` et `repli_nombre_drapeaux_hors_catalogue_sans_passage`
  (19 occurrences chacun) ne se déclenchent qu'à l'assemblage du calque (une fois par portage). Ils ont disparu
  parce que le calque n'est plus construit, pas parce qu'un commit les retire. Ils restent au registre des replis.
  Vérifié : avec la règle neutralisée dans un clone de travail (retrait de `captures <= rafales`, non commité), ils reviennent
  (42 occurrences).

### 1.2 Les rafales de ce film ne sont pas des captures (vérifié, mécanisme déduit)

- Les 3 rafales tombent à 237,2 s, 597,5 s et 957,9 s (horloge du match) : exactement 360,36 s d'écart, aux
  morceaux 12, 30 et 48 du manifeste, à 16,9 s après le début de chaque morceau. Les 5 captures lues dans le tableau
  des scores tombent à 169,8 s, 398,3 s, 617,3 s, 651,7 s et 1013,9 s. Distance de chaque rafale à la capture la plus
  proche : 20 s au mieux.
- Quatre de ces cinq captures coïncident à 0,2 s près avec les quatre points de la courbe d'équipes du document
  (décalage d'horloge 34,1 s). La cinquième tombe à 979,8 s du document, la fin du film étant à 987 s : elle est
  absente de la courbe d'équipes (2 contre 2). Déduit : une capture finale (3 contre 2) que la courbe ne porte pas ;
  pour le vérifier, relire la dernière seconde du film et le score final du match.
- Déduit : les rafales sont ici une retransmission périodique de la table de score (6 paliers vus dans une même trame),
  pas un évènement de capture. Pour le vérifier : lister les trames qui portent les 6 paliers dans les morceaux 12, 30
  et 48 et regarder si elles portent aussi un point de la courbe d'équipes.
- Conséquence : la règle « captures <= rafales » suppose que les rafales comptent des captures (sa justification
  d'origine : un film tronqué a moins de captures lues que de rafales). Ici elle est fausse parce que le signal de
  référence est faux, pas parce que les captures sont fausses.

### 1.3 Ce que publiait l'avant (vérifié sur le document schéma 125)

- 19 portages, 2 porteurs sur 7 réels, un seul objet drapeau d'équipe -1 (neutre), 37 intervalles qui alternent
  « porté » et « lâché » toutes les 0,5 à 1,5 s (fragments d'un même porteur), 33,8 s de temps de porteur au total
  (22,2 s et 11,6 s) contre 141,5 s à l'API.
- Cohérence avec les captures : un seul portage se termine sur une capture (364,0 s, capture à 364,2 s). Les quatre
  autres captures ne sont pas couvertes. Le drapeau est publié « lâché » de 649 s à la fin du film (338 s), alors
  qu'une capture a lieu à 980 s. Le calque d'avant n'était donc pas cohérent avec le score du match.
- Il tenait parce que la bande de huit sièges cachait quatre captures sur cinq au test de reconnaissance (accident
  heureux), pas par une lecture juste. Ses deux replis étaient actifs sur tous ses portages.

### 1.4 Ce que donne la lecture quand la règle est levée (vérifié, un clone de travail, non publié)

Temps de porteur par joueur (secondes), rangs par temps API décroissant :

| Rang | API | Avant (publié) | v1.6.1 règle levée |
|---|---|---|---|
| 1 | 40,2 | 0 | 38,1 |
| 2 | 37,5 | 0 | 39,3 |
| 3 | 20,4 | 22,2 | 27,2 |
| 4 | 18,1 | 0 | 30,4 |
| 5 | 10,3 | 0 | 13,4 |
| 6 | 9,8 | 11,6 | 17,8 |
| 7 | 5,2 | 0 | 5,3 |
| 8 à 24 | 0 | 0 | 0 |
| Somme | 141,5 | 33,8 | 171,5 |
| Erreur absolue cumulée | | 114,9 | 34,1 |

- Les 7 vrais porteurs sont tous trouvés, aucun faux porteur parmi les 17 autres. Les fins de portage tombent à moins
  de 0,2 s des cinq captures (135,6 s, 364,0 s, 583,0 s, 617,5 s, 979,6 s ; captures du tableau, décalage 34,1 s).
- Défauts restants de cette lecture (ne justifient pas le blocage) : somme en excès de 21 %, 42 portages fragmentés,
  un seul objet drapeau (équipe inconnue sur les 42 : carte sans socle dans le catalogue, `spawns` = 0, base non
  lisible dans le film), 3 chevauchements.
- Règle qui bloque : `captures <= rafales` à `flagfilm.go:79`. La correction est dans la reconnaissance du mode, pas
  dans la lecture du porteur.

### 1.5 Registre des changements de sortie (vérifié)

Aucune mention de cette perte dans `replay/document_chronicle.go`. L'entrée du cinquième apport du schéma 127 (« le
drapeau lit tous les sièges ») annonce « aucun joueur juste ne devient faux » sur 13 films à drapeau ; be758198 n'en
fait pas partie. Une entrée plus ancienne (« sièges non ouverts au drapeau », autour de la ligne 4430) note que 24
films BTB sans rafale n'ont aucun calque, avec ou sans bande ouverte. La perte n'est donc pas une décision inscrite :
c'est un effet de bord non mesuré.

### 1.6 Autres films du même cas dans le parc (vérifié sur les sorties du parc)

- 236 rejeux de CTF au registre ; 189 sont reconnus après, 47 non. Parmi les 47, 45 ont des captures et des vols lus
  (tous à 16 joueurs ou plus, tous des grandes batailles) mais pas de calque, avant comme après.
  - 42 n'ont AUCUNE rafale : la règle les écarte, comme avant. Pas une régression.
  - 3 ont des rafales mais plus de captures que de rafales après : be758198 (seul à avoir un calque avant), puis
    9ae1d4ae (Fiesta CTF sur Command, 1 rafale, 3 captures, 7 vols, 28 joueurs) et a991ea77 (Fiesta CTF sur Breaker,
    1 rafale, 3 captures, 4 vols, 25 joueurs), qui n'en avaient pas avant (captures 0 et 2 contre 1 rafale).
- Aucun autre film reconnu avant ne perd sa reconnaissance (185 reconnus avant et après, 5 gagnent, 1 perd).
- Dans les 189 CTF reconnus après, captures <= rafales dans 100 % des cas (154 égales, 35 inférieures) : la règle ne
  gêne que ces trois films, mais elle les prive tous d'une lecture que le tableau des scores donne.

## 2. Films d'arène 6a711138 (Détachement) et 70a1c6c6 (Némésis), build 1.13

Verdict : VOULU (le portage perdu était faux). Vérifié ; l'hypothèse de la jauge est infirmée.

### 2.1 L'hypothèse de la jauge ne tient pas (vérifié)

Lectures de jauge, intervalles de jauge et appariements sont identiques avant et après (142 lectures, 2 appariements
pour l'un ; 362 lectures pour l'autre). Le compteur neuf `gaugeReadsOffDrops` (8 et 29) est un compteur de plus, sans
effet sur le calque : il n'écarte rien qui entrait dans un portage. La perte vient de la lecture du tableau des scores.

### 2.2 Cause prouvée (vérifié)

- Dans les deux films, une émission du tableau des scores d'un siège compte un vol multiple à un même instant :
  2 vols de ce siège à 49,3 s (6a711138) et 3 vols de ce siège à 201,3 s (70a1c6c6). Avant : 8 et 13 vols ; après :
  6 et 10. Les deux portages perdus sont ouverts exactement à ces instants (38,3 s du document dans l'un, 153,7 s dans
  l'autre, décalage d'horloge de 10,9 s et 47,5 s).
- L'enregistrement en cause n'est pas une lecture juste : à v1.4.1 il ne porte qu'un composant et ses valeurs sont
  hors domaine (une valeur de 12 489 en face d'un compteur de 2 ; une valeur négative de -79 en face d'un compteur de
  3 et une autre de -11). C'est un enregistrement lu sur un en-tête fortuit.
- Le tag v1.5.0 (schéma 126, squash dc967c2) lit désormais les valeurs figées des manches (composants 28 à 55,
  `internal/grammar/signaux/statborg_figees.go`, appel dans `statborg.go:343`) : un enregistrement dont la lecture
  échoue sur l'un d'eux est abandonné (`statborg.go:197`, compté par `repli_enregistrement_statborg_abandonne`, 5 953
  puis 6 105 sur le grand format). Cet enregistrement-là est abandonné ; les vols fantômes disparaissent. La bascule
  est à v1.5.0 (décodage à ce tag : 33 portages et 10 vols pour 70a1c6c6 ; 6 vols pour 6a711138, dont un portage de moins).
- Pas de lien avec la bande à huit sièges : ces films ont 9 à 10 joueurs.

### 2.3 Les portages perdus étaient faux (vérifié contre l'API)

| Film | Portage perdu | Suite publiée avant | Effet sur le temps de porteur du joueur | API |
|---|---|---|---|---|
| 6a711138 | 0,6 s (38,3 s à 38,9 s) | drapeau « lâché » 19,8 s, jusqu'à la prise suivante | 13,9 s avant, 13,3 s après | 13,2 s |
| 70a1c6c6 | 8,2 s (153,7 s à 161,9 s) | drapeau « lâché » 21,2 s | 9,3 s avant, 1,1 s après | 1,0 s |

Erreur absolue cumulée contre l'API : 1,6 s puis 0,9 s (6a711138), 12,3 s puis 4,0 s (70a1c6c6). Après, le drapeau
reste à sa base sur ces intervalles, ce que la prise suivante confirme. Les captures voisines sont inchangées (3 et 4).

## 3. Récapitulatif

| Film | Avant | Après | API | Verdict | Cause |
|---|---|---|---|---|---|
| be758198 | 19 portages, 33,8 s | 0 portage | 141,5 s | défaut | v1.6.0 ouvre tous les sièges, `captures <= rafales` (`flagfilm.go:79`) devient faux sur des rafales périodiques |
| 6a711138 | 9 portages | 8 | | voulu | v1.5.0 abandonne un enregistrement du tableau des scores fortuit |
| 70a1c6c6 | 34 portages | 33 | | voulu | idem |

## 4. Correction prescrite (défaut seulement)

Remplacer la règle « captures <= rafales » de `IsFlagFilm` par une condition qui ne suppose pas que les rafales
comptent les captures (par exemple l'accord des captures lues avec la courbe d'équipes, ou le rapport vols sur captures
qui écarte Oddball : 994 vols pour 6 captures), puis la mesurer sur les 45 CTF sans calque et sur les films non CTF
(le critère d'origine : zéro faux positif). Lever la règle seule est insuffisant à prouver : seule la lecture de
be758198 est mesurée ici ; les 42 films sans aucune rafale demandent une autre condition de reconnaissance, non étudiée.

## 5. Limites

- Le décodage hors production n'a ni base ni lignes du match : le pont d'identité n'est complété que par les morts.
  Les résultats reproduisent pourtant à l'identique les documents publiés (compteurs de couverture, nombre de
  portages) sur les trois films.
- Le temps de porteur de l'API est un témoin par joueur (une valeur par film) ; je n'ai pas d'API par capture.
- Le mécanisme exact qui produit les rafales périodiques n'est pas prouvé (voir 1.2).

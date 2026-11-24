# D.1 - Controle des comptes de repliques et d'annonces contre le temoin tiers

Date : 2026-11-24. Aucun code de produit modifie ; sondes `zz_d1_*_test.go` (etiquette `research`) supprimees, arbre du decodeur propre.

## Methode (verifie)

- Decodeur : branche `integ/contenu`, tete `4d0e9b7`. Deux sondes : lecture de la passe des tetes (messages lus, genres 75 et 76, avec horodatage, son, masque) et cuisson complete du document (`chatter[]`, `announcements[]`, couverture).
- Temoin : libfilmshell (Rust, MIT) compile en local, hors ligne, avec un petit programme d'ecriture neuve qui liste ses evenements `Chatter` (AIDialog) et `VoiceLine` (Dialogue2D) avec instant, son, locuteur, masque. Aucune sortie anterieure du temoin ne donnait de comptes : le temoin a ete execute.
- Films (cartes donnees par le catalogue des bornes) : d9781168 (Dredge, 8 joueurs), bcb6d393 (Cliffhanger, 15 joueurs dont 4 bots), fb1a1a72 (Banished Narrows, 8 joueurs). Un film a la fois.
- Limite du temoin (verifie) : sur 9 films du parc essayes, le temoin n'en lit que 3 (les 6 autres : erreur `UnknownHeader`, build de film qu'il ne connait pas). Le controle porte donc sur ces 3 films, tous d'un meme build lisible par le temoin.

## Regle de lecture, point par point (verifie dans les sources du temoin et les notres)

| Point | Temoin | Nous |
|---|---|---|
| Repliques | evenement AIDialog retenu quand il porte un son | genre 75, tous les messages ; le son est present sur 100 % d'entre eux ici |
| Annonces | evenement Dialogue2D retenu quand il porte un son ; un evenement par destinataire | genre 76 ; un message par destinataire a la lecture, regroupes par (frame, son) a la publication |
| Locuteur | unite -> joueur par la propriete du slot sur tout le film, bots compris | unite -> piste a la frame ; un bot ou une unite sans piste = publiee sans joueur, comptee |
| Hors match | tous les evenements, y compris avant le coup d'envoi | ecartes a la publication (`beforeOrigin`) |

## Comptes

| Film | Temoin repliques | Nous lues / publiees | Temoin annonces | Nous lues / ecartees avant origine / reprises / publiees |
|---|---|---|---|---|
| d9781168 | 365 | 365 / 365 | 951 | 951 / 6 / 464 / 481 |
| bcb6d393 | 165 | 165 / 165 | 217 | 217 / 7 / 69 / 141 |
| fb1a1a72 | 418 | 418 / 418 | 538 | 538 / 8 / 168 / 362 |

## Egalites exactes (verifie, comparaison message par message)

- Lues : repliques et annonces egales au temoin sur les 3 films, dans le meme ordre ; sons identiques un a un ; masques des destinataires identiques un a un ; priorite et indice de replique (champ `line`) identiques un a un pour les repliques publiees.
- Echelle de temps : la pente entre instants du temoin (ticks) et nos horodatages vaut 16 667 microsecondes par tick, a 0,5 % pres, sur tous les messages des deux genres.
- Locuteurs : la partition des repliques par locuteur est la meme (correspondance un a un entre les joueurs du temoin et les notres, sans conflit ni collision) pour toutes les repliques publiees avec joueur : 8/8 joueurs, 11 des 12 locuteurs, 8/8.

## Ecarts et explications

1. Annonces ecartees avant l'origine (6, 7, 8 selon le film) : le temoin les compte, nous les ecartons parce qu'elles precedent le coup d'envoi (verifie : ce sont les premiers messages du film, instants du temoin 499 a 706 sur le premier film). EXPLIQUE.
2. Annonces reprises (464, 69, 168) : le jeu ecrit l'annonce une fois par destinataire dans la meme trame ; nous en faisons une ligne par (frame, son) avec l'union des masques. Verifie : en regroupant les messages lus par (frame de 33 ms, son) avec l'origine ajustee sur les repliques, on retrouve exactement 481, 141 et 362 lignes ; les masques unis sont egaux a ceux du temoin regroupe par (instant, son) sur les films 2 et 3 ; sur le film 1, les masques sont egaux message a message en lecture (le regroupement du temoin par instant ne suffit pas a reproduire notre regroupement par frame de 33 ms, qui fusionne 3 groupes de plus). EXPLIQUE.
3. Repliques d'un bot publiees sans joueur (film bcb6d393 : 2 repliques, instants du temoin 19341 et 19488, sons 42707f7a et bbe12d5e) : le temoin les attribue a un joueur de type bot (verifie, c'est un bot), nous les publions sans joueur par choix documente et les comptons (`bots` = 2). EXPLIQUE.
4. Une replique publiee sans joueur, hors bot (film fb1a1a72, `withoutPlayer`=1, `bots`=0) : INEXPLIQUE a la racine, localise. Instant du temoin 28804 (frame 12993 du document), genre 75, replique d'indice 850 (nom `shieldbreak`), son 340630bc, sans nom de joueur chez nous. Le temoin l'attribue a un joueur dont il compte 18 morts ; le document n'a que 17 pistes pour ce joueur : la vie qui se termine a la frame environ 12998 (mort du temoin a l'instant 28815, onze ticks apres la replique) n'a pas de piste dans le document (le joueur a une piste jusqu'a 12571, puis la suivante commence a 13303). La replique est donc posee dans une vie sans piste. Ce n'est pas une erreur de lecture de la replique (lue, instant et son egaux) : c'est une vie manquante dans les pistes. Verifie pour les 18 morts contre 17 pistes et les frames ; la cause de l'absence de cette piste n'est pas cherchee ici (hors perimetre).
   Exemples demandes : un seul cas sur les 3 films (instant 28804, genre 75, sans nom de joueur).

Autres points verifies : aucune replique ni annonce ecartee apres la fin (`afterEnd` = 0) ; aucune trame arretee (`stoppedFrames` = 0, la lecture est complete) ; les identites de la couverture tiennent (lues = avant origine + apres fin + publiees pour les repliques ; lues = avant origine + reprises + publiees pour les annonces).

## Verdict

Comptes lus egaux au temoin sur 3 films sur 3, message par message ; comptes publies egaux a un ecart explique pres (annonces avant le coup d'envoi, annonces regroupees par destinataire, repliques de bot sans joueur). Un residu inexplique de 1 replique sur 948 au total, dans les pistes (une vie sans piste), pas dans la lecture. Reserve : le temoin ne lit que 3 des 9 films essayes, d'un meme build.

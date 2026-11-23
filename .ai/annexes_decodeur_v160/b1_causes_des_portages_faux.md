# B.1 - cause des 16 portages dont l'etat "au socle" est faux

Mesure sans code de production. Joueurs designes par leur rang dans le roster du document publie (J0, J1...), films par leur identifiant court. Aucun identifiant de joueur. Tout est "verifie" (commande rejouee, piece relue) sauf mention "deduit".

## Methode (verifie)

- Decodeur : tete de `piste/mes-b` (commit bf69129, version en cours), aucun fichier de production modifie. Seul fichier ajoute au worktree : la sonde `research/b1/b1_research_test.go` (etiquette `research`, non commitee).
- Chaine : les 13 films decodes un a un par `replay-build` (LevelUp, bati avec le decodeur du worktree par `-modfile`, `GOWORK=off`), racine de donnees propre (`pistes/b1/repo`), faits et oracle de l'API exportes de la copie `verite` (`pistes/b1/faits`, contient des identifiants : hors depot). Les 16 portages sont retrouves a l'identique au tete de la version (portage qui suit l'etat "au socle" et dont la prise est a plus de 8 m du socle) : 16 sur 16, memes films, memes instants que le plan. La regle de la 1.5.0 qui ecarte les emissions hors domaine est deja en place.
- Les simulations de correction tournent sur une COPIE du decodeur (`pistes/b1/copie`, hors depot), avec un interrupteur par correction. Tableaux : `portages.tsv` (les 16 portages), `justes_avant.tsv` (joueurs justes avant), `temps.py`, `cmp_sim.py`, `ctx.py`, `jhist.py`, `orph.py` (scripts d'analyse, hors depot).
- Critere "juste" : temps de porteur publie a 2 s ou moins du temps de l'API (`time_as_flag_carrier_seconds`).

## Les 16 portages (verifie)

Voir `portages.tsv` (film, build, mode, instant du match en secondes, porteur publie, equipe, distance de la prise au socle publie "au socle"). Builds : HI_1_10_0 (6), HI_1_11_0 (1), HI_1_13_0 (9). Dans chacun, le porteur publie et l'ouverture sont justes (le film et l'API confirment la prise) ; c'est l'etat AVANT la prise qui est faux : le drapeau gisait ailleurs.

## Famille 1 : porteur precedent hors des 8 premieres places ou sans humain nomme (7 portages)

Cause PROUVEE pour 6 sur 7 : le calque du drapeau ne lit que les enregistrements des 8 premiers sieges (slots statborg 10 a 24), `replay/porteurs_entrees.go:95` (`objectives.RecordsDeLaBandeSure`, borne `StatborgSlotBandeSure = 24`). Les prises et vols faits depuis un siege au-dela (slots 26 a 56) ne sont pas lus : le drapeau reste publie au socle, et le premier joueur de la bande qui le ramasse ensuite le prend loin du socle.

Pieces :
- 0f9550e5 (jeu a drapeau neutre) : vol puis prise d'un joueur du slot 26 a 194,8 s et 200,2 s, aucun portage publie ; l'API lui donne 11,5 s de porteur, publie 0,0. Le portage cible (206,8 s, 15,1 m du socle) est la reprise du drapeau qu'il a lache.
- 390b1de5 : 396,3 s, vols et prises des slots 56 (359,7 s) et 26 (375,3 s) non lus ; 659,9 s, prises des slots 40 (636,6 s et 652,3 s) et 54 (657,6 s) non lues. L'API : temps de porteur 46,3 s, 15,0 s, 5,0 s, 7,1 s a ces quatre joueurs, publie 0,0.
- 5f90df2e : slot 44 vole a 228,4 s (API : 60,2 s, publie 0,0), la prise cible a 248,7 s en est la suite.
- cdb0ffcf (2 portages) : 274,4 s et 328,1 s, prises des slots 28, 42, 52, 54, 26 non lues (API : 16,4 s, 77,3 s, 101,3 s, 21,5 s, publie 0,0).
- Les deux voies de nommage d'un siege (pont par les morts sur tous les sieges, lecture de l'entree d'entite) donnent le meme joueur pour 1 049 evenements du drapeau sur 1 049 nommes par les deux, sur ces 13 films ; l'entree nomme 66 evenements de plus que le pont, le pont 6 de plus que l'entree (joueurs sans entree). Le risque cite par `EntreeDuDrapeau` (un siege qui change d'occupant) n'apparait pas dans ce relevé.

Simulation (copie, bande ouverte au drapeau, `Records: recs`) : 0f9550e5, les deux 390b1de5, 5f90df2e et les deux cdb0ffcf passent a 0 portage loin du socle ; joueurs a 2 s de l'API sur les 13 films : 161 sur 218 avant, 197 apres, aucun joueur juste ne devient faux.

Le 7e portage (3b289bd8, 559,9 s) a une autre cause, PROUVEE aussi : le porteur precedent (vol a 541,8 s, siege 3, dans la bande) est un bot arrive en cours de match (identite `bid`, sans identifiant de joueur ; sa piste publiee porte un identifiant vide). Le portage est ouvert puis ecarte faute de piste (`coverage.flagCarries.noTrack` = 1, `replay/flag_carries.go:418`). Correction non simulee (deduit) : cle de piste du bot par son `bid` quand le pont le nomme, `flagCarrierTrackIndex` (`replay/flag_carries.go:396`, `replay/flag_carrier_tracks.go`). Regle du projet : les bots comptent partout.

Correction prescrite : lire les sieges au-dela de la bande pour le drapeau en nommant le porteur par l'entree d'entite complete par le pont (`EntreeDuDrapeau`, `porteurs_entrees.go:95`, aussi `SeatsBeyondBand`) ; une LECTURE, aucun seuil. Regle des 6 portages reglee : 6 (7 avec la cle de piste du bot, non simulee).
Effet ailleurs : fort mais dans le bon sens sur ces films (+36 joueurs justes) ; a mesurer sur le parc de 186 films : l'ancienne tentative d'ouverture (E.3) avait donne 85,7 %, cette mesure ne la reproduit pas sur 13 films. Effet de bord a surveiller : voir "Combinaison".

## Famille 2 : variante a un seul drapeau publiee avec deux drapeaux d'equipe (4 portages)

Cause PROUVEE : sur "BTB:One Flag CTF" (Refuge), il n'existe qu'un drapeau, au socle (95,4 ; 96,0) que le catalogue range en equipe 0. Les deux camps le volent, un camp par manche. La couverture le dit : `filmBases` = 1 (les vols des deux camps tombent au meme point, base neutre lue) et `filmBaseContradict` = 1. Mais `flagChoiceOfNeutralBase` (`replay/flag_film_bases.go:390`) n'adopte la base neutre lue que si le socle du catalogue a sa portee ne nomme aucun camp (`flagCamplessSpawnNear`, ligne 410) ; ici il nomme l'equipe 0 : le film est ignore, la contradiction est comptee, deux drapeaux d'equipe sont publies (socle 2 a (131,5 ; 82,5), un point de capture). Quand un joueur de l'equipe 0 vole (manches 1 et 3), l'invariant "jamais son propre drapeau" (`flag_assign.go:293-300`) refuse le socle (95,4 ; 96,0), et le portage tombe sur l'autre drapeau, a 38 a 39 m (distance publiee : 38,1 a 39,3 m). Preuve geometrique : tous les portages publies de ces films a partir de l'etat "au socle" sont a 0,2 a 3,6 m de (95,4 ; 96,0) pour l'equipe 1 et sur le drapeau a 39 m pour l'equipe 0.

Simulation (copie, base neutre lue adoptee meme quand le catalogue nomme un camp a portee) : les 4 portages passent a 0 ; aucun joueur ne change de temps de porteur (161 sur 218, identique) ; aucun portage juste ne devient faux.

Correction prescrite : dans `flagChoiceOfNeutralBase`, retenir comme socle unique de la variante neutre le socle du catalogue le plus proche de la base lue (a moins de 3 m) meme s'il nomme un camp ; un seul drapeau, equipe neutre, plus de refus d'invariant. Lecture du film (les deux camps volent au meme point), pas de nom de variante. Fichier : `replay/flag_film_bases.go`. Regle des 4 portages reglee : 4 sur 4.
Effet ailleurs : sur les variantes ordinaires les camps volent a des points distincts (separation minimale connue 8,46 m, seuil 6 m) : aucun changement attendu (deduit : a verifier sur les 186 films). Le document publie un drapeau au lieu de deux sur ces films : changement de sortie a annoncer.

## Famille 3 : residu (5 portages), deux causes

Cause A PROUVEE pour 4 portages (50f16538, 71d56200, a306a162, cfb85a58) : un siege qui change d'occupant en cours de manche emet un enregistrement ou tous les compteurs valent 0/0. Le compteur d'un siege est lu par la "plus longue sous-suite non decroissante" de la manche (`internal/facts/objectives/score.go:231` `longestRun`, appelee a `named_series.go:221`) ; quand la suite de l'occupant partant et celle de l'occupant qui arrive ont la meme longueur, la derniere l'emporte et les vols et prises du partant disparaissent. Pieces (compteur vol, composant 24 A) :
- a306a162, siege 2 : suite 1 (119,9 s), 2 (182,6 s), 0 (252,1 s, remise a zero, le joueur est parti a cet instant), 1 (486,9 s) ; les deux vols du partant sont perdus (API : 2 vols, 27 s ; publie 16,6 s). Le portage cible (129,7 s) reprend le drapeau qu'il portait depuis 119,9 s.
- 71d56200, siege 6 : suite 1 (444,1 s), 0 (498,7 s, remise a zero) ; le vol de 444,1 s est perdu (API : 1 vol, 23,2 s ; publie 18,2 s). Le drapeau a ete lache a 450,5 s (vie libre de l'objet nee a 1,3 m du lieu de la reprise, hors socle) puis repris a 454,4 s.
- 50f16538, siege 7 : suite 1 (110,2 s), 2 (174,8 s), 0 (251,0 s, remise a zero) ; les deux vols perdus (API : 2 vols, 12,4 s ; publie 0,0).
- cfb85a58, siege 3 : vol 63,4 s et prise 67,6 s, remise a zero a 79,7 s (le joueur part a la 79,6 s du film) ; perdus.
Simulation (copie : coupure de la suite d'un siege a chaque enregistrement tout a zero apres une valeur non nulle, `rawSeriesJugees`) : 50f16538, 71d56200 et a306a162 passent a 0 ; 6 joueurs devenus justes (161 -> 167 sur 218), aucun devenu faux. cfb85a58 recupere ses deux ouvertures (62 -> 64) mais elles sont ecartees faute de nom : le pont par les morts ne nomme pas le premier occupant du siege 3 (`non_resolu`, 0 a 1 987 images), alors que l'entree d'entite le nomme. D'ou une seconde pièce pour ce film : nommer par l'entree (meme correction de nommage que la famille 1, deduit).
Correction prescrite : couper la suite d'un siege aux changements d'occupant, plutot qu'a une heuristique de valeurs a zero : la frontiere est un fait du film (fin de la plage de listage du partant, deja lue pour `presence`, et arrivee du suivant) ; la sonde a utilise le zero total faute de mieux, risque de fausse coupure si un enregistrement a zero apparait sans changement d'occupant (deduit : a mesurer sur le parc). Fichiers : `internal/facts/objectives/named_series.go` (`rawSeriesJugees`, `rawSeriesByRound`), `namedEventsOfTable` dans `named.go`. Attention : touche les compteurs nommes de tous les modes, pas seulement le drapeau. Regle des portages reglee : 3 (4 avec le nommage par l'entree).

Cause B PROUVEE pour 1 portage (d0293d39, 321,0 s) : le vol du meme joueur a 308,1 s est ouvert mais ecarte faute de position : la piste publiee a un trou de 29 images (957 ms) entre l'image 9131 et l'image 9160 alors que le joueur est immobile sur le socle (meme point a 0,02 m avant et apres : le film ne replique que ce qui change), et la tolerance de position est de 200 ms (`replay/flag_carrier_tracks.go:161`, `pointOfXUIDAt` ligne 170) : `coverage.noTrack` = 1. Simulation : tenir la position quand les deux points voisins de la meme piste sont a moins de 0,15 m ; d0293d39 passe a 0, J6 devient juste (161 -> 164 en isole), aucun devenu faux. Correction prescrite : cette lecture (corps immobile), `pointOfXUIDAt`. Effet : ouvre davantage de portages sur les autres films (portages aujourd'hui ecartes par `noTrack`) ; position tenue seulement si le corps n'a pas bouge, donc sans position inventee. Regle : 1.

## Combinaison (simulation de toutes les corrections, copie)

Bande ouverte + coupure par occupant + position d'un corps immobile + un seul drapeau : portages loin du socle 16 -> 3 ; joueurs a 2 s de l'API 161 -> 209 sur 218 ; aucun joueur juste devenu faux (`devenus faux` vide sur les 13 films). Restent : 3b289bd8 (bot), cfb85a58 (premier occupant du siege non nomme par le pont), et un portage NOUVEAU sur 5f90df2e (J17, 11,1 m) : avec la bande ouverte et la coupure par occupant, plus de vols sont lus, la lecture des bases du film (`readFlagFilmBases`) passe a deux camps, retient 2 socles sur 6 et publie 2 drapeaux au lieu de 6 ; un des deux est mal choisi. Risque reel sur les cartes a 6 socles : a regarder avant de combiner (non instruit au-dela de la couverture : spawns 6 -> 2, `filmBases` 2).

## Pour le gate : joueurs JUSTES aujourd'hui (a 2 s de l'API)

161 joueurs sur 218 ayant une ligne d'API, sur les 13 films, listes un par un (film, rang J, temps publie, temps API) dans `justes_avant.tsv`, colonne `juste_a_2s` = oui. Aucun ne doit devenir faux apres correction ; dans toutes les simulations ci-dessus : 0 joueur devenu faux. Les 57 "non" de la meme table sont les ecarts a reduire.

## Ce qui n'est pas etabli

- Effet des corrections sur les 186 films a drapeau : non mesure (13 films decodes). A mesurer avant d'ecrire : bande ouverte (famille 1) et coupure par occupant (famille 3) sur le parc entier, en comparant le temps de porteur par joueur.
- Nommage du premier occupant du siege 3 de cfb85a58 par l'entree d'entite et cle de piste d'un bot : deduits, non simules.
- Frontiere exacte du changement d'occupant (listage plutot qu'enregistrement a zero) : non simulee.
- Le joueur cfb85a58 J4 a 8,7 s de plus que l'API (48,3 contre 39,6 s publie) : hors des 16 portages, non instruit.

## Hygiene

Aucun processus de decodage reste en cours (verifie : seuls des `go vet` / `go build` d'autres sessions tournent). Le worktree `piste/mes-b` ne contient que la sonde non suivie `research/b1/`. Le dossier de donnees du worktree `LevelUp-wt-dec160/data/cache` a recu une ecriture parasite lors du premier essai (rejeu et faits de d0293d39, deplaces dans `pistes/b1/stray/`) ; un dossier vide `replays/halo_infinite` y est reapparu apres, non cree par mes commandes (mes sorties vont dans `pistes/b1/repo`), laisse en place.

# Causes des pertes du controle du corpus (v1.5.0 contre tete eb0ae98 de feat/v1.6.0)

Methode : clone local `clone150`, un binaire `replay-build` par commit (espace de travail Go local, hote non modifie), un film a la fois, carte donnee par le catalogue (084a804d Fortitude Heavies, a349fea8 Fragmentation Heavies, 4f77afc1 Flood Gulch, 50247b26 Oasis, bcb6d393 Cliffhanger, d9781168 Dredge). Les sondes (impressions) ont ete posees dans le clone puis retirees ; le worktree de la v1.6.0 n'a pas ete touche. Aucun identifiant de joueur reel : "joueur n°1", "joueur n°2", etc. ; les films sont nommes par leurs 8 premiers caracteres.
Les documents "avant" et "apres" du gate (dossier de travail du controle) ont aussi ete relus tels quels ; les binaires de bissection reproduisent a l'identique les compteurs du gate (084a804d : 183 -> 0 reproduit).

## Famille 1. Jauge du drapeau perdue (084a804d)

Commit : `3f6eda6` ("le drapeau lit les prises de tous les sieges"). Bissection : parent `fc9fc37` = jauge 183 points, gaugePaired 1 ; `3f6eda6` = 0 (captures 1 -> 5, vols 3 -> 10, portages 23 -> 41, seatsBeyondBand 16 -> 0). vérifié.
Ligne : `replay/flag_return_gauge.go` : `bindFlagGauges` (l. 176), seuil `flagReturnGaugeMinShare = 0.80` (l. 75, test l. 185). Le commit change les portages, pas ce fichier.

Ce que la jauge de 183 points mesurait : une serie reelle (le slot de jauge monte de 0 a 0,98 par cycles et le drapeau revient seul a la fin : dernier cycle 20283-20702, retour a 20707). Mais la v1.5.0 la posait sur le MAUVAIS drapeau : un drapeau de l'equipe 0 declare "lache" d'un seul bloc de 13 700 frames (452 s, 7006-20706) parce que les portages des sieges au-dela des 8 premiers n'etaient pas lus ; le slot (equipe 1) tombait dedans a 100 %. vérifié (spans et serie relus dans les deux documents).
Pourquoi la tete ne la produit plus : avec tous les sieges lus, ce bloc fantome disparait (le drapeau est capture entre 7656 et 8815). Le slot de jauge 1639 (230 echantillons) tombe a 36 % dans les lachers du drapeau 0 et 59 % dans ceux du drapeau 1 (deux drapeaux de la meme equipe, 94,8 % pour leur union) ; aucun ne passe 80 %. Meme sort pour le slot 1644 (equipe 0 : 79,4 % sur le drapeau 4, 97,9 % avec le drapeau 2). vérifié (sonde de `bindFlagGauges`). La chronique le savait deja (`replay/document_chronicle.go` v127, ligne "Ouvrir la bande fait aussi perdre a 084a804d son appariement de jauge ... 0,793, sous 0,80").
Autres films a drapeau du corpus : seuls 3 sur 19 ont un calque de drapeau. bcb6d393 (1 jauge, 3 slots) et fb1a1a72 (2 jauges, 6 slots) : jauges inchangees. Aucun film ne gagne de jauge. 084a804d est le seul film a plusieurs drapeaux par equipe (6 drapeaux). vérifié.
Verdict : DEFAUT (perte reelle de lecture ; la v1.5.0 etait, elle, mal rattachee, donc la sortie "avant" n'etait pas juste non plus). Le jeu porte une jauge par equipe qui sert plusieurs "drapeaux" du document (déduit), et la regle "un slot = un drapeau a 80 %" n'a ete mesuree que sur des films a un drapeau par equipe.
Correction prescrite : apparier un slot a l'ensemble des drapeaux d'une meme equipe (part calculee sur l'union de leurs lachers), puis poser chaque point dans le lacher qui le contient.
Aussi : `carrierTeamUnknown` 23 -> 41 = 100 % des portages avant comme apres (23 sur 23, 41 sur 41) : le denominateur grandit, rien ne regresse. Verdict voulu. vérifié.

## Famille 2. Socles (padDating.uncovered / ambiguous, groundWeapons.unknown)

Hypothese : CONFIRMEE. Commit `2a563d3` ("socles d'equipement") ; bissection sur a349fea8 : parent `2ac0a7e` = 48 socles, uncovered 97, ambiguous 36, unknown 67 ; `2a563d3` = 67 socles, 133 / 40 / 77. vérifié.
Preuves sur 4 films (sondes avant / apres sur la datation de chaque occupation, par socle, clef = position + arme) :

| film | socles avant | socles apres | socles anciens identiques (datation, bornes, statut) | socles neufs | classes des neufs (date / ambigu / non couvert) |
|---|---|---|---|---|---|
| a349fea8 | 48 | 67 | 48 sur 48 | 19 | 65 / 4 / 36 |
| 084a804d | 30 | 45 | 30 sur 30 | 15 | 55 / 13 / 54 |
| 4f77afc1 | 23 | 31 | 23 sur 23 | 8 | 43 / 5 / 38 |
| d9781168 | 8 | 11 | 8 sur 8 | 3 | 20 / 2 / 2 |

- La liste des socles d'ARME (clef a huit chiffres hexa) : zero socle d'arme neuf sur les 4 films ; les neufs sont tous des equipements (grappin, repulseur, mur, ecran, capteur, propulseur, champ de reparation, balise de translocation). vérifié.
- Les socles anciens (armes ET power-ups d'avant) : meme presence, memes apparitions, memes ramassages (instant, ramasseur), memes comptes par classe, meme statut "unknown" par socle. Les 14 autres films : 0 difference sur les socles anciens et leurs ramassages (comparaison des documents du gate). vérifié.
- Les hausses sont exactement les socles neufs : a349fea8 uncovered +36, ambiguous +4, dated +65 = +105 occupations = `powerupOccupations` +105 ; 084a804d uncovered +54 (89 -> 143), ambiguous +13 (71 -> 84). `groundWeapons.unknown` : la somme des socles anciens vaut exactement le compte "avant" (67, 127), l'ecart "apres" ne peut donc venir que des socles neufs. La liste `groundWeapons` (armes au sol) est identique pour 19 films sur 19. vérifié.
Aucun socle d'arme n'a regresse.
Verdict : VOULU (le perimetre compte a grandi, les compteurs gardent la meme definition ; les clefs de couverture du gate devraient etre lues par famille de socle, ou le gate devrait comparer sur le sous-ensemble ancien).

## Famille 3. Tirs et touches

### 084a804d : tir 475 -> 474 (champ `v` de vehicule)
Un seul tir a change, a la frame 15804, arme de vehicule 0xA8000594 (tiree une seule fois dans le film). Avant : joueur n°2 (au siege de pilote du vehicule 822 de 15638 a 15876), avec sa position et le vehicule. Apres : joueur n°3, a pied, 55 m plus loin, sans vehicule. Commit `f6e3441` (parent `2a49ff4` = 475). vérifié.
Cause : le record de tir porte deux champs en desaccord : place 21 et unite 765. Le registre rattache le slot 765 au joueur n°3 (pont par slot, sans date : la piste de ce slot n'existe qu'a partir de 20662, longtemps apres), donc `liaisonsDesTirs` (`replay/places_de_controle.go` l. 149) lie la place 21 au joueur n°3 pour une frame. La lecture des places prime (`replay/tirs_par_place.go` l. 123) et reattribue le tir. La place 21 est tenue par le joueur n°2 sur les 11 autres tirs de la place dans +/- 400 frames : la lecture compte elle-meme un "relais" suivi d'un "retour" (`returns` 1) pour une seule frame. C'est le seul desaccord entre la lecture et le siege du roster sur ce film (1 sur 9 654). Le mecanisme (slot reutilise lu comme occupant) est déduit ; l'effet et l'incoherence (siege du vehicule, position, tenue de la place) sont vérifiés.
Verdict : DEFAUT (un tir juste devient faux). Correction prescrite : refuser une liaison isolee entre deux tenues du meme joueur (un retour en une frame) et ne lier que si le corps de l'unite existe a l'instant du tir.

### 4f77afc1 : 2 019 -> 2 005 touches avec arme (`hits.w`)
Les 2 621 touches restent toutes publiees ; 14 perdent leur champ `w` (arme du dernier tir), aucune autre difference. Commit `f6e3441` (parent `2a49ff4` : 2 019 ; lui : 2 005). vérifié.
Cause : `buildHits` recoit les tirs reattribues par la lecture des places, mais `poserUneTouche` cherche l'arme avec l'index du message de degat (`replay/document_hits.go` l. 371, `derniereArme` l. 196). Pour ces touches, la place 14 n'avait pas de tenant au siege du roster (index laisse tel quel, lu comme un bot) et la lecture la donne au joueur n°1 (index 29, 310 tirs) : les tirs y sont maintenant ranges sous A, la touche cherche toujours sous l'index 14 et ne trouve rien. Les autres touches : meme mecanisme sur les index 1 et 2 (relais de joueurs). vérifié (sonde : cle de recherche sans aucun tir, tirs de la fenetre ranges sous d'autres joueurs, meme arme).
Verdict : DEFAUT (un champ juste devient absent ; les tirs de la place sont, eux, mieux attribues : identifies par joueur, +20 et +21 sur deux joueurs, 331 -> 315 non resolus). Correction prescrite : chercher l'arme avec le tireur publie de la touche (son xuid), pas avec l'index du message.

### 4f77afc1 : un joueur passe de 100 a 0 de score
Courbe du score du joueur n°1 : avant [100 a 12497] ; apres [100 a 12497, 0 a 26806]. Commit `436c09c` (parent `46be3cf` : 100 ; lui : 0). vérifié. Le meme document porte, pour ce joueur, une trahison comptee a 26806 (compteur "trahisons" = 1, identique avant et apres) : le jeu retire 100 points d'une trahison, la baisse tombe exactement a cet instant et vaut exactement 100, ce qui est la regle de confirmation de `internal/facts/objectives/score_penalties.go` (le compteur monte dans la meme emission). La valeur de l'API n'est pas dans la reference locale (les faits du film ne portent pas de score) : l'accord avec l'API est déduit (a verifier sur le score personnel de l'API de ce joueur : attendu 0).
Verdict : VOULU (plus juste : un score qui ignore la penalite d'une trahison est faux).

### bcb6d393 : ambiguousReturns 0 -> 1, noBridge 0 -> 1, objectives.noSlot 0 -> 2
Rien n'est perdu : tous les spans de drapeau et toutes les lignes d'objectif de la v1.5.0 sont presents a l'identique ; la tete AJOUTE 3 portages du drapeau de l'equipe 0 (3696 a 4717 : deux porteurs, trois prises, deux lachers) et 13 lignes d'objectif (vols de drapeau, prises, securisations, kills autour de 3698-4098). Les lignes d'objectif et la prise de 3698 se confirment mutuellement (marqueur du film confirme +1, vols 4 -> 6). Les compteurs montent parce que ces objets neufs sont comptes : noBridge 1 = une ouverture neuve sans pont, ambiguousReturns 1 = un retour neuf ambigu, noSlot 2 = 2 des 8 lignes neuves non rattachees (attached 31 -> 37, aucun ancien detache). Commits `3f6eda6` (sieges lus) et `2734cea` (compteurs coupes au changement d'occupant). vérifié (comparaison ensembliste des spans et des objectifs) ; l'attribution fine des deux compteurs a un portage precis n'est pas faite (déduit).
Verdict : VOULU (gain de couverture ; les trois compteurs sont des denominateurs neufs).

## Famille 4. Repli `repli_tireur_par_le_siege_de_la_place`

Commit `f6e3441` (`replay/tirs_par_place.go` l. 189 : `DeclencheN(..., sansLecture)`, ou `sansLecture` compte les tirs dont la source n'est PAS la place lue, c'est-a-dire siege ET sans tenant ; `replay/fire_bursts.go` pour les rafales). vérifié.
Ce que compte le repli : 084a804d 2 (tous "sans tenant" : un tir sans unite presente, un tir d'une unite hors piste) ; 4f77afc1 6 (siege : tous place 2, frames 8985-9495) ; 50247b26 1 256 (tous "sans tenant").
Pourquoi 1 256 sur 50247b26 : film de version majeure 31, aucun roster ni siege publie (`coverage.seats` tout a 0, `controlPlaces` : 0 liaison, 0 joueur, shotsUnheld 1 256). Aucune place ne peut etre lue ni rapportee au siege : les 1 256 tirs porteurs d'un tireur (100 % des evenements de tir de ce film) tombent dans la branche "sans tenant" qui laisse l'index tel quel, et declenchent pourtant le repli. Part des tirs du film : 100 % ; ailleurs : 0,02 % (084a804d : 2 sur 9 654), 0,08 % (4f77afc1 : 6 sur 7 983), 0 sur bcb6d393 (1 199 tirs, tous par la place lue). Bots, nombre de joueurs, mode : sans effet ; la seule variable est un roster vide (build ancien). vérifié pour les comptes ; la cause "roster vide" est lue dans la couverture du document.
Coherence : sur 50247b26, tirs, touches, bilans, rafales sont IDENTIQUES a la v1.5.0 (hors le champ nouveau `o`) ; aucun tir n'est attribue a quelqu'un. Sur 4f77afc1 les 6 tirs du siege vont au joueur n°4 (index 30), qui est aussi celui que la lecture donne a la place 2 sur 329 autres tirs, et dont le slot publie est le meme avant et apres ; il est present (presence de 6730 a 33667). vérifié. Le repli a ete ajoute sans mise a jour de FORMAT.md (registre a 128 entrees contre "127 declarations" : documentaire, deja note par le controle).
Verdict : VOULU quant a la sortie (rien ne change) ; le gate le range en "faux" parce que le compteur tourne aussi quand rien n'est substitue (branche sans tenant). Decision a prendre : compter le repli seulement quand le siege est reellement utilise (source siege), ou inscrire 50247b26 dans la decision du gate.

## Tableau de synthese

| Famille | Commit | Verdict |
|---|---|---|
| 1 Jauge du drapeau (084a804d) | 3f6eda6 | defaut (perte de lecture ; l'ancienne etait mal rattachee) |
| 2 Socles | 2a563d3 | voulu (hypothese confirmee, 0 socle d'arme regresse) |
| 3a Tir 475 -> 474 (084a804d) | f6e3441 | defaut (liaison place -> joueur isolee, une frame) |
| 3b Touches avec arme 2 019 -> 2 005 (4f77afc1) | f6e3441 | defaut (recherche d'arme non alignee sur la nouvelle attribution) |
| 3c Score 100 -> 0 (4f77afc1) | 436c09c | voulu (penalite de trahison, acces API non verifie) |
| 3d Compteurs bcb6d393 | 3f6eda6, 2734cea | voulu (portages et objectifs neufs) |
| 4 Repli du tireur | f6e3441 | voulu pour la sortie ; compteur trop large pour le gate |

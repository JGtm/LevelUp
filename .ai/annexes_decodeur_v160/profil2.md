# Second profil du decodeur de film (tete de `feat/v1.6.0`, apres G2, G6 et G1)

Mesure du 2026-11-23, un decodage a la fois, machine a 16 coeurs, 15 Go de memoire libre au depart (13,1 Go au plus bas).
Aucune modification du depot du decodeur. Les trois optimisations mesurees ici : liste d'une trame non localisee
recuperee une fois (G2), filtres de positions en place (G6), pre-test de tete avant un decodage d'essai (G1).

## 0. Conditions de mesure

- Avant le releve : aucun `go`, `film-bench`, `replay-build`, `replay-equiv` en cours. 24 processus `node` presents mais
  dormants (0 a 5 Mo, aucun temps CPU).
- Echantillonnage toutes les 5 s (processus de la liste, temps CPU de tout processus hors `film-bench`) pendant le
  releve (81 echantillons), puis pendant les profils et le A/B (160 au total) : AUCUNE activite etrangere de la liste ;
  seul l'editeur de code tournait, a environ 0,6 s de CPU par tranche de 5 s (12 % d'un coeur sur 16), dans presque
  tous les echantillons. Mes propres commandes `go` (analyse de profil, compilation de la reference) n'ont tourne qu'entre
  deux mesures, jamais pendant une. Memoire libre minimale pendant un decodage : 13,1 Go.
- Releve de reference (v1.5.0) : pris, d'apres la consigne, sous charge. Pour savoir si l'ecart sur les films non touches
  est du bruit, j'ai recompile le commit de reference (`e2b9c96`, extrait hors depot) et alterne reference et tete sur
  4 films, 2 tours (section 1.2).

## 1. Releve complet (13 films, 5 passes, plafond 3 Gio)

Empreintes : les 13 films portent, pour le document ET pour les faits, EXACTEMENT les empreintes du releve de la v1.5.0.
Aucune ne differe. (Les deux colonnes d'empreintes de `tete.tsv` et de `bench/releves/v1.5.0.tsv` sont comparees film par
film, ligne `total`.)

Medianes en millisecondes. Colonnes : decodage, publication et total, reference v1.5.0 puis tete ; variation du total ;
pic memoire (Mio, empreinte memguard) reference puis tete.

| film | decodage ref | decodage tete | publication ref | publication tete | total ref | total tete | variation | pic ref | pic tete |
|---|---|---|---|---|---|---|---|---|---|
| film01 | 2 051 | 2 153 | 252 | 277 | 2 310 | 2 438 | +5,5 % | 508 | 466 |
| film02 | 1 032 | 1 019 | 84 | 92 | 1 123 | 1 121 | -0,2 % | 250 | 234 |
| film03 | 1 333 | 1 360 | 126 | 133 | 1 461 | 1 502 | +2,9 % | 329 | 334 |
| film04 | 18 518 | 13 068 | 1 110 | 1 202 | 19 644 | 14 302 | -27,2 % | 2 198 | 2 045 |
| film05 | 485 | 495 | 33 | 35 | 525 | 537 | +2,4 % | 128 | 100 |
| film06 | 17 895 | 12 139 | 201 | 215 | 18 096 | 12 362 | -31,7 % | 454 | 434 |
| film07 | 7 469 | 6 467 | 543 | 604 | 8 031 | 7 100 | -11,6 % | 992 | 898 |
| film08 | 2 299 | 2 111 | 232 | 257 | 2 542 | 2 376 | -6,5 % | 397 | 452 |
| film09 | 9 551 | 6 493 | 136 | 142 | 9 701 | 6 643 | -31,5 % | 349 | 286 |
| film10 | 1 450 | 1 488 | 125 | 138 | 1 586 | 1 638 | +3,3 % | 332 | 347 |
| film11 | 1 250 | 1 262 | 111 | 124 | 1 371 | 1 396 | +1,8 % | 379 | 318 |
| film12 | 3 757 | 2 543 | 202 | 223 | 3 973 | 2 775 | -30,2 % | 520 | 500 |
| film13 | 1 097 | 1 123 | 91 | 100 | 1 197 | 1 236 | +3,2 % | 277 | 261 |
| cumul | 68 187 | 51 721 | 3 245 | 3 544 | 71 560 | 55 426 | -22,5 % | max 2 198 | max 2 045 |

Cumul du decodage : -24,1 %. Cumul de la publication : +9,2 %. Les ecarts de pic de quelques dizaines de Mio (film08 :
397 puis 452) sont dans le bruit de l'echantillonnage a 250 ms ; seul le pic de film04 (2 198 puis 2 045) est a retenir.

### 1.1 Ce que la table dit

- Les quatre films a liste non localisee (film04, film06, film09, film12) gagnent 27 a 32 % ; film07 (-11,6 %) et film08
  (-6,5 %) un peu.
- Les autres films (film01, film02, film03, film05, film10, film11, film13) varient de -0,2 a +5,5 % : voir 1.2.

### 1.2 Contre-mesure A/B sur les films non touches (reference recompilee, tours alternes)

Films 01, 03, 10, 13, 5 passes, deux tours (reference, tete, reference, tete), meme machine, meme heure. Dans chaque case :
tour 1 / tour 2.

| film | decodage ref | decodage tete | publication ref | publication tete | total ref | total tete |
|---|---|---|---|---|---|---|
| film01 | 2 138 / 2 134 | 2 110 / 2 109 | 250 / 255 | 269 / 269 | 2 404 / 2 407 | 2 396 / 2 395 |
| film03 | 1 349 / 1 349 | 1 339 / 1 312 | 125 / 124 | 132 / 133 | 1 484 / 1 482 | 1 481 / 1 455 |
| film10 | 1 505 / 1 481 | 1 477 / 1 474 | 125 / 127 | 140 / 137 | 1 640 / 1 620 | 1 627 / 1 636 |
| film13 | 1 124 / 1 157 | 1 134 / 1 142 | 94 / 91 | 100 / 100 | 1 226 / 1 259 | 1 244 / 1 250 |

Lecture : le DECODAGE est identique ou un peu plus court (ecart de -2,7 a +0,9 %) : les +2 a +5 % du releve sur les films
non touches etaient du bruit de la machine, pas un cout des trois optimisations. En revanche la PUBLICATION est plus lente
de facon reproductible, dans les deux tours et sur les quatre films : +6 a +12 % (de +6 a +19 ms ; +92 ms sur film04 au
releve complet). C'est la seule regression mesuree. Sa cause n'est pas etablie (section 10).

## 2. Methode des profils

Une passe par film sous `runtime/pprof` (profil CPU, puis profil du tas dans une passe separee, `-seulement filmNN`),
binaire de la tete. Le profil CPU ne ralentit presque pas la passe (film04 : 13,09 s profilee contre 13,07 au releve ;
film06 : 12,1 contre 12,1) : les durees sont donc comparables au releve, contrairement au premier profil. Les parts sont
des parts des echantillons CPU (le total depasse la duree murale de 0 a 9 % : le ramasse-miettes travaille en parallele).
Les noeuds de moins de 0,06 s sont abandonnes par le profileur ; un `-` dans une table veut dire « sous ce seuil ».
Le ramasse-miettes de film04 a de plus ete trace (`GODEBUG=gctrace=1`, une passe) pour la memoire.

## 3. Part de chaque etape (cumule, parts des echantillons CPU)

Les lignes `dont` sont contenues dans la ligne au-dessus ; les lignes « Bit a bit » traversent les etapes.

| etape | film01 | film06 | film09 | film04 | film12 |
|---|---|---|---|---|---|
| echantillons CPU (ms) | 2 570 | 12 390 | 6 660 | 15 320 | 2 970 |
| Marche des trames (`marcheurDesTrames.parcourir`) | 600 (23,3 %) | 1 700 (13,7 %) | 850 (12,8 %) | 4 780 (31,2 %) | 730 (24,6 %) |
| &nbsp;&nbsp;dont `localiserLaListe` (essais, candidats, boucles) | - | 1 260 (10,2 %) | 610 (9,2 %) | 2 140 (14,0 %) | 300 (10,1 %) |
| &nbsp;&nbsp;dont `lectureDEssai` | 220 (8,6 %) | 630 (5,1 %) | 220 (3,3 %) | 2 060 (13,4 %) | 100 (3,4 %) |
| &nbsp;&nbsp;&nbsp;&nbsp;dont `World.Snapshot` (copie de la table des slots) | 90 (3,5 %) | 550 (4,4 %) | 210 (3,2 %) | 1 330 (8,7 %) | 60 (2,0 %) |
| Marche des images-cles (`marcheDesImagesCles.parcourir`) | 990 (38,5 %) | 1 370 (11,1 %) | 930 (14,0 %) | 2 440 (15,9 %) | 970 (32,7 %) |
| &nbsp;&nbsp;dont recherche d'ancres (`kfRecherche.suivante`) | 380 (14,8 %) | 620 (5,0 %) | 380 (5,7 %) | 990 (6,5 %) | 220 (7,4 %) |
| &nbsp;&nbsp;dont table de datums (`tableDeDatumsDe`) | 220 (8,6 %) | 260 (2,1 %) | 180 (2,7 %) | 510 (3,3 %) | 250 (8,4 %) |
| &nbsp;&nbsp;dont mots par record (`motsParRecord`) | 250 (9,7 %) | 350 (2,8 %) | 260 (3,9 %) | 620 (4,0 %) | 300 (10,1 %) |
| Cloture du canal des lectures de bipedes (`Clore`) | 20 (0,8 %) | 2 480 (20,0 %) | 1 250 (18,8 %) | 660 (4,3 %) | 130 (4,4 %) |
| Balayage : positions (`balayerPositions`) | 510 (19,8 %) | 640 (5,2 %) | 440 (6,6 %) | 1 140 (7,4 %) | 510 (17,2 %) |
| Balayage : monde, vehicules, poses (`balayerMonde`) | 110 (4,3 %) | 5 610 (45,3 %) | 2 890 (43,4 %) | 1 910 (12,5 %) | 340 (11,4 %) |
| Balayage : pont, grenades, sieges (`balayerPont`) | 130 (5,1 %) | 550 (4,4 %) | 340 (5,1 %) | 450 (2,9 %) | 180 (6,1 %) |
| Balayage : equipement (`balayerCapacites`) | 50 (1,9 %) | - | - | 2 210 (14,4 %) | - |
| Publication (`assembler`) | 270 (10,5 %) | 190 (1,5 %) | 120 (1,8 %) | 1 150 (7,5 %) | 220 (7,4 %) |
| Bit a bit : ancrage des bipedes (`walkDeltaBipedPayloadJusqua`) | - | 3 350 (27,0 %) | 1 630 (24,5 %) | 780 (5,1 %) | 140 (4,7 %) |
| Bit a bit : ancres de bandes d'objets (`echantillonsDesBandes`) | - | 1 500 (12,1 %) | 750 (11,3 %) | 560 (3,7 %) | 60 (2,0 %) |
| Bit a bit : creations (`passeDesCreations.payload`) | - | 940 (7,6 %) | 490 (7,4 %) | 250 (1,6 %) | 40 (1,3 %) |
| Bit a bit : tirs et visee (`scanBipedAimRecordsJusqua`) | - | 1 220 (9,8 %) | 680 (10,2 %) | 340 (2,2 %) | 70 (2,4 %) |
| Bit a bit : recuperation d'equipement (`scanEquipRecoveryPacket`) | 40 (1,6 %) | - | - | 2 200 (14,4 %) | - |
| Lecture de bits, cumulee (`source.BitsAt`) | 220 (8,6 %) | 1 980 (16,0 %) | 1 100 (16,5 %) | 1 940 (12,7 %) | 400 (13,5 %) |
| GC : `gcBgMarkWorker` | 130 (5,1 %) | - | - | 730 (4,8 %) | 180 (6,1 %) |
| Allocation : `mallocgc` | 50 (1,9 %) | 190 (1,5 %) | 120 (1,8 %) | 690 (4,5 %) | 140 (4,7 %) |
| Volume alloue sur la passe (tas) | 3,4 Gio | 5,1 Gio | 2,8 Gio | 20,9 Gio | 3,4 Gio |
| dont `World.cloneSlots` | 0,29 Gio (8,7 %) | 2,07 Gio (40,5 %) | 1,01 Gio (35,9 %) | 6,16 Gio (29,5 %) | 0,25 Gio (7,5 %) |
| dont `replay.indexBySlot` (publication) | 0,54 Gio (16,1 %) | 0,51 Gio (10,1 %) | 0,29 Gio (10,4 %) | 3,15 Gio (15,1 %) | 0,60 Gio (17,8 %) |

Les cinq boucles qui testent un en-tete a CHAQUE position de bit (ancrage des bipedes, ancres de bandes, creations, visee,
recuperation d'equipement) pesent, en cumule non recouvrant : film06 7,0 s sur 12,4 (57 %), film09 3,6 sur 6,7 (53 %),
film04 4,1 sur 15,3 (27 %), film12 0,3 sur 3,0 (10 %), film01 ~0.

## 4. Ce qui a disparu, ce qui reste, ce qui domine

Comparaison avec le premier profil (echantillons CPU ; le premier profil a ete pris a 20,6 s d'echantillons pour film06,
celui-ci a 12,4 s) :

| fonction | film06 avant | film06 apres | film04 avant | film04 apres |
|---|---|---|---|---|
| `ScanMarcheDesTramesAvec` | 11,37 s | 5,07 s | 14,60 s | 6,98 s |
| `marcheurDesTrames.parcourir` | 7,96 s | 1,70 s | 12,29 s | 4,78 s |
| `LocaliserBoucleDeRecords` | 6,71 s | 0,27 s | 7,31 s | 0,34 s |
| `recupererLaListe` | 4,84 s | 0,19 s | 4,87 s | 0,20 s |
| `TryDeltaAt` (decodage d'un delta d'essai) | 5,92 s | sous 0,06 s | 6,70 s | sous 0,06 s |
| `marchLocateStrict` / `marchSignature123` | 4,85 s / 4,60 s | 0,27 s / disparu | 5,08 s / 4,93 s | 0,33 s / disparu |
| `debutDeLaVueBDeCuisson` | 2,82 s | 1,26 s | 5,24 s | 2,77 s |
| `walkDeltaBipedPayloadJusqua` | 4,03 s | 3,35 s | - | 0,78 s |
| `balayerMonde` | 6,50 s | 5,61 s | 2,37 s | 1,91 s |
| `Clore` (canal des lectures de bipedes) | 2,51 s | 2,48 s | - | 0,66 s |
| `scanEquipRecoveryPacket` | - | - | 2,57 s | 2,20 s |
| `World.Snapshot` | - | 0,55 s | 1,73 s | 1,33 s |

Disparu : la localisation a tatons de la liste (de 32 % a 2 % du total), sa relecture par les canaux, le decodage d'essai
de la signature du slot 123 (`TryDeltaAt`, `marchSignature123`, `decodeDelta` sur ce chemin). Reste a peu pres inchange :
tout ce qui est bit a bit apres la marche (ancrage des bipedes, vehicules, equipement) et la cloture des lectures de bipedes.

Ce qui domine maintenant, en cinq lignes :

1. Les cinq boucles d'en-tete a chaque position de bit (ancrage des bipedes, ancres de bandes d'objets, creations, visee,
   recuperation d'equipement) : 57 % de film06, 53 % de film09, 27 % de film04, 10 % de film12 ; elles ne pesent rien sur film01.
2. La copie de la table des slots a chaque lecture d'essai (`World.Snapshot` / `cloneSlots`) : 2 a 9 % du temps, mais 30 a
   40 % des octets alloues de film04, film06 et film09.
3. La marche des images-cles (recherche d'ancres, table de datums, `motsParRecord`) : 33 a 38 % de film01 et film12,
   11 a 16 % des autres films.
4. Les balayages d'apres la marche, qui tournent a la file sur un seul fil : monde 43 a 45 % de film06 et film09,
   equipement 14 % de film04.
5. La publication, devenue +6 a +12 % plus lente qu'en v1.5.0, et qui tient le pic memoire (section 6).


## 5. Ce qu'il reste a gagner : vitesse

Classe par rapport gain sur risque, du meilleur au moins bon. Les gains sont des ESTIMATIONS faites sur le profil
(part du temps total de chaque film), pas des mesures : aucun code n'a ete ecrit. Ils ne s'additionnent pas toujours.
Preuve d'equivalence disponible pour tous : empreintes du document et des faits des 13 films (section 1), identiques
a la v1.5.0 a ce jour.

| rang | candidat | lieu | gain estime sur le total par film | risque |
|---|---|---|---|---|
| 1 | **H1** : calculer `lay.TotalBits()` une fois par payload, pas a chaque bit candidat | `offline_biped.go:319` (`matchBipedHeader`, evalue AVANT toute verification) ; l'appelant `delta_biped_walk.go:96` l'a deja | film06 -3,0 % (0,37 s), film09 -3,5 % (0,23 s), film04 -0,5 % ; film01 et film12 rien | quasi nul (valeur constante, fonction pure) |
| 2 | **G8'** (remplace G8) : test d'appartenance de la fenetre glissante de 32 bits sans table de hachage (pre-filtre par masque de bits, puis la table seulement sur les rares positifs) | `keyframe_loadout.go:191` (`if !jeu[w]`, une recherche en table PAR BIT de chaque payload d'image-cle ; `motsParRecord` l.164-203, 85 % de son cout en `mapaccess1_fast32`) | film12 -7 a -9 % (0,25 s), film01 -6 a -8 % (0,16 s), film04 -3 % (0,53 s), film09 -3 %, film06 -2 % | faible (test d'appartenance pur, meme resultat) |
| 3 | **G5** : ne plus copier la table des slots a chaque essai (copie a la premiere ecriture, ou journal d'annulation) | `world.go:403-407` (`cloneSlots`), `world.go:420-424` (`Snapshot`, `Restore`) ; appelants `debut_de_liste.go:231-233` (`lectureDEssai`), `canal_des_morts.go:201`, `calibration_de_la_marche.go:68-69` | film04 -6 a -8 % (1,33 s) et -6,2 Gio alloues (-29 %) ; film06 -3 a -4 % (0,55 s) et -2,1 Gio (-40 %) ; film09 -2 a -3 % et -1,0 Gio ; film01 -2 a -3 % ; film12 -1,5 % | faible a moyen : les ecritures de la table sont toutes dans `world.go` (9 sites : l.190-193, 216-221, 274-279, 310-314, 319-324, 334-339, 349, 353-362, 384-389) et les lectures ailleurs ne font que lire (`frame_infer.go:249`, `marche_trames_entites.go:15`) ; l'invariant a tenir : `Restore` rend EXACTEMENT la table de `Snapshot`, y compris `Pos`, `Liaison`, `Soft` |
| 4 | **G4** : avancer par mot dans les balayages bit a bit (sauter les positions dont le bit de marque est nul, lire l'en-tete en une seule lecture de 24 bits au lieu de quatre appels) | `delta_biped_walk.go:98-106` (boucle `p++`), `offline_biped.go:333-361` (`matchBipedHeaderRaw` : marque `BitsStricts(pay,p,1)`, puis `LireHandleDelta`, bande de slots, deux bits nuls en `p+16`, masque) ; meme test d'en-tete ligne a ligne dans `offline_aim_only.go:228-236`, `pistes_du_monde.go:189-210`, `creations_du_monde.go:229`, `equipment_recovery.go:210` | marcheur d'ancrage SEUL : film06 -9 a -12 %, film09 -9 a -11 %, film04 -2 %, film12 -2 % ; avec les quatre autres boucles : film06 -17 a -25 %, film09 -16 a -23 %, film04 -8 a -11 %, film12 -3 a -4 %, film01 rien | moyen : `Accepte` et `Has` sont purs (verifie : `generations_vivantes.go:178-186`), donc l'ordre des tests est libre ; l'avance doit garder `p+minRecord <= total`, `p < fin` et le pas de non-chevauchement `p = i0 + i0Bits` (l.106) |
| 5 | **G9** (nouveau) : calculer a l'avance, sur plusieurs fils, la marche d'ancres de chaque payload d'image-cle | `keyframe_world_marche.go:81-104` : la marche d'un payload est une fonction pure des octets et de la preuve du film, deja memoisee sous verrou (`mu`, `parPayload`), et rend des copies ; la recherche (`kfRecherche.suivante`) pese 5 a 15 % | a 4 fils, rendement 80 % : film01 -11 %, film12 -5,5 %, film04 -5 %, film06 -4 %, film09 -4 % ; en y ajoutant la table de datums (`tableDeDatumsDe`, memo `map[*byte]` non verrouillee, `recuperations.go:24`, a verrouiller) : film01 et film12 -5 % de plus | faible a moyen : memo deja verrouillee ; un meme payload peut etre calcule deux fois (aucune erreur, meme resultat) ; l'ordre des consommateurs ne change pas |
| 6 | **G3** : balayages d'apres la marche en parallele | voir section 8 | film06 -21 a -25 % a 2 fils, jusqu'a -36 % a 4 fils ideaux ; film09 -24 % / -37 % ; film04 -17 % / -18 % ; film12 -7 % ; film01 -5 % | moyen a eleve : le contexte de film est mute par les balayages (voir section 8) |
| 7 | V2 : plusieurs morceaux a la fois | voir section 7 | borne a 10-15 % reel sur film04, 5-8 % ailleurs | eleve : a abandonner |

Observations sur la table :

- **G8 d'origine (index de slots par tableau)** : abandonne. Le gros du `mapaccess1_fast32` n'est plus la table des
  slots : il est dans `motsParRecord` (G8') ; le reste vient de la copie de `cloneSlots` (qui disparait avec G5).
  Hors ces deux sources, la table des slots ne pese que 0,1 s sur film04.
- **G7 (equipement)** : retire, a ne pas rouvrir seul. `scanEquipRecoveryPacket` pese toujours 2,20 s (14,4 %) de film04 et
  rien ailleurs ; la boucle d'en-tete de G4 est la meme famille (`equipment_recovery.go:210`), mais un gain limite a un
  seul film ne justifie pas un chantier a part : a prendre dans G4 si G4 est fait.
- **Marche des images-cles, recherche d'ancres** : le moteur de recherche est deja a huit positions par pas et sans branche
  (`internal/source/motif64.go:47-68`) ; rien a gagner de ce cote, seulement a la repartir (G9).
- **Publication +6 a +11 %** : a comprendre avant de choisir. Elle ne coute que 10 a 100 ms mais elle est reproductible
  (section 1.2). Ce n'est pas un candidat de gain, c'est une regression a expliquer (cause non etablie : section 10).

Gain combine plausible apres H1, G8', G5 et G4 (estimation, meme film, non additif strict) : film06 -25 a -32 %,
film09 -22 a -30 %, film04 -12 a -18 %, film12 -10 a -14 %, film01 -8 a -11 %.

## 6. Ce qu'il reste a gagner : memoire (film04 et le pic de 2,05 Gio)

### 6.1 Ce qui tient le pic

`GODEBUG=gctrace=1` sur une passe de film04 (cycle de ramasse-miettes, tas en Mio : debut -> fin -> vivant) :

| instant de la passe mesuree | tas au debut du cycle | vivant apres le cycle |
|---|---|---|
| mi-decodage | 500 a 650 | 250 a 330 |
| fin du decodage (cycle 145, a 42,0 s) | 1 057 | 587 |
| publication (cycle 151, a 42,7 s) | 1 178 | 554 |
| fin de la publication (cycle 157, a 43,1 s) | 1 619 | 834 |

Le pic du processus (2 045 Mio d'empreinte au releve) est donc atteint dans les 0,4 dernieres secondes de la
PUBLICATION : le vivant monte de 554 a 834 Mio (+280 Mio), et le ramasse-miettes, au reglage par defaut, laisse le tas
croitre jusqu'a environ le double du vivant (1 631 Mio de tas, plus le reste de l'empreinte du processus).

Ce que le vivant contient (profil du tas pris au plus haut tas echantillonne, 585 Mio vivants a ce moment) :

| structure vivante | Mio | lieu |
|---|---|---|
| suite des positions de bipedes lue (1,32 M positions de 176 octets) | 221 | `positions_lues.go:168-190` (`lireLesPositions`, deja pre-dimensionnee) |
| octets du film (morceaux decompresses) | 87 | `os.ReadFile` du chargement |
| records lus par paquet (`positionsBipedes.noter` 40, `ajouterLePaquet` 37) | 77 | `positions_lues.go:60-65`, `offline_biped.go` |
| vitesses d'etats de mouvement | 32 | `(*movementStateScanner).vitesse` |
| `indexBySlot` (a ce moment, une partie) | 31 | `replay/shots.go:174` |
| le reste (canaux, rejets, projectiles...) | ~137 | divers |

Les +280 Mio de la fin de publication correspondent, par la taille, a UNE copie complete des positions par
`indexBySlot` (`replay/shots.go:174-189` : `m[p.Slot] = append(m[p.Slot], p)` recopie chaque position de 176 octets,
soit 232 Mio pour 1,32 M positions, plus l'ecart d'agrandissement) : il est appele 8 fois pendant la publication
(`death_context.go:199`, `film_inputs.go:290`, `grenades.go:100`, `identity_registry_pont.go:30`, `killpos.go:241`,
`killpos_a_l_instant.go:135`, `shots.go:94`, `tracks_publication.go:217`), pour 3,15 Gio alloues sur film04, et
`vehiclePositionsBySlot` (`vehicle_tracks.go:245`) recopie de meme (1,68 Gio). CETTE ATTRIBUTION EST DEDUITE (tailles et
instants), pas lue sur un profil pris a l'instant exact du pic : le profil du tas ne montre que l'etat du dernier cycle.

### 6.2 Leviers, classes

| rang | levier | gain estime | sortie | risque |
|---|---|---|---|---|
| M0 | **Plafond souple plus bas que le plafond dur.** Deja mesure : film04, 3 passes, plafond de 2 Gio : pic 2 021 Mio, total 14 270 ms ; plafond de 1 Gio : pic **1 161 Mio**, total 14 461 ms (+1,3 %). `-plafond-gio` regle la sentinelle (`memguard/memguard.go:140-165` : plafond souple `debug.SetMemoryLimit` et plafond dur echantillonne), sans que le plafond dur ait arrete la passe a 1 Gio | pic -0,85 Gio sur film04 pour +1,3 % de temps | inchangee | faible : reglage de l'hote, pas du decodeur. Limite : un film plus gros que film04 qui garderait un vivant proche du plafond souple ferait travailler le ramasse-miettes en continu (non mesure) |
| M1 | `indexBySlot` et `vehiclePositionsBySlot` sans copier les positions : index de positions (4 octets par position) au lieu d'une copie de 176 octets ; 37 usages de `.pts` dans 11 fichiers de `replay/` (`closures.go`, `tracks_publication.go`, `shots.go`...) | vivant -230 Mio en publication, donc pic -0,4 a -0,5 Gio ; alloue -4,8 Gio (`indexBySlot` + `vehiclePositionsBySlot`) sur film04 | inchangee | moyen (refonte mecanique d'un type partage ; empreintes des 13 films comme preuve) |
| M1' | variante pauvre : construire `indexBySlot` UNE fois par publication et le partager (8 appels -> 1) | alloue -2,4 Gio sur film04, CPU -1 % ; pic inchange | inchangee | faible, mais aliasing a verifier (les appelants ne doivent pas trier le resultat en place) |
| M2 | liberer avant la publication ce que la publication ne relit pas : octets du film (87 Mio), records par paquet (77 Mio), vitesses (32 Mio) | vivant -100 a -200 Mio, pic -0,2 a -0,4 Gio | inchangee | moyen : il faut prouver que rien ne les relit ; non verifie |
| M3 | `BipedPosition` (176 octets, `offline_biped.go:65-91`) : reordonner les champs (rangs de 8 octets, booleen en queue) | -20 a -60 Mio de vivant | inchangee | faible, gain mineur |

La premiere ligne seule suffit a rendre de la marge pour G3 ou G9 ; M1 la rend sans toucher au reglage.

## 7. Plusieurs morceaux a la fois (V2) : verdict

**Verdict : impossible sans changer la sortie ; possible seulement par speculation avec raccord et re-decodage ; non
recommande.** Argument, ligne par ligne :

1. UN seul monde pour tout le film : `marche_trames.go:122` (`monde: NewWorld(reg)`), table anticipee posee une fois
   (`:129`), boucle sur les morceaux (`:135-154`). Aucun etat n'est remis a zero entre deux morceaux : le morceau
   suivant part de ce que le precedent a laisse.
2. L'image-cle d'un morceau ne retablit pas tout l'etat. Elle est jouee a l'entree du morceau
   (`marche_trames.go:143`, `lierLesImagesClesDuChunk`, `keyframe_liaison.go:129-155`). Ce qu'elle fait :
   - OUBLIE toute liaison dont le slot n'est porte ni par la chaine de ses records, ni par sa table de datums, ni par
     un candidat ecarte (`keyframe_liaison.go:134-147`, `world.go:353-362`) : cette partie de l'etat est donc
     independante du morceau precedent ;
   - REECRIT en entier la liaison de chaque record de la chaine (`world.go:274-279`, `BindImageCle` : `Pos`, `Liaison`,
     `Soft`, `Vue` remis) : independante aussi ;
   - mais pour un slot present SEULEMENT dans la table de datums et deja lie sous le meme archetype, `lierLesDatums`
     NE FAIT RIEN (`keyframe_datums.go:291-301`, test `ArchetypeForEID` puis `continue`) : la liaison du morceau
     precedent survit avec sa provenance (`Liaison` = lue sur un NEW, `Soft` faux, `Vue`, `Pos`/`PosValid`), alors qu'un
     monde neuf lui aurait donne `LiaisonDatum`, `Soft` vrai (`world.go:310-314`) ;
   - les NEW refuses du morceau precedent attendent leur verdict a la PREMIERE image-cle du morceau suivant
     (`keyframe_liaison.go:144-146`, `neufsRefuses`) : un etat qui traverse la frontiere.
3. Cet etat residuel est PUBLIE : la provenance de chaque record sort dans la structure de lecture
   (`frame_records.go:328` `rec.Liaison = w.liaisonDu(slot)`, `marche_trames_entites.go:15-20`), et `Soft` decide de
   `HardBound` (`world.go:343-346`), qui est l'ancre de confirmation de l'inference de chaine (`frame_infer.go`). Un
   morceau decode depuis un monde vide differerait donc de ce que la marche en suite produit, pour tout slot vivant a
   la frontiere, absent de la chaine de records de l'image-cle et ne de nouveau en cours de morceau.
4. Autres etats qui traversent les morceaux : `nsImageCle`, rang de vue memorise a la PREMIERE liaison du film
   (`world.go:292-299`) ; `anticipationDite`, une fois par monde (`world.go:139-146`) ; les compteurs et crochets de
   l'observation `cfg.Obs`, qui publient PENDANT la marche dans l'ordre du flux (`marche_trames.go:24-25`) ; l'etat
   cumule des canaux (morts, vitalites, equipement...) que `Distribuer` nourrit trame par trame.
5. Une passe de raccord existerait en principe : decoder chaque morceau depuis un etat suppose (les slots de la table de
   datums vus comme `LiaisonDatum`), puis, quand le morceau precedent est fini, comparer l'etat reel a l'etat suppose aux
   slots de la frontiere et RE-DECODER le morceau si l'un differe. Ce que je ne peux pas etablir : la frequence de la
   difference. Tout slot ne au milieu du morceau precedent, vivant a la frontiere et absent de la chaine de records de
   l'image-cle, la provoque : c'est vraisemblablement le cas de la plupart des morceaux (non verifie, il faut l'essai que
   la consigne interdit). Si elle est frequente, le raccord re-decode presque tout et le gain tombe a zero.
6. Le gain borne : la marche des trames pese 13 a 31 % du CPU (film04 31 %), et ce chiffre contient deja le travail des
   canaux, qui restent en serie. A 4 fils, parfait, ce serait -23 % sur film04 ; apres raccord et sans les canaux, au mieux
   10 a 15 % sur film04 et 5 a 8 % ailleurs. Memoire : un monde et une arene de paquet par fil.

Les gains de G5, G4 et G8' agissent sur la meme marche, sans ce risque.

## 8. Balayages d'apres la marche en parallele (G3) : verdict

**Verdict : possible, rentable sur film06, film09 et film04, mais PAS tel quel : il faut d'abord sortir de la
structure partagee, ce qui en fait un chantier de risque moyen a eleve.** A faire apres H1, G8', G5, G4, pas avant.

### 8.1 Les balayages, dans l'ordre (`replay/build_from_film.go:307-315`, `scanFilmInputs`)

| balayage | fichier | lit | ecrit | depend de | part (film06 / film04) |
|---|---|---|---|---|---|
| marche des trames (`lireLaMarcheDesTrames`) | `film_scan_mouvement.go` | films, memos du contexte | `s.marche`, memos `recup.*` | rien | 40,9 % / 45,6 % (serie, avant tout) |
| positions (`balayerPositions`) : pont d'identite, translocations, coups de recul, positions, creations, tirs, images-cles, naissances | `film_scan.go:28-110` | memos de la marche | `s.pont`, `s.etats`, `s.in.Positions`... | la marche | 5,2 % / 7,4 % |
| monde (`balayerMonde`) : lunette, poses, evenements d'apparition, socles, vehicules (nuage + visees), calques gardes | `film_scan.go:341-380` | memos, largeurs MPP de la calibration des poses | `s.in.Placements`, `Pads`, `Vehicles`... | positions (images-cles), calibration des poses | 45,3 % / 12,5 % |
| etats de mouvement (`balayerEtatsDeMouvement`) | `film_scan_mouvement.go` | `s.marche` (deja lu) | `s.in.MovementStates`... | monde (`Vehicles` pour les morts d'objet) | < 1 % |
| portage (`balayerPortage`, `balayerInventaire`) | `film_scan.go:127-225` | loadouts, creations | `s.in.WeaponChanges`, `Pickups`, `Inventory...` | positions | < 1 % |
| equipement (`balayerCapacites`) : rangs, changements, equipement porte, camouflage, grappin, impulsions, charges | `film_scan.go:228-340` | positions BRUTES (`birthOfLives(s.in.Positions)`) | `s.in.AbilityRanks`, `EquipmentChanges`... | positions | 0 % / 14,4 % |
| pont (`balayerPont`) : grenades, projectiles, detonations, fil des morts, sieges, equipes | `film_scan_pont.go:96-` | `s.pont` | `s.in.Grenades`, `Projectiles`, `Deaths`... | positions (pont) | 4,4 % / 2,9 % |

A l'interieur de `balayerMonde`, trois blocs sont eux-memes independants : poses (`decodeFilmPlacements`, 2,55 s sur
film06 : pistes 1,54 + creations 0,95), nuage des vehicules (1,80 s) et visees des occupants (1,24 s, `build_vehicles.go:128-136`) ;
les socles (`Pads`) et les vehicules ne veulent que les largeurs MPP, qui sortent de la calibration des poses (moins de 1 %).

### 8.2 Etat partage, et pourquoi ils ne sont pas independants EN L'ETAT

- **Le contexte de film (`FilmContext`) est muté par les balayages.** `PoserMPP` / `PoserProfilDeBalayage` posent des
  valeurs sur le contexte puis les restaurent : `equipment_placements.go:155,176,180`, `equipment_creation_width.go:262`,
  `build_ground_weapons.go:164`, `marche_images_cles.go:113-114`, `killsource/decode.go:398-401`. Deux balayages en
  parallele s'ecraseraient les largeurs (`c.bal.MPP`, `film_context.go:192-197`) que tout lecteur de bits lit ensuite.
- **Memos paresseux sans verrou** : `film_context.go:109-161` ne porte aucun `sync` ; `c.recup` (ancres, pistes,
  creations, objets, lectures, equipement porte, tables : `recuperations.go:13-25`, la table de datums est une
  `map[*byte]...` ecrite au premier appel), `c.vies`, `c.chunks`, `c.slots`, `c.reg`, `c.obs`. Une ecriture simultanee
  par deux balayages est une course. Seule la memoire des marches d'ancres est verrouillee
  (`keyframe_world_marche.go:56`).
- **Observation** : `FilmContext.obs` recoit les crochets que chaque balayage installe puis retire (27 appels
  `Observation()` dans `grammar` et `replay`).
- **Ordre publie** : chaque balayage appelle `s.opt.observe(...)` (etapes nommees, horloge `stepClock` partagee,
  `s.opt.clock`) ; le garde-rail `observe_test.go` reconstitue la liste ORDONNEE des etapes ; les journaux `slog` suivent
  le meme ordre.
- **Independants une fois ces quatre points traites** : monde (ses trois blocs), equipement, pont, portage ecrivent des
  champs differents de `s.in` ; les compteurs de replis (`fallback/compteur.go:44`) ont deja un verrou.
- **Dependances de donnees a respecter** : positions avant tout le reste ; calibration des poses avant socles et vehicules ;
  monde avant etats de mouvement.

### 8.3 Gain estime (duree murale du decodage ; hypothese : la marche et les positions restent en serie)

Rendement de 70 a 80 % retenu pour des balayages qui lisent la meme memoire et partagent le ramasse-miettes.

| film | cout des balayages en serie | a 2 fils | a 4 fils |
|---|---|---|---|
| film06 | positions 0,64 + monde 5,61 + pont 0,55 = 6,8 s | monde en deux blocs (poses 2,55 / vehicules 3,04, pont sur le plus leger) : 3,7 s, soit **-3,1 s (-21 a -25 %)** | poses en deux (pistes / creations), nuage et visees separes : 2,4 s, soit **-4,4 s ideaux (-30 a -36 %)**, -3,5 s avec le rendement |
| film09 | 0,44 + 2,89 + 0,34 = 3,7 s | 2,1 s : **-1,5 s (-20 a -24 %)** | -2,4 s ideaux (-30 a -37 %) |
| film04 | 1,14 + 1,91 + 2,21 + 0,45 = 5,7 s | equipement a part (2,21) contre monde + pont (2,36) : 3,5 s, **-2,2 s (-15 a -17 %)** | borne par l'equipement seul (2,21 s, une seule boucle) : -2,35 s (-17 a -18 %), pas mieux |
| film12 | 0,51 + 0,34 + 0,18 = 1,0 s | -0,2 s (-6 a -7 %) | idem |
| film01 | 0,51 + 0,11 + 0,13 + 0,05 = 0,8 s | -0,15 s (-5 a -7 %) | idem |

### 8.4 Surcout de memoire au pic

Estimation, non mesuree. Le pic de film04 est atteint en PUBLICATION (section 6.1), apres tous les balayages : le tas
y arrive a 1,63 Gio, alors qu'il plafonne a environ 1,06-1,2 Gio pendant les balayages (cycles 61 a 67 et 139 a 145 de la
trace). Faire tourner 2 ou 4 balayages ensemble ajoute leurs structures de travail simultanees (nuage de vehicules,
pistes, creations, fenetres d'equipement : quelques dizaines a quelques centaines de Mio sur film04, car 21 Gio sont alloues
en 13 s sans que le vivant depasse 590 Mio), mais cet ajout se loge sous le pic de la publication tant que le vivant ajoute reste sous
230 Mio environ (le ramasse-miettes double l'ajout, et la marge entre le tas de la publication, 1,63 Gio, et celui des
balayages, 1,2 Gio, est de 450 Mio). Attendu : pic inchange a +150 Mio sur film04 ; a MESURER avant de s'y fier, car si le surcout depassait
la marge, la fin de decodage passerait au-dessus de la publication. Les autres films (< 1 Gio de pic) n'ont pas de
probleme. M0 ou M1 (section 6.2) ramenent le pic assez bas pour que G3 y soit sans enjeu.

## 9. Recommandation

**A faire, dans cet ordre (une etape = un releve + empreintes identiques) :**

1. H1 (une ligne, risque nul), G8' (risque faible), G5 (risque faible a moyen) : ensemble, -8 a -15 % sur les films a
   marche chargee (film04, film12, film01), et G5 retire jusqu'a 40 % des octets alloues.
2. G4 : le plus gros gain restant (film06 et film09 : -9 a -12 % avec le seul marcheur d'ancrage, jusqu'a -25 % avec les
   quatre autres boucles de la meme famille). A faire en un helper commun d'en-tete avec son garde-rail (regle des trois
   copies : cinq boucles ont aujourd'hui le meme test).
3. M0 si le plafond de production doit rester a 3 Gio avec de la marge pour la suite : reglage de l'hote, +1,3 % de temps.
   M1 ensuite si on veut descendre le pic sans compter sur le ramasse-miettes.
4. Re-profiler. Puis G9 (faible a moyen) si la recherche d'ancres ressort au-dessus de 5 % sur au moins trois films.
5. G3 seulement ensuite, et seulement si film06, film09 et film04 comptent encore dans l'objectif de vitesse : -15 a
   -25 % pour un chantier qui reecrit le partage du contexte de film.

**A abandonner :** V2 (plusieurs morceaux a la fois : gain borne, sortie non garantie, raccord incertain) ; G8 d'origine
(table des slots par tableau : la table n'est plus le cout) ; G7 isole ; V1, V3.2 (deja actes).

**A ouvrir a part, sans le mettre dans les gains :** la regression de la publication (+6 a +11 %).

**Quand le gain ne vaut plus le risque.** Regle proposee : au-dessous de 5 % du total sur au moins trois des cinq films
profiles, un candidat de risque moyen ou plus ne se fait pas. Appliquee ici : H1, G8', G5, G4 passent ; G9 est a la limite
(4 a 5,5 % sur quatre films, 11 % sur film01) ; G3 passe sur trois films mais pour un risque eleve, donc en dernier ; V2
ne passe pas ; tout ce qui reste apres eux (candidatsDeTete 3 %, `consumeByName` 5 % sur film04 seul, publication
7,5 % sur film04 seul, GC 5 %) est sous la barre ou borne a un film. Apres H1, G8', G5, G4, le temps de film04 serait
d'environ 11,5 a 12,5 s et celui de film06 de 8,5 a 9,5 s : en dessous, chaque pas coute plus qu'il ne rend.

## 10. Ce que je n'ai pas pu etablir

- **La cause de la regression de la publication (+6 a +11 %).** Mesuree et reproduite (section 1.2), pas expliquee. Les
  filtres de positions en place (G6) ou la taille de la suite pre-dimensionnee sont des pistes, sans preuve ; la publication
  n'a pas ete profilee par ce travail a la reference.
- **L'independance des morceaux (V2)** : verdict par lecture, pas par essai. La frequence des slots « table seule, deja lies »
  a la frontiere d'un morceau n'est pas mesuree.
- **Les gains de la section 5** sont des estimations sur profil : aucun essai de code. Celui de G4 est le moins sur (il
  depend de la part des candidats rejetes a la premiere verification, que je n'ai pas comptee).
- **Le gain reel de G3 et son surcout de memoire** : estimes (section 8) ; le rendement a 2 et 4 fils d'un balayage
  memoire-lie n'est pas mesure.
- **Le moment exact du pic de memoire et ce qui le compose** : deduit de la trace du ramasse-miettes et des tailles ; le
  profil du tas pris au plus haut tas echantillonne (585 Mio vivants) n'est pas celui du pic de 2 045 Mio.
- **M0** : mesure sur UN film, 3 passes par plafond ; son effet sur un film plus gros que film04 n'est pas connu.
- **Le releve de la v1.5.0 a ete pris sous une charge que je ne connais pas** : l'A/B a quatre films le recoupe, pas les neuf
  autres.
- **La serialisation** du document et des faits n'est pas dans le banc.
- Les fichiers de profil portent un identifiant de construction contenant un chemin du poste : ils restent hors depot.

## Fichiers (hors depot)

`tete.tsv`, `tete.log` (releve complet) ; `ab_ref_N.tsv`, `ab_tete_N.tsv` (A/B) ; `filmNN.cpu.pprof`, `filmNN.cpu.txt`,
`filmNN.cpu.cum.txt`, `filmNN.cpu.flat.txt`, `filmNN.cpu.cumfull.txt`, `filmNN.tas.pprof`, `filmNN.tas.pprof.pic`,
`filmNN.tas.txt` ; `film04.gctrace.txt` ; `plafond2.tsv`, `plafond1.tsv` ; `echant.log`, `echant_releve.log` (processus
vus) ; `etapes.py`, `gen_tables.py`, `run_prof.sh`, `run_ab.sh`.
Lecture : `go tool pprof -top -cum filmNN.cpu.pprof` ; `go tool pprof -sample_index=alloc_space -top filmNN.tas.pprof`.

## Annexe A. Les 15 fonctions les plus couteuses, par film

### film01 (2570 ms d echantillons CPU)

Cumule (hors racines ; une chaine de fonctions de meme cout n est donnee qu une fois, par sa premiere) :

| fonction | cumule ms | part | etape |
|---|---|---|---|
| `internal/grammar.(*marcheDesImagesCles).parcourir` | 990 | 38.5 % | Marche des images-cles |
| `internal/grammar.distribuerLaDemande` | 770 | 30.0 % | Marche des images-cles |
| `internal/grammar.(*marcheurDesTrames).parcourir` | 600 | 23.3 % | Marche des trames |
| `replay.(*filmScan).balayerPositions` | 510 | 19.8 % | Balayage : positions |
| `internal/grammar.(*marcheurDesTrames).marcherLePaquet` | 470 | 18.3 % | Marche des trames |
| `internal/grammar.(*marcheDesImagesCles).marcherLePaquet` | 390 | 15.2 % | Marche des images-cles |
| `internal/grammar.decodeInferLoop` | 320 | 12.5 % | Marche des trames |
| `internal/grammar.distribuerLesImagesClesSeules` | 300 | 11.7 % | Marche des images-cles |
| `internal/grammar.(*canalDeLEtatCompletBipede).ImageCle` | 280 | 10.9 % | Marche des images-cles |
| `internal/grammar.(*canalDeLEtatCompletBipede).fenetresDerriereLaLecture` | 270 | 10.5 % | Marche des images-cles |
| `internal/grammar.motsParRecord` | 250 | 9.7 % | Marche des images-cles |
| `internal/grammar.(*FilmContext).tableDeDatumsDe` | 220 | 8.6 % | Marche des images-cles |
| `internal/grammar.ScanPontDIdentite` | 210 | 8.2 % | Balayage : pont |
| `internal/grammar.traverseComponentLoop` | 180 | 7.0 % | Marche des trames |
| `internal/grammar.traverseComponentLoopFrom` | 170 | 6.6 % | Marche des trames |

A plat :

| fonction | plat ms | part | etape |
|---|---|---|---|
| `internal/source.jugement.huit` | 320 | 12.5 % | Marche des images-cles |
| `internal/source.BitsAt` | 180 | 7.0 % | Lecture de bits (source) |
| `internal/source.ChercherMotifs64` | 150 | 5.8 % | Marche des images-cles |
| `internal/runtime/maps.ctrlGroup.matchH2` | 90 | 3.5 % | Tables de hachage (runtime) |
| `runtime.mapaccess1_fast32` | 90 | 3.5 % | Tables de hachage (runtime) |
| `internal/grammar.(*balayageDeTable).enteteA` | 70 | 2.7 % | Marche des images-cles |
| `internal/grammar.motsParRecord` | 60 | 2.3 % | Marche des images-cles |
| `runtime.mapassign_fast32` | 60 | 2.3 % | Tables de hachage (runtime) |
| `internal/source.(*Bits).ReadBits` | 50 | 1.9 % | Lecture de bits (source) |
| `encoding/binary.bigEndian.Uint64` | 40 | 1.6 % | Lecture de bits (source) |
| `internal/grammar.(*World).VuePossede` | 40 | 1.6 % | Autre |
| `internal/grammar.lireEntreeDeControle` | 40 | 1.6 % | Autre |
| `internal/source.blocDeHuit` | 40 | 1.6 % | Lecture de bits (source) |
| `runtime.lock2` | 40 | 1.6 % | GC et allocation |
| `aeshashbody` | 30 | 1.2 % | Tables de hachage (runtime) |

### film06 (12390 ms d echantillons CPU)

Cumule (hors racines ; une chaine de fonctions de meme cout n est donnee qu une fois, par sa premiere) :

| fonction | cumule ms | part | etape |
|---|---|---|---|
| `replay.(*filmScan).balayerMonde` | 5610 | 45.3 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.walkDeltaBipedPayloadJusqua` | 3350 | 27.0 % | Bit a bit (ancrage d en-tete) |
| `replay.decodeFilmVehicleScan` | 3040 | 24.5 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.ScanEquipmentPlacements` | 2550 | 20.6 % | Balayage : monde (vehicules, poses) |
| `internal/source.BitsAt` | 1980 | 16.0 % | Lecture de bits (source) |
| `internal/grammar.balayerLesPositions` | 1830 | 14.8 % | Balayage : positions |
| `internal/source.BitsTolerants` | 1750 | 14.1 % | Lecture de bits (source) |
| `internal/grammar.(*FilmContext).parcourirLesAncresBipedesBornees` | 1600 | 12.9 % | Cloture des lectures de bipedes |
| `internal/grammar.(*FilmContext).pistesDeLaBande` | 1540 | 12.4 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.(*marcheurDesTrames).marcherLePaquet` | 1470 | 11.9 % | Marche des trames |
| `internal/grammar.(*marcheDesImagesCles).parcourir` | 1370 | 11.1 % | Marche des images-cles |
| `internal/grammar.debutDeLaVueBDeCuisson` | 1260 | 10.2 % | Marche des trames |
| `internal/grammar.scanBipedAimRecordsJusqua` | 1220 | 9.8 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.matchBipedHeaderRaw` | 1170 | 9.4 % | Bit a bit (ancrage d en-tete) |
| `internal/grammar.distribuerLaDemande` | 1110 | 9.0 % | Marche des images-cles |

A plat :

| fonction | plat ms | part | etape |
|---|---|---|---|
| `internal/source.BitsAt` | 1710 | 13.8 % | Lecture de bits (source) |
| `internal/grammar.matchBipedHeader` | 1450 | 11.7 % | Bit a bit (ancrage d en-tete) |
| `internal/source.BitsTolerants` | 800 | 6.5 % | Lecture de bits (source) |
| `internal/grammar.(*passeDesCreations).payload` | 510 | 4.1 % | Balayage : monde (vehicules, poses) |
| `internal/source.jugement.huit` | 450 | 3.6 % | Marche des images-cles |
| `internal/grammar.echantillonsDesBandes` | 430 | 3.5 % | Balayage : monde (vehicules, poses) |
| `internal/profile.I0Layout.TotalBits` | 370 | 3.0 % | Bit a bit (ancrage d en-tete) |
| `internal/grammar.archetypeDeLEnTeteNEW` | 360 | 2.9 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.matchAimOnlyRecord` | 360 | 2.9 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.walkDeltaBipedPayloadJusqua` | 340 | 2.7 % | Bit a bit (ancrage d en-tete) |
| `encoding/binary.bigEndian.Uint64` | 290 | 2.3 % | Lecture de bits (source) |
| `internal/grammar.bipedCreationWalk.scanPayloadJusqua` | 270 | 2.2 % | Autre |
| `internal/grammar.LireHandleDelta` | 230 | 1.9 % | Bit a bit (ancrage d en-tete) |
| `internal/grammar.matchWorldObjectNewHeaderIn` | 230 | 1.9 % | Autre |
| `internal/grammar.matchBipedHeaderRaw` | 220 | 1.8 % | Bit a bit (ancrage d en-tete) |

### film09 (6660 ms d echantillons CPU)

Cumule (hors racines ; une chaine de fonctions de meme cout n est donnee qu une fois, par sa premiere) :

| fonction | cumule ms | part | etape |
|---|---|---|---|
| `replay.(*filmScan).balayerMonde` | 2890 | 43.4 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.walkDeltaBipedPayloadJusqua` | 1630 | 24.5 % | Bit a bit (ancrage d en-tete) |
| `replay.decodeFilmVehicleScan` | 1530 | 23.0 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.ScanEquipmentPlacements` | 1350 | 20.3 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.(*canalDesLecturesBipedes).Clore` | 1250 | 18.8 % | Cloture des lectures de bipedes |
| `internal/source.BitsAt` | 1100 | 16.5 % | Lecture de bits (source) |
| `internal/grammar.(*marcheDesImagesCles).parcourir` | 930 | 14.0 % | Marche des images-cles |
| `internal/source.BitsTolerants` | 880 | 13.2 % | Lecture de bits (source) |
| `internal/grammar.(*marcheurDesTrames).parcourir` | 850 | 12.8 % | Marche des trames |
| `internal/grammar.(*FilmContext).parcourirLesAncresBipedesBornees` | 800 | 12.0 % | Cloture des lectures de bipedes |
| `internal/grammar.(*FilmContext).pistesDeLaBande` | 760 | 11.4 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.(*marcheurDesTrames).marcherLePaquet` | 720 | 10.8 % | Marche des trames |
| `internal/grammar.ScanBipedAimOnly` | 680 | 10.2 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.LireHandleDelta` | 610 | 9.2 % | Bit a bit (ancrage d en-tete) |
| `internal/grammar.matchAimOnlyRecord` | 580 | 8.7 % | Balayage : monde (vehicules, poses) |

A plat :

| fonction | plat ms | part | etape |
|---|---|---|---|
| `internal/source.BitsAt` | 930 | 14.0 % | Lecture de bits (source) |
| `internal/grammar.matchBipedHeader` | 560 | 8.4 % | Bit a bit (ancrage d en-tete) |
| `internal/source.BitsTolerants` | 360 | 5.4 % | Lecture de bits (source) |
| `internal/source.jugement.huit` | 310 | 4.7 % | Marche des images-cles |
| `internal/grammar.(*passeDesCreations).payload` | 270 | 4.1 % | Balayage : monde (vehicules, poses) |
| `internal/profile.I0Layout.TotalBits` | 230 | 3.5 % | Bit a bit (ancrage d en-tete) |
| `internal/grammar.archetypeDeLEnTeteNEW` | 220 | 3.3 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.matchAimOnlyRecord` | 220 | 3.3 % | Balayage : monde (vehicules, poses) |
| `internal/source.BitsStricts` | 190 | 2.9 % | Lecture de bits (source) |
| `internal/grammar.matchBipedHeaderRaw` | 170 | 2.6 % | Bit a bit (ancrage d en-tete) |
| `encoding/binary.bigEndian.Uint64` | 160 | 2.4 % | Lecture de bits (source) |
| `internal/grammar.matchWorldObjectNewHeaderIn` | 150 | 2.3 % | Autre |
| `internal/grammar.LireHandle` | 140 | 2.1 % | Bit a bit (ancrage d en-tete) |
| `internal/grammar.bipedCreationWalk.scanPayloadJusqua` | 140 | 2.1 % | Autre |
| `internal/grammar.echantillonsDesBandes` | 120 | 1.8 % | Balayage : monde (vehicules, poses) |

### film04 (15320 ms d echantillons CPU)

Cumule (hors racines ; une chaine de fonctions de meme cout n est donnee qu une fois, par sa premiere) :

| fonction | cumule ms | part | etape |
|---|---|---|---|
| `internal/grammar.(*marcheurDesTrames).parcourir` | 4780 | 31.2 % | Marche des trames |
| `internal/grammar.(*marcheurDesTrames).marcherLePaquet` | 3950 | 25.8 % | Marche des trames |
| `internal/grammar.debutDeLaVueBDeCuisson` | 2770 | 18.1 % | Marche des trames |
| `internal/grammar.(*marcheDesImagesCles).parcourir` | 2440 | 15.9 % | Marche des images-cles |
| `replay.(*filmScan).balayerCapacites` | 2210 | 14.4 % | Balayage : equipement |
| `internal/grammar.localiserLaListe` | 2140 | 14.0 % | Marche des trames |
| `internal/grammar.lectureDEssai` | 2060 | 13.4 % | Marche des trames |
| `internal/source.BitsAt` | 1940 | 12.7 % | Lecture de bits (source) |
| `internal/grammar.lireTrameParRangs` | 1790 | 11.7 % | Marche des trames |
| `internal/grammar.decodeInferLoop` | 1680 | 11.0 % | Marche des trames |
| `internal/grammar.debutParFermetureRangee` | 1430 | 9.3 % | Marche des trames |
| `internal/grammar.(*World).Snapshot` | 1330 | 8.7 % | Marche des trames |
| `internal/grammar.traverseComponentLoop` | 1240 | 8.1 % | Marche des trames |
| `internal/grammar.decodeDelta` | 1190 | 7.8 % | Marche des trames |
| `replay.(*filmScan).assembler` | 1150 | 7.5 % | Publication |

A plat :

| fonction | plat ms | part | etape |
|---|---|---|---|
| `internal/source.BitsAt` | 1670 | 10.9 % | Lecture de bits (source) |
| `internal/source.jugement.huit` | 750 | 4.9 % | Marche des images-cles |
| `internal/grammar.scanEquipRecoveryPacket` | 480 | 3.1 % | Balayage : equipement |
| `internal/source.BitsTolerants` | 430 | 2.8 % | Lecture de bits (source) |
| `internal/grammar.matchBipedHeader` | 360 | 2.3 % | Bit a bit (ancrage d en-tete) |
| `runtime.memhash32` | 360 | 2.3 % | Tables de hachage (runtime) |
| `runtime.memmove` | 360 | 2.3 % | GC et allocation |
| `internal/runtime/maps.ctrlGroup.matchH2` | 350 | 2.3 % | Tables de hachage (runtime) |
| `internal/grammar.LireHandleDelta` | 280 | 1.8 % | Bit a bit (ancrage d en-tete) |
| `runtime.mapassign_fast32` | 280 | 1.8 % | Tables de hachage (runtime) |
| `internal/source.BitsStricts` | 270 | 1.8 % | Lecture de bits (source) |
| `runtime.mapaccess1_fast32` | 270 | 1.8 % | Tables de hachage (runtime) |
| `encoding/binary.bigEndian.Uint64` | 250 | 1.6 % | Lecture de bits (source) |
| `internal/grammar.LireHandle` | 230 | 1.5 % | Bit a bit (ancrage d en-tete) |
| `runtime.memclrNoHeapPointers` | 230 | 1.5 % | GC et allocation |

### film12 (2970 ms d echantillons CPU)

Cumule (hors racines ; une chaine de fonctions de meme cout n est donnee qu une fois, par sa premiere) :

| fonction | cumule ms | part | etape |
|---|---|---|---|
| `internal/grammar.(*marcheDesImagesCles).parcourir` | 970 | 32.7 % | Marche des images-cles |
| `internal/grammar.(*marcheurDesTrames).parcourir` | 730 | 24.6 % | Marche des trames |
| `internal/grammar.distribuerLaDemande` | 700 | 23.6 % | Marche des images-cles |
| `internal/grammar.distribuerLesImagesCles.func1` | 660 | 22.2 % | Marche des images-cles |
| `internal/grammar.(*marcheurDesTrames).marcherLePaquet` | 570 | 19.2 % | Marche des trames |
| `replay.(*filmScan).balayerPositions` | 510 | 17.2 % | Balayage : positions |
| `internal/source.BitsAt` | 400 | 13.5 % | Lecture de bits (source) |
| `internal/grammar.distribuerLesImagesClesSeules` | 360 | 12.1 % | Marche des images-cles |
| `replay.(*filmScan).balayerMonde` | 340 | 11.4 % | Balayage : monde (vehicules, poses) |
| `internal/grammar.ScanEtatsDesImagesCles` | 310 | 10.4 % | Marche des images-cles |
| `internal/grammar.(*canalDeLEtatCompletBipede).ImageCle` | 300 | 10.1 % | Marche des images-cles |
| `runtime.mapaccess1_fast32` | 280 | 9.4 % | Tables de hachage (runtime) |
| `internal/grammar.(*marcheDesImagesCles).marcherLePaquet` | 260 | 8.8 % | Marche des images-cles |
| `internal/grammar.(*FilmContext).tableDeDatumsDe` | 250 | 8.4 % | Marche des images-cles |
| `internal/grammar.(*FilmContext).BipedSlots` | 220 | 7.4 % | Marche des images-cles |

A plat :

| fonction | plat ms | part | etape |
|---|---|---|---|
| `internal/source.BitsAt` | 310 | 10.4 % | Lecture de bits (source) |
| `runtime.mapaccess1_fast32` | 200 | 6.7 % | Tables de hachage (runtime) |
| `internal/source.jugement.huit` | 190 | 6.4 % | Marche des images-cles |
| `encoding/binary.bigEndian.Uint64` | 100 | 3.4 % | Lecture de bits (source) |
| `internal/grammar.(*balayageDeTable).enteteA` | 70 | 2.4 % | Marche des images-cles |
| `internal/grammar.filmPacketsOf` | 70 | 2.4 % | Autre |
| `internal/source.(*Bits).ReadBits` | 60 | 2.0 % | Lecture de bits (source) |
| `runtime.mapassign_fast32` | 60 | 2.0 % | Tables de hachage (runtime) |
| `runtime.memclrNoHeapPointers` | 60 | 2.0 % | GC et allocation |
| `internal/source.ChercherMotifs64` | 50 | 1.7 % | Marche des images-cles |
| `runtime.memmove` | 50 | 1.7 % | GC et allocation |
| `runtime.nextFreeFast` | 50 | 1.7 % | GC et allocation |
| `aeshashbody` | 40 | 1.3 % | Tables de hachage (runtime) |
| `internal/grammar.matchBipedHeader` | 40 | 1.3 % | Bit a bit (ancrage d en-tete) |
| `internal/grammar.motsParRecord` | 40 | 1.3 % | Marche des images-cles |

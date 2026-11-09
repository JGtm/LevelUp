# Carte de fermeture des trames delta — 2026-10-03

Films mesures : 20 ; echecs : 0. Contexte d instrument (largeurs d axe lues dans le film, profil par defaut, sans calibration killsource).

Table ECS : `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/integ2/ecs_table_tete.tsv`.

## Par build

| Build | Films | Paquets fermes | Vue A | Vue B | Vue C | Records utiles fermes | Entrees de controle lues | Declencheur (>= 95 %) | Pic memoire max |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 22246/129369 (17,2 %) | 16776/79272 (21,2 %) | 22246/111673 (19,9 %) | 22246/26526 (83,9 %) | 298477/1567213 (19,0 %) | 348855 | non atteint | 308 Mio |
| HI_1_11_0 | 1 | 4147/16824 (24,6 %) | 2967/9073 (32,7 %) | 4147/15098 (27,5 %) | 4147/4630 (89,6 %) | 80067/289563 (27,7 %) | 74842 | non atteint | 118 Mio |
| HI_1_12_0 | 1 | 5879/21864 (26,9 %) | 5249/19134 (27,4 %) | 5879/21583 (27,2 %) | 5879/5942 (98,9 %) | 35492/122669 (28,9 %) | 30418 | non atteint | 76 Mio |
| HI_1_13_0 | 10 | 260393/335960 (77,5 %) | 222500/271063 (82,1 %) | 260393/320292 (81,3 %) | 260393/263207 (98,9 %) | 2355178/2811122 (83,8 %) | 1863008 | non atteint | 225 Mio |
| HI_1_4_1 | 1 | 692/11130 (6,2 %) | 651/6174 (10,5 %) | 692/9840 (7,0 %) | 692/3021 (22,9 %) | 78/184588 (0,0 %) | 0 | non atteint | 207 Mio |
| HI_1_8_0 | 1 | 13946/49696 (28,1 %) | 13146/43640 (30,1 %) | 13946/46437 (30,0 %) | 13946/15317 (91,0 %) | 83669/271810 (30,8 %) | 79784 | non atteint | 151 Mio |
| HI_1_9_0 | 1 | 5629/17629 (31,9 %) | 4417/10304 (42,9 %) | 5629/15991 (35,2 %) | 5629/5957 (94,5 %) | 65621/248938 (26,4 %) | 74235 | non atteint | 117 Mio |
| version-31 | 1 | 139/17919 (0,8 %) | 130/10140 (1,3 %) | 139/15662 (0,9 %) | 139/4700 (3,0 %) | 274/320306 (0,1 %) | 0 | non atteint | 155 Mio |
| version-33 | 1 | 424/28751 (1,5 %) | 402/14259 (2,8 %) | 424/24712 (1,7 %) | 424/4421 (9,6 %) | 3654/479794 (0,8 %) | 0 | non atteint | 226 Mio |

## Causes d arret (premiere cause de chaque paquet non ferme)

Gain potentiel = records utiles LUS et non fermes dans les paquets que la cause arrete : BORNE SUPERIEURE (une autre cause peut suivre ; les records d apres l arret ne sont pas lus du tout).

| Rang | Cause | Archetype | Index | Statut | Usage produit | Paquets bloques | Gain potentiel | Builds |
|---|---|---|---|---|---|---|---|---|
| 1 | vue B : sortie par rejet | - |  | - | - | 229840 | 3006116 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 2 | liste d evenements non localisee | - |  | - | - | 47854 | 0 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 3 | ti=43 device-animation-layer-state | 43 | i35 | non_porte | aucun | 12202 | 68132 | HI_1_10_0, HI_1_12_0, HI_1_8_0 |
| 4 | vue C : terminateur hors cadre | - |  | - | - | 11841 | 156240 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 5 | ecrivain : masque au-dela de l archetype | - |  | - | - | 4267 | 12368 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 6 | vue C : kind non porte | - |  | - | - | 2712 | 55053 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 7 | vue C : bloc 0xbc (desalignement) | - |  | - | - | 961 | 19763 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 8 | ti=48 forge-player-data-edited-objects-ids | 48 | i0 | non_porte | aucun | 951 | 8319 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 9 | ti=43 device-position-group | 43 | i21 | non_porte | aucun | 779 | 11820 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 10 | vue B : fin de payload | - |  | - | - | 707 | 4010 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 11 | ti=12 managed-navpoint-visual-state-groups-component-1 | 12 | i21 | non_porte | aucun | 412 | 5486 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 12 | ti=12 managed-navpoint-visual-state-groups-component-2 | 12 | i22 | non_porte | aucun | 401 | 5507 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-31, version-33 |
| 13 | ti=10 managed-object-navpoint | 10 | i10, i11, i12, i13, i15, i16, i17, i2, i3, i4, i5, i6, i7, i8 | non_porte | aucun | 350 | 4434 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 14 | ecrivain : ordre de la vue B | - |  | - | - | 256 | 1013 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 15 | ti=35 biped-spartan-ability-non-predicted-state | 35 | i58, i59 | partiel | rejeu : grappleLines[] (schema 8) — la ligne blanche du joueur vers son ancre, fenetre [t0,t1] par vie | 241 | 2013 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 16 | ti=56 archetype hors registre | 56 |  | - | - | 234 | 1170 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 17 | ecrivain : masque epars non croissant | - |  | - | - | 134 | 418 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 18 | ti=12 managed-navpoint-visual-state-groups-component-0 | 12 | i20 | non_porte | aucun | 116 | 1269 | HI_1_13_0, HI_1_4_1, version-33 |
| 19 | ti=12 managed-navpoint-override-flags | 12 | i16 | non_porte | aucun | 91 | 566 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 20 | ti=11 managed-objective-interaction-filter | 11 | i4 | non_porte | aucun | 90 | 722 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0 |
| 21 | ti=43 device-machine-flags | 43 | i39 | non_porte | aucun | 90 | 1324 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, version-31, version-33 |
| 22 | ti=18 effect-state-data | 18 | i0, i1, i10, i12, i13, i19, i2, i22, i23, i28, i3, i31, i4, i6, i7, i8, i9 | partiel | aucun | 65 | 261 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31 |
| 23 | ti=0 forge-engine-player-roles | 0 | i18 | non_porte | aucun | 60 | 410 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 24 | ti=10 managed-object-interaction-filter | 10 | i22 | non_porte | aucun | 55 | 762 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 25 | ti=51 archetype hors registre | 51 |  | - | - | 54 | 484 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
| 26 | ti=59 archetype hors registre | 59 |  | - | - | 53 | 439 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 27 | vue C : debordement | - |  | - | - | 47 | 199 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 28 | ti=43 device-position-animation-name | 43 | i19 | non_porte | aucun | 41 | 382 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-33 |
| 29 | ti=19 sound-placement-state-data | 19 | i0, i1, i18, i2, i25, i3, i31, i4, i6, i8, i9 | non_porte | aucun | 40 | 267 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
| 30 | ti=50 archetype hors registre | 50 |  | - | - | 40 | 178 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
## Carte v2 — « vue C : terminateur hors cadre » ventile (item 1.1)

Sortie de la vue B : comment la boucle de records s est arretee avant la vue C ; vue C : vide (son terminateur seul) ou non ; reste : bits du payload derriere le terminateur de la vue C.

### Par build

| Build | Films | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 773 | 773/773 (100,0 %) | 0/773 (0,0 %) | 0/773 (0,0 %) | 0/773 (0,0 %) | 564/773 (73,0 %) | 0/773 (0,0 %) | 0/773 (0,0 %) | 773/773 (100,0 %) | 0/773 (0,0 %) | 4106 |
| HI_1_11_0 | 1 | 291 | 291/291 (100,0 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 268/291 (92,1 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 291/291 (100,0 %) | 0/291 (0,0 %) | 1045 |
| HI_1_12_0 | 1 | 54 | 54/54 (100,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 49/54 (90,7 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 54/54 (100,0 %) | 0/54 (0,0 %) | 2 |
| HI_1_13_0 | 10 | 1334 | 1334/1334 (100,0 %) | 0/1334 (0,0 %) | 0/1334 (0,0 %) | 0/1334 (0,0 %) | 1173/1334 (87,9 %) | 0/1334 (0,0 %) | 7/1334 (0,5 %) | 1327/1334 (99,5 %) | 0/1334 (0,0 %) | 9019 |
| HI_1_4_1 | 1 | 2042 | 2042/2042 (100,0 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 169/2042 (8,3 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 2042/2042 (100,0 %) | 0/2042 (0,0 %) | 39302 |
| HI_1_8_0 | 1 | 1007 | 1007/1007 (100,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 899/1007 (89,3 %) | 0/1007 (0,0 %) | 96/1007 (9,5 %) | 911/1007 (90,5 %) | 0/1007 (0,0 %) | 5386 |
| HI_1_9_0 | 1 | 207 | 207/207 (100,0 %) | 0/207 (0,0 %) | 0/207 (0,0 %) | 0/207 (0,0 %) | 180/207 (87,0 %) | 0/207 (0,0 %) | 0/207 (0,0 %) | 207/207 (100,0 %) | 0/207 (0,0 %) | 1536 |
| version-31 | 1 | 3111 | 3111/3111 (100,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 94/3111 (3,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 3111/3111 (100,0 %) | 0/3111 (0,0 %) | 41137 |
| version-33 | 1 | 3022 | 3022/3022 (100,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 213/3022 (7,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 3022/3022 (100,0 %) | 0/3022 (0,0 %) | 54707 |
| **corpus** | 20 | 11841 | 11841/11841 (100,0 %) | 0/11841 (0,0 %) | 0/11841 (0,0 %) | 0/11841 (0,0 %) | 3609/11841 (30,5 %) | 0/11841 (0,0 %) | 103/11841 (0,9 %) | 11738/11841 (99,1 %) | 0/11841 (0,0 %) | 156240 |

### Par film

| Film | Build | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 084a804d | HI_1_10_0 | 142 | 142/142 (100,0 %) | 0/142 (0,0 %) | 0/142 (0,0 %) | 0/142 (0,0 %) | 133/142 (93,7 %) | 0/142 (0,0 %) | 0/142 (0,0 %) | 142/142 (100,0 %) | 0/142 (0,0 %) | 182 |
| 111fa685 | HI_1_10_0 | 402 | 402/402 (100,0 %) | 0/402 (0,0 %) | 0/402 (0,0 %) | 0/402 (0,0 %) | 216/402 (53,7 %) | 0/402 (0,0 %) | 0/402 (0,0 %) | 402/402 (100,0 %) | 0/402 (0,0 %) | 3428 |
| 1c4c63c2 | HI_1_10_0 | 229 | 229/229 (100,0 %) | 0/229 (0,0 %) | 0/229 (0,0 %) | 0/229 (0,0 %) | 215/229 (93,9 %) | 0/229 (0,0 %) | 0/229 (0,0 %) | 229/229 (100,0 %) | 0/229 (0,0 %) | 496 |
| e5adf7b2 | HI_1_11_0 | 291 | 291/291 (100,0 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 268/291 (92,1 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 291/291 (100,0 %) | 0/291 (0,0 %) | 1045 |
| bcb6d393 | HI_1_12_0 | 54 | 54/54 (100,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 49/54 (90,7 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 54/54 (100,0 %) | 0/54 (0,0 %) | 2 |
| 0797ce72 | HI_1_13_0 | 970 | 970/970 (100,0 %) | 0/970 (0,0 %) | 0/970 (0,0 %) | 0/970 (0,0 %) | 854/970 (88,0 %) | 0/970 (0,0 %) | 6/970 (0,6 %) | 964/970 (99,4 %) | 0/970 (0,0 %) | 5939 |
| 396cfc92 | HI_1_13_0 | 13 | 13/13 (100,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 11/13 (84,6 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 13/13 (100,0 %) | 0/13 (0,0 %) | 109 |
| 4f77afc1 | HI_1_13_0 | 97 | 97/97 (100,0 %) | 0/97 (0,0 %) | 0/97 (0,0 %) | 0/97 (0,0 %) | 91/97 (93,8 %) | 0/97 (0,0 %) | 0/97 (0,0 %) | 97/97 (100,0 %) | 0/97 (0,0 %) | 2505 |
| 51ebbc0f | HI_1_13_0 | 6 | 6/6 (100,0 %) | 0/6 (0,0 %) | 0/6 (0,0 %) | 0/6 (0,0 %) | 3/6 (50,0 %) | 0/6 (0,0 %) | 0/6 (0,0 %) | 6/6 (100,0 %) | 0/6 (0,0 %) | 27 |
| bf15f7ab | HI_1_13_0 | 38 | 38/38 (100,0 %) | 0/38 (0,0 %) | 0/38 (0,0 %) | 0/38 (0,0 %) | 6/38 (15,8 %) | 0/38 (0,0 %) | 1/38 (2,6 %) | 37/38 (97,4 %) | 0/38 (0,0 %) | 257 |
| bfecd02b | HI_1_13_0 | 9 | 9/9 (100,0 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 8/9 (88,9 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 9/9 (100,0 %) | 0/9 (0,0 %) | 81 |
| c75f33b8 | HI_1_13_0 | 2 | 2/2 (100,0 %) | 0/2 (0,0 %) | 0/2 (0,0 %) | 0/2 (0,0 %) | 2/2 (100,0 %) | 0/2 (0,0 %) | 0/2 (0,0 %) | 2/2 (100,0 %) | 0/2 (0,0 %) | 16 |
| d9781168 | HI_1_13_0 | 189 | 189/189 (100,0 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 188/189 (99,5 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 189/189 (100,0 %) | 0/189 (0,0 %) | 21 |
| f75e7053 | HI_1_13_0 | 0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0 |
| fb1a1a72 | HI_1_13_0 | 10 | 10/10 (100,0 %) | 0/10 (0,0 %) | 0/10 (0,0 %) | 0/10 (0,0 %) | 10/10 (100,0 %) | 0/10 (0,0 %) | 0/10 (0,0 %) | 10/10 (100,0 %) | 0/10 (0,0 %) | 64 |
| a521164d | HI_1_4_1 | 2042 | 2042/2042 (100,0 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 169/2042 (8,3 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 2042/2042 (100,0 %) | 0/2042 (0,0 %) | 39302 |
| 60ae07c4 | HI_1_8_0 | 1007 | 1007/1007 (100,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 899/1007 (89,3 %) | 0/1007 (0,0 %) | 96/1007 (9,5 %) | 911/1007 (90,5 %) | 0/1007 (0,0 %) | 5386 |
| 11de8353 | HI_1_9_0 | 207 | 207/207 (100,0 %) | 0/207 (0,0 %) | 0/207 (0,0 %) | 0/207 (0,0 %) | 180/207 (87,0 %) | 0/207 (0,0 %) | 0/207 (0,0 %) | 207/207 (100,0 %) | 0/207 (0,0 %) | 1536 |
| 50247b26 | version-31 | 3111 | 3111/3111 (100,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 94/3111 (3,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 3111/3111 (100,0 %) | 0/3111 (0,0 %) | 41137 |
| a349fea8 | version-33 | 3022 | 3022/3022 (100,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 213/3022 (7,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 3022/3022 (100,0 %) | 0/3022 (0,0 %) | 54707 |

### Corpus : sortie x vue C x reste

| Sortie B | Vue C | Reste | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| terminateur | 1 entree(s) | >= 64 | 7411 | 122000 |
| terminateur | vide | >= 64 | 3543 | 17964 |
| terminateur | 2 entree(s) | >= 64 | 506 | 13710 |
| terminateur | 0 entree (kind 3 seul) | >= 64 | 275 | 2109 |
| terminateur | vide | 8-63 | 66 | 393 |
| terminateur | 0 entree (kind 3 seul) | 8-63 | 29 | 0 |
| terminateur | 1 entree(s) | 8-63 | 8 | 52 |
| terminateur | 3 entree(s) | >= 64 | 3 | 12 |

### Corpus : dernier record et dernier composant lus avant la fin de la vue B

| Sortie B | Dernier lu | Paquets | Utiles en jeu |
|---|---|---|---|
| terminateur | DELTA ti=40 i25 unit-command-tick-component | 1682 | 13105 |
| terminateur | DELTA ti=37 i3 object-angular-velocity-component | 1592 | 30943 |
| terminateur | DELTA ti=10 i26 managed-object-rtpc-component | 1024 | 23905 |
| terminateur | DELTA ti=4 i0 high-frequency | 1008 | 98 |
| terminateur | DEL | 946 | 15138 |
| terminateur | DELTA ti=43 i18 device-position-component | 828 | 344 |
| terminateur | DELTA ti=42 i3 object-angular-velocity-component | 698 | 18487 |
| terminateur | DELTA ti=21 i16 flock-position-component | 337 | 1 |
| terminateur | DELTA ti=42 i2 object-forward-and-up-component | 309 | 8300 |
| terminateur | DELTA ti=37 i29 equipment-command-tick-component | 291 | 6777 |
| terminateur | DELTA ti=37 i1 object-translational-velocity-component | 237 | 6979 |
| terminateur | DELTA ti=37 i2 object-forward-and-up-component | 216 | 5203 |
| terminateur | DELTA ti=42 i1 object-translational-velocity-component | 196 | 5401 |
| terminateur | DELTA ti=3 sans composant | 186 | 0 |
| terminateur | DELTA ti=20 i1 spawn-filter-weight-component | 176 | 2127 |
| terminateur | DELTA ti=47 i1 managed-object-networked-splash-message-dynamic-component | 162 | 3231 |
| terminateur | NEW ti=42 sans composant | 129 | 7 |
| terminateur | DELTA ti=20 i0 spawn-filter-type-component | 123 | 1501 |
| terminateur | NEW ti=41 sans composant | 109 | 13 |
| terminateur | DELTA ti=12 i14 managed-navpoint-radial-progress | 90 | 2260 |
| terminateur | DELTA ti=38 i3 object-angular-velocity-component | 59 | 1534 |
| terminateur | DELTA ti=37 i28 equipment-tracked-object-handles-stack-component | 57 | 893 |
| terminateur | DELTA ti=35 i25 unit-command-tick-component | 55 | 85 |
| terminateur | DELTA ti=0 i2 game-engine-current-state-component | 49 | 1195 |
| terminateur | NEW ti=40 sans composant | 44 | 0 |
| terminateur | DELTA ti=38 i1 object-translational-velocity-component | 43 | 1119 |
| terminateur | NEW ti=38 sans composant | 42 | 50 |
| terminateur | NEW ti=41 i18 projectile-at-rest-state | 42 | 42 |
| terminateur | DELTA ti=42 i19 item-ignore-player-component | 41 | 978 |
| terminateur | NEW ti=37 sans composant | 41 | 0 |

## Sorties de la vue B, tous paquets qui l atteignent

| Build | Sortie B | Paquets | Fermes | Hors cadre | Autres causes |
|---|---|---|---|---|---|
| HI_1_10_0 | ouverte | 1422 | 0/1422 (0,0 %) | 0/1422 (0,0 %) | 1422/1422 (100,0 %) |
| HI_1_10_0 | rejet hors datum | 83725 | 0/83725 (0,0 %) | 0/83725 (0,0 %) | 83725/83725 (100,0 %) |
| HI_1_10_0 | terminateur | 26526 | 22246/26526 (83,9 %) | 773/26526 (2,9 %) | 3507/26526 (13,2 %) |
| HI_1_11_0 | ouverte | 150 | 0/150 (0,0 %) | 0/150 (0,0 %) | 150/150 (100,0 %) |
| HI_1_11_0 | rejet hors datum | 10318 | 0/10318 (0,0 %) | 0/10318 (0,0 %) | 10318/10318 (100,0 %) |
| HI_1_11_0 | terminateur | 4630 | 4147/4630 (89,6 %) | 291/4630 (6,3 %) | 192/4630 (4,1 %) |
| HI_1_12_0 | ouverte | 12336 | 0/12336 (0,0 %) | 0/12336 (0,0 %) | 12336/12336 (100,0 %) |
| HI_1_12_0 | rejet hors datum | 3305 | 0/3305 (0,0 %) | 0/3305 (0,0 %) | 3305/3305 (100,0 %) |
| HI_1_12_0 | terminateur | 5942 | 5879/5942 (98,9 %) | 54/5942 (0,9 %) | 9/5942 (0,2 %) |
| HI_1_13_0 | ouverte | 2304 | 0/2304 (0,0 %) | 0/2304 (0,0 %) | 2304/2304 (100,0 %) |
| HI_1_13_0 | rejet hors datum | 54781 | 0/54781 (0,0 %) | 0/54781 (0,0 %) | 54781/54781 (100,0 %) |
| HI_1_13_0 | terminateur | 263207 | 260393/263207 (98,9 %) | 1334/263207 (0,5 %) | 1480/263207 (0,6 %) |
| HI_1_4_1 | ouverte | 143 | 0/143 (0,0 %) | 0/143 (0,0 %) | 143/143 (100,0 %) |
| HI_1_4_1 | rejet hors datum | 6676 | 0/6676 (0,0 %) | 0/6676 (0,0 %) | 6676/6676 (100,0 %) |
| HI_1_4_1 | terminateur | 3021 | 692/3021 (22,9 %) | 2042/3021 (67,6 %) | 287/3021 (9,5 %) |
| HI_1_8_0 | ouverte | 741 | 0/741 (0,0 %) | 0/741 (0,0 %) | 741/741 (100,0 %) |
| HI_1_8_0 | rejet hors datum | 30379 | 0/30379 (0,0 %) | 0/30379 (0,0 %) | 30379/30379 (100,0 %) |
| HI_1_8_0 | terminateur | 15317 | 13946/15317 (91,0 %) | 1007/15317 (6,6 %) | 364/15317 (2,4 %) |
| HI_1_9_0 | ouverte | 165 | 0/165 (0,0 %) | 0/165 (0,0 %) | 165/165 (100,0 %) |
| HI_1_9_0 | rejet hors datum | 9869 | 0/9869 (0,0 %) | 0/9869 (0,0 %) | 9869/9869 (100,0 %) |
| HI_1_9_0 | terminateur | 5957 | 5629/5957 (94,5 %) | 207/5957 (3,5 %) | 121/5957 (2,0 %) |
| version-31 | ouverte | 151 | 0/151 (0,0 %) | 0/151 (0,0 %) | 151/151 (100,0 %) |
| version-31 | rejet hors datum | 10811 | 0/10811 (0,0 %) | 0/10811 (0,0 %) | 10811/10811 (100,0 %) |
| version-31 | terminateur | 4700 | 139/4700 (3,0 %) | 3111/4700 (66,2 %) | 1450/4700 (30,9 %) |
| version-33 | ouverte | 315 | 0/315 (0,0 %) | 0/315 (0,0 %) | 315/315 (100,0 %) |
| version-33 | rejet hors datum | 19976 | 0/19976 (0,0 %) | 0/19976 (0,0 %) | 19976/19976 (100,0 %) |
| version-33 | terminateur | 4421 | 424/4421 (9,6 %) | 3022/4421 (68,4 %) | 975/4421 (22,1 %) |
| **corpus** | ouverte | 17727 | 0/17727 (0,0 %) | 0/17727 (0,0 %) | 17727/17727 (100,0 %) |
| **corpus** | rejet hors datum | 229840 | 0/229840 (0,0 %) | 0/229840 (0,0 %) | 229840/229840 (100,0 %) |
| **corpus** | terminateur | 333721 | 313495/333721 (93,9 %) | 11841/333721 (3,5 %) | 8385/333721 (2,5 %) |

## Sorties par rejet contre le bloc de type 1 (item 1.2)

Etat du slot rejete dans le bloc de type 1 de SON chunk ; naissance : le bloc du chunk SUIVANT porte-t-il une allocation de cet eid que le chunk ne portait pas, sans NEW lu ?

| Sortie B | Paquet | Etat au bloc | Naissance | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| rejet hors datum | vue B : sortie par rejet | vide | naissance non lue | 109379 | 1629022 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue | 78751 | 1109009 |
| rejet hors datum | vue B : sortie par rejet | vivant | vivant au bloc du chunk | 14769 | 36413 |
| rejet hors datum | vue B : sortie par rejet | vide | aucune allocation | 11039 | 50917 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue, generation 0 | 8089 | 147314 |
| rejet hors datum | vue B : sortie par rejet | vide | non mesurable | 3789 | 12669 |
| rejet hors datum | vue B : sortie par rejet | trace | aucune allocation | 2092 | 6812 |
| rejet hors datum | vue B : sortie par rejet | trace | non mesurable | 805 | 6806 |
| rejet hors datum | vue B : sortie par rejet | trace | libere avant le chunk (meme generation) | 392 | 2065 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · naissance non lue | 366 | 3786 |
| rejet hors datum | vue B : sortie par rejet | vide | realloue sous une autre generation | 142 | 363 |
| rejet hors datum | vue B : sortie par rejet | vivant | non mesurable | 64 | 184 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · aucune allocation | 60 | 338 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu dans le chunk | 39 | 56 |
| rejet hors datum | vue B : sortie par rejet | trace | realloue sous une autre generation | 30 | 145 |
| rejet hors datum | vue B : sortie par rejet | vivant | aucune allocation | 9 | 54 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · naissance non lue | 6 | 3 |
| rejet hors datum | vue B : sortie par rejet | vivant | realloue sous une autre generation | 6 | 48 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu dans le chunk | 5 | 37 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · libere avant le chunk (meme generation) | 3 | 50 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · aucune allocation | 2 | 25 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu dans le chunk | 2 | 0 |
| rejet hors datum | vue B : sortie par rejet | absent | aucune allocation | 1 | 0 |

| Build | Rejets | dont paquet hors cadre | dont ferme | Vivant au bloc | Naissance non lue | Non mesurable |
|---|---|---|---|---|---|---|
| HI_1_10_0 | 83725 | 0/83725 (0,0 %) | 0/83725 (0,0 %) | 10706/83725 (12,8 %) | 64408/83725 (76,9 %) | 787/83725 (0,9 %) |
| HI_1_11_0 | 10318 | 0/10318 (0,0 %) | 0/10318 (0,0 %) | 56/10318 (0,5 %) | 8526/10318 (82,6 %) | 0/10318 (0,0 %) |
| HI_1_12_0 | 3305 | 0/3305 (0,0 %) | 0/3305 (0,0 %) | 0/3305 (0,0 %) | 3064/3305 (92,7 %) | 0/3305 (0,0 %) |
| HI_1_13_0 | 54781 | 0/54781 (0,0 %) | 0/54781 (0,0 %) | 2255/54781 (4,1 %) | 44814/54781 (81,8 %) | 2643/54781 (4,8 %) |
| HI_1_4_1 | 6676 | 0/6676 (0,0 %) | 0/6676 (0,0 %) | 0/6676 (0,0 %) | 5932/6676 (88,9 %) | 342/6676 (5,1 %) |
| HI_1_8_0 | 30379 | 0/30379 (0,0 %) | 0/30379 (0,0 %) | 209/30379 (0,7 %) | 26118/30379 (86,0 %) | 204/30379 (0,7 %) |
| HI_1_9_0 | 9869 | 0/9869 (0,0 %) | 0/9869 (0,0 %) | 137/9869 (1,4 %) | 7712/9869 (78,1 %) | 192/9869 (1,9 %) |
| version-31 | 10811 | 0/10811 (0,0 %) | 0/10811 (0,0 %) | 1/10811 (0,0 %) | 10236/10811 (94,7 %) | 486/10811 (4,5 %) |
| version-33 | 19976 | 0/19976 (0,0 %) | 0/19976 (0,0 %) | 1486/19976 (7,4 %) | 17320/19976 (86,7 %) | 4/19976 (0,0 %) |
| **corpus** | 229840 | 0/229840 (0,0 %) | 0/229840 (0,0 %) | 14850/229840 (6,5 %) | 188130/229840 (81,9 %) | 4658/229840 (2,0 %) |

## Entrees de controle utiles (item 1.3)

Utile = entree `kind 0` qui porte le bloc de 0x68 octets, la seule que le tir continu lit. Denominateur ESTIME = utiles fermees + moyenne par vue C fermee du film x paquets non fermes. PAR CONSTRUCTION, la part estimee d un film egale sa part de vues C fermees (U / (U + U/F x N) = F / (F + N)) : cette estimation ne dit rien de plus que les paquets.

| Build | Paquets | Vues C fermees | Entrees fermees | Utiles fermees | Moyenne par vue C fermee | Denominateur estime | Part estimee | Utiles lues hors fermeture (non prouvees) |
|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 129369 | 22246/129369 (17,2 %) | 348855 | 348855 | 15.68 | 2053031 | 17,0 % | 73627 |
| HI_1_11_0 | 16824 | 4147/16824 (24,6 %) | 74842 | 74842 | 18.05 | 303627 | 24,6 % | 1897 |
| HI_1_12_0 | 21864 | 5879/21864 (26,9 %) | 30418 | 30418 | 5.17 | 113125 | 26,9 % | 24 |
| HI_1_13_0 | 335960 | 260393/335960 (77,5 %) | 1863008 | 1863008 | 7.15 | 2455686 | 75,9 % | 15367 |
| HI_1_4_1 | 11130 | 692/11130 (6,2 %) | 0 | 0 | 0.00 | 0 | - | 2049 |
| HI_1_8_0 | 49696 | 13946/49696 (28,1 %) | 79784 | 79784 | 5.72 | 284307 | 28,1 % | 454 |
| HI_1_9_0 | 17629 | 5629/17629 (31,9 %) | 74235 | 74235 | 13.19 | 232490 | 31,9 % | 1485 |
| version-31 | 17919 | 139/17919 (0,8 %) | 0 | 0 | 0.00 | 0 | - | 4205 |
| version-33 | 28751 | 424/28751 (1,5 %) | 0 | 0 | 0.00 | 0 | - | 3699 |
| **corpus** | 629142 | 313495/629142 (49,8 %) | 2471142 | 2471142 | 7.88 | 5442267 | 45,4 % | 102807 |

## Chunk des temps forts : compte declare contre evenements trouves (item 1.5)

| Film | Build | Chunk | Paquets type 9 | Declares | Trouves | Ecart | Kills | Deaths | Medailles | Mode | Fil des morts | Refus |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 084a804d | HI_1_10_0 | 56 | 1 | 988 | 988 | 0 | 354 | 356 | 199 | 79 | 356 | - |
| 111fa685 | HI_1_10_0 | 30 | 1 | 472 | 472 | 0 | 191 | 193 | 85 | 3 | 193 | - |
| 1c4c63c2 | HI_1_10_0 | 68 | 1 | 1305 | 1305 | 0 | 494 | 495 | 255 | 61 | 495 | - |
| e5adf7b2 | HI_1_11_0 | 30 | 1 | 491 | 491 | 0 | 197 | 199 | 87 | 8 | 199 | - |
| bcb6d393 | HI_1_12_0 | 20 | 1 | 136 | 136 | 0 | 49 | 49 | 21 | 17 | 49 | - |
| 0797ce72 | HI_1_13_0 | 24 | 1 | 220 | 220 | 0 | 92 | 93 | 35 | 0 | 93 | - |
| 396cfc92 | HI_1_13_0 | 28 | 1 | 218 | 218 | 0 | 67 | 67 | 26 | 58 | 67 | - |
| 4f77afc1 | HI_1_13_0 | 62 | 1 | 844 | 844 | 0 | 298 | 300 | 177 | 69 | 300 | - |
| 51ebbc0f | HI_1_13_0 | 27 | 1 | 244 | 244 | 0 | 70 | 71 | 28 | 75 | 71 | - |
| bf15f7ab | HI_1_13_0 | 27 | 1 | 195 | 195 | 0 | 75 | 76 | 44 | 0 | 76 | - |
| bfecd02b | HI_1_13_0 | 28 | 1 | 207 | 207 | 0 | 84 | 84 | 38 | 1 | 84 | - |
| c75f33b8 | HI_1_13_0 | 25 | 1 | 150 | 150 | 0 | 62 | 61 | 27 | 0 | 61 | - |
| d9781168 | HI_1_13_0 | 38 | 1 | 461 | 461 | 0 | 142 | 144 | 51 | 124 | 144 | - |
| f75e7053 | HI_1_13_0 | 25 | 1 | 252 | 252 | 0 | 82 | 83 | 36 | 51 | 83 | - |
| fb1a1a72 | HI_1_13_0 | 42 | 1 | 388 | 388 | 0 | 140 | 141 | 62 | 45 | 141 | - |
| a521164d | HI_1_4_1 | 20 | 1 | 277 | 277 | 0 | 101 | 106 | 38 | 32 | 106 | - |
| 60ae07c4 | HI_1_8_0 | 43 | 1 | 490 | 490 | 0 | 162 | 163 | 54 | 111 | 163 | - |
| 11de8353 | HI_1_9_0 | 32 | 1 | 393 | 393 | 0 | 165 | 166 | 54 | 8 | 166 | - |
| 50247b26 | version-31 | 31 | 1 | 395 | 395 | 0 | 150 | 154 | 78 | 13 | 154 | - |
| a349fea8 | version-33 | 50 | 1 | 817 | 817 | 0 | 290 | 293 | 133 | 101 | 293 | - |

## Mode borne : lectures au-dela de la fin du payload (item 1.6)

| Build | Records lus (vue B) | Records debordants | Composants debordants | NEW propres debordants | Paquets avec debordement | Terminateurs de vue B lus au-dela | Chunks consultes sans bloc de type 1 | Blocs de type 1 illisibles |
|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 1809371 | 17/1809371 (0,0 %) | 68 | 9 | 17 | 3 | 2 | 0 |
| HI_1_11_0 | 324140 | 3/324140 (0,0 %) | 31 | 1 | 3 | 1 | 0 | 0 |
| HI_1_12_0 | 169656 | 6/169656 (0,0 %) | 22 | 1 | 6 | 1 | 0 | 0 |
| HI_1_13_0 | 3894103 | 283/3894103 (0,0 %) | 1700 | 70 | 283 | 11 | 7 | 0 |
| HI_1_4_1 | 230281 | 1/230281 (0,0 %) | 2 | 1 | 1 | 1 | 1 | 0 |
| HI_1_8_0 | 342533 | 392/342533 (0,1 %) | 1623 | 89 | 392 | 12 | 1 | 0 |
| HI_1_9_0 | 276734 | 38/276734 (0,0 %) | 204 | 30 | 38 | 1 | 1 | 0 |
| version-31 | 545602 | 4/545602 (0,0 %) | 13 | 1 | 4 | 1 | 1 | 0 |
| version-33 | 565676 | 7/565676 (0,0 %) | 38 | 5 | 7 | 1 | 1 | 0 |
| **corpus** | 8158096 | 751/8158096 (0,0 %) | 3701 | 207 | 751 | 32 | 14 | 0 |

## Lot L0 — fermes au bit pres contre fermes (regles de l ecrivain)

| Build | Fermes au bit | Fermes | Retires | Utiles lus | Utiles fermes au bit | Utiles fermes | Part variable | Fixe | Part fixe |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 29299 | 22246 | 7053 | 1567213 | 305108 | 298477 | 19,0 % | 2423551 | 12,3 % |
| HI_1_11_0 | 4310 | 4147 | 163 | 289563 | 80623 | 80067 | 27,7 % | 383476 | 20,9 % |
| HI_1_12_0 | 5885 | 5879 | 6 | 122669 | 35507 | 35492 | 28,9 % | 148160 | 24,0 % |
| HI_1_13_0 | 262032 | 260393 | 1639 | 2811122 | 2361989 | 2355178 | 83,8 % | 3073267 | 76,6 % |
| HI_1_4_1 | 702 | 692 | 10 | 184588 | 80 | 78 | 0,0 % | 198500 | 0,0 % |
| HI_1_8_0 | 14114 | 13946 | 168 | 271810 | 84165 | 83669 | 30,8 % | 359291 | 23,3 % |
| HI_1_9_0 | 5760 | 5629 | 131 | 248938 | 65874 | 65621 | 26,4 % | 324613 | 20,2 % |
| version-31 | 153 | 139 | 14 | 320306 | 277 | 274 | 0,1 % | 328128 | 0,1 % |
| version-33 | 469 | 424 | 45 | 479794 | 3663 | 3654 | 0,8 % | 519304 | 0,7 % |
| **corpus** | 322724 | 313495 | 9229 | 6296003 | 2937286 | 2922510 | 46,4 % | 7758290 | 37,7 % |

### Corpus : fermetures au bit retirees, par premiere regle

| Regle | Paquets | Utiles |
|---|---|---|
| ecrivain : masque au-dela de l archetype | 4267 | 12368 |
| ecrivain : masque epars non croissant | 134 | 418 |
| ecrivain : ordre de la vue B | 256 | 1013 |
| ecrivain : vue C en-tete cdc04 pose | 6 | 6 |
| ecrivain : vue C kind non nul | 2 | 0 |
| vue B : sortie par rejet | 4564 | 971 |

### Corpus : temoins decales (vue C relue depuis un depart decale de k bits)

| k | Refermes | Part des fermes |
|---|---|---|
| -8 | 1186 | 0,4 % |
| -7 | 1786 | 0,6 % |
| -6 | 3562 | 1,1 % |
| -5 | 4647 | 1,5 % |
| -4 | 14659 | 4,7 % |
| -3 | 10469 | 3,3 % |
| -2 | 10697 | 3,4 % |
| -1 | 10903 | 3,5 % |
| +1 | 25156 | 8,0 % |
| +2 | 24335 | 7,8 % |
| +3 | 17127 | 5,5 % |
| +4 | 15815 | 5,0 % |
| +5 | 15216 | 4,9 % |
| +6 | 14988 | 4,8 % |
| +7 | 14782 | 4,7 % |
| +8 | 0 | 0,0 % |


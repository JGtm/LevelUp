# Carte de fermeture des trames delta — 2026-10-03

Films mesures : 20 ; echecs : 0. Contexte d instrument (largeurs d axe lues dans le film, profil par defaut, sans calibration killsource).

Table ECS : `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L4a/ecs_table_avant.tsv`.

## Par build

| Build | Films | Paquets fermes | Vue A | Vue B | Vue C | Records utiles fermes | Entrees de controle lues | Declencheur (>= 95 %) | Pic memoire max |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 22118/129369 (17,1 %) | 16699/79272 (21,1 %) | 22118/111243 (19,9 %) | 22118/26214 (84,4 %) | 296041/1558435 (19,0 %) | 346689 | non atteint | 312 Mio |
| HI_1_11_0 | 1 | 4123/16824 (24,5 %) | 2957/9073 (32,6 %) | 4123/15097 (27,3 %) | 4123/4603 (89,6 %) | 79457/289357 (27,5 %) | 74386 | non atteint | 123 Mio |
| HI_1_12_0 | 1 | 5830/21864 (26,7 %) | 5201/19134 (27,2 %) | 5830/21581 (27,0 %) | 5830/5886 (99,0 %) | 35143/121628 (28,9 %) | 30125 | non atteint | 73 Mio |
| HI_1_13_0 | 10 | 223591/335960 (66,6 %) | 190673/271063 (70,3 %) | 223591/319898 (69,9 %) | 223591/227310 (98,4 %) | 2023371/2507329 (80,7 %) | 1612522 | non atteint | 225 Mio |
| HI_1_4_1 | 1 | 692/11130 (6,2 %) | 651/6174 (10,5 %) | 692/9839 (7,0 %) | 692/2971 (23,3 %) | 78/181607 (0,0 %) | 0 | non atteint | 215 Mio |
| HI_1_8_0 | 1 | 13802/49696 (27,8 %) | 13002/43640 (29,8 %) | 13802/46436 (29,7 %) | 13802/15155 (91,1 %) | 82730/268352 (30,8 %) | 78935 | non atteint | 144 Mio |
| HI_1_9_0 | 1 | 5612/17629 (31,8 %) | 4404/10304 (42,7 %) | 5612/15988 (35,1 %) | 5612/5952 (94,3 %) | 65230/248287 (26,3 %) | 73906 | non atteint | 118 Mio |
| version-31 | 1 | 139/17919 (0,8 %) | 130/10140 (1,3 %) | 139/15662 (0,9 %) | 139/4674 (3,0 %) | 274/319053 (0,1 %) | 0 | non atteint | 161 Mio |
| version-33 | 1 | 420/28751 (1,5 %) | 398/14259 (2,8 %) | 420/24710 (1,7 %) | 420/4328 (9,7 %) | 3595/469451 (0,8 %) | 0 | non atteint | 233 Mio |

## Causes d arret (premiere cause de chaque paquet non ferme)

Gain potentiel = records utiles LUS et non fermes dans les paquets que la cause arrete : BORNE SUPERIEURE (une autre cause peut suivre ; les records d apres l arret ne sont pas lus du tout).

| Rang | Cause | Archetype | Index | Statut | Usage produit | Paquets bloques | Gain potentiel | Builds |
|---|---|---|---|---|---|---|---|---|
| 1 | vue B : sortie par rejet | - |  | - | - | 259088 | 2997765 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 2 | liste d evenements non localisee | - |  | - | - | 48688 | 0 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 3 | vue C : terminateur hors cadre | - |  | - | - | 12481 | 152448 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 4 | ti=43 device-animation-layer-state | 43 | i35 | non_porte | aucun | 12114 | 67608 | HI_1_10_0, HI_1_12_0, HI_1_8_0 |
| 5 | ecrivain : masque au-dela de l archetype | - |  | - | - | 4078 | 11404 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 6 | ti=2 managed-engine-timers | 2 | i15 | non_porte | aucun | 3996 | 28 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 7 | vue C : kind non porte | - |  | - | - | 2856 | 54079 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 8 | ti=3 low-frequency | 3 | i0 | non_porte | aucun | 1187 | 135 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 9 | vue C : bloc 0xbc (desalignement) | - |  | - | - | 953 | 19378 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 10 | ti=48 forge-player-data-edited-objects-ids | 48 | i0 | non_porte | aucun | 931 | 8143 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 11 | ti=40 vehicle-auto-turret-aiming-vector | 40 | i31 | non_porte | aucun | 808 | 4827 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-33 |
| 12 | vue B : fin de payload | - |  | - | - | 604 | 3093 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 13 | ti=43 device-position-group | 43 | i21 | non_porte | aucun | 603 | 10410 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 14 | ti=40 vehicle-weapon-set | 40 | i37, i38 | non_porte | aucun | 590 | 12509 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 15 | ti=10 managed-object-navpoint | 10 | i10, i11, i12, i13, i15, i16, i17, i2, i3, i4, i5, i6, i7, i8 | non_porte | aucun | 344 | 4285 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 16 | ti=12 managed-navpoint-visual-state-groups-component-2 | 12 | i22 | non_porte | aucun | 341 | 5030 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-31, version-33 |
| 17 | ti=12 managed-navpoint-visual-state-groups-component-1 | 12 | i21 | non_porte | aucun | 333 | 4698 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 18 | ti=35 biped-spartan-ability-non-predicted-state | 35 | i58, i59 | partiel | rejeu : grappleLines[] (schema 8) — la ligne blanche du joueur vers son ancre, fenetre [t0,t1] par vie | 240 | 2008 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 19 | ti=56 archetype hors registre | 56 |  | - | - | 231 | 1141 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 20 | ecrivain : ordre de la vue B | - |  | - | - | 224 | 1009 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 21 | ecrivain : masque epars non croissant | - |  | - | - | 120 | 387 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 22 | ti=12 managed-navpoint-visual-state-groups-component-0 | 12 | i20 | non_porte | aucun | 115 | 1258 | HI_1_13_0, HI_1_4_1, version-33 |
| 23 | ti=12 managed-navpoint-override-flags | 12 | i16 | non_porte | aucun | 88 | 454 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 24 | ti=2 matchflow-isplaying-flags | 2 | i17 | non_porte | aucun | 85 | 13 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 25 | ti=11 managed-objective-interaction-filter | 11 | i4 | non_porte | aucun | 83 | 656 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0 |
| 26 | ti=40 vehicle-type-state | 40 | i32, i33 | non_porte | aucun | 83 | 1786 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_9_0, version-31 |
| 27 | ti=43 device-machine-flags | 43 | i39 | non_porte | aucun | 83 | 1176 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, version-31, version-33 |
| 28 | ti=2 game-engine-soft-ceilings | 2 | i11 | non_porte | aucun | 67 | 420 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 29 | ti=18 effect-state-data | 18 | i0, i1, i10, i12, i13, i19, i2, i22, i23, i28, i3, i31, i4, i6, i7, i8, i9 | partiel | aucun | 66 | 201 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31 |
| 30 | ti=40 air-drop-flight | 40 | i44, i45 | non_porte | aucun | 59 | 1444 | HI_1_11_0, HI_1_4_1, version-31, version-33 |
## Carte v2 — « vue C : terminateur hors cadre » ventile (item 1.1)

Sortie de la vue B : comment la boucle de records s est arretee avant la vue C ; vue C : vide (son terminateur seul) ou non ; reste : bits du payload derriere le terminateur de la vue C.

### Par build

| Build | Films | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 770 | 770/770 (100,0 %) | 0/770 (0,0 %) | 0/770 (0,0 %) | 0/770 (0,0 %) | 561/770 (72,9 %) | 0/770 (0,0 %) | 0/770 (0,0 %) | 770/770 (100,0 %) | 0/770 (0,0 %) | 4119 |
| HI_1_11_0 | 1 | 288 | 288/288 (100,0 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 266/288 (92,4 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 288/288 (100,0 %) | 0/288 (0,0 %) | 1042 |
| HI_1_12_0 | 1 | 49 | 49/49 (100,0 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 44/49 (89,8 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 49/49 (100,0 %) | 0/49 (0,0 %) | 1 |
| HI_1_13_0 | 10 | 2096 | 2096/2096 (100,0 %) | 0/2096 (0,0 %) | 0/2096 (0,0 %) | 0/2096 (0,0 %) | 1637/2096 (78,1 %) | 0/2096 (0,0 %) | 9/2096 (0,4 %) | 2087/2096 (99,6 %) | 0/2096 (0,0 %) | 7786 |
| HI_1_4_1 | 1 | 2002 | 2002/2002 (100,0 %) | 0/2002 (0,0 %) | 0/2002 (0,0 %) | 0/2002 (0,0 %) | 160/2002 (8,0 %) | 0/2002 (0,0 %) | 0/2002 (0,0 %) | 2002/2002 (100,0 %) | 0/2002 (0,0 %) | 38445 |
| HI_1_8_0 | 1 | 994 | 994/994 (100,0 %) | 0/994 (0,0 %) | 0/994 (0,0 %) | 0/994 (0,0 %) | 887/994 (89,2 %) | 0/994 (0,0 %) | 96/994 (9,7 %) | 898/994 (90,3 %) | 0/994 (0,0 %) | 5301 |
| HI_1_9_0 | 1 | 225 | 225/225 (100,0 %) | 0/225 (0,0 %) | 0/225 (0,0 %) | 0/225 (0,0 %) | 199/225 (88,4 %) | 0/225 (0,0 %) | 0/225 (0,0 %) | 225/225 (100,0 %) | 0/225 (0,0 %) | 1835 |
| version-31 | 1 | 3096 | 3096/3096 (100,0 %) | 0/3096 (0,0 %) | 0/3096 (0,0 %) | 0/3096 (0,0 %) | 91/3096 (2,9 %) | 0/3096 (0,0 %) | 0/3096 (0,0 %) | 3096/3096 (100,0 %) | 0/3096 (0,0 %) | 40785 |
| version-33 | 1 | 2961 | 2961/2961 (100,0 %) | 0/2961 (0,0 %) | 0/2961 (0,0 %) | 0/2961 (0,0 %) | 204/2961 (6,9 %) | 0/2961 (0,0 %) | 0/2961 (0,0 %) | 2961/2961 (100,0 %) | 0/2961 (0,0 %) | 53134 |
| **corpus** | 20 | 12481 | 12481/12481 (100,0 %) | 0/12481 (0,0 %) | 0/12481 (0,0 %) | 0/12481 (0,0 %) | 4049/12481 (32,4 %) | 0/12481 (0,0 %) | 105/12481 (0,8 %) | 12376/12481 (99,2 %) | 0/12481 (0,0 %) | 152448 |

### Par film

| Film | Build | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 084a804d | HI_1_10_0 | 141 | 141/141 (100,0 %) | 0/141 (0,0 %) | 0/141 (0,0 %) | 0/141 (0,0 %) | 132/141 (93,6 %) | 0/141 (0,0 %) | 0/141 (0,0 %) | 141/141 (100,0 %) | 0/141 (0,0 %) | 176 |
| 111fa685 | HI_1_10_0 | 401 | 401/401 (100,0 %) | 0/401 (0,0 %) | 0/401 (0,0 %) | 0/401 (0,0 %) | 215/401 (53,6 %) | 0/401 (0,0 %) | 0/401 (0,0 %) | 401/401 (100,0 %) | 0/401 (0,0 %) | 3427 |
| 1c4c63c2 | HI_1_10_0 | 228 | 228/228 (100,0 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 214/228 (93,9 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 228/228 (100,0 %) | 0/228 (0,0 %) | 516 |
| e5adf7b2 | HI_1_11_0 | 288 | 288/288 (100,0 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 266/288 (92,4 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 288/288 (100,0 %) | 0/288 (0,0 %) | 1042 |
| bcb6d393 | HI_1_12_0 | 49 | 49/49 (100,0 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 44/49 (89,8 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 49/49 (100,0 %) | 0/49 (0,0 %) | 1 |
| 0797ce72 | HI_1_13_0 | 966 | 966/966 (100,0 %) | 0/966 (0,0 %) | 0/966 (0,0 %) | 0/966 (0,0 %) | 850/966 (88,0 %) | 0/966 (0,0 %) | 6/966 (0,6 %) | 960/966 (99,4 %) | 0/966 (0,0 %) | 5908 |
| 396cfc92 | HI_1_13_0 | 12 | 12/12 (100,0 %) | 0/12 (0,0 %) | 0/12 (0,0 %) | 0/12 (0,0 %) | 10/12 (83,3 %) | 0/12 (0,0 %) | 0/12 (0,0 %) | 12/12 (100,0 %) | 0/12 (0,0 %) | 95 |
| 4f77afc1 | HI_1_13_0 | 52 | 52/52 (100,0 %) | 0/52 (0,0 %) | 0/52 (0,0 %) | 0/52 (0,0 %) | 46/52 (88,5 %) | 0/52 (0,0 %) | 0/52 (0,0 %) | 52/52 (100,0 %) | 0/52 (0,0 %) | 1294 |
| 51ebbc0f | HI_1_13_0 | 331 | 331/331 (100,0 %) | 0/331 (0,0 %) | 0/331 (0,0 %) | 0/331 (0,0 %) | 176/331 (53,2 %) | 0/331 (0,0 %) | 0/331 (0,0 %) | 331/331 (100,0 %) | 0/331 (0,0 %) | 32 |
| bf15f7ab | HI_1_13_0 | 37 | 37/37 (100,0 %) | 0/37 (0,0 %) | 0/37 (0,0 %) | 0/37 (0,0 %) | 5/37 (13,5 %) | 0/37 (0,0 %) | 0/37 (0,0 %) | 37/37 (100,0 %) | 0/37 (0,0 %) | 250 |
| bfecd02b | HI_1_13_0 | 9 | 9/9 (100,0 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 8/9 (88,9 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 9/9 (100,0 %) | 0/9 (0,0 %) | 81 |
| c75f33b8 | HI_1_13_0 | 13 | 13/13 (100,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 10/13 (76,9 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 13/13 (100,0 %) | 0/13 (0,0 %) | 16 |
| d9781168 | HI_1_13_0 | 189 | 189/189 (100,0 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 188/189 (99,5 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 189/189 (100,0 %) | 0/189 (0,0 %) | 21 |
| f75e7053 | HI_1_13_0 | 0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0 |
| fb1a1a72 | HI_1_13_0 | 487 | 487/487 (100,0 %) | 0/487 (0,0 %) | 0/487 (0,0 %) | 0/487 (0,0 %) | 344/487 (70,6 %) | 0/487 (0,0 %) | 3/487 (0,6 %) | 484/487 (99,4 %) | 0/487 (0,0 %) | 89 |
| a521164d | HI_1_4_1 | 2002 | 2002/2002 (100,0 %) | 0/2002 (0,0 %) | 0/2002 (0,0 %) | 0/2002 (0,0 %) | 160/2002 (8,0 %) | 0/2002 (0,0 %) | 0/2002 (0,0 %) | 2002/2002 (100,0 %) | 0/2002 (0,0 %) | 38445 |
| 60ae07c4 | HI_1_8_0 | 994 | 994/994 (100,0 %) | 0/994 (0,0 %) | 0/994 (0,0 %) | 0/994 (0,0 %) | 887/994 (89,2 %) | 0/994 (0,0 %) | 96/994 (9,7 %) | 898/994 (90,3 %) | 0/994 (0,0 %) | 5301 |
| 11de8353 | HI_1_9_0 | 225 | 225/225 (100,0 %) | 0/225 (0,0 %) | 0/225 (0,0 %) | 0/225 (0,0 %) | 199/225 (88,4 %) | 0/225 (0,0 %) | 0/225 (0,0 %) | 225/225 (100,0 %) | 0/225 (0,0 %) | 1835 |
| 50247b26 | version-31 | 3096 | 3096/3096 (100,0 %) | 0/3096 (0,0 %) | 0/3096 (0,0 %) | 0/3096 (0,0 %) | 91/3096 (2,9 %) | 0/3096 (0,0 %) | 0/3096 (0,0 %) | 3096/3096 (100,0 %) | 0/3096 (0,0 %) | 40785 |
| a349fea8 | version-33 | 2961 | 2961/2961 (100,0 %) | 0/2961 (0,0 %) | 0/2961 (0,0 %) | 0/2961 (0,0 %) | 204/2961 (6,9 %) | 0/2961 (0,0 %) | 0/2961 (0,0 %) | 2961/2961 (100,0 %) | 0/2961 (0,0 %) | 53134 |

### Corpus : sortie x vue C x reste

| Sortie B | Vue C | Reste | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| terminateur | 1 entree(s) | >= 64 | 7543 | 119464 |
| terminateur | vide | >= 64 | 3981 | 16753 |
| terminateur | 2 entree(s) | >= 64 | 525 | 13678 |
| terminateur | 0 entree (kind 3 seul) | >= 64 | 324 | 2102 |
| terminateur | vide | 8-63 | 68 | 387 |
| terminateur | 0 entree (kind 3 seul) | 8-63 | 29 | 0 |
| terminateur | 1 entree(s) | 8-63 | 8 | 52 |
| terminateur | 3 entree(s) | >= 64 | 3 | 12 |

### Corpus : dernier record et dernier composant lus avant la fin de la vue B

| Sortie B | Dernier lu | Paquets | Utiles en jeu |
|---|---|---|---|
| terminateur | DELTA ti=40 i25 unit-command-tick-component | 1674 | 13027 |
| terminateur | DELTA ti=37 i3 object-angular-velocity-component | 1563 | 30159 |
| terminateur | DEL | 1046 | 14879 |
| terminateur | DELTA ti=10 i26 managed-object-rtpc-component | 1019 | 23782 |
| terminateur | DELTA ti=4 i0 high-frequency | 1008 | 98 |
| terminateur | DELTA ti=43 i18 device-position-component | 828 | 344 |
| terminateur | DELTA ti=42 i3 object-angular-velocity-component | 684 | 18118 |
| terminateur | DELTA ti=3 i1 high-frequency | 386 | 8 |
| terminateur | DELTA ti=21 i16 flock-position-component | 337 | 1 |
| terminateur | DELTA ti=42 i2 object-forward-and-up-component | 304 | 8169 |
| terminateur | DELTA ti=37 i29 equipment-command-tick-component | 272 | 6393 |
| terminateur | DELTA ti=37 i1 object-translational-velocity-component | 230 | 6758 |
| terminateur | DELTA ti=37 i2 object-forward-and-up-component | 213 | 5117 |
| terminateur | DELTA ti=42 i1 object-translational-velocity-component | 192 | 5281 |
| terminateur | DELTA ti=3 sans composant | 186 | 0 |
| terminateur | DELTA ti=20 i1 spawn-filter-weight-component | 176 | 2127 |
| terminateur | DELTA ti=47 i1 managed-object-networked-splash-message-dynamic-component | 160 | 3188 |
| terminateur | NEW ti=42 sans composant | 130 | 7 |
| terminateur | DELTA ti=20 i0 spawn-filter-type-component | 123 | 1501 |
| terminateur | NEW ti=41 sans composant | 109 | 13 |
| terminateur | NEW ti=33 sans composant | 91 | 26 |
| terminateur | DELTA ti=12 i14 managed-navpoint-radial-progress | 89 | 2236 |
| terminateur | DELTA ti=38 i3 object-angular-velocity-component | 59 | 1534 |
| terminateur | DELTA ti=37 i28 equipment-tracked-object-handles-stack-component | 57 | 893 |
| terminateur | DELTA ti=35 i25 unit-command-tick-component | 55 | 85 |
| terminateur | DELTA ti=8 sans composant | 55 | 8 |
| terminateur | NEW ti=33 i0 tacmap-displayasset | 55 | 1 |
| terminateur | NEW ti=32 i0 tacmap-areaofinterest | 44 | 6 |
| terminateur | NEW ti=40 sans composant | 44 | 0 |
| terminateur | DELTA ti=38 i1 object-translational-velocity-component | 43 | 1119 |

## Sorties de la vue B, tous paquets qui l atteignent

| Build | Sortie B | Paquets | Fermes | Hors cadre | Autres causes |
|---|---|---|---|---|---|
| HI_1_10_0 | ouverte | 2094 | 0/2094 (0,0 %) | 0/2094 (0,0 %) | 2094/2094 (100,0 %) |
| HI_1_10_0 | rejet hors datum | 82935 | 0/82935 (0,0 %) | 0/82935 (0,0 %) | 82935/82935 (100,0 %) |
| HI_1_10_0 | terminateur | 26214 | 22118/26214 (84,4 %) | 770/26214 (2,9 %) | 3326/26214 (12,7 %) |
| HI_1_11_0 | ouverte | 266 | 0/266 (0,0 %) | 0/266 (0,0 %) | 266/266 (100,0 %) |
| HI_1_11_0 | rejet hors datum | 10228 | 0/10228 (0,0 %) | 0/10228 (0,0 %) | 10228/10228 (100,0 %) |
| HI_1_11_0 | terminateur | 4603 | 4123/4603 (89,6 %) | 288/4603 (6,3 %) | 192/4603 (4,2 %) |
| HI_1_12_0 | ouverte | 12425 | 0/12425 (0,0 %) | 0/12425 (0,0 %) | 12425/12425 (100,0 %) |
| HI_1_12_0 | rejet hors datum | 3270 | 0/3270 (0,0 %) | 0/3270 (0,0 %) | 3270/3270 (100,0 %) |
| HI_1_12_0 | terminateur | 5886 | 5830/5886 (99,0 %) | 49/5886 (0,8 %) | 7/5886 (0,1 %) |
| HI_1_13_0 | ouverte | 6379 | 0/6379 (0,0 %) | 0/6379 (0,0 %) | 6379/6379 (100,0 %) |
| HI_1_13_0 | rejet hors datum | 86209 | 0/86209 (0,0 %) | 0/86209 (0,0 %) | 86209/86209 (100,0 %) |
| HI_1_13_0 | terminateur | 227310 | 223591/227310 (98,4 %) | 2096/227310 (0,9 %) | 1623/227310 (0,7 %) |
| HI_1_4_1 | ouverte | 342 | 0/342 (0,0 %) | 0/342 (0,0 %) | 342/342 (100,0 %) |
| HI_1_4_1 | rejet hors datum | 6526 | 0/6526 (0,0 %) | 0/6526 (0,0 %) | 6526/6526 (100,0 %) |
| HI_1_4_1 | terminateur | 2971 | 692/2971 (23,3 %) | 2002/2971 (67,4 %) | 277/2971 (9,3 %) |
| HI_1_8_0 | ouverte | 1284 | 0/1284 (0,0 %) | 0/1284 (0,0 %) | 1284/1284 (100,0 %) |
| HI_1_8_0 | rejet hors datum | 29997 | 0/29997 (0,0 %) | 0/29997 (0,0 %) | 29997/29997 (100,0 %) |
| HI_1_8_0 | terminateur | 15155 | 13802/15155 (91,1 %) | 994/15155 (6,6 %) | 359/15155 (2,4 %) |
| HI_1_9_0 | ouverte | 249 | 0/249 (0,0 %) | 0/249 (0,0 %) | 249/249 (100,0 %) |
| HI_1_9_0 | rejet hors datum | 9787 | 0/9787 (0,0 %) | 0/9787 (0,0 %) | 9787/9787 (100,0 %) |
| HI_1_9_0 | terminateur | 5952 | 5612/5952 (94,3 %) | 225/5952 (3,8 %) | 115/5952 (1,9 %) |
| version-31 | ouverte | 280 | 0/280 (0,0 %) | 0/280 (0,0 %) | 280/280 (100,0 %) |
| version-31 | rejet hors datum | 10708 | 0/10708 (0,0 %) | 0/10708 (0,0 %) | 10708/10708 (100,0 %) |
| version-31 | terminateur | 4674 | 139/4674 (3,0 %) | 3096/4674 (66,2 %) | 1439/4674 (30,8 %) |
| version-33 | ouverte | 954 | 0/954 (0,0 %) | 0/954 (0,0 %) | 954/954 (100,0 %) |
| version-33 | rejet hors datum | 19428 | 0/19428 (0,0 %) | 0/19428 (0,0 %) | 19428/19428 (100,0 %) |
| version-33 | terminateur | 4328 | 420/4328 (9,7 %) | 2961/4328 (68,4 %) | 947/4328 (21,9 %) |
| **corpus** | ouverte | 24273 | 0/24273 (0,0 %) | 0/24273 (0,0 %) | 24273/24273 (100,0 %) |
| **corpus** | rejet hors datum | 259088 | 0/259088 (0,0 %) | 0/259088 (0,0 %) | 259088/259088 (100,0 %) |
| **corpus** | terminateur | 297093 | 276327/297093 (93,0 %) | 12481/297093 (4,2 %) | 8285/297093 (2,8 %) |

## Sorties par rejet contre le bloc de type 1 (item 1.2)

Etat du slot rejete dans le bloc de type 1 de SON chunk ; naissance : le bloc du chunk SUIVANT porte-t-il une allocation de cet eid que le chunk ne portait pas, sans NEW lu ?

| Sortie B | Paquet | Etat au bloc | Naissance | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| rejet hors datum | vue B : sortie par rejet | vide | naissance non lue | 109073 | 1619009 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue | 80297 | 1111947 |
| rejet hors datum | vue B : sortie par rejet | vivant | vivant au bloc du chunk | 14697 | 36054 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · naissance non lue | 14584 | 424 |
| rejet hors datum | vue B : sortie par rejet | vide | aucune allocation | 11114 | 48928 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue, generation 0 | 8770 | 149655 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · naissance non lue | 8754 | 2658 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · naissance non lue, generation 0 | 4085 | 72 |
| rejet hors datum | vue B : sortie par rejet | vide | non mesurable | 3589 | 12398 |
| rejet hors datum | vue B : sortie par rejet | trace | aucune allocation | 2020 | 6612 |
| rejet hors datum | vue B : sortie par rejet | trace | non mesurable | 788 | 6718 |
| rejet hors datum | vue B : sortie par rejet | trace | libere avant le chunk (meme generation) | 391 | 2057 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · non mesurable | 277 | 1 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · non mesurable | 247 | 0 |
| rejet hors datum | vue B : sortie par rejet | vide | realloue sous une autre generation | 138 | 362 |
| rejet hors datum | vue B : sortie par rejet | vivant | non mesurable | 64 | 184 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · aucune allocation | 61 | 323 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu dans le chunk | 45 | 56 |
| rejet hors datum | vue B : sortie par rejet | trace | realloue sous une autre generation | 29 | 94 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu desynchronise · vivant au bloc du chunk | 28 | 0 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu dans le chunk | 9 | 0 |
| rejet hors datum | vue B : sortie par rejet | vivant | aucune allocation | 9 | 54 |
| rejet hors datum | vue B : sortie par rejet | vivant | realloue sous une autre generation | 5 | 47 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu dans le chunk | 4 | 28 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · aucune allocation | 3 | 34 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · libere avant le chunk (meme generation) | 3 | 50 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu desynchronise · aucune allocation | 3 | 0 |
| rejet hors datum | vue B : sortie par rejet | absent | aucune allocation | 1 | 0 |

| Build | Rejets | dont paquet hors cadre | dont ferme | Vivant au bloc | Naissance non lue | Non mesurable |
|---|---|---|---|---|---|---|
| HI_1_10_0 | 82935 | 0/82935 (0,0 %) | 0/82935 (0,0 %) | 10701/82935 (12,9 %) | 63886/82935 (77,0 %) | 785/82935 (0,9 %) |
| HI_1_11_0 | 10228 | 0/10228 (0,0 %) | 0/10228 (0,0 %) | 55/10228 (0,5 %) | 8453/10228 (82,6 %) | 0/10228 (0,0 %) |
| HI_1_12_0 | 3270 | 0/3270 (0,0 %) | 0/3270 (0,0 %) | 0/3270 (0,0 %) | 3037/3270 (92,9 %) | 0/3270 (0,0 %) |
| HI_1_13_0 | 86209 | 0/86209 (0,0 %) | 0/86209 (0,0 %) | 2261/86209 (2,6 %) | 47693/86209 (55,3 %) | 2441/86209 (2,8 %) |
| HI_1_4_1 | 6526 | 0/6526 (0,0 %) | 0/6526 (0,0 %) | 0/6526 (0,0 %) | 5807/6526 (89,0 %) | 330/6526 (5,1 %) |
| HI_1_8_0 | 29997 | 0/29997 (0,0 %) | 0/29997 (0,0 %) | 203/29997 (0,7 %) | 25813/29997 (86,1 %) | 204/29997 (0,7 %) |
| HI_1_9_0 | 9787 | 0/9787 (0,0 %) | 0/9787 (0,0 %) | 137/9787 (1,4 %) | 7674/9787 (78,4 %) | 191/9787 (2,0 %) |
| version-31 | 10708 | 0/10708 (0,0 %) | 0/10708 (0,0 %) | 1/10708 (0,0 %) | 10145/10708 (94,7 %) | 486/10708 (4,5 %) |
| version-33 | 19428 | 0/19428 (0,0 %) | 0/19428 (0,0 %) | 1457/19428 (7,5 %) | 16862/19428 (86,8 %) | 4/19428 (0,0 %) |
| **corpus** | 259088 | 0/259088 (0,0 %) | 0/259088 (0,0 %) | 14815/259088 (5,7 %) | 189370/259088 (73,1 %) | 4441/259088 (1,7 %) |

## Entrees de controle utiles (item 1.3)

Utile = entree `kind 0` qui porte le bloc de 0x68 octets, la seule que le tir continu lit. Denominateur ESTIME = utiles fermees + moyenne par vue C fermee du film x paquets non fermes. PAR CONSTRUCTION, la part estimee d un film egale sa part de vues C fermees (U / (U + U/F x N) = F / (F + N)) : cette estimation ne dit rien de plus que les paquets.

| Build | Paquets | Vues C fermees | Entrees fermees | Utiles fermees | Moyenne par vue C fermee | Denominateur estime | Part estimee | Utiles lues hors fermeture (non prouvees) |
|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 129369 | 22118/129369 (17,1 %) | 346689 | 346689 | 15.67 | 2052680 | 16,9 % | 70484 |
| HI_1_11_0 | 16824 | 4123/16824 (24,5 %) | 74386 | 74386 | 18.04 | 303534 | 24,5 % | 1873 |
| HI_1_12_0 | 21864 | 5830/21864 (26,7 %) | 30125 | 30125 | 5.17 | 112977 | 26,7 % | 21 |
| HI_1_13_0 | 335960 | 223591/335960 (66,6 %) | 1612522 | 1612522 | 7.21 | 2433653 | 66,3 % | 14636 |
| HI_1_4_1 | 11130 | 692/11130 (6,2 %) | 0 | 0 | 0.00 | 0 | - | 2015 |
| HI_1_8_0 | 49696 | 13802/49696 (27,8 %) | 78935 | 78935 | 5.72 | 284216 | 27,8 % | 448 |
| HI_1_9_0 | 17629 | 5612/17629 (31,8 %) | 73906 | 73906 | 13.17 | 232161 | 31,8 % | 1442 |
| version-31 | 17919 | 139/17919 (0,8 %) | 0 | 0 | 0.00 | 0 | - | 4182 |
| version-33 | 28751 | 420/28751 (1,5 %) | 0 | 0 | 0.00 | 0 | - | 3624 |
| **corpus** | 629142 | 276327/629142 (43,9 %) | 2216563 | 2216563 | 8.02 | 5419221 | 40,9 % | 98725 |

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
| HI_1_10_0 | 1798224 | 15/1798224 (0,0 %) | 62 | 8 | 15 | 3 | 2 | 0 |
| HI_1_11_0 | 323840 | 3/323840 (0,0 %) | 31 | 1 | 3 | 1 | 0 | 0 |
| HI_1_12_0 | 168157 | 2/168157 (0,0 %) | 10 | 0 | 2 | 1 | 0 | 0 |
| HI_1_13_0 | 3359278 | 270/3359278 (0,0 %) | 1630 | 68 | 270 | 11 | 7 | 0 |
| HI_1_4_1 | 226385 | 1/226385 (0,0 %) | 2 | 1 | 1 | 1 | 1 | 0 |
| HI_1_8_0 | 338248 | 337/338248 (0,1 %) | 1312 | 79 | 337 | 12 | 1 | 0 |
| HI_1_9_0 | 275920 | 19/275920 (0,0 %) | 83 | 6 | 19 | 1 | 1 | 0 |
| version-31 | 542836 | 4/542836 (0,0 %) | 13 | 1 | 4 | 1 | 1 | 0 |
| version-33 | 553211 | 7/553211 (0,0 %) | 38 | 5 | 7 | 1 | 1 | 0 |
| **corpus** | 7586099 | 658/7586099 (0,0 %) | 3181 | 169 | 658 | 32 | 14 | 0 |

## Lot L0 — fermes au bit pres contre fermes (regles de l ecrivain)

| Build | Fermes au bit | Fermes | Retires | Utiles lus | Utiles fermes au bit | Utiles fermes | Part variable | Fixe | Part fixe |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 28763 | 22118 | 6645 | 1558435 | 302396 | 296041 | 19,0 % | 2423551 | 12,2 % |
| HI_1_11_0 | 4285 | 4123 | 162 | 289357 | 80011 | 79457 | 27,5 % | 383476 | 20,7 % |
| HI_1_12_0 | 5835 | 5830 | 5 | 121628 | 35158 | 35143 | 28,9 % | 148160 | 23,7 % |
| HI_1_13_0 | 225177 | 223591 | 1586 | 2507329 | 2029470 | 2023371 | 80,7 % | 3073267 | 65,8 % |
| HI_1_4_1 | 701 | 692 | 9 | 181607 | 80 | 78 | 0,0 % | 198500 | 0,0 % |
| HI_1_8_0 | 13969 | 13802 | 167 | 268352 | 83225 | 82730 | 30,8 % | 359291 | 23,0 % |
| HI_1_9_0 | 5739 | 5612 | 127 | 248287 | 65454 | 65230 | 26,3 % | 324613 | 20,1 % |
| version-31 | 153 | 139 | 14 | 319053 | 277 | 274 | 0,1 % | 328128 | 0,1 % |
| version-33 | 463 | 420 | 43 | 469451 | 3602 | 3595 | 0,8 % | 519304 | 0,7 % |
| **corpus** | 285085 | 276327 | 8758 | 5963499 | 2599673 | 2585919 | 43,4 % | 7758290 | 33,3 % |

### Corpus : fermetures au bit retirees, par premiere regle

| Regle | Paquets | Utiles |
|---|---|---|
| ecrivain : masque au-dela de l archetype | 4078 | 11404 |
| ecrivain : masque epars non croissant | 120 | 387 |
| ecrivain : ordre de la vue B | 224 | 1009 |
| ecrivain : vue C en-tete cdc04 pose | 6 | 6 |
| ecrivain : vue C kind non nul | 1 | 0 |
| vue B : sortie par rejet | 4329 | 948 |

### Corpus : temoins decales (vue C relue depuis un depart decale de k bits)

| k | Refermes | Part des fermes |
|---|---|---|
| -8 | 1089 | 0,4 % |
| -7 | 1586 | 0,6 % |
| -6 | 3226 | 1,2 % |
| -5 | 4566 | 1,7 % |
| -4 | 14086 | 5,1 % |
| -3 | 10218 | 3,7 % |
| -2 | 10398 | 3,8 % |
| -1 | 10574 | 3,8 % |
| +1 | 23733 | 8,6 % |
| +2 | 23008 | 8,3 % |
| +3 | 15841 | 5,7 % |
| +4 | 14572 | 5,3 % |
| +5 | 14022 | 5,1 % |
| +6 | 13842 | 5,0 % |
| +7 | 13666 | 4,9 % |
| +8 | 0 | 0,0 % |


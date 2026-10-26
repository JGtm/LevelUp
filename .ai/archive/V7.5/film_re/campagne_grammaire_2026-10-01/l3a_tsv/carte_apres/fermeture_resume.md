# Carte de fermeture des trames delta — 2026-10-02

Films mesures : 20 ; echecs : 0. Contexte d instrument (largeurs d axe lues dans le film, profil par defaut, sans calibration killsource).

Table ECS : `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L3a/ecs_table_lot.tsv`.

## Par build

| Build | Films | Paquets fermes | Vue A | Vue B | Vue C | Records utiles fermes | Entrees de controle lues | Declencheur (>= 95 %) | Pic memoire max |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 22192/129369 (17,2 %) | 16755/79272 (21,1 %) | 22192/111647 (19,9 %) | 22192/26453 (83,9 %) | 297230/1566502 (19,0 %) | 347835 | non atteint | 314 Mio |
| HI_1_11_0 | 1 | 4124/16824 (24,5 %) | 2958/9073 (32,6 %) | 4124/15097 (27,3 %) | 4124/4604 (89,6 %) | 79458/289358 (27,5 %) | 74386 | non atteint | 122 Mio |
| HI_1_12_0 | 1 | 5879/21864 (26,9 %) | 5249/19134 (27,4 %) | 5879/21583 (27,2 %) | 5879/5942 (98,9 %) | 35492/122669 (28,9 %) | 30418 | non atteint | 71 Mio |
| HI_1_13_0 | 10 | 226783/335960 (67,5 %) | 193323/271063 (71,3 %) | 226783/320008 (70,9 %) | 226783/230554 (98,4 %) | 2074170/2538265 (81,7 %) | 1645819 | non atteint | 223 Mio |
| HI_1_4_1 | 1 | 692/11130 (6,2 %) | 651/6174 (10,5 %) | 692/9840 (7,0 %) | 692/3000 (23,1 %) | 78/184473 (0,0 %) | 0 | non atteint | 213 Mio |
| HI_1_8_0 | 1 | 13946/49696 (28,1 %) | 13146/43640 (30,1 %) | 13946/46437 (30,0 %) | 13946/15317 (91,0 %) | 83669/271810 (30,8 %) | 79784 | non atteint | 152 Mio |
| HI_1_9_0 | 1 | 5622/17629 (31,9 %) | 4414/10304 (42,8 %) | 5622/15989 (35,2 %) | 5622/5971 (94,2 %) | 65460/248837 (26,3 %) | 74102 | non atteint | 119 Mio |
| version-31 | 1 | 139/17919 (0,8 %) | 130/10140 (1,3 %) | 139/15662 (0,9 %) | 139/4682 (3,0 %) | 274/320107 (0,1 %) | 0 | non atteint | 162 Mio |
| version-33 | 1 | 424/28751 (1,5 %) | 402/14259 (2,8 %) | 424/24710 (1,7 %) | 424/4392 (9,7 %) | 3654/479511 (0,8 %) | 0 | non atteint | 231 Mio |

## Causes d arret (premiere cause de chaque paquet non ferme)

Gain potentiel = records utiles LUS et non fermes dans les paquets que la cause arrete : BORNE SUPERIEURE (une autre cause peut suivre ; les records d apres l arret ne sont pas lus du tout).

| Rang | Cause | Archetype | Index | Statut | Usage produit | Paquets bloques | Gain potentiel | Builds |
|---|---|---|---|---|---|---|---|---|
| 1 | vue B : sortie par rejet | - |  | - | - | 259764 | 2997368 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 2 | liste d evenements non localisee | - |  | - | - | 48169 | 0 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 3 | vue C : terminateur hors cadre | - |  | - | - | 12587 | 154390 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 4 | ti=43 device-animation-layer-state | 43 | i35 | non_porte | aucun | 12202 | 68132 | HI_1_10_0, HI_1_12_0, HI_1_8_0 |
| 5 | ecrivain : masque au-dela de l archetype | - |  | - | - | 4233 | 12068 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 6 | vue C : kind non porte | - |  | - | - | 2887 | 54746 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 7 | ti=3 low-frequency | 3 | i0 | non_porte | aucun | 1188 | 141 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 8 | vue C : bloc 0xbc (desalignement) | - |  | - | - | 961 | 19589 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 9 | ti=48 forge-player-data-edited-objects-ids | 48 | i0 | non_porte | aucun | 951 | 8319 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 10 | ti=40 vehicle-auto-turret-aiming-vector | 40 | i31 | non_porte | aucun | 815 | 4867 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-33 |
| 11 | vue B : fin de payload | - |  | - | - | 701 | 3815 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 12 | ti=43 device-position-group | 43 | i21 | non_porte | aucun | 612 | 10593 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 13 | ti=40 vehicle-weapon-set | 40 | i37, i38 | non_porte | aucun | 598 | 12686 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 14 | ti=10 managed-object-navpoint | 10 | i10, i11, i12, i13, i15, i16, i17, i2, i3, i4, i5, i6, i7, i8 | non_porte | aucun | 348 | 4385 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 15 | ti=12 managed-navpoint-visual-state-groups-component-2 | 12 | i22 | non_porte | aucun | 347 | 5131 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-31, version-33 |
| 16 | ti=12 managed-navpoint-visual-state-groups-component-1 | 12 | i21 | non_porte | aucun | 339 | 4930 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 17 | ecrivain : ordre de la vue B | - |  | - | - | 262 | 1025 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 18 | ti=35 biped-spartan-ability-non-predicted-state | 35 | i58, i59 | partiel | rejeu : grappleLines[] (schema 8) — la ligne blanche du joueur vers son ancre, fenetre [t0,t1] par vie | 241 | 2013 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 19 | ti=56 archetype hors registre | 56 |  | - | - | 235 | 1170 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 20 | ecrivain : masque epars non croissant | - |  | - | - | 129 | 394 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 21 | ti=12 managed-navpoint-visual-state-groups-component-0 | 12 | i20 | non_porte | aucun | 116 | 1264 | HI_1_13_0, HI_1_4_1, version-33 |
| 22 | ti=12 managed-navpoint-override-flags | 12 | i16 | non_porte | aucun | 88 | 530 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 23 | ti=40 vehicle-type-state | 40 | i32, i33 | non_porte | aucun | 85 | 1829 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_9_0, version-31 |
| 24 | ti=43 device-machine-flags | 43 | i39 | non_porte | aucun | 84 | 1204 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, version-31, version-33 |
| 25 | ti=11 managed-objective-interaction-filter | 11 | i4 | non_porte | aucun | 83 | 660 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0 |
| 26 | ti=18 effect-state-data | 18 | i0, i1, i10, i12, i13, i19, i2, i22, i23, i28, i3, i31, i4, i6, i7, i8, i9 | partiel | aucun | 66 | 201 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31 |
| 27 | ti=0 forge-engine-player-roles | 0 | i18 | non_porte | aucun | 61 | 410 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 28 | ti=40 air-drop-flight | 40 | i44, i45 | non_porte | aucun | 60 | 1450 | HI_1_11_0, HI_1_4_1, HI_1_8_0, version-31, version-33 |
| 29 | ti=10 managed-object-interaction-filter | 10 | i22 | non_porte | aucun | 55 | 762 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 30 | ti=51 archetype hors registre | 51 |  | - | - | 55 | 484 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
## Carte v2 — « vue C : terminateur hors cadre » ventile (item 1.1)

Sortie de la vue B : comment la boucle de records s est arretee avant la vue C ; vue C : vide (son terminateur seul) ou non ; reste : bits du payload derriere le terminateur de la vue C.

### Par build

| Build | Films | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 771 | 771/771 (100,0 %) | 0/771 (0,0 %) | 0/771 (0,0 %) | 0/771 (0,0 %) | 562/771 (72,9 %) | 0/771 (0,0 %) | 0/771 (0,0 %) | 771/771 (100,0 %) | 0/771 (0,0 %) | 4104 |
| HI_1_11_0 | 1 | 288 | 288/288 (100,0 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 266/288 (92,4 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 288/288 (100,0 %) | 0/288 (0,0 %) | 1042 |
| HI_1_12_0 | 1 | 54 | 54/54 (100,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 49/54 (90,7 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 54/54 (100,0 %) | 0/54 (0,0 %) | 2 |
| HI_1_13_0 | 10 | 2109 | 2109/2109 (100,0 %) | 0/2109 (0,0 %) | 0/2109 (0,0 %) | 0/2109 (0,0 %) | 1648/2109 (78,1 %) | 0/2109 (0,0 %) | 10/2109 (0,5 %) | 2099/2109 (99,5 %) | 0/2109 (0,0 %) | 7793 |
| HI_1_4_1 | 1 | 2026 | 2026/2026 (100,0 %) | 0/2026 (0,0 %) | 0/2026 (0,0 %) | 0/2026 (0,0 %) | 161/2026 (7,9 %) | 0/2026 (0,0 %) | 0/2026 (0,0 %) | 2026/2026 (100,0 %) | 0/2026 (0,0 %) | 39068 |
| HI_1_8_0 | 1 | 1007 | 1007/1007 (100,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 899/1007 (89,3 %) | 0/1007 (0,0 %) | 96/1007 (9,5 %) | 911/1007 (90,5 %) | 0/1007 (0,0 %) | 5386 |
| HI_1_9_0 | 1 | 231 | 231/231 (100,0 %) | 0/231 (0,0 %) | 0/231 (0,0 %) | 0/231 (0,0 %) | 205/231 (88,7 %) | 0/231 (0,0 %) | 0/231 (0,0 %) | 231/231 (100,0 %) | 0/231 (0,0 %) | 1928 |
| version-31 | 1 | 3100 | 3100/3100 (100,0 %) | 0/3100 (0,0 %) | 0/3100 (0,0 %) | 0/3100 (0,0 %) | 92/3100 (3,0 %) | 0/3100 (0,0 %) | 0/3100 (0,0 %) | 3100/3100 (100,0 %) | 0/3100 (0,0 %) | 40860 |
| version-33 | 1 | 3001 | 3001/3001 (100,0 %) | 0/3001 (0,0 %) | 0/3001 (0,0 %) | 0/3001 (0,0 %) | 205/3001 (6,8 %) | 0/3001 (0,0 %) | 0/3001 (0,0 %) | 3001/3001 (100,0 %) | 0/3001 (0,0 %) | 54207 |
| **corpus** | 20 | 12587 | 12587/12587 (100,0 %) | 0/12587 (0,0 %) | 0/12587 (0,0 %) | 0/12587 (0,0 %) | 4087/12587 (32,5 %) | 0/12587 (0,0 %) | 106/12587 (0,8 %) | 12481/12587 (99,2 %) | 0/12587 (0,0 %) | 154390 |

### Par film

| Film | Build | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 084a804d | HI_1_10_0 | 142 | 142/142 (100,0 %) | 0/142 (0,0 %) | 0/142 (0,0 %) | 0/142 (0,0 %) | 133/142 (93,7 %) | 0/142 (0,0 %) | 0/142 (0,0 %) | 142/142 (100,0 %) | 0/142 (0,0 %) | 182 |
| 111fa685 | HI_1_10_0 | 401 | 401/401 (100,0 %) | 0/401 (0,0 %) | 0/401 (0,0 %) | 0/401 (0,0 %) | 215/401 (53,6 %) | 0/401 (0,0 %) | 0/401 (0,0 %) | 401/401 (100,0 %) | 0/401 (0,0 %) | 3427 |
| 1c4c63c2 | HI_1_10_0 | 228 | 228/228 (100,0 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 214/228 (93,9 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 228/228 (100,0 %) | 0/228 (0,0 %) | 495 |
| e5adf7b2 | HI_1_11_0 | 288 | 288/288 (100,0 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 266/288 (92,4 %) | 0/288 (0,0 %) | 0/288 (0,0 %) | 288/288 (100,0 %) | 0/288 (0,0 %) | 1042 |
| bcb6d393 | HI_1_12_0 | 54 | 54/54 (100,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 49/54 (90,7 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 54/54 (100,0 %) | 0/54 (0,0 %) | 2 |
| 0797ce72 | HI_1_13_0 | 967 | 967/967 (100,0 %) | 0/967 (0,0 %) | 0/967 (0,0 %) | 0/967 (0,0 %) | 851/967 (88,0 %) | 0/967 (0,0 %) | 6/967 (0,6 %) | 961/967 (99,4 %) | 0/967 (0,0 %) | 5915 |
| 396cfc92 | HI_1_13_0 | 13 | 13/13 (100,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 11/13 (84,6 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 13/13 (100,0 %) | 0/13 (0,0 %) | 109 |
| 4f77afc1 | HI_1_13_0 | 52 | 52/52 (100,0 %) | 0/52 (0,0 %) | 0/52 (0,0 %) | 0/52 (0,0 %) | 46/52 (88,5 %) | 0/52 (0,0 %) | 0/52 (0,0 %) | 52/52 (100,0 %) | 0/52 (0,0 %) | 1294 |
| 51ebbc0f | HI_1_13_0 | 331 | 331/331 (100,0 %) | 0/331 (0,0 %) | 0/331 (0,0 %) | 0/331 (0,0 %) | 176/331 (53,2 %) | 0/331 (0,0 %) | 0/331 (0,0 %) | 331/331 (100,0 %) | 0/331 (0,0 %) | 11 |
| bf15f7ab | HI_1_13_0 | 38 | 38/38 (100,0 %) | 0/38 (0,0 %) | 0/38 (0,0 %) | 0/38 (0,0 %) | 6/38 (15,8 %) | 0/38 (0,0 %) | 1/38 (2,6 %) | 37/38 (97,4 %) | 0/38 (0,0 %) | 257 |
| bfecd02b | HI_1_13_0 | 9 | 9/9 (100,0 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 8/9 (88,9 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 9/9 (100,0 %) | 0/9 (0,0 %) | 81 |
| c75f33b8 | HI_1_13_0 | 13 | 13/13 (100,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 10/13 (76,9 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 13/13 (100,0 %) | 0/13 (0,0 %) | 16 |
| d9781168 | HI_1_13_0 | 189 | 189/189 (100,0 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 188/189 (99,5 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 189/189 (100,0 %) | 0/189 (0,0 %) | 21 |
| f75e7053 | HI_1_13_0 | 0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0 |
| fb1a1a72 | HI_1_13_0 | 497 | 497/497 (100,0 %) | 0/497 (0,0 %) | 0/497 (0,0 %) | 0/497 (0,0 %) | 352/497 (70,8 %) | 0/497 (0,0 %) | 3/497 (0,6 %) | 494/497 (99,4 %) | 0/497 (0,0 %) | 89 |
| a521164d | HI_1_4_1 | 2026 | 2026/2026 (100,0 %) | 0/2026 (0,0 %) | 0/2026 (0,0 %) | 0/2026 (0,0 %) | 161/2026 (7,9 %) | 0/2026 (0,0 %) | 0/2026 (0,0 %) | 2026/2026 (100,0 %) | 0/2026 (0,0 %) | 39068 |
| 60ae07c4 | HI_1_8_0 | 1007 | 1007/1007 (100,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 899/1007 (89,3 %) | 0/1007 (0,0 %) | 96/1007 (9,5 %) | 911/1007 (90,5 %) | 0/1007 (0,0 %) | 5386 |
| 11de8353 | HI_1_9_0 | 231 | 231/231 (100,0 %) | 0/231 (0,0 %) | 0/231 (0,0 %) | 0/231 (0,0 %) | 205/231 (88,7 %) | 0/231 (0,0 %) | 0/231 (0,0 %) | 231/231 (100,0 %) | 0/231 (0,0 %) | 1928 |
| 50247b26 | version-31 | 3100 | 3100/3100 (100,0 %) | 0/3100 (0,0 %) | 0/3100 (0,0 %) | 0/3100 (0,0 %) | 92/3100 (3,0 %) | 0/3100 (0,0 %) | 0/3100 (0,0 %) | 3100/3100 (100,0 %) | 0/3100 (0,0 %) | 40860 |
| a349fea8 | version-33 | 3001 | 3001/3001 (100,0 %) | 0/3001 (0,0 %) | 0/3001 (0,0 %) | 0/3001 (0,0 %) | 205/3001 (6,8 %) | 0/3001 (0,0 %) | 0/3001 (0,0 %) | 3001/3001 (100,0 %) | 0/3001 (0,0 %) | 54207 |

### Corpus : sortie x vue C x reste

| Sortie B | Vue C | Reste | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| terminateur | 1 entree(s) | >= 64 | 7610 | 121210 |
| terminateur | vide | >= 64 | 4018 | 16916 |
| terminateur | 2 entree(s) | >= 64 | 526 | 13710 |
| terminateur | 0 entree (kind 3 seul) | >= 64 | 324 | 2096 |
| terminateur | vide | 8-63 | 69 | 394 |
| terminateur | 0 entree (kind 3 seul) | 8-63 | 29 | 0 |
| terminateur | 1 entree(s) | 8-63 | 8 | 52 |
| terminateur | 3 entree(s) | >= 64 | 3 | 12 |

### Corpus : dernier record et dernier composant lus avant la fin de la vue B

| Sortie B | Dernier lu | Paquets | Utiles en jeu |
|---|---|---|---|
| terminateur | DELTA ti=40 i25 unit-command-tick-component | 1680 | 13067 |
| terminateur | DELTA ti=37 i3 object-angular-velocity-component | 1586 | 30766 |
| terminateur | DEL | 1058 | 15030 |
| terminateur | DELTA ti=10 i26 managed-object-rtpc-component | 1022 | 23855 |
| terminateur | DELTA ti=4 i0 high-frequency | 1008 | 98 |
| terminateur | DELTA ti=43 i18 device-position-component | 828 | 344 |
| terminateur | DELTA ti=42 i3 object-angular-velocity-component | 694 | 18388 |
| terminateur | DELTA ti=3 i1 high-frequency | 391 | 8 |
| terminateur | DELTA ti=21 i16 flock-position-component | 337 | 1 |
| terminateur | DELTA ti=42 i2 object-forward-and-up-component | 308 | 8274 |
| terminateur | DELTA ti=37 i29 equipment-command-tick-component | 282 | 6648 |
| terminateur | DELTA ti=37 i1 object-translational-velocity-component | 235 | 6911 |
| terminateur | DELTA ti=37 i2 object-forward-and-up-component | 213 | 5117 |
| terminateur | DELTA ti=42 i1 object-translational-velocity-component | 195 | 5371 |
| terminateur | DELTA ti=3 sans composant | 186 | 0 |
| terminateur | DELTA ti=20 i1 spawn-filter-weight-component | 176 | 2127 |
| terminateur | DELTA ti=47 i1 managed-object-networked-splash-message-dynamic-component | 160 | 3188 |
| terminateur | NEW ti=42 sans composant | 130 | 7 |
| terminateur | DELTA ti=20 i0 spawn-filter-type-component | 123 | 1501 |
| terminateur | NEW ti=41 sans composant | 109 | 13 |
| terminateur | NEW ti=33 sans composant | 91 | 26 |
| terminateur | DELTA ti=12 i14 managed-navpoint-radial-progress | 90 | 2260 |
| terminateur | DELTA ti=38 i3 object-angular-velocity-component | 59 | 1534 |
| terminateur | DELTA ti=8 sans composant | 58 | 8 |
| terminateur | DELTA ti=37 i28 equipment-tracked-object-handles-stack-component | 57 | 893 |
| terminateur | DELTA ti=35 i25 unit-command-tick-component | 55 | 85 |
| terminateur | NEW ti=33 i0 tacmap-displayasset | 55 | 1 |
| terminateur | NEW ti=32 i0 tacmap-areaofinterest | 44 | 6 |
| terminateur | NEW ti=40 sans composant | 44 | 0 |
| terminateur | DELTA ti=38 i1 object-translational-velocity-component | 43 | 1119 |

## Sorties de la vue B, tous paquets qui l atteignent

| Build | Sortie B | Paquets | Fermes | Hors cadre | Autres causes |
|---|---|---|---|---|---|
| HI_1_10_0 | ouverte | 1687 | 0/1687 (0,0 %) | 0/1687 (0,0 %) | 1687/1687 (100,0 %) |
| HI_1_10_0 | rejet hors datum | 83507 | 0/83507 (0,0 %) | 0/83507 (0,0 %) | 83507/83507 (100,0 %) |
| HI_1_10_0 | terminateur | 26453 | 22192/26453 (83,9 %) | 771/26453 (2,9 %) | 3490/26453 (13,2 %) |
| HI_1_11_0 | ouverte | 264 | 0/264 (0,0 %) | 0/264 (0,0 %) | 264/264 (100,0 %) |
| HI_1_11_0 | rejet hors datum | 10229 | 0/10229 (0,0 %) | 0/10229 (0,0 %) | 10229/10229 (100,0 %) |
| HI_1_11_0 | terminateur | 4604 | 4124/4604 (89,6 %) | 288/4604 (6,3 %) | 192/4604 (4,2 %) |
| HI_1_12_0 | ouverte | 12336 | 0/12336 (0,0 %) | 0/12336 (0,0 %) | 12336/12336 (100,0 %) |
| HI_1_12_0 | rejet hors datum | 3305 | 0/3305 (0,0 %) | 0/3305 (0,0 %) | 3305/3305 (100,0 %) |
| HI_1_12_0 | terminateur | 5942 | 5879/5942 (98,9 %) | 54/5942 (0,9 %) | 9/5942 (0,2 %) |
| HI_1_13_0 | ouverte | 4163 | 0/4163 (0,0 %) | 0/4163 (0,0 %) | 4163/4163 (100,0 %) |
| HI_1_13_0 | rejet hors datum | 85291 | 0/85291 (0,0 %) | 0/85291 (0,0 %) | 85291/85291 (100,0 %) |
| HI_1_13_0 | terminateur | 230554 | 226783/230554 (98,4 %) | 2109/230554 (0,9 %) | 1662/230554 (0,7 %) |
| HI_1_4_1 | ouverte | 215 | 0/215 (0,0 %) | 0/215 (0,0 %) | 215/215 (100,0 %) |
| HI_1_4_1 | rejet hors datum | 6625 | 0/6625 (0,0 %) | 0/6625 (0,0 %) | 6625/6625 (100,0 %) |
| HI_1_4_1 | terminateur | 3000 | 692/3000 (23,1 %) | 2026/3000 (67,5 %) | 282/3000 (9,4 %) |
| HI_1_8_0 | ouverte | 742 | 0/742 (0,0 %) | 0/742 (0,0 %) | 742/742 (100,0 %) |
| HI_1_8_0 | rejet hors datum | 30378 | 0/30378 (0,0 %) | 0/30378 (0,0 %) | 30378/30378 (100,0 %) |
| HI_1_8_0 | terminateur | 15317 | 13946/15317 (91,0 %) | 1007/15317 (6,6 %) | 364/15317 (2,4 %) |
| HI_1_9_0 | ouverte | 194 | 0/194 (0,0 %) | 0/194 (0,0 %) | 194/194 (100,0 %) |
| HI_1_9_0 | rejet hors datum | 9824 | 0/9824 (0,0 %) | 0/9824 (0,0 %) | 9824/9824 (100,0 %) |
| HI_1_9_0 | terminateur | 5971 | 5622/5971 (94,2 %) | 231/5971 (3,9 %) | 118/5971 (2,0 %) |
| version-31 | ouverte | 235 | 0/235 (0,0 %) | 0/235 (0,0 %) | 235/235 (100,0 %) |
| version-31 | rejet hors datum | 10745 | 0/10745 (0,0 %) | 0/10745 (0,0 %) | 10745/10745 (100,0 %) |
| version-31 | terminateur | 4682 | 139/4682 (3,0 %) | 3100/4682 (66,2 %) | 1443/4682 (30,8 %) |
| version-33 | ouverte | 458 | 0/458 (0,0 %) | 0/458 (0,0 %) | 458/458 (100,0 %) |
| version-33 | rejet hors datum | 19860 | 0/19860 (0,0 %) | 0/19860 (0,0 %) | 19860/19860 (100,0 %) |
| version-33 | terminateur | 4392 | 424/4392 (9,7 %) | 3001/4392 (68,3 %) | 967/4392 (22,0 %) |
| **corpus** | ouverte | 20294 | 0/20294 (0,0 %) | 0/20294 (0,0 %) | 20294/20294 (100,0 %) |
| **corpus** | rejet hors datum | 259764 | 0/259764 (0,0 %) | 0/259764 (0,0 %) | 259764/259764 (100,0 %) |
| **corpus** | terminateur | 300915 | 279801/300915 (93,0 %) | 12587/300915 (4,2 %) | 8527/300915 (2,8 %) |

## Sorties par rejet contre le bloc de type 1 (item 1.2)

Etat du slot rejete dans le bloc de type 1 de SON chunk ; naissance : le bloc du chunk SUIVANT porte-t-il une allocation de cet eid que le chunk ne portait pas, sans NEW lu ?

| Sortie B | Paquet | Etat au bloc | Naissance | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| rejet hors datum | vue B : sortie par rejet | vide | naissance non lue | 109038 | 1623408 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue | 80247 | 1107708 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · naissance non lue | 14817 | 441 |
| rejet hors datum | vue B : sortie par rejet | vivant | vivant au bloc du chunk | 14763 | 36270 |
| rejet hors datum | vue B : sortie par rejet | vide | aucune allocation | 11360 | 49414 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · naissance non lue | 8838 | 2673 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue, generation 0 | 8725 | 148134 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · naissance non lue, generation 0 | 4139 | 74 |
| rejet hors datum | vue B : sortie par rejet | vide | non mesurable | 3613 | 12521 |
| rejet hors datum | vue B : sortie par rejet | trace | aucune allocation | 2100 | 6670 |
| rejet hors datum | vue B : sortie par rejet | trace | non mesurable | 791 | 6748 |
| rejet hors datum | vue B : sortie par rejet | trace | libere avant le chunk (meme generation) | 396 | 2058 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · non mesurable | 278 | 1 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · non mesurable | 251 | 0 |
| rejet hors datum | vue B : sortie par rejet | vide | realloue sous une autre generation | 141 | 363 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · aucune allocation | 64 | 338 |
| rejet hors datum | vue B : sortie par rejet | vivant | non mesurable | 64 | 184 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu dans le chunk | 45 | 56 |
| rejet hors datum | vue B : sortie par rejet | trace | realloue sous une autre generation | 29 | 94 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu desynchronise · vivant au bloc du chunk | 28 | 0 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu dans le chunk | 9 | 0 |
| rejet hors datum | vue B : sortie par rejet | vivant | aucune allocation | 9 | 54 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu dans le chunk | 5 | 37 |
| rejet hors datum | vue B : sortie par rejet | vivant | realloue sous une autre generation | 5 | 47 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · libere avant le chunk (meme generation) | 3 | 50 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu desynchronise · aucune allocation | 3 | 0 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · aucune allocation | 2 | 25 |
| rejet hors datum | vue B : sortie par rejet | absent | aucune allocation | 1 | 0 |

| Build | Rejets | dont paquet hors cadre | dont ferme | Vivant au bloc | Naissance non lue | Non mesurable |
|---|---|---|---|---|---|---|
| HI_1_10_0 | 83507 | 0/83507 (0,0 %) | 0/83507 (0,0 %) | 10702/83507 (12,8 %) | 64218/83507 (76,9 %) | 787/83507 (0,9 %) |
| HI_1_11_0 | 10229 | 0/10229 (0,0 %) | 0/10229 (0,0 %) | 55/10229 (0,5 %) | 8453/10229 (82,6 %) | 0/10229 (0,0 %) |
| HI_1_12_0 | 3305 | 0/3305 (0,0 %) | 0/3305 (0,0 %) | 0/3305 (0,0 %) | 3064/3305 (92,7 %) | 0/3305 (0,0 %) |
| HI_1_13_0 | 85291 | 0/85291 (0,0 %) | 0/85291 (0,0 %) | 2292/85291 (2,7 %) | 46419/85291 (54,4 %) | 2455/85291 (2,9 %) |
| HI_1_4_1 | 6625 | 0/6625 (0,0 %) | 0/6625 (0,0 %) | 0/6625 (0,0 %) | 5895/6625 (89,0 %) | 340/6625 (5,1 %) |
| HI_1_8_0 | 30378 | 0/30378 (0,0 %) | 0/30378 (0,0 %) | 209/30378 (0,7 %) | 26118/30378 (86,0 %) | 204/30378 (0,7 %) |
| HI_1_9_0 | 9824 | 0/9824 (0,0 %) | 0/9824 (0,0 %) | 137/9824 (1,4 %) | 7690/9824 (78,3 %) | 192/9824 (2,0 %) |
| version-31 | 10745 | 0/10745 (0,0 %) | 0/10745 (0,0 %) | 1/10745 (0,0 %) | 10182/10745 (94,8 %) | 486/10745 (4,5 %) |
| version-33 | 19860 | 0/19860 (0,0 %) | 0/19860 (0,0 %) | 1485/19860 (7,5 %) | 17246/19860 (86,8 %) | 4/19860 (0,0 %) |
| **corpus** | 259764 | 0/259764 (0,0 %) | 0/259764 (0,0 %) | 14881/259764 (5,7 %) | 189285/259764 (72,9 %) | 4468/259764 (1,7 %) |

## Entrees de controle utiles (item 1.3)

Utile = entree `kind 0` qui porte le bloc de 0x68 octets, la seule que le tir continu lit. Denominateur ESTIME = utiles fermees + moyenne par vue C fermee du film x paquets non fermes. PAR CONSTRUCTION, la part estimee d un film egale sa part de vues C fermees (U / (U + U/F x N) = F / (F + N)) : cette estimation ne dit rien de plus que les paquets.

| Build | Paquets | Vues C fermees | Entrees fermees | Utiles fermees | Moyenne par vue C fermee | Denominateur estime | Part estimee | Utiles lues hors fermeture (non prouvees) |
|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 129369 | 22192/129369 (17,2 %) | 347835 | 347835 | 15.67 | 2052300 | 16,9 % | 73350 |
| HI_1_11_0 | 16824 | 4124/16824 (24,5 %) | 74386 | 74386 | 18.04 | 303460 | 24,5 % | 1873 |
| HI_1_12_0 | 21864 | 5879/21864 (26,9 %) | 30418 | 30418 | 5.17 | 113125 | 26,9 % | 24 |
| HI_1_13_0 | 335960 | 226783/335960 (67,5 %) | 1645819 | 1645819 | 7.26 | 2437493 | 67,5 % | 15324 |
| HI_1_4_1 | 11130 | 692/11130 (6,2 %) | 0 | 0 | 0.00 | 0 | - | 2039 |
| HI_1_8_0 | 49696 | 13946/49696 (28,1 %) | 79784 | 79784 | 5.72 | 284307 | 28,1 % | 454 |
| HI_1_9_0 | 17629 | 5622/17629 (31,9 %) | 74102 | 74102 | 13.18 | 232363 | 31,9 % | 1442 |
| version-31 | 17919 | 139/17919 (0,8 %) | 0 | 0 | 0.00 | 0 | - | 4189 |
| version-33 | 28751 | 424/28751 (1,5 %) | 0 | 0 | 0.00 | 0 | - | 3681 |
| **corpus** | 629142 | 279801/629142 (44,5 %) | 2252344 | 2252344 | 8.05 | 5423048 | 41,5 % | 102376 |

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
| HI_1_10_0 | 1808191 | 16/1808191 (0,0 %) | 67 | 8 | 16 | 3 | 2 | 0 |
| HI_1_11_0 | 323844 | 3/323844 (0,0 %) | 31 | 1 | 3 | 1 | 0 | 0 |
| HI_1_12_0 | 169655 | 5/169655 (0,0 %) | 19 | 1 | 5 | 1 | 0 | 0 |
| HI_1_13_0 | 3402089 | 297/3402089 (0,0 %) | 1720 | 78 | 297 | 11 | 7 | 0 |
| HI_1_4_1 | 229986 | 1/229986 (0,0 %) | 2 | 1 | 1 | 1 | 1 | 0 |
| HI_1_8_0 | 342533 | 389/342533 (0,1 %) | 1622 | 85 | 389 | 12 | 1 | 0 |
| HI_1_9_0 | 276556 | 26/276556 (0,0 %) | 109 | 23 | 26 | 1 | 1 | 0 |
| version-31 | 544568 | 4/544568 (0,0 %) | 13 | 1 | 4 | 1 | 1 | 0 |
| version-33 | 565062 | 7/565062 (0,0 %) | 38 | 5 | 7 | 1 | 1 | 0 |
| **corpus** | 7662484 | 748/7662484 (0,0 %) | 3621 | 203 | 748 | 32 | 14 | 0 |

## Lot L0 — fermes au bit pres contre fermes (regles de l ecrivain)

| Build | Fermes au bit | Fermes | Retires | Utiles lus | Utiles fermes au bit | Utiles fermes | Part variable | Fixe | Part fixe |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 29223 | 22192 | 7031 | 1566502 | 303795 | 297230 | 19,0 % | 2423551 | 12,3 % |
| HI_1_11_0 | 4286 | 4124 | 162 | 289358 | 80012 | 79458 | 27,5 % | 383476 | 20,7 % |
| HI_1_12_0 | 5885 | 5879 | 6 | 122669 | 35507 | 35492 | 28,9 % | 148160 | 24,0 % |
| HI_1_13_0 | 228420 | 226783 | 1637 | 2538265 | 2080746 | 2074170 | 81,7 % | 3073267 | 67,5 % |
| HI_1_4_1 | 702 | 692 | 10 | 184473 | 80 | 78 | 0,0 % | 198500 | 0,0 % |
| HI_1_8_0 | 14114 | 13946 | 168 | 271810 | 84165 | 83669 | 30,8 % | 359291 | 23,3 % |
| HI_1_9_0 | 5751 | 5622 | 129 | 248837 | 65700 | 65460 | 26,3 % | 324613 | 20,2 % |
| version-31 | 153 | 139 | 14 | 320107 | 277 | 274 | 0,1 % | 328128 | 0,1 % |
| version-33 | 467 | 424 | 43 | 479511 | 3661 | 3654 | 0,8 % | 519304 | 0,7 % |
| **corpus** | 289001 | 279801 | 9200 | 6021532 | 2653943 | 2639485 | 43,8 % | 7758290 | 34,0 % |

### Corpus : fermetures au bit retirees, par premiere regle

| Regle | Paquets | Utiles |
|---|---|---|
| ecrivain : masque au-dela de l archetype | 4233 | 12068 |
| ecrivain : masque epars non croissant | 129 | 394 |
| ecrivain : ordre de la vue B | 262 | 1025 |
| ecrivain : vue C en-tete cdc04 pose | 6 | 6 |
| ecrivain : vue C kind non nul | 2 | 0 |
| vue B : sortie par rejet | 4568 | 965 |

### Corpus : temoins decales (vue C relue depuis un depart decale de k bits)

| k | Refermes | Part des fermes |
|---|---|---|
| -8 | 1097 | 0,4 % |
| -7 | 1603 | 0,6 % |
| -6 | 3252 | 1,2 % |
| -5 | 4614 | 1,6 % |
| -4 | 14184 | 5,1 % |
| -3 | 10389 | 3,7 % |
| -2 | 10609 | 3,8 % |
| -1 | 10812 | 3,9 % |
| +1 | 23993 | 8,6 % |
| +2 | 23226 | 8,3 % |
| +3 | 16021 | 5,7 % |
| +4 | 14726 | 5,3 % |
| +5 | 14133 | 5,1 % |
| +6 | 13913 | 5,0 % |
| +7 | 13710 | 4,9 % |
| +8 | 0 | 0,0 % |


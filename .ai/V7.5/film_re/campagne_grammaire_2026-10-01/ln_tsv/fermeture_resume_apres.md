# Carte de fermeture des trames delta — 2026-10-04

Films mesures : 20 ; echecs : 0. Contexte d instrument (largeurs d axe lues dans le film, profil par defaut, sans calibration killsource).

Table ECS : `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LN/ecs_table_base.tsv`.

## Par build

| Build | Films | Paquets fermes | Vue A | Vue B | Vue C | Records utiles fermes | Entrees de controle lues | Declencheur (>= 95 %) | Pic memoire max |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 28488/129369 (22,0 %) | 21669/79272 (27,3 %) | 28488/124584 (22,9 %) | 28488/33586 (84,8 %) | 491550/1895882 (25,9 %) | 502842 | non atteint | 319 Mio |
| HI_1_11_0 | 1 | 5497/16824 (32,7 %) | 3537/9073 (39,0 %) | 5497/16295 (33,7 %) | 5497/6314 (87,1 %) | 116119/316670 (36,7 %) | 101003 | non atteint | 118 Mio |
| HI_1_12_0 | 1 | 5924/21864 (27,1 %) | 5249/19134 (27,4 %) | 5924/21797 (27,2 %) | 5924/5984 (99,0 %) | 35828/124415 (28,8 %) | 30679 | non atteint | 72 Mio |
| HI_1_13_0 | 10 | 278753/335960 (83,0 %) | 228961/271063 (84,5 %) | 278753/332668 (83,8 %) | 278753/280767 (99,3 %) | 2613644/2997500 (87,2 %) | 2043486 | non atteint | 222 Mio |
| HI_1_4_1 | 1 | 692/11130 (6,2 %) | 651/6174 (10,5 %) | 692/9840 (7,0 %) | 692/3021 (22,9 %) | 78/184588 (0,0 %) | 0 | non atteint | 204 Mio |
| HI_1_8_0 | 1 | 13946/49696 (28,1 %) | 13146/43640 (30,1 %) | 13946/46437 (30,0 %) | 13946/15317 (91,0 %) | 83669/271810 (30,8 %) | 79784 | non atteint | 151 Mio |
| HI_1_9_0 | 1 | 7242/17629 (41,1 %) | 5272/10304 (51,2 %) | 7242/17046 (42,5 %) | 7242/7837 (92,4 %) | 101868/267100 (38,1 %) | 107159 | non atteint | 120 Mio |
| version-31 | 1 | 139/17919 (0,8 %) | 130/10140 (1,3 %) | 139/15662 (0,9 %) | 139/4700 (3,0 %) | 274/320306 (0,1 %) | 0 | non atteint | 159 Mio |
| version-33 | 1 | 424/28751 (1,5 %) | 402/14259 (2,8 %) | 424/24712 (1,7 %) | 424/4421 (9,6 %) | 3654/479794 (0,8 %) | 0 | non atteint | 222 Mio |

## Causes d arret (premiere cause de chaque paquet non ferme)

Gain potentiel = records utiles LUS et non fermes dans les paquets que la cause arrete : BORNE SUPERIEURE (une autre cause peut suivre ; les records d apres l arret ne sont pas lus du tout).

| Rang | Cause | Archetype | Index | Statut | Usage produit | Paquets bloques | Gain potentiel | Builds |
|---|---|---|---|---|---|---|---|---|
| 1 | vue B : sortie par rejet | - |  | - | - | 227976 | 3048785 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 2 | liste d evenements non localisee | - |  | - | - | 20101 | 0 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 3 | vue C : terminateur hors cadre | - |  | - | - | 15243 | 156941 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 4 | ti=43 device-animation-layer-state | 43 | i35 | non_porte | aucun | 12568 | 70538 | HI_1_10_0, HI_1_12_0, HI_1_8_0 |
| 5 | vue C : kind non porte | - |  | - | - | 3145 | 55844 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 6 | ecrivain : masque au-dela de l archetype | - |  | - | - | 1229 | 5841 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 7 | vue C : bloc 0xbc (desalignement) | - |  | - | - | 1040 | 19830 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 8 | ti=12 managed-navpoint-override-flags | 12 | i16 | non_porte | aucun | 988 | 638 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 9 | ti=43 device-position-group | 43 | i21 | non_porte | aucun | 862 | 13335 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 10 | vue B : fin de payload | - |  | - | - | 756 | 4092 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 11 | ti=10 managed-object-navpoint | 10 | i10, i11, i12, i14, i15, i16, i17, i2, i3, i4, i5, i6, i7, i8, i9 | non_porte | aucun | 536 | 6614 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 12 | ti=12 managed-navpoint-visual-state-groups-component-2 | 12 | i22 | non_porte | aucun | 464 | 6483 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-31, version-33 |
| 13 | ti=12 managed-navpoint-visual-state-groups-component-1 | 12 | i21 | non_porte | aucun | 436 | 5932 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 14 | ti=35 biped-spartan-ability-non-predicted-state | 35 | i58, i59 | partiel | rejeu : grappleLines[] (schema 8) — la ligne blanche du joueur vers son ancre, fenetre [t0,t1] par vie | 277 | 2213 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 15 | ti=56 archetype hors registre | 56 |  | - | - | 246 | 1170 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 16 | ti=58 archetype hors registre | 58 |  | - | - | 174 | 85 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-33 |
| 17 | ti=12 managed-navpoint-visual-state-groups-component-0 | 12 | i20 | non_porte | aucun | 128 | 1360 | HI_1_13_0, HI_1_4_1, version-33 |
| 18 | ti=11 managed-objective-interaction-filter | 11 | i4 | non_porte | aucun | 97 | 804 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0 |
| 19 | ti=18 effect-state-data | 18 | i0, i1, i10, i12, i13, i14, i16, i19, i2, i20, i22, i23, i24, i28, i3, i31, i4, i6, i7, i8, i9 | partiel | aucun | 95 | 250 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31 |
| 20 | ti=43 device-machine-flags | 43 | i39 | non_porte | aucun | 91 | 1363 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, version-31, version-33 |
| 21 | ecrivain : ordre de la vue B | - |  | - | - | 87 | 638 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 22 | ti=0 forge-engine-player-roles | 0 | i18 | non_porte | aucun | 84 | 414 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 23 | ti=51 archetype hors registre | 51 |  | - | - | 79 | 682 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
| 24 | ti=59 archetype hors registre | 59 |  | - | - | 66 | 574 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 25 | ti=10 managed-object-interaction-filter | 10 | i22 | non_porte | aucun | 64 | 875 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 26 | ti=43 device-position-animation-name | 43 | i19 | non_porte | aucun | 52 | 347 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-33 |
| 27 | ti=19 sound-placement-state-data | 19 | i0, i1, i12, i18, i2, i25, i3, i31, i4, i5, i6, i8, i9 | non_porte | aucun | 51 | 267 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
| 28 | ecrivain : masque epars non croissant | - |  | - | - | 49 | 239 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 29 | ti=0 forge-engine-baking-data | 0 | i24, i25, i26 | non_porte | aucun | 48 | 14 | HI_1_10_0, HI_1_11_0, HI_1_13_0 |
| 30 | ti=52 archetype hors registre | 52 |  | - | - | 47 | 295 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
## Carte v2 — « vue C : terminateur hors cadre » ventile (item 1.1)

Sortie de la vue B : comment la boucle de records s est arretee avant la vue C ; vue C : vide (son terminateur seul) ou non ; reste : bits du payload derriere le terminateur de la vue C.

### Par build

| Build | Films | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 3603 | 3603/3603 (100,0 %) | 0/3603 (0,0 %) | 0/3603 (0,0 %) | 0/3603 (0,0 %) | 3086/3603 (85,7 %) | 0/3603 (0,0 %) | 0/3603 (0,0 %) | 3603/3603 (100,0 %) | 0/3603 (0,0 %) | 5863 |
| HI_1_11_0 | 1 | 635 | 635/635 (100,0 %) | 0/635 (0,0 %) | 0/635 (0,0 %) | 0/635 (0,0 %) | 587/635 (92,4 %) | 0/635 (0,0 %) | 0/635 (0,0 %) | 635/635 (100,0 %) | 0/635 (0,0 %) | 1243 |
| HI_1_12_0 | 1 | 54 | 54/54 (100,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 49/54 (90,7 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 54/54 (100,0 %) | 0/54 (0,0 %) | 2 |
| HI_1_13_0 | 10 | 1290 | 1290/1290 (100,0 %) | 0/1290 (0,0 %) | 0/1290 (0,0 %) | 0/1290 (0,0 %) | 1126/1290 (87,3 %) | 0/1290 (0,0 %) | 7/1290 (0,5 %) | 1283/1290 (99,5 %) | 0/1290 (0,0 %) | 7580 |
| HI_1_4_1 | 1 | 2042 | 2042/2042 (100,0 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 169/2042 (8,3 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 2042/2042 (100,0 %) | 0/2042 (0,0 %) | 39302 |
| HI_1_8_0 | 1 | 1007 | 1007/1007 (100,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 899/1007 (89,3 %) | 0/1007 (0,0 %) | 96/1007 (9,5 %) | 911/1007 (90,5 %) | 0/1007 (0,0 %) | 5386 |
| HI_1_9_0 | 1 | 479 | 479/479 (100,0 %) | 0/479 (0,0 %) | 0/479 (0,0 %) | 0/479 (0,0 %) | 401/479 (83,7 %) | 0/479 (0,0 %) | 0/479 (0,0 %) | 479/479 (100,0 %) | 0/479 (0,0 %) | 1721 |
| version-31 | 1 | 3111 | 3111/3111 (100,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 94/3111 (3,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 3111/3111 (100,0 %) | 0/3111 (0,0 %) | 41137 |
| version-33 | 1 | 3022 | 3022/3022 (100,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 213/3022 (7,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 3022/3022 (100,0 %) | 0/3022 (0,0 %) | 54707 |
| **corpus** | 20 | 15243 | 15243/15243 (100,0 %) | 0/15243 (0,0 %) | 0/15243 (0,0 %) | 0/15243 (0,0 %) | 6624/15243 (43,5 %) | 0/15243 (0,0 %) | 103/15243 (0,7 %) | 15140/15243 (99,3 %) | 0/15243 (0,0 %) | 156941 |

### Par film

| Film | Build | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 084a804d | HI_1_10_0 | 785 | 785/785 (100,0 %) | 0/785 (0,0 %) | 0/785 (0,0 %) | 0/785 (0,0 %) | 691/785 (88,0 %) | 0/785 (0,0 %) | 0/785 (0,0 %) | 785/785 (100,0 %) | 0/785 (0,0 %) | 322 |
| 111fa685 | HI_1_10_0 | 735 | 735/735 (100,0 %) | 0/735 (0,0 %) | 0/735 (0,0 %) | 0/735 (0,0 %) | 487/735 (66,3 %) | 0/735 (0,0 %) | 0/735 (0,0 %) | 735/735 (100,0 %) | 0/735 (0,0 %) | 3726 |
| 1c4c63c2 | HI_1_10_0 | 2083 | 2083/2083 (100,0 %) | 0/2083 (0,0 %) | 0/2083 (0,0 %) | 0/2083 (0,0 %) | 1908/2083 (91,6 %) | 0/2083 (0,0 %) | 0/2083 (0,0 %) | 2083/2083 (100,0 %) | 0/2083 (0,0 %) | 1815 |
| e5adf7b2 | HI_1_11_0 | 635 | 635/635 (100,0 %) | 0/635 (0,0 %) | 0/635 (0,0 %) | 0/635 (0,0 %) | 587/635 (92,4 %) | 0/635 (0,0 %) | 0/635 (0,0 %) | 635/635 (100,0 %) | 0/635 (0,0 %) | 1243 |
| bcb6d393 | HI_1_12_0 | 54 | 54/54 (100,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 49/54 (90,7 %) | 0/54 (0,0 %) | 0/54 (0,0 %) | 54/54 (100,0 %) | 0/54 (0,0 %) | 2 |
| 0797ce72 | HI_1_13_0 | 977 | 977/977 (100,0 %) | 0/977 (0,0 %) | 0/977 (0,0 %) | 0/977 (0,0 %) | 860/977 (88,0 %) | 0/977 (0,0 %) | 6/977 (0,6 %) | 971/977 (99,4 %) | 0/977 (0,0 %) | 5987 |
| 396cfc92 | HI_1_13_0 | 13 | 13/13 (100,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 11/13 (84,6 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 13/13 (100,0 %) | 0/13 (0,0 %) | 109 |
| 4f77afc1 | HI_1_13_0 | 37 | 37/37 (100,0 %) | 0/37 (0,0 %) | 0/37 (0,0 %) | 0/37 (0,0 %) | 30/37 (81,1 %) | 0/37 (0,0 %) | 0/37 (0,0 %) | 37/37 (100,0 %) | 0/37 (0,0 %) | 979 |
| 51ebbc0f | HI_1_13_0 | 6 | 6/6 (100,0 %) | 0/6 (0,0 %) | 0/6 (0,0 %) | 0/6 (0,0 %) | 3/6 (50,0 %) | 0/6 (0,0 %) | 0/6 (0,0 %) | 6/6 (100,0 %) | 0/6 (0,0 %) | 27 |
| bf15f7ab | HI_1_13_0 | 38 | 38/38 (100,0 %) | 0/38 (0,0 %) | 0/38 (0,0 %) | 0/38 (0,0 %) | 6/38 (15,8 %) | 0/38 (0,0 %) | 1/38 (2,6 %) | 37/38 (97,4 %) | 0/38 (0,0 %) | 257 |
| bfecd02b | HI_1_13_0 | 10 | 10/10 (100,0 %) | 0/10 (0,0 %) | 0/10 (0,0 %) | 0/10 (0,0 %) | 8/10 (80,0 %) | 0/10 (0,0 %) | 0/10 (0,0 %) | 10/10 (100,0 %) | 0/10 (0,0 %) | 90 |
| c75f33b8 | HI_1_13_0 | 3 | 3/3 (100,0 %) | 0/3 (0,0 %) | 0/3 (0,0 %) | 0/3 (0,0 %) | 3/3 (100,0 %) | 0/3 (0,0 %) | 0/3 (0,0 %) | 3/3 (100,0 %) | 0/3 (0,0 %) | 29 |
| d9781168 | HI_1_13_0 | 194 | 194/194 (100,0 %) | 0/194 (0,0 %) | 0/194 (0,0 %) | 0/194 (0,0 %) | 193/194 (99,5 %) | 0/194 (0,0 %) | 0/194 (0,0 %) | 194/194 (100,0 %) | 0/194 (0,0 %) | 26 |
| f75e7053 | HI_1_13_0 | 0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0 |
| fb1a1a72 | HI_1_13_0 | 12 | 12/12 (100,0 %) | 0/12 (0,0 %) | 0/12 (0,0 %) | 0/12 (0,0 %) | 12/12 (100,0 %) | 0/12 (0,0 %) | 0/12 (0,0 %) | 12/12 (100,0 %) | 0/12 (0,0 %) | 76 |
| a521164d | HI_1_4_1 | 2042 | 2042/2042 (100,0 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 169/2042 (8,3 %) | 0/2042 (0,0 %) | 0/2042 (0,0 %) | 2042/2042 (100,0 %) | 0/2042 (0,0 %) | 39302 |
| 60ae07c4 | HI_1_8_0 | 1007 | 1007/1007 (100,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 0/1007 (0,0 %) | 899/1007 (89,3 %) | 0/1007 (0,0 %) | 96/1007 (9,5 %) | 911/1007 (90,5 %) | 0/1007 (0,0 %) | 5386 |
| 11de8353 | HI_1_9_0 | 479 | 479/479 (100,0 %) | 0/479 (0,0 %) | 0/479 (0,0 %) | 0/479 (0,0 %) | 401/479 (83,7 %) | 0/479 (0,0 %) | 0/479 (0,0 %) | 479/479 (100,0 %) | 0/479 (0,0 %) | 1721 |
| 50247b26 | version-31 | 3111 | 3111/3111 (100,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 94/3111 (3,0 %) | 0/3111 (0,0 %) | 0/3111 (0,0 %) | 3111/3111 (100,0 %) | 0/3111 (0,0 %) | 41137 |
| a349fea8 | version-33 | 3022 | 3022/3022 (100,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 213/3022 (7,0 %) | 0/3022 (0,0 %) | 0/3022 (0,0 %) | 3022/3022 (100,0 %) | 0/3022 (0,0 %) | 54707 |

### Corpus : sortie x vue C x reste

| Sortie B | Vue C | Reste | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| terminateur | 1 entree(s) | >= 64 | 7645 | 122240 |
| terminateur | vide | >= 64 | 6558 | 18247 |
| terminateur | 2 entree(s) | >= 64 | 512 | 13711 |
| terminateur | 0 entree (kind 3 seul) | >= 64 | 421 | 2285 |
| terminateur | vide | 8-63 | 66 | 393 |
| terminateur | 0 entree (kind 3 seul) | 8-63 | 29 | 0 |
| terminateur | 1 entree(s) | 8-63 | 8 | 52 |
| terminateur | 3 entree(s) | >= 64 | 4 | 13 |

### Corpus : dernier record et dernier composant lus avant la fin de la vue B

| Sortie B | Dernier lu | Paquets | Utiles en jeu |
|---|---|---|---|
| terminateur | DELTA ti=40 i25 unit-command-tick-component | 1688 | 13147 |
| terminateur | DELTA ti=37 i3 object-angular-velocity-component | 1592 | 30943 |
| terminateur | DEL | 1180 | 15218 |
| terminateur | aucun record | 1094 | 0 |
| terminateur | NEW ti=41 sans composant | 1079 | 15 |
| terminateur | DELTA ti=10 i26 managed-object-rtpc-component | 1024 | 23905 |
| terminateur | DELTA ti=4 i0 high-frequency | 1006 | 10 |
| terminateur | DELTA ti=43 i18 device-position-component | 828 | 344 |
| terminateur | DELTA ti=42 i3 object-angular-velocity-component | 698 | 18487 |
| terminateur | DELTA ti=21 i16 flock-position-component | 341 | 1 |
| terminateur | DELTA ti=42 i2 object-forward-and-up-component | 310 | 8301 |
| terminateur | DELTA ti=37 i29 equipment-command-tick-component | 303 | 7021 |
| terminateur | NEW ti=41 i15 object-low-frequency-component | 240 | 9 |
| terminateur | DELTA ti=37 i1 object-translational-velocity-component | 237 | 6979 |
| terminateur | DELTA ti=37 i2 object-forward-and-up-component | 217 | 5204 |
| terminateur | DELTA ti=42 i1 object-translational-velocity-component | 195 | 5401 |
| terminateur | DELTA ti=3 sans composant | 190 | 1 |
| terminateur | NEW ti=42 sans composant | 185 | 7 |
| terminateur | DELTA ti=20 i1 spawn-filter-weight-component | 181 | 2172 |
| terminateur | DELTA ti=47 i1 managed-object-networked-splash-message-dynamic-component | 162 | 3231 |
| terminateur | DELTA ti=20 i0 spawn-filter-type-component | 126 | 1550 |
| terminateur | DELTA ti=12 i14 managed-navpoint-radial-progress | 90 | 2260 |
| terminateur | NEW ti=41 i21 projectile-deceleration-disabled-state | 79 | 51 |
| terminateur | DELTA ti=37 i28 equipment-tracked-object-handles-stack-component | 72 | 1215 |
| terminateur | NEW ti=41 i18 projectile-at-rest-state | 70 | 70 |
| terminateur | DELTA ti=38 i3 object-angular-velocity-component | 59 | 1534 |
| terminateur | NEW ti=40 sans composant | 57 | 21 |
| terminateur | DELTA ti=35 i25 unit-command-tick-component | 55 | 85 |
| terminateur | NEW ti=35 i57 biped-spartan-ability-component | 54 | 54 |
| terminateur | NEW ti=41 i19 projectile-tether-state | 54 | 42 |

## Sorties de la vue B, tous paquets qui l atteignent

| Build | Sortie B | Paquets | Fermes | Hors cadre | Autres causes |
|---|---|---|---|---|---|
| HI_1_10_0 | ouverte | 1690 | 0/1690 (0,0 %) | 0/1690 (0,0 %) | 1690/1690 (100,0 %) |
| HI_1_10_0 | rejet hors datum | 89308 | 0/89308 (0,0 %) | 0/89308 (0,0 %) | 89308/89308 (100,0 %) |
| HI_1_10_0 | terminateur | 33586 | 28488/33586 (84,8 %) | 3603/33586 (10,7 %) | 1495/33586 (4,5 %) |
| HI_1_11_0 | ouverte | 235 | 0/235 (0,0 %) | 0/235 (0,0 %) | 235/235 (100,0 %) |
| HI_1_11_0 | rejet hors datum | 9746 | 0/9746 (0,0 %) | 0/9746 (0,0 %) | 9746/9746 (100,0 %) |
| HI_1_11_0 | terminateur | 6314 | 5497/6314 (87,1 %) | 635/6314 (10,1 %) | 182/6314 (2,9 %) |
| HI_1_12_0 | ouverte | 12730 | 0/12730 (0,0 %) | 0/12730 (0,0 %) | 12730/12730 (100,0 %) |
| HI_1_12_0 | rejet hors datum | 3083 | 0/3083 (0,0 %) | 0/3083 (0,0 %) | 3083/3083 (100,0 %) |
| HI_1_12_0 | terminateur | 5984 | 5924/5984 (99,0 %) | 54/5984 (0,9 %) | 6/5984 (0,1 %) |
| HI_1_13_0 | ouverte | 2871 | 0/2871 (0,0 %) | 0/2871 (0,0 %) | 2871/2871 (100,0 %) |
| HI_1_13_0 | rejet hors datum | 49030 | 0/49030 (0,0 %) | 0/49030 (0,0 %) | 49030/49030 (100,0 %) |
| HI_1_13_0 | terminateur | 280767 | 278753/280767 (99,3 %) | 1290/280767 (0,5 %) | 724/280767 (0,3 %) |
| HI_1_4_1 | ouverte | 143 | 0/143 (0,0 %) | 0/143 (0,0 %) | 143/143 (100,0 %) |
| HI_1_4_1 | rejet hors datum | 6676 | 0/6676 (0,0 %) | 0/6676 (0,0 %) | 6676/6676 (100,0 %) |
| HI_1_4_1 | terminateur | 3021 | 692/3021 (22,9 %) | 2042/3021 (67,6 %) | 287/3021 (9,5 %) |
| HI_1_8_0 | ouverte | 741 | 0/741 (0,0 %) | 0/741 (0,0 %) | 741/741 (100,0 %) |
| HI_1_8_0 | rejet hors datum | 30379 | 0/30379 (0,0 %) | 0/30379 (0,0 %) | 30379/30379 (100,0 %) |
| HI_1_8_0 | terminateur | 15317 | 13946/15317 (91,0 %) | 1007/15317 (6,6 %) | 364/15317 (2,4 %) |
| HI_1_9_0 | ouverte | 242 | 0/242 (0,0 %) | 0/242 (0,0 %) | 242/242 (100,0 %) |
| HI_1_9_0 | rejet hors datum | 8967 | 0/8967 (0,0 %) | 0/8967 (0,0 %) | 8967/8967 (100,0 %) |
| HI_1_9_0 | terminateur | 7837 | 7242/7837 (92,4 %) | 479/7837 (6,1 %) | 116/7837 (1,5 %) |
| version-31 | ouverte | 151 | 0/151 (0,0 %) | 0/151 (0,0 %) | 151/151 (100,0 %) |
| version-31 | rejet hors datum | 10811 | 0/10811 (0,0 %) | 0/10811 (0,0 %) | 10811/10811 (100,0 %) |
| version-31 | terminateur | 4700 | 139/4700 (3,0 %) | 3111/4700 (66,2 %) | 1450/4700 (30,9 %) |
| version-33 | ouverte | 315 | 0/315 (0,0 %) | 0/315 (0,0 %) | 315/315 (100,0 %) |
| version-33 | rejet hors datum | 19976 | 0/19976 (0,0 %) | 0/19976 (0,0 %) | 19976/19976 (100,0 %) |
| version-33 | terminateur | 4421 | 424/4421 (9,6 %) | 3022/4421 (68,4 %) | 975/4421 (22,1 %) |
| **corpus** | ouverte | 19118 | 0/19118 (0,0 %) | 0/19118 (0,0 %) | 19118/19118 (100,0 %) |
| **corpus** | rejet hors datum | 227976 | 0/227976 (0,0 %) | 0/227976 (0,0 %) | 227976/227976 (100,0 %) |
| **corpus** | terminateur | 361947 | 341105/361947 (94,2 %) | 15243/361947 (4,2 %) | 5599/361947 (1,5 %) |

## Sorties par rejet contre le bloc de type 1 (item 1.2)

Etat du slot rejete dans le bloc de type 1 de SON chunk ; naissance : le bloc du chunk SUIVANT porte-t-il une allocation de cet eid que le chunk ne portait pas, sans NEW lu ?

| Sortie B | Paquet | Etat au bloc | Naissance | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| rejet hors datum | vue B : sortie par rejet | vide | naissance non lue | 118756 | 1856594 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue | 62143 | 909813 |
| rejet hors datum | vue B : sortie par rejet | vivant | vivant au bloc du chunk | 17608 | 46677 |
| rejet hors datum | vue B : sortie par rejet | vide | aucune allocation | 10411 | 51620 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue, generation 0 | 5537 | 93575 |
| rejet hors datum | vue B : sortie par rejet | vide | non mesurable | 3917 | 14605 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · naissance non lue | 2669 | 28276 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · naissance non lue | 2438 | 25689 |
| rejet hors datum | vue B : sortie par rejet | trace | aucune allocation | 2434 | 7317 |
| rejet hors datum | vue B : sortie par rejet | trace | non mesurable | 1016 | 11101 |
| rejet hors datum | vue B : sortie par rejet | trace | libere avant le chunk (meme generation) | 699 | 2186 |
| rejet hors datum | vue B : sortie par rejet | vide | realloue sous une autre generation | 107 | 388 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · aucune allocation | 64 | 360 |
| rejet hors datum | vue B : sortie par rejet | vivant | non mesurable | 64 | 184 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu dans le chunk | 39 | 56 |
| rejet hors datum | vue B : sortie par rejet | trace | realloue sous une autre generation | 26 | 80 |
| rejet hors datum | vue B : sortie par rejet | vivant | aucune allocation | 10 | 54 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · aucune allocation | 9 | 25 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu dans le chunk | 8 | 40 |
| rejet hors datum | vue B : sortie par rejet | vivant | realloue sous une autre generation | 6 | 48 |
| rejet hors datum | vue B : sortie par rejet | absent | aucune allocation | 5 | 1 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · libere avant le chunk (meme generation) | 4 | 50 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · realloue sous une autre generation | 2 | 24 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu dans le chunk | 2 | 0 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · realloue sous une autre generation | 1 | 0 |
| rejet hors datum | vue B : sortie par rejet | vivant | naissance non lue | 1 | 22 |

| Build | Rejets | dont paquet hors cadre | dont ferme | Vivant au bloc | Naissance non lue | Non mesurable |
|---|---|---|---|---|---|---|
| HI_1_10_0 | 89308 | 0/89308 (0,0 %) | 0/89308 (0,0 %) | 13291/89308 (14,9 %) | 65469/89308 (73,3 %) | 1095/89308 (1,2 %) |
| HI_1_11_0 | 9746 | 0/9746 (0,0 %) | 0/9746 (0,0 %) | 58/9746 (0,6 %) | 8133/9746 (83,4 %) | 0/9746 (0,0 %) |
| HI_1_12_0 | 3083 | 0/3083 (0,0 %) | 0/3083 (0,0 %) | 0/3083 (0,0 %) | 2843/3083 (92,2 %) | 0/3083 (0,0 %) |
| HI_1_13_0 | 49030 | 0/49030 (0,0 %) | 0/49030 (0,0 %) | 2505/49030 (5,1 %) | 37823/49030 (77,1 %) | 2673/49030 (5,5 %) |
| HI_1_4_1 | 6676 | 0/6676 (0,0 %) | 0/6676 (0,0 %) | 0/6676 (0,0 %) | 5932/6676 (88,9 %) | 342/6676 (5,1 %) |
| HI_1_8_0 | 30379 | 0/30379 (0,0 %) | 0/30379 (0,0 %) | 209/30379 (0,7 %) | 26118/30379 (86,0 %) | 204/30379 (0,7 %) |
| HI_1_9_0 | 8967 | 0/8967 (0,0 %) | 0/8967 (0,0 %) | 141/8967 (1,6 %) | 7026/8967 (78,4 %) | 193/8967 (2,2 %) |
| version-31 | 10811 | 0/10811 (0,0 %) | 0/10811 (0,0 %) | 1/10811 (0,0 %) | 10236/10811 (94,7 %) | 486/10811 (4,5 %) |
| version-33 | 19976 | 0/19976 (0,0 %) | 0/19976 (0,0 %) | 1486/19976 (7,4 %) | 17320/19976 (86,7 %) | 4/19976 (0,0 %) |
| **corpus** | 227976 | 0/227976 (0,0 %) | 0/227976 (0,0 %) | 17691/227976 (7,8 %) | 180900/227976 (79,4 %) | 4997/227976 (2,2 %) |

## Entrees de controle utiles (item 1.3)

Utile = entree `kind 0` qui porte le bloc de 0x68 octets, la seule que le tir continu lit. Denominateur ESTIME = utiles fermees + moyenne par vue C fermee du film x paquets non fermes. PAR CONSTRUCTION, la part estimee d un film egale sa part de vues C fermees (U / (U + U/F x N) = F / (F + N)) : cette estimation ne dit rien de plus que les paquets.

| Build | Paquets | Vues C fermees | Entrees fermees | Utiles fermees | Moyenne par vue C fermee | Denominateur estime | Part estimee | Utiles lues hors fermeture (non prouvees) |
|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 129369 | 28488/129369 (22,0 %) | 502842 | 502842 | 17.65 | 2306059 | 21,8 % | 21129 |
| HI_1_11_0 | 16824 | 5497/16824 (32,7 %) | 101003 | 101003 | 18.37 | 309128 | 32,7 % | 582 |
| HI_1_12_0 | 21864 | 5924/21864 (27,1 %) | 30679 | 30679 | 5.18 | 113229 | 27,1 % | 16 |
| HI_1_13_0 | 335960 | 278753/335960 (83,0 %) | 2043486 | 2043486 | 7.33 | 2477967 | 82,5 % | 4787 |
| HI_1_4_1 | 11130 | 692/11130 (6,2 %) | 0 | 0 | 0.00 | 0 | - | 2049 |
| HI_1_8_0 | 49696 | 13946/49696 (28,1 %) | 79784 | 79784 | 5.72 | 284307 | 28,1 % | 454 |
| HI_1_9_0 | 17629 | 7242/17629 (41,1 %) | 107159 | 107159 | 14.80 | 260854 | 41,1 % | 878 |
| version-31 | 17919 | 139/17919 (0,8 %) | 0 | 0 | 0.00 | 0 | - | 4205 |
| version-33 | 28751 | 424/28751 (1,5 %) | 0 | 0 | 0.00 | 0 | - | 3699 |
| **corpus** | 629142 | 341105/629142 (54,2 %) | 2864953 | 2864953 | 8.40 | 5751544 | 49,8 % | 37799 |

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
| HI_1_10_0 | 2205537 | 56/2205537 (0,0 %) | 263 | 28 | 56 | 3 | 2 | 0 |
| HI_1_11_0 | 358930 | 13/358930 (0,0 %) | 71 | 8 | 13 | 1 | 0 | 0 |
| HI_1_12_0 | 172295 | 6/172295 (0,0 %) | 22 | 1 | 6 | 1 | 0 | 0 |
| HI_1_13_0 | 4156215 | 287/4156215 (0,0 %) | 1716 | 72 | 287 | 11 | 7 | 0 |
| HI_1_4_1 | 230281 | 1/230281 (0,0 %) | 2 | 1 | 1 | 1 | 1 | 0 |
| HI_1_8_0 | 342533 | 392/342533 (0,1 %) | 1623 | 89 | 392 | 12 | 1 | 0 |
| HI_1_9_0 | 301296 | 45/301296 (0,0 %) | 241 | 34 | 45 | 1 | 1 | 0 |
| version-31 | 545602 | 4/545602 (0,0 %) | 13 | 1 | 4 | 1 | 1 | 0 |
| version-33 | 565676 | 7/565676 (0,0 %) | 38 | 5 | 7 | 1 | 1 | 0 |
| **corpus** | 8878365 | 811/8878365 (0,0 %) | 3989 | 239 | 811 | 32 | 14 | 0 |

## Lot L0 — fermes au bit pres contre fermes (regles de l ecrivain)

| Build | Fermes au bit | Fermes | Retires | Utiles lus | Utiles fermes au bit | Utiles fermes | Part variable | Fixe | Part fixe |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 30331 | 28488 | 1843 | 1895882 | 495020 | 491550 | 25,9 % | 2423551 | 20,3 % |
| HI_1_11_0 | 5548 | 5497 | 51 | 316670 | 116313 | 116119 | 36,7 % | 383476 | 30,3 % |
| HI_1_12_0 | 5926 | 5924 | 2 | 124415 | 35839 | 35828 | 28,8 % | 148160 | 24,2 % |
| HI_1_13_0 | 279203 | 278753 | 450 | 2997500 | 2616495 | 2613644 | 87,2 % | 3082896 | 84,8 % |
| HI_1_4_1 | 702 | 692 | 10 | 184588 | 80 | 78 | 0,0 % | 198500 | 0,0 % |
| HI_1_8_0 | 14114 | 13946 | 168 | 271810 | 84165 | 83669 | 30,8 % | 359291 | 23,3 % |
| HI_1_9_0 | 7307 | 7242 | 65 | 267100 | 102182 | 101868 | 38,1 % | 324613 | 31,4 % |
| version-31 | 153 | 139 | 14 | 320306 | 277 | 274 | 0,1 % | 328128 | 0,1 % |
| version-33 | 469 | 424 | 45 | 479794 | 3663 | 3654 | 0,8 % | 519304 | 0,7 % |
| **corpus** | 343753 | 341105 | 2648 | 6858065 | 3454034 | 3446684 | 50,3 % | 7767919 | 44,4 % |

### Corpus : fermetures au bit retirees, par premiere regle

| Regle | Paquets | Utiles |
|---|---|---|
| ecrivain : masque au-dela de l archetype | 1229 | 5841 |
| ecrivain : masque epars non croissant | 49 | 239 |
| ecrivain : ordre de la vue B | 87 | 638 |
| ecrivain : vue C en-tete cdc04 pose | 2 | 6 |
| vue B : sortie par rejet | 1281 | 626 |

### Corpus : temoins decales (vue C relue depuis un depart decale de k bits)

| k | Refermes | Part des fermes |
|---|---|---|
| -8 | 1164 | 0,3 % |
| -7 | 1564 | 0,5 % |
| -6 | 3145 | 0,9 % |
| -5 | 4083 | 1,2 % |
| -4 | 13995 | 4,1 % |
| -3 | 9778 | 2,9 % |
| -2 | 9992 | 2,9 % |
| -1 | 10193 | 3,0 % |
| +1 | 24662 | 7,2 % |
| +2 | 24036 | 7,0 % |
| +3 | 16976 | 5,0 % |
| +4 | 15765 | 4,6 % |
| +5 | 15196 | 4,5 % |
| +6 | 14982 | 4,4 % |
| +7 | 14781 | 4,3 % |
| +8 | 0 | 0,0 % |


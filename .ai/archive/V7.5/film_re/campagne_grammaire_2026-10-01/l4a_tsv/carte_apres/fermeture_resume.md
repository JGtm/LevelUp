# Carte de fermeture des trames delta — 2026-10-03

Films mesures : 20 ; echecs : 0. Contexte d instrument (largeurs d axe lues dans le film, profil par defaut, sans calibration killsource).

Table ECS : `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L4a/ecs_table_apres.tsv`.

## Par build

| Build | Films | Paquets fermes | Vue A | Vue B | Vue C | Records utiles fermes | Entrees de controle lues | Declencheur (>= 95 %) | Pic memoire max |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 22171/129369 (17,1 %) | 16719/79272 (21,1 %) | 22171/111266 (19,9 %) | 22171/26286 (84,3 %) | 297268/1559136 (19,1 %) | 347690 | non atteint | 326 Mio |
| HI_1_11_0 | 1 | 4146/16824 (24,6 %) | 2966/9073 (32,7 %) | 4146/15098 (27,5 %) | 4146/4629 (89,6 %) | 80066/289562 (27,7 %) | 74842 | non atteint | 117 Mio |
| HI_1_12_0 | 1 | 5830/21864 (26,7 %) | 5201/19134 (27,2 %) | 5830/21581 (27,0 %) | 5830/5886 (99,0 %) | 35143/121628 (28,9 %) | 30125 | non atteint | 69 Mio |
| HI_1_13_0 | 10 | 224924/335960 (66,9 %) | 191556/271063 (70,7 %) | 224924/319947 (70,3 %) | 224924/228694 (98,4 %) | 2048856/2516040 (81,4 %) | 1629458 | non atteint | 222 Mio |
| HI_1_4_1 | 1 | 692/11130 (6,2 %) | 651/6174 (10,5 %) | 692/9839 (7,0 %) | 692/2992 (23,1 %) | 78/181722 (0,0 %) | 0 | non atteint | 218 Mio |
| HI_1_8_0 | 1 | 13802/49696 (27,8 %) | 13002/43640 (29,8 %) | 13802/46436 (29,7 %) | 13802/15155 (91,1 %) | 82730/268352 (30,8 %) | 78935 | non atteint | 144 Mio |
| HI_1_9_0 | 1 | 5619/17629 (31,9 %) | 4407/10304 (42,8 %) | 5619/15990 (35,1 %) | 5619/5964 (94,2 %) | 65391/248382 (26,3 %) | 74039 | non atteint | 116 Mio |
| version-31 | 1 | 139/17919 (0,8 %) | 130/10140 (1,3 %) | 139/15662 (0,9 %) | 139/4692 (3,0 %) | 274/319252 (0,1 %) | 0 | non atteint | 163 Mio |
| version-33 | 1 | 420/28751 (1,5 %) | 398/14259 (2,8 %) | 420/24712 (1,7 %) | 420/4357 (9,6 %) | 3595/469728 (0,8 %) | 0 | non atteint | 235 Mio |

## Causes d arret (premiere cause de chaque paquet non ferme)

Gain potentiel = records utiles LUS et non fermes dans les paquets que la cause arrete : BORNE SUPERIEURE (une autre cause peut suivre ; les records d apres l arret ne sont pas lus du tout).

| Rang | Cause | Archetype | Index | Statut | Usage produit | Paquets bloques | Gain potentiel | Builds |
|---|---|---|---|---|---|---|---|---|
| 1 | vue B : sortie par rejet | - |  | - | - | 259234 | 2999628 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 2 | liste d evenements non localisee | - |  | - | - | 48611 | 0 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 3 | vue C : terminateur hors cadre | - |  | - | - | 12585 | 154719 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 4 | ti=43 device-animation-layer-state | 43 | i35 | non_porte | aucun | 12114 | 67608 | HI_1_10_0, HI_1_12_0, HI_1_8_0 |
| 5 | ecrivain : masque au-dela de l archetype | - |  | - | - | 4097 | 11572 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 6 | ti=2 managed-engine-timers | 2 | i15 | non_porte | aucun | 3996 | 28 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 7 | vue C : kind non porte | - |  | - | - | 2868 | 54384 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 8 | ti=3 low-frequency | 3 | i0 | non_porte | aucun | 1187 | 135 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 9 | vue C : bloc 0xbc (desalignement) | - |  | - | - | 959 | 19539 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 10 | ti=48 forge-player-data-edited-objects-ids | 48 | i0 | non_porte | aucun | 931 | 8143 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 11 | vue B : fin de payload | - |  | - | - | 610 | 3132 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 12 | ti=43 device-position-group | 43 | i21 | non_porte | aucun | 607 | 10522 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 13 | ti=10 managed-object-navpoint | 10 | i10, i11, i12, i13, i15, i16, i17, i2, i3, i4, i5, i6, i7, i8 | non_porte | aucun | 346 | 4335 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 14 | ti=12 managed-navpoint-visual-state-groups-component-2 | 12 | i22 | non_porte | aucun | 343 | 5066 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-31, version-33 |
| 15 | ti=12 managed-navpoint-visual-state-groups-component-1 | 12 | i21 | non_porte | aucun | 334 | 4723 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 16 | ti=35 biped-spartan-ability-non-predicted-state | 35 | i58, i59 | partiel | rejeu : grappleLines[] (schema 8) — la ligne blanche du joueur vers son ancre, fenetre [t0,t1] par vie | 240 | 2008 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 17 | ti=56 archetype hors registre | 56 |  | - | - | 231 | 1141 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 18 | ecrivain : ordre de la vue B | - |  | - | - | 226 | 1011 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 19 | ecrivain : masque epars non croissant | - |  | - | - | 123 | 391 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 20 | ti=12 managed-navpoint-visual-state-groups-component-0 | 12 | i20 | non_porte | aucun | 115 | 1258 | HI_1_13_0, HI_1_4_1, version-33 |
| 21 | ti=12 managed-navpoint-override-flags | 12 | i16 | non_porte | aucun | 88 | 454 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 22 | ti=43 device-machine-flags | 43 | i39 | non_porte | aucun | 88 | 1266 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, version-31, version-33 |
| 23 | ti=2 matchflow-isplaying-flags | 2 | i17 | non_porte | aucun | 85 | 13 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 24 | ti=11 managed-objective-interaction-filter | 11 | i4 | non_porte | aucun | 83 | 656 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0 |
| 25 | ti=18 effect-state-data | 18 | i0, i1, i10, i12, i13, i19, i2, i22, i23, i28, i3, i31, i4, i6, i7, i8, i9 | partiel | aucun | 68 | 261 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31 |
| 26 | ti=2 game-engine-soft-ceilings | 2 | i11 | non_porte | aucun | 67 | 420 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 27 | ti=51 archetype hors registre | 51 |  | - | - | 56 | 493 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
| 28 | ti=10 managed-object-interaction-filter | 10 | i22 | non_porte | aucun | 55 | 762 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 29 | ti=59 archetype hors registre | 59 |  | - | - | 54 | 439 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 30 | vue C : debordement | - |  | - | - | 47 | 199 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
## Carte v2 — « vue C : terminateur hors cadre » ventile (item 1.1)

Sortie de la vue B : comment la boucle de records s est arretee avant la vue C ; vue C : vide (son terminateur seul) ou non ; reste : bits du payload derriere le terminateur de la vue C.

### Par build

| Build | Films | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 772 | 772/772 (100,0 %) | 0/772 (0,0 %) | 0/772 (0,0 %) | 0/772 (0,0 %) | 563/772 (72,9 %) | 0/772 (0,0 %) | 0/772 (0,0 %) | 772/772 (100,0 %) | 0/772 (0,0 %) | 4121 |
| HI_1_11_0 | 1 | 291 | 291/291 (100,0 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 268/291 (92,1 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 291/291 (100,0 %) | 0/291 (0,0 %) | 1045 |
| HI_1_12_0 | 1 | 49 | 49/49 (100,0 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 44/49 (89,8 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 49/49 (100,0 %) | 0/49 (0,0 %) | 1 |
| HI_1_13_0 | 10 | 2144 | 2144/2144 (100,0 %) | 0/2144 (0,0 %) | 0/2144 (0,0 %) | 0/2144 (0,0 %) | 1685/2144 (78,6 %) | 0/2144 (0,0 %) | 9/2144 (0,4 %) | 2135/2144 (99,6 %) | 0/2144 (0,0 %) | 9021 |
| HI_1_4_1 | 1 | 2018 | 2018/2018 (100,0 %) | 0/2018 (0,0 %) | 0/2018 (0,0 %) | 0/2018 (0,0 %) | 168/2018 (8,3 %) | 0/2018 (0,0 %) | 0/2018 (0,0 %) | 2018/2018 (100,0 %) | 0/2018 (0,0 %) | 38679 |
| HI_1_8_0 | 1 | 994 | 994/994 (100,0 %) | 0/994 (0,0 %) | 0/994 (0,0 %) | 0/994 (0,0 %) | 887/994 (89,2 %) | 0/994 (0,0 %) | 96/994 (9,7 %) | 898/994 (90,3 %) | 0/994 (0,0 %) | 5301 |
| HI_1_9_0 | 1 | 228 | 228/228 (100,0 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 202/228 (88,6 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 228/228 (100,0 %) | 0/228 (0,0 %) | 1855 |
| version-31 | 1 | 3107 | 3107/3107 (100,0 %) | 0/3107 (0,0 %) | 0/3107 (0,0 %) | 0/3107 (0,0 %) | 93/3107 (3,0 %) | 0/3107 (0,0 %) | 0/3107 (0,0 %) | 3107/3107 (100,0 %) | 0/3107 (0,0 %) | 41062 |
| version-33 | 1 | 2982 | 2982/2982 (100,0 %) | 0/2982 (0,0 %) | 0/2982 (0,0 %) | 0/2982 (0,0 %) | 212/2982 (7,1 %) | 0/2982 (0,0 %) | 0/2982 (0,0 %) | 2982/2982 (100,0 %) | 0/2982 (0,0 %) | 53634 |
| **corpus** | 20 | 12585 | 12585/12585 (100,0 %) | 0/12585 (0,0 %) | 0/12585 (0,0 %) | 0/12585 (0,0 %) | 4122/12585 (32,8 %) | 0/12585 (0,0 %) | 105/12585 (0,8 %) | 12480/12585 (99,2 %) | 0/12585 (0,0 %) | 154719 |

### Par film

| Film | Build | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 084a804d | HI_1_10_0 | 141 | 141/141 (100,0 %) | 0/141 (0,0 %) | 0/141 (0,0 %) | 0/141 (0,0 %) | 132/141 (93,6 %) | 0/141 (0,0 %) | 0/141 (0,0 %) | 141/141 (100,0 %) | 0/141 (0,0 %) | 176 |
| 111fa685 | HI_1_10_0 | 402 | 402/402 (100,0 %) | 0/402 (0,0 %) | 0/402 (0,0 %) | 0/402 (0,0 %) | 216/402 (53,7 %) | 0/402 (0,0 %) | 0/402 (0,0 %) | 402/402 (100,0 %) | 0/402 (0,0 %) | 3428 |
| 1c4c63c2 | HI_1_10_0 | 229 | 229/229 (100,0 %) | 0/229 (0,0 %) | 0/229 (0,0 %) | 0/229 (0,0 %) | 215/229 (93,9 %) | 0/229 (0,0 %) | 0/229 (0,0 %) | 229/229 (100,0 %) | 0/229 (0,0 %) | 517 |
| e5adf7b2 | HI_1_11_0 | 291 | 291/291 (100,0 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 268/291 (92,1 %) | 0/291 (0,0 %) | 0/291 (0,0 %) | 291/291 (100,0 %) | 0/291 (0,0 %) | 1045 |
| bcb6d393 | HI_1_12_0 | 49 | 49/49 (100,0 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 44/49 (89,8 %) | 0/49 (0,0 %) | 0/49 (0,0 %) | 49/49 (100,0 %) | 0/49 (0,0 %) | 1 |
| 0797ce72 | HI_1_13_0 | 969 | 969/969 (100,0 %) | 0/969 (0,0 %) | 0/969 (0,0 %) | 0/969 (0,0 %) | 853/969 (88,0 %) | 0/969 (0,0 %) | 6/969 (0,6 %) | 963/969 (99,4 %) | 0/969 (0,0 %) | 5932 |
| 396cfc92 | HI_1_13_0 | 12 | 12/12 (100,0 %) | 0/12 (0,0 %) | 0/12 (0,0 %) | 0/12 (0,0 %) | 10/12 (83,3 %) | 0/12 (0,0 %) | 0/12 (0,0 %) | 12/12 (100,0 %) | 0/12 (0,0 %) | 95 |
| 4f77afc1 | HI_1_13_0 | 97 | 97/97 (100,0 %) | 0/97 (0,0 %) | 0/97 (0,0 %) | 0/97 (0,0 %) | 91/97 (93,8 %) | 0/97 (0,0 %) | 0/97 (0,0 %) | 97/97 (100,0 %) | 0/97 (0,0 %) | 2505 |
| 51ebbc0f | HI_1_13_0 | 331 | 331/331 (100,0 %) | 0/331 (0,0 %) | 0/331 (0,0 %) | 0/331 (0,0 %) | 176/331 (53,2 %) | 0/331 (0,0 %) | 0/331 (0,0 %) | 331/331 (100,0 %) | 0/331 (0,0 %) | 32 |
| bf15f7ab | HI_1_13_0 | 37 | 37/37 (100,0 %) | 0/37 (0,0 %) | 0/37 (0,0 %) | 0/37 (0,0 %) | 5/37 (13,5 %) | 0/37 (0,0 %) | 0/37 (0,0 %) | 37/37 (100,0 %) | 0/37 (0,0 %) | 250 |
| bfecd02b | HI_1_13_0 | 9 | 9/9 (100,0 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 8/9 (88,9 %) | 0/9 (0,0 %) | 0/9 (0,0 %) | 9/9 (100,0 %) | 0/9 (0,0 %) | 81 |
| c75f33b8 | HI_1_13_0 | 13 | 13/13 (100,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 10/13 (76,9 %) | 0/13 (0,0 %) | 0/13 (0,0 %) | 13/13 (100,0 %) | 0/13 (0,0 %) | 16 |
| d9781168 | HI_1_13_0 | 189 | 189/189 (100,0 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 188/189 (99,5 %) | 0/189 (0,0 %) | 0/189 (0,0 %) | 189/189 (100,0 %) | 0/189 (0,0 %) | 21 |
| f75e7053 | HI_1_13_0 | 0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0/0 | 0 |
| fb1a1a72 | HI_1_13_0 | 487 | 487/487 (100,0 %) | 0/487 (0,0 %) | 0/487 (0,0 %) | 0/487 (0,0 %) | 344/487 (70,6 %) | 0/487 (0,0 %) | 3/487 (0,6 %) | 484/487 (99,4 %) | 0/487 (0,0 %) | 89 |
| a521164d | HI_1_4_1 | 2018 | 2018/2018 (100,0 %) | 0/2018 (0,0 %) | 0/2018 (0,0 %) | 0/2018 (0,0 %) | 168/2018 (8,3 %) | 0/2018 (0,0 %) | 0/2018 (0,0 %) | 2018/2018 (100,0 %) | 0/2018 (0,0 %) | 38679 |
| 60ae07c4 | HI_1_8_0 | 994 | 994/994 (100,0 %) | 0/994 (0,0 %) | 0/994 (0,0 %) | 0/994 (0,0 %) | 887/994 (89,2 %) | 0/994 (0,0 %) | 96/994 (9,7 %) | 898/994 (90,3 %) | 0/994 (0,0 %) | 5301 |
| 11de8353 | HI_1_9_0 | 228 | 228/228 (100,0 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 202/228 (88,6 %) | 0/228 (0,0 %) | 0/228 (0,0 %) | 228/228 (100,0 %) | 0/228 (0,0 %) | 1855 |
| 50247b26 | version-31 | 3107 | 3107/3107 (100,0 %) | 0/3107 (0,0 %) | 0/3107 (0,0 %) | 0/3107 (0,0 %) | 93/3107 (3,0 %) | 0/3107 (0,0 %) | 0/3107 (0,0 %) | 3107/3107 (100,0 %) | 0/3107 (0,0 %) | 41062 |
| a349fea8 | version-33 | 2982 | 2982/2982 (100,0 %) | 0/2982 (0,0 %) | 0/2982 (0,0 %) | 0/2982 (0,0 %) | 212/2982 (7,1 %) | 0/2982 (0,0 %) | 0/2982 (0,0 %) | 2982/2982 (100,0 %) | 0/2982 (0,0 %) | 53634 |

### Corpus : sortie x vue C x reste

| Sortie B | Vue C | Reste | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| terminateur | 1 entree(s) | >= 64 | 7574 | 120259 |
| terminateur | vide | >= 64 | 4054 | 18229 |
| terminateur | 2 entree(s) | >= 64 | 525 | 13678 |
| terminateur | 0 entree (kind 3 seul) | >= 64 | 324 | 2102 |
| terminateur | vide | 8-63 | 68 | 387 |
| terminateur | 0 entree (kind 3 seul) | 8-63 | 29 | 0 |
| terminateur | 1 entree(s) | 8-63 | 8 | 52 |
| terminateur | 3 entree(s) | >= 64 | 3 | 12 |

### Corpus : dernier record et dernier composant lus avant la fin de la vue B

| Sortie B | Dernier lu | Paquets | Utiles en jeu |
|---|---|---|---|
| terminateur | DELTA ti=40 i25 unit-command-tick-component | 1676 | 13065 |
| terminateur | DELTA ti=37 i3 object-angular-velocity-component | 1569 | 30336 |
| terminateur | DEL | 1050 | 14985 |
| terminateur | DELTA ti=10 i26 managed-object-rtpc-component | 1021 | 23832 |
| terminateur | DELTA ti=4 i0 high-frequency | 1008 | 98 |
| terminateur | DELTA ti=43 i18 device-position-component | 828 | 344 |
| terminateur | DELTA ti=42 i3 object-angular-velocity-component | 688 | 18217 |
| terminateur | DELTA ti=3 i1 high-frequency | 386 | 8 |
| terminateur | DELTA ti=21 i16 flock-position-component | 337 | 1 |
| terminateur | DELTA ti=42 i2 object-forward-and-up-component | 305 | 8195 |
| terminateur | DELTA ti=37 i29 equipment-command-tick-component | 275 | 6469 |
| terminateur | DELTA ti=37 i1 object-translational-velocity-component | 232 | 6826 |
| terminateur | DELTA ti=37 i2 object-forward-and-up-component | 216 | 5203 |
| terminateur | DELTA ti=42 i1 object-translational-velocity-component | 193 | 5311 |
| terminateur | DELTA ti=3 sans composant | 186 | 0 |
| terminateur | DELTA ti=20 i1 spawn-filter-weight-component | 176 | 2127 |
| terminateur | DELTA ti=47 i1 managed-object-networked-splash-message-dynamic-component | 162 | 3231 |
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
| terminateur | DELTA ti=0 i2 game-engine-current-state-component | 49 | 1195 |
| terminateur | NEW ti=32 i0 tacmap-areaofinterest | 44 | 6 |
| terminateur | NEW ti=40 sans composant | 44 | 0 |

## Sorties de la vue B, tous paquets qui l atteignent

| Build | Sortie B | Paquets | Fermes | Hors cadre | Autres causes |
|---|---|---|---|---|---|
| HI_1_10_0 | ouverte | 1833 | 0/1833 (0,0 %) | 0/1833 (0,0 %) | 1833/1833 (100,0 %) |
| HI_1_10_0 | rejet hors datum | 83147 | 0/83147 (0,0 %) | 0/83147 (0,0 %) | 83147/83147 (100,0 %) |
| HI_1_10_0 | terminateur | 26286 | 22171/26286 (84,3 %) | 772/26286 (2,9 %) | 3343/26286 (12,7 %) |
| HI_1_11_0 | ouverte | 152 | 0/152 (0,0 %) | 0/152 (0,0 %) | 152/152 (100,0 %) |
| HI_1_11_0 | rejet hors datum | 10317 | 0/10317 (0,0 %) | 0/10317 (0,0 %) | 10317/10317 (100,0 %) |
| HI_1_11_0 | terminateur | 4629 | 4146/4629 (89,6 %) | 291/4629 (6,3 %) | 192/4629 (4,1 %) |
| HI_1_12_0 | ouverte | 12425 | 0/12425 (0,0 %) | 0/12425 (0,0 %) | 12425/12425 (100,0 %) |
| HI_1_12_0 | rejet hors datum | 3270 | 0/3270 (0,0 %) | 0/3270 (0,0 %) | 3270/3270 (100,0 %) |
| HI_1_12_0 | terminateur | 5886 | 5830/5886 (99,0 %) | 49/5886 (0,8 %) | 7/5886 (0,1 %) |
| HI_1_13_0 | ouverte | 5461 | 0/5461 (0,0 %) | 0/5461 (0,0 %) | 5461/5461 (100,0 %) |
| HI_1_13_0 | rejet hors datum | 85792 | 0/85792 (0,0 %) | 0/85792 (0,0 %) | 85792/85792 (100,0 %) |
| HI_1_13_0 | terminateur | 228694 | 224924/228694 (98,4 %) | 2144/228694 (0,9 %) | 1626/228694 (0,7 %) |
| HI_1_4_1 | ouverte | 270 | 0/270 (0,0 %) | 0/270 (0,0 %) | 270/270 (100,0 %) |
| HI_1_4_1 | rejet hors datum | 6577 | 0/6577 (0,0 %) | 0/6577 (0,0 %) | 6577/6577 (100,0 %) |
| HI_1_4_1 | terminateur | 2992 | 692/2992 (23,1 %) | 2018/2992 (67,4 %) | 282/2992 (9,4 %) |
| HI_1_8_0 | ouverte | 1283 | 0/1283 (0,0 %) | 0/1283 (0,0 %) | 1283/1283 (100,0 %) |
| HI_1_8_0 | rejet hors datum | 29998 | 0/29998 (0,0 %) | 0/29998 (0,0 %) | 29998/29998 (100,0 %) |
| HI_1_8_0 | terminateur | 15155 | 13802/15155 (91,1 %) | 994/15155 (6,6 %) | 359/15155 (2,4 %) |
| HI_1_9_0 | ouverte | 209 | 0/209 (0,0 %) | 0/209 (0,0 %) | 209/209 (100,0 %) |
| HI_1_9_0 | rejet hors datum | 9817 | 0/9817 (0,0 %) | 0/9817 (0,0 %) | 9817/9817 (100,0 %) |
| HI_1_9_0 | terminateur | 5964 | 5619/5964 (94,2 %) | 228/5964 (3,8 %) | 117/5964 (2,0 %) |
| version-31 | ouverte | 196 | 0/196 (0,0 %) | 0/196 (0,0 %) | 196/196 (100,0 %) |
| version-31 | rejet hors datum | 10774 | 0/10774 (0,0 %) | 0/10774 (0,0 %) | 10774/10774 (100,0 %) |
| version-31 | terminateur | 4692 | 139/4692 (3,0 %) | 3107/4692 (66,2 %) | 1446/4692 (30,8 %) |
| version-33 | ouverte | 813 | 0/813 (0,0 %) | 0/813 (0,0 %) | 813/813 (100,0 %) |
| version-33 | rejet hors datum | 19542 | 0/19542 (0,0 %) | 0/19542 (0,0 %) | 19542/19542 (100,0 %) |
| version-33 | terminateur | 4357 | 420/4357 (9,6 %) | 2982/4357 (68,4 %) | 955/4357 (21,9 %) |
| **corpus** | ouverte | 22642 | 0/22642 (0,0 %) | 0/22642 (0,0 %) | 22642/22642 (100,0 %) |
| **corpus** | rejet hors datum | 259234 | 0/259234 (0,0 %) | 0/259234 (0,0 %) | 259234/259234 (100,0 %) |
| **corpus** | terminateur | 298655 | 277743/298655 (93,0 %) | 12585/298655 (4,2 %) | 8327/298655 (2,8 %) |

## Sorties par rejet contre le bloc de type 1 (item 1.2)

Etat du slot rejete dans le bloc de type 1 de SON chunk ; naissance : le bloc du chunk SUIVANT porte-t-il une allocation de cet eid que le chunk ne portait pas, sans NEW lu ?

| Sortie B | Paquet | Etat au bloc | Naissance | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| rejet hors datum | vue B : sortie par rejet | vide | naissance non lue | 109080 | 1618660 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue | 80324 | 1112253 |
| rejet hors datum | vue B : sortie par rejet | vivant | vivant au bloc du chunk | 14703 | 36197 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · naissance non lue | 14584 | 424 |
| rejet hors datum | vue B : sortie par rejet | vide | aucune allocation | 11188 | 50359 |
| rejet hors datum | vue B : sortie par rejet | trace | naissance non lue, generation 0 | 8776 | 149786 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · naissance non lue | 8754 | 2658 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · naissance non lue, generation 0 | 4085 | 72 |
| rejet hors datum | vue B : sortie par rejet | vide | non mesurable | 3590 | 12411 |
| rejet hors datum | vue B : sortie par rejet | trace | aucune allocation | 2035 | 6709 |
| rejet hors datum | vue B : sortie par rejet | trace | non mesurable | 789 | 6734 |
| rejet hors datum | vue B : sortie par rejet | trace | libere avant le chunk (meme generation) | 396 | 2080 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · non mesurable | 277 | 1 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · non mesurable | 247 | 0 |
| rejet hors datum | vue B : sortie par rejet | vide | realloue sous une autre generation | 139 | 362 |
| rejet hors datum | vue B : sortie par rejet | vivant | non mesurable | 64 | 184 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu desynchronise · aucune allocation | 61 | 323 |
| rejet hors datum | vue B : sortie par rejet | vide | NEW lu dans le chunk | 45 | 56 |
| rejet hors datum | vue B : sortie par rejet | trace | realloue sous une autre generation | 31 | 145 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu desynchronise · vivant au bloc du chunk | 28 | 0 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu dans le chunk | 9 | 0 |
| rejet hors datum | vue B : sortie par rejet | vivant | aucune allocation | 9 | 54 |
| rejet hors datum | vue B : sortie par rejet | vivant | realloue sous une autre generation | 6 | 48 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu dans le chunk | 4 | 28 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · aucune allocation | 3 | 34 |
| rejet hors datum | vue B : sortie par rejet | trace | NEW lu desynchronise · libere avant le chunk (meme generation) | 3 | 50 |
| rejet hors datum | vue B : sortie par rejet | vivant | NEW lu desynchronise · aucune allocation | 3 | 0 |
| rejet hors datum | vue B : sortie par rejet | absent | aucune allocation | 1 | 0 |

| Build | Rejets | dont paquet hors cadre | dont ferme | Vivant au bloc | Naissance non lue | Non mesurable |
|---|---|---|---|---|---|---|
| HI_1_10_0 | 83147 | 0/83147 (0,0 %) | 0/83147 (0,0 %) | 10705/83147 (12,9 %) | 64074/83147 (77,1 %) | 785/83147 (0,9 %) |
| HI_1_11_0 | 10317 | 0/10317 (0,0 %) | 0/10317 (0,0 %) | 56/10317 (0,5 %) | 8526/10317 (82,6 %) | 0/10317 (0,0 %) |
| HI_1_12_0 | 3270 | 0/3270 (0,0 %) | 0/3270 (0,0 %) | 0/3270 (0,0 %) | 3037/3270 (92,9 %) | 0/3270 (0,0 %) |
| HI_1_13_0 | 85792 | 0/85792 (0,0 %) | 0/85792 (0,0 %) | 2262/85792 (2,6 %) | 47280/85792 (55,1 %) | 2441/85792 (2,8 %) |
| HI_1_4_1 | 6577 | 0/6577 (0,0 %) | 0/6577 (0,0 %) | 0/6577 (0,0 %) | 5844/6577 (88,9 %) | 332/6577 (5,0 %) |
| HI_1_8_0 | 29998 | 0/29998 (0,0 %) | 0/29998 (0,0 %) | 203/29998 (0,7 %) | 25813/29998 (86,0 %) | 204/29998 (0,7 %) |
| HI_1_9_0 | 9817 | 0/9817 (0,0 %) | 0/9817 (0,0 %) | 137/9817 (1,4 %) | 7696/9817 (78,4 %) | 191/9817 (1,9 %) |
| version-31 | 10774 | 0/10774 (0,0 %) | 0/10774 (0,0 %) | 1/10774 (0,0 %) | 10199/10774 (94,7 %) | 486/10774 (4,5 %) |
| version-33 | 19542 | 0/19542 (0,0 %) | 0/19542 (0,0 %) | 1458/19542 (7,5 %) | 16935/19542 (86,7 %) | 4/19542 (0,0 %) |
| **corpus** | 259234 | 0/259234 (0,0 %) | 0/259234 (0,0 %) | 14822/259234 (5,7 %) | 189404/259234 (73,1 %) | 4443/259234 (1,7 %) |

## Entrees de controle utiles (item 1.3)

Utile = entree `kind 0` qui porte le bloc de 0x68 octets, la seule que le tir continu lit. Denominateur ESTIME = utiles fermees + moyenne par vue C fermee du film x paquets non fermes. PAR CONSTRUCTION, la part estimee d un film egale sa part de vues C fermees (U / (U + U/F x N) = F / (F + N)) : cette estimation ne dit rien de plus que les paquets.

| Build | Paquets | Vues C fermees | Entrees fermees | Utiles fermees | Moyenne par vue C fermee | Denominateur estime | Part estimee | Utiles lues hors fermeture (non prouvees) |
|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 129369 | 22171/129369 (17,1 %) | 347690 | 347690 | 15.68 | 2053409 | 16,9 % | 70752 |
| HI_1_11_0 | 16824 | 4146/16824 (24,6 %) | 74842 | 74842 | 18.05 | 303700 | 24,6 % | 1897 |
| HI_1_12_0 | 21864 | 5830/21864 (26,7 %) | 30125 | 30125 | 5.17 | 112977 | 26,7 % | 21 |
| HI_1_13_0 | 335960 | 224924/335960 (66,9 %) | 1629458 | 1629458 | 7.24 | 2435671 | 66,9 % | 14671 |
| HI_1_4_1 | 11130 | 692/11130 (6,2 %) | 0 | 0 | 0.00 | 0 | - | 2025 |
| HI_1_8_0 | 49696 | 13802/49696 (27,8 %) | 78935 | 78935 | 5.72 | 284216 | 27,8 % | 448 |
| HI_1_9_0 | 17629 | 5619/17629 (31,9 %) | 74039 | 74039 | 13.18 | 232289 | 31,9 % | 1482 |
| version-31 | 17919 | 139/17919 (0,8 %) | 0 | 0 | 0.00 | 0 | - | 4198 |
| version-33 | 28751 | 420/28751 (1,5 %) | 0 | 0 | 0.00 | 0 | - | 3642 |
| **corpus** | 629142 | 277743/629142 (44,1 %) | 2235089 | 2235089 | 8.05 | 5422263 | 41,2 % | 99136 |

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
| HI_1_10_0 | 1799380 | 15/1799380 (0,0 %) | 62 | 8 | 15 | 3 | 2 | 0 |
| HI_1_11_0 | 324136 | 3/324136 (0,0 %) | 31 | 1 | 3 | 1 | 0 | 0 |
| HI_1_12_0 | 168158 | 3/168158 (0,0 %) | 13 | 0 | 3 | 1 | 0 | 0 |
| HI_1_13_0 | 3371207 | 272/3371207 (0,0 %) | 1665 | 69 | 272 | 11 | 7 | 0 |
| HI_1_4_1 | 226680 | 1/226680 (0,0 %) | 2 | 1 | 1 | 1 | 1 | 0 |
| HI_1_8_0 | 338248 | 337/338248 (0,1 %) | 1312 | 79 | 337 | 12 | 1 | 0 |
| HI_1_9_0 | 276066 | 19/276066 (0,0 %) | 83 | 6 | 19 | 1 | 1 | 0 |
| version-31 | 543870 | 4/543870 (0,0 %) | 13 | 1 | 4 | 1 | 1 | 0 |
| version-33 | 553816 | 7/553816 (0,0 %) | 38 | 5 | 7 | 1 | 1 | 0 |
| **corpus** | 7601561 | 661/7601561 (0,0 %) | 3219 | 170 | 661 | 32 | 14 | 0 |

## Lot L0 — fermes au bit pres contre fermes (regles de l ecrivain)

| Build | Fermes au bit | Fermes | Retires | Utiles lus | Utiles fermes au bit | Utiles fermes | Part variable | Fixe | Part fixe |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 28835 | 22171 | 6664 | 1559136 | 303682 | 297268 | 19,1 % | 2423551 | 12,3 % |
| HI_1_11_0 | 4309 | 4146 | 163 | 289562 | 80622 | 80066 | 27,7 % | 383476 | 20,9 % |
| HI_1_12_0 | 5835 | 5830 | 5 | 121628 | 35158 | 35143 | 28,9 % | 148160 | 23,7 % |
| HI_1_13_0 | 226505 | 224924 | 1581 | 2516040 | 2055058 | 2048856 | 81,4 % | 3073267 | 66,7 % |
| HI_1_4_1 | 701 | 692 | 9 | 181722 | 80 | 78 | 0,0 % | 198500 | 0,0 % |
| HI_1_8_0 | 13969 | 13802 | 167 | 268352 | 83225 | 82730 | 30,8 % | 359291 | 23,0 % |
| HI_1_9_0 | 5748 | 5619 | 129 | 248382 | 65628 | 65391 | 26,3 % | 324613 | 20,1 % |
| version-31 | 153 | 139 | 14 | 319252 | 277 | 274 | 0,1 % | 328128 | 0,1 % |
| version-33 | 465 | 420 | 45 | 469728 | 3604 | 3595 | 0,8 % | 519304 | 0,7 % |
| **corpus** | 286520 | 277743 | 8777 | 5973802 | 2627334 | 2613401 | 43,7 % | 7758290 | 33,7 % |

### Corpus : fermetures au bit retirees, par premiere regle

| Regle | Paquets | Utiles |
|---|---|---|
| ecrivain : masque au-dela de l archetype | 4097 | 11572 |
| ecrivain : masque epars non croissant | 123 | 391 |
| ecrivain : ordre de la vue B | 226 | 1011 |
| ecrivain : vue C en-tete cdc04 pose | 6 | 6 |
| ecrivain : vue C kind non nul | 1 | 0 |
| vue B : sortie par rejet | 4324 | 953 |

### Corpus : temoins decales (vue C relue depuis un depart decale de k bits)

| k | Refermes | Part des fermes |
|---|---|---|
| -8 | 1089 | 0,4 % |
| -7 | 1585 | 0,6 % |
| -6 | 3226 | 1,2 % |
| -5 | 4565 | 1,6 % |
| -4 | 14086 | 5,1 % |
| -3 | 10220 | 3,7 % |
| -2 | 10401 | 3,7 % |
| -1 | 10579 | 3,8 % |
| +1 | 23739 | 8,5 % |
| +2 | 23012 | 8,3 % |
| +3 | 15844 | 5,7 % |
| +4 | 14575 | 5,2 % |
| +5 | 14025 | 5,0 % |
| +6 | 13844 | 5,0 % |
| +7 | 13666 | 4,9 % |
| +8 | 0 | 0,0 % |


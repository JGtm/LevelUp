# Carte de fermeture des trames delta — 2026-10-01

Films mesures : 20 ; echecs : 0. Contexte d instrument (largeurs d axe lues dans le film, profil par defaut, sans calibration killsource).

Table ECS : `internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv`.

## Par build

| Build | Films | Paquets fermes | Vue A | Vue B | Vue C | Records utiles fermes | Entrees de controle lues | Declencheur (>= 95 %) | Pic memoire max |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 28741/129369 (22,2 %) | 16686/79272 (21,0 %) | 28741/111238 (25,8 %) | 28741/109144 (26,3 %) | 302662/1559051 (19,4 %) | 420684 | non atteint | 321 Mio |
| HI_1_11_0 | 1 | 4285/16824 (25,5 %) | 2959/9073 (32,6 %) | 4285/15097 (28,4 %) | 4285/14831 (28,9 %) | 80032/289378 (27,7 %) | 76224 | non atteint | 126 Mio |
| HI_1_12_0 | 1 | 5835/21864 (26,7 %) | 5201/19134 (27,2 %) | 5835/21581 (27,0 %) | 5835/9156 (63,7 %) | 35158/121628 (28,9 %) | 30137 | non atteint | 69 Mio |
| HI_1_13_0 | 10 | 224818/335960 (66,9 %) | 190545/271063 (70,3 %) | 224818/319871 (70,3 %) | 224818/313492 (71,7 %) | 2017679/2504223 (80,6 %) | 1619514 | non atteint | 231 Mio |
| HI_1_4_1 | 1 | 701/11130 (6,3 %) | 656/6174 (10,6 %) | 701/9839 (7,1 %) | 701/9497 (7,4 %) | 80/181607 (0,0 %) | 5 | non atteint | 216 Mio |
| HI_1_8_0 | 1 | 13969/49696 (28,1 %) | 13054/43640 (29,9 %) | 13969/46436 (30,1 %) | 13969/45152 (30,9 %) | 83225/268352 (31,0 %) | 79294 | non atteint | 151 Mio |
| HI_1_9_0 | 1 | 5739/17629 (32,6 %) | 4404/10304 (42,7 %) | 5739/15988 (35,9 %) | 5739/15739 (36,5 %) | 65452/248285 (26,4 %) | 75327 | non atteint | 123 Mio |
| version-31 | 1 | 153/17919 (0,9 %) | 130/10140 (1,3 %) | 153/15662 (1,0 %) | 153/15382 (1,0 %) | 277/319053 (0,1 %) | 3 | non atteint | 159 Mio |
| version-33 | 1 | 463/28751 (1,6 %) | 399/14259 (2,8 %) | 463/24710 (1,9 %) | 463/23756 (1,9 %) | 3602/469451 (0,8 %) | 6 | non atteint | 234 Mio |

## Causes d arret (premiere cause de chaque paquet non ferme)

Gain potentiel = records utiles LUS et non fermes dans les paquets que la cause arrete : BORNE SUPERIEURE (une autre cause peut suivre ; les records d apres l arret ne sont pas lus du tout).

| Rang | Cause | Archetype | Index | Statut | Usage produit | Paquets bloques | Gain potentiel | Builds |
|---|---|---|---|---|---|---|---|---|
| 1 | vue C : terminateur hors cadre | - |  | - | - | 264757 | 3142830 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 2 | liste d evenements non localisee | - |  | - | - | 48720 | 0 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 3 | ti=43 device-animation-layer-state | 43 | i35 | non_porte | aucun | 12114 | 67608 | HI_1_10_0, HI_1_12_0, HI_1_8_0 |
| 4 | vue C : kind non porte | - |  | - | - | 4971 | 66730 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 5 | ti=2 managed-engine-timers | 2 | i15 | non_porte | aucun | 3996 | 28 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 6 | vue C : bloc 0xbc non porte | - |  | - | - | 1570 | 22121 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 7 | ti=3 low-frequency | 3 | i0 | non_porte | aucun | 1187 | 135 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 8 | ti=48 forge-player-data-edited-objects-ids | 48 | i0 | non_porte | aucun | 931 | 8143 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 9 | ti=40 vehicle-auto-turret-aiming-vector | 40 | i31 | non_porte | aucun | 808 | 4827 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-33 |
| 10 | vue B : fin de payload | - |  | - | - | 604 | 3093 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 11 | ti=43 device-position-group | 43 | i21 | non_porte | aucun | 603 | 10410 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 12 | ti=40 vehicle-weapon-set | 40 | i37, i38 | non_porte | aucun | 590 | 12509 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 13 | ti=10 managed-object-navpoint | 10 | i10, i11, i12, i13, i15, i16, i17, i2, i3, i4, i5, i6, i7, i8 | non_porte | aucun | 344 | 4285 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 14 | ti=12 managed-navpoint-visual-state-groups-component-2 | 12 | i22 | non_porte | aucun | 341 | 5030 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-31, version-33 |
| 15 | ti=12 managed-navpoint-visual-state-groups-component-1 | 12 | i21 | non_porte | aucun | 333 | 4698 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 16 | ti=35 biped-spartan-ability-non-predicted-state | 35 | i58, i59 | partiel | rejeu : grappleLines[] (schema 8) — la ligne blanche du joueur vers son ancre, fenetre [t0,t1] par vie | 240 | 2008 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 17 | ti=56 archetype hors registre | 56 |  | - | - | 231 | 1141 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 18 | vue C : debordement | - |  | - | - | 147 | 275 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 19 | ti=12 managed-navpoint-visual-state-groups-component-0 | 12 | i20 | non_porte | aucun | 115 | 1258 | HI_1_13_0, HI_1_4_1, version-33 |
| 20 | ti=12 managed-navpoint-override-flags | 12 | i16 | non_porte | aucun | 88 | 454 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 21 | ti=2 matchflow-isplaying-flags | 2 | i17 | non_porte | aucun | 85 | 13 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 22 | ti=11 managed-objective-interaction-filter | 11 | i4 | non_porte | aucun | 83 | 656 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0 |
| 23 | ti=40 vehicle-type-state | 40 | i32, i33 | non_porte | aucun | 83 | 1786 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_9_0, version-31 |
| 24 | ti=43 device-machine-flags | 43 | i39 | non_porte | aucun | 83 | 1176 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, version-31, version-33 |
| 25 | ti=2 game-engine-soft-ceilings | 2 | i11 | non_porte | aucun | 67 | 420 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 26 | ti=18 effect-state-data | 18 | i0, i1, i10, i12, i13, i19, i2, i22, i23, i28, i3, i31, i4, i6, i7, i8, i9 | partiel | aucun | 66 | 201 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31 |
| 27 | ti=40 air-drop-flight | 40 | i44, i45 | non_porte | aucun | 59 | 1444 | HI_1_11_0, HI_1_4_1, version-31, version-33 |
| 28 | ti=51 archetype hors registre | 51 |  | - | - | 56 | 493 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
| 29 | ti=10 managed-object-interaction-filter | 10 | i22 | non_porte | aucun | 55 | 762 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 30 | ti=59 archetype hors registre | 59 |  | - | - | 52 | 415 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 31 | ti=40 vehicle-low-frequency | 40 | i46, i47 | non_porte | aucun | 48 | 1024 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 32 | ti=2 GameEngineComposerLetterboxComponent | 2 | i14 | non_porte | aucun | 42 | 441 | HI_1_10_0, HI_1_12_0, HI_1_8_0, HI_1_9_0 |
| 33 | ti=43 device-position-animation-name | 43 | i19 | non_porte | aucun | 42 | 382 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-33 |
| 34 | ti=50 archetype hors registre | 50 |  | - | - | 41 | 178 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 35 | ti=19 sound-placement-state-data | 19 | i0, i1, i13, i18, i2, i3, i31, i4, i6, i8, i9 | non_porte | aucun | 40 | 241 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 36 | ti=40 warp | 40 | i45, i46 | non_porte | aucun | 36 | 861 | HI_1_11_0, HI_1_4_1, version-31, version-33 |
| 37 | ti=52 archetype hors registre | 52 |  | - | - | 36 | 259 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 38 | ti=0 game-engine-soft-ceilings | 0 | i11 | non_porte | aucun | 34 | 233 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 39 | ti=55 archetype hors registre | 55 |  | - | - | 29 | 224 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
| 40 | ti=58 archetype hors registre | 58 |  | - | - | 29 | 83 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-33 |
## Carte v2 — « vue C : terminateur hors cadre » ventile (item 1.1)

Sortie de la vue B : comment la boucle de records s est arretee avant la vue C ; vue C : vide (son terminateur seul) ou non ; reste : bits du payload derriere le terminateur de la vue C.

### Par build

| Build | Films | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 3 | 79964 | 784/79964 (1,0 %) | 79180/79964 (99,0 %) | 0/79964 (0,0 %) | 0/79964 (0,0 %) | 79446/79964 (99,4 %) | 0/79964 (0,0 %) | 56/79964 (0,1 %) | 79908/79964 (99,9 %) | 0/79964 (0,0 %) | 1235464 |
| HI_1_11_0 | 1 | 10198 | 288/10198 (2,8 %) | 9910/10198 (97,2 %) | 0/10198 (0,0 %) | 0/10198 (0,0 %) | 10016/10198 (98,2 %) | 0/10198 (0,0 %) | 4/10198 (0,0 %) | 10194/10198 (100,0 %) | 0/10198 (0,0 %) | 202898 |
| HI_1_12_0 | 1 | 3249 | 49/3249 (1,5 %) | 3200/3249 (98,5 %) | 0/3249 (0,0 %) | 0/3249 (0,0 %) | 3193/3249 (98,3 %) | 0/3249 (0,0 %) | 0/3249 (0,0 %) | 3249/3249 (100,0 %) | 0/3249 (0,0 %) | 18529 |
| HI_1_13_0 | 10 | 86921 | 2096/86921 (2,4 %) | 84825/86921 (97,6 %) | 0/86921 (0,0 %) | 0/86921 (0,0 %) | 85758/86921 (98,7 %) | 1/86921 (0,0 %) | 170/86921 (0,2 %) | 86750/86921 (99,8 %) | 0/86921 (0,0 %) | 450803 |
| HI_1_4_1 | 1 | 8364 | 2002/8364 (23,9 %) | 6362/8364 (76,1 %) | 0/8364 (0,0 %) | 0/8364 (0,0 %) | 6391/8364 (76,4 %) | 1/8364 (0,0 %) | 1/8364 (0,0 %) | 8362/8364 (100,0 %) | 0/8364 (0,0 %) | 171027 |
| HI_1_8_0 | 1 | 30283 | 994/30283 (3,3 %) | 29289/30283 (96,7 %) | 0/30283 (0,0 %) | 0/30283 (0,0 %) | 29765/30283 (98,3 %) | 0/30283 (0,0 %) | 203/30283 (0,7 %) | 30080/30283 (99,3 %) | 0/30283 (0,0 %) | 176241 |
| HI_1_9_0 | 1 | 9792 | 225/9792 (2,3 %) | 9567/9792 (97,7 %) | 0/9792 (0,0 %) | 0/9792 (0,0 %) | 9690/9792 (99,0 %) | 0/9792 (0,0 %) | 0/9792 (0,0 %) | 9792/9792 (100,0 %) | 0/9792 (0,0 %) | 177455 |
| version-31 | 1 | 13780 | 3096/13780 (22,5 %) | 10684/13780 (77,5 %) | 0/13780 (0,0 %) | 0/13780 (0,0 %) | 10770/13780 (78,2 %) | 0/13780 (0,0 %) | 0/13780 (0,0 %) | 13780/13780 (100,0 %) | 0/13780 (0,0 %) | 276937 |
| version-33 | 1 | 22206 | 2961/22206 (13,3 %) | 19245/22206 (86,7 %) | 0/22206 (0,0 %) | 0/22206 (0,0 %) | 19277/22206 (86,8 %) | 0/22206 (0,0 %) | 0/22206 (0,0 %) | 22206/22206 (100,0 %) | 0/22206 (0,0 %) | 433476 |
| **corpus** | 20 | 264757 | 12495/264757 (4,7 %) | 252262/264757 (95,3 %) | 0/264757 (0,0 %) | 0/264757 (0,0 %) | 254306/264757 (96,1 %) | 2/264757 (0,0 %) | 434/264757 (0,2 %) | 264321/264757 (99,8 %) | 0/264757 (0,0 %) | 3142830 |

### Par film

| Film | Build | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 084a804d | HI_1_10_0 | 23082 | 141/23082 (0,6 %) | 22941/23082 (99,4 %) | 0/23082 (0,0 %) | 0/23082 (0,0 %) | 23055/23082 (99,9 %) | 0/23082 (0,0 %) | 0/23082 (0,0 %) | 23082/23082 (100,0 %) | 0/23082 (0,0 %) | 439828 |
| 111fa685 | HI_1_10_0 | 10944 | 401/10944 (3,7 %) | 10543/10944 (96,3 %) | 0/10944 (0,0 %) | 0/10944 (0,0 %) | 10540/10944 (96,3 %) | 0/10944 (0,0 %) | 0/10944 (0,0 %) | 10944/10944 (100,0 %) | 0/10944 (0,0 %) | 199075 |
| 1c4c63c2 | HI_1_10_0 | 45938 | 242/45938 (0,5 %) | 45696/45938 (99,5 %) | 0/45938 (0,0 %) | 0/45938 (0,0 %) | 45851/45938 (99,8 %) | 0/45938 (0,0 %) | 56/45938 (0,1 %) | 45882/45938 (99,9 %) | 0/45938 (0,0 %) | 596561 |
| e5adf7b2 | HI_1_11_0 | 10198 | 288/10198 (2,8 %) | 9910/10198 (97,2 %) | 0/10198 (0,0 %) | 0/10198 (0,0 %) | 10016/10198 (98,2 %) | 0/10198 (0,0 %) | 4/10198 (0,0 %) | 10194/10198 (100,0 %) | 0/10198 (0,0 %) | 202898 |
| bcb6d393 | HI_1_12_0 | 3249 | 49/3249 (1,5 %) | 3200/3249 (98,5 %) | 0/3249 (0,0 %) | 0/3249 (0,0 %) | 3193/3249 (98,3 %) | 0/3249 (0,0 %) | 0/3249 (0,0 %) | 3249/3249 (100,0 %) | 0/3249 (0,0 %) | 18529 |
| 0797ce72 | HI_1_13_0 | 5510 | 966/5510 (17,5 %) | 4544/5510 (82,5 %) | 0/5510 (0,0 %) | 0/5510 (0,0 %) | 4844/5510 (87,9 %) | 1/5510 (0,0 %) | 38/5510 (0,7 %) | 5471/5510 (99,3 %) | 0/5510 (0,0 %) | 35108 |
| 396cfc92 | HI_1_13_0 | 8362 | 12/8362 (0,1 %) | 8350/8362 (99,9 %) | 0/8362 (0,0 %) | 0/8362 (0,0 %) | 8359/8362 (100,0 %) | 0/8362 (0,0 %) | 123/8362 (1,5 %) | 8239/8362 (98,5 %) | 0/8362 (0,0 %) | 56886 |
| 4f77afc1 | HI_1_13_0 | 9535 | 52/9535 (0,5 %) | 9483/9535 (99,5 %) | 0/9535 (0,0 %) | 0/9535 (0,0 %) | 9527/9535 (99,9 %) | 0/9535 (0,0 %) | 0/9535 (0,0 %) | 9535/9535 (100,0 %) | 0/9535 (0,0 %) | 179588 |
| 51ebbc0f | HI_1_13_0 | 16929 | 331/16929 (2,0 %) | 16598/16929 (98,0 %) | 0/16929 (0,0 %) | 0/16929 (0,0 %) | 16748/16929 (98,9 %) | 0/16929 (0,0 %) | 0/16929 (0,0 %) | 16929/16929 (100,0 %) | 0/16929 (0,0 %) | 29399 |
| bf15f7ab | HI_1_13_0 | 1658 | 37/1658 (2,2 %) | 1621/1658 (97,8 %) | 0/1658 (0,0 %) | 0/1658 (0,0 %) | 1539/1658 (92,8 %) | 0/1658 (0,0 %) | 0/1658 (0,0 %) | 1658/1658 (100,0 %) | 0/1658 (0,0 %) | 9831 |
| bfecd02b | HI_1_13_0 | 3154 | 9/3154 (0,3 %) | 3145/3154 (99,7 %) | 0/3154 (0,0 %) | 0/3154 (0,0 %) | 3153/3154 (100,0 %) | 0/3154 (0,0 %) | 1/3154 (0,0 %) | 3153/3154 (100,0 %) | 0/3154 (0,0 %) | 23320 |
| c75f33b8 | HI_1_13_0 | 2296 | 13/2296 (0,6 %) | 2283/2296 (99,4 %) | 0/2296 (0,0 %) | 0/2296 (0,0 %) | 2293/2296 (99,9 %) | 0/2296 (0,0 %) | 0/2296 (0,0 %) | 2296/2296 (100,0 %) | 0/2296 (0,0 %) | 10245 |
| d9781168 | HI_1_13_0 | 11249 | 189/11249 (1,7 %) | 11060/11249 (98,3 %) | 0/11249 (0,0 %) | 0/11249 (0,0 %) | 11239/11249 (99,9 %) | 0/11249 (0,0 %) | 4/11249 (0,0 %) | 11245/11249 (100,0 %) | 0/11249 (0,0 %) | 64305 |
| f75e7053 | HI_1_13_0 | 4287 | 0/4287 (0,0 %) | 4287/4287 (100,0 %) | 0/4287 (0,0 %) | 0/4287 (0,0 %) | 4287/4287 (100,0 %) | 0/4287 (0,0 %) | 1/4287 (0,0 %) | 4286/4287 (100,0 %) | 0/4287 (0,0 %) | 26439 |
| fb1a1a72 | HI_1_13_0 | 23941 | 487/23941 (2,0 %) | 23454/23941 (98,0 %) | 0/23941 (0,0 %) | 0/23941 (0,0 %) | 23769/23941 (99,3 %) | 0/23941 (0,0 %) | 3/23941 (0,0 %) | 23938/23941 (100,0 %) | 0/23941 (0,0 %) | 15682 |
| a521164d | HI_1_4_1 | 8364 | 2002/8364 (23,9 %) | 6362/8364 (76,1 %) | 0/8364 (0,0 %) | 0/8364 (0,0 %) | 6391/8364 (76,4 %) | 1/8364 (0,0 %) | 1/8364 (0,0 %) | 8362/8364 (100,0 %) | 0/8364 (0,0 %) | 171027 |
| 60ae07c4 | HI_1_8_0 | 30283 | 994/30283 (3,3 %) | 29289/30283 (96,7 %) | 0/30283 (0,0 %) | 0/30283 (0,0 %) | 29765/30283 (98,3 %) | 0/30283 (0,0 %) | 203/30283 (0,7 %) | 30080/30283 (99,3 %) | 0/30283 (0,0 %) | 176241 |
| 11de8353 | HI_1_9_0 | 9792 | 225/9792 (2,3 %) | 9567/9792 (97,7 %) | 0/9792 (0,0 %) | 0/9792 (0,0 %) | 9690/9792 (99,0 %) | 0/9792 (0,0 %) | 0/9792 (0,0 %) | 9792/9792 (100,0 %) | 0/9792 (0,0 %) | 177455 |
| 50247b26 | version-31 | 13780 | 3096/13780 (22,5 %) | 10684/13780 (77,5 %) | 0/13780 (0,0 %) | 0/13780 (0,0 %) | 10770/13780 (78,2 %) | 0/13780 (0,0 %) | 0/13780 (0,0 %) | 13780/13780 (100,0 %) | 0/13780 (0,0 %) | 276937 |
| a349fea8 | version-33 | 22206 | 2961/22206 (13,3 %) | 19245/22206 (86,7 %) | 0/22206 (0,0 %) | 0/22206 (0,0 %) | 19277/22206 (86,8 %) | 0/22206 (0,0 %) | 0/22206 (0,0 %) | 22206/22206 (100,0 %) | 0/22206 (0,0 %) | 433476 |

### Corpus : sortie x vue C x reste

| Sortie B | Vue C | Reste | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| rejet hors datum | vide | >= 64 | 249931 | 2978203 |
| terminateur | 1 entree(s) | >= 64 | 7543 | 119464 |
| terminateur | vide | >= 64 | 3995 | 17041 |
| rejet hors datum | 1 entree(s) | >= 64 | 970 | 4321 |
| rejet hors datum | 0 entree (kind 3 seul) | >= 64 | 835 | 5914 |
| terminateur | 2 entree(s) | >= 64 | 525 | 13678 |
| terminateur | 0 entree (kind 3 seul) | >= 64 | 324 | 2102 |
| rejet hors datum | vide | 8-63 | 311 | 607 |
| rejet hors datum | 2 entree(s) | >= 64 | 186 | 898 |
| terminateur | vide | 8-63 | 68 | 387 |
| terminateur | 0 entree (kind 3 seul) | 8-63 | 29 | 0 |
| rejet hors datum | 0 entree (kind 3 seul) | 8-63 | 9 | 51 |
| rejet hors datum | 1 entree(s) | 8-63 | 9 | 19 |
| rejet hors datum | 3 entree(s) | >= 64 | 8 | 64 |
| terminateur | 1 entree(s) | 8-63 | 8 | 52 |
| terminateur | 3 entree(s) | >= 64 | 3 | 12 |
| rejet hors datum | 1 entree(s) | 0-7 (non nuls) | 1 | 0 |
| rejet hors datum | 4 entree(s) | >= 64 | 1 | 9 |
| rejet hors datum | vide | 0-7 (non nuls) | 1 | 8 |

### Corpus : dernier record et dernier composant lus avant la fin de la vue B

| Sortie B | Dernier lu | Paquets | Utiles en jeu |
|---|---|---|---|
| rejet hors datum | DELTA ti=35 i25 unit-command-tick-component | 79041 | 696289 |
| rejet hors datum | DELTA ti=40 i25 unit-command-tick-component | 55996 | 1157854 |
| rejet hors datum | DELTA ti=4 i0 high-frequency | 43715 | 1007 |
| rejet hors datum | DELTA ti=37 i3 object-angular-velocity-component | 17019 | 289982 |
| rejet hors datum | DELTA ti=10 i26 managed-object-rtpc-component | 10025 | 178728 |
| rejet hors datum | DELTA ti=42 i3 object-angular-velocity-component | 7171 | 127923 |
| rejet hors datum | DELTA ti=37 i2 object-forward-and-up-component | 4850 | 62957 |
| rejet hors datum | DELTA ti=41 i2 object-forward-and-up-component | 3966 | 66360 |
| rejet hors datum | DELTA ti=37 i26 equipment-energy-delay-ticks-left-component | 2897 | 53973 |
| rejet hors datum | DELTA ti=40 i34 vehicle-type-physics-component | 2240 | 46712 |
| rejet hors datum | DELTA ti=42 i2 object-forward-and-up-component | 2092 | 36412 |
| terminateur | DELTA ti=40 i25 unit-command-tick-component | 1674 | 13027 |
| rejet hors datum | DELTA ti=41 i20 projectile-command_tick | 1654 | 35812 |
| terminateur | DELTA ti=37 i3 object-angular-velocity-component | 1563 | 30159 |
| rejet hors datum | DELTA ti=12 i14 managed-navpoint-radial-progress | 1549 | 25616 |
| rejet hors datum | DELTA ti=42 i1 object-translational-velocity-component | 1434 | 24410 |
| rejet hors datum | aucun record | 1410 | 0 |
| rejet hors datum | DELTA ti=43 i18 device-position-component | 1361 | 7334 |
| rejet hors datum | DELTA ti=40 i37 vehicle-emp-timer-component | 1061 | 20709 |
| terminateur | DEL | 1046 | 14879 |
| terminateur | DELTA ti=10 i26 managed-object-rtpc-component | 1019 | 23782 |
| rejet hors datum | DELTA ti=41 i3 object-angular-velocity-component | 1012 | 18649 |
| terminateur | DELTA ti=4 i0 high-frequency | 1008 | 98 |
| rejet hors datum | DELTA ti=35 i59 biped-spartan-ability-non-predicted-state | 956 | 8914 |
| terminateur | DELTA ti=43 i18 device-position-component | 828 | 344 |
| rejet hors datum | DELTA ti=20 i1 spawn-filter-weight-component | 745 | 8324 |
| rejet hors datum | DEL | 718 | 4346 |
| rejet hors datum | DELTA ti=37 i29 equipment-command-tick-component | 686 | 14222 |
| terminateur | DELTA ti=42 i3 object-angular-velocity-component | 684 | 18118 |
| rejet hors datum | DELTA ti=35 i28 unit-active-camo-state-component | 655 | 4800 |
| rejet hors datum | DELTA ti=47 i2 personal-ai-data-component | 650 | 4301 |
| rejet hors datum | DELTA ti=21 i16 flock-position-component | 630 | 1 |
| rejet hors datum | DELTA ti=35 i5 object-shield-vitality-component | 599 | 2894 |
| rejet hors datum | DELTA ti=40 i4 object-body-vitality-component | 486 | 8707 |
| rejet hors datum | DELTA ti=37 i1 object-translational-velocity-component | 453 | 8015 |
| rejet hors datum | DELTA ti=41 i1 object-translational-velocity-component | 427 | 7869 |
| rejet hors datum | DELTA ti=35 i54 biped-mobility-action-component | 417 | 3880 |
| rejet hors datum | DELTA ti=38 i0 object-position-component | 402 | 2028 |
| terminateur | DELTA ti=3 i1 high-frequency | 386 | 8 |
| terminateur | DELTA ti=21 i16 flock-position-component | 337 | 1 |

## Sorties de la vue B, tous paquets qui l atteignent

| Build | Sortie B | Paquets | Fermes | Hors cadre | Autres causes |
|---|---|---|---|---|---|
| HI_1_10_0 | ouverte | 2094 | 0/2094 (0,0 %) | 0/2094 (0,0 %) | 2094/2094 (100,0 %) |
| HI_1_10_0 | rejet hors datum | 83184 | 3756/83184 (4,5 %) | 79180/83184 (95,2 %) | 248/83184 (0,3 %) |
| HI_1_10_0 | terminateur | 25960 | 24985/25960 (96,2 %) | 784/25960 (3,0 %) | 191/25960 (0,7 %) |
| HI_1_11_0 | ouverte | 266 | 0/266 (0,0 %) | 0/266 (0,0 %) | 266/266 (100,0 %) |
| HI_1_11_0 | rejet hors datum | 10229 | 61/10229 (0,6 %) | 9910/10229 (96,9 %) | 258/10229 (2,5 %) |
| HI_1_11_0 | terminateur | 4602 | 4224/4602 (91,8 %) | 288/4602 (6,3 %) | 90/4602 (2,0 %) |
| HI_1_12_0 | ouverte | 12425 | 0/12425 (0,0 %) | 0/12425 (0,0 %) | 12425/12425 (100,0 %) |
| HI_1_12_0 | rejet hors datum | 3270 | 2/3270 (0,1 %) | 3200/3270 (97,9 %) | 68/3270 (2,1 %) |
| HI_1_12_0 | terminateur | 5886 | 5833/5886 (99,1 %) | 49/5886 (0,8 %) | 4/5886 (0,1 %) |
| HI_1_13_0 | ouverte | 6379 | 0/6379 (0,0 %) | 0/6379 (0,0 %) | 6379/6379 (100,0 %) |
| HI_1_13_0 | rejet hors datum | 86562 | 590/86562 (0,7 %) | 84825/86562 (98,0 %) | 1147/86562 (1,3 %) |
| HI_1_13_0 | terminateur | 226930 | 224228/226930 (98,8 %) | 2096/226930 (0,9 %) | 606/226930 (0,3 %) |
| HI_1_4_1 | ouverte | 342 | 0/342 (0,0 %) | 0/342 (0,0 %) | 342/342 (100,0 %) |
| HI_1_4_1 | rejet hors datum | 6526 | 7/6526 (0,1 %) | 6362/6526 (97,5 %) | 157/6526 (2,4 %) |
| HI_1_4_1 | terminateur | 2971 | 694/2971 (23,4 %) | 2002/2971 (67,4 %) | 275/2971 (9,3 %) |
| HI_1_8_0 | ouverte | 1284 | 0/1284 (0,0 %) | 0/1284 (0,0 %) | 1284/1284 (100,0 %) |
| HI_1_8_0 | rejet hors datum | 29997 | 98/29997 (0,3 %) | 29289/29997 (97,6 %) | 610/29997 (2,0 %) |
| HI_1_8_0 | terminateur | 15155 | 13871/15155 (91,5 %) | 994/15155 (6,6 %) | 290/15155 (1,9 %) |
| HI_1_9_0 | ouverte | 249 | 0/249 (0,0 %) | 0/249 (0,0 %) | 249/249 (100,0 %) |
| HI_1_9_0 | rejet hors datum | 9788 | 53/9788 (0,5 %) | 9567/9788 (97,7 %) | 168/9788 (1,7 %) |
| HI_1_9_0 | terminateur | 5951 | 5686/5951 (95,5 %) | 225/5951 (3,8 %) | 40/5951 (0,7 %) |
| version-31 | ouverte | 280 | 0/280 (0,0 %) | 0/280 (0,0 %) | 280/280 (100,0 %) |
| version-31 | rejet hors datum | 10708 | 8/10708 (0,1 %) | 10684/10708 (99,8 %) | 16/10708 (0,1 %) |
| version-31 | terminateur | 4674 | 145/4674 (3,1 %) | 3096/4674 (66,2 %) | 1433/4674 (30,7 %) |
| version-33 | ouverte | 954 | 0/954 (0,0 %) | 0/954 (0,0 %) | 954/954 (100,0 %) |
| version-33 | rejet hors datum | 19428 | 23/19428 (0,1 %) | 19245/19428 (99,1 %) | 160/19428 (0,8 %) |
| version-33 | terminateur | 4328 | 440/4328 (10,2 %) | 2961/4328 (68,4 %) | 927/4328 (21,4 %) |
| **corpus** | ouverte | 24273 | 0/24273 (0,0 %) | 0/24273 (0,0 %) | 24273/24273 (100,0 %) |
| **corpus** | rejet hors datum | 259692 | 4598/259692 (1,8 %) | 252262/259692 (97,1 %) | 2832/259692 (1,1 %) |
| **corpus** | terminateur | 296457 | 280106/296457 (94,5 %) | 12495/296457 (4,2 %) | 3856/296457 (1,3 %) |

## Sorties par rejet contre le bloc de type 1 (item 1.2)

Etat du slot rejete dans le bloc de type 1 de SON chunk ; naissance : le bloc du chunk SUIVANT porte-t-il une allocation de cet eid que le chunk ne portait pas, sans NEW lu ?

| Sortie B | Paquet | Etat au bloc | Naissance | Paquets | Utiles en jeu |
|---|---|---|---|---|---|
| rejet hors datum | hors cadre | vide | naissance non lue | 117824 | 1621645 |
| rejet hors datum | hors cadre | trace | naissance non lue | 95209 | 1121045 |
| rejet hors datum | hors cadre | vivant | vivant au bloc du chunk | 14723 | 36054 |
| rejet hors datum | hors cadre | trace | realloue sous une autre generation | 12866 | 149793 |
| rejet hors datum | hors cadre | vide | aucune allocation | 5773 | 35305 |
| rejet hors datum | hors cadre | vide | non mesurable | 3856 | 12388 |
| rejet hors datum | ferme | vide | aucune allocation | 3045 | 0 |
| rejet hors datum | vue C : kind non porte | vide | aucune allocation | 1836 | 10686 |
| rejet hors datum | ferme | trace | aucune allocation | 1261 | 0 |
| rejet hors datum | hors cadre | trace | non mesurable | 1000 | 6718 |
| rejet hors datum | hors cadre | trace | aucune allocation | 667 | 5183 |
| rejet hors datum | vue C : bloc 0xbc non porte | vide | aucune allocation | 584 | 2475 |
| rejet hors datum | hors cadre | trace | libere avant le chunk (meme generation) | 173 | 1303 |
| rejet hors datum | vue C : kind non porte | trace | aucune allocation | 157 | 1158 |
| rejet hors datum | ferme | trace | libere avant le chunk (meme generation) | 138 | 0 |
| rejet hors datum | vue C : debordement | vide | aucune allocation | 100 | 76 |
| rejet hors datum | ferme | vide | realloue sous une autre generation | 97 | 0 |
| rejet hors datum | vue C : kind non porte | trace | libere avant le chunk (meme generation) | 91 | 709 |
| rejet hors datum | hors cadre | vivant | non mesurable | 64 | 184 |
| rejet hors datum | hors cadre | vide | NEW lu dans le chunk | 44 | 56 |
| rejet hors datum | ferme | trace | non mesurable | 36 | 0 |
| rejet hors datum | hors cadre | vide | realloue sous une autre generation | 36 | 291 |
| rejet hors datum | vue C : bloc 0xbc non porte | trace | aucune allocation | 21 | 143 |
| rejet hors datum | ferme | trace | realloue sous une autre generation | 13 | 0 |
| rejet hors datum | vue C : kind non porte | vide | realloue sous une autre generation | 11 | 50 |
| rejet hors datum | hors cadre | vivant | aucune allocation | 10 | 54 |
| rejet hors datum | vue C : bloc 0xbc non porte | trace | libere avant le chunk (meme generation) | 9 | 88 |
| rejet hors datum | hors cadre | vivant | NEW lu dans le chunk | 8 | 0 |
| rejet hors datum | vue C : kind non porte | vide | non mesurable | 8 | 9 |
| rejet hors datum | vue C : kind non porte | trace | realloue sous une autre generation | 5 | 28 |
| rejet hors datum | hors cadre | trace | NEW lu dans le chunk | 4 | 28 |
| rejet hors datum | hors cadre | vivant | realloue sous une autre generation | 4 | 47 |
| rejet hors datum | vue C : kind non porte | trace | naissance non lue | 4 | 11 |
| rejet hors datum | ferme | vide | naissance non lue | 2 | 0 |
| rejet hors datum | ferme | vide | non mesurable | 2 | 0 |
| rejet hors datum | ferme | vivant | realloue sous une autre generation | 2 | 0 |
| rejet hors datum | ferme | vivant | vivant au bloc du chunk | 2 | 0 |
| rejet hors datum | hors cadre | absent | aucune allocation | 1 | 0 |
| rejet hors datum | vue C : bloc 0xbc non porte | vide | naissance non lue | 1 | 22 |
| rejet hors datum | vue C : bloc 0xbc non porte | vide | realloue sous une autre generation | 1 | 15 |

| Build | Rejets | dont paquet hors cadre | dont ferme | Vivant au bloc | Naissance non lue | Non mesurable |
|---|---|---|---|---|---|---|
| HI_1_10_0 | 83184 | 79180/83184 (95,2 %) | 3756/83184 (4,5 %) | 10702/83184 (12,9 %) | 63930/83184 (76,9 %) | 786/83184 (0,9 %) |
| HI_1_11_0 | 10229 | 9910/10229 (96,9 %) | 61/10229 (0,6 %) | 55/10229 (0,5 %) | 8453/10229 (82,6 %) | 0/10229 (0,0 %) |
| HI_1_12_0 | 3270 | 3200/3270 (97,9 %) | 2/3270 (0,1 %) | 0/3270 (0,0 %) | 3064/3270 (93,7 %) | 0/3270 (0,0 %) |
| HI_1_13_0 | 86562 | 84825/86562 (98,0 %) | 590/86562 (0,7 %) | 2261/86562 (2,6 %) | 71292/86562 (82,4 %) | 2965/86562 (3,4 %) |
| HI_1_4_1 | 6526 | 6362/6526 (97,5 %) | 7/6526 (0,1 %) | 0/6526 (0,0 %) | 5807/6526 (89,0 %) | 330/6526 (5,1 %) |
| HI_1_8_0 | 29997 | 29289/29997 (97,6 %) | 98/29997 (0,3 %) | 203/29997 (0,7 %) | 25813/29997 (86,1 %) | 204/29997 (0,7 %) |
| HI_1_9_0 | 9788 | 9567/9788 (97,7 %) | 53/9788 (0,5 %) | 137/9788 (1,4 %) | 7674/9788 (78,4 %) | 191/9788 (2,0 %) |
| version-31 | 10708 | 10684/10708 (99,8 %) | 8/10708 (0,1 %) | 1/10708 (0,0 %) | 10145/10708 (94,7 %) | 486/10708 (4,5 %) |
| version-33 | 19428 | 19245/19428 (99,1 %) | 23/19428 (0,1 %) | 1457/19428 (7,5 %) | 16862/19428 (86,8 %) | 4/19428 (0,0 %) |
| **corpus** | 259692 | 252262/259692 (97,1 %) | 4598/259692 (1,8 %) | 14816/259692 (5,7 %) | 213040/259692 (82,0 %) | 4966/259692 (1,9 %) |

## Entrees de controle utiles (item 1.3)

Utile = entree `kind 0` qui porte le bloc de 0x68 octets, la seule que le tir continu lit. Denominateur ESTIME = utiles fermees + moyenne par vue C fermee du film x paquets non fermes. PAR CONSTRUCTION, la part estimee d un film egale sa part de vues C fermees (U / (U + U/F x N) = F / (F + N)) : cette estimation ne dit rien de plus que les paquets.

| Build | Paquets | Vues C fermees | Entrees fermees | Utiles fermees | Moyenne par vue C fermee | Denominateur estime | Part estimee | Utiles lues hors fermeture (non prouvees) |
|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 129369 | 28741/129369 (22,2 %) | 420684 | 420612 | 14.63 | 1924866 | 21,9 % | 68 |
| HI_1_11_0 | 16824 | 4285/16824 (25,5 %) | 76224 | 76222 | 17.79 | 299267 | 25,5 % | 51 |
| HI_1_12_0 | 21864 | 5835/21864 (26,7 %) | 30137 | 30136 | 5.16 | 112921 | 26,7 % | 10 |
| HI_1_13_0 | 335960 | 224818/335960 (66,9 %) | 1619514 | 1619501 | 7.20 | 2420887 | 66,9 % | 207 |
| HI_1_4_1 | 11130 | 701/11130 (6,3 %) | 5 | 0 | 0.00 | 0 | - | 2015 |
| HI_1_8_0 | 49696 | 13969/49696 (28,1 %) | 79294 | 79279 | 5.68 | 282042 | 28,1 % | 104 |
| HI_1_9_0 | 17629 | 5739/17629 (32,6 %) | 75327 | 75327 | 13.13 | 231389 | 32,6 % | 10 |
| version-31 | 17919 | 153/17919 (0,9 %) | 3 | 2 | 0.01 | 234 | 0,9 % | 4180 |
| version-33 | 28751 | 463/28751 (1,6 %) | 6 | 3 | 0.01 | 186 | 1,6 % | 3621 |
| **corpus** | 629142 | 284704/629142 (45,3 %) | 2301194 | 2301082 | 8.08 | 5271792 | 43,6 % | 10266 |

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
| HI_1_10_0 | 1799156 | 15/1799156 (0,0 %) | 62 | 8 | 15 | 3 | 2 | 0 |
| HI_1_11_0 | 323863 | 3/323863 (0,0 %) | 31 | 1 | 3 | 1 | 0 | 0 |
| HI_1_12_0 | 168157 | 2/168157 (0,0 %) | 10 | 0 | 2 | 1 | 0 | 0 |
| HI_1_13_0 | 3355181 | 270/3355181 (0,0 %) | 1630 | 68 | 270 | 11 | 7 | 0 |
| HI_1_4_1 | 226385 | 1/226385 (0,0 %) | 2 | 1 | 1 | 1 | 1 | 0 |
| HI_1_8_0 | 338252 | 337/338252 (0,1 %) | 1312 | 79 | 337 | 12 | 1 | 0 |
| HI_1_9_0 | 275918 | 19/275918 (0,0 %) | 83 | 6 | 19 | 1 | 1 | 0 |
| version-31 | 542836 | 4/542836 (0,0 %) | 13 | 1 | 4 | 1 | 1 | 0 |
| version-33 | 553211 | 7/553211 (0,0 %) | 38 | 5 | 7 | 1 | 1 | 0 |
| **corpus** | 7582959 | 658/7582959 (0,0 %) | 3181 | 169 | 658 | 32 | 14 | 0 |


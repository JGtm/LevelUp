# Cuisson des fonds de carte — 2026-08-12

Habillage `jeu` · 0.0920 m/px · sortie `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\reference\map_backgrounds`

`ancres avec sol` est l'ORACLE FAIBLE : une ancre d'objectif sans sol dessine
sous elle est un trou de reconstruction. `matiere` compte les instances de bsp
dessinees pour une carte native, et les OBJETS poses pour une carte Forge — une
carte Forge n'a pas d'instance de bsp dessinee, sa carte EST son rack d'objets.

`toits` est la part de matiere qui cache un sol praticable (voie de reference,
himap/rendu_reference.go) : au-dela d'un tiers la carte est COUVERTE et montre
l'etage de jeu au lieu du plafond, dans la portee des ancres.

| carte | noms | px | Ko | ancres avec sol | matiere | frontiere | eau | toits | degradations |
|---|---|---|---:|---|---|---|---:|---|---|
| `btb_drydock` | deadlock | 1892x1937 | 1258 | 46/46 | 3538 instances | 76 plans | 129926 | 21 % |  |
| `btb_engine` | scarr | 2021x1593 | 524 | 19/19 | 6287 instances | 44 plans | 7167 | COUVERTE 46 % (359228 px) |  |
| `btb_exiled` | oasis | 1900x2204 | 735 | 30/35 | 9076 instances | 12 plans | 95096 | 23 % |  |
| `btb_fragmentation` | fragmentation | 1632x2065 | 1343 | 36/46 | 3753 instances | 60 plans | 0 | 20 % | eau : aucun volume d'eau dans le sddt de btb_fragmentation-rtx-new.module |
| `btb_highpower` | highpower | 1955x2055 | 956 | 38/51 | 7475 instances | 76 plans | 13515 | 23 % |  |
| `catalyst` | Catalyst | 1406x1553 | 221 | 24/24 | 7796 instances | 20 plans | 0 | 28 % | eau : aucun volume d'eau dans le sddt de catalyst-rtx-new.module |
| `chasm` | chasm | 1418x1479 | 86 | 17/17 | 1507 instances | 12 plans | 0 | COUVERTE 37 % (121549 px) | eau : aucun volume d'eau dans le sddt de chasm-rtx-new.module |
| `ctf_aquarius` | aquarius | 1501x1374 | 61 | 22/22 | 14761 instances | 12 plans | 0 | COUVERTE 67 % (116018 px) | eau : aucun volume d'eau dans le sddt de ctf_aquarius-rtx-new.module |
| `ctf_bazaar` | bazaar | 1581x1291 | 126 | 12/12 | 8483 instances | 12 plans | 0 | 5 % | eau : aucun volume d'eau dans le sddt de ctf_bazaar-rtx-new.module |
| `ctf_breaker` | breaker | 1553x2046 | 527 | 41/42 | 7563 instances | 12 plans | 0 | 31 % | eau : aucun volume d'eau dans le sddt de ctf_breaker-rtx-new.module |
| `ctf_forbidden` | forbidden | 1345x1434 | 158 | 12/13 | 7116 instances | 28 plans | 7430 | COUVERTE 35 % (390850 px) |  |
| `ctf_illusion` | illusion | 1312x1473 | 146 | 12/12 | 8109 instances | 12 plans | 0 | COUVERTE 38 % (152000 px) | eau : aucun volume d'eau dans le sddt de ctf_illusion-rtx-new.module |
| `fo08_wetland` | Vagabond | 1332x1287 | 76 | 4/4 | 3558/4709 objets Forge | non | 0 | COUVERTE 59 % (257039 px) |  |
| `fo11_blank` | corpo | 1088x1379 | 34 | 4/4 | 1725/1988 objets Forge | non | 0 | COUVERTE 45 % (39536 px) |  |
| `forest` | forest | 1423x1562 | 328 | 14/18 | 6880 instances | 12 plans | 34703 | 21 % |  |
| `ridgeline` | cliffhanger | 1633x1627 | 690 | 14/14 | 5102 instances | 12 plans | 5468 | 25 % |  |
| `sgh_blueprint` | recharge, recharge_-_ranked | 1401x1356 | 43 | 12/12 | 2300 instances | 12 plans | 30970 | 20 % |  |
| `sgh_crystalcaves` | prism | 1400x1436 | 476 | 14/14 | 3997 instances | 20 plans | 2070 | COUVERTE 61 % (310884 px) |  |
| `sgh_interlock` | live_fire_-_ranked, live_fire | NON CUISINABLE | | | | | | | instances du module : himap: aucun tag sbsp dans C:\Program Files (x86)\Steam\steamapps\common\Halo Infinite\deploy\pc\levels\multi\sgh_interlock\sgh_interlock-rtx-new.module |
| `sgh_streets` | streets | 1353x1417 | 73 | 16/16 | 5824 instances | 60 plans | 0 | 7 % | eau : aucun volume d'eau dans le sddt de sgh_streets-rtx-new.module |
| `va_behemoth` | behemoth | 1431x1550 | 266 | 14/14 | 6515 instances | 116 plans | 0 | 19 % | eau : aucun volume d'eau dans le sddt de va_behemoth-rtx-new.module |
| `va_launchsite` | launch_site, launch site | 1468x1584 | 147 | 12/13 | 12861 instances | 24 plans | 52289 | COUVERTE 36 % (199713 px) |  |

1 modules du catalogue non installes localement : map

# Mesure GB-1 des vies du bipede — 2026-09-27

Films mesures : 19 ; echecs : 0. Contexte d instrument (largeurs d axe lues dans le film, profil par defaut, sans calibration killsource).

## GB-1 — vies du bipede par generation du handle

Vie = (slot, generation) connue par un record de creation ou d image-cle. Filtre prod = `DefaultScanFilmOptions` (tag 1), en quanta (sans filtre de vitesse). Generation vivante = en-tetes du meme marcheur filtre desarme dont (slot, tag) est une vie connue, isolement 15 s par (slot, tag). Orphelin = en-tete brut dont (slot, tag) n est aucune vie connue (faux positif potentiel). Durees en ms ; `durationMs` publie reconstitue par la formule de `replay` sur les positions prod.

| Film | Build | Statut | Vies | dont gen >= 2 | Slots a plusieurs vies | Creations gen 0/1/2/3 | Slots crees gen >= 2 | Sans position (prod) | dont gen >= 2 | Sans position (gen. vivante) | Gen >= 2 positionnees (gen. vivante) | Orphelins bruts (gardes) | Orphelins prod | durationMs publie | Duree des paquets delta (tous paquets) | Duree vivante |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 084a804d | HI_1_10_0 | mesure | 379 | 123 | 123 | 0/256/123/0 | 123 | 123 | 123 | 0 | 123 | 129 (24) | 0 | 972400 | 1084101 (1084183) | 1055800 |
| 1c4c63c2 | HI_1_10_0 | mesure | 586 | 330 | 256 | 0/256/256/74 | 256 | 330 | 330 | 0 | 330 | 164 (64) | 0 | 825900 | 1333456 (1333557) | 1288700 |
| a349fea8 | version-33 | mesure | 323 | 67 | 67 | 0/249/67/0 | 67 | 74 | 67 | 8 | 66 | 299 (38) | 0 | 933100 | 965100 (965163) | 948000 |
| bcb6d393 | HI_1_12_0 | mesure | 57 | 0 | 0 | 0/57/0/0 | 0 | 0 | 0 | 0 | 0 | 6 (0) | 0 | 346500 | 365437 (365465) | 346500 |
| fb1a1a72 | HI_1_13_0 | mesure | 145 | 0 | 0 | 0/145/0/0 | 0 | 1 | 0 | 1 | 0 | 124 (101) | 0 | 756800 | 814499 (814533) | 756800 |
| d9781168 | HI_1_13_0 | mesure | 160 | 0 | 0 | 0/160/0/0 | 0 | 0 | 0 | 0 | 0 | 76 (40) | 0 | 705100 | 728690 (728722) | 705100 |
| c75f33b8 | HI_1_13_0 | mesure | 80 | 0 | 0 | 0/80/0/0 | 0 | 0 | 0 | 0 | 0 | 31 (10) | 0 | 411700 | 464431 (464459) | 411700 |
| bf15f7ab | HI_1_13_0 | mesure | 83 | 0 | 0 | 0/83/0/0 | 0 | 0 | 0 | 0 | 0 | 12 (4) | 0 | 488900 | 519134 (519166) | 488900 |
| 51ebbc0f | HI_1_13_0 | mesure | 84 | 0 | 0 | 0/84/0/0 | 0 | 0 | 0 | 0 | 0 | 52 (44) | 0 | 451400 | 512518 (512549) | 451400 |
| 0797ce72 | HI_1_13_0 | mesure | 101 | 0 | 0 | 0/99/0/0 | 0 | 2 | 0 | 2 | 0 | 22 (5) | 0 | 436700 | 446633 (446663) | 436700 |
| 111fa685 | HI_1_10_0 | mesure | 212 | 0 | 0 | 0/212/0/0 | 0 | 0 | 0 | 0 | 0 | 62 (14) | 0 | 526500 | 580189 (580268) | 526500 |
| e5adf7b2 | HI_1_11_0 | mesure | 223 | 0 | 0 | 0/217/0/0 | 0 | 7 | 0 | 7 | 0 | 85 (14) | 0 | 539900 | 561850 (561923) | 539900 |
| 60ae07c4 | HI_1_8_0 | mesure | 173 | 0 | 0 | 0/171/0/0 | 0 | 4 | 0 | 4 | 0 | 58 (9) | 0 | 811700 | 829574 (829611) | 811700 |
| a521164d | HI_1_4_1 | mesure | 132 | 0 | 0 | 0/128/0/0 | 0 | 4 | 0 | 4 | 0 | 85 (3) | 0 | 339700 | 373520 (373580) | 339700 |
| 11de8353 | HI_1_9_0 | mesure | 182 | 0 | 0 | 0/182/0/0 | 0 | 0 | 0 | 0 | 0 | 38 (6) | 0 | 514400 | 607269 (607348) | 514400 |
| 50247b26 | version-31 | mesure | 181 | 0 | 0 | 0/175/0/0 | 0 | 9 | 0 | 9 | 0 | 187 (50) | 0 | 588200 | 599223 (599284) | 588200 |
| bfecd02b | HI_1_13_0 | mesure | 90 | 0 | 0 | 0/90/0/0 | 0 | 0 | 0 | 0 | 0 | 39 (6) | 0 | 503300 | 521856 (521886) | 503300 |
| 4f77afc1 | HI_1_13_0 | mesure | 333 | 77 | 77 | 0/256/77/0 | 77 | 77 | 77 | 0 | 77 | 199 (49) | 0 | 1111100 | 1203693 (1203764) | 1111100 |
| 396cfc92 | HI_1_13_0 | mesure | 73 | 0 | 0 | 0/73/0/0 | 0 | 1 | 0 | 1 | 0 | 22 (14) | 0 | 501900 | 536030 (536062) | 501900 |


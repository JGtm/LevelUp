# Synthese de vitesse, decodeur v1.6.0 contre v1.5.0

Medianes du temps total (decodeur seul, options minimales, ms), 5 passes par serie, 13 films du banc.
Releve v1.6.0 : `bench/releves/v1.6.0.tsv` (arbre v1.5.0-130-g61fbd41, 2026-11-24). Releve v1.5.0 : `bench/releves/v1.5.0.tsv`
(un seul fil, mesure le 2026-11-23). Gain = (v1.5.0 - v1.6.0) / v1.5.0. Les 4 series 1/2/4/8 fils ont des empreintes
identiques entre elles ; elles different de celles de la v1.5.0 (sortie changee par la version).
Plafond du pic : 2 Gio en v1.6.0, 3 Gio en v1.5.0.

## Les 13 films

| film | mode | joueurs | v1.5.0 | 1 fil | 2 fils | 4 fils | 8 fils | gain 1 | gain 8 | pic 1.5.0 | pic 1.6.0 (1/8 fils) |
|---|---|---|---|---|---|---|---|---|---|---|---|
| film01 | Grande bataille (Slayer Fiesta) | 28 | 2310 | 1921 | 1554 | 1320 | 1244 | 17% | 46% | 508 | 413 / 446 |
| film02 | Slayer | 8 | 1123 | 860 | 630 | 578 | 554 | 23% | 51% | 250 | 217 / 231 |
| film03 | Slayer | 8 | 1461 | 1131 | 871 | 740 | 698 | 23% | 52% | 329 | 334 / 293 |
| film04 | Drapeau (BTB, un drapeau) | 24 | 19644 | 10260 | 9249 | 8814 | 8606 | 48% | 56% | 2198 | 1869 / 1704 |
| film05 | Drapeau (arène, drapeau neutre) | 13 | 525 | 413 | 277 | 261 | 257 | 21% | 51% | 128 | 111 / 110 |
| film06 | Grande bataille (Slayer BTB) | 30 | 18096 | 5659 | 5066 | 4815 | 4691 | 69% | 74% | 454 | 404 / 386 |
| film07 | Grande bataille (Drapeau Heavies) | 26 | 8031 | 5315 | 4219 | 3731 | 3479 | 34% | 57% | 992 | 791 / 880 |
| film08 | Grande bataille (Slayer Fiesta) | 27 | 2542 | 1799 | 1420 | 1261 | 1182 | 29% | 53% | 397 | 382 / 437 |
| film09 | Autres (Total Control, BTB Heavies) | 27 | 9701 | 3227 | 2854 | 2680 | 2617 | 67% | 73% | 349 | 291 / 294 |
| film10 | Bases | 8 | 1586 | 1239 | 921 | 810 | 778 | 22% | 51% | 332 | 350 / 298 |
| film11 | Colline | 8 | 1371 | 1087 | 791 | 715 | 681 | 21% | 50% | 379 | 313 / 295 |
| film12 | Crâne (classé) | 8 | 3973 | 1936 | 1556 | 1431 | 1385 | 51% | 65% | 520 | 481 / 483 |
| film13 | Autres (Bombe, Assaut) | 12 | 1197 | 945 | 654 | 594 | 571 | 21% | 52% | 277 | 232 / 238 |
| **Total** | | | 71560 | 35791 | 30062 | 27751 | 26742 | 50% | 63% | | |

Pic du plus gros film (film04) : 1869 Mio a 1 fil, 1704 Mio a 8 fils (2198 Mio en v1.5.0).

## Films de la build la plus recente du jeu (HI_1_13_0)

| film | v1.5.0 | 1 fil | 2 fils | 4 fils | 8 fils | pic 1.5.0 / 1.6.0 (Mio) |
|---|---|---|---|---|---|---|
| film02 | 1123 | 860 | 630 | 578 | 554 | 250 / 217 |
| film05 | 525 | 413 | 277 | 261 | 257 | 128 / 111 |
| film10 | 1586 | 1239 | 921 | 810 | 778 | 332 / 350 |
| film11 | 1371 | 1087 | 791 | 715 | 681 | 379 / 313 |
| film13 | 1197 | 945 | 654 | 594 | 571 | 277 / 232 |

Le plus recent du banc : film13 (Bombe, Assaut, 12 joueurs, 27 morceaux).

<!-- mappower-build --mesure --reglage v1, 2026-09-20 -->

## Corpus mesure

| Carte | Axe | Matchs | Kills | Cellules peuplees | Artefacts | Points de piste | Pistes sans issue | Ecart variantes (m) |
|---|---|---|---|---|---|---|---|---|
| recharge | arene | 78 | 7344 | 2992 | 6 | 195965 | 37 | 2.55 |
| streets | arene | 75 | 6950 | 2947 | 5 | 118661 | 64 | 0.17 |
| aquarius | arene | 57 | 4945 | 3184 | 5 | 143910 | 20 | 1.10 |

## Couverture du rang du tueur (match_csrs_latest, base partagee)

Corpus : 12051 rangs sur 1634 matchs des cartes mesurees ; rampe mesuree sur les 28937 rangs de tout le titre : p10 1251, mediane 1442, p90 1590.

| Carte | Kills lus | Kills exclus (variante) | Kills a rang connu | part | Matchs | Matchs avec >= 1 rang | part |
|---|---|---|---|---|---|---|---|
| recharge | 7344 | 0 | 3442 | 47% | 888 | 686 | 77% |
| streets | 6950 | 0 | 2666 | 38% | 730 | 542 | 74% |
| aquarius | 4945 | 0 | 926 | 19% | 579 | 406 | 70% |

## Plancher de rarete — rayon du nuage (p99 de la distance au barycentre, en m)

| Carte | >=1 match(s) | >=2 match(s) | >=3 match(s) | >=5 match(s) |
|---|---|---|---|---|
| recharge | 19.1 (2694) | 18.7 (2289) | 18.2 (1916) | 16.4 (1142) |
| streets | 19.7 (2641) | 19.4 (2250) | 19.0 (1848) | 16.6 (1153) |
| aquarius | 21.6 (2649) | 20.4 (2041) | 19.3 (1460) | 18.4 (654) |

_Entre parentheses : le nombre de cellules retenues par ce plancher._

## Taille d'echantillon par cellule (engagements = kills depuis + morts dedans)

| Carte | p50 | p75 | p90 | p99 | max | cellules >=6 | cellules >=12 | cellules >=25 |
|---|---|---|---|---|---|---|---|---|
| recharge | 4 | 7 | 10 | 22 | 41 | 1013 | 212 | 25 |
| streets | 4 | 7 | 10 | 17 | 28 | 1008 | 193 | 2 |
| aquarius | 2 | 5 | 7 | 13 | 19 | 536 | 46 | 0 |

## Dispersion des signaux (cellules a >=6 engagements)

| Carte | n | duel p10 | p25 | p50 | p75 | p90 | portee p50 (m) | p90 | denivele p10 (m) | p50 | p90 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| recharge | 1013 | 0.30 | 0.38 | 0.50 | 0.60 | 0.71 | 4.5 | 9.1 | -0.5 | 0.0 | 0.8 |
| streets | 1008 | 0.29 | 0.38 | 0.50 | 0.62 | 0.71 | 4.7 | 9.2 | -0.6 | 0.0 | 0.6 |
| aquarius | 536 | 0.29 | 0.38 | 0.50 | 0.62 | 0.71 | 4.1 | 8.2 | -0.5 | 0.0 | 0.8 |

## Signaux angulaires par disque scorable (dispersion corrigee du biais, cf. powerpos/angles.go)

| Carte | Disques | N sortant p10 / p50 / p90 | N entrant p10 / p50 / p90 | couverture p10 / p50 / p90 | abri p10 / p50 / p90 | asymetrie (couv + abri) / 2 p10 / p50 / p90 | corr(couv, abri) | corr(couv, avantage) | corr(abri, avantage) | corr(asym, avantage) | corr(asym, hauteur) |
|---|---|---|---|---|---|---|---|---|---|---|
| recharge | 1915 | 68 / 121 / 216 | 71 / 120 / 202 | 0.53 / 0.71 / 0.91 | 0.07 / 0.26 / 0.47 | 0.42 / 0.49 / 0.56 | -0.70 | -0.06 | -0.13 | -0.24 | -0.20 |
| streets | 1842 | 63 / 121 / 192 | 66 / 119 / 185 | 0.53 / 0.72 / 0.91 | 0.06 / 0.24 / 0.47 | 0.41 / 0.49 / 0.55 | -0.73 | -0.12 | -0.12 | -0.32 | -0.05 |
| aquarius | 1456 | 47 / 82 / 127 | 48 / 82 / 121 | 0.58 / 0.82 / 1.00 | 0.00 / 0.22 / 0.43 | 0.42 / 0.51 / 0.60 | -0.54 | -0.05 | -0.15 | -0.21 | -0.33 |

## Occupation par equipe (artefacts de rejeu)

| Carte | Artefacts | Cellules avec presence | Cellules a >=3 matchs | ecart p10 | p50 | p90 |
|---|---|---|---|---|---|---|
| recharge | 6 | 2954 | 2673 | -0.36 | 0.06 | 0.46 |
| streets | 5 | 2918 | 2593 | -0.30 | 0.20 | 0.62 |
| aquarius | 5 | 3163 | 2903 | -0.42 | 0.11 | 0.62 |

## Axes du score par disque scorable (p10 / p50 / p90, etalement p90 - p10)

| Carte | avantage | intensite | hauteur | portee | couverture | abri | score |
|---|---|---|---|---|---|---|---|
| recharge | 0.40 / 0.50 / 0.58 [0.19] | 0.76 / 0.87 / 0.97 [0.21] | 0.41 / 0.51 / 0.67 [0.25] | 0.05 / 0.50 / 1.00 [0.95] | 0.53 / 0.71 / 0.91 [0.38] | 0.07 / 0.26 / 0.47 [0.40] | 0.52 / 0.59 / 0.68 [0.16] |
| streets | 0.42 / 0.50 / 0.59 [0.17] | 0.77 / 0.89 / 0.97 [0.20] | 0.38 / 0.51 / 0.61 [0.23] | 0.00 / 0.50 / 1.00 [1.00] | 0.53 / 0.72 / 0.91 [0.39] | 0.06 / 0.24 / 0.47 [0.41] | 0.51 / 0.59 / 0.69 [0.18] |
| aquarius | 0.43 / 0.51 / 0.58 [0.15] | 0.79 / 0.90 / 0.99 [0.20] | 0.43 / 0.54 / 0.62 [0.19] | 0.19 / 0.50 / 1.00 [0.81] | 0.58 / 0.82 / 1.00 [0.42] | 0.00 / 0.22 / 0.43 [0.43] | 0.53 / 0.61 / 0.69 [0.15] |

## Score (reglage v1) — cellules scorables

| Carte | Cellules peuplees | Scorables | part | score p50 | p75 | p90 | p99 | max | seuil applique |
|---|---|---|---|---|---|---|---|---|---|
| recharge | 2992 | 1915 | 64% | 0.593 | 0.634 | 0.680 | 0.748 | 0.793 | 0.680 |
| streets | 2947 | 1842 | 63% | 0.589 | 0.648 | 0.691 | 0.747 | 0.786 | 0.691 |
| aquarius | 3184 | 1456 | 46% | 0.611 | 0.646 | 0.687 | 0.728 | 0.756 | 0.687 |

## Selection — cellules retenues et composantes (avant / apres fermeture)

| Carte | Seuil amorce | Seuil croissance | Cellules amorce | Retenues | Apres fermeture | Composantes avant (tailles, 8 plus grandes) | Composantes apres (tailles) | Retenues (>= taille min) |
|---|---|---|---|---|---|---|---|---|
| recharge | 0.680 | 0.680 | 192 | 192 | 192 | 17 : 52 48 44 14 8 5 5 3 | 17 : 52 48 44 14 8 5 5 3 | 4 |
| streets | 0.691 | 0.691 | 185 | 185 | 185 | 18 : 47 40 38 19 9 8 6 4 | 18 : 47 40 38 19 9 8 6 4 | 4 |
| aquarius | 0.687 | 0.687 | 146 | 146 | 146 | 26 : 29 22 21 21 18 9 3 2 | 26 : 29 22 21 21 18 9 3 2 | 5 |

## Positions retenues

| Carte | Positions | Cellules (min/med/max) | Aire m2 (min/max) | Score moyen (min/max) | Centres (x, y) |
|---|---|---|---|---|---|
| recharge | 4 | 14 / 46 / 52 | 6.0 / 22.5 | 0.692 / 0.725 | (23, 3) (13, 2) (19, -10) (8, 5) |
| streets | 4 | 19 / 39 / 47 | 9.8 / 25.5 | 0.705 / 0.734 | (8, -3) (0, 9) (-5, -9) (-7, 9) |
| aquarius | 5 | 18 / 21 / 29 | 8.1 / 13.8 | 0.702 / 0.714 | (2, 5) (-13, -0) (-14, -6) (2, -4) (13, -0) |


# Mesure des plafonds — 2026-08-25

Instrument `cmd/mapplafond-mesure` · marge 5.0 m · corpus `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/replays/halo_infinite` (35 documents lus, 11 rattaches a une carte)

Definitions des colonnes :

- `sol joue` : altitude du sol de jeu deduite des ancres d'objectifs, publiee par le
  fond de carte (`playLevelZ`). `plafond actuel` = `sol joue` + 28 m, la tranche de jeu
  universelle appliquee aujourd'hui par la cuisson (`himap.TrancheDeJeu`).
- `h med / p99 / max` : altitudes des positions de joueur du corpus, en metres monde
  (champ `tracks[].points[].z` des artefacts cuits).
- `seuil` : le plafond PROPOSE, `h max` + marge.
- `image changee` : part des pixels porteurs de matiere dont la surface AFFICHEE est
  au-dessus du seuil — ce que la coupe changerait a l'ecran.
- `volumes coupes` : instances de geometrie entierement au-dessus du seuil (supprimees)
  et a cheval sur lui (decapitees), sur le total des instances rendues.
- `zones nommees au-dessus` : zones NOMMEES du jeu (tag `levl`) dont le plancher est
  au-dessus du seuil. C'est le detecteur de FAUX POSITIFS independant du corpus : une
  zone nommee est un espace de jeu dessine par le designer, pas un toit.
- `perte si -1 film` : de combien `h max` descendrait si UN film manquait au corpus.
  Une perte de plusieurs metres dit que le corpus borne le CORPUS, pas la carte. Un
  tiret sous DEUX films : le controle n'est pas stable, il est impossible.

## Coupe au maximum frequente + marge

| carte | films | positions | sol joue | h med | h p99 | h max | plafond actuel | seuil | image changee | volumes coupes | zones nommees au-dessus | perte si -1 film |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---:|
| `btb_drydock` | 0 | 0 | 75.8 | — | — | — | 103.8 | — | — | — | — | — |
| `btb_engine` | 0 | 0 | -0.7 | — | — | — | 27.3 | — | — | — | — | — |
| `btb_exiled` | 0 | 0 | 24.3 | — | — | — | 52.3 | — | — | — | — | — |
| `btb_fragmentation` | 0 | 0 | 5.8 | — | — | — | 33.8 | — | — | — | — | — |
| `btb_highpower` | 1 | 23285 | 44.1 | 42.9 | 50.0 | 58.9 | 72.1 | 63.9 | 4.57 % | 175 supprimes (2.06 %) · 343 decapites (4.05 %) / 8479 | 0 | — |
| `catalyst` | 2 | 55209 | 24.4 | 25.2 | 27.8 | 30.7 | 52.4 | 35.7 | 37.54 % | 139 supprimes (1.72 %) · 571 decapites (7.05 %) / 8099 | 0 | 1.2 |
| `chasm` | 1 | 10915 | -136.7 | -138.0 | -134.0 | -132.8 | -108.7 | -127.8 | 15.39 % | 333 supprimes (14.89 %) · 160 decapites (7.15 %) / 2237 | 0 | — |
| `ctf_aquarius` | 0 | 0 | 4.1 | — | — | — | 32.1 | — | — | — | — | — |
| `ctf_bazaar` | 1 | 10457 | 1.2 | 1.2 | 3.9 | 4.3 | 29.2 | 9.3 | 2.67 % | 141 supprimes (1.47 %) · 183 decapites (1.91 %) / 9585 | 0 | — |
| `ctf_breaker` | 1 | 30224 | 16.1 | 16.4 | 21.2 | 25.2 | 44.1 | 30.2 | 6.69 % | 18 supprimes (0.22 %) · 62 decapites (0.75 %) / 8281 | 0 | — |
| `ctf_forbidden` | 0 | 0 | 1.9 | — | — | — | 29.9 | — | — | — | — | — |
| `ctf_illusion` | 0 | 0 | 0.5 | — | — | — | 28.5 | — | — | — | — | — |
| `forest` | 0 | 0 | 1.8 | — | — | — | 29.8 | — | — | — | — | — |
| `ridgeline` | 2 | 48156 | -2.5 | -1.8 | 2.0 | 7.1 | 25.5 | 12.1 | 23.21 % | 85 supprimes (1.36 %) · 362 decapites (5.81 %) / 6231 | 0 | 0.0 |
| `sgh_blueprint` | 2 | 37952 | 1.9 | 2.2 | 5.2 | 5.5 | 29.9 | 10.5 | 13.00 % | 27 supprimes (1.02 %) · 90 decapites (3.38 %) / 2660 | 0 | 0.2 |
| `sgh_crystalcaves` | 0 | 0 | 17.7 | — | — | — | 45.7 | — | — | — | — | — |
| `sgh_streets` | 0 | 0 | 0.7 | — | — | — | 28.7 | — | — | — | — | — |
| `va_behemoth` | 1 | 39959 | 8.7 | 8.4 | 13.2 | 14.7 | 36.7 | 19.7 | 16.06 % | 2 supprimes (0.03 %) · 47 decapites (0.66 %) / 7101 | 0 | — |
| `va_launchsite` | 0 | 0 | -1.3 | — | — | — | 26.7 | — | — | — | — | — |

## Variante : coupe au 99e centile + marge

Ce n'est PAS une proposition : c'est le chiffrage de ce qu'on gagnerait a ignorer le
dernier pour cent des positions (grappin, canon, chute) et de ce qu'on y perdrait.

| carte | seuil p99 | image changee | volumes coupes | zones nommees au-dessus | positions au-dessus |
|---|---:|---:|---|---|---:|
| `btb_highpower` | 55.0 | 18.42 % | 540 supprimes (6.37 %) · 628 decapites (7.41 %) / 8479 | 0 | 90 |
| `catalyst` | 32.8 | 44.19 % | 354 supprimes (4.37 %) · 636 decapites (7.85 %) / 8099 | 0 | 0 |
| `chasm` | -129.0 | 17.02 % | 333 supprimes (14.89 %) · 162 decapites (7.24 %) / 2237 | 0 | 0 |
| `ctf_bazaar` | 8.8 | 2.95 % | 155 supprimes (1.62 %) · 198 decapites (2.07 %) / 9585 | 0 | 0 |
| `ctf_breaker` | 26.2 | 10.13 % | 69 supprimes (0.83 %) · 218 decapites (2.63 %) / 8281 | 1 : Promontoire | 0 |
| `ridgeline` | 7.0 | 33.31 % | 174 supprimes (2.79 %) · 493 decapites (7.91 %) / 6231 | 0 | 5 |
| `sgh_blueprint` | 10.2 | 13.00 % | 27 supprimes (1.02 %) · 92 decapites (3.46 %) / 2660 | 0 | 0 |
| `va_behemoth` | 18.2 | 22.43 % | 2 supprimes (0.03 %) · 58 decapites (0.82 %) / 7101 | 0 | 0 |

## Ou vit la matiere dessinee (ecart au sol joue)

Centiles de l'altitude des pixels porteurs de matiere, en metres AU-DESSUS du sol
joue. `h max - sol` rappelle a quelle hauteur relative monte le corpus.

| carte | sol joue | matiere p50 | p90 | p99 | max | h max - sol | seuil - sol |
|---|---:|---:|---:|---:|---:|---:|---:|
| `btb_drydock` | 75.8 | 1.6 | 11.6 | 18.3 | 28.0 | — | — |
| `btb_engine` | -0.7 | 0.7 | 5.2 | 15.4 | 27.4 | — | — |
| `btb_exiled` | 24.3 | 1.2 | 10.3 | 26.1 | 28.0 | — | — |
| `btb_fragmentation` | 5.8 | 1.3 | 15.6 | 22.4 | 28.0 | — | — |
| `btb_highpower` | 44.1 | 1.0 | 14.5 | 25.7 | 28.0 | 14.9 | 19.9 |
| `catalyst` | 24.4 | 5.7 | 17.5 | 19.3 | 28.0 | 6.3 | 11.3 |
| `chasm` | -136.7 | 0.1 | 12.8 | 22.8 | 27.3 | 3.9 | 8.9 |
| `ctf_aquarius` | 4.1 | -0.9 | 1.3 | 7.3 | 9.9 | — | — |
| `ctf_bazaar` | 1.2 | -0.4 | 6.4 | 10.2 | 13.9 | 3.1 | 8.1 |
| `ctf_breaker` | 16.1 | 0.3 | 10.2 | 17.8 | 21.5 | 9.1 | 14.1 |
| `ctf_forbidden` | 1.9 | 1.1 | 8.7 | 9.0 | 27.4 | — | — |
| `ctf_illusion` | 0.5 | 2.5 | 8.3 | 11.7 | 20.8 | — | — |
| `forest` | 1.8 | 1.3 | 10.9 | 21.0 | 28.0 | — | — |
| `ridgeline` | -2.5 | 2.6 | 21.0 | 27.2 | 28.0 | 9.5 | 14.5 |
| `sgh_blueprint` | 1.9 | 4.5 | 9.6 | 13.4 | 13.6 | 3.6 | 8.6 |
| `sgh_crystalcaves` | 17.7 | 0.6 | 5.9 | 24.5 | 28.0 | — | — |
| `sgh_streets` | 0.7 | 3.6 | 8.2 | 10.9 | 13.5 | — | — |
| `va_behemoth` | 8.7 | 0.6 | 12.0 | 25.5 | 28.0 | 5.9 | 10.9 |
| `va_launchsite` | -1.3 | 1.5 | 8.9 | 21.2 | 28.0 | — | — |

## Cartes ecartees de la mesure

- academy_tutorial (pas de fond publie)
- pve_house (pas de fond publie)
- sgh_interlock (pas de fond publie)

## Films non rattaches a une carte

Un film ECARTE ne pese sur aucune hauteur : mieux vaut une carte sans corpus qu'une
carte dont la hauteur maximale vient d'un match joue ailleurs.

- `00162144` : z=99.5 ; (aucune) 0% contre (aucune) 0%
- `008e1bba` : z=76.4 ; (aucune) 0% contre (aucune) 0%
- `00ba2e1c` : z=101.0 ; (aucune) 0% contre (aucune) 0%
- `06dfe6d9` : z=176.6 ; (aucune) 0% contre (aucune) 0%
- `084a804d` : z=93.7 ; btb_drydock 3% contre (aucune) 0%
- `0a247154` : z=79.7 ; (aucune) 0% contre (aucune) 0%
- `0f9550e5` : z=55.7 ; (aucune) 0% contre (aucune) 0%
- `11de8353` : z=54.1 ; (aucune) 0% contre (aucune) 0%
- `3372e7eb` : z=118.3 ; (aucune) 0% contre (aucune) 0%
- `3685373c` : z=70.7 ; (aucune) 0% contre (aucune) 0%
- `51101d1d` : z=70.8 ; (aucune) 0% contre (aucune) 0%
- `64e8adfa` : z=25.7 ; catalyst 90% contre ctf_breaker 77%
- `696a9d7c` : z=52.6 ; (aucune) 0% contre (aucune) 0%
- `7344d24f` : z=52.6 ; (aucune) 0% contre (aucune) 0%
- `8076f97f` : z=64.7 ; (aucune) 0% contre (aucune) 0%
- `82f29378` : z=24.6 ; ctf_breaker 92% contre btb_exiled 86%
- `83ee3f9f` : z=176.5 ; (aucune) 0% contre (aucune) 0%
- `846044ba` : z=8.2 ; va_behemoth 60% contre (aucune) 0%
- `a17e61a2` : z=77.6 ; btb_drydock 17% contre (aucune) 0%
- `a32ee8d2` : z=82.8 ; (aucune) 0% contre (aucune) 0%
- `af13e2b2` : z=62.3 ; (aucune) 0% contre (aucune) 0%
- `c0a82e88` : z=141.0 ; (aucune) 0% contre (aucune) 0%
- `c5c9db26` : z=61.4 ; btb_drydock 17% contre (aucune) 0%
- `cf040013` : z=72.2 ; (aucune) 0% contre (aucune) 0%

## Degradations de cuisson

- `btb_fragmentation` : eau : aucun volume d'eau dans le sddt de btb_fragmentation-rtx-new.module
- `catalyst` : eau : aucun volume d'eau dans le sddt de catalyst-rtx-new.module
- `chasm` : eau : aucun volume d'eau dans le sddt de chasm-rtx-new.module
- `ctf_aquarius` : eau : aucun volume d'eau dans le sddt de ctf_aquarius-rtx-new.module
- `ctf_bazaar` : eau : aucun volume d'eau dans le sddt de ctf_bazaar-rtx-new.module
- `ctf_breaker` : eau : aucun volume d'eau dans le sddt de ctf_breaker-rtx-new.module
- `ctf_illusion` : eau : aucun volume d'eau dans le sddt de ctf_illusion-rtx-new.module
- `sgh_streets` : eau : aucun volume d'eau dans le sddt de sgh_streets-rtx-new.module
- `va_behemoth` : eau : aucun volume d'eau dans le sddt de va_behemoth-rtx-new.module


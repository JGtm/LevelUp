# Films recents du decodeur : ou part le temps, et mesure a blanc du parallelisme

Date : 2026-11-23. Tete mesuree : `785c38e` (branche `feat/v1.6.0`), depot lu en lecture seule. Instrumentation dans un worktree
temporaire detache, retire a la fin (aucun commit, aucune jonction) ; le patch complet est range dans `instrumentation/`.
Machine : 16 coeurs, un decodage a la fois, plus de 13 Gio de memoire libre, aucun autre `film-bench` pendant les mesures.
Marques : MESURE = compteur, chronometre, profil ou sonde ; DEDUIT = lecture du code ou estimation (avec ce qui la verifierait).
Films : les six films recents `film02`, `film03` (build HI_1_12_0, pas 41), `film05`, `film10`, `film11`, `film13` ; en comparaison, pour la
partie 2, `film01`, `film04`, `film06`, `film12`. La « moyenne des cinq » = film02, 03, 10, 11, 13 (film05 est donne a part).

## Methode et limites

- Temps mural : chronometres poses aux frontieres de chaque etape (patch), 15 passes par film apres un echauffement, medianes. Les etapes se
  somment au total mesure a moins de 1 % pres (ecart de 2 a 13 ms par film). Les chronometres coutent moins de 3 ms par passe.
- CPU : 30 passes sous un meme profil pprof a 1 kHz, chaque echantillon range dans UNE etape (premier marqueur de la racine a la feuille).
  Sous Windows le profileur ne retient qu environ 80 % du temps reel : lire les PARTS, pas les durees. Octets alloues : profil du tas
  (un echantillon tous les 4 Kio), 3 passes.
- Sonde du parallelisme : voir partie 2. Durees par morceau : medianes de 3 passes sans ombres (la sonde complete ralentit la marche).
- Un seul poste ; le jeu fixe n est pas un tirage au hasard ; les ecarts de quelques % ne sont pas significatifs.

## Partie 1. Ou part le temps d un film recent

### 1.1 Reponse courte (MESURE)

Un film recent se partage en quatre blocs : la **famille des images-cles** (4 parcours des memes paquets : environ 40 % du temps mural),
le **decodage des trames** (22 % avec les crochets des canaux), les **balayages hors images-cles apres la marche** (environ 25 %) et la
**publication** (9 %). Le chargement ne pese rien (0,8 %). La recherche a tatons de la liste d evenements, les essais de monde et l ancrage
bit a bit sont a 0 % sur ces films (1.2).

### A. Temps mural par etape exclusive (medianes de 15 passes, ms et % du total)
| etape | film02 | film03 | film10 | film11 | film13 | film05 | moyenne des 5 (%) |
|---|---|---|---|---|---|---|---|
| TOTAL (chargement+decodage+publication) | 1006 | 1328 | 1439 | 1241 | 1093 | 480 | |
| chargement (lecture, decompression, decoupage) | 10 (0.9 %) | 10 (0.7 %) | 11 (0.7 %) | 10 (0.8 %) | 9 (0.8 %) | 6 (1.3 %) | 0.8 % |
| avant le balayage (contexte, profil, carte) | 0 (0.0 %) | 0 (0.0 %) | 1 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0.0 % |
| marche : preparation (registre, bande de slots bipedes) | 97 (9.6 %) | 172 (12.9 %) | 197 (13.7 %) | 169 (13.6 %) | 137 (12.5 %) | 47 (9.8 %) | 12.5 % |
| &nbsp;&nbsp;dont dont bande de slots bipedes (marche d ancres des images-cles) | 94 (9.4 %) | 170 (12.8 %) | 194 (13.5 %) | 166 (13.3 %) | 134 (12.3 %) | 47 (9.7 %) | 12.3 % |
| marche : images-cles | 149 (14.8 %) | 202 (15.2 %) | 217 (15.1 %) | 179 (14.5 %) | 165 (15.1 %) | 73 (15.1 %) | 14.9 % |
| marche : trames, decodage des paquets | 169 (16.8 %) | 205 (15.4 %) | 231 (16.1 %) | 205 (16.5 %) | 165 (15.1 %) | 63 (13.1 %) | 16.0 % |
| marche : trames, crochets des canaux (distribution) | 66 (6.6 %) | 91 (6.8 %) | 90 (6.2 %) | 85 (6.8 %) | 66 (6.0 %) | 25 (5.3 %) | 6.5 % |
| marche : trames, reste de la boucle | 2 (0.2 %) | 3 (0.2 %) | 3 (0.2 %) | 1 (0.1 %) | 0 (0.0 %) | 1 (0.2 %) | 0.2 % |
| marche : cloture des canaux | 11 (1.1 %) | 14 (1.0 %) | 15 (1.0 %) | 14 (1.1 %) | 11 (1.0 %) | 6 (1.1 %) | 1.1 % |
| marche : apres (morts, replis des vitalites) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0.0 % |
| marche : residu | 1 (0.1 %) | 6 (0.4 %) | 4 (0.3 %) | 1 (0.1 %) | 1 (0.1 %) | 0 (0.0 %) | 0.2 % |
| positions : pont d identite (translocations, creations, positions, temps forts, index, horloge) | 96 (9.6 %) | 119 (9.0 %) | 135 (9.4 %) | 123 (9.9 %) | 135 (12.4 %) | 56 (11.7 %) | 10.0 % |
| &nbsp;&nbsp;dont translocations | 7 (0.6 %) | 8 (0.6 %) | 8 (0.6 %) | 7 (0.5 %) | 6 (0.6 %) | 3 (0.5 %) | 0.6 % |
| &nbsp;&nbsp;dont creations de bipede | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0.0 % |
| &nbsp;&nbsp;dont positions des bipedes | 18 (1.8 %) | 27 (2.0 %) | 27 (1.9 %) | 23 (1.9 %) | 20 (1.8 %) | 8 (1.6 %) | 1.9 % |
| &nbsp;&nbsp;dont temps forts (morts, medailles) | 12 (1.2 %) | 10 (0.8 %) | 18 (1.2 %) | 14 (1.1 %) | 8 (0.7 %) | 5 (0.9 %) | 1.0 % |
| &nbsp;&nbsp;dont index des joueurs | 60 (6.0 %) | 75 (5.7 %) | 83 (5.8 %) | 79 (6.4 %) | 101 (9.3 %) | 42 (8.7 %) | 6.6 % |
| &nbsp;&nbsp;dont origine d horloge | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0.0 % |
| positions : coups de recul | 7 (0.6 %) | 8 (0.6 %) | 9 (0.6 %) | 7 (0.6 %) | 7 (0.6 %) | 3 (0.5 %) | 0.6 % |
| positions : tirs | 6 (0.6 %) | 7 (0.5 %) | 9 (0.6 %) | 7 (0.6 %) | 6 (0.5 %) | 2 (0.4 %) | 0.6 % |
| positions : etats des images-cles (4e passe des images-cles) | 84 (8.3 %) | 106 (7.9 %) | 116 (8.1 %) | 89 (7.1 %) | 87 (8.0 %) | 37 (7.6 %) | 7.9 % |
| positions : dotations de naissance | 3 (0.3 %) | 4 (0.3 %) | 5 (0.3 %) | 4 (0.3 %) | 4 (0.3 %) | 2 (0.3 %) | 0.3 % |
| positions : reste | 1 (0.1 %) | 1 (0.1 %) | -0 (-0.0 %) | 0 (0.0 %) | -0 (-0.0 %) | 0 (0.0 %) | 0.0 % |
| balayage : monde | 56 (5.6 %) | 81 (6.1 %) | 78 (5.4 %) | 69 (5.5 %) | 74 (6.7 %) | 42 (8.8 %) | 5.9 % |
| &nbsp;&nbsp;dont zoomEvents | 7 (0.7 %) | 12 (0.9 %) | 8 (0.6 %) | 7 (0.5 %) | 6 (0.6 %) | 3 (0.6 %) | 0.6 % |
| &nbsp;&nbsp;dont placements | 38 (3.8 %) | 57 (4.3 %) | 54 (3.7 %) | 49 (3.9 %) | 56 (5.2 %) | 35 (7.2 %) | 4.2 % |
| &nbsp;&nbsp;dont spawnEvents | 6 (0.6 %) | 7 (0.5 %) | 9 (0.6 %) | 8 (0.6 %) | 6 (0.5 %) | 0 (0.0 %) | 0.6 % |
| &nbsp;&nbsp;dont pads | 5 (0.4 %) | 5 (0.4 %) | 6 (0.4 %) | 6 (0.5 %) | 5 (0.4 %) | 0 (0.0 %) | 0.4 % |
| balayage : etats de mouvement | 2 (0.1 %) | 3 (0.2 %) | 4 (0.3 %) | 3 (0.2 %) | 2 (0.2 %) | 1 (0.2 %) | 0.2 % |
| balayage : portage | 6 (0.6 %) | 8 (0.6 %) | 9 (0.6 %) | 7 (0.6 %) | 6 (0.6 %) | 3 (0.5 %) | 0.6 % |
| &nbsp;&nbsp;dont pickups | 6 (0.5 %) | 6 (0.5 %) | 7 (0.5 %) | 6 (0.5 %) | 5 (0.5 %) | 0 (0.0 %) | 0.5 % |
| balayage : capacites | 36 (3.5 %) | 20 (1.5 %) | 24 (1.7 %) | 14 (1.1 %) | 1 (0.1 %) | 0 (0.0 %) | 1.6 % |
| &nbsp;&nbsp;dont equipmentChanges | 35 (3.5 %) | 20 (1.5 %) | 24 (1.7 %) | 14 (1.1 %) | 0 (0.0 %) | 0 (0.0 %) | 1.6 % |
| balayage : pont | 109 (10.9 %) | 136 (10.2 %) | 150 (10.4 %) | 127 (10.3 %) | 118 (10.8 %) | 72 (15.0 %) | 10.5 % |
| &nbsp;&nbsp;dont grenades | 4 (0.3 %) | 5 (0.4 %) | 6 (0.4 %) | 5 (0.4 %) | 4 (0.4 %) | 0 (0.0 %) | 0.4 % |
| &nbsp;&nbsp;dont detonations | 6 (0.6 %) | 7 (0.5 %) | 8 (0.6 %) | 7 (0.6 %) | 6 (0.5 %) | 0 (0.0 %) | 0.6 % |
| &nbsp;&nbsp;dont filmTable | 97 (9.7 %) | 122 (9.2 %) | 134 (9.3 %) | 114 (9.2 %) | 106 (9.7 %) | 67 (14.0 %) | 9.4 % |
| publication : faits (versement, capture) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0.0 % |
| publication : assemblage du document | 91 (9.0 %) | 131 (9.8 %) | 134 (9.3 %) | 120 (9.7 %) | 95 (8.7 %) | 34 (7.1 %) | 9.3 % |
| &nbsp;&nbsp;dont movementStates | 0 (0.0 %) | 3 (0.2 %) | 4 (0.3 %) | 3 (0.2 %) | 0 (0.0 %) | 0 (0.0 %) | 0.1 % |

### B. Echantillons CPU par etape exclusive (30 passes, 1 kHz ; % du total des echantillons)
| etape | film02 | film03 | film10 | film11 | film13 | film05 | moyenne des 5 |
|---|---|---|---|---|---|---|---|
| echantillons (ms) | 25538 | 35167 | 36545 | 33577 | 29403 | 12556 | |
| marche | 43.0 % | 46.4 % | 46.6 % | 46.4 % | 44.3 % | 38.5 % | 45.3 % |
| positions | 17.0 % | 16.6 % | 16.6 % | 16.6 % | 19.2 % | 17.5 % | 17.2 % |
| balayage pont | 9.4 % | 8.7 % | 9.1 % | 8.8 % | 9.1 % | 12.8 % | 9.0 % |
| publication : assemblage | 7.8 % | 8.4 % | 8.3 % | 8.6 % | 7.7 % | 5.8 % | 8.2 % |
| runtime en fond (balayage, scavenger, autres) | 7.4 % | 6.7 % | 6.1 % | 6.8 % | 6.9 % | 10.5 % | 6.8 % |
| balayage monde | 5.0 % | 5.2 % | 4.8 % | 5.1 % | 6.2 % | 7.7 % | 5.2 % |
| GC en fond (marquage, autres coeurs) | 5.0 % | 4.6 % | 4.8 % | 4.5 % | 4.4 % | 4.2 % | 4.7 % |
| balayage capacites (equipement) | 3.1 % | 1.3 % | 1.5 % | 0.9 % | 0.1 % | 0.1 % | 1.4 % |
| banc : lecture disque et decompression chronometree | 0.8 % | 0.6 % | 0.8 % | 0.8 % | 0.7 % | 1.0 % | 0.7 % |
| chargement | 0.8 % | 0.6 % | 0.6 % | 0.7 % | 0.6 % | 1.0 % | 0.7 % |
| balayage portage | 0.5 % | 0.5 % | 0.6 % | 0.5 % | 0.5 % | 0.5 % | 0.5 % |
| balayage etats de mouvement | 0.2 % | 0.2 % | 0.2 % | 0.2 % | 0.2 % | 0.3 % | 0.2 % |
| banc : divers (runtime.GC explicite, liberation) | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.1 % | 0.0 % |
| BuildFromFilmAvecFaits : autre | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % |
| publication : faits | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % |

Couts transverses (cumules, recouvrent les etapes) : % des echantillons
- allocation : film02 8.6 % ; film03 9.1 % ; film10 8.7 % ; film11 9.3 % ; film13 8.6 % ; film05 7.1 %
- lecture de bits : film02 32.6 % ; film03 34.4 % ; film10 35.0 % ; film11 34.1 % ; film13 35.8 % ; film05 36.2 %
- tables de hachage : film02 7.5 % ; film03 7.4 % ; film10 7.4 % ; film11 7.2 % ; film13 6.9 % ; film05 6.3 %

### C. Detail des sous-lignes (moyenne des 5 films, % des echantillons CPU), classees
- marche / images-cles : table des datums : 13.02 %
- marche / trames : lecture des records par rangs : 11.94 %
- marche / images-cles : recherche d ancres : 10.76 %
- balayage pont / sieges et occupants : 8.07 %
- runtime en fond (balayage, scavenger, autres) /  : 6.80 %
- positions / etats des images-cles : 6.70 %
- positions / index des joueurs : 6.04 %
- marche / trames : crochets des canaux (Trame/Tete) : 5.81 %
- GC en fond (marquage, autres coeurs) /  : 4.67 %
- balayage monde / poses : 3.80 %
- publication : assemblage / passerLaPorte : 2.22 %
- marche / trames : autre (arene, rangement) : 2.02 %
- positions / positions des bipedes (ancrage compris) : 1.68 %
- publication : assemblage / poserLesPistes : 1.62 %
- publication : assemblage / poserTirsProjectilesEtGrenades : 1.33 %
- balayage capacites (equipement) / changements d equipement : 1.29 %
- marche / cloture des canaux : 1.01 %
- publication : assemblage / poserArmesAuSolEtVehicules : 0.89 %
- positions / temps forts : 0.87 %
- banc : lecture disque et decompression chronometree /  : 0.73 %
- publication : assemblage / BuildFromFilmAvecFaits : 0.69 %
- chargement /  : 0.66 %
- publication : assemblage / poserPrisesEtSocles : 0.62 %
- balayage monde / lunette : 0.53 %
- positions / coups de recul : 0.53 %
- positions / translocations : 0.53 %
- positions / tirs : 0.52 %
- balayage monde / evenements d apparition : 0.50 %
- balayage pont / detonations : 0.46 %
- balayage portage / ramassages natifs : 0.44 %
- balayage monde / socles : 0.37 %
- publication : assemblage / ouvrir : 0.35 %
- balayage pont / grenades : 0.34 %
- marche / images-cles : autre : 0.30 %
- positions / dotations de naissance : 0.28 %
- balayage etats de mouvement /  : 0.20 %
- balayage pont / projectiles : 0.17 %
- marche / preparation / apres : autre : 0.16 %
- marche / images-cles : corps des records (parcours) : 0.16 %
- publication : assemblage / poserLesTouches : 0.11 %

### D. Octets alloues par etape (profil du tas, 3 passes ; % du total alloue)
| etape | film02 | film03 | film10 | film11 | film13 | film05 |
|---|---|---|---|---|---|---|
| alloue par passe (Gio) | 5.69 | 9.46 | 9.11 | 8.23 | 6.77 | 2.62 |
| publication : assemblage | 51.7 % | 59.4 % | 55.5 % | 53.1 % | 54.7 % | 55.6 % |
| marche | 27.8 % | 24.6 % | 26.5 % | 28.8 % | 27.0 % | 22.2 % |
| positions | 6.2 % | 5.3 % | 5.7 % | 5.7 % | 5.8 % | 5.9 % |
| chargement | 5.9 % | 4.4 % | 5.0 % | 5.1 % | 5.1 % | 6.5 % |
| banc : lecture disque et decompression chronometree | 3.4 % | 2.5 % | 3.0 % | 2.9 % | 2.9 % | 4.2 % |
| balayage monde | 2.8 % | 2.1 % | 2.4 % | 2.4 % | 2.6 % | 3.0 % |
| balayage pont | 1.3 % | 1.0 % | 1.1 % | 1.1 % | 1.2 % | 1.3 % |
| balayage portage | 0.5 % | 0.4 % | 0.5 % | 0.4 % | 0.5 % | 0.5 % |
| balayage etats de mouvement | 0.4 % | 0.3 % | 0.3 % | 0.3 % | 0.3 % | 0.8 % |
| balayage capacites (equipement) | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % |
| banc : divers (runtime.GC explicite, liberation) | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % |
| BuildFromFilmAvecFaits : autre | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % |
| publication : faits | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % | 0.0 % |

Lecture des tables :
- Table A (temps mural) : les lignes de niveau 0 somment a 100 % (moyenne des cinq : 99,7 %, residu 0,3 %). La premiere ligne « marche :
  preparation » est la recherche d ancres des images-cles, necessaire a la creation d un canal avant la marche.
- Ramasse-miettes : 4,7 % des echantillons en marquage de fond (autres coeurs, hors chemin critique), 6,8 % de runtime de fond, et
  l allocation (mallocgc, growslice, memclr) 7 a 9 % des echantillons, ventilee dans les etapes. Monter GOGC n achete RIEN en temps (MESURE, 1.4)
  et coute +37 a +95 % de pic.
- Octets alloues : la publication en alloue 52 a 59 % (copies des positions par `indexBySlot`, tris), la marche 22 a 29 %, les positions 6 %.
- Publication, decoupage (part des echantillons CPU, moyenne des cinq) : assemblage 8,2 % dont `passerLaPorte` 2,2, `poserLesPistes` 1,6,
  `poserTirsProjectilesEtGrenades` 1,3, `poserArmesAuSolEtVehicules` 0,9 ; l index par slot (`indexBySlot`) pese 1,9 a 2,3 % cumule, le tri
  de float32 (`axesDesPositions`) 1,2 % ; capture des faits : 0 %.

### 1.2 La lecture principale va-t-elle au bout des trames ? (MESURE)

Compteurs de la marche (journal « etats de mouvement lus », film02 et film10) et sonde de verdict de fermeture, six films :

| film | trames delta | fermees | refusee / queue opaque | debut de liste lu par la vue A | liste localisee a tatons |
|---|---|---|---|---|---|
| film02 | 26 439 | 26 438 (99,996 %) | 0 / 1 | 13 796 | 0 |
| film03 | 33 412 | 33 411 | 1 / 0 | 5 573 | 0 |
| film05 | 12 622 | 12 621 | 1 / 0 | 1 599 | 0 |
| film10 | 36 350 | 36 349 | 1 / 0 | 6 703 | 0 |
| film11 | 33 102 | 33 101 | 1 / 0 | 5 187 | 0 |
| film13 | 28 884 | 28 883 | 1 / 0 | 4 153 | 0 |

(Les autres paquets a evenements ont leur debut en tete de paquet.) Soit **99,996 % de trames fermees** et **zero trame partie en recherche
de repli** : `paquetsEvenementsLocalises = 0`, `NonLocalises = 0`, aucun repli d ancrage bipede ni de liste dans les compteurs de couverture
(seuls tournent `repli_ancre_d_image_cle_par_election` 479 a 1 107, `repli_pistes_du_monde_apres_la_marche` 73 a 104,
`repli_origine_au_sol_lachee_par_fenetre` 198 a 228). Contraste : film04 compte 3 800 trames non localisees et 4 270 refusees, film06 13 827 refusees.

Ce que font alors les balayages d apres la marche (MESURE, profil, parts cumulees du CPU, cinq films) : `walkDeltaBipedPayloadJusqua` 0,02 %,
`matchBipedHeader` 0,00 a 0,01 %, `scanBipedAimRecordsJusqua` 0,00 %, `passeDesCreations` 0,03 %, `echantillonsDesBandes` 0,04 %,
`recupererLaListe`, `localiserLaListe`, `lectureDEssai`, `World.Snapshot` 0 % : les boucles « en-tete a chaque position de bit » ne tournent
presque plus. Les balayages **relisent ce que la marche a deja lu** (ADR 0037 : le lecteur rejoue les publications des records de la marche
sans relire un bit) : `canalDesLecturesBipedes` 2,4 a 2,8 %. Il reste des lectures sur des octets que la marche n a PAS lus : index des
joueurs (recherche de motif 64 bits dans chaque chunk de replication, 5 a 9 % du temps mural), temps forts, tables de joueurs et d equipes
(parcours des images-cles), poses d equipement (3,4 a 4,8 % du CPU), `scanEquipRecoveryPacket` (0 a 3 % : boucle d en-tete bit a bit, film02, 03, 10, 11).
La double lecture n est donc pas dans l ancrage : elle est dans les parcours des images-cles (1.3).

### 1.3 Les images-cles : 18 parcours, 4 qui comptent (MESURE)

Compteur : `distribuerLesImagesCles` est appele **18 fois** par film (414 a 558 paquets d image-cle parcourus en tout). Quatre parcours portent
29 a 33 % du CPU ; chacun fait un travail DIFFERENT sur les memes octets :

| parcours | appelant | travail | part du CPU (film02 / 03 / 10 / 11 / 13) |
|---|---|---|---|
| bande de slots bipedes | `nouveauCanalDesLecturesBipedes` | recherche d ancres (`kfRecherche`), la seule du film : memoisee, les autres parcours la reutilisent | 8,4 / 11,6 / 12,0 / 11,6 / 10,6 % |
| marche principale | `ScanMarcheDesTramesAvec` | table des datums (`EntetesDeTable`, `enteteA` : test d en-tete a chaque position de bit) | 13,3 / 13,7 / 13,8 / 12,9 / 13,5 % |
| etats des images-cles | `ScanEtatsDesImagesCles` (positions) | `motsParRecord` (fenetre glissante de 32 bits, deja pre-filtree) | 6,9 / 6,8 / 6,8 / 6,3 / 6,7 % |
| equipes | `ScanPlayerTeams` (pont) | `slotsDEntetesExacts` (`BitsBourres` et `BitsAt`) | 5,3 / - / 5,7 / - / - % |

Chaque travail est une **fonction pure du payload d image-cle** (DEDUIT : lecture de `keyframe_world_marche.go`, `keyframe_datums.go`,
`offline_biped_band.go` ; verification : memes empreintes document et faits sur les 13 films apres deplacement). Aucun travail n est refait au
sens strict (les memos `mem.parPayload` et `recup.tables` evitent de recalculer l ancre et la table) : le cout est la somme de quatre
recherches differentes de la meme famille, pas une relecture.

### 1.4 Gains possibles sur un film recent, classes gain / risque (DEDUIT : aucun code ecrit)

Gains en % du temps mural total (moyenne des cinq).

| rang | gain | ou | gain estime | risque |
|---|---|---|---|---|
| a | GC : monter GOGC | reglage de l hote | **0 %** (MESURE : film10 1 575, 1 600, 1 589 ms a GOGC 100, 200, 400 ; film02 1 084, 1 356 (bruit), 1 117 ; pic +37 a +95 %) | sans objet |
| b | rejet par mot de 64 bits dans la table des datums | `balayageDeTable.enteteA` (`EntetesDeTable`, `keyframe_datums.go`), meme famille que G4 deja livre | -4 a -7 % (13 % du CPU, dont `BitsAt` 5 % et `enteteA` 4,4 % a plat) | moyen (empreintes des 13 films) |
| c | meme rejet dans `slotsDEntetesExacts` / `BitsBourres` | `ScanPlayerTeams` | -2 a -3 % | moyen |
| d | index des joueurs : recherche de tous les motifs en une passe par chunk (fenetre 64 bits + table) au lieu d une passe par motif | `weaponv3.ResolveXuidToPI`, `source.ChercherMotifs64` (3,4 % a plat, 1,3 % ferme), un appel par chunk | -2 a -4 % (5 a 9 % du temps mural) | faible a moyen |
| e | `BitsAt` : 11 % du CPU a plat, appele par tous les balayages ; chargement de 64 bits sans verification de borne dans les boucles d en-tete | `internal/source` | -2 a -3 % | moyen |
| f | publication : `indexBySlot` construit une fois (8 appels, 52 a 59 % des octets alloues), tri de float32, tri stable des positions | `replay/shots.go`, `build.go` | -1 a -2 % de temps, -1 a -2 Gio alloues par passe ; pic -0,2 a -0,5 Gio sur le plus gros film | faible |
| g | `motsParRecord` : 3,1 % a plat restent apres le pre-filtre | `keyframe_loadout.go` | -1 a -2 % | faible |
| h | allocations : tampons reutilises (`growslice` 1,5 %) | marche, publication | -1 a -2 % | faible |

**Reponse nette : il n existe PAS de gain majeur (au moins 15 % du total sur au moins trois des cinq films) accessible sans parallelisme.**
Le meilleur candidat seul (b) vaut 4 a 7 %. Les petits gains additionnes (b a h) donneraient 12 a 20 % au prix de sept chantiers distincts de
risque faible a moyen ; chacun est largement sous la barre de 15 %. Les anciens leviers (localisation a tatons, essais de monde, ancrage) sont
consommes : 0 % sur ces films. Les 15 % et plus se trouvent avec du parallelisme (1.5, parties 2 et 3).

### 1.5 Pre-decodage des images-cles en fonction pure (DEDUIT ; l estimation la plus rentable)

Les quatre parcours de 1.3 sont des fonctions pures de chaque payload d image-cle (ancre d abord ; table des datums, mots par record et
en-tetes exacts, ensuite independants). Part du temps mural : bande 12,3 % + images-cles principales 14,5 % + etats 7,5 % + equipes environ 6 %
= **environ 40 %**. Pre-decoder tous les payloads (une vingtaine de paquets, 4 travaux chacun) sur W fils avant la marche, puis publier dans
l ordre : a un rendement de 90 / 80 / 70 % a 2 / 4 / 8 fils, ce bloc passe a 22,6 / 12,7 / 7,3 % du temps total, soit **-18 % / -28 % / -33 %** de
temps mural. Verification : prototype sur film10 et empreintes des 13 films. Points durs lus dans le code : `FilmContext.recup.tables` est une
`map` sans verrou ; `nouvelleMarcheDesImagesCles` pose puis restaure `bal.MPP` du contexte a chaque parcours (etat de contexte mute : a passer
en parametre) ; la memoire des ancres est deja verrouillee.

## Partie 2. Mesure a blanc du parallelisme par morceau

### 2.1 Ce qui a ete instrumente

Pendant la marche normale en ordre, a l entree de chaque morceau : copie de la table des slots, et **ombres** de cette table telle qu elle
etait a l entree des morceaux i-1 a i-12 (profondeurs 1 a 12 ; le morceau 0 se decode en ordre, donc une ombre d avant le morceau 1 n est pas
utilisee). Chaque ecriture reelle de la table (poser, retirer, annulation d essai, oubli de l image-cle, position) est rejouee sur chaque
ombre ; chaque LECTURE d une liaison est comparee (ombre contre etat reel) quand le slot n a pas ete ecrit dans le morceau avant d etre lu
(liaison HERITEE). Un essai ouvert ecrit dans une couche a part, jetee a l annulation. Le rang de vue des images-cles est lu comme une
liaison supplementaire. Quatre criteres de reussite, tous calcules : liaison COMPLETE egale (comme le parseur voisin), champs UTILISES par le
site de lecture, utilises sans provenance, utilises sans provenance ni position.

Sites de lecture de la table (tous passes par un point unique ; verifie par recherche de `.slots` dans le paquet) et champs utilises :

| site | champs lus | publie ? |
|---|---|---|
| `ArchetypeForEID` | archetype, generation (eid complet, `GenAny`) | decide le decodage |
| `ArchetypeForSlot` | archetype | decide le decodage |
| `GenerationMatches` | generation (`FullID`, `GenAny`) | decide le decodage |
| `VuePossede` | vue | decide le rejet de vue |
| `HardBound` | caractere souple (`Soft`) | decide l ancre de l inference de chaine |
| `contreditUneEntiteVivante` | souple, archetype | decide le refus d un NEW |
| `liaisonDu` | provenance (`Liaison`) | **publie** (`Record.Liaison`, `frame_records.go:328`) |
| `entitesDuMonde.Entite` | eid, archetype, vue, provenance, generation connue | **publie** aux crochets (`lecture.Entite`) |
| `PosOf` / `SetPos` | position et validite ; `SetPos` relit TOUS les autres champs puis les reecrit | la position sort dans les positions publiees |
| rang de vue des images-cles (scalaire) | `nsImageCle` | decide la vue des liaisons |

### 2.2 Resultat par film et par profondeur (MESURE)

Morceaux qui passeraient / morceaux juges. Les quatre criteres (strict, utile, sans provenance, sans provenance ni position) donnent des
resultats IDENTIQUES sur les 10 films.

| film | version | morceaux | lectures heritees / morceau (med.) | d=1 | d=2 | d=4 | d=12 | echecs : nature (d=4) | 1re difference |
|---|---|---|---|---|---|---|---|---|---|
| film02 | 41 | 24 | 129 | 17/22 | 16/21 | 14/19 | 8/11 | absent_de 897 | avant 1er paquet : 5/5 echecs |
| film03 | HI_1_12 | 29 | 73 | 25/27 | 24/26 | 22/24 | 14/16 | absent_de 1151 | avant 1er paquet : 2/2 echecs |
| film05 | 41 | 12 | 40 | 10/10 | 9/9 | 7/7 | 0/0 |  | avant 1er paquet : 0/0 echecs |
| film10 | 41 | 32 | 57 | 30/30 | 29/29 | 27/27 | 19/19 |  | avant 1er paquet : 0/0 echecs |
| film11 | 41 | 29 | 50 | 27/27 | 26/26 | 24/24 | 16/16 |  | avant 1er paquet : 0/0 echecs |
| film13 | 41 | 26 | 48 | 23/24 | 21/23 | 17/21 | 8/13 | absent_de 886 | avant 1er paquet : 4/4 echecs |
| film01 | 40 | 30 | 316 | 21/28 | 20/27 | 18/25 | 14/17 | absent_de 752 | avant 1er paquet : 7/7 echecs |
| film04 | 39 | 68 | 3286 | 55/66 | 52/65 | 50/63 | 40/55 | absent_de 75325, lie_dans 25, generation_differente 398, provenance_differente 242 | avant 1er paquet : 8/13 echecs |
| film06 | 31 | 31 | 3543 | 21/29 | 16/28 | 11/26 | 1/18 | absent_de 1921, archetype_different 187, generation_differente 6 | avant 1er paquet : 3/15 echecs |
| film12 | 37 | 43 | 361 | 37/41 | 35/40 | 32/38 | 24/30 | absent_de 2014, lie_dans 307 | avant 1er paquet : 3/6 echecs |

Lecture :
- **Films recents : film05, film10 et film11 passent a 100 % a toutes les profondeurs ; film03 de 25/27 a 14/16 ; film13 de 23/24 a 8/13 ; film02 de 17/22 a 8/11.**
  Anciens : film01 21/28, film04 55/66, film12 37/41 ; film06 21/29 puis 1/18 a 12.
- **Nature des echecs : quasi toujours « slot absent de la copie, lie dans l etat reel »** (100 % des lectures differentes sur film02, 03, 13). « archetype
  different » n apparait que sur film06 (187 a 408 lectures) ; « generation differente » 398 lectures sur film04 a profondeur 4, jamais seule ;
  « lie dans la copie, absent en reel » : 2 a 25 lectures (film03, film04, film12) ; provenance differente : film04 seul (242 a 4 212 lectures), toujours
  dans des morceaux qui echouent deja pour une autre raison ; position differente : 0.
- **La premiere difference tombe avant le premier paquet** (a l entree du morceau, pendant la liaison des images-cles) dans tous les echecs des films recents
  (5/5 film02, 2/2 film03, 4/4 film13), a 0,1 a 0,5 % des lectures heritees. Un re-decodage PARTIEL a partir de l ecart ne rend rien : un morceau qui echoue se
  refait entier (meme constat que pour le parseur voisin). Les morceaux touches sont ceux ou des entites nees dans un morceau precedent survivent sans etre portees
  par la chaine de records de l image-cle (cas decrit en 7 de `profil2.md`), ici seulement 0 a 25 % des morceaux, pas 90 %.
- Volume : 40 a 130 lectures heritees par morceau sur les recents (sur 63 000 a 75 000 lectures de la table par morceau, le reste portant sur des slots ecrits dans
  le morceau) ; 3 300 a 3 500 sur film04 et film06.
- Etat hors table : NEW refuses en attente de verdict a l entree d un morceau : 0 morceau sur les six recents, 15 sur film04, 2 sur film06 (ne touche que des compteurs).
  32 crochets d observation publient PENDANT le decodage (`observateur.go`) et alimentent des canaux a etat (lectures bipedes, morts, vitalites) : la sonde ne les voit pas (2.4).

### 2.3 Gain de duree murale estime (DEDUIT a partir des chiffres de 2.2 et des durees par morceau mesurees)

Modele : fenetre glissante de W morceaux en vol, chacun decode sur une copie ; validation dans l ordre ; un morceau en echec est redecode
entier apres la validation du precedent ; les crochets des canaux restent en serie dans l ordre. Entrees MESUREES : duree de decodage et de
crochets de chaque morceau, reussite par profondeur. Entrees supposees : surcout de 5 % du decodage (suivi des lectures) ; copie de la table
(300 a 500 slots, 30 a 40 Kio par morceau) negligeable. Le morceau 0 est decode en ordre.

| film | total ms | boucle (decodage + crochets) ms | 2 fils | 4 fils | 8 fils | pire cas (4 fils, 70 % du decodage parallelisable, surcout 15 %) |
|---|---|---|---|---|---|---|
| film02 | 1006 | 252 | -86 ms (-8,6 %) [18/23] | -123 ms (-12,2 %) [18/23] | -126 ms (-12,5 %) [18/23] | -88 ms (-8,8 %) |
| film03 | 1328 | 317 | -136 ms (-10,3 %) [26/28] | -199 ms (-15,0 %) [26/28] | -200 ms (-15,1 %) [26/28] | -140 ms (-10,6 %) |
| film05 | 480 | 96 | -43 ms (-9,1 %) [11/11] | -59 ms (-12,3 %) [11/11] | -62 ms (-12,9 %) [11/11] | -44 ms (-9,1 %) |
| film10 | 1439 | 349 | -161 ms (-11,2 %) [31/31] | -233 ms (-16,2 %) [31/31] | -240 ms (-16,7 %) [31/31] | -170 ms (-11,8 %) |
| film11 | 1241 | 312 | -146 ms (-11,8 %) [28/28] | -204 ms (-16,5 %) [28/28] | -211 ms (-17,0 %) [28/28] | -149 ms (-12,0 %) |
| film13 | 1093 | 252 | -107 ms (-9,8 %) [24/25] | -140 ms (-12,8 %) [22/25] | -128 ms (-11,7 %) [20/25] | -101 ms (-9,2 %) |

([ok/morceaux speculés] entre crochets.) La boucle de la marche des trames ne pese que 22 a 25 % du temps total d un film recent (decodage 16 %,
crochets 6,5 %) : c est ce plafond, plus que les echecs, qui borne le gain a **-9 a -17 % a 4 fils**, avec moins d un point de plus a 8 fils.

**Memoire au pic (DEDUIT)** : l arene d un morceau decode (`Records` et `Comps`) fait 0,9 a 1,5 Mo (mesure) et la copie de la table 0,03 a 0,04 Mo ;
a 4 fils, 4 morceaux en vol plus leurs tampons de crochets (non mesures, borne x2) : **+5 a +25 Mio au pic, plus le gachis de ramasse-miettes
des 4 fils (borne +50 Mio)**, soit un pic de 270 a 400 Mio contre 100 a 347 Mio aujourd hui (plafond de production 3 Gio : sans risque).
Verification : prototype et mesure du pic par l outil.

**Ce que la provenance impose en plus du parseur voisin** : (1) elle entre dans l ensemble des lectures validees : le champ doit etre enregistre
et compare (MESURE : n ajoute aucun echec sur les 10 films) ; (2) `SetPos` relit tous les champs non position d une liaison pour la reecrire :
c est une lecture heritee a tracer, que le parseur voisin n a pas ; (3) la sortie publie la liaison de chaque record (`Record.Liaison`) et celle lue
par les crochets (`Entite`) : pour rejouer les crochets dans l ordre apres une speculation, la table d entites vue par chaque paquet doit etre
materialisee par paquet (les crochets lisent aujourd hui le monde vivant : 7 a 10 lectures par morceau sur les recents), faute de quoi ils
verraient l etat de FIN de morceau ; (4) les 32 crochets de l observation publient pendant le decodage et nourrissent des canaux a etat : chez le
parseur voisin la projection vient apres coup, ici les crochets devraient etre mis en tampon (non estime : c est le principal risque).

### 2.4 Ce que la sonde ne couvre pas (DEDUIT, a verifier par prototype)

Etat de canaux et de memos de contexte qui traversent les morceaux (non lus par la sonde) ; memos sans verrou (`FilmContext.vies`, `recup.*`,
`slotsLus`) ; compteurs d observation. La sonde dit « les liaisons de la table se rejouent » ; elle ne dit pas « le decodage est parallelisable
sans toucher aux canaux ».

## Partie 3. Les balayages apres la marche, en parallele

### 3.1 Les balayages et leurs durees (MESURE, moyenne des cinq films, % du temps mural)

Positions 19,4 % (pont d identite 10,0 dont index des joueurs 6,6 ; etats des images-cles 7,9 ; coups de recul, tirs et naissances 1,5) ; monde 5,9
(poses 4,2) ; etats de mouvement 0,2 ; portage 0,6 ; capacites 1,6 ; pont 10,5 (tables de joueurs et d equipes 9,4). **Total en serie : 36 a 45 % du temps
mural (216 a 537 ms).**

### 3.2 Etat partage qui les empeche de tourner ensemble (MESURE, recherche dans le code a la tete)

- Un seul primitif de synchronisation dans tout `internal/grammar` (`keyframe_world_marche.go`, memoire des marches d ancres). Les memos paresseux du
  contexte de film (`chunksLus`, `slotsLus`, `layLu`, `regLu`, `profLu`, `vies`, `recup.*` dont la table des datums `map[*byte]`) sont lus et ecrits sans verrou.
- Les largeurs MPP du contexte (`bal.MPP`) sont posees puis restaurees par `PoserMPP` dans les poses (`equipment_placements.go:155,176,180`), les largeurs
  de creation (`equipment_creation_width.go:262`), les armes au sol (`build_ground_weapons.go:164`) et CHAQUE parcours d images-cles (`marche_images_cles.go:113`) :
  deux balayages ensemble s ecraseraient ces largeurs.
- L observation (`FilmContext.obs`) recoit les crochets que chaque balayage installe puis retire ; l horloge d etapes et l observateur d etapes imposent
  l ordre publie (`observe_test.go`).
- Dependances de donnees : positions vers capacites ; etats des images-cles vers naissances vers portage ; poses vers socles et vehicules vers etats de
  mouvement ; temps forts vers index des joueurs vers pont ; tables de joueurs et d equipes, lunette, evenements d apparition, grenades, detonations : sans
  dependance entre elles.

### 3.3 Gain simule (DEDUIT : durees mesurees de chaque balayage, graphe de dependances ci-dessus, rendement 90 / 80 / 70 % a 2 / 4 / 8 fils)

| film | serie (ms, % du total) | chemin critique (fils illimites) | 2 fils | 4 fils | 8 fils |
|---|---|---|---|---|---|
| film02 | 404 (40 %) | 97 ms | -157 ms (-15,6 %) | -239 ms (-23,7 %) | -265 ms (-26,4 %) |
| film03 | 491 (37 %) | 122 ms | -217 ms (-16,3 %) | -314 ms (-23,7 %) | -318 ms (-23,9 %) |
| film05 | 216 (45 %) | 67 ms | -86 ms (-17,8 %) | -132 ms (-27,6 %) | -120 ms (-25,1 %) |
| film10 | 537 (37 %) | 134 ms | -220 ms (-15,3 %) | -338 ms (-23,5 %) | -346 ms (-24,0 %) |
| film11 | 449 (36 %) | 114 ms | -195 ms (-15,7 %) | -285 ms (-23,0 %) | -287 ms (-23,1 %) |
| film13 | 439 (40 %) | 111 ms | -187 ms (-17,1 %) | -288 ms (-26,4 %) | -280 ms (-25,6 %) |

Contrairement aux films anciens (un balayage « monde » dominant : 45 % de film06), les balayages d un film recent sont **equilibres** (aucun ne depasse
10 %) : le gain monte presque lineairement jusqu a 4 fils (**-15 a -18 % a 2 fils, -23 a -28 % a 4 fils**) puis plafonne.
Memoire au pic (DEDUIT) : les balayages allouent environ 10 % des octets de la passe mais ne retiennent que des tranches de resultat deja comptees dans le pic, atteint
en publication ; attendu **+50 a +150 Mio** a 4 fils sur les recents, a mesurer par prototype. Le chemin critique (97 a 134 ms) est domine par les tables de joueurs et
d equipes (9,4 %), independantes de la marche : elles peuvent meme demarrer AVANT la marche, avec les temps forts et les index des joueurs (7,6 %).

## Recommandation d ordre (DEDUIT)

1. **Pre-decodage pur des images-cles (1.5)** : -18 a -33 % de temps mural, risque faible a moyen (fonctions pures, memos a verrouiller, `bal.MPP` a passer par parametre).
2. **Balayages en parallele (partie 3)** : -15 a -28 % ; chantier plus lourd (contexte de film a rendre sur appel : memos verrouilles, largeurs MPP par balayage,
   observateur d etapes). A faire apres 1, qui retire deja un parcours d images-cles de ce partage.
3. **Rejet par mot dans la table des datums et les en-tetes d equipes (1.4 b et c)** : -6 a -10 % sans parallelisme, utile si le gain doit etre independant du nombre de coeurs.
4. **Parallelisme par morceau (partie 2)** : en dernier. Les films recents le laissent passer (100 % sur film05, 10, 11 ; 70 a 95 % ailleurs), mais il ne vise que 22 a 25 %
   du temps total, plafonne a -9 a -17 % a 4 fils, et c est le chantier de risque le plus eleve (crochets de l observation, canaux a etat, table d entites a materialiser).
5. Rien du cote GC et allocation en temps ; l index par slot construit une fois et le plafond souple restent des sujets de memoire.

## Ce qui n a pas pu etre etabli

- Le cout reel des tampons de crochets et de canaux pour une speculation (partie 2) : la sonde ne les voit pas.
- Les gains de 1.4 b a h et du pre-decodage : estimations sur profil, aucun code ecrit.
- Le surcout de memoire des parties 2 et 3 : deduit des tailles mesurees d arenes et d allocations, pas mesure sur prototype.
- La part des versions recentes dans le parc reel (le jeu fixe en compte 5 sur 13).
- Les durees absolues du profil CPU (le profileur de ce poste en retient environ 80 %).
- Les diagnostics d erreur « vies publiees sans identite » vus dans les journaux de film05, 06 et 11 sont des constats sur les donnees, pas des echecs de mesure.

## Fichiers (sous `bench/recents/`)

`instrumentation/` (patch complet et `sonde_morceaux.go` ; pour rejouer : l appliquer sur `785c38e`, construire avec l etiquette `sondemorceaux`),
`etapes/*.jsonl` (15 passes par film, chronometres), `etapes/*.couv.*` (journal d information et couverture), `prof/*.cpu.pprof` et `*.tas.pprof` (profils),
`prof/*.classe.json` (classement par etape), `sonde/*.jsonl` (ombres) et `sonde/*.sans.jsonl` (durees), `gogc/` (essai GOGC), `tables.md`, `sonde_table.md`,
scripts d analyse `*.py`. Les profils portent un identifiant de construction contenant un chemin du poste : ils restent hors depot.

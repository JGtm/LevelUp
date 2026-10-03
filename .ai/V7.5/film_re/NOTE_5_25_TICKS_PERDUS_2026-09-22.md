# NOTE 5.25 — COMBIEN DE TICKS DE JOUEUR LE CALQUE DES ETATS PERD-IL ENCORE ?

Lot 5.25, branche `feat/decfilm-75`, base `3c7eef0b3` (integration des lots 5.1 a 5.23,
schema 68, `grammar-2026-09-22.12`). MESURE SEULE : un instrument `research`, cette note, une
section de plan, un commit. AUCUN code de production, AUCUNE grammaire, AUCUNE lecture Ghidra,
AUCUNE hypothese sur la cause, AUCUNE recommandation de lot.

Le lot 5.23 a ferme 74,7 % des rejets et laisse **5 893 (25,3 %)**. L utilisateur a fait
remarquer le 2026-09-22 que **25 % des REJETS ne veut pas dire 25 % des DONNEES**. Cette note
rend le chiffre qui compte — les ticks de bipede de JOUEUR attendus, lus et perdus — sur deux
films denses.

Instrument : `mouvement_5_25_ticks_perdus_research_test.go` + `mouvement_5_25_tableaux_research_test.go`
(`//go:build research`, paquet `grammar`, `TestTicks525`). Variables `MOUV511_FILM`,
`MOUV511_CARTE`, `MOUV511_BORNES`. Une passe, UN decodage par paquet, sous la marche de
PRODUCTION du calque (`ScanMovementStates` : cadre du film, images-cles, table de datums, table
anticipee du 5.23, trois rangs de vue). Il emprunte `t519Marcher` (5.19) et `a523IdRejete`
(5.23) ; aucune marche n est recopiee. Le crochet `Observation.EtatMouvementHook` fournit, dans
le MEME decodage, les etats lus et la vitesse tenue.

**CONTROLE DE L INSTRUMENT.** Sur `bfecd02b` il rend **3 940 / 30 387** trames a reste NUL,
**26 397** abandonnees, **50** debordements, **16 129** rejets hors datum et **254** liaisons par
anticipation — c est-a-dire, AU PAQUET, ce que `TestGate516` rend sur la MEME base (rejoue :
identique sur les cinq chiffres). Les 3 919 / 26 418 du plan 5.23 sont ceux de la branche
`feat/decfilm-73`, dont la base precede la fusion du 5.22 : l ecart de 21 paquets vient de la
BASE, pas de l instrument (cf. §4).

---

## 1. (a) LES TRAMES, ET CE QU ELLES LAISSENT SUR LA TABLE

| | `bfecd02b` (snowbound, 8 joueurs) | `4f77afc1` (flood gulch, BTB) |
|---|---:|---:|
| paquets delta | 31 232 | 35 499 |
| dont LOCALISES | **30 387** | **30 490** |
| non localises | 845 | 5 009 |
| horodatages distincts | 31 232 (un par paquet) | 35 499 (un par paquet) |
| fermees a reste NUL | **3 940 (13,0 %)** | **2 781 (9,1 %)** |
| **ABANDONNEES** | **26 397 (86,9 %)** | **27 671 (90,8 %)** |
| debordements | 50 | 38 |
| bits de payload | 57 631 104 | 197 482 824 |
| bits LUS | 37 474 807 (**65,0 %**) | 135 425 937 (**68,6 %**) |
| bits NON LUS | 20 156 297 (**35,0 %**) | 62 056 887 (**31,4 %**) |
| bits non lus des seules trames abandonnees | 17 951 824 / 51 950 064 (34,6 %) | 72 522 823 / 162 095 096 (44,7 %) |

**UNE TRAME ABANDONNEE N EST PAS UNE TRAME PERDUE : elle est lue aux deux tiers.** La part de
trame lue avant l abandon, par dixiemes (`bfecd02b`) : 0-10 % **6,3 %**, 40-50 % 6,3 %,
50-60 % 16,3 %, 60-70 % 16,5 %, 70-80 % 15,5 %, 80-90 % **18,6 %**, 90-100 % 16,3 %. La bosse
est haute (la marche va loin avant de tomber) et la queue basse est mince mais reelle
(1 670 trames coupees dans leur premier dixieme). Sur `4f77afc1` la bosse est plus basse
(50-70 % : 42,7 %) et la queue plus epaisse (0-10 % : 7,6 %).

Vitesse TENUE a la lecture (`bfecd02b`, 164 228 lectures de bipede) : le mode est a
**2,5-3,0 m/s** (57 736), la classe `0,0-0,5 m/s` ne pese que **9 190 lectures (5,6 %)**. C est
ce qui rend le seuil de mouvement (0,5 m/s) peu discriminant, donc peu manipulable : 94 % des
lectures sont au-dessus.

---

## 2. (b) OU LE REJET TOMBE DANS LA TRAME

L attendu de records d une trame abandonnee est mesure sur les trames **FERMEES** de sa fenetre
+-5 — la seule population dont on sache ce qu une trame complete porte a cet instant.

| | `bfecd02b` | `4f77afc1` |
|---|---:|---:|
| trames abandonnees SUR UN REJET | 15 474 | 10 434 |
| dont une voisine fermee a +-5 | **111** | **14** |
| sans aucune voisine fermee | 15 363 | 10 420 |
| records lus avant le rejet (ces trames) | 865 · **7,79/trame** | 330 · **23,57/trame** |
| attendu des voisines fermees | 1 018 · **9,17/trame** | 333 · **23,79/trame** |
| **manque** | **15,0 %** | **0,9 %** |

**LA COMPARAISON DE VOISINAGE NE PORTE QUE SUR 111 TRAMES SUR 15 474, ET C EST LE FAIT**, pas
une limite de l instrument : les trames fermees se concentrent au DEBUT de chaque chunk (le
chunk s effondre apres sa premiere faute — 5.19.2), si bien qu une trame abandonnee du milieu de
chunk n a aucune voisine fermee. La ou la comparaison est possible, le rejet tombe **tard** :
27,0 % des trames a 80-90 % de l attendu, 21,6 % a 90-100 %, 19,8 % AU-DESSUS de 100 %.

Le confondant que le 5.19.1 (c) avait deja nomme se relit ici : les trames abandonnees portent
PLUS de records que les fermees (`ti=35` **5,75/trame** contre 3,09 sur `bfecd02b` ;
**15,69** contre 1,94 sur `4f77afc1`) — elles sont les trames denses du milieu de match. Le
compte global de records ne dit donc rien de la perte ; seule la comparaison par VIE le dit,
c est-a-dire le tableau (c).

---

## 3. (c) LE CHIFFRE QUI COMPTE — LES TICKS DE BIPEDE DE JOUEUR

**LA METHODE, EN TROIS LIGNES.** La VIE est le record de creation de bipede
(`ScanBipedCreations`, lot E2 : slot, generation, index de participant, que `chunk_00` nomme).
Le TICK est une trame delta (~60 par seconde sur les deux films). L ATTENDU : entre DEUX
lectures consecutives d une meme vie dont au moins une porte une vitesse tenue non nulle, toutes
les trames intermediaires sont des ticks que le calque AURAIT du voir.

**ET L ATTENDU EST ETALONNE.** Sur les paires dont TOUTES les trames intermediaires sont
FERMEES, la marche ne perd rien par abandon ; le taux de trous qui y subsiste est le taux
NATUREL de non-replication d un bipede en mouvement. Il vaut **1,30 %** sur `bfecd02b`
(141 trous sur 10 821 trames de fenetre) et **0,27 %** sur `4f77afc1` (4 sur 1 458). C est ce
taux qu on retranche pour obtenir la perte NETTE de l abandon.

### `bfecd02b` — 8 joueurs, 90 vies, snowbound

| joueur | vies | ticks attendus | lus | **PERDUS** | perte |
|---|---:|---:|---:|---:|---:|
| Tataaannn | 9 | 41 337 | 21 446 | **19 891** | 48,1 % |
| Chocoboflor | 13 | 38 986 | 19 778 | **19 208** | 49,3 % |
| JGtm | 10 | 34 289 | 18 070 | **16 219** | 47,3 % |
| indahoopty8751 | 16 | 33 717 | 18 248 | **15 469** | 45,9 % |
| MEK1906 | 17 | 21 491 | 16 597 | 4 894 | 22,8 % |
| SHN Lups99 | 6 | 26 136 | 22 059 | 4 077 | 15,6 % |
| Madina97294 | 12 | 22 892 | 19 800 | 3 092 | 13,5 % |
| Draconewt | 7 | 23 863 | 20 870 | 2 993 | 12,5 % |
| **TOTAL** | **90** | **242 711** | **156 868** | **85 843** | **35,4 %** |

Perte NETTE de l abandon (moins le taux naturel de 1,30 %) : **34,1 %, soit 82 681 ticks**.

**LA PERTE N EST PAS UNIFORME : elle va de 12,5 % a 49,3 % selon le joueur, un facteur
QUATRE.** Les 90 vies rendues par la creation de bipede sont exactement les 90 pistes que le
document publie sur ce film (lot 5.23.3), et les 8 joueurs sont ceux du roster de `chunk_00`.

### `4f77afc1` — BTB, 330 vies, flood gulch

| | valeur |
|---|---:|
| vies | **330** (24 joueurs nommes + 7 index hors table publiee) |
| ticks attendus | **857 388** |
| lus | **396 763** |
| **PERDUS** | **460 625** |
| **perte** | **53,7 %** (nette **53,4 %**, 458 273 ticks) |

Extremes : `AJM002` **90,7 %**, `BLADERUNNER3141` 70,5 %, `CU3RV0187` 67,6 % … `MiniScotsMin`
23,1 %, `XN3RDXD3VILX` 27,1 %. Meme forme que sur `bfecd02b`, a un niveau plus haut : un film
BTB est plus dense, et la cascade y coute davantage.

---

## 4. (d) LA BORNE DES TRANSITIONS D ETAT QUE LE CALQUE PEUT MANQUER

| | `bfecd02b` | `4f77afc1` |
|---|---:|---:|
| couples (trame, vie) : corps en mouvement, NON LU dans une trame abandonnee, changement d etat lu pour lui a +-5 trames | **286** | **1 881** |
| trames distinctes concernees | 263 | 1 232 |
| intervalles d etat LUS et fermes (accroupi, glissade, escalade, sprint) | 495 | 1 679 |
| dont au moins un BORD dans une trame abandonnee | **473 (95,6 %)** | **1 677 (99,9 %)** |
| zones abandonnees contigues de plus de 5 trames, dans une fenetre de mouvement, sans une lecture | **397** | **2 971** |

**UN BORD DANS UNE TRAME ABANDONNEE N EST PAS UN INTERVALLE PERDU** : une trame abandonnee est
lue aux deux tiers, et le record qui porte le bord a ete lu AVANT le rejet. Ce que le chiffre de
95,6 % dit, c est que la quasi-totalite des intervalles publies vient de trames incompletes —
donc que leurs bords sont potentiellement DECALES, du quantum d une trame ou plus.

**CE QUI PEUT ETRE PERDU EN ENTIER, ce sont les 397 zones** (2 971 sur `4f77afc1`) : plus de
cinq trames consecutives, dans une fenetre de mouvement, sans une seule lecture du corps. Par
joueur sur `bfecd02b` : JGtm 82, Tataaannn 74, MEK1906 51, Chocoboflor 48, Madina97294 48,
indahoopty8751 41, SHN Lups99 40, Draconewt 13.

Les 495 intervalles fermes de l instrument se comparent aux **514** intervalles LUS que le
document publie sur ce film (sprint 501 + mobilite 12 + accroupi 1, glissade 0 — lot 5.23.3) :
l instrument ne compte que les intervalles FERMES, le document porte aussi ceux qui restent
ouverts a la fin du film. Les 327 `jumpDerived` ne sont pas comptes ici — ils sont DERIVES de la
vitesse, pas lus.

---

## 5. (e) CE QUE SONT LES ENTITES REJETEES

Apres la table anticipee, la marche de production rencontre **15 474 en-tetes rejetes** sur
`bfecd02b` (10 434 sur `4f77afc1`). Ce compte est SUPERIEUR aux 5 893 mesures avant le repli,
et le 5.23.4 dit pourquoi : la marche, allant plus loin, atteint des rejets qu elle n atteignait
pas.

| `bfecd02b` | rejets | declares par une image-cle du film | JAMAIS declares |
|---|---:|---:|---:|
| tete `0` | 131 | 0 | 131 |
| tete `1` | **13 874** | 0 | **13 874** |
| tete `2` | 1 232 | 0 | 1 232 |
| tete `3` | 237 | 0 | 237 |

**AUCUN des eid rejetes n est declare par une image-cle du film, a aucune tete** (32 sur
`4f77afc1`, soit 0,3 %) : le repli du 5.23 a bien pris tout ce qu il pouvait prendre, et ce qui
reste est, par construction, hors de sa portee.

**801 eid distincts** portent ces 15 474 rejets (871 pour 10 434 sur `4f77afc1`).

| duree entre le PREMIER et le DERNIER rejet d un meme eid | `bfecd02b` | `4f77afc1` |
|---|---:|---:|
| 0 a 100 ms | **581 (72,5 %)** | **539 (61,9 %)** |
| 100 a 500 ms | 43 (5,4 %) | 87 (10,0 %) |
| 500 ms a 1 s | 34 (4,2 %) | 89 (10,2 %) |
| 1 a 5 s | 98 (12,2 %) | 108 (12,4 %) |
| 5 a 30 s | 29 (3,6 %) | 17 (2,0 %) |
| >= 30 s | 16 (2,0 %) | 31 (3,6 %) |

| nombre de rejets par eid | `bfecd02b` | `4f77afc1` |
|---|---:|---:|
| 1 | **547 (68,3 %)** | **489 (56,1 %)** |
| 2 a 5 | 66 (8,2 %) | 89 (10,2 %) |
| 5 a 20 | 48 (6,0 %) | 162 (18,6 %) |
| 20 a 100 | 83 (10,4 %) | 119 (13,7 %) |
| 100 a 1 000 | 57 (7,1 %) | 12 (1,4 %) |

**DEUX TIERS DES EID REJETES N APPARAISSENT QU UNE FOIS ET VIVENT MOINS DE 100 ms**, et une
minorite (57 eid sur `bfecd02b`, 7,1 %) porte des centaines de rejets. La distribution est donc
bimodale : des entites ephemeres, et une poignee d entites longues.

| evenement de tete du paquet du PREMIER rejet de chaque eid | `bfecd02b` | `4f77afc1` |
|---|---:|---:|
| aucun (`-1`) | **663 (82,8 %)** | **424 (48,7 %)** |
| type `36` (`action_weapon_fire`) | 44 (5,5 %) | **143 (16,4 %)** |
| type `5` | 30 (3,7 %) | 36 (4,1 %) |
| type `0` | 7 (0,9 %) | 63 (7,2 %) |
| type `21` | 6 (0,7 %) | 47 (5,4 %) |
| type `15` | 10 (1,2 %) | 32 (3,7 %) |
| type `6` | 4 (0,5 %) | 40 (4,6 %) |
| type `1` | 9 (1,1 %) | 13 (1,5 %) |

Le paquet du premier rejet ne porte aucun evenement de tete quatre fois sur cinq sur
`bfecd02b` ; sur `4f77afc1`, film BTB, la moitie en porte un et `action_weapon_fire` est le
premier d entre eux. **Ce tableau est une observation, pas une attribution** : rien ici ne dit
que l entite rejetee EST le projectile du tir de tete.

---

## 6. LA CONCLUSION, CHIFFREE, EN UNE LIGNE

> **Le calque des etats perd 35,4 % des ticks de bipede de joueur sur `bfecd02b`
> (85 843 sur 242 711 attendus ; 34,1 % nets de l etalon) et 53,7 % sur `4f77afc1`
> (460 625 sur 857 388 ; 53,4 % nets), et au plus 286 transitions d etat sur `bfecd02b`
> (1 881 sur `4f77afc1`), avec 397 zones (2 971) assez longues pour avaler un intervalle
> entier.**

Aucune recommandation de lot : le pilote et l utilisateur decident sur le chiffre.

---

## 7. CE QUE LE LOT LAISSE OUVERT

Deux observations, au §4 du plan, non traitees : le gate du 5.23 vaut **3 940 / 26 397** sur la
base d integration (contre 3 919 / 26 418 sur la base du lot, qui precede la fusion du 5.22) ;
et la comparaison de voisinage du tableau (b) ne peut porter que sur 111 trames sur 15 474,
parce que les trames fermees se groupent au debut de chaque chunk.

# RAPPORT — lot V7 : TROUVER L'EVENEMENT DE DESTRUCTION DE VEHICULE DANS LA LISTE D'EVENEMENTS

> Execute le 2026-09-03 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`, aucune ecriture DuckDB, AUCUN fichier de production modifie
> (huit `*_test.go` neufs). Mesures en AVANT-PLAN, `CGO_ENABLED=0`, GOCACHE isole
> (`scratchpad/gocache_v7*`), donnees reelles du checkout principal
> (`C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks`, LECTURE SEULE). `apps/web/` et
> `cmd/weapon-sounds/` : NON touches. Ghidra : MORT — tout ce rapport est de la MESURE.

---

## 0. LE RESULTAT EN SIX LIGNES

1. **AUCUN DES 28 TYPES NON DECODES NE DATE LA DESTRUCTION D'UN VEHICULE.** Six angles
   d'attaque independants, tous negatifs, tous chiffres aux § 3 a 10. `VehicleTrack.End` reste
   `unknown` : le lot ne publie RIEN sur la cause de fin de vie, et c'est une mesure, pas une
   lacune.
2. **MAIS LE LOT TROUVE OU EST LE VEHICULE DANS LA LISTE, ET C'EST L'ACQUIS QUI COMPTE : LE
   DOMAINE 1 EST CELUI DES UNITES — BIPEDES *ET* VEHICULES.** Sur les references de domaine 1 a
   sonde = 1, la partition « slot dans la bande bipede » OU « slot dans la bande `ti=40` » vaut
   **100,0 %** (types 1 et 7), **99,96 %** (type 0, 18 489 / 18 497) et **99,8 %** (type 36) sur
   12 films. La base est la meme pour les deux (le minimum de la bande bipede) et l'index de
   9 bits porte au-dela. Dit autrement, et c'est la forme qui porte la preuve : parmi les
   references dont l'index sort de la bande bipede, **99,6 a 100,0 %** tombent dans la bande
   `ti=40`, la ou le hasard en mettrait **3 a 16 %**. Le lot V6 avait cherche le vehicule en
   DOMAINE 7 et l'y avait refute ; il est en domaine 1.
3. **CONSEQUENCE IMMEDIATE, MESUREE : LA REFERENCE 1 DE L'EVENEMENT DE SORTIE *EST* LE VEHICULE.**
   **105 / 105** sorties de 12 films, 100,0 % en bande `ti=40`, zero bipede, zero hors bande. Le
   chantier resout aujourd'hui le vehicule d'un episode par la GEOMETRIE (ancre de 3 m, lot V6
   § 2.3) alors que l'evenement le NOMME. C'est une decouverte HORS PERIMETRE de cette mission :
   elle est chiffree ici et NON traitee (§ 14, item 1), conformement a la regle du perimetre.
4. **LA VOIE NOMMEE PAR LE RAPPORT V3 § 6 — « l'evenement de degat de type 0 » — EST OUVERTE ET
   REFUTEE.** Le type 0 vise bien une unite : **3 099** de ses 18 684 instances (12 films)
   designent un VEHICULE. Mais leur instant se repartit sur toute la vie de la cible (position
   relative mediane **0,73**, quartiles 0,46-0,87) et seules **15,1 %** tombent dans la fenetre
   de disparition (temoin +60 s : 5,7 %). Le type 0 est un DEGAT, pas une destruction.
5. **LE TEMOIN NATUREL DU CORPUS FONCTIONNE, ET IL DESIGNE QUATRE CANDIDATS QUI NE TIENNENT PAS.**
   Sur 40 films (27 sans vehicule, 13 avec), les deux evenements vehicule connus se comportent
   exactement comme l'oracle l'annonce (type 22 sortie : **0,04** evenement par film sans
   vehicule contre **23,54** avec, r = 0,973 ; type 8 embarquement : **0,00** contre **1,08**,
   r = 0,921). Les types NEUFS a signature identique — **2, 40, 41, 118** — sont depouilles
   instance par instance au § 10. Un seul resout un vehicule, le **118** (89,5 %), et **ZERO**
   de ses instances tombe en fin de vie : ce sont des salves de 3 a 6 evenements au MILIEU de la
   vie du vehicule — un etat, pas une fin.
6. **LE BALAYAGE AVEUGLE EST UN NEGATIF NET.** 12 films, 28 types, bits 9 a 120, largeurs
   {7, 8, 9, 11, 13}, deux variantes d'adressage : le meilleur couple de chaque type coincide
   avec une fin de vie de vehicule dans **3,1 % a 13,3 %** des cas pour tous les types a effectif
   exploitable — jamais les 90 % du gate.

---

## 1. L'ORACLE, ECRIT AVANT LA MESURE

Il vit dans l'en-tete de chaque instrument, en constantes nommees. Rappel integral.

Un type d'evenement T **date la destruction d'un vehicule** si :

1. **CIBLE** — chaque instance retenue RESOUT un vehicule en bande `ti=40` (>= 90 %) ;
2. **INSTANT** — le timestamp du paquet tombe dans la fenetre de fin de vie de CE vehicule
   (>= 90 %) : `dernier recensement <= t <= premiere image-cle qui ne le recense plus` ;
3. **TEMOIN** — la meme mesure decalee de +60 s s'effondre ;
4. **ORDRE DE GRANDEUR** — quelques unites a quelques dizaines d'instances par film ; une
   destruction est rare ;
5. **CONTRASTE** — le taux sur les fins AVEC MORT A BORD (3 sur 80 episodes, V3 § 2.2) doit
   ecraser celui sur les fins par ABANDON, qui sont la majorite.

**LE TEMOIN NATUREL, ecrit avant la mesure lui aussi** : un type qui date une destruction de
VEHICULE ne peut pas exister dans un film SANS vehicule.

**LES DEUX CONTROLES POSITIFS** sont les deux types dont la reponse est connue : l'embarquement
(type 8) et la sortie (type 22). Tout instrument de ce lot les inclut ; un instrument qui ne les
verrait pas ne vaudrait rien.

---

## 2. LE CORPUS ET LES INSTRUMENTS

| corpus | films | usage |
|---|---|---|
| **V3** | 12 (`0d76e8f1`, `fccc61cd`, `4898d586`, `e1bdb97f`, `32a37698`, `e3b10d4b`, `51d3ab9f`, `d99e5dbd`, `e232ffce`, `b232e02d`, `d332c3a9`, `c6250266`) | toutes les mesures fines (Behemoth / Launch Site, Super Fiesta et non-SF) |
| **LARGE** | 40 premiers repertoires de `film_chunks` — **27 SANS vehicule** (bande `ti=40` vide ou <= 2 vies), **13 AVEC** | le temoin naturel et le depouillement des types rares |

Bandes mesurees sur le corpus V3 : bande `ti=40` de **13 a 67 slots** (minimum **768** sur les
douze films), **13 a 62 vies** recensees dont **8 a 52 a fin bornee** ; bande bipede de **89 a
104 slots**.

| instrument (tous `*_test.go`, garde `V7_ROOT`) | ce qu'il mesure |
|---|---|
| `vehicules_v7_refs_test.go` | balayage AVEUGLE : tout bit x toute largeur x deux adressages, critere « fenetre de fin » |
| `vehicules_v7_temps_test.go` | coincidence temporelle avec la FIN SERREE (dernier echantillon de position) |
| `vehicules_v7_correl_test.go` | le temoin naturel : effectif par type dans les films SANS / AVEC vehicule |
| `vehicules_v7_dom1_test.go` | la reference 0 decodee en domaine 1 (garde, sonde, largeur consequente) |
| `vehicules_v7_chaine_test.go` | les TROIS references chainees en domaine 1 |
| `vehicules_v7_cible_test.go` | depouillement instance par instance des types candidats |
| `vehicules_v7_letal_test.go` | recherche d'un DRAPEAU de letalite dans la charge |
| `vehicules_v7_delta_test.go` | ecart signe entre l'evenement et la FIN SERREE du vehicule vise |

---

## 3. MESURE A — LE BALAYAGE AVEUGLE DES REFERENCES (`TestV7Refs`) : NEGATIF

**Le principe.** Le corps d'un evenement commence toujours au bit 9 et porte trois references
gardees. Ce qui manque pour les types non decodes est le DOMAINE de chaque reference, donc sa
largeur. On la CHERCHE : pour chaque type, chaque decalage de bit de 9 a 120 et chaque largeur de
{7, 8, 9, 11, 13}, la valeur lue est testee comme un slot (brut, puis rapportee au minimum de la
bande `ti=40`).

**Premier resultat, qui a fait changer le critere.** Un premier balayage jugeait « la valeur
designe un vehicule VIVANT a cet instant ». Il ne discrimine RIEN : une vie de vehicule dure
presque tout le film, donc le critere degenere en « la valeur est petite ». Le type 36 (le tir)
le passait a **99,0 %** — avec un temoin temporel a **98,9 %**, c'est-a-dire zero information.
Le critere retenu est donc la FENETRE DE DISPARITION, propre a chaque vie.

**Mesure, 12 films.** Meilleur couple (bit, largeur, adressage) de chaque type, par exces sur le
temoin de chance analytique :

| type | corpus | echant. | bit | larg. | **FIN** | temoin +60 s | temoin `ti=42` | chance |
|---|---|---|---|---|---|---|---|---|
| 0 | 18 684 | 4 800 | 115 | 7 | **11,2 %** | 5,0 % | 9,0 % | 0,59 % |
| 1 | 1 611 | 1 611 | 12 | 8 | 6,5 % | 11,7 % | 0,4 % | 0,40 % |
| 5 | 5 460 | 4 207 | 20 | 8 | 5,8 % | 3,3 % | 0,4 % | 0,47 % |
| 6 | 1 385 | 1 385 | 31 | 9 | 3,8 % | 2,0 % | 1,1 % | 0,26 % |
| 7 | 2 339 | 2 339 | 12 | 8 | 4,6 % | 2,8 % | 0,3 % | 0,54 % |
| **8** (board) | 15 | 15 | 64 | 13 | 13,3 % | 0,0 % | 0,0 % | 0,01 % |
| 9 | 1 289 | 1 289 | 75 | 8 | 3,1 % | 1,9 % | 0,6 % | 0,34 % |
| 11 | 199 | 199 | 68 | 7 | 5,0 % | 4,0 % | 0,5 % | 0,78 % |
| 15 | 2 943 | 2 643 | 9 | 11 | 4,8 % | 4,4 % | 3,0 % | 0,05 % |
| 21 | 3 824 | 3 805 | 10 | 8 | 4,8 % | 3,1 % | 0,9 % | 0,44 % |
| **22** (exit) | 105 | 105 | 25 | 8 | 12,4 % | 20,0 % | 1,0 % | 0,43 % |
| 36 | 18 036 | 4 800 | 11 | 8 | 7,8 % | 5,2 % | 1,1 % | 0,39 % |
| 38 | 3 755 | 3 741 | 10 | 7 | 4,9 % | 3,1 % | 0,5 % | 0,94 % |
| 39 | 673 | 673 | 10 | 7 | 4,9 % | 3,6 % | 0,1 % | 0,90 % |
| 41 | 7 | 7 | 37 | 11 | 14,3 % | 0,0 % | 0,0 % | 0,02 % |
| 47 | 23 | 23 | 100 | 8 | 8,7 % | 0,0 % | 4,3 % | 0,32 % |
| 58 | 9 | 9 | 11 | 7 | 11,1 % | 0,0 % | 0,0 % | 0,35 % |
| 75 | 1 721 | 1 721 | 11 | 8 | 4,1 % | 3,1 % | 1,0 % | 0,40 % |
| 76 | 1 168 | 1 168 | 105 | 13 | 3,4 % | 3,0 % | 1,2 % | 0,01 % |
| 82 | 7 131 | 4 800 | 24 | 13 | 5,3 % | 3,8 % | 1,8 % | 0,01 % |
| 103 | 353 | 353 | 59 | 7 | 4,8 % | 1,7 % | 0,0 % | 1,05 % |
| 117 | 8 | 8 | 26 | 9 | 37,5 % | 0,0 % | 0,0 % | 0,15 % |
| 118 | 1 | 1 | 52 | 7 | 100,0 % | 0,0 % | 0,0 % | 0,78 % |

Types a 3 instances ou moins (2, 3, 23, 40) : aucun couple ne rend le moindre coup.

**LECTURE.** Le maximum reel sur un effectif exploitable est **13,3 %** (type 8, n = 15). Les deux
valeurs elevees (117 a 37,5 % sur n = 8, 118 a 100 % sur n = 1) sont le MEILLEUR DE 1 120 CELLULES
tirees sur moins de dix instances : c'est ce que le hasard rend quand on le laisse choisir. Le
gate demande 90 % ; personne ne s'en approche. **La reference du vehicule, si elle existe, n'est
pas a un decalage de bit constant lu comme un slot brut.** La suite du lot montre pourquoi : elle
y est, mais il fallait DECODER la reference (garde et sonde) au lieu de lire des bits fixes.

---

## 4. MESURE B — LA COINCIDENCE TEMPORELLE (`TestV7Temps`) : NEGATIF

**Le principe.** Aucune lecture de charge : on ne se sert que de l'INSTANT. La FIN SERREE d'une
vie de vehicule est son dernier echantillon de position (~0,5 s, contre ~20 s pour le recensement),
obtenue par `ScanFilmBipedPositionsForBand` en `QuantaOnly` — aucune coordonnee monde, donc aucun
catalogue de bornes.

Corpus V3 : **195 vies a fin serree** sur 12 films (de 8 a 25 par film). Deux sens de mesure, le
temoin etant la meme fin serree decalee de +60 s (et -60 s).

| type | n evts | A: <= 1 s | A: <= 3 s | A: temoin +60 | A: temoin -60 | B: <= 1 s | B: <= 3 s | B: temoin +60 |
|---|---|---|---|---|---|---|---|---|
| 0 | 18 684 | 46,2 % | 75,4 % | 35,9 % | 42,1 % | 3,8 % | 12,6 % | 5,3 % |
| 1 | 1 611 | 28,7 % | 64,6 % | 21,5 % | 25,6 % | 5,3 % | 15,2 % | 4,8 % |
| 5 | 5 460 | 45,6 % | 73,3 % | 34,9 % | 43,1 % | 6,1 % | 16,2 % | 5,9 % |
| 6 | 1 385 | 31,8 % | 54,9 % | 24,6 % | 28,2 % | 6,1 % | 15,4 % | 7,1 % |
| 7 | 2 339 | 25,6 % | 48,7 % | 17,4 % | 22,6 % | 6,2 % | 16,7 % | 4,4 % |
| **8** | 15 | 0,0 % | 0,0 % | 0,0 % | 0,8 % | 0,0 % | 0,0 % | 6,7 % |
| 9 | 1 289 | 24,6 % | 52,8 % | 10,3 % | 19,0 % | 5,4 % | 15,4 % | 3,9 % |
| 11 | 199 | 5,1 % | 13,8 % | 5,6 % | 4,1 % | 6,5 % | 19,1 % | 5,0 % |
| 15 | 2 943 | 50,8 % | 86,2 % | 35,9 % | 36,9 % | 4,8 % | 14,5 % | 4,4 % |
| 21 | 3 824 | 43,1 % | 71,3 % | 44,6 % | 49,2 % | 3,7 % | 11,6 % | 4,8 % |
| **22** | 105 | 3,1 % | 12,3 % | 2,6 % | 4,1 % | 5,7 % | 19,0 % | 7,6 % |
| 36 | 18 036 | 57,9 % | 85,6 % | 47,7 % | 56,9 % | 4,7 % | 13,5 % | 4,6 % |
| 38 | 3 755 | 54,4 % | 81,5 % | 41,5 % | 47,7 % | 5,6 % | 14,0 % | 5,0 % |
| 39 | 673 | 15,4 % | 36,9 % | 10,8 % | 12,8 % | 5,1 % | 12,5 % | 5,2 % |
| 47 | 23 | 0,0 % | 11,4 % | 0,0 % | 0,0 % | 0,0 % | 17,4 % | 0,0 % |
| 58 | 9 | 1,0 % | 1,0 % | 0,0 % | 0,0 % | 11,1 % | 11,1 % | 0,0 % |
| 75 | 1 721 | 31,8 % | 65,1 % | 30,3 % | 37,4 % | 4,6 % | 12,9 % | 5,4 % |
| 76 | 1 168 | 36,9 % | 62,6 % | 22,1 % | 31,8 % | 6,6 % | 15,7 % | 5,6 % |
| 82 | 7 131 | 76,4 % | 94,4 % | 57,9 % | 68,2 % | 5,6 % | 16,0 % | 4,7 % |
| 103 | 353 | 10,5 % | 26,9 % | 12,3 % | 11,7 % | 5,4 % | 14,2 % | 6,2 % |
| 117 | 8 | 4,5 % | 4,5 % | 0,0 % | 0,0 % | 12,5 % | 12,5 % | 0,0 % |

**LECTURE, ET C'EST LE SENS B QUI TRANCHE.** Une destruction est PURE : chacune de ses occurrences
tue un vehicule, donc la part de ses evenements a moins d'une seconde d'une fin serree doit
approcher 100 %. **Le maximum mesure est 12,5 %, sur huit instances.** Tous les types frequents
sont entre 3,7 % et 6,6 % — c'est-a-dire au niveau de leur propre temoin (3,9 a 7,6 %). Le sens A,
lui, ne fait que refleter la DENSITE : le type 82 « couvre » 76,4 % des fins parce qu'il emet
7 131 fois, et son temoin en couvre deja 57,9 %.

---

## 5. MESURE C — LE TEMOIN NATUREL DU CORPUS (`TestV7Correlation`) : LES CANDIDATS, ET LEUR CHUTE

**Le principe.** Chercher le signal ENTRE les films, sans rien supposer de la grammaire de charge :
seul le type de tete est lu, et son cadrage est prouve bit-exact (garde-fou
`fire_events == head type36`).

40 films : **27 SANS vehicule** (<= 2 vies `ti=40`), **13 AVEC**.

| type | total | films | moy/film | **moy SANS veh** | **moy AVEC veh** | r(fins bornees) |
|---|---|---|---|---|---|---|
| **22** (SORTIE, controle) | 307 | 11 | 7,7 | **0,04** | **23,54** | **0,973** |
| **8** (EMBARQUEMENT, controle) | 14 | 7 | 0,3 | **0,00** | **1,08** | **0,921** |
| **41** | 30 | 4 | 0,8 | **0,00** | **2,31** | 0,829 |
| **2** | 21 | 5 | 0,5 | **0,00** | **1,62** | 0,728 |
| **118** | 19 | 4 | 0,5 | **0,00** | **1,46** | 0,477 |
| **40** | 2 | 2 | 0,1 | **0,00** | **0,15** | 0,507 |
| 76 | 6 197 | 39 | 154,9 | 111,15 | 245,85 | 0,927 |
| 21 | 12 860 | 40 | 321,5 | 212,59 | 547,69 | 0,895 |
| 38 | 12 300 | 40 | 307,5 | 233,85 | 460,46 | 0,845 |
| 82 | 26 166 | 39 | 654,1 | 559,63 | 850,46 | 0,836 |
| 1 | 3 517 | 40 | 87,9 | 56,96 | 152,23 | 0,812 |
| 11 | 400 | 38 | 10,0 | 6,37 | 17,54 | 0,787 |
| 75 | 5 676 | 39 | 141,9 | 121,04 | 185,23 | 0,789 |
| 36 | 73 063 | 40 | 1 826,6 | 1 556,30 | 2 387,92 | 0,733 |
| 6 | 3 376 | 34 | 84,4 | 52,04 | 151,62 | 0,729 |
| 9 | 7 463 | 40 | 186,6 | 165,63 | 230,08 | 0,651 |
| 0 | 26 826 | 40 | 670,6 | 487,26 | 1 051,54 | 0,545 |
| 7 | 4 655 | 37 | 116,4 | 94,52 | 161,77 | 0,478 |
| 5 | 12 149 | 38 | 303,7 | 268,81 | 376,23 | 0,253 |
| 15 | 35 322 | 40 | 883,0 | **1 027,44** | 583,15 | 0,017 |
| 109 | 117 | 1 | 2,9 | 4,33 | **0,00** | -0,067 |
| 117 | 6 | 3 | 0,1 | 0,19 | 0,08 | -0,027 |

Types presents dans UN SEUL film (73, 74, 80, 93, 98, 101, 104, 108, 113) : leur r vaut
mecaniquement 0,076 et ne dit rien — ils sont propres a un mode, pas aux vehicules.

**LECTURE.** L'instrument FONCTIONNE : les deux controles positifs sortent en tete de la
signature cherchee (zero ou presque sans vehicule, r > 0,92), et deux types la contredisent
franchement (le 15 est ANTI-correle, le 109 n'existe que dans un film sans vehicule). Les
candidats neufs sont donc **41, 2, 118** et, symboliquement, **40** — tous a **0,00 dans les
27 films sans vehicule**.

Leur depouillement instance par instance est au § 10.

---

## 6. MESURE D — L'ACQUIS DU LOT : LE DOMAINE 1 EST CELUI DES UNITES (`TestV7Dom1`)

**L'hypothese, ecrite avant la mesure.** Le lecteur d'identifiant du moteur (`FUN_1406d3140`) ne
lit une SONDE que pour le domaine 1 : sonde = 1 rend un index de 9 bits, sonde = 0 un index de
13 bits. Le chantier lit le cas sonde = 1 comme « index relatif a la bande bipede ». Or dans la
taxonomie Halo, le BIPEDE et le VEHICULE sont deux specialisations d'UNITE. Si le domaine 1 est le
domaine des unites, un evenement designe un vehicule par une reference de domaine 1, avec la MEME
base — l'index de 9 bits (0..511) porte de la bande bipede (base + 0..102) jusqu'a la bande
`ti=40` (base + 256..302, soit les slots 768..814 quand la base vaut 512).

**Mesure, 12 films, reference 0 decodee (garde, sonde, largeur consequente).**

| type | n | ref0 absente | sonde = 1 | dont BIPEDE | dont **VEHICULE** | dont HORS des deux bandes |
|---|---|---|---|---|---|---|
| 0 | 18 684 | 0,0 % | 99,0 % (18 497) | 83,2 % (15 390) | **3 099** | **8** (0,04 %) |
| 1 | 1 611 | 0,0 % | 73,7 % (1 187) | 63,5 % (754) | **434** | **0** |
| 7 | 2 339 | 0,0 % | 75,7 % (1 771) | 73,6 % (1 304) | **467** | **0** |
| 36 (tir) | 18 036 | 0,0 % | 100,0 % | 97,8 % (17 639) | **369** | 28 (0,2 %) |
| 75 | 1 721 | 0,0 % | 100,0 % | 99,9 % | 0 | 2 |
| **22** (sortie) | 105 | 0,0 % | 100,0 % | **100,0 %** | 0 | 0 |
| 58 | 9 | 0,0 % | 100,0 % | 77,8 % | 2 | 0 |

**LA PARTITION EST TOTALE.** Sur cinq types de familles differentes et 12 films, une reference 0
de domaine 1 a sonde = 1 tombe dans la bande bipede OU dans la bande vehicule, et NULLE PART
AILLEURS.

**LE CHIFFRE QUI PORTE LA CONCLUSION EST CELUI DES INDEX HAUTS.** Un index de domaine 1 vaut
0..511. Ceux qui tombent dans la bande bipede (0..102) ne prouvent rien de neuf. La question est :
QUE FONT LES AUTRES ? Reponse, par type et sur 12 films :

| type | references a index HORS bande bipede | dont dans la bande VEHICULE | attendu par hasard |
|---|---|---|---|
| 0 | 3 111 | **3 099 = 99,6 %** | 3 a 16 % |
| 1 | 434 | **434 = 100,0 %** | " |
| 7 | 467 | **467 = 100,0 %** | " |
| 36 | 402 | **369 = 91,8 %** | " |

L'attendu par hasard est le rapport `taille de la bande vehicule / (512 - taille de la bande
bipede)`, soit **3,1 % a 16,4 %** selon le film (bandes mesurees : 13 a 67 slots vehicule, 89 a
104 bipedes). **Observer 91,8 a 100,0 % ferme la question.**

**LE TEMOIN DE CADRAGE A +1 BIT EST DEGENERE, ET C'EST DIT PLUTOT QUE PASSE SOUS SILENCE.** Il rend
99,0 a 100,0 % pour les memes types, ce qui ressemble a un echec de temoin. Il n'en est pas un : il
est INAPPLICABLE ici, par construction. Relue au bit 10, la garde tombe sur la SONDE (= 1) et la
sonde tombe sur le bit de poids fort de l'index vrai ; le temoin ne retient donc QUE les references
dont l'index depasse 255 — c'est-a-dire exactement les references VEHICULE — et il lit ensuite
`(index - 256) x 2`, qui retombe mecaniquement dans la bande bipede. Ce temoin ne mesure rien, et
la table ci-dessus le remplace.

Les types 5 et 6, eux, ne se lisent PAS en domaine 1 (respectivement 1 845 et 458 references hors
des deux bandes) : leur reference 0 est d'un autre domaine, et ce fichier ne pretend rien d'elle.
Les types 15, 76 et 82 ont une reference 0 **absente a 100 %** — fait structurel gratuit, publie
tel quel.

**CE QUE CET ACQUIS FAIT A LA QUESTION DU LOT.** Il rend la question DECIDABLE : on sait desormais
lire « quel vehicule cet evenement vise ». La reponse reste negative, mais elle n'est plus une
absence d'outil.

| type | instances visant un VEHICULE | dans la fenetre de disparition | temoin +60 s | position relative dans la vie (min / q1 / **med** / q3 / max) |
|---|---|---|---|---|
| 0 | 3 099 | **15,1 %** | 5,7 % | 0,01 / 0,46 / **0,73** / 0,87 / 1,00 |
| 1 | 434 | 21,4 % | **38,0 %** | 0,34 / 0,80 / **0,81** / 0,82 / 1,00 |
| 5 | 245 | 4,1 % | 2,9 % | 0,08 / 0,32 / **0,59** / 0,78 / 0,99 |
| 6 | 66 | 0,0 % | 3,0 % | 0,24 / 0,35 / **0,66** / 0,80 / 0,91 |
| 7 | 467 | **21,4 %** | 3,6 % | 0,04 / 0,52 / **0,73** / 0,86 / 1,00 |
| 36 | 369 | 9,8 % | 11,7 % | 0,09 / 0,43 / **0,57** / 0,78 / 1,00 |

Le type 36 est ici un CONTROLE INTERNE precieux : c'est le tir/degat, dont on sait qu'il frappe un
vehicule a n'importe quel moment de sa vie. Sa mediane de position relative vaut 0,57 et son taux
de fenetre de fin (9,8 %) est SOUS son propre temoin (11,7 %). Les types 0 et 7 lui ressemblent
(mediane 0,73, taux 15,1 % et 21,4 %) : ce sont des degats, pas des destructions. Une destruction
aurait une mediane a 1,00 et un taux a 100 %.

---

## 7. MESURE E — LES TROIS REFERENCES CHAINEES (`TestV7Chaine3`) : LA SORTIE NOMME SON VEHICULE

La reference 0 d'un evenement de degat est l'ATTAQUANT (97,8 % bipede pour le type 36) : ce n'est
pas la qu'une CIBLE se lit. Le lot V6 a mesure que la sortie occupe `9 + [13 + 13 + 1] + 6`, soit
DEUX references de domaine 1 ; la seconde n'avait jamais ete regardee.

**Mesure, 12 films, les trois references chainees en domaine 1 :**

| type | ref | n | absente | sonde=1 BIPEDE | **VEHICULE** | HORS | FIN \| VEH | temoin +60 |
|---|---|---|---|---|---|---|---|---|
| **22** (sortie) | **1** | 105 | **0,0 %** | **0,0 %** | **105 (100,0 %)** | **0** | 12,4 % | 20,0 % |
| 22 | 0 | 105 | 0,0 % | 100,0 % | 0 | 0 | — | — |
| 22 | 2 | 105 | 100,0 % | — | — | — | — | — |
| 0 | 1 | 18 684 | 56,9 % | 100,0 % | 0 | 2 | — | — |
| 5 | 1 | 5 460 | 87,3 % | 0,0 % | **628** | 42 | 2,4 % | 2,9 % |
| 7 | 1 | 2 339 | 38,3 % | 0,0 % | 0 | 0 | — | — |
| 9 | 2 | 1 289 | 66,5 % | 0,0 % | 32 | 150 | 0,0 % | 3,1 % |
| 36 | 1 | 18 036 | 100,0 % | — | — | — | — | — |

**LA REFERENCE 1 DE LA SORTIE EST LE VEHICULE : 105 SUR 105, ZERO BIPEDE, ZERO HORS BANDE.** Le
troisieme emplacement de reference est absent a 100 % — ce qui recoupe exactement la decomposition
V6 `13 + 13 + 1`.

**CE N'EST PAS UNE DESTRUCTION** (12,4 % en fenetre de fin, temoin a 20,0 % : aucun signal), et
c'est HORS PERIMETRE de cette mission. C'est neanmoins l'information la plus utile que le lot
produise pour la production : voir § 14.

Le type 5 porte lui aussi un vehicule en reference 1 (628 instances, 90,6 % des references
presentes), sans aucune concentration en fin de vie (2,4 % contre un temoin a 2,9 %).

---

## 8. MESURE F — LE DRAPEAU DE LETALITE DANS LA CHARGE (`TestV7Letal`) : NEGATIF

**Le principe.** Si la destruction est DANS le type 0 (la voie nommee par le rapport V3 § 6), elle
y est distinguee par un DRAPEAU — le precedent est dans le depot, `fire_events.go` lisant cinq
drapeaux aux bits 108..112 du meme record. On cherche donc un bit `b` tel que, sur les seules
instances qui VISENT un vehicule, `P(fenetre de fin | b = 1) >= 90 %` alors que `P(... | b = 0)`
reste au plancher. 247 bits sont balayes par type ; la colonne `n(b=1)` est publiee parce que le
MEILLEUR DE 247 TIRAGES doit etre juge comme tel.

| type | instances visant un vehicule | deja en fenetre de fin | temoin +60 s | bit | n(b=1) | **FIN si b=1** | FIN si b=0 | temoin +60 si b=1 |
|---|---|---|---|---|---|---|---|---|
| 0 | 3 099 | 15,1 % | 5,7 % | 192 | 101 | **44,6 %** | 14,1 % | 4,0 % |
| 0 | " | " | " | 45 | 1 034 | 28,3 % | 8,5 % | 2,9 % |
| 0 | " | " | " | 15 | 1 020 | 27,2 % | 9,2 % | 4,0 % |
| 1 | 434 | 21,4 % | 38,0 % | 61 | **11** | 81,8 % | 19,9 % | 0,0 % |
| 1 | " | " | " | 63 | 19 | 73,7 % | 19,0 % | 0,0 % |
| 5 | 245 | 4,1 % | 2,9 % | 248 | 87 | 8,0 % | 1,9 % | 4,6 % |
| 7 | 467 | 21,4 % | 3,6 % | 148 | **19** | 52,6 % | 20,1 % | 10,5 % |
| 7 | " | " | " | 132 | 163 | 35,0 % | 14,1 % | 3,7 % |
| **36** (controle) | 369 | 9,8 % | 11,7 % | 60 | **8** | 37,5 % | 9,1 % | 0,0 % |

**LECTURE.** Aucun bit ne separe. Les seules valeurs elevees vivent sur 8 a 19 instances,
c'est-a-dire exactement ce que le meilleur de 247 tirages rend par hasard — et LE CONTROLE LE
PROUVE : le type 36, dont on sait qu'il n'est PAS une destruction, produit lui aussi un « meilleur
bit » a 37,5 %. Sur des effectifs de plusieurs centaines, le maximum reel est **44,6 %** (type 0,
bit 192, n = 101), tres loin des 90 % du gate — et ce bit ne selectionne que 101 des
3 099 instances (3,3 %) : meme s'il signifiait quelque chose, il ne daterait la fin d'aucune vie
de facon exhaustive.

---

## 9. MESURE G — L'ECART A LA FIN SERREE (`TestV7Delta`) : LE NEGATIF LE PLUS NET DU LOT

**Le principe, et pourquoi c'est cette mesure qui tranche.** Elle combine les deux acquis : le
domaine 1 dit QUEL vehicule l'evenement vise, et le flux de position dit QUAND ce vehicule a cesse
d'emettre, a la demi-seconde. On mesure donc l'ecart SIGNE `t_evenement - fin serree`, en secondes.
Un evenement de destruction vaut zero.

Corpus V3, 195 vies a fin serree, types dont la reference 0 se lit proprement en domaine 1 :

| type | n apparies | vies visees | **<= 1 s** | <= 3 s | temoin +60 s | min | q1 | **MEDIANE** | q3 | max |
|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 2 929 | 41 | **0,7 %** | 2,4 % | 0,4 % | -457,4 | -61,4 | **-21,0** | -10,8 | +192,0 |
| 1 | 414 | 37 | 2,9 % | 7,7 % | 1,2 % | -158,2 | -62,2 | **-15,2** | -9,5 | +3,4 |
| 5 | 33 | 22 | 3,0 % | 9,1 % | 0,0 % | -169,2 | -87,0 | **-39,1** | +0,4 | +506,5 |
| 6 | 7 | 6 | 0,0 % | 0,0 % | 0,0 % | -317,4 | -267,2 | **-81,1** | -55,0 | -36,4 |
| 7 | 414 | 39 | 2,7 % | 8,0 % | 0,0 % | -285,3 | -43,6 | **-16,5** | -8,9 | +231,4 |
| **36** (controle) | 249 | 14 | 2,0 % | 4,4 % | 0,8 % | -382,5 | -51,9 | **-14,5** | +3,7 | +53,1 |

**LECTURE.** Tous les types qui visent un vehicule le visent **PENDANT** sa vie, pas a sa fin : la
mediane est de **-14,5 a -81,1 secondes**, et la part a moins d'une seconde plafonne a **3,0 %**.
Le type 36, le tir, sert de reference de forme : il est a -14,5 s de mediane et 2,0 % a une
seconde. Les types 0, 1 et 7 lui sont indiscernables. **Il n'existe, dans les types decodables,
aucun evenement qui tombe sur la fin d'un vehicule.**

Ce resultat recoupe et explique le lot V3 : le flux de position d'un vehicule survit 13 a 36 s
(mediane par lot) a son abandon. Un evenement de degat emis pendant qu'on le conduit tombe donc
NECESSAIREMENT bien avant sa fin serree — c'est ce que -15 a -21 s de mediane dit, chiffre.

---

## 10. MESURE H — LE DEPOUILLEMENT DES CANDIDATS (`TestV7Cible`) : NEGATIF

Les quatre candidats du temoin naturel (2, 40, 41, 118), plus les types rares du registre V6,
depouilles INSTANCE PAR INSTANCE sur les 40 films, reference 0 decodee en domaine 1 :

| type | n | films | ref0 absente | **VEHICULE** | BIPEDE | hors des deux bandes | FIN \| VEH | temoin +60 |
|---|---|---|---|---|---|---|---|---|
| **2** | 21 | 5 | 0,0 % | **0,0 %** | 0,0 % | 100,0 % | — | — |
| **40** | 2 | 2 | 0,0 % | **0,0 %** | 0,0 % | 100,0 % | — | — |
| **41** | 30 | 4 | 0,0 % | **3,3 %** | 20,0 % | 76,7 % | 0,0 % | 0,0 % |
| **118** | 19 | 4 | 0,0 % | **89,5 %** | 10,5 % | 0,0 % | **0,0 %** | 5,9 % |
| 3 | 19 | 13 | 15,8 % | 0,0 % | 5,3 % | 78,9 % | — | — |
| 23 | 333 | 10 | 0,0 % | 0,0 % | 0,3 % | 99,7 % | — | — |
| 47 | 181 | 20 | 0,0 % | 1,1 % | 6,6 % | 92,3 % | 0,0 % | 0,0 % |
| 100 | 48 | 10 | 0,0 % | 0,0 % | **100,0 %** | 0,0 % | — | — |
| 105 | 16 | 8 | 0,0 % | 0,0 % | 6,2 % | 93,8 % | — | — |
| 106 | 8 | 7 | 12,5 % | 0,0 % | 0,0 % | 87,5 % | — | — |
| 117 | 6 | 3 | 0,0 % | 0,0 % | 0,0 % | 100,0 % | — | — |
| **8** (controle) | 14 | 7 | 0,0 % | 0,0 % | 28,6 % | 71,4 % | — | — |
| **22** (controle) | 307 | 11 | 0,0 % | 0,0 % | **100,0 %** | 0,0 % | — | — |

**LE SEUL CANDIDAT QUI RESOUT UN VEHICULE EST LE TYPE 118 — ET IL NE DATE PAS SA DESTRUCTION.**
17 de ses 19 instances designent un vehicule (89,5 %, le gate 1 a un cheveu), mais **ZERO** tombe
dans la fenetre de disparition. Le depouillement dit ce que c'est : des SALVES de 3 a 6 evenements
espaces de 0,5 a 4 s sur le MEME slot, au MILIEU de la vie du vehicule (position relative 0,25 a
0,79). C'est un evenement d'ETAT repete, pas une fin. Il est note ici avec ses chiffres pour le
prochain lot ; ce lot-ci ne le nomme pas.

Les types 2, 40, 41 ne se lisent pas en domaine 1 (leurs references resolvent des slots 3 024 a
7 633, tres au-dela des deux bandes) : leur domaine est autre, et ce rapport ne pretend rien
d'eux — sinon qu'ils sont ABSENTS des 27 films sans vehicule, ce qui reste le fait a exploiter.

Le controle 22 (sortie) sort a 100 % BIPEDE en reference 0, comme il doit ; le controle 8
(embarquement) sort a 71,4 % « hors », ce qui est ATTENDU et sain : ses references sont en
domaines 2/3/7 (lecture Ghidra du 2026-09-02), pas en domaine 1, donc la lecture de cet
instrument est fausse pour lui — et le dire est une verification de plus que l'instrument ne
fabrique pas de resultats.

---

## 11. VERDICT DES GATES, ECRITS AVANT MESURE

| gate | enonce | mesure | verdict |
|---|---|---|---|
| **1** | l'evenement retenu RESOUT un vehicule en bande (>= 90 %) | UN SEUL type l'approche : le **118**, a **89,5 %** (17 / 19). Tous les autres sont sous 20,0 % (type 0 : 3 099 / 18 684 = 16,6 % ; type 7 : 467 / 2 339 = 20,0 % ; type 41 : 3,3 % ; types 2 et 40 : 0,0 %) | **ECHOUE (sauf le 118, qui tombe au gate 2)** |
| **2** | son timestamp tombe dans la fenetre de fin de la vie correspondante (>= 90 %) | type 118 : **0,0 %** ; maximum tous types **21,4 %** (types 1 et 7) ; le controle « degat » (type 36) rend 9,8 %. Ecart median a la FIN SERREE : **-14,5 a -81,1 s** selon le type, part a moins d'une seconde <= **3,0 %** | **ECHOUE** |
| **3** | le temoin par decalage temporel s'effondre | il ne s'effondre PAS pour le type 1 (38,0 % contre 21,4 % : le temoin DEPASSE le reel) ; il s'effondre pour le type 7 (3,6 %) mais sur un taux reel de 21,4 %, qui ne passe pas le gate 2 | **SANS OBJET (gate 2 ferme)** |
| **4** | les 3 morts-a-bord attestees de V3 ont leur evenement | **NON EVALUE** : le gate 2 echoue de 69 points, aucun candidat n'atteint le stade ou cette verification aurait un sens. Dit tel quel, non maquille | **[!] non traite, justifie** |
| **5** | le taux sur les fins AVEC MORT A BORD ecrase celui sur les fins par ABANDON | **NON EVALUE**, meme raison | **[!] non traite, justifie** |
| **6** | *controle positif* — l'instrument voit-il les evenements vehicule CONNUS ? | OUI, par trois canaux independants : le temoin naturel (type 22 a 0,04 / 23,54, r = 0,973 ; type 8 a 0,00 / 1,08, r = 0,921), la reference 0 (sortie : 105/105 bipede) et la reference 1 (sortie : 105/105 VEHICULE) | **PASSE** |
| **7** | *controle negatif* — le temoin naturel distingue-t-il ? | OUI : le type 15 est ANTI-correle (1 027 par film SANS vehicule contre 583 AVEC), le type 109 n'existe que dans un film sans vehicule | **PASSE** |

**VERDICT GLOBAL : la destruction d'un vehicule N'EST PAS DATABLE par la liste d'evenements dans
l'etat actuel du decodage.** Les deux gates de contenu echouent, et les deux controles passent —
c'est-a-dire que l'echec n'est pas un echec d'instrument.

---

## 12. CE QUI EST LIVRE, ET CE QUI NE L'EST PAS

| item | statut | justification |
|---|---|---|
| Decodeur additif de la destruction dans `event_list.go` | **[!] NON ECRIT** | le gate ne passe pas. Publier un `end = "destroyed"` sur un signal a 21 % serait exactement le « dead code museum » (anti-patron n° 1) que le lot V3 avait deja refuse d'ecrire pour la meme raison. **La consigne le dit : pas d'invention.** |
| `VehicleTrack.End` / `tEnd` | **[~] INCHANGE** | `End` reste `VehicleEndUnknown`. `SchemaVersion` reste **30**, aucune regeneration OpenAPI, aucun type web, aucun golden touche |
| Reconstruction des artefacts de demonstration | **[~] SANS OBJET** | aucun fichier de production modifie : la cuisson rendrait octet pour octet le meme artefact. Une reconstruction « pour la forme » consommerait 20 minutes pour prouver l'identite de deux fichiers que rien n'a touches |
| Huit instruments de mesure | **[x] NEUFS** | tous `*_test.go`, tous sous garde `V7_ROOT`, tous SKIP sans environnement (verifie : 8 SKIP, 0 FAIL) |

Aucun fichier de production n'a ete modifie par ce lot. `internal/analysis/filmdec/` ne recoit que
des fichiers `_test.go` : le garde-fou de cadrage `fire_events == head type36` est donc intact par
construction, et les comptes corpus de sorties et d'embarquements sont inchanges par construction.

| fichier neuf | lignes | ce qu'il porte |
|---|---|---|
| `internal/analysis/filmdec/vehicules_v7_refs_test.go` | 369 | `TestV7Refs` — le balayage aveugle et son temoin de chance analytique |
| `internal/analysis/filmdec/vehicules_v7_temps_test.go` | 323 | `TestV7Temps` — fins serrees et coincidence temporelle (deux sens, temoins +/-60 s) |
| `internal/analysis/filmdec/vehicules_v7_correl_test.go` | 200 | `TestV7Correlation` — le temoin naturel du corpus, 40 films |
| `internal/analysis/filmdec/vehicules_v7_dom1_test.go` | 233 | `TestV7Dom1` — la reference 0 en domaine 1, et la partition bipede/vehicule |
| `internal/analysis/filmdec/vehicules_v7_chaine_test.go` | 170 | `TestV7Chaine3` — les trois references chainees |
| `internal/analysis/filmdec/vehicules_v7_cible_test.go` | 226 | `TestV7Cible` — depouillement instance par instance des types candidats |
| `internal/analysis/filmdec/vehicules_v7_letal_test.go` | 227 | `TestV7Letal` — recherche d'un drapeau de letalite |
| `internal/analysis/filmdec/vehicules_v7_delta_test.go` | 178 | `TestV7Delta` — l'ecart signe a la fin serree |

Reutilisation stricte, aucune grammaire recopiee : `PacketHeadEventType`, `readDom1Ref`,
`readPlainRef`, `eventPayloadStartBit`, `dom7RefWidth` (`event_list.go`) ;
`ScanFilmWorldObjectKeyframes`, `bipedSlotBand`, `ScanFilmBipedPositionsForBand`, `WalkPackets`,
`ReadFilmChunk`, `CountFilmChunks`, `readBitsAt`, `LockProcessDecode`, `v0VehiculeTI`,
`GroundWeaponTypeIndex`. La regle de fenetre de vie est celle de `replay.vehicleLives` /
`assignVehicleWindows`, reprise a l'identique (tolerance de 20 s, frontiere partagee entre deux
vies d'un meme slot).

---

## 13. LES GATES D'EXECUTION

| gate | commande | resultat |
|---|---|---|
| gofmt | `gofmt -l internal/analysis/{filmdec,replay}/` | **sortie VIDE** |
| vet | `CGO_ENABLED=0 go vet ./internal/analysis/filmdec/ ./internal/analysis/replay/` | **exit 0** |
| tests SANS environnement | `CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1` | `ok filmdec 144,1 s` · `ok replay 171,5 s` — **`grep -c '^--- FAIL:'` = 0** |
| SKIP propre des instruments | `go test ./internal/analysis/filmdec/ -run TestV7 -v` | **8 SKIP, 0 FAIL** |
| OpenAPI / types web / golden | contrat INCHANGE (aucun fichier de production touche) | **sans objet** |
| seuils de fichier | le plus gros instrument : 369 L | **PASSE** |
| perimetre | `apps/web/`, `cmd/weapon-sounds/`, `internal/himap/` : non touches | **PASSE** |

Commandes de rejeu (avant-plan, GOCACHE isole, `CGO_ENABLED=0`) :

```
export V7_ROOT=<data>/cache/film_chunks
export V7_FILMS="0d76e8f1,fccc61cd,4898d586,e1bdb97f,32a37698,e3b10d4b,51d3ab9f,d99e5dbd,e232ffce,b232e02d,d332c3a9,c6250266"
go test ./internal/analysis/filmdec/ -run TestV7Dom1   -v -timeout 180m   # la partition domaine 1
go test ./internal/analysis/filmdec/ -run TestV7Chaine3 -v -timeout 180m  # les trois references
go test ./internal/analysis/filmdec/ -run TestV7Delta  -v -timeout 180m   # l ecart a la fin serree
go test ./internal/analysis/filmdec/ -run TestV7Letal  -v -timeout 180m   # le drapeau de letalite
go test ./internal/analysis/filmdec/ -run TestV7Refs   -v -timeout 180m   # le balayage aveugle
go test ./internal/analysis/filmdec/ -run TestV7Temps  -v -timeout 180m   # la coincidence temporelle

# le temoin naturel et le depouillement : SANS V7_FILMS, sur les N premiers films du cache
unset V7_FILMS
V7_MAX=40 go test ./internal/analysis/filmdec/ -run TestV7Correlation -v -timeout 180m
V7_MAX=40 V7_DETAIL=1 go test ./internal/analysis/filmdec/ -run TestV7Cible -v -timeout 180m
```

---

## 14. CE QUI RESTE OUVERT (note, NON traite — regle du perimetre)

1. **LA REFERENCE 1 DE LA SORTIE EST LE VEHICULE, ET LA PRODUCTION NE S'EN SERT PAS.** Mesure :
   **105 / 105** sur 12 films, 100,0 % en bande `ti=40`, zero bipede, zero hors bande (§ 7). Le
   calque resout aujourd'hui le vehicule d'un episode d'occupation par la GEOMETRIE — ancre de
   3 m, 48 rattachements sur 49 episodes, 1 ambigu (V6 § 2.3) — alors que l'evenement le NOMME a
   la milliseconde et sans ambiguite possible. Ce que cela changerait, si un lot le prenait :
   `VehicleEvent` gagne un champ `VehicleSlot` (lecture additive dans `decodeExitRefs`, qui lit
   deja `r1` et le jette), `vehicleRideFromEpisode` cesse d'appeler `vehicleAnchorAt`, le champ
   `ambiguous` peut redescendre, et les episodes dont l'ancre ne trouvait aucun vehicule frais
   (1 sur 49) deviennent rattachables. **C'est un changement de comportement du calque : il
   demande ses propres gates (avant / apres sur les deux artefacts) et n'a rien a faire dans ce
   lot.** Ce qui reste a verifier avant : que la reference 1 designe bien LA MEME vie que
   la geometrie sur les 48 episodes ou les deux repondent.
2. **LE DOMAINE 1 = UNITES rouvre la lecture des types 0, 1, 7, 36, 75.** Le type 0 (18 684
   instances, 12 films) a une reference 0 d'unite a 99,0 % ; le type 7 (2 339) a 75,7 % ; le
   type 75 (1 721) est PUREMENT bipede (99,9 %). Nommer ces trois types est desormais un
   probleme d'oracle, plus un probleme de grammaire — et les fils des morts / du kill-feed sont
   la verite terrain disponible pour le faire.
3. **LE TYPE 118 est un evenement d'etat de vehicule** : 89,5 % de ses instances designent un
   vehicule, par salves de 3 a 6 au milieu de sa vie, dans 4 films sur 13 a vehicules (§ 10).
   Non nomme, non date.
4. **LES TYPES 2, 40, 41 sont vehicule-dependants sans etre lisibles** : strictement absents des
   27 films sans vehicule, mais leurs references resolvent des slots de 3 024 a 7 633, donc un
   autre domaine que le 1. Sans la table de domaines par type (Ghidra), leur charge reste close.
5. **LA DESTRUCTION N'EST PEUT-ETRE PAS DANS LA LISTE.** Le lot V6 a montre que la liste TERMINE
   apres un evenement vehicule (99/100) et que rien ne s'y cache ; ce lot montre qu'aucun type de
   TETE ne la date. Il reste deux possibilites que ni V6 ni V7 n'excluent : que la destruction
   soit un evenement de tete d'un type ABSENT du corpus de 40 films, ou qu'elle ne soit pas un
   evenement du tout mais un etat de la trame ECS — auquel cas la voie est la grammaire d'`i2`/
   `i3` pour `ti=40`, condition de reprise deja posee au lot V2b et toujours ouverte.
6. **Le decodeur ne lit que l'evenement de TETE.** V6 a refute l'existence d'un second evenement
   apres un evenement vehicule (99/100 de fins de liste, temoins a 5 % et 0 %), mais n'a rien
   mesure pour les AUTRES types. Un evenement de destruction qui serait toujours le SECOND d'une
   liste de deux resterait invisible a tous les instruments de ce lot. La marche sequentielle
   reste fermee faute de longueur de charge (V6 § 1.5).

---

## 15. STATUT DES ITEMS

| item | statut | justification |
|---|---|---|
| 1. ORACLE ecrit AVANT mesure | `[x]` | en-tete de chaque instrument, en constantes nommees, temoins definis avant. Trois oracles ecrits avant leur propre mesure (cible, instant, temoin naturel) |
| 2a. Longueur constante par type pour les types inconnus (methode V6) | `[!]` **NON TENTE, et remplace** | V6 § 1.5 a REFUTE l'ancrage aval comme oracle de longueur (« fin propre » 0,0 % au vrai debut ; argmax agrege a 47 au lieu de 43). Rejouer la methode aurait consomme le lot sans rien rendre. Trois substituts ont ete construits a la place, et ils repondent a la question SANS longueur : le balayage aveugle (§ 3), la partition de domaine (§ 6) et le chainage des references (§ 7) |
| 2b. Decoder les refs avec le dictionnaire du chantier et correler | `[x]` | §§ 6, 7, 10 — et c'est la qu'est l'acquis : le domaine 1 est celui des UNITES |
| 2c. Distinguer les grenades / bidons par la cible (`ti=40` vs autre) | `[x]` | par construction : toute mesure classe la cible en BIPEDE / VEHICULE / hors, et le temoin de bande `ti=42` (armes au sol) est publie au § 3 |
| 3a. Le type 0 nommement | `[x]` | § 6 et § 9 : 3 099 instances visant un vehicule, position relative mediane 0,73, ecart median a la fin serree **-21,0 s**. C'est un DEGAT. La voie du rapport V3 § 6 est explicitement fermee |
| 3b. Les types qui n'apparaissent QUE dans les films a vehicules | `[x]` | § 5 (temoin naturel, 40 films) et § 10 (depouillement) : 2, 40, 41, 118 identifies, aucun ne date une destruction |
| 4. GATES si trouve | `[!]` **SANS OBJET** | § 11 : les deux gates de contenu echouent, les deux controles passent |
| 4bis. Decodeur additif + `end="destroyed"` + `tEnd` + bump + openapi + golden + artefacts | `[!]` **NON FAIT, conformement a la consigne** | « SI RIEN NE PASSE : `end` reste `unknown` — pas d'invention. » Aucun fichier de production modifie, `SchemaVersion` reste 30 |
| 5. Rapport + thought_log | `[x]` | ce fichier ; entree en tete de `.ai/thought_log.md` |
| Gates d'execution (gofmt, vet, tests sans env, <= 500 L) | `[x]` | § 13 |
| Perimetre (`apps/web/`, `cmd/weapon-sounds/`) | `[x]` | non touches ; aucun commit, aucun `git add` |

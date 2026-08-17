# Lots E + F — phase 0 : le deuxieme axe de la visee, et quatre sondes du registre

> Executeur, worktree `../LevelUp-wt-visee-sondes` (branche `wt/visee-sondes`, base `829c64c0e`).
> Perimetre : **E.0.1** (elevation de visee) et **F2/F3/F4/F5** (sondes). E.0.2 et F1 sont `[~]`
> (portes par le lot P). Mesures et sondes SEULEMENT : aucune publication, aucun champ de
> document, aucune ecriture DuckDB. Le seul code de production ajoute est l'accesseur
> `BipedPosition.AimPitchDeg()` — E.0.1 le demande explicitement.
>
> Regle machine D17 tenue : **un film par processus**, avant-plan, plafond memoire surveille
> (`Start-Process -PassThru`, `PeakWorkingSet64`), cout mesure sur le premier film avant les
> suivants. Aucun processus n'a approche le plafond de 3 Go (maximum observe : 170 Mo).

## 0. Instruments

| fichier | ce qu'il porte | garde |
|---|---|---|
| `apps/go-api/internal/analysis/replay/visee_elevation_test.go` | E.0.1 : distribution de `PitchRaw`, oracle de signe, oracle angulaire, ajustement du quantum, controle de l'accesseur | `AIM_FILM` + `AIM_MAP` + `AIM_BOUNDS` |
| `apps/go-api/internal/analysis/filmdec/offline_aim.go` | l'accesseur `AimPitchDeg()` (seul code de production du lot) | — |

Sorties : `lotEF/<short8>_E01_pitch_hist.tsv` (histogramme complet, valeur par valeur) et
`lotEF/<short8>_E01_oracle.tsv` (la population de l'oracle, kill par kill).

---

## 1. E.0.1 — l'elevation de visee (i21, R(11))

### 1.1 Ce que la mesure devait etablir

Le champ est LU et STOCKE depuis longtemps (`componentDirs.PitchRaw`, `offline_aim.go:58,282`)
et n'a jamais eu ni accesseur ni convention ecrite. Trois inconnues : ou est « a plat », quel
signe regarde vers le haut, et quel est le quantum. Aucune ne pouvait etre supposee.

### 1.2 Distribution brute (3 films, 3 modes, 3 cartes)

| film | mode / carte | positions | portent i21 | bornes | mode | p50 | sous 1024 | sur 1024 |
|---|---|---|---|---|---|---|---|---|
| `000d5950` | Slayer / Cliffhanger | 171 826 | 90 341 (52,6 %) | [588, 1306] | 1006 | 997 | 77,4 % | 21,4 % |
| `530820e5` | CTF / Catalyst | 148 907 | 77 083 (51,8 %) | [537, 1401] | **1024** | 1004 | 67,2 % | 31,1 % |
| `7344d24f` | Strongholds / Vagabond | 188 979 | 113 062 (59,8 %) | [590, 1490] | 1013 | 1003 | 72,8 % | 25,8 % |

Trois faits, tous attendus d'une convention lineaire centree :

1. **Le mode tombe sur le centre theorique 1024, ou juste dessous** (1024 / 1013 / 1006). Le
   leger biais vers le bas est physique et non un defaut de decodage : on vise des CORPS depuis
   une hauteur d'oeil, donc quelques degres sous l'horizontale.
2. **Le support est BORNE et loin des extremites du champ.** Reunies, les trois bornes donnent
   [537, 1490] — strictement a l'interieur de la moitie centrale [512, 1536].
3. La dispersion est etroite (p5..p95 = 900..1070 sur le temoin Slayer) : un joueur vise a plat
   la plupart du temps, ce qu'on attend d'un tireur.

Le catalogue de bornes est **CONTROLE avant chaque mesure** contre le decoupage lu dans le film
(`DetectI0Layout`) : `[13 13 14]` Cliffhanger, `[15 15 15]` Catalyst, `[15 15 17]` Vagabond,
concordants tous les trois. Sans ce controle, un `dz` en metres serait un `dz` dans une unite
inconnue.

### 1.3 Oracle du kill — la piece qui ne partage rien avec le champ

Au moment du kill, le reticule du tueur est SUR sa victime. La geometrie entre les deux bipedes
au meme instant donne donc l'angle vise, mesure sans toucher au champ. Les couples viennent du
chunk highlight (`analysis.ParseHighlightEvents`) : un event `kill` et un event `death` au MEME
instant, identites distinctes. **Aucun couple n'est reconstruit ni recolle** — les instants
ambigus sont comptes et ecartes. Le pont slot -> xuid est celui de PRODUCTION (`buildOwners`).

| film | instants de kill | couples | ambigus ecartes | pont slot->xuid | vies | ecart tueur/victime (median / max) |
|---|---|---|---|---|---|---|
| `000d5950` | 93 | 76 | 17 | 93 slots | 105 | 0 ms / 33 ms |
| `530820e5` | 92 | 85 | 7 | 97 slots | 98 | 0 ms / 267 ms |
| `7344d24f` | 117 | 96 | 21 | 117 slots | 124 | 0 ms / 0 ms |

Attrition (couples -> population d'oracle) : 7/9/1 sans echantillon de tueur, 2/1/0 sans
echantillon de victime, 0/1/0 hors fenetre de 150 ms, puis les deux seuils propres a chaque
oracle. **Zero collision d'index de joueur sur les trois films.**

#### (a) Oracle de SIGNE — l'enonce du plan, et sa limite

Seuil ecrit avant la mesure : accord >= 80 % sur les kills a |dz| >= 1 m.

| film | accord | temoin (elevations permutees) | plancher du predicteur constant |
|---|---|---|---|
| `000d5950` | **15 / 15 = 100,0 %** | 86,7 % | 93,3 % (dz > 0 dans 1 cas sur 15) |
| `530820e5` | **22 / 24 = 91,7 %** | 58,3 % | 66,7 % (8 sur 24) |
| `7344d24f` | **19 / 19 = 100,0 %** | 47,4 % | 63,2 % (7 sur 19) |
| **total** | **56 / 58 = 96,6 %** | — | — |

**Le gate est tenu sur les trois films, mais il faut dire ce qu'il ne prouve pas.** Sur le temoin
Slayer, 14 des 15 kills retenus ont la victime SOUS le tueur : un predicteur constant y aurait
obtenu 93,3 %, et le temoin permute y monte a 86,7 %. Le seul enonce du plan aurait donc valide
la convention sur une population quasi constante — c'est exactement le defaut que le lot C avait
rencontre (« critere non discriminant »). Le plancher du predicteur constant est publie ici pour
cette raison, et les deux films a objectif (temoin permute 58,3 % et 47,4 %, plancher 66,7 % et
63,2 %) sont ceux qui portent reellement le resultat.

#### (b) Oracle ANGULAIRE — celui qui porte la convention

Le signe ne dit rien de l'ECHELLE, alors que la convention demandee est une formule en degres.
On regresse donc l'angle geometrique `atan2(dz, dxy)` sur le nombre de pas de quantum
(`PitchRaw − 1024`), sans aucun seuil sur dz.

| film | kills | correlation r | pente brute (deg/pas) |
|---|---|---|---|
| `000d5950` | 51 | **0,930** | 0,1599 |
| `530820e5` | 46 | **0,916** | 0,1499 |
| `7344d24f` | 67 | **0,969** | 0,1683 |
| **total** | **164** | | |

La pente brute est BIAISEE et on sait pourquoi : `dz` separe deux ORIGINES de bipede, alors que
le tir part de l'OEIL du tueur et arrive sur le CORPS de la victime. Il manque une hauteur
constante h, que la geometrie transforme en une erreur d'angle inversement proportionnelle a la
distance — c'est elle qui faisait varier le rapport angle/pas d'un facteur trois selon la tranche
d'amplitude (0,047 / 0,089 / 0,159 deg/pas sur `000d5950`). Le modele l'absorbe :

    dz = dxy · tan(c · pas) − h

| film | c ajuste (deg/pas) | h (m) | R2 | candidat 180/2048 = 0,0879 | candidat 360/2048 = 0,1758 |
|---|---|---|---|---|---|
| `000d5950` | 0,1706 | **0,054** | 0,896 | SSE ×3,34 | **SSE ×1,01** |
| `530820e5` | 0,1385 | **0,296** | 0,718 | SSE ×1,38 | SSE ×1,26 |
| `7344d24f` | 0,1685 | **0,108** | 0,922 | SSE ×4,06 | **SSE ×1,03** |

**h tombe entre 5 et 30 cm sur les trois films** : le modele decrit bien la geometrie du tir (une
valeur aberrante aurait invalide c du meme coup). Second estimateur, independant de l'ajustement
— rapport median sur les kills a longue portee (dxy >= 8 m, |pas| >= 20) : **0,1678** / 0,1728 /
0,1495 deg/pas.

**VERDICT DE CONVENTION.** Le quantum du CAP (180/2048 = 0,0879 deg/pas), qui etait l'hypothese
naturelle puisque cap et elevation ont la meme resolution apparente, est **REFUTE** : il coute
3,3 et 4,1 fois la meilleure somme des carres sur les deux films ou l'ajustement est net. Le
quantum retenu est **360/2048 = 0,17578125 deg/pas**, soit DEUX FOIS celui du cap, a 1,01 et
1,03 fois l'optimum. Formule publiee :

    AimPitchDeg = 360 × (PitchRaw + 0,5) / 2048 − 180        (positif = vers le HAUT)

`530820e5` est le film le moins net (R2 0,72, les deux candidats a 1,26 et 1,38) : Catalyst est
une carte a etages, les kills y sont courts et le biais de hauteur y pese plus. Il ne contredit
pas le resultat, il le porte moins.

#### (c) Controle de bout en bout de l'accesseur

L'instrument appelle l'accesseur de production et compare son resultat a la geometrie, sur les
kills a longue portee (dxy >= 8 m, ou le biais de hauteur ne masque plus rien) :

| film | kills | ecart median | p90 |
|---|---|---|---|
| `000d5950` | 16 | **0,82 deg** | 7,03 deg |
| `530820e5` | 10 | **0,66 deg** | 9,48 deg |
| `7344d24f` | 7 | **0,67 deg** | 2,45 deg |

Un ecart median SOUS LE DEGRE entre un champ de 11 bits decode et une geometrie mesuree
independamment : la formule est la bonne.

### 1.4 Reserve honnete sur la plage

Les valeurs observees tiennent dans [537, 1490], donc dans la moitie centrale du champ
([512, 1536] = ±90 deg avec le quantum retenu). Cette mesure **ne peut pas distinguer** « le champ
couvre ±180 deg et le jeu borne le tangage a ±90 deg » de « le champ ne code que ±90 deg sur la
moitie de ses valeurs » : les deux rendent EXACTEMENT les memes degres sur tout ce que le film
transmet. La formule publiee est celle a ±180 deg sur tout le champ ; le jour ou une valeur
sortirait de [512, 1536], c'est elle qui la rendrait, et il faudra la reverifier. La reserve est
ecrite dans le commentaire de l'accesseur.

### 1.5 Cout machine (D17)

| film | positions | `ScanFilmBipedPositions` | `ScanFilmPlayerIndices` | processus complet | pic memoire |
|---|---|---|---|---|---|
| `000d5950` | 171 826 | 7,9 - 10,1 s | 1 min 24 | 97 - 160 s | 130 - 170 Mo |
| `530820e5` | 148 907 | 11,3 - 12,2 s | 1 min 13 | 85 - 88 s | 115 - 119 Mo |
| `7344d24f` | 188 979 | 13,6 - 14,5 s | 1 min 27 | 101 - 103 s | 135 - 138 Mo |

**Le balayage des bipedes n'est PAS le poste dominant** — contrairement a ce que l'ordre de
mission supposait. Il coute 8 a 15 s ; c'est `ScanFilmPlayerIndices` (le second maillon du pont,
qui balaie tous les chunks a la recherche du motif xuid) qui pese 1 min 15 a 1 min 30, soit
80 % du temps. Decouverte notee au §4, non traitee (hors perimetre).

Plafond de 3 Go : jamais approche (maximum 170 Mo, soit 6 % du plafond).

### 1.6 Statut des items du lot E

- `[x]` **E.0.1** — convention etablie et publiee, distribution sur 3 films, oracle tenu
  (56/58 = 96,6 % en signe ; r = 0,916 a 0,969 en angle sur 164 kills), accesseur
  `AimPitchDeg()` pose avec sa convention mesuree et sa reserve.
- `[~]` E.0.2 — porte par le lot P (P.0.4), hors perimetre de cet executeur.
- E.1 / E.2 — phases de publication et de rendu, hors perimetre de la phase 0.

---

## 4. Decouvertes (hors perimetre — notees, NON traitees)

1. **`ScanFilmPlayerIndices` coute 1 min 15 a 1 min 30 par film**, soit 80 % du temps de
   l'instrument, pour un balayage de motif qui ne rend que huit couples (xuid, index). Tout
   instrument qui a besoin du pont slot -> xuid paie ce prix. A chiffrer et memoiser avec le
   re-parse du registre `chunk_00` deja note au §6 du plan.
2. **Le fil des morts perd 7 a 21 instants de kill par film** en instants « ambigus » (deux morts
   ou deux kills a la meme milliseconde). `killsource` les recolle par voisinage immediat et
   mesure que ce recollage echoue MOINS que les couples directs ; l'instrument E.0.1 ne l'a pas
   fait (il n'en avait pas besoin). Un lot qui voudrait la population complete des couples doit
   reprendre cette regle plutot qu'en inventer une.

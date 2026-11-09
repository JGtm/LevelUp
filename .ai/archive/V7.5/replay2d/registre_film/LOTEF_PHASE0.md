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
| `apps/go-api/internal/analysis/replay/visee_elevation_test.go` | E.0.1 : l'instrument et la distribution brute de `PitchRaw` | `AIM_FILM` + `AIM_MAP` + `AIM_BOUNDS` |
| `apps/go-api/internal/analysis/replay/visee_elevation_oracle_test.go` | E.0.1 : l'oracle du kill — couples du fil, geometrie, signe, angle, controle de l'accesseur | (meme test) |
| `apps/go-api/internal/analysis/replay/visee_elevation_ajustement_test.go` | E.0.1 : la statistique pure — ajustement du quantum, regression (elle ne connait ni le film ni les kills) | (meme test) |
| `apps/go-api/internal/analysis/filmdec/offline_aim.go` | l'accesseur `AimPitchDeg()` (seul code de production du lot) | — |
| `apps/go-api/internal/analysis/filmdec/sonde_registre_scan_test.go` | F2-F5 : le moteur — resolution des archetypes PAR NOM, bandes et fantomes, UNE passe delta, marche des composants sous `SetProbeHook` | `PROBE_FILM` |
| `apps/go-api/internal/analysis/filmdec/sonde_registre_verdicts_test.go` | F2-F5 : les quatre verdicts, densites, treillis, periodicites | `PROBE_CACHE` + `PROBE_SHORT` + `PROBE_OBJTYPE` (facultatifs) |
| `apps/go-api/internal/analysis/filmdec/sonde_registre_outils_test.go` | F2-F5 : l'outillage partage (comptage de valeurs, densite, TSV) | (meme test) |

Sorties : `lotEF/<short8>_E01_pitch_hist.tsv` (histogramme complet, valeur par valeur),
`lotEF/<short8>_E01_oracle.tsv` (la population de l'oracle, kill par kill) et
`lotEF/<short8>_F2_{0,1}_valeurs.tsv` (toutes les valeurs des deux composants de ti=47).

**Les quatre sondes tiennent en UNE passe par film.** Elles interrogent des archetypes
differents mais lisent les memes paquets : quatre instruments auraient coute quatre lectures
completes du film. C'est la regle D17 appliquee a la conception de l'instrument, pas seulement
a son execution.

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

## 2. F2 — ti=47 `splash-message` : quelles VALEURS, et quand

### 2.1 Ce que la sonde regarde

Le lot C avait deja mesure la DENSITE des ANNONCES AU MASQUE de ti=47 autour des captures et
conclu que ses deux composants portes ne sont pas les messages « zone capturee ». F2 pose la
question un cran plus bas : **quelles valeurs le hook rend-il**, combien sont distinctes, et
lesquelles apparaissent (ou n'apparaissent JAMAIS) pres d'un evenement d'objectif.

Controle de purete de la bande ti=47 : 5,0 % de records hors grammaire sur `7344d24f`, 8,3 %
sur `696a9d7c`, 22,1 % et 15,9 % sur les CTF, 18,5 % en KOTH, **85,2 % sur le temoin Slayer** —
les chiffres du Slayer sont a lire avec cette reserve, exactement comme le lot C l'avait ecrit.

### 2.2 Volumes et valeurs

| film | mode | i0 static : emissions / valeurs distinctes | i1 dynamic : emissions / valeurs distinctes |
|---|---|---|---|
| `7344d24f` | Strongholds | 85 / 62 | 2 492 / 314 |
| `696a9d7c` | Strongholds | 86 / 48 | 2 485 / 321 |
| `530820e5` | CTF | 76 / 50 | 5 574 / 399 |
| `64e8adfa` | CTF | 111 / 82 | 12 912 / 431 |
| `0a247154` | KOTH | 77 / 70 | 342 / 55 |
| `000d5950` | Slayer (temoin) | 42 / 40 | 455 / 77 |

Les comptes du STATIC recoupent ceux du lot C a l'unite pres (84-85 sur `7344d24f`, 86 sur
`696a9d7c`, 76 sur `530820e5`, 111 contre 119 sur `64e8adfa`) alors que les deux instruments ne
partagent aucun code de comptage : l'ancrage est le meme, et la mesure est reproductible.

### 2.3 Le R(24) dynamique n'est pas une enumeration : c'est un TREILLIS

En triant les valeurs distinctes du composant dynamique et en regardant leurs ecarts, une
structure saute aux yeux — **l'ecart dominant vaut exactement 4 584**, sur tous les films et
tous les modes :

| film | ecarts multiples de 4 584 | les trois ecarts les plus frequents |
|---|---|---|
| `7344d24f` | 223 / 313 = 71,2 % | 4 584 (223 fois) · 4 583 (46) · 1 (31) |
| `530820e5` | 282 / 398 = 70,9 % | 4 584 (282) · 4 581 (31) · 3 (31) |
| `64e8adfa` | 248 / 430 = 57,7 % | 4 584 (248) · 4 583 (46) · 4 581 (31) |

Le pas median des transitions croissantes vaut 4 584 sur CINQ films sur six. Les ecarts
residuels sont soit le meme pas a une ou trois unites pres, soit des ecarts MINUSCULES (1 a 3)
qui forment de petites grappes autour de chaque niveau.

**Une enumeration d'identifiants de message n'a aucune raison de se poser sur un treillis
arithmetique de pas 4 584** — qui n'est meme pas une puissance de deux, donc ce n'est pas non
plus un champ de bits decale. La lecture qui colle a la mesure est : le R(24) porte un
SCALAIRE QUANTIFIE, un gros champ de pas 4 584 accompagne d'un petit champ de 0 a 3 environ. Le
controle independant le confirme : la correlation entre la valeur et l'instant du paquet est
nulle (r = -0,02 a 0,29 selon le film), donc ce n'est pas une horloge non plus.

### 2.4 Densite autour des evenements d'objectif — et c'est TRANCHE par mode

Fenetre +/- 2 s, ecrite avant la mesure. Les evenements de COMBAT (frags, assistances) sont
ecartes des fenetres : la question porte sur les captures. Les fenetres sont FUSIONNEES avant
d'etre comptees en secondes — elles se recouvrent, et les additionner gonflerait le
denominateur au point de faire passer un canal quelconque pour un canal concentre.

| film | mode | evenements retenus | i1 dedans / dehors | secondes dedans / dehors | **densite i1** | densite i0 |
|---|---|---|---|---|---|---|
| `7344d24f` | zone | 71 | 3 / 2 489 | 179,2 / 411,7 | **0,003x** | 0,63x |
| `696a9d7c` | zone | 77 | 9 / 2 476 | 180,5 / 375,5 | **0,008x** | 0,96x |
| `530820e5` | flag | 56 | 2 605 / 2 969 | 163,7 / 307,9 | **1,65x** | 2,64x |
| `64e8adfa` | flag | 116 | 5 990 / 6 922 | 288,1 / 547,6 | **1,65x** | 0,93x |

**VERDICT F2. Non : les deux composants portes de ti=47 ne sont pas le flux des messages plein
ecran.** Quatre raisons, dont trois independantes de la densite :

- Le canal dynamique est **anti-correle aux captures de zone d'un facteur 100 a 300** (0,003x
  et 0,008x : trois et neuf emissions en fenetre pour pres de 2 500 hors fenetre). Un message
  « Zone capturee » ferait exactement l'inverse. Le lot C mesurait 0,01x sur les annonces au
  masque ; la mesure au niveau des VALEURS donne le meme signe, en plus net.
- En CTF le meme canal SUIT les evenements de drapeau a 1,65x sur les deux films — mais avec
  2 605 et 5 990 emissions en fenetre pour 56 et 116 evenements, cela fait des DIZAINES
  d'emissions par evenement : ce n'est pas un message, c'est un canal continu dont le DEBIT
  monte quand un drapeau est en jeu.
- La structure en treillis (§2.3) ferme la question sans recourir a la densite : un
  identifiant de message ne se pose pas sur un pas de 4 584.
- Aucune valeur ne se comporte en message : sur `7344d24f`, les huit valeurs les plus
  frequentes du canal dynamique (16 occurrences chacune) sont TOUTES a zero emission en
  fenetre.

**Piste laissee ouverte, avec sa condition de reprise.** Le canal dynamique est un scalaire
quantifie dont le debit depend du mode (12 912 emissions en CTF `64e8adfa`, 342 en KOTH). Ce
qu'il mesure reste inconnu et ne se saura pas par densite : il faudra la retro-ingenierie du
deserialiseur (`FUN_140daebd0`) ou un oracle CONTINU, jamais un oracle d'evenement.

---

## 3. F3 — ti=4 `high-frequency` : RESOLU

### 3.1 L'archetype se resout par le NOM, et il le fallait

`high-frequency` est porte par DEUX archetypes du registre (3 et 4) sur les six films. Retenir
le premier venu prenait ti=3, qui n'a aucun slot dans les images-cle, et la sonde rendait
« aucune emission » sur un canal qui en compte 36 000. L'ambiguite est tranchee par ce que le
FILM montre — l'archetype qui a des slots — jamais par l'ordre du registre : un ordre n'est pas
une identite.

Le corpus contient d'ailleurs DEUX builds : `0a247154` porte 116 archetypes et l'empreinte
`2979f0b2e8596331`, les cinq autres 118 et `61e492dd4de7fd4e`. La resolution par nom traverse
les deux sans une ligne de code particuliere — la demonstration vivante de ce que le lot 0
avait etabli.

### 3.2 Volumes, purete, et loi de succession

| film | records ti=4 | fantome | rapport | masque annonce i0 | hors grammaire | emissions |
|---|---|---|---|---|---|---|
| `7344d24f` | 36 202 | 162 | **223x** | 99,58 % | 0,63 % | 36 051 |
| `696a9d7c` | 34 227 | 144 | **238x** | 99,17 % | 0,99 % | 33 943 |
| `530820e5` | 28 998 | 56 | **518x** | 99,81 % | 0,20 % | 28 944 |
| `64e8adfa` | 50 759 | 434 | **117x** | 99,59 % | 0,53 % | 50 550 |
| `0a247154` | 48 586 | 1 273 | **38x** | 97,55 % | 2,63 % | 47 397 |
| `000d5950` | 30 796 | 555 | **55x** | 98,70 % | 1,42 % | 30 396 |

C'est de tres loin le canal le plus PUR du corpus : une bande d'UN slot (deux sur `64e8adfa`),
38 a 518 fois son fantome, moins de 3 % de records hors grammaire, et un masque qui annonce
son unique composant dans 97,5 a 99,8 % des cas.

| film | valeurs distinctes / 256 | succession « +1 modulo 256 » | identiques | ecart median | ecart p95 |
|---|---|---|---|---|---|
| `7344d24f` | 256 | **98,8 %** | 0,0 % | 16 ms | 17 ms |
| `696a9d7c` | 256 | **98,9 %** | 0,0 % | 16 ms | 17 ms |
| `530820e5` | 256 | **99,5 %** | 0,0 % | 16 ms | 17 ms |
| `64e8adfa` | 256 | **99,0 %** | 0,0 % | 16 ms | 17 ms |
| `0a247154` | 256 | **98,7 %** | 0,0 % | 16 ms | 17 ms |
| `000d5950` | 256 | **99,6 %** | 0,0 % | 16 ms | 17 ms |

Les 256 valeurs sont toutes vues, chacune a ~0,4 % du total (distribution plate), la valeur
suivante vaut la precedente + 1 dans 98,7 a 99,6 % des cas, et jamais deux fois la meme valeur
d'affilee sur aucun des six films. Distribution des ecarts sur `7344d24f` : 16 ms dans 66,1 %
des cas, 17 ms dans 21,4 %, 15 ms dans 6,9 %.

### 3.3 Verdict

**VERDICT F3 : c'est le TIC DE SIMULATION.** Le R(8) de `high-frequency` est un compteur
modulo 256 incremente a chaque emission, emis toutes les 16 a 17 ms — la periode de 60 Hz du
moteur (l'alternance 16 ms / 17 ms est exactement celle d'un pas de 16,67 ms arrondi a la
milliseconde). Les 0,4 a 1,3 % de transitions non-`+1` correspondent aux trous d'emission
(ecarts de 33 a 305 ms observes en queue de distribution).

**Ce que ca donne, et que ce plan ne fait pas** (D9 : aucune publication ici) : une HORLOGE DE
TRAME entiere, continue, independante des horodatages de paquet, disponible sur tous les modes
et sur les deux builds du corpus. Elle numerote les trames modulo 256, soit 4,27 s de periode,
et pourrait servir de SECOND TEMOIN au calage temporel du rejeu — dont l'origine est
aujourd'hui calee sur le premier paquet de position (`build.go:323`). Ligne de registre a
poser ; condition de reprise : un lot qui touche a `originMs` (report `:123` du registre).

---

## 4. F4 — tacmap (ti=34, ti=30) : NEGATIF ECRIT

| film | ti=34 slots KF | records de bande | annoncent i7 `waypointstate` | hors grammaire | ti=30 slots KF | records ti=30 |
|---|---|---|---|---|---|---|
| `7344d24f` | 1 | 6 453 | **8** (0,12 %) | 96,4 % | **0** | **0** |
| `696a9d7c` | 1 | 5 873 | **16** (0,27 %) | 94,8 % | **0** | **0** |
| `530820e5` | 1 | 5 883 | **8** (0,14 %) | 97,1 % | **0** | **0** |
| `64e8adfa` | 1 | 3 215 | **6** (0,19 %) | 92,0 % | **0** | **0** |
| `0a247154` | 1 | 5 946 | **12** (0,20 %) | 91,5 % | **0** | **0** |
| `000d5950` | 1 | 1 643 | **14** (0,85 %) | 90,7 % | **0** | **0** |

**VERDICT F4 : NEGATIF, sur les six films, cinq modes et deux builds.**

- **ti=30 `tacmap-poiicon` n'existe pas en multijoueur** : zero slot dans les images-cle, zero
  record delta, sur les six films. Rien a lire, rien a esperer.
- **ti=34 a bien UN slot**, mais son composant identifiant n'est annonce que 6 a 16 fois par
  film (0,12 a 0,85 % des records de sa bande) et **90 a 97 % des records captes par cette
  bande annoncent un index que ti=34 NE POSSEDE PAS** (i >= 17). Autrement dit : la bande d'un
  seul slot ne capte presque que la contamination des archetypes voisins, et le vrai ti=34 est
  quasi muet. Les quelques annonces restantes sont du meme ordre de grandeur que le bruit.

**Le recensement des masques acheve la demonstration.** Sur `7344d24f`, les CINQ index les plus
annonces par la bande ti=34 sont `i27` (2 208), `i26` (1 524), `i59` (1 470), `i58` (715) et
`i60` (616) : **tous les cinq sont HORS de la grammaire de ti=34**, qui n'a que 17 composants.
La bande ne capte donc pas un ti=34 discret, elle capte les records d'autres archetypes. A titre
de comparaison, le meme recensement sur ti=4 rend `i0 high-frequency` a 36 051 en tete, et sur
ti=47 `i2 personal-ai-data-component` a 14 315 — les archetypes qui parlent vraiment mettent
leur propre composant en tete.

Le negatif de l'artefact « marqueurs a position tacmap » est donc confirme et ELARGI : le lot C
l'avait mesure sur deux films par le compte d'annonces ; il l'est ici sur six films, cinq modes
et deux builds, avec le controle de purete de bande en plus. **`tacmap-waypointstate` (ti=34 i7)
et `tacmap-poiicon` (ti=30 i0) sortent de la reserve** : ils appartiennent a la campagne. Le
seul marqueur vivant en multijoueur reste ti=12 `managed-navpoint`, deja au perimetre du lot C.

---

## 5. F5 — ti=13 `managed-object-property-name` : rien d'exploitable, ti=13 reste STOPPE

| film | records de bande | fantome | rapport | annoncent i0 | hors grammaire | emissions | valeurs distinctes |
|---|---|---|---|---|---|---|---|
| `7344d24f` | 35 948 | 1 660 | 21,7x | 0,52 % | **70,1 %** | 187 | 114 |
| `696a9d7c` | 36 582 | 1 530 | 23,9x | 0,46 % | **76,8 %** | 170 | 113 |
| `530820e5` | 2 941 | 946 | 3,1x | 8,02 % | 39,0 % | 236 | 126 |
| `64e8adfa` | 10 163 | 2 531 | 4,0x | 4,60 % | **51,2 %** | 467 | 341 |
| `0a247154` | 2 762 | 2 251 | **1,2x** | 5,18 % | 34,7 % | 143 | 125 |
| `000d5950` | 658 | 790 | **0,8x** | 5,62 % | **59,4 %** | 37 | 35 |

Quatre faits, et ils vont tous dans le meme sens :

1. **La bande ti=13 tombe AU PLANCHER DE BRUIT sur deux films sur six** : 1,2x le fantome en
   KOTH, et 0,8x — donc SOUS le fantome — sur le temoin Slayer. Une bande qui ne se distingue
   pas de son temoin ne mesure rien.
2. **Elle est massivement contaminee la ou elle se distingue** : 35 a 77 % de ses records
   annoncent un index que la grammaire de ti=13 (34 composants) ne possede pas. Tout chiffre
   tire de cette bande porte cette marge.
3. **Le composant PORTE (i0 `property-name`) est quasi muet** : 0,5 a 8 % des records de la
   bande l'annoncent, soit 37 a 467 emissions par film. Le canal BAVARD de ti=13 est i1
   `managed-object-property-component`, qui n'est PAS porte — et dont le lot C phase 1a a
   arrete la retro-ingenierie (variante a 11 branches, `LOTC_PHASE1A.md`).
4. **Les valeurs n'ont pas d'alphabet** : 114 valeurs distinctes pour 187 emissions, 341 pour
   467, 35 pour 37. Un « nom de propriete » resolu en identifiant de chaine aurait un petit
   vocabulaire repete ; ici presque chaque emission a sa valeur. Une seule ressort
   (`1789061888`, 19,3 % des emissions de `7344d24f`), le reste est disperse.

Le recensement des masques de la bande ti=13 sur `7344d24f` le dit d'un coup : `i38` (22 831
annonces) est HORS grammaire — c'est la contamination — puis viennent `i1
managed-object-property-component` (3 769) et trois occurrences de
`managed-object-player-masked-property-component` (`i21` 3 350, `i13` 3 347, `i17` 2 040), tous
NON portes. Le composant porte `i0` arrive loin derriere avec 187. **La ou ti=13 parle, le
depot ne lit pas ; la ou le depot lit, ti=13 se tait.**

**Lien avec ti=10 : NON ETABLI, et le test ne peut pas conclure.** La part des instants de
ti=13 qui portent aussi un record ti=10 vaut 44,8 a 92,1 % — mais le temoin (part des paquets
delta portant ti=10, sans aucun lien suppose) vaut 29,2 a 78,2 %, soit un rapport de **1,11x a
1,53x**. ti=10 parle dans une si grande part des paquets que la coincidence d'instant ne
discrimine rien : c'est le meme defaut de critere que la clause 2 du gate du lot C. Un test
discriminant demanderait de comparer l'OBJET designe, pas l'instant — donc la RE du champ de
reference, que le lot C a deja arretee.

**VERDICT F5 : aucun canal exploitable dans le composant porte de ti=13.** Le negatif est
ecrit ; ti=13 reste STOPPE en RE, comme la phase 1a du lot C l'avait tranche. Condition de
reprise si quelqu'un y revient : porter i1 (le canal bavard) AVANT de re-mesurer i0, et
disposer d'un ancrage plus pur qu'une bande de slots — 70 % de contamination interdit toute
conclusion fine, et le plancher de bruit avale la bande sur deux films du corpus.

---

## 6. Cout machine des sondes (D17)

Une seule passe par film sert les quatre sondes. Un film par processus, avant-plan, plafond
surveille.

| film | duree processus | pic memoire | passe delta | paquets delta | emissions du hook |
|---|---|---|---|---|---|
| `7344d24f` | 15,3 s | 25,4 Mo | 1,76 s | 36 350 | 38 815 |
| `696a9d7c` | 11,9 s | 23,1 Mo | 1,51 s | 34 276 | 36 684 |
| `530820e5` | 9,5 s | 21,5 Mo | 0,98 s | 29 148 | 34 830 |
| `64e8adfa` | 13,2 s | 31,0 Mo | 1,41 s | 50 956 | 64 040 |
| `0a247154` | 10,0 s | 25,9 Mo | 1,39 s | 47 854 | 47 959 |
| `000d5950` | 8,2 s | 21,2 Mo | 2,45 s | 30 418 | 30 930 |

**8 a 16 secondes et 21 a 31 Mo par film** — 1 % du plafond de 3 Go. Le poste dominant n'est
PAS la passe delta (1 a 2,5 s) mais le RECENSEMENT des images-cle (11,7 s sur `7344d24f`), qui
relit tout le film. Mutualiser les quatre sondes en une seule passe a divise le cout par quatre
par rapport a quatre instruments separes.

### Statut des items du lot F

- `[~]` F1 — porte par le lot P (P.0.2), hors perimetre de cet executeur.
- `[x]` **F2** — verdict rendu : les deux composants portes de ti=47 ne sont PAS les messages
  plein ecran ; le R(24) dynamique est un scalaire quantifie sur un treillis de pas 4 584,
  anti-correle aux captures de zone (0,003-0,008x) et suivant les evenements de drapeau
  (1,65x). Piste ouverte avec sa condition de reprise (RE du deser, pas un oracle d'evenement).
- `[x]` **F3** — RESOLU : compteur modulo 256 emis toutes les 16-17 ms = le tic de simulation a
  60 Hz. Six films, cinq modes, deux builds, 98,7 a 99,6 % de transitions `+1`.
- `[x]` **F4** — NEGATIF ECRIT : ti=30 absent des six films, ti=34 quasi muet (6 a 16 annonces)
  dans une bande contaminee a 90-97 %. tacmap = campagne, confirme et elargi.
- `[x]` **F5** — NEGATIF ECRIT : le composant porte de ti=13 est quasi muet et sans alphabet,
  la bande est au plancher de bruit sur deux films et contaminee a 35-77 % sur les autres, et
  le lien avec ti=10 n'est pas mesurable par coincidence d'instant. ti=13 reste STOPPE.

---

## 7. Decouvertes (hors perimetre — notees, NON traitees)

1. **`ScanFilmPlayerIndices` coute 1 min 15 a 1 min 30 par film**, soit 80 % du temps de
   l'instrument, pour un balayage de motif qui ne rend que huit couples (xuid, index). Tout
   instrument qui a besoin du pont slot -> xuid paie ce prix. A chiffrer et memoiser avec le
   re-parse du registre `chunk_00` deja note au §6 du plan.
2. **Le fil des morts perd 7 a 21 instants de kill par film** en instants « ambigus » (deux morts
   ou deux kills a la meme milliseconde). `killsource` les recolle par voisinage immediat et
   mesure que ce recollage echoue MOINS que les couples directs ; l'instrument E.0.1 ne l'a pas
   fait (il n'en avait pas besoin). Un lot qui voudrait la population complete des couples doit
   reprendre cette regle plutot qu'en inventer une.
3. **`high-frequency` est porte par DEUX archetypes du registre (3 et 4)** sur les six films.
   Tout code qui resoudrait cet archetype par le premier nom trouve prendrait ti=3, qui n'a
   aucun slot. Aucun code de production ne le fait aujourd'hui (le composant n'a aucun
   consommateur), mais la remarque vaut pour toute resolution par nom : un nom de composant
   n'identifie pas toujours un archetype a lui seul.
4. **Le tic de simulation de F3 est une horloge de trame utilisable** (compteur 60 Hz, modulo
   256, present sur tous les modes et les deux builds) alors que le rejeu cale aujourd'hui son
   origine sur le premier paquet de position. Non traite ici (D9 : les sondes ne publient
   rien) ; a rattacher au report `:123` du registre (correction d'origine).
5. **Le recensement des images-cle coute 12 s par film** dans l'instrument des sondes, contre
   1 a 2,5 s pour la passe delta elle-meme. Tout instrument qui a besoin d'une bande de slots
   paie ce prix. Meme famille de dette que la decouverte n°1 et que le re-parse du registre
   `chunk_00` deja note au §6 du plan.

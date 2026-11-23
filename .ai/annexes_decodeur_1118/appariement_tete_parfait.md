# H.7 (mesure) : appariement des tirs a la tete et des frags parfaits aux eliminations

Lecture seule, Node seulement, un rejeu a la fois. Corpus : 1 455 rejeux `halo_infinite` du poste, schema 125 pour tous,
`frameIntervalMs = 33` pour tous, `originMs` present partout. Un rejeu se designe par ses 8 premiers caracteres ; aucun
identifiant de joueur n'est ecrit nulle part (les joueurs sont des indices locaux a chaque rejeu dans `compact.jsonl`).
Scripts (dossier `<poste>/Downloads/Scripts/dec-v150-recup/h7/`) : `extract.js` (une passe, produit
`compact.jsonl`), puis `axis.js`, `ax2.js`, `hs4.js`..`hs8.js`, `coh2.js`, `mf.js`, `pf1.js`, `pf2.js`, `ex1.js`, `np.js`, `id1.js`.

Conventions. Ecart signe = montee (ou medaille) moins elimination, en images : positif = la montee vient APRES l'elimination.
Deux ensembles d'eliminations sont utilises, parce que le document n'en a pas d'autre :

- **A** = `scoreActions[]` de nom `killPersonalScore` (123 865 lignes) : tueur (`xuid`), victime (`target`, absente sur 1 746 lignes = 1,4 %),
  image `t`. C'est le seul calque a porter tueur ET victime datees. Il est absent ou partiel sur les vieux builds (voir 2.0).
- **K** = pas +1 (ou +n) de la courbe `scoreTimeline.players[].kills.total` : tueur et image, SANS victime. Presente pour tous les
  joueurs qui ont des compteurs, y compris quand A manque.

Le document n'a ni calque `kills`, ni `neutralDeaths` ici, ni champ de mode (voir 7).

## 1. Calques, horloges, axe commun

| Calque | Cle joueur | Horloge |
|---|---|---|
| `scoreTimeline.players[].stats.headshots.total` / `.kills.total` | `xuid` | `t` = index d'image, points aux changements |
| `scoreActions[]` (A) | `xuid` (tueur), `target` (victime) | `t` = index d'image |
| `medals[]` | `xuid` | `t` = index d'image ET `ms` (horloge des temps forts) |
| `tracks[]` (mort de la victime) | `xuid` | `releaseFrame`, `endFrame` en index d'image |

Tout est deja sur l'axe des images. Seul `ms` des medailles est sur l'autre horloge : `t = floor((ms - originMs) / frameIntervalMs)`.

Controles chiffres de l'axe :

- Formule des medailles : `t == floor((ms-originMs)/33)` pour 62 537 medailles sur 62 537 (round : 32 383, ceil : 1 932, donc floor seul est exact).
- Mort de la victime : pour 122 109 `killPersonalScore` a victime nommee, ecart entre l'image de l'elimination et le `releaseFrame` le plus proche de la
  piste de la victime = 0 pour 121 847 (99,79 %), -1 pour 31, au-dela de +-20 pour 231. Meme axe, sans decalage.
- Compteur de kills contre A, joueurs dont le nombre de lignes A egale la valeur finale du compteur (122 078 eliminations), i-eme pas contre i-eme ligne :
  0 : 114 593 (93,87 %), -1 : 4 476 (3,67 %), +1 : 2 956 (2,42 %), +-2 ou +-3 : 53. Les messages de score et le compteur tombent donc dans
  la meme image a +-1 pres, comme annonce, et la distribution est bornee.
- Les cles joueurs : toutes les cles de `scoreActions` (tueur et victime) sont dans `roster` (0 exception sur 123 865 lignes) ; tous les joueurs de
  `scoreTimeline.players` sont dans le roster sauf 1 (rejeu de build vide). Les medailles hors roster (459) sont toutes dans les 5 rejeux de build
  vide (roster vide). Aucune table de correspondance n'est necessaire a l'interieur du document : tout est nomme par le meme `xuid` decimal.
  (`identity.players` passe par `filmIndex`, mais les calques ci-dessus ne s'en servent pas.)

## 2. Tirs a la tete

### 2.0 Le compteur

- Joueurs du parc : 14 034 ; avec courbe `headshots` : 9 585 ; sans la cle `headshots` mais avec `stats` : 4 443 (4 018 avec au moins un frag, 425 sans) ;
  sans `stats` du tout : 6 (voir 5). Une cle absente se lit « zero tir a la tete » par convention (voir 7 : non prouvable).
- Points de courbe : 36 984 = 36 847 pas de +1, 12 pas de +2 ou plus, 125 points d'ouverture a v=0 (pas de montee).
- Baisses de la courbe cumulee : 0 sur tout le parc, y compris les 25 rejeux a plusieurs manches (`rounds >= 2`).
- `coverage.score.statsGuarded` : absent dans les 1 455 rejeux, `statsWithheld` : 0 partout.
- Disponibilite de A : seuls 86,2 % des frags du parc (compteur : 143 639 ; A : 123 865) ont une ligne A. Joueurs a frags>0 :
  A egale le compteur (« A fiable ») pour 11 523 joueurs, A absent pour 1 790, A incomplet (moins de lignes) 273, A excedentaire 9.
  Montees +1 par categorie : A fiable 30 833, A absent 5 183, A incomplet 806, A excedentaire 25.

### 2.1 Distribution de l'ecart (A fiable : 11 523 joueurs, 122 078 eliminations, 30 833 montees de +1)

```
<-6 : 1   -4 : 4   -2 : 5   -1 : 1 180 (3,83 %)   0 : 28 879 (93,66 %)   +1 : 763 (2,47 %)   >6 : 1
```

Contre K (tous joueurs, pas de kills de meme joueur), meme image : 36 839 des 36 847 montees ont exactement un frag dans la meme image
(99,98 %), 5 aucun, 3 plusieurs. La montee des tirs a la tete et celle des frags sont ecrites par le meme enregistrement du tableau.

### 2.2 Fenetre justifiee : +-1 image

La masse est a 0 (93,66 %) et +-1 (6,30 %) ; au-dela il ne reste que 11 montees sur 30 833 (0,04 %), eparpillees de -6 a >6 : ce sont des
orphelins, pas une queue de distribution. Elargir ne ramene rien (zero candidat : 11 a +-1, 6 a +-2, 6 a +-3) mais cree des ambiguïtes (>=2
candidates : 7 a +-1, 14 a +-2, 17 a +-3). La fenetre +-0 manque les 6,3 % decales d'une image.

Montees +1 sur A fiable, nombre d'eliminations candidates (0 / 1 / >=2) :

| Fenetre | 0 | 1 | >=2 |
|---|---|---|---|
| +-0 | 1 954 (6,34 %) | 28 877 (93,66 %) | 2 |
| **+-1** | **11 (0,04 %)** | **30 815 (99,94 %)** | **7 (0,02 %)** |
| +-2 | 6 | 30 813 | 14 |

Sens inverse (fenetre +-1) : sur 122 078 eliminations, 30 825 recoivent exactement une montee (25,3 %), 2 en recoivent deux, 91 251 aucune.

Les 7 montees a deux candidates sont des paires de frags du meme joueur a 0 ou 1 image d'ecart (ex. 022d2e0e images 9731 et 9732 : deux
frags, deux montees, une par image : l'appariement un-a-un par image exacte les separe). Les 2 cas a une seule montee pour deux frags de la meme
image sont traites en 2.5.

### 2.3 Coherence par joueur (compteur final = eliminations appariees)

- A fiable, joueurs dont le compteur final est > 0 : 8 120 ; coherents (chaque montee de +1 prend une elimination distincte a +-1, aucun pas >= 2,
  compteur final = nombre de montees) : 8 103 = **99,79 %**. Les 17 autres : 9 avec une montee sans elimination a +-1, 8 avec un pas de +2 ou plus.
  Les 3 403 autres joueurs A fiables ont un compteur final a 0 (3 341 sans cle, 62 avec cle a 0) : coherents par construction.
- Sur K (tous joueurs, toutes versions, compteur final > 0) : 9 483 sur 9 495 = 99,87 %. Par version : 1.13 : 6 916/6 923, 1.12 : 1 040/1 042,
  1.11 : 903/904, 1.10 : 436/437, 1.9 : 32/33, 1.8 : 156/156.

### 2.4 Par version du jeu (`coverage.decoder.build`)

Montees +1 contre A (population A fiable), fenetre +-1, 0 / 1 / >=2 candidates ; contre K (tous joueurs), fenetre meme image (+-0) puis +-1 :

| Build | Rejeux | Joueurs A fiables | Montees A fiable | A +-1 (0/1/>=2) | Eliminations A fiables recevant une montee | Montees K (tous) | K +-0 | K +-1 |
|---|---|---|---|---|---|---|---|---|
| HI_1_13_0 | 1 200 | 9 724 | 25 350 | 7 / 25 336 / 7 | 25 346 sur 105 628 | 25 682 | 1 / 25 679 / 2 | 1 / 25 674 / 7 |
| HI_1_12_0 | 145 | 1 194 | 4 427 | 4 / 4 423 / 0 | 4 423 sur 12 113 | 4 905 | 0 / 4 905 / 0 | 0 / 4 905 / 0 |
| HI_1_11_0 | 57 | 545 | 1 047 | 0 / 1 047 / 0 | 1 047 sur 4 100 | 3 282 | 0 / 3 282 / 0 | 0 / 3 282 / 0 |
| HI_1_10_0 | 31 | 51 | 9 | 0 / 9 / 0 | 9 sur 216 | 1 992 | 3 / 1 988 / 1 | 3 / 1 988 / 1 |
| HI_1_9_0 | 3 | 9 | 0 | - | - | 117 | 1 / 116 / 0 | 1 / 116 / 0 |
| HI_1_8_0 | 13 | 0 | 0 | - | - | 869 | 0 / 869 / 0 | 0 / 869 / 0 |
| HI_1_4_1 | 1 | 0 | 0 | - | - | 0 | - | - |
| build vide | 5 | 0 | 0 | - | - | 0 | - | - |

Lecture : par A, l'appariement n'est ouvert que sur 1.11 a 1.13 (les builds 1.8 a 1.10 ont A absent ou partiel : 1.10 compte 223 joueurs A
incomplet, 435 A absent). Par K, il l'est sur tous les builds a 99,9 % ou plus, mais sans victime.

### 2.5 Par mode

Le document n'a pas de champ de mode. Proxy par les calques presents (`flagCarries`, bombe, zones, cranes, vip, equipes) ; etiquettes
approximatives, a lire comme « type de rejeu » :

| Proxy de mode | Rejeux | Montees +1 (A fiable) | A +-1 (0/1/>=2) | K tous : montees | K +-0 (0/1/>=2) |
|---|---|---|---|---|---|
| frags en equipes | 1 123 | 24 388 | 8 / 24 374 / 6 | 28 936 | 2 / 28 931 / 3 |
| drapeau | 186 | 3 876 | 3 / 3 872 / 1 | 4 838 | 3 / 4 835 / 0 |
| zones/colline | 125 | 2 267 | 0 / 2 267 / 0 | 2 425 | 0 / 2 425 / 0 |
| bombe | 9 | 156 | 0 / 156 / 0 | 241 | 0 / 241 / 0 |
| cranes | 7 | 146 | 0 / 146 / 0 | 252 | 0 / 252 / 0 |
| vip | 3 | 0 (A absent) | - | 29 | 0 / 29 / 0 |
| frags solo | 2 | 0 (A absent) | - | 126 | 0 / 126 / 0 |

Aucun mode ne se detache. Les 25 rejeux a plusieurs manches (drapeau 15, bombe 3, cranes 7) ont tous la courbe cumulee monotone et le
meme taux d'accord.

### 2.6 Cas a part : plusieurs frags du meme joueur dans la meme image, une seule montee

- Groupes (A) : 588 groupes (1 187 frags) de 2 frags ou plus du meme joueur dans la meme image ; 4 portent une montee ; 2 ont une montee
  plus petite que le nombre de frags (01db4132 image 12230 : 2 frags, +1 ; 941f7f8c image 9856 : 2 frags, +1) ; les 2 autres ont une montee
  egale au nombre de frags (tous les frags sont des tirs a la tete : decidable).
- Sur K (toutes les montees de frags de +2 ou plus : 817) : 6 portent une montee a +-1 ; 3 sont partielles (les deux ci-dessus plus
  3274c610 image 19314, ou A est absent) ; 3 totales.
- Departage par le document : non. A donne la victime et l'ordre de message, jamais l'arme ; `hits` et `damage` portent l'arme et l'effet mais
  aucune categorie « tir a la tete ». Rien ne dit lequel des deux est le tir a la tete : indecidable par le document seul (3 cas sur 1 455 rejeux).

### 2.7 Montees de +2 ou plus et valeurs impossibles

- 12 pas de +2 ou plus dans 11 rejeux (1ea71def, 3fd617b4, 508bd2fb, 58e3825b, 5e121187, 885b158a, 93fcc545, a26d3c4d, a9dcfe1a, bcb6d393,
  eba1e63f) ; 3 coincident avec un pas de frags >= a +-1 image (doubles reels dans la meme image), 9 non expliques.
- 5 points dont la valeur cumulee depasse le compteur de frags a l'image suivante (faux points) : 3fd617b4 image 889 (+2 pour 0 frag),
  508bd2fb image 1 (+4, 0 frag), 93fcc545 image 10401 (+15, 5 frags), bcb6d393 image 3817 (+4, 1 frag), 1c4c63c2 image 16140 (7 tirs a
  la tete pour 6 frags). A comptes a part : jamais apparies.
- Baisses : 0.

## 3. Frags parfaits

Nom exact : **`Perfect`** (id de medaille 109). 1 928 medailles dans 773 rejeux. Une autre medaille proche, **`Perfection`** (id 13, 3
occurrences, gagnee sur un match sans mort), n'est pas un frag parfait et n'est pas comptee. Plusieurs `Perfect` du meme joueur dans la meme
image : 0.

Ecart (medaille - pas du compteur de frags du meme joueur, K), 1 897 medailles ayant un compteur (31 medailles sont dans les 5 rejeux de build vide, sans
joueur ni compteur) :

```
-2 : 3   -1 : 97 (5,1 %)   0 : 1 594 (84,0 %)   +1 : 203 (10,7 %)
```

Fenetre justifiee : +-2 (toute la masse est dans [-2, +1], rien au-dela). Nombre de frags candidats (0 / 1 / >=2) :

| Fenetre | 0 | 1 | >=2 |
|---|---|---|---|
| +-0 | 303 | 1 594 | 0 |
| +-1 | 3 (0,16 %) | 1 894 | 0 |
| **+-2** | **0** | **1 897 (100 %)** | **0** |
| +-3 | 0 | 1 897 | 0 |

Contre A (1 411 medailles dont le joueur a des lignes A ; les 517 autres sont des rejeux sans A) : ecart -1 : 26, 0 : 1 166, +1 : 121, -2 : 1,
|ecart| > 6 : 97 (85 en 1.10 et 12 en 1.9, ou A est partiel) ; a +-1 : 98 sans candidate (86 en 1.10, 12 en 1.9), 1 313 avec une seule, 0 avec
plusieurs. Les 3 orphelins de K a +-1 : 34bb3bc8 image 10652, a2454be5 image 9347, a409730c image 1655 (le pas de frag est 2 images apres la
medaille).

Par version, K, +-2 : 100 % d'une seule candidate sur 1.8 (78), 1.9 (15), 1.10 (221), 1.11 (318), 1.12 (205), 1.13 (1 060) ; a +-1, 1.11 en perd 2,
1.10 en perd 1. Par mode : drapeau 217, zones 91, cranes 25, bombe 17, frags equipes 1 547 sur 1 578 (+31 sans compteur) : 100 % a +-2.
Sens inverse : sans objet.

Le `Perfect` s'ancre donc sur le pas de frags, pas sur la ligne A (qui manque sur les vieux builds). Ce que le document ne donne pas : la
victime du frag parfait quand A manque.

## 4. Lecture actuelle par elimination

Aucune. Le document de rejeu (schema 125) ne porte aucune marque de tir a la tete par elimination : les 9 585 occurrences du mot
`headshot` (insensible a la casse) dans les 1 455 rejeux sont TOUTES la cle `headshots` du compteur, une par joueur ayant une courbe. Les
types `Hit`, `Damage`, `ScoreAction`, `Medal` n'ont pas de tel champ. Le decodeur connait la categorie de degat `Headshot` /
`HeadshotMultiplier` dans sa couche de source des frags (`internal/facts/killsource/kill.go`), mais le document de schema 125 ne la publie pas.
La marque actuelle de l'application (si elle existe) est hors de ma portee : non comparee.

## 5. Joueurs sans compteur

- `stats` absent : 6 joueurs dans 6 rejeux (1550b5c0, be565134 en 1.11 ; d912604d en 1.12 ; e624c2a4, e6ccc947, f6a9d127 en 1.13), tous des films
  d'une seule manche, `statsWithheld = 0`. Leur nombre de frags dans A est 0 et la valeur du compteur est inconnue.
- Aucun rejeu du parc n'a `statsWithheld > 0` : le cas « film a plusieurs manches dont le calque retient les compteurs » n'existe pas dans
  ce corpus (les 25 rejeux a plusieurs manches publient tous leurs compteurs).
- 995 entrees du roster (737 rejeux) n'ont aucune ligne dans `scoreTimeline.players` ; elles n'ont aucune ligne A non plus. Je ne peux pas
  dire si ce sont des robots ou des humains sans compteur (voir 7).
- Les 5 rejeux de build vide n'ont ni roster, ni joueur, ni A ; ils portent des medailles (459, dont 31 `Perfect`) sans aucune elimination
  datee a laquelle les rattacher.
- Part des frags du parc sans verite de tir a la tete : au minimum les frags des 6 joueurs sans `stats` (inconnus) et ceux des 995 entrees
  sans ligne de score (inconnus) ; hors ces cas, tout joueur avec `stats` a une verite (compteur present, ou cle absente lue comme zero).
  Mesurable seulement cote victime : 13,8 % des frags du parc n'ont pas de ligne A (donc pas de victime nommee dans le document) mais ont leur pas
  de compteur.

## 6. Verdict

Appariement sur les pas de compteur (K : meme image, tueur sur le compteur de frags) : sur, a 99,98 % d'une seule candidate (36 839 / 36 847) et 99,87 % de
joueurs coherents. Appariement sur A (tueur, victime, image) : sur a 99,94 % (30 815 / 30 833, fenetre +-1) sur 1.11 a 1.13 ; fiable sur 85 % des frags
du parc seulement. Frags parfaits : 100 % d'une seule candidate a +-2 images (1 897 / 1 897), 99,84 % a +-1.

| | Montees tirs a la tete | Frags parfaits |
|---|---|---|
| Une seule candidate | 99,94 % (A, +-1) ; 99,98 % (K, meme image) | 100 % (K, +-2) |
| Aucune (orphelins) | 0,04 % (11 sur 30 833, A) ; 0,01 % (5, K) | 0 % (K, +-2) ; 3 a +-1 |
| Plusieurs (ambiguïte) | 0,02 % (7, A) ; 0,02 % (8, K a +-1) | 0 |

Sur quelles versions : 1.11, 1.12, 1.13 de bout en bout (A et K). Sur 1.8 a 1.10, K seul (pas de victime) ; l'accord K y est de 99,8 % (1.10), 99,1 % (1.9) et 100 % (1.8), sur 1 992, 117 et 869 montees.

Reste indecidable par le document seul :

1. Deux frags du meme joueur dans la meme image avec une seule montee (3 cas dans 3 rejeux) : rien ne dit lequel est a la tete.
2. Les montees de +2 ou plus non expliquees (9 pas dans 9 rejeux) et les 5 faux points (valeur au-dessus du compteur de frags).
3. Les 11 montees orphelines de A fiable (decalees de 2 a 8 images, ou une de 151).
4. Les frags sans ligne A (13,8 % du parc, 81 rejeux sans scoreActions + builds 1.8 a 1.10) : l'instant et le tueur sont connus, jamais la victime.
5. Les 6 joueurs sans `stats`, les 995 entrees sans ligne de score et les 5 rejeux de build vide : aucune verite.
6. Le rattachement a une ligne du fil des eliminations de l'application (cle tueur-victime-temps) : hors du document, non mesure.

Exemples de cas ambigus ou orphelins :

1. 01db4132, image 12230 : 2 frags du meme joueur (victimes differentes), 1 montee de tir a la tete. Indecidable.
2. 941f7f8c, image 9856 : idem.
3. 3274c610, image 19314 : pas de frags de +2, tir a la tete +1 ; A absent (pas de victime). Indecidable.
4. 022d2e0e, images 9731 et 9732 : 2 frags a 1 image d'ecart, 2 montees ; fenetre +-1 donne 2 candidates, l'image exacte les separe.
5. 0014603f, images 8470, 8696, 8721, 8849 : 4 montees dont l'elimination la plus proche est 4 images apres (orphelins a +-1).
6. 58d09c44, images 6305, 6387, 11608 : montees dont l'elimination la plus proche est 2 images apres.
7. dbb4cd56, image 10924 : montee dont l'elimination la plus proche est 8 images apres.
8. 57f4be06, image 19213 : montee dont l'elimination la plus proche est a 151 images (orphelin franc).
9. 93fcc545, image 10401 : un seul pas de +15 sur le compteur de tirs a la tete, alors que le joueur n'a que 5 frags : faux point.
10. 508bd2fb, image 1 : un pas de +4 a l'image 1, 0 frag. Faux point. (Voir aussi 3fd617b4 image 889 : +2, 0 frag.)
11. 34bb3bc8 (image 10652), a2454be5 (9347), a409730c (1655) : medaille `Perfect` a +2 images du pas de frags (hors +-1).
12. 13b00e35 : build vide, roster vide, 71 medailles sans aucune elimination ; 31 `Perfect` du parc sont dans ce cas (5 rejeux).

## 7. Ce que je n'ai pas pu etablir

- Le mode du match : aucun champ de mode dans le document ni dans la `.meta`. Le proxy par calques presents est approximatif (par exemple, « frags
  en equipes » regroupe les modes sans objectif lu).
- Que l'absence de la cle `headshots` veuille dire zero : 4 018 joueurs avec au moins un frag n'ont pas la cle ; cote 1.13, 22,8 % des joueurs a 8 frags
  ou plus (1 638 sur 7 200) sont dans ce cas, 9 rejeux ont tous leurs joueurs a 8 frags et plus sans cle. Plausible (un joueur sans tir a la tete) mais pas
  prouve : les compteurs de melee et de grenade apparaissent aussi sans cle pour 2 429 joueurs.
- Si les 995 entrees du roster sans ligne de score sont des robots ; si leurs frags existent et manquent.
- Le rattachement des montees ou medailles a une ligne du fil des eliminations de l'application : le fil (temps forts, millisecondes) n'est pas
  dans le document ; seule la clef (tueur, victime, image) via A l'autorise, a 86 % des frags. L'accord image du document contre milliseconde du fil
  n'est pas mesure ici. L'ecart de 1 image entre compteurs et messages A (6 %) suggere de ne pas apparier a la milliseconde.
- La cause des 9 pas de +2 ou plus non expliques et des 11 orphelins ; les 5 faux points sont des valeurs impossibles, pas des causes.
- La part des tirs a la tete portee par chaque arme ou victime (rien dans le document ne le dit).

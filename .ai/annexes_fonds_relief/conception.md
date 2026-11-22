# Conception du fond « Relief » — item 0.1 du plan `PLAN_FONDS_RELIEF_2026-11-18.md`

Rédigé le 2026-11-18, en lecture seule, sur `release/v8.3.0` (`38f133c22d`) et la brique
`halo-infinite-map-images` v1.3.1 (clone `C:\t\relief-brique`, `3003197`). Les décisions D-1 à D-11 du
plan sont prises telles quelles. Chaque choix porte sa mesure sur les quatre cartes témoins : Recharge
(`sgh_blueprint`), Streets (`sgh_streets`), Behemoth (`va_behemoth`), Lattice (`1a6cfc2e-…`).

Unités. Les hauteurs et les distances sont en **unités du jeu**, notées `u`. Le code les appelle
« mètres » (`metersPerPixel`), ce qu'elles ne sont pas : sur 1,5 million de points de piste, la vitesse
médiane au sol est de 2,2 à 2,4 u/s et le sprint de 3,05 u/s, soit environ 3 m par unité (10 pieds, l'unité
du moteur). Un étage fait 1,8 à 2,2 u. Le pixel du fond vaut 0,029 u (Recharge), 0,036 u (Streets),
0,0496 u (Behemoth), 0,0438 u (Lattice) : un seuil en pixels ne vaut donc pas la même distance d'une carte à
l'autre. Les poids sont en Ko de 1 024 octets.

Mesures : scripts jetables, hors dépôt, dans
`C:\Users\GUILLA~1\AppData\Local\Temp\claude\17871fa88c29004d\63a04594-e948-4485-9732-90ff5a088fa7\scratchpad\conception`
(`m1` à `m9`, sorties `m1.txt` à `m9.txt`). Ce dossier est temporaire : les chiffres utiles sont recopiés ici.

---

## 0. Ce que la mesure change à la proposition du plan

| Sujet | Proposition du plan | Retenu | Mesure qui tranche |
|---|---|---|---|
| Rangement des 16 bits | R, G | **R = octet faible, G = octet fort** | 6 à 17 % de moins (Behemoth 625 → 521 Ko) |
| Pas de quantification | amplitude / 65 535 (maquette) | **1/256 u, fixe** | 43 à 45 % de moins ; erreur d'ombrage ≤ 1,9 niveau sur 255 |
| Drapeaux | sur la carte, sol du maillage, toit, « ce qu'il faut pour la coupe » | **2 bits** : sur la carte, sol du maillage | le toit se juge par sol de la pile, pas par pixel (§2) |
| Hauteurs basse et haute par trait | oui, la coupe filtre par elles | **non** : la coupe filtre par le masque | les deux règles divergent sur 5 à 29 % de la longueur (§3.6) |
| Sols du maillage | faces `[x, y, z, …]` | **sommets indexés**, centièmes d'unité | 30 à 45 % de moins avant gzip |
| Calcul lourd dans le navigateur | « une seconde par carte » | empilement 18 à 51 ms, ombrage 46 à 93 ms, coupe 24 à 61 ms | le lourd était les traits, qui partent à la cuisson |
| Coupe de la maquette | reprise telle quelle | **corrigée** | jusqu'à 57 % du sol d'un étage y manque (§4.4) |
| Écart minimal entre étages | 1,2 u | **1,4 u** (2 × bande) | deux étages de Behemoth à 1,19 u se recouvraient |
| Suivi du joueur | seuil sur la hauteur de la piste | **sol porteur** + seuil 0,5 u + délai 300 ms | 3 à 5 fois moins de temps hors bande (§8) |

Trois faits nouveaux pèsent sur le plan et sont repris au §9 : le « dessus » du fond publié est **plus bas
qu'un sol praticable** sur 9,5 à 28,4 % de la zone ; le dossier des fonds **double** environ ; le fond
publié porte 1,4 à 10,9 % de matière **de plus** que la recuisson d'aujourd'hui.

---

## 1. Format des fichiers par carte

### 1.1 `<clé>.relief.webp`

WebP sans perte (VP8L, `nativewebp`, comme les autres fonds), **mêmes dimensions et même calage que le fond
publié** (D-5). Par pixel :

| Canal | Contenu |
|---|---|
| A | 255 quand la hauteur est connue, 0 sinon. Jamais d'autre valeur. A = 0 impose R = G = B = 0 |
| R | octet **faible** de `q` |
| G | octet **fort** de `q` |
| B | drapeaux (§2) ; bits 2 à 7 à zéro |

`q = R + 256 × G` (0 à 65 535) ; **hauteur = `z0` + `q` × `step`**, avec `step` = 1/256 u = 0,00390625 et
`z0` = partie entière (vers le bas) de la plus petite hauteur de la carte. `z0` et `step` sont écrits dans
le JSON : le web ne les suppose pas. L'octet fort se lit alors « unités entières au-dessus de `z0` ».

Capacité : 255,99 u d'amplitude. Mesuré : 16,7 u (Streets), 19,4 u (Recharge), 25,0 u (Lattice), 40,0 u
(Behemoth) ; la tranche de jeu du rendu borne la matière à 40 u par construction (`TrancheDeJeuMin` −12,
`TrancheDeJeuMax` +28). Au-delà : écrêtage et dégradation `hauteurs-ecretees`.

Pourquoi A ∈ {0, 255} : le canevas 2D range ses pixels prémultipliés ; une alpha intermédiaire écraserait
R, G, B à la relecture (`getImageData`). L'outil `mapfond-webp` compare lui aussi en `image.RGBA`
prémultipliée (`versRGBA`). Vérifié en Go sur les 4 témoins : aller-retour identique au bit près, en
comparaison `NRGBA` stricte comme en comparaison de l'outil ; `webp.Decode` rend une `*image.NRGBA`.

Une hauteur est écrite partout où elle est connue, **zone gardée ou non** : la coupe sur le Rendu (D-2) en a
besoin là où une carte n'est pas rognée au maillage. Sur les 4 témoins, 0 à 128 pixels seulement ont une
hauteur hors de la zone ; l'écrire ou non ne change pas le poids (83 134 contre 83 084 octets sur Recharge).

**Poids mesurés** (WebP sans perte, compression maximale ; « dessus » de la maquette, voir §4.2 pour (b)) :

| Variante | Recharge | Streets | Behemoth | Lattice |
|---|---|---|---|---|
| Proposition du plan telle que la maquette : R fort, G faible, pas = amplitude / 65 535, 4 bits | 143,5 | 216,6 | 1 189,2 | 366,7 |
| même chose, pas 1/1024 u | 107,8 | 161,1 | 1 017,0 | 289,2 |
| même chose, pas 1/256 u | 81,2 | 123,5 | 653,6 | 203,5 |
| pas 1/256 u, 2 bits | 76,7 | 116,9 | 625,3 | 194,6 |
| **RETENU : pas 1/256 u, 2 bits, R faible et G fort** | **72,1** | **105,0** | **521,2** | **169,5** |
| retenu, avec le dessus relevé au sol du maillage (§4.2, option b) | 75,2 | 119,7 | 478,4 | 159,4 |
| pour mémoire : pas 1/128 u (R fort) | 69,2 | 107,0 | 541,8 | 170,9 |
| pour mémoire : demi-résolution (R fort, 2 bits) | 37,1 | 49,8 | 240,1 | 80,2 |

Gzip par-dessus le WebP : aucun gain (−0,04 à −0,8 %). PNG au même contenu : plus lourd de 14 à 34 % ; on
reste en WebP, la chaîne existe.

Autres répartitions essayées et écartées :
- **octet faible en dents de scie repliées** (zigzag, pour éviter le saut de 255 à 0) : gain nul (−0,2 à −0,7 %) ;
- **gris ombré cuit dans B** (l'image s'afficherait sans calcul) : 204,7 / 260,5 / 968,1 / 381,4 Ko, soit
  1,9 à 2,8 fois le retenu, et l'ombrage serait figé pour l'orientation zéro (§6.4) ; l'ombrage se calcule en
  46 à 93 ms dans le navigateur ;
- **pas 1/128 u** : 13 à 17 % de moins à rangement égal, mais l'erreur d'ombrage monte à 4,2 niveaux sur 255 ;
- **demi-résolution** : 2,1 à 2,6 fois moins lourd, mais le remplissage perd le détail des petits objets ; jamais vu
  à l'écran, donc non retenu (levier du §9).

Erreur d'ombrage selon le pas (écart au gris calculé sur les hauteurs en virgule flottante, en niveaux sur
255, maximum des 4 cartes) : pas de la maquette 0,2 ; 1/1024 → 0,6 ; 1/512 → 0,9 ; **1/256 → 1,9** (moyenne
0,09, aucun pixel au-delà de 2) ; 1/128 → 4,2 (0,4 % des pixels au-delà de 2) ; 1/64 → 7,9 (3 %).

### 1.2 `<clé>.relief.json`

Écrit sur une ligne (aucun diff lisible à attendre d'un fichier de traits). Clés en anglais, comme le sidecar.

```json
{
  "schemaVersion": 1,
  "calibration": { "metersPerPixel": 0.029, "originX": -10.011253052711481,
                   "originY": 28.708047019958506, "widthPx": 1605, "heightPx": 1771 },
  "height": { "z0": -7, "step": 0.00390625, "low": -0.02, "high": 11.38 },
  "floors": {
    "levels": [0, 2.21, 4.4], "min": -0.01, "max": 4.42,
    "merge": 0.4, "tolerance": 0.3, "meshShare": 0.138,
    "vertices": [-161, 322, 220, -161, 340, 220, "…"],
    "faces": [[0, 1, 2], [3, 4, 5, 6, 7], "…"]
  },
  "lines": {
    "steps":    [[1052, 3310, 0, 25, 5, 5, "…"], "…"],
    "contours": [["…"]],
    "outline":  [["…"]]
  },
  "degradations": []
}
```

| Champ | Unité, sens | Lecteur |
|---|---|---|
| `schemaVersion` | 1 | service (refus d'une version inconnue), web |
| `calibration` | copie du calage du fond publié auquel le relief a été projeté | **service** : s'il diffère du calage du sidecar, le relief est périmé et n'est pas servi (§5.4) |
| `height.z0`, `height.step` | u ; décodage de l'image | web |
| `height.low`, `height.high` | u ; 2e et 98e centiles des hauteurs de la zone | web : bornes de la rampe de clarté |
| `floors.levels` | u, croissant ; les étages détectés (§4.5) | web : pastilles |
| `floors.min`, `floors.max` | u ; 1er et 99e centiles des hauteurs de sol | web : bornes du curseur |
| `floors.merge`, `floors.tolerance` | u ; fusion des sols empilés, écart toléré entre un dessus et son sol | web : la pile se reconstruit avec les seuils de la cuisson |
| `floors.meshShare` | 0..1 ; part de la zone dont la hauteur vient du maillage | web : mention du fond Relief (question 5 du §9) |
| `floors.vertices` | **centièmes d'unité, entiers, monde** : `x, y, z` à plat | web : pile des sols, sol porteur, murs de la coupe |
| `floors.faces` | une liste d'indices de sommets par face gardée (polygone convexe) | idem |
| `lines.steps` | ruptures de hauteur ; polylignes (§3.5) | web : traits |
| `lines.contours` | courbes de niveau | web : traits |
| `lines.outline` | liseré ; anneaux fermés, sans répéter le premier point | web : traits |
| `degradations` | codes, liste fermée : `sans-maillage`, `hauteurs-ecretees`, `pile-tronquee` | web : `sans-maillage` cache la commande d'étage (D-7) |

Une polyligne est `[x0, y0, dx1, dy1, dx2, dy2, …]` en **dixièmes de pixel entiers** du cadre publié. Le
premier couple est absolu, les suivants sont l'écart au point précédent. L'entier désigne le **centre** du
pixel :

```
xMonde = originX + (u / 10 + 0,5) × metersPerPixel
yMonde = originY − (v / 10 + 0,5) × metersPerPixel
```

Le JSON **ne nomme pas l'image** : `reliefImage` du sidecar est la seule source du nom (c'est lui que
`mapfond-webp -convertir` réécrit), `reliefData` celle du nom du JSON. Les dégradations sont des **codes**,
pas des phrases : le web les traduit, le serveur n'envoie aucun texte français.

Carte sans maillage (D-7) : `degradations: ["sans-maillage"]`, `floors.levels`, `vertices` et `faces` vides,
`min` et `max` absents ; tous les pixels de matière sont « sur la carte » ; le liseré suit le bord de la matière.

**Poids mesurés** du JSON retenu (tolérance 0,75 px, miettes ôtées, §3.4) :

| | Recharge | Streets | Behemoth | Lattice |
|---|---|---|---|---|
| JSON, dessus de la maquette | 146,4 | 282,4 | 362,1 | 241,4 |
| après gzip (niveau 5, celui de l'API) | 49,5 | 88,6 | 125,8 | 79,8 |
| dont traits, avant / après gzip | 125,1 / 40,6 | 254,3 / 77,0 | 224,1 / 72,9 | 204,6 / 65,6 |
| dont sols, avant / après gzip | 21,0 / 8,3 | 27,7 / 10,8 | 137,6 / 52,5 | 36,4 / 13,5 |
| JSON, dessus relevé (option b) | 129,4 | 242,0 | 287,6 | 186,6 |
| après gzip | 45,6 | 75,9 | 104,0 | 63,6 |

Repères : traits sans simplification ni filtre 864 / 1 332 / 1 651 / 1 004 Ko (158 / 251 / 283 / 214 après
gzip) ; faces à plat à deux décimales 29,9 / 41,5 / 252,2 / 60,0 Ko (7,7 / 10,0 / 61,4 / 13,7 après gzip :
l'indexation gagne sur disque, pas sur le fil, mais elle donne les arêtes partagées dont les murs de la coupe
ont besoin). Brotli ne fait pas mieux que gzip (−1 %).
L'API compresse déjà le JSON (`chimiddleware.Compress(5)`, `internal/api/server.go`) : le poids transféré
est celui « après gzip ».

**Transfert à la première ouverture** (image + JSON compressé) : Recharge 121 Ko, Streets 194 Ko, Behemoth
647 Ko (582 Ko en option b), Lattice 249 Ko. À comparer au fond et à l'image plan publiés de Behemoth :
241 Ko + 143 Ko.

**Dossier des fonds.** Sur disque, sans compression : 218 à 883 Ko par carte, 475 Ko en moyenne des 4 témoins
(420 Ko en option b). Pour 114 cartes : **40 à 55 Mo ajoutés** aux 45 Mo d'aujourd'hui. Estimation, pas une
mesure : 92 cartes sur 114 sont des Forge, plus proches de Lattice (346 à 411 Ko) que de Behemoth. À mesurer en E.1.

---

## 2. Drapeaux (canal B, liste fermée)

| Bit | Nom | Vrai quand | Lecteur |
|---|---|---|---|
| 0 (1) | `SUR_CARTE` | la hauteur est connue ET le pixel est dans la zone gardée (§4.3) | web : seuls ces pixels se teintent ; masque de la coupe |
| 1 (2) | `SOL_MAILLAGE` | la hauteur vient du maillage et non de la géométrie (D-6, et option b du §4.2) | web : compte la part ; voir question 5 du §9 |
| 2 à 7 | réservés | toujours 0 | le décodeur web refuse une image qui en porte un (§6.1) |

**Pas de bit « toit au-dessus d'un sol », ni rien d'autre pour la coupe.** Raison mesurée : 8,2 à 12,5 % du
sol porte **deux sols ou plus** l'un sur l'autre (Recharge 12,5 %, Streets 9,1 %, Behemoth 8,2 %, Lattice
11,6 % ; jamais plus de trois). Sur ces pixels, « il y a un toit » dépend du sol que la coupe regarde : le
sol du haut est le toit de celui du bas. Un bit par pixel ne dit pas lequel ; la coupe compare donc le dessus
à chaque sol de la pile (§6.3), et la pile se reconstruit dans le navigateur en 18 à 51 ms à partir de
`floors`. Un bit `TOIT` serait une donnée dérivée sans lecteur (règle 18).

Pour mémoire, ce qu'auraient valu ces bits (part de la zone) : le dessus est un sol praticable 56,1 % /
32,3 % / 62,9 % / 56,9 % ; un sol à plus de 1,0 u sous le dessus 24,5 % / 30,9 % / 20,0 % / 2,4 % ; un sol à
plus de 0,3 u **au-dessus** du dessus 9,5 % / 19,1 % / 14,0 % / 28,4 %.

---

## 3. Algorithme des traits (Go, item A.2)

Entrées, toutes dans la grille du cadre publié (largeur `W`, hauteur `H`, pixel `p = j × W + i`) :
`connu[p]`, `h[p]` (hauteur du dessus, §4.2), `zone[p]` (`SUR_CARTE`), `hl[p]` (hauteur lissée : moyenne
sur le carré 5 × 5 des seuls pixels connus). Coordonnées des traits : indices de pixel, l'entier au centre.

### 3.1 Ruptures : les pas du treillis

La maquette traçait, par cellule, une iso-ligne à un seuil **propre à la cellule** : deux cellules voisines
ne posaient pas leur extrémité au même endroit, donc les segments ne se chaînaient pas. La forme retenue
pose les nœuds sur le treillis, où ils sont partagés par construction.

```
pasH[p] = connu[p] et connu[p+1] et (zone[p] ou zone[p+1]) et |h[p] − h[p+1]| > SeuilRupture   // entre (i,j) et (i+1,j)
pasV[p] = connu[p] et connu[p+W] et (zone[p] ou zone[p+W]) et |h[p] − h[p+W]| > SeuilRupture   // entre (i,j) et (i,j+1)

Nœuds (identifiant entier, position) :
  milieu du pas horizontal de p : 2p        (i + 0,5 ; j)
  milieu du pas vertical de p   : 2p + 1    (i ; j + 0,5)
  centre de la cellule de p     : 2WH + p   (i + 0,5 ; j + 0,5)

Pour chaque cellule (i < W−1, j < H−1) :
  haut = pasH[p] ; bas = pasH[p+W] ; gauche = pasV[p] ; droite = pasV[p+1] ; n = leur nombre
  n ≤ 1 : rien
  n = 2 : un segment entre les deux milieux
  n ≥ 3 : un segment de chaque milieu au centre de la cellule (jonction)
```

Équivalence avec la maquette, mesurée en cellules de rupture : 100,6 % / 100,2 % / 97,2 % / 100,4 %.
Comptes : 50 254 / 76 363 / 52 009 / 58 591 segments, 1 335 / 3 830 / 1 963 / 2 801 jonctions.

### 3.2 Courbes de niveau : marche des carrés sur les hauteurs lissées

```
Pour chaque cellule de coins a=(i,j), b=(i+1,j), c=(i+1,j+1), d=(i,j+1), éligible si :
  les 4 coins sont connus, l'un au moins est dans la zone,
  aucun de ses 4 pas n'est une rupture,
  pente ≥ PenteMinCourbe, avec pente = hypot((b−a+c−d)/2, (d−a+c−b)/2) / mpp   (valeurs de hl)
Pour chaque niveau t = k × Equidistance avec min(a,b,c,d) < t < max(a,b,c,d), k entier :
  indice = 8·(a > t) + 4·(b > t) + 2·(c > t) + 1·(d > t)
  CAS = [[], [3,2], [2,1], [3,1], [0,1], [0,1,3,2], [0,2], [0,3], [0,3], [0,2], [0,3,2,1], [0,1], [3,1], [2,1], [3,2], []]
  arêtes : 0 = haut (a-b), 1 = droite (b-c), 2 = bas (d-c), 3 = gauche (a-d) ; chaque paire de CAS[indice] est un segment
  nœud sur l'arête (u, v) de valeurs hl : identifiant (arête du treillis, k) ; position à r = (t − u) / (v − u) du premier pixel
```

L'identifiant d'un nœud est le couple (arête de treillis, niveau) : deux cellules voisines le calculent à
partir des mêmes deux pixels, dans le même ordre, donc au même flottant. La pente minimale sert à une seule
chose : un sol plat posé exactement sur un niveau (le rez-de-chaussée de Recharge est à 0,00) ne doit pas
tracer le bruit de sa surface.

### 3.3 Liseré : le bord de la zone

Marche des carrés sur `zone` (0 hors du cadre), cellules de `i = −1` à `W−1`, `j = −1` à `H−1`, nœuds au
**milieu** des arêtes de treillis où `zone` change. Une cellule a 0, 2 ou 4 arêtes qui changent ; à 4 (deux
coins opposés dans la zone), les deux segments entourent chacun son coin (la zone ne se relie pas en
diagonale). Les chaînes sont toujours des anneaux fermés (vérifié : 42 / 16 / 278 / 46 chaînes, toutes fermées).

Avant les traits, **les miettes de zone s'éteignent** : une composante de `zone` (4-connexité) de moins de
`AireMinAnneau` perd son bit `SUR_CARTE`. Un trou de la zone de moins de cette aire reste un trou, sans anneau.

### 3.4 Chaînage, miettes, simplification

```
Chaînage : graphe (nœuds, segments). Les nœuds sont parcourus PAR IDENTIFIANT CROISSANT (jamais dans l'ordre
d'une map Go : la sortie doit être identique d'une exécution à l'autre).
  1. de chaque nœud de degré ≠ 2 (bout, jonction), pour chaque segment non visité : suivre les nœuds de
     degré 2 jusqu'à un nœud de degré ≠ 2 ou au retour au départ ;
  2. les segments restants forment des boucles : partir du plus petit nœud non visité, tourner jusqu'au retour.
Miettes :
  ruptures : COMPOSANTE CONNEXE (toutes ses chaînes) de longueur totale < LongueurMinRupture → ôtée ;
  courbes  : CHAÎNE de longueur < LongueurMinCourbe → ôtée ;
  liseré   : anneau d'aire < AireMinAnneau → ôté.
Simplification : Douglas-Peucker itératif (pile, pas de récursion), tolérance ToleranceTraits en pixels,
  extrémités gardées ; une chaîne fermée est coupée en deux au point le plus loin du premier, gardé lui aussi.
Ordre de sortie : polylignes triées par (y0, x0) après encodage.
```

Pourquoi la composante et non la chaîne pour les ruptures : une chaîne s'arrête à chaque jonction, et un
mur fait de dix chaînes courtes entre onze jonctions n'est pas une miette. Mesuré : 76 à 85 % des **chaînes**
de rupture font moins de 0,5 u et portent 15 à 31 % de la longueur ; 48 à 78 % des **composantes** font
moins de 0,5 u et n'en portent que 1,6 à 6,1 %. Pour les courbes, la chaîne est la bonne maille : il n'y a
pas de jonction, et les chaînes courtes (82 à 87 % du nombre, 11 à 25 % de la longueur) sont le
clignotement de la pente autour de son seuil.

| Réglage | Recharge | Streets | Behemoth | Lattice |
|---|---|---|---|---|
| Sans filtre, tolérance 0 : JSON des traits / gzip | 864 / 158 | 1 332 / 251 | 1 651 / 283 | 1 004 / 214 |
| Sans filtre, tolérance 0,75 px | 331 / 107 | 553 / 173 | 516 / 169 | 442 / 146 |
| **Retenu : 0,75 px, rupture ≥ 0,5 u (composante), courbe ≥ 0,5 u, anneau ≥ 0,05 u²** | **125 / 41** | **254 / 77** | **224 / 73** | **205 / 66** |
| idem, tolérance 0,5 px | 145 / 45 | 272 / 82 | 256 / 81 | 224 / 71 |
| idem, tolérance 1,0 px | 113 / 37 | 228 / 69 | 203 / 66 | 188 / 60 |
| idem, courbe ≥ 1,0 u | 107 / 34 | 210 / 61 | 192 / 62 | 165 / 51 |

Les trois dernières lignes ont été mesurées avec une composante de rupture d'au moins 1 u, ce qui pèse 0,7 à
2,7 % de moins que 0,5 u. Polylignes et points retenus : ruptures 2 315 / 6 737, 6 281 / 16 496, 3 810 / 9 942, 4 883 / 13 024 ;
courbes 1 652 / 8 851, 3 336 / 15 488, 3 467 / 17 338, 2 836 / 12 954 ; liseré 19 / 442, 15 / 353,
57 / 1 029, 14 / 347.

La tolérance est en **pixels** parce qu'elle efface un défaut de trame : les nœuds tombent sur des
demi-pixels, et un bord oblique dessine des marches d'un demi-pixel au plus. 0,75 px les efface et vaut
0,022 à 0,037 u selon la carte, soit 7 à 11 cm réels d'écart au plus entre le trait et le bord qu'il suit.

### 3.5 Encodage

Positions arrondies au dixième de pixel **avant** de faire la différence (`dx = arrondi(10·x_k) −
arrondi(10·x_{k−1})`) : aucune dérive. Un point confondu avec le précédent après arrondi est sauté. Les
nœuds de rupture et de liseré tombent sur des demi-pixels (multiples de 5 dixièmes) : exacts.

### 3.6 Pas de hauteur par trait : la coupe filtre par le masque

Le plan proposait une hauteur basse et une haute par polyligne, que le web comparerait à la bande de la
coupe. Mesuré contre la règle de la maquette (un segment est gardé si la coupe garde le pixel de son milieu,
ou, pour une rupture, un pixel à 1,5 px de part et d'autre) :

| Coupe à | part de la longueur des ruptures gardée par le masque | par les hauteurs basse / haute | accord des deux |
|---|---|---|---|
| Recharge 0,00 / 2,21 / 4,40 | 19,0 / 72,6 / 48,5 % | 14,2 / 44,1 / 38,2 % | 94,4 / 71,2 / 87,3 % |
| Streets 0,51 / 2,31 / 4,45 | 43,7 / 58,4 / 37,4 % | 25,4 / 34,5 / 31,7 % | 81,3 / 75,3 / 92,3 % |
| Behemoth 3,76 / 8,53 | 20,9 / 64,5 % | 0,6 / 45,2 % | 79,6 / 77,7 % |
| Lattice 50,03 / 51,80 / 54,09 | 41,4 / 80,0 / 28,7 % | 38,5 / 63,9 / 19,8 % | 95,1 / 82,7 / 90,3 % |

La règle par hauteurs perd ce qui tient **sur** le sol de l'étage (un pilier, une caisse, un mur dont le
pied n'est pas au niveau du sol), que la maquette validée garde. Et une chaîne ne porte pas une hauteur :
le long d'une même chaîne de Behemoth, la hauteur basse varie de plus de 0,5 u dans 39 % des cas. Le fichier
ne porte donc aucune hauteur par trait ; le filtre vit dans le web (§6.3).

Le filtre s'applique aux segments **simplifiés**. Comparé segment brut par segment brut : accord de 96,5 à
98,5 % (ruptures) et 96,8 à 99,2 % (courbes), sauf Behemoth à 8,53 u : 92,9 % et 89,4 %, à cause de ses longs
segments. Le web échantillonne donc le masque par tronçons d'au plus **0,5 u** le long de chaque segment
(avec des segments bornés à 1 u, l'accord remonte à 97,1 % au moins partout). Le fichier ne borne pas la
longueur des segments : cela coûterait 4 à 16 % de JSON pour ce que le web fait sans rien stocker.

### 3.7 Tests attendus (A.4), en plus de ceux du plan

- une marche verticale entre les colonnes 10 et 11 donne un trait à `u = 105` (le bord entre les deux
  pixels), pas à 100 ni à 110 : c'est le test de la convention du demi-pixel ;
- une jonction en T donne trois chaînes qui partagent le centre de la cellule ;
- un carré de zone donne un anneau fermé, sans point répété, simplifié à ses 4 coins (ou 8 si chanfreinés) ;
- deux exécutions sur la même entrée donnent deux sorties identiques à l'octet ;
- un sol plat à 2,00 u exactement, bruité de ± 0,01 u, ne trace aucune courbe.

---

## 4. Seuils (unités du jeu)

### 4.1 Table

| Seuil | Valeur | Côté | Mesure |
|---|---|---|---|
| Pas de hauteur | 1/256 u | Go | §1.1 |
| Fusion des sols empilés | 0,4 u | Go, web (par le JSON) | plus petit écart réel entre deux sols d'un même pixel : 1,0 u (Lattice), 0,9 u (Behemoth, 4 pixels, puis 1,1 u), 1,4 u (Recharge), 1,7 u (Streets) ; un seul pixel à 0,3 u sur 2,8 millions. De 0,4 à 0,6 u la pile est identique ; à 1,0 u, 4 pixels changent |
| Couches de la pile | 4 | Go, web | maximum mesuré : 3 (2 / 3 848 / 4 032 / 368 pixels) ; au-delà, `pile-tronquee` |
| Tolérance dessus / sol | 0,3 u | Go, web (par le JSON) | écart dessus − sol le plus haut : < 0,1 u pour 38 à 55 % des pixels, 0,1 à 0,3 u pour 0,6 à 13 %, 0,3 à 0,4 u pour 0,1 à 3,7 % |
| Marge de la zone | 0,5 u (`himap.MargeNavmesh`) | Go | marge 0,25 u : la zone perd 0 à 4 % ; 1,0 u : rien ne change (le rendu est déjà rogné à 0,5) |
| Fermeture des passages | 0,5 u | Go | sans effet mesurable sur les 4 témoins (0,0 % entre 0 et 1,0 u), tous rognés au maillage ; à juger en E.1 sur les Forge non rognées |
| Rupture | 0,35 u entre deux pixels voisins | Go | longueur à 0,25 / 0,35 / 0,5 u : 1 399 / 1 322 / 1 093 (Recharge), 2 519 / 2 367 / 2 190, 2 509 / 2 151 / 1 827, 2 386 / 2 176 / 1 959. Recharge a un creux d'histogramme entre 0,2 et 0,3 u, les trois autres décroissent sans creux : la mesure ne désigne pas mieux que la valeur validée à l'écran |
| Équidistance des courbes | 0,5 u (un quart d'étage) | Go | longueur proportionnelle : 0,25 u double, 1,0 u divise par deux |
| Pente minimale d'une courbe | 0,08 | Go | de 0,04 à 0,2 la longueur ne bouge que de 1 % : le seuil n'écarte que le bruit des sols plats |
| Lissage des hauteurs | carré 5 × 5 pixels | Go (courbes), web (ombrage) | repris de la maquette et de Den |
| Tolérance des traits | 0,75 px | Go | §3.4 |
| Miettes : rupture, courbe, anneau | 0,5 u, 0,5 u, 0,05 u² | Go | §3.4 ; 23 / 1 / 221 / 32 anneaux sous 0,05 u² |
| Étages : pas d'histogramme | 0,25 u | Go | à 0,1 u Streets gagne un faux étage, à 0,5 u il en perd un |
| Étages : part minimale | 3 % des échantillons de sol (compte lissé) | Go | de 2 à 8 % rien ne change sauf sur Behemoth |
| Étages : écart minimal | **1,4 u** | Go | §4.5 |
| Étages : nombre maximal | 5 | Go | 3, 3, 2, 3 trouvés |
| Bande d'étage, fondu | ± 0,7 u, 0,25 u | web | part du sol à ± 0,7 u d'un étage : 90,8 / 87,6 / 74,8 / 92,7 % ; à ± 0,35 u : 84,7 / 53,2 / 47,1 / 73,9 % |
| Opacité sous / sur la coupe | 0,28 / 0,10 | web | valeurs de Den, celles de la maquette |
| Proche d'un sol de la coupe | 0,25 u | web | maquette |
| Toit | 1,0 u au-dessus d'un sol | web | entre 0,6 et 1,0 u il n'y a presque rien (0 à 2,3 % du sol) ; 1,0 u dépasse le sommet d'un saut (0,82 à 0,89 u, §8) |
| Suivi : appui, seuil, délai | 0,35 u, 0,5 u, 300 ms | web | §8 |

Les seuils de la coupe restent côté web : ils règlent une vue, pas une donnée, et la maquette des commandes
(D.1) peut encore les bouger sans recuire 114 cartes.

### 4.2 Le « dessus » : hauteur du rendu, complétée par le maillage

D-6 donne la hauteur du maillage là où la cuisson n'a **aucune** matière : 4,3 % / 0,0 % / 32,2 % / 8,6 % de
la zone (chiffres du plan retrouvés). La mesure montre un second cas, que le plan ne voyait pas : la cuisson
a de la matière, mais **plus bas qu'un sol praticable**.

| Part de la zone où un sol du maillage est à plus de 0,3 u AU-DESSUS du dessus du rendu | Recharge | Streets | Behemoth | Lattice |
|---|---|---|---|---|
| | 9,5 % | 19,1 % | 14,0 % | 28,4 % |

Par étage, la part du sol que le rendu montre plus bas que lui : Recharge 0,8 / 7,3 / **36,4 %** ; Streets
**25,2** / 14,9 / 13,3 % ; Behemoth 0,0 / 12,0 % ; Lattice 8,2 / 36,9 / **59,2 %**. La cause est la même que
pour D-6 : il manque de la géométrie à la hauteur où l'on marche (ou la cuisson a choisi la surface du
dessous, par substitution ou écrêtage), et le tampon de profondeur montre ce qu'il y a en dessous.

Deux définitions possibles :

- **(a) la maquette** : dessus = hauteur du rendu ; maillage seulement sans matière (D-6 à la lettre).
- **(b) recommandée** : en plus, quand le sol le plus haut du maillage dépasse le dessus du rendu de plus de
  la tolérance (0,3 u), il **devient** le dessus, et le pixel porte `SOL_MAILLAGE`. Autrement dit : le dessus
  n'est jamais sous un sol praticable.

| | Recharge | Streets | Behemoth | Lattice |
|---|---|---|---|---|
| Hauteur tirée du maillage, (a) → (b) | 4,3 → 13,8 % | 0,0 → 19,1 % | 32,2 → 46,3 % | 8,6 → 37,0 % |
| Longueur des ruptures, (a) → (b) | 1 322 → 1 172 u | 2 367 → 2 026 u | 2 151 → 1 275 u | 2 176 → 1 479 u |
| Jonctions | 1 335 → 919 | 3 830 → 3 362 | 1 963 → 838 | 2 801 → 1 828 |
| Sol de l'étage montré par la coupe de la maquette, (a) | 99,5 / 93,0 / 76,4 % | 69,1 / 83,5 / 88,4 % | 100 / 92,3 % | 99,4 / 72,5 / 42,6 % |
| idem, (b) | 99,5 / 100 / 99,9 % | 99,9 / 99,6 / 100 % | 100 / 98,2 % | 99,8 / 99,7 / 99,8 % |

Avec (b), « Tout » montre les passerelles et les étages hauts (comme le fait l'image Plan, qui garde le sol
du dessus), les traits perdent 11 à 41 % de bruit, et la coupe de la maquette devient juste sans retouche.
Avec (a), « Tout » reste exactement ce que le user a validé, mais 59 % des plates-formes hautes de Lattice
n'y figurent pas. Aperçus teintés des deux, à moitié de la taille, vue « Tout », pour les 4 cartes :
`C:\Users\GUILLA~1\AppData\Local\Temp\claude\17871fa88c29004d\63a04594-e948-4485-9732-90ff5a088fa7\scratchpad\conception\sortie`
(`<clé>.a.apercu.png`, `<clé>.b.apercu.png`, avec les `relief.webp` et `relief.json` au format retenu). Sur
Lattice la bande sombre en diagonale de (a) est la tranchée que le rendu montre sous les passerelles ; sur
Streets les rectangles sombres de (a) sont des surfaces sous le niveau de la rue. **C'est la question 1 du
§9.** Le format est le même dans les deux cas : le choix est une constante de A.1.

### 4.3 Zone gardée

`zone` = hauteur connue ET dans : sols du maillage gardés, élargis de la marge, passages plus étroits que
deux fois la fermeture refermés, creux enclos comblés (ce que le bord de la grille n'atteint pas). Les faces
gardées sont celles de `ComposantesRetenues` avec la matière du **fond publié** (`matiereDeImage`), comme
`--plan-seul` : le relief et le plan des murs publiés tiennent alors aux mêmes composantes. La brique a déjà
les briques : `couvreTriangles`, `dilate`, `remplitTrous`.

### 4.4 Coupe : un sol de la coupe se montre toujours

La coupe de la maquette ne sait montrer un sol caché que s'il est sous un toit (dessus > sol + 1,0 u). Quand
le dessus est **plus bas** que le sol (cas du §4.2 sous la définition a), elle affiche le dessous en pâle :
sur l'étage haut de Lattice, 43 % seulement du sol paraît ; 69 % au niveau de la rue de Streets. La règle retenue (§6.3) traite les deux cas de la même façon,
et montre 98,1 à 100 % du sol sous les deux définitions. Elle est inoffensive sous (b), où le second cas ne
se présente plus.

### 4.5 Étages

```
échantillons = une hauteur par pixel et par sol empilé (pondération par l'aire, en pixels)
bornes : P1 et P99 des échantillons (floors.min, floors.max)
histogramme au pas de 0,25 u depuis P1 ; lissage [0,5 ; 1 ; 0,5]
pic = case ≥ sa voisine de gauche, > sa voisine de droite, et > 3 % du nombre d'échantillons
pics par poids décroissant : on garde un pic s'il est à plus de 1,4 u de tous ceux déjà gardés, 5 au plus
hauteur de l'étage = MÉDIANE des échantillons à ± 0,25 u du pic (et non le centre de la case)
```

| | Pics (centres de case) | Étages (médiane) |
|---|---|---|
| Recharge | −0,01 ; 2,24 ; 4,49 | 0,00 ; 2,21 ; 4,40 |
| Streets | 0,49 ; 2,24 ; 4,49 | 0,51 ; 2,31 ; 4,45 |
| Behemoth | 3,73 ; 8,48 ; (9,73 écarté) | 3,76 ; 8,53 |
| Lattice | 50,00 ; 51,75 ; 54,00 | 50,03 ; 51,80 ; 54,09 |

L'écart passe de 1,2 à 1,4 u : deux étages plus proches que deux fois la bande montrent la même chose. Seul
Behemoth change (8,53 et 9,72 étaient à 1,19 u l'un de l'autre). Les trois autres cartes ont leurs étages à
1,77 u ou plus. Sur un terrain naturel, les « étages » sont des paliers de fréquence, pas des planchers :
75 % seulement du sol de Behemoth est à ± 0,7 u d'un étage. Le curseur reste la commande juste sur ces cartes.

---

## 5. Projection dans le calage publié (D-5)

### 5.1 Le constat, mesuré

| | Cadre publié | Grille recuite | Décalage | Pixels du cadre hors grille |
|---|---|---|---|---|
| Recharge | 1605 × 1771 | 1605 × 1736 | 0 px en X, **35 px** en Y, même échelle | 56 175 (2,0 %) |
| Streets, Behemoth, Lattice | identiques à la grille | | 0 | 0 |

Sur Recharge, les 35 lignes manquantes sont en haut ; le fond publié comme l'image plan y sont **vides**
(0 pixel de matière), et le maillage n'y pose aucun sol. C'est de la marge.

### 5.2 Échantillonnage

Tout le relief (hauteurs, zone, traits) se calcule **dans la grille du cadre publié**. Le rendu recuit n'est
lu qu'une fois, au plus proche voisin, au centre du pixel publié :

```
pour chaque pixel (px, py) du cadre publié :
  x = OrigineX + (px + 0,5) × mpp ;  y = OrigineY − (py + 0,5) × mpp
  i = plancher((x − r.Min[0]) / r.Cell) ;  j = plancher((y − r.Min[1]) / r.Cell)     // le j du rendu monte avec y
  z, ok = r.Altitude(i, j)                                                            // faux hors de la grille
```

Jamais d'interpolation : une hauteur moyennée de part et d'autre d'un mur invente une pente.

### 5.3 Les cas

| Cas | Règle |
|---|---|
| Pixel du cadre publié hors de la grille recuite | pas de hauteur du rendu ; sol du maillage s'il y en a un (D-6), sinon hauteur inconnue (A = 0). Compté au journal |
| Matière recuite hors du cadre publié | perdue (le cadre publié est la référence). Comptée au journal ; au-delà de 1 % de la matière : avertissement |
| Même échelle, décalage entier | copie exacte (cas des 4 témoins) |
| Même échelle, décalage non entier | plus proche voisin ; erreur d'au plus un demi-pixel (0,015 à 0,025 u). Le décalage est écrit au journal. Arrive quand les ancres d'objectifs, donc `Rendu.Min`, ont bougé depuis la publication |
| Échelle différente (écart relatif > 10⁻⁶) | **erreur de cette carte**, rien n'est écrit : « échelle du fond publié différente de la cuisson, recuire le fond ». Échantillonner une grille fine dans une grossière fabriquerait des marches, donc de fausses courbes |
| Sidecar ou fond publié absent | carte laissée telle quelle, comme `--plan-seul` |

### 5.4 Le relief ne doit pas survivre à son fond

`relief.json` recopie le calage auquel il a été projeté. Deux garde-fous en découlent :
- une cuisson complète (`publie`) qui change le fond **refait le relief** dans le même geste (elle a le rendu
  en mémoire), ou à défaut retire `reliefImage` et `reliefData` du sidecar, comme `poseLePlan` remet le plan à zéro ;
- le service (C.2) compare le calage du JSON à celui du sidecar ; s'ils diffèrent, il journalise et répond
  comme pour un relief absent. Le web retombe alors sur le Plan, au lieu de dessiner un relief décalé.

### 5.5 Dérive entre le fond publié et la recuisson

Le fond publié porte de la matière que la recuisson d'aujourd'hui n'a pas, et jamais l'inverse :

| | Recharge | Streets | Behemoth | Lattice |
|---|---|---|---|---|
| Matière du fond publié absente de la recuisson | 10,85 % | 8,66 % | 2,23 % | 1,40 % |
| Pixels du fond publié sans hauteur dans le relief (après D-6) | 11,9 % | 8,7 % | 4,7 % | 1,4 % |

La cause n'est pas établie. Deux pistes : les aplats de sol supposé, qui se peignent sans altitude
(`solSuppose`, `z` à −∞ : `Rendu.Altitude` rend faux), et un écart entre la brique du 13 et du 14 novembre et
la v1.3.1 (les statistiques de Recharge diffèrent : 546 603 cellules substituées publiées, 523 472 recuites).
Conséquence : sur ces pixels, le fond Relief a un trou là où le Rendu montre un aplat. C'est déjà ce que
montrait la maquette. À instruire en B.1 : le journal de `--relief-seul` compte « pixels du fond publié sans
hauteur » par carte, et le test du geste fixe la valeur des témoins.

---

## 6. Pièges côté web

### 6.1 Lire l'image sans que le navigateur y touche

```ts
const blob = await api.getBlob(chemin)          // lib/api seul fait un fetch (garde rawFetch)
const bitmap = await createImageBitmap(blob, { colorSpaceConversion: 'none', premultiplyAlpha: 'none' })
const ctx = canvas.getContext('2d', { willReadFrequently: true, colorSpace: 'srgb' })
ctx.imageSmoothingEnabled = false
ctx.drawImage(bitmap, 0, 0)                     // taille 1:1, jamais de mise à l'échelle
const pixels = ctx.getImageData(0, 0, w, h).data
```

- Le canevas range ses pixels prémultipliés : c'est pour cela que A ne vaut que 0 ou 255. Avec A = 255 la
  relecture est exacte ; avec A = 0 le pixel revient à zéro, ce que le format impose déjà.
- **Contrôle à la lecture, obligatoire** : sur tous les pixels, A ∈ {0, 255} ; B ≤ 3 ; A = 0 ⇒ R = G = B = 0 ;
  dimensions égales à celles du calage. Un seul manquement : le relief est tenu pour illisible, journalisé,
  et la vue retombe sur le Plan. C'est le seul filet contre un navigateur qui aurait converti les couleurs.
- Si `createImageBitmap` refuse les options (vieux WebKit) : repli sur l'élément image (`loadedImage.ts`),
  avec le même contrôle. Une image WebP sans profil est traitée en sRGB, sans conversion.
- jsdom n'a pas `createImageBitmap` : la logique testée (vitest) prend un tableau de pixels, pas une image.
- Ne jamais lire l'image par `<img>` dessinée à une autre taille, ni par un canevas dont le lissage est actif.
- La hauteur d'un point de piste absente du JSON vaut **zéro** (`z` est `omitempty` côté Go) : `z ?? 0`.
  Le rez-de-chaussée de Recharge est à 0,00.

### 6.2 Le calcul va dans un Web Worker

Temps mesurés dans Node sur le poste de mesure, Behemoth ou Recharge (2,84 millions de pixels) :

| Étape | Temps | Quand |
|---|---|---|
| Ombrage (lissage 5 × 5 et gradient) | 46 à 93 ms | à l'ouverture du fond Relief, et à chaque changement d'orientation |
| Empilement des sols (faces → pile par pixel) | 18 à 51 ms | **à la première coupe seulement** |
| Une coupe | 24 à 61 ms | à chaque nouvelle hauteur |

Rien de cela ne fait une seconde sur un poste de bureau. Aucun téléphone n'a été mesuré ; il faut s'attendre
à plusieurs fois ces temps, d'où le worker, qui garde la lecture fluide pendant le calcul. L'application n'a aucun worker aujourd'hui : c'est le premier
(`new Worker(new URL('./relief.worker.ts', import.meta.url), { type: 'module' })`). La politique de sécurité
du serveur ne porte aucune directive de chargement (`minimalCSP`), donc rien ne le bloque ; la coquille
Tauri est un Chromium.

- Le worker n'est qu'une enveloppe de messages. Décodage, ombrage, pile, coupe, filtre des traits et sol
  porteur sont des **fonctions pures** de `lib/replay/relief/`, testées sans worker.
- « Tout » n'a besoin ni de la pile ni des faces : la pile se construit à la première coupe.
- Mémoire : hauteurs 11 Mo, pile en `Int16` au pas du fichier × 4 couches 23 Mo, sorties d'une coupe en
  `Uint8` 9 Mo, image teintée 11 Mo. Ne pas garder la pile en `Float32` (45 Mo) ni plus de 4 coupes en cache.
- Les tableaux reviennent par transfert (`postMessage(…, [buffer])`), pas par copie.
- Le curseur : une seule coupe en calcul, la dernière demande gagne ; cache des coupes par hauteur arrondie
  au quart d'unité.
- La teinte (table de 256 couleurs tirée du jeton `map-relief-base`) s'applique dans le worker ; un changement
  de palette redemande la teinte, pas la coupe.

### 6.3 La coupe (fonction pure, à la hauteur `z`)

```
poids(v) = 1 dans [z − BANDE ; z + BANDE], décroît linéairement jusqu'à 0 sur FONDU au-delà
pour chaque pixel : meilleur = le sol de la pile de plus grand poids (aucun si tous sont à 0)
                    solDansCoupe = poids(meilleur) > 0,5
pres = solDansCoupe dilaté de PRES (rayon en pixels = arrondi(PRES / mpp), 1 au moins)
pour chaque pixel SUR_CARTE, de dessus d :
  si meilleur existe et (d > meilleur + TOIT  ou  d < meilleur − TOLERANCE) :
      // un sol de la coupe caché par un toit, ou que le fond a rendu plus bas : on le montre à plat
      dedans = poids(meilleur) ; gris = mélange(gris du pixel, gris de la hauteur de meilleur, dedans) ; garde = 0
  sinon :
      dessusEstUnSol = un sol de la pile est à moins de TOLERANCE de d
      dedans = (dessusEstUnSol ou non pres ou d < z) ? poids(d) : 1     // ce qui tient sur un sol de la coupe y est
      garde = dedans > 0,5
  alpha = dedans + (1 − dedans) × (d > z ? OPACITE_DESSUS : OPACITE_DESSOUS)
```

- **Fond Relief** : gris teinté, opacité `alpha`.
- **Fond Rendu** (D-2) : masque d'opacité = `alpha` ; un pixel du rendu **sans hauteur** dans le relief (4,7 à
  11,9 % du rendu, §5.5) prend `OPACITE_DESSOUS` dès qu'une coupe est active, sinon il resterait seul en clair.
- **Fond Plan** (D-2) : l'image Plan montre le sol le plus haut de chaque pixel ; son masque se calcule donc
  sur **ce sol** (`poids` du sol le plus haut de la pile), pas sur le dessus du relief. Sans sol (traits
  d'arête de l'image) : l'`alpha` du relief. Limite assumée : sous un étage, le Plan ne montre pas le sol
  recouvert, il n'en a pas l'image ; seul le Relief le révèle.
- Les masques se composent comme `rognerAuPlan` (`lib/replay/planMask`) : même rectangle, même rotation que
  le fond.
- **Traits** : chaque segment est découpé en tronçons d'au plus 0,5 u ; un tronçon de courbe est tracé si
  `garde` vaut 1 au pixel de son milieu ; un tronçon de rupture, si `garde` vaut 1 à son milieu ou à 1,5 px
  de part et d'autre de sa normale. Le liseré n'est jamais filtré. Les murs du maillage (arêtes de `floors`
  portées par une seule face, hauteur moyenne dans la bande) se tracent sous une coupe, comme dans la maquette.

### 6.4 Orientation par quarts de tour (ADR 0048 D-4)

- La rotation se fait dans le monde, avant la projection. Le relief teinté et les masques sont des images au
  calage du fond : elles passent par `drawMapBackground` telles quelles (`rotateCanvasAbout`). Ne réécrire
  aucune rotation : `orientation.guard.test.ts` l'interdit.
- Les traits se convertissent en monde (formule du §1.2) puis passent par la projection de la vue
  (`projectTo`), comme `strokeWallPlan`. Leur épaisseur est en pixels d'écran, pas en unités.
- Le filtre des traits lit `garde` dans le repère de l'image, **avant** toute rotation.
- **Lumière.** Une carte tournée de 180° serait éclairée par le bas à droite : le relief s'inverse à l'œil
  (les bosses se lisent en creux). L'ombrage se calcule dans le navigateur, donc la lumière peut rester en
  haut à gauche de l'**écran** : direction dans le repère de l'image, pour `q` quarts de tour horaires,
  `(−1 ; −1)`, `(−1 ; +1)`, `(+1 ; +1)`, `(+1 ; −1)` pour `q` = 0, 1, 2, 3 (x vers la droite, y vers le bas).
  Coût : un ombrage (46 à 93 ms) par changement d'orientation, mis en cache par orientation. Question 4 du §9.
- La demi-unité : le pixel `(px, py)` a son centre en `originX + (px + 0,5) × mpp`. Un trait décalé d'un
  demi-pixel se voit au zoom maximal contre le bord teinté du relief.

---

## 7. Calques de la Tactique : lesquels portent une hauteur (D-2)

| Calque | Fichier de dessin | Donnée | Champ de hauteur | Filtré par étage |
|---|---|---|---|---|
| Zones nommées (contours et noms) | `planPaint.ts`, `lib/replay/calloutsPaint.ts` | `TacticalRaster.zones[]` (`CalloutZone`) | `z`, `z_bottom`, `z_top` | **oui** : la zone reste si `[z_bottom ; z_top]` croise la bande de la coupe |
| Réapparitions adverses : teinte par zone et étiquettes | `TacticalReapparitionsZones.tsx`, `TacticalReapparitionsPlan.tsx` | les mêmes zones (`CalloutZone`), parts par `TacticalPartDeZone.volume_index` | `z_bottom`, `z_top` de la zone | **oui**, même règle |
| Chaleur lissée | `planPaint.ts`, `lib/replay/chaleurLissee.ts` | `TacticalRaster.cellules[]` (`CelluleTactique`) | aucun (`centre_x`, `centre_y`) | non |
| Plan des murs | `lib/replay/wallPlanPaint.ts` | `MapBackgroundPlan.edges` `[x1, y1, x2, y2]` | aucun | non |
| Tueurs et victimes | `tueursPaint.ts`, `TacticalPlanTueurs.tsx` | `TacticalContribution.point`, `autre_point` (`TacticalPoint`) | aucun (`x`, `y`) | non |
| Coéquipiers à la mort, cercle du radar | `coequipiersPaint.ts`, `TacticalPlanCoequipiers.tsx` | `TacticalCoequipier.point`, `TacticalRaster.rayons_radar_m` | aucun | non |
| Séquence avant la mort (traînes) | `coequipiersPaint.ts` (`peindreLaScene`), scène de `mort.logic.ts` | `TacticalSequenceActeur.points[]`, `TacticalSequenceEtape.centre`, `TacticalSequence.mort` | aucun (`x`, `y`) | non |
| Grappes de réapparition | `TacticalPlanVies.tsx` : une pilule de filtre, rien n'est dessiné sur le plan | `TacticalRaster.grappes[]` (`TacticalGrappe`) | aucun (`x`, `y`) | sans objet |
| Anneau et étiquette de la cellule choisie | `TacticalPlanCalque.tsx` | adresse de cellule | aucun | non |

Vérifié sur `apps/web/src/lib/api/generated.ts` et sur les types Go (`domain/tactical*.go`,
`replaydoc.CalloutZone`). Liste fermée : **deux calques filtrés, tous deux par les zones nommées**. Les
autres calques dessinés restent entiers et le disent dans leur légende (D-2). Leur donner une hauteur demanderait de changer
le contrat du serveur : ce n'est pas dans ce plan.

Occupation du terrain : `MatchPositionDTO.z` existe. La hauteur d'une piste est celle des **pieds** (§8),
mais un joueur est en l'air 10 % du temps : filtrer sur `z` seul ferait disparaître les sauts. Une position
reste si son **sol porteur** (§8) a un poids supérieur à 0,5 dans la coupe ; sans sol sous elle (1,3 à 3,2 %
des points), on juge sur `z`.

---

## 8. Suivi de l'étage du joueur ouvert (D-2, D.6)

### 8.1 Où est la hauteur

`Document.tracks[].points[]` porte `t` (numéro d'image, pas de 33 ms), `x`, `y`, `z` ; une piste par vie
(`replaydoc.Track`, `replaydoc.Point`, `internal/domain/replaydoc/document.go`). Les documents se lisent sans
base : `data/cache/replays/halo_infinite/<8 caractères>.json.gz` (`PathResolver.ReplayArtifactPath`). Le
document ne nomme pas sa carte ; les rejeux des témoins ont été reconnus par l'emprise de leurs pistes, puis
confirmés contre le maillage (97 à 99 % des points sur le maillage, 84 à 93 % à moins de 0,6 u d'un sol).

Mesure : 6 rejeux par carte (1 seul sur Lattice : c'est le seul du poste), 1,55 million de points, 1 839 vies,
953 minutes de vie.

### 8.2 Ce que disent les pistes

| | Recharge | Streets | Behemoth | Lattice |
|---|---|---|---|---|
| Hauteur de la piste − sol porteur, P25 / médiane / P75 (u) | −0,01 / 0,00 / 0,02 | −0,01 / 0,00 / 0,03 | −0,02 / 0,01 / 0,05 | 0,00 / 0,01 / 0,05 |
| idem, P90 / P95 / P99 | 0,54 / 1,00 / 2,88 | 0,67 / 1,06 / 2,63 | 0,76 / 1,59 / 5,88 | 0,69 / 0,90 / 2,15 |
| Sommet d'un saut, médiane / P90 (u) | 0,89 / 3,04 | 0,85 / 1,32 | 0,82 / 1,12 | 0,86 / 1,45 |
| Durée d'un saut, médiane / P90 (s) | 1,02 / 1,62 | 1,02 / 1,35 | 1,02 / 1,16 | 1,02 / 1,19 |
| Points hors des pixels du maillage | 2,3 % | 1,3 % | 2,0 % | 3,2 % |

- La hauteur d'une piste est celle des **pieds** : aucun décalage à retrancher.
- Un saut ordinaire monte de **0,82 à 0,89 u** et dure une seconde : plus haut que la bande (0,7 u). Un seuil
  sur la hauteur seule est donc franchi à chaque saut.
- Un joueur est à plus de 0,5 u de son sol 10 % du temps.

### 8.3 Deux règles, jugées sur le temps passé au sol

R1 : la hauteur de la piste seule. R2 : le **sol porteur**, le sol du maillage le plus haut sous les pieds
du joueur à sa position (un sol compte jusqu'à 0,35 u au-dessus des pieds). Pendant un saut le sol porteur
ne bouge pas ; en tombant d'un rebord il devient le sol du bas dès que le joueur passe le bord.

| Règle, seuil, délai | Changements de coupe par minute | dont allers-retours en moins de 3 s | Sol du joueur hors de la bande |
|---|---|---|---|
| R1, 0,7 u, 0 ms | 16,9 / 18,6 / 24,9 / 15,1 | 4,5 / 4,3 / 6,1 / 3,9 | 0,3 / 1,3 / 0,7 / 3,7 % |
| R1, 0,7 u, 600 ms | 7,2 / 10,4 / 12,4 / 7,4 | 0,8 / 1,8 / 1,8 / 1,6 | 5,6 / 9,7 / 11,4 / 6,3 % |
| R1, 1,0 u, 800 ms | 4,6 / 6,0 / 7,8 / 4,1 | 0,3 / 0,9 / 0,6 / 0,5 | 12,8 / 18,2 / 21,5 / 11,8 % |
| R2, 0,5 u, 0 ms | 8,8 / 12,5 / 18,7 / 8,8 | 1,1 / 1,1 / 3,2 / 1,8 | 0,0 % |
| **R2, 0,5 u, 300 ms** | **7,1 / 11,0 / 14,5 / 7,3** | **0,8 / 1,0 / 2,3 / 1,5** | **1,6 / 2,0 / 2,0 / 1,7 %** |
| R2, 0,7 u, 500 ms | 6,3 / 8,5 / 10,5 / 5,8 | 0,6 / 1,2 / 1,5 / 1,1 | 4,3 / 6,6 / 8,3 / 4,2 % |

(Recharge / Streets / Behemoth / Lattice.) À nombre de changements égal, R2 laisse le joueur 3 à 5 fois
moins longtemps hors de sa bande que R1.

### 8.4 Règle retenue

```
solPorteur(x, y, z) = le plus haut des sols du maillage en (x, y) dont la hauteur ≤ z + 0,35 ; aucun hors maillage
   // calcul direct sur floors : boîte englobante puis test point-dans-polygone convexe, hauteur interpolée sur la face ;
   // 440 à 4 072 faces, quelques microsecondes par image, sur le fil principal, sans attendre la pile du worker
cible  = solPorteur ; à défaut le dernier sol porteur connu de cette vie ; à défaut z
accroche(v) = l'étage détecté le plus proche si |v − étage| ≤ 0,7, sinon v arrondi au quart d'unité

à chaque instant lu, pour le joueur ouvert :
  pas de piste vivante (mort, pas encore réapparu)       → la coupe ne bouge pas
  première image d'une vie, ou saut de lecture (clic sur la barre) → coupe = accroche(cible), sans délai
  |cible − coupe| ≤ 0,5                                  → rien ; la minuterie retombe
  sinon                                                  → la minuterie part ; au bout de 300 ms d'écart ininterrompu,
                                                           coupe = accroche(cible)
```

Seuil **0,5 u**, délai **300 ms**, appui **0,35 u**. Sur une rampe entre deux étages, la coupe passe par un
ou deux paliers intermédiaires au quart d'unité : le sol sous le joueur reste dans la bande. Les marqueurs
des joueurs ne sont jamais masqués par la coupe dans le rejeu : seul le fond change.

Tests de D.6 (logique pure) : marche à plat (aucun changement), saut sur place de 0,9 u (aucun), chute d'un
étage (un changement, 300 ms après le passage du bord), rampe de 2,2 u (deux à trois changements, jamais
d'aller-retour), mort puis réapparition à un autre étage (un changement, immédiat), point hors maillage
(la coupe tient), `z` absent (zéro).

---

## 9. Risques, et questions pour le user

### Questions

1. **Quel « dessus » dans la vue « Tout » ?** (§4.2) Recommandé : (b), le dessus n'est jamais sous un sol
   praticable. Cela change 9,5 à 28,4 % de la zone par rapport à la maquette validée (passerelles et étages
   hauts visibles, comme sur le Plan), retire 11 à 41 % de traits parasites, et rend la coupe juste. Les
   aperçus des deux versions sont prêts pour la page de choix de l'item 0.3.
2. **Le dossier des fonds double environ** : 40 à 50 Mo ajoutés aux 45 Mo d'aujourd'hui (estimation, §1.2).
   Leviers, du moins au plus visible : courbes d'au moins 1 u (−12 à −20 % du JSON) ; tolérance 1 px
   (−8 à −11 %) ; image du relief à demi-résolution (−52 à −62 % de l'image, à juger à l'écran).
3. **Libellé des étages.** Les hauteurs ne sont pas des mètres (une unité ≈ 3 m). Afficher « +2,2 m » serait
   faux. Proposition pour la maquette des commandes (D.1) : des rangs (« Bas », « Étage 1 », « Étage 2 »),
   sans unité.
4. **Lumière quand la carte est tournée.** Recommandé : toujours depuis le haut à gauche de l'écran (§6.4),
   et non depuis le nord-ouest de la carte, pour que le relief ne s'inverse pas à l'œil.
5. **Le drapeau « sol tiré du maillage » (D-6) n'a pas de lecteur** dans la maquette validée. La règle 18
   en veut un. Proposition : l'info-bulle du fond Relief dit la part (« relief tiré du maillage de navigation
   sur 46 % de la carte »), lue de `floors.meshShare`. Sans cette mention, retirer le bit et la part.

### Risques

- **Trous du Relief là où le Rendu a un aplat** (§5.5) : 1,4 à 11,9 % des pixels du rendu n'ont pas de
  hauteur. Cause à établir en B.1 ; compteur au journal.
- **Le Plan ne révèle pas un sol recouvert** sous une coupe (§6.3) : seul le Relief le fait. À dire dans la
  maquette des commandes si le user s'y attend.
- **Relief périmé** après une recuisson du fond : tenu par le calage recopié et le contrôle du service (§5.4).
- **Lecture de l'image** : `createImageBitmap` avec options n'a pas pu être essayé dans un navigateur ici
  (consigne). Le contrôle à la lecture (§6.1) transforme une mauvaise lecture en repli sur le Plan, pas en
  relief faux. À vérifier sur Safari au Gate D.
- **Premier worker de l'application** : chemin de build Vite, tests, coquille Tauri à vérifier en D.2.
- **Mesure du suivi sur Lattice : un seul rejeu.** Les trois autres cartes en ont six ; les chiffres de
  Lattice vont dans le même sens.
- **Fermeture des passages et marge : aucun effet mesuré** sur 4 cartes déjà rognées au maillage. Leur
  utilité se jugera sur les Forge non rognées, en E.1.
- **Terrain tiré du maillage** : sur Behemoth, 32 à 46 % du relief est fait de faces planes ; l'ombrage y
  montre des facettes. C'est ce que la maquette montrait déjà.
- **Cartes sans maillage** (D-7) : relief sans découpe ni coupe ; non mesuré ici, aucune des 4 témoins n'en est.

### Hors périmètre, noté pour la section 9 du plan

- Le code et le sidecar appellent « mètres » des unités du jeu d'environ 3 m (`metersPerPixel`, « m » des
  commentaires, `anchorMedianGapM`). Toute distance affichée « en mètres » à partir de ces valeurs est à
  relire (rayons du radar de la Tactique, distances de tir).
- Le fond publié de Recharge ne se retrouve pas à l'identique par la recuisson d'aujourd'hui (cadre de 35
  lignes de moins, 10,85 % de matière en moins).

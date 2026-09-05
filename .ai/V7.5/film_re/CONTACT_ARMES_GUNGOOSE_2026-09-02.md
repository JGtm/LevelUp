# Planche-contact des armes du Gungoose — enumeration, mesures, verdict (RAPPORT)

> **Reponse a la question exacte : NON, le sprite actuel n'a PAS les deux canons — et OUI, il y a
> un module a placer, exactement comme pour les Warthog : c'est l'objet `scen 0x004164ed`
> (`mode 0x004164ea`), une paire de canons jumeles, attache au Mongoose par la meme `sofa` que le
> `weap 0x0042678e` de la banque `veh_un_wargoose`.**

> Ecrit le 2026-09-02, worktree `LevelUp-wt-vehicules`. Aucun commit, aucun `git add`. Build isole
> (GOCACHE dedie), CGO winlibs, tout en avant-plan, un seul module 7 Go en RAM a la fois. Pas de
> Ghidra. `sprites_v4/gungoose.png` **NON reecrit** : les candidats sont dans
> `gungoose_candidats/`, l'utilisateur pointe d'abord.
>
> Planche : `PLANCHE_CONTACT_ARMES_GUNGOOSE_2026-09-02.png` (1526 x 7532, fond sombre, 13 entrees
> numerotees, Mongoose valide en tete `n.REF`). Colonnes : ISOLE | POSE (chassis + piece, ordre
> peintre) | etiquette source / hash / sections / dimensions. Bandeau bas : deux images de
> DIFFERENCE (rouge = ce que la piece ajoute au chassis nu).
>
> Corrige `REWORK_WARTHOG_GUNGOOSE_2026-09-01.md` §5, qui concluait « les pods avant sont la
> permutation `0x02c9ed0a`, sections 20/23/35/38 ». C'est FAUX sur les trois points : ces sections
> ne sont pas des pods, elles ne sont pas a l'avant, et la permutation n'est pas celle du Gungoose.

## TL;DR — les quatre verdicts

1. **Verdict sur la permutation `0x02c9ed0a` : SOCLE, pas arme — plus precisement PNEUS +
   GARDE-BOUE.** Le `mode 0x9e581380` (Mongoose) a 19 regions et **une seule** permutation
   non-`default`. Region par region (mesures §3) elle remplace : les **4 pneus** (sections 23, 30,
   35, 46 — 1 764 sommets *chacun*, **maillage identique**, 0,386 x 0,102 x 0,343 m, dX ~ dZ =
   disque, aux 4 coins ±0,262 en Y) et **3 garde-boue** (sections 20, 38 arriere, 43 avant pleine
   largeur), et SUPPRIME la region[06]. Preuve visuelle : l'image de difference chassis nu vs
   chassis permute n'allume QUE les quatre roues (bandeau bas de la planche, 33 607 px changes,
   zero devant le carenage). Exactement le meme piege que sur le Warthog (« socles / garde-boue,
   jamais l'arme »).
2. **Verdict `weap` / objet-enfant : il n'y a AUCUN enfant `vehi` d'arme pour le Gungoose** (la
   piste Warthog ne marche pas ici) — **mais il y a un `scen`**. Recherche INVERSE exhaustive de
   `weap 0x0042678e` (le weap du Gungoose, nomme par la banque `veh_un_wargoose` du manifeste V3) :
   8 686 tags balayes en passe 1 (`vehi weap hlmt mode bloc eqip mach scen char crea ctrl coll phmo
   pmdf proj cusc styl siin mdsv drdf sadt xong cddf`) et 6 831 en passe 2 (pc:multiplayer +
   pc:common) -> **0 `vehi` porteur**, seulement 3 catalogues `char` (listes globales d'armes).
   La geometrie arrive par une autre chaine (§2).
3. **Chaine PROUVEE (le « module a placer »)** :
   `vcdd 0xe3ef53f0` -> `vehi 0x000025aa` (Mongoose) + `sofd 0x79b55260` -> `sofa 0x6d1193cc` ->
   { `uwfa 0xe3d5dc10` -> **`weap 0x0042678e`** (banque `veh_un_wargoose`, manifeste V3) ;
   **`scen 0x004164ed`** -> `hlmt 0x004164eb` -> **`mode 0x004164ea`** }. L'arme sonore et la
   geometrie sont sur **le meme tag `sofa`** : l'identification ne repose pas sur la forme.
   Temoin negatif : la variante Mongoose NUE (`vcdd 0x29c4c340` -> `sofd 0x7a0a191d` ->
   `sofa 0xf8eb3113`) ne porte **ni `uwfa` ni `scen`**.
4. **Sens de X : `+X` = AVANT pour le Mongoose** (etabli sur la geometrie, §1). Le sprite final est
   donc pivote de 180 degres comme la famille Warthog validee. **Note : `sprites_v4/mongoose.png`
   est aujourd'hui NEZ EN BAS** (il precede la correction du 2026-09-02) — il faudra lui appliquer
   la meme rotation pour rester coherent avec `razorback.png` / `warthog.png`.

**Avis (non impose) : les deux lance-missiles / canons = `n.10`**, poses par alignement des
noeuds, translation seule `T = (+0,278 ; 0,000 ; +0,341) m`, **aucune rotation** (bouches vers
`+X` = l'avant, comme l'orientation authored). `n.11` est le meme objet a 2,5 cm pres (pose par
detection de zone) : si `n.10` parait 2 px trop en arriere, `n.11` est le voisin immediat.

## 1. Sens de +X du Mongoose : +X = AVANT (etabli sur la geometrie, pas suppose)

Le sens de X est propre a chaque modele (Warthog `+X` = avant, corrige par l'utilisateur le
2026-09-02 ; Scorpion `+X` = arriere). Pour le Mongoose, rendu de PROFIL (axe haut = Y, plan X-Z,
`+X` a droite) et de dessus :

| indice | mesure | cote |
|---|---|---|
| guidon / colonne de direction (bloc haut etroit, sec 49/51 : 0,43 x 0,24 x 0,30, cZ +0,38) | cX **+0,32** | +X |
| poignees / arceau tubulaire : paire symetrique sec 47/48/52/53 (0,33 x 0,16 x 0,16, cZ +0,34) | cX **+0,37**, cY ±0,20 | +X |
| pare-chocs / barre avant sec 54/55 (0,48 x 0,18 x 0,15) | cX **+0,41** | +X |
| garde-boue avant **pleine largeur** (perm sec 43, une seule piece 0,36 x 0,673) | cX **+0,286** | +X |
| porte-bagages plat + rambarde tubulaire (sec 6/8 : 0,33 x 0,30 x **0,09**, plat) | cX **-0,52..-0,55** | -X |
| antenne verticale (sec 11 : 0,23 x 0,13 x 0,10, **cZ +0,41 = section la plus haute**) | cX **-0,46** | -X |
| crochet de remorquage (visible en profil, hauteur de moyeu, extremite du modele) | X **-0,655** | -X |
| garde-boue arriere en **deux pieces separees** (perm sec 20 et 38) | cX **-0,45** | -X |
| squelette : les deux roues `+X` ont un parent COMMUN sur l'axe, n[016] `0x3f376ed4` (+0,342 ; 0 ; +0,141) = direction ; les roues `-X` ont deux parents distincts (bras tires) | — | +X = essieu directeur |

Aucun de ces indices n'est ambigu et ils concordent tous. **`+X` = AVANT, `-X` = ARRIERE.**
Consequence immediate : les sections 23 et 35 de la permutation (cX **-0,445**) sont a l'ARRIERE,
pas a l'avant — le rapport du 2026-09-01 s'etait trompe de bout du vehicule *en plus* de se
tromper sur la nature de la piece.

## 2. Ce qui a ete enumere (aucune source sautee)

| source | comment | resultat |
|---|---|---|
| **`vehi` (67 tags)** | scan des 2 passes de modules, chaines ASCII + `RefModeleVehicule` + refs par groupe | 3 `vehi` Mongoose (`0x000025aa`, `0xaf31ab1a`, `0xde26e3d7`), tous -> `mode 0x9e581380`, tous -> `weap 0x033e41df`. **Aucun `gungoose_g` / `wargoose_g` / `_g` de la famille goose.** |
| **recherche INVERSE de `weap 0x0042678e`** | nouvelle sous-fonction `-refvers` : balayage octet a octet de tous les tags des groupes demandes | passe 1 : 8 686 tags, **6 porteurs, dont 0 `vehi`** (3 catalogues `char`, + les 3 `vehi` Mongoose qui portent l'AUTRE weap `0x033e41df`) ; passe 2 (pc:multiplayer + pc:common) : 6 831 tags, **0 porteur** |
| **chaine R-VEHICULE `vehi -> vcdd -> sofd -> sofa -> uwfa -> weap`** | `-refsde` (refs sortantes par groupe) + `-refvers` en remontee | **c'est la bonne piste** : `sofa 0x6d1193cc` porte `uwfa 0xe3d5dc10` (-> weap Gungoose) ET `scen 0x004164ed` (-> `mode 0x004164ea`) |
| **permutations du `mode 0x9e581380`** | 19 regions, chaque (region, permutation) non-`default` rendue isolee au canevas fixe + mesuree | 7 permutations reelles : 4 pneus + 3 garde-boue (§3) |
| **`mode` de tourelle candidats** | `turret_g` du lot Warthog re-mesures a la meme echelle | `0x1c645961` (le « best-guess » V4 du Gungoose), `0x1c8f09d8`, `0x56fd2500` : objets UNIQUES et hauts, weap sans rapport (§4) |
| **4e `vehi` nomme `mongoose_p` (`0xb889ed92`)** | resolu en passe 2 : `mode 0xca724b38@pc:common` | **fausse piste** : 2,262 x 1,192 x 0,840 m = un chassis de WARTHOG (campagne), 8 permutations de livree, meme geometrie a l'octet |

Detail de la chaine (tous les `sofd`/`sofa` du Mongoose) :

```
vcdd 0x29c4c340 -> vehi 0x000025aa + sofd 0x7a0a191d -> sofa 0xf8eb3113                     (aucun uwfa, aucun scen)  = MONGOOSE NU
vcdd 0xe3ef53f0 -> vehi 0x000025aa + sofd 0x79b55260 -> sofa 0xf8eb3113, 0x30e3a8f6, 0x6d1193cc                        = GUNGOOSE
vcdd 0xe7e21bd9 -> vehi 0x000025aa + sofd 0xf5b495dc -> sofa 0xf8eb3113, 0x6d1193cc          (+ uivi)                  = GUNGOOSE (2e variante)
      sofa 0x6d1193cc -> uwfa 0xe3d5dc10 -> weap 0x0042678e   (banque veh_un_wargoose 0x38167604, manifeste_v3.json)
      sofa 0x6d1193cc -> scen 0x004164ed -> hlmt 0x004164eb -> mode 0x004164ea   <= LA GEOMETRIE
      sofa 0x30e3a8f6 -> gmpm 0x8d10f7d3 (pas de geometrie)
```

Les **deux** `vcdd` du Gungoose portent la permutation de chassis `0x42c9679f` = `default`
(octets 320 des deux tags). **Aucun** ne porte `0x02c9ed0a`. Ce point, a lui seul, disqualifie le
sprite actuel.

## 3. Tableau des candidats (numeros de la planche)

Repere modele, metres, unites compactes (chassis Mongoose = 1,155 m). `c(...)` = centroide.

| n. | source | hash | sections | dX x dY x dZ | centroide | nature mesuree |
|---|---|---|---|---|---|---|
| REF | perm `default` | `0x42c9679f` | 49 / 56 | 1,155 x 0,663 x 0,574 | (+0,045 ; -0,011 ; +0,243) | **Mongoose VALIDE** (reference d'echelle : 116 x 66 px a 10 mm/px) |
| 02 | perm r08 | `0x02c9ed0a` | 30 (1) | 0,386 x 0,102 x 0,343 | (+0,291 ; +0,262 ; +0,120) | **pneu AVANT gauche** (remplace 28-29 : 0,30 x 0,14 x 0,30) |
| 03 | perm r14 | `0x02c9ed0a` | 46 (1) | 0,386 x 0,102 x 0,343 | (+0,291 ; -0,262 ; +0,120) | **pneu AVANT droit** (remplace 44-45) |
| 04 | perm r05 | `0x02c9ed0a` | 23 (1) | 0,386 x 0,102 x 0,343 | (-0,445 ; +0,262 ; +0,120) | **pneu ARRIERE gauche** (remplace 21-22) — dit « barillet avant » le 2026-09-01 |
| 05 | perm r10 | `0x02c9ed0a` | 35 (1) | 0,386 x 0,102 x 0,343 | (-0,445 ; -0,262 ; +0,120) | **pneu ARRIERE droit** (remplace 33-34) |
| 06 | perm r13 | `0x02c9ed0a` | 43 (1) | 0,360 x **0,673** x 0,273 | (+0,286 ; 0,000 ; +0,125) | **garde-boue AVANT** pleine largeur (remplace 41-42) |
| 07 | perm r04 | `0x02c9ed0a` | 20 (1) | 0,356 x 0,148 x 0,273 | (-0,447 ; +0,291 ; +0,125) | **garde-boue ARRIERE gauche** (remplace 18-19) |
| 08 | perm r11 | `0x02c9ed0a` | 38 (1) | 0,356 x 0,148 x 0,273 | (-0,453 ; -0,291 ; +0,125) | **garde-boue ARRIERE droit** (remplace 36-37) ; r06 : sections 24-25 SUPPRIMEES |
| 09 | groupe perm | `0x02c9ed0a` | 56 | 1,155 x 0,673 x 0,596 | (+0,026 ; -0,010 ; +0,224) | **= `sprites_v4/gungoose.png` ACTUEL — A REJETER** (n.02..n.08 et rien d'autre) |
| **10** | **`scen 0x004164ed`** | **`mode 0x004164ea`** | **1 (3 686 som.)** | **0,303 x 0,346 x 0,123** | **(-0,005 ; 0,000 ; +0,002)** | **CANONS JUMELES — pose par les noeuds, `T = (+0,278 ; 0 ; +0,341)`** |
| 11 | idem n.10 | idem | idem | idem | idem | meme objet, pose par detection de zone, `T = (+0,303 ; 0 ; +0,439)` (ecart 2,5 cm en X) |
| 12 | n.10 + perm | — | 56 + 1 | — | — | hypothese cumulee (canons ET pneus) — **non demandee par les `vcdd`** |
| 13 | `vehi 0x3a8060e2` (`turret_g`) | `mode 0x1c645961` | 3 | 0,668 x 0,330 x 0,550 | (-0,038 ; -0,001 ; +0,401) | le « best-guess » V4 — **A REJETER** (§4) |

Voisins de n.13, mesures a la meme echelle pour completude : `0x1c8f09d8` (6 sec, 0,539 x 0,502 x
0,533), `0x56fd2500` (4 sec, 0,554 x 0,231 x 0,508), `0xdb2a499b` (3 sec, 1,203 x 1,128 x 0,598).

## 4. Pourquoi n.10 est l'arme, et pourquoi n.13 ne l'est pas

**n.10 — `mode 0x004164ea`, ce que la mesure dit :**

- **Deux exemplaires symetriques.** 6 noeuds : racine `0x1730bd18` = **`b_pedestal`** (nom resolu
  par brute-force murmur3, 61 560 noms testes — 1 resolu), 3 noeuds a l'origine, et **2 noeuds
  symetriques** `0x77a954af` / `0xbd6679e0` a **(+0,007 ; ∓0,128 ; -0,019)**. Un objet, deux futs.
- **Forme.** Vue de PROFIL (axe haut Y), chaque unite montre dans l'ordre : culasse inclinee avec
  couvre-alimentation, **chemise a ailettes** (les nervures de refroidissement), puis une **bouche
  etagee**. Vue de DESSUS : deux blocs identiques cote a cote, fut vers `+X`. C'est une arme, pas
  un socle.
- **Le point d'attache tombe pile.** Le squelette du chassis Mongoose (146 noeuds, echelle 1,000
  partout) porte un couple de marqueurs **n[007] `0xcb458294` (+0,285 ; +0,126 ; +0,322)** et
  **n[024] `0xb9e02725` (+0,285 ; -0,126 ; +0,322)** (doubles par n[027] / n[031] a la meme
  position). Ecart en Y avec les futs de l'arme : **2 mm**. Alignement des deux couples ->
  `T = (+0,285 - 0,007 ; 0 ; +0,322 + 0,019) = (+0,278 ; 0,000 ; +0,341)`.
- **Croisement independant.** La detection de zone (methode Warthog V2/V3, moitie avant `X > 0`,
  erosion 1 px) trouve **deux pastilles plates symetriques** de 11 x 22 px a `(X +0,310 ; Y ±0,175)`,
  Z mediane 0,379 ; centre reuni `(X +0,310 ; Y 0,000)`. La pose qui en decoule (n.11) est a
  **2,5 cm** de la pose par noeuds. Deux methodes independantes, meme resultat.
- **Resultat visuel.** Difference chassis nu vs chassis + n.10 : **10 380 px**, tous groupes en
  **deux blocs symetriques a l'avant**, de part et d'autre de l'axe, bouches vers le nez. C'est
  exactement la verite terrain decrite par l'utilisateur.

**n.13 — `mode 0x1c645961` (le choix V4), ce qui l'elimine :**

- objet **unique** (pas un couple), **haut de 0,55 m** (le Mongoose entier fait 0,574 m de haut) ;
- le `vehi 0x3a8060e2` qui le porte reference `weap 0x4d39877f` et `0x3d30b955` — **aucun rapport**
  avec `0x0042678e` ;
- **aucun `vcdd` du Mongoose ne le reference** ; c'est un emplacement de tourelle autonome.

## 5. Pose retenue, echelle, orientation

- **Canevas** : cadre ±5 m, **10 mm/px**, 1012 x 1012, origine locale au pixel (505,5 ; 505,5),
  symetrique. **C'est l'echelle de `mongoose.png`** (mesure : 115 px opaques pour 1,155 m =
  10,04 mm/px ; idem `razorback.png` 211 px / 2,10 m et `warthog.png` 210 px / 2,10 m).
  **`sprites_v4/gungoose.png` actuel est a 6 mm/px** (192 px opaques) : **1,67x trop grand** par
  rapport au reste du lot. Deuxieme defaut du sprite actuel, independant du premier.
- **Orientation de l'arme : AUCUNE rotation** (lecon V3 du Warthog : l'orientation authored du
  `mode` enfant fait foi). Bouches vers `+X` = l'avant du Mongoose.
- **Pose** : ordre peintre (chassis, puis arme), rognage marge 6 px, **rotation finale de 180
  degres de l'image** (rotation, pas miroir) -> nez en haut, comme `warthog.png` / `razorback.png`.
- Emprises posees : n.10 `X[+0,126..+0,429] Y[±0,173] Z[+0,281..+0,403]` ; n.11
  `X[+0,152..+0,454] Y[±0,173] Z[+0,379..+0,501]`. Dans les deux cas les bouches s'arretent
  **avant** le pare-chocs (`X +0,500`) : les canons **ne saillent pas devant le nez** — ce que
  l'utilisateur avait deja pressenti au tour precedent, mais pour la mauvaise piece.

## 6. Livrables

| fichier | taille | contenu |
|---|---|---|
| `gungoose_candidats/gungoose_A_canons_noeuds.png` | 78 x 128 | **n.10** — chassis `default` + canons, `T = (+0,278 ; 0 ; +0,341)`, nez en haut, 10 mm/px |
| `gungoose_candidats/gungoose_B_canons_plateau.png` | 78 x 128 | **n.11** — meme objet, `T = (+0,303 ; 0 ; +0,439)` (detection de zone) |
| `gungoose_candidats/gungoose_C_permutation_roues.png` | 80 x 128 | **n.09** — le sprite actuel refait a la BONNE echelle (pneus seuls, sans canons) |
| `gungoose_candidats/gungoose_D_canons_et_permutation.png` | 80 x 128 | **n.12** — canons + permutation (cumul, non demande par les tags) |
| `gungoose_candidats/_reference_mongoose_nez_en_haut.png` | 78 x 128 | Mongoose `default` nez en haut (reference d'echelle et d'orientation) |
| `gungoose_candidats/_canons_isoles_dessus.png` | 46 x 42 | l'arme seule, vue de dessus, 10 mm/px |
| `gungoose_candidats/_canons_isoles_profil.png` | 500 x 500 | l'arme seule **de profil** (2 mm/px) : culasse, ailettes, bouche |
| `gungoose_candidats/_diff_A_ce_que_les_canons_ajoutent.png` | 344 x 590 | rouge = les deux canons, gris = silhouette commune |
| `gungoose_candidats/_diff_C_ce_que_la_permutation_change.png` | 348 x 590 | rouge = les quatre roues, et rien d'autre |
| `PLANCHE_CONTACT_ARMES_GUNGOOSE_2026-09-02.png` | 1526 x 7532 | la planche numerotee |

`sprites_v4/gungoose.png` : **NON reecrit**.

## 7. CR honnete

**Prouve (mesure ou tag) :**

- la permutation `0x02c9ed0a` est un jeu de 4 pneus + 3 garde-boue (maillages identiques 4x,
  centres sur les 4 moyeux, image de difference concluante) ; les deux `vcdd` du Gungoose imposent
  `default` et ne la demandent jamais ;
- il n'existe aucun objet-enfant `vehi` d'arme dans la famille goose (recherche inverse exhaustive
  sur 15 517 tags, deux passes de modules) ;
- `sofa 0x6d1193cc` porte a la fois le `weap` du Gungoose (via `uwfa`) et un `scen` avec un
  `render_model` — l'identite de la geometrie ne repose donc PAS sur la ressemblance ;
- `mode 0x004164ea` = deux unites symetriques a futs (noeuds a ±0,128), qui tombent a 2 mm pres sur
  un couple de marqueurs du chassis situe a l'avant, en haut (+0,285 ; ±0,126 ; +0,322) ;
- `+X` = avant pour le Mongoose (9 indices geometriques concordants) ;
- `mongoose.png` est a 10 mm/px, `gungoose.png` actuel a 6 mm/px.

**Choix, dits sans les maquiller :**

- entre n.10 (noeuds) et n.11 (zone detectee), j'ai retenu n.10 : le couple de noeuds est une
  donnee du fichier, la detection de zone est une heuristique d'image. L'ecart est de 2,5 cm
  (2,5 px sur le sprite) ; les deux sont fournis ;
- le nom du `scen`, celui du marqueur (`0xcb458294`, `0xb9e02725`) et celui de la permutation
  (`0x02c9ed0a`) restent **non resolus** (brute-force murmur3 sur 61 560 noms : seul
  `0x1730bd18 = b_pedestal` sort). L'identification est structurelle (chaine de tags), pas
  nominale ;
- l'appellation « lance-missiles » vient de l'utilisateur ; la geometrie, elle, montre deux futs a
  **chemise a ailettes** avec bouche etagee — visuellement des canons/mitrailleuses jumeles. Je le
  signale sans trancher le vocabulaire : c'est bien la paire d'armes avant du Gungoose ;
- Z de la pose (0,341 par les noeuds, 0,439 par la zone) est **sans effet en vue de dessus**
  (ordre peintre) ; je ne l'ai pas arbitre autrement ;
- `sprites_v4/mongoose.png` est nez en BAS et a la bonne echelle : il faudra le pivoter de 180
  degres quand le Gungoose sera valide, sinon les deux sprites de la famille seront tete-beche.
  **Non fait dans ce lot** (hors perimetre, l'utilisateur pointe d'abord).

**Non fait :** aucun Ghidra ; aucune modification de `internal/himap/*` ni de
`cmd/vehicle-sprite/*` ; aucun commit ; la 2e variante Gungoose (`vcdd 0xe7e21bd9`, avec `uivi`) a
la meme `sofa` d'arme et n'a donc pas ete rendue separement.

## 8. Outils et reproduction

Driver jetable (worktree, non commite) : `cmd/vs-measure/` — nouveau fichier `goose.go`
(`-refvers` recherche inverse, `-refsde` refs sortantes par groupe, `-groupes` inventaire,
`-extrait` octets bruts, `-axe` vue de profil) ; `armes.go` et `plateau.go` etendus
(`-axe`, `-tforce`, `-permchassis`). `gofmt`, `go vet`, `go build` propres ; tous les fichiers
≤ 500 lignes.

```
# 1. famille goose + recherche inverse du weap du Gungoose (passe pc:globals)
vsmeasure.exe armes "-modules=pc:globals-rtx-new.module,any:globals-rtx-new.module,any:common-rtx-new.module,any:multiplayer-rtx-new.module,any:multiplayer_r1-rtx-new.module,any:multiplayer_r3-rtx-new.module" \
  -vehiscan=goose,mongoose,gungoose,wargoose -refvers=0x0042678e,0x033e41df \
  "-refgroupes=vehi,weap,hlmt,mode,bloc,eqip,mach,scen,char,crea,ctrl,coll,phmo,pmdf,proj,cusc,styl,siin,mdsv,drdf,sadt,xong,cddf" -out=OUT
# 2. remontee de la chaine R-VEHICULE
vsmeasure.exe armes "-modules=..." -refsde=0x000025aa,0x7a0a191d,0x6d1193cc,0x004164ed "-refsdegroupes=*" \
  -refvers=0xe3d5dc10,0x6d1193cc,0xf8eb3113 "-refgroupes=vehi,sofd,sofa,uwfa,vcdd,scen,mode,hlmt"
# 3. permutations du chassis + arme isolee, canevas commun 10 mm/px
vsmeasure.exe armes "-modules=pc:globals-rtx-new.module,any:globals-rtx-new.module" \
  -chassis=0x9e581380 -pieces=0x004164ea -out=OUT -cadre=5 -cellmm=10
vsmeasure.exe armes "-modules=..." -pieces=0x004164ea -out=OUT -cadre=0.25 -cellmm=1 -axe=y   # profil
# 4. squelettes (146 noeuds chassis / 6 noeuds arme) + brute-force des noms
vsmeasure.exe "-modules=pc:globals-rtx-new.module,any:globals-rtx-new.module" -nodes=0x9e581380,0x004164ea
vsmeasure.exe armes -hashes=0x1730bd18,0x77a954af,0xbd6679e0,0xcb458294,0x02c9ed0a
# 5. poses (n.10 par les noeuds, n.11 par detection de zone)
vsmeasure.exe plateau "-modules=pc:globals-rtx-new.module,any:globals-rtx-new.module" -chassis=0x9e581380 \
  "-armes=gungoose=0x004164ea" -out=OUT -cadre=5 -cellmm=10 -erode=1 -union=0.6 -xplateau=1 \
  -rotarme=false -rot180 "-tforce=0.278,0,0.341"     # n.10 ; sans -tforce : n.11
```

Planche et images de difference : scripts jetables du scratchpad (`planche_gungoose.ps1`,
`pngbox` en Go), non versionnes.

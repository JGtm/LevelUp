# Planche-contact des armes arriere Warthog — enumeration exhaustive, mesures, verdict `weap` (RAPPORT)

> Ecrit le 2026-09-01, worktree `LevelUp-wt-vehicules`. Aucun commit, aucun `git add`. Build isole
> (GOCACHE dedie), CGO winlibs, tout en avant-plan, un seul module 7 Go en RAM a la fois.
> Contexte : trois rejets successifs de la famille Warthog armee (chaingun / roquettes / gauss) ;
> seul le Razorback (cargo, `unarmed`) est valide. Consigne : NE PAS DEVINER, enumerer TOUTES les
> geometries candidates d'arme arriere et laisser l'utilisateur pointer.
>
> Planche : `PLANCHE_CONTACT_ARMES_WARTHOG_2026-09-01.png` (2073 x 5282, fond sombre, 27 candidats
> numerotes + la reference). Colonnes : ISOLE | ASSEMBLE T=0 | ASSEMBLE T=noeud (+0,765 m arriere).
> Repere : +X = ARRIERE (bas de l'image), Y lateral, Z haut ; metres du modele (echelle compacte,
> chassis = 2,10 m). Tous les rendus au MEME canevas fixe (cadre 5 m, 8 mm/px, origine commune).

## TL;DR

1. **Piste `weap` : les tags `weap` de tourelle n'ont PAS de modele — et ce n'est PAS un effet du
   plancher anti-parasite.** Avec le resolveur corrige, 100/192 `weap` (passe pc:globals) et 28/192
   (passe pc:multiplayer+common) resolvent bien un `mode` : ce sont TOUS des armes TENUES d'infanterie
   (largeur <= 0,30 m, repere a la poignee). Les 9 `weap` de la famille Warthog/vehicule
   (`0x0000a4bc` chaingun chassis, `0xc7d50912` rockethog, `0x0042678e` gungoose, `0x033e41df`
   mongoose, `0x0000e0d9`, `0x8647925a`, `0x0131c29e`, `0x31982437`, `0x0c6fd911`) n'ont **AUCUNE
   ref `hlmt`/`mode`/`rtgo` au balayage brut (tout ID, sans plancher)** : rien a resoudre. Verdict
   honnete : la piste `weap` ne fournit AUCUNE geometrie d'arme arriere.
2. **MAIS la piste `weap` fournit la CLE D'IDENTIFICATION** qui manquait : chaque objet-enfant
   `warthog_g`/`warthog_b_g` reference SON propre `weap`, et l'un d'eux est nomme par le chantier
   sons : **`vehi 0xbcfb852f` (warthog_b_g, mode `0xbe74e831`) -> `weap 0xc7d50912` = ROCKETHOG**
   (banque `veh_un_rockethog`, manifeste V3). Le pod de roquettes est donc identifie par
   croisement, pas par la forme (**n.13** sur la planche).
3. **Decouverte structurelle : les objets-enfants NE SONT PAS co-reperes avec le chassis.** Leur
   base est a Z = 0 et leur pivot de tourelle (noeud `0x99d45ed9`, partage par 4 enfants) est a
   (+0,27, 0, +0,40) dans LEUR repere ; pose a translation nulle, un pod aurait sa base au niveau
   du sol AU MILIEU du chassis. Le rapport ASSEMBLAGE avait conclu « co-reperes » sur le Scorpion,
   dont le pivot est justement a l'origine (cas non discriminant). Il faut une translation vers le
   noeud d'attache du chassis : candidat par position **n[006] `0xe1a390ba` (+0,765, 0, +0,541)**,
   dont Z = 0,541 coincide avec le SOMMET des socles de region[17] (Z = 0,549). Nom non resolu
   (brute-force murmur3 de 61 560 noms : 0 resolu). La planche montre les deux poses (T=0 et
   T=noeud) pour chaque enfant : l'utilisateur pointe.
4. **Les 3 sprites rejetes sont exactement les groupes de permutations n.10 / n.11 / n.12** : ce sont
   des SOCLES (0,18-0,33 m de long) + garde-boue + plaques, jamais une arme. C'est pourquoi « rien
   n'est monte a l'arriere ».
5. **Avis (non impose)** : roquettes = **n.13** (`0xbe74e831`, prouve) ; chaingun/LAAG = **n.14**
   (`0x6b17fdb5`, weap classe `turret`, 44 noeuds, canon long) ; gauss = **n.15** (`0xc0803caa`,
   `warthog_b_g_l`, fut long et etroit). Tous trois en pose T=noeud, par-dessus le socle de
   region[17] correspondant. Les deux derniers restent des choix PAR FORME (voir §6).

## 1. Ce qui a ete enumere (les trois sources + une quatrieme)

| source | comment | resultat |
|---|---|---|
| **`weap`** (192 tags indexes) | `RefModeleVehicule` (agnostique du groupe) sur chaque `weap` ; refs brutes par groupe (tout ID) ; mesure de chaque mode resolu ; rendu des modes >= 0,25 m de large | 128 modes resolus au total, tous armes tenues ; 5 rendus (n.23-27) ; 0 arme de tourelle |
| **`vehi` enfants** (67 vehi, 27 « warthog ») | re-rendus au canevas fixe, squelettes dumpes, weap reference par enfant | 7 warthog_g/b_g + 3 turret_g (n.13-22) |
| **permutations** du `mode 0x561f2ca7` (20 regions, 5 noms) | CHAQUE (region, permutation) non-`default` rendue isolee + les 3 groupes V4 | 9 perms individuelles + 3 groupes (n.1-12) |
| **`mode` par nom** (1 875 modes pc:globals + 375 pc:mp/common) | chaines ASCII contenant warthog/hog/chaingun/laag/gauss/rocket/turret | **0** : les `mode` ne portent aucun nom lisible (StringId seulement) |

## 2. Tableau des candidats (numeros de la planche)

Mesures = boite englobante dX x dY x dZ (m) et centroide c(X, Y, Z) ; « monte » = ou la piece se
trouve une fois posee (perm : par construction ; enfant : selon la pose T=noeud, +0,765 en X).

| n. | source | hash | sections | dX x dY x dZ | centroide | ou c'est monte / remarque |
|---|---|---|---|---|---|---|
| REF | perm `unarmed` + base | `0x4e154ee8` | 58 | 2,12 x 1,01 x 0,83 | (+0,17, 0, +0,43) | **Razorback VALIDE** (reference d'echelle) |
| 1 | perm r17 | `0x06c86db1` (V4 « chaingun ») | 84-85 (2) | 0,26 x 0,76 x 0,39 | (+0,93, 0, +0,36) | socle arriere-centre, sommet Z 0,55 |
| 2 | perm r17 | `0x13d24f1f` (V4 « roquettes ») | 80-81 (2) | 0,33 x 0,66 x 0,28 | (+0,94, 0, +0,33) | socle arriere-centre |
| 3 | perm r17 | `0xad03512a` (V4 « gauss ») | 82-83 (2) | 0,18 x 0,76 x 0,37 | (+0,97, 0, +0,30) | socle arriere-centre |
| 4 | perm r00 | `0xad03512a` | 6 (1) | 0,79 x 0,73 x 0,52 | (+0,91, 0, +0,39) | bloc arriere haut, X +0,22..+1,01 |
| 5 | perm r00 | `0x13d24f1f` | 5 (1) | 0,12 x 0,65 x 0,08 | (+0,95, 0, +0,32) | plaque plate arriere |
| 6 | perm r11 | `0x13d24f1f` | 57-58 (2) | 0,42 x 0,60 x 0,13 | (+0,36, 0, +0,64) | plaque de toit (point le plus haut) |
| 7 | perm r11 | `0xad03512a` | 59-60 (2) | 0,45 x 0,59 x 0,13 | (+0,38, 0, +0,64) | plaque de toit |
| 8 | perm r01-04 | `0x06c86db1` (= `0xad03512a` a 5 mm) | 4 x 2 | 0,41 x 0,21 x 0,41 (x4) | (-0,66/+0,63, +-0,41) | garde-boue blindes des 4 roues — pas une arme |
| 9 | perm r18 | `0x13d24f1f` | 86-89 (4) | 0,05 x 0,97 x 0,28 | (-0,91, 0, +0,22) | plaque AVANT — pas une arme |
| 10 | groupe V4 | `0x06c86db1` | 1 + 8 | — | — | = `sprites_v4/warthog.png` (REJETE) |
| 11 | groupe V4 | `0x13d24f1f` | 2 + 5 + 6 + 9 | — | — | = `sprites_v4/rockethog.png` (REJETE) |
| 12 | groupe V4 | `0xad03512a` | 3 + 4 + 7 + garde-boue | — | — | = `sprites_v4/warthog_gauss.png` (REJETE) |
| 13 | vehi warthog_b_g `0xbcfb852f` | mode `0xbe74e831` (pc:globals) | 5 | 0,58 x 0,58 x 0,69 | (+0,14, 0, +0,46), base Z 0 | **weap `0xc7d50912` = ROCKETHOG** ; T=noeud : X +0,59..+1,17 |
| 14 | vehi warthog_g/b_g `0x4ccc20e6` | mode `0x6b17fdb5` (pc:common) | 8 (44 noeuds) | 1,02 x 0,50 x 0,69 | (+0,13, 0, +0,24), base Z 0 | weap `0x31982437` classe `turret` ; canon depasse l'arriere (T=noeud : X +0,60..+1,62) |
| 15 | vehi warthog_b_g_l `0xdd7f9102` | mode `0xc0803caa` (pc:globals) | 9 | 0,95 x 0,35 x 0,65 | (+0,25, +0,01, +0,43), base Z 0 | weap `0x0c6fd911` classe `turret` ; fut long ETROIT |
| 16 | vehi warthog_b_g `0x64b925eb` | mode `0x9c7f3b54` (pc:globals) | 6 | 0,95 x 0,61 x 0,72 | (+0,21, -0,11, +0,52), base Z 0 | weap `0x8647925a` classe `fixed` ; asymetrique (bloc a gauche) |
| 17 | vehi warthog_g `0x1779ea58` | mode `0x0261f134` (pc:common) | 6 (9 noeuds, b_root) | 1,15 x 0,70 x 0,84 | (+0,37, 0, +0,63), base Z 0 | weap `0x0131c29e` classe `turret` ; grand disque (bouclier/dome) Z jusqu'a 0,80 |
| 18 | vehi warthog_g/b_g `0x00409dac` | mode `0x00409881` (pc:common) | 1 | 0,45 x 0,23 x 0,43 | (+0,18, 0, +0,22), base Z 0 | aucun weap ; petit support seul |
| 19 | vehi warthog_g `0x0000e0ca` | mode `0x0000e0da` (pc:common) | 3 | 2,39 x 0,85 x 0,62 | (+1,74, 0, +0,18), X +0,06..+2,45 | weap `0x0000e0d9` classe `fixed` ; **REJETE** (barillet deploye, pivot a +1,69) |
| 20 | vehi turret_g `0x003f00c7` | mode `0x1c8f09d8` | 6 | 0,54 x 0,50 x 0,53 | (+0,02, 0, +0,25) | generique (V4 « rockethog ») |
| 21 | vehi turret_g `0x3a8060e2` | mode `0x1c645961` | 3 | 0,67 x 0,33 x 0,55 | (-0,04, 0, +0,40) | generique (V4 « gungoose ») |
| 22 | vehi turret_g `0x038df01a` | mode `0x1ae526e1` | 28 | 1,26 x 1,23 x 0,57 | (-0,03, +0,01, +0,33) | tourelle_montee (Shade-like), trop large |
| 23 | weap `0x0041c4e3` (« turret ») | hlmt `0x0041c4e0` -> mode `0x0041c4e2` | 1 | 0,53 x 0,30 x 0,22 | (+0,08, -0,08, -0,05) | arme tenue |
| 24 | weap `0xc7789529`/`0xfb6ee710` | hlmt `0x46c10f4c` -> mode `0xcd3b271b` | 2 | 0,38 x 0,28 x 0,24 | (+0,16, -0,01, -0,01) | arme tenue |
| 25 | weap `0x00412474` (« turret ») | hlmt `0x00412444` -> mode `0x00412446` | 1 | 0,62 x 0,24 x 0,27 | (+0,29, 0, -0,01) | arme tenue (ref aussi `vehicle_partial_emp`) |
| 26 | weap `0xab8cd22e` (« turret ») | hlmt `0x054758fa` -> mode `0x1944a439` | 2 | 0,53 x 0,17 x 0,17 | (+0,21, -0,01, -0,03) | arme tenue |
| 27 | weap `0x5bd53639` | mode `0x5e204f8d` | 3 | 0,33 x 0,33 x 0,32 | (0, 0, +0,14) | cylindre (objet posable), pas une tourelle |

Les permutations `unarmed` (r05, r10-r16 : caisse cargo, sieges arriere, plaques) sont dans la
reference Razorback validee et ne sont pas des armes ; elles ne figurent pas sur la planche.

## 3. Verdict detaille sur la piste `weap`

- Index pc:globals (+ tous les `any`) : 192 `weap`, **100 resolvent un `mode`** avec le resolveur
  corrige (`RefModeleVehicule` marche tel quel sur un `weap` : simple balayage de refs `hlmt` ->
  `mode`). Index pc:multiplayer + pc:common : 28 resolvent (dont 4 dont le `hlmt` etait vu en
  passe 1 mais le `mode` vivait en pc:common : `0x00412446`, `0x0041c4e2`, `0x1944a439`, `0xcd3b271b`).
- **Toutes** ces geometries sont des armes TENUES : dX 0,08-0,67 m, **dY <= 0,30 m**, dZ <= 0,27 m
  (a une exception pres, un objet vertical 0,24 x 0,11 x 0,79 = non-arme), origine a la poignee.
  Le lot V4 avait donc raison sur le fond (« les weap de tourelle n'ont pas de modele ») mais pour
  une raison differente de celle craint : ce n'est pas le plancher, c'est l'absence pure et
  simple de ref `hlmt` dans ces 9 tags (compte brut : `adlg aigl cddf cusc effe foot jpt! lsnd
  pmcg proj rasg snd! sngl trak wpdp` — zero `hlmt`/`mode`/`rtgo`).
- La chaine ASCII des `weap` distingue deux classes : `fixed` (`0x0000a4bc`, `0xc7d50912`,
  `0x0042678e`, `0x033e41df`, `0x0000e0d9`, `0x8647925a`) et `turret` (`0x0131c29e`, `0x31982437`,
  `0x0c6fd911`, et ~40 autres). Les 3 enfants n.14/15/17 portent un weap `turret` ; n.13 (roquettes)
  et n.16/19 un weap `fixed`.
- Chaine sonore weap -> snd!/lsnd -> sbnk : les `snd!` ne referencent pas la banque par tagref
  (le chantier sons l'a trouvee par intersection des IDs de wem), donc le nommage par FNV-1 n'a
  rien donne ici ; l'identite ROCKETHOG de `0xc7d50912` vient du manifeste V3 (`manifeste_v3.json`,
  « weap c7d50912 -> snd! 155f1354 -> banque veh_un_rockethog a52af042 »).
- `0x0bf807fe -> mode 0xa57a133e` (pc:globals) : le `mode` est illisible pour `NewRenderModelAsset`
  (« tag mode illisible ») — un seul cas, note, non traite.

## 4. Structure parent/enfant : ce que les squelettes disent

- Chassis `0x561f2ca7` : 106 noeuds, echelle 1 partout (confirme). Candidats d'attache arriere-
  centre : **n[006] `0xe1a390ba` (+0,765, 0, +0,541)**, n[010] `0x27097f82` (+0,715, +0,005, +0,584),
  n[005]/n[011] (+0,809, +-0,296, +0,523) = probablement les mains/poignees du gunner.
- Enfants `0x6b17fdb5`, `0xbe74e831`, `0xc0803caa`, `0x9c7f3b54` partagent les noeuds `0x1a2b72c5`
  (n[001], racine) et **`0x99d45ed9` (n[004], le pivot) a (+0,27..0,28, 0, +0,38..0,42)** ; trois
  d'entre eux ont `0xecd9032e` (bouche/extremite du fut : +0,80 / +0,70 / +0,64). `0x0000e0da`
  (n.19) porte le meme `0x99d45ed9` mais a (+1,687, 0, +0,227) : c'est un asset authore autrement
  (barillet deploye), ce qui explique le rejet.
- Consequence : **la pose correcte d'un enfant est origine-enfant = noeud d'attache du chassis**
  (mecanisme etabli par le rapport Ghidra : transformee du noeud nomme, ici translation pure,
  echelle 1). Avec n[006], le pivot enfant tombe a X = +1,04 (bord arriere du socle r17, X
  +0,76..+1,02) et Z = +0,94 ; la base de l'enfant (Z 0) se pose sur le sommet du socle (Z 0,55).
  Si l'utilisateur juge n[006] trop en arriere, n[010] (+0,715) est le voisin immediat (5 cm).
- Le nom du noeud reste non resolu : brute-force murmur3 (`mapvar.LabelHash`) sur 61 560 noms
  generes (prefixe x coeur x suffixe : turret/gun/gunner/mount/attach/hardpoint/…) : 0 resolu pour
  `0xe1a390ba` et `0x27097f82`.

## 5. Modules (contrainte RAM, pour la regeneration)

- pc:globals (7,8 Go) : chassis `0x561f2ca7` + enfants `0xbe74e831` (n.13), `0xc0803caa` (n.15),
  `0x9c7f3b54` (n.16) -> assemblage IN-MODULE possible (z-buffer partage, meilleure occlusion).
- pc:common (2,9 Go) + pc:multiplayer (1 Go) : enfants `0x6b17fdb5` (n.14), `0x0261f134` (n.17),
  `0x00409881` (n.18), `0x0000e0da` (n.19), turret_g (n.20-22) -> composition 2D au canevas fixe
  (`assemble -cadre` + `compose2d`), l'enfant translate de (+0,765, 0) en X/Y.

## 6. Avis honnete sur le trio (a confirmer par l'utilisateur, pas impose)

| arme | candidat | pose | confiance | pourquoi |
|---|---|---|---|---|
| roquettes | **n.13** `0xbe74e831` | T=noeud | **forte** (croisement weap -> banque sons) | deux caissons a tubes de 0,58 m, seul enfant lie au weap rockethog |
| chaingun / LAAG | **n.14** `0x6b17fdb5` | T=noeud | moyenne (forme) | weap classe `turret`, 44 noeuds (mecanique de tir), canon de 1,02 m debordant l'arriere comme en jeu ; alternative n.17 (`0x0261f134`, disque-bouclier) |
| gauss | **n.15** `0xc0803caa` | T=noeud | moyenne (forme + suffixe `_l`) | fut le plus long et le plus etroit (0,35 m) ; alternative n.16 (`0x9c7f3b54`, bloc lateral asymetrique) |

Sous chacun, le socle de region[17] de la variante (n.1 / n.2 / n.3 — mapping V4 par forme, non
prouve mais sans consequence visuelle : 0,2-0,3 m sous l'arme). Ce que je ne sais PAS : lequel de
n.14 / n.17 est la LAAG « standard » et lequel de n.15 / n.16 est le Gauss — seuls des noms de
tag (strippes) ou une capture en jeu trancheraient. Si l'utilisateur ne reconnait aucun de n.13-17,
alors la geometrie d'arme arriere n'est dans AUCUNE des quatre sources explorees (weap, vehi,
permutations, modes par nom) et il faudra le dire tel quel.

## 7. Outils et reproduction

Driver jetable (worktree, non commite, ne touche aucun code partage) : `cmd/vs-measure/armes.go` +
`armes_noms.go` (sous-commande `armes` ; `main.go` : 4 lignes d'aiguillage). `gofmt`, `go vet`,
`go build` propres. Planche : script PowerShell/System.Drawing dans le scratchpad (non versionne).

```
# passe 1 (pc:globals) : weap + chassis/permutations + enfants globals
vsmeasure.exe armes -modules="pc:globals-rtx-new.module,any:globals-rtx-new.module,any:common-rtx-new.module,any:multiplayer-rtx-new.module,any:multiplayer_r1-rtx-new.module,any:multiplayer_r3-rtx-new.module" \
  -weapscan="warthog,hog,chaingun,laag,gauss,rocket,turret" -weapmesure -vehiscan=warthog \
  -chassis=0x561f2ca7 -pieces="0x9c7f3b54,0xbe74e831,0xc0803caa" -nodes="0x561f2ca7,0xbe74e831,0xc0803caa,0x9c7f3b54" \
  -modescan="warthog,hog,chaingun,laag,gauss,rocket,turret" -out=OUT -cadre=5 -cellmm=8
# passe 2 (pc:multiplayer + pc:common) : weap + enfants common + sons + squelettes
vsmeasure.exe armes -modules="pc:multiplayer-rtx-new.module,pc:common-rtx-new.module,pc:multiplayer_r1-rtx-new.module,pc:multiplayer_r3-rtx-new.module,any:...(idem)" \
  -weapscan=... -weapmesure -ids="0x0000e0d9,0x8647925a,0x00412444,..." -sons="0x0000a4bc,0xc7d50912,..." \
  -pieces="0x0000e0da,0x0261f134,0x00409881,0x6b17fdb5,0x1c645961,0x1c8f09d8,0x1ae526e1,0x00412446,0x1944a439" \
  -nodes="0x0000e0da,0x0261f134,0x6b17fdb5,0x00409881" -out=OUT -cadre=5 -cellmm=8
# brute-force des noms de noeuds (sans modules)
vsmeasure.exe armes -hashes="0xe1a390ba,0x27097f82"
```

`sprites_v4/warthog.png`, `rockethog.png`, `warthog_gauss.png` : NON reecrits (l'utilisateur pointe
d'abord).

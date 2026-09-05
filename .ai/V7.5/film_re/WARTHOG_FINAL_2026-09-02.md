# Warthog arme — pose finale de l'arme au centre du plateau arriere (RAPPORT)

> Ecrit le 2026-09-02, worktree `LevelUp-wt-vehicules`. Aucun commit, aucun `git add`. Build isole
> (GOCACHE dedie), CGO winlibs, tout en avant-plan, module pc:globals seul en RAM (les 4 modes sont
> dedans). Suite de `CONTACT_ARMES_WARTHOG_2026-09-01.md` : l'utilisateur a pointe n.13 / n.15 / n.16
> et rejete la pose « T = noeud n[006] » ; consigne : « au CENTRE DU RECTANGLE BLANC a l'arriere ».
> Planche : `PLANCHE_WARTHOG_FINAL_2026-09-02.png` (1984 x 3834). Repere modele : +X = ARRIERE
> (bas de l'image), Y lateral, Z haut ; metres du modele (echelle compacte, chassis 2,10 m).

## 1. Hashes retenus (lus dans le tableau §2 de CONTACT_ARMES_WARTHOG, colonne n.)

| n. | arme | vehi | mode (pc:globals) | weap | sprite |
|---|---|---|---|---|---|
| 13 | LANCE-ROQUETTES | `0xbcfb852f` (warthog_b_g) | `0xbe74e831` | `0xc7d50912` (banque `veh_un_rockethog`) | `sprites_v4/rockethog.png` |
| 15 | MITRAILLEUSE / LAAG | `0xdd7f9102` (warthog_b_g_l) | `0xc0803caa` | `0x0c6fd911` (classe `turret`) | `sprites_v4/warthog.png` |
| 16 | CANON GAUSS | `0x64b925eb` (warthog_b_g) | `0x9c7f3b54` | `0x8647925a` (classe `fixed`) | `sprites_v4/warthog_gauss.png` |

Chassis : mode `0x561f2ca7`, permutation `default` de chaque region (43 sections, 2,100 x 1,008 x
0,769 m, X[-1,106..+0,995]) — sans les socles de region[17] ni les garde-boue/plaques des groupes V4.
Les hashes de memoire du brief etaient corrects (n.15 = `0xc0803caa`, n.16 = `0x9c7f3b54`).

## 2. Canevas et echelle

- Canevas fixe : cadre +-5 m, **10 mm/px**, 1012 x 1012 px, `Min = (-5,060, -5,060)`, origine locale
  (0,0) au pixel (505,5 ; 505,5). Pixel (px, py) -> local : `Y = Min0 + (px+0,5)*cell`,
  `X = -(Min1 + (NY-1-py+0,5)*cell)` (remap objet_isole `{Y, -X}`, nez en haut).
- Echelle calee sur `razorback.png` VALIDE : emprise opaque 101 x 211 px pour 1,008 x 2,117 m ->
  10,0 mm/px. Le chassis `default` fait 210 px de long dans les trois sprites (le Razorback a 211 px :
  sa plaque avant `unarmed` atteint X = -1,121 au lieu de -1,106). Largeur 101 px identique.

## 3. Detection du plateau (rectangle blanc arriere)

Masque = pixels avec alpha > 0 ET non noirs (R > 128), restreints a X local > 0 (moitie arriere),
1 erosion (4-voisins), composantes 4-connexes, tri par aire. Sur le chassis `default` a 10 mm/px :

| cand. | aire (px) | rect px (x, y) | taille | centre px | centre local (X, Y) | centroide local | Z med. | nature |
|---|---|---|---|---|---|---|---|---|
| 1 | 1257 | x[483..528] y[529..568] | 46 x 40 | (505,5 ; 548,5) | (+0,430 ; 0,000) | (+0,398 ; -0,005) | 0,674 | interieur du V du couvercle-moteur (surface haute) |
| 2 | 1171 | x[469..540] y[532..589] | 72 x 58 | (504,5 ; 560,5) | (+0,550 ; -0,010) | (+0,651 ; -0,073) | 0,569 | U du plancher du pont, autour du V |
| 3 | 199 | x[529..542] y[532..562] | 14 x 31 | (535,5 ; 547,0) | (+0,415 ; +0,300) | — | 0,578 | bande laterale droite |
| 4 | 38 | x[478..496] y[517..519] | 19 x 3 | — | (+0,125 ; -0,185) | — | 0,242 | fond de siege |
| 5 | 36 | x[504..507] y[507..519] | 4 x 13 | — | (+0,075 ; 0,000) | — | 0,720 | console centrale |

**Hesitation et choix.** Les deux premieres composantes ont des aires voisines (1257 / 1171) et sont
imbriquees : le pont arriere est coupe par les deux traits en V du couvercle-moteur (surface a Z 0,67
au milieu, plancher a Z 0,57 autour). Aucune des deux n'est « le rectangle » a elle seule : la 1 est
un trapeze, la 2 un U. Le rectangle blanc percu par l'utilisateur (« la plus grande aire sans rien »,
entre les passages de roue arriere, du dossier des sieges au pare-chocs) est leur **reunion**. Regle
implementee (`-union=0.6`) : les composantes dont l'aire atteint 60 % de la premiere sont reunies ;
le plateau = rectangle englobant de la reunion, Z = la plus haute des medianes.

**PLATEAU retenu** : rect x[469..540] y[529..589] = **0,72 x 0,61 m** ; centre
**(cx, cy) = (504,5 ; 559,0) px = local (X = +0,535 m, Y = -0,010 m)** ; Z = 0,674.
Meme (cx, cy) pour les trois armes.

Robustesse (centre X / Y en m) : 10 mm/px erosion 0 -> +0,535 / +0,005 (sans erosion, les deux zones
restent separees par les traits) ; erosion 1 -> +0,535 / -0,010 (retenu) ; erosion 2 -> +0,535 / -0,015.
A 8 mm/px erosion 1 -> +0,532 / -0,068 : le U se scinde (sa moitie droite passe sous le seuil), d'ou un
biais lateral de 6 cm — c'est la faiblesse connue de la regle ; a 10 mm/px elle ne se produit pas et le
resultat est symetrique a 1 px pres.

Verification a l'oeil (planche, ligne 1, tuile 2) : le rectangle vert couvre bien le pont arriere, pas
le capot (moitie avant exclue) ni le toit (les plaques de toit sont dans la moitie avant, X ~ +0,36
n'est pas concerne : elles ne sont pas dans la permutation `default`).

## 4. Pose des armes (repere de l'enfant -> repere du chassis)

Centre d'arme = centroide XY de la **tranche basse** (sommets avec Z <= Zmin + 0,08 m) = la base
ronde / le socle de l'objet. Translation `T = (X_plateau - base_x, Y_plateau - base_y, Z_plateau - Zmin)`.

| arme | emprise isolee X / Y / Z | base : n sommets, centre (X, Y) | centre d'emprise | pivot `0x99d45ed9` | **T (m)** | emprise posee X / Y / Z |
|---|---|---|---|---|---|---|
| rockethog `0xbe74e831` | [-0,176..+0,404] / [-0,292..+0,292] / [-0,004..+0,687] | 1652, (+0,065 ; -0,001) | (+0,114 ; 0,000) | (+0,277 ; 0 ; +0,380) | **(+0,470 ; -0,009 ; +0,678)** | [+0,294..+0,873] / [-0,302..+0,282] / [0,674..1,365] |
| warthog `0xc0803caa` | [-0,193..+0,761] / [-0,152..+0,199] / [+0,001..+0,653] | 795, (-0,008 ; 0,000) | (+0,284 ; +0,023) | (+0,281 ; 0 ; +0,417) | **(+0,543 ; -0,010 ; +0,673)** | [+0,351..+1,304] / [-0,162..+0,189] / [0,674..1,326] |
| warthog_gauss `0x9c7f3b54` | [-0,230..+0,719] / [-0,386..+0,219] / [+0,001..+0,721] | 728, (-0,037 ; -0,001) | (+0,245 ; -0,084) | (+0,273 ; 0 ; +0,401) | **(+0,572 ; -0,009 ; +0,673)** | [+0,342..+1,290] / [-0,395..+0,210] / [0,674..1,394] |

- Pourquoi la base et non le pivot : la base ronde est centree a X ~ 0 du repere de l'arme, 28 cm
  DEVANT le pivot `0x99d45ed9` (+0,27..0,28). Centrer le pivot aurait decale l'arme visible de 28 px
  vers l'avant, hors du rectangle. Le centre d'emprise (qui inclut le canon) n'est pas retenu non plus :
  il aurait recule la LAAG/le Gauss de ~0,29 m et mis le fut au-dela du pare-chocs.
- Controle par construction : le centre de base pose retombe au pixel (504,5 ; 559,0) = centre du
  plateau, **ecart (0,00 ; 0,00) cellule** pour les trois (imprime par le driver, croix rouge sur la
  planche). Le rocket pod (0,58 x 0,58) remplit le pont (X +0,29..+0,87 pour un pont +0,26..+0,84).
- Canon de la LAAG et du Gauss : depasse le pare-chocs (X = +0,995) de 0,31 / 0,30 m — attendu.
- Z : la base est posee a 0,674 (couvercle-moteur, la surface la plus haute sous l'arme) et non a
  0,54-0,55 (sommet des socles region[17], absents du chassis `default`). En vue de dessus seul l'ordre
  peintre compte ; le Z reste coherent (aucune interpenetration avec le chassis, dont le max est 0,751
  a la console, hors du plateau).
- Ecart avec les poses precedentes : noeud n[006] a +0,765 (rejete) -> l'arme est maintenant 23 cm
  plus en avant ; T = 0 (planche-contact) -> 47-57 cm plus en arriere.

## 5. Assemblage et livrables

Ordre peintre : chassis rendu seul (canevas fixe), arme rendue seule au meme canevas avec `T`
(`PartAssemblage.Translation`), superposition source-over (arme au-dessus), puis rognage du composite
(marge 6 px). Pas de z-buffer partage (la LAAG passe au-dessus du chassis quoi qu'il arrive).

| fichier | taille | contenu |
|---|---|---|
| `sprites_v4/warthog.png` | 112 x 253 | chassis default + LAAG n.15 (ECRASE l'ancien 177 x 361 a 6 mm/px) |
| `sprites_v4/rockethog.png` | 112 x 222 | chassis default + roquettes n.13 (ECRASE l'ancien V4 127 x 250) |
| `sprites_v4/warthog_gauss.png` | 112 x 252 | chassis default + Gauss n.16 (ECRASE l'ancien V4 127 x 250) |
| `PLANCHE_WARTHOG_FINAL_2026-09-02.png` | 1984 x 3834 | Razorback valide + chassis avec les 3 composantes (orange/magenta/cyan) et le plateau (vert) ; par arme : isolee / assemblage + controle / sprite final ; zoom x3 |

Style identique au lot (remplissage blanc, aretes noires, alpha porteur de l'ombrage), meme echelle
(10 mm/px) que `razorback.png`. Les sprites sont rognes serres (l'ancien canevas 127 x 250 du lot V4
portait des marges plus larges) ; la longueur du chassis en pixels est la reference d'echelle.

## 6. CR honnete

- Ce qui est PROUVE : l'arme est au centre du rectangle detecte (ecart nul), le rectangle detecte est
  le pont arriere entre les roues (verifie a l'oeil, robuste a l'erosion 0/1/2 a 10 mm/px), l'echelle
  est celle du Razorback valide (210 vs 211 px).
- Ce qui est un CHOIX : (a) la reunion des deux composantes (seuil 60 %) — si l'utilisateur voulait le
  seul interieur du V, le centre serait a X = +0,430 (10 px plus en avant) ; (b) le centre de base
  plutot que le pivot ; (c) Z = 0,674.
- Ce qui n'est PAS fait : pas de socle region[17] sous l'arme (il serait cache par l'arme en vue de
  dessus, et son mapping variante <-> socle n'etait pas prouve) ; pas de garde-boue/plaques des
  groupes V4 (rejetes avec les sprites V4) ; le noeud d'attache reel du moteur reste non nomme (le
  placement est visuel, pas moteur).
- Faiblesse connue : a 8 mm/px la regle d'union scinde le U et biaise Y de 6 cm ; on reste a 10 mm/px.

## 7. Reproduction

Driver jetable (worktree, non commite) : `cmd/vs-measure/plateau.go` + `plateau_detect.go`
(sous-commande `plateau`, aiguillage 4 lignes dans `main.go`). `gofmt`, `go vet`, `go build` propres.
Aucun fichier de `internal/himap/*` ni `cmd/vehicle-sprite/*` modifie. Planche : script
PowerShell/System.Drawing dans le scratchpad (non versionne).

```
vsmeasure.exe plateau "-modules=pc:globals-rtx-new.module,any:globals-rtx-new.module,any:common-rtx-new.module,any:multiplayer-rtx-new.module,any:multiplayer_r1-rtx-new.module,any:multiplayer_r3-rtx-new.module" \
  "-chassis=0x561f2ca7" "-armes=warthog=0xc0803caa,rockethog=0xbe74e831,warthog_gauss=0x9c7f3b54" \
  "-out=OUT" "-cadre=5" "-cellmm=10" "-erode=1" "-union=0.6"
# (15 s ; sorties : chassis.png, chassis_detect.png, arme_<nom>_isolee.png, arme_<nom>_posee.png, <nom>.png, <nom>_detect.png)
```

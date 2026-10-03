# Warthog arme V2 — avant/arriere inverses, corrige par l'utilisateur (RAPPORT)

> Ecrit le 2026-09-02, worktree `LevelUp-wt-vehicules`. Aucun commit, aucun `git add`. Build isole
> (GOCACHE dedie), CGO winlibs, tout en avant-plan, un seul module 7 Go en RAM a la fois. Remplace
> `WARTHOG_FINAL_2026-09-02.md` (V1) dont la pose etait dans le MAUVAIS rectangle. Planche :
> `PLANCHE_WARTHOG_FINAL_V2_2026-09-02.png` (2640 x 3060). Hashes inchanges (tableau §2 de
> `CONTACT_ARMES_WARTHOG_2026-09-01.md`) : n.13 roquettes vehi `0xbcfb852f` -> mode `0xbe74e831` ;
> n.15 mitrailleuse vehi `0xdd7f9102` -> mode `0xc0803caa` ; n.16 gauss vehi `0x64b925eb` -> mode
> `0x9c7f3b54`. Chassis : mode `0x561f2ca7`, permutation `default` (43 sections).

> **MIS A JOUR V3 (2026-09-02, §7)** : le placement V2 est valide mais la rotation Z de 180 degres
> des armes n.15 / n.16 (§3) est REFUSEE par l'utilisateur : orientation authored conservee (canon
> vers l'AVANT). `warthog.png` et `warthog_gauss.png` regeneres (§7) ; rockethog et razorback inchanges.
> Planche V3 : `PLANCHE_WARTHOG_FINAL_V3_2026-09-02.png`.

## 0. Ce que l'utilisateur a corrige, et la decision qui en decoule

- V1 avait pose l'arme au centre d'un « plateau » detecte dans la moitie BASSE de l'image (X local
  > 0) : c'etait l'interieur du V du couvercle-moteur, donc le CAPOT. L'utilisateur : « tu as
  confondu l'avant et l'arriere du vehicule, je te parle bien du rectangle EN HAUT de l'image ».
- Lecture du rendu (remap `{Y, -X}`, `+X` en bas) : le HAUT de l'image (X < 0) porte le grand
  rectangle blanc du plateau, entre les passages de roue ; le BAS (X > 0) porte le capot en V et le
  pare-chocs anguleux. Disposition reelle du Warthog : capot devant, sieges au milieu, plateau plat
  avec la tourelle derriere.
- **DECISION : pour la famille Warthog, `+X` modele = AVANT** (inverse du Scorpion ou `+X` = arriere
  avait ete valide). Le sens de X n'est donc PAS une convention universelle des `mode` : il est par
  modele. Consequence : le Razorback (meme chassis, valide en forme) etait lui aussi rendu nez en
  bas -> il est pivote de 180 degres comme les trois autres pour respecter la convention nez-en-haut
  du lot (celle que la rotation du rejeu suppose).
- L'en-tete de `cmd/vs-measure/main.go` disait « +X = ARRIERE » : corrige (sens de X par modele).

## 1. Canevas (inchange) et chassis

- Canevas fixe : cadre +-5 m, **10 mm/px**, 1012 x 1012 px, `Min = (-5,060, -5,060)`, origine locale
  (0,0) au pixel (505,5 ; 505,5). Symetrique autour de l'origine (verifie par le driver :
  `symetrique=true`) — c'est ce qui rend la rotation d'image de l'arme exacte (§3).
- Pixel (px, py) -> local : `Y = Min0 + (px+0,5)*cell`, `X = -(Min1 + (NY-1-py+0,5)*cell)`.
- Chassis `default` : 43 sections, 97 852 sommets, X[-1,106..+0,995] Y[-0,504..+0,504]
  Z[-0,018..+0,751]. Emprise opaque au canevas : **100 x 210 px** = x[456..555] y[395..604]
  (2,10 m de long). Razorback valide : 101 x 211 px (sa plaque avant `unarmed` depasse de 1 px).

## 2. Detection du rectangle du HAUT (moitie X < 0)

Meme methode que V1 : masque = pixels alpha > 0 ET non noirs (R > 128), restreints a **X local < 0**
(`-xplateau=-1`), erosion 1 px (4-voisins), composantes 4-connexes, tri par aire, regle d'union
`-union=0.6` (composantes >= 60 % de la premiere reunies).

| cand. | aire (px) | rect px (x, y) | taille | centre px | centre local (X, Y) | centroide local | Z med. | nature |
|---|---|---|---|---|---|---|---|---|
| 1 | 1634 | x[486..525] y[422..472] | 40 x 51 | (505,5 ; 447,0) | (-0,585 ; 0,000) | (-0,589 ; 0,000) | 0,420 | **interieur du grand rectangle du plateau arriere** |
| 2 | 240 | x[532..545] y[431..464] | 14 x 34 | (538,5 ; 447,5) | (-0,580 ; +0,330) | (-0,582 ; +0,339) | 0,545 | bande laterale droite, entre le cadre du rectangle et la roue |
| 3 | 221 | x[487..512] y[408..417] | 26 x 10 | (499,5 ; 412,5) | (-0,930 ; -0,060) | (-0,929 ; -0,062) | 0,617 | bande sur le pare-chocs arriere |
| 4 | 111 | x[466..470] y[436..465] | 5 x 30 | (468,0 ; 450,5) | (-0,550 ; -0,375) | (-0,559 ; -0,379) | 0,541 | bande laterale gauche |
| 5 | 88 | x[540..547] y[409..431] | 8 x 23 | (543,5 ; 420,0) | (-0,855 ; +0,380) | (-0,856 ; +0,371) | 0,539 | coin arriere droit |

**PLATEAU retenu = candidat 1 seul** (le 2e fait 15 % du 1er, sous le seuil de 60 % : aucune reunion
necessaire cette fois, le rectangle n'est pas coupe par un trait interieur). Rect x[486..525]
y[422..472] = **0,40 x 0,51 m** = local X[-0,835..-0,335] Y[-0,195..+0,195] ; centre
**(cx, cy) = (505,5 ; 447,0) px = local (X = -0,585 m, Y = 0,000 m)** ; Z = 0,420 (mediane).
Meme (cx, cy) pour les trois armes.

Verification a l'oeil (planche, bloc A, tuile 1) : le rectangle vert est bien le grand rectangle blanc
du haut, a l'interieur de son cadre noir ; les bandes 2 et 4 sont a l'exterieur du cadre (entre le
cadre et les roues). Les reunir ne bougerait pas le centre : elles sont symetriques (Y +0,33 / -0,375,
meme intervalle en X) — Y resterait 0,000 et X -0,585.

Robustesse (centre X / Y en m, 10 mm/px) : erosion 0 -> -0,575 / 0,000 (42 x 55 px, une rangee de
plus en bas) ; erosion 1 -> **-0,585 / 0,000** (retenu) ; erosion 2 -> -0,585 / 0,000 (38 x 49 px).
Ecart max 1 px en X, 0 en Y.

## 3. Pose des armes : rotation Z de 180 degres, puis translation

Les trois armes sont authored **canon vers +X** (emprise X de -0,19 a +0,76 pour la LAAG, base ronde
a X ~ 0) : posees telles quelles, le canon pointerait vers le BAS de l'image = vers l'AVANT du
Warthog. **Elles sont donc pivotees de 180 degres autour de Z** (canon vers -X = arriere = haut de
l'image, debordant le pare-chocs ARRIERE), roquettes comprises (pod symetrique, meme regle).

Mise en oeuvre : `himap.PartAssemblage` n'offre qu'une translation. L'arme est rendue seule au
canevas fixe avec la translation `T_rendu = -T` (XY), puis son IMAGE est pivotee de 180 degres
autour du centre de la grille ; le canevas etant symetrique autour de l'origine locale, cela vaut
exactement `(X, Y) -> (-X, -Y)` en coordonnees modele. Le point rendu `p + T_rendu` devient
`-(p + T_rendu) = -p + T` : c'est la rotation Z de 180 degres suivie de la translation `T`.
Centre d'arme = centroide XY de la tranche basse (Z <= Zmin + 0,08 m) ; `T = (X_plateau + base_x,
Y_plateau + base_y, Z_plateau - Zmin)` (signe + a cause de la rotation).

| arme | emprise isolee X / Y / Z | base : n sommets, centre (X, Y) | pivot `0x99d45ed9` | rotation Z | **T (m), apres rotation** | emprise posee X / Y / Z |
|---|---|---|---|---|---|---|
| rockethog `0xbe74e831` | [-0,176..+0,404] / [-0,292..+0,292] / [-0,004..+0,687] | 1652, (+0,065 ; -0,001) | (+0,277 ; 0 ; +0,380) | 180 deg | **(-0,520 ; -0,001 ; +0,424)** | [-0,923..-0,344] / [-0,292..+0,292] / [0,420..1,111] |
| warthog `0xc0803caa` | [-0,193..+0,761] / [-0,152..+0,199] / [+0,001..+0,653] | 795, (-0,008 ; 0,000) | (+0,281 ; 0 ; +0,417) | 180 deg | **(-0,593 ; 0,000 ; +0,420)** | [-1,354..-0,401] / [-0,199..+0,152] / [0,420..1,073] |
| warthog_gauss `0x9c7f3b54` | [-0,230..+0,719] / [-0,386..+0,219] / [+0,001..+0,721] | 728, (-0,037 ; -0,001) | (+0,273 ; 0 ; +0,401) | 180 deg | **(-0,622 ; -0,001 ; +0,419)** | [-1,340..-0,392] / [-0,220..+0,385] / [0,420..1,140] |

- Canon de la LAAG / du Gauss : deborde le pare-chocs arriere (X = -1,106) de 0,25 / 0,23 m — vers
  le HAUT du rendu brut, vers le BAS du sprite final.
- Le Gauss est asymetrique (bloc a Y -0,39 dans son repere) : apres rotation Z le bloc est a Y +0,39
  (a DROITE du rendu brut, bloc A) ; apres la rotation finale d'image il revient a GAUCHE du sprite
  (bloc B), exactement comme sur l'arme authored : deux rotations de 180 degres, jamais de miroir.
- Z : base posee a 0,420 (mediane du z-buffer sur le rectangle) ; ordre peintre (arme au-dessus du
  chassis), aucune interpenetration a gerer en vue de dessus.
- Controle par construction, verifie sur le pixel pivote : centre de base pose au pixel
  (505,5 ; 447,0) = centre du rectangle, **ecart (0,00 ; 0,00) cellule** pour les trois (croix rouge
  sur croix verte, planche bloc A et rappels bloc B).

## 4. Rotation finale de 180 degres et livrables

Sprite = composite (chassis puis arme) rogne (marge 6 px) puis **pivote de 180 degres dans le plan
de l'image** : `(x, y) -> (W-1-x, H-1-y)` — rotation (deux miroirs composes), pas un miroir.
Meme operation sur `razorback.png` (lu, pivote, reecrit ; 127 x 250, opaque 101 x 211 inchanges).
Le sens est le meme pour les quatre (un demi-tour n'a pas de sens).

| fichier | taille | chassis (px) | contenu / position dans le sprite |
|---|---|---|---|
| `sprites_v4/warthog.png` | 112 x 246 | 210 (y[6..215]) | chassis default + LAAG n.15, canon vers le bas ; centre du rectangle a (55,5 ; 163,0) |
| `sprites_v4/rockethog.png` | 112 x 222 | 210 (y[6..215]) | chassis default + roquettes n.13 ; centre du rectangle a (55,5 ; 163,0) |
| `sprites_v4/warthog_gauss.png` | 112 x 245 | 210 (y[6..215]) | chassis default + Gauss n.16, canon vers le bas, bloc a gauche ; centre du rectangle a (55,5 ; 163,0) |
| `sprites_v4/razorback.png` | 127 x 250 | 211 | Razorback valide, pivote de 180 degres (nez en haut) |
| `PLANCHE_WARTHOG_FINAL_V2_2026-09-02.png` | 2640 x 3060 | — | bloc (A) orientation vue par l'utilisateur (arriere en haut) : chassis + rectangle vert, 3 assemblages + controle ; bloc (B) 4 sprites finaux nez en haut + rappels du rectangle ; longueurs de chassis etiquetees |

Les quatre ecrasent les fichiers precedents (V1 pour les 3 armes ; Razorback nez en bas).
Rognages (canevas) : warthog x[450..561] y[365..610] ; rockethog y[389..610] ; gauss y[366..610] —
le bas du rognage est le meme (pare-chocs avant + 6 px), d'ou un chassis a y[6..215] dans les trois
sprites finaux, et un centre de rectangle a la meme ligne 163.

## 5. CR honnete

- PROUVE : le rectangle detecte est le grand rectangle blanc du HAUT (vu a l'oeil, robuste a
  l'erosion 0/1/2 : X a 1 px pres, Y = 0 exactement) ; l'arme est centree dessus (ecart nul) ; canon
  vers l'arriere ; les quatre sprites ont le meme chassis 210/211 px a 10 mm/px et la meme rotation.
- CHOIX : (a) candidat 1 seul (l'interieur du cadre noir) plutot que le rectangle cadre compris —
  meme centre ; (b) centre de base plutot que pivot `0x99d45ed9` (28 cm devant la base, hors du
  rectangle) ; (c) Z = 0,420 (sans consequence en vue de dessus).
- LIMITE de la rotation par image : l'eclairement (`LumiereRendu`, direction fixe) tourne avec
  l'image. L'arme subit deux rotations (Z puis finale) donc garde son ombrage authored ; le chassis
  n'en subit qu'une : son ombrage est inverse par rapport aux autres vehicules du lot. L'alpha ne
  varie que dans [0,80 ; 1], l'effet est subtil ; la seule alternative propre serait une rotation de
  maillage dans `himap` (interdit dans ce lot). Le Razorback pivote a la meme inversion d'ombrage.
- NON FAIT : pas de socle region[17] sous l'arme ; pas de garde-boue/plaques V4 ; le noeud
  d'attache moteur reste non nomme (placement visuel). Aucune modification de `internal/himap/*` ni
  de `cmd/vehicle-sprite/*`.
- V1 (`WARTHOG_FINAL_2026-09-02.md`) est conserve comme trace de l'erreur ; ses chiffres de plateau
  (+0,535) decrivent le capot, pas le plateau.

## 6. Reproduction

Driver jetable (worktree, non commite) : `cmd/vs-measure/plateau.go` + `plateau_detect.go`
(nouveaux drapeaux `-xplateau`, `-rotarme`, `-rot180`, `-pivote`). `gofmt`, `go vet`, `go build`
propres. Planche : `planche_v2.ps1` (System.Drawing) dans le scratchpad, non versionne.

```
vsmeasure.exe plateau "-modules=pc:globals-rtx-new.module,any:globals-rtx-new.module,any:common-rtx-new.module,any:multiplayer-rtx-new.module,any:multiplayer_r1-rtx-new.module,any:multiplayer_r3-rtx-new.module" \
  "-chassis=0x561f2ca7" "-armes=warthog=0xc0803caa,rockethog=0xbe74e831,warthog_gauss=0x9c7f3b54" \
  "-out=OUT" "-cadre=5" "-cellmm=10" "-erode=1" "-union=0.6" "-xplateau=-1" "-rotarme" "-rot180" \
  "-pivote=.ai/V7.5/film_re/sprites_v4/razorback.png"
# (~15 s ; sorties : chassis.png, chassis_detect.png, arme_<nom>_isolee.png, arme_<nom>_posee.png,
#  <nom>.png (pivote), <nom>_detect.png (brut), <nom>_detect_rot.png (pivote), razorback.png (pivote))
```

## 7. V3 : arme non pivotee (correction utilisateur, 2026-09-02)

L'utilisateur valide le placement V2 (« parfait ») et le rockethog n.13 tel quel, mais REFUSE la
rotation de 180 degres autour de Z appliquee en §3 aux armes n.15 (mitrailleuse) et n.16 (gauss) :
« pourquoi as-tu fait pivoter la tourelle ? Le placement est parfait mais tu as change
l'orientation ». L'orientation AUTHORED est la bonne : canon vers +X modele = vers l'AVANT du
Warthog, par-dessus l'habitacle (position de repos de la LAAG et du Gauss). V2 avait deduit
« canon vers l'arriere » du seul fait que le plateau est a l'arriere : sur-interpretation. La
phrase de §3 « posees telles quelles, le canon pointerait vers l'AVANT » decrivait donc le bon
resultat, pas un defaut.

**Ce qui est refait** : n.15 (vehi `0xdd7f9102` -> mode `0xc0803caa`) et n.16 (vehi `0x64b925eb`
-> mode `0x9c7f3b54`) avec EXACTEMENT la chaine V2 — canevas cadre 5 m / 10 mm/px (1012 x 1012,
symetrique), meme rectangle du haut (cand. 1, 40 x 51 px, centre (505,5 ; 447,0) px = local
X -0,585 / Y 0,000 / Z 0,420, identique au run V2 a l'octet pres : memes 5 candidats), centre de
base = centroide de la tranche basse 8 cm pose sur le centre du rectangle, ordre peintre, rognage
(marge 6 px), rotation finale de 180 degres de l'image (rotation, pas miroir) — mais
**`-rotarme=false`** : translation seule, l'arme n'est pas pivotee. Le drapeau `-rotarme` existe
toujours (defaut bascule a `false`, en-tete de `plateau.go` corrige), il n'est pas active.
`rockethog.png` et `razorback.png` ne sont PAS touches (valides).

| arme | base : centre (X, Y) | rotation Z | **T (m), V3** | T V2 (rappel) | emprise posee X / Y / Z |
|---|---|---|---|---|---|
| warthog `0xc0803caa` | (-0,008 ; 0,000) | aucune | **(-0,577 ; 0,000 ; +0,420)** | (-0,593 ; 0,000 ; +0,420) | [-0,769..+0,184] / [-0,152..+0,199] / [0,420..1,073] |
| warthog_gauss `0x9c7f3b54` | (-0,037 ; -0,001) | aucune | **(-0,548 ; +0,001 ; +0,419)** | (-0,622 ; -0,001 ; +0,419) | [-0,778..+0,170] / [-0,385..+0,220] / [0,420..1,140] |

- T = plateau - base (sans le retournement du centre de base) : l'ecart avec V2 est de
  2 x base_x = 1,6 cm (LAAG) / 7,4 cm (Gauss) en X, 0 / 0,2 cm en Y — attendu.
- Controle : centre de base pose au pixel (505,5 ; 447,0) = centre du rectangle, **ecart
  (0,00 ; 0,00) cellule** pour les deux.
- Le canon ne deborde plus le pare-chocs : il s'arrete a X +0,18 / +0,17 (au-dessus des sieges),
  loin du pare-chocs avant (+0,995). Les deux sprites font donc **112 x 222 px** (V2 : 246 / 245),
  rognage canevas x[450..561] y[389..610] = celui du rockethog ; chassis 210 px a y[6..215],
  centre du rectangle dans le sprite a (55,5 ; 163,0) — identiques au rockethog.
- Gauss : bloc lateral a Y -0,39 dans le repere de l'arme (a GAUCHE du rendu brut, arriere en
  haut) -> apres la rotation finale, a DROITE du sprite nez en haut = cote -Y du vehicule. Une
  seule rotation de 180 degres (l'image), jamais de miroir : la chiralite authored est conservee.
  (En V2 le bloc etait a gauche du sprite final parce que l'arme subissait une rotation Z de plus.)
- Verifie a l'oeil (planche `PLANCHE_WARTHOG_FINAL_V3_2026-09-02.png`, 2640 x 3010) : bloc (A)
  orientation brute arriere en haut, arme non pivotee au milieu du rectangle vert, croix rouge sur
  croix verte, canon vers le BAS (= avant) ; bloc (B) sprites finaux nez en haut, arme en bas au
  centre du rectangle, canon vers le HAUT par-dessus l'habitacle ; rockethog et razorback inchanges
  a cote ; rappel des rectangles pivotes et Gauss authored isole (reference de chiralite).

| fichier | taille | chassis (px) | contenu |
|---|---|---|---|
| `sprites_v4/warthog.png` | 112 x 222 | 210 (y[6..215]) | chassis default + LAAG n.15 non pivotee, canon vers le HAUT (avant) |
| `sprites_v4/warthog_gauss.png` | 112 x 222 | 210 (y[6..215]) | chassis default + Gauss n.16 non pivote, canon vers le HAUT, bloc a droite |
| `sprites_v4/rockethog.png` | 112 x 222 | 210 | INCHANGE (V2) |
| `sprites_v4/razorback.png` | 127 x 250 | 211 | INCHANGE (V2) |

Reproduction (memes modules et drapeaux que §6, sans `-pivote`, `-rotarme=false`) :

```
vsmeasure.exe plateau "-modules=..." "-chassis=0x561f2ca7" "-armes=warthog=0xc0803caa,warthog_gauss=0x9c7f3b54" \
  "-out=OUT" "-cadre=5" "-cellmm=10" "-erode=1" "-union=0.6" "-xplateau=-1" "-rotarme=false" "-rot180"
```

Lecon : ne pas deduire l'orientation d'une arme de la position de son point d'attache ;
l'orientation authored du `mode` enfant fait foi tant qu'aucune donnee (squelette, animation de
repos) ne dit le contraire. Les §3 et §4 ci-dessus restent la trace de V2 ; pour n.15 / n.16 c'est
cette section qui fait foi.

# Rework Warthog + Gungoose — enumeration, mesures, diagnostic, correctif (RAPPORT)

> Ecrit le 2026-09-01, worktree `LevelUp-wt-vehicules`. Aucun commit, aucun `git add`.
> Build isole (GOCACHE dedie), CGO winlibs, tout en avant-plan. Repond au retour utilisateur :
> Warthog « LAAG (via `warthog_g` mode `0x0000e0da`) trop grosse / mauvais asset / mal placee » ;
> Gungoose « manque-t-il les lance-missiles avant, possible qu'ils ne depassent pas a l'avant ? ».
> Integre la directive coordinateur (rapport Ghidra `GHIDRA_ATTACHEMENT_VEHICULE_2026-09-01.md` :
> attache = noeud nomme -> transformee {echelle, rotation, translation}).

Repere (convention rectifiee `objet_isole.go`) : X longitudinal, **+X = ARRIERE, -X = AVANT** ;
Y lateral ; Z haut. Modeles en unites COMPACTES (~0,46x l'echelle lore) : chassis warthog = 2,24 m.

## TL;DR — verdicts

- **WARTHOG = MAUVAIS ASSET, PAS un probleme de transformee de noeud.** Mesure decisive : les
  **106 noeuds du squelette du chassis ont TOUS une echelle = 1,000** (locale ET model-space).
  Il n'y a AUCUNE echelle de noeud a appliquer -> l'hypothese « tourelle trop grosse = echelle de
  noeud ignoree » est REFUTEE pour le Warthog. La cause « trop grosse » est l'asset lui-meme :
  `warthog_g 0x0000e0da` mesure **dX = 2,391 m** (plus long que le chassis entier 2,244 m),
  centroide **cX = +1,738** (au-dela de l'arriere du chassis a +1,123), 3 sections seulement.
  C'est le barillet de la LAAG modelise DEPLOYE vers l'arriere. Compose a translation nulle il
  deborde massivement -> « trop grosse et mal placee ».
- **Correctif Warthog : la LAAG est DEJA dans le render_model du chassis, en PERMUTATION.** Le
  mode `0x561f2ca7` porte la tourelle chaingun/LAAG en permutation `0x06c86db1` (region[17],
  sections 84-85 : **0,26 x 0,76 m, cX = +0,93 = arriere-centre**), co-reperee par construction,
  a la bonne echelle et au bon endroit. C'est la piste « silhouette » vers laquelle la directive
  coordinateur demande de revenir quand le noeud vaut l'identite. `sprites_v4/warthog.png` =
  variante `0x06c86db1` (chassis + LAAG integree), pas d'enfant compose.
- **GUNGOOSE = les pods avant NE MANQUENT PAS.** Ce sont une **PERMUTATION** (`0x02c9ed0a`) du
  render_model du mongoose (`0x9e581380`), pas un objet-enfant separe (aucun `gungoose_g` /
  `wargoose_g` au scan). Sections d'arme : 20,23,35,38 (avant, cY gauche/droite) + 30,43,46
  (arriere). Les pods AVANT sont a cX = -0,45, portant jusqu'a X ~ -0,64 ; le nez du chassis est a
  X = -0,655 -> **les pods atteignent le nez mais ne DEPASSENT quasi pas devant** (fidele au
  modele : c'est exactement l'observation de l'utilisateur). `sprites_v4/gungoose.png` regenere a
  echelle superieure (cellmm=6) : les deux pods avant sont VISIBLES.

## 1. Enumeration des candidats (scan)

`scan` Passe 1 (`pc:globals` + `any`) et Passe 2 (`pc:multiplayer`+`pc:common`+`any`). Famille
warthog (arme) et mongoose/gungoose :

| vehi | nom maillage | mode resolu | module | note |
|---|---|---|---|---|
| `0x0000e0ca` | warthog_g | `0x0000e0da` | pc/multiplayer | **LAAG actuelle (rejetee)** |
| `0x1779ea58` | warthog_g | `0x0261f134` | pc/multiplayer | variante |
| `0x00409dac` | warthog_b_g/warthog_g | `0x00409881` | pc/multiplayer | petit |
| `0x4ccc20e6` | warthog_b_g/warthog_g | `0x6b17fdb5` | pc/multiplayer | moyen |
| `0x64b925eb` | warthog_b_g | `0x9c7f3b54` | pc/globals | |
| `0xbcfb852f` | warthog_b_g | `0xbe74e831` | pc/globals | compact |
| `0xdd7f9102` | warthog_b_g_l | `0xc0803caa` | pc/globals | barillet etroit |
| `0x3a8060e2` | turret_g | `0x1c645961` | pc/mp+common | generique (V4 supposait « gungoose ») |
| `0x003f00c7` | turret_g | `0x1c8f09d8` | pc/mp+common | generique (V4 supposait « rockethog ») |
| `0x038df01a` | turret_g | `0x1ae526e1` | pc/mp+common | tourelle_montee (Shade-like) |
| `0x000025aa` | mongoose_p | `0x9e581380` | pc/globals | chassis mongoose |
| `0xaf31ab1a`,`0xde26e3d7` | mongoose | `0x9e581380` | pc/globals | variantes chassis |

**Aucun `gungoose_g` / `wargoose_g`** parmi les 67 vehi -> l'arme du gungoose n'est pas un
objet-enfant, c'est une permutation (confirme §5).

## 2. Mesures (boite englobante + centroide, metres, repere modele)

Outil : `vsmeasure` (cf. §7), agrege TOUS les sommets d'un `mode`.

| mode | ident | sections | dX (long) | dY (lat) | dZ (haut) | centroide (cX,cY,cZ) |
|---|---|---|---|---|---|---|
| `0x561f2ca7` | **chassis warthog** | 90 | 2,244 | 1,012 | 0,831 | (+0,091, +0,001, +0,286) |
| (perm) | **LAAG integree sec 84-85** | 2 | **0,26** | **0,76** | — | **cX = +0,93** (arriere) |
| `0x0000e0da` | **warthog_g (rejetee)** | 3 | **2,391** | 0,853 | 0,623 | **(+1,738**, +0,001, +0,183) |
| `0x0261f134` | warthog_g | 6 | 1,153 | 0,702 | 0,837 | (+0,367, -0,002, **+0,629**) |
| `0x00409881` | warthog_g | 1 | 0,454 | 0,225 | 0,427 | (+0,181, +0,001, +0,223) |
| `0x6b17fdb5` | warthog_g | 8 | 1,019 | 0,499 | 0,685 | (+0,130, 0, +0,235) |
| `0x9c7f3b54` | warthog_b_g | 6 | 0,949 | 0,605 | 0,720 | (+0,212, -0,113, +0,517) |
| `0xbe74e831` | warthog_b_g | 5 | 0,579 | 0,584 | 0,691 | (+0,138, -0,006, +0,463) |
| `0xc0803caa` | warthog_b_g_l | 9 | 0,954 | 0,351 | 0,652 | (+0,248, +0,012, +0,425) |
| `0x1c645961` | turret_g | 3 | 0,668 | 0,330 | 0,550 | (-0,038, -0,001, +0,401) |
| `0x1c8f09d8` | turret_g | 6 | 0,539 | 0,502 | 0,533 | (+0,019, -0,002, +0,250) |
| `0x1ae526e1` | tourelle_montee | 28 | 1,264 | 1,234 | 0,566 | (-0,030, +0,013, +0,326) |
| `0x9e581380` | chassis mongoose | 56 | 1,155 | 0,673 | 0,596 | (+0,026, -0,010, +0,224) |

Empreintes des sections tourelle du chassis warthog (region[17], arriere) : rockethog `0x13d24f1f`
= sec 80-81 (dx0,33 dy0,66) ; gauss `0xad03512a` = sec 82-83 (dx0,18 dy0,76) ; **chaingun/LAAG
`0x06c86db1` = sec 84-85 (dx0,26 dy0,76), cX +0,93**.

## 3. Diagnostic (a) / (b) / (c) du « trop grosse »

- **(a) mauvais asset — CONFIRME.** `warthog_g 0x0000e0da` = **dX 2,391 m** (barillet LAAG
  deploye vers l'arriere, cf. `V4 §... barillet ~2,5 m pointant vers l'arriere`), centroide
  cX +1,738 (hors du chassis). Ce n'est pas la LAAG « trepied compacte » attendue (~1-1,5 m lore
  ~ 0,5-0,7 m modele). C'est la cause premiere du rejet.
- **(b) echelle fausse — ECARTE.** Comparee au chassis (dX 2,244), la boite `0x0000e0da` n'est pas
  a une echelle d'unites differente : elle est simplement, en unites du modele, un asset trop long
  (barillet deploye). Les autres candidats sont a la meme echelle que le chassis (coherent).
- **(c) co-repérage / transformee de noeud — MESURE puis ECARTE comme cause du « trop grosse ».**
  Directive coordinateur (Ghidra) : verifier si le noeud d'attache porte une echelle != 1
  (« trop grosse ») + translation (« mal placee »). **Squelette du chassis dumpe : 106 noeuds,
  TOUS a echelle 1,000 (locale et model-space)** — aucune echelle a appliquer. Le noeud d'attache
  d'arme le plus plausible PAR POSITION (arriere-centre, sureleve) est `n[006]` StringId
  `0xe1a390ba`, transformee model-space = translation **(+0,765, 0,000, +0,541)**, rotation
  ~identite, **echelle 1,000** (candidats voisins : `n[010]` `0x27097f82` (+0,715,+0,005,+0,584) ;
  `n[005]`/`n[011]` (+0,809,±0,296,+0,523)). Aucun StringId de noeud ne resout aux noms d'arme
  candidats (gunner/turret/weapon/mount...). **Conclusion : la translation de noeud (~+0,77 m
  vers l'arriere) ne RETRECIT pas un asset de 2,39 m ; l'appliquer a `0x0000e0da` le placerait
  seulement plus loin en arriere (pire).** Le probleme est l'ASSET, pas la transformee — cas
  explicitement prevu par la directive (« si noeud = identite d'echelle -> mauvais asset ->
  reviens a la silhouette »).

## 4. Asset choisi (Warthog) : la LAAG integree en permutation

La bonne LAAG n'est pas un enfant a composer : elle est **deja dans le render_model du chassis**,
en permutation `0x06c86db1` (chaingun) de la region[17] (sections 84-85), **co-reperee** a
cX +0,93 (arriere-centre), 0,26 x 0,76 m — compacte et bien placee par construction. C'est la
LAAG que le jeu rend pour le warthog de base. Aucune transformee de noeud, aucune composition,
aucun risque de co-repérage.

`sprites_v4/warthog.png` = variante `0x06c86db1` rendue au cadre du modele (cellmm=6) puis rognee.
La difference mesuree avec la variante `default` (sans arme) est reelle (diff : ~5,3 % des pixels :
la tourelle 84-85 a l'arriere + les garde-boue blindes des regions 1-4). Regarde a l'oeil : silhouette
de warthog propre, LAAG compacte a l'arriere ; PLUS le gros bloc surdimensionne de l'ancienne
version. Corrige les DEUX defauts du rejet : surdimensionnement ET placement (l'ancienne LAAG
apparaissait a l'avant/au sommet).

Les autres enfants sont ecartes (mesure) : `0x0261f134` (1,15 m, flotte a cZ +0,63) ; `0x00409881`
(0,45 m mais 1 section) ; `0x6b17fdb5` / `warthog_b_g` (centroides cX +0,13..+0,25 = vers le
CENTRE, pas a l'arriere-centre du mount ; co-repérage different). `turret_g 0x1c64/0x1c8f/0x1ae5`
= emplacements generiques (centroides ~ origine), pas la LAAG.

## 5. Gungoose : permutation, pas d'enfant ; pods avant presents

- Mongoose `0x9e581380` a **2 variantes** : `default` (`0x42c9679f`) et `0x02c9ed0a` (= gungoose,
  la seule armee ; nom non resolu par murmur3 mais unique par elimination).
- Sections propres au gungoose : **20, 23, 35, 38 (AVANT**, cX -0,45, cY ±0,26/±0,29 = gauche/droite,
  barillets dx0,39 dy0,10 + montures dx0,36 dy0,15**) + 30, 43, 46 (arriere)**. Le nez du chassis
  (guidon, sec 6/8) est a cX -0,55 portant jusqu'a X -0,655 ; les pods avant portent jusqu'a
  X ~ -0,64 -> ils atteignent le nez **sans depasser franchement devant** (repond precisement a la
  question de l'utilisateur : rien ne manque, mais le modele ne les fait pas saillir loin a l'avant).
- Le meme raisonnement de noeud d'attache ne s'applique PAS : les pods etant une PERMUTATION
  (geometrie du render_model du chassis, deja en repere modele), il n'y a aucun objet-enfant a
  attacher, donc aucune transformee de noeud en jeu.
- `sprites_v4/gungoose.png` regenere : variante `0x02c9ed0a` a cellmm=6, rognee. A cette echelle
  les deux pods avant sont VISIBLES (structures boxy aux cotes avant, absentes du mongoose default —
  verifie a l'oeil, planche §Gungoose). L'ancien `gungoose.png` portait deja la meme permutation
  mais a une echelle ou les pods ne se lisaient pas.

## 6. Preuve moteur (Ghidra) confrontee a la mesure

Le rapport `GHIDRA_ATTACHEMENT_VEHICULE_2026-09-01.md` prouve le MECANISME : le moteur attache un
enfant a un noeud nomme via une transformee `{echelle uniforme, R3x3, T}` (compose `FUN_140474790`,
`resultat[0]=a[0]*b[0]`). J'ai porte ce mecanisme (`himap.Compose`, `NodeModelTransform`,
`quatVersRot`, teste) et parse le squelette (bloc racine +64, 124 o/noeud : name +0, parent +4,
pos +12, quaternion +24, **echelle +40**). Resultat sur le Warthog : **echelle = 1,000 sur les 106
noeuds** -> le mecanisme est reel mais **sa valeur est l'identite d'echelle ici** ; le symptome
« trop grosse » vient donc de l'asset, pas d'une echelle de noeud ignoree. Le crochet
`PartAssemblage` (objet_isole.go) n'a **volontairement PAS ete etendu** avec rotation+echelle :
aucun enfant du perimetre n'a de transformee de noeud non-identite a appliquer, l'etendre
ajouterait des branches mortes (regle « 0 code mort »). La brique reste disponible (`noeuds.go`,
testee) pour un futur vehicule dont un noeud d'attache serait non-identite.

## 7. Outils, code, fichiers

Code himap (worktree, NON commite) — brique reutilisable + test, autorisee par le coordinateur :
- `internal/himap/noeuds.go` — `ModeNodes` (parse le squelette), `Transforme`/`Compose`/
  `NodeModelTransform`/`quatVersRot` (composition model-space, reproduit `FUN_140474790`),
  `NoeudParNom`, `EchelleEstUnitaire`.
- `internal/himap/noeuds_test.go` — tests purs (quaternion->rotation, compose echelle/translation,
  transformee model-space d'une chaine, anti-cycle). `go test` OK ; `gofmt`/`go vet` propres.

Outil de diagnostic (worktree, NON commite ; ne touche pas `cmd/vehicle-sprite`) :
- `cmd/vs-measure/` — `-modes`/`-vehis` (boite+centroide agreges), `-nodes` (dump squelette +
  transformee model-space + repere echelle!=1 et noms d'arme). C'est le driver des mesures §2/§3.

Assets livres :
- `sprites_v4/warthog.png` — REGENERE : variante chaingun `0x06c86db1` (chassis + LAAG integree),
  177x361, blanc+traits noirs, teintable. ECRASE l'ancien (LAAG `0x0000e0da` surdimensionnee).
- `sprites_v4/gungoose.png` — REGENERE : variante `0x02c9ed0a` (mongoose + pods avant), 120x200.
- `PLANCHE_WARTHOG_GUNGOOSE_2026-09-01.png` — planche : (1) 6 candidats warthog a echelle commune
  150 px/m (hauteur = longueur reelle : `0x0000e0da` domine, LAAG integree minuscule et compacte) ;
  (2) warthog AVANT/APRES ; (3) gungoose mongoose/avant/apres/pods isoles.

Reproduire (depuis `apps/go-api`, GOCACHE dedie + CC winlibs + CGO_ENABLED=1) :
```
go build -o v5tool.exe ./cmd/vehicle-sprite
go build -o vsmeasure.exe ./cmd/vs-measure
# mesures + squelette
vsmeasure.exe -modules="pc:globals-rtx-new.module" -modes="0x561f2ca7,..." 
vsmeasure.exe -modules="pc:globals-rtx-new.module" -nodes="0x561f2ca7"      # 106 noeuds, echelle=1
# warthog final = permutation chaingun ; gungoose final = permutation gungoose
v5tool.exe variantes -modules="pc:globals-rtx-new.module" -id="0x561f2ca7,0x9e581380" -cellmm=6 \
  -map="0x561f2ca7:0x06c86db1:wh_chaingun,0x9e581380:0x02c9ed0a:gg_gungoose"
v5tool.exe compose2d -in="wh_chaingun.png" -out="warthog.png"   # rognage
v5tool.exe compose2d -in="gg_gungoose.png" -out="gungoose.png"
```

## 8. CR honnete

**Certain (mesure) :**
- `warthog_g 0x0000e0da` = 2,39 m, cX +1,74 : trop long, hors chassis. Cause du rejet.
- Les 106 noeuds du chassis warthog ont echelle 1,000 : pas d'echelle de noeud a l'origine du
  « trop grosse ». Le noeud de mount plausible (par position) est arriere-centre a +0,765 m, echelle 1.
- La LAAG correcte est la permutation `0x06c86db1` (sec 84-85), 0,26 x 0,76 m, cX +0,93 : compacte,
  arriere-centre, co-reperee.
- Le gungoose n'a pas d'enfant d'arme ; ses pods avant sont la permutation `0x02c9ed0a`, presents,
  atteignant le nez sans le depasser.

**Choix / limite (dit sans le maquiller) :**
- Le NOM du noeud d'attache d'arme du warthog n'est pas resolu (StringId `0xe1a390ba` non dans le
  vocabulaire ; les libelles `warthog_gunner*` du Ghidra sont des sieges, pas forcement le noeud du
  render_model). Je l'identifie par POSITION (arriere-centre, sureleve), pas par nom. Sans
  importance ici puisque son echelle vaut 1 et la LAAG integree suffit.
- Le warthog livre repose sur la PERMUTATION (la LAAG telle que le jeu la rend), pas sur un enfant
  compose via transformee de noeud : c'est la piste que la directive prescrit quand le noeud vaut
  l'identite d'echelle. La LAAG compacte est peu proeminente en vue de dessus (elle l'est aussi
  en jeu) — c'est fidele, et c'est l'inverse exact du defaut « trop grosse ».
- Le gungoose corrige porte la meme geometrie que l'ancien (permutation) a une echelle superieure :
  le « correctif » est la confirmation que les pods existent + un rendu ou ils se lisent, pas une
  geometrie nouvelle. Si l'utilisateur attend des pods SAILLANT loin devant, le modele ne les
  porte pas ainsi (fidelite > invention).

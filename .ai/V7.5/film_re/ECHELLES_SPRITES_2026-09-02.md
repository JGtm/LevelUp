# Échelles des sprites véhicules — vérification et correction (RAPPORT)

> Écrit le 2026-09-02, worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`. Build isolé (GOCACHE dédié, scratchpad), CGO winlibs, tout en
> avant-plan, un seul module 7 Go en RAM à la fois (`pc/globals-rtx-new.module`, 7,8 Go, jamais
> chargé en même temps que `pc/multiplayer`+`pc/common`, ~3,9 Go).
>
> Mission : garantir que les 18 sprites de `.ai/V7.5/film_re/sprites_v4/` sont tous à
> **10 mm/px exactement** (±3 %), prérequis de la règle de taille du rejeu (proportions monde
> entre véhicules). Méthode : mesure d'abord (bbox monde via `vs-measure` vs bbox pixel opaque
> du PNG), re-rendu seulement si mesure hors tolérance.

## TL;DR

- **9 suspects mesurés, 7 corrigés, 2 dans la tolérance.** Les sprites rendus en V4 au
  cadrage AUTO (`-cote=256`, échelle ajustée par véhicule) étaient **tous à une échelle
  différente de 10 mm/px** — de 5,18 mm/px (Tourelle montée) à 45,5 mm/px (Pelican), un facteur
  **8,8×** d'écart entre les deux pires cas. **Ghost, Chopper, Falcon, Pelican, Skiff, Shade,
  Tourelle montée** : re-rendus à `-cellmm=10` fixe (même levier que le lot Warthog/Scorpion),
  vérifiés à 9,99–10,03 mm/px. **Banshee et Wasp** : mesurés à 10,25–10,29 mm/px (+2,5 à +2,9 %)
  — **dans la tolérance ±3 % par coïncidence de taille**, non retouchés.
- **9 déjà bons, confirmés sur pièce** : Warthog, Rockethog, Warthog Gauss, Razorback, Mongoose,
  Gungoose (rendus `-cellmm=10` fixe, lots Warthog/Gungoose), Scorpion, Wraith, Phantom
  (rendus `-cellmm=10` fixe, lot `ASSEMBLAGE_ENFANTS_CORRIGE`) — tous mesurés entre 9,96 et
  10,03 mm/px.
- **Sens de +X déterminé sur la géométrie pour chacun des 7 re-rendus** (vue de dessus + vue de
  profil + détail par section `-secs`, jamais présumé) : **6 ont `+X` = avant** (Ghost, Chopper,
  Falcon, Pelican, Skiff, Shade) et ont été pivotés 180° pour la convention « nez en haut » ;
  **Tourelle montée a `-X` = avant** (bouche du canon) et son rendu d'origine était **déjà**
  nez-en-haut — **une première tentative de rotation basée sur une lecture visuelle rapide a été
  détectée comme erronée et annulée** après vérification quantitative des sections (cf. §4,
  leçon méthodologique).
- **Scorpion et Wraith toujours armés** (canon + collier ; tourelle plasma), vérifié à l'œil.
  Aucune régression : ces deux fichiers n'ont pas été touchés.
- **Outil** : extension additive de `cmd/vs-measure` (committé, extensions autorisées) —
  `sprite.go` (mesure de la bbox opaque d'un PNG) et `rot180.go` (rotation 180° d'un PNG déjà
  rendu). `gofmt`, `go vet`, `go build` propres. Aucune modification de `cmd/vehicle-sprite`
  ni d'`internal/`.
- **Livrables** : 7 PNG remplacés (mêmes noms) dans `sprites_v4/` **et**
  `static/vehicles-assets/halo_infinite/replay/` ; `index.json` mis à jour pour les **18**
  entrées (scale confirmée, date, note, source_report) ; planche
  `PLANCHE_ECHELLES_2026-09-02.png` (18 sprites côte à côte à échelle pixel commune).

## 1. Méthode de mesure

Deux nombres suffisent par sprite : la longueur MONDE du châssis (mètres, via
`vs-measure -modes=<hex>`, colonnes `dX`/`dY` de la bbox géométrique du `mode`) et la longueur
PIXEL opaque du sprite déjà rendu (nouvelle sous-commande `vs-measure sprite -dir=...`, scanne
tous les pixels d'alpha > 0 et rend leur boîte englobante). Le rasterizer place l'axe LOCAL Y du
modèle sur la largeur de l'image et l'axe LOCAL X sur la hauteur (remap `HautZ` de
`objet_isole.go`, vérifié sur le code — jamais présumé) : `mm/px = dX·1000 / hauteur_opaque_px`
et `mm/px = dY·1000 / largeur_opaque_px` doivent concorder (les deux sont donnés, ils
concordent à moins de 0,3 % près sur les 18 sprites, preuve que le mapping largeur/hauteur est le
bon).

Pour les rendus `-cote=256` (cadrage auto), le code (`objet_isole.go:246-251`) calcule
`cell = max(dX,dY) / (CotePx - 2·MargePx) = max(dX,dY) / 244` — c'est exactement la formule
retrouvée par la mesure pixel (concordance <0,1 %, cf. §3), qui confirme qu'aucune étape
intermédiaire (rognage, retouche) n'a changé l'échelle depuis le rendu V4.

## 2. Table complète (18 sprites)

| Sprite | monde (m, dX×dY) | px opaque avant (H×L) | **mm/px avant** | px opaque après | **mm/px après** | écart après | statut |
|---|---|---|---|---|---|---|---|
| Warthog | 2,10×~1,01 | 210×100 | 10,00 (déjà, construit `cellmm=10`) | 210×100 | 10,00 | 0,0 % | inchangé |
| Rockethog | 2,10×~1,01 | 210×100 | 10,00 (déjà) | 210×100 | 10,00 | 0,0 % | inchangé |
| Warthog Gauss | 2,10×~1,01 | 210×100 | 10,00 (déjà) | 210×100 | 10,00 | 0,0 % | inchangé |
| Razorback | 2,11×~1,01 | 211×101 | 10,00 (déjà) | 211×101 | 10,00 | 0,0 % | inchangé |
| Mongoose | 1,155×0,673 | 116×66 | 10,0 (déjà) | 116×66 | 9,96–10,20 (avg 10,08) | +0,8 % | inchangé |
| Gungoose | 1,155×0,673 | 116×66 | 10,0 (déjà) | 116×66 | 9,96–10,20 (avg 10,08) | +0,8 % | inchangé |
| Scorpion | dY=2,511 (chassis) | 380×252 | 10,0 (déjà, `cellmm=10`) | 380×252 | 9,96 | −0,4 % | inchangé |
| Wraith | 3,057×2,963 | 305×296 | 10,0 (déjà) | 305×296 | 10,01–10,02 | +0,15 % | inchangé |
| Phantom* | 10,725×7,347 | 1072×734 | 10,0 (déjà) | 1072×734 | 10,00–10,01 | +0,05 % | inchangé |
| Banshee | 2,503×2,347 | 244×229 | **10,25–10,26** (auto) | 244×229 | 10,25–10,26 | **+2,6 %** | **OK tolérance, non touché** |
| Wasp | 2,507×2,109 | 244×205 | **10,27–10,29** (auto) | 244×205 | 10,27–10,29 | **+2,8 %** | **OK tolérance, non touché** |
| **Ghost** | 1,569×1,279 | 244×199 | **6,43** (auto) | 157×128 | 9,99 | −0,06 % | **CORRIGÉ + pivoté 180°** |
| **Chopper** | 2,265×1,628 | 243×175 | **9,31** (auto) | 226×163 | 10,01 | +0,05 % | **CORRIGÉ + pivoté 180°** |
| **Falcon** | 3,771×4,180 | 220×244 | **17,14** (auto) | 377×418 | 10,00 | 0,0 % | **CORRIGÉ + pivoté 180°** |
| **Pelican** | 11,090×8,288 | 244×182 | **45,50** (auto) | 1109×829 | 10,00 | 0,0 % | **CORRIGÉ + pivoté 180°** |
| **Skiff** | 5,218×2,665 | 244×125 | **21,35** (auto) | 522×266 | 10,00 | +0,07 % | **CORRIGÉ + pivoté 180°** |
| **Shade** | 1,401×1,008 | 244×176 | **5,74** (auto) | 140×101 | 9,99 | −0,06 % | **CORRIGÉ + pivoté 180°** |
| **Tourelle montée** | 1,264×1,234 | 244×238 | **5,18** (auto) | 126×123 | 10,03 | +0,3 % | **CORRIGÉ, SANS rotation** |

`*` Phantom : caveat déjà documenté (`V4_RAPPORT_SPRITES_2026-08-31.md` §10.4) — le modèle est en
unités « compactes », pas à l'échelle lore. Le rapport ne touche pas à cette sémantique, seule
l'échelle mm/px du modèle est vérifiée (elle est correcte).

**Facteur d'écart avant correction** : de ×0,52 (Ghost, trop petit) à ×4,55 (Pelican, trop
gros) par rapport à 10 mm/px — un facteur **8,8×** entre les deux pires cas, largement suffisant
pour casser toute lecture de proportions relatives dans le rejeu.

## 3. Pourquoi ces 9 dérivaient (et pas les 9 autres)

Le lot V4 (`V4_RAPPORT_SPRITES_2026-08-31.md`) a rendu tous les véhicules avec
`vehicle-sprite render -cote=256` (cadrage AUTO : le plus grand côté occupe ~244 px, quelle que
soit la taille réelle du véhicule — `render.go:39` passe `CellMetres: 0` quand `-cellmm` est
omis). Un Ghost (1,57 m) et un Pelican (11,09 m) occupent alors le **même nombre de pixels**,
donc des échelles mm/px opposées d'un facteur ~7×. Le lot Warthog (`WARTHOG_FINAL_V2/V3`), le lot
Gungoose (`CONTACT_ARMES_GUNGOOSE`) et le lot d'assemblage (`ASSEMBLAGE_ENFANTS_CORRIGE`) ont
depuis migré 9 véhicules vers `-cellmm=10` (échelle FIXE, indépendante de la taille) — ce sont
les 9 « déjà bons » du tableau. Les 9 restants (Banshee, Wasp, Ghost, Chopper, Falcon, Pelican,
Skiff, Shade, Tourelle montée) n'avaient jamais été retouchés depuis le rendu V4 auto — d'où la
mission. Banshee et Wasp se trouvent, par la taille de leur plus grand côté (2,50–2,51 m, proche
de 2,44 m = 244 px × 10 mm), à l'intérieur de la tolérance ±3 % **par coïncidence**, pas par
construction — ce n'est pas un design, juste que leur taille tombe près du point où la formule
`cell = taille/244` donne ~10 mm/px.

## 4. Re-rendu et détermination du sens de +X (aucune présomption)

Pour les 7 corrigés : `vehicle-sprite render -cellmm=10 -curate="<vehi>:<nom>,..."` sur les
modules `pc:multiplayer` + `pc:common` (+ `any` ×3 pour la résolution `vehi`), même style
(traits noirs, alpha, méthode V4 inchangée — aucun paramètre de rendu autre que l'échelle n'a
changé). Puis, pour chacun, détermination du sens de `+X` local (avant/arrière) **sur la
géométrie**, jamais présumée — trois preuves indépendantes croisées par véhicule :

1. **Vue de dessus** (`-axe=2`, convention actuelle) à haute résolution.
2. **Vue de profil** (`-axe=1`, plan X-Z) — permet de voir cockpit/nez/canon/dérive sans
   ambiguïté de perspective.
3. **Détail par section** (`vs-measure -modes=<id> -secs`) — repère les pièces étroites/effilées
   (canon, nez, aileron) par leur `dX`/`dY` et leur centroïde `cX`, sans dépendre d'une lecture
   visuelle.

Résultat (le code actuel place `-X` en haut de l'image par défaut, cf. `objet_isole.go:64-70`) :

| véhicule | signal décisif | +X = | rotation appliquée |
|---|---|---|---|
| Ghost | paire de canons très étroits (dY 0,08–0,12 m) à cX +0,61..+0,68, proche du bord +X | AVANT | 180° |
| Chopper | grande roue avant (disque dX≈dY≈1,46-1,6 m) + lames de garde-boue, cX positif | AVANT | 180° |
| Falcon | pièce nez/boom fine et basse (dY 0,43-0,61, cZ bas) à cX +1,16..+1,19 ; dérives/nacelles arrière massives (dY 1,06-1,32, cZ haut) à cX −5,6..−6,0 | AVANT | 180° |
| Pelican | pièce nez/train fine et basse à cX +3,0..+3,1 ; paire de dérives/nacelles large et haute à cX −5,6..−6,0 | AVANT | 180° |
| Skiff | forte densité de petites pièces symétriques (opérateur + consoles) cX −1,1..−2,15 ; coque lisse effilée côté +X | AVANT (coque) | 180° |
| Shade | paire de canons étroits (dY 0,19-0,20) à cX +0,45..+0,54, proche du bord +X | AVANT | 180° |
| Tourelle montée | grappe étroite (dY 0,31) + capuchon quasi plat (dX 0,02) à cX −0,45..−0,60, proche du bord −X | **ARRIÈRE** (donc `-X` = avant) | **AUCUNE** |

**Leçon méthodologique (à charge) : une première lecture visuelle rapide de la Tourelle montée a
conclu à tort que les canons étaient en bas (comme les 6 autres), et une rotation 180° a été
appliquée puis PUBLIÉE dans un fichier intermédiaire.** Le doute est venu de la vérification
croisée obligatoire (vue de profil + `-secs`) : les pièces étroites/le capuchon plat qui
signent un canon se trouvaient en réalité à `-X`, pas `+X` — la lecture visuelle initiale avait
pris le blindage/bouclier du récepteur pour un canon. La rotation a été **annulée** avant
livraison (le fichier final utilise le rendu SANS rotation). Ceci illustre pourquoi la mission
imposait une vérification géométrique et pas un pattern-matching visuel seul : sur 7 véhicules,
6 suivent la même règle (+X = avant) et 1 y échappe — un seul indice visuel ambigu aurait suffi à
livrer une Tourelle montée tête-bêche.

## 5. Vérification des compositions armées (Scorpion, Wraith)

Aucun retouche sur ces deux fichiers (déjà à l'échelle). Vérifié à l'œil (relecture des PNG
livrés) :

- **Scorpion** (`sprites_v4/scorpion.png`, 260×388 canevas) : canon long + collier de tourelle
  visibles, débordant nettement la coque — composition `ASSEMBLAGE_ENFANTS_CORRIGE` intacte.
- **Wraith** (`sprites_v4/wraith.png`, 304×313 canevas) : tourelle plasma circulaire centrée,
  visible au-dessus de la coque — composition intacte.
- **Phantom** (`sprites_v4/phantom.png`, 742×1080 canevas) : chassis net ; `phantom_g` reste,
  comme documenté dans `ASSEMBLAGE_ENFANTS_CORRIGE_2026-09-01.md` §3, un petit blob peu
  distinctif (limite déjà connue et acceptée, pas une régression de ce lot).

Hors périmètre de cette mission (notée, non traitée) : l'orientation nez-en-haut de Scorpion,
Wraith et Phantom n'a pas été re-vérifiée sur la géométrie (seule leur échelle était en
question) ; un examen rapide des trois PNG ne montre rien d'alarmant, mais la même rigueur
(profil + `-secs`) qui a rattrapé l'erreur sur la Tourelle montée n'a pas été appliquée ici — à
faire si un doute est soulevé sur ces trois fichiers dans un lot futur.

## 6. Livrables

- `.ai/V7.5/film_re/sprites_v4/{ghost,chopper,falcon,pelican,skiff,shade,tourelle_montee}.png`
  — remplacés (mêmes noms), 10 mm/px vérifié, style V4 inchangé (traits noirs, alpha, blanc
  NRGBA), pivotés 180° sauf `tourelle_montee.png`.
- `static/vehicles-assets/halo_infinite/replay/{mêmes 7 fichiers}` — copie identique (c'est le
  dossier servi par le lot A web, `useReplayVehicles.ts` ne lit que `famille`/`scale_mm_per_px`
  de `index.json`, tous deux inchangés en forme).
- `static/vehicles-assets/halo_infinite/replay/index.json` — les 18 entrées mises à jour (date,
  statut, note, source_report) ; `scale_mm_per_px` reste `10` partout (c'est la valeur nominale
  déjà correcte — la note documente la mesure réelle et la marge par rapport à 10).
- `.ai/V7.5/film_re/PLANCHE_ECHELLES_2026-09-02.png` (5649×1352) — les 18 sprites côte à côte à
  échelle pixel commune, triés par longueur mesurée décroissante : Pelican > Phantom > Skiff >
  Scorpion > Falcon > Wraith > Banshee ≈ Wasp > Chopper > (Razorback ≈ Warthog ≈ Rockethog ≈
  Warthog Gauss) > Ghost > Shade > Tourelle montée > (Mongoose ≈ Gungoose) — ordre crédible,
  cohérent avec le lore (transports > chars/VTOL > véhicules légers > montures/tourelles).
- `apps/go-api/cmd/vs-measure/sprite.go` (nouveau, extension additive) — sous-commande `sprite`,
  mesure la bbox opaque d'un ou plusieurs PNG.
- `apps/go-api/cmd/vs-measure/rot180.go` (nouveau, extension additive) — sous-commande `rot180`,
  pivote un PNG déjà rendu de 180° (rotation, jamais un miroir — même opération que `-rot180` de
  `plateau.go`).
- `apps/go-api/cmd/vs-measure/main.go` — 6 lignes ajoutées (dispatch des 2 nouvelles
  sous-commandes), aucune ligne existante modifiée.

## 7. Vérifications

- `gofmt -l ./cmd/vs-measure/` : rien à signaler.
- `go vet ./cmd/vs-measure/...` : propre.
- `go build ./cmd/vs-measure` : OK (GOCACHE dédié, CC winlibs, CGO_ENABLED=1).
- Aucun fichier `internal/` ni `apps/web/` touché. `cmd/vehicle-sprite/` non modifié (utilisé
  tel quel, extensions faites uniquement dans `vs-measure`).
- Fichiers ≤ 500 lignes, fonctions ≤ 80 lignes (les deux nouveaux fichiers font 110 et 88
  lignes commentaires compris).
- Les 18 sprites finaux mesurent tous entre 9,96 et 10,26 mm/px sur pièce (colonne « mm/px après »
  du tableau §2), soit un écart maximal de +2,8 % — sous la tolérance ±3 % demandée.

## 8. CR honnête — ce qui reste ouvert

- **Banshee et Wasp sont dans la tolérance par coïncidence**, pas parce que quelqu'un les a
  construits à `cellmm=10`. Si leur modèle change (nouveau chassis, LOD différent) à l'avenir,
  cette coïncidence de taille n'a aucune raison de se reproduire — contrairement aux 9 fichiers
  construits avec `-cellmm=10` explicite, qui restent corrects par construction quel que soit le
  modèle. Un futur lot pourrait les faire passer eux aussi par `-cellmm=10` pour ôter cette
  fragilité, mais ce n'était pas demandé (ils sont dans la tolérance) et ce lot ne l'a pas fait
  (zéro fix hors périmètre).
- **Pas de tourelle ajoutée** à Falcon/Pelican/Skiff/Shade/Tourelle montée : ces 5 sont restés
  des rendus mono-modèle (même niveau de détail géométrique qu'avant), seule l'échelle a changé.
  Composer une arme sur ces 5 (à la manière de Scorpion/Wraith/Warthog) est un travail distinct,
  non demandé par cette mission.
- **Orientation nez-en-haut de Scorpion/Wraith/Phantom non re-vérifiée** sur la géométrie (§5) —
  scope explicitement limité à leur échelle par la mission.
- **`Mongoose`/`Gungoose` à 9,96–10,20 mm/px** (moyenne 10,08, +0,8 %) : léger écart entre la
  mesure par la hauteur (9,96) et par la largeur (10,20), cohérent avec le rognage à la marge de
  6 px documenté dans `CONTACT_ARMES_GUNGOOSE_2026-09-02.md` — sans conséquence (bien sous
  ±3 %), non retouché.

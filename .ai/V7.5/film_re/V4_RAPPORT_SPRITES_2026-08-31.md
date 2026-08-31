# Lot V4 — sprites vue de dessus des vehicules (RAPPORT)

> Ecrit le 2026-08-31, worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Execution du lot V4 du chantier « vehicules et tourelles », sur la voie tranchee au scout S2
> (`SCOUT_SPRITES_VEHICULES_2026-08-31.md`) : rendu maison depuis les modeles 3D du jeu.
> Aucun commit, aucun `git add`. Build isole (GOCACHE dedie), CGO winlibs, tout en avant-plan.

## TL;DR

- **JALON 1 FRANCHI** et regarde a l'oeil : le Warthog rend une vraie vue de dessus,
  reconnaissable (chassis 4 roues, garde-boue avant en haut), teintable. Puis **13 vehicules**
  rendus, tous reconnus visuellement (planche de revue).
- **V2 (retour « presque parfait »)** : (1) **traits noirs de volume** ajoutes aux 14 sprites
  (roues, cockpit, rotors, pales, panneaux) via detection d'aretes profondeur+normale — LIVRE
  et regarde a l'oeil ; (2) **tourelles** : Shade (deja la) + 1 tourelle montee ; les Gauss /
  roquette / mitrailleuse « tourelle » n'existent PAS comme modeles propres (constat §10.2) ;
  (3) **variantes Warthog/Mongoose** : NON composees — l'arme est une piece jointe runtime hors
  de portee du balayage (constat honnete §10.3).
- La chaine `vehi -> hlmt -> mode -> triangles -> rasterizer local -> PNG` est ecrite sur le
  patron Forge. Seul maillon neuf : le resolveur `RefModeleVehicule` (recursif, filtre anti-parasite).
- Sprites livres : `.ai/V7.5/film_re/sprites_v4/*.png` (**14 fichiers**, ~256 px, `image.NRGBA`
  remplissage blanc + traits noirs, teintable **en MULTIPLY** cote web — cf. note §10.1).
- Code de production : `apps/go-api/cmd/vehicle-sprite/` + `internal/himap/{vehicules,objet_isole}.go`.

## 1. La chaine exacte

```
vehi (tag de definition, variante ANY)
  -> hlmt (model)                      RefModeleVehicule / modeleParHlmtRecursif  (vehicules.go)
       -> [recursion dans hlmt enfants si le hlmt est composite]
  -> mode (render_model, variante PC)  ExtractWithResources  (moduleindex.go)
  -> triangles/sommets                 NewRenderModelAsset   (geometry.go, DEJA en prod)
  -> rasterizer top-down repere LOCAL  RenduObjetIsole       (objet_isole.go, NOUVEAU)
  -> PNG silhouette blanche + alpha    SpriteObjetPNG        (objet_isole.go, NOUVEAU)
```

Points-cle du decodage :

1. **Le `vehi` ne se decode PAS structurellement.** Comme le `food` Forge, il reference son
   modele `hlmt` par un GlobalID inline ; `RefsInline` (cuisson_forge.go) le balaye. Aucun
   offset propre au tag `vehi` n'a ete leve (ni Reclaimer, ni Ghidra) — inutile.
2. **hlmt COMPOSITES.** Certains vehicules (Ghost, Banshee, Wraith, Chopper...) ont un hlmt
   PARENT qui ne porte aucun `mode` direct mais reference des hlmt ENFANTS porteurs de la
   geometrie (mesure Ghost : hlmt `0x3b3038e6` = 12 hlmt enfants, 0 mode direct). Le resolveur
   est donc RECURSIF (profondeur 6, ensemble `vus` anti-cycle), et EPUISE le premier hlmt —
   enfants compris — avant tout repli.
3. **Filtre anti-parasite `minTagIDVehicule = 0x00010000`.** Le balayage d'octets prend pour
   une reference toute valeur 4 octets qui resout dans l'index. Deux tags PARTAGES a ID
   minuscule derailaient 15 vehicules vers un meme modele parasite : `hlmt 0x0000001f`
   (octets « 1f 00 00 00 » = l'entier 31, omnipresent dans n'importe quel tag) capte comme
   « premier hlmt », qui porte un `mode 0x00003a73` de repli (un modele plat de 452 sections).
   Tous les VRAIS modeles de vehicule ont un GlobalID de hash large (>= 0x06000000) ; ecarter
   les ID sous 0x10000 supprime les deux parasites sans toucher un modele reel. Filtre PROPRE
   au vehicule (le Forge n'a pas ce piege).
4. **Le rasterizer est reutilise tel quel** (`Rendu`, rendu.go, DEJA en prod pour les fonds de
   carte) : z-buffer (surface la plus haute par pixel) + normale + Lambert oblique. Le mode
   « objet isole » l'enveloppe : emprise = bornes du maillage, repere local, PAS de tranche de
   jeu ni de bornage carte, instance identite.

### Topologie des modules (decouverte de terrain, non triviale)

- **Les tags `vehi` ne vivent QUE dans la variante `any`** (metadonnees), jamais dans `pc` :
  67 tags `vehi` repartis sur `any/globals` (13), `any/common`, `any/multiplayer`. La variante
  `pc` en a 0.
- **Les `mode` (geometrie de rendu) vivent dans `pc`, mais REPARTIS** :
  - `pc/globals-rtx-new.module` : Warthog, Mongoose, Scorpion, Wasp.
  - `pc/multiplayer` + `pc/common` : Ghost, Banshee, Wraith, Chopper, Phantom, Pelican, Skiff,
    Shade, Falcon.
- Consequence pratique : l'index complet (`any` x3 + `pc/globals` + `pc/multiplayer` +
  `pc/common`) pese **~16,7 Go** en RAM (`os.ReadFile` charge tout, companions `_hd1` compris),
  au-dela des 12,6 Go libres du poste. **Rendu en DEUX PASSES** complementaires (le filtre
  anti-parasite garantit zero faux positif dans chaque passe : un vehicule dont le mode n'est
  pas dans la passe courante ressort « SANS MODELE », jamais un mauvais modele) :
  - Passe 1 : `pc:globals` + `any` x3.
  - Passe 2 : `pc:multiplayer` + `pc:common` + `any` x3.
- L'index resout un GlobalID sur plusieurs modules (`ModuleIndex`, premier arrive premier
  servi). L'ordre `pc` d'abord n'a PAS suffi seul : le vrai hlmt du Ghost est identique dans
  les deux variantes (36 120 o) et ne porte de toute facon pas de `mode` direct — d'ou la
  recursion (point 2) plutot qu'un simple reordonnancement.

## 2. L'axe « haut » retenu et sa preuve

**Repere moteur (Halo Infinite) : X = AVANT, Y = GAUCHE, Z = HAUT.** La vue de dessus regarde
vers `-Z`.

Preuve (le scout ne pouvait pas la faire, il ne compilait pas) : projete selon **Z haut**, le
Warthog rend une empreinte coherente de vehicule — chassis allonge ~2:1, quatre roues aux
quatre coins, garde-boue avant en barres verticales. Une projection selon X ou Y donnerait une
vue de PROFIL ou de FACE (plus haute que large, silhouette differente) — ce n'est pas ce qu'on
obtient. L'axe Z est donc confirme sur piece, pas par postulat.

**Orientation canonique du sprite : AVANT (+X) vers le HAUT de l'image, GAUCHE (+Y) a gauche**
(« nez en haut »), l'orientation attendue d'une icone que le rejeu fait tourner selon le cap.
Remap applique (objet_isole.go) : `(x,y,z) -> (-y, x, z)`, puis `SpriteObjetPNG` retourne l'axe
Y de l'image. Verifie sur le Warthog (garde-boue avant en haut) et coherent sur les 13.
Caveat mineur : la face avant visuelle depend du modele ; la convention retenue est « +X = avant
= haut », uniforme, le rejeu gere la rotation par cap.

## 3. Format de sortie

- **PNG `image.NRGBA`** (NON premultiplie), remplissage **blanc pur** (255,255,255), **traits
  noirs** (0,0,0), alpha porteur de la forme. **Piege corrige** : `color.RGBA` de Go est
  alpha-PREMULTIPLIE — un blanc a alpha variable y ressort gris fonce apres encodage
  (R = 255 - A, mesure). `NRGBA` garde le RGB tel quel.
- L'alpha du remplissage module l'ombrage de Lambert entre `alphaBase` (0,80, faces rasantes)
  et 1 (faces vues de dessus) : silhouette pleine, relief present.
- **Traits noirs de VOLUME** (V2, cf. §10) : contour exterieur + aretes internes (roues,
  cockpit, pales, panneaux). C'est ce qui montre le volume demande.
- Cote maitre ~256 px (le rejeu affiche downscale a 24-64 px). Emprise propre a chaque
  vehicule (portrait, hauteur ~256).

## 4. Table vehicule -> tag `vehi` -> `mode` -> sprite

| Vehicule | vehi (GlobalID) | mode (GlobalID) | sections | module du mode | sprite |
|---|---|---|---|---|---|
| Warthog  | 0x00002705 | 0x561f2ca7 | 90  | pc/globals            | warthog.png  |
| Mongoose | 0x000025aa | 0x9e581380 | 56  | pc/globals            | mongoose.png |
| Scorpion | 0x0000d3db | 0x39918211 | 84  | pc/globals            | scorpion.png |
| Wasp     | 0xb65b3b4a | 0x06d6ce82 | 74  | pc/globals            | wasp.png     |
| Ghost    | 0x0000d3dc | 0x34b6148f | 25  | pc/multiplayer+common | ghost.png    |
| Banshee  | 0x000026ed | 0xb07d7500 | 42  | pc/multiplayer+common | banshee.png  |
| Wraith   | 0x00002706 | 0x3a98ee2d | 35  | pc/multiplayer+common | wraith.png   |
| Chopper  | 0x002ba902 | 0xf22c06af | 47  | pc/multiplayer+common | chopper.png  |
| Phantom  | 0x000026f2 | 0x8acf1ab6 | 37  | pc/multiplayer+common | phantom.png  |
| Pelican  | 0x000026f0 | 0x0bb06a30 | 225 | pc/multiplayer+common | pelican.png  |
| Skiff    | 0x86799cb6 | 0xa3aaa279 | 43  | pc/multiplayer+common | skiff.png    |
| Shade    | 0x000df0c4 | 0xfa88c317 | 22  | pc/multiplayer+common | shade.png    |
| Falcon   | 0x0000254b | 0xa0ca8a6f | 76  | pc/multiplayer+common | falcon.png   |

Identification des vehicules : par leurs **noms de maillage internes** (chassis/roue/panneau),
lus dans les octets du tag `vehi` (ex. `warthog_p_rf_ma`, `ghost_b_f`, `scorpion_frt_bf`). Les
noms de fichier de tags sont strippes en release, mais ces noms de maillage subsistent et
suffisent a classer la famille de chassis (`classify.go`).

## 5. Jalon 1 (preuve)

- **PNG Warthog** : `.ai/V7.5/film_re/sprites_v4/warthog.png` (256 px). Chaine complete
  parcourue : vehi `0x00002705` -> hlmt -> mode `0x561f2ca7` (90 sections) -> triangles ->
  rasterizer local (axe haut Z) -> PNG silhouette+alpha. **Regarde a l'oeil** (compose sur fond
  gris ET teinte rouge) : silhouette de vehicule 4 roues vue de dessus, reconnaissable, relief
  present, teinte fonctionnelle. C'est exactement la verification que le scout n'avait pas pu
  faire (il ne compilait pas).

## 6. Ce qui reste

- **Desambiguation des VARIANTES d'un meme chassis.** Rockethog / Razorback partagent le
  render_model du Warthog (`0x561f2ca7`) ; Gungoose partage celui du Mongoose (`0x9e581380`).
  En vue de dessus, chassis identique => sprite identique. Distinguer la variante exige de lire
  l'ARMEMENT/attachement au niveau du `vehi` (refs `weap`), pas la geometrie du chassis. Pour
  le rejeu, un sprite par chassis suffit ; si une distinction visuelle est voulue (tourelle
  rocket vs chaingun), il faudra composer l'attachement — hors de ce lot.
- **22/67 vehi non resolus** (par passe) : doublons (campagne, variantes EMP), fragments
  « turret »/« gun » a 1-6 sections (canons detaches, pas un chassis), et quelques `vehi`
  nommes `inconnu` (modeles a 1-2 sections = pieces isolees). Aucun vehicule pilotable MP ne
  manque : les 8 du perimetre (Warthog, Mongoose, Ghost, Banshee, Wasp, Scorpion, Wraith,
  Chopper) sont tous rendus, plus 5 bonus (transports Phantom/Pelican, Skiff, Shade, Falcon).
- **Bruit interne mineur.** Le `mode` d'un vehicule contient AUSSI ses maillages d'etat
  detruit (`_b_d_*`), rendus par-dessus le chassis intact. Ils partagent l'empreinte, donc la
  silhouette reste juste ; ils ajoutent des lignes internes. Invisibles au downscale 24-64 px,
  mais un rendu « intact seul » demanderait de parser les regions/permutations du render_model
  (non fait, non bloquant).
- **Deux passes obligees** tant qu'on charge les modules par `os.ReadFile` complet. Un lecteur
  `.module` en mmap (au lieu de tout charger) permettrait une passe unique — chantier `himodule`
  distinct.
- **Tourelle Scorpion detachable** (`scorpion_g`, modes `0x60dd0e4e` / `0x8bf43b79`, 16-23
  sections) et tourelles fixes (`turret_*`) : resolues mais non curées dans le set final (hors
  coeur pilotable). Rendables a la demande via le meme outil.

## 7. Decouvertes hors perimetre

- **67 tags `vehi`** dans le jeu (bien plus que les 12 pilotables) : variantes de campagne,
  d'etat, tourelles, transports (Pelican/Phantom), et vehicules Banished (Skiff, Chopper,
  Wraith, Ghost). Les noms de maillage internes correspondent aux index nommes de l'atlas kill
  feed (26-38) documente par le scout icones.
- **Structure hlmt COMPOSITE** (parent -> enfants) : pattern moteur non documente jusqu'ici
  dans `.ai/`, utile a tout futur travail qui suit une chaine `objet -> hlmt -> mode` (Forge y
  echappe car ses hlmt referencent `mode` directement).
- **Le faux-positif `0x00003a73`** (mode plat 452 sections) et `hlmt 0x0000001f` sont des
  parasites recurrents du balayage d'octets a ID bas — a garder en tete pour toute resolution
  de tag par `RefsInline` (le Forge pourrait en souffrir sur un tag futur ; il ne le fait pas
  aujourd'hui car ses IDs sont larges).

## 10. V2 — traits noirs, tourelles, variantes (retour utilisateur « presque parfait »)

### 10.1 Traits noirs de volume (LIVRE, les 14 sprites)

Ajoute une passe de detection d'aretes sur le z-buffer du `Rendu` (`aretesObjet`,
objet_isole.go). Un pixel de matiere devient un trait NOIR (RGB 0,0,0, alpha inchange) si l'un
de ses 4 voisins declenche :

- **contour exterieur** : voisin sans matiere ;
- **rupture de PROFONDEUR** : |dz| entre voisins > `SeuilProfCell x cell` — un saut d'occlusion
  (roue devant carrosserie, bord de cockpit, pale). Seuil en MULTIPLE de la taille d'un
  pixel-monde (independant de l'echelle du vehicule) ;
- **rupture de NORMALE** : angle entre faces retenues > `SeuilAngleDeg` (produit scalaire des
  normales, ramenees dans l'hemisphere superieur, < cos(angle)) — arete de capot, panneau, pale.

Puis un **despeckle** retire les pixels d'arete ISOLES (bruit de z-buffer des maillages d'etat
detruit qui se superposent ; un vrai trait a toujours des voisins).

**Seuils retenus** (regardes a l'oeil sur Warthog = roues+cockpit, Wasp/Falcon = rotors,
Banshee = ailes) : `SeuilProfCell = 7` (7 pixels verticaux), `SeuilAngleDeg = 30`. Detail
riche voulu par l'utilisateur ; le residu de bruit sur les fuselages courbes (superposition
etat-detruit) est mineur et disparait au downscale. Reglables via `OptionsSprite`
(`SansAretes` pour couper).

**NOTE TEINTE — a repercuter cote web (follow-up hors ce lot).** Avec des traits noirs, la
teinte d'equipe doit passer en **MULTIPLY** (couleur x blanc = couleur d'equipe ; couleur x
noir = noir, les traits survivent). Le calque `tintedIconCanvas` fait aujourd'hui un
`source-in` (remplace tout le RGB par la couleur, effacant les traits) — a basculer en
`globalCompositeOperation = 'multiply'` sur un fond blanc du sprite, ou equivalent. C'est le
SEUL changement web requis par cette v2.

### 10.2 Tourelles (PARTIEL + constat honnete)

- **Shade** (Covenant/Bannis) : deja livre (§4), tourelle propre et reconnaissable.
- **`tourelle_montee.png`** (vehi `0x038df01a` -> mode `0x1ae526e1`, 28 sections) : la
  tourelle-vehi la plus complete, lit comme un emplacement monte (corps + canon). Ajoutee.
- **CE QUI N'EXISTE PAS proprement dans les fichiers** : aucun tag `vehi`/`bloc`/`mach` nomme
  « gauss », « rocket », « machinegun », « chaingun », « gatling », « emplacement »... (scan
  exhaustif des trois groupes, 0 resultat). Les autres tourelles-`vehi` sont des COMPOSANTS a
  3-6 sections (canons, anneaux de montage), pas des emplacements reconnaissables. La
  mitrailleuse UNSC detachable (AIE-486H), le Gauss et la roquette « tourelle » du besoin sont
  soit des VARIANTES d'armement du Warthog (cf. 10.3), soit des `weap` d'infanterie (barils
  fins ~5:1 en vue de dessus, inexploitables comme icone). Le turret Bannis distinct du Shade
  n'a pas ete trouve. Verdict : au-dela de Shade + `tourelle_montee`, il n'y a pas de modele de
  tourelle propre a extraire — pas un echec d'outil, une absence dans les tags.

### 10.3 Variantes Warthog / Mongoose (NON compose — constat honnete)

Mesure decisive (`modes` walker sur l'arbre hlmt) : l'arbre geometrique du Warthog `0x00002705`
ne contient **QU'UN mode, le chassis `0x561f2ca7`** — aucun `weap` ni mode d'arme atteignable.
Idem Mongoose. **L'armement des variantes (roquettes du Rockethog, canon du Gungoose, benne du
Razorback) est une PIECE JOINTE RUNTIME** : le vehi la reference par un petit ID / string-id +
un marqueur de montage, hors de portee du balayage d'octets (`RefsInline`) qui fonde toute la
chaine. Les `weap` « missile turret » / « arifle turret » resolvent bien un modele (`0xcf38e84b`
148 sections, `0xe7f1a0dc` 101 sections...), mais ce sont les modeles d'ARME TENUE (barils fins,
aspect ~5:1 vu de dessus), PAS le pod de tourelle du vehicule — les composer sur le chassis ne
donnerait pas une silhouette de Rockethog reconnaissable.

**Consequence** : top-down, Warthog / Rockethog / Razorback partagent une silhouette de chassis
IDENTIQUE (`0x561f2ca7`), Mongoose / Gungoose de meme (`0x9e581380`). Un sprite par chassis
couvre donc toutes les variantes. Distinguer visuellement les variantes exigerait de parser la
structure d'attachement du `vehi` (ref d'arme du vehicule + transforme du marqueur de montage) —
de la vraie retro-ingenierie de format, au-dela d'un best-effort par balayage. **Non fait,
assume.** Piste si repris : lever l'offset du bloc « seats/attachments » du tag `vehi` par
Ghidra (lecture seule) pour extraire ref-arme + marqueur, puis composer les deux modes dans un
meme `Rendu` a la transforme du marqueur.

## 8. Fichiers

Production (worktree, NON commite) :
- `apps/go-api/internal/himap/vehicules.go` — `GroupeVehi`, `RefModeleVehicule`
  (resolveur recursif + filtre `minTagIDVehicule`), `EntreesDuGroupe`. (NON modifie en V2 :
  coordination avec l'agent sons qui pouvait editer ce fichier.)
- `apps/go-api/internal/himap/objet_isole.go` — `AxeHaut`, `RenduObjetIsole`, `SpriteObjetPNG`
  (remplissage blanc + traits noirs), `aretesObjet` + `despeckle` (V2, detection d'aretes).
- `apps/go-api/cmd/vehicle-sprite/` — `main.go`, `scan.go` (enumere + identifie), `render.go`
  (rend / curate), `classify.go` (famille de chassis par noms de maillage).

Assets : `.ai/V7.5/film_re/sprites_v4/*.png` — **14 sprites teintables** (13 vehicules + 1
tourelle montee), tous avec traits noirs de volume.

Verification : `gofmt` propre, `go build` OK, `go vet ./internal/himap/... ./cmd/vehicle-sprite/...`
propre. Seuils respectes (fichiers <= 500 L, fonctions <= 80 L).

Piste de durcissement non faite (hors scope, notee) : un test unitaire NON-gamefiles sur
`remapMesh` / `bornesPlan` / `classeVehicule` / `SpriteObjetPNG` (fonctions pures, rapides),
plus un gamefiles-test qui verifie que la chaine resout le Warthog vers un mode a >50 sections.

## 9. Comment reproduire

```
# depuis apps/go-api, avec GOCACHE dedie + CC winlibs + CGO_ENABLED=1
go build -o v4tool.exe ./cmd/vehicle-sprite

# passe 1 (modes dans pc/globals)
v4tool.exe render -variant=any -cote=256 -out=<dir> \
  -modules="pc:globals-rtx-new.module,globals-rtx-new.module,common-rtx-new.module,multiplayer-rtx-new.module" \
  -curate="0x00002705:warthog,0x000025aa:mongoose,0x0000d3db:scorpion,0xb65b3b4a:wasp"

# passe 2 (modes dans pc/multiplayer+common)
v4tool.exe render -variant=any -cote=256 -out=<dir> \
  -modules="globals-rtx-new.module,common-rtx-new.module,multiplayer-rtx-new.module,pc:multiplayer-rtx-new.module,pc:common-rtx-new.module" \
  -curate="0x0000d3dc:ghost,0x000026ed:banshee,0x00002706:wraith,0x002ba902:chopper,0x000026f2:phantom,0x000026f0:pelican,0x86799cb6:skiff,0x000df0c4:shade,0x0000254b:falcon"

# enumerer/identifier tous les vehi : v4tool.exe scan -variant=any -modules="..."
```

# Lot V4 — sprites vue de dessus des vehicules (RAPPORT)

> Ecrit le 2026-08-31, worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Execution du lot V4 du chantier « vehicules et tourelles », sur la voie tranchee au scout S2
> (`SCOUT_SPRITES_VEHICULES_2026-08-31.md`) : rendu maison depuis les modeles 3D du jeu.
> Aucun commit, aucun `git add`. Build isole (GOCACHE dedie), CGO winlibs, tout en avant-plan.

## TL;DR

- **JALON 1 FRANCHI** et regarde a l'oeil : le Warthog rend une vraie vue de dessus,
  reconnaissable (chassis 4 roues, garde-boue avant en haut), teintable. Puis **13 vehicules**
  rendus, tous reconnus visuellement (planche de revue).
- **V2 (retour « presque parfait »)** : (1) **traits noirs de volume** ajoutes a tous les sprites
  (roues, cockpit, rotors, pales, panneaux) via detection d'aretes profondeur+normale — LIVRE
  et regarde a l'oeil ; (2) **tourelles** : Shade (deja la) + 1 tourelle montee ; les Gauss /
  roquette / mitrailleuse « tourelle » n'existent PAS comme modeles propres (constat §10.2).
- **V3 (variantes)** : composees en 2D best-effort (placeholders) — `warthog`, `rockethog`,
  `warthog_gauss`, `razorback` (chassis nu = cargo, correct), `gungoose`.
- **V4 (variantes REELLES) : ABOUTI.** Ecrit le **marcheur de permutations** du render_model
  (`himap.ModeRegions`, regions -> permutations -> plages de sections) : les variantes d'un
  vehicule sont des PERMUTATIONS d'une region du mode PARTAGE, pas des modeles separes. Rendu
  chaque variante = base + sa permutation, meme cadre, traits noirs. Les 4 variantes Warthog
  (chaingun/rockets/Gauss/Razorback) et Mongoose/Gungoose sont **ENFIN NETTEMENT DISTINCTES**,
  vraie geometrie aux vrais emplacements (§10.5). StringId `default`/`unarmed` resolus par
  murmur3 ; les 3 armes mappees par forme de tourelle (§10.5).
- La voie Ghidra pour parser le bloc seat du `vehi` etait une IMPASSE (weap sans modele, binaire
  strippe) — le vrai chemin etait les permutations du render_model (§10.4 = diagnostic, §10.5 = solution).
- La chaine `vehi -> hlmt -> mode -> triangles -> rasterizer local -> PNG` est ecrite sur le
  patron Forge. Seul maillon neuf : le resolveur `RefModeleVehicule` (recursif, filtre anti-parasite).
- Sprites livres : `.ai/V7.5/film_re/sprites_v4/*.png` (**18 fichiers** : 13 vehicules + 1 tourelle
  + 4 variantes composees, ~256 px, `image.NRGBA` remplissage blanc + traits noirs, teintable **en
  MULTIPLY** cote web — cf. note §10.1).
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

### 10.3 Variantes Warthog / Mongoose (COMPOSEES — best-effort, placement approxime)

**Ce qui est ETABLI (mesure, pas suppose).** Les 6 `vehi` Warthog resolvent TOUS le meme chassis
`0x561f2ca7`, et referencent TOUS le meme `weap 0x0000a4bc` (« fixed » = chaingun de base — le
chantier son le voit aussi « non resolu »). Les 3 `vehi` Mongoose : chassis `0x9e581380`, weap
`0x033e41df`. **Donc le `vehi` de base ne distingue PAS les variantes** : l'arme distinctive
(roquettes / Gauss / benne) est attachee via la chaine `vcdd -> sofd -> sofa -> uwfa -> weap`
(regle R-VEHICULE, `VEHICULES_ARCHETYPE_40.md`). Cette chaine est ANONYME : les 63 `vcdd` n'ont
aucun nom lisible (que des fragments de tag-refs), et les `weap` « missile turret » resolvent des
modeles d'ARME TENUE (barils fins ~5:1 vu de dessus), pas des pods de tourelle. **Le pod exact de
chaque variante n'est donc pas identifiable proprement dans les tags.**

**Ce qui a ete FAIT quand meme (composition 2D a echelle fixe).** Nouveau levier :
`render -cellmm=<mm/px>` fixe l'echelle (metres/pixel) au lieu de l'ajuster a `-cote`. Deux objets
rendus au MEME `cellmm` sont a la meme echelle -> composables. Contrainte RAM (chassis en
`pc/globals`, tourelles en `pc/multiplayer+common`, jamais chargeables ensemble) : on rend chacun
SEPAREMENT au meme `cellmm`, puis on COMPOSE en 2D (la tourelle posee sur le chassis, System.Drawing).
Le resultat garde traits noirs + alpha teintable.

Modeles de tourelle-`vehi` employes (les seuls modeles de tourelle propres du jeu), MAPPING PAR
FORME VISUELLE, NON confirme par les tags :

| variante | fichier | tourelle (vehi -> mode) | placement | confiance |
|---|---|---|---|---|
| Warthog (base) | `warthog.png` | `0x038df01a -> 0x1ae526e1` (bloc+canon) | centre-arriere | arme = best-guess ; « a une tourelle arriere » = OK |
| Rockethog | `rockethog.png` | `0x003f00c7 -> 0x1c8f09d8` (compact, pods) | centre-arriere | best-guess |
| Gauss | `warthog_gauss.png` | `0x000025a5 -> 0x56fd2500` (canon long fin) | arriere, deborde | best-guess |
| Razorback | `razorback.png` | AUCUNE (chassis nu) | — | **CORRECT** (cargo = sans tourelle) |
| Gungoose | `gungoose.png` | `0x3a8060e2 -> 0x1c645961` (canon) | avant | best-guess |

**Placement APPROXIME** (assume) : le marqueur de montage exact du `vehi` n'a pas ete extrait ;
la tourelle est posee a sa position PLAUSIBLE (Warthog = centre-arriere, Gungoose = avant). La
taille de la tourelle est legerement exageree pour la lisibilite d'icone.

**CR honnete.** NET : Razorback (chassis sans tourelle) est correct et clairement distinct ;
Warthog montre desormais une tourelle arriere (repond au retour utilisateur). APPROXIME : le
mapping tourelle->variante (chaingun/roquettes/Gauss/gungoose) est un best-guess par forme, non
confirme par les tags — top-down, Warthog/Rockethog/Gauss restent proches (meme chassis, petite
tourelle arriere), la difference est subtile parce que la GEOMETRIE SOURCE l'est. La seule voie
d'une distinction certaine : parser le bloc d'attachement du `vehi` (ref-arme de vehicule +
transforme du marqueur) par Ghidra, puis composer les modes 3D a la transforme reelle — vraie RE
de format, hors du temps de ce lot.

### 10.4 Variantes REELLES (passe Ghidra) — BLOCAGE PRECIS + prochain pas localise

Objectif : les VRAIES tourelles distinctes au vrai marqueur. Resultat : **non abouti ce tour**,
mais le blocage est mesure et nomme, pas suppose. Ce qui a ete etabli sur pieces :

1. **Les vrais weap de variantes viennent du chantier son** (`manifeste_v3.json`) :
   Rockethog `weap 0xc7d50912`, Gungoose `0x0042678e`, chaingun de base `0x0000a4bc`,
   Scorpion `0x00015cfa`, Wasp `0x11725dc4`, Falcon `0x00015cd3`.
2. **Ces weap n'ont AUCUN render_model.** Dump des refs par groupe (tout ID) : rien que du
   comportement de tir — `effe`, `jpt!`, `snd!`/`lsnd`, `proj`, `wpdp`, `grfr`. Zero
   `hlmt`/`mode`/`rtgo`/`bloc`/`vehi`. Le modele de la tourelle n'est PAS sur le weap.
3. **La chaine d'arme R-VEHICULE entiere est sans modele.** Les 16 `uwfa` referencent chacun un
   `weap` (dont Gungoose `0x0042678e` via `uwfa 0xe3d5dc10`) mais AUCUN ne porte de ref de
   modele. `vcdd` (63 tags) : aucun nom lisible, aucune ref de modele resolvable.
4. **Les turret-`vehi` (turret_g) ne sont pas les tourelles de vehicule** : elles referencent des
   weap SANS RAPPORT avec les weap de variantes (0x0bf807fe, 0x000026b6...) — ce sont des
   emplacements autonomes (AIE, etc.), pas le pod du Warthog.
5. **Ghidra (HaloInfinite.exe, 311k fn, base 140000000)** : accesseurs `ManagedGameVariant_*`
   presents, mais **aucun symbole de structure de tag** (pas de fonction `Seat`), et **aucune
   chaine de nom de champ** (« child object », « primary weapon », « seat »... = 0 resultat) —
   binaire release, noms de champs strippes. Parser le bloc seat/weapon du `vehi` par Ghidra =
   trace d'offsets AVEUGLE dans des `FUN_` = multi-session, pas ce tour.
6. **Piste REELLE localisee (le vrai prochain pas)** : les variantes de tourelle sont tres
   probablement des **PERMUTATIONS d'une region du render_model partage** (le rendu actuel les
   SUPERPOSE toutes, d'ou un « rear » charge mais indistinct). Mesure : le mode `0x561f2ca7` a un
   **bloc regions a l'offset racine 40** (480 o = 20 regions de 24 o = `name` StringId + TagBlock
   de permutations, comptes 1-4), un bloc Sections a l'offset 192 (90 sections = 90x60), un
   BoundingBox a 232. Rendu section par section (levier ajoute puis retire, cf. plus bas) :
   les **sections 51/52/53** sont des formes de TOURELLE distinctes (canon central + base evasee) ;
   64/65/66/67 sont des LOD du corps entier. **RESTE A FAIRE** (session dediee) : field-walker
   region -> permutations -> plage de sections (comme le walker `sbsp`), resoudre les StringId
   des noms de region/permutation (« turret »/« rocket »/« gauss »), puis rendre par permutation
   au meme cadre. Ca donnerait les vraies tourelles au vrai endroit, sans Ghidra.

**Diagnostic V4** : le modele de tourelle n'est ni sur le `weap`, ni sur la chaine
`uwfa/sofa/sofd/vcdd`, ni sur une turret-`vehi` liee — il est en PERMUTATION du render_model du
chassis (region a l'offset 40). C'est ce field-walker qui a ete ecrit en §10.5, et qui resout
les variantes pour de bon.

### 10.5 Variantes REELLES par le marcheur de permutations (ABOUTI)

Le field-walker `himap.ModeRegions` (regions.go) marche : `mode` -> bloc regions (champ racine
+40, 24 o/region) -> pour chaque region, TagBlock de permutations resolu via la struct-table
(meme mecanique que `lods()`) -> permutation (12 o : Name StringId +0, SectionIndex u16 +4,
SectionCount u16 +6). VERIFIE : les plages couvrent proprement les 90 sections du Warthog.

**Structure decouverte.** Une VARIANTE = un nom de permutation PARTAGE entre regions. Le Warthog
(mode `0x561f2ca7`, 20 regions) a **5 permutations** ; le Mongoose (`0x9e581380`, 19 regions) en
a **2**. Pour rendre une variante : par region, on prend sa permutation portant ce nom ; si
absente OU `SectionIndex < 0` (= herite), on retombe sur la permutation de BASE (`default`). Sans
cette regle d'heritage, `unarmed` perdait ses roues.

**Resolution des noms (StringId = murmur3, `mapvar.LabelHash`).**

| StringId | nom resolu | fichier livre | mapping |
|---|---|---|---|
| `0x42c9679f` | **default** | (Mongoose -> `mongoose.png`) | StringId confirme |
| `0x4e154ee8` | **unarmed** | `razorback.png` | StringId confirme (sans arme = cargo) |
| `0x06c86db1` | (non resolu) | `warthog.png` | forme : tourelle boxy = chaingun |
| `0x13d24f1f` | (non resolu) | `rockethog.png` | forme : arriere large a pods = roquettes |
| `0xad03512a` | (non resolu) | `warthog_gauss.png` | forme : canon central long = Gauss |
| `0x02c9ed0a` | (non resolu, Mongoose) | `gungoose.png` | forme : mount avant = canons |

**Resultat (regarde a l'oeil, planche de comparaison).** Les 4 variantes Warthog different
NETTEMENT a l'arriere (tourelle boxy / pods larges / canon central / cargo plat) et Mongoose vs
Gungoose au mount avant. C'est de la vraie geometrie du modele, a la vraie position (les sections
sont deja placees dans le repere du modele). Plus de placement approxime, plus de best-guess de
modele.

**CR honnete.** Ce qui est CERTAIN : la structure regions/permutations, les plages de sections,
et les noms `default`/`unarmed` (murmur3). Ce qui reste un CHOIX : l'affectation
chaingun/roquettes/Gauss aux trois StringId non resolus (`06c86db1`/`13d24f1f`/`ad03512a`) est
faite par la FORME de la tourelle arriere, pas par le nom — un dictionnaire de StringId plus
complet (ou l'ordre canonique des variantes du jeu) leverait ce dernier doute. Les silhouettes,
elles, sont les vraies.

## 8. Fichiers

Production (worktree, NON commite) :
- `apps/go-api/internal/himap/vehicules.go` — `GroupeVehi`, `RefModeleVehicule`
  (resolveur recursif + filtre `minTagIDVehicule`), `EntreesDuGroupe`. (NON modifie en V2 :
  coordination avec l'agent sons qui pouvait editer ce fichier.)
- `apps/go-api/internal/himap/objet_isole.go` — `AxeHaut`, `RenduObjetIsole`, `SpriteObjetPNG`
  (remplissage blanc + traits noirs), `aretesObjet` + `despeckle` (V2), `OptionsSprite`
  (`CellMetres` echelle fixe, `SectionsChoisies` filtre de sections, `CadreMin/Max` cadre force).
- `apps/go-api/internal/himap/regions.go` — **`ModeRegions`** (V4) : marcheur regions ->
  permutations -> plages de sections d'un render_model. C'est le levier des variantes reelles.
- `apps/go-api/cmd/vehicle-sprite/` — `main.go`, `scan.go`, `render.go` (`-cellmm`),
  `classify.go`, **`variantes.go`** (V4 : rend une image par permutation, noms via murmur3).

Assets : `.ai/V7.5/film_re/sprites_v4/*.png` — **18 sprites teintables** (12 vehicules + 1
tourelle montee + variantes Warthog `warthog`/`rockethog`/`warthog_gauss`/`razorback` et
`mongoose`/`gungoose`), tous avec traits noirs de volume ; variantes = vraies permutations du
render_model (§10.5).

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

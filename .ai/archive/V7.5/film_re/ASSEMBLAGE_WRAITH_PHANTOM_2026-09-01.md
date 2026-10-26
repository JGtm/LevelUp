# Assemblage enfant CORRIGE (ordre-peintre) — Scorpion, Warthog, Wraith, Phantom (RAPPORT)

> Ecrit le 2026-09-01, worktree `LevelUp-wt-vehicules`. Aucun commit, aucun `git add`.
> Build isole (GOCACHE dedie), CGO winlibs, tout en avant-plan. Code partage NON modifie
> (`internal/himap/*`, `cmd/vehicle-sprite/*`) : cet agent ne fait que LANCER l'outil et
> ecrire des PNG + planche + rapport.
> Suite du lot d'assemblage (`ASSEMBLAGE_ENFANTS_2026-09-01.md`) : sur retour utilisateur,
> l'assemblage passe de « z-buffer in-module » a « ordre-peintre sur canevas fixe » pour
> GARANTIR placement + echelle + superposition. Perimetre etendu au Scorpion et au Warthog.

## TL;DR

- **Methode corrigee (ordre-peintre / canevas fixe)** : chassis et tourelle rendus
  SEPAREMENT sur un MEME canevas fixe (`assemble -cadre=7 -cellmm=10`, origine vehicule au
  centre, PNG NON rognes, 1413x1413), puis `compose2d` en source-over avec la TOURELLE EN
  DERNIER (toujours au-dessus), et rognage du COMPOSITE final SEULEMENT. Le co-repérage des
  enfants fait tomber la tourelle a sa place, a la meme echelle monde->pixel que le chassis.
  Ne depend PAS du z-buffer in-module (qui pouvait laisser la coque occulter la tourelle).
- **4 sprites regeneres** en methode corrigee : `sprites_v4/{scorpion,warthog,wraith,phantom}.png`.
- **Modes resolus MOI-MEME** (via `scan`, resolveur corrige) — chassis :
  Wraith `0x3a98ee2d`, Phantom `0x8acf1ab6`, Scorpion `0x39918211`, Warthog `0x561f2ca7`.
- **Verdict a l'oeil** : Scorpion NET (canon centre sur la tourelle, au-dessus, barillet
  debordant), Wraith BON (tourelle plasma circulaire centrale, au-dessus, centree), Warthog
  BON (LAAG centree a l'arriere, au-dessus). **Phantom = limite honnete** : `phantom_g`
  (mode PARTAGE `0x56fd2500`) est un modele MINUSCULE (~0,5 m) authored a l'ORIGINE du
  vehicule -> compose il apparait comme un blob central negligeable face a l'enorme fuselage.
  Present et au-dessus, mais non distinctif en vue de dessus.
- **Bonus Ghost / Banshee / Chopper** : armes INTEGREES au chassis, AUCUN vehi-enfant de
  tourelle. Verifie par `scan` (aucun `ghost_g`/`banshee_g`/`chopper_g`) + `diag -group`
  (le parent ne reference que son hlmt de chassis, lui-meme, et des `weap` SANS modele).
  Sprites V4 laisses tels quels.
- **Planche** : `PLANCHE_ASSEMBLAGE_ENFANTS_CORRIGE_2026-09-01.png` (fond sombre) : pour
  chacun des 4, chassis seul / tourelle seule / assemblage corrige (colonne assemblage
  bordee de vert), + rangee bonus Ghost/Banshee/Chopper.

## 1. Modes resolus (par l'outil, pas devines)

`scan -modules="globals,common,multiplayer(any) + pc:multiplayer + pc:common"` — 46/67 vehi
resolus. Resolveur corrige (plancher `0x100` + parasites nommes, chaine hlmt d'abord).

| vehicule | vehi (chassis) | mode chassis | tourelle-enfant (vehi) | mode tourelle | module modes |
|---|---|---|---|---|---|
| Scorpion | `0x0000d3db` | `0x39918211` | `scorpion_c` `0x0000d4ff` + `scorpion_g` `0x0000d500` | `0x60dd0e4e` (canon) + `0x8bf43b79` (collier) | pc/globals |
| Warthog  | `0x00002705` | `0x561f2ca7` | `warthog_g` `0x0000e0ca` | `0x0000e0da` (LAAG) | chassis pc/globals ; LAAG pc/multiplayer |
| Wraith   | `0x00002706` | `0x3a98ee2d` | `wraith_g` `0x001b33fc` | `0x2b282da9` (tourelle plasma) | pc/multiplayer+common |
| Phantom  | `0x000026f2` | `0x8acf1ab6` | `phantom_g` `0x0000515c` | `0x56fd2500` (tourelle PARTAGEE) | pc/multiplayer+common |

Note : `0x56fd2500` est un mode de tourelle PARTAGE (plusieurs `vehi` y pointent :
`phantom_g` `0x0000515c`, `turret_g` `0x000025a5` et `0xde214073`, `0x7226ebb5`). C'est la
racine de la limite Phantom (§4).

## 2. La methode corrigee, etape par etape

1. **Canevas fixe partage.** `assemble -cadre=7 -cellmm=10` : demi-emprise +/-7 m, 10 mm/px,
   origine vehicule au centre, `MargePx=0` -> chaque piece est un PNG 1413x1413 NON rogne, ou
   le pixel central = l'origine du modele. Deux pieces au meme `-cadre`/`-cellmm` tombent sur
   la MEME grille de pixels (contrainte verifiee par `compose2d` : refus si tailles differentes).
   `cadre=7` couvre le plus grand (le Phantom-fichier fait ~11 m, demi-emprise ~5,5 m — les
   modeles sont en unites COMPACTES, pas a l'echelle « lore »).
2. **Deux passes de modules** (contrainte RAM, un seul gros module a la fois) :
   - Passe 1 = `pc:globals` : Scorpion (chassis+canon+collier) + Warthog chassis.
   - Passe 2 = `pc:multiplayer`+`pc:common` : Warthog LAAG + Wraith (chassis+tourelle) +
     Phantom (chassis+tourelle).
   MEME `-cadre`/`-cellmm` aux deux passes (le Warthog a son chassis en passe 1 et sa LAAG en
   passe 2 : leurs canevas DOIVENT coincider pour composer).
3. **Ordre peintre.** `compose2d -in="chassis[,collier],tourelle" -out=...` superpose en
   source-over du fond vers le sommet ; la TOURELLE en dernier -> toujours au-dessus, jamais
   occultee. Le compose2d ROGNE le composite final (une fois, ensemble) — jamais les pieces
   separement (sinon l'offset relatif chassis<->tourelle serait perdu).
   - Scorpion : `chassis + collier + canon` (canon au sommet).
   - Warthog : `chassis + LAAG`.
   - Wraith : `chassis + wraith_g`.
   - Phantom : `chassis + phantom_g`.

Pourquoi PAS le z-buffer in-module seul : le retour utilisateur est correct — la coque peut
gagner en hauteur (Z) la ou la tourelle devrait etre vue, et le z-buffer partage la laisserait
alors sous la coque. L'ordre-peintre 2D garantit la superposition independamment de la hauteur.

## 3. Verdict a l'oeil (obligatoire, fait) — 3 criteres : placement / echelle / superposition

Planche `PLANCHE_ASSEMBLAGE_ENFANTS_CORRIGE_2026-09-01.png`.

- **Scorpion** (asm 260x388) — **NET**. Le chassis seul n'a que l'anneau de tourelle vide ; le
  composite pose le canon (`scorpion_c`) + le collier (`scorpion_g`) a l'aplomb de l'anneau,
  barillet le long de l'axe debordant la coque. Centre sur la monture, au-dessus, a l'echelle.
- **Wraith** (asm 304x313) — **BON**. Le chassis seul a une douille hexagonale centrale vide ;
  le composite y pose la tourelle plasma circulaire (`wraith_g`, anneau + fut) centree, au-dessus.
  (Le gros capot du mortier est deja dans le chassis ; `wraith_g` = la tourelle plasma du poste
  de tir, ~0,7 x 1,15 m, montee dans la douille.)
- **Warthog** (asm 110x365) — **BON**. LAAG (`warthog_g`) posee au centre-arriere, au-dessus.
  Barillet allonge (~2,5 m avec la monture) : c'est la POSE DE REPOS authored (arme pointant
  vers l'arriere), fidele au modele. Sprite etroit+long parce que le Warthog l'est (chassis
  ~1,1 x 2,3 m + barillet).
- **Phantom** (asm 742x1080) — **LIMITE, dit honnetement**. Chassis NET et reconnaissable
  (nez, nacelles/ailes, derive). Mais `phantom_g` (mode PARTAGE `0x56fd2500`) est un modele
  MINUSCULE (~0,32 x 0,64 m au rendu) authored a l'ORIGINE (0,0,0) du vehicule — PAS co-repere
  a la position du menton comme le sont `scorpion_c`/`warthog_g` dans leur repere vehicule.
  Compose en ordre-peintre, il devient VISIBLE (au-dessus) mais comme un petit blob au centre
  du fuselage, negligeable a l'echelle du Phantom. Il ne produit pas de tourelle distinctive en
  vue de dessus. (Confirme au prealable : a translation nulle en z-buffer, la piece etait
  ENFERMEE dans le fuselage — identique au chassis sur les 3 projections dessus/profil/face.)

## 4. Pourquoi le Phantom est different (co-repérage vs modele partage)

L'insight « modeles enfants authored CO-REPERES » (rapport precedent) tient pour les enfants
authored DANS le repere du vehicule parent (`scorpion_c`, `scorpion_g`, `warthog_g`, `wraith_g`)
— ils tombent pile a leur monture a translation nulle. Il NE tient PAS pour une tourelle dont
le modele est PARTAGE entre plusieurs vehicules (`phantom_g -> 0x56fd2500`) : ce modele est
authored a son propre origine (0,0,0) comme une tourelle autonome ; le jeu le place au menton
du Phantom par la TRANSFORMEE DU MARQUEUR d'attache au runtime — transformee qui n'est PAS
extractible du render_model (elle vit dans les noeuds du squelette / bloc d'attachement, non
parse ; etabli au rapport precedent §4). Sans elle, le mieux honnete est translation nulle ->
centre du fuselage. Aucun placement invente (le best-guess produirait un sprite douteux).

## 5. Bonus — Ghost / Banshee / Chopper : arme INTEGREE, pas de vehi-enfant

- **`scan`** : les familles `ghost`/`banshee`/`chopper` n'apparaissent QUE comme pieces de
  chassis (`ghost_b_*`, `banshee_b_*`, `chopper_b_*`). AUCUN `ghost_g`/`banshee_g`/`chopper_g`
  ni turret-`vehi` de leur famille.
- **`diag -group="vehi,weap,mode,hlmt,bloc,mach"`** sur les 3 parents :
  - Ghost `0x0000d3dc` -> hlmt chassis `0x3b3038e6`, SELF `0x0000d3dc`, `weap 0x00015435`.
  - Banshee `0x000026ed` -> hlmt `0xdf38bc96`, SELF, `weap 0x0000aa68`+`0x0000aa69`.
  - Chopper `0x002ba902` -> hlmt `0xfe29d23d`, SELF, `weap 0xb40e9618`(+`0xbd252644`).
  Aucun n'a de reference vers un AUTRE `vehi` (pas d'enfant de tourelle). Les `weap` referencees
  n'ont pas de render_model (etabli V4 §10.4) : la geometrie d'arme visible est BAKEE dans le
  mode du chassis. Meme pattern que le Wasp.
- **Verdict** : rien a regenerer pour Ghost/Banshee/Chopper. Sprites V4 laisses tels quels.

## 6. Fichiers

Livrables (worktree, NON commites) :
- `sprites_v4/scorpion.png` — REGENERE ordre-peintre (chassis + collier + canon), 260x388.
- `sprites_v4/warthog.png`  — REGENERE ordre-peintre (chassis + LAAG), 110x365.
- `sprites_v4/wraith.png`   — REGENERE ordre-peintre (chassis + tourelle plasma), 304x313.
- `sprites_v4/phantom.png`  — REGENERE ordre-peintre (chassis + phantom_g), 742x1080
  (turret negligeable — cf. §3/§4).
- `PLANCHE_ASSEMBLAGE_ENFANTS_CORRIGE_2026-09-01.png` — planche maitresse (992x1722).

NON touches (hors perimetre demande) : variantes permutation `rockethog.png`,
`warthog_gauss.png`, `razorback.png` (rendus mono-modele, arme dans le render_model, pas de
tourelle-enfant), et `warthog_laag.png`, `warthog_v2.png`, `scorpion_canon_enfant.png`.
Planche interimaire `PLANCHE_ASSEMBLAGE_WRAITH_PHANTOM_2026-09-01.png` (methode in-module,
superseded) : supprimee.

Verification : `go build -o v5tool.exe ./cmd/vehicle-sprite` OK (GOCACHE dedie, CC winlibs).
Aucun code Go modifie -> gofmt trivialement propre.

## 7. Reproduire

```
# env : GOCACHE dedie + CC winlibs + CGO_ENABLED=1 ; depuis apps/go-api
go build -o v5tool.exe ./cmd/vehicle-sprite

# PASSE 1 (pc:globals) — canevas fixe cadre=7 cellmm=10
v5tool.exe assemble -modules="pc:globals-rtx-new.module,globals-rtx-new.module" \
  -out=OUT -cadre=7 -cellmm=10 \
  -batch="scorp_chassis=0x39918211;scorp_canon=0x60dd0e4e;scorp_collier=0x8bf43b79;wthg_chassis=0x561f2ca7"

# PASSE 2 (pc:multiplayer+common) — meme cadre/cellmm
v5tool.exe assemble -modules="pc:multiplayer-rtx-new.module,pc:common-rtx-new.module" \
  -out=OUT -cadre=7 -cellmm=10 \
  -batch="wthg_laag=0x0000e0da;wraith_chassis=0x3a98ee2d;wraith_g=0x2b282da9;phantom_chassis=0x8acf1ab6;phantom_g=0x56fd2500"

# ORDRE PEINTRE (tourelle en dernier) + rognage du composite
v5tool.exe compose2d -in="OUT/scorp_chassis.png,OUT/scorp_collier.png,OUT/scorp_canon.png" -out=scorpion.png
v5tool.exe compose2d -in="OUT/wthg_chassis.png,OUT/wthg_laag.png"     -out=warthog.png
v5tool.exe compose2d -in="OUT/wraith_chassis.png,OUT/wraith_g.png"    -out=wraith.png
v5tool.exe compose2d -in="OUT/phantom_chassis.png,OUT/phantom_g.png"  -out=phantom.png

# bonus : verif absence d'enfant
v5tool.exe scan  -modules="...any... + pc:multiplayer + pc:common"          # 0 ghost_g/banshee_g/chopper_g
v5tool.exe diag  -modules="...idem..." -id="0x0000d3dc,0x000026ed,0x002ba902" -group="vehi,weap,mode,hlmt"
```

## 8. CR honnete — synthese

**Certain / net :** Scorpion (canon+collier), Wraith (tourelle plasma), Warthog (LAAG) sont
places, a l'echelle, et au-dessus (ordre-peintre). Ghost/Banshee/Chopper : arme integree,
aucun enfant (scan+diag).

**Limite dite sans maquiller :** le Phantom. `phantom_g` est un modele de tourelle PARTAGE,
minuscule et authored a l'origine ; sans la transformee du marqueur (non extractible), le
composite honnete le met au centre du fuselage ou il est negligeable. Le sprite Phantom livre
est donc, en pratique, le chassis (net) + un blob central. Pas de placement invente.

**Choix (non un fait) :** l'ordre des couches Scorpion (collier puis canon) et le fait que la
LAAG au repos pointe vers l'arriere sont ce que donnent les modeles ; aucune orientation propre
a la tourelle n'a ete introduite (elle suit le chassis, co-reperee — convention V4 « +X = avant
= haut », le rejeu tourne l'icone par le cap).

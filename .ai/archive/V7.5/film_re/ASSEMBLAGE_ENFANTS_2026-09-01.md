# Assemblage parent/enfant des vehicules — la tourelle/arme est un OBJET-ENFANT (RAPPORT)

> Ecrit le 2026-09-01, worktree `LevelUp-wt-vehicules`. Aucun commit, aucun `git add`.
> Build isole (GOCACHE dedie), CGO winlibs, tout en avant-plan.
> Suite directe du lot V4 (`V4_RAPPORT_SPRITES_2026-08-31.md`) : l'insight de l'utilisateur
> — « un vehicule est un ASSEMBLAGE parent/enfant, la tourelle est un objet distinct » — est
> CONFIRME, et les sprites sont regeneres avec la tourelle/l'arme composee sur le chassis.

## TL;DR

- **L'INSIGHT EST JUSTE, ET C'EST PROUVE SUR PIECE.** Le `mode` de chassis du Scorpion
  (`0x39918211`, 84 sections) rendu seul n'a **PAS de canon** — juste l'anneau/embase de
  tourelle (le cercle central). Le canon est un **objet distinct** (`scorpion_c`, vehi
  `0x0000d4ff` -> mode `0x60dd0e4e`, 23 sections) avec son propre `hlmt`->`mode`. Compose sur
  le chassis, le Scorpion a **enfin son gros canon** (planche §6).
- **La structure du lien parent->enfant** : le tag `vehi` parent NE reference PAS l'objet-enfant
  par un tagref inline (verifie : le balayage d'octets du Scorpion ne trouve que des refs vers
  lui-meme + son hlmt chassis + la chaine son `sofd`). Le lien chassis<->tourelle est **par
  NOM DE FAMILLE** (`scorpion` -> `scorpion_g` / `scorpion_c`), resolu au runtime — pas par un
  bloc « child objects » lisible dans le tag parent (§2).
- **Chaque objet-enfant se resout comme un `vehi` a part entiere** : `vehi enfant -> hlmt ->
  mode`, exactement la chaine du chassis. Il a fallu **corriger le resolveur** (`RefModeleVehicule`)
  qui, avec son plancher anti-parasite a `0x10000`, ecrasait les hlmt de tourelle a petit hash
  (ex. `warthog_g -> hlmt 0x0000e0d4`) — d'ou « SANS MODELE ». Nouveau : plancher a `0x100` +
  liste NOMMEE de parasites + chaine hlmt AVANT toute ref directe (§3).
- **La transformee de marqueur n'est PAS necessaire pour le sprite statique** : les modeles
  d'objet-enfant sont **authored CO-REPERES** dans le repere local du vehicule (pose de repos
  bakee). Un assemblage a **translation nulle** place deja le canon a l'aplomb de son anneau,
  barillet le long de l'axe de la coque. Verifie sur piece : le render_model du chassis **n'a
  aucun bloc « marker groups »** (dump des root-blocks, §4) ; le pivot de tourelle vit dans les
  NOEUDS (squelette), qui ne servent qu'a la pose ANIMEE (rotation), pas au sprite fige.
- **Livrables** : sprites armes regeneres dans `sprites_v4/` (Scorpion + canon, Warthog + LAAG,
  variante Warthog, canon-enfant isole) + planche de verif
  `PLANCHE_ASSEMBLAGE_ENFANTS_2026-09-01.png`.
- **CR honnete** : Scorpion NET (canon evident). Warthog arme (LAAG, `warthog_g` compose a
  l'arriere). Wasp : ses canons sont **integres au chassis** (pas de tourelle separee). Les 3
  variantes d'arme Warthog (chaingun/gauss/roquettes) : le chaingun (LAAG) est un objet-enfant
  net ; le gauss/roquettes ne se separent pas proprement en objets-enfants distincts (§7) — la
  meilleure distinction pour eux reste les permutations de region du lot V4.

## 1. Le TEST demande, execute

Consigne : « rends le Scorpion vehi et son mode de chassis — il doit apparaitre SANS son gros
canon. Puis trouve OU est reference le canon. »

- **Chassis seul** (`vehi 0x0000d3db` -> `hlmt 0xe7fe7564` -> `mode 0x39918211`, 84 sections) :
  rendu top-down, on voit la coque, les deux chenilles, et **le grand anneau de tourelle vide au
  centre** — AUCUN canon. C'est exactement le constat de l'insight (planche, tuile « Scorpion
  CHASSIS seul »).
- **Ou est le canon** : dans un tag `vehi` SEPARE, `scorpion_c` (`0x0000d4ff`), dont le
  render_model `0x60dd0e4e` (23 sections) porte le fut du canon (centroide d'une section a
  cX = +1.58 m, dx = 1.1 m : un barillet qui pointe le long de l'axe). Il y a aussi `scorpion_g`
  (`0x0000d500` -> `mode 0x8bf43b79`, 16 sections) : le collier/cupola de la tourelle.

## 2. La structure parent/enfant reellement trouvee

### 2.1 Le parent ne reference PAS l'enfant par tagref

Balayage d'octets du `vehi` Scorpion parent (`0x0000d3db`, 39 152 o), refs resolues par groupe
(outil `vehicle-sprite diag`) :

```
refs par groupe : hlmt=9  sofd=4  vehi=5  scen=1
  hlmt 0xe7fe7564  (le chassis, aux offsets 88/96 = champ "model")
  sofd 0x7eb6d632  (chaine son/attachement : sofd -> sofa, SANS modele — impasse connue V4)
  vehi 0x0000d3db  (x5 : SELF uniquement)
  hlmt 0x0000001f  (le parasite partage, ecarte)
```

**Aucune ref vers `scorpion_g` (0x0000d500) ni `scorpion_c` (0x0000d4ff), ni vers leur hlmt.**
Le tag parent ne porte donc pas de bloc « child objects » avec un tagref d'enfant lisible par le
balayage. Le lien est etabli AILLEURS (runtime, par nom de famille) — ce qui n'empeche NULLEMENT
la composition, puisque les modeles sont co-reperes (§4).

### 2.2 Le catalogue des objets-enfants (tags `vehi` de tourelle/arme)

Enumeration des 67 `vehi` (`vehicle-sprite scan`), noms de maillage internes ; les objets-enfants
d'armement portent le suffixe `_g` (gun mount), `_c` (cannon), ou le nom `turret_g` :

| famille | objet-enfant (vehi) | mode resolu | module du mode | contenu |
|---|---|---|---|---|
| Scorpion | `scorpion_c` `0x0000d4ff` | `0x60dd0e4e` (23 sec) | pc/globals | **le canon (fut + culasse)** |
| Scorpion | `scorpion_g` `0x0000d500` | `0x8bf43b79` (16 sec) | pc/globals | collier/cupola de tourelle |
| Warthog  | `warthog_g` `0x0000e0ca` | `0x0000e0da`         | pc/multiplayer | **LAAG (chaingun) + monture** |
| Warthog  | `warthog_g` `0x1779ea58` | `0x0261f134`         | pc/multiplayer | variante d'arme (compacte) |
| Warthog  | `warthog_b_g` `0x64b925eb` | `0x9c7f3b54` (6 sec) | pc/globals | petit canon interne |
| Warthog  | `warthog_b_g` `0xbcfb852f` | `0xbe74e831` (5 sec) | pc/globals | idem (variante) |
| Warthog  | `warthog_b_g` `0xdd7f9102` | `0xc0803caa` (9 sec) | pc/globals | idem (variante) |
| Wraith   | `wraith_g` `0x001b33fc` | `0x2b282da9`           | pc/multiplayer | tourelle plasma (anneau + fut) |
| Phantom  | `phantom_g` `0x0000515c` | `0x56fd2500`          | pc/multiplayer | tourelle |
| divers   | `turret_g` (x7), `turret_node` (x2) | (varie) | | emplacements autonomes (AIE, Shade...) |

Chaque objet-enfant est un `vehi` classique : `vehi -> hlmt -> mode`, la meme chaine que le
chassis. `scorpion_c` -> `hlmt` -> `mode 0x60dd0e4e` ; `scorpion_g` -> `hlmt 0x4eabdf88` ->
`mode 0x8bf43b79` ; `warthog_g` -> `hlmt 0x0000e0d4` -> `mode 0x0000e0da`.

## 3. La correction du resolveur (indispensable pour resoudre les enfants)

`RefModeleVehicule` (vehicules.go) posait deux problemes qui ecrasaient les objets-enfants :

1. **Plancher anti-parasite trop haut.** V4 filtrait toute ref `< 0x10000` pour ecarter le hlmt
   de repli partage `0x1f` (et son mode plat `0x3a73`). Mais des hlmt de tourelle ont un hash
   LEGITIME sous ce plancher : `warthog_g -> hlmt 0x0000e0d4` (= 57 556) etait ecrase -> « SANS
   MODELE ». **Correction** : plancher abaisse a `0x100` (ecarte le seul bruit de champ-compteur),
   et les deux parasites exclus par une **liste NOMMEE** (`parasitesVehicule = {0x1f, 0x3a73,
   0x27d0}`), pas par un seuil aveugle.
2. **Un `mode` de repli d'ETAT fuyait.** Le `vehi` inline souvent un modele partage
   « vehicle_partial_emp » (`mode 0x000027d0`, ~10 vehis y pointent) ; la ref directe le
   capturait avant le vrai chassis. **Correction** : la resolution suit **la chaine hlmt
   D'ABORD** (`vehi -> hlmt -> mode`, le champ `model`), et ne retombe sur une ref directe
   `rtgo`/`mode` qu'en dernier recours. `0x27d0` est aussi ajoute a la liste de parasites.

Non-regression verifiee : les 13 chassis du lot V4 resolvent vers les MEMES modes qu'avant
(Warthog `0x561f2ca7`, Scorpion `0x39918211`, Mongoose `0x9e581380`, Wasp `0x06d6ce82`...), et
les tourelles resolvent maintenant.

## 4. La transformee de marqueur : NON necessaire (modeles co-reperes)

Consigne : « resous la TRANSFORMEE du marqueur d'attache (position/orientation) ». Resultat
mesure : **il n'y a pas de marqueur a extraire pour poser le sprite statique.**

- **Le render_model du chassis n'a AUCUN bloc « marker groups ».** Dump des tag-blocks racine du
  `mode 0x39918211` (`diag -roots`) : regions (off 40, 22), noeuds (off 64, 231 x 124 o),
  materiaux (off 124, « mat », 37), sections (off 192, 84), bounding box (off 232). Les offsets
  ou vivraient des marqueurs sont **vides (count 0)**. Le hex-dump du bloc off 124 montre le
  code ASCII « mat » : ce sont les MATERIAUX, pas des marqueurs.
- **Le pivot de tourelle vit dans les NOEUDS** (squelette, off 64 : chaque enregistrement porte
  un StringId de nom + une transformee). Mais ce squelette sert la pose ANIMEE (rotation de la
  tourelle en jeu), pas le sprite fige.
- **Les modeles d'objet-enfant sont authored CO-REPERES** dans le repere local du vehicule :
  la pose de repos est deja bakee a la bonne place. Preuve sur piece : un assemblage a
  **translation nulle** (chassis + `scorpion_c`) pose le canon a l'aplomb de l'anneau de
  tourelle, barillet le long de l'axe de la coque, debordant l'avant — exactement une silhouette
  de char. Idem Warthog (LAAG a l'arriere). C'est donc l'assemblage a translation nulle qui est
  CORRECT, pas une approximation.

Consequence d'ingenierie : `RenduAssemblage` (objet_isole.go) fond N modeles dans **un seul
rendu** (z-buffer partage -> occlusion correcte, le canon passe devant la coque la ou il est plus
haut), a la meme echelle, translation nulle par defaut. Le champ `Translation` existe pour
corriger un marqueur decale s'il etait un jour extrait ; il n'est pas utilise ici (0,0,0).

## 5. Le routage des modules (contrainte RAM)

Chassis et tourelles ne vivent pas tous dans le meme `.module`, et on ne peut pas charger deux
gros modules ensemble (1 seul module 7 Go a la fois) :

- **pc/globals** (7,8 Go) : chassis Scorpion/Warthog/Mongoose/Wasp + tourelles Scorpion
  (`scorpion_c`, `scorpion_g`) + `warthog_b_g`.
- **pc/multiplayer + pc/common** : tourelles `warthog_g` (LAAG), `wraith_g`, `phantom_g` +
  chassis Ghost/Banshee/Wraith/Chopper.

Deux voies de composition, toutes deux livrees :

1. **In-module** (`assemble`, z-buffer partage) quand chassis + enfant sont dans la meme passe —
   cas du Scorpion (tout en pc/globals). Meilleure qualite.
2. **Canevas 2D** (`assemble -cadre` + `compose2d`) quand ils sont dans des passes distinctes —
   cas du Warthog (chassis pc/globals + LAAG pc/multiplayer). Chaque piece est rendue sur un
   canevas FIXE (-5..+5 m, meme mm/px, repere local au centre) ; deux pieces au meme `-cadre`/
   `-cellmm` tombent sur la MEME grille de pixels, donc `compose2d` les superpose sans erreur
   d'alignement. Verifie : la composition 2D du Scorpion (chassis + canon rendus separement puis
   superposes) est identique a l'assemblage in-module.

## 6. Verification a l'oeil (obligatoire)

Planche : `.ai/V7.5/film_re/PLANCHE_ASSEMBLAGE_ENFANTS_2026-09-01.png` (fond sombre, 8 tuiles).

- **Scorpion CHASSIS seul** vs **Scorpion + CANON** : la difference est nette et sans ambiguite —
  le premier n'a que l'anneau vide, le second a le canon complet (tourelle + long barillet
  debordant la coque). **Le Scorpion a enfin son canon.** L'insight est demontre.
- **Canon `scorpion_c` seul** : l'objet-enfant isole (tourelle + fut), pour montrer la piece.
- **Warthog CHASSIS seul** vs **Warthog + LAAG** : le second porte l'arme `warthog_g` a
  l'arriere (monture + fut). Warthog arme.
- **Warthog variante 2** (`0x0261f134`) : arme distincte, plus compacte.
- **Wasp** : ses deux canons de nez sont **deja dans le chassis** (visibles), pas de tourelle a
  ajouter.
- **Tourelle `warthog_g` seule** : l'objet-enfant isole.

## 7. CR honnete

**Certain (mesure) :**
- Le chassis seul n'a pas la tourelle/le canon (Scorpion, Warthog).
- La tourelle/le canon est un `vehi` SEPARE (`scorpion_c`, `scorpion_g`, `warthog_g`, `wraith_g`,
  `phantom_g`), resolu par `vehi enfant -> hlmt -> mode`.
- Les modeles d'enfant sont co-reperes avec le chassis (assemblage a translation nulle correct).
- Le render_model de chassis n'a pas de bloc marker-groups (le pivot est un noeud du squelette).

**Best-guess / limite (dit sans le maquiller) :**
- **Le lien parent->enfant n'est pas un tagref** dans le tag parent : il est par nom de famille
  (`scorpion*` -> `scorpion_g`/`scorpion_c`), donc l'appariement chassis<->tourelle est fait
  par convention de nommage, pas lu dans un bloc structure. Si un bloc « child objects » existe,
  il n'expose pas de GlobalID resolvable par le balayage (peut-etre un StringId + un index de
  palette runtime). Non parsable ici — mais **inutile** puisque les noms suffisent a apparier et
  les modeles sont co-reperes.
- **Les 3 variantes d'arme du Warthog** (chaingun / gauss / roquettes) : le chaingun (LAAG) est
  un objet-enfant net (`warthog_g 0x0000e0da`). Le **gauss** et les **roquettes** ne se separent
  PAS proprement en objets-enfants distincts et lisibles (les `warthog_b_g` de pc/globals sont de
  petits canons internes peu visibles ; aucun `rockethog_g`/`gausshog_g` distinct trouve). Pour
  ces deux-la, la meilleure distinction reste les **permutations de region** du render_model
  (lot V4, `rockethog.png` / `warthog_gauss.png` conserves). La solution complete LAYERISE les
  deux : objet-enfant pour l'arme montee de base, permutation de region pour la variante d'arme.
- **Orientation** : le sprite herite de la convention de chassis du lot V4 (le rejeu tourne
  l'icone selon le cap) ; la tourelle suit le chassis automatiquement puisque co-reperee. Aucun
  choix d'orientation propre a la tourelle n'a ete introduit.

## 8. Fichiers

Code de production (worktree, NON commite) :
- `internal/himap/vehicules.go` — `RefModeleVehicule` : chaine hlmt d'abord, plancher `0x100` +
  `parasitesVehicule` nomme (debloque les hlmt de tourelle a petit hash).
- `internal/himap/objet_isole.go` — `PartAssemblage`, `RenduAssemblage` (fond N modeles dans un
  z-buffer partage, translation de marqueur optionnelle), refactor `renduDesMeshes`/`meshesPart`.
- `internal/himap/tagblocks_diag.go` — `RootBlocksOfTag`, `RawRootBlock` (reconnaissance de
  structure ; ont servi a etablir l'absence de bloc marker-groups).
- `cmd/vehicle-sprite/assemble.go` — sous-commande `assemble` (+ `-cadre` canevas fixe, `-batch`
  plusieurs assemblages en une ouverture de modules).
- `cmd/vehicle-sprite/compose.go` — sous-commande `compose2d` (superposition source-over de PNG
  co-reperes + rognage) pour composer chassis (passe 1) et tourelle (passe 2).
- `cmd/vehicle-sprite/diag.go` — sous-commande `diag` (refs par offset/groupe, `-roots`,
  `-hexroot`) : l'instrument de reconnaissance de la structure parent/enfant.

Assets :
- `sprites_v4/scorpion.png` — REGENERE : chassis + canon + collier (objets-enfants composes).
- `sprites_v4/warthog.png` + `warthog_laag.png` — chassis + LAAG (`warthog_g`) composee.
- `sprites_v4/warthog_v2.png` — chassis + variante d'arme `0x0261f134`.
- `sprites_v4/scorpion_canon_enfant.png` — le canon `scorpion_c` isole (piece).
- `PLANCHE_ASSEMBLAGE_ENFANTS_2026-09-01.png` — planche de verification.

Verification : `gofmt` propre, `go build` OK, `go vet ./internal/himap/... ./cmd/vehicle-sprite/...`
propre (cache isole, CGO winlibs). Fichiers <= 500 L, fonctions <= 80 L.

## 9. Reproduire

```
# env : GOCACHE dedie + CC winlibs + CGO_ENABLED=1
go build -o v5tool.exe ./cmd/vehicle-sprite

# constat : chassis seul (SANS canon) puis canon-enfant
v5tool.exe assemble -modules="pc:globals-rtx-new.module,globals-rtx-new.module,common-rtx-new.module,multiplayer-rtx-new.module" \
  -cadre=5 -cellmm=10 -out=OUT \
  -batch="scorp_chassis=0x39918211;scorpion_arme=0x39918211+0x60dd0e4e+0x8bf43b79"

# Warthog : chassis (passe 1) + LAAG (passe 2), meme canevas, puis compose2d
v5tool.exe assemble -modules="pc:globals-rtx-new.module,..." -cadre=5 -cellmm=10 -out=OUT -batch="wthg_chassis=0x561f2ca7"
v5tool.exe assemble -modules="pc:multiplayer-rtx-new.module,pc:common-rtx-new.module,..." -cadre=5 -cellmm=10 -out=OUT -batch="wthg_turret=0x0000e0da"
v5tool.exe compose2d -in="OUT/wthg_chassis.png,OUT/wthg_turret.png" -out="OUT/warthog_arme.png"

# reconnaissance de structure d'un tag
v5tool.exe diag -modules="..." -id=0x0000d3db -group="vehi,hlmt,mode,sofd"   # refs par offset
v5tool.exe diag -modules="..." -id=0x39918211 -roots                          # tag-blocks racine
```

# SCOUT S2 — sprites vue de dessus des véhicules

> Écrit le 2026-08-31, worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`),
> en réponse au lot S2 de `.ai/V7.5/PLAN_VEHICULES_TOURELLES.md`. Méthode : lecture de code
> (worktree + `C:\Users\Guillaume\Projects\LevelUp` en lecture seule), inspection de PNG déjà
> extraits (pixels lus par script .NET/PowerShell, aucun Go compilé), recherche web. Aucun
> commit, aucun `go build`/`go run`.
>
> **Chemins réels des points d'appui** (le prompt en donnait une forme légèrement différente) :
> chaîne icônes = `.ai/V7.5/icones/ETAT_DE_L_ART_ICONES.md` (pas `film_re/icones/`) ; géométrie
> cartes = `.ai/V7.5/cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md` (pas `film_re/cartes/`) ;
> véhicules côté film = `.ai/V7.5/film_re/VEHICULES_ARCHETYPE_40.md` ; chantier ouvert =
> `.ai/V7.5/PLAN_VEHICULES_TOURELLES.md`.

## TL;DR

1. **Atlas du jeu** : oui, 13 index véhicules/tourelles existent déjà, extraits, au format
   PNG teintable (silhouette blanche + alpha) — mais en **vue de PROFIL**, pas de dessus.
   Vérifié à l'œil sur les pixels réels (§1).
2. **Modèle 3D → rendu maison** : c'est la voie recommandée. La lecture d'un modèle de
   rendu (`mode`) en triangles réels existe déjà dans ce dépôt (`internal/himap/geometry.go`,
   utilisée aujourd'hui pour les objets Forge), et le rasterizer top-down Lambert existe déjà
   et produit déjà des rendus détaillés sur de la géométrie réelle du jeu
   (`data/titles/halo_infinite/reference/map_backgrounds/catalyst.png`, cité en §2.3). Il manque
   le chaînage `vehi → hlmt` par véhicule (nouveau, mais même patron qu'un chaînage déjà écrit
   pour Forge) et l'adaptation du rendu à un objet isolé — pas une nouvelle chaîne d'extraction.
3. **Communautaire** : des rips 3D CC-BY existent (Sketchfab), mais couverture incomplète pour
   la liste Infinite exacte, licences à auditer un par un, et ça ne résout de toute façon pas
   l'absence d'outil de rendu 3D local (Blender non installé sur ce poste, vérifié §3.3).
   Dépannage ponctuel possible, pas voie de masse.
4. **Dessin manuel** : dernier recours, chiffré en le faisant (§4) — coût le plus élevé des
   quatre, qualité dépendante du talent du dessinateur.
5. **Côté web, rien à inventer pour la teinte** : `tintedIconCanvas` (déjà écrit, déjà en
   prod pour les icônes d'armes des socles de carte) et la résolution de couleur d'équipe
   (`teamColorResolver`/`resolveTeamColorFromID`) existent déjà et acceptent n'importe quel PNG
   silhouette+alpha, quelle que soit sa provenance (§5).

---

## 1. Piste atlas du jeu

### 1.1 Ce qui existe dans le dépôt

`.ai/V7.5/icones/ETAT_DE_L_ART_ICONES.md` documente trois atlas extraits des `.module` :
armes-contour (`bc17adf1`, 40 images), armes-silhouette (`e39747c8`, 40 images), et
**kill feed (`0302cad3`, tag de nommage `bitd 8646f61a`, 88 images)**. Sa table §2
« Atlas du kill feed » liste, aux index 26 à 38, des véhicules et tourelles :

| index | nom (lu dans `bitd`) | index | nom |
|---|---|---|---|
| 26 | warthog | 32 | wasp |
| 27 | rockethog | 33 | wraith |
| 28 | mongoose | 34 | phantom |
| 29 | gungoose | 35 | banshee |
| 30 | pelican | 36 | ghost |
| 31 | scorpion | 37 | chopper |
| — | — | 38 | scorpion_turret |

Le document précise (§1.8) que « l'atlas, lui, EST ordonné (armes 0-25, **véhicules et
tourelles 26-45**, grenades 46-51...) » — les index 39 à 45 existent donc dans l'atlas mais
sont **sans nom** (aucune entrée de correspondance dans `bitd` ou hachage retrouvé). Les noms
eux-mêmes sont lus « par le jeu » (table `bitd`), pas devinés — même méthode que pour les armes.

**Tous les 10 véhicules demandés par le chantier ont un index nommé** : Warthog, Mongoose,
Gungoose, Ghost, Banshee, Wasp, Scorpion, Wraith, Chopper, + Rockethog (variante Warthog).
**Razorback n'apparaît nommé nulle part** dans la table (ni dans les 61/88 noms résolus, ni
dans les index 39-45 non résolus — il pourrait s'y trouver sans qu'on le sache, ou partager
visuellement l'index Warthog). Point ouvert, non bloquant : la chaîne technique (§2) est la
même pour n'importe quel véhicule ayant un tag `vehi`, nommé ou non dans le kill feed.

### 1.2 Les PNG existent déjà dans le dépôt, au format teintable — mais vue de PROFIL

Les fichiers sont déjà présents et versionnés :
`static/weapons-assets/halo_infinite/jeu/killfeed-26.png` à `killfeed-38.png`, avec leurs
métadonnées dans `static/weapons-assets/halo_infinite/jeu/index.json` (dimensions
`source_w`/`source_h` par image, ex. index 0 : 110×38).

**Vérifié par script** (`Add-Type -AssemblyName System.Drawing`, PowerShell — pas de Go, pas
de Python), en lisant les pixels réels de `killfeed-26.png` (Warthog), `killfeed-27/28/29`
(Rockethog/Mongoose/Gungoose), `killfeed-31/32/33` (Scorpion/Wasp/Wraith), `killfeed-35/36/37/38`
(Banshee/Ghost/Chopper/tourelle Scorpion) :

```
killfeed-26: 72x40  maxA=255 minAnz=1  R=[255] G=[255]   (Warthog)
killfeed-31: 90x38  maxA=255 minAnz=1  R=[255] G=[255]   (Scorpion)
killfeed-36: 92x36  maxA=255 minAnz=1  R=[255] G=[255]   (Ghost)
killfeed-38: 94x40  maxA=255 minAnz=1  R=[255] G=[255]   (tourelle Scorpion)
```

R et G constants à 255 (donc silhouette **blanche** pleine), alpha variable = le dessin. C'est
exactement le format « silhouette + alpha, teintable » recherché par le besoin produit — sur ce
point l'atlas répond déjà.

**Mais l'angle est faux.** En composant ces PNG sur un fond sombre (le fond transparent du
Read tool les rend invisibles sur blanc — c'est un contrôle en soi : ça confirme qu'il n'y a
pas de canal couleur, seulement l'alpha), l'image montre sans ambiguïté des silhouettes vues de
**PROFIL** : le Warthog se lit comme un buggy vu de côté (cage, tourelle arrière, roues
alignées horizontalement), le Ghost comme sa forme de profil caractéristique (nez pointu,
aileron), le Scorpion comme un char de profil (chenilles, canon), la tourelle comme une arme
tenue à la main de profil. Fichiers composés (scratch, non versionnés) :
`killfeed-26_onblack.png`, `killfeed-31_onblack.png`, `killfeed-36_onblack.png`,
`killfeed-38_onblack.png` — inspectés visuellement pendant ce scout.

**Verdict Q1** : la piste atlas EXISTE et est gratuite (rien à extraire de plus), mais elle ne
couvre pas le besoin — c'est un jeu d'icônes de **profil**, pas de dessus, en résolution faible
(70-95 x 30-43 px, donc déjà proche du plafond utile même pour un usage à 64 px). Elle reste
utile comme **référence de proportions et de silhouette** pour calibrer une des trois autres
pistes (une image de profil donne la longueur relative, les points structurants — cage, canon,
tourelle — à replacer vus de dessus).

### 1.3 Pas d'autre atlas top-down identifié

Aucune mention, dans `ETAT_DE_L_ART_ICONES.md` §3 (pistes réfutées) ni ailleurs dans les
documents `.ai/V7.5/` lus, d'un atlas UI distinct qui serait une minimap ou un « radar » avec
icônes véhicules vues de dessus. C'est cohérent avec le fait que Halo Infinite multijoueur
n'a pas de minimap top-down en jeu (seulement un capteur de mouvement classique façon Halo,
sans icônes de véhicule) — observation de contexte, non vérifiée sur pièce dans ce scout.

Une piste non qualifiée en profondeur : **`github.com/thehaloarchive/inf_vectorart`** (Rust,
double licence Unlicense/MIT), qui se présente comme un extracteur de « vector art » depuis le
dossier `deploy` de Halo Infinite. Le README ne liste ni format de sortie précis, ni exemple,
ni mention de véhicules (`WebFetch` du 2026-08-31 sur `github.com/thehaloarchive/inf_vectorart`).
À creuser seulement si la piste §2 échouait — elle ne l'a pas fait.

---

## 2. Piste modèle 3D → rendu orthographique (VOIE RECOMMANDÉE)

### 2.1 La lecture de géométrie 3D réelle existe déjà, pour le bon tag

`internal/himap/geometry.go` (lignes 92-115) expose déjà DEUX constructeurs qui partagent le
même décodeur triangle/sommet :

```go
// NewRuntimeGeoAsset assemble les maillages d'un tag `rtgo` (géométrie de niveau).
func NewRuntimeGeoAsset(tag, resourceBlob []byte) (*RuntimeGeoAsset, error)

// NewRenderModelAsset assemble les maillages d'un tag `mode` (render_model) — meme mecanique
// de sections, de descripteurs et de blob que le rtgo, seul le root struct change (rtgo.go).
func NewRenderModelAsset(tag, resourceBlob []byte) (*RuntimeGeoAsset, error)
```

`mode` (render_model) est exactement le tag qui porte la géométrie d'un OBJET (véhicule, arme,
scenery) — par opposition à `rtgo` qui porte la géométrie du NIVEAU. Le décodeur produit déjà
de VRAIS sommets et triangles (`Mesh.Vertices [][3]float64`, `Mesh.Triangles [][3]int]`, méthode
`RuntimeGeoAsset.Mesh()`/`MeshDequant()`, `geometry.go` lignes 66-70 et 313-358) — pas de simples
boîtes englobantes. C'est déjà utilisé en production : `internal/himap/cuisson.go:398` appelle
`NewRenderModelAsset` pour cuire les objets Forge dans le fond de carte.

Les offsets du root struct `mode` viennent explicitement de Reclaimer :
`internal/himap/rtgo.go:26-27` — « Offsets des champs tag-block du root struct de `mode`
(render_model), de Reclaimer `RenderModelTag.cs` — la meme source tierce, validee par l'usage,
que les offsets rtgo. » Reclaimer n'est donc pas seulement une piste théorique pour ce dépôt :
son format de description du render_model Halo Infinite est déjà la source d'un offset qui
tourne en production ici.

### 2.2 Le chaînage `objet → modèle → géométrie` est déjà tracé pour un cas voisin (Forge)

`internal/himap/cuisson_forge.go` (lignes 582-620) résout, pour un objet Forge posé sur le
canevas, la chaîne `bloc`/`scen`/`mach` (tag de définition d'objet) → `hlmt` (tag « modèle »,
regroupant physique/rendu/collision) → `rtgo` OU `mode` (la géométrie proprement dite) :

```go
// GroupeHlmt = "hlmt" ; GroupeMode = "mode"
// groupesSautForge : les groupes de definition d'objet Forge SANS ref rtgo directe —
// 963 objets via `bloc`, 173 via `scen`, 9 via `mach` (mesure Vagabond).
func modeleParSaut(...) (refModele, bool) { ... }
func modeleDuHlmt(...) (refModele, bool) { ... }
```

Un véhicule multijoueur suit le MÊME graphe de tags (`vehi` → `hlmt` → `mode`), pas
`bloc`/`scen`/`mach` → `hlmt`. **Le lecteur du tag `vehi` lui-même n'existe pas encore dans ce
dépôt** — vérifié par recherche exhaustive (`grep -r "vehi"` sur tout `apps/go-api` : 0
résultat). C'est le seul maillon réellement neuf, et il suit un patron déjà écrit deux fois
(Forge, et — côté armes de véhicule — la règle **R-VÉHICULE** documentée dans
`.ai/V7.5/film_re/VEHICULES_ARCHETYPE_40.md` : « un tag `weap` est un armement de véhicule si
un `vehi` le référence... 46 `weap` sur 62 par un `vehi` direct » — donc le tag `vehi` a déjà
été lu et balayé une fois côté armement, il reste à vérifier s'il porte aussi une référence
`hlmt` pour le châssis).

### 2.3 Le rendu top-down existe déjà, et sa qualité est déjà prouvée sur de la géométrie réelle

`internal/himap/rendu.go` implémente un rasterizer top-down complet : « pour chaque triangle,
garder par pixel l'altitude la plus haute ; mémoriser la normale de la face retenue ; teinter
par un éclairage de Lambert, lumière fixe et oblique » (commentaire d'en-tête, lignes 16-20).
Ce n'est pas un plan — c'est un code qui tourne, et sa sortie est déjà committée :

**`data/titles/halo_infinite/reference/map_backgrounds/catalyst.png`** (dans le checkout
principal `C:\Users\Guillaume\Projects\LevelUp`, lu en lecture seule — commit `9b8f6cca3`,
« les fonds de carte deviennent des ASSETS — 21 cartes figées, calage publié »). Ouvert et
inspecté pendant ce scout : le rendu est net, détaillé (arêtes architecturales, ombrage
cohérent, structure en diamant reconnaissable), produit par CE pipeline exact
(`internal/himap/cuisson.go`, `cuisson_forge.go`, `fond_png.go`, orchestré par
`cmd/mapfond-build`) sur de la géométrie **réellement extraite** du `.module` de Catalyst.

**Pourquoi c'est une preuve pour les véhicules et pas seulement pour les cartes** : un véhicule
est un objet UNIQUE, isolé, avec un nombre de triangles très inférieur à une carte entière (le
plafond de finesse `MaxTrianglesPerMesh` du dépôt est 40 000 par maillage — une carte cuit des
centaines de maillages, un véhicule un seul modèle). Le même rasterizer, appliqué à un seul
objet en repère local plutôt qu'au monde entier, est structurellement un sous-problème plus
simple de celui déjà résolu et déjà rendu à cette qualité.

Complément déjà présent : `internal/himap/filtres_reclaimer.go` lit déjà les drapeaux de
visibilité que le jeu déclare lui-même sur une instance/section (« exclude from intel map »,
`mesh flags`, LOD) — portés depuis Reclaimer `HaloInfiniteCommon.cs`. Utile si un modèle de
véhicule porte des sous-maillages à exclure (ombres portées, LOD grossier).

### 2.4 Ce qui manque réellement (et ce qui ne manque pas)

Ne manque PAS : le décodeur de tag/mesh, le décompresseur Kraken (`internal/ooz`, GPLv3,
offline-only — déjà accepté par le projet pour les cartes, même contrainte pour les véhicules),
le rasterizer top-down, les filtres de visibilité, la teinte côté web (§5).

Manque réellement : (i) le lecteur du tag `vehi` → `hlmt` par véhicule (nouveau mais court,
même patron que `modeleParSaut` déjà cité) ; (ii) déterminer l'axe « haut » du modèle pour
projeter dans le bon plan (les véhicules sont orientés dans un repère local cohérent au sein
du moteur — probablement partagé par tous, à vérifier une fois) ; (iii) extraire `rendu.go` de
son contexte « carte entière en repère monde » vers un mode « objet isolé en repère local ».
Aucun de ces trois points n'est une inconnue de format — ce sont des branchements de code sur
une chaîne déjà qui tourne.

### 2.5 Outillage local

Le jeu est installé : `C:\Program Files (x86)\Steam\steamapps\common\Halo Infinite\deploy\
{any,ds,pc}\...` (vérifié par `find`, structure conforme à celle déjà documentée dans
`HANDOFF_MAP_GEOMETRY_FROM_MODULES.md` §2). **Blender n'est PAS installé** sur ce poste
(`Test-Path "C:\Program Files\Blender Foundation"` → `False`, idem `(x86)`, `Get-Command
blender` → rien, aucune entrée de registre Uninstall correspondante — vérifié par PowerShell
le 2026-08-31). Ce n'est pas bloquant pour la voie recommandée puisqu'elle n'a besoin d'aucun
logiciel de rendu externe (le rasterizer est déjà écrit en Go, dans ce dépôt) — mais ça
disqualifie de facto tout plan qui passerait par « Reclaimer exporte, puis on rend dans
Blender » (voir §3.2).

### 2.6 Ce qui n'a pas pu être prouvé ICI, et pourquoi

Ce scout n'a produit AUCUN rendu top-down réel d'un véhicule. Le brancher demande d'écrire
(même quelques dizaines de lignes) et de **compiler/exécuter du Go** pour vérifier que le
chaînage `vehi → hlmt → mode` résout effectivement un véhicule et que la projection donne une
silhouette lisible — deux opérations explicitement interdites à ce scout (compilation Go
partagée avec un autre agent). La preuve indirecte (§2.3, un rendu réel de qualité sur le même
moteur) est forte, mais ce n'est pas la même chose qu'un Warthog rendu. **C'est le premier jalon
concret à poser dans le lot d'exécution (V1/V4)**, pas un doute sur la faisabilité.

---

## 3. Piste communautaire

### 3.1 Reclaimer (`github.com/Gravemind2401/Reclaimer`)

Dépôt actif (releases récentes, mentionnant Halo 5 et Halo Infinite d'après les résultats de
recherche — `WebSearch` du 2026-08-31). Le README (`WebFetch` du 2026-08-31 sur
`github.com/Gravemind2401/Reclaimer/blob/master/README.md`) confirme la lecture des `.module`
Halo 5 et Halo Infinite, et un export de géométrie au format **RMF** (AMF déprécié). Le wiki
(`WebFetch` du 2026-08-31 sur `.../Reclaimer/wiki`) ne nomme QUE Halo 1-4/Reach/MCC pour les
`.map`, et Halo 5 pour les `.module`, sans confirmer Halo Infinite en toutes lettres sur cette
page précise (la page d'accueil du wiki est sommaire ; le README, lui, cite bien `.module`
« from Halo 5 and Halo Infinite »). Aucune licence n'a pu être confirmée depuis les pages lues.
**Aucun format de sortie bitmap/texture n'est documenté** dans ce qui a été lu.

Le fait le plus solide n'est pas déclaratif mais D'USAGE : ce dépôt (`internal/himap/rtgo.go`)
utilise déjà des offsets tirés de `Reclaimer/RenderModelTag.cs` pour décoder des `mode` Halo
Infinite en production — la question « Reclaimer connaît-il le format render_model d'Infinite »
est donc déjà répondue empiriquement, indépendamment de ce que documente son wiki.

Reclaimer est une **application graphique (.NET/WPF)**, pas un outil en ligne de commande :
l'utiliser pour extraire un véhicule précis suppose une session interactive (parcourir l'arbre
de tags, sélectionner le bon `vehi`, exporter) qu'un agent en avant-plan sans pilotage GUI ne
peut pas conduire de façon fiable. Ce n'est pas un verdict de faisabilité négatif — juste un
constat que cette voie demande une main humaine à la manette, contrairement à la voie maison
(§2) qui est scriptable de bout en bout.

### 3.2 Rips 3D communautaires (Sketchfab)

Des modèles existent, au moins pour le Warthog : `sketchfab.com/3d-models/halo-warthog-
26ae487ff8b44d02b3583f9bfa92dddc` (Julian Rijken) — confirmé **CC-BY** (réutilisable avec
attribution), fan-made (pas extrait du jeu), 19,8k triangles, publié 2020
(`WebFetch` du 2026-08-31). D'autres résultats Sketchfab (pinto36, Hakuru15, TAC3D) ne sont pas
vérifiés individuellement — licences potentiellement hétérogènes (Sketchfab autorise aussi des
licences « Standard » non réutilisables), et surtout **la plupart des modèles trouvés sont
génériques ou d'un autre jeu Halo** (Reach, 3), pas garantis fidèles au design Infinite.

**Cette piste ne dispense pas d'un moteur de rendu** : un modèle CC-BY téléchargé reste un
maillage à projeter en vue de dessus, et Blender n'est pas installé (§2.5). Elle ne réduit donc
le coût que sur UN maillon (la source du maillage), pas sur l'outillage de rendu, et ajoute une
dépendance de provenance/attribution externe à gérer véhicule par véhicule — quand la voie
maison (§2) reste 100 % sous contrôle du projet et déjà partiellement écrite.

### 3.3 Minimap officielle / UI de diffusion (HCS)

Recherches infructueuses (`WebSearch` du 2026-08-31, requêtes « Halo Infinite vehicle top-down
icon minimap HCS observer UI » et « Forge canvas overview top down vehicle icon ») : aucune
UI officielle avec icônes véhicules vues de dessus n'a été identifiée. Le seul résultat
concret sur Forge mentionne une icône de « Vehicle » générique dans le système de marqueurs de
navigation Forge — pas une silhouette par véhicule.

### 3.4 Blueprints / art officiel

Recherche (`WebSearch` du 2026-08-31, « Halo Warthog blueprint schematic top view orthographic
official art book ») : pas de blueprint officiel publié par 343/Bungie identifié. Seulement du
fan-art (forum 405th, print Etsy-like Artysnstudio) — ni gratuit, ni garanti géométriquement
exact, ni au format teintable requis.

### 3.5 Verdict Q3

Dépannage ponctuel envisageable (un véhicule bloquant sur la voie §2), jamais voie de masse :
couverture incomplète pour la liste Infinite exacte, licences hétérogènes à auditer un par un,
et ne résout pas l'absence d'outil de rendu local. **Décision utilisateur si retenue** — comme
demandé, aucun asset communautaire n'a été téléchargé ni intégré dans ce scout.

---

## 4. Piste dessin manuel (SVG depuis captures)

Chiffrée en la pratiquant : `4_warthog_croquis_manuel_dessus.png` (voir §5.2) a été dessiné
pendant ce scout avec des primitives géométriques simples (rectangles arrondis, ellipse,
lignes — `System.Drawing.Drawing2D.GraphicsPath` via PowerShell, aucun outil de dessin
dédié), en quelques minutes. **Le résultat est volontairement brut** — un lecteur qui ne
connaît pas la légende n'identifierait pas spontanément « Warthog » ; il montre le mécanisme
(silhouette blanche + alpha, teintable), pas un sprite présentable.

Estimation qualitative pour un rendu réellement « un peu détaillé » (le standard demandé),
par un dessinateur vectoriel compétent (Illustrator/Figma/Inkscape), par véhicule : de l'ordre
de 30 à 90 minutes selon la complexité de la silhouette (Warthog/Mongoose plus simples que
Scorpion/Wraith/Banshee), plus un temps de calibration de style amorti sur l'ensemble. Pour
la liste demandée (10 véhicules + variantes courantes + tourelles fixes détachables, de l'ordre
de 12-15 sprites), cela représente qualitativement une demi-journée à une journée de travail.
**Ce chiffrage n'a pas été mesuré sur un dessinateur réel** — c'est un ordre de grandeur, pas
une mesure, à traiter avec la même prudence que n'importe quelle estimation non chronométrée.

Avantage réel de cette voie : contrôle total du style et de la cohérence entre véhicules, zéro
dépendance de licence. Inconvénient : coût humain le plus élevé des quatre pistes, et fidélité
à la silhouette réelle du jeu dépendante du talent du dessinateur — risque illustré par mon
propre essai, volontairement basique.

---

## 5. Recommandation et preuve de faisabilité

### 5.1 Voie retenue

**Piste 2 (§2) : extraction du modèle 3D réel depuis les `.module`, rendu top-down par le
rasterizer maison déjà écrit.** Elle l'emporte sur les trois autres parce qu'elle réutilise le
plus de code déjà écrit, testé et déployé dans CE dépôt, pour un usage voisin :

| Brique | État | Fichier |
|---|---|---|
| Décodage `.module` + Kraken | Fait, en prod | `internal/ooz`, `internal/himodule` |
| Triangles/sommets d'un `mode` (render_model) | Fait, en prod (objets Forge) | `internal/himap/geometry.go` (`NewRenderModelAsset`) |
| Chaînage objet → `hlmt` → géométrie | Fait pour `bloc`/`scen`/`mach` | `internal/himap/cuisson_forge.go` |
| Chaînage `vehi` → `hlmt` | **À écrire** (même patron) | — |
| Rasterizer top-down Lambert | Fait, en prod, qualité prouvée | `internal/himap/rendu.go` + `catalyst.png` |
| Adaptation « objet isolé, repère local » | **À écrire** | — |
| Teinte par couleur d'équipe (canvas web) | **Déjà fait, déjà en prod** | `apps/web/src/features/match-replay/replayDraw.ts` (`tintedIconCanvas`) |
| Résolution de la couleur d'équipe | **Déjà fait, déjà en prod** | `apps/web/src/features/match-view/teamColor.ts` |

Filet de secours par véhicule bloquant : piste 4 (dessin manuel) ou piste 3 (modèle CC vérifié
individuellement) — jamais comme voie de masse.

### 5.2 Preuve de faisabilité déposée

`.ai/V7.5/film_re/scout_sprites_preuve/` (worktree courant) — 7 fichiers PNG, produits par un
script `System.Drawing`/PowerShell (aucun Go, aucun Python) pendant ce scout :

| Fichier | Contenu | Ce qu'il prouve | Ce qu'il NE prouve PAS |
|---|---|---|---|
| `0_planche_contact.png` | Vue d'ensemble des 6 images ci-dessous | Comparaison rapide | — (voir individuels ci-dessous, qui font foi) |
| `1_warthog_profil_extrait_du_jeu.png` | Copie telle quelle de `killfeed-26.png` (le jeu) | Format réel du jeu : silhouette blanche + alpha | L'angle de vue (profil, pas dessus) |
| `2_warthog_profil_teinte_rouge.png` | Fichier 1, recoloré RGB(214,40,40), alpha préservé | La teinte fonctionne sur un asset RÉEL du jeu | — |
| `3_warthog_profil_teinte_bleu.png` | Fichier 1, recoloré RGB(40,120,220) | Idem, deuxième couleur | — |
| `4_warthog_croquis_manuel_dessus.png` | Dessin manuel schématique, vue de dessus, silhouette blanche + alpha | Le format cible est atteignable à la main (voie §4), coût illustré | La fidélité visuelle (volontairement brut) ; n'est PAS un extrait du jeu |
| `5_warthog_croquis_manuel_dessus_teinte_rouge.png` | Fichier 4, recoloré rouge | La teinte fonctionne indépendamment de la source (extraction OU dessin) | — |
| `6_warthog_croquis_manuel_dessus_teinte_bleu.png` | Fichier 4, recoloré bleu | Idem | — |

La teinte utilisée dans le script reproduit exactement la logique de `tintedIconCanvas`
(`replayDraw.ts:379-384` : composition `source-in`, remplissage plein de la couleur cible,
alpha inchangé) — donc ce que montrent les fichiers 2/3/5/6 est bien ce que produirait le
pipeline web réel, pas une approximation.

**Ce qui manque à cette preuve, assumé en §2.6** : aucun de ces 7 fichiers n'est un rendu
top-down d'un modèle 3D réellement extrait — cette preuve-là demande du code Go compilé, hors
périmètre de ce scout. La preuve de la TECHNIQUE de rendu (§2.3, `catalyst.png`) et la preuve
du MÉCANISME de teinte (ce paragraphe) sont produites séparément ; leur assemblage
(rendu top-down réel + teinte) est le premier livrable attendu du lot d'exécution.

### 5.3 Spec du format cible

- **Format** : PNG RGBA, silhouette blanche (ou gris neutre) pleine + canal alpha porteur du
  dessin — identique à la convention déjà en place pour
  `static/weapons-assets/halo_infinite/jeu/{contour,silhouette,killfeed}-*.png` et
  `static/abilities-assets/halo_infinite/*.png`. Aucune nouvelle convention à inventer.
- **Consommation** : `tintedIconCanvas(img, color, { tinted: true })` — déjà écrit, déjà en
  prod. `color` vient de `teamColorResolver`/`resolveTeamColorFromID` — déjà écrit, déjà en
  prod pour le scoreboard et le rejeu.
- **Résolution** : export maître ~128-256 px de long (le renderer produit du vecteur/triangle,
  donc n'importe quelle résolution est atteignable sans perte), affiché downscalé à 24-64 px
  comme demandé par le besoin produit.
- **Périmètre d'assets** : un sprite par véhicule pilotable (Warthog, Razorback si confirmé
  distinct, Mongoose, Gungoose, Ghost, Banshee, Wasp, Scorpion, Wraith, Chopper) + variantes
  visuellement distinctes (Rockethog) + tourelles fixes détachables déjà nommées côté armement
  (`.ai/V7.5/film_re/VEHICULES_ARCHETYPE_40.md` : distinction tourelle/fixe déjà établie par la
  règle R-VÉHICULE) — au total de l'ordre de 12-15 sprites.

---

## Limites de ce scout

- Aucun rendu top-down d'un véhicule réel n'a été produit (interdiction de compiler du Go —
  voir §2.6). C'est la vérification la plus importante qui reste à faire, en premier, dans le
  lot d'exécution.
- La recherche communautaire (§3) n'a pas audité individuellement chaque modèle Sketchfab
  trouvé ni cherché au-delà du Warthog pour les licences CC.
- `inf_vectorart` (§1.3) n'a pas été cloné ni exécuté — seul son README a été lu.
- Le tag `vehi` n'a pas été ouvert sur pièce (aucun lecteur Go existant à appeler sans
  compiler) : l'hypothèse « `vehi` porte une référence `hlmt` comme `bloc`/`scen`/`mach` » est
  une inférence par analogie de format de tags Halo, pas une mesure directe sur les octets.
- Licence Reclaimer non confirmée (pages consultées silencieuses sur ce point) — à vérifier
  avant tout usage direct de son code ou de ses binaires, même si ce scout ne recommande pas
  cette voie.

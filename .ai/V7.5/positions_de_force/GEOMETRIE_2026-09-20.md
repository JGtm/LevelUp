# Géométrie des positions de force — inventaire, prototype, réglage figé (2026-09-20)

> Item 2bis.C du plan `.ai/PLAN_POSITIONS_DE_FORCE_2026-09-20.md`. Branche `wt/power-positions`.
> Code : paquet pur `apps/go-api/internal/analysis/powerpos/geo/` (14 tests) + outil hors ligne
> `apps/go-api/cmd/mapgeo-build/` (+ 1 test `gamefiles`). Sorties :
> `.ai/V7.5/positions_de_force/geometrie_2026-09-20/` (CSV par nœud, 8 planches PNG par carte,
> `positions_geo.json`, `_rapport.md`, journaux). Rien n'est écrit en base ni sous `data/`.
> Le réglage (section 6) a été figé sur Recharge / Aquarius / Streets SANS regarder l'oracle ;
> le verdict revient à 2bis.D (commande en section 9 de `VERDICT_V2_2026-09-20.md`).

## 1. Inventaire des sources d'occlusion (vérifié sur pièces)

| Source | Cartes natives | Ce qu'elle donne | Verdict |
|---|---|---|---|
| Maillage de navigation (`internal/hinavmesh`, `navmesh.blob`) | **AUCUNE**. Le blob n'est publié qu'à côté de la variante UGC d'une carte Forge (`cmd/mapnav-fetch` lit la page `__NEXT_DATA__` de l'asset ; « les cartes natives n'en ont pas », `NAVMESH_FORGE_2026-08-27.md` l. 40). Le dépôt hors ligne `.ai/re_dump/navmesh` n'existe même pas sur ce poste (gitignoré, jamais rapatrié ici). | polygones convexes du sol, quand il existe | inutilisable pour Recharge et les cinq autres cartes de l'oracle |
| `reference/map_structure/{module}.json` | 2 modules seulement (`ridgeline`, `sgh_streets`) | boîtes AABB des instances de bsp (Streets : 10 908 boîtes, aucune forme — « l'AABB d'un anneau est un carré plein », handoff) | inutilisable pour des rayons |
| Triangles de rendu des `.module` (`internal/himap` : sbsp → instances → tags `rtgo` → déquantification u16) | toutes les cartes installées ; Live Fire via `common-rtx-new.module` (redirection `moduleGeometrie` de `map_fond_reglages.json`, lue telle quelle) | des millions de triangles monde par carte (table ci-dessous) | **retenue**, avec deux écarts au producteur de fonds : le filtre « décor grossier » n'est pas appliqué (une dalle grossière arrête les balles ; comptée : 239 instances sur Recharge), et le bornage à la boîte de l'instance écarte le triangle entier au lieu de le rogner |
| Coquille de mort `sddt` (variante `any/`, `himap.LitSddt`, parité de rayon) | 5 cartes sur 6 ; Live Fire : 0 plan dans `sgh_interlock` (non appliquée, journalisé) | le volume de survie ; règle du fond : appliquée seulement si elle garde toutes les ancres | **retenue** pour borner les sols (7 000 à 10 900 candidats écartés par carte) |
| Bloc `instanced physics instances` du sbsp (collision, handoff §9.2 : 1,4 Mo sur catalyst) | présent | LA géométrie de collision | **non décodée dans le dépôt** — c'est la limite structurelle de tout ce qui suit (section 5) |

Triangles extraits dans le cadre (ancres ± 15 m, tranche `TrancheDeJeu` resserrée à [z jeu − 8 ; + 14]) :

| carte | module | instances du bsp | lues | triangles retenus | emprise z des triangles |
|---|---|---:|---:|---:|---|
| recharge | `sgh_blueprint` | 5 164 | 2 539 | 3 341 105 | −45,1 .. 15,4 (boîtes touchant la tranche) |
| aquarius | `ctf_aquarius` | — | — | 18 111 247 | — |
| streets | `sgh_streets` | — | — | 9 685 871 | — |
| live fire | `sgh_interlock` → `common-rtx-new` | — | — | 7 248 584 | — |
| forbidden | `ctf_forbidden` | — | — | 26 316 119 | — |
| bazaar | `ctf_bazaar` | — | — | 8 244 699 | — |

**Recharge, sol praticable lisible ?** Oui, mais DÉRIVÉ : 23 045 candidats de sol (surfaces
montantes au centre des cellules de 0,5 m), dont 9 645 hors de la coquille de mort, 4 541
« sous une dalle » (face inférieure d'un bloc), 3 651 sans hauteur libre, 1 119 inatteignables
depuis une ancre ou un socle, 1 188 sans retour vers un objectif → **2 901 nœuds**, 25/25 ancres
et 10/10 socles posés, une composante principale de 4 006 nœuds avant élagage (les quatre autres
font 37, 25, 20, 1). Murs et plafonds sont exploitables pour des rayons 2,5D : 263 979 voxels
occupés (0,25 m) sur 13 431 cellules × 88 couches.

## 2. Ce que le prototype calcule (paquet `geo`, pur)

1. **Voxelisation** des triangles (test exact triangle / boîte d'Akenine-Möller) dans une grille
   de 0,25 m en XY (plus fine que la cellule tactique de 0,5 m, sinon un mur qui effleure un coin
   de cellule la condamne) et 0,25 m en Z, sur le cadre aligné sur `tactical.Grille` (même
   ancrage monde que la voie empirique : une cellule géométrique et une cellule empirique de la
   même carte sont la même cellule).
2. **Sol praticable** : candidats = surfaces dont la normale fait moins de 50° avec la verticale,
   à la verticale du centre de cellule, dédoublonnées à 0,25 m ; rejet « sous dalle » si une
   surface (toute orientation) passe entre z + 0,25 et z + 0,75 au même point (les voxels ne
   suffisent pas : la contremarche d'un escalier occupe la même sous-colonne) ; rejet « hauteur
   libre » si la sous-colonne du centre est occupée entre z + 0,75 (zone de marche) et
   z + 1,5 ; rejet « hors arène » par la coquille de mort à hauteur d'yeux. Une cellule peut
   porter plusieurs nœuds (Attic sur Batteries).
3. **Graphe de déplacement ORIENTÉ** : arcs vers les nœuds à ≤ 2 cellules (un Spartan enjambe
   une cellule manquante) si la montée ≤ 1,6 m (saut + clamber, surcoût = dénivelé) ou la
   descente ≤ 5 m (gratuite) ; rayon de passage à hauteur de poitrine (1,2 m) au niveau du plus
   haut, sinon à hauteur de saut (1,9 m : muret franchi, surcoût) ; pour un saut ou une chute,
   verticale libre au-dessus du nœud bas (on ne tombe pas à travers un plancher). Atteignabilité
   depuis les GERMES (ancres d'objectif + socles) par arcs sortants ; puis élagage des nœuds d'où
   AUCUN objectif n'est atteignable (arcs entrants).
4. **H** = z − moyenne des z des nœuds à ≤ 6 m en XY. **V** = part des cibles (un nœud toutes
   les 2 cellules, tous niveaux) visibles depuis les yeux (1,7 m) par traversée de voxels
   (Amanatides-Woo), portée 60 m, en parallèle. **E** = part des 36 secteurs de 10° d'où une
   cible visible regarde le nœud. **R** = 1 − min(d, 20 m)/20 m où d = distance de déplacement
   (Dijkstra sur les arcs entrants) au plus proche entre arme forte / powerup et objectif.
   **M** = 1 − d/12 m, d = distance de déplacement au premier nœud que la majorité des cibles
   qui voient le nœud ne voient plus (Dijkstra borné, intersection de tableaux de bits).
5. **Score** = w1·H + w2·V − w3·E + w4·R + w5·M sur variables normalisées PAR CARTE par min-max
   robuste (p5..p95 → [0, 1] ; un rang aurait étalé une variable plate). **Sélection** : maxima
   locaux (domine à 3 m de marche) au-dessus du p90 de la carte, croissance bornée à 4 m de
   marche autour de chaque maximum (sans cette borne la halle de Recharge sortait en une
   position de 128 m²), ≥ 12 nœuds, ≤ 8 positions, polygone = `powerpos.Enveloppe` (même forme
   que la v1), z min / z max publiés.

## 3. Coût mesuré par carte (poste de développement, 20 cœurs, `go run`)

| carte | cellules | nœuds | cibles | rayons | voxelisation | sol | visibilité | couvert | total | mémoire système (Go, cumulée) | positions |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| recharge | 13 431 | 2 901 | 721 | 2,09 M | 2,1 s | 0,3 s | 0,03 s | 0,01 s | **3,4 s** | 974 Mo | 7 |
| aquarius | 15 618 | 3 266 | 817 | 2,67 M | 8,5 s | 1,7 s | 0,04 s | 0,01 s | **13,7 s** | 4,8 Go | 6 |
| streets | 13 310 | 2 975 | 773 | 2,30 M | 4,7 s | 1,0 s | 0,04 s | 0,01 s | **8,0 s** | 4,8 Go | 5 |
| live fire | 14 430 | 3 082 | 765 | 2,36 M | 3,8 s | 0,7 s | 0,03 s | 0,01 s | **6,3 s** | 4,8 Go | 7 |
| forbidden | 14 170 | 5 048 | 1 261 | 6,36 M | 14,0 s | 2,6 s | 0,1 s | 0,03 s | **23,3 s** | 6,6 Go | 8 |
| bazaar | 15 048 | 5 914 | 1 493 | 8,83 M | 4,4 s | 0,9 s | 0,1 s | 0,03 s | **7,4 s** | 6,6 Go | 8 |

Le coût est dans la VOXELISATION des triangles (60 à 80 %), pas dans les rayons : 8,8 millions
de rayons coûtent 0,1 s. Le sous-échantillonnage des cibles (1 m) n'est donc pas une contrainte
de coût ; il pourrait descendre à toutes les cellules. La mémoire est celle de `runtime.MemStats`
`Sys` en fin de carte, cumulée sur la passe (le tas n'est pas rendu au système entre deux
cartes) : elle vient des 3 à 26 millions de triangles retenus (72 o chacun) et des candidats de
sol, pas des tableaux de bits de visibilité (< 10 Mo). Les six cartes se cuisent en **62 s**.

## 4. Ce que H, V, E montrent sur Recharge (chiffres et planches)

- **H** : p5 −1,50 m, p50 0,00, p95 +1,59 m. La planche `recharge_H.png` colore en rouge le
  plateau intérieur de la halle des turbines (2,2 m, entouré par le niveau −3), l'étage 4,4 du
  centre et les balcons ; en bleu la fosse (0 m) et le couloir sous le pont. Un nœud
  d'Attic vaut +1,5 à +2 m contre Batteries.
- **V** : p50 0,10 (10 % des 721 cibles vues), p95 0,27. `recharge_V.png` : la halle et le
  centre ouvert voient le plus ; les couloirs latéraux, la fosse et les recoins presque rien.
- **E** : p50 0,50 (18 secteurs sur 36 exposés), p95 0,92. Corrélé à V à **0,82** (0,85 sur
  Aquarius, 0,89 sur Streets) : sous des rayons symétriques, ce qu'on voit est ce qui nous
  voit ; seule la DIFFÉRENCE des deux agrégations discrimine (un lieu qui voit beaucoup et n'est
  vu que d'un côté). C'est ce qui a décidé des poids (section 6).
- **M** est anti-corrélé à V et E (−0,50 / −0,60) : un couvert proche, c'est moins de vue. **R**
  est indépendant de H sur les trois cartes de calibrage (|r| ≤ 0,14). H est la variable la
  plus indépendante des quatre autres (|r| ≤ 0,33).
- **Positions Recharge (réglage figé)** : 7, de 6,8 à 30,5 m² — (23,0 ; −2,8) à z 3,2 (H 0,98,
  R 0,98), (19,5 ; −10,6) à z 4,4, (22,6 ; 4,4) à z 2,2 (30,5 m², M 0,96), (2,6 ; 2,9) à z 4,4,
  deux positions côté ouest à z 4,4–4,8, (12,1 ; −6,9) à z 2,2–3,5 (V 0,87). Toutes en hauteur
  (H normalisé 0,71 à 0,99) : la géométrie retient les étages, la voie empirique retenait les
  bases (`VERDICT_ORACLE`). `recharge_score.png` les trace en vert sur le score par nœud.
- La planche `_composantes` a été l'instrument de tout le chantier du sol : c'est elle qui a
  montré, en dix couleurs, que la marche seule coupait la carte le long de ses fosses.

## 5. Limites mesurées (à lire avant de juger)

1. **Le maillage de rendu n'est pas le maillage de collision.** Preuve : les deux artefacts de
   rejeu de Recharge (retrouvés par leurs bornes, `3923bede`, `e85d7bad`) portent des joueurs à
   z 0,5 / 1,0 / 1,5 m dans la fenêtre x 13–16,5, y 0,5–3,5 — ils montent l'escalier de la
   fosse — là où des corniches de rendu à 1,6–2,0 m au-dessus des marches rejetaient les
   candidats à 1,9 m de hauteur libre. D'où `HauteurLibreM = 1,5` (un Spartan accroupi passe).
   Le bloc de collision du sbsp est la source juste ; il n'est pas décodé.
2. **Un piège subsiste sur Recharge** : le niveau bas de la halle des turbines (z = −3,0,
   1 147 nœuds) se rejoint en tombant et ne se quitte pas dans le graphe (l'escalier de sortie
   n'est pas reconstruit) ; l'élagage « sans retour » l'écarte. Les joueurs y vont (artefact
   `e85d7bad`, z min −2,98).
3. **Cibles hors arène** : sur Streets, Forbidden et Bazaar, 7 969 / 7 643 / 11 305 nœuds
   (toits, terrasses, décor atteint par une chaîne de sauts sur du décor) étaient atteints depuis
   les germes sans retour possible ; avant l'élagage ils écrasaient la normalisation (V médian
   0,66 sur Bazaar : la moitié des nœuds voyaient tout, depuis dehors).
4. **Live Fire** : coquille de mort à 0 plan dans `sgh_interlock` (variante `any/`) — arène non
   bornée, 4 932 candidats rejetés par la hauteur libre seulement ; ses chiffres restent sous la
   réserve de la découverte 1 du plan.
5. **Occlusion surestimée** : un garde-corps ajouré bloque comme un mur ; V et E sont
   sous-estimés de façon homogène. Les rayons sont à hauteur d'yeux vers hauteur d'yeux ; le
   corps n'est pas testé.
6. **Distances** : le saut coûte son dénivelé, la chute rien, le grappin n'existe pas.

## 6. Réglage géométrique figé — 2026-09-20 (`geo.ReglageGeoV1`)

Figé sur les distributions de **Recharge, Aquarius et Streets** (sections 3-4 et `_rapport.md`),
sans regarder l'oracle, avant tout verdict :

```
SCORE = 0,30·H + 0,20·V − 0,15·E + 0,20·R + 0,15·M        (variables normalisées p5..p95 par carte)
R : portée 20 m ; M : portée 12 m (paramètre physique)
Seuil : p90 du score de la carte ; maximum local dominant à 3 m de marche ;
croissance à 4 m de marche autour du maximum ; ≥ 12 nœuds ; ≤ 8 positions par carte.
```

Pourquoi ces poids (une ligne chacun, détail dans `score.go`) : H est la plus indépendante et
la première citée par les guides → 0,30 ; V et E portent presque la même information
(r 0,82–0,89) → leur différence pondérée doit rester petite, V un peu au-dessus de E pour qu'un
lieu qui voit beaucoup et n'est vu que d'un côté ressorte sans que l'ouverture brute l'emporte ;
R indépendant de H et second dans les guides → 0,20 ; M anti-corrélé à V/E → 0,15, il équilibre
la paire. Paramètres physiques (`geo.ParametresParDefaut`) : voxels 0,25 m, yeux 1,7 m, hauteur
libre 1,5 m, marche 0,75 m, saut 1,6 m, chute 5 m, passage 1,2 m / saut 1,9 m, pente 50°,
rayon H 6 m, cibles toutes les 2 cellules, portée de vue 60 m, 36 secteurs, couvert 12 m.

Aucun de ces nombres ne sera retouché après le verdict de 2bis.D : une retouche = nouvelle
mesure, nouveau document.

## 7. Cartes cuites et fichiers

Six cartes, 62 s : calibrage Recharge / Aquarius / Streets ; validation Live Fire / Forbidden /
Bazaar. Par carte : `<carte>.csv` (une ligne par nœud : cellule, niveau, composante, x y z,
H V E bruts, nb visibles, distances arme forte / arme / objectif / couvert, R et M proximités,
les cinq normalisés, score), `<carte>_rejets.csv` (candidats écartés et motif), `<carte>_coutures.csv`
(paires de composantes voisines coupées par un rayon, voxel qui bloque), 8 planches
`<carte>_{sol,composantes,H,V,E,R,M,score}.png` (fond publié + nœuds à leur emprise, hauts
par-dessus bas, zones nommées en contour, positions retenues en vert sur `_score`).
`positions_geo.json` (schéma 1, clés du `positions.json` empirique : `carte`, `axe`, `module`,
`map_id_dominant`, `map_ids`, positions avec `polygone`, `score_moyen`, `aire_m2`, `centre_x/y`,
plus `z_min`, `z_max`, `variables` normalisées) est l'entrée du harnais de 2bis.D.

Commande (depuis `apps/go-api`, jeu installé, données de la racine principale) :

```
CGO_ENABLED=1 go run ./cmd/mapgeo-build --data-root <racine contenant data/> \
  --cartes "recharge,aquarius,streets,live fire,forbidden,bazaar" \
  --sortie ../../.ai/V7.5/positions_de_force/geometrie_2026-09-20
```

## 8. Ce qui a été essayé et écarté (dans l'ordre, tout mesuré sur Recharge)

| essai | mesure | sort |
|---|---|---|
| hauteur libre testée sur toute la colonne de 0,5 m | 2 385 nœuds, 10 composantes, 1 971 sans chemin | remplacé par la sous-colonne de 0,25 m du centre |
| graphe de marche seule (≤ 0,75 m, 8 voisins) | 10 composantes séparées par les bords de fosse (planche `_composantes`) | remplacé par le graphe orienté saut / chute |
| marche mise à l'échelle sur la diagonale (1,06 m) | le dessus d'une caisse de 1 m se rattache | marche fixe |
| « sous dalle » lu dans les voxels (couche +1 occupée) | tue les escaliers (contremarche dans la sous-colonne) | lu sur les surfaces exactes au centre |
| rayon de passage à 1,2 m seul | murets à hauteur de poitrine pris pour des murs (63 puis 614 arcs « obstacle ») | second rayon à 1,9 m, surcoût |
| hauteur libre 1,9 m | passage sous le pont à 4,0 m mort à 2,05 m par arrondi de couche ; escalier de la fosse rejeté sous des corniches de rendu | couches entièrement comprises + 1,5 m |
| voisinage à 1 cellule | 1 174 nœuds sans remontée (une marche manquante coupe la montée) | 2 cellules |
| positions = composantes connexes au-dessus du seuil | halle de Recharge en une position de 128 m² | croissance bornée à 4 m autour du maximum |
| pas d'élagage « sans retour » | 74 % des nœuds de Streets hors jeu, V médian 0,66 sur Bazaar | élagage |

## 9. Découvertes hors périmètre (reportées au plan)

Voir la section « Découvertes » du plan, entrées 17 à 21 : bloc de collision non décodé, dépôt
`.ai/re_dump/navmesh` absent, `sddt` de Live Fire vide, `map_positions_jouees.json` limité aux
cartes Forge (aucune position jouée native pour servir d'oracle du sol), peintre PNG en
deuxième copie.

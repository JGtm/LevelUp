# Handoff — dessin de la carte par les TRIANGLES

> Écrit le 2026-07-26, à l'arrêt volontaire du chantier. **La géométrie est au-delà du besoin de
> la V1** : le fond de carte est produit, transparent, borné, et reconnu à l'œil par l'utilisateur
> (fer à cheval en anneau, zone sud reliée par deux ponts — ses deux critères d'acceptation).
> Ce document existe pour qu'une reprise éventuelle ne recoûte pas une journée.
>
> **Ce qui reste à faire tient en une ligne** : porter la recette en Go pour cuire l'image des
> **29 autres cartes**. Aujourd'hui elle n'existe qu'en Python jetable, sur ridgeline seule.

---

## 1. LA CHAÎNE, DE BOUT EN BOUT

```
.module (pc/levels/multi/<carte>/<carte>-rtx-new.module)
  -> tag sbsp : bloc `instanced geometry instances` a l'offset 420 du tag
       record 320 o (0x140) : TransformScale@0 · Matrix4x4 Transform@12
                              RuntimeGeoMeshReference@60 · MeshIndex@116 · BoundsIndex@118
  -> tag rtgo designe par RuntimeGeoMeshReference
       PerMeshData@16 (record 144 o) · Sections@64 · BoundingBoxes@104
       TotalVertexBufferCount@190 · MeshResourceGroups@196
  -> champ racine `meshes` : le TagBlock enfant a foff = meshIndex*60
       -> records de « LOD render data » de 148 octets
            u16 @0x64 = index du tampon de SOMMETS
            u16 @0x8A = index du tampon d'INDICES
  -> deux tables de descripteurs : 0x50 pour les sommets, 0x48 pour les indices
       `off` est un OFFSET D'OCTETS dans la CONCATENATION des entrees-ressource du tag
       (ce n'est PAS un identifiant a resoudre contre le manifeste)
  -> sommets : u16 x4, la 4e composante nulle
  -> dequantification, puis placement par la transformation de l'instance
```

**Couverture obtenue** : 1 247/1 247 couples (globalID, meshIndex) reliés · 29 683 descripteurs,
**0 hors bornes** · 10 357 instances de bsp=0, **28,88 M triangles, 0 non résolue**.

**Les offsets de tag sont confirmés par une implémentation tierce** : `Gravemind2401/Reclaimer`,
`Reclaimer.Blam/Blam/HaloInfinite/{RuntimeGeoTag,ScenarioStructureBspTag,ModuleItem}.cs`. C# ,
exporte des modèles ouverts dans Blender, donc validé par l'usage. Vérification sur nos tags :
le bloc `PerMeshData` mesure 864 o = **6,00 × 144** (et 1 296 = 9,00 × 144, 720 = 5,00 × 144) —
multiple entier parfait, l'offset 16 et le pas de 144 sont établis sans ambiguïté.

---

## 2. TROIS CORRECTIONS À NE PAS PERDRE

### 2.1 Les sommets sont en `u16` BRUT, pas en `i16 + 32768`

Écart aux bornes : **5,8 mm** contre **84 mm** avec la mauvaise lecture. Les scripts
`scratchpad/py/geo2.py` (ligne 143) et `scratchpad/py/s26_gold.py` (fonction `dequant`)
appliquent la mauvaise formule. **Tous les résultats géométriques produits avant le 2026-07-26
sont entachés d'une erreur médiane de 8,4 cm** (`py/raster.npz`, `py/analysis.npz`,
`py/floors.npz`, `out/*.npz`) et doivent être régénérés avant réutilisation.

### 2.2 Le chaînage ne passe PAS par `@0x88`/`@0x8c`

Ce champ est **réfuté** : 0,0 % de résolution valide. C'est un hash 64 bits par maillage. Le vrai
chemin est celui du §1, via le champ racine `meshes` et le bloc de LOD render data.

### 2.3 Le « critère de validation en or » par l'AABB est TAUTOLOGIQUE — à retirer

`.ai/V7.5/cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md` et mes propres consignes de la journée présentaient
comme critère décisif : « les bornes locales transformées doivent reproduire l'AABB monde connu
de l'instance ». **C'est vrai par construction.** Mesure : l'AABB monde de l'instance EST
exactement la boîte `bmin`/`bmax` transformée (écart médian **0,0000 m**), et le critère ne
départage pas un bon tampon d'un tampon tiré au hasard (0,19 m contre 0,22 m).

**Il a probablement masqué des chaînages faux dans les itérations passées.** Le remplacer par :

> **T1** — l'indice maximal du tampon d'INDICES doit être strictement inférieur au nombre de
> sommets du tampon apparié. Rend **100 %** pour le bon appariement et **5,1 %** pour un LOD
> voisin.

---

## 3. CE QUE LE RÉSULTAT VAUT, ET CE QU'IL NE VAUT PAS

| mesure | valeur | référence |
|---|---|---|
| vide dans la zone du fer à cheval | **12,9 m²** | 0,00 m² en boîtes alignées ET orientées |
| désertion du disque central, reconstruite | **×63,8** | ×64 mesurée sur les seules trajectoires |
| rang du centre réel | **99,7 %** | sur 5 862 disques tirés au hasard dans la zone |
| centroïde du vide | à **0,39 m** du centre attesté | — |
| positions de joueur à moins de 25 cm du sol | 82,0 % | 22,8 % attendus par hasard |

**La passe de réfutation n'a jamais tourné** (limite de session). Deux points restent donc
ouverts, et il ne faut pas les oublier en reprenant :

1. Le témoin de non-régression est donné à **82,0 % sous 25 cm**, quand le fond en boîtes était
   validé à **80,6 % sous 5 cm**. Seuil cinq fois plus large : **les deux nombres ne se
   comparent pas.** Refaire la mesure au même seuil avant de conclure que rien n'a régressé.
2. Les bornes de déquantification. Reclaimer les place dans un bloc `BoundingBoxes` **unique par
   tag** (84 octets, trois paires parfaitement symétriques : ±2,6357 en X, ±1,7432 en Y,
   ±3,0514 en Z), désigné par `BoundsIndex` (+118 de l'instance). Cette reconstruction lit des
   bornes **par maillage**. Deux lectures produisent des chiffres, une seule est juste.

### Ce que le donut est vraiment

**Le trou n'est PAS dans le plancher.** Sous le disque de rayon 1 m, la surface projetée des
triangles est couverte à **100,0 %**. L'obstacle est un **BLOC PLEIN** de z = −0,8 à +2,0 m, et
le sol de l'étage inférieur (z = −2,46 m, celui où circulent les joueurs) s'arrête net à son
aplomb. Ce que l'utilisateur voit est juste ; sa cause est un volume, pas une absence.

---

## 4. PORTES DÉFINITIVEMENT FERMÉES

| piste | pourquoi, avec le chiffre |
|---|---|
| le compagnon `.module_hd1` | **Preuve d'absence, pas absence de preuve.** Base forcée arithmétiquement : les `dataOffset` forment un espace d'adressage GLOBAL CONTINU entre module et compagnon — tant que `offset < hd1Delta` la donnée est dans le principal, au-delà elle est dans le compagnon à `offset − hd1Delta`. Trois égalités exactes le prouvent AVANT toute décompression : `min(offset)` des 59 entrées hd1 = `hd1Delta` au bit près · `max(offset+compSize)` des 4 945 entrées non-hd1 tombe juste EN DESSOUS · `max(offset+compSize) − hd1Delta` des entrées hd1 = **la taille exacte du fichier compagnon, à l'octet**. Extraction : 59/59 entrées, 249/249 blocs, 286,0 Mio. Contenu : **59 tags `bitm`**, un chacun (BC1 2048², BC7 2048²/4096²). Les **1 784 ressources `rtgo`** sont TOUTES dans le module principal, **zéro** déportée. |
| le test « int16×4 à 4e composante nulle » comme détecteur de sommets | **TAUTOLOGIQUE sur ce corpus** : un bloc DXT1 plat (`c0 == c1`, indices nuls) rend `w == 0`. Une entrée atteint **0,915** de `w == 0` alors que c'est une TEXTURE. Ce test aurait fait conclure « sommets » à tort. |
| le test de SATURATION de la plage comme critère de maillage normalisé | **TAUTOLOGIQUE** : les vrais tampons ont un span médian de **0,888**, et des plages tirées au hasard dans le même blob ont **0,890**. |
| la collision `scgt` lue en `float32` brut | **FAUX POSITIF** : 53 % de triplets « dans les bornes de la carte » contre 14 % au hasard, mais **95 à 97 % des points sont à moins d'un mètre de l'origine** — le nuage est une croix, signature d'octets quelconques lus comme des flottants. **Toujours DESSINER un résultat, jamais seulement le compter.** |
| les boîtes, alignées comme orientées | **0,00 m² de vide** dans le fer à cheval, dans les deux cas. Les neuf instances qui en bouchent le centre sont à yaw nul, `|up.z| = 1,000` et échelle unité : leur boîte orientée est IDENTIQUE à leur boîte alignée. L'anneau vit dans les triangles. |

---

## 5. LA COLLISION — non explorée, et probablement inutile désormais

Le bloc `instanced physics instances` du sbsp est **entièrement craqué** : 3 928 records de 0xC0,
layout validé champ par champ, base Forward/Left/Up orthonormée à **100,00 %** à l'offset 0x48
contre **0,00 %** à ±4 et ±8 octets. La forme est dans les tags de groupe **`scgt`**, dont
**366 sont déjà extraits** (`scratchpad/geo_any/res/*_scgt.bin`).

Le format interne d'un `scgt` **n'est pas craqué**. Réserve structurelle à connaître :
**195 des 552 modèles de collision vivent dans des modules GLOBAUX partagés**, pas dans celui de
la carte — toute chaîne « une carte = un module » est donc incomplète pour la collision.

Cette voie devient secondaire puisque les triangles de rendu suffisent.

---

## 6. OÙ SONT LES CHOSES

```
scratchpad/py/geo2.py          parseur rtgo complet (CORRIGER la dequantification, cf. 2.1)
scratchpad/py/himod.py         lecteur .module en Python + ooz par ctypes
scratchpad/ooz/ooz.dll         decompresseur Kraken autonome (GPLv3, hors ligne uniquement)
scratchpad/sc/world.npz        V (6,2 M sommets) + T (9,86 M triangles) de ridgeline
scratchpad/sc/raster.npz       champ d'altitude : marchable_zmax / marchable_sol, grille 5 cm
scratchpad/structure_*.png     les rendus neutres, fond transparent
scratchpad/geo/                dump du module pc  (instances.csv, manifest.csv, 1 592 tags)
scratchpad/geo_any/            dump du module any (366 scgt)
scratchpad/refs/*.cs           les lecteurs Reclaimer telecharges, pour reference
```

Côté dépôt, ce qui est déjà en place et à réutiliser :

```
internal/himodule/             lecteur .module — dataOffset 48 bits + drapeau UseHd1 PORTES
internal/himap/instances.go    instances, LocalToWorld (convention vecteur-ligne)
cmd/tmp_geodump/               dump de la matiere brute, filtre --groups
cmd/mapstruct-build/           production de l'asset fige — c'est LUI qu'il faut faire evoluer
internal/analysis/replay/structure.go   rendu de la surface cote document
```

---

## 7. LE TRAVAIL RESTANT, EN UNE PHRASE

Porter la recette du §1 en Go dans `cmd/mapstruct-build`, avec les trois corrections du §2, et
cuire le fond de carte des **29 autres cartes**. Les garde-fous à poser en tests de
non-régression sont mesurés et disponibles : **0 descripteur hors bornes**, et **T1 à 100 %**.

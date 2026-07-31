# Handoff — Reconstruction 2D des maps depuis les `.module` Halo Infinite

> Branche : `feat/filmdec-continuation` (worktree `filmdec-continuation`).
> Statut : **PARTIELLEMENT RÉSOLU** — voir l'avertissement ci-dessous avant de lire le reste.
> Objectif initial : reconstruire la carte d'une map en 2D (fond pour overlay des positions film).

> **SUITE DONNÉE (2026-07-26, soir)** : la chaîne complète jusqu'aux TRIANGLES a depuis été
> établie, et elle est consignée dans **`.ai/HANDOFF_GEOMETRIE_TRIANGLES.md`**. Ce document-ci
> reste utile pour le format `.module` et le contexte, mais **trois de ses affirmations sont
> corrigées là-bas** : le chaînage ne passe pas par `@0x88`, les sommets sont en `u16` brut et
> non `i16+32768`, et son « critère de validation en or » par l'AABB est **tautologique**.

## AVERTISSEMENT (2026-07-26) — « RÉSOLU » était trop fort

Ce document se déclarait **RÉSOLU** au titre du §9. Mesure faite le 2026-07-26 : la chaîne
`Per Mesh Data` du §9.3 **ne fonctionne pas sur ridgeline** (Cliffhanger).

    cmd/tmp_ridgemesh sur ridgeline-rtx-new.module :
      497 ressources runtime_geo, 2 369 maillages lus
      -> 1 908 REJETÉS (NaN ou bornes inversées), soit 80,5 %
      -> bbox monde globale X[-8,4e33 ; 1,8e31] — sans aucun sens
    alors que les vraies bornes de la carte sont X[-41,10 ; 72,11] Y[-56,61 ; 57,21].

C'est la **même dérive de version** que celle rencontrée sur `sbsp.xml` et contournée au §9.1 par le
matching par RANG — le contournement n'a jamais été appliqué à `rawg.xml`.

**Ce qui marche réellement aujourd'hui** : la voie **instance** (`internal/himap`, bloc
`instanced geometry instances` du sbsp, AABB monde @0x7C). Elle produit 10 223 emprises sur ridgeline,
100 % de couverture, validées à 8 mm de médiane sous les joueurs. Mais elle ne rend que des **boîtes**,
donc aucune forme : l'AABB d'un anneau est un carré plein.

**Portée exacte du §9** : établi sur **catalyst**, non reproduit sur ridgeline. Le lire comme une
recette générale conduit à l'erreur. Corollaire : le §9.5 (« reste pour affiner ») n'est pas un
raffinement optionnel, c'est le **chantier bloquant** — et il faut d'abord réparer le layout par carte.

---

## 0. TL;DR

La géométrie des maps **n'est pas dans les films** (le film ne réplique que le dynamique).
Elle est dans les **tag-files `.module` du jeu** (offline, on les a). Chaîne complète **PROUVÉE** :
`.module → décompression Kraken (ooz) → tag/resource → walker version-aware → bornes monde par
mesh → carte 2D`. Sur catalyst : 277 resources `runtime_geo`, 523 meshes, empreinte ~45×30 m
(structure centrale en diamant reconnaissable). **Voir §9 pour la résolution finale.**

Clé du déblocage : la géométrie inline du tag sbsp est VIDE (streamée) ; les bornes monde par
mesh vivent dans les resources `runtime_geo` (ucsh), champ `Per Mesh Data` (Position + Bounds).
Et le mismatch de version se contourne par **matching par RANG** (pas par offset absolu) entre
les champs `_40` du plugin et les TagBlocks de la struct-table.

---

## 1. Ce qui marche (livré, réutilisable)

### Packages
- **`internal/ooz`** — décodeur Oodle clean-room (ooz/Powzix, **GPLv3**, cf `NOTICE.md`),
  cgo. `ooz.Decompress(comp []byte, rawLen int) ([]byte, error)`. **Offline-only** (licence).
- **`internal/himodule`** — lecteur d'archive `.module` (mohd v53). API :
  - `Open(path) (*Module, error)`
  - `(*Module).Files(group string) []File` (group "" = toutes ; "sbsp", "mode"…)
  - `(*Module).Extract(File) ([]byte, error)` (assemble + décompresse ; gère tags ET resources)

### Outils throwaway (`cmd/tmp_*`)
| Outil | Rôle |
|---|---|
| `tmp_moduleread` | lit le container, recense les groupes de tags (mode scan-répertoire) |
| `tmp_modfiles` | liste toutes les entrées (tags + resources) par taille |
| `tmp_module_extract` | extrait+décompresse un tag sbsp (preuve décompression) |
| `tmp_resgeo` | extrait une resource, détecte les vertex buffers (float32/int16) |
| `tmp_sbsp_parse` | parse l'en-tête de tag + table des data-blocks |
| `tmp_meshbounds` | (tentative) localiser `compression info` + rendre les bbox |
| `tmp_mapcloud`, `tmp_mapdense` | exploration des positions FILM (écartées, cf §5) |

---

## 2. Format `.module` (mohd v53) — VÉRIFIÉ

Réfs : Kaitai matty45 (imprécis), `Krevil/InfiniteModuleReader`, hand-decode.

- **Header** : magic `mohd`@0, version@4 (=53), moduleId@8 (u64), `fileCount`@0x10,
  `firstResourceIndex`@0x20, `fileNameSize`@0x24 (=0 en release → noms strippés),
  `numResources`@0x28, `numBlocks`@0x2C, buildVersion@0x30.
- **Entrée fichier (88o = 0x58)**, header = **0x48**, stride 0x58 :
  `blockCount`@+0x0A (u16), `firstBlockIndex`@+0x0C (u32), `group` fourCC@+0x14 (u32 LE),
  `dataOffset`@+0x18 (u32), `compressedSize`@+0x20 (u32), `uncompressedSize`@+0x24 (u32).
- **Block (20o)** : `compOffset`@0, `compSize`@4, `decompOffset`@8, `decompSize`@12,
  `bCompressed`@16. `compOffset` est RELATIF au `dataOffset` du fichier.
- **dataBase** = `len(fichier) − max(dataOffset + compressedSize)` sur tous les fichiers.
- **Resources** : entrées à `blockCount==0` → **un seul blob Oodle** à `dataOffset`
  (pas via la table des blocs).
- Variantes : `deploy/{any,ds,pc}/levels/multi/<map>/<map>-rtx-new.module`. **`ds/`
  (dedicated server)** = petite, porte la géométrie (sbsp+mode+rtgo). `any/`=matériaux,
  `pc/`=rendu pleine-réso (layout entrée différent). ~chaque map MP a 1-3 `sbsp` en ds/.

## 3. Décompression Kraken

- Halo lie Oodle statiquement (pas de `oo2core` dans le jeu). Le seul `oo2core` du PC
  (RDR2 `oo2core_5`, Oodle 2.5) **refuse** le flux Halo (2021+). → décodeur clean-room ooz.
- Octets compressés : en-tête Kraken `0x8c 0x06 …` (nibble haut 0x8 = Kraken).
- `ooz.Decompress` validé : sbsp catalyst = 2 021 172 o, + btb_drydock/forest/ctf_aquarius/
  fo11_blank. Sortie = magic tag **`ucsh`** + GUID asset = tag valide.

## 4. Format de tag HI (CachedTag) — VÉRIFIÉ

Réf : `ElDewrito/AusarDocs/FileFormats/CachedTag.md`.

- **Header** : magic `ucsh`@0, version@4 (=27). **Décalage -4 vs AusarDocs en v27**
  (unknown8 = 4o) — auto-lock par invariant `headerSize + dataSize == len(tag)`.
  catalyst : deps=329, dataBlocks=7419, structs=7466, dataRefs=2, tagRefs=7405,
  strIds=28852, headerSize=0x84ac4, dataSize=0x168c70.
- **tablesStart = 0x50** (header AusarDocs plein). Ordre : dependencies(0x18) →
  dataBlocks(0x10 : size@0, section@6 enum16, offset@8 u64 ; section 0=header base0,
  1=tagdata base=headerSize) → tagStructs(0x20) → dataRefs(0x14) → tagRefs(0x10) → strings.
- **TagStruct (0x20)** : guid u128@0, struct_type u16@0x10 (0=Main, 1=TagBlock, 2=Resource),
  location u16@0x12, target_index i32@0x14 (= data-block de l'array/struct), field_block
  i32@0x18 + field_offset u32@0x1C (= OÙ le champ référence ce struct).
- Structure data-blocks catalyst : 1 gros (0x15af80 = attributs float32 normales/tangentes/UVs)
  + root (0x2518, sparse) + ~7400 petits (0x20-0x120 = instances de structs).

## 5. Géométrie : où sont les vertices

- **Pas dans le tag sbsp** (block#15 = attributs float32). Dans une **RESOURCE séparée**.
- catalyst resource **#678** (0xf0000 = 983040 o) = **vertices int16×4, w=0** : `(x,y,z,0)`.
  21368 vertices (run stride-8 à 4e int16 constant). Autres resources int16×4 : #682, #681,
  #677, #676, #672 (positions/normales/attributs par mesh-groupe).
- Les int16 sont **normalisés PAR MESH** (chaque mesh dans sa bbox) → plottés bruts ils se
  superposent (≠ map). **Déquant** : `world = bbox.min + (u16/65535)·(bbox.max − bbox.min)`.

## 6. CE QUI BLOQUE — les bornes de déquantification

- Source des bbox = struct **`compression info`** (sbsp.xml ligne 1421, IRTV) : record 0x54 =
  flags(2)+pad(2)+`position bounds 0`(min vec3 @+4)+`position bounds 1`(max @+0x10)+
  4×texcoord(_16)+2×unused(_14).
- **Introuvable dans catalyst** :
  1. Content-scan inefficace : struct-data dense (floats partout + padding `0xbc` + strings
     « Forcing Full Import because seams do not match » + tag-refs `tgcs`) → faux positifs.
  2. **Aucun data-block n'est un array ×0x54** → le layout du build catalyst **diffère du
     plugin IRTV** (versions HI divergentes ; offsets/tailles de champs non applicables tels quels).
- Floats world-bound-ish observés (ex bloc 0x870a0 : −250 / 47.5 / 158.4 / 153.2) mais non
  attribuables sans le walker.

## 7. Prochaine étape — field-walker version-aware

Le seul chemin fiable (le scan de contenu est définitivement écarté) :

1. **Field-walker** piloté par `sbsp.xml` (IRTV) : partir du MainStruct (`target_index` =
   root block), sommer les tailles de champs (table `group_lengths_dict` de
   `irtv/Halo/TagObjects/TagLayouts.cs` : `_38`struct=0 inline, `_40`block=20o,
   `_17`=12, `_16`=8, `_3E`=4, `_3C`=1, `_3D`=2, `_3B`=0, `_39`array=32, `_D`=4…) pour
   atteindre l'offset d'un champ `_40` ; résoudre via la struct-table (TagStruct dont
   field_block/field_offset matchent → `target_index` = data-block de l'array).
2. **Réconcilier le layout version-spécifique** du build catalyst (les défs IRTV ne matchent
   pas → valider chaque offset contre la donnée ; possible différence de version de tag).
3. Atteindre `compression info` (bbox/mesh) + le mapping mesh↔vertex-buffer (`vertex buffer
   indices` par mesh, sbsp.xml ligne 1283 ; `total vertex buffer count` ligne 1505).
4. Déquantifier chaque mesh + assembler en monde + rendu top-down 2D.
5. Liaison match→map : `map_id` (DB) → module ds/ de la map.

**Architecture cible produit** : extraction OFFLINE (à cause d'ooz GPLv3) → bake des contours
2D comme assets data → l'app overlay les positions film dessus (l'app ne linke pas ooz).

## 8. Références (clonées dans le scratch de session)

- `ElDewrito/AusarDocs` — format CachedTag (header + struct/block/ref layouts).
- `Gamergotten/Infinite-runtime-tagviewer` — plugins `Plugins/{sbsp,mode}.xml` (défs de
  champs) + `Halo/TagObjects/TagLayouts.cs` (table taille-par-type).
- `TheHaloArchive/infinite-rs` — lecteur de tag générique (mécanique struct-tree, Rust).
- `Gravemind2401/Reclaimer` — export de géométrie (réf C# la plus complète).
- `powzix/ooz` — décodeur Kraken clean-room (GPLv3), vendoré dans `internal/ooz`.
- `Krevil/InfiniteModuleReader` — lecteur de container `.module`.

---

## 9. RÉSOLUTION (carte 2D reconstruite) — `cmd/tmp_walker` + `cmd/tmp_geores`

### 9.1 Le field-walker version-aware (`cmd/tmp_walker`)
Parse un plugin IRTV (`sbsp.xml`) en arbre de champs, calcule l'offset de chaque champ via la
table de tailles (`group_lengths_dict` : `_38`=0 inline recurse, `_40`=20, `_17`=12, `_16`=8,
`_34/_35`=attr length…), et résout chaque tag-block `_40` via la struct-table (TagStruct dont
`field_block`==rootBlock, par `field_offset`).
- **Validation** : 23/42 offsets calculés matchent exactement la struct-table → layout compatible
  avec une **dérive** qui s'accumule (tailles de champs qui diffèrent entre versions).
- **Contournement de la dérive = matching par RANG** : les champs `_40` du plugin ET les TagBlocks
  de la struct-table sont dans le MÊME ordre (l'ordre des champs est stable entre versions). Le
  k-ième `_40` du plugin = le k-ième TagBlock (trié par field_offset). Robuste aux décalages.
  (catalyst : 42 champs XML vs 40 TagBlocks → les 2 derniers, decorator sets/runtime, absents.)

### 9.2 Découverte structurelle clé
Le tag sbsp de catalyst a sa **géométrie inline VIDE** (`meshes`, `compression info`, `mesh/vertex/
index resource look up` → block -1). Seul `instanced physics instances` = block#15 (collision 1.4Mo).
→ **La géométrie est STREAMÉE dans les resources du module** (`runtime_geo`, magic `ucsh`, plugin
`rawg.xml`). Ces resources (catalyst : ~277, indices ~388-666) portent `Per Mesh Data`.

### 9.3 `Per Mesh Data` (rawg.xml, record 0x90) = bornes MONDE par mesh
`Name`(4) `Mesh index`(2) flags(1) lightmap(1) `Scale/Forward/Left/Up`(4×12) **`Position`@0x38**
**`Bounds min`@0x44 `Bounds max`@0x50** (vec3) `sphere center`(12) `sphere radius`(4) `Lod levels`(20)
… → `Position` = placement MONDE du mesh ; `Bounds min/max` = AABB LOCALE. bbox monde = Position+Bounds.

### 9.4 Rendu (`cmd/tmp_geores`)
Itère les resources `ucsh`, parse via rawg.xml (Per Mesh Data offset 0x10), agrège les bbox monde
(Position+Bounds), filtre les meshes >40m (skybox/dalles), remplit l'empreinte 2D (plan = 2 axes
les plus larges). catalyst : 477 meshes, aire ~X[-13,6] Z[-8,54], structure centrale en diamant.

### 9.5 Reste pour AFFINER (optionnel)
- Vertices fins : maintenant qu'on a les bornes par mesh, déquantifier les int16 des resources
  buffers (`world = bbox.min + u16/65535·(bbox.max-bbox.min)`) pour des contours de mesh nets.
- Multi-map + bake : produire un asset 2D par map (via map_id → module ds/), overlay positions film.
- **Plugins IRTV** (`sbsp.xml`/`rawg.xml`) vendorés dans `cmd/tmp_{walker,geores}/` — provenance
  Gamergotten/Infinite-runtime-tagviewer (vérifier licence avant usage hors RE/offline).

---

## 10. OVERLAY POSITIONS JOUEURS sur la carte — `cmd/tmp_mapoverlay`

Génère une **image PNG** : carte 2D reconstruite (footprint + densité de meshes) + positions
joueurs décodées du film, alignées empiriquement.

- **Source positions** : `internal/analysis/positions.DecodeKeyframePositions` (positions full-state
  keyframe, ancre comb, ~1 snapshot/chunk ≈ intervalle ~20s). **MATCH-LEVEL** (pas d'attribution
  xuid — cf le mur d'attribution §3 de RESEARCH_THEATER_RE.md).
- **Alignement repères** : les positions film NE sont PAS dans le même repère absolu que la carte
  (§N RESEARCH : "engine state, pas coords cartésiennes brutes" ; échelle/offset/axes différents).
  → on **normalise** chaque nuage [0,1]² sur son plan horizontal (2 axes de plus grand span, bornes
  ROBUSTES 1-99 pct) puis on cherche la meilleure des **8 orientations** (rotations+reflets) qui
  maximise le recouvrement positions↔footprint. catalyst+film 01e1f945 : **62/74 positions (84%)**
  tombent sur la géométrie → repères cohérents après normalisation.
- **Trouver un match d'une map** : `cmd/tmp_mapmatch <mapname>` (DuckDB → match_id + film en cache).
  NB 000d5950=Cliffhanger, 7344d24f=Vagabond (les 2 films avec world_dump CE). catalyst : ex 01e1f945.
- Usage : `go run ./cmd/tmp_mapoverlay <module> <filmDir> <out.png>`.

### 10.1 TRAJECTOIRES par joueur — `cmd/tmp_traj`
- **CLÉ (insight user, 2026-06-30)** : les positions film sont **RELATIVES (deltas) à la précédente,
  depuis le SPAWN** — PAS des absolus. Trajectoire = `spawn (1er PosKindAbsolute) + accumulation des
  deltas (PosKindDelta8/DeltaAxis)` dans l'ordre temporel. Mon erreur initiale = connecter les absolus
  (épars + mal-attribués) → scribble. Avec l'accumulation des deltas → **chemin propre et cohérent**.
- **Résultat 000d5950** : slot519 (JGtm) = 180 deltas → **trajectoire nette de 181 points** (rendu
  polyligne, couleur Okabe-Ito daltonien-safe, spawn=disque blanc, fin=disque couleur).
- **Limite = COUVERTURE des slots, pas la qualité** : seul slot519 a des deltas (180) ; les 7 autres
  en ont ~0 (slot515:4, autres:0) car le **frame-decoder desync** avant d'atteindre leurs records (§3).
  → 1 trajectoire nette, pas 8. Les 8 nécessitent de finir le frame-decoder.

### 10.3 Diagnostic frame-decoder (`cmd/tmp_desync`, pourquoi pas 8 joueurs)
Sur 000d5950 (29257 frames type-0) : **44% des frames desync** (32% avec world_dump_full), et en moyenne
seulement **~1.4 record décodé avant le desync**. Records bipèdes décodés OK : slot519=**23847**, slot512=868,
slot515=776, les 5 autres 3-13. MAIS records ≠ positions exploitables :
- **slot519** (joueur enregistreur, POV) = record 0 de quasi CHAQUE frame → 180 deltas → trajectoire nette.
- **slot515/512** = surtout des absolus, mais **scatter/bruités** : leurs positions sautent >55u entre
  échantillons (plus d'1/2 boîte [-100,100]) = mis-attribution, pas un chemin.
- **autres slots** = quasi jamais atteints (le frame s'arrête au desync avant eux).
**Cause racine = binding incomplet** : le desync dominant est `typeIdx=0 i0` (~6000-10000×) = **slots NON liés
dans le World** (decodeDelta échoue car archetype inconnu). **1109 slots distincts non-liés** (entités
dynamiques : objets/projectiles/respawns) absents du snapshot CE world_dump. Ces records arrivent TÔT et
bloquent les bipèdes suivants. Pistes testées qui NE marchent PAS : world_dump_full (32% vs 44%, insuffisant) ;
World **persistant** (pire, 38% : les NEW desyncés mis-bindent) ; bootstrap World offline (0 bipède).
**Conclusion : décoder les 8 joueurs = compléter le binding de TOUTES les entités dynamiques (décoder les
records NEW proprement sur le flux continu) = le mur central du frame-decoder.** Chantier multi-session.
- **Décodage dense exige le `world_dump` CE** → seulement Cliffhanger 000d5950 / Vagabond 7344d24f ;
  bootstrap World OFFLINE depuis keyframe NE marche PAS (`cmd/tmp_kf_world` : 0 bipède sur catalyst).
- **bots** (alerte user) : nuage full-state = joueurs+bots+objets ; pour per-joueur filtrer slots bots.

### 10.2 Disponibilité des maps (vérif GUID, 2026-06-30)
- L'install local (Steam D: ET Xbox C:, identiques) a **31 modules de maps** seulement (catalyst, chasm,
  forest, ridgeline, ctf_*, btb_*, sgh_*, va_*, fo_*). **Cliffhanger / Vagabond NON présents** (maps
  saisonnières/live, lazy-download). Donc trajectoire-propre (Cliffhanger 000d5950) + sa carte ne se
  combinent pas localement.
- **GUID DB ≠ GUID module** : `map_id` et `map_version_id` (match_registry) sont des GUID d'**asset UGC
  (variante de map)**, espace d'ID SÉPARÉ des tags moteur. Vérifié définitivement (`scan d'octets`) : même
  le GUID `map_version_id` de Catalyst (témoin) n'apparaît PAS dans le module catalyst. → impossible de
  pointer un module depuis la DB par GUID. Identifier une map non-locale via la DB n'est pas faisable.
- **Pour obtenir Cliffhanger** : la charger une fois en jeu (déclenche le download) → apparaît dans
  `deploy/ds/levels/multi/<codename>/`, puis reconstructible.
- **Bots** (alerte user) : le nuage full-state contient TOUTES les entités répliquées (joueurs + bots
  + objets). On filtre 2 artefacts structurels (§N) mais PAS les bots → un match avec bots gonfle le
  nuage. Pour du per-joueur, les slots bots sont à écarter (roster PLAYER_METADATA vs BOT_METADATA pkt 12).
- **Alignement absolu** (au lieu de normalisé) : nécessiterait de caler le repère film sur le monde
  via une transformée affine (registration ICP positions↔géométrie, ou résoudre le scale [-100,100] box).

### 10.4 Recherche "coordonnée par-joueur du décodage des armes" + mur all-8 (2026-06-30)
Investigation suite à l'hypothèse user (« on est tombés sur les coordonnées par joueur pendant le décodage
des armes, encodé de manière typique »). Sources : `HANDOFF_KILLFEED_VOIE_A.md`, `WALK_PORT_NOTES.md`.
- **Verdict** : la "coordonnée par-joueur" = le composant **i0 `object-position-dynamic-precision`**, lu
  AVANT le desync. C'est EXACTEMENT ce que `tmp_traj` utilise déjà. **Pas d'encodage alternatif** qui
  contournerait le mur — c'est le même chemin.
- **2 fixes `tmp_traj`** (rendent 2 joueurs au lieu d'1) :
  1. capture i0 **où qu'il soit** dans les composants (bug : on ne captait que `Comps[0]`).
  2. filtre de continuité (`thr=55`) **désactivé** (`thr=400`) : il rejetait du mouvement RÉEL. slot515
     a 111 absolus qui forment un **vrai chemin monde cohérent** (mode `ABS=1` → absTrajectory).
  → **slot519** (POV/JGtm) = 181 pts via **deltas** (repère relatif spawn) ; **slot515** = 111 pts via
  **absolus** (repère monde). Les deux N'ONT PAS le même repère (deltas accumulés ≠ absolus monde) → pas
  d'overlay trivial ; slot519 en absolus = **bruit** (530 abs scatter), slot515 en deltas = ~0.
- **FAIT ACTÉ (user, ne pas contredire)** : les positions des 8 joueurs (et plus selon départs/arrivées,
  modes) SONT toutes dans UN SEUL film. Mon hypothèse "LOD réseau / impossible" était FAUSSE — retirée.
  C'est un **problème de décodage**, pas de disponibilité. Le blocage réel a 2 composantes :
  1. **Desync séquentiel** : 44% des frames desync (entités non-bipèdes non-liées) avant d'atteindre les
     autres bipèdes → seul slot519 (POV, record précoce) est lu densément. (cf §10.3)
  2. **Encodage keep-baseline** : les bipèdes NON-POV émettent i0 en mode **keep-baseline** (`bUsePred=1`,
     `PosKindRaw`, lecteur `readRawVec3`/`FUN_1406d676c` = 96 bits) — slot512 = 793 samples Raw. Ce mode
     n'était PAS exploité (ignoré dans la construction de trajectoire).
- **BUG CORRIGÉ (réel)** : `readRawVec3` (position_capture.go) lisait 3×R(32) MSB-first SANS byteswap.
  Or les float32 du film sont **little-endian** (prouvé par `positions.readFloat32LE`). Sans swap → garbage
  (e-34/e+35). Avec swap → une composante devient saine et **incrémente dans le temps** (slot512 : ~9.4→13.4
  = une vraie coordonnée), mais les 2 autres ne sont **pas** des float32 → le keep-baseline 96 bits N'EST PAS
  3 float32 contigus (scan d'offsets `cmd/tmp_rawalign` : aucun offset ne donne 3 coords saines). C'est soit
  une **position prédite** (dead-reckoning baseline+vélocité), soit un **format quantifié**. Bits consommés
  inchangés (96) — fix sûr pour le décodage, corrige seulement la valeur capturée.
- **CHANTIER POUR ALL-8 (Ghidra)** : (a) RE `FUN_1406d676c` (lecteur keep-baseline 96 bits) pour décoder la
  position des bipèdes non-POV ; (b) lier les entités non-bipèdes (`new-entity` non-#35 default-state) pour
  tuer le desync séquentiel et atteindre tous les bipèdes. Ghidra requis (pas d'instance active au 2026-06-30).
- **TRANCHÉ PAR GHIDRA (2026-06-30, connexion OK)** :
  - `FUN_1406cfe44` (deser i0) : le chemin **keep-baseline** (`bUsePred==1`) lit 1 bit + `FUN_1406d676c`
    mais **n'écrit JAMAIS le champ position** (`lVar3+4`). Branche delta full-precision : `local_58 = local_48`
    = copie la **baseline** (position précédente, depuis `*param_4`). ⇒ **keep-baseline = position INCHANGÉE**,
    `PosKindRaw` N'EST PAS une position (piste écartée). `FUN_1406d676c` = helper bas-niveau copie-N-bits
    (avec byteswap → confirme la convention LE). Le byteswap dans `readRawVec3` reste correct mais MOOT.
  - **Le mur = desync séquentiel** (`cmd/tmp_newdiag`, world_dump_full=2229 slots) : 32% frames desync ;
    desyncer = **DELTA sur slot non-lié** (typeIdx=0 ×6196) >> NEW (×1095). Cause : ~40 archétypes dont les
    records **NEW ne décodent pas** (default-state non porté, taux clean faibles : typeIdx=5→6/68, 9→13/70,
    18→35/68, 2→37/111, 40→27/66...). Chaque NEW raté = slot jamais lié = cascade DELTA-desync.
  - **MÉCANISME DES JOUEURS NON-POV TROUVÉ (le + important)** : mesure exhaustive `cmd/tmp_allslots`
    (i0 sur TOUS les slots, tout le film) → SEULS slot519 (POV/JGtm, 675 abs+180 deltas) et slot515 (114 abs)
    streament i0 dense. slot512 = **793 keep-baseline** (position tenue). Raison : le client enregistreur
    **dead-reckone les autres joueurs via la VÉLOCITÉ i1** (`object-translational-velocity`), corrigeant
    rarement (keep-baseline i0 = "fais confiance à la prédiction"). Donc leur mouvement EST dans le film
    (user a raison) mais en **vélocité à intégrer**, pas en position i0. **Décoder i1** = `consumeDynPrecVec3`
    R(1)présent + R(19) direction packée (`FUN_1406d8288` = unpack octaédrique : table `DAT_1447084d0`[idx*8]
    + ~9 const float DAT_143cd83xx/cf77xx) + R(10) scale (log/exp `FUN_14076d6dc`). Le port actuel CONSOMME
    mais ne décode pas la valeur. **PIPELINE all-8** : porter unpack+scale → capturer i1 par biped → intégrer
    (vel×dt, dt=Δts frames) depuis le 1er absolu (spawn) → corriger à chaque keyframe absolu. Caveats : dérive
    entre corrections ; l'attribution per-joueur des positions keyframe reste non résolue (match-level).
  - **FINIR (voie B) = porter le default-state deser (`vtable+0x60`) des ~40 archétypes** → NEW décodent →
    binding complet → plus de desync → tous les bipèdes lus chaque frame. = "long pole" documenté, multi-passes.
  - **Calibration data-driven du desync TESTÉE** (`cmd/tmp_resync`, World persistant + brute-force largeur
    default-state par NEW + lookahead 1-niveau) : frames entièrement décodées 44%→57% (chunks 2-4), MAIS
    n'a PAS révélé les autres bipèdes (ils sont keep-baseline, pas un pb de desync). Confirme : voie A (i1).
  - **Dispatch confirmé (`FUN_1408f1aa4`, lecteur new-entity)** : R(6) typeIndex → classe archétype =
    `*(*(world+0x18)+8+typeIndex*8)` (table RUNTIME du monde) → appel VIRTUEL `(*plVar2+0x60)(...,param_4=bitreader,1)`
    = default-state, puis `(*plVar2+0x88)` = binding-default, puis `FUN_14076cb60` = itérateur mask+composants.
    **PAS de lecteur générique unique** : chaque archétype a son `vtable+0x60`. Mapper typeIndex→fonction exige
    la table runtime → **debugging live** (break `FUN_1408f1aa4`, lire `plVar2`, `*plVar2`, `+0x60`) avec le
    jeu en cours de match. Biped #35 déjà porté (`FUN_140F44C38`). Reste : capturer le mapping des ~40 autres
    + porter chacun. = chantier multi-session runtime-dépendant (purist offline, cf `feedback_always_offline_pure`).
- **ANCRE-SCAN par slot-id = ÉCHEC** (`cmd/tmp_bipedscan`) : scanner le flux pour les ids 512-519 + décoder
  delta produit du **bruit** (step médian ~100 sur boîte ±100) — démarrer mid-stream casse l'alignement.
  L'alignement séquentiel est indispensable → le desync doit être réparé, pas contourné par scan.
- **Keyframes type-2 = seule source all-8** (full-state périodique, 1/20s, **26 keyframes** sur 000d5950
  via `cmd/tmp_kfpos`). Mais :
  - comb-anchor (package `positions`) ne rend que **~2.9 positions/keyframe** (76 total), **match-level**
    (sans identité). Sur les 8 bipèdes, comb-273 = **(0,0,0)** → la position n'est PAS à cet offset pour eux.
  - `DecodeFrameRecords` sur le keyframe **desync au 1er record** (endBit ~15) : le keyframe utilise la
    structure **new-entity** (full-state), pas le format delta (`cmd/tmp_kf_world`).
  - décodage structuré des 8 bipèdes (`cmd/tmp_loadout`, ancrage sur littéraux d'armes) **calibre 1/8**
    (les 7 autres desyncent avant d'atteindre WST i43). Le record biped keyframe **démarre à i15** (PAS
    de i0) : la position n'est PAS dans les component-records.
  - **scan flottants autour des combs** (`tmp_around` jetable) : pas de triplet (x,y,z) propre, seulement
    des valeurs **mono-axe** (alignement) → la position keyframe est **quantifiée**.
  - **LOCALISATION** : la position keyframe des bipèdes est très probablement dans **i15
    `object-low-frequency-component`** (`consumeObjectLowFrequency`, `FUN_142f071f4`, descripteur
    143d0bf08 +0x38) — la boucle keyframe y lit des **paires quantifiées 12 bits** (lignes ~170-172 de
    `components_object_state.go`). Le port **consomme** mais **ne décode pas les coordonnées**, et est
    marqué "re-valider avant de s'y fier". **C'est LE chantier RE pour les positions all-8 (coarse).**
- **RECO** : solide livrable actuel = **2-3 trajectoires denses + carte reconstruite**. All-8 coarse
  (1 pt/20s) = session Ghidra ciblée sur `FUN_142f071f4` (déquant transform i15) → bornée mais incertaine.
- **Probes (untracked, NON committées sauf accord)** : `cmd/tmp_kfpos`, `cmd/tmp_kfprobe`. `cmd/tmp_traj`
  (tracké) modifié : capture i0-anywhere + mode `ABS=1`. `cmd/tmp_loadout` : mode `trace <start> <d> <rsp>`.

---

## 11. Extension — NOMS DE ZONES / CALLOUTS (investigation 2026-06-26)

Question : peut-on sortir les **noms de zones** d'une map (callouts comms "Bunker/Rampe", zones
d'objectif Strongholds/CTF/KOTH), les **délimiter**, les **placer sur la 2D** ?

Réponse (investigation empirique via `cmd/tmp_moduleread`) : **oui, faisable, faible risque** — c'est
dans le module **`any/`** (tag scénario/placements gameplay), **PAS** dans le film ni dans la `ds/`
(géométrie). Recensement catalyst : `ds/` = 7 tags géométrie/rendu/son (aucun scénario) ; `any/` = 41
tags dont un tag scénario/script (`hscn`/`hsc*`), `pfnd` (pathfinding/nav), `scen` (scenery), `sddt`
(structure-design). Même espace monde que la géométrie → la transfo monde→2D d'ici s'applique. Callouts
comms = points+rayon (Voronoï) ; zones d'objectif = volumes. Frictions : layout scénario
version-spécifique (field-walker comme ici) + résolution string-ID→texte + confirmer où sont les zones
d'objectif (scénario de map vs module game-mode). **Étape de confirmation non faite** : décompresser le
tag scénario `any/` (himodule.Extract) et lister ses "named locations".

**Doc détaillé : `.ai/INVESTIGATION_MAP_ZONE_CALLOUT_NAMES.md`.**

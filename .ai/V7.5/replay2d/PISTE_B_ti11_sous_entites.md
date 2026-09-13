# PISTE B — ti=11 i16-i31 `sub-objective-entities` : RE de feuille + conception

> Agent RE + CONCEPTION, lecture seule. Aucun code Go produit, aucun `go build/test/run`,
> aucune mesure, aucune écriture au dépôt hors ce fichier scratchpad.
> Ghidra `HaloInfinite.exe`, image base `0x140000000`, 311 103 fonctions, lecture seule
> (MCP opérationnel). Chaque fait porte une adresse Ghidra ou un `fichier:ligne`.
> Daté 2026-08-27. Branche de travail `feat/v75`.
>
> OBJET : établir si on peut lire NATIVEMENT l'identité des sous-zones d'un objectif
> (A/B/C) depuis les 16 composants `managed-objective-sub-objective-entities-component`
> (ti=11 i16..i31), jamais adressés jusqu'ici. Débloquerait Total Control (ancrage ti=13
> cassé, `[!]`) et fiabiliserait Bastion/Strongholds/KOTH.

---

## 0. Résumé exécutif

**La feuille est RÉSOLUE, et c'est la meilleure des issues de grammaire.** Les 16 composants
i16..i31 partagent UN désérialiseur `FUN_142ed5974` qui est **byte-pour-byte identique** au
désérialiseur d'i3 object-reference `FUN_142ed5550` (confirmé trivial et bit-exact au
catalogue du 27/08), à la SEULE différence de destination : i3 écrit un `R(32)` à `dst+0x40`
(fixe), chaque sous-entité écrit un `R(32)` à `dst + 0x19c + index*4`, l'`index` (0..15)
venant du descripteur (`*(param_1+8)`), exactement le partage de rôle des 4 `rtpc` de ti=10
et des 32 `masked-property` de ti=13.

**Ce que chaque slot porte** : un `R(32)` qui est une RÉFÉRENCE D'ENTITÉ (un GlobalID), de
même nature machine qu'i3 object-reference. Ce ne sont PAS des états par sous-zone, PAS des
index, PAS un tableau à compte variable : ce sont 16 pointeurs d'entité à largeur fixe,
lus inconditionnellement, sans porte, sans quantification, sans vec3.

**La structure d'état est un ARBRE parent/enfant d'objectifs**, prouvée par les offsets
contigus : i14 state `+0x194`, i15 `parent-objective` `+0x198` (`R(32)`, une référence),
i16..i31 `sub-objective-entities` `+0x19c..+0x1d8` (16 × `R(32)`), i32 `+0x1dc`. Le tableau
des enfants est donc EXACTEMENT borné à 16 slots — l'offset d'i32 le prouve.

**Verdict** : **GO-CONDITIONNÉ** sur le MÊME test de cadre que le reste de ti=11 (le corps
d'image-clé 108b n'a jamais été reproduit sur le film ; plafond 0,51 % bit-exact sur le
bipède ti=35, mais ce plafond est dans les FEUILLES du bipède, pas dans le cadre). i16-i31
n'ajoute AUCUN risque de grammaire par-dessus i3 : ses feuilles sont le même code. Le
risque résiduel est entièrement le cadre. **OUI, ce chemin contourne l'ancrage ti=13 cassé
de Total Control** : il ne dépend d'aucun balayage de bande de slots ti=13, seulement du
dispatch d'état complet ti=11 par entité — un ancrage STRUCTUREL, pas géométrique.

---

## 1. Chaîne de résolution Ghidra (méthode + preuve)

Méthode du scout du 27/08 (`RE_LECTEUR_IMAGE_CLE_SCOUT_2026-08-27.md` §2.1, worktree
`LevelUp-wt-re-lecteur`), rejouée ici de bout en bout pour la feuille sous-entités :

`search_strings` (nom) → xref DATA = getter `vtable[0x08]` → xref DATA du getter = entrée de
vtable → `read_memory` de la vtable → deser en `vtable[0x30]`.

| Étape | Adresse | Preuve |
|---|---|---|
| Chaîne `.rdata` du composant | `143c954c0` | `search_strings "sub-objective-entities"` → 1 seule occurrence, `"managed-objective-sub-objective-entities-component"` |
| Getter de nom (vtable[0x08]) | `14064c7c0` | `read_memory` : `48 8d 05 f9 8c 64 03` = `LEA RAX,[143c954c0]` puis `c3` = `RET`. Getter à 2 instructions. Non défini comme fonction par Ghidra (octets bruts entre padding `cc`), d'où l'échec de `decompile` — normal pour un getter feuille |
| Entrée de vtable pointant le getter | `143c97128` | `get_xrefs_to 14064c7c0` = « From `143c97128` [DATA] » → base de vtable = `143c97128 - 0x08` = **`143c97120`** |
| Vtable complète | `143c97120` | `read_memory` 80 o, parsée ci-dessous |

Vtable `143c97120` (mêmes slots partagés que toute la famille objectif/bipède) :

| offset | valeur | rôle |
|---|---|---|
| +0x00 | `0x14117b4a0` | destructeur (identique famille) |
| +0x08 | `0x14064c7c0` | getter de nom (résolu ci-dessus) |
| +0x10 | `0x1404ab600` | ret-false (identique famille) |
| +0x18 | `0x142edba24` | **écrivain** |
| +0x20 | `0x1411c8f80` | int3 (identique famille) |
| +0x28 | `0x14076ce9c` | **thunk** (le MÊME que bipède/i3 — pur forwarder vers +0x30) |
| +0x30 | `0x142ed5974` | **DÉSÉRIALISEUR** (la feuille) |
| +0x38 | `0x1408d8220` | — |

Les 4 slots invariants (`0x14117b4a0`, `0x1404ab600`, `0x1411c8f80`, `0x14076ce9c`)
concordent EXACTEMENT avec les 5 feuilles déjà résolues par le scout (i3/i5/i12/i13/i14) :
c'est bien un composant de la même famille d'archétype, lu par le même dispatch
`vtable[0x28]` → `vtable[0x30]`.

---

## 2. GRAMMAIRE des feuilles i16-i31 (réponse Q1)

### 2.1 Le désérialiseur `FUN_142ed5974` — un `R(32)` pur, indexé

`decompile_function 142ed5974` ; `get_function_callees` = **aucun** (feuille pure, aucun
sous-read caché). Corps (résumé fidèle) :

```
iVar1 = *(param_2 + 0x38)        ; bits libres dans le mot courant du BitReader
lVar2 = *(param_3 + 0x10)        ; base de l'état de destination (dst)
uVar6 = high32(*(param_2 + 0x30)) ; les 32 bits de tête du buffer
if (0x40 - iVar1 < 0x20) { ... recharge 8 octets byte-swappés, extrait 32 bits ... }
else { *(param_2+0x2c) += 0x20 ; buffer <<= 0x20 ; ... }
*(param_2 + 0x38) = ...          ; met à jour l'état du BitReader
*(uint *)(lVar2 + 0x19c + (*(int *)(param_1 + 8)) * 4) = uVar6   ; ÉCRIT LE R(32)
return CONCAT71(...,1)            ; rend toujours vrai
```

Marques d'une feuille TRIVIALE (grille adversariale du catalogue §1.4) — toutes NÉGATIVES :

- **Largeur** : `0x20` partout (`0x40 - iVar1 < 0x20`, `+= 0x20`, `iVar1 - 0x20`). Un `R(32)`
  MSB-first, la MÊME machine de BitReader que `filmdec.BitReader` (état `+0x28/+0x2c/+0x30/
  +0x38/+0x40`, recharge big-endian). Aucune branche ne change la largeur.
- **Aucune porte** (`R(1)` de présence) : la lecture est INCONDITIONNELLE.
- **Aucune quantification** : pas de `MUL/DIV/CVT/float`, pas de min/max, pas d'appel à
  `FUN_1406d84b4`. La valeur est rangée BRUTE (`uVar6`, un `uint`).
- **Aucun compte data-dépendant, aucun vec3, aucun drapeau d'encodage.** Rend toujours `1`
  (`MOV AL,0x1` équivalent) → **aucun désync possible**.
- **Destination indexée** : `dst + 0x19c + (*(param_1+8))*4`. `param_1` est le descripteur du
  composant ; `*(param_1+8)` est l'index de slot (0..15), lu dans le descripteur — le même
  mécanisme que `rtpc` ti=10 (`etat + 0x17c + index*8`, `components_managed_object.go:113-118`)
  et `masked-property` ti=13 (`index = descripteur+8, 0..31`, `components_managed_property.go:26-33`).

### 2.2 Preuve croisée décisive : identique à i3 object-reference

`decompile_function 142ed5550` (i3, déserialiseur confirmé trivial et bit-exact par le
catalogue du 27/08, `CATALOGUE_OBJECTIFS_DECRITS_2026-08-27.md:45,152`) rend un corps
**byte-pour-byte identique** à `FUN_142ed5974`, à UNE ligne près :

- i3 : `*(uint *)(lVar2 + 0x40) = uVar6;`               (destination FIXE `dst+0x40`)
- i16-31 : `*(uint *)(lVar2 + 0x19c + *(param_1+8)*4) = uVar6;` (destination INDEXÉE)

Autrement dit : **une sous-entité est exactement un i3 object-reference, rangé dans un
tableau au lieu d'un champ scalaire.** i3 a été établi comme « la référence vers l'objet
physique » ; par identité de code, chaque slot i16..i31 est une RÉFÉRENCE D'ENTITÉ de même
nature — un `R(32)` GlobalID.

### 2.3 L'écrivain `142edba24` — symétrique, verrouille « aucun désync »

`read_memory 142edba24` (non défini comme fonction ; désassemblage manuel des octets) :

```
49 8b 40 30                 MOV  RAX,[R8+0x30]              ; R8 = param_3 (état), RAX = dst
48 63 49 08                 MOVSXD RCX,[RCX+0x8]            ; RCX = param_1, index = *(param_1+8)
44 8b 52 38                 MOV  R10D,[RDX+0x38]            ; RDX = état d'écriture de bits
4c 8b 42 30                 MOV  R8,[RDX+0x30]
44 8b 8c 88 9c 01 00 00     MOV  R9D,[RAX + RCX*4 + 0x19c]  ; SOURCE = dst + 0x19c + index*4
b9 20 00 00 00              MOV  ECX,0x20                   ; LARGEUR = 32
01 4a 2c                    ADD  [RDX+0x2c],ECX             ; avance le compteur de bits de 32
...
```

L'écrivain LIT le même champ `dst + 0x19c + index*4` et écrit `0x20 = 32` bits — largeur
immédiate figée. Lecteur et écrivain se MIROITENT (`R(32)` des deux côtés, largeur constante
dans les deux sens) : c'est le critère qui rend le statut `porte` et non `partiel` (même
raisonnement que ti=10 §1b et ti=13). **Aucune désynchronisation n'est structurellement
possible.**

### 2.4 Réponse Q1, tranchée

- **Forme** : 16 lectures `R(32)` INDÉPENDANTES, pas un tableau à compte variable, pas une
  liste taguée. Chaque slot est lu inconditionnellement (état complet, aucun masque de
  présence : `WalkKeyframeFullState` pose `t.Mask = ^uint64(0)`,
  `keyframe_fullstate_loop.go:90`).
- **Largeur** : 32 bits chacun, fixe. Coût total des 16 slots dans un record ti=11 = **512
  bits**, invariant.
- **Offset de champ** : `dst + 0x19c + slot*4`, slot 0..15 (voir §3).
- **Ce que porte chaque slot** : un GlobalID d'entité (`R(32)`), de la MÊME nature qu'i3
  object-reference (référence vers une entité). PAS un index de zone, PAS un état par
  sous-zone. L'identité de zone se lit dans l'ORDRE DES SLOTS (voir §4), la référence donne
  le pont vers l'entité qui porte l'état/la position de cette sous-zone.

---

## 3. Structure d'état ti=11 et arbre parent/enfant (offsets prouvés)

Les feuilles voisines, résolues par la même méthode Ghidra ce jour, montrent une struct C++
CONTIGUË et cohérente (`dst = *(param_3+0x10)`) :

| i | composant | deser | offset | grammaire |
|---|---|---|---|---|
| i3 | object-reference | `FUN_142ed5550` | `+0x40` | `R(32)` (scout + vérif ce jour) |
| i5 | type | `FUN_1410fc4a4`→`FUN_14080dec4` | `+0x150` | `R(32)` nommé `"objective-type"` (scout) |
| i12 | progress | `FUN_142ed575c` | `+0x18c` | `R(32)` (scout) |
| i13 | required-progress | `FUN_142ed5844` | `+0x190` | `R(32)` (scout) |
| i14 | state | `FUN_142ed5948`→`FUN_1424d9a30` | `+0x194` | `R(3)` (scout) |
| **i15** | **parent-objective** | **`FUN_142ed5674`** | **`+0x198`** | **`R(32)`** (résolu ce jour) |
| **i16..i31** | **sub-objective-entities** | **`FUN_142ed5974`** | **`+0x19c..+0x1d8`** | **16 × `R(32)`** (résolu ce jour) |
| i32 | outro-phase-duration | `FUN_142ed5634` | `+0x1dc` | 8 bits QUANTIFIÉS (`FUN_1406d84b4`) — NON trivial |

Résolutions neuves de ce jour (mêmes 4 pas Ghidra qu'en §1) :

- **i15 parent-objective** : chaîne `143c95558` → getter `141177ee0` → vtable `143d08fd0`
  (deser en +0x30 = `142ed5674`). `decompile 142ed5674` = corps identique à i3/i16-31, sortie
  `*(uint *)(lVar2 + 0x198) = uVar6;` → **`R(32)` vers `dst+0x198`**. C'est le pointeur INVERSE
  de l'arbre : la référence de l'objectif PARENT.
- **i32 outro-phase-duration** : chaîne `143c954f8` → getter `141177ed0` → vtable `143d08f80`
  (deser en +0x30 = `142ed5634`). `decompile 142ed5634` : `FUN_1406d84b4(param_2,param_2,0,
  DAT_143cd84b8,8,0,1)` puis `*(undefined4 *)(lVar1 + 0x1dc) = uVar2;` → **8 bits déquantifiés
  vers `dst+0x1dc`** (hors périmètre : non trivial, mais sa PLACE prouve la borne du tableau).

**Le tableau des sous-entités est EXACTEMENT `0x19c..0x1d8` = 16 slots de 4 octets = 0x40
octets, et se termine PILE à `0x1dc` (i32).** L'affirmation « 16 slots, index local 0..15 »
n'est donc PAS une inférence : elle est bornée par l'offset d'i32. Si un slot avait un index
16..31, il écrirait à `0x1dc..0x218` et écraserait i32/i33 — ce que la struct exclut.

Lecture de l'arbre : `+0x198` (parent) ↔ `+0x19c..+0x1d8` (16 enfants) forment un lien
bidirectionnel entre entités ti=11. Un objectif RACINE (le mode) a `parent = 0/0xFFFFFFFF`
et ses slots enfants peuplés ; un objectif FEUILLE (une zone) a `parent = GlobalID racine` et
ses slots enfants vides.

---

## 4. SÉMANTIQUE : les 16 slots ↔ zones A/B/C (réponse Q2)

### 4.1 Ce que le nom + la structure établissent

`managed-objective` est le SUIVI D'OBJECTIF DU HUD (34 composants `managed-objective-*`),
pas l'objet physique (lot R4, registre des reports ligne 205 : « ti=11 N'EST PAS L'OBJET,
c'est le DESCRIPTEUR d'objectif ; AUCUN composant ne porte de position »). Le couple
`parent-objective` (i15) / `sub-objective-entities` (i16..i31) est un ARBRE d'objectifs HUD :

- Un mode à bases (Strongholds, Total Control, Bastion) instancie UN objectif RACINE (l'objectif
  global du mode) ET N objectifs ENFANTS, un par zone.
- La racine liste ses zones dans `sub-objective-entities` : **slot 0 = zone A, slot 1 = zone B,
  …** L'INDICE DE SLOT est l'identité stable de la sous-zone (un ordre fixé à la construction
  de l'archétype, donc constant image-par-image et film-par-film).
- Chaque enfant (une entité ti=11 à part) porte, par ses propres feuilles déjà résolues :
  i3 object-reference (→ l'objet physique de la zone : sa position sort par la chaîne objet),
  i5 type, i14 state (contesté/neutre/tenu), i1 color (propriétaire, feuille non encore
  adressée).

### 4.2 Combien de slots peuplés pour un objectif à N zones

Prédiction structurelle (à confronter, pas à supposer) : sur l'objectif RACINE, le nombre de
slots NON nuls (`!= 0` et `!= 0xFFFFFFFF`) = le nombre de sous-zones DÉCLARÉES du mode :

- **Bastion / Strongholds** : 3 zones → 3 slots peuplés, 13 vides.
- **Total Control** : le mode DÉCLARE 13 à 18 zones sur la carte
  (`PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md:87`) mais n'en ACTIVE QUE 3 par manche. Deux
  lectures possibles, que la mesure départagera :
  - (H-A) la racine liste les 13-18 DÉCLARÉES ; les 3 ACTIVES se distinguent par l'i14 state
    de chaque enfant (seules 3 ont un état « actif »/« tenu »). C'est le débouché le plus
    riche : identité ET activité par manche, sans ancrage.
  - (H-B) la racine ne liste que les 3 ACTIVES de la manche courante (jusqu'à 16, > 3
    possibles en variantes) ; le cardinal des slots peuplés = 3.
  - Dans les DEUX cas, le cardinal ≤ 16 (borné par la struct) — ce qui, seul, effondre le
    défaut ti=13 (« jusqu'à 77 désignations simultanées croissant avec le film »,
    `document.go:190-193`). 16 est un plafond DUR, 77 est un artefact d'ancrage.
- **KOTH** : 1 colline active → 1 slot peuplé (ou une racine à N slots pour les N collines de
  la rotation, l'active distinguée par i14).

### 4.3 Recoupement avec le nombre de formes servies par carte

Le catalogue de formes (`config/titles/halo_infinite/mappings/objective_roles.toml`,
`map_objectives.json`) sert 3 zones Bastion, 13-18 `totalcontrol_zone` (entrée RETIRÉE le
27/08, `objective_roles.toml:187-188`), etc. La confrontation SÉMANTIQUE (§6) est : le nombre
de slots peuplés de la racine doit ÉGALER le nombre de formes du catalogue pour ce mode/carte
(H-A), OU le nombre d'actives par manche (H-B). C'est le test qui distingue H-A de H-B.

### 4.4 Un oracle d'auto-cohérence GRATUIT (interne au film)

Le lot R4 a mesuré que **les GlobalID des entités ti=11 sont STABLES d'un film à l'autre**,
décalés d'une constante par film : `[1383 1399 1400 1415 1416]` sur deux films CTF, `+1676`
sur un troisième (`REGISTRE_REPORTS.md:207`). Les `R(32)` des slots sub-objective sont des
GlobalID dans CE MÊME espace de numérotation (même nature qu'i3). Donc : **les valeurs non
nulles des slots de la racine doivent être un SOUS-ENSEMBLE des GlobalID des entités ti=11
présentes dans le même film** (les enfants existent comme entités lues ailleurs dans la même
image-clé). C'est un contrôle de cohérence purement interne, sans oracle externe — et il
distingue aussi la lecture « enfant ti=11 » de « objet physique » (si les valeurs matchent des
GlobalID ti=11 → enfants objectifs ; si elles matchent des GlobalID d'objets `ti=42/23` → objets
physiques directs). Les deux restent exploitables ; ce contrôle dit LEQUEL.

---

## 5. EXTENSION de `keyframe_fullstate_loop.go` (réponse Q3)

### 5.1 Ce qui est déjà en place

`WalkKeyframeFullState` (`keyframe_fullstate_loop.go:69`) porte le CADRE d'état complet
(en-tête 108b, bloc par défaut, boucle de composants) et REUTILISE `traverseComponentLoop`
(`traverse.go:1201`), qui itère `arch.Components` dans l'ordre et dispatche chaque composant
par `consumeByNameCapturing` → `consumeByName` (`traverse.go:194`). En état complet,
`t.Mask = ^uint64(0)` : **les 34 composants ti=11 sont visités en ordre i0..i33, tous
présents.** Le harnais d'oracle (frontières de records) existe déjà.

### 5.2 Les 7 feuilles triviales à câbler dans `consumeByName`

Toutes des `R(n)` à largeur fixe, adresses Ghidra prêtes (aucun `default_state` d'archétype à
jouer — ces feuilles vivent dans la boucle de composants, pas dans le bloc par défaut) :

| composant | case à ajouter | grammaire | champ à publier |
|---|---|---|---|
| `managed-objective-type-component` (i5) | `br.ReadBits(32)` | `R(32)` | type d'objectif |
| `managed-objective-state-component` (i14) | `br.ReadBits(3)` | `R(3)` | état vivant |
| `managed-objective-progress-component` (i12) | `br.ReadBits(32)` | `R(32)` | numérateur |
| `managed-objective-required-progress-component` (i13) | `br.ReadBits(32)` | `R(32)` | dénominateur |
| `managed-objective-object-reference-component` (i3) | `br.ReadBits(32)` | `R(32)` | réf. objet physique (LE PORTEUR) |
| `managed-objective-parent-objective-component` (i15) | `br.ReadBits(32)` | `R(32)` | réf. objectif parent |
| `managed-objective-sub-objective-entities-component` (i16..i31) | `br.ReadBits(32)` | `R(32)` | réf. sous-objectif du slot |

### 5.3 Le patron sous-entités = celui des `rtpc` (ti=10) / `masked-property` (ti=13)

Les 16 slots ont le MÊME nom de composant. `consumeByName` reçoit le nom, pas l'index — comme
pour les 4 `rtpc` et les 32 `masked-property`. Le patron établi (`components_managed_object.go:
69-82,101-126` ; `components_managed_property.go:212-229`) : publier par un HOOK sans index, et
laisser l'APPELANT reconstruire le slot depuis l'ORDRE d'apparition. Champ produit :

```
// nouveau fichier components_managed_objective.go (patron default_state_ti42.go /
// components_managed_object.go : hook nommé par famille d'archétype, valeurs BRUTES)
type ManagedObjectiveField int
const (
    ManagedObjectiveObjectRef ManagedObjectiveField = iota // i3   R(32) GlobalID
    ManagedObjectiveType                                    // i5   R(32)
    ManagedObjectiveProgress                                // i12  R(32)
    ManagedObjectiveRequired                                // i13  R(32)
    ManagedObjectiveState                                   // i14  R(3)
    ManagedObjectiveParent                                  // i15  R(32) GlobalID
    ManagedObjectiveSubEntity                               // i16..i31 R(32) GlobalID (+ slot)
)
// hook(field, slot, value) ; slot = compteur d'occurrences pour SubEntity (0..15),
// -1 sinon. L'appelant tient LockProcessDecode (comme les autres hooks du paquet).
```

L'identité stable slot→zone est donc l'INDICE D'OCCURRENCE (0 pour la 1re sous-entité lue,
1 pour la 2e, …). Que le descripteur du jeu range à `0x19c + descIndex*4` ou à une permutation
n'a AUCUNE incidence sur le port : l'ordre de lecture est fixe (table i16→i31), donc
l'étiquette d'occurrence est stable image-par-image et film-par-film, ce qui suffit à une
identité de zone. (On ne compare jamais au tableau en RAM du jeu ; on compare entre images et
au pairing zone_states — voir §6.)

### 5.4 Précautions de câblage (garde-fous existants)

- `consumeCorruptionCheck` est appliqué APRÈS chaque composant par `traverseComponentLoopFrom`
  (`traverse.go:1250`) : ne PAS le rejouer dans les cases (le mot de contrôle par composant est
  déjà géré par la boucle).
- Retourner `ported = true` pour ces 7 cases (aucune est data-dépendante → jamais de désync ;
  cf. §2.3).
- `//nolint:unused` d'`consumeObjectiveFormattedText` (`components_batch3.go:22`) : sa condition
  de retrait est « quand ti=11 sera décodé » — à traiter dans le MÊME lot si le cadre passe, ou
  à laisser tel quel sinon (i2/i9 n'ont PAS d'adresse EXE établie, `ecs_table.tsv:270,277`).

---

## 6. VALIDATION sans build (réponse Q4) — protocole et seuils écrits AVANT mesure

Deux étages INDÉPENDANTS. Le premier gate le second : sans cadre, la sémantique est illisible.

### 6.1 Étage A — LE CADRE (préalable commun à tout ti=11)

Question : le corps d'image-clé (en-tête 108b, ordre de table, mots de contrôle) reproduit-il
sur le payload type-2 du film ? Instrument : l'oracle de frontières de records
`.ai/V7.5/dumps/kf_capture_sample.txt` (400 frontières, jamais consommé) + le harnais
`WalkKeyframeFullState`. ti=11 est le test le PLUS PROPRE du cadre parce que ses feuilles sont
triviales (le bipède ti=35 ne pouvait pas l'isoler : ses feuilles dérivent aussi).

**Protocole** : sur un film à objectif du corpus, dérouler `WalkKeyframeFullState` sur les
records ti=11 et exiger l'ATTERRISSAGE bit-exact — le bit de fin calculé (108b + bloc défaut +
34 feuilles, dont 16×32 = 512 bits de sous-entités) doit coïncider avec le début du record
suivant, comme le fait le chaînage validé pour ti=42 (`REGISTRE_REPORTS.md:209`, 282/289) et
ti=13 (`components_managed_property.go:187-188`, chaînage 87-99 %).

**Seuil écrit AVANT mesure** : chaînage de record ti=11 ≥ **85 %** (borne basse du témoin
ti=13, qui est le comparable le plus proche : même archétype managed, feuilles simples), témoin
d'ancrage faux (décalage de l'en-tête ou deser d'un AUTRE archétype passé sur le même flux)
≤ **10 %**. Sous ce seuil : le mur est dans le CADRE (contredit R7-e « le cadre n'est pas la
cause »), NO-GO pour TOUT ti=11 y compris i16-31, et on documente que le blocage est le format
type-2, pas les feuilles.

### 6.2 Étage B — LA SÉMANTIQUE (seulement si A passe)

**B1 — auto-cohérence interne (gratuit, pas d'oracle externe)** : les valeurs non nulles des
slots sub-objective de la racine sont un SOUS-ENSEMBLE des GlobalID ti=11 du même film
(§4.4). Seuil : ≥ **90 %** des slots non nuls matchent un GlobalID ti=11 présent ; témoin =
tirer 16 valeurs 32b au hasard → ≤ **1 %** de match (l'espace des GlobalID est creux). Ce test
distingue aussi enfant-objectif vs objet-physique direct.

**B2 — Bastion (3 zones connues, `zone_states` publié)** : sur un film Bastion/Strongholds, la
racine doit avoir **exactement 3** slots peuplés (H-A : 3 des 13-18 déclarées avec i14 actif ;
H-B : 3 tout court), et ces 3 sous-entités doivent APPARIER les 3 formes déjà publiées par
`zone_states.go` (par l'objet physique référencé i3 de chaque enfant, comparé à la position de
la forme). Seuils : cardinal = **3** sur ≥ **2** films ; appariement des 3 sous-entités aux 3
formes ≥ **90 %** (le chiffre tenu par Bastion aujourd'hui, `PLAN §2.3` seuil propriétaire) ;
témoin = formes décalées de 12 m (`defaultWitnessOffsetM`) → appariement ≤ **20 %**.

**B3 — Total Control (le débloquage)** : sur un film Total Control (corpus D1 : 4 films + 3
Fiesta, `REGISTRE_REPORTS.md:438`), la racine doit ISOLER les 3 zones ACTIVES de la manche
courante parmi 13-18. Seuils (identiques au gate §2.3 du plan, écrits d'avance) : cardinal des
zones actives par manche = **exactement 3** sur ≥ **2** films ; attribution des prises nommées
`zone_captures` à la forme de l'enfant apparié ≥ **80 %**, témoin décalé ≤ **20 %** ;
propriétaire (i14 state / i1 color de l'enfant vs équipe du capteur) ≥ **90 %**. Le tout SANS
toucher au balayage ti=13 : l'ancrage est le dispatch ti=11 par entité.

### 6.3 Ce qu'aucune de ces mesures ne fait

Aucune n'est exécutée ici (build interdit). Ce sont des protocoles à REMETTRE, avec seuils
gelés, pour la reprise porteuse.

---

## 7. VERDICT (réponse Q5)

### GO-CONDITIONNÉ — feuille RÉSOLUE, gate = le cadre 108b (le même que tout ti=11)

- **Grammaire** : RÉSOLUE et TRIVIALE. 16 × `R(32)` (`FUN_142ed5974`), byte-identique à i3
  object-reference, écrivain symétrique, borne à 16 prouvée par l'offset d'i32. i16-31 n'ajoute
  **aucun** risque de grammaire par-dessus i3/i5/i12/i13/i14 : c'est le même code de feuille.
- **Le SEUL risque résiduel est le CADRE** — jamais reproduit sur le film type-2 (0/34 dispatch
  ti=11 câblé ; plafond 0,51 % bit-exact sur ti=35, mais ce plafond est dans les feuilles du
  bipède, PAS dans le cadre, R7-e). L'étage A du §6 tranche à bon marché.
- **Ordre de reprise** : câbler les 7 feuilles ti=11 d'un coup (§5.2) — le coût est le cadre,
  une fois tenu chaque feuille ≈ 1 `case`. Mesurer l'étage A ; s'il passe, mesurer B1→B2→B3.

### CONTOURNE-T-IL l'ancrage ti=13 cassé de Total Control ? — OUI

L'ancrage ti=13 échoue par CONSTRUCTION : `scanPayload` balaie EXHAUSTIVEMENT une bande de
slots ti=13 dans l'image-clé avec une porte faible (2 bits) → jusqu'à 77 désignations
simultanées croissant avec le film (`PLAN §D3t`, `document.go:190-193`). Le chemin sous-entités
NE BALAIE RIEN : il lit l'objectif RACINE ti=11 par le dispatch d'état complet PAR ENTITÉ, et
la racine ÉNUMÈRE ses zones (≤ 16, borne DURE). C'est un ancrage STRUCTUREL (l'arbre
parent/enfant du jeu lui-même), pas géométrique ni statistique. Si l'étage A passe, c'est
exactement le contournement propre que Total Control attend, ET il nomme au passage le PORTEUR
Oddball (i3, le `[!]` de 5 campagnes) et fiabilise Bastion/KOTH (i14 state par zone).

### NO-GO explicites

- **NO-GO** si l'étage A échoue : le mur serait alors le CADRE/format type-2 (pas les feuilles),
  et AUCUNE feuille ti=11 — i16-31 comprise — n'est atteignable par le film. Ce que type-0
  exigerait alors : décoder l'écrivain du bloc type-2, qui **n'est pas dans `HaloInfinite.exe`**
  (recherche R6 bornée et négative, `REGISTRE_REPORTS.md:210`) — donc une voie par le CONTENU
  (chaîne de la table vs oracle Cheat Engine), plus coûteuse. Chiffrer le NO-GO = le taux de
  chaînage ti=11 mesuré à l'étage A contre le seuil 85 %.
- **NO-GO** sur toute idée de rejouer l'ancrage ti=13 pour Total Control : `[!]` établi sur 5
  sous-campagnes D3t, ne pas y revenir.
- **Hors périmètre** : i32 outro (`FUN_142ed5634`) est quantifié 8 bits, non trivial — ne PAS
  le classer trivial ; i1 color et i2/i9 formatted-text restent NON adressés (pas d'adresse EXE
  pour i2/i9).

---

## 8. Réserves portées par ce livrable

1. **Attribution i-index par NOM à l'exécution.** `FUN_142e2c690` dispatche par nom apparié, pas
   par index statique (`CATALOGUE §1.3`). L'étiquette « i16..i31 » repose sur `ecs_table.tsv:
   284-299` (assertion de la table canonique), CORROBORÉE par la concordance struct : les 16
   composants de même nom écrivent un tableau contigu `0x19c..0x1d8` borné par i32 `0x1dc` — ce
   qui n'a de sens QUE si ce sont bien les 16 slots sub-objective. La GRAMMAIRE et le CHAMP de
   destination sont vérifiés de première main ; l'étiquette d'index porte cette réserve.
2. **Permutation slot↔zone.** Le mapping descripteur→slot (`*(param_1+8)`) est une donnée de
   construction d'archétype, non lisible dans le code de la feuille. Sans incidence sur le port
   (§5.3) : l'ordre de lecture est fixe, l'étiquette d'occurrence est stable. La correspondance
   slot→zone A/B/C RÉELLE est établie EMPIRIQUEMENT par l'appariement B2/B3, pas supposée.
3. **Enfant-objectif vs objet-physique.** Le `R(32)` est un GlobalID ; le code ne dit pas s'il
   pointe une entité ti=11 enfant ou directement l'objet physique. Les deux sont exploitables ;
   le test B1 (§4.4) dit lequel. Ne pas trancher a priori.
4. **Le cadre n'est PAS mesuré ici** (build interdit). Tout le §6 est un protocole à remettre.
5. **Doc RE_LECTEUR** : présent dans le worktree `LevelUp-wt-re-lecteur` (le catalogue le
   croyait absent du dépôt principal — les deux constats coexistent, worktree ≠ dépôt principal).
   Toutes les feuilles neuves de ce livrable sont re-vérifiées de première main sous Ghidra,
   indépendamment de ce doc.

---

## 9. Texte prêt pour `.ai/thought_log.md` (NON écrit par cette passe)

```
### [2026-08-27] PISTE B — ti=11 i16-i31 sub-objective-entities : feuille RÉSOLUE, GO-CONDITIONNÉ

Statut : Complété (RE de feuille + conception, lecture seule, aucun port, aucun build)

Décision technique. Passe Ghidra LECTURE SEULE sur les 16 composants
managed-objective-sub-objective-entities-component (ti=11 i16..i31), jamais adressés. Chaîne
résolue : chaîne 143c954c0 -> getter 14064c7c0 (vtable[0x08]) -> vtable 143c97120 -> deser
vtable[0x30] = FUN_142ed5974. La feuille est un R(32) PUR (aucun callee, largeur immédiate
0x20, aucune porte/quantif/vec3), byte-identique à i3 object-reference FUN_142ed5550, seule la
destination diffère : dst+0x19c + (*(param_1+8))*4 (indexé par descripteur, comme rtpc ti=10 /
masked-property ti=13) contre dst+0x40 (fixe) pour i3. Écrivain 142edba24 symétrique (MOV R9D,
[RAX+RCX*4+0x19c] ; MOV ECX,0x20) => aucun désync possible. Borne du tableau PROUVÉE : i15
parent-objective FUN_142ed5674 -> dst+0x198, i32 outro FUN_142ed5634 -> dst+0x1dc, donc les 16
slots occupent 0x19c..0x1d8 (index local 0..15, pas une inférence). Chaque slot porte un
GlobalID d'entité ; l'arbre parent(i15)/enfants(i16-31) donne l'identité de sous-zone par
l'INDICE DE SLOT.

Résultats observés. Q1 : 16 R(32) indépendants (512 bits fixes/record), pas de tableau à compte
variable ; chaque slot = référence d'entité. Q2 : slot -> zone A/B/C par ordre ; cardinal peuplé
<= 16 (borne DURE, contre 77 désignations ti=13). Q3 : câbler 7 cases R(n) dans consumeByName +
hook nommé (patron components_managed_object.go), slot = indice d'occurrence. Q4 : étage A cadre
(oracle kf_capture_sample.txt, chaînage ti=11 >= 85 %, témoin <= 10 %) gate étage B sémantique
(B1 auto-cohérence GlobalID >= 90 % ; B2 Bastion cardinal=3 + appariement 3 formes >= 90 %,
témoin décalé <= 20 % ; B3 Total Control 3 actives, attribution >= 80 %). Q5 : GO-CONDITIONNÉ,
gate = cadre 108b (même que tout ti=11 ; risque hors des feuilles). CONTOURNE l'ancrage ti=13
cassé de Total Control : ancrage STRUCTUREL par entité, aucun balayage de bande.

Conclusion / prochaine étape. Feuille acquise et adressée ; le gain reste conditionné au cadre
d'image-clé 108b. Reprise = câbler les 7 feuilles ti=11 d'un coup et mesurer l'étage A ; s'il
passe, B1->B2->B3. NO-GO si A échoue (mur = format type-2, écrivain absent de l'EXE).
```

## 10. Ligne prête pour `.ai/V7.5/REGISTRE_REPORTS.md` (NON écrite par cette passe)

```
[2026-08-27] ti=11 i16-i31 managed-objective-sub-objective-entities (16 slots) — feuille RÉSOLUE
  en Ghidra (déser FUN_142ed5974 = R(32) pur, byte-identique à i3 object-reference FUN_142ed5550,
  dst+0x19c+index*4 ; écrivain 142edba24 symétrique R(32) ; borne 16 slots PROUVÉE par i15
  parent-objective +0x198 et i32 outro +0x1dc). Chaque slot = GlobalID d'entité ; arbre
  parent/enfants = identité de sous-zone A/B/C par l'indice de slot, cardinal peuplé <= 16 (borne
  DURE contre les 77 désignations ti=13). C'est le CONTOURNEMENT STRUCTUREL de l'ancrage ti=13
  cassé de Total Control (aucun balayage de bande). STATUT : GO-CONDITIONNÉ. CONDITION DE REPRISE
  = câbler les 7 feuilles ti=11 (i3/i5/i12/i13/i14/i15/i16-31, toutes R(n) triviales, adresses
  prêtes) dans consumeByName + hook nommé, puis MESURER l'étage A (cadre 108b, chaînage ti=11
  >= 85 % sur kf_capture_sample.txt) ; s'il passe, étage B sémantique (B1 auto-cohérence GlobalID,
  B2 Bastion 3 formes, B3 Total Control 3 actives, seuils gelés au doc). NO-GO si A échoue : mur =
  format type-2, écrivain absent de HaloInfinite.exe. Doc : scratchpad PISTE_B_ti11_sous_entites.md.
```

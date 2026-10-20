# Relevés Ghidra du lot J6.1 — lecteur `FUN_14076e524`, `DAT_144632be0`, `FUN_140ce59bc`

- Date des relevés : **2026-09-27** (s'applique à chaque valeur citée ci-dessous, règle D-3 n° 1).
- Programme : `HaloInfinite.exe` (base d'image 0x140000000). Projet Ghidra créé le 2026-06-04, le
  même que pour les relevés du 2026-09-16 (`profile/loi_largeurs.go`).
- Accès : API REST de Ghidra en lecture seule (`get_xrefs_to`, `decompile_function`,
  `disassemble_function`, `read_memory`, `inspect_memory_content`, `get_function_by_address`).
  Rien n'a été écrit dans Ghidra.
- Code Go comparé : `LevelUp-wt-suite-audit-j5` à `9f1563ff2`. La reprise M4b `3cca6cf47` en fait
  partie (contrôle fait avec `git merge-base --is-ancestor`).
- Méthode pour l'immédiat de niveau : lecture des octets qui précèdent chaque `CALL`
  (`41 B9 imm32` = `MOV R9D,imm`, `41 B8 imm32` = `MOV R8D,imm`, `45 8D 41 xx` =
  `LEA R8D,[R9+xx]` après `45 33 C9` = `XOR R9D,R9D`, etc.), recoupée par la décompilation pour les
  cas où la valeur passe par un registre.
- Deux enveloppes transmettent le niveau :
  - `FUN_14076e494` : `MOV R9D,R8D` en `14076e49a`, donc le niveau est son 3e argument ;
  - `FUN_14076e4ec` : `MOV R9D,R8D` en `14076e505`.

  `FUN_1424e0e38` (thunk vers `e494`) et `FUN_14076e420` (`MOV ESI,R8D` en `14076e43c`, puis
  `MOV R8D,ESI` en `14076e45a`) le transmettent tel quel. Les sites indirects sont donc relevés
  eux aussi.

## 1. Sites d'appel et immédiat de niveau

Légende de l'écart : **OK** = le portage Go lit ce que lit le jeu. **ÉCART** = divergence de
largeur ou de grammaire. **non porté** = le Go ne décode pas ce site en production. **à
rapprocher** = portage mesuré dont la correspondance au bit près n'a pas été établie ici.

### 1.a Appels directs de `FUN_14076e524` (24 xrefs : 22 appels + 2 DATA `1453f6be8`, `143ed91b8` = .pdata / table)

| Appelant | CALL | Niveau (jeu) | Autres arguments constants | Composant (jeu) | Portage Go actuel | Niveau / largeurs Go | Écart |
|---|---|---|---|---|---|---|---|
| `FUN_14076e494` | `14076e4c0` | transmis (R8D) | — | enveloppe : `f91c` → R(96), sinon `e524`, sinon `FUN_141f85880` si param_6 ≠ 0 | `consumeSimStateHandleTail` (niveau 16 figé) | 16 | voir 1.b |
| `FUN_14076e4ec` | `14076e516` | transmis (R8D) | — | enveloppe identique (param_6 dans R9) | — | — | voir 1.b |
| `FUN_1406cfe44` | `1406d009d` | **0x10** (`MOV R9D,0x10` en `1406d008a`) | garde `f91c` en ligne (`1406d0076`) | i0 `object-position-dynamic-precision` du bipède, branche absolue | `consumeAbsoluteWithGate` / `consumeAbsolutePayload` | idx = `IndexW` ; axes = `absAxisWFor` (défaut L16 ou carte) | **OK** pour la charge ; **ÉCART** sur precHigh=1 (voir 2.c) |
| `FUN_14076f3ec` | `14226a6c7` | **0x10** (`14226a6b8`) | — | repli absolu du delta prédit (bit à 1 → `e524` nu) | `consumePredictedDelta` → `consumeAbsoluteWithGate` | precHigh + `f91c` + charge + R(2) | **ÉCART probable** : le jeu lit la porte, l'index et les 3 axes, sans bit precHigh ni R(2) dans `f3ec`. Son R(2) (`FUN_14076e304`) est lu en fin de `FUN_1406cfe44` (`LAB_1406cffd7`) pour tous les chemins. Budget à vérifier en J6.2 |
| `FUN_1408f02c8` | `1408f03c7` | **0x10** (`MOV R9D,EBP`, avec `MOV EBP,0x10` dans le prologue, octets `bd 10 00 00 00`) | — | i54 `biped-mobility-action`, 1re position | `consumeSimStateHandleTail` | 16 | **OK** (reprise M4b) |
| `FUN_140ee7270` | `140ee7293` | **0x10** (`140ee7288`) | garde `f91c` | ti=21 i16 `flock-position-component` | `consumeFlockPosition` → `consumeQuantVec3WithGate(quantAxisWidth(level))` | idx R(1) figé ; axes 6 + niveau du registre (6 bits au niveau 0) | **ÉCART GA2-2 confirmé** : le jeu lit l'index sur `DAT_144632be0` bits, puis 22/22/22 (idx = -1) ou les largeurs de la carte au L16 (idx ≥ 0). Sous-lecture de 3×(22−6) = 48 bits ou de 40−18 = 22 bits (Cliffhanger), c'est-à-dire les « 22 à 48 bits » de l'audit |
| `FUN_140f04d88` (appelé par `FUN_140f04d74`) | `140f04de0` | **0x10** (`140f04dd5`) | garde `f91c` | ti=34 i7 `tacmap-waypointstate` : R(1) + `waypoint-lockedto` R(32) + pos + [R(1) si param_4 > 1] | `consumeE524PositionBody` (`dispatch_item.go:108-111`) | largeurs `traversal()` (descripteur du DELTA) | **ÉCART** : même défaut que GA2-1. Le R(1) final (param_4 > 1) n'est pas lu |
| `FUN_140f04f18` | `140f04f3d` | **0x10** (`140f04f32`) | garde `f91c` | événement joueur 0xe9, cas 7 (palette) | tests de recherche seulement | — | **non porté** |
| `FUN_140f04f68` | `140f04f8b` | **0x10** (`140f04f80`) | garde `f91c` | composant du descripteur `143d08268` (+0x28), écrit `dst+0x714` ; nom non résolu | — | — | **non porté** |
| `FUN_140f04fb8` | `140f04ff0`, `140f05023` | **0x10** ×2 (`140f04fe5`, `140f05018`) | garde `f91c` | `EquipmentTranslocatorTeleportEffects`, positions A et B | `readTranslocVec` (`transloc_events.go:207-230`) | défaut 22 (constante `translocDefaultAxisBits`) ; index = `EffectiveRegionIndexBits` ; carte = `AxisWidths` | **OK en valeur** (22 = loi à L16), mais c'est un second portage : constantes recopiées, pas de garde `f91c`. Oracle 18/18 |
| `FUN_140fb8af0` | `140fb8b3e` | **0x10** (`140fb8b33`) | garde `f91c` | tableau d'éléments de 0x14 o : R(1) + pos + [`FUN_1424e268c` si param_4 > 1] ; descripteur `143c96c50` (+0x28) ; nom non résolu | — | — | **non porté** |
| `FUN_1408096f8` | `140809783` | **0x0F** | — | charge `projectile_detonate` | tests de recherche | — | **non porté** |
| `FUN_1410f03b4` | `1410f045b` | **0x0C** | — | charge `projectile_impact_effect` | tests de recherche | — | **non porté** |
| `FUN_14112134c` | `141121387` | **0x0C** (`LEA R9D,[RDI+0xc]`, EDI = 0) | — | charge `ObjectCollisionDamage` | tests de recherche | — | **non porté** |
| `FUN_141dc8600`, `FUN_141dcc4a0`, `FUN_141dcf120`, `FUN_141dcfbc0` (×2), `FUN_141dd86b0`, `FUN_141dda340`, `FUN_141ddae80` | `141dc8879`, `141dcca2e`, `141dcf619`, `141dd00a8`, `141dd0143`, `141dd8719`, `141dda837`, `141ddb377` | **0x1E** ×8 | garde `f91c` en ligne (`DAT_144e61ea0` / `DAT_145121140`) | union étiquetée (tags 0..0x5c, R(16) + R(32) puis position), appelée par `FUN_141df06c0` / `FUN_141df1130` ; hors film probable | — | — | **non porté** (hors périmètre probable) |

### 1.b Sites indirects : le niveau passe par `FUN_14076e494` (3e argument)

53 xrefs d'appel, plus une DATA (`1453f6bd0`). Chaque fois, `param_6` = 0, posé par
`AND qword [RSP+0x28],0` : c'est donc le chemin `e524`, sauf mention contraire.

| Appelant (CALL) | Niveau | Composant (jeu) | Portage Go | Écart |
|---|---|---|---|---|
| `FUN_1424e0e38` (`1424e0e47`) | transmis | thunk (voir 1.c) | — | — |
| `FUN_14076e420` (`14076e470`) | transmis | precHigh R(1), puis `e494` avec param_6 = `&DAT_143b8c6d0` si precHigh = 1 (voir 1.d) | — | — |
| `FUN_14058c058` (`1422cddc1`, `1422cde0e`) | **0x10** ×2 (décompilé : `FUN_14076e494(param_2,…,0x10,0,param_3,0)`) | `unit-control`, boucle des emplacements de visée | `consumeQuat16` = R(16) (`unit_control.go:165,170,194`) | **ÉCART** : l'immédiat 0x10 a été lu comme une largeur de 16 bits. Le jeu lit la garde, la porte, l'index et les 3 axes au L16 |
| `FUN_1431a0cbc` (`1431a0d0d`) | 0x10 | vecteur du bloc d'action | `bloc_action.go:143` `consumeSimStateHandleTail` | OK |
| `FUN_14080c1f8` (`14080cb06`) | 0x10 | record de tir (désérialiseur des tirs) | `fire_events.go` / `fire_aim_modal.go` | à rapprocher (non ouvert ici) |
| `FUN_1408efb58` (`1408efe11`) | 0x10 | état par défaut ti=41 | `default_state_ti41.go:65` `consumeSimStateHandleTail` | OK |
| `FUN_142f04664` (`142f0482b`) | 0x10 | sous-lecteur de l'état ti=41 | `consume142f04664` → `consumeSimStateHandleTail` (`default_state_ti41.go:119`) | OK |
| `FUN_1408f02c8` (`1408f0758`) | **0x10** (`MOV R8D,EBP`, EBP = 0x10) | i54, 2e position | `components_biped_ability.go:297` | OK (reprise M4b) |
| `FUN_140f44c38` (`142451b5d`) | 0x10 | état par défaut du bipède, branche « trame média » | `consumeBipedDefaultStateMediaFrame` (`unit_control.go:~258`) : porte + R(1) seulement | **ÉCART** (site GA2-3) : ni garde `f91c`, ni les 3 axes, et index figé à 1 |
| `FUN_1410a5a74` (`1424a3a1e`) | 0x10 | état par défaut ti=40 | `default_state_ti40.go:87` | OK |
| `FUN_141454340` (`14145437e`) | 0x10 (param_5 = 1) | `selectable-zone-data` (appelé par `FUN_142ed6cec`) | `consumeSelectableZoneData` (sans appelant) → `consumeE524PositionBody` | **ÉCART** (inerte aujourd'hui) |
| `FUN_142f25a3c`, `FUN_142f263ac`, `FUN_142f264f4` (`142f25d46`, `142f263d9` [`LEA R8D,[RBX+0x10]`, EBX = 0], `142f26586`) | 0x10 | posture, étiquettes 1..3 | `components_biped_posture.go:171,189,214` | OK |
| `FUN_142f262d4` (`142f2638b`) | 0x10 | i57, étiquette 3 | `components_biped_spartan.go:421` | OK |
| `FUN_142ed6d88` (`142ed6fd5`) | 0x10 | i60 `simulation-state` | `traverse.go:243` | OK |
| `FUN_142f25e90` (`142f2605d`) | 0x10 (`LEA R8D,[RBP+0x11]`, décompilé : 0x10) | i59, ancre du grappin (une branche d'étiquette) | `components_biped_anchor.go:152` : 3 axes aux largeurs de carte, sans porte d'index explicite (« R(3) drapeaux ») | à rapprocher (J6.2) |
| `FUN_142f036f0` (`142f03837`) | **0x10** (param_5 = `*(param_3+0x38)`) | **ti=38 i18** `generic-rigid-body-transforms` : R(8) masque, puis par bit `FUN_140c1e79c` + `e494(0x10)` | `consumeGenericRigidBodyTransforms` → `consumeE524PositionBody` (`components_world.go:170`) | **ÉCART** (site laissé par M4b) : largeurs `traversal()`, pas de garde `f91c` |
| `FUN_142b6eeec` (`142b6ef31`) | 0x10 (param_5 = 1) | `spawn-filter-type`, étiquette 3 (appelé par `FUN_142ecf744`) | `consumeSpawnFilterType` → `consumeE524PositionBody` | **ÉCART** |
| `FUN_142ed9120` (`142ed918e`) | 0x10 | ti=14 i0 `crew-order` (appelé par `FUN_142ed4274`) : `FUN_142b1cf3c` + R(1) + `e494` | `dispatch_player.go:102` `consumeQuantVec3(6+niveau)` | **ÉCART** : un bit precHigh de trop, largeurs 6 + niveau du registre au lieu de L16, index figé à 1 |
| `FUN_142ed9530` (`142ed9556`) ×5 (depuis `FUN_142ed3c64`) | **0x1E** | ti=44 i0 `asset-transform-component` | `dispatch_biped.go:250` `consumeQuantVec3WithGate(quantAxisWidth(level))` | **ÉCART MAJEUR** : au niveau 30, les deux tables donnent **26/26/26** (pas(30) < 1e-4). Le Go lit 7/7/7 (niveau 1) et un index à 1 bit |
| `FUN_142a9ced8` (`142a9cf08`), `FUN_142a9cf8c` (`142a9cfb2`), `FUN_142f0391c` (`142f0395c`) | 0x1E | non rattachés | — | non porté (ou non identifié) |
| `FUN_142ef4344` (`142ef4374`), `FUN_142ef8e08` (`142ef8e87`), `FUN_142f17eec` (`142f17fde`, `MOV R8D,0xc`) | 0x0C | non rattachés | — | non porté (ou non identifié) |
| `FUN_142f1c0b0` (`142f1c219`) | 0x0D | non rattaché | — | non porté (ou non identifié) |
| `FUN_142add1b0`, `FUN_1427e39a4` (×2), `FUN_1427e3ba4`, `FUN_142ed8c44` (`LEA ESI,[R9+0x10]`), `FUN_142eefa5c` (×3), `FUN_142ef15e0`, `FUN_142ef42b8`, `FUN_142ef9470` (×2), `FUN_142f03ec8`, `FUN_142f15c9c`, `FUN_142f15cf8` (×2), `FUN_142f15ec0`, `FUN_142f160c4`, `FUN_142f163e0` (`MOV R8D,0x10`, R9B = 1), `FUN_142f1699c`, `FUN_142f16ad8`, `FUN_142f16cac`, `FUN_142f17348`, `FUN_142f17500` | 0x10 | non rattachés (pas de mention Go, sauf `FUN_142f17500` dans un test de recherche) | — | non porté (ou non identifié) |

### 1.c Via `FUN_1424e0e38` (niveau en R8D, 14 appels)

| Appelant (CALL) | Niveau | Composant | Portage Go | Écart |
|---|---|---|---|---|
| `FUN_142ed7764` (`142ed7853`), appelé par `FUN_142ed3c50` | 0x10 | `tacmap-areaofinterest` | `consumeE524PositionBody` | **ÉCART** |
| `FUN_142ed4198` (`142ed41ba`) | 0x10 | `tacmap-cooptetherarea` | `consumeE524PositionBody` | **ÉCART** |
| `FUN_142ed7d38` (`142ed7edf`), appelé par `FUN_142ed433c` | 0x10 | `tacmap-displayasset` | `consumeE524PositionBody` | **ÉCART** |
| `FUN_142ed8418` (`142ed86d7`) | 0x10 | ti=30 i0 `tacmap-poiicon` | `consumeQuantVec3(6+niveau)` (`dispatch_player.go:117`) | **ÉCART** : bit precHigh de trop (le thunk passe param_6 = 0) et largeurs 6 + niveau du registre |
| `FUN_143203158` (`143203182`) | 0x0C | charge (test de recherche r7 lot 4) | — | non porté |
| `FUN_142ed485c`, `FUN_142ed4aec` (×2, `LEA R13D,[R9+0x10]`), `FUN_142ed6cc8`, `FUN_142ed7f64` (R9B = 1), `FUN_142ef7fdc`, `FUN_142ef9284`, `FUN_142ef93e0` (×2) | 0x10 | non rattachés (candidats possibles, non vérifiés : `tacmap-poiiconoffset`, `flock-destination`, `player-desired-respawn-location`) | — | à relever |

### 1.d Via `FUN_14076e420` (precHigh, puis `e494`) et `FUN_14076e4ec`

| Appelant (CALL) | Niveau | Composant | Portage Go | Écart |
|---|---|---|---|---|
| `FUN_14076e29c` (`14076e2c0`) | 0x10 | world-object i0 `object-position` (ti=36..43) | `dispatch_object.go:158-163` | **ÉCART GA2-5 confirmé** (voir §5) |
| `FUN_141fdae44` (`141fdb0f7`) | **0x10 ou 0x1E** : `(-(DAT_145121140 == 1) & 0xe) + 0x10`, octets `451bc0 4183e00e 4183c010` | bloc de 0xbc de la vue C | non porté (`ArretVueCBlocBC`) | seul niveau NON constant du relevé |
| `FUN_142ed92c4` (`142ed93e9`, `142ed93fe`) | 0x10 ×2 (R9B = 1) | non rattaché | — | à relever |
| `FUN_140f7ea14` (`140f7ea5c`, via `e4ec`) | 0x10 (`MOV R8D,0x10`) | i0 bipède, branche predFlag = 1 | `consumePredictedAbsolute` | OK |
| `FUN_142ed9578` (`142ed95ba`, via `e4ec`) | 0x10 | non rattaché | — | à relever |

**Bilan du tableau.**

- Niveaux relevés : 0x10 (très majoritaire), 0x1E, 0x0C, 0x0D, 0x0F, et 0x10/0x1E conditionnel
  pour un seul site (`FUN_141fdae44`).
- **Aucun site ne transmet le niveau du registre de `chunk_00` au lecteur.** Dans le jeu, ce
  niveau arrive aux désérialiseurs comme `param_4`, mais il n'y sert qu'à des bits de queue
  (`if (1 < param_4)` dans `FUN_140f04d88` et `FUN_140fb8af0`).
- L'hypothèse Go « largeur = 6 + niveau du registre » (`quantAxisWidth`), déjà notée PISTE dans
  `components_position_i0.go:376-384`, est **infirmée** pour les quatre sites vérifiés qui
  l'emploient : `flock-position`, `asset-transform`, `crew-order` et `tacmap-poiicon`.

## 2. Ce que fait `FUN_14076e524(dst, lecteur, indexOut, LEVEL)`

### 2.a La fonction elle-même

Décompilation et désassemblage, `14076e524`–`14076e6xx` :

```c
lVar12 = (longlong)param_4;                 // MOVSXD R14,R9D  (14076e546) : LEVEL
cVar6 = FUN_1406cf008(param_2);             // R(1) porte
uVar7 = 0xffffffff;
if (cVar6 == '\0') {                        // porte a 0 -> lit l'index
  uVar7 = R(DAT_144632be0);                 // MOV R10,[0x144632be0] (14076e56c)
  if (uVar7 != 0xffffffff) {
    pfVar13 = &DAT_14462cbe0 + idx*3;       // bornes de la plage idx (0x18 o par plage)
    widths  = DAT_1445ccbe0 + (idx*0x20 + LEVEL)*0xc;   // table PAR INDEX
    goto LAB_14076e5f3;
  }
}
pfVar13 = &DAT_1445cc9c8;                   // bornes du BUILD (copie de DAT_143b8c6b8 = +/-20000)
widths  = DAT_1445cc9e0 + LEVEL*0xc;        // table DEFAUT
LAB_14076e5f3: FUN_140cc5128(lecteur, ..., widths);  // 3 axes aux largeurs choisies
// dequant : (max-min)/2^w * q + min + pas*DAT_143cd84b0
```

- Le niveau ne choisit qu'une **ligne** de largeurs, dans l'une des deux tables : défaut, ou par
  index de plage (32 niveaux par plage, 0x180 o par plage). Le seul autre choix, défaut ou plage,
  est fait par la porte et l'index lus dans le flux.
- La largeur de l'index vaut `DAT_144632be0`. Elle ne dépend pas du niveau.
- `FUN_14076e524` ne lit ni garde `f91c`, ni bit precHigh, ni R(2) de queue : ce sont ses
  appelants qui les lisent.

### 2.b Les deux tables sont remplies par la loi (rappel de `loi_largeurs.go`, relue sur pièces le 2026-09-27)

`FUN_140de9a1c` copie les bornes du build en `DAT_1445cc9c8` et vide les deux tables ;
`FUN_140be9a14` les remplit ensuite par `FUN_140be9b88(niveau, bornes)`. Valeurs qui en
découlent :

- table défaut (±20000) : 6 + L pour L ≤ 16 ; 22 pour L de 17 à 22 (plafond de comptage 2^22) ;
  26 pour L ≥ 23 (pas < 1e-4). Donc **L16 = 22/22/22** et **L30 = 26/26/26** ;
- table par index : la même loi sur les bornes de la plage de la carte. À L16, ce sont les
  largeurs du catalogue (`axisWidths`). À L ≥ 23, c'est 26 pour tous les axes, quelles que soient
  les bornes.

`quantAxisWidth` (`min(26, 6+L)`) reproduit donc la table défaut pour L ≤ 16 seulement, et **elle
est fausse pour L entre 17 et 22**.

### 2.c Les enveloppes, et un défaut que ce relevé met au jour

- `FUN_14076e494(lecteur, dst, LEVEL, p4, p5, p6)` : si `FUN_14076f91c()` (garde d'exécution,
  `DAT_144e61ea0 != 0 || DAT_145121140 == 1`, 0 bit) est vraie → `FUN_1411b259c` =
  `FUN_1406d676c(…, 0x60)` = R(96). Sinon, si `p6 == 0` → `e524(LEVEL)`, et sinon
  `FUN_141f85880(dst, lecteur, p6, LEVEL)`.
- `FUN_14076e420(lecteur, dst, LEVEL, p4)` : lit R(1) precHigh, puis `e494(..., LEVEL, p4, 0,
  precHigh ? &DAT_143b8c6d0 : 0)`.
- **`FUN_141f85880` n'est PAS « 0 bit ».** Décompilation :

  ```c
  FUN_140be9b88(param_4 /*LEVEL*/, param_3 /*bornes*/, …, local_38);
  FUN_1424cbed4(…) -> FUN_140cc5128(...)
  ```

  Elle calcule par la loi les largeurs du niveau sur les bornes passées, puis lit 3 axes. Les
  bornes `DAT_143b8c6d0` valent **±100 ×3** (`read_memory` : `0000c8c2 0000c842` ×3, et
  `DAT_143b8c6b8` = ±20000). Au niveau 16, cela donne 3 × 14 = **42 bits**.
  - Le Go dit le contraire à deux endroits : `consumeAbsoluteWithGate` (precHigh = 1 →
    `return`, 0 bit) et `consumeQuantVec3Values` (precHigh = 1 → 0 bit).
  - Le chemin world-object mesuré (precHigh = 1 → R(59), `dispatch_object.go`) est cohérent avec
    42 bits + queue de poignée + R(2), mais cette décomposition n'a pas été vérifiée au bit près.

### 2.d `consumeSimStateHandleTail` est-il un portage fidèle ?

**Oui, pour `FUN_14076e494(…, 0x10, …, p6 = 0)` au niveau 16.** Il lit :

- la garde `f91c` → R(96) ;
- sinon la porte, puis l'index sur `worldObjectPrecision().IndexW` (= `DAT_144632be0` de la
  carte) ;
- puis `absAxisWFor(br, idx, i)` : ligne L16 de la table défaut (22) si idx = -1, ligne L16 de la
  carte si idx ≥ 0.

**Il n'est pas encore paramétrable par le niveau.** `absAxisWFor` fige
`profile.NiveauPositionDObjet` et `worldObjectPrecision().AxisW`, c'est-à-dire les largeurs du
catalogue au L16. Pour porter l'immédiat, il faut :

- **idx = -1** : `profile.LargeursAxeParDefautDuBuild(L)`, qui existe déjà ;
- **idx ≥ 0** : `profile.LargeursAxeDuNiveau(bornesDeLaPlage, L)`, qui existe déjà aussi. Il
  faut en plus les bornes de la plage, qu'on a pour la plage cataloguée (`MapQuantEntry.Range()`).
  Pour L ≥ 23, la valeur est 26 sans bornes ;
- **déquantification** : les mêmes bornes (`dequantWorldAxis`), déjà alignées sur l'index.

La même lecture doit avoir trois points d'entrée fins :

- `e524(L)` : porte, index, 3 axes ;
- `e494(L)` : garde `f91c` + `e524`, ou `FUN_141f85880` si p6 ;
- `e420(L)` : R(1) precHigh + `e494`.

Aujourd'hui, ces formes sont éparpillées dans plusieurs portages concurrents :
`consumeAbsoluteWithGate`, `consumeAbsolutePayload`, `consumeQuantVec3`,
`consumeQuantVec3WithGate`, `consumeE524PositionBody`, `consumeQuat16`, `readTranslocVec`, la
branche en ligne de `dispatch_object.go` et `consumeBipedDefaultStateMediaFrame`.

## 3. `DAT_144632be0` — la largeur de l'index de plage

- **Valeur dans l'image** : `read_memory(0x144632be0, 8)` = `00 00 00 00 01 00 00 00`, donc
  **0**. La valeur utile est posée à l'exécution.
- **Écrivains** (xrefs, 5 écritures) :
  - `FUN_140365eb0` (`140365f3a` / `140365f45`), initialiseur statique référencé en
    `14366d020` : copie la table des plages, vide `DAT_1445ccbe0` (0x60000 o), puis
    `DAT_144632be0 = ceilLog2(0x400)` = **10** (`BSR`, `8905a0cc2c04`), ou 0 si le BSR
    échoue ;
  - `FUN_140de9a1c` (`140de9a9a`), réinitialisation appelée par `FUN_140aa3814` : bornes du build
    → `DAT_1445cc9c8`, vide les tables, puis `DAT_144632be0 = FUN_1406d310c(0x400)` = **10** ;
  - `FUN_140be9a14` (`140be9b24` et `1423c6b40`), remplisseur de fin de chargement de carte :
    `CMP ECX,0x1` (`140be9b16`). Si la carte déclare UNE plage, le saut mène à
    `1423c6b40 : MOV dword [DAT_144632be0],1` (`c7 05 … 01 00 00 00`) ; sinon
    `DAT_144632be0 = ceilLog2(compte)` (`140be9b24 : MOV [DAT],EAX`).
- **Lecteurs** :
  - `FUN_14076e524` (`14076e56c`, `14076e58d`, `14076e59a`) ;
  - `FUN_1407eb6a8` (`1407eb6d5`) : écrivain symétrique, R(1) (index == -1) puis l'index sur
    `DAT_144632be0` bits, appelé par `FUN_1407eb61c` et `FUN_141dccaa0` ;
  - les six écrivains de la famille 0x1E (`FUN_141dc8a70`, `FUN_141dcf690`, `FUN_141dd0210`,
    `FUN_141dd8870`, `FUN_141dda8f0`, `FUN_141ddb460`).
- **Ce qu'elle gouverne** : seulement la largeur de l'index de plage lu par `e524` et écrit par
  `FUN_1407eb6a8`. Elle ne dépend pas du niveau.
- **Verdict GA2-3.** Ce n'est **pas une constante**. Elle vaut 10 hors carte, **1** quand la
  carte déclare une plage (78 cartes du catalogue sur 79), et **ceilLog2(compte brut)** sinon
  (Live Fire : 4 plages déclarées, donc 2). Les sites qui câblent 1 sont justes sur 78 cartes et
  faux sur Live Fire :
  - `components_position_i0.go:218` et `:418` ;
  - `default_state.go:306` ;
  - `unit_control.go` (`consumeBipedDefaultStateMediaFrame`).

  Il faut **la lire du profil** : `worldObjectPrecision().IndexW`, déjà calculée par
  `profile.LargeurIndexDePlage`, comme le fait `consumeSimStateHandleTail`.

## 4. `FUN_140ce59bc` et son « jumeau » (GA2-4)

- **Dans le jeu, une seule fonction et un seul appel de variant.** `FUN_140ce59bc(lecteur, _,
  ctx)` lit **R(4)** d'étiquette, puis appelle **toujours** `FUN_140ce5aa4(tag, ctx)` (appel en
  `140ce5a09`).
  - `FUN_140ce5aa4` lit une charge qui dépend de l'étiquette et de `ctx->index`
    (`*(uint*)(param_2+2)`, donc offset 0x10). Avec `index >= 0x20`, les étiquettes 1, 2, 5 et 6
    lisent (R(8), R(1), R(32) `string-id-value`, R(32)) et les étiquettes 3 et 4 délèguent à
    `FUN_140ce558c` / `FUN_140ce5720`. Avec `index < 0x20`, c'est le miroir : le mode B.
  - Il n'y a pas de fonction jumelle dans l'exécutable. Les appelants de `FUN_140ce59bc` sont
    `FUN_140ce5554` (`140ce557e`), `FUN_140ce55e8` (`140ce5672`, `140ce569f`) et `FUN_140ce593c`
    (`140ce598a`). Le seul appelant de `FUN_140ce5aa4` est `FUN_140ce59bc`.
- **Les deux portages Go de cette même fonction :**
  - `default_state_arch.go:176-190` (`consumeDefaultStateTI13`) lit `R(4)` seul, pour 1 ou
    32 variants ;
  - `components_managed_property.go:147-155` (`consumeManagedPropertyVariant`) lit l'étiquette,
    puis `managedPropertyPayloadBits(tag, modeA)`. C'est le « jumeau » de l'audit.
- **Ce que lit l'état par défaut ti=13** (`FUN_140ce55e8`, désassemblage `140ce565b`–`140ce56a9`) :

  ```
  R(1)[R(8)]  (préfixe de version) ; "propertyName" R(32) ; g = R(1)
  g == 0 : OR dword [RSP+0x30],0xffffffff ; LEA R8,[RSP+0x20] ; CALL 140ce59bc   -> index = -1 : MODE A
  g == 1 : MOV dword [RSP+0x30],EDI ; CALL 140ce59bc ; INC EDI ; CMP EDI,0x20      -> index = 0..31 : MODE B
  ```

- **Verdict GA2-4 : le jumeau `consumeManagedPropertyVariant` est le bon.**
  `consumeDefaultStateTI13` doit l'appeler en mode A pour la variante unique et en mode B pour les
  32. Le mode A du composant i1 est confirmé par `FUN_140ce5554` (`local_18 = 0xffffffff`). Le
  R(4) seul de `default_state_arch.go` sous-lit chaque variant dont l'étiquette porte une charge.

## 5. GA2-5 — largeurs de carte avec `idx = -1` (`dispatch_object.go:158-163`)

- Chaîne du jeu : `FUN_14076e29c` → `FUN_14076e420(lecteur, dst+4, 0x10)` (CALL `14076e2c0`,
  `LEA R8D,[R9+0x10]`) → `e494(0x10)` → `e524(0x10)`.
- Quand la porte vaut 1 (index = -1), les largeurs sont celles de la **table défaut au L16**,
  `DAT_1445cc9e0 + 0x10*0xc`, soit **22/22/22**, avec les bornes ±20000. Ce ne sont jamais celles
  de la carte.
- Le Go lit `worldObjectPrecision().AxisW[a]` quel que soit l'index : c'est faux pour idx = -1.
  La correction attendue est exactement `absAxisWFor(br, idx, a)`, ce que le plan appelle
  « aligné sur `absAxisWFor` ». Elle est confirmée.
- Deux remarques sur le même site, à traiter avec le portage unique :
  - (i) la garde `f91c` (R(96)) n'est pas modélisée dans la branche en ligne ;
  - (ii) precHigh = 1 passe par `FUN_141f85880` : 3 axes à la loi L16 sur ±100, soit 42 bits ;
    voir 2.c. Le R(59) mesuré y est cohérent.
- Le reste de `FUN_14076e29c` (`FUN_14076e3e4` queue de poignée, puis `FUN_140492128` →
  `FUN_14076e304` R(2)) est hors de ce relevé.

## 6. Conclusions pour J6.3

1. **Un seul portage paramétré par le niveau, qui part de `consumeSimStateHandleTail` :
   JUSTIFIÉ.**
   - `consumeSimStateHandleTail` est fidèle à `e494(0x10, p6 = 0)`.
   - Le niveau est toujours un immédiat du site. Le seul cas conditionnel est
     `FUN_141fdae44`, qui n'est pas porté.
   - `e524` ne dépend du niveau que par la ligne de table choisie.

   La paramétrisation demande :
   - de remplacer, dans `absAxisWFor`, `NiveauPositionDObjet` par le niveau du site, et
     `worldObjectPrecision().AxisW` par `LargeursAxeDuNiveau(bornesPlage, L)` (égal au catalogue
     à L16) ;
   - d'exposer trois entrées : `e524`, `e494` (garde `f91c`) et `e420` (precHigh) ;
   - de porter `FUN_141f85880` (loi sur ±100 au même niveau) au lieu du « 0 bit ».

   Sites Go à rebrancher, avec l'immédiat du jeu :

   | Site | Niveau (jeu) |
   |---|---|
   | `flock-position` | 0x10 |
   | `asset-transform` | 0x1E |
   | `crew-order` | 0x10 |
   | `tacmap-poiicon` | 0x10 |
   | `tacmap-waypointstate` | 0x10 |
   | `tacmap-areaofinterest` | 0x10 |
   | `tacmap-displayasset` | 0x10 |
   | `tacmap-cooptetherarea` | 0x10 |
   | `spawn-filter-type` étiquette 3 | 0x10 |
   | `selectable-zone-data` | 0x10 |
   | ti=38 i18 `generic-rigid-body-transforms` | 0x10 |
   | `unit-control` (`consumeQuat16` ×2) | 0x10 |
   | trame média de l'état par défaut du bipède | 0x10 |
   | world-object i0 (`dispatch_object`) | 0x10 |
   | transloc (constantes 22 recopiées) | 0x10 |

   Deux cas sont à vérifier en J6.2 :
   - le repli absolu de `FUN_14076f3ec` (`e524` nu, sans precHigh ni R(2)) ;
   - l'ancre i59.

   Le test en table des sites peut reprendre les colonnes « appelant / CALL / niveau » de §1.
2. **GA2-4 : tranché pour `consumeManagedPropertyVariant`.** L'état par défaut ti=13 appelle
   `FUN_140ce59bc` puis `FUN_140ce5aa4`, en mode A (index -1) pour g = 0 et en mode B (index
   0..31) pour g = 1. Le R(4) seul est faux.
3. **`DAT_144632be0` : à lire du profil, pas une constante.** Elle vaut 1 sur une carte à une
   plage et `ceilLog2(compte brut)` sinon ; elle est posée par `FUN_140be9a14` ; elle vaut 10 hors
   carte (`FUN_140365eb0`, `FUN_140de9a1c`). La source est `worldObjectPrecision().IndexW` /
   `profile.LargeurIndexDePlage`.
4. **GA2-5 : confirmé.** Avec idx = -1, il faut la table défaut L16 (22/22/22), ce qui aligne ce
   site sur `absAxisWFor`.

## 7. Ce qui reste incertain ou non fait

- **Composants non nommés.** Environ 35 appelants indirects au niveau 0x10, et ceux aux niveaux
  0x1E, 0x0C et 0x0D, ne sont pas rattachés à un nom de composant. La résolution par descripteur
  (`+0x28`) a échoué : le « getName » supposé en `141179610` est une fonction repliée par
  l'éditeur de liens (COMDAT). Même chose pour `FUN_140f04f68` (descripteur `143d08268`) et
  `FUN_140fb8af0` (descripteur `143c96c50`).
- **Trois lecteurs Go `consumeQuantVec3(6+niveau)` sans fonction du jeu identifiée ici** :
  `tacmap-poiiconoffset`, `flock-destination` et `player-desired-respawn-location`. On ne sait
  pas encore s'ils passent par `e494` / `1424e0e38` (candidats listés en 1.c) : à relever.
- **Budget binaire de i0 bipède.** L'écart du repli `FUN_14076f3ec` (precHigh et R(2) lus par le
  Go, absents du jeu dans `f3ec`) n'est pas tranché. Le R(2) est lu en fin de `FUN_1406cfe44`
  pour tous les chemins, et le Go peut compenser ailleurs. Il faut un test rouge (J6.2) contre la
  mesure CE de 47 bits.
- **Deux sites non comparés au bit près** : l'ancre i59 (`FUN_142f25e90`, branche lue ici, grammaire
  Go « R(3) drapeaux » mesurée) et le record de tir (`FUN_14080c1f8`).
- **Fréquence d'`asset-transform` (ti=44 i0).** Elle est inconnue. L'écart 7 contre 26 bits par
  axe est certain dans le code du jeu, mais son effet sur le corpus n'est pas mesuré ici (J6.4).
- **Famille 0x1E (`FUN_141dc86xx`–`141ddaxxx`).** L'hypothèse « hors film » n'est pas prouvée :
  les appelants `FUN_141df06c0` / `FUN_141df1130` n'ont pas été ouverts.
- **Hors périmètre J6, noté sans traitement** (règle 5) : deux portages Go de `FUN_140c1e79c`
  (`consume140c1e79c` dans `traverse.go` et `consumeCompressedDir140c1e79c` dans
  `components_world.go`), identiques aujourd'hui. Même classe de défaut que J6.

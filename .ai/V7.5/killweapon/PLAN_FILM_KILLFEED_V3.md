# PLAN — Kill-feed fiable par reconstruction ECS (la « vraie v3 »)

> **Décision user (2026-06-05)** : build complet. Reproduire le kill-feed in-game (tueur/victime/temps + ARME)
> en reconstruisant l'état ECS depuis le film, parce qu'aucun champ « arme-du-kill figée » n'est répliqué
> (établi exhaustivement : 3 workflows / 9 agents). L'arme se **reconstruit** comme le fait le moteur au replay.

## Constat fondateur (ce que le film réplique vraiment)

| Donnée | Statut film | Source |
|---|---|---|
| Tueur ↔ victime | ✅ répliqué | killer-index `rec+0x2f4` (record joueur) + events type-3 (killer/victim/time_ms) |
| Temps du kill | ✅ répliqué | type-3 `time_ms` ; kill-FRAME (type-0) à Δ≤3ms |
| Position de mort | ✅ répliqué | `rec+0x2e8/0x2ec/0x2f0` (record joueur) |
| Arme **tenue** (held) | ✅ répliqué (commutable) | composant `'obje'` variant_name R(32) @rec+0x14 (deser générique FUN_14080c1f8) ; keyframes type-2 (~20s) + FRAME deltas (33-200ms) |
| Position/état entités | ✅ répliqué | composants ECS (positions, projectiles) |
| **Arme-du-kill figée** | ❌ **PAS répliqué** | reconstruite client-side au replay (handle objet live → DamageReport local) |

**Conséquence** : l'arme du kill = la **source du dégât** au tick du kill :
- **Hitscan** (majorité : BR/Sidekick/Sniper/Bandit/Commando/Shock/Stalker/Mangler/AR/Disruptor…) → arme **tenue** du tueur au tick = `held@kill-tick`. Fiable.
- **Projectile** (roquette/Hydra/Skewer/Cindershot/FuelRod/Needler/grenades) → le **projectile létal** (entité 0x1003 répliquée, porte son arme **gelée**), identifié par la **position de mort** de la victime. Fiable.

## Architecture (couches, dans `internal/analysis/filmdec/`)

1. **L1 BitReader** (✅ livré, validé) — MSB-first big-endian.
2. **L2 deser générique** `DecodeEntityRecord` / `DecodeEntityRecordQ` (✅ livré, validé sur keyframe type-2) — composant `'obje'` + records joueur (FUN_14080c1f8).
3. **L3 FRAME-delta decoder** (à livrer) — paquet type-0 = conteneur de N records : `[R1 more][R2 type 1=new/2=del/3=delta][R7 entity-id]` → dispatch → component iterator `FUN_14076cb60` → bitmask composants `FUN_1406d7610` (R1 flag ; compact R3+N×R6 / full R64) → deser par composant présent.
4. **L4 état ECS** (à livrer) — init depuis keyframe type-2, application des deltas FRAME chronologiques. Map `entity-id → { type, held-weapon, position, killer-index (joueurs) }`. Timeline held-weapon par joueur.
5. **L5 résolution kill** (à livrer) — par kill (type-3 / killer-index) : identifier le tueur ; hitscan → `held@kill-tick` ; projectile → projectile à la position de mort → arme gelée.

## Schéma de composants — DANS le film (chunk_00), blocage levé (2026-06-05)

Décoder une entité COMPLÈTE exige la liste ORDONNÉE de ses composants + le deser de chacun. **Cette liste EST dans
`chunk_00` du film** (zlib, inflate 1.97Mo ; confirmé `cmd/tmp_registry`) : (a) liste globale 264 composants
(slots 260o `[u32 kind][u32 b][nom ASCII]`), puis (b) **blocs d'archétype ordonnés** — BIPED @0x08e300 = 84 composants
dans l'ordre d'itération de `FUN_14076cb60`. Recette nom→deser : nom .rdata → thunk `lea+ret` → vtable+0x08 →
deser = `*(vtable_base+0x30)` (vtable+0x28 = thunk partagé `0x14076ce9c` → `jmp [+0x30]`). EXCEPTION : le fire-event
`'obje'` (vtable 0x143d0ace0) a +0x28 = `FUN_14080c1f8` direct (pas un composant d'entité, c'est un highlight).

Composants/desers clés (archétype BIPED #33) : i0 position-dyn-prec `FUN_1406cfe44` ; i2/66 forward-up `FUN_14076e278` ;
i4 body-vitality `FUN_140fb8978` (santé) ; i5 shield-vitality ; i11/75 dead-state `FUN_140c1dce0` (MORT) ;
i30..41 weapon-state-ammo `FUN_140ea1018` ; **i43..46 weapon-state-type-info `FUN_1407f06bc` (HELD WEAPON variant-name ×4 slots)** ;
i64 object-position `FUN_14076e29c`. Grammaire bit de chacun = workflow `wjnq2sjwy` (Front B).

**Stratégie traversée** : (1) parser chunk_00 → liste ordonnée des composants par archétype + mapping typeIndex(R6)→archétype ;
(2) table Go index→deser ; (3) itérer header→mask(`FUN_1406d7610`)→deser par composant présent dans l'ordre, en consommant
TOUS les composants d'index < cible (sinon désalignement) ; (4) lire held-weapon (i43..46), dead-state (i11), position.
Le slot actif (arme tenue) vs slots secondaires se discrimine via `biped-desired-weapon-set` (i42) [à confirmer].
**Bugs Go corrigés** : `buildGate0x24()`→3 ; `decodeQuatBlock` câblé (quat 16-bit tag0). Reste à valider empiriquement la
traversée d'un record biped complet (held-weapon décodé == arme attendue).

## Phases (suivi : TodoWrite)

- **P1** — Décodeur FRAME-delta (L3) : boucle records + bitmask, validé en chaînant ≥N records de la kill-FRAME.
- **P2** — `decodeQuatBlock` (FUN_1431a0cbc) + extraction killer-index `rec+0x2f4` / death-pos `rec+0x2e8`.
- **P3** — Tracker d'état ECS (L4) : keyframe init + deltas → timeline held-weapon par joueur.
- **P4** — Résolution kill hitscan (L5) : killer/victim/time + held@kill-tick ; valider oracle.
- **P5** — Arme projectile : suivi projectiles + match position de mort.
- **P6** — Productionisation : table `kill_weapons` (append-only, ART-safe) + service + API + frontend kill-feed.

## P4 — Portage des desers BIPED (workflow whdm9i3f4) : état + caveats

Fichiers livrés (compilent + vet OK, **DRAFTS à valider empiriquement**) :
`filmdec/components_object.go` (i0..i17 partiel) + `filmdec/unit_weaponstate.go` (i18..i46).

**Held-weapon RÉSOLU bit-exact** (`consumeWeaponStateTypeInfoVariant`, Front A) : `FUN_1407f06bc` =
`R(1) gate [FUN_14080d69c]` ; si set → **`R(32) handle [FUN_14080d6f0]` + `R(32) variant-name [FUN_14080dec4] = L'ARME`**
+ R(12) + R(7) + [R1→R4+R6] + R(1) + liste(2494) + bloc(0550) + [R1;si0 R5] ; **tail TOUJOURS** : [R1→R8] + R(3) + 2×[R1;si0 R2].
⚠️ `FUN_14080d69c = R(1) + R(32)` CONFIRMÉ par décompile (gate→FUN_14080d6f0=R32) → l'arme est le **2e** R(32) (bits 33..64),
pas le 1er. La version Front B (`ConsumeWeaponStateTypeInfo`) OMET le handle R(32) = **erreur 32 bits, NE PAS utiliser**.

**Desers résolus** : i0 position-dyn-prec, i2 forward-up, i4 body-vitality (R8+3×R1), i5 shield (template), i11 dead-state
(forme BIPED lourde via FUN_140c1dd44 + R1@0xc4 — PAS un simple R1), i18..i42 (unit-* + weapon-state-ammo/rounds/overheated
×4 + biped-desired-weapon-set), i43..46 held-weapon. Recette `FUN_1406d84b4` largeur = arg pile (crouch=10, equipment/aiming/held=12,
overheated=7).

**CAVEATS (à valider/fermer)** :
1. **default-state BIPED (vtable+0x60)** : bit-count NON résolu statiquement (table runtime) → instrumenter empiriquement
   (Front C : 76 bits pour typeIndex=40 ; BIPED #35 à mesurer). **Contournement : décoder les DELTAS** (type-3, PAS de default-state).
2. **Object desers manquants** : i1 (translational-vel, supposé = i0), i3 (angular-vel), i6 (region-state), i7 (damage-sections),
   i8 (constraint), i10 (parent-state), i12..i17 (scale/max-vit/dissolver/low-freq/physics-flags/frame-config) → à décompiler
   (même recette). Bloquent la traversée si présents (mask) avant i43.
3. **unit-actor-control (i19) / actor-state (i20)** : 2-3 largeurs quantif dir/magnitude (dirQuantW) non récupérées statiquement.
4. **Front B vs A** : doublon `consumeWeaponStateTypeInfoVariant` (A, correct) vs `ConsumeWeaponStateTypeInfo` (B, erreur 32b).

**Stratégie de validation (P5)** : viser les **DELTAS** (évite default-state). Pour chaque record delta : header outer
(R1 flag + R2 type + R15 id) → mask → si bits présents < 43 tous portés → consommer → held-weapon i43 → cross-check
`analysis.WeaponIDToName` (high-32 + suffixe 0x42c9679f). Itérer : combler les desers manquants au fur et à mesure des desyncs.

### P5 — Framework de traversée VALIDÉ (2026-06-05) + état

- **`filmdec/traverse.go`** : `TraverseEntity(br, reg, defaultStateBits)` (header R6 typeIndex + default-state + gate +
  mask `consumeMask` + boucle composants dispatchée par `consumeByName`). `consumeMask` = `FUN_1406d7610` (R1 ; 0→R3+N×R6 ; 1→R64).
- **VALIDÉ (sonde `cmd/tmp_traverse`)** : 1ère entité keyframe chunk_02 → typeIndex=40, default-state=**76** (sweep unique),
  mask=`0xfff9800000000600`, **i9 obje décode @bit148 → variant `0x67abd42a`** (l'ancre), désync propre à i10. ⇒ registre + mask +
  dispatch + deser obje PROUVÉS alignés bit-exact.
- **Brute-force biped = bruité** : scan R6==35 + sweep default-state ancré sur obje i9 → faux positifs (les vrais bipeds n'ont
  probablement pas i9 présent ; multiplayer-properties par défaut). **Pas de raccourci brute-force fiable** sans ancre forte.
- **Chemin restant (mécanique, itératif)** : traverser le keyframe entité-par-entité depuis bit 0, en **portant les desers au fil
  des desyncs** (chaque désync nomme le composant manquant via le registre), jusqu'à un biped (typeIndex=35), puis lire i43.
  Desers à porter (recette nom→vtable+0x30) : object-parent-state (i10), object-position/velocity/angular (i0/i1/i3 + axisW),
  region-state/damage-sections/constraint (i6/i7/i8), scale/max-vit/dissolver/low-freq/physics/frame-config (i12-17), +
  unit-actor-control/state/malleable (recordStateParam/dirQuantW). Chaque deser validé immédiatement par le framework (l'obje/held
  doit décoder à un variant plausible/connu). default-state par archétype = mesuré empiriquement (sweep, ancré sur 1er composant présent).

## Validation (vérité-terrain)

- **Oracle hitscan** : kill « Frag Parfait » (Sidekick) du match Slayer 000d5950. `killer_victim_pairs` (80+ kills, killer/victim/time_ms).
- **Oracle projectile** : à identifier (un kill SPNKr/Hydra/Skewer dans les chunks 00-09).
- Cross-check arme décodée vs `analysis.WeaponIDToName` (Murmur3_32 du nom pour les variants).

## Adresses RE clés (consolidé 3 workflows)

- Packet loop type-0 : `FUN_1406cd128` (R1 more + R2 type + R7 id) → dispatch `FUN_1406cbaa0` (1 new/2 del/3 delta).
- Delta : `FUN_1406caad8` → `FUN_14076cb60` (component iterator, count @archetype+0x4320) → `FUN_1406d7610` (bitmask : R1 ; 1→R64 ; 0→R3+N×R6) → deser vtable+0x28.
- Deser générique 'obje'/joueur : `FUN_14080c1f8` (vtable descripteur arme @0x143d0ace0, +0x28). variant_name R(32) @rec+0x14 = arme (high-32 + suffixe 0x42c9679f).
- Bloc joueur : `FUN_1431a0cbc` = quat orientation (tag-dispatch ; tag1→FUN_14076dc04, tag0→FUN_14076e524 quat 16-bit). killer-index `rec+0x2f4`, death-pos `rec+0x2e8/2ec/2f0` (via deser delta).

## L3 — Grammaire FRAME CONFIRMÉE (décompile FUN_1406cd128 + FUN_141f86b58, 2026-06-05)

**Paquets** : type-0 = FRAME (≈60 fps, 1199/chunk de 20s ; type-10 = moitié appairée). Certaines frames préfixées du FrameMarker byte-aligné `a0 7b 42` (marqueur de POSITION, pas universel — type-0 #4 ne l'a pas). Le bit-reader FRAME n'est PAS à bit 0 du payload de façon naïve.

**Boucle de records (`FUN_1406cd128`)** : par record →
- (option) prefixe 32 bits si flag global `cVar1`.
- `type` : champ ~2 bits → **1=new, 2=del, 3=delta, 0=fin de frame**.
- `id` : **7 bits** (`FUN_1406d3140(0,br,7,&out)`) = entity-index. Sert d'index dans la table d'entités `param_1+0x38` (stride **0xa0**, qui porte l'archétype/type-id).
- dispatch : 1→`FUN_141f86704` (new) ; 2→read 32 bits (del) ; 3→`FUN_141f86b58` (delta).
- chaque record traité est poussé dans un tableau de sortie (stride **0xc0**), count dans `*param_6` → puis `FUN_1406cbaa0` (dispatch new/del/delta).

**Delta (`FUN_141f86b58`)** = **bonne nouvelle** : récupère l'archétype (`FUN_1406cb5f0`), **memcpy l'état précédent** (`lVar4+0xb8`, taille `lVar4+0xc0`) PUIS applique le delta via **`FUN_14076cb60` = l'itérateur de composants** (mask `FUN_1406d7610` + desers des composants présents). ⇒ **le bitstream du delta = [mask][composants présents]** — PAS de default-state dans le flux (il est en RAM). C'est EXACTEMENT `filmdec.consumeMask` + `consumeByName`, sans le `R6 typeIndex + default-state` du keyframe. **Le delta est donc plus simple à décoder que le keyframe** (qui butait sur le default-state non-mesuré).

**Conséquence build L3** (sur `internal/analysis/filmdec`, NE PAS réimplémenter les desers) :
1. Parser la boucle `[type 2b][id 7b]` sur le payload type-0 (trouver l'offset de départ du bit-reader ; le marker a07b42 n'est pas systématique).
2. Mapping **entity-id → archétype** : init depuis le keyframe type-2 (entités avec R6 typeIndex) + maj sur records `new` (type 1). C'est la pièce L4 manquante.
3. Delta (type 3) : `consumeMask` + `consumeByName` selon la liste de composants de l'archétype de l'entité → held-weapon i43 (variant) quand présent dans le mask.
4. Blocages restants = (a) les desers L2 pas tous bit-exact (desyncs i12/14/16) à fermer **avec vérité-terrain CE** (kill→arme = held-weapon du tueur au tick) ; (b) le mapping entity-id→archétype.

**Marqueur kill — VOIE FERMÉE (agent 2026-06-05, 8 sondes + contrôles)** : aucun raccourci littéral. `[killer_pi][arme][victim_pi]` = bruit (31% vs 28% contrôle) ; FourCC 'obje' absent des type-0 ; new/delta au tick non-discriminant ; littéraux d'arme trop rares au tick (1/93 ≤5ms). Le flux `d2` (519 paquets, weapon-id 64-bit @bit44, slot 90% pur) = **armes-monde ambiantes SANS holder**. ⇒ l'arme du kill n'est résoluble QUE via le held-weapon du record biped du tueur (cette voie L3).

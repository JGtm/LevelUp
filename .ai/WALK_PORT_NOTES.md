# WALK PORT NOTES — Voie A, finir le walk biped (branches i54/i59/i63) — 2026-06-07

> Notes de port bit-exact pour `internal/analysis/filmdec`. Objectif : atteindre 8 bipeds/frame clean
> (slot 519 ~29% → ~90%+) pour lire l'arme équipée du tueur par kill. Cf `HANDOFF_KILLFEED_VOIE_A.md` §3.
> **Ghidra MCP : OPÉRATIONNEL** (après redémarrage Ghidra+plugin ; le bug AF_UNIX de reconnexion mid-session
> se contourne par un démarrage propre). `decompile_function` OK.

## i59 — `FUN_142f25e90(param_1=struct sortie, param_2=BITREADER, param_3=flag)`  [tag==3 et autres]
Lecteur de sous-composant à TAGS. Lit d'abord un tag via `FUN_142f21c0c(param_2,param_2, param_1+0x4e)` → `*pcVar1`.
Switch sur le tag (cVar2) :
- **tag 0** : rien (juste `FUN_140f03e58(param_1)` à la fin) → 4 sentinelles 0xffffffff implicites.
- **tag 1** : param_1[0..3]=0xffffffff ; puis `FUN_1407f08bc(param_2, param_1+0x76)` (read short ~16b).
- **tag 2** : param_1[0..1]=0xffffffff ; `FUN_1408f0ac4(param_1+2, param_2, 5)` ; puis `FUN_1407f08bc`.
- **tag 3** : `FUN_1408f0ac4(param_1,param_2,0)` ; `FUN_1408f0ac4(param_1+2,param_2,5)` ;
  `FUN_142f26e9c(local_18,param_2)` ×3 (→ 3 vec3, param_1[4..6], [7..9], [10..12]) ;
  `FUN_14076dc04(param_2, uVar3, param_1+0x1a, 0x18)` (packed dir, **largeur 0x18=24**) ; puis REFILL bitreader (+9 bits).
- **tag 4 / 5** : param_1[0..1]=0xffffffff ; `FUN_1408f0ac4(param_1+2,param_2,5)` ; `FUN_142f26e9c` ×1 (param_1[10..12]) ;
  `FUN_14076e494(param_2, param_1+0xd, 0x10, 0,0,0)` (**0x10=16**) ; `FUN_14076dc04(param_2)` ; REFILL (+9 bits).
- **tag 6** : `FUN_1407f08bc(param_2, param_1+0x1e)` (read short) ; conditionnel ; `FUN_1408f0ac4(param_1,param_2,0)` ;
  `FUN_142f26e9c` ×2 (param_1[4..6],[7..9]) ; `FUN_1406cf008(param_2)` (**read 1 bit**) ; `FUN_14076dc04(param_2)`.
- **default (≥7)** : return (rien).

Le REFILL bitreader (état param_2 : +0x28 bitcount-cumul, +0x2c bitpos9, +0x30 registre 64-bit, +0x38 count, +0x40 ptr,
+0x10 fin) est le MÊME modèle MSB-first big-endian que `filmdec.BitReader` (byte-swap explicite visible : `uVar14>>0x38 | ... | uVar14<<0x38`).

## HELPERS À MESURER (décompiler pour les largeurs bit exactes)
- `FUN_142f26e9c` — lecteur **vec3 quantisé**. DÉCOMPILÉ : wrapper mince → `FUN_14076d528(br,br,dst, scale=DAT_143cd839c,
  param_4, 0xc, 0x18)` (0xc=12 o = 3 floats ; 0x18=24 ?). **= probablement le `ReadQuantizedVec3` DÉJÀ porté dans filmdec**
  (cf `quantize_test.go::TestReadQuantizedVec3_World100`). À confirmer : largeur exacte de `FUN_14076d528` vs le ReadQuantizedVec3 existant.
- `FUN_14076dc04` — lecteur **packed** (largeur passée en arg : 0x18 ici ; cf `consume1432026f4` largeur 0x13 dans
  `components_biped_ability.go`). À confirmer.
- `FUN_1408f0ac4(dst, br, mode)` — lecteur conditionnel (mode 0 et 5). À décompiler.
- `FUN_1407f08bc(br, dst)` — read short (handle 16b ? gate+R16 ?). À décompiler.
- `FUN_14076e494(br, dst, 0x10, 0,0,0)` — read 16 champs / 16 bits. À décompiler.
- `FUN_1406cf008(br)` — read 1 bit (déjà connu).
- `FUN_140f03e58`, `FUN_142f26e40`, `FUN_14297ea84`, `FUN_1407f08bc` — annexes (fin/init).

## CORRECTION SCOPE (2026-06-07, lecture `traverse.go`) — le walk est DÉJÀ partiellement porté
Les cases sont **câblées** (pas des stubs) dans `traverse.go` `consumeByName` :
- **i54** = `consumeBipedMobilityAction` (`FUN_1408f0264`) — PORTÉ.
- **i59** = `consumeBipedSpartanAbilityNonPredictedState` (`FUN_142f02994`) — PORTÉ ; `FUN_142f25e90` (décompilé ci-dessus)
  = son **sous-handler tag==3**. À VÉRIFIER : le consume Go gère-t-il tag==3 (vec3×3 + packed 0x18 + refill) comme `FUN_142f25e90` ?
- **i63** = `consumeBipedAction` (`FUN_142f027f4`→`FUN_142f26a20`) — **retourne `ported=false` sur la boucle value-gated
  count>0** (désync VOLONTAIRE propre ; cas commun 196 bits). C'est LE résidu principal (slot 519 ~29%).
⇒ Le travail = **fixer des BRANCHES spécifiques**, pas porter à zéro. Cibles : (1) i63 count>0 (`FUN_142f26a20` boucle) ;
(2) confirmer i59 tag==3 (`consumeBipedSpartanAbilityNonPredictedState` vs `FUN_142f25e90`).
PROCHAIN CONCRET : décompiler `FUN_142f26a20` (i63 loop count>0) + lire `consumeBipedAction` (Go) + `consumeBipedSpartanAbilityNonPredictedState`
(Go) → comparer aux décompiles → porter les largeurs manquantes → `tmp_worldreplay` pour mesurer.

## MÉTHODE DE PORT
1. Décompiler chaque helper → largeur/séquence bit exacte.
2. Porter dans filmdec (nouveau consume pour i59 + helpers vec3/packed manquants).
3. Relancer `cmd/tmp_worldreplay` (ou `tmp_i63tags`) → mesurer slot 519 clean %. Itérer jusqu'à 8 bipeds/frame.
4. Puis : lire WST i43 (famille) du slot tueur au kill, croiser chunk_27, valider narration.

---
# CONCLUSIONS INVESTIGATION WALK (2026-06-07, autonome) — réorientation
1. **LIMITE DURE i63** : `consumeBipedAction` count2 = `FUN_1409fe718(state,0x49)` = popcount masque RAM 73 bits,
   **0 bit dans le flux**. Quand count2>0, le walk NE PEUT PAS être bit-exact (donnée hors film). C'est l'irréductible.
   count1>0 = aussi value-gated (tags). Donc le walk 8-bipeds/frame parfait n'est PAS atteignable.
2. **WST i43 (arme) lu AVANT la désync** : bipeds désyncent à i51/i55/i63 (APRÈS i43). `tmp_loadout` lit l'arme par
   record malgré desyncAt=i51+. ⇒ pour l'ARME, pas besoin du record complet — juste jusqu'à i43.
3. **tmp_worldreplay** : biped 519 décode CLEAN (desyncAt=i-1) sur certains type-0 ; la désync est souvent une entité
   NON-biped (ex slot 505 typeIdx=0 game-engine-alliance-component i10), pas le biped.
4. **Verrou réel = record→joueur** : loadouts keyframe en ordre roster (record_i≈pi_i, À CONFIRMER) ; et il faut décoder
   les loadouts à TOUS les keyframes (26, un type-2 par ~18s), pas juste chunk_02.

## VOIE PRAGMATIQUE (recommandée, ≠ walk complet)
Arme par kill = arme équipée du tueur au keyframe le plus proche ≤ t_kill, cross-checkée par un record de dégât
(519) de même famille près de t_kill. Étapes :
A. Décoder les 8 bipeds (WST i43) à CHAQUE keyframe (type-2 par chunk) — lire l'arme avant désync ; mapper record_i→pi_i.
B. Kill feed chunk_27 (tueur slot→pi, temps) → arme du tueur au keyframe ≤ t.
C. Cross-check 519 records (`tmp_dmgscan`) : famille présente près de t ? → confiance.
D. Valider narration (BR75 JGtm→Akatsuki ; marteau IKE→JGtm). Granularité keyframe (~18s) = limite acceptée pour swaps.

---
# NOUVELLE PISTE (2026-06-07, question user "dégâts à la victime") — LIER PAR POSITION
Décompiles FUN_14080c1f8 (deser) + FUN_14080a9d4 (reader @14080AADE) confirment :
- Reader = flux d'events : lit **7 bits = event-type** (uVar9<0x7b=123), indexe `*(param_1+0x18)+0x210+type*8` → handler,
  appelle son deser (vtable+0x68, param=bitreader). PAS de cible/victime lue. param_1 du deser = handler (pas une unité).
- Deser record dégât : n'utilise PAS param_1 (contexte). Lit arme (+0x14/+0x0c), slot (+0x08), **hit-sections** (+0x40 count +0x34 ;
  +0x110 count +0xf8), **POSITIONS D'IMPACT** (+0x2a0 via FUN_1406cd5b8, +0x2bc). => la victime est IMPLICITE (position d'impact), pas un ID.

**PISTE** : lier record→victime par POSITION (la victime est connue via chunk_27 ; l'attaquant non, mais on n'en a pas besoin) :
1. Décoder la position d'impact du record (+0x2a0) pour les 519 records (étendre tmp_dmgscan).
2. Décoder la position joueur (biped **i0** = object-position-dynamic-precision-component, lue AVANT désync i51+ ; slot via id record → slot → joueur chunk_27 b36/b37) par frame.
3. Par kill (victime slot+temps T) : position victime à T ⋈ record dont impact-pos ≈ victime-pos près de T → coup létal → ARME.
Précision temporelle records = ms (ts paquet). Positions joueur = i0 lisible malgré désync aval.
RISQUES : positions joueur sparse (seuls les bipeds en delta) ; impact-pos = point de contact (≈ centre victime). À MESURER sur narration.

## DÉTAIL BUILD position-linking (2026-06-07) — spec complète côté victime
- i0 = `consumeObjectPositionDynamicPrecisionD(br, TraversalPrecision)` (components_movement.go:117). 3 chemins :
  - keep/baseline + full-precision : `br.ReadBits(rawVec3Bits=0x60)` = **vec3 BRUT 96 bits = 3 float32** (directement extractible).
  - absolu (`consumeAbsoluteWithGate`) : 3 axes `br.ReadBits(pd.AxisW[i])` = **quantisé** → déquant via `ReadQuantizedVec3(bits, rng Vec3Range)`
    (quantize.go:31, RETOURNE [3]float32) ; **range monde à dériver** (cf `quantize_test.go::TestReadQuantizedVec3_World100`, TraversalPrecision a les AxisW).
  - delta prédit (`consumePredictedDelta`) : delta vs frame précédente → **accumuler** depuis le baseline keyframe.
- PLAN côté victime : (1) capturer la pos i0 (ajouter capture dans le décode, non-breaking) ; (2) baseline keyframe absolu + accumuler deltas par slot ; (3) range monde pour déquant.
- Côté impact : porter FUN_14080c1f8 complet (param_5=1 → +0x27c FUN_140c9e4d8 ; param_5=0 → +0x2a0 FUN_1406cd5b8) → pos impact par record.
- = gros chantier (3 parties : pos victime quantisée+delta, pos impact deser complet, match). Parallélisable.

---
# VERDICT POSITION-LINKING (workflow wrn7n6r5n, 2026-06-07) — NÉGATIF DÉFINITIF, chiffré
- **Pos victime (P1)** : dispo 22/93 seulement, aliasée (boîte ±100 trop petite). PARTIEL.
- **Pos impact (P2)** : décodable **0/519** records (gate 7/519, vec3 0/519). Ce N'EST PAS une coordonnée monde —
  c'est un **offset RELATIF ±10 log-quantisé**. Donc inexploitable pour matcher la position victime. NON.
- **Match position (P3)** : **0/93** kills (P1∩P2 vide), à ±1500ms ET ±3000ms. Narration **0/6** (aucun BR75/Hammer
  dans la fenêtre, seulement armes tierces). Probe : `cmd/tmp_poslink` (Go pur, additif).
- **CAUSE RACINE (confirmée une Nième fois)** : rien dans le film ne relie un record de dégât à sa VICTIME (ni à
  l'attaquant). La famille @+0x14 est bit-exact (519/519), mais le LIEN record→kill n'est pas sérialisé.
⇒ **Le position-linking est structurellement mort.** Tous les chemins offline (walk, keyframe, records, position)
butent sur le MÊME mur : l'association kill↔source est résolue en RAM au replay, jamais dans le flux.
SEULS chemins donnant le lien : debugger (FUN_14080c1f8/FUN_1407e00ac au replay, par film, non scalable) ; OU walk
biped bit-exact (arme équipée du tueur, mais limite count2 RAM + mapping). Angle non épuisé (incertain) : hit-sections
+0x40/+0x110 (largeurs value-gated, RE FUN_1406d310c/FUN_1406d84b4) — pourraient porter une cible ? peu probable.

---
# 🔑 TOURNANT (2026-06-07, recherche communautaire @acurtis166/@JGtm) — PLAYER INDEX décodable dans fire/melee/grenade events
La conclusion "pas d'attaquant dans le film" était vraie pour les RECORDS DE DÉGÂT, **FAUSSE pour les events fire/melee/grenade** :
ceux-ci portent le **PLAYER INDEX (tireur/lanceur)** + l'arme + (fire) un aim vector + hit/miss. Recettes bit-packées :
- **GRENADE** : marqueur `0x4c0c00` (24b) → grenade id 32b (FRAG 0xB0171062, PLASMA 0xC0E34C44, SHOCK 0x3B2567D4, SPIKE 0x9212E428)
  → +47b → **player index 5b**. ✅ DÉCODÉ : `cmd/tmp_meleegrenade` = **70 grenades, player index 0-7 tous valides** (IKE x20…).
- **MELEE** : marqueur `0b10100110010` (11b, 0x532) ; anchor +3b ; player idx @+20 (5b) ; type @+76 (0x42/0x47/0x60) ;
  weapon @+86 (0x47 Hammer)/+88(0x42)/+101/103(0x60). ⚠️ trouve 0 → marqueur/offsets à CALER vs notre film (debug).
- **FIRE** : weapon id 32b (high32=famille) + shot counter 0-127 + bit burst (final shot) + bit hit/miss + player index + AIM VECTOR
  (cubemap 30b, face=÷0xAAA8000, code décodage fourni). high32 = la famille (= notre clé).
⇒ **VOIE ROUVERTE** : par kill (tueur via chunk_27, temps T) → event fire/melee/grenade du TUEUR près de T → arme.
  Reste : (1) caler la mêlée (marteau) ; (2) décoder les fire events (player index + hit/miss) ; (3) croiser avec chunk_27
  (mapper player index ↔ slot tueur b36/b37) ; (4) valider narration. Grenade = OK ; mêlée/fire = à finir.
Validation décode : grenade ids + weapon ids communautaires == nos valeurs (Disruptor 0x84BD29ED…).

## CALIBRATION MÊLÉE (2026-06-07, debug hammer 0x841ac5e5) — corrections recette
- **MARQUEUR réel = 0x534 (HIT) / 0x535 (MISS)**, 11 bits — PAS 0x532. (Source disait 0b10100110010 ; réel = 0b1010011010x,
  x=bit hit/miss.) Cohérent : 0x534>>3=0x34, 0x535>>3=0x35 = le "anchor 0x34/0x35".
- **TYPE 0x47 confirmé** à anchor+76 (= W-10 si weapon@W). ~40 occurrences hammer dans le match.
- weapon@anchor+86 (=W) ; anchor=W-86 ; marker@W-89.
- **PLAYER INDEX : anchor+20 (=W-66) = FAUX (lit toujours 0)** → offset à retrouver (scanner les offsets avant le weapon
  pour un champ 5b cohérent = IKE/pi4 sur les events près des kills IKE).
- Events hammer présents près du kill narré 355.7s (356.9/357.9/358.6s marker 0x535 type 0x47). Données mêlée OK, reste le player index.
- TODO : (1) caler player-index mêlée ; (2) décoder FIRE events (high32 famille + player index + hit/miss + aim vector cubemap) ;
  (3) croiser fire/melee/grenade events (player index → slot tueur chunk_27) par kill → arme ; (4) valider narration.

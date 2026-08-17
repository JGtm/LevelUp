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

---

# IMAGE-CLE — la grammaire du CORPS d'un record (lot R5, 2026-08-17)

> Lecture Ghidra STRICTEMENT read-only (instance PID 10104, `HaloInfinite.exe`, GhidraMCP
> 127.0.0.1:8089 ; `decompile_function` / `get_xrefs_to` / `read_memory` /
> `get_assembly_context` uniquement — aucun rename, aucun script, aucune analyse relancee).
> Contexte : `PLAN_R5_GRAMMAIRE_IMAGE_CLE.md`. Deux lots (R3 `ti=37`, R4 `ti=11`) avaient
> conclu le meme jour que « la grammaire du corps d'un record d'image-cle n'est resolue
> nulle part ». Ce que la lecture montre est plus precis, et different.

## 1. Les DEUX lecteurs de record NEW du jeu portent la MEME grammaire

| adresse | role | qui l'appelle |
|---|---|---|
| `FUN_141f86704` | deser NEW, variante BUFFERISEE | `FUN_1406cd128` a `0x1422f46b8` (unique xref CODE) |
| `FUN_1408f1aa4` | deser NEW, variante DIRECTE | `FUN_1406cbaa0` a `0x1406cbc?` (`iVar13 = FUN_1408f1aa4(*plVar3, param_2, param_5, param_6, param_7)`) |

Sequence, IDENTIQUE dans les deux (verifiee ligne a ligne) :

```
R(6) typeIndex
desc = *(*(param_1 + 0x18) + 8 + ti*8)          (registre d'archetypes, cap 0x32 = 50)
n     = vtable[0x20](desc)                       0 bit  (taille de l'etat)
m     = vtable[0x10](desc)                       0 bit  (taille du masque par defaut)
vtable[0x60](desc, n, dst, READER, 1)            = ETAT PAR DEFAUT  <- le seul appel qui lit
vtable[0x88](desc, n, dst, m, buf)               0 bit  (remplit le MASQUE PAR DEFAUT)
vtable[0x30]()                                   0 bit
si (FUN_1404f2b4c() != 0 ET DAT_144c232e1 != 0)  : porte = R(1)   [mode FILM, lu TOT]
si (masqueParDefaut != 0 OU porte != 0) :
    si la porte n'a pas deja ete lue : porte = R(1)
    si porte != 0 : FUN_14076cb60(desc + 1, ctx)  = MASQUE + COMPOSANTS
```

`FUN_14076cb60` (boucle de composants) :

```
FUN_1406d7610(desc, reader, &masque)     = R(1) ; si 0 -> R(3) compte + compte x R(6) index
                                                  ; si 1 -> R(64)
pour i de 0 a *(desc + 0x4320) - 1 :
    si (masque >> (i - sautes)) & 1 :
        pred = (ctx[4] == 0) ? 0 : vtable[0x48](comp, ...)     0 bit
        vtable[0x28](comp, reader, ctx, &pred, n)              = LE DESER DU COMPOSANT
        si mode film : R(1) ; si 1 -> R(32) sentinelle 0xbcddcba
```

## 2. Reponse a l'hypothese « le corps d'image-cle appelle le deser FEUILLE `+0x28` la ou le delta appelle un wrapper » : REFUTEE PAR LECTURE

Le chemin DELTA (`FUN_141f86b58`) appelle **la meme fonction `FUN_14076cb60`**, avec un
contexte de meme forme (memset 0x40, puis `ctx+0x00` sortie, `+0x08` taille, `+0x10` etat,
`+0x28` reader, `+0x30` vtable[0](desc), `+0x34` id, `+0x38` = 1, `+0x39` flag). Dans les
TROIS lecteurs (NEW bufferise, NEW direct, DELTA) `ctx+0x18` et `ctx+0x20` restent NULS, donc
`param_2[4] == 0` dans `FUN_14076cb60` et la baseline predite vaut zero partout.

**Il n'existe pas de second site d'appel de composant.** Chaque composant present passe par
`vtable[0x28]`, en image-cle comme en delta. La difference NEW / DELTA se resume a l'en-tete
que NEW porte en plus : `R(6) typeIndex` + `vtable[0x60]` (etat par defaut). C'est exactement
ce que `filmdec.TraverseEntity` fait deja.

## 3. Ce qui DIFFERE entre les chemins, et qui n'est PAS dans le corps du record

- `FUN_1406cd128`, tete de chaque iteration, mode film (`FUN_14076cea8()`) : **`R(32)`**.
- `FUN_1406cd128`, branches NEW et DEL, mode film : **`R(1)` ; si 1 -> `R(8)`** AVANT le deser.
- `FUN_1406cbaa0`, avant `FUN_1408f1aa4` (`0x1406cb?46`-`149`) : le MEME prologue
  **`R(1)` ; si 1 -> `R(8)`**.
- `FUN_1406cd128` est **DESACTIVEE quand la porte d'image-cle `*(param_1 + 0x12)` est mise** :
  `uVar14 = -(uint)(cVar3 != 0) & 2` puis `if (uVar14 != 0) break`. Elle NE decode PAS la
  table d'image-cle. Le chemin d'image-cle passe par `FUN_142f2913c` (baseline-emit), qui
  draine une file par-entite et redispatche par `FUN_1406cbaa0`.

**Consequence, et c'est le point de methode** : ces trois lectures appartiennent au CADRE
(la boucle de records du paquet delta), pas au CORPS. La table d'image-cle du film (payload
type-2) a son propre en-tete, `[id:32][field:26][ti:6]` (etabli empiriquement, 249/250
entites contre un oracle Cheat Engine, `keyframe_world.go:17-23`) ; le consommateur de ce
payload n'est PAS identifie statiquement dans cette passe, et il n'a pas besoin de l'etre :
le CORPS qui suit l'en-tete est celui d'un record NEW, et les deux lecteurs de record NEW du
jeu sont d'accord sur sa grammaire.

## 4. `ti=42` (arme au sol) — etat par defaut RESOLU

Chaine de resolution (celle de `default_state_arch.go:5-18`, rejouee) :

```
FUN_140e453b4 (registrar)  ->  FUN_140e45fc4(world, 0x2a, &PTR_PTR_144701780)   @0x140e4578f
xref [WRITE] sur 0x144701780 -> FUN_1403721d0 : PTR_PTR_144701780 = &PTR_LAB_1436fd790
vtable 0x1436fd790 , *(vtable + 0x60) = 0x1407f0c68
```

`FUN_1407f0c68` (lu ligne a ligne, chaque feuille touche `reader+0x2c`) :

| # | lecture | source |
|---|---|---|
| 1 | `V` = `R(1)` ; si 1 -> `R(8)` | `FUN_1406cf008` + bloc inline `+8` |
| 2 | `FUN_1407f2224(desc, 0x60, dst, reader, flag)` = `V` + `FUN_14080cfe8` | `MOV EDX,0x60` @`0x1407f0cd1` ; `FUN_14080cfe8` = le bloc multiplayer-properties DEJA porte bit-exact (`consumeMultiplayerPropertiesBlock`) — donc `FUN_1407f2224` == `consumeDefaultStateTI36` |
| 3 | `R(12)` -> `dst+0x60` | bloc inline `+0xc` |
| 4 | `R(7)` -> `dst+0x64` | `FUN_1406d84b4`, largeur figee par `C7 44 24 20 07 00 00 00` (`MOV dword [RSP+0x20],7`) @`0x1407f0d30`, juste avant `CALL 0x1406d84b4` @`0x1407f0d38` |
| 5 | `FUN_1407f2494(dst+0x68, reader)` — bloc de liste, ci-dessous | `CALL` @`0x1407f0d49` |
| 6 | `ECS_ReadEntityRefIndex5` = `FUN_1407f2058` = `R(1)` ; si 0 -> `R(5)` | -> `dst+0xa4` apres resolution de handle (0 bit) |

`FUN_1407f2494` :

```
porte = R(1)                                  (FUN_1406cf008 @0x1407f24c1)
si porte == 0 : FUN_14080d69c(_, reader, ...) = R(1) ; si 1 -> R(32)
                (args verifies au desassemblage : MOV RDX,RBP = le reader, CALL @0x1407f24dc)
sinon         : n = R(4)                      (FUN_1424e1d48, `+4` inline)
                n fois : R(1) ; si 1 -> R(32) (FUN_1406cf008 + FUN_14080d6f0)
```

`FUN_14080d69c` verifie : `R(1)` ; si 0 -> valeur par defaut en RAM ; si 1 -> `FUN_14080d6f0`
= `R(32)`. `FUN_1424e1d48` verifie : `R(4)`.

**Table de la vtable de `ti=42`** (`0x1436fd790`, lue octet a octet) : `+0x40` et `+0x70`
pointent le stub `0x1408d8220` (`return 1`, 0 bit) ; `+0x60` = `0x1407f0c68` (ci-dessus) ;
`+0x88` = `0x140ea3ef4` (masque par defaut, 0 bit).

## 5. Ce qui reste NON resolu apres cette passe, et il faut le dire

- Le CONSOMMATEUR du payload type-2 du film (celui qui lit l'en-tete `[id:32][field:26][ti:6]`)
  n'a pas ete identifie statiquement. Consequence : la SEMANTIQUE des 26 bits de `field`
  reste inconnue. Le balayeur du depot (`keyframe_world.go:70`) n'accepte une ancre QUE si
  ces 26 bits sont NULS — c'est un filtre, pas une lecture.
- `vtable[0x88]` (masque par defaut) n'est porte pour aucun archetype. Il ne lit aucun bit,
  mais il commande la lecture de la porte `R(1)` : un archetype dont le masque par defaut est
  non nul lit la porte meme quand le flux ne la porte pas.

# IMAGE-CLE — file par entite, et le vrai sort du payload type-2 (lot R6, 2026-08-17)

> Lecture Ghidra STRICTEMENT read-only (instance PID 10104, `HaloInfinite.exe`).
> **Note d'acces** : le pont `mcp__ghidra__*` refuse de se connecter (l'instance UDS se
> declare `unknown`, `connect_instance` refuse tout repli TCP). L'API HTTP du plugin
> (`127.0.0.1:8089`) a ete utilisee a la place — memes endpoints
> (`/decompile_function`, `/disassemble_function`, `/get_xrefs_to`, `/read_memory`),
> meme programme, aucun rename, aucun script, aucune analyse relancee.
> Contexte : `PLAN_R6_FILE_PAR_ENTITE.md`. Ouvre sur la condition de reprise ecrite par R5
> (« decompiler le CONSOMMATEUR du payload type-2 »).

## 1. La file par entite n'est PAS une transformation — c'est un REPORT

| adresse | ce que la decompile montre |
|---|---|
| `FUN_142f29538` | **push d'un tampon circulaire**. Element de 0x38 = 56 octets (`param_1 + idx*0xe + 4`, en ints). Recopie `param_2[0..3]`, un handle refcompte (`FUN_142a777e8`), puis `param_2[8..0xd]`. **Aucune ecriture de bits, aucun reencodage.** |
| `FUN_142f2913c` | **drain**. Par item : `FUN_14064c350` (init reader), `FUN_1411b149c(reader, *(item+0x10), *(item+0x20))`, `FUN_1406d5cc0(reader,3)`, `FUN_1432fe23c(reader, *(item+0x2c))`, puis `FUN_1406cbaa0(*(item+0x30), *(item+0x28), param_1+0x38, param_1, item, reader, 0)` |
| `FUN_1411b149c` | `*(r+8)=data ; *(r+0x10)=data+len ; *(r+0x18)=len` -> `item+0x10` = **pointeur de tampon**, `item+0x20` = **longueur du TAMPON** (pas du record) |
| `FUN_1432fe23c` | `*(r+0x2c) = *(r+0x28) = param_2 ; *(r+0x20) = 2` -> `+0x2c` est la **position en bits** du reader (meme champ que la capture live de juillet). Donc `item+0x2c` = **le bit de depart du record** |
| `FUN_142f25334` | **`memcpy` du tampon ENTIER du paquet** (`src = *(reader+8)`, `n = *(reader+0x18)`) dans un bloc alloue refcompte. Chemin source revele par l'allocateur : `...\engine\source\blofeld\networking\replication\replication_entity_manager_view.cpp` |
| `FUN_1406cd128` @`0x1422f44fb` | **UNIQUE xref CODE du push** (l'autre xref, `0x145691d64`, est une DATA). Branche `DAT_14474cd78 != 0` : `uVar23 = *(param_3+0x2c)` (bit courant, capture AVANT decodage), `FUN_1406cbaa0(..., 1)`, construction de l'item, `FUN_142f29538(*(param_1+0x1b320), &local_110)` |
| `FUN_142f2b5c4` | la garde du push : vrai si l'entite n'est pas encore reliee cote vue (`*(*param_2+0x68)`, `FUN_142f287f0`) |

**Structure d'item, champ par champ** (56 octets, ordre des locales de `FUN_1406cd128`) :

```
+0x00  16 o   *param_2 / param_2[1]   (contexte de paquet)
+0x10   8 o   handle vers la COPIE du tampon de paquet   (FUN_142f25334 = memcpy integral)
+0x20   4 o   longueur du tampon        (= *(reader+0x18))
+0x24   4 o   horodatage                (FUN_1405f50b8)
+0x28   4 o   id d'entite
+0x2c   4 o   BIT DE DEPART du record dans le tampon
+0x30   4 o   type de record (1=NEW, 2=DEL, 3=DELTA)
+0x34   4 o   index de vue
```

**REPONSE A LA QUESTION DU LOT** : il n'existe **aucune transformation
« payload type-2 -> file par-entite »**. Un item ne porte pas de bitstream reconstruit : il
porte une COPIE OCTET POUR OCTET du tampon du paquet, plus la position en bits du record.
Le drain repose un reader sur cette copie, se replace au meme bit, et appelle le MEME
`FUN_1406cbaa0` avec la MEME grammaire. **La file DIFFERE des records ; elle ne les
reecrit pas.** L'objet que la condition de reprise de R5 demandait de decompiler n'existe
pas.

## 2. Le demultiplexeur de paquets du film — et le sort du type-2

Chaine remontee : `FUN_142f2913c` / `FUN_1406cd128` <- `FUN_142987460` (3 vues x
[drain + boucle de records], puis `vtable[0x48]` applique) <- `FUN_14298816c` <-
`FUN_1428e2778` <- **`FUN_1428e22c0`** <- `FUN_1428e27c0` (boucle de paquets).

`FUN_14298816c` porte la chaine source `...\engine\source\blofeld\saved_games\SavedFilmChunks.cpp`.

**`FUN_1428e22c0` = l'aiguillage par TYPE DE PAQUET** (`sVar2 = *param_3`, le `u16` de tete
de l'en-tete de 16 octets). Verifie au DESASSEMBLAGE (`0x1428e22ca` `MOVSX EDX,word ptr [R8]`,
puis la chaine `CMP 8 / TEST / SUB 1 / SUB 1 / SUB 4 / CMP 1`) :

| type | handler |
|---|---|
| 0 | `FUN_1428e2778` -> `FUN_14298816c` -> `FUN_142987460` (**decodeur de replication**) |
| 1 | `FUN_142989418` |
| **2** | **AUCUN** — `JZ 0x1428e2412` = `XOR SIL,SIL` (retourne 0) + telemetrie `FilmBlockReadError` |
| 3, 4, 5 | AUCUN — meme cible `0x1428e2412` |
| 6 | `FUN_142988084` · 7 `FUN_142985698` · 8 `FUN_142987bd4` · 9 compteur `+0xf8` · 10 `FUN_142988244` · 11 `FUN_1429882c8` · 12 `FUN_1429875e4` |

**Et pourtant la lecture du film ne casse pas** — parce que le paquet type-2 n'arrive JAMAIS
a l'aiguillage :

```
FUN_142989418 (handler du type 1) :
  *(ctx+0xf8) += *(hdr+4)                     ; saute le payload du type-1 (343 019 o)
  FUN_142988338(ctx, local_18, 0x10, 0)       ; lit l'en-tete SUIVANT (16 o)
  si ok : *(ctx+0xf8) += local_14             ; saute AUSSI le payload suivant (= le type-2)
```

`FUN_142988338(ctx, dst, n, 0)` = `memcpy` de `n` octets depuis le chunk inflate au curseur
`*(ctx+0xf8)`, borne par `*(ctx+0xe8)`, puis `curseur += n`. `local_14` est le champ `size`
(`+4`) de l'en-tete relu.

**CONCLUSION, ET C'EST LE RESULTAT DE FOND DU LOT R6** : **le jeu ne decode JAMAIS le payload
type-2 en lecture de film. Il le SAUTE**, en meme temps que la table de precision type-1, par
le handler du type-1. Il n'y a donc pas de consommateur a decompiler : il n'y en a pas.
Le bloc type-2 est ecrit par l'ENREGISTREUR et ignore par le LECTEUR.

Cela explique a posteriori le negatif de R5 (« le corps d'un record d'image-cle n'est pas un
record NEW », 128 decalages x 16 lectures x 3 films, jamais plus de 1,8 %) : ces records ne
sont relus par aucun deserialiseur du jeu.

## 3. Ce que le jeu utilise a la place : le PREMIER paquet type-0

Structure d'entree de session, mesuree sur les 3 films oracles (decoupe de paquets, chaine
independante de toute lecture de bits) :

```
#0 type=1  343 019 o   (table de precision — SAUTE)
#1 type=2  138 340 / 140 837 / 142 695 o   (etat monde — SAUTE)
#2 type=6  4 o   ·  #3 type=8  25 124 o  ·  #4 type=12  4 o  ·  ... 
#8 type=0  9 297 / 11 312 / 11 066 o      (PREMIER PAQUET DELTA)
```

Les 16 premiers octets du premier paquet type-0 sont **identiques d'un film a l'autre** :
`88 00 15 84 00 2c 54 0c 61 c9 00 0b ff ff ff fc`. Ce sont exactement les 16 premiers octets
de `.ai/V7.5/dumps/keyframe_buffer_live.bin` (11 485 o) ET de `kf_slot0_live.bin` (7 286 o).

**La capture live de juillet, etiquetee « keyframe » depuis, a ete prise sur le PREMIER
PAQUET DELTA**, pas sur le payload type-2 — ce que sa propre pile d'appel disait deja
(`FUN_1406cbaa0` <- `FUN_1406cd128`), puisque `FUN_1406cd128` ne s'execute jamais quand la
porte d'image-cle `*(param_1+0x12)` est mise.

## 4. Ce qui reste NON resolu apres cette passe

- La SEMANTIQUE du payload type-2 reste inconnue, et elle le restera par cette voie : aucun
  code de lecture ne l'interprete. Le seul levier serait l'ECRIVAIN (cote enregistrement) : voir §5,
  ou il est cherche et NON trouve.
- `vtable[0x88]` (masque par defaut) n'est toujours porte pour aucun archetype (report R5).
- `DAT_144731d20` (`FUN_1428e27c0` @`0x1428e2980`) est un MASQUE DE TYPES « a traiter
  immediatement dans le meme tick » ; les autres types sont reportes au tick suivant. Non
  exploite ici.

## 5. L ECRIVAIN du bloc type-2 — recherche BORNEE, negatif

Puisque le LECTEUR saute le bloc type-2 (§2), la question devient « qui l'ECRIT, et selon
quelle grammaire ». Recherche bornee a trois sondes de chaines/xrefs, toutes negatives :

| sonde | resultat |
|---|---|
| xrefs de la chaine d'allocation `...\saved_games\SavedFilmChunks.cpp` (`0x143ddb470`) | **UNE seule** : `0x1429881b6` dans `FUN_14298816c` — le LECTEUR |
| `search_strings` sur `saved_games` (min 10, tout le binaire) | **1 seule chaine**, celle ci-dessus |
| `search_strings` sur `FilmBlock` / `RecordFilm` / `film_block` | **1 seule chaine** : `FilmBlockReadError` (`0x143dcb3b0`), cote LECTURE |
| xrefs de l'encodeur de snapshot `FUN_142f2e174` (vtable `0x1436a87e0` + 0x10) | **aucun appel direct** : uniquement la case de vtable `0x1436a87f0` et une DATA. Ses appels passent par `vtable[0x10]`, non resolus statiquement |
| cvar `SavedFilmValidateEntityState` (`0x143732020` -> `FUN_141129458`, valeur `DAT_1450fb320`) | aucune lecture directe de la valeur : elle passe par l'objet cvar `DAT_1450fb2f0` |

**Conclusion bornee** : le chemin d'ECRITURE des chunks de film ne laisse dans
`HaloInfinite.exe` **ni site d'allocation nomme, ni chaine d'erreur**, la ou le chemin de
LECTURE laisse les deux. La grammaire du bloc type-2 n'est donc pas atteignable par la voie
chaines/xrefs dans ce binaire. La piste restante, non ouverte ici : le format se deduit du
CONTENU (la table est deja balayee a 249/250 entites contre un oracle Cheat Engine) ou d'un
autre binaire (serveur dedie), hors perimetre offline-pur.

---
# BIPEDE IMAGE-CLE BIT-EXACT — decompiles portes (lot R7-b, 2026-08-17)

> Acces Ghidra : instance PID 10104 (`HaloInfinite.exe`), **API HTTP du plugin**
> `http://127.0.0.1:8089` (`/decompile_function`, `/disassemble_function`, `/read_memory`),
> LECTURE SEULE — le pont `mcp__ghidra__*` refuse toujours la connexion (instance UDS
> `unknown`), contournement deja consigne par R6. Aucun rename, aucun script, aucune analyse.

## 1. `i60 simulation-state-component` — la QUEUE est resolue, le predicat est vrai par construction

`FUN_142ED6D88` (thunk `142f02434`). Grammaire confirmee, decompile + disasm :

```
cVar1 = FUN_1406cf008(br)                       R(1)   -> dst+0x28
si cVar1 == 0 : FUN_14058c250(dst)              0 bit, FIN
sinon :
  2x ECS_ReadEntityRefIndex5 (= FUN_1407f2058)  R(1)[si 0 -> R(5)]   -> dst+0x00, +0x04
  4x FUN_142ee2194                              R(16) chacun         -> dst+0x08..+0x14
  2x R(2) inline                                                     -> dst+0x29, +0x2a
  4x FUN_142ee2194                              R(16) chacun         -> dst+0x18..+0x24
  FUN_140c1e79c(br, ?, out=dst+0x2c, dir=dst+0x38)   R(1)[si 0 -> R(19)] + R(8)
  cVar1 = FUN_140501798(dst+0x2c, dst+0x38)     0 bit
  si cVar1 != 0 : FUN_14076e494(br, dst+0x44, 0x10, 0,0,0) ; FUN_140492128(dst+0x44)  0 bit
```

`FUN_142ee2194` = `FUN_1406d84b4(..., DAT_143cd8f84 = -100.0, DAT_143cd84a8 = +100.0, 0x10, 0, 1)`
-> **R(16)** dequantifie dans [-100, +100].

**POURQUOI LE PREDICAT EST VRAI.** `FUN_140c1e79c` decode une DIRECTION unitaire en `dst+0x38`
(porte R(1) ; sur 0, packed R(19) -> `FUN_1406d8288`), lit un ANGLE R(8), puis appelle
`FUN_1406d8678(dir, angle, out)` : produit vectoriel de `dir` avec un vecteur de base, division
par la norme, rotation de Rodrigues autour de `dir` par l'angle, normalisation finale
(`FUN_1404fec88`). `out` (= `dst+0x2c`) est donc **unitaire et PERPENDICULAIRE** a `dir`.
`FUN_140501798(v1, v2)` teste exactement : `|‖v1‖² − DAT_143cd8374| < DAT_143cd84bc`, idem pour
`v2`, et `|v1·v2 − DAT_143cd8370| < DAT_143cd84bc`. Constantes LUES dans le binaire
(`/read_memory`) : `DAT_143cd8374 = 0x3f800000 = 1.0`, `DAT_143cd8370 = 0.0`,
`DAT_143cd8380 = 0x7fffffff` (masque de valeur absolue), `DAT_143cd84bc = 0x3a83126f = 1e-3`.
Une base orthonormee CONSTRUITE satisfait les trois tests : **la queue est lue**, elle n'est pas
conditionnelle en pratique. C'est ce qui leve le « NON resoluble offline sans ground-truth CE »
qui gardait `simStateComplete` a `false` depuis des mois.

## 2. `FUN_14076e494` — la queue handle, partagee par `i60` ET `i57`

```
cVar1 = FUN_14076f91c()      0 bit — garde RUNTIME (DAT_144e61ea0 / DAT_145121140)
                             = exactement le global deja modelise `PositionFullPrecision`
si cVar1 != 0 : FUN_1411b259c -> FUN_1406d676c(br, br, dst, 0x60)      = R(96) brut
sinon (retail, dominant) : FUN_14076e524(dst, br, idxOut, LEVEL = 0x10) :
    R(1) porte d'index ; si 0 -> R(DAT_144632be0) index de region
    FUN_140cc5128 : 3 axes aux largeurs de la ligne LEVEL=16
        index != -1 -> table DAT_1445ccbe0 + (index*0x20 + LEVEL)*0xc
        index == -1 -> table DAT_1445cc9e0 + LEVEL*0xc
```

C'est **le lecteur absolu de `consumeAbsoluteWithGate` MOINS son bit `precHigh`** (ici la garde
est runtime, pas un bit du flux) **et MOINS son `R(2)` « fini » de queue** — `FUN_14076e494`
n'appelle pas `FUN_14076e304`. Porte en Go : `consumeSimStateHandleTail` (`traverse.go`).

## 3. `i57 biped-spartan-ability-component`, branche `tag == 3` — PARTIELLEMENT portable

`FUN_142f262d4(dst, br, param_3)` :

```
FUN_140f03dfc()                          0 bit
a = FUN_1406cf008(br)   R(1)  -> dst[0]
si a != 0 :
    FUN_14297ea84(br)                    R(6)  (decompile 2026-08-17)
    si (dst[2] & 1) != 0 : b = R(1) ; branche gardee par (dst[2] & 0x10) ;
                           FUN_142f04664(dst+4, br, b, param_3)
t = FUN_1406cf008(br)   R(1)  -> dst[1]
si t != 0 : FUN_14076e494(br, dst+0x18, 0x10, 0, param_3, 0)     (la queue du §2)
```

`dst[2]` est un **octet d'ETAT RUNTIME**, invisible du flux : la branche `a != 0` reste
irreductible hors ligne. La branche `a == 0` est ENTIEREMENT portable et l'est desormais
(`consumeSpartanAbilityTag3`, `components_biped_ability.go`) : `R(1)` nul, porte de queue
`R(1)`, et la queue du §2 quand elle est ouverte.

## 4. `i9 object-multiplayer-properties-component` — la PORTE ETAIT INVERSEE

`FUN_1407d4c94(br, dst)` :

```
cVar1 = FUN_1406cf008(br)                R(1)
si cVar1 == 0 : R(5) tag (FUN_1407d54ac) puis le flux TLV (FUN_1407d4e10)   <- PRESENT
sinon         : *(dst + 0xbc) = 0                                            <- ABSENT, 0 bit
```

`FUN_1406cf008` rend la **VALEUR** du bit (MSB du registre, `>> 0x3f`) : `cVar1 == 0` veut donc
dire « bit a 0 ». Le port Go lisait le bloc TLV quand le bit valait **1** — polarite inversee sur
le composant present dans 100 % des records de bipede, et le suspect n°1 de l'histogramme de
decrochage de l'image-cle (25 % des franchissements de frontiere, R7-a). La semantique du `else`
(`dst+0xbc = 0`, effacement d'un champ absent) confirme la lecture independamment.
Corrige le 2026-08-17 dans `components_batch7.go`.

---
# L ECRIVAIN PAR VTABLE — la case `+0x18` du descripteur de composant (lot R7-d, 2026-08-17)

> Acces Ghidra : instance PID 10104 (`HaloInfinite.exe`), **API HTTP** `http://127.0.0.1:8089`
> (`/decompile_function`, `/disassemble_function`, `/read_memory`, `/get_xrefs_to`,
> `/list_segments`, `/get_function_by_address`), LECTURE SEULE — aucun rename, aucun script,
> aucune analyse relancee. Contexte : `PLAN_R7D_ECRIVAIN_VTABLE.md`.
> **Piege d'outillage** : `/disassemble_function` sur une adresse SANS fonction definie rend
> 200 Mo (il desassemble jusqu'au bout du segment). Utiliser `/read_memory` sur 32-48 octets.

## 1. Methode — retrouver une vtable de composant sans xref

`DAT_144e61d88` est nul statiquement (la table de descripteurs est batie a l'init), et Ghidra ne
cree AUCUNE reference vers les cases de vtable (octets non types). La chaine qui marche :

1. dumper `.rdata` (`0x143606000-0x144395200`, 14 217 728 o) par `/read_memory` (55 lectures de
   256 Kio, champ `hex`) ;
2. y chercher le pointeur 8 o du deser connu (table `RECETTE_DECODAGE_FILM_CHUNKS.md` section 6) —
   il est **UNIQUE** pour chacun des 52 desers du bipede ;
3. la case trouvee est `vtable[0x30]` quand la case precedente vaut le thunk `FUN_14076ce9c`
   (`jmp [rax+0x30]`), sinon `vtable[0x28]` (cas d `i0`) ;
4. verifier la base par `vtable[0x08]` = getter de NOM (`lea rax,[rip+X]; ret` puis la chaine) et,
   quand elle existe, par la xref DATA depuis `.data` (l objet descripteur lui-meme).

Exemple ferme : `0x143d0ce00` (`biped-action`) est reference par `0x144747368` (.data) et son
`+0x08` = `0x141177560` pointe `0x143c98cf0` = `"biped-action-component"`.

## 2. Le layout d une vtable de descripteur de composant — UNIFORME sur 51/52

```
+0x00  getter d ID     (la boucle de lecture FUN_14076cb60 en prend le retour ; stubs partages)
+0x08  getter de NOM               (per-composant, `lea rax,[rip+X]; ret`)
+0x10  PREDICAT DE FILTRE (FUN_1404ab600 = `return false`) : c est lui qui decale le masque
+0x18  **SERIALISEUR (ECRITURE)**  (per-composant)
+0x20  FUN_1411c8f80 = `int3`      (virtuelle pure ; exceptions : i1, i25)
+0x28  thunk FUN_14076ce9c = `jmp [rax+0x30]`
+0x30  **DESERIALISEUR (LECTURE)** (per-composant)
+0x38  FUN_1408d8220 = `return 1`  /  FUN_1404ab600
+0x40  FUN_141191ab0 = `*p = 0`
+0x48  BASELINE/PREDICAT (FUN_14076ced0 = zero 16 o en `rdx`) : appele par la boucle de lecture, 0 bit
(+0x50 case supplementaire pour quelques classes : predicat de validation, 0 bit)
```

**Il n y a QUE DEUX cases qui touchent le flux de bits : `+0x18` ECRIT, `+0x30` LIT.** Toutes les
autres sont des stubs constants, un nom, un accesseur ou un `int3`. L hypothese « les cases
voisines portent la serialisation » est donc VRAIE, et elle est fermee : une seule case, `+0x18`.

L archetype porte la MEME paire, decalee : vtable du bipede `0x143737178` (double confirmation —
`+0x60` = `FUN_140f44c38`, deser d etat par defaut connu ; `+0xa0` = `FUN_14076ca20`, getter de
masque par defaut connu). **`+0x58` = `FUN_142f14a68` = l ECRIVAIN de l etat par defaut**, et il
porte les NOMS de ses champs en clair (`"player-representation-name"`,
`"customization-source-participant"`) : le parametre `nom` des primitives d ecriture est un
parametre MORT, conserve en retail.

## 3. Les primitives d ECRITURE (miroirs de `FUN_1406cf008` / `FUN_14076e304`)

Etat d ecrivain : `+0x2c` = compteur de bits ecrits, `+0x30` = accumulateur 64 bits,
`+0x38` = bits tenus, `+0x40` = pointeur de sortie, `+0x10` = fin de tampon, `+0x28` = bits vides.
Le `+0x2c` est ce qui rend la lecture facile : **chaque `+= N` est la LARGEUR ecrite**.

| fonction | largeur |
|---|---|
| `FUN_1406d49c4(w, ., b)` | **W(1)** |
| `FUN_142f1f71c(w, b)` | **W(2)** |
| `FUN_1407edaf4(w, "nom", v)` | **W(32)** |
| `FUN_142f22020(w, ., p)` | **3 x W(32) = vec3 IEEE, 96 bits** |
| `FUN_1407eb6a8(w, p, idx, ., .)` | `W(1) (idx == -1)` ; si `idx != -1` : `W(DAT_144632be0)` ; puis `FUN_140ffc6b0` = les 3 axes |
| `FUN_1406d60f4(w, ., p, 0x60)` | vec3 BRUT, 96 bits |

## 4. Ce que l ecrivain dit de chaque composant lu

### `i60 simulation-state` — le port de R7-b est CONFIRME

`vtable[0x18]` = `FUN_142f04e2c` (`sub = etat + 0xa48`) puis `FUN_142edd10c(sub, w)` :
`W(1) *(sub+0x28)` ; si non nul : 2 x `FUN_14299813c`, 4 x `FUN_142ee5630`, `W(2) *(sub+0x29)`, ...
C est le MIROIR EXACT du lecteur (`R(1)` vers `dst+0x28` ; si 0, fin). Aucune divergence.

### `i63 biped-action` — le port est CONFIRME, et la limite dure aussi

`vtable[0x18]` = `FUN_142f05144` (`sub = etat + 0xaa8`) puis `FUN_142f27a68(sub, w)` :

```
FUN_142f22020(w, sub)              vec3 96 bits
W(4) count1 = *(uint*)(sub+0x18)
count1 x { W(7) *(item+0x30) ; FUN_142ef5340(item, w) }      item stride 0x38
count2 = FUN_1409fe718(sub, 0x49)                            POPCOUNT RAM, 0 bit
count2 x { W(1) (c != -1) ; si oui : FUN_142f1f71c(w, c) = W(2) }
FUN_142f22020(w, ...)              vec3 96 bits
```

Identique, largeur pour largeur, a `consumeBipedAction` (96 + R(4) + ... + 96 = 196 au minimum).
**Et le compte `count2` sort du MEME popcount RAM cote ECRITURE** : la limite « non recuperable
hors ligne » de R7-b n est pas un defaut de lecture, elle est dans le format. Fil ferme.

### `i9 object-multiplayer-properties` — la correction de R7-b est CONFIRMEE INDEPENDAMMENT

`vtable[0x18]` = `FUN_142f075d8` (`sub = etat + 0x2b4`) puis `FUN_1420ab570(sub, w)` :

```
c = *(char*)(sub + 0xbc)
W(1) (c == 0)                      le bit vaut 0 QUAND le bloc est present
si c != 0 : W(5) c ; puis le bloc TLV (FUN_1407d5768/5714 puis FUN_14346074c(buf, w))
```

La polarite corrigee par R7-b (bit a 0 = bloc PRESENT) et le `R(5)` de tag sont tous deux
confirmes par l ecrivain. Aucune divergence.

### `i0 object-position-dynamic-precision` — LA DIVERGENCE

`vtable[0x18]` = `FUN_14320678c` — et c est l ecrivain d ETAT COMPLET, pas de delta : il appelle
son corps avec une reference de base **NULLE** et un drapeau **0**.

```
FUN_14320678c(desc, w, ?) :
  sub  = *(desc + 0x30)
  ref  = {0, 0}                                  AUCUNE baseline
  FUN_14320696c(w, sub, &ref, desc, 0)           param_5 = 0
  FUN_142f1f71c(w, *(byte*)(sub + 0x10))         W(2), INCONDITIONNEL, EN DERNIER

FUN_14320696c(w, sub, ref, desc, flagRaw) :      (desassemble, args resolus au registre)
  W(1) flagRaw                                   (= 0 pour l etat complet)
  W(1) (ref[1] != 0)                             (= 0 : pas de baseline)
  h = *FUN_1404d4f04(sub + 0x2a4)                0 bit
  si flagRaw != 0 : W(1) (h != -1) ; FUN_1406d60f4(w, sub+4, 0x60) = 96 bits BRUTS
  sinon si ref[1] != 0 : FUN_142e2c9bc(...)      chemin RELATIF
  sinon                : FUN_142e2d86c(w, sub+4, *(short*)(sub+0x12)) avec 5e arg = (h != -1)
                          = W(1) (h != -1) ; FUN_1407eb61c(w, ..., 0x10)
  si h != -1 : FUN_143206854(w, sub, desc)       la QUEUE DE HANDLE, gardee par h
```

Face au lecteur porte (`consumeObjectPositionDynamicPrecisionD` + `consumeAbsoluteWithGate` +
`consumePositionHandleTail`), **trois ecarts, tous dans le meme sens** :

1. **Le 3e bit n est pas un « precHigh » qui SUPPRIME la charge utile.** L ecrivain ecrit ce bit
   (`h != -1`) PUIS la position, toujours. Le lecteur, lui, fait `precHigh := R(1) ; if precHigh
   { return }` : quand ce bit vaut 1 il saute toute la position. C est une SOUS-LECTURE de
   `1 + [idxW] + 3 x axisW` bits.
2. **Ce meme 3e bit est la PORTE DE LA QUEUE DE HANDLE**, lisible dans le flux. Le lecteur la
   garde sur un booleen RUNTIME (`PositionDeltaHasHandleTail`, faux par defaut) et la force a
   `false` sur le chemin absolu.
3. **Le champ de 2 bits est INCONDITIONNEL et il vient EN DERNIER**, apres la queue de handle.
   Le lecteur ne le lit qu a l interieur du chemin absolu, avant la queue, et pas du tout quand
   il est sorti tot.

Consequence de forme : dans une image-cle, `i0` **n est PAS un vec3 brut de 96 bits** — l ecrivain
d etat complet prend le chemin QUANTIFIE (`flagRaw = 0`, `ref = NULL`), donc une largeur
DEPENDANTE DE LA CARTE : `2 + 1 + 1 + [idxW] + 3 x axisW + [queue] + 2`. Sur un decoupage 13/13/14
et sans index ni queue : **46 bits**, la ou le lecteur porte en consomme 117 en mediane.
Ceci CONTREDIT la decouverte n2 de R7-b (« en image-cle i0 prend le chemin brut, 117 bits,
identiques sur trois cartes ») : ces 117 bits sont ce que le LECTEUR consomme, pas ce que
l ECRIVAIN pose.

### `i5 object-shield-vitality` — le port est CONFIRME (5e composant lu, hors perimetre initial)

`vtable[0x18]` = `FUN_142f07d68` (`sub = etat + 0x30`) :

```
FUN_1406d22c0(w, ..., *(uint*)(sub+0x60), 0, DAT_143cd893c, 8, 0, 1)   W(8) quantifie
FUN_1432073f4(sub + 0x6a, w) :
    W(1) *(p+4) ; si non nul : 2 x FUN_141e78f88(p, w)   = la paire de regen
W(16) *(short*)(sub + 100)
4 x W(1)  (sub+0x66, puis trois autres)
```

Face a `decodeObjectShieldVitality` (`R(8)` ; `R(1)` regen puis 2 x [`R(1)` + `R(12)`] ; `R(16)` ;
4 x `R(1)`) : identique, largeur pour largeur, y compris les QUATRE bits de queue. Aucune
divergence.



### `i0` — LE LECTEUR DU JEU DIT LA MEME CHOSE QUE L ECRIVAIN : c est le PORT Go qui est faux

La divergence ci-dessus n est pas « l ecrivain contre le lecteur » : c est **l ecrivain ET le
lecteur du jeu contre le port Go**. La case `+0x30` d `i0` (`FUN_14076e29c`, le lecteur de la
paire) se lit en trois lignes :

```
FUN_14076e29c(., reader, desc) :
  sub = *(desc + 0x10)
  h = FUN_14076e420(reader, sub+4, 0x10)    <- R(1) puis la CHARGE UTILE, toujours
  FUN_14076e3e4(reader, sub, ., h)          <- la queue de handle, GARDEE PAR h
  si FUN_140492128(sub+4) : FUN_14076e304(reader, sub+0x10)   <- R(2), en DERNIER

FUN_14076e420(reader, dst, w, f) :
  cVar1 = FUN_1406cf008()                   R(1)
  table = (cVar1 != 0) ? &DAT_143b8c6d0 : NULL      <- il CHOISIT UNE TABLE DE PLAGE
  FUN_14076e494(reader, dst, w, f, 0, table)        <- la charge utile, INCONDITIONNELLE
  return cVar1
```

**Le bit que le port Go appelle `precHigh` ne supprime rien : il selectionne la table de
dequantification** (`DAT_143b8c6d0` ou aucune), et il est **la porte de la queue de handle**. La table `DAT_143b8c6d0`, lue octet a octet, vaut `{-100.0, +100.0} x 3` puis `60` : une plage LOCALE de plus ou moins 100 unites, pas les bornes de la carte.
Le port Go (`consumeAbsoluteWithGate`) en fait un `if precHigh { return }` a zero bit de charge,
et force la queue a `false`. Les deux lectures du jeu — celle qui ecrit et celle qui lit — sont
d accord contre lui.

Grammaire d `i0` en ETAT COMPLET, etablie des DEUX cotes :

```
R(1) flagRaw   (0)          en-tete FUN_1406cfe44 / FUN_14320696c
R(1) hasRef    (0)
R(1) h                      selecteur de plage ET porte de queue
R(1) idxSel ; si 0 : R(idxW)
3 x R(axisW)                les axes de la CARTE
si h : queue de handle      FUN_14076e3e4 / FUN_143206854
R(2)                        FUN_14076e304 / FUN_142f1f71c, EN DERNIER
```

Soit **`6 + axisW[0] + axisW[1] + axisW[2]`** bits dans le cas dominant : 46 sur 13/13/14,
56 sur 17/17/16, 59 sur 18/18/17.
### La QUEUE DE HANDLE d `i0`, cote ecriture

`FUN_143206854(w, sub, desc)` — appelee seulement quand `h != -1` :

```
FUN_1409a5ff0(sub + 0x2a4, w, 0, *(byte*)(desc + 0x8c), ...)   ecriture du handle (largeur variable)
h2 = *FUN_1404d4f04(sub + 0x2a4)                               0 bit
W(1) (h2 != -1)
si h2 != -1 : W(11)                                            le mot de region
```

Meme forme que la queue du lecteur (`consumePositionHandleTail` : selecteur, mot d index, puis
`R(1)` presence de region et `R(11)`) — mais **gardee par un bit du FLUX** (`h`), pas par le
booleen runtime `PositionDeltaHasHandleTail` que le lecteur porte force a `false`.

### L ETAT PAR DEFAUT du bipede — l ecrivain `FUN_142f14a68` CONFIRME le lecteur

Case `+0x58` de la vtable d archetype (`0x143737178`), face au lecteur `FUN_140f44c38`
(case `+0x60`, porte dans `default_state.go`) :

```
FUN_1406d49c4(w, ., 1)                          W(1) = 1        <- la porte g0, toujours VRAIE
W(8) = 0x0d                                     le selecteur (13), donc toujours ecrit
FUN_1406d49c4(w, ., (rep != 1 && rep != -1))    W(1) gRep
si gRep : FUN_1407edaf4(w, "player-representation-name", *(sub+100))   W(32)
FUN_142b549c0(w, *(sub+0x68), "customization-source-participant")
FUN_142f1bc2c(sub, w)
FUN_1406d49c4(w, ...)                           W(1)
si *(sub+0x6c) != -1 : W(6)
W(1) *(char*)(sub+0x61)
FUN_1407edb6c(w, sub+0x70)
si *(sub+0x74) != -1 : FUN_1407eb600(w, sub+0x78, 0x10) ; FUN_14299813c(w)
FUN_1407eb9bc(w) ; FUN_1406d49c4(w) ; FUN_1407edb6c()
```

Le lecteur porte lit `g0 = R(1)` puis, si `g0`, `R(8)` de selecteur, puis `gRep = R(1)` et
`R(32)` de nom : **meme ordre, memes largeurs**. L ecrivain ajoute la seule chose que la lecture
ne pouvait pas savoir — `g0` vaut TOUJOURS 1 et le selecteur vaut TOUJOURS 13 sur ce chemin.
Les NOMS de champs (`"player-representation-name"`, `"customization-source-participant"`) sont
des parametres MORTS des primitives d ecriture, conserves en retail : ils nomment les champs
sans rien couter au flux.

### Les cases vont par PAIRES : etat complet / delta

`i0` a un DEUXIEME ecrivain, en `vtable[0x20]` : `FUN_143206a88`. Il appelle le meme corps
`FUN_14320696c` mais avec une reference de base REELLE (`param_4`) et un drapeau brut calcule sur
un octet d etat runtime, puis le meme `FUN_142f1f71c` = `W(2)` de queue. La lecture des deux
cases se lit donc ainsi :

```
+0x18  ECRIRE l ETAT COMPLET   (ref NULLE, drapeau 0)        <-> +0x28  LIRE l etat complet
+0x20  ECRIRE le DELTA         (ref reelle)                  <-> +0x30  LIRE le delta
```

Pour les composants a une seule forme (la grande majorite), `+0x20` est `int3` — pas d ecrivain
de delta distinct — et `+0x28` est le thunk `FUN_14076ce9c` qui renvoie sur `+0x30` : une seule
lecture, une seule ecriture. C est ce qui donne le layout apparent de la section 2.

**Consequence directe pour l image-cle** : le format a bien un serialiseur d ETAT COMPLET par
composant, et pour `i0` c est nommement `FUN_14320678c`. Ce n est plus une inference, c est la
fonction qui ecrit la position d un record d image-cle.
## 5. Bilan de lecture — 5 composants + l etat par defaut, UNE seule divergence

| composant | ecrivain `vtable[0x18]` | verdict |
|---|---|---|
| `i0 object-position-dynamic-precision` | `FUN_14320678c` | **DIVERGE** (3 ecarts, cf. section 4) |
| `i5 object-shield-vitality` | `FUN_142f07d68` | conforme |
| `i9 object-multiplayer-properties` | `FUN_142f075d8` puis `FUN_1420ab570` | conforme (confirme la correction R7-b) |
| `i60 simulation-state` | `FUN_142f04e2c` puis `FUN_142edd10c` | conforme |
| `i63 biped-action` | `FUN_142f05144` puis `FUN_142f27a68` | conforme (et confirme la limite dure `count2`) |

Sur les 52 composants du bipede balayes, **seuls TROIS ont un `vtable[0x20]` autre que `int3`**,
donc seuls trois ont une forme DELTA distincte de leur forme ETAT COMPLET : `i0`
(`FUN_143206a88`), `i1 object-translational-velocity` (`FUN_14320ca60`) et `i25 unit-command-tick`
(`FUN_142ee007c`). Les 49 autres ecrivent la meme chose dans les deux cas.

**RESTE OUVERT (non traite par ce lot)** : la BOUCLE d ecriture au niveau du RECORD — le miroir
de `FUN_14076cb60` qui appelle, composant par composant, la case `+0x18`. Elle n a pas ete
localisee : `FUN_142f14a68` (ecrivain d etat par defaut) n a AUCUNE xref CODE, seulement sa case
de vtable `0x1437371d0` ; les voisines de la vtable d archetype (`+0x50` `FUN_142f1dcc0`,
`+0x68` `FUN_142f156b0`, `+0x78` `FUN_142f09490`, `+0x80` `FUN_142f12a4c`) sont respectivement de
la telemetrie, un nettoyage, un bloc de pertinence et un wrapper — aucune ne boucle sur les
composants ; et les fonctions voisines des lecteurs de record (`FUN_1408f1948`, `FUN_141f865d8`)
non plus. C est elle qui dirait si le record d image-cle porte un CADRE (en-tete, masque, ordre)
different de celui du record NEW.

# HANDOFF — Arme par kill SAME-CLOCK (100% offline)

> 2026-07-05. Débloque le plafond 58% Team Slayer du warp. Lire
> `../../README_KILLWEAPON_INDEX.md` pour l'historique complet ; ce doc = la voie same-clock
> aboutie. Mémoire : [[project_killfeed_sameclock_localized]].

## Le problème (rappel)

L'arme par kill était résolue par le **warp** (`cmd/tmp_offwarp`, 96% Fiesta / **58% Team
Slayer**) : il décode le dégât `0xd2` (attaquant + arme, horloge FRAME `ts`) et le kill feed
`chunk_27` (tueur/victime, horloge TEMPS-JEU `TimeMS`), puis **corrèle 2 horloges** par un fit
linéaire itératif (`warp(ts)→TimeMS`). Résidu ~1-2s irréductible → en Team Slayer, BR75/MA40
alternés en 1-2s = mauvaise rafale → plafond 58%. La seule voie >90% (index §3 Phase 3) = lire
tueur/victime dans le **flux FRAME** (même horloge que le `0xd2`), ce qui exigeait de
**localiser le kill-event** dans le flux — resté "ouvert".

## Le déblocage

1. **Confirmation CE (même horloge)** : le kill-event `FUN_14104bd08` et le dégât `0xd2`
   (`FUN_14080c1f8`) sont décodés depuis le **MÊME buffer runtime** (`0x2B98D1F0000`, vérifié
   double-hook) → même flux, MÊME HORLOGE. Cf `project_killfeed_sameclock_localized`. (Le
   dead-state du biped, lui, est une impasse : victime idx0 POV-only + GID constant.)
2. **Localisation offline** : `cmd/tmp_framemarkers` (histogramme `payload[0]` des frames
   type-0) + `cmd/tmp_killframe` (teste la grammaire §174 par marqueur+offset). Trouvé :
   - **Frame kill-event = marqueur `0xE6`** (98 frames sur 000d5950, 97 sur 9b191a7f ≈ nb kills
     du chunk_27). Universel (2 films, 2 modes).
   - **tueur/victime/assistant à l'offset ~80**, grammaire §174 : `R1+optR5 ×3`, présence =
     `R(1)==0 → R(5)`. Puis `R32` scalaires (dont `s1` = médaille/type de kill, PAS l'arme :
     `0x103D` mappe vers 6 armes).

## Le pipeline (`cmd/tmp_sameclockkw`)

```
CGO_ENABLED=0 go run ./cmd/tmp_sameclockkw [filmID]   # défaut 000d5950
```

1. Itère les frames type-0 de tous les chunks. Deux extractions, **même horloge `ts = d[off+8]`** :
   - `0xE6` → kill (tueur/victime à offset 80).
   - `0xd2` + frères DamageReport (`0xe9/0x89/0xc7/0xc0/0xc2/0xc3/0xd3/0xca/0xc4`) → dégât
     (attaquant `R5 bit 36 >>1`). `0xd2` avec famille+suffixe `0x42c9679f` = **arme à feu nommée** ;
     les frères = confirmation de kill / mêlée / grenade (pas d'arme à feu).
2. **Attribution** : pour chaque kill, le dégât ARME-À-FEU du tueur (`attaquant == tueur`) **le
   plus proche** du kill = l'arme (le coup fatal). Si aucun `0xd2` → mort non-arme-à-feu (couverte
   par un marqueur frère). Si aucun dégât du tueur → suicide/chute.
   - **La temporalité ne gate PAS** : on prend le dégât le plus proche (le coup fatal),
     l'écart `dt` (ms) n'est qu'un contrôle de santé — il est serré (~8-15ms) = c'est le bon.

## Résultats — VALIDÉS vs vérité-terrain CE, MÊME MATCH (2026-07-05, correction majeure)

> Correction : les « 100% » précédents étaient de la **couverture** (chaque `0xE6` recevait UNE
> arme), PAS de l'accuracy, et la validation Fiesta 000d5950 comparait à une **capture d'un AUTRE
> match** (fallback silencieux `dmgcapture_run2.bin` dans `tmp_offwarp`, car `000d5950_dmg.bin`
> n'existe pas). Le SEUL film avec une paire dmg+kill propre = **9b191a7f** (Team Slayer). Outil de
> validation same-match, par joueur, sans fallback : `cmd/tmp_kwval [film]`.

**Comparaison équitable sur 9b191a7f (vs sa vraie capture live)** :

| Métrique | Warp (`tmp_offwarp`, chunk_27 + 2 horloges) | Same-clock (`tmp_kwval`, 0xE6) |
|---|---|---|
| Kills trouvés | **87** | 74 (0xE6) |
| Distribution globale vs live | **73/92 = 79 %** | 59/92 = 64 % |
| Par-kill / par-joueur | 58 % (pont tsc) | 38 % (greedy) / 32 % (identité) |

**Le warp est DEVANT le same-clock sur le bon match (79 % vs 64 %).** L'idée same-clock reste saine
(une horloge = pas de résidu du fit warp), mais elle est bridée par un **décodage 0xE6 trop
lessiveux** : il ne trouve que 74 kills sur 95, avec des slots-fantômes (8,10,12,14,18 = tueurs mal
décodés) et une numérotation slot ≠ idx live (permutation confirmée par greedy cosine). La source
d'arme `0xd2`, elle, est BONNE (armes par slot alignées au live : slot0↔idx0 BR75/MA40, slot2↔idx2
Fuel Rod, slot4↔idx1 BR75…).

## Le vrai verrou : framing VARIABLE du kill-event 0xE6 (oracle-prouvé)

`cmd/tmp_kwval <film> oracle` cherche le décalage de bits 0xE6 qui reproduit le multiset tueurs du
live. Résultat : **aucun offset fixe ne dépasse ~59-62 kills sur 95** (bitoff 79 ≈ le mien = déjà
l'optimum ; bitoff 112 décode 91 frames mais s'effondre sur 2 valeurs = garbage). Donc le
tueur/victime `0xE6` est à une **position de bits VARIABLE** (préfixe de longueur variable avant les
champs) — un offset fixe plafonne structurellement le recall à ~62 %. C'est CE qu'il faut RE pour
débloquer l'avantage same-clock : parser le préfixe du frame `0xE6` (position calculée par frame),
en s'appuyant sur l'oracle live (95 kills tueur/victime alignables chronologiquement aux 97 frames).

## RE Ghidra du kill-event (2026-07-05) — grammaire CONNUE, position VARIABLE

Décompilation du désérialiseur `FUN_14104bd08` (localisé en CE, dispatché par descripteur/vtable —
RCX=descripteur, RDX=0x28=taille sortie, R9=reader déjà positionné ; **aucun appelant statique**).
Grammaire EXACTE de la struct de sortie (param_3, 0x28 octets) :

```
[R5 tueur][R5 victime][R32 scalaire][R1 bool][R5 assist][R32 scalaire]   (+ queue optionnelle)
```

- Chaque champ tueur/victime/assist = **5 bits** (`FUN_1407f2058` lit 5 bits MSB-first, TOUJOURS
  consommés) = un **index LOCAL** (slot 0-31). **Pas de bit de présence** : `FUN_14049746c` résout
  l'index au registre d'entités (présent → handle via `FUN_140e958c4`, sinon `-1`) SANS consommer de
  bit ; `FUN_140e958c4` reconstruit le handle depuis le registre (pas depuis le bitstream). Offline =
  index local brut. → Ma grammaire précédente `R1+optR5` était FAUSSE (il n'y a pas de bit présence).

**Mais la POSITION du kill-event dans la frame est pleinement VARIABLE.** Le kill-event est un
archétype du flux ECS générique, dispatché par descripteur ; ses champs commencent là où la boucle de
frame générique (type-index + entity-id + …) a laissé le reader. Preuves (`cmd/tmp_kwval align`) :
- Aucun S fixe ne donne une confusion propre. Le meilleur S par DISTRIBUTION (oracle) = 83, mais la
  confusion **same-clock** (tueur `0xE6` R5@83 × attaquant du `0xd2` fatal, MÊME horloge) a une pureté
  de **32 %** = pas le vrai tueur (une coïncidence de distribution).
- Scan par-frame du S où `R5@S == attaquant 0xd2 fatal` : S éparpillé (40,41,43,…,121), aucune bande
  = bruit (≈3,75 matchs aléatoires/frame sur 120 positions). → position **entièrement variable**.

**Conséquence** : localiser le tueur `0xE6` offline exige le **décodeur de flux ECS complet** (le
chantier keyframe/delta record-loop, bit-exact mais itératif par-archétype, cf `project_kill_feed_frame_decoder`
+ `reference_ecs_runtime_deser_table`), PAS un offset. Le « 83 % global » de `tmp_kwval` (défaut) venait
du fallback fatal, pas d'une attribution par tueur validée.

## DÉBLOCAGE 2026-07-05 (soir) — framing du paquet-événement 0xE6 CRACKÉ (Ghidra), tueur à 93 %

Le kill-event n'est PAS une frame autonome ni la record-loop `0xA0` : c'est **un événement dans un
paquet-événement**, traité par le **dispatcher générique `FUN_14080a9d4`** (décompilé) :
```
[R7 code de type d'événement]  (= 7 bits de poids fort du marqueur : 0xE6=115 kill, 0xd2=105 damage)
[3× R1 gate]                    (le "3-loop" ; vtable[0x58] appelé SANS le bitreader => 0 bit lu, donc 3 bits fixes)
[déser vtable[0x68] = DeserKillEvent_ECS]  (le kill-event proprement dit)
```
Le déser lit chaque champ entité via `FUN_1407f2058` = **R1 gate + optR5** (index LOCAL). Donc le
**tueur commence à bit 10** (7+3), PAS 80. (Le bitpos CE 578/2756 était la position CUMULÉE du buffer
runtime, qui ≠ mes chunks — d'où l'échec du signature-match.)

**MESURÉ (`cmd/tmp_kwval <film> decode [preSkip]`, défaut 10)** sur 9b191a7f vs la vérité-terrain live :
- Balayage preSkip : 42%(8) 67%(9) **93%(10)** 74%(11) 0%(12) → **pic NET à 10** = confirme l'offset Ghidra.
- **Tueur : accord de distribution (permutation-invariant) = 88/95 = 93 %** — au-dessus du warp (79 %).
- **field0 = UNE seule entité fiable** (93-94 %) ; field1 (readOpt juste après) ET field4 (après R32+R1)
  = **majoritairement absents (-1)** = assisters optionnels. Donc le kill-event porte 1 réf claire + des
  optionnels. field0 matche autant le tueur (93 %) que la victime (94 %) — distributions symétriques en
  Slayer, indiscernables par distribution. Confusion same-clock basse (field0 × attaquant 0xd2 = 34 %)
  → **field0 = probablement la VICTIME** (pas l'attaquant).
- **Implication à trancher** : si field0 = VICTIME, on a victime (93 %) + tueur/arme via 0xd2 fatal
  same-clock (= mode fatal 73 %, SOUS le warp 79 %). Si field0 = TUEUR, on peut faire l'arme par-tueur
  same-clock (bat le warp). **Décisif = validation PAR-KILL** : appliquer le fit ts→TimeMS du warp aux
  ts-frame des 0xE6 pour aligner sur les kills live (chunk_27 / 9b191a7f_kill.bin) → trancher tueur vs
  victime + mesurer l'accuracy réelle. (Le rang chrono échoue : 2 horloges désynchronisées.)
- **Reste** : (1) trancher field0 killer/victim via le fit warp ; (2) table index-local (8-15 = bipeds)
  → idx live ; (3) intégrer l'arme same-clock ; (4) 2e entité si besoin.
- Fonctions Ghidra renommées : `FUN_14080a9d4` = dispatcher d'événements ; `FUN_1406cf008` = R1 gate.

## LE MUR VERS 100% (2026-07-08) — walker le déser de dégât (chantier frame-decoder)

Mécanisme 100% vérifié (CE) : kill-event DANS le paquet de dégât fatal, field0=victime field1=tueur
(93% au curseur), arme=famille du dégât. Détecteur fatal=taille. MAIS localiser le kill-event OFFLINE
sans CE échoue par tout raccourci :
- Ancre bit-pattern IMPOSSIBLE : le type du kill-event = R7 **0x55 (85) = 1010101** (motif alterné très
  fréquent). Préambule fixe 10 bits (0x2A8, constant 134/134) MAIS le locator « 1er 0x2A8+kill-event »
  ne tombe exact que **16/134** (faux motifs alternés partout avant le vrai curseur). Idem R7=85 (14/134).
  Offset fixe / suffixe+204 : ~75% (approx). Aucun ne donne 100%.
- **Le curseur EXACT = fin du déser de dégât.** Le paquet = `[event dégât : R7=105 + 3loop + déser dégât
  (longueur L data-dépendante)][event kill : R7=85 + 3loop + kill-event]`. cursor = 20 + L. Il FAUT L.

**Déser de dégât = `FUN_14080c1f8`** (décompilé). Lit : 2×R(1), `FUN_141fcf670`, `FUN_1407f2034`,
`FUN_1406d00ec`, `FUN_14080d69c` (R1+optR32), `FUN_14080dec4` (variant_name=arme), R(1), `FUN_1406cf008`,
puis compteur `FUN_14080cc68` → **boucle** (R2+gate+R32/quantized), 2e **boucle** `param_3+0xf8` (R4+gate+
quantized `FUN_1406d310c(6)`/`FUN_1406d3140`), R(30), R(6)×N, `FUN_14076e494` (quat)… = data-dépendant +
largeurs quantifiées runtime (LE MUR habituel, cf `project_kill_feed_frame_decoder`). Porter ce déser
(+ ses ~10 sous-desers) pour calculer L = chantier multi-session. Oracle de vérif : les curseurs CE
(`cmd/tmp_kwval fataldet`) donnent L = cursor-20 par paquet → vérifier chaque bout porté.

⇒ 100% offline = porter le déser de dégât (frame-decoder). Bounded mais gros. Le pipeline actuel
(`tmp_kwval pipeline`, ~75%/couverture partielle) est l'approximation atteignable sans ce port.

## Honnêteté / limites

- Tant que le 0xE6 n'est pas RE en framing variable, **le warp (79 % global / 58 % par-kill) reste la
  meilleure méthode offline** ; ne PAS le remplacer par le same-clock en l'état (régression).
- La source d'arme `0xd2` est solide des deux côtés (même famille h32 + suffixe `0x42c9679f`).
- Cause non-arme-à-feu : confirmée dans le `0xd2` (mécanisme uniforme, cf `cmd/tmp_dmgfam`) mais
  sur 9b191a7f 1 seul kill non-firearm (`cause-164B3CFA`) — marginal ici.
- Piège de validation à ne JAMAIS refaire : comparer à une capture d'un autre match. `tmp_kwval`
  refuse le fallback ; `tmp_offwarp` l'a (à corriger ou à n'utiliser que sur 9b191a7f).

## Reste à faire (productionisation)

1. **Valider** tueur/victime `0xE6` vs `chunk_27` (roster type-8, comme `tmp_offwarp`) + mesurer
   l'accuracy vs la capture CE.
2. **Nommer les non-arme-à-feu** (point 2) : RE des marqueurs frères (mêlée/grenade/splatter) —
   nouveau, le warp ne les décode pas. **Premier pas fait** (`cmd/tmp_sameclockkw` reporte les
   marqueurs frères fataux ; `cmd/tmp_killframe <film> <hex> dump` dumpe leur structure) : les
   **CORRECTION 2026-07-05 : la cause EST dans le `0xd2` (mécanisme uniforme).** Mon « cul-de-sac »
   précédent était FAUX (je m'étais appuyé sur la campagne précédente sans vérifier le total). Vérifié
   avec `cmd/tmp_dmgfam` : sur Team Slayer, **2874 records `0xd2` dont 115 SANS le suffixe firearm
   `0x42c9679f`** (que le warp jetait) — ce sont des dégâts non-arme-à-feu avec un **suffixe de type
   différent** = la cause. 2 familles : `0x592CF3E8/E9` et `0x164B3CF8/FA` (2 types : mêlée/grenade/
   splatter). Attaquant bien à **bit 36** (joueurs 2/3/6). Fiesta (000d5950) : 0 non-firearm ; ça
   dépend du mode/armes. Les frères 0xC0/0xC2 = apparence (ça, c'était juste) mais **la cause n'y est
   pas — elle est dans le `0xd2`**. `tmp_sameclockkw` décode maintenant TOUS les `0xd2` (firearm →
   arme ; autre suffixe → `cause-<suffixe>`). **Reste** : (1) nommer les 2 suffixes → cause (grenade/
   mêlée, via cross-ref ou tag-map) ; (2) attribuer le BON `0xd2` fatal (distinguer grenade vs arme à
   feu du même tueur — le plus proche n'est pas toujours le fatal) ; (3) améliorer le matching
   `0xE6`↔`0xd2` (les 13-15 « suicide » = surtout des numérotations/timing ratés).
3. **Brancher en prod** : remplacer dans `internal/sync/backfill_weapons.go` le
   `chunk_27 + warp 2-horloges` par le `0xE6 same-clock` (garder le décodage `0xd2` du warp).
   Le same-clock supprime la fonction `warp()` et son fit → gain surtout Team Slayer.

## Outils

- `cmd/tmp_framemarkers [film]` — histogramme des marqueurs `payload[0]` (frames type-0).
- `cmd/tmp_killframe [film] [markerHex]` — teste la grammaire §174 par marqueur+offset.
- `cmd/tmp_sameclockkw [film] [killOffset]` — le pipeline same-clock (résultats ci-dessus).
- `cmd/tmp_offwarp [film]` — le warp historique (référence 96%/58%).
